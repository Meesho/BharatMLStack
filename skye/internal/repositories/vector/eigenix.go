package vector

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	pb "github.com/Meesho/BharatMLStack/skye/internal/external/eigenix"
	"github.com/Meesho/BharatMLStack/skye/pkg/grpcclient"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog/log"
)

var (
	eigenixDb   Database
	eigenixOnce sync.Once
)

type Eigenix struct {
	EigenixClients map[string]*EigenixClient
	configManager  config.Manager
}

type EigenixClient struct {
	ReadClient    *grpcclient.GRPCClient
	WriteClient   *grpcclient.GRPCClient
	ReadHost      string
	WriteHost     string
	Port          string
	Deadline      int
	WriteDeadline int
}

func initEigenixInstance() Database {
	if eigenixDb == nil {
		eigenixOnce.Do(func() {
			eigenixDb = createEigenixInstance()
		})
	}
	return eigenixDb
}

func createEigenixInstance() *Eigenix {
	EigenixClients := make(map[string]*EigenixClient)
	configManager := config.NewManager(config.DefaultVersion)
	entities, err := configManager.GetEntities()
	if err != nil {
		log.Panic().Msgf("Error getting vector configs from etcd: %v", err)
	}
	for entity, models := range entities {
		for model, modelConfig := range models.Models {
			for variant, variantConfig := range modelConfig.Variants {
				if variantConfig.VectorDbType != enums.EIGENIX {
					continue
				}
				vectorConfig := variantConfig.VectorDbConfig
				key := GetClientKey(entity, model, variant)
				if variantConfig.Enabled {
					EigenixClients[key] = &EigenixClient{
						ReadHost:      vectorConfig.ReadHost,
						WriteHost:     vectorConfig.WriteHost,
						Port:          vectorConfig.Port,
						Deadline:      vectorConfig.Http2Config.Deadline,
						WriteDeadline: vectorConfig.Http2Config.WriteDeadline,
					}
					if vectorConfig.ReadHost != "" {
						var readGrpcClient *grpcclient.GRPCClient
						readGrpcClient = nil
						if vectorConfig.ReadHost != "" {
							readGrpcClient, _ = createEigenixClient(vectorConfig, vectorConfig.ReadHost, key)
						}
						_ = EigenixHealthCheck(model, variant, readGrpcClient)
						EigenixClients[key].ReadClient = readGrpcClient
						log.Error().Msgf("NGT Read Client Created for %s %s", model, variant)
					}
					if vectorConfig.WriteHost != "" {
						var writeGrpcClient *grpcclient.GRPCClient
						writeGrpcClient = nil
						if vectorConfig.WriteHost != "" {
							writeGrpcClient, _ = createEigenixClient(vectorConfig, vectorConfig.WriteHost, key)
						}
						_ = EigenixHealthCheck(model, variant, writeGrpcClient)
						EigenixClients[key].WriteClient = writeGrpcClient
						log.Error().Msgf("NGT Write Client Created for %s %s", model, variant)
					}
				}
			}
		}
	}

	EigenixInstance := &Eigenix{
		EigenixClients: EigenixClients,
		configManager:  configManager,
	}

	return EigenixInstance
}

func createEigenixClient(vectorConfig config.VectorDbConfig, host string, envPrefix string) (*grpcclient.GRPCClient, error) {
	grpcClient := grpcclient.NewConnFromConfig(&grpcclient.Config{
		Host:                host,
		Port:                vectorConfig.Port,
		DeadLine:            vectorConfig.Http2Config.Deadline,
		PlainText:           vectorConfig.Http2Config.IsPlainText,
		LoadBalancingPolicy: "round_robin",
	}, envPrefix)
	return grpcClient, nil
}

