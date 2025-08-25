package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	retrieve "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/onfs/retrieve"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type ApiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message"`
}

func retrieveFeatures(c *gin.Context) {
	result, err := retrieveFeaturesInternal()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Error:   err.Error(),
			Message: "Failed to retrieve features",
		})
		return
	}

	c.JSON(http.StatusOK, ApiResponse{
		Success: true,
		Data:    result.String(),
		Message: "Features retrieved successfully",
	})
}

func retrieveFeaturesInternal() (*retrieve.Result, error) {
	fmt.Println("Attempting to connect to the feature store...")

	conn, err := grpc.NewClient(
		"online-feature-store-api-mp.prd.meesho.int:80",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		return nil, fmt.Errorf("did not connect: %v", err)
	}
	defer conn.Close()

	client := retrieve.NewFeatureServiceClient(conn)

	fmt.Println("Connection successful. Retrieving features...")

	md := metadata.New(map[string]string{
		"online-feature-store-auth-token": "test",
		"online-feature-store-caller-id":  "model-proxy-service-experiment",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := client.RetrieveFeatures(ctx, &retrieve.Query{
		EntityLabel: "catalog",
		FeatureGroups: []*retrieve.FeatureGroup{
			{
				Label:         "derived_2_fp32",
				FeatureLabels: []string{"sbid_value"},
			},
			{
				Label: "derived_fp16",
				FeatureLabels: []string{
					"search__organic_clicks_by_views_3_days_percentile",
					"search__organic_clicks_by_views_5_days_percentile",
				},
			},
		},
		KeysSchema: []string{"catalog_id"},
		Keys: []*retrieve.Keys{
			{Cols: []string{"176"}},
			{Cols: []string{"179"}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve features: %v", err)
	}

	return resp, nil
}

func main() {
	// Create Gin router
	r := gin.Default()

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Header("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Add route
	r.POST("/retrieve-features", retrieveFeatures)

	// Start server
	fmt.Println("Starting Go Feature Store API server on http://0.0.0.0:8081")
	log.Fatal(r.Run(":8081"))
}