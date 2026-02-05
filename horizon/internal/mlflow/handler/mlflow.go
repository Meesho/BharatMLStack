package handler

import (
	"bytes"
	"io"
	"net/http"
	"time"

	mlflowpkg "github.com/Meesho/BharatMLStack/horizon/internal/mlflow"
	"github.com/rs/zerolog/log"
)

type MLFlowHandler struct {
	httpClient *http.Client
	baseURL    string
}

func NewMLFlowHandler() *MLFlowHandler {
	return &MLFlowHandler{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: mlflowpkg.MLFlowHostURL,
	}
}

// ConstructMLFlowURL rewrites the path from /api/v1/mlflow/* to /api/2.0/mlflow/*
func (h *MLFlowHandler) ConstructMLFlowURL(mlflowPath string) string {
	// mlflowPath comes in as something like "/runs/log-metric"
	// We need to construct: http://localhost:5001/api/2.0/mlflow/runs/log-metric
	return h.baseURL + "/api/2.0/mlflow" + mlflowPath
}

// ForwardRequest forwards the HTTP request to the MLFlow backend
func (h *MLFlowHandler) ForwardRequest(method, url string, body io.Reader, headers http.Header) (*http.Response, error) {
	// Read body if present to ensure it can be sent with the request
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			log.Error().Err(err).Msg("Failed to read body in handler")
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Failed to create MLFlow request")
		return nil, err
	}

	// Copy headers from the original request, but skip problematic ones
	// that might cause CORS issues with MLFlow backend
	skipHeaders := map[string]bool{
		"Origin":     true, // Skip Origin to avoid CORS issues
		"Referer":    true, // Skip Referer as it points to the frontend
		"Host":       true, // Host will be set automatically by http.Client
		"Connection": true, // Connection is managed by http.Client
	}

	for key, values := range headers {
		if skipHeaders[key] {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Make the request to MLFlow backend
	resp, err := h.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Failed to forward request to MLFlow")
		return nil, err
	}

	return resp, nil
}

// GetHTTPClient returns the HTTP client instance
func (h *MLFlowHandler) GetHTTPClient() *http.Client {
	return h.httpClient
}
