package vector

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/qdrant/go-client/qdrant"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

var (
	vectorDb Database
	syncOnce sync.Once
)

const (
	SearchIndexedOnly = "search_indexed_only"
)

type Qdrant struct {
	QdrantClients map[string]*QdrantClient
	configManager config.Manager
	AppConfig     *structs.AppConfig
}

type QdrantClient struct {
	ReadClient    *qdrant.Client
	WriteClient   *qdrant.Client
	BackupClient  *qdrant.Client
	ReadHost      string
	WriteHost     string
	BackupHost    string
	Deadline      int
	WriteDeadline int
}

// GetQdrantInstance initializes and returns a Database instance for Qdrant.
func initQdrantInstance() Database {
	if vectorDb == nil {
		syncOnce.Do(func() {
			vectorDb = createQdrantInstance()
		})
	}
	return vectorDb
}

// createQdrantInstance sets up the Qdrant instance with its configuration.
func createQdrantInstance() *Qdrant {
	resolver.SetDefaultScheme("dns")
	qdrantClients := make(map[string]*QdrantClient)
	configManager := config.NewManager(config.DefaultVersion)
	skyeConfig, err := configManager.GetSkyeConfig()
	if err != nil {
		log.Panic().Msgf("Error getting vector configs from etcd: %v", err)
	}
	entities := skyeConfig.Entity
	for entity, models := range entities {
		for model, modelConfig := range models.Models {
			for variant, variantConfig := range modelConfig.Variants {
				if variantConfig.VectorDbType != enums.QDRANT {
					continue
				}
				vectorConfig := variantConfig.VectorDbConfig
				key := GetClientKey(entity, model, variant)
				if variantConfig.Enabled {
					qdrantClients[key] = &QdrantClient{
						ReadHost:      vectorConfig.ReadHost,
						WriteHost:     vectorConfig.WriteHost,
						Deadline:      vectorConfig.Http2Config.Deadline,
						WriteDeadline: vectorConfig.Http2Config.WriteDeadline,
					}
					if vectorConfig.ReadHost != "" {
						var readGrpcClient *qdrant.Client
						readGrpcClient = nil
						if vectorConfig.ReadHost != "" {
							readGrpcClient, _ = createQdrantClient(vectorConfig, vectorConfig.ReadHost)
						}
						_ = healthCheck(model, variant, readGrpcClient)
						qdrantClients[key].ReadClient = readGrpcClient
						log.Error().Msgf("Read Client Created for %s %s", model, variant)
					}
					if vectorConfig.WriteHost != "" {
						var writeGrpcClient *qdrant.Client
						writeGrpcClient = nil
						if vectorConfig.WriteHost != "" {
							writeGrpcClient, _ = createQdrantClient(vectorConfig, vectorConfig.WriteHost)
						}
						_ = healthCheck(model, variant, writeGrpcClient)
						qdrantClients[key].WriteClient = writeGrpcClient
						log.Error().Msgf("Write Client Created for %s %s", model, variant)
					}
				}
				if variantConfig.BackupConfig.Enabled && variantConfig.BackupConfig.Host != "" {
					var backupClientGrpc *qdrant.Client
					backupClientGrpc, _ = createQdrantClient(variantConfig.VectorDbConfig, variantConfig.BackupConfig.Host)
					_ = healthCheck(model, variant, backupClientGrpc)
					value, exists := qdrantClients[key]
					if exists {
						value.BackupClient = backupClientGrpc
						value.BackupHost = variantConfig.BackupConfig.Host
					} else {
						qdrantClients[key] = &QdrantClient{
							ReadHost:      variantConfig.BackupConfig.Host,
							WriteHost:     variantConfig.BackupConfig.Host,
							Deadline:      vectorConfig.Http2Config.Deadline,
							WriteDeadline: vectorConfig.Http2Config.WriteDeadline,
							BackupClient:  backupClientGrpc,
							BackupHost:    variantConfig.BackupConfig.Host,
						}
					}
					log.Error().Msgf("Backup Client Created for %s %s with new Host %s", model, variant, variantConfig.BackupConfig.Host)
				}
			}
		}
	}
	qdrantInstance := &Qdrant{
		QdrantClients: qdrantClients,
		configManager: configManager,
		AppConfig:     structs.GetAppConfig(),
	}

	return qdrantInstance
}

func GetClientKey(entity, model, variant string) string {
	return fmt.Sprintf("%s:%s:%s", entity, model, variant)
}

// createQdrantClient creates a new Qdrant client based on the given vector configuration.
func createQdrantClient(vectorConfig config.VectorDbConfig, host string) (*qdrant.Client, error) {
	port, err := strconv.Atoi(vectorConfig.Port)
	if err != nil {
		log.Error().Msgf("Could not convert port to int: %v", err)
		return nil, err
	}
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
		GrpcOptions: []grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
		},
	})
	if err != nil {
		log.Error().Msgf("Could not create qdrant client: %v", err)
		return nil, err
	}

	return client, nil
}

