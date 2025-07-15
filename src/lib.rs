use self::persist::feature_service_client::FeatureServiceClient as PersistClient;
use self::retrieve::feature_service_client::FeatureServiceClient as RetrieveClient;
use tonic::transport::Channel;

pub mod persist {
    tonic::include_proto!("persist");
}

pub mod retrieve {
    tonic::include_proto!("retrieve");
}

pub struct BharatMLClient {
    persist_client: PersistClient<Channel>,
    retrieve_client: RetrieveClient<Channel>,
}

impl BharatMLClient {
    pub async fn new(host: &str, port: u16) -> Result<Self, tonic::transport::Error> {
        let addr = format!("http://{}:{}", host, port);
        let channel = Channel::from_shared(addr)
            .unwrap()
            .connect()
            .await?;

        let persist_client = PersistClient::new(channel.clone());
        let retrieve_client = RetrieveClient::new(channel);

        Ok(Self {
            persist_client,
            retrieve_client,
        })
    }

    pub async fn persist_features(&mut self, request: persist::Query) -> Result<tonic::Response<persist::Result>, tonic::Status> {
        self.persist_client.persist_features(request).await
    }

    pub async fn retrieve_features(&mut self, request: retrieve::Query) -> Result<tonic::Response<retrieve::Result>, tonic::Status> {
        self.retrieve_client.retrieve_features(request).await
    }

    pub async fn retrieve_decoded_features(&mut self, request: retrieve::Query) -> Result<tonic::Response<retrieve::DecodedResult>, tonic::Status> {
        self.retrieve_client.retrieve_decoded_result(request).await
    }
} 