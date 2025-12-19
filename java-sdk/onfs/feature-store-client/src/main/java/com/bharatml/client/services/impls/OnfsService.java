package com.bharatml.client.services.impls;

import com.bharatml.client.adaptor.ProtoConvertor;
import com.bharatml.client.dtos.DecodedResult;
import com.bharatml.client.dtos.Query;
import com.bharatml.client.dtos.Result;
import com.bharatml.client.services.IOnfsService;
import com.bharatml.client.grpc.service.RetrieveProto;
import com.bharatml.client.grpc.service.FeatureServiceGrpc;
import com.bharatml.client.grpc.config.OnfsClientConfig;
import io.grpc.ManagedChannel;
import io.grpc.Metadata;
import io.grpc.StatusRuntimeException;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.stereotype.Service;
import static com.bharatml.client.Constants.ONFS_GRPC_CHANNEL_TEMPLATE;
import static com.bharatml.client.Constants.ONFS_GRPC_PROPERTIES;
import lombok.extern.slf4j.Slf4j;


@Service
@Slf4j
public class OnfsService implements IOnfsService {

    private final ManagedChannel managedChannel;
    private final OnfsClientConfig onfsClientConfig;

    public OnfsService(
            @Qualifier(ONFS_GRPC_CHANNEL_TEMPLATE) ManagedChannel managedChannel,
            @Qualifier(ONFS_GRPC_PROPERTIES) OnfsClientConfig onfsClientConfig) {
        this.managedChannel = managedChannel;
        this.onfsClientConfig = onfsClientConfig;
    }

    @Override
    public Result retrieveFeatures(Query request) {
        return retrieveFeatures(null, request);
    }

    @Override
    public Result retrieveFeatures(Metadata metadata, Query request) {
        RetrieveProto.Query queryProto = ProtoConvertor.convertToQueryProto(request);
        FeatureServiceGrpc.FeatureServiceBlockingStub stub = FeatureServiceGrpc.newBlockingStub(managedChannel);
        stub = stub.withDeadlineAfter(onfsClientConfig.getHttp2Config().getGrpcDeadline(), java.util.concurrent.TimeUnit.MILLISECONDS);
        if (metadata != null) {
            stub = stub.withInterceptors(io.grpc.stub.MetadataUtils.newAttachHeadersInterceptor(metadata));
        }
        try {
            return ProtoConvertor.convertToResult(stub.retrieveFeatures(queryProto));
        } catch (StatusRuntimeException e) {
            log.error("gRPC call to retrieveFeatures failed: status={}, description={}, entityLabel={}",
                    e.getStatus().getCode(), e.getStatus().getDescription(),
                    request != null ? request.getEntityLabel() : "unknown", e);
            throw e;
        }
    }

    @Override
    public DecodedResult retrieveDecodedResult(Query request) {
        return retrieveDecodedResult(null, request);
    }

    @Override
    public DecodedResult retrieveDecodedResult(Metadata metadata, Query request) {
        RetrieveProto.Query queryProto = ProtoConvertor.convertToQueryProto(request);
        FeatureServiceGrpc.FeatureServiceBlockingStub stub = FeatureServiceGrpc.newBlockingStub(managedChannel);
        stub = stub.withDeadlineAfter(onfsClientConfig.getHttp2Config().getGrpcDeadline(), java.util.concurrent.TimeUnit.MILLISECONDS);
        if (metadata != null) {
            stub = stub.withInterceptors(io.grpc.stub.MetadataUtils.newAttachHeadersInterceptor(metadata));
        }
        try {
            return ProtoConvertor.convertToDecodedResult(stub.retrieveDecodedResult(queryProto));
        } catch (StatusRuntimeException e) {
            log.error("gRPC call to retrieveDecodedResult failed: status={}, description={}, entityLabel={}",
                    e.getStatus().getCode(), e.getStatus().getDescription(),
                    request != null ? request.getEntityLabel() : "unknown", e);
            throw e;
        }
    }
} 