package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
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
	conns   []*grpc.ClientConn // Store connections for cleanup
	counter uint64             // Atomic counter for round-robin selection
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
	// Set headers to encourage connection reuse
	c.Header("Connection", "keep-alive")
	c.Header("Keep-Alive", "timeout=300")

	var requestBody RetrieveFeaturesRequest
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use request context instead of Background() for better cancellation handling
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
	conns := make([]*grpc.ClientConn, 0, CONNECTION_POOL_SIZE)
	for i := 0; i < CONNECTION_POOL_SIZE; i++ {
		// Each Dial creates a separate connection
		// Note: grpc.Dial is non-blocking - connections are established lazily on first use
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
		conns = append(conns, conn)
		clients = append(clients, retrieve.NewFeatureServiceClient(conn))
		log.Printf("Created gRPC client %d/%d", i+1, CONNECTION_POOL_SIZE)
	}
	log.Println("Connection pool initialized with", CONNECTION_POOL_SIZE, "clients")

	pool := &ClientPool{
		clients: clients,
		conns:   conns,
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

	// Configure HTTP server for high concurrency with connection reuse
	// Key settings for preventing port exhaustion:
	// - Long IdleTimeout allows connections to be reused
	// - ReadTimeout/WriteTimeout prevent hung connections
	// - Keep-alive enabled for connection reuse
	srv := &http.Server{
		Addr:           ":8081",
		Handler:        r,
		ReadTimeout:    30 * time.Second,  // Increased to allow longer requests
		WriteTimeout:   30 * time.Second,  // Increased to allow longer responses
		IdleTimeout:    300 * time.Second, // 5 minutes - allows long connection reuse
		MaxHeaderBytes: 1 << 20,           // 1MB
	}

	// Enable HTTP keep-alive for connection reuse
	// This prevents port exhaustion by reusing connections
	// IdleTimeout (set above) controls the keep-alive period
	srv.SetKeepAlivesEnabled(true)

	// Create listener with SO_REUSEADDR and SO_REUSEPORT for better connection handling
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}

	// Use TCP keep-alive to detect dead connections and enable socket reuse
	tcpListener := listener.(*net.TCPListener)
	keepAliveListener := &keepAliveListener{
		TCPListener:     tcpListener,
		KeepAlivePeriod: 60 * time.Second, // Longer keep-alive for connection reuse
	}

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("🚀 Go gRPC Client running on http://0.0.0.0:8081")
		if err := srv.Serve(keepAliveListener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Close all gRPC connections
	pool.Close()
	log.Println("Server exited")
}

// keepAliveListener wraps TCPListener to enable TCP keep-alive
type keepAliveListener struct {
	*net.TCPListener
	KeepAlivePeriod time.Duration
}

func (ln *keepAliveListener) Accept() (net.Conn, error) {
	conn, err := ln.TCPListener.AcceptTCP()
	if err != nil {
		return nil, err
	}
	if err := conn.SetKeepAlive(true); err != nil {
		conn.Close()
		return nil, err
	}
	if err := conn.SetKeepAlivePeriod(ln.KeepAlivePeriod); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}