// CreateCollection creates a new collection in the vector database.
func (q *Qdrant) CreateCollection(entity, model, variant string, version int) error {
	client := q.getQdrantClient(entity, model, variant)
	modelConfig, err := q.configManager.GetModelConfig(entity, model)
	if err != nil {
		log.Error().Msgf("Could not get model config: %v", err)
		return err
	}
	collectionName := getCollectionName(variant, model, strconv.Itoa(version))
	variantConfig, err := q.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Could not get variant params: %v", err)
		return err
	}
	if err := q.createCollection(client, collectionName, modelConfig.ModelConfig, variantConfig.VectorDbConfig.Params, entity, model, variant, version); err != nil {
		return err
	}
	log.Info().Msgf("Collection created: %v", collectionName)
	if variantConfig.VectorDbConfig.Params["after_collection_index_payload"] != "" && variantConfig.VectorDbConfig.Params["after_collection_index_payload"] == "true" {
		return nil
	}
	return q.CreateFieldIndexes(entity, model, variant, version)
}

// DeleteCollection deletes a collection from the vector database.
func (q *Qdrant) DeleteCollection(entity, model, variant string, version int) error {
	client := q.getQdrantClient(entity, model, variant)
	collectionName := getCollectionName(variant, model, strconv.Itoa(version))
	if err := q.deleteCollection(client, entity, model, variant, version, collectionName); err != nil {
		log.Error().Msgf("Failed to delete collection %v: %v", collectionName, err)
		return err
	}
	log.Info().Msgf("Collection deleted: %v", collectionName)
	return nil
}

// BulkUpsert upsert a batch of data into the vector database.
func (q *Qdrant) BulkUpsert(upsertRequest UpsertRequest) error {
	for key, data := range upsertRequest.Data {
		startTime := time.Now()
		parts := strings.Split(key, "|")
		if len(parts) != 4 {
			return fmt.Errorf("invalid key format: %s", key)
		}
		entity, model, variant, version := parts[0], parts[1], parts[2], parts[3]
		client := q.getQdrantClient(entity, model, variant)
		metric.Incr("vector_db_bulk_upsert", getMetricTags(entity, model, variant, version))
		upsertPoints, err := q.prepareUpsertPoints(data)
		if err != nil {
			log.Error().Msgf("Failed to prepare upsert points: %v", err)
			return err
		}
		if err := q.upsertPoints(client, variant, model, entity, version, upsertPoints); err != nil {
			log.Error().Msgf("Could not upsert points: %v", err)
			metric.Incr("vector_db_bulk_upsert_error", getMetricTags(entity, model, variant, version))
			return err
		}
		metric.Timing("vector_db_bulk_upsert_latency", time.Since(startTime), getMetricTags(entity, model, variant, version))
	}
	return nil
}

// BulkDelete deletes a batch of data from the vector database.
func (q *Qdrant) BulkDelete(deleteRequest DeleteRequest) error {
	for key, data := range deleteRequest.Data {
		startTime := time.Now()
		parts := strings.Split(key, "|")
		if len(parts) != 4 {
			return fmt.Errorf("invalid key format: %s", key)
		}
		entity, model, variant, version := parts[0], parts[1], parts[2], parts[3]
		client := q.getQdrantClient(entity, model, variant)
		metric.Incr("vector_db_bulk_delete", getMetricTags(entity, model, variant, version))
		deletePoints, err := q.prepareDeletePoints(data)
		if err != nil {
			log.Error().Msgf("Failed to prepare delete points: %v", err)
			return err
		}
		if err := q.deletePoints(client, entity, model, variant, version, deletePoints); err != nil {
			log.Error().Msgf("Could not delete points: %v", err)
			metric.Incr("vector_db_bulk_delete_error", getMetricTags(entity, model, variant, version))
			return err
		}
		metric.Timing("vector_db_bulk_delete_latency", time.Since(startTime), getMetricTags(entity, model, variant, version))
	}
	return nil
}

// BulkUpsertPayload upsert a batch of payload data into the vector database.
func (q *Qdrant) BulkUpsertPayload(upsertPayloadRequest UpsertPayloadRequest) error {
	for key, data := range upsertPayloadRequest.Data {
		startTime := time.Now()
		parts := strings.Split(key, "|")
		if len(parts) != 4 {
			return fmt.Errorf("invalid key format: %s", key)
		}
		entity, model, variant, version := parts[0], parts[1], parts[2], parts[3]
		client := q.getQdrantClient(entity, model, variant)
		metric.Incr("vector_db_bulk_upsert_payload", getMetricTags(entity, model, variant, version))
		if err := q.upsertPayloads(client, entity, model, variant, version, data); err != nil {
			log.Error().Msgf("Could not delete points: %v", err)
			metric.Incr("vector_db_bulk_upsert_payload_error", getMetricTags(entity, model, variant, version))
			return err
		}
		metric.Timing("vector_db_bulk_upsert_payload_latency", time.Since(startTime), getMetricTags(entity, model, variant, version))
	}
	return nil
}

// UpdateIndexingThreshold updates the indexing threshold for a specified collection in the vector database.
func (q *Qdrant) UpdateIndexingThreshold(entity, model, variant string, version int, indexingThreshold string) error {
	client := q.getQdrantClient(entity, model, variant)
	collectionName := getCollectionName(variant, model, strconv.Itoa(version))
	int64Threshold := getInt64FromStringValue(indexingThreshold)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.WriteDeadline)*time.Millisecond)
	defer cancel()
	err := q.updateCollectionIndexingThreshold(ctx, client, entity, model, variant, version, collectionName, &int64Threshold)
	if err != nil {
		log.Error().Msgf("Failed to update indexing threshold for collection %s: %v", collectionName, err)
		return err
	}
	log.Info().Msgf("Indexing threshold updated for collection %s to %d", collectionName, int64Threshold)
	return nil
}

