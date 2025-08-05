use crate::handler::handler::proto::numerix_server::NumerixServer;
use crate::handler::handler::MyNumerixService;
use crate::logger;
use crate::pkg::config::config;
use crate::pkg::metrics::metrics;
use crate::pkg::middleware::middleware::GrpcMiddlewareLayer;
use futures::StreamExt;
use hyper::service::service_fn;
use hyper::Body;
use hyper::{Request as HyperRequest, Response as HyperResponse};
use std::convert::Infallible;
use tokio::net::TcpListener;
use tokio::net::TcpStream;
use tokio::sync::mpsc;
use tokio::time::{interval, Duration};
use tonic::transport::Server;

pub async fn init_server() {
    let configs = config::get_config();
    let address = format!("0.0.0.0:{}", configs.application_port.to_string());
    let channel_buffer_size = configs.channel_buffer_size as usize;

    let listener = TcpListener::bind(&address).await;
    let listener = match listener {
        Ok(listener) => listener,
        Err(e) => logger::fatal(
            format!("Failed to bind TCP listener to address {}", address),
            Some(&e),
        ),
    };

    let (grpc_tx, grpc_rx) = mpsc::channel::<TcpStream>(channel_buffer_size);
    let (http_tx, mut http_rx) = mpsc::channel::<TcpStream>(channel_buffer_size);

    let grpc_tx_metrics = grpc_tx.clone();

    tokio::spawn(async move {
        let mut interval_timer = interval(Duration::from_secs(60)); // Report every 10 seconds
        loop {
            interval_timer.tick().await;

            let grpc_capacity = grpc_tx_metrics.capacity();
            let grpc_max_capacity = grpc_tx_metrics.max_capacity();
            let grpc_used = grpc_max_capacity - grpc_capacity;

            let _ = metrics::count("numerix.channel.grpc.queue_size", grpc_used as u64, &[]);
            let _ = metrics::count(
                "numerix.channel.grpc.queue_capacity",
                grpc_max_capacity as u64,
                &[],
            );
        }
    });

    let numerix_service = MyNumerixService::default();
    let grpc_service = NumerixServer::new(numerix_service);
    let layer = tower::ServiceBuilder::new()
        .layer(GrpcMiddlewareLayer::default())
        .into_inner();

    let grpc_rx_stream =
        tokio_stream::wrappers::ReceiverStream::new(grpc_rx).map(Ok::<_, std::io::Error>);

    tokio::spawn(async move {
        if let Err(e) = Server::builder()
            .layer(layer)
            .add_service(grpc_service)
            .serve_with_incoming(grpc_rx_stream)
            .await
        {
            logger::error("Failed to start gRPC server with incoming stream", Some(&e))
        }
    });

    async fn http_router(req: HyperRequest<Body>) -> Result<HyperResponse<Body>, Infallible> {
        match req.uri().path() {
            "/health/self" => Ok(HyperResponse::new(Body::from("true"))),
            _ => {
                let mut response = HyperResponse::new(Body::from("Not Found"));
                *response.status_mut() = hyper::StatusCode::NOT_FOUND;
                Ok(response)
            }
        }
    }

    tokio::spawn(async move {
        loop {
            match http_rx.recv().await {
                Some(stream) => {
                    tokio::spawn(async move {
                        if let Err(e) = hyper::server::conn::Http::new()
                            .serve_connection(stream, service_fn(http_router))
                            .await
                        {
                            logger::error("Failed to serve HTTP connection", Some(&e));
                        }
                    });
                }
                None => {
                    logger::info("HTTP receiver channel closed, shutting down HTTP handler");
                    break;
                }
            }
        }
    });

    logger::info(format!(
        "Numerix service is running at port {}",
        configs.application_port
    ));
    loop {
        let accept_result = listener.accept().await;
        let (stream, addr) = match accept_result {
            Ok((stream, addr)) => (stream, addr),
            Err(e) => {
                logger::error(
                    format!("Failed to accept incoming TCP connection on {}", address),
                    Some(&e),
                );
                continue;
            }
        };

        let grpc_sender = grpc_tx.clone();
        let http_sender = http_tx.clone();

        tokio::spawn(async move {
            let mut buff = [0; 24];

            match stream.peek(&mut buff).await {
                Ok(bytes_read) if bytes_read >= 24 => {
                    if &buff[0..24] == b"PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n" {
                        if let Err(e) = grpc_sender.send(stream).await {
                            logger::error(
                                format!("Failed to send gRPC stream from {} to server", addr),
                                Some(&e),
                            );
                        }
                    } else {
                        if let Err(e) = http_sender.send(stream).await {
                            logger::error(
                                format!("Failed to send HTTP stream from {} to server", addr),
                                Some(&e),
                            );
                        }
                    }
                }
                Ok(bytes_read) => {
                    logger::error(
                        format!(
                            "Insufficient protocol data from {}: got {} bytes, need 24",
                            addr, bytes_read
                        ),
                        None,
                    );
                }
                Err(e) => {
                    logger::error(
                        format!("Failed to peek connection data from {}", addr),
                        Some(&e),
                    );
                }
            }
        });
    }
}
