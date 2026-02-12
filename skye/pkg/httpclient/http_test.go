package httpclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/Meesho/go-core/v2/tracing"
)

// TestPathNormalization tests the path normalization logic
func TestPathNormalization(t *testing.T) {
	path := "/users/1234/orders/5678"
	expected := "/users/{id}/orders/{id}"

	normalized := getNormalizedPath(path)

	if normalized != expected {
		t.Errorf("Expected normalized path %s, got %s", expected, normalized)
	}
}

func TestPathNormalization_WithUUID(t *testing.T) {
	path := "/users/123e4567-e89b-12d3-a456-426614174000/orders/5678"
	expected := "/users/{uuid}/orders/{id}"

	normalized := getNormalizedPath(path)

	if normalized != expected {
		t.Errorf("Expected normalized path %s, got %s", expected, normalized)
	}
}

func TestGetNormalizedPath_NegativeCases(t *testing.T) {
	// Test with a path that doesn't match any pattern
	path := "/no/matching/patterns/here"
	expected := "/no/matching/patterns/here"
	normalized := getNormalizedPath(path)
	if normalized != expected {
		t.Errorf("Expected normalized path %s, got %s", expected, normalized)
	}

	// Test with an empty path
	path = ""
	expected = ""
	normalized = getNormalizedPath(path)
	if normalized != expected {
		t.Errorf("Expected normalized path %s, got %s", expected, normalized)
	}
}

func TestGetNormalizedPath_objectId(t *testing.T) {
	path := "/users/5f6e0d2b4e2d6b0001d1c1f5/orders/5678"
	expected := "/users/{objectId}/orders/{id}"

	normalized := getNormalizedPath(path)

	if normalized != expected {
		t.Errorf("Expected normalized path %s, got %s", expected, normalized)
	}
}

func TestGetNormalizedPath_IdUUidObjectIdInOnePath(t *testing.T) {
	path := "/users/5f6e0d2b4e2d6b0001d1c1f5/orders/5678/123e4567-e89b-12d3-a456-426614174000"
	expected := "/users/{objectId}/orders/{id}/{uuid}"

	normalized := getNormalizedPath(path)

	if normalized != expected {
		t.Errorf("Expected normalized path %s, got %s", expected, normalized)
	}
}

func TestGetHTTPClientWithOtel(t *testing.T) {
	// Positive scenario: Tracer is initialized
	t.Run("with tracer initialized", func(t *testing.T) {
		// Setup tracer
		viper.Set("APP_NAME", "test-http-client")
		os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
		tracing.Init()
		defer tracing.ShutdownTracer()
		defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		defer viper.Reset()

		config := &Config{
			TimeoutInMs: 100,
			Transport: &TransportConfig{
				DialTimeoutInMs:      1000,
				MaxIdleConns:         100,
				MaxIdleConnsPerHost:  100,
				IdleConnTimeoutInMs:  30000,
				KeepAliveTimeoutInMs: 30000,
			},
		}

		client := getHTTPClient(config)

		assert.NotNil(t, client)
		assert.IsType(t, &otelhttp.Transport{}, client.Transport, "Transport should be of type otelhttp.Transport")
	})

	// Negative scenario: Tracer is not initialized
	t.Run("without tracer initialized", func(t *testing.T) {
		// No tracer setup here, so otel will use a no-op tracer provider.
		config := &Config{
			TimeoutInMs: 100,
			Transport: &TransportConfig{
				DialTimeoutInMs:      1000,
				MaxIdleConns:         100,
				MaxIdleConnsPerHost:  100,
				IdleConnTimeoutInMs:  30000,
				KeepAliveTimeoutInMs: 30000,
			},
		}

		client := getHTTPClient(config)

		// otelhttp.NewTransport does not panic or error if the global tracer provider is not set.
		// It gracefully falls back to a no-op tracer. So we still expect an otelhttp.Transport.
		assert.NotNil(t, client)
		assert.IsType(t, &otelhttp.Transport{}, client.Transport, "Transport should be of type otelhttp.Transport even without an initialized tracer")
	})
}

func TestHTTPClient_Do_NoTracer(t *testing.T) {
	// Setup a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	}))
	defer server.Close()

	// Parse server URL to create client config
	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	config := &Config{
		Scheme:      serverURL.Scheme,
		Host:        serverURL.Hostname(),
		Port:        serverURL.Port(),
		TimeoutInMs: 1000,
		Transport: &TransportConfig{
			DialTimeoutInMs:      1000,
			MaxIdleConns:         100,
			MaxIdleConnsPerHost:  100,
			IdleConnTimeoutInMs:  30000,
			KeepAliveTimeoutInMs: 30000,
		},
	}

	// Create HTTPClient without initializing the tracer
	client := NewConnFromConfig(config, "TEST_PREFIX")
	assert.NotNil(t, client)

	// Create a request to the mock server
	req, err := http.NewRequest("GET", server.URL, nil)
	assert.NoError(t, err)

	// Send the request
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Assert the response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHTTPClient_Do_WithTracer(t *testing.T) {
	// Setup a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	}))
	defer server.Close()

	// Parse server URL to create client config
	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	config := &Config{
		Scheme:      serverURL.Scheme,
		Host:        serverURL.Hostname(),
		Port:        serverURL.Port(),
		TimeoutInMs: 1000,
		Transport: &TransportConfig{
			DialTimeoutInMs:      1000,
			MaxIdleConns:         100,
			MaxIdleConnsPerHost:  100,
			IdleConnTimeoutInMs:  30000,
			KeepAliveTimeoutInMs: 30000,
		},
	}

	// Create HTTPClient with initializing the tracer
	tracing.Init()
	defer tracing.ShutdownTracer()

	client := NewConnFromConfig(config, "TEST_PREFIX")
	assert.NotNil(t, client)

	// Create a request to the mock server
	req, err := http.NewRequest("GET", server.URL, nil)
	assert.NoError(t, err)

	// Send the request
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Assert the response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
