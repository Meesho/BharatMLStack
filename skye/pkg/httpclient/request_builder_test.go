package httpclient

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestNewHttpRequestBuilder(t *testing.T) {
	builder := NewHttpRequestBuilder()
	assert.NotNil(t, builder)
	assert.NotNil(t, builder.headers)
}

func TestWithHost(t *testing.T) {
	builder := NewHttpRequestBuilder().WithHost("example.com")
	assert.Equal(t, "example.com", builder.host)
}

func TestWithPort(t *testing.T) {
	builder := NewHttpRequestBuilder().WithPort(8080)
	assert.Equal(t, 8080, builder.port)
}

func TestWithPath(t *testing.T) {
	builder := NewHttpRequestBuilder().WithPath("/api/endpoint")
	assert.Equal(t, "/api/endpoint", builder.path)
}

func TestWithMethod(t *testing.T) {
	builder := NewHttpRequestBuilder().WithMethod(http.MethodPost)
	assert.Equal(t, http.MethodPost, builder.method)
}

func TestWithHeader(t *testing.T) {
	builder := NewHttpRequestBuilder().WithHeader("Authorization", "Bearer token")
	assert.Equal(t, "Bearer token", builder.headers["Authorization"])
}

func TestWithHeaders(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer token",
		"Content-Type":  "application/json",
	}
	builder := NewHttpRequestBuilder().WithHeaders(headers)
	assert.Equal(t, headers, builder.headers)
}

func TestWithBody(t *testing.T) {
	body := map[string]interface{}{
		"key": "value",
	}
	builder := NewHttpRequestBuilder().WithBody(body)
	assert.Equal(t, body, builder.body)
}

func TestRequestBuilder_WithContext(t *testing.T) {
	builder := NewHttpRequestBuilder().WithContext(context.Background())
	assert.NotNil(t, builder.ctx)
}

func TestBuildContentTypeJson(t *testing.T) {
	body := map[string]interface{}{
		"key": "value",
	}
	ctx := context.Background()
	builder := NewHttpRequestBuilder().
		WithHost("example.com").
		WithPort(8080).
		WithPath("/api/endpoint").
		WithMethod(http.MethodPost).
		WithHeader("Authorization", "Bearer token").
		WithBody(body).
		WithContext(ctx)

	req, err := builder.BuildContentTypeJson()
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, "example.com:8080", req.Host)
	assert.Equal(t, "/api/endpoint", req.URL.Path)
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "Bearer token", req.Header.Get("Authorization"))
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	assert.Equal(t, ctx, req.Context())

	var reqBody map[string]interface{}
	err = json.NewDecoder(req.Body).Decode(&reqBody)
	assert.NoError(t, err)
	assert.Equal(t, body, reqBody)
}

func TestBuildContentTypeJson_InvalidHost(t *testing.T) {
	builder := NewHttpRequestBuilder().
		WithPort(8080).
		WithPath("/api/endpoint").
		WithMethod(http.MethodPost)

	_, err := builder.BuildContentTypeJson()
	assert.Error(t, err)
	assert.Equal(t, "host is required", err.Error())
}

func TestBuildContentTypeJson_InvalidPort(t *testing.T) {
	builder := NewHttpRequestBuilder().
		WithHost("example.com").
		WithPath("/api/endpoint").
		WithMethod(http.MethodPost)

	_, err := builder.BuildContentTypeJson()
	assert.Error(t, err)
	assert.Equal(t, "invalid port", err.Error())
}

func TestBuildContentTypeJson_InvalidPath(t *testing.T) {
	builder := NewHttpRequestBuilder().
		WithHost("example.com").
		WithPort(8080).
		WithMethod(http.MethodPost)

	_, err := builder.BuildContentTypeJson()
	assert.Error(t, err)
	assert.Equal(t, "path is required", err.Error())
}

func TestBuildContentTypeJson_InvalidMethod(t *testing.T) {
	builder := NewHttpRequestBuilder().
		WithHost("example.com").
		WithPort(8080).
		WithPath("/api/endpoint")

	_, err := builder.BuildContentTypeJson()
	assert.Error(t, err)
	assert.Equal(t, "method is required", err.Error())
}

func TestBuildContentTypeJson_NoContext(t *testing.T) {
	builder := NewHttpRequestBuilder().
		WithHost("example.com").
		WithPort(8080).
		WithPath("/api/endpoint").
		WithMethod(http.MethodPost)

	_, err := builder.BuildContentTypeJson()
	assert.Error(t, err)
	assert.Equal(t, "context is required, pass context.Background() if not required", err.Error())
}
