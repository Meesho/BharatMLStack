package vector

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/Meesho/BharatMLStack/skye/internal/external/ngt/proto"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/pkg/grpcclient"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog/log"
)

const (
	EntityName       = "ngt"
	FeatureGroupName = "int_mappings"
	FeatureName      = "product_id"
	EntityKey        = "index_sequence_id"
)

var (
	ngtDb   Database
	ngtOnce sync.Once
)

type Ngt struct {
	NgtClients    map[string]*NgtClient
	configManager config.Manager
}

type NgtClient struct {
	ReadClient    *grpcclient.GRPCClient
	WriteClient   *grpcclient.GRPCClient
	ReadHost      string
	WriteHost     string
	Port          string
	Deadline      int
	WriteDeadline int
}

func initNgtInstance() Database {
	if ngtDb == nil {
		ngtOnce.Do(func() {
			ngtDb = createNgtInstance()
		})
	}
	return ngtDb
}

func createNgtInstance() *Ngt {
	ngtClients := make(map[string]*NgtClient)
	configManager := config.NewManager(config.DefaultVersion)
	entities, err := configManager.GetEntities()
	if err != nil {
		log.Panic().Msgf("Error getting vector configs from etcd: %v", err)
	}
	for entity, models := range entities {
		for model, modelConfig := range models.Models {
			for variant, variantConfig := range modelConfig.Variants {
				if variantConfig.VectorDbType != enums.NGT {
					continue
				}
				vectorConfig := variantConfig.VectorDbConfig
				key := GetClientKey(entity, model, variant)
				if variantConfig.Enabled {
					ngtClients[key] = &NgtClient{
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
							readGrpcClient, _ = createNgtClient(vectorConfig, vectorConfig.ReadHost, key)
						}
						_ = ngtHealthCheck(model, variant, readGrpcClient)
						ngtClients[key].ReadClient = readGrpcClient
						log.Error().Msgf("NGT Read Client Created for %s %s", model, variant)
					}
					if vectorConfig.WriteHost != "" {
						var writeGrpcClient *grpcclient.GRPCClient
						writeGrpcClient = nil
						if vectorConfig.WriteHost != "" {
							writeGrpcClient, _ = createNgtClient(vectorConfig, vectorConfig.WriteHost, key)
						}
						_ = ngtHealthCheck(model, variant, writeGrpcClient)
						ngtClients[key].WriteClient = writeGrpcClient
						log.Error().Msgf("NGT Write Client Created for %s %s", model, variant)
					}
				}
			}
		}
	}

	ngtInstance := &Ngt{
		NgtClients:    ngtClients,
		configManager: configManager,
	}

	return ngtInstance
}

// createNgtClient creates a new NGT gRPC client based on the given vector configuration.
func createNgtClient(vectorConfig config.VectorDbConfig, host string, envPrefix string) (*grpcclient.GRPCClient, error) {
	grpcClient := grpcclient.NewConnFromConfig(&grpcclient.Config{
		Host:                host,
		Port:                vectorConfig.Port,
		DeadLine:            vectorConfig.Http2Config.Deadline,
		PlainText:           vectorConfig.Http2Config.IsPlainText,
		LoadBalancingPolicy: "round_robin",
	}, envPrefix)
	return grpcClient, nil
}

func (n *Ngt) BatchQuery(bulkRequest *BatchQueryRequest, metricTags []string) (*BatchQueryResponse, error) {
	startTime := time.Now()
	metric.Incr("vector_db_batch_query", append([]string{"vector_db_type", "ngt"}, metricTags...))
	var similarCandidatesList = make(map[string][]*SimilarCandidate, len(bulkRequest.RequestList))
	batchQueryResponse := BatchQueryResponse{}
	for _, searchRequestDetails := range bulkRequest.RequestList {
		queryRequest := QueryRequest{
			Entity:               bulkRequest.Entity,
			Model:                bulkRequest.Model,
			Variant:              bulkRequest.Variant,
			Version:              bulkRequest.Version,
			SearchRequestDetails: nil,
		}
		queryRequest.SearchRequestDetails = searchRequestDetails
		queryResponse, err := n.Query(&queryRequest, metricTags)
		if err != nil {
			return nil, err
		}
		similarCandidatesList[searchRequestDetails.CacheKey] = queryResponse.Candidates
	}
	batchQueryResponse.SimilarCandidatesList = similarCandidatesList
	metric.Timing("vector_db_batch_query_latency", time.Since(startTime), append([]string{"vector_db_type", "ngt"}, metricTags...))
	return &batchQueryResponse, nil
}

