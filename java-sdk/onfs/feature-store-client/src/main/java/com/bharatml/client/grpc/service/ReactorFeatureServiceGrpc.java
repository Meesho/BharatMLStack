package com.bharatml.client.grpc.service;

import static com.bharatml.client.grpc.service.FeatureServiceGrpc.getServiceDescriptor;
import static io.grpc.stub.ServerCalls.asyncUnaryCall;
import static io.grpc.stub.ServerCalls.asyncServerStreamingCall;
import static io.grpc.stub.ServerCalls.asyncClientStreamingCall;
import static io.grpc.stub.ServerCalls.asyncBidiStreamingCall;


@javax.annotation.Generated(
value = "by ReactorGrpc generator",
comments = "Source: retrieve.proto")
public final class ReactorFeatureServiceGrpc {
    private ReactorFeatureServiceGrpc() {}

    public static ReactorFeatureServiceStub newReactorStub(io.grpc.Channel channel) {
        return new ReactorFeatureServiceStub(channel);
    }

    public static final class ReactorFeatureServiceStub extends io.grpc.stub.AbstractStub<ReactorFeatureServiceStub> {
        private FeatureServiceGrpc.FeatureServiceStub delegateStub;

        private ReactorFeatureServiceStub(io.grpc.Channel channel) {
            super(channel);
            delegateStub = FeatureServiceGrpc.newStub(channel);
        }

        private ReactorFeatureServiceStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
            super(channel, callOptions);
            delegateStub = FeatureServiceGrpc.newStub(channel).build(channel, callOptions);
        }

        @Override
        protected ReactorFeatureServiceStub build(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
            return new ReactorFeatureServiceStub(channel, callOptions);
        }

        public reactor.core.publisher.Mono<com.bharatml.client.grpc.service.RetrieveProto.Result> retrieveFeatures(reactor.core.publisher.Mono<com.bharatml.client.grpc.service.RetrieveProto.Query> reactorRequest) {
            return com.salesforce.reactorgrpc.stub.ClientCalls.oneToOne(reactorRequest, delegateStub::retrieveFeatures, getCallOptions());
        }

        public reactor.core.publisher.Mono<com.bharatml.client.grpc.service.RetrieveProto.DecodedResult> retrieveDecodedResult(reactor.core.publisher.Mono<com.bharatml.client.grpc.service.RetrieveProto.Query> reactorRequest) {
            return com.salesforce.reactorgrpc.stub.ClientCalls.oneToOne(reactorRequest, delegateStub::retrieveDecodedResult, getCallOptions());
        }

        public reactor.core.publisher.Mono<com.bharatml.client.grpc.service.RetrieveProto.Result> retrieveFeatures(com.bharatml.client.grpc.service.RetrieveProto.Query reactorRequest) {
           return com.salesforce.reactorgrpc.stub.ClientCalls.oneToOne(reactor.core.publisher.Mono.just(reactorRequest), delegateStub::retrieveFeatures, getCallOptions());
        }