// GetCollectionInfo retrieves information about a specified collection in the vector database.
func (q *Qdrant) GetCollectionInfo(entity, model, variant string, version int) (*CollectionInfoResponse, error) {
	// Defer recovery from panic to handle unexpected errors gracefully
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic occurred: %v", r)
			log.Error().Msgf("%s", err)
		}
	}()
	client := q.getQdrantClient(entity, model, variant)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.WriteDeadline)*time.Millisecond)
	defer cancel()
	writeCollectionsClient := qdrant.NewCollectionsClient(client.WriteClient.GetConnection())
	collectionName := getCollectionName(variant, model, strconv.Itoa(version))
	response, err := writeCollectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: collectionName,
	})
	if err != nil || response == nil {
		log.Error().Msgf("Failed to get collection info for %s: %v", collectionName, err)
		return nil, err
	}
	variantConfig, err := q.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for %s %s %s: %v", entity, model, variant, err)
		return nil, err
	}

	// Map response fields to the response struct
	return q.mapCollectionInfoResponse(response, variantConfig.VectorDbConfig.Payload), nil
}

// GetCollectionInfo retrieves information about a specified collection in the vector database.
func (q *Qdrant) GetReadCollectionInfo(entity, model, variant string, version int) (*CollectionInfoResponse, error) {
	// Defer recovery from panic to handle unexpected errors gracefully
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic occurred: %v", r)
			log.Error().Msgf("%s", err)
		}
	}()
	client := q.getQdrantClient(entity, model, variant)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.WriteDeadline)*time.Millisecond)
	defer cancel()
	readCollectionsClient := qdrant.NewCollectionsClient(client.ReadClient.GetConnection())
	collectionName := getCollectionName(variant, model, strconv.Itoa(version))
	response, err := readCollectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: collectionName,
	})
	if err != nil || response == nil {
		log.Error().Msgf("Failed to get collection info for %s: %v", collectionName, err)
		return nil, err
	}
	variantConfig, err := q.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for %s %s %s: %v", entity, model, variant, err)
		return nil, err
	}

	// Map response fields to the response struct
	return q.mapCollectionInfoResponse(response, variantConfig.VectorDbConfig.Payload), nil
}

func (q *Qdrant) BatchQuery(request *BatchQueryRequest, metricTags []string) (*BatchQueryResponse, error) {
	startTime := time.Now()
	metric.Incr("vector_db_batch_query", append([]string{"vector_db_type", "qdrant"}, metricTags...))
	skyeConfig, err := q.configManager.GetSkyeConfig()
	if err != nil {
		log.Error().Msgf("Error getting skye config: %v", err)
		return nil, err
	}
	variantConfig := skyeConfig.Entity[request.Entity].Models[request.Model].Variants[request.Variant]
	client := q.getQdrantClient(request.Entity, request.Model, request.Variant)
	collectionName := getCollectionName(request.Variant, request.Model, strconv.Itoa(request.Version))

	searchPoints := make([]*qdrant.SearchPoints, 0, len(request.RequestList))
	if err := q.prepareQueryPointsFromRequestList(&searchPoints, request, collectionName, variantConfig.VectorDbConfig.Payload); err != nil {
		metric.Incr("vector_db_query_prepare_failure", append([]string{"vector_db_type", "qdrant"}, metricTags...))
		log.Error().Msgf("Error preparing query points for request %+v, error %+v", *request, err)
		return nil, err
	}
	var pointsClient qdrant.PointsClient
	if variantConfig.BackupConfig.Enabled && rand.Intn(101) < variantConfig.BackupConfig.RoutePercentage {
		pointsClient = qdrant.NewPointsClient(client.BackupClient.GetConnection())
	} else {
		pointsClient = qdrant.NewPointsClient(client.ReadClient.GetConnection())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.Deadline)*time.Millisecond)
	defer cancel()
	queryBatchResponse, e := pointsClient.SearchBatch(ctx, &qdrant.SearchBatchPoints{
		CollectionName: collectionName,
		SearchPoints:   searchPoints,
	})
	if e != nil {
		metric.Incr("vector_db_batch_query_failure", append([]string{"vector_db_type", "qdrant"}, metricTags...))
		log.Error().Msgf("Error fetch simirlar candidate for request %+v, error %+v", *request, e)
		return nil, e
	}
	log.Info().Msgf("vectory db query response %+v", queryBatchResponse)
	result := parseBatchResponse(queryBatchResponse.GetResult(), request.RequestList, request)
	metric.Timing("vector_db_batch_query_latency", time.Since(startTime),
		append([]string{"vector_db_type", "qdrant"}, metricTags...))
	return result, nil
}

