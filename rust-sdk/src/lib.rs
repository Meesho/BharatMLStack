pub mod persist {
    tonic::include_proto!("persist");
}

pub mod retrieve {
    tonic::include_proto!("retrieve");
}

use persist::feature_service_client::FeatureServiceClient as PersistClient;
use retrieve::feature_service_client::FeatureServiceClient as RetrieveClient;
use tonic::transport::Channel;

pub struct BharatMLClient {
    persist_client: PersistClient<Channel>,
    retrieve_client: RetrieveClient<Channel>,
}

impl BharatMLClient {
    pub async fn connect<D>(dst: D) -> Result<Self, tonic::transport::Error>
    where
        D: TryInto<tonic::transport::Endpoint>,
        D::Error: Into<tonic::codegen::StdError>,
    {
        let channel = tonic::transport::Endpoint::new(dst)?.connect().await?;
        let persist_client = PersistClient::new(channel.clone());
        let retrieve_client = RetrieveClient::new(channel);

        Ok(Self {
            persist_client,
            retrieve_client,
        })
    }

    pub async fn persist_features(
        &mut self,
        request: impl tonic::IntoRequest<persist::Query>,
    ) -> Result<tonic::Response<persist::Result>, tonic::Status> {
        self.persist_client.persist_features(request).await
    }

    pub async fn retrieve_features(
        &mut self,
        request: impl tonic::IntoRequest<retrieve::Query>,
    ) -> Result<tonic::Response<retrieve::Result>, tonic::Status> {
        self.retrieve_client.retrieve_features(request).await
    }
}
