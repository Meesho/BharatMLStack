package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	retrieve "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/onfs/retrieve" // adjust path
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// ApiResponse matches your Rust ApiResponse struct
type ApiResponse struct {
	Success bool    `json:"success"`
	Data    *string `json:"data,omitempty"`
	Error   *string `json:"error,omitempty"`
	Message string  `json:"message"`
}

// AppState stores gRPC client
type AppState struct {
	client retrieve.FeatureServiceClient
}

// retrieveFeatures handles HTTP request
func (s *AppState) retrieveFeatures(c *gin.Context) {
	authToken := "atishay"
	callerID := "test-3"

	result, err := s.retrieveFeaturesInternal(authToken, callerID)
	if err != nil {
		log.Printf("❌ gRPC Error: %v", err)
		errMsg := err.Error()
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Data:    nil,
			Error:   &errMsg,
			Message: "Failed to retrieve features",
		})
		return
	}

	data := fmt.Sprintf("%v", result)
	c.JSON(http.StatusOK, ApiResponse{
		Success: true,
		Data:    &data,
		Error:   nil,
		Message: "Features retrieved successfully",
	})
}

// retrieveFeaturesInternal calls gRPC backend
func (s *AppState) retrieveFeaturesInternal(authToken, callerID string) (*retrieve.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attach metadata
	md := metadata.New(map[string]string{
		"online-feature-store-auth-token": authToken,
		"online-feature-store-caller-id":  callerID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Build gRPC request
	req := &retrieve.Query{
		EntityLabel: "catalog",
		FeatureGroups: []*retrieve.FeatureGroup{
			{
				Label: "derived_fp32",
				FeatureLabels: []string{
					"clicks_by_views_3_days",
				},
			},
		},
		KeysSchema: []string{"catalog_id"},
		Keys: []*retrieve.Keys{
			{Cols: []string{"176"}},
			{Cols: []string{"179"}},
		},
	}

	log.Println("📡 Retrieving features...")
	resp, err := s.client.RetrieveFeatures(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func main() {
	log.Println("Connecting to feature store...")

	// gRPC channel
	conn, err := grpc.Dial(
		"online-feature-store-api.int.meesho.int:80",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect gRPC: %v", err)
	}
	defer conn.Close()

	client := retrieve.NewFeatureServiceClient(conn)
	state := &AppState{client: client}

	// Gin server
	router := gin.Default()

	// Allow CORS permissive (similar to Rust)
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.POST("/retrieve-features", state.retrieveFeatures)

	port := "8080"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	log.Printf("🚀 Starting go-caller on http://0.0.0.0:%s\n", port)
	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