func (q *Qdrant) prepareQueryPointsFromRequestList(queryPoints *[]*qdrant.SearchPoints, request *BatchQueryRequest,
	collectionName string, payloadSchema map[string]config.Payload) error {
	for _, request := range request.RequestList {
		filter, err := parseFiltersToQdrantFilters(request, payloadSchema)
		if err != nil {
			return err
		}
		limit := uint64(request.CandidateLimit)
		offset := uint64(request.Offset)
		searchIndexedOnly := true
		*queryPoints = append(*queryPoints, &qdrant.SearchPoints{
			Vector:         request.Embedding,
			CollectionName: collectionName,
			Filter:         filter,
			Limit:          limit,
			Offset:         &offset,
			WithPayload:    qdrant.NewWithPayloadInclude(request.Payload...),
			Params:         &qdrant.SearchParams{IndexedOnly: &searchIndexedOnly},
		})
	}
	return nil
}

// createCollection creates the collection in the Qdrant database.
func (q *Qdrant) createCollection(client *QdrantClient, collectionName string, modelConfig config.ModelConfig, params map[string]string, entity, model, variant string, version int) error {
	indexingThreshold := getInt64FromStringValue("0")
	segmentNumber := uint64(8)
	maxSegmentSize := uint64(204800)
	m := uint64(32)
	efConstruct := uint64(200)
	if _, ok := params["segment_number"]; ok {
		segmentNumber = *getInt64FromString(params, "segment_number")
	}
	if _, ok := params["max_segment_size_in_mb"]; ok {
		maxSegmentSize = *getInt64FromString(params, "max_segment_size_in_mb") * 1024
	}
	if _, ok := params["m"]; ok {
		m = *getInt64FromString(params, "m")
	}
	if _, ok := params["ef_construct"]; ok {
		efConstruct = *getInt64FromString(params, "ef_construct")
	}
	maxIndexingThreads := getInt64FromString(params, "max_indexing_threads")
	replicationFactor := getInt32FromString(params, "replication_factor")
	if q.AppConfig.Configs.AppEnv == "int" {
		uInt := uint64(0)
		maxIndexingThreads = &uInt
		uInt32 := uint32(1)
		replicationFactor = &uInt32
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.WriteDeadline)*time.Millisecond)
	defer cancel()
	createCollection := &qdrant.CreateCollection{
		CollectionName:         collectionName,
		ShardNumber:            getInt32FromString(params, "shard_number"),
		ReplicationFactor:      replicationFactor,
		WriteConsistencyFactor: getInt32FromString(params, "write_consistency_factor"),
		OnDiskPayload:          getBoolFromString(params, "on_disk_payload"),
		VectorsConfig: &qdrant.VectorsConfig{Config: &qdrant.VectorsConfig_Params{
			Params: &qdrant.VectorParams{
				Size:     modelConfig.VectorDimension,
				Distance: convertToDistance(modelConfig.DistanceFunction),
			},
		}},
		OptimizersConfig: &qdrant.OptimizersConfigDiff{
			DefaultSegmentNumber: &segmentNumber,
			IndexingThreshold:    &indexingThreshold,
			MaxSegmentSize:       &maxSegmentSize,
		},
		HnswConfig: &qdrant.HnswConfigDiff{
			M:                  &m,
			EfConstruct:        &efConstruct,
			MaxIndexingThreads: maxIndexingThreads,
		},
	}
	WriteClient := qdrant.NewCollectionsClient(client.WriteClient.GetConnection())
	_, err := WriteClient.Create(ctx, createCollection)
	if err != nil {
		log.Error().Msgf("Could not create collection: %v", err)
		return err
	}
	variantConfig, err := q.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Could not get payload: %v", err)
		return err
	}
	if variantConfig.BackupConfig.Enabled && variantConfig.BackupConfig.DualWriteEnabled && variantConfig.BackupConfig.Host != "" {
		backupClient := qdrant.NewCollectionsClient(client.BackupClient.GetConnection())
		_, err = backupClient.Create(ctx, createCollection)
		if err != nil {
			metric.Incr("vector_db_create_collection_backup_error", getMetricTags(entity, model, variant, strconv.Itoa(version)))
		}
	}

	return nil
}

// CreateFieldIndexes creates field indexes for the given model payload.
func (q *Qdrant) CreateFieldIndexes(entity, model, variant string, version int) error {
	client := q.getQdrantClient(entity, model, variant)
	variantConfig, err := q.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Could not get payload: %v", err)
		return err
	}
	writeClient := qdrant.NewPointsClient(client.WriteClient.GetConnection())
	for key, modelPayloadValue := range variantConfig.VectorDbConfig.Payload {
		fieldIndexType := GetFieldIndexType(modelPayloadValue.FieldSchema)
		fieldIndexName := key
		err = q.createFieldIndex(writeClient, getCollectionName(variant, model, strconv.Itoa(version)), fieldIndexName, fieldIndexType, modelPayloadValue)
		if err != nil {
			return err
		}
	}
	if variantConfig.BackupConfig.Enabled && variantConfig.BackupConfig.DualWriteEnabled && variantConfig.BackupConfig.Host != "" {
		go func() {
			backupClient := qdrant.NewPointsClient(client.BackupClient.GetConnection())
			for key, modelPayloadValue := range variantConfig.VectorDbConfig.Payload {
				fieldIndexType := GetFieldIndexType(modelPayloadValue.FieldSchema)
				fieldIndexName := key
				err = q.createFieldIndex(backupClient, getCollectionName(variant, model, strconv.Itoa(version)), fieldIndexName, fieldIndexType, modelPayloadValue)
				if err != nil {
					metric.Incr("vector_db_field_index_creation_backup_error", getMetricTags(entity, model, variant, strconv.Itoa(version)))
				}
			}
		}()
	}
	return nil
}

