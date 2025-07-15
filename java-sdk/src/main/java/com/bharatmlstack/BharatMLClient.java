package com.bharatmlstack;

import com.bharatmlstack.persist.FeatureServiceGrpc;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

public class BharatMLClient {

    private final ManagedChannel channel;
    private final com.bharatmlstack.persist.FeatureServiceGrpc.FeatureServiceBlockingStub persistClient;
    private final com.bharatmlstack.retrieve.FeatureServiceGrpc.FeatureServiceBlockingStub retrieveClient;

    public BharatMLClient(String host, int port) {
        this.channel = ManagedChannelBuilder.forAddress(host, port)
                .usePlaintext()
                .build();
        this.persistClient = com.bharatmlstack.persist.FeatureServiceGrpc.newBlockingStub(channel);
        this.retrieveClient = com.bharatmlstack.retrieve.FeatureServiceGrpc.newBlockingStub(channel);
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