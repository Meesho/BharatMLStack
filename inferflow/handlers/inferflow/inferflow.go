//go:build !meesho

package inferflow

import (
	"context"
	"fmt"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/etcd"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/matrix"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"

	"github.com/Meesho/BharatMLStack/inferflow/dag-topology-executor/handlers/dag"
	extCache "github.com/Meesho/BharatMLStack/inferflow/dag-topology-executor/pkg/cache"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/components"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/internal/errors"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/utils"
	pb "github.com/Meesho/BharatMLStack/inferflow/server/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	executor   *dag.Executor
	headersMap = map[string]string{
		UserId:         userId,
		appVersionCode: appVersion,
	}
	componentProvider *ComponentProviderHandler
)

const (
	UserId         = "USER-ID"
	appVersionCode = "APP-VERSION-CODE"
	appVersion     = "app_version_code"
	defaultErrMsg  = "something went wrong!"
	userId         = "user_id"
	modelConfigId  = "model-config-id"
)

func InitInferflowHandler(configs *configs.AppConfigs) {

	componentProvider = &ComponentProviderHandler{
		componentMap: make(map[string]dag.AbstractComponent, 0),
	}

	InferflowConfig := etcd.Instance().GetConfigInstance().(*config.ModelConfig)
	config.SetModelConfigMap(InferflowConfig)

	// register components in config map
	componentProvider.RegisterComponent(config.GetModelConfigMap())

	// initializes dag cache which caches dag config
	cache := extCache.InitRistrettoCache(configs.Configs.DagTopologyCacheSize, configs.Configs.DagTopologyCacheTTL)
	executor = &dag.Executor{
		Registry: &dag.ComponentGraphRegistry{
			Cache: cache,
			Initializer: &dag.ComponentInitializer{
				ComponentProvider: componentProvider,
			},
		},
	}
	logger.Info("Inferflow handler initialized")
}

func ReloadModelConfigMapAndRegisterComponents() error {
	updatedConfig, ok := etcd.Instance().GetConfigInstance().(*config.ModelConfig)
	if ok {
		config.SetModelConfigMap(updatedConfig)

		// register components in config map
		componentProvider.RegisterComponent(updatedConfig)

		return nil
	}
	return &errors.ParsingError{ErrorMsg: "failed to parse model config from etcd"}
}

func (s *Inferflow) RetrieveModelScore(ctx context.Context, req *pb.InferflowRequestProto) (*pb.InferflowResponseProto, error) {
	startTime := time.Now()
	conf, err := config.GetModelConfig(req.ModelConfigId)
	if err != nil {
		errMsg := getErrMsg(err)
		return processInferflowResponse(nil, &errMsg), status.Error(codes.InvalidArgument, errMsg)
	}

	tags := []string{"model-id", req.ModelConfigId}
	metrics.Count("inferflow.retrievemodelscore.request.total", 1, tags)
	metrics.Count("inferflow.retrievemodelscore.request.batch.size", int64(len(req.Entities[0].Ids)), tags)

	InferflowReq, err := createInferflowRequestProto(ctx, req)
	if err != nil {
		errMsg := getErrMsg(err)
		return processInferflowResponse(nil, &errMsg), status.Error(codes.InvalidArgument, errMsg)
	}

	headers := getAllHeaders(ctx)
	headers[modelConfigId] = req.ModelConfigId
	componentReq := components.ComponentRequest{
		ComponentData:   &matrix.ComponentMatrix{},
		Entities:        InferflowReq.Entity,
		EntityIds:       InferflowReq.EntityIds,
		Features:        InferflowReq.Features,
		ComponentConfig: &conf.ComponentConfig,
		ModelId:         req.ModelConfigId,
		Headers:         headers,
	}

	// get userId
	var userId string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		userIdList := md.Get(UserId)
		if len(userIdList) > 0 {
			userId = userIdList[0]
		}
	}
	executor.Execute(conf.DAGExecutionConfig.ComponentDependency, componentReq)
	InferflowRes := prepareInferflowResponse(userId, conf, &componentReq)

	res := processInferflowResponse(InferflowRes, nil)

	responseTime := time.Since(startTime)
	metrics.Timing("inferflow.retrievemodelscore.request.latency", responseTime, tags)

	return res, nil
}

func getAllHeaders(ctx context.Context) map[string]string {
	hMap := make(map[string]string, 0)
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		for headerKey := range headersMap {
			headerValues := md.Get(headerKey)
			if len(headerValues) > 0 {
				hMap[headerKey] = headerValues[0]
			}
		}
	}
	return hMap
}