// createFieldIndex creates a field index in the specified collection.
func (q *Qdrant) createFieldIndex(pointsClient qdrant.PointsClient, collectionName, fieldIndexName string, fieldType qdrant.FieldType, payloadSchema config.Payload) error {
	fieldIndexParams := q.getFieldIndexParams(fieldType, payloadSchema)

	_, err := pointsClient.CreateFieldIndex(context.Background(), &qdrant.CreateFieldIndexCollection{
		CollectionName:   collectionName,
		FieldName:        fieldIndexName,
		FieldType:        &fieldType,
		FieldIndexParams: fieldIndexParams,
	})
	if err != nil {
		log.Error().Msgf("Could not create field index: %v", err)
		return err
	}
	return nil
}

// getFieldIndexParams returns the appropriate PayloadIndexParams based on field type.
func (q *Qdrant) getFieldIndexParams(fieldType qdrant.FieldType, payloadSchema config.Payload) *qdrant.PayloadIndexParams {
	switch fieldType {
	case qdrant.FieldType_FieldTypeKeyword:
		return &qdrant.PayloadIndexParams{
			IndexParams: &qdrant.PayloadIndexParams_KeywordIndexParams{
				KeywordIndexParams: &qdrant.KeywordIndexParams{},
			},
		}
	case qdrant.FieldType_FieldTypeInteger:
		return &qdrant.PayloadIndexParams{
			IndexParams: &qdrant.PayloadIndexParams_IntegerIndexParams{
				IntegerIndexParams: &qdrant.IntegerIndexParams{
					Lookup:      qdrant.PtrOf(payloadSchema.LookupEnabled),
					IsPrincipal: qdrant.PtrOf(payloadSchema.IsPrincipal),
				},
			},
		}
	case qdrant.FieldType_FieldTypeBool:
		return &qdrant.PayloadIndexParams{
			IndexParams: &qdrant.PayloadIndexParams_BoolIndexParams{
				BoolIndexParams: &qdrant.BoolIndexParams{},
			},
		}
	default:
		return nil
	}
}

// deleteCollection performs the actual deletion of the collection.
func (q *Qdrant) deleteCollection(client *QdrantClient, entity, model, variant string, version int, collectionName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.WriteDeadline)*time.Millisecond)
	defer cancel()
	WriteClient := qdrant.NewCollectionsClient(client.WriteClient.GetConnection())
	_, err := WriteClient.Delete(ctx, &qdrant.DeleteCollection{
		CollectionName: collectionName,
	})
	if err != nil {
		log.Error().Msgf("Failed to delete collection %v: %v", collectionName, err)
		return err
	}
	variantConfig, err := q.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for %s %s %s: %v", entity, model, variant, err)
		return err
	}
	if variantConfig.BackupConfig.Enabled && variantConfig.BackupConfig.DualWriteEnabled && variantConfig.BackupConfig.Host != "" {
		go func() {
			if version-1 < 0 {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.WriteDeadline)*time.Millisecond)
			defer cancel()
			collectionName := getCollectionName(variant, model, strconv.Itoa(version-1))
			backupClient := qdrant.NewCollectionsClient(client.BackupClient.GetConnection())
			_, err := backupClient.Delete(ctx, &qdrant.DeleteCollection{
				CollectionName: collectionName,
			})
			if err != nil {
				metric.Incr("vector_db_delete_collection_backup_error", getMetricTags(entity, model, variant, strconv.Itoa(version)))
			}
		}()
	}
	return err
}

// prepareUpsertPoints prepares the points for the upsert operation.
func (q *Qdrant) prepareUpsertPoints(data []Data) ([]*qdrant.PointStruct, error) {
	var upsertPoints []*qdrant.PointStruct
	for _, d := range data {
		payload := make(map[string]*qdrant.Value)
		for key, value := range d.Payload {
			payload[key] = adaptToPayloadValue(value)
		}
		upsertPoint := &qdrant.PointStruct{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Num{Num: getInt64FromStringValue(d.Id)},
			},
			Payload: payload,
			Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: d.Vectors}}},
		}
		upsertPoints = append(upsertPoints, upsertPoint)
	}
	return upsertPoints, nil
}

