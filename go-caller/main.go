package main

import (
	"context"
	"log"
	"net/http"
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
	EntityLabel   string         `json:"entity_label" binding:"required"`
	FeatureGroups []FeatureGroup `json:"feature_groups" binding:"required"`
	KeysSchema    []string       `json:"keys_schema" binding:"required"`
	Keys          []KeysRequest  `json:"keys" binding:"required"`
}

type FeatureGroup struct {
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
		log.Printf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
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
		log.Printf("Failed to retrieve features: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve features"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func main() {
	log.Println("Starting go-caller with 4 threads version 4")
	gin.SetMode(gin.ReleaseMode)

	conn, err := grpc.Dial(
		"online-feature-store-api.int.meesho.int:80",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	state := &AppState{
		client: retrieve.NewFeatureServiceClient(conn),
		metadata: metadata.MD{
			"online-feature-store-auth-token": []string{"atishay"},
			"online-feature-store-caller-id":  []string{"test-3"},
		},
	}

	r := gin.New()
	r.POST("/retrieve-features", state.handler)
	log.Println("🚀 Go gRPC Client running on http://0.0.0.0:8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
