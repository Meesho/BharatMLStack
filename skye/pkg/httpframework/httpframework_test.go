package httpframework

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/Meesho/BharatMLStack/skye/pkg/tracing"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// reset restores the httpframework to its initial state for isolated tests.
func reset() {
	router = nil
	once = sync.Once{}
}

// runTestInSubprocess is a helper to test functions that call log.Fatal.
func runTestInSubprocess(t *testing.T, testName, expectedMsg string) {
	cmd := exec.Command(os.Args[0], "-test.run="+testName)
	cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS=1")
	output, err := cmd.CombinedOutput()

	exitErr, ok := err.(*exec.ExitError)
	assert.True(t, ok, "Expected command to exit with an error")
	assert.False(t, exitErr.Success(), "Expected command to fail")
	assert.Contains(t, string(output), expectedMsg)
}

func TestInit(t *testing.T) {
	// Subprocess fatal tests
	if os.Getenv("GO_TEST_SUBPROCESS") == "" {
		t.Run("should fatal if APP_NAME is not set", func(t *testing.T) {
			runTestInSubprocess(t, "TestAppNameFatal", "APP_NAME cannot be empty!!!")
		})
	}

	t.Run("should initialize router successfully", func(t *testing.T) {
		reset()
		defer reset()

		viper.Set("APP_NAME", "test-app")
		defer viper.Reset()

		Init()
		assert.NotNil(t, router)
		// 3 default middlewares: otelgin, httplogger, httprecovery
		assert.Len(t, router.Handlers, 3)
	})

	t.Run("should initialize with tracer and serve endpoint", func(t *testing.T) {
		reset()
		defer reset()

		viper.Set("APP_NAME", "test-app-with-tracer")
		os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
		tracing.Init()
		defer tracing.ShutdownTracer()
		defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		defer viper.Reset()

		Init()
		assert.NotNil(t, router)

		// Define an endpoint and test it
		router.GET("/ping", func(c *gin.Context) {
			c.String(http.StatusOK, "pong")
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/ping", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "pong", w.Body.String())
	})

	t.Run("should initialize without tracer and serve endpoint", func(t *testing.T) {
		reset()
		defer reset()

		viper.Set("APP_NAME", "test-app-no-tracer")
		defer viper.Reset()

		Init() // No tracer initialized
		assert.NotNil(t, router)

		// Define an endpoint and test it
		router.GET("/ping", func(c *gin.Context) {
			c.String(http.StatusOK, "pong")
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/ping", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "pong", w.Body.String())
	})

	t.Run("should be idempotent", func(t *testing.T) {
		reset()
		defer reset()

		viper.Set("APP_NAME", "test-app-idempotent")
		defer viper.Reset()

		Init()
		firstInstance := Instance()
		assert.Len(t, firstInstance.Handlers, 3)

		// Call Init again with another middleware
		Init(func(c *gin.Context) {})
		secondInstance := Instance()

		assert.Same(t, firstInstance, secondInstance)
		assert.Len(t, secondInstance.Handlers, 3, "Init should not add more middlewares on subsequent calls")
	})
}

// TestAppNameFatal is the actual test that will be run in a subprocess.
func TestAppNameFatal(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") != "1" {
		t.Skip("skipping subprocess test in main process")
	}
	reset()
	viper.Set("APP_NAME", "")
	Init() // This should call log.Fatal
}

func TestInstance(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") == "" {
		t.Run("should fatal if router is not initialized", func(t *testing.T) {
			runTestInSubprocess(t, "TestInstanceFatal", "Router not initialized")
		})
	}

	t.Run("should return the gin engine instance", func(t *testing.T) {
		reset()
		defer reset()

		viper.Set("APP_NAME", "test-app")
		defer viper.Reset()

		Init()
		instance := Instance()
		assert.NotNil(t, instance)
		assert.Equal(t, router, instance)
	})
}

// TestInstanceFatal is the actual test that will be run in a subprocess.
func TestInstanceFatal(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") != "1" {
		t.Skip("skipping subprocess test in main process")
	}
	reset()
	Instance() // This should call log.Fatal
}