// upsertPoints performs the actual upsert operation in the database.
func (q *Qdrant) upsertPoints(client *QdrantClient, variant, model, entity, version string, upsertPoints []*qdrant.PointStruct) error {
	waitUpsert := true
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.WriteDeadline)*time.Millisecond)
	defer cancel()
	writePointsClient := qdrant.NewPointsClient(client.WriteClient.GetConnection())
	_, err := writePointsClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: getCollectionName(variant, model, version),
		Wait:           &waitUpsert,
		Points:         upsertPoints,
	})
	if err != nil {
		log.Error().Msgf("Failed to upsert points: %v", err)
		return err
	}
	variantConfig, err := q.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for %s %s %s: %v", entity, model, variant, err)
		return err
	}
	if variantConfig.BackupConfig.Enabled && variantConfig.BackupConfig.DualWriteEnabled && variantConfig.BackupConfig.Host != "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.WriteDeadline)*time.Millisecond)
			defer cancel()
			backupClient := qdrant.NewPointsClient(client.BackupClient.GetConnection())
			_, err = backupClient.Upsert(ctx, &qdrant.UpsertPoints{
				CollectionName: getCollectionName(variant, model, version),
				Wait:           &waitUpsert,
				Points:         upsertPoints,
			})
			metric.Incr("vector_db_bulk_upsert_backup", getMetricTags(entity, model, variant, version))
			if err != nil {
				metric.Incr("vector_db_bulk_upsert_backup_error", getMetricTags(entity, model, variant, version))
			}
		}()
	}
	return err
}

// prepareDeletePoints prepares the points for the delete operation.
func (q *Qdrant) prepareDeletePoints(data []Data) ([]*qdrant.PointId, error) {
	var deletePoints []*qdrant.PointId
	for _, d := range data {
		deletePoint := &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Num{Num: getInt64FromStringValue(d.Id)},
		}
		deletePoints = append(deletePoints, deletePoint)
	}
	return deletePoints, nil
}

// deletePoints performs the actual delete operation in the database.
func (q *Qdrant) deletePoints(client *QdrantClient, entity, model, variant, version string, deletePoints []*qdrant.PointId) error {
	waitDelete := true
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.WriteDeadline)*time.Millisecond)
	defer cancel()
	var err error
	pointSelector := &qdrant.PointsSelector{
		PointsSelectorOneOf: &qdrant.PointsSelector_Points{
			Points: &qdrant.PointsIdsList{
				Ids: deletePoints,
			},
		},
	}
	writePointsClient := qdrant.NewPointsClient(client.WriteClient.GetConnection())
	_, err = writePointsClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: getCollectionName(variant, model, version),
		Wait:           &waitDelete,
		Points:         pointSelector,
	})
	if err != nil {
		log.Error().Msgf("Failed to delete points: %v", err)
		return err
	}
	variantConfig, err := q.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for %s %s %s: %v", entity, model, variant, err)
		return err
	}
	if variantConfig.BackupConfig.Enabled && variantConfig.BackupConfig.DualWriteEnabled && variantConfig.BackupConfig.Host != "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.WriteDeadline)*time.Millisecond)
			defer cancel()
			backupClient := qdrant.NewPointsClient(client.BackupClient.GetConnection())
			_, err = backupClient.Delete(ctx, &qdrant.DeletePoints{
				CollectionName: getCollectionName(variant, model, version),
				Wait:           &waitDelete,
				Points:         pointSelector,
			})
			metric.Incr("vector_db_bulk_delete_backup", getMetricTags(entity, model, variant, version))
			if err != nil {
				metric.Incr("vector_db_bulk_delete_backup_error", getMetricTags(entity, model, variant, version))
			}
		}()
	}
	return err
}

// upsertPayloads handles the upsert of payload data into the database.
func (q *Qdrant) upsertPayloads(client *QdrantClient, entity, model, variant, version string, data []Data) error {
	for _, d := range data {
		upsertPayload, upsertPayloadPointSelector, err := q.prepareUpsertPayload(d)
		if err != nil {
			log.Error().Msgf("Failed to prepare upsert payload: %v", err)
			return err
		}
		if err := q.setPayload(client, variant, model, entity, version, upsertPayload, upsertPayloadPointSelector, true); err != nil {
			log.Error().Msgf("Could not upsert points: %v", err)
			metric.Count("vector_db_bulk_upsert_payload_error", int64(len(data)), []string{"vector_db_type", "qdrant", "entity_name", entity, "model_name", model, "variant_name", variant, "variant_version", version})
			return err
		}
	}

	return nil
}

// prepareUpsertPayload prepares the payload and point selector for the upsert operation.
func (q *Qdrant) prepareUpsertPayload(d Data) (map[string]*qdrant.Value, *qdrant.PointsSelector, error) {
	upsertPayload := make(map[string]*qdrant.Value)

	// Prepare the point ID.
	upsertPayloadPoint := &qdrant.PointId{
		PointIdOptions: &qdrant.PointId_Num{
			Num: getInt64FromStringValue(d.Id),
		},
	}

	for key, value := range d.Payload {
		upsertPayload[key] = adaptToPayloadValue(value)
	}

	upsertPayloadPoints := []*qdrant.PointId{upsertPayloadPoint}
	pointSelector := &qdrant.PointsSelector{
		PointsSelectorOneOf: &qdrant.PointsSelector_Points{
			Points: &qdrant.PointsIdsList{
				Ids: upsertPayloadPoints,
			},
		},
	}

	return upsertPayload, pointSelector, nil
}

