package com.bharatmlstack;

import com.bharatmlstack.persist.FeatureServiceGrpc;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.Metadata;
import io.grpc.stub.MetadataUtils;

public class BharatMLClient {

    private final ManagedChannel channel;
    private final com.bharatmlstack.persist.FeatureServiceGrpc.FeatureServiceBlockingStub persistClient;
    private final com.bharatmlstack.retrieve.FeatureServiceGrpc.FeatureServiceBlockingStub retrieveClient;

    public BharatMLClient(String host, int port) {
        this.channel = ManagedChannelBuilder.forAddress(host, port)
                .usePlaintext()
                .build();
        
        // Create metadata with required headers
        Metadata metadata = new Metadata();
        Metadata.Key<String> authTokenKey = Metadata.Key.of("online-feature-store-auth-token", Metadata.ASCII_STRING_MARSHALLER);
        Metadata.Key<String> callerIdKey = Metadata.Key.of("online-feature-store-caller-id", Metadata.ASCII_STRING_MARSHALLER);
        metadata.put(authTokenKey, "atishay");
        metadata.put(callerIdKey, "test-3");
        
        this.persistClient = MetadataUtils.attachHeaders(
            com.bharatmlstack.persist.FeatureServiceGrpc.newBlockingStub(channel), metadata);
        this.retrieveClient = MetadataUtils.attachHeaders(
            com.bharatmlstack.retrieve.FeatureServiceGrpc.newBlockingStub(channel), metadata);
    }

    public void shutdown() throws InterruptedException {
        channel.shutdown().awaitTermination(5, java.util.concurrent.TimeUnit.SECONDS);
    }

    public com.bharatmlstack.retrieve.Result retrieveFeatures(com.bharatmlstack.retrieve.Query request) {
        return retrieveClient.retrieveFeatures(request);
    }

    public com.bharatmlstack.retrieve.DecodedResult retrieveDecodedFeatures(com.bharatmlstack.retrieve.Query request) {
        return retrieveClient.retrieveDecodedResult(request);
    }

    public com.bharatmlstack.persist.Result persistFeatures(com.bharatmlstack.persist.Query request) {
        return persistClient.persistFeatures(request);
    }
} 