        public reactor.core.publisher.Mono<com.bharatml.client.grpc.service.RetrieveProto.DecodedResult> retrieveDecodedResult(com.bharatml.client.grpc.service.RetrieveProto.Query reactorRequest) {
           return com.salesforce.reactorgrpc.stub.ClientCalls.oneToOne(reactor.core.publisher.Mono.just(reactorRequest), delegateStub::retrieveDecodedResult, getCallOptions());
        }

    }

    public static abstract class FeatureServiceImplBase implements io.grpc.BindableService {

        public reactor.core.publisher.Mono<com.bharatml.client.grpc.service.RetrieveProto.Result> retrieveFeatures(com.bharatml.client.grpc.service.RetrieveProto.Query request) {
            return retrieveFeatures(reactor.core.publisher.Mono.just(request));
        }

        public reactor.core.publisher.Mono<com.bharatml.client.grpc.service.RetrieveProto.Result> retrieveFeatures(reactor.core.publisher.Mono<com.bharatml.client.grpc.service.RetrieveProto.Query> request) {
            throw new io.grpc.StatusRuntimeException(io.grpc.Status.UNIMPLEMENTED);
        }

        public reactor.core.publisher.Mono<com.bharatml.client.grpc.service.RetrieveProto.DecodedResult> retrieveDecodedResult(com.bharatml.client.grpc.service.RetrieveProto.Query request) {
            return retrieveDecodedResult(reactor.core.publisher.Mono.just(request));
        }

        public reactor.core.publisher.Mono<com.bharatml.client.grpc.service.RetrieveProto.DecodedResult> retrieveDecodedResult(reactor.core.publisher.Mono<com.bharatml.client.grpc.service.RetrieveProto.Query> request) {
            throw new io.grpc.StatusRuntimeException(io.grpc.Status.UNIMPLEMENTED);
        }

        @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
            return io.grpc.ServerServiceDefinition.builder(getServiceDescriptor())
                    .addMethod(
                            com.bharatml.client.grpc.service.FeatureServiceGrpc.getRetrieveFeaturesMethod(),
                            asyncUnaryCall(
                                    new MethodHandlers<
                                            com.bharatml.client.grpc.service.RetrieveProto.Query,
                                            com.bharatml.client.grpc.service.RetrieveProto.Result>(
                                            this, METHODID_RETRIEVE_FEATURES)))
                    .addMethod(
                            com.bharatml.client.grpc.service.FeatureServiceGrpc.getRetrieveDecodedResultMethod(),
                            asyncUnaryCall(
                                    new MethodHandlers<
                                            com.bharatml.client.grpc.service.RetrieveProto.Query,
                                            com.bharatml.client.grpc.service.RetrieveProto.DecodedResult>(
                                            this, METHODID_RETRIEVE_DECODED_RESULT)))
                    .build();
        }

        protected io.grpc.CallOptions getCallOptions(int methodId) {
            return null;
        }

        protected Throwable onErrorMap(Throwable throwable) {
            return com.salesforce.reactorgrpc.stub.ServerCalls.prepareError(throwable);
        }
    }

    public static final int METHODID_RETRIEVE_FEATURES = 0;
    public static final int METHODID_RETRIEVE_DECODED_RESULT = 1;

    private static final class MethodHandlers<Req, Resp> implements
            io.grpc.stub.ServerCalls.UnaryMethod<Req, Resp>,
            io.grpc.stub.ServerCalls.ServerStreamingMethod<Req, Resp>,
            io.grpc.stub.ServerCalls.ClientStreamingMethod<Req, Resp>,
            io.grpc.stub.ServerCalls.BidiStreamingMethod<Req, Resp> {
        private final FeatureServiceImplBase serviceImpl;
        private final int methodId;

        MethodHandlers(FeatureServiceImplBase serviceImpl, int methodId) {
            this.serviceImpl = serviceImpl;
            this.methodId = methodId;
        }

        @java.lang.Override
        @java.lang.SuppressWarnings("unchecked")
        public void invoke(Req request, io.grpc.stub.StreamObserver<Resp> responseObserver) {
            switch (methodId) {
                case METHODID_RETRIEVE_FEATURES:
                    com.salesforce.reactorgrpc.stub.ServerCalls.oneToOne((com.bharatml.client.grpc.service.RetrieveProto.Query) request,
                            (io.grpc.stub.StreamObserver<com.bharatml.client.grpc.service.RetrieveProto.Result>) responseObserver,
                            serviceImpl::retrieveFeatures, serviceImpl::onErrorMap);
                    break;
                case METHODID_RETRIEVE_DECODED_RESULT:
                    com.salesforce.reactorgrpc.stub.ServerCalls.oneToOne((com.bharatml.client.grpc.service.RetrieveProto.Query) request,
                            (io.grpc.stub.StreamObserver<com.bharatml.client.grpc.service.RetrieveProto.DecodedResult>) responseObserver,
                            serviceImpl::retrieveDecodedResult, serviceImpl::onErrorMap);
                    break;
                default:
                    throw new java.lang.AssertionError();
            }
        }

        @java.lang.Override
        @java.lang.SuppressWarnings("unchecked")
        public io.grpc.stub.StreamObserver<Req> invoke(io.grpc.stub.StreamObserver<Resp> responseObserver) {
            switch (methodId) {
                default:
                    throw new java.lang.AssertionError();
            }
        }
    }

}
