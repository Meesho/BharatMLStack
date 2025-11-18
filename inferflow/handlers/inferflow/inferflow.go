package inferflow

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/etcd"
	"github.com/rs/zerolog/log"

	extPrism "github.com/Meesho/BharatMLStack/inferflow/handlers/external/prism"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/inmemorycache"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/matrix"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	convertor "github.com/Meesho/go-core/datatypeconverter/typeconverter"

	"strings"

	pb "github.com/Meesho/BharatMLStack/inferflow/client/grpc"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/components"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	kv "github.com/Meesho/BharatMLStack/inferflow/handlers/external/keyvalue"
	kvProto "github.com/Meesho/BharatMLStack/inferflow/handlers/external/keyvalue/proto/kvstore"
	"github.com/Meesho/BharatMLStack/inferflow/internal/errors"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/utils"
	"github.com/Meesho/dag-topology-executor/handlers/dag"
	extCache "github.com/Meesho/dag-topology-executor/pkg/cache"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	executor   *dag.Executor
	headersMap = map[string]string{
		meeshoUserId:          userId,
		meeshoParentCatalogId: parentCatalogId,
		meeshoStateCode:       stateCode,
		meeshoParentUserId:    parentUserId,
		meeshoClpId:           clpId,
		meeshoCollectionId:    collectionId,
		meeshoSearchQuery:     searchQuery,
		meeshoUserPincode:     pincode,
		meeshoRealEstate:      realEstate,
		meeshoUserContext:     userContext,
		appVersionCode:        appVersion,
		meeshoClientId:        clientId,
		meeshoCPCWeight:       cpcWeight,
		meeshoMinCpc:          meeshoMinCpc,
		meeshoMaxCpc:          meeshoMaxCpc,
	}
	componentProvider *ComponentProviderHandler
)

const (
	configPrefix                       = "dagTopologyCache"
	meeshoUserId                       = "MEESHO-USER-ID"
	meeshoParentCatalogId              = "MEESHO-PARENT-CATALOG-ID"
	meeshoParentUserId                 = "MEESHO-PARENT-USER-ID"
	meeshoClpId                        = "MEESHO-CLP-ID"
	meeshoCollectionId                 = "MEESHO-COLLECTION-ID"
	meeshoStateCode                    = "MEESHO-USER-STATE-CODE"
	meeshoUserPincode                  = "MEESHO-USER-pincode"
	meeshoSearchQuery                  = "MEESHO-SEARCH-QUERY"
	MEESHO_SEARCH_RELEVANCE_SCORE_DATE = "MEESHO-SEARCH-RELEVANCE-SCORE-DATE"
	meeshoRealEstate                   = "REAL-ESTATE"
	meeshoUserContext                  = "MEESHO-USER-CONTEXT"
	appVersionCode                     = "APP-VERSION-CODE"
	meeshoClientId                     = "MEESHO-CLIENT-ID"
	meeshoCPCWeight                    = "CPC-WEIGHT"
	modelDefaultResponse               = "model_default_response"
	clientId                           = "client_id"
	userContext                        = "user_context"
	appVersion                         = "app_version_code"
	defaultErrMsg                      = "something went wrong!"
	userId                             = "user_id"
	parentCatalogId                    = "parent_catalog_id"
	parentUserId                       = "parent_user_id"
	clpId                              = "clp_id"
	collectionId                       = "collection_id"
	searchQuery                        = "search_query"
	pincode                            = "pincode"
	realEstate                         = "realestate"
	cpcWeight                          = "cpc_weight"
	meeshoMinCpc                       = "min_cpc"
	meeshoMaxCpc                       = "max_cpc"
	meeshoPrefix                       = "MEESHO-"
	keySeparator                       = ":"
	modelConfigId                      = "model-config-id"
	stateCode                          = "state_code"
)

func InitInferflowHandler(configs *configs.AppConfigs) {

	componentProvider = &ComponentProviderHandler{
		componentMap: make(map[string]dag.AbstractComponent, 0),
	}

	InferflowConfig := etcd.Instance().GetConfigInstance().(*config.ModelConfig)
	updateDrModelConfigId(InferflowConfig)
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
	logger.Info("Model Proxy handler initialized")
}

func ReloadModelConfigMapAndRegisterComponents() error {
	updatedConfig, ok := etcd.Instance().GetConfigInstance().(*config.ModelConfig)
	if ok {
		updateDrModelConfigId(updatedConfig)
		config.SetModelConfigMap(updatedConfig)

		// register components in config map
		componentProvider.RegisterComponent(updatedConfig)

		return nil
	}
	return &errors.ParsingError{ErrorMsg: "failed to parse model config from etcd"}
}

