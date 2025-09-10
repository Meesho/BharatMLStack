package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // Enable pprof endpoints
	"runtime"
	"sync/atomic"
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

type Metrics struct {
	RequestsTotal       int64     `json:"requests_total"`
	RequestsSuccess     int64     `json:"requests_success"`
	RequestsError       int64     `json:"requests_error"`
	AvgResponseTime     float64   `json:"avg_response_time_ms"`
	ConnectionTime      float64   `json:"avg_connection_time_ms"`
	GrpcCallTime        float64   `json:"avg_grpc_call_time_ms"`
	LastUpdated         time.Time `json:"last_updated"`
}

var (
	requestsTotal       int64
	requestsSuccess     int64
	requestsError       int64
	totalResponseTime   int64
	totalConnectionTime int64
	totalGrpcCallTime   int64
)

func metricsHandler(c *gin.Context) {
	total := atomic.LoadInt64(&requestsTotal)
	metrics := Metrics{
		RequestsTotal:   total,
		RequestsSuccess: atomic.LoadInt64(&requestsSuccess),
		RequestsError:   atomic.LoadInt64(&requestsError),
		LastUpdated:     time.Now(),
	}

	if total > 0 {
		metrics.AvgResponseTime = float64(atomic.LoadInt64(&totalResponseTime)) / float64(total) / 1e6 // ns to ms
		metrics.ConnectionTime = float64(atomic.LoadInt64(&totalConnectionTime)) / float64(total) / 1e6
		metrics.GrpcCallTime = float64(atomic.LoadInt64(&totalGrpcCallTime)) / float64(total) / 1e6
	}

	c.JSON(http.StatusOK, metrics)
}

func healthHandler(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"memory": gin.H{
			"alloc_mb":      float64(m.Alloc) / 1024 / 1024,
			"total_alloc_mb": float64(m.TotalAlloc) / 1024 / 1024,
			"sys_mb":       float64(m.Sys) / 1024 / 1024,
			"gc_cycles":    m.NumGC,
		},
		"goroutines": runtime.NumGoroutine(),
	})
}

func retrieveFeatures(c *gin.Context) {
	start := time.Now()
	atomic.AddInt64(&requestsTotal, 1)
	
	log.Printf("🔥 Processing feature retrieval request")
	
	result, connTime, grpcTime, err := retrieveFeaturesInternal()
	duration := time.Since(start)
	
	// Update metrics
	atomic.AddInt64(&totalResponseTime, duration.Nanoseconds())
	atomic.AddInt64(&totalConnectionTime, connTime.Nanoseconds())
	atomic.AddInt64(&totalGrpcCallTime, grpcTime.Nanoseconds())
	
	if err != nil {
		atomic.AddInt64(&requestsError, 1)
		log.Printf("❌ Request failed in %v: %v", duration, err)
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Error:   err.Error(),
			Message: "Failed to retrieve features",
		})
		return
	}

	atomic.AddInt64(&requestsSuccess, 1)
	log.Printf("✅ Request successful in %v (conn: %v, grpc: %v)", duration, connTime, grpcTime)
	c.JSON(http.StatusOK, ApiResponse{
		Success: true,
		Data:    result.String(),
		Message: "Features retrieved successfully",
	})
}

func retrieveFeaturesInternal() (*retrieve.Result, time.Duration, time.Duration, error) {
	log.Printf("🔌 Attempting to connect to the feature store...")

	// Measure connection time
	connStart := time.Now()
	conn, err := grpc.NewClient(
		"online-feature-store-api.int.meesho.int:80",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("did not connect: %v", err)
	}
	defer conn.Close()
	connTime := time.Since(connStart)

	client := retrieve.NewFeatureServiceClient(conn)
	log.Printf("✅ gRPC connection established in %v", connTime)

	md := metadata.New(map[string]string{
		"online-feature-store-auth-token": "atishay",
		"online-feature-store-caller-id":  "test-3",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Measure gRPC call time
	grpcStart := time.Now()
	resp, err := client.RetrieveFeatures(ctx, &retrieve.Query{
		EntityLabel: "catalog",
		FeatureGroups: []*retrieve.FeatureGroup{
			{
				Label:         "derived_fp32",
				FeatureLabels: []string{"clicks_by_views_3_days"},
			},
		},
		KeysSchema: []string{"catalog_id"},
		Keys: []*retrieve.Keys{
			{Cols: []string{"176"}},
			{Cols: []string{"179"}},
		},
	})
	grpcTime := time.Since(grpcStart)
	
	if err != nil {
		log.Printf("❌ gRPC call failed in %v: %v", grpcTime, err)
		return nil, connTime, grpcTime, fmt.Errorf("could not retrieve features: %v", err)
	}

	log.Printf("✅ gRPC call completed in %v", grpcTime)
	return resp, connTime, grpcTime, nil
}

func main() {
	// Start pprof server on a separate port for profiling
	go func() {
		log.Printf("🔍 Starting pprof server on :6060")
		log.Printf("📊 CPU Profile: http://localhost:6060/debug/pprof/profile")
		log.Printf("🧠 Heap Profile: http://localhost:6060/debug/pprof/heap")
		log.Printf("🔄 Goroutine Profile: http://localhost:6060/debug/pprof/goroutine")
		log.Fatal(http.ListenAndServe(":6060", nil))
	}()

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

	// Add routes
	r.POST("/retrieve-features", retrieveFeatures)
	r.GET("/metrics", metricsHandler)
	r.GET("/health", healthHandler)

	// Start main server
	log.Printf("🚀 Starting Go Feature Store API server on http://0.0.0.0:8081")
	log.Printf("📊 Metrics: http://localhost:8081/metrics")
	log.Printf("💚 Health: http://localhost:8081/health")
	log.Printf("🔍 Profiling: http://localhost:6060/debug/pprof/")
	log.Fatal(r.Run(":8081"))
}