func (n *Ngt) Query(query *QueryRequest, metricTags []string) (*QueryResponse, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Recovered from panic: %v", r)
		}
	}()
	startTime := time.Now()
	metric.Incr("vector_db_query", append([]string{"vector_db_type", "ngt"}, metricTags...))
	var epsilon float64
	variantConfig, _ := n.configManager.GetVariantConfig(query.Entity, query.Model, query.Variant)
	if variantConfig.VectorDbConfig.Params["epsilon"] == "" {
		epsilon = 0.1
	} else {
		epsilon, _ = strconv.ParseFloat(variantConfig.VectorDbConfig.Params["epsilon"], 64)
	}
	var radius float64
	if variantConfig.VectorDbConfig.Params["radius"] == "" {
		radius = 10000.0
	} else {
		radius, _ = strconv.ParseFloat(variantConfig.VectorDbConfig.Params["radius"], 64)
	}

	// Convert []float32 to []float64
	embedding := make([]float64, len(query.SearchRequestDetails.Embedding))
	for i, v := range query.SearchRequestDetails.Embedding {
		embedding[i] = float64(v)
	}

	// Create gRPC request
	searchRequest := &pb.SearchRequest{
		Embedding: embedding,
		K:         int32(query.SearchRequestDetails.CandidateLimit),
		Epsilon:   epsilon,
		Radius:    radius,
	}

	client := n.getClient(query.Entity, query.Model, query.Variant)
	if client == nil || client.ReadClient == nil {
		return nil, fmt.Errorf("NGT client not found for %s %s %s", query.Entity, query.Model, query.Variant)
	}

	// Create NGT service client
	embeddingClient := pb.NewEmbeddingServiceClient(client.ReadClient.Conn)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.Deadline)*time.Millisecond)
	defer cancel()

	// Make gRPC call
	response, err := embeddingClient.Search(ctx, searchRequest)
	if err != nil {
		log.Error().Msgf("Error while getting response from NGT gRPC service, error - %v", err)
		metric.Count("vector_db_query_failure", 1, append([]string{"vector_db_type", "ngt"}, metricTags...))
		return nil, err
	}

	metric.Timing("vector_db_query_latency", time.Since(startTime),
		append([]string{"vector_db_type", "ngt"}, metricTags...))

	// Convert response to QueryResponse format
	return mapNgtGrpcToQueryResponse(query, response), nil
}

func mapNgtGrpcToQueryResponse(query *QueryRequest, ngtResp *pb.SearchResponse) *QueryResponse {
	queryResp := &QueryResponse{}

	for _, result := range ngtResp.Results {
		candidate := &SimilarCandidate{
			Id:      fmt.Sprintf("%d", result.Id),
			Score:   float32(result.Distance),
			Payload: map[string]string{},
		}
		queryResp.Candidates = append(queryResp.Candidates, candidate)
	}
	if len(ngtResp.Results) == 0 {
		return queryResp
	}

	return queryResp
}

func (n *Ngt) getClient(entity, model, variant string) *NgtClient {
	return n.NgtClients[GetClientKey(entity, model, variant)]
}

func (n *Ngt) CreateCollection(entity string, model string, variant string, version int) error {
	//TODO implement me
	panic("implement me")
}

func (n *Ngt) BulkUpsert(upsertRequest UpsertRequest) error {
	//TODO implement me
	panic("implement me")
}

func (n *Ngt) BulkDelete(deleteRequest DeleteRequest) error {
	//TODO implement me
	panic("implement me")
}

func (n *Ngt) BulkUpsertPayload(upsertPayloadRequest UpsertPayloadRequest) error {
	//TODO implement me
	panic("implement me")
}

func (n *Ngt) DeleteCollection(entity, model, variant string, version int) error {
	//TODO implement me
	panic("implement me")
}

func (n *Ngt) UpdateIndexingThreshold(entity, model, variant string, version int, indexingThreshold string) error {
	//TODO implement me
	panic("implement me")
}

func (n *Ngt) GetCollectionInfo(entity, model, variant string, version int) (*CollectionInfoResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n *Ngt) GetReadCollectionInfo(entity, model, variant string, version int) (*CollectionInfoResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n *Ngt) CreateFieldIndexes(entity, model, variant string, version int) error {
	//TODO implement me
	panic("implement me")
}

