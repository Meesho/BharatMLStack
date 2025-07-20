use std::{future::Future, mem::replace, pin::Pin, task::{Context, Poll}, time::Instant};
use hyper::{ Request, Response};
use tower::{Layer, Service};
use std::time::Duration;
use hyper::StatusCode;
use crate::pkg::metrics::metrics;
use crate::logger;
use std::fmt;

#[derive(Debug, Default, Clone)]
pub struct GrpcMiddlewareLayer;

impl<S> Layer<S> for GrpcMiddlewareLayer {
    type Service = GrpcMiddlewareService<S>;

    fn layer(&self, service: S) -> Self::Service {
        GrpcMiddlewareService { inner : service }
    }
}

#[derive(Debug, Clone)]
pub struct GrpcMiddlewareService<S> {
    inner : S,
}

type BoxFuture<'a, T> = Pin<Box<dyn Future<Output = T> + Send + 'a>>;

impl<S, ReqBody, ResBody> Service<Request<ReqBody>> for GrpcMiddlewareService<S> 
where
    S: Service<Request<ReqBody>, Response = Response<ResBody>> + Clone + Send + 'static,
    S::Future: Send + 'static,
    ReqBody: Send + 'static,
    S::Error: fmt::Display + fmt::Debug + Send + Sync + 'static,
{
    type Response = S::Response;
    type Error = S::Error;
    type Future = BoxFuture<'static, Result<Self::Response, Self::Error>>;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.inner.poll_ready(cx)
    }

    fn call(&mut self, req: Request<ReqBody>) -> Self::Future
    {
        let clone = self.inner.clone();
        let mut inner = replace(&mut self.inner, clone);

        Box::pin(async move {
            let start = Instant::now();
            let method = req.method().clone();
                        
            let response = inner.call(req).await;
            
            let status_code = match &response {
                Ok(resp) => resp.status(),
                Err(_) => StatusCode::INTERNAL_SERVER_ERROR,
            };
        
            let duration = start.elapsed();

            

            let response = match response {
                Ok(resp) => {
                    resp
                },
                Err(e) => {
                    logger::error(
                        format!("Request failed: method={}, status={}, duration={:?}, error={}", 
                            method, status_code, duration, e), 
                        None
                    );
                    return Err(e);
                }
            };

            telemetry_middleware(method.as_str(), duration, status_code);

            Ok(response)
        })
    }
}

fn telemetry_middleware(method: &str, duration: Duration, status_code: StatusCode)  {
    let tags = vec![("api", method), ("status", status_code.as_str())];
    
    let _ = metrics::timing("router.api.request.latency", duration, &tags);
    let _ = metrics::count("router.api.request.total", 1, &tags);
}