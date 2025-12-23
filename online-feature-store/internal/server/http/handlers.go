package http

import (
	"context"
	"net/http"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/handler/feature"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/persist"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

// RegisterRoutes registers all HTTP API routes that mirror gRPC endpoints
func RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// Feature retrieval endpoints (mirroring gRPC FeatureService)
		api.POST("/features/retrieve", handleRetrieveFeatures)
		api.POST("/features/retrieve/decoded", handleRetrieveDecodedResult)

		// Feature persistence endpoint (mirroring gRPC PersistFeatureService)
		api.POST("/features/persist", handlePersistFeatures)
	}
}

// createContextWithAuth creates a context with authentication metadata from HTTP headers
func createContextWithAuth(c *gin.Context) context.Context {
	callerId := c.GetHeader(callerIdHeader)
	authToken := c.GetHeader(AuthTokenHeader)

	md := metadata.New(map[string]string{
		callerIdHeader:  callerId,
		AuthTokenHeader: authToken,
	})

	return metadata.NewIncomingContext(c.Request.Context(), md)
}

// handleError converts gRPC errors to HTTP responses
func handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	st, ok := status.FromError(err)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	httpStatus := grpcToHTTPStatus(st.Code())
	c.JSON(httpStatus, gin.H{
		"error": st.Message(),
		"code":  st.Code().String(),
	})
}

// grpcToHTTPStatus converts gRPC status codes to HTTP status codes
func grpcToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.Canceled:
		return http.StatusRequestTimeout
	default:
		return http.StatusInternalServerError
	}
}

// handleRetrieveFeatures handles POST /api/v1/features/retrieve
func handleRetrieveFeatures(c *gin.Context) {
	ctx, err := createContextWithAuth(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var query retrieve.Query
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	handler := feature.InitV1()
	result, err := handler.RetrieveFeatures(ctx, &query)
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert protobuf to JSON using protojson
	jsonBytes, err := protojson.Marshal(result)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal result to JSON")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize response"})
		return
	}

	c.Data(http.StatusOK, "application/json", jsonBytes)
}

// handleRetrieveDecodedResult handles POST /api/v1/features/retrieve/decoded
func handleRetrieveDecodedResult(c *gin.Context) {
	ctx, err := createContextWithAuth(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var query retrieve.Query
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	handler := feature.InitV1()
	result, err := handler.RetrieveDecodedResult(ctx, &query)
	if err != nil {
		handleError(c, err)
		return
	}

	jsonBytes, err := protojson.Marshal(result)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal result to JSON")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize response"})
		return
	}

	c.Data(http.StatusOK, "application/json", jsonBytes)
}

// handlePersistFeatures handles POST /api/v1/features/persist
func handlePersistFeatures(c *gin.Context) {
	ctx, err := createContextWithAuth(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var query persist.Query
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	handler := feature.InitPersistHandler()
	result, err := handler.PersistFeatures(ctx, &query)
	if err != nil {
		handleError(c, err)
		return
	}

	jsonBytes, err := protojson.Marshal(result)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal result to JSON")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize response"})
		return
	}

	c.Data(http.StatusOK, "application/json", jsonBytes)
}
