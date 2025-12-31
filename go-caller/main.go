package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
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

// AppState stores gRPC client and metadata
type AppState struct {
	client   retrieve.FeatureServiceClient
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

	_, err := s.client.RetrieveFeatures(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "response": "resp"})
}

func main() {
	log.Println("Starting go-caller with 4 threads version 6")
	gin.SetMode(gin.ReleaseMode)

	conn, err := grpc.NewClient(
		"online-feature-store-api.int.meesho.int:80",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing gRPC connection: %v", err)
		}
	}()

	state := &AppState{
		client: retrieve.NewFeatureServiceClient(conn),
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