func createInferflowRequestProto(ctx context.Context, protoReq *pb.InferflowRequestProto) (*InferflowRequest, error) {

	featureMap := make(map[string][]string, 0)
	entities := make([]string, 0)
	entityIds := make([][]string, 0)
	uniqueMap := make(map[string]bool)
	for _, protoEntities := range protoReq.Entities {
		if _, exists := uniqueMap[protoEntities.Entity]; !exists {
			uniqueMap[protoEntities.Entity] = true
			entities = append(entities, protoEntities.Entity)
			entityIds = append(entityIds, protoEntities.Ids)
		}
		// Add default features if present
		if len(protoEntities.Features) > 0 {
			for _, feature := range protoEntities.Features {
				if _, exists := uniqueMap[feature.GetName()]; !exists {
					featureMap[feature.GetName()] = feature.GetIdsFeatureValue()
					uniqueMap[feature.GetName()] = true
				}
			}
		}
	}

	populateEntitiesFromHeaders(ctx, &entities, &entityIds, &uniqueMap)

	InferflowReq := &InferflowRequest{
		Entity:        &entities,
		EntityIds:     &entityIds,
		Features:      &featureMap,
		ModelConfigId: protoReq.ModelConfigId,
		TrackingId:    protoReq.TrackingId,
	}

	isValid := validateInferflowRequest(InferflowReq)
	if !isValid {
		logger.Error(fmt.Sprintf("Invalid Inferflow request %v ", InferflowReq), nil)
		return nil, &errors.RequestError{ErrorMsg: fmt.Sprintf("Invalid Inferflow request %v", InferflowReq)}
	}
	return InferflowReq, nil
}

func populateEntitiesFromHeaders(ctx context.Context, entities *[]string, entityIds *[][]string, uniqueMap *map[string]bool) {

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {

		// Add  entities from headers
		for mdKey, entityKey := range headersMap {

			values := md.Get(mdKey)
			if len(values) > 0 {
				if _, exists := (*uniqueMap)[entityKey]; !exists {
					*entities = append(*entities, entityKey)
					*entityIds = append(*entityIds, values)
					(*uniqueMap)[entityKey] = true
				}
			}
		}
	}
}

func processInferflowResponse(response *InferflowResponse, errMsg *string) *pb.InferflowResponseProto {

	protoResp := &pb.InferflowResponseProto{}

	if response != nil {
		for _, row := range response.ComponentData {
			protoRow := &pb.InferflowResponseProto_ComponentData{Data: row}
			protoResp.ComponentData = append(protoResp.ComponentData, protoRow)
		}
	}
	if errMsg != nil {
		protoResp.Error = &pb.InferflowResponseProto_Error{
			Message: *errMsg,
		}
	}
	return protoResp
}

func prepareInferflowResponse(userId string, c *config.Config, req *components.ComponentRequest) *InferflowResponse {
	var responseMatrix [][]string
	resConfig := c.ResponseConfig
	if !utils.IsNilOrEmpty(resConfig) {
		matrixUtil := *req.ComponentData
		var features []string
		modelSchemaEnabled := utils.IsEnableForUserForToday(userId, resConfig.ModelSchemaPerc)
		if modelSchemaEnabled {
			features = getRankerModelSchema(c)
		} else {
			features = resConfig.Features
		}
		responseMatrix = matrixUtil.GetMatrixOfColumnSliceInString(features)
	}
	InferflowRes := &InferflowResponse{
		ComponentData: responseMatrix,
	}

	return InferflowRes
}

func getErrMsg(err error) string {

	errMsg := defaultErrMsg
	if cast, ok := err.(*errors.RequestError); ok {
		errMsg = cast.ErrorMsg
	}
	return errMsg
}

func getRankerModelSchema(c *config.Config) []string {

	rFeatures := make([]string, 0)
	rFeaturesMap := make(map[string]bool)
	for _, iRankerComp := range c.ComponentConfig.PredatorComponentConfig.Values() {
		if rankerComp, ok := iRankerComp.(config.PredatorComponentConfig); ok {
			for _, in := range rankerComp.Inputs {
				for _, feature := range in.Features {
					if !rFeaturesMap[feature] {
						rFeatures = append(rFeatures, feature)
						rFeaturesMap[feature] = true
					}
				}
			}
		}
	}

	if len(c.ResponseConfig.Features) > 0 {
		for _, selectiveFeature := range c.ResponseConfig.Features {
			if !rFeaturesMap[selectiveFeature] {
				rFeatures = append(rFeatures, selectiveFeature)
			}
		}
	}
	return rFeatures
}