func (n *Eigenix) BatchQuery(bulkRequest *BatchQueryRequest, metricTags []string) (*BatchQueryResponse, error) {
	startTime := time.Now()
	metric.Incr("vector_db_batch_query", append([]string{"vector_db_type", "Eigenix"}, metricTags...))

	eigenixClient := n.getClient(bulkRequest.Entity, bulkRequest.Model, bulkRequest.Variant)
	if eigenixClient == nil || eigenixClient.ReadClient == nil {
		return nil, fmt.Errorf("Eigenix client not found for %s %s %s", bulkRequest.Entity, bulkRequest.Model, bulkRequest.Variant)
	}

	indexName := bulkRequest.Model + "_" + bulkRequest.Variant + "_" + strconv.Itoa(bulkRequest.Version)

	protoVectors := make([]*pb.Vector, 0, len(bulkRequest.RequestList))
	for _, searchRequest := range bulkRequest.RequestList {
		protoVectors = append(protoVectors, &pb.Vector{
			Values: searchRequest.Embedding,
		})
	}

	variantConfig, _ := n.configManager.GetVariantConfig(bulkRequest.Entity, bulkRequest.Model, bulkRequest.Variant)
	useIvf := true
	if variantConfig.VectorDbConfig.Params["use_ivf"] != "" {
		useIvf, _ = strconv.ParseBool(variantConfig.VectorDbConfig.Params["use_ivf"])
	}
	var centCount int32
	centCount = 10
	if variantConfig.VectorDbConfig.Params["cent_count"] != "" {
		centCount64, _ := strconv.ParseInt(variantConfig.VectorDbConfig.Params["cent_count"], 10, 32)
		centCount = int32(centCount64)
	}
	var databaseType pb.DatabaseType
	databaseType = pb.DatabaseType_QDRANT
	if variantConfig.VectorDbConfig.Params["database_type"] != "" {
		databaseType = pb.DatabaseType(pb.DatabaseType_value[variantConfig.VectorDbConfig.Params["database_type"]])
	}
	grpcRequest := &pb.BatchSearchRequest{
		IndexName: indexName,
		Vectors:   protoVectors,
		Limit:     bulkRequest.RequestList[0].CandidateLimit,
		SearchParams: &pb.SearchParams{
			UseIvf:            useIvf,
			CentCount:         centCount,
			SearchIndexedOnly: true,
		},
		DatabaseType: databaseType,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(eigenixClient.Deadline)*time.Millisecond)
	defer cancel()

	grpcResponse, err := pb.NewEigenixServiceClient(eigenixClient.ReadClient.Conn).BatchSearch(ctx, grpcRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to call Eigenix batch search: %w", err)
	}

	batchQueryResponse := mapProtoBatchResponseToBatchQueryResponse(grpcResponse, bulkRequest.RequestList, bulkRequest)

	metric.Timing("vector_db_batch_query_latency", time.Since(startTime), append([]string{"vector_db_type", "Eigenix"}, metricTags...))
	return batchQueryResponse, nil
}

func mapProtoBatchResponseToBatchQueryResponse(protoResp *pb.BatchSearchResponse, requestList []*QueryDetails, bulkRequest *BatchQueryRequest) *BatchQueryResponse {
	similarCandidatesList := make(map[string][]*SimilarCandidate, len(protoResp.ResultSets))

	for i, resultSet := range protoResp.ResultSets {
		similarCandidates := make([]*SimilarCandidate, 0, len(resultSet.Results))
		totalScore := 0.0
		totalResults := 0
		for _, result := range resultSet.Results {
			candidate := &SimilarCandidate{
				Id:      fmt.Sprintf("%d", result.Id),
				Score:   result.Distance,
				Payload: map[string]string{},
			}
			similarCandidates = append(similarCandidates, candidate)
			totalScore += float64(result.Distance)
			totalResults++
		}
		similarCandidatesList[requestList[i].CacheKey] = similarCandidates

		if totalResults > 0 {
			maxScore := resultSet.Results[0].Distance
			minScore := resultSet.Results[len(resultSet.Results)-1].Distance
			avgScore := totalScore / float64(totalResults)

			tags := []string{
				"vector_db_type", "eigenix",
				"entity", bulkRequest.Entity,
				"model_name", bulkRequest.Model,
				"variant", bulkRequest.Variant,
			}

			metric.Gauge("eigenix_query_similarity_mean_score", avgScore, tags)
			metric.Gauge("eigenix_query_similarity_max_score", float64(maxScore), tags)
			metric.Gauge("eigenix_query_similarity_min_score", float64(minScore), tags)
		}
	}

	return &BatchQueryResponse{
		SimilarCandidatesList: similarCandidatesList,
	}
}

func (n *Eigenix) getClient(entity, model, variant string) *EigenixClient {
	return n.EigenixClients[GetClientKey(entity, model, variant)]
}

func (n *Eigenix) CreateCollection(entity string, model string, variant string, version int) error {
	//TODO implement me
	panic("implement me")
}

func (n *Eigenix) BulkUpsert(upsertRequest UpsertRequest) error {
	//TODO implement me
	panic("implement me")
}

func (n *Eigenix) BulkDelete(deleteRequest DeleteRequest) error {
	//TODO implement me
	panic("implement me")
}

func (n *Eigenix) BulkUpsertPayload(upsertPayloadRequest UpsertPayloadRequest) error {
	//TODO implement me
	panic("implement me")
}

func (n *Eigenix) DeleteCollection(entity, model, variant string, version int) error {
	//TODO implement me
	panic("implement me")
}

func (n *Eigenix) UpdateIndexingThreshold(entity, model, variant string, version int, indexingThreshold string) error {
	//TODO implement me
	panic("implement me")
}