// setPayload performs the actual payload setting operation in the database.
func (q *Qdrant) setPayload(client *QdrantClient, variant, model, entity, version string, upsertPayload map[string]*qdrant.Value, pointSelector *qdrant.PointsSelector, waitUpsert bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.WriteDeadline)*time.Millisecond)
	defer cancel()
	writePointsClient := qdrant.NewPointsClient(client.WriteClient.GetConnection())
	_, err := writePointsClient.SetPayload(ctx, &qdrant.SetPayloadPoints{
		CollectionName: getCollectionName(variant, model, version),
		Wait:           &waitUpsert,
		Payload:        upsertPayload,
		PointsSelector: pointSelector,
	})
	if err != nil {
		log.Error().Msgf("Failed to set payload: %v", err)
		return err
	}
	variantConfig, err := q.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for %s %s %s: %v", entity, model, variant, err)
		return err
	}
	if variantConfig.BackupConfig.Enabled && variantConfig.BackupConfig.DualWriteEnabled && variantConfig.BackupConfig.Host != "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.WriteDeadline)*time.Millisecond)
			defer cancel()
			backupClient := qdrant.NewPointsClient(client.BackupClient.GetConnection())
			_, err := backupClient.SetPayload(ctx, &qdrant.SetPayloadPoints{
				CollectionName: getCollectionName(variant, model, version),
				Wait:           &waitUpsert,
				Payload:        upsertPayload,
				PointsSelector: pointSelector,
			})
			metric.Incr("vector_db_bulk_upsert_payload_backup", getMetricTags(entity, model, variant, version))
			if err != nil {
				metric.Incr("vector_db_bulk_upsert_payload_backup_error", getMetricTags(entity, model, variant, version))
			}
		}()
	}
	return err
}

// updateCollectionIndexingThreshold performs the actual update operation on the collection's indexing threshold.
func (q *Qdrant) updateCollectionIndexingThreshold(ctx context.Context, client *QdrantClient, entity, model, variant string, version int, collectionName string, indexingThreshold *uint64) error {
	collectionsClient := qdrant.NewCollectionsClient(client.WriteClient.GetConnection())
	_, err := collectionsClient.Update(ctx, &qdrant.UpdateCollection{
		CollectionName: collectionName,
		OptimizersConfig: &qdrant.OptimizersConfigDiff{
			IndexingThreshold: indexingThreshold,
		},
	})
	if err != nil {
		log.Error().Msgf("Failed to update indexing threshold for collection %s: %v", collectionName, err)
		return err
	}
	variantConfig, err := q.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for %s %s %s: %v", entity, model, variant, err)
		return err
	}
	if variantConfig.BackupConfig.Enabled && variantConfig.BackupConfig.DualWriteEnabled && variantConfig.BackupConfig.Host != "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.WriteDeadline)*time.Millisecond)
			defer cancel()
			backupClient := qdrant.NewCollectionsClient(client.BackupClient.GetConnection())
			_, err := backupClient.Update(ctx, &qdrant.UpdateCollection{
				CollectionName: collectionName,
				OptimizersConfig: &qdrant.OptimizersConfigDiff{
					IndexingThreshold: indexingThreshold,
				},
			})
			if err != nil {
				metric.Incr("vector_db_update_indexing_threshold_backup_error", getMetricTags(entity, model, variant, strconv.Itoa(version)))
			}
		}()
	}
	return err
}

// mapCollectionInfoResponse maps the Qdrant collection info response to the custom CollectionInfoResponse struct.
func (q *Qdrant) mapCollectionInfoResponse(response *qdrant.GetCollectionInfoResponse, payloadSchema map[string]config.Payload) *CollectionInfoResponse {
	status := adaptToStatus(response.Result.Status)
	var pointsCount, indexedVectorCount float64
	if response.Result.IndexedVectorsCount != nil {
		indexedVectorCount = float64(*response.Result.IndexedVectorsCount)
	}
	if response.Result.PointsCount != nil {
		pointsCount = float64(*response.Result.PointsCount)
	}
	var payloadPointsCount []float64
	if response.Result.PayloadSchema != nil {
		for key, _ := range payloadSchema {
			if response.Result.PayloadSchema[key] != nil {
				payloadPointsCount = append(payloadPointsCount, float64(*response.Result.PayloadSchema[key].Points))
			}
		}
	}
	return &CollectionInfoResponse{
		Status:              status,
		IndexedVectorsCount: indexedVectorCount,
		PointsCount:         pointsCount,
		PayloadPointsCount:  payloadPointsCount,
	}
}

func healthCheck(model string, variant string, client *qdrant.Client) error {
	healthCheckResult, err := client.HealthCheck(context.Background())
	if err != nil {
		log.Info().Msgf("Could not get health: %v , %v , %v", model, variant, err)
	} else {
		log.Info().Msgf("Client version: %s", healthCheckResult.GetVersion())
	}
	return err
}

func (q *Qdrant) getQdrantClient(entity, model, variant string) *QdrantClient {
	return q.QdrantClients[GetClientKey(entity, model, variant)]
}

func getMetricTags(entity string, model string, variant string, version string) []string {
	return []string{"vector_db_type", "qdrant", "entity_name", entity, "model_name", model, "variant_name", variant, "variant_version", version}
}