func updateDrModelConfigId(modelConfig *config.ModelConfig) {
	if modelConfig == nil || len(modelConfig.ConfigMap) == 0 {
		return
	}
	for key, conf := range modelConfig.ConfigMap {
		if !utils.IsNilOrEmpty(conf.DrConfig) && !utils.IsNilOrEmpty(conf.DrConfig.Keys) && conf.DrConfig.Keys.Id != "" && conf.DrConfig.Res != nil && len(conf.DrConfig.Res) > 0 && conf.DrConfig.Version > 0 {
			if conf.DrConfig.OverrideConfigKey != "" {
				conf.DrConfig.DrModelConfigId = hashKeyWithCRC32Split(conf.DrConfig.OverrideConfigKey)
			} else {
				conf.DrConfig.DrModelConfigId = hashKeyWithCRC32Split(key)
			}
		}
		modelConfig.ConfigMap[key] = conf
	}
}

func hashKeyWithCRC32Split(key string) string {
	hash := crc32.ChecksumIEEE([]byte(key))
	return fmt.Sprintf("%08x", hash)
}

func (s *Inferflow) RetrieveModelScore(ctx context.Context, req *pb.InferflowRequestProto) (*pb.InferflowResponseProto, error) {
	startTime := time.Now()
	conf, err := config.GetModelConfig(req.ModelConfigId)
	if err != nil {
		errMsg := getErrMsg(err)
		return processInferflowResponse(nil, &errMsg), status.Errorf(codes.InvalidArgument, errMsg)
	}

	tags := []string{"model-id", req.ModelConfigId}
	metrics.Count("inferflow.retrievemodelscore.request.total", 1, tags)
	metrics.Count("inferflow.retrievemodelscore.request.batch.size", int64(len(req.Entities[0].Ids)), tags)
	if !utils.IsNilOrEmpty(conf.DrConfig) && !utils.IsNilOrEmpty(conf.DrConfig.Keys) && conf.DrConfig.Keys.Id != "" && conf.DrConfig.Res != nil && len(conf.DrConfig.Res) > 0 && conf.DrConfig.Version > 0 {
		enabled, entityIdsIdx := isDrEnabled(req.Entities, conf.DrConfig.Keys.Id)
		if enabled && entityIdsIdx >= 0 {
			kvResponse := RetrieveModelScoreFromKVService(ctx, req, conf, entityIdsIdx)
			responseTime := time.Since(startTime)
			metrics.Timing("inferflow.retrievemodelscore.request.latency", responseTime, tags)
			return kvResponse, nil
		}
	}

	InferflowReq, err := createInferflowRequestProto(ctx, req)
	if err != nil {
		errMsg := getErrMsg(err)
		return processInferflowResponse(nil, &errMsg), status.Errorf(codes.InvalidArgument, errMsg)
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
		userIdList := md.Get(meeshoUserId)
		if len(userIdList) > 0 {
			userId = userIdList[0]
		}
	}
	executor.Execute(conf.DAGExecutionConfig.ComponentDependency, componentReq)
	InferflowRes := prepareInferflowResponse(userId, conf, &componentReq)

	if !utils.IsNilOrEmpty(conf.ResponseConfig) &&
		utils.IsEnableForUserForToday(userId, conf.ResponseConfig.LoggingPerc) &&
		InferflowReq.TrackingId != "" {
		go logInferflowResponse(ctx, userId, InferflowReq.TrackingId, conf, &componentReq)
	}

	res := processInferflowResponse(InferflowRes, nil)

	responseTime := time.Since(startTime)
	metrics.Timing("inferflow.retrievemodelscore.request.latency", responseTime, tags)

	return res, nil
}
func isDrEnabled(reqEntities []*pb.InferflowRequestProto_Entity, entityId string) (bool, int) {
	random := rand.Intn(100)
	entityIdsIdx := -1
	for i, protoEntities := range reqEntities {
		if entityId == protoEntities.Entity {
			entityIdsIdx = i
			break
		}
	}
	return random < config.GetModelConfigMap().DrPercent, entityIdsIdx
}
func RetrieveModelScoreFromKVService(ctx context.Context, req *pb.InferflowRequestProto, conf *config.Config, entityIdsIdx int) *pb.InferflowResponseProto {

	headers := getAllHeaders(ctx)
	entityIds := req.Entities[entityIdsIdx].Ids

	keys, headerMap := buildKVKeys(headers, entityIds, conf)
	reqEntitiesMap := buildRequestEntitiesMap(req.Entities)
	var wg sync.WaitGroup
	var kvResponse *kvProto.BatchGetResponse
	var defaultModelResponse *kvProto.GetResponse

	wg.Add(2)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Msgf("Recovered from panic in GetKVServiceBatchResponse: %v", r)
			}
			wg.Done()
		}()
		kvResponse = kv.GetKVServiceBatchResponse(keys)
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Msgf("Recovered from panic in getDefaultModelResponseWithCache: %v", r)
			}
			wg.Done()
		}()
		defaultModelResponse = getDefaultModelResponseWithCache(conf.DrConfig.DrModelConfigId, conf)
	}()

	wg.Wait()

	responseMatrix := buildResponseMatrix(conf, kvResponse, req.ModelConfigId, entityIds, defaultModelResponse, headerMap, reqEntitiesMap, entityIdsIdx)

	return processInferflowResponse(&InferflowResponse{
		ComponentData: responseMatrix,
	}, nil)
}

