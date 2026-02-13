package tracing

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// setup sets up viper and environment variables for tests and returns a cleanup function.
func setup() func() {
	viper.Set("APP_NAME", "test-app")
	viper.SetDefault(samplerArgEnv, 0.1)
	os.Setenv(endpointEnv, "localhost:4317")

	// The cleanup function
	return func() {
		if tp != nil {
			tp.Shutdown(context.Background())
		}
		tp = nil
		once = sync.Once{}
		initialized = false
		otel.SetTracerProvider(nil)
		os.Unsetenv(endpointEnv)
		viper.Reset()
	}
}

// runTestInSubprocess is a helper to test functions that call log.Fatal.
// It runs the test in a separate process and checks for a non-zero exit code.
func runTestInSubprocess(t *testing.T, testName, expectedMsg string) {
	cmd := exec.Command(os.Args[0], "-test.run="+testName)
	cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS=1")
	output, err := cmd.CombinedOutput()

	// We expect the command to exit with a non-zero status code.
	exitErr, ok := err.(*exec.ExitError)
	assert.True(t, ok, "Expected command to exit with an error")
	assert.False(t, exitErr.Success(), "Expected command to fail")
	assert.Contains(t, string(output), expectedMsg)
}

func TestInit(t *testing.T) {
	// This runs only in the main test process
	if os.Getenv("GO_TEST_SUBPROCESS") == "" {
		t.Run("should fatal if APP_NAME is not set", func(t *testing.T) {
			runTestInSubprocess(t, "TestAppNameFatal", "APP_NAME cannot be empty!!!")
		})
		t.Run("should fatal if OTEL_EXPORTER_OTLP_ENDPOINT is not set", func(t *testing.T) {
			runTestInSubprocess(t, "TestOtelEndpointFatal", "OTEL_EXPORTER_OTLP_ENDPOINT env is not set!!!")
		})
	}

	t.Run("should initialize tracer provider successfully", func(t *testing.T) {
		cleanup := setup()
		defer cleanup()

		// Before Init, tracer provider should be nil
		assert.Nil(t, tp)

		Init()

		// After Init, tracer provider should be set
		assert.NotNil(t, tp)
		assert.IsType(t, &trace.TracerProvider{}, tp)
		assert.Equal(t, tp, otel.GetTracerProvider())
	})
}

// TestAppNameFatal is the actual test that will be run in a subprocess.
func TestAppNameFatal(t *testing.T) {
	// This runs only in the subprocess
	if os.Getenv("GO_TEST_SUBPROCESS") != "1" {
		t.Skip("skipping subprocess test in main process")
	}
	cleanup := setup()
	defer cleanup()
	viper.Set("APP_NAME", "")
	Init() // This should call log.Fatal and exit
}

// TestOtelEndpointFatal is the actual test that will be run in a subprocess.
func TestOtelEndpointFatal(t *testing.T) {
	// This runs only in the subprocess
	if os.Getenv("GO_TEST_SUBPROCESS") != "1" {
		t.Skip("skipping subprocess test in main process")
	}
	cleanup := setup()
	defer cleanup()
	os.Unsetenv(endpointEnv)
	Init() // This should call log.Fatal and exit
}

func TestTracerInstance(t *testing.T) {
	t.Run("should return a tracer instance", func(t *testing.T) {
		cleanup := setup()
		defer cleanup()

		Init()
		tracer := GetTracer("test-app")
		assert.NotNil(t, tracer)

		// test if the tracer is working
		spanCtx, span := tracer.Start(context.Background(), "test-span")
		assert.NotNil(t, spanCtx)
		assert.NotNil(t, span)
		span.End()
		assert.NotNil(t, spanCtx)
		assert.NotNil(t, span)
	})

	t.Run("should return a noop tracer provider if called before Init", func(t *testing.T) {
		cleanup := setup()
		defer cleanup()
		tp = nil // ensure it's nil

		viper.Set("APP_NAME", "test-app")
		defer viper.Reset()

		tracer := GetTracer("test-app")
		assert.IsType(t, noop.Tracer{}, tracer)

		// test if the tracer is working but no telemetry is sent
		spanCtx, span := tracer.Start(context.Background(), "test-span")
		assert.NotNil(t, spanCtx)
		assert.NotNil(t, span)
		span.End()
		assert.NotNil(t, spanCtx)
		assert.NotNil(t, span)
	})
}

func TestShutdownTracer(t *testing.T) {
	t.Run("should not panic when tracer provider is nil", func(t *testing.T) {
		cleanup := setup()
		defer cleanup()
		tp = nil

		assert.NotPanics(t, func() {
			ShutdownTracer()
		})
	})

	t.Run("should shutdown the tracer provider", func(t *testing.T) {
		cleanup := setup()
		defer cleanup()

		Init()
		assert.NotNil(t, tp)

		assert.NotPanics(t, func() {
			ShutdownTracer()
		})
	})
}