func ngtHealthCheck(model string, variant string, grpcClient *grpcclient.GRPCClient) error {
	if grpcClient == nil {
		return fmt.Errorf("gRPC client is nil")
	}
	// You can add a health check call here if your NGT service supports it
	log.Info().Msgf("NGT gRPC client created for %s %s", model, variant)
	return nil
}

func (n *Ngt) RefreshClients(key, value, eventType string) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Recovered from panic: %v", r)
		}
	}()
	if eventType == "DELETE" {
		return nil
	}

	entity, model, variant := n.extractNgtKey(key)
	if entity == "" || model == "" || variant == "" {
		return nil
	}

	variantConfig, err := n.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for %s %s %s: %v", entity, model, variant, err)
		return nil
	}

	if variantConfig.VectorDbType != enums.NGT {
		return nil
	}
	log.Error().Msgf("NGT entity config change detected - Key: %s, EventType: %s", key, eventType)
	clientKey := GetClientKey(entity, model, variant)

	if variantConfig.Enabled {
		vectorConfig := variantConfig.VectorDbConfig
		if _, exists := n.NgtClients[clientKey]; !exists {
			var readGrpcClient *grpcclient.GRPCClient
			readGrpcClient = nil
			var writeGrpcClient *grpcclient.GRPCClient
			writeGrpcClient = nil
			if vectorConfig.ReadHost != "" {
				readGrpcClient, _ = createNgtClient(vectorConfig, vectorConfig.ReadHost, clientKey)
				err = ngtHealthCheck(model, variant, readGrpcClient)
			}
			if vectorConfig.WriteHost != "" {
				writeGrpcClient, _ = createNgtClient(vectorConfig, vectorConfig.WriteHost, clientKey)
				err = ngtHealthCheck(model, variant, writeGrpcClient)
			}
			if err != nil {
				log.Error().Msgf("Failed to create NGT client for %s %s: %v", model, variant, err)
				return err
			}
			n.NgtClients[clientKey] = &NgtClient{
				ReadClient:    readGrpcClient,
				WriteClient:   writeGrpcClient,
				ReadHost:      vectorConfig.ReadHost,
				WriteHost:     vectorConfig.WriteHost,
				Deadline:      vectorConfig.Http2Config.Deadline,
				WriteDeadline: vectorConfig.Http2Config.WriteDeadline,
			}
			log.Error().Msgf("NGT Read and Write Client created on runtime %s %s", model, variant)
		} else {
			// Handle client refresh logic similar to Qdrant
			if n.NgtClients[clientKey].ReadHost != vectorConfig.ReadHost {
				var readGrpcClient *grpcclient.GRPCClient
				readGrpcClient = nil
				if vectorConfig.ReadHost != "" {
					readGrpcClient, _ = createNgtClient(vectorConfig, vectorConfig.ReadHost, clientKey)
					err = ngtHealthCheck(model, variant, readGrpcClient)
				}
				if err != nil {
					log.Error().Msgf("Failed to create NGT read client for %s %s: %v", model, variant, err)
					return err
				}
				n.NgtClients[clientKey].ReadHost = vectorConfig.ReadHost
				n.NgtClients[clientKey].ReadClient = readGrpcClient
				log.Error().Msgf("NGT Read Client Refreshed for %s %s with new Host %s", model, variant, vectorConfig.ReadHost)
			}
			if n.NgtClients[clientKey].WriteHost != vectorConfig.WriteHost {
				var writeGrpcClient *grpcclient.GRPCClient
				writeGrpcClient = nil
				if vectorConfig.WriteHost != "" {
					writeGrpcClient, _ = createNgtClient(vectorConfig, vectorConfig.WriteHost, clientKey)
					err = ngtHealthCheck(model, variant, writeGrpcClient)
				}
				if err != nil {
					log.Error().Msgf("Failed to create NGT write client for %s %s: %v", model, variant, err)
					return err
				}
				n.NgtClients[clientKey].WriteHost = vectorConfig.WriteHost
				n.NgtClients[clientKey].WriteClient = writeGrpcClient
				log.Error().Msgf("NGT Write Client Refreshed for %s %s with new Host %s", model, variant, vectorConfig.WriteHost)
			}
		}
	}
	return nil
}

func (n *Ngt) extractNgtKey(key string) (string, string, string) {
	parts := strings.Split(key, "/")
	for _, part := range parts {
		if part == "vector-db-config" || part == "enabled" {
			return parts[4], parts[6], parts[8]
		}
	}
	return "", "", ""
}
