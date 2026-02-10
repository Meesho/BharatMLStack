package handler

import (
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
	return h.baseURL + mlflowPath
}

// ForwardRequest forwards the HTTP request to the MLFlow backend
func (h *MLFlowHandler) ForwardRequest(method, url string, body io.Reader, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Failed to create MLFlow request")
		return nil, err
	}

	skipHeaders := map[string]bool{
		"Origin":     true,
		"Referer":    true,
		"Host":       true,
		"Connection": true,
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