func (n *Eigenix) GetCollectionInfo(entity, model, variant string, version int) (*CollectionInfoResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n *Eigenix) GetReadCollectionInfo(entity, model, variant string, version int) (*CollectionInfoResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n *Eigenix) CreateFieldIndexes(entity, model, variant string, version int) error {
	//TODO implement me
	panic("implement me")
}

func EigenixHealthCheck(model string, variant string, grpcClient *grpcclient.GRPCClient) error {
	if grpcClient == nil {
		return fmt.Errorf("gRPC client is nil")
	}
	// You can add a health check call here if your NGT service supports it
	log.Info().Msgf("Eigenix gRPC client created for %s %s", model, variant)
	return nil
}

func (n *Eigenix) RefreshClients(key, value, eventType string) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Recovered from panic: %v", r)
		}
	}()
	if eventType == "DELETE" {
		return nil
	}

	entity, model, variant := n.extractEigenixKey(key)
	if entity == "" || model == "" || variant == "" {
		return nil
	}

	variantConfig, err := n.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for %s %s %s: %v", entity, model, variant, err)
		return nil
	}

	if variantConfig.VectorDbType != enums.EIGENIX {
		return nil
	}
	log.Error().Msgf("Eigenix entity config change detected - Key: %s, EventType: %s", key, eventType)
	clientKey := GetClientKey(entity, model, variant)

	if variantConfig.Enabled {
		vectorConfig := variantConfig.VectorDbConfig
		if _, exists := n.EigenixClients[clientKey]; !exists {
			var readGrpcClient *grpcclient.GRPCClient
			readGrpcClient = nil
			var writeGrpcClient *grpcclient.GRPCClient
			writeGrpcClient = nil
			if vectorConfig.ReadHost != "" {
				readGrpcClient, _ = createEigenixClient(vectorConfig, vectorConfig.ReadHost, clientKey)
				err = EigenixHealthCheck(model, variant, readGrpcClient)
			}
			if vectorConfig.WriteHost != "" {
				writeGrpcClient, _ = createEigenixClient(vectorConfig, vectorConfig.WriteHost, clientKey)
				err = EigenixHealthCheck(model, variant, writeGrpcClient)
			}
			if err != nil {
				log.Error().Msgf("Failed to create Eigenix client for %s %s: %v", model, variant, err)
				return err
			}
			n.EigenixClients[clientKey] = &EigenixClient{
				ReadClient:    readGrpcClient,
				WriteClient:   writeGrpcClient,
				ReadHost:      vectorConfig.ReadHost,
				WriteHost:     vectorConfig.WriteHost,
				Deadline:      vectorConfig.Http2Config.Deadline,
				WriteDeadline: vectorConfig.Http2Config.WriteDeadline,
			}
			log.Error().Msgf("Eigenix Read and Write Client created on runtime %s %s", model, variant)
		} else {
			// Handle client refresh logic similar to Qdrant
			if n.EigenixClients[clientKey].ReadHost != vectorConfig.ReadHost {
				var readGrpcClient *grpcclient.GRPCClient
				readGrpcClient = nil
				if vectorConfig.ReadHost != "" {
					readGrpcClient, _ = createEigenixClient(vectorConfig, vectorConfig.ReadHost, clientKey)
					err = EigenixHealthCheck(model, variant, readGrpcClient)
				}
				if err != nil {
					log.Error().Msgf("Failed to create Eigenix read client for %s %s: %v", model, variant, err)
					return err
				}
				n.EigenixClients[clientKey].ReadHost = vectorConfig.ReadHost
				n.EigenixClients[clientKey].ReadClient = readGrpcClient
				log.Error().Msgf("Eigenix Read Client Refreshed for %s %s with new Host %s", model, variant, vectorConfig.ReadHost)
			}
			if n.EigenixClients[clientKey].WriteHost != vectorConfig.WriteHost {
				var writeGrpcClient *grpcclient.GRPCClient
				writeGrpcClient = nil
				if vectorConfig.WriteHost != "" {
					writeGrpcClient, _ = createEigenixClient(vectorConfig, vectorConfig.WriteHost, clientKey)
					err = EigenixHealthCheck(model, variant, writeGrpcClient)
				}
				if err != nil {
					log.Error().Msgf("Failed to create Eigenix write client for %s %s: %v", model, variant, err)
					return err
				}
				n.EigenixClients[clientKey].WriteHost = vectorConfig.WriteHost
				n.EigenixClients[clientKey].WriteClient = writeGrpcClient
				log.Error().Msgf("Eigenix Write Client Refreshed for %s %s with new Host %s", model, variant, vectorConfig.WriteHost)
			}
		}
	}
	return nil
}

func (n *Eigenix) extractEigenixKey(key string) (string, string, string) {
	parts := strings.Split(key, "/")
	for _, part := range parts {
		if part == "vector-db-config" || part == "enabled" {
			return parts[4], parts[6], parts[8]
		}
	}
	return "", "", ""
}