func (q *Qdrant) RefreshClients(key, value, eventType string) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Recovered from panic: %v", r)
		}
	}()
	if eventType == "DELETE" {
		return nil
	}
	entity, model, variant := q.extractQdrantKey(key)
	if entity == "" || model == "" || variant == "" {
		return nil
	}
	variantConfig, err := q.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for %s %s %s: %v", entity, model, variant, err)
		return nil
	}

	if variantConfig.VectorDbType != enums.QDRANT {
		return nil
	}
	log.Info().Msgf("Qdrant entity config change detected - Key: %s, EventType: %s", key, eventType)
	clientKey := GetClientKey(entity, model, variant)
	if variantConfig.Enabled {
		vectorConfig := variantConfig.VectorDbConfig
		if _, exists := q.QdrantClients[clientKey]; !exists {
			var readGrpcClient *qdrant.Client
			readGrpcClient = nil
			var writeGrpcClient *qdrant.Client
			writeGrpcClient = nil
			if vectorConfig.ReadHost != "" {
				readGrpcClient, _ = createQdrantClient(vectorConfig, vectorConfig.ReadHost)
				err = healthCheck(model, variant, readGrpcClient)
			}
			if vectorConfig.WriteHost != "" {
				writeGrpcClient, _ = createQdrantClient(vectorConfig, vectorConfig.WriteHost)
				err = healthCheck(model, variant, writeGrpcClient)
			}
			if err != nil {
				log.Error().Msgf("Failed to create Client client for %s %s: %v", model, variant, err)
				return err
			}
			q.QdrantClients[clientKey] = &QdrantClient{
				ReadClient:    readGrpcClient,
				WriteClient:   writeGrpcClient,
				ReadHost:      vectorConfig.ReadHost,
				WriteHost:     vectorConfig.WriteHost,
				Deadline:      vectorConfig.Http2Config.Deadline,
				WriteDeadline: vectorConfig.Http2Config.WriteDeadline,
			}
			if variantConfig.BackupConfig.Enabled && variantConfig.BackupConfig.Host != "" {
				var backupClientGrpc *qdrant.Client
				backupClientGrpc, _ = createQdrantClient(vectorConfig, variantConfig.BackupConfig.Host)
				err = healthCheck(model, variant, backupClientGrpc)
				if err != nil {
					log.Error().Msgf("Failed to create Client client for %s %s: %v", model, variant, err)
					return err
				}
				q.QdrantClients[clientKey].BackupClient = backupClientGrpc
				q.QdrantClients[clientKey].BackupHost = variantConfig.BackupConfig.Host
				log.Error().Msgf("Backup Client Created for %s %s with new Host %s", model, variant, variantConfig.BackupConfig.Host)
			}
			log.Error().Msgf("Read and Write Client created on runtime %s %s", model, variant)
		} else {
			if q.QdrantClients[clientKey].ReadHost != vectorConfig.ReadHost {
				var readGrpcClient *qdrant.Client
				readGrpcClient = nil
				if vectorConfig.ReadHost != "" {
					readGrpcClient, _ = createQdrantClient(vectorConfig, vectorConfig.ReadHost)
					err = healthCheck(model, variant, readGrpcClient)
				}
				if err != nil {
					log.Error().Msgf("Failed to create Client client for %s %s: %v", model, variant, err)
					return err
				}
				q.QdrantClients[clientKey].ReadHost = vectorConfig.ReadHost
				q.QdrantClients[clientKey].ReadClient = readGrpcClient
				log.Error().Msgf("Read Client Refreshed for %s %s with new Host %s", model, variant, vectorConfig.ReadHost)
			}
			if q.QdrantClients[clientKey].WriteHost != vectorConfig.WriteHost {
				var writeGrpcClient *qdrant.Client
				writeGrpcClient = nil
				if vectorConfig.WriteHost != "" {
					writeGrpcClient, _ = createQdrantClient(vectorConfig, vectorConfig.WriteHost)
					err = healthCheck(model, variant, writeGrpcClient)
				}
				if err != nil {
					log.Error().Msgf("Failed to create Client client for %s %s: %v", model, variant, err)
					return err
				}
				q.QdrantClients[clientKey].WriteHost = vectorConfig.WriteHost
				q.QdrantClients[clientKey].WriteClient = writeGrpcClient
				log.Error().Msgf("Write Client Refreshed for %s %s with new Host %s", model, variant, vectorConfig.WriteHost)
			}
			if q.QdrantClients[clientKey].BackupHost != variantConfig.BackupConfig.Host {
				if variantConfig.BackupConfig.Enabled && variantConfig.BackupConfig.Host != "" {
					var backupClientGrpc *qdrant.Client
					backupClientGrpc, _ = createQdrantClient(vectorConfig, variantConfig.BackupConfig.Host)
					err = healthCheck(model, variant, backupClientGrpc)
					if err != nil {
						log.Error().Msgf("Failed to create Client client for %s %s: %v", model, variant, err)
						return err
					}
					q.QdrantClients[clientKey].BackupClient = backupClientGrpc
					q.QdrantClients[clientKey].BackupHost = variantConfig.BackupConfig.Host
					log.Error().Msgf("Backup Client Refreshed for %s %s with new Host %s", model, variant, variantConfig.BackupConfig.Host)
				}
			}
		}
	}
	return nil
}

func (q *Qdrant) extractQdrantKey(key string) (string, string, string) {
	parts := strings.Split(key, "/")
	for _, part := range parts {
		if part == "vector-db-config" || part == "enabled" || part == "backup-config" {
			return parts[4], parts[6], parts[8]
		}
	}
	return "", "", ""
}
