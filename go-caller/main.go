package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	retrieve "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/onfs/retrieve"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
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

const (
	// CONNECTION_POOL_SIZE defines the number of gRPC connections in the pool
	// Each HTTP/2 connection supports ~100 concurrent streams
	// With 16 connections: 16 * 100 = 1,600 concurrent streams capacity
	CONNECTION_POOL_SIZE = 16
)

// ClientPool manages a pool of gRPC clients for load distribution
type ClientPool struct {
	clients []retrieve.FeatureServiceClient
	conns   []*grpc.ClientConn
	counter uint64 // Atomic counter for round-robin selection
}

// NewClientPool creates a new pool of gRPC connections and clients
func NewClientPool(address string) (*ClientPool, error) {
	clients := make([]retrieve.FeatureServiceClient, 0, CONNECTION_POOL_SIZE)
	conns := make([]*grpc.ClientConn, 0, CONNECTION_POOL_SIZE)

	for i := 0; i < CONNECTION_POOL_SIZE; i++ {
		conn, err := grpc.NewClient(
			address,
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
			// Cleanup already created connections on error
			for _, c := range conns {
				_ = c.Close()
			}
			return nil, err
		}
		conns = append(conns, conn)
		clients = append(clients, retrieve.NewFeatureServiceClient(conn))
	}

	return &ClientPool{
		clients: clients,
		conns:   conns,
	}, nil
}

// Next returns the next client from the pool using round-robin distribution
func (p *ClientPool) Next() retrieve.FeatureServiceClient {
	idx := atomic.AddUint64(&p.counter, 1) - 1
	return p.clients[idx%uint64(len(p.clients))]
}

// Close closes all connections in the pool
func (p *ClientPool) Close() error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(p.conns))

	for i := range p.conns {
		wg.Add(1)
		go func(c *grpc.ClientConn) {
			defer wg.Done()
			if err := c.Close(); err != nil {
				errChan <- err
			}
		}(p.conns[i])
	}

	wg.Wait()
	close(errChan)

	// Collect any errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errs[0] // Return first error
	}
	return nil
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
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

	client := s.pool.Next()
	resp, err := client.RetrieveFeatures(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "response": resp})
}

func main() {
	log.Printf("Starting go-caller with connection pool (size: %d)", CONNECTION_POOL_SIZE)
	gin.SetMode(gin.ReleaseMode)

	pool, err := NewClientPool("online-feature-store-api.int.meesho.int:80")
	if err != nil {
		log.Fatalf("Failed to create gRPC connection pool: %v", err)
	}
	defer func() {
		log.Println("Closing gRPC connection pool...")
		if err := pool.Close(); err != nil {
			log.Printf("Error closing gRPC connection pool: %v", err)
		}
	}()

	state := &AppState{
		pool: pool,
		metadata: metadata.MD{
			"online-feature-store-auth-token": []string{"atishay"},
			"online-feature-store-caller-id":  []string{"test-3"},
		},
	}

	r := gin.New()
	r.POST("/retrieve-features", state.handler)

	srv := &http.Server{
		Addr:    ":8081",
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		log.Println("🚀 Go gRPC Client running on http://0.0.0.0:8081")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}
