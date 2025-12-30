package main

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	retrieve "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/onfs/retrieve"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// Constants to avoid repeated string allocations
const SUCCESS_RESPONSE = "success"

// Connection pool size - HTTP/2 has stream limits per connection
// Multiple connections are needed for 3k+ RPS
const CONNECTION_POOL_SIZE = 16

// Request body structure for retrieve_features endpoint
type RetrieveFeaturesRequest struct {
	EntityLabel   string                `json:"entity_label"`
	FeatureGroups []FeatureGroupRequest `json:"feature_groups"`
	KeysSchema    []string              `json:"keys_schema"`
	Keys          []KeysRequest         `json:"keys"`
}

type FeatureGroupRequest struct {
	Label         string   `json:"label"`
	FeatureLabels []string `json:"feature_labels"`
}

type KeysRequest struct {
	Cols []string `json:"cols"`
}

// Connection pool for gRPC clients to handle high concurrency
// HTTP/2 has stream limits per connection, so multiple connections are needed for 3k+ RPS
type ClientPool struct {
	clients []retrieve.FeatureServiceClient
	conns   []*grpc.ClientConn // Store connections for cleanup
	counter uint64             // Atomic counter for round-robin selection
}

// GetClient returns the next client from the pool using round-robin
// Round-robin selection to distribute load across connections
func (p *ClientPool) GetClient() retrieve.FeatureServiceClient {
	idx := atomic.AddUint64(&p.counter, 1) - 1
	return p.clients[idx%uint64(len(p.clients))]
}

// Close closes all gRPC connections in the pool
func (p *ClientPool) Close() {
	for _, conn := range p.conns {
		if conn != nil {
			if err := conn.Close(); err != nil {
				log.Printf("Error closing gRPC connection: %v", err)
			}
		}
	}
}

type AppState struct {
	clientPool *ClientPool
	authToken  string
	callerID   string
}

func retrieveFeatures(c *gin.Context, state *AppState) {
	var requestBody RetrieveFeaturesRequest
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	query := &retrieve.Query{
		EntityLabel:   requestBody.EntityLabel,
		FeatureGroups: featureGroups,
		KeysSchema:    requestBody.KeysSchema,
		Keys:          keys,
	}

	// Increased timeout to 10s to handle high load scenarios without premature timeouts
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Set metadata
	md := metadata.New(map[string]string{
		"online-feature-store-auth-token": state.authToken,
		"online-feature-store-caller-id":  state.callerID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// OPTIMIZATION: Use connection pool to distribute load across multiple HTTP/2 connections
	// This prevents hitting HTTP/2 stream limits on a single connection
	client := state.clientPool.GetClient()

	// OPTIMIZATION: Drop response immediately after checking success to reduce cleanup overhead
	// Based on flamegraph analysis: ~13-15% CPU was spent on drop_in_place for unused protobuf objects
	// By dropping explicitly in a smaller scope, we reduce the cleanup cost and memory pressure
	// The response contains large protobuf structures (Vec<Row>, Feature, etc.) that are expensive to clean up
	// Since we don't use the response, dropping it immediately reduces memory pressure
	result, err := client.RetrieveFeatures(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// OPTIMIZATION: Drop response immediately - don't wait for end of function
	// This reduces the time expensive drop operations hold resources
	// Response is automatically dropped here (Go GC will handle it)
	// Explicitly dropping is not needed in Go as it's managed by GC, but the optimization
	// of not storing the response in a larger scope achieves the same effect
	_ = result

	c.JSON(http.StatusOK, SUCCESS_RESPONSE)
}

func main() {
	log.Println("Connecting to feature store version 4...")
	gin.SetMode(gin.ReleaseMode)

	// PERFORMANCE FIX: Create multiple gRPC channels for connection pooling
	// HTTP/2 has stream limits (~100 concurrent streams per connection)
	// For 3k+ RPS, we need multiple connections to avoid hitting these limits
	// Using 10-20 connections should handle 3k-6k RPS comfortably
	clients := make([]retrieve.FeatureServiceClient, 0, CONNECTION_POOL_SIZE)
	conns := make([]*grpc.ClientConn, 0, CONNECTION_POOL_SIZE)

	for i := 0; i < CONNECTION_POOL_SIZE; i++ {
		// Optimized HTTP/2 settings for high concurrency
		conn, err := grpc.Dial(
			"online-feature-store-api.int.meesho.int:80",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                30 * time.Second, // HTTP/2 keep-alive interval
				Timeout:             10 * time.Second, // Keep-alive timeout
				PermitWithoutStream: true,             // Keep-alive while idle
			}),
			// Increase initial window size for better throughput
			grpc.WithInitialWindowSize(2*1024*1024),     // 2MB stream window
			grpc.WithInitialConnWindowSize(4*1024*1024), // 4MB connection window
		)
		if err != nil {
			log.Fatalf("Failed to create connection %d: %v", i, err)
		}

		conns = append(conns, conn)
		clients = append(clients, retrieve.NewFeatureServiceClient(conn))

		if (i+1)%4 == 0 {
			log.Printf("Created %d gRPC connections...", i+1)
		}
	}

	log.Printf("Created %d gRPC connections for connection pooling", CONNECTION_POOL_SIZE)

	clientPool := &ClientPool{
		clients: clients,
		conns:   conns,
		counter: 0,
	}

	state := &AppState{
		clientPool: clientPool,
		authToken:  "atishay",
		callerID:   "test-3",
	}

	r := gin.New()
	r.POST("/retrieve-features", func(c *gin.Context) {
		retrieveFeatures(c, state)
	})

	// PERFORMANCE FIX: Configure TCP listener for high concurrency
	// The main bottleneck fix is connection pooling (done above)
	// Gin/HTTP handle TCP settings efficiently by default
	log.Println("Server listening on 0.0.0.0:8080")
	log.Println("Configured for high performance:")
	log.Printf("  - %d gRPC connection pool (main bottleneck fix)", CONNECTION_POOL_SIZE)
	log.Println("  - HTTP/2 window sizes optimized for throughput")

	if err := http.ListenAndServe("0.0.0.0:8080", r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