func buildRequestEntitiesMap(entities []*pb.InferflowRequestProto_Entity) map[string][]string {
	reqEntitiesMap := make(map[string][]string, 0)
	for _, protoEntities := range entities {
		reqEntitiesMap[protoEntities.Entity] = protoEntities.Ids

		for _, feature := range protoEntities.Features {
			if _, exists := reqEntitiesMap[feature.GetName()]; !exists {
				reqEntitiesMap[feature.GetName()] = feature.GetIdsFeatureValue()
			}
		}
	}
	return reqEntitiesMap
}
func buildKVKeys(headers map[string]string, entityIds []string, conf *config.Config) ([]string, map[string]string) {
	keys := make([]string, len(entityIds))
	key := conf.DrConfig.DrModelConfigId + keySeparator +
		strconv.Itoa(conf.DrConfig.Version) + keySeparator
	headerMap := make(map[string]string, len(headers))
	//remove meeshoPrefix from header key and then replace - to _ and convert to lower case
	for hKey, hValue := range headers {
		hKey = strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(hKey, meeshoPrefix), "-", "_"))
		headerMap[hKey] = hValue
	}

	for _, headerKey := range conf.DrConfig.Keys.HeaderKeys {
		if _, exists := headerMap[headerKey]; exists {
			key += headerMap[headerKey] + keySeparator
		}
	}

	for i, entityId := range entityIds {
		keys[i] = key + entityId
	}
	return keys, headerMap
}

func getDefaultModelResponseWithCache(modelConfigId string, conf *config.Config) *kvProto.GetResponse {
	key := modelConfigId + keySeparator + strconv.Itoa(conf.DrConfig.Version)
	cache := inmemorycache.InMemoryCacheInstance

	if blob, err := cache.Get([]byte(key)); err == nil {
		value := string(blob)
		return &kvProto.GetResponse{
			Found: true,
			Value: value,
		}
	}
	response, err := kv.GetKVServiceResponse(key)
	if err != nil {
		return nil
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Msgf("Recovered from panic in getDefaultModelResponseWithCache: %v", r)
			}
		}()
		_ = cache.SetEx([]byte(key), []byte(response.GetValue()), int((time.Hour)))
	}()
	return response
}

