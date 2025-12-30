package main

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	retrieve "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/onfs/retrieve" // adjust path
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

const (
	// Connection pool size - each connection can handle ~100 concurrent streams
	// With 16 connections, we can handle ~1600 concurrent requests
	CONNECTION_POOL_SIZE = 16
)

// Request body structures for retrieve_features endpoint
type RetrieveFeaturesRequest struct {
	EntityLabel   string                `json:"entity_label" binding:"required"`
	FeatureGroups []FeatureGroupRequest `json:"feature_groups" binding:"required"`
	KeysSchema    []string              `json:"keys_schema" binding:"required"`
	Keys          []KeysRequest         `json:"keys" binding:"required"`
}

type FeatureGroupRequest struct {
	Label         string   `json:"label" binding:"required"`
	FeatureLabels []string `json:"feature_labels" binding:"required"`
}

type KeysRequest struct {
	Cols []string `json:"cols" binding:"required"`
}

// ClientPool manages a pool of gRPC clients for connection pooling
type ClientPool struct {
	clients []retrieve.FeatureServiceClient
	counter uint64 // Atomic counter for round-robin selection
}

// Next returns the next client from the pool using round-robin
func (p *ClientPool) Next() retrieve.FeatureServiceClient {
	idx := atomic.AddUint64(&p.counter, 1) - 1
	return p.clients[idx%uint64(len(p.clients))]
}

// AppState stores gRPC client pool and metadata
type AppState struct {
	pool     *ClientPool
	metadata metadata.MD
}

func (s *AppState) handler(c *gin.Context) {
	var requestBody RetrieveFeaturesRequest
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, s.metadata)

	// Convert request body to protobuf Query
	featureGroups := make([]*retrieve.FeatureGroup, 0, len(requestBody.FeatureGroups))
	for _, fg := range requestBody.FeatureGroups {
		featureGroups = append(featureGroups, &retrieve.FeatureGroup{
			Label:         fg.Label,
			FeatureLabels: fg.FeatureLabels,
		})
	}

	keys := make([]*retrieve.Keys, 0, len(requestBody.Keys))
	for _, k := range requestBody.Keys {
		keys = append(keys, &retrieve.Keys{
			Cols: k.Cols,
		})
	}

	req := &retrieve.Query{
		EntityLabel:   requestBody.EntityLabel,
		FeatureGroups: featureGroups,
		KeysSchema:    requestBody.KeysSchema,
		Keys:          keys,
	}

	// Get next client from pool using round-robin
	client := s.pool.Next()
	_, err := client.RetrieveFeatures(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, "success")
}

func main() {
	log.Println("Starting go-caller with connection pooling (pool size:", CONNECTION_POOL_SIZE, ")")
	gin.SetMode(gin.ReleaseMode)

	// Create connection pool
	clients := make([]retrieve.FeatureServiceClient, 0, CONNECTION_POOL_SIZE)
	for i := 0; i < CONNECTION_POOL_SIZE; i++ {
		conn, err := grpc.Dial(
			"online-feature-store-api.int.meesho.int:80",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                30 * time.Second,
				Timeout:             10 * time.Second,
				PermitWithoutStream: true,
			}),
			grpc.WithInitialWindowSize(2*1024*1024),     // 2MB stream window
			grpc.WithInitialConnWindowSize(4*1024*1024), // 4MB connection window
		)
		if err != nil {
			log.Fatalf("Failed to create connection %d: %v", i, err)
		}
		clients = append(clients, retrieve.NewFeatureServiceClient(conn))
	}

	pool := &ClientPool{
		clients: clients,
		counter: 0,
	}

	state := &AppState{
		pool: pool,
		metadata: metadata.MD{
			"online-feature-store-auth-token": []string{"atishay"},
			"online-feature-store-caller-id":  []string{"test-3"},
		},
	}

	r := gin.New()
	r.POST("/retrieve-features", state.handler)

	log.Println("🚀 Go gRPC Client running on http://0.0.0.0:8081")
	r.Run(":8081")
}
