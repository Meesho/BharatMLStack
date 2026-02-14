package inferflow

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/components"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	pb "github.com/Meesho/BharatMLStack/inferflow/server/grpc/predict"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PredictService implements the Predict gRPC service with PointWise, PairWise, and SlateWise RPCs.
// It reuses the existing DAG executor and component pipeline via thin adapters.
type PredictService struct {
	pb.UnimplementedPredictServer
}

func (s *PredictService) InferPointWise(ctx context.Context, req *pb.PointWiseRequest) (*pb.PointWiseResponse, error) {
	startTime := time.Now()
	tags := []string{"model-id", req.ModelConfigId, "inference-mode", "pointwise"}
	metrics.Count("predict.infer.request.total", 1, tags)

	conf, err := config.GetModelConfig(req.ModelConfigId)
	if err != nil {
		logger.Error(fmt.Sprintf("InferPointWise: config not found for %s", req.ModelConfigId), err)
		return &pb.PointWiseResponse{RequestError: err.Error()}, status.Error(codes.InvalidArgument, err.Error())
	}

	headers := getAllHeaders(ctx)
	componentReq, err := adaptPointWiseRequest(req, conf, headers)
	if err != nil {
		logger.Error(fmt.Sprintf("InferPointWise: adapter error for %s", req.ModelConfigId), err)
		return &pb.PointWiseResponse{RequestError: err.Error()}, status.Error(codes.InvalidArgument, err.Error())
	}

	executor.Execute(conf.DAGExecutionConfig.ComponentDependency, *componentReq)

	resp := buildPointWiseResponse(componentReq.ComponentData, conf)
	maybeLogInferenceResponse(ctx, req.TrackingId, conf, componentReq)

	metrics.Timing("predict.infer.request.latency", time.Since(startTime), tags)
	metrics.Count("predict.infer.request.batch.size", int64(len(req.Targets)), tags)

	return resp, nil
}

func (s *PredictService) InferPairWise(ctx context.Context, req *pb.PairWiseRequest) (*pb.PairWiseResponse, error) {
	startTime := time.Now()
	tags := []string{"model-id", req.ModelConfigId, "inference-mode", "pairwise"}
	metrics.Count("predict.infer.request.total", 1, tags)

	conf, err := config.GetModelConfig(req.ModelConfigId)
	if err != nil {
		logger.Error(fmt.Sprintf("InferPairWise: config not found for %s", req.ModelConfigId), err)
		return &pb.PairWiseResponse{RequestError: err.Error()}, status.Error(codes.InvalidArgument, err.Error())
	}

	headers := getAllHeaders(ctx)
	componentReq, err := adaptPairWiseRequest(req, conf, headers)
	if err != nil {
		logger.Error(fmt.Sprintf("InferPairWise: adapter error for %s", req.ModelConfigId), err)
		return &pb.PairWiseResponse{RequestError: err.Error()}, status.Error(codes.InvalidArgument, err.Error())
	}

	executor.Execute(conf.DAGExecutionConfig.ComponentDependency, *componentReq)

	resp := buildPairWiseResponse(componentReq.ComponentData, componentReq.SlateData, conf)
	maybeLogInferenceResponse(ctx, req.TrackingId, conf, componentReq)

	metrics.Timing("predict.infer.request.latency", time.Since(startTime), tags)
	metrics.Count("predict.infer.request.batch.size", int64(len(req.Pairs)), tags)

	return resp, nil
}

func (s *PredictService) InferSlateWise(ctx context.Context, req *pb.SlateWiseRequest) (*pb.SlateWiseResponse, error) {
	startTime := time.Now()
	tags := []string{"model-id", req.ModelConfigId, "inference-mode", "slatewise"}
	metrics.Count("predict.infer.request.total", 1, tags)

	conf, err := config.GetModelConfig(req.ModelConfigId)
	if err != nil {
		logger.Error(fmt.Sprintf("InferSlateWise: config not found for %s", req.ModelConfigId), err)
		return &pb.SlateWiseResponse{RequestError: err.Error()}, status.Error(codes.InvalidArgument, err.Error())
	}

	headers := getAllHeaders(ctx)
	componentReq, err := adaptSlateWiseRequest(req, conf, headers)
	if err != nil {
		logger.Error(fmt.Sprintf("InferSlateWise: adapter error for %s", req.ModelConfigId), err)
		return &pb.SlateWiseResponse{RequestError: err.Error()}, status.Error(codes.InvalidArgument, err.Error())
	}

	executor.Execute(conf.DAGExecutionConfig.ComponentDependency, *componentReq)

	resp := buildSlateWiseResponse(componentReq.ComponentData, componentReq.SlateData, conf)
	maybeLogInferenceResponse(ctx, req.TrackingId, conf, componentReq)

	metrics.Timing("predict.infer.request.latency", time.Since(startTime), tags)
	metrics.Count("predict.infer.request.batch.size", int64(len(req.Slates)), tags)

	return resp, nil
}

func maybeLogInferenceResponse(ctx context.Context, trackingID string, conf *config.Config, componentReq *components.ComponentRequest) {
	// Inference logging: log response features based on config
	if conf.ResponseConfig.LoggingPerc > 0 &&
		rand.Intn(100)+1 <= conf.ResponseConfig.LoggingPerc &&
		trackingID != "" {
		modelConfigMap := config.GetModelConfigMap()
		// V2 logging: route based on configured format type
		switch modelConfigMap.ServiceConfig.V2LoggingType {
		case "proto":
			go logInferflowResponseBytes(ctx, "", trackingID, conf, componentReq)
		case "arrow":
			go logInferflowResponseArrow(ctx, "", trackingID, conf, componentReq)
		case "parquet":
			go logInferflowResponseParquet(ctx, "", trackingID, conf, componentReq)
		}
	}
}
