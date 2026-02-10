package controller

import (
	"bytes"
	"io"
	"net/http"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/mlflow/handler"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type MLFlowController interface {
	ProxyToMLFlow(ctx *gin.Context)
}

var (
	mlflowController     MLFlowController
	mlflowControllerOnce sync.Once
)

type V1 struct {
	handler *handler.MLFlowHandler
}

func NewMLFlowController() MLFlowController {
	if mlflowController == nil {
		mlflowControllerOnce.Do(func() {
			mlflowController = &V1{
				handler: handler.NewMLFlowHandler(),
			}
		})
	}
	return mlflowController
}

// ProxyToMLFlow forwards all requests to the MLFlow backend
func (v *V1) ProxyToMLFlow(ctx *gin.Context) {
	// Extract the wildcard path (everything after /api/v1/mlflow)
	mlflowPath := ctx.Param("mlflowpath")

	// Construct the full MLFlow backend URL with path rewrite (v1 -> 2.0)
	mlflowURL := v.handler.ConstructMLFlowURL(mlflowPath)

	// Append query parameters if present
	if ctx.Request.URL.RawQuery != "" {
		mlflowURL += "?" + ctx.Request.URL.RawQuery
	}

	// Read and buffer the request body
	var bodyReader io.Reader
	if ctx.Request.Body != nil {
		bodyBytes, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			log.Error().Err(err).Msg("Failed to read request body")
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "Failed to read request body",
			})
			return
		}

		// Create a new reader from the bytes for forwarding
		if len(bodyBytes) > 0 {
			bodyReader = bytes.NewReader(bodyBytes)
		}
	}

	// Forward the request to MLFlow backend
	resp, err := v.handler.ForwardRequest(
		ctx.Request.Method,
		mlflowURL,
		bodyReader,
		ctx.Request.Header,
	)
	if err != nil {
		log.Error().Err(err).Str("url", mlflowURL).Msg("Failed to proxy request to MLFlow")
		ctx.JSON(http.StatusBadGateway, gin.H{
			"error": "Failed to connect to MLFlow backend",
		})
		return
	}
	defer resp.Body.Close()

	// Copy response headers from MLFlow backend
	for key, values := range resp.Header {
		for _, value := range values {
			ctx.Writer.Header().Add(key, value)
		}
	}

	// Copy response body from MLFlow backend
	ctx.Status(resp.StatusCode)
	_, err = io.Copy(ctx.Writer, resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to copy response body from MLFlow")
	}
}
