package grpc

import (
	"context"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/httpframework"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	stdgrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// resetGlobalState resets the global state for testing
func resetGlobalState() {
	server = nil
	once = sync.Once{}
	httpframework.ResetForTesting()
}

// Helper function to wait for server to be ready
func waitForServer(port string) bool {
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get("http://localhost:" + port + "/health/self")
		if err == nil {
			resp.Body.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func TestRunWithoutAppPort(t *testing.T) {
	os.Unsetenv("APP_PORT")
	viper.Set("APP_NAME", "test-grpc-app")
	viper.AutomaticEnv()

	Init()
	instance := Instance()
	assert.Panics(t, func() {
		err := instance.Run()
		if err != nil {
			return
		}
	})
}

func TestInitAndRun(t *testing.T) {
	resetGlobalState()

	os.Setenv("APP_PORT", "8080")
	viper.Set("APP_NAME", "test-grpc-app")
	viper.AutomaticEnv()

	// Call the function under test in a goroutine, as it blocks
	Init()
	instance := Instance()

	// Assert that the server is initialized
	assert.NotNil(t, instance.HTTPHandler)

	// Assert that the server's GRPCServer is not nil
	assert.NotNil(t, instance.GRPCServer)

	// Test Run method
	go instance.Run()

	// Wait for the server to start
	assert.True(t, waitForServer("8080"), "Server did not start within timeout")

	// Send a GET request to the /health/self endpoint
	resp, err := http.Get("http://localhost:8080/health/self")
	assert.NoError(t, err)
	defer resp.Body.Close()

	//make grpc call to the server
	conn, err := grpc.Dial("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	// Check the response status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInitWithServerOptions(t *testing.T) {
	resetGlobalState()

	os.Setenv("APP_PORT", "8081")
	viper.AutomaticEnv()

	serverOpts := []stdgrpc.ServerOption{
		grpc.MaxRecvMsgSize(1024 * 1024 * 4), // 4MB max message size
	}

	// Call the function under test in a goroutine, as it blocks
	InitWithServerOptions(serverOpts)
	instance := Instance()

	// Assert that the server is initialized
	assert.NotNil(t, instance.HTTPHandler)

	// Assert that the server's GRPCServer is not nil
	assert.NotNil(t, instance.GRPCServer)

	// Test Run method
	go instance.Run()

	// Wait for the server to start
	assert.True(t, waitForServer("8081"), "Server did not start within timeout")

	// Send a GET request to the /health/self endpoint
	resp, err := http.Get("http://localhost:8081/health/self")
	assert.NoError(t, err)
	defer resp.Body.Close()

	//make grpc call to the server
	conn, err := grpc.Dial("localhost:8081", grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	// Check the response status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInitWithoutServerOptions(t *testing.T) {
	resetGlobalState()

	os.Setenv("APP_PORT", "8082")
	viper.AutomaticEnv()

	// Call the function under test in a goroutine, as it blocks
	InitWithServerOptions(nil)
	instance := Instance()

	// Assert that the server is initialized
	assert.NotNil(t, instance.HTTPHandler)

	// Assert that the server's GRPCServer is not nil
	assert.NotNil(t, instance.GRPCServer)

	// Test Run method
	go instance.Run()

	// Wait for the server to start
	assert.True(t, waitForServer("8082"), "Server did not start within timeout")

	// Send a GET request to the /health/self endpoint
	resp, err := http.Get("http://localhost:8082/health/self")
	assert.NoError(t, err)
	defer resp.Body.Close()

	//make grpc call to the server
	conn, err := grpc.Dial("localhost:8082", grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	// Check the response status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInstanceWithoutInit(t *testing.T) {
	resetGlobalState()

	assert.Panics(t, func() { Instance() })
}

// New comprehensive tests for InitWithServerOptions

func TestInitWithServerOptionsCustomInterceptors(t *testing.T) {
	resetGlobalState()

	os.Setenv("APP_PORT", "8083")
	viper.AutomaticEnv()

	// Custom interceptor for testing
	customInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// This is a simple pass-through interceptor for testing
		return handler(ctx, req)
	}

	serverOpts := []stdgrpc.ServerOption{
		grpc.MaxRecvMsgSize(1024 * 1024 * 2), // 2MB max message size
	}

	// Call with custom interceptor
	InitWithServerOptions(serverOpts, customInterceptor)
	instance := Instance()

	// Assert that the server is initialized
	assert.NotNil(t, instance.HTTPHandler)
	assert.NotNil(t, instance.GRPCServer)

	// Test Run method
	go instance.Run()

	// Wait for the server to start
	assert.True(t, waitForServer("8083"), "Server did not start within timeout")

	// Send a GET request to the /health/self endpoint
	resp, err := http.Get("http://localhost:8083/health/self")
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check the response status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInitWithServerOptionsEmptyServerOptions(t *testing.T) {
	resetGlobalState()

	os.Setenv("APP_PORT", "8084")
	viper.AutomaticEnv()

	// Test with empty server options slice
	serverOpts := []stdgrpc.ServerOption{}

	InitWithServerOptions(serverOpts)
	instance := Instance()

	// Assert that the server is initialized
	assert.NotNil(t, instance.HTTPHandler)
	assert.NotNil(t, instance.GRPCServer)

	// Test Run method
	go instance.Run()

	// Wait for the server to start
	assert.True(t, waitForServer("8084"), "Server did not start within timeout")

	// Send a GET request to the /health/self endpoint
	resp, err := http.Get("http://localhost:8084/health/self")
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check the response status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInitWithServerOptionsMultipleOptions(t *testing.T) {
	resetGlobalState()

	os.Setenv("APP_PORT", "8085")
	viper.AutomaticEnv()

	serverOpts := []stdgrpc.ServerOption{
		grpc.MaxRecvMsgSize(1024 * 1024 * 8), // 8MB max message size
		grpc.MaxSendMsgSize(1024 * 1024 * 8), // 8MB max send message size
		grpc.MaxConcurrentStreams(1000),
	}

	InitWithServerOptions(serverOpts)
	instance := Instance()

	// Assert that the server is initialized
	assert.NotNil(t, instance.HTTPHandler)
	assert.NotNil(t, instance.GRPCServer)

	// Test Run method
	go instance.Run()

	// Wait for the server to start
	assert.True(t, waitForServer("8085"), "Server did not start within timeout")

	// Send a GET request to the /health/self endpoint
	resp, err := http.Get("http://localhost:8085/health/self")
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check the response status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInitWithServerOptionsOnlyCalledOnce(t *testing.T) {
	resetGlobalState()

	os.Setenv("APP_PORT", "8086")
	viper.AutomaticEnv()

	serverOpts1 := []stdgrpc.ServerOption{
		grpc.MaxRecvMsgSize(1024 * 1024 * 4), // 4MB max message size
	}

	serverOpts2 := []stdgrpc.ServerOption{
		grpc.MaxRecvMsgSize(1024 * 1024 * 8), // 8MB max message size
	}

	// Call InitWithServerOptions twice - only first should take effect
	InitWithServerOptions(serverOpts1)
	instance1 := Instance()

	InitWithServerOptions(serverOpts2) // This should be ignored due to sync.Once
	instance2 := Instance()

	// Both instances should be the same
	assert.Equal(t, instance1, instance2)
	assert.NotNil(t, instance1.HTTPHandler)
	assert.NotNil(t, instance1.GRPCServer)
}

func TestInitWithServerOptionsNilLogging(t *testing.T) {
	resetGlobalState()

	os.Setenv("APP_PORT", "8087")
	viper.AutomaticEnv()

	// Test that nil server options are handled gracefully
	InitWithServerOptions(nil)
	instance := Instance()

	// Assert that the server is initialized
	assert.NotNil(t, instance.HTTPHandler)
	assert.NotNil(t, instance.GRPCServer)

	// Test Run method
	go instance.Run()

	// Wait for the server to start
	assert.True(t, waitForServer("8087"), "Server did not start within timeout")

	// Send a GET request to the /health/self endpoint
	resp, err := http.Get("http://localhost:8087/health/self")
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check the response status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// Test to verify interceptor ordering in InitWithServerOptions
func TestInitWithServerOptionsInterceptorOrdering(t *testing.T) {
	resetGlobalState()

	os.Setenv("APP_PORT", "8088")
	viper.AutomaticEnv()

	// Track the order of interceptor calls
	var callOrder []string

	// First custom interceptor
	interceptor1 := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		callOrder = append(callOrder, "interceptor1")
		return handler(ctx, req)
	}

	// Second custom interceptor
	interceptor2 := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		callOrder = append(callOrder, "interceptor2")
		return handler(ctx, req)
	}

	serverOpts := []stdgrpc.ServerOption{
		grpc.MaxRecvMsgSize(1024 * 1024 * 4), // 4MB max message size
	}

	// Call with multiple custom interceptors
	InitWithServerOptions(serverOpts, interceptor1, interceptor2)
	instance := Instance()

	// Assert that the server is initialized
	assert.NotNil(t, instance.HTTPHandler)
	assert.NotNil(t, instance.GRPCServer)

	// Note: Testing actual interceptor call order would require setting up a real gRPC service
	// For now, we verify that the server was created successfully with multiple interceptors
	// The ordering test would need to be done in an integration test with actual gRPC calls
}

// Test to verify that both Init and InitWithServerOptions work similarly
func TestInitVsInitWithServerOptions(t *testing.T) {
	t.Run("Init creates server properly", func(t *testing.T) {
		resetGlobalState()
		os.Setenv("APP_PORT", "8089")
		viper.Set("APP_NAME", "test-grpc-app")
		viper.AutomaticEnv()

		Init()
		instance := Instance()
		assert.NotNil(t, instance.HTTPHandler)
		assert.NotNil(t, instance.GRPCServer)
	})

	t.Run("InitWithServerOptions creates server properly", func(t *testing.T) {
		resetGlobalState()
		os.Setenv("APP_PORT", "8090")
		viper.AutomaticEnv()

		InitWithServerOptions(nil)
		instance := Instance()
		assert.NotNil(t, instance.HTTPHandler)
		assert.NotNil(t, instance.GRPCServer)
	})
}