func buildResponseMatrix(conf *config.Config, kvResponse *kvProto.BatchGetResponse, modelConfigId string, entityIds []string, defaultModelResponse *kvProto.GetResponse, headerMap map[string]string, reqEntitiesMap map[string][]string, entityIdIdx int) [][]string {
	features := conf.ResponseConfig.Features
	resConfig := conf.DrConfig.Res
	overrideFeatures := conf.DrConfig.Override
	numEntities := len(entityIds)
	numFeatures := len(features)

	featureIndexMap := make(map[string]int, numFeatures)
	for idx, f := range features {
		featureIndexMap[f] = idx
	}

	responseMatrix := make([][]string, numEntities+1)
	responseMatrix[0] = features
	initResponseMatrix(responseMatrix, numEntities, numFeatures)

	if kvResponse == nil && defaultModelResponse == nil {
		for i := range responseMatrix[1:] {
			for key, val := range headerMap {
				if idx, ok := featureIndexMap[key]; ok {
					responseMatrix[i+1][idx] = val
				}
			}

			for key, values := range reqEntitiesMap {
				if idx, ok := featureIndexMap[key]; ok && idx < len(values) {
					responseMatrix[i+1][idx] = values[i]
				}
			}
		}
		return responseMatrix
	}

	var defaultValues []string
	if defaultModelResponse != nil {
		defaultValues = decodeJSON(defaultModelResponse.GetValue(), len(resConfig))
	}

	var kvData []*kvProto.GetResponse
	if kvResponse != nil {
		kvData = kvResponse.GetData()
	}
	var count int
	for i := 0; i < numEntities; i++ {
		row := responseMatrix[i+1]
		buildDataRow(row, kvData[i], resConfig, featureIndexMap, reqEntitiesMap, headerMap, overrideFeatures, defaultValues, i)
		if kvData[i] == nil || !kvData[i].GetFound() {
			count++
		}
	}

	if count > 0 {
		metrics.Count("inferflow.retrievemodelscorefromkvservice.request.success", int64(count), []string{"partial", "true", "model-id", modelConfigId})
	}

	if (numEntities - count) > 0 {
		metrics.Count("inferflow.retrievemodelscorefromkvservice.request.success", int64(numEntities-count), []string{"partial", "false", "model-id", modelConfigId})
	}

	return responseMatrix
}

func buildDataRow(
	row []string,
	response *kvProto.GetResponse,
	resConfig []string,
	featureIndexMap map[string]int,
	reqEntitiesMap map[string][]string,
	headerMap map[string]string,
	overrideFeatures map[string]string,
	defaultValues []string,
	index int,
) {

	for key, val := range overrideFeatures {
		if idx, ok := featureIndexMap[key]; ok {
			row[idx] = val
		}
	}

	for key, val := range headerMap {
		if idx, ok := featureIndexMap[key]; ok {
			row[idx] = val
		}
	}

	for key, values := range reqEntitiesMap {
		if idx, ok := featureIndexMap[key]; ok && idx < len(values) {
			row[idx] = values[index]
		}
	}

	var values []string
	if response != nil && response.GetFound() {
		values = decodeJSON(response.GetValue(), len(resConfig))
	} else {
		values = defaultValues
	}

	if values == nil {
		return
	}

	for i, feature := range resConfig {
		if idx, ok := featureIndexMap[feature]; ok && i < len(values) {
			row[idx] = values[i]
		}
	}
}

func decodeJSON(jsonStr string, expectedLen int) []string {
	if jsonStr == "" {
		return nil
	}
	values := make([]string, expectedLen)
	if err := json.Unmarshal([]byte(jsonStr), &values); err != nil {
		log.Error().Msgf("Error unmarshalling kv response: %v", err)
		return nil
	}
	return values
}

