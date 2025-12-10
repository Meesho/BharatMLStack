package com.bharatml.orchestrator.grpc.service;

import static io.grpc.MethodDescriptor.generateFullMethodName;

/**
 */
@javax.annotation.Generated(
    value = "by gRPC proto compiler (version 1.54.0)",
    comments = "Source: retrieve.proto")
@io.grpc.stub.annotations.GrpcGenerated
public final class FeatureServiceGrpc {

  private FeatureServiceGrpc() {}

  public static final String SERVICE_NAME = "retrieve.FeatureService";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<com.bharatml.orchestrator.grpc.service.RetrieveProto.Query,
      com.bharatml.orchestrator.grpc.service.RetrieveProto.Result> getRetrieveFeaturesMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "RetrieveFeatures",
      requestType = com.bharatml.orchestrator.grpc.service.RetrieveProto.Query.class,
      responseType = com.bharatml.orchestrator.grpc.service.RetrieveProto.Result.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.bharatml.orchestrator.grpc.service.RetrieveProto.Query,
      com.bharatml.orchestrator.grpc.service.RetrieveProto.Result> getRetrieveFeaturesMethod() {
    io.grpc.MethodDescriptor<com.bharatml.orchestrator.grpc.service.RetrieveProto.Query, com.bharatml.orchestrator.grpc.service.RetrieveProto.Result> getRetrieveFeaturesMethod;
    if ((getRetrieveFeaturesMethod = FeatureServiceGrpc.getRetrieveFeaturesMethod) == null) {
      synchronized (FeatureServiceGrpc.class) {
        if ((getRetrieveFeaturesMethod = FeatureServiceGrpc.getRetrieveFeaturesMethod) == null) {
          FeatureServiceGrpc.getRetrieveFeaturesMethod = getRetrieveFeaturesMethod =
              io.grpc.MethodDescriptor.<com.bharatml.orchestrator.grpc.service.RetrieveProto.Query, com.bharatml.orchestrator.grpc.service.RetrieveProto.Result>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "RetrieveFeatures"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.bharatml.orchestrator.grpc.service.RetrieveProto.Query.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.bharatml.orchestrator.grpc.service.RetrieveProto.Result.getDefaultInstance()))
              .setSchemaDescriptor(new FeatureServiceMethodDescriptorSupplier("RetrieveFeatures"))
              .build();
        }
      }
    }
    return getRetrieveFeaturesMethod;
  }

  private static volatile io.grpc.MethodDescriptor<com.bharatml.orchestrator.grpc.service.RetrieveProto.Query,
      com.bharatml.orchestrator.grpc.service.RetrieveProto.DecodedResult> getRetrieveDecodedResultMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "RetrieveDecodedResult",
      requestType = com.bharatml.orchestrator.grpc.service.RetrieveProto.Query.class,
      responseType = com.bharatml.orchestrator.grpc.service.RetrieveProto.DecodedResult.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.bharatml.orchestrator.grpc.service.RetrieveProto.Query,
      com.bharatml.orchestrator.grpc.service.RetrieveProto.DecodedResult> getRetrieveDecodedResultMethod() {
    io.grpc.MethodDescriptor<com.bharatml.orchestrator.grpc.service.RetrieveProto.Query, com.bharatml.orchestrator.grpc.service.RetrieveProto.DecodedResult> getRetrieveDecodedResultMethod;
    if ((getRetrieveDecodedResultMethod = FeatureServiceGrpc.getRetrieveDecodedResultMethod) == null) {
      synchronized (FeatureServiceGrpc.class) {
        if ((getRetrieveDecodedResultMethod = FeatureServiceGrpc.getRetrieveDecodedResultMethod) == null) {
          FeatureServiceGrpc.getRetrieveDecodedResultMethod = getRetrieveDecodedResultMethod =
              io.grpc.MethodDescriptor.<com.bharatml.orchestrator.grpc.service.RetrieveProto.Query, com.bharatml.orchestrator.grpc.service.RetrieveProto.DecodedResult>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "RetrieveDecodedResult"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.bharatml.orchestrator.grpc.service.RetrieveProto.Query.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.bharatml.orchestrator.grpc.service.RetrieveProto.DecodedResult.getDefaultInstance()))
              .setSchemaDescriptor(new FeatureServiceMethodDescriptorSupplier("RetrieveDecodedResult"))
              .build();
        }
      }
    }
    return getRetrieveDecodedResultMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static FeatureServiceStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<FeatureServiceStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<FeatureServiceStub>() {
        @java.lang.Override
        public FeatureServiceStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new FeatureServiceStub(channel, callOptions);
        }
      };
    return FeatureServiceStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static FeatureServiceBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<FeatureServiceBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<FeatureServiceBlockingStub>() {
        @java.lang.Override
        public FeatureServiceBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new FeatureServiceBlockingStub(channel, callOptions);
        }
      };
    return FeatureServiceBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static FeatureServiceFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<FeatureServiceFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<FeatureServiceFutureStub>() {
        @java.lang.Override
        public FeatureServiceFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new FeatureServiceFutureStub(channel, callOptions);
        }
      };
    return FeatureServiceFutureStub.newStub(factory, channel);
  }

  /**
   */
  public interface AsyncService {

    /**
     */
    default void retrieveFeatures(com.bharatml.orchestrator.grpc.service.RetrieveProto.Query request,
        io.grpc.stub.StreamObserver<com.bharatml.orchestrator.grpc.service.RetrieveProto.Result> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getRetrieveFeaturesMethod(), responseObserver);
    }

    /**
     */
    default void retrieveDecodedResult(com.bharatml.orchestrator.grpc.service.RetrieveProto.Query request,
        io.grpc.stub.StreamObserver<com.bharatml.orchestrator.grpc.service.RetrieveProto.DecodedResult> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getRetrieveDecodedResultMethod(), responseObserver);
    }
  }

  /**
   * Base class for the server implementation of the service FeatureService.
   */
  public static abstract class FeatureServiceImplBase
      implements io.grpc.BindableService, AsyncService {

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return FeatureServiceGrpc.bindService(this);
    }
  }

  /**
   * A stub to allow clients to do asynchronous rpc calls to service FeatureService.
   */
  public static final class FeatureServiceStub
      extends io.grpc.stub.AbstractAsyncStub<FeatureServiceStub> {
    private FeatureServiceStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected FeatureServiceStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new FeatureServiceStub(channel, callOptions);
    }

    /**
     */
    public void retrieveFeatures(com.bharatml.orchestrator.grpc.service.RetrieveProto.Query request,
        io.grpc.stub.StreamObserver<com.bharatml.orchestrator.grpc.service.RetrieveProto.Result> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getRetrieveFeaturesMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     */
    public void retrieveDecodedResult(com.bharatml.orchestrator.grpc.service.RetrieveProto.Query request,
        io.grpc.stub.StreamObserver<com.bharatml.orchestrator.grpc.service.RetrieveProto.DecodedResult> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getRetrieveDecodedResultMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * A stub to allow clients to do synchronous rpc calls to service FeatureService.
   */
  public static final class FeatureServiceBlockingStub
      extends io.grpc.stub.AbstractBlockingStub<FeatureServiceBlockingStub> {
    private FeatureServiceBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected FeatureServiceBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new FeatureServiceBlockingStub(channel, callOptions);
    }

    /**
     */
    public com.bharatml.orchestrator.grpc.service.RetrieveProto.Result retrieveFeatures(com.bharatml.orchestrator.grpc.service.RetrieveProto.Query request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getRetrieveFeaturesMethod(), getCallOptions(), request);
    }

    /**
     */
    public com.bharatml.orchestrator.grpc.service.RetrieveProto.DecodedResult retrieveDecodedResult(com.bharatml.orchestrator.grpc.service.RetrieveProto.Query request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getRetrieveDecodedResultMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do ListenableFuture-style rpc calls to service FeatureService.
   */
  public static final class FeatureServiceFutureStub
      extends io.grpc.stub.AbstractFutureStub<FeatureServiceFutureStub> {
    private FeatureServiceFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected FeatureServiceFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new FeatureServiceFutureStub(channel, callOptions);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<com.bharatml.orchestrator.grpc.service.RetrieveProto.Result> retrieveFeatures(
        com.bharatml.orchestrator.grpc.service.RetrieveProto.Query request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getRetrieveFeaturesMethod(), getCallOptions()), request);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<com.bharatml.orchestrator.grpc.service.RetrieveProto.DecodedResult> retrieveDecodedResult(
        com.bharatml.orchestrator.grpc.service.RetrieveProto.Query request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getRetrieveDecodedResultMethod(), getCallOptions()), request);
    }
  }

  private static final int METHODID_RETRIEVE_FEATURES = 0;
  private static final int METHODID_RETRIEVE_DECODED_RESULT = 1;

  private static final class MethodHandlers<Req, Resp> implements
      io.grpc.stub.ServerCalls.UnaryMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ServerStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ClientStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.BidiStreamingMethod<Req, Resp> {
    private final AsyncService serviceImpl;
    private final int methodId;

    MethodHandlers(AsyncService serviceImpl, int methodId) {
      this.serviceImpl = serviceImpl;
      this.methodId = methodId;
    }

    @java.lang.Override
    @java.lang.SuppressWarnings("unchecked")
    public void invoke(Req request, io.grpc.stub.StreamObserver<Resp> responseObserver) {
      switch (methodId) {
        case METHODID_RETRIEVE_FEATURES:
          serviceImpl.retrieveFeatures((com.bharatml.orchestrator.grpc.service.RetrieveProto.Query) request,
              (io.grpc.stub.StreamObserver<com.bharatml.orchestrator.grpc.service.RetrieveProto.Result>) responseObserver);
          break;
        case METHODID_RETRIEVE_DECODED_RESULT:
          serviceImpl.retrieveDecodedResult((com.bharatml.orchestrator.grpc.service.RetrieveProto.Query) request,
              (io.grpc.stub.StreamObserver<com.bharatml.orchestrator.grpc.service.RetrieveProto.DecodedResult>) responseObserver);
          break;
        default:
          throw new AssertionError();
      }
    }

    @java.lang.Override
    @java.lang.SuppressWarnings("unchecked")
    public io.grpc.stub.StreamObserver<Req> invoke(
        io.grpc.stub.StreamObserver<Resp> responseObserver) {
      switch (methodId) {
        default:
          throw new AssertionError();
      }
    }
  }

  public static final io.grpc.ServerServiceDefinition bindService(AsyncService service) {
    return io.grpc.ServerServiceDefinition.builder(getServiceDescriptor())
        .addMethod(
          getRetrieveFeaturesMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.bharatml.orchestrator.grpc.service.RetrieveProto.Query,
              com.bharatml.orchestrator.grpc.service.RetrieveProto.Result>(
                service, METHODID_RETRIEVE_FEATURES)))
        .addMethod(
          getRetrieveDecodedResultMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.bharatml.orchestrator.grpc.service.RetrieveProto.Query,
              com.bharatml.orchestrator.grpc.service.RetrieveProto.DecodedResult>(
                service, METHODID_RETRIEVE_DECODED_RESULT)))
        .build();
  }

  private static abstract class FeatureServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    FeatureServiceBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return com.bharatml.orchestrator.grpc.service.RetrieveProto.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("FeatureService");
    }
  }

  private static final class FeatureServiceFileDescriptorSupplier
      extends FeatureServiceBaseDescriptorSupplier {
    FeatureServiceFileDescriptorSupplier() {}
  }

  private static final class FeatureServiceMethodDescriptorSupplier
      extends FeatureServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final String methodName;

    FeatureServiceMethodDescriptorSupplier(String methodName) {
      this.methodName = methodName;
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.MethodDescriptor getMethodDescriptor() {
      return getServiceDescriptor().findMethodByName(methodName);
    }
  }

  private static volatile io.grpc.ServiceDescriptor serviceDescriptor;

  public static io.grpc.ServiceDescriptor getServiceDescriptor() {
    io.grpc.ServiceDescriptor result = serviceDescriptor;
    if (result == null) {
      synchronized (FeatureServiceGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new FeatureServiceFileDescriptorSupplier())
              .addMethod(getRetrieveFeaturesMethod())
              .addMethod(getRetrieveDecodedResultMethod())
              .build();
        }
      }
    }
    return result;
  }
}