func initResponseMatrix(responseMatrix [][]string, numEntities int, numFeatures int) {
	stringBuffer := make([]string, numEntities*numFeatures)

	for i := range responseMatrix[1:] {
		responseMatrix[i+1] = stringBuffer[i*numFeatures : (i+1)*numFeatures]
	}
}
func getAllHeaders(ctx context.Context) map[string]string {
	hMap := make(map[string]string, 0)
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		for headerKey, _ := range headersMap {
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

func getComputeId(c *config.Config) string {

	var numerixComputeId string
	for _, iNumerixComp := range c.ComponentConfig.NumerixComponentConfig.Values() {
		if numerixComp, ok := iNumerixComp.(config.NumerixComponentConfig); ok {
			numerixComputeId = numerixComp.ComputeId
		}
	}
	return numerixComputeId
}

func logInferflowResponse(ctx context.Context, userId, trackingId string, conf *config.Config, compRequest *components.ComponentRequest) {
	// Add nil check for component data
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Recovered from panic in logInferflowResponse: %v", r)
		}
	}()

	if compRequest == nil || compRequest.ComponentData == nil {
		logger.Error("Component request or component data is nil", nil)
		return
	}

	itemsLoggingData := extPrism.ItemsLoggingData{
		UserId:            userId,
		InferflowConfigId: compRequest.ModelId,
		IopId:             trackingId,
		ItemsMeta:         extPrism.ItemsMeta{},
		Items:             make([]extPrism.Item, 0),
	}

	var featuresToLog map[string]bool
	if !utils.IsNilOrEmpty(conf.ResponseConfig) &&
		conf.ResponseConfig.LogFeatures &&
		!utils.IsNilOrEmpty(conf.ResponseConfig.Features) {
		featuresToLog = make(map[string]bool)
		for _, feature := range conf.ResponseConfig.Features {
			featuresToLog[feature] = true
		}
	} else {
		featuresToLog = nil
	}

	// Structure to map new schema index to original column info
	type ColumnMapping struct {
		IsStringColumn bool
		OriginalIndex  int
		DataType       string // for byte columns
	}

	shouldInclude := func(name string) bool {
		return featuresToLog == nil || featuresToLog[name]
	}

	totalCols := len(compRequest.ComponentData.ByteColumnIndexMap) + len(compRequest.ComponentData.StringColumnIndexMap)
	if featuresToLog != nil {
		totalCols = len(featuresToLog)
	}
	featureSchema := make([]string, 0, totalCols)
	columnMappings := make([]ColumnMapping, 0, totalCols)
	nameToSchemaIndex := make(map[string]int, totalCols)

	if &conf.ResponseConfig == nil || len(conf.ResponseConfig.Features) == 0 {
		return
	}

	idColName := conf.ResponseConfig.Features[0]
	col, ok := compRequest.ComponentData.StringColumnIndexMap[idColName]
	if ok {
		idx := len(featureSchema)
		featureSchema = append(featureSchema, idColName)
		nameToSchemaIndex[idColName] = idx
		columnMappings = append(columnMappings, ColumnMapping{
			IsStringColumn: true,
			OriginalIndex:  col.Index,
		})
	}

	// Collect all unique column names from byte columns
	for name, col := range compRequest.ComponentData.ByteColumnIndexMap {
		if name == idColName || !shouldInclude(name) {
			continue
		}
		if _, exists := nameToSchemaIndex[name]; exists {
			continue
		}
		idx := len(featureSchema)
		featureSchema = append(featureSchema, name)
		nameToSchemaIndex[name] = idx
		columnMappings = append(columnMappings, ColumnMapping{
			IsStringColumn: false,
			OriginalIndex:  col.Index,
			DataType:       col.DataType,
		})
	}

	// Collect all unique column names from string columns
	for name, col := range compRequest.ComponentData.StringColumnIndexMap {
		if name == idColName || !shouldInclude(name) {
			continue
		}
		if idx, exists := nameToSchemaIndex[name]; exists {
			columnMappings[idx] = ColumnMapping{
				IsStringColumn: true,
				OriginalIndex:  col.Index,
			}
		} else {
			idx := len(featureSchema)
			featureSchema = append(featureSchema, name)
			nameToSchemaIndex[name] = idx
			columnMappings = append(columnMappings, ColumnMapping{
				IsStringColumn: true,
				OriginalIndex:  col.Index,
			})
		}
	}

	itemsLoggingData.ItemsMeta.FeatureSchema = featureSchema

	// Optionally add Numerix compute ID
	if numerixComputeId := getComputeId(conf); numerixComputeId != "" {
		itemsLoggingData.ItemsMeta.NUMERIXComputeID = numerixComputeId
	}

	// Add headers from incoming context
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		hMap := make(map[string]string)
		for headerKey, field := range headersMap {
			headerValues := md.Get(headerKey)
			if len(headerValues) > 0 {
				hMap[field] = headerValues[0]
			}
		}
		itemsLoggingData.ItemsMeta.Headers = hMap
	}

	// Build Items using the new schema mapping
	schemaLength := len(itemsLoggingData.ItemsMeta.FeatureSchema)
	for _, row := range compRequest.ComponentData.Rows {
		fullRow := make([]string, schemaLength)

		// Populate data based on schema mapping
		for schemaIndex, columnMapping := range columnMappings {
			if columnMapping.IsStringColumn {
				// Handle string columns
				fullRow[schemaIndex] = row.StringData[columnMapping.OriginalIndex]
			} else {
				// Handle byte columns
				decoded, err := convertor.BytesToString(row.ByteData[columnMapping.OriginalIndex], columnMapping.DataType)
				if err != nil {
					log.Warn().AnErr("Error converting bytes to string while sending prism response", err)
					fullRow[schemaIndex] = ""
				} else {
					fullRow[schemaIndex] = decoded
				}
			}
		}

		itemsLoggingData.Items = append(itemsLoggingData.Items, extPrism.Item{Id: fullRow[0], Features: fullRow})
	}

	extPrism.SendPrismEventUsingMq(&itemsLoggingData, conf.ResponseConfig.LogBatchSize)
}
