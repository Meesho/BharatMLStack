package http

import (
	"context"
	"net/http"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/handler"
	"github.com/Meesho/BharatMLStack/interaction-store/pkg/proto/timeseries"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

func RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		api.POST("/timeseries/persist/click", handlePersistClickInteractions)
		api.POST("/timeseries/persist/order", handlePersistOrderInteractions)
		api.POST("/timeseries/retrieve/click", handleRetrieveClickInteractions)
		api.POST("/timeseries/retrieve/order", handleRetrieveOrderInteractions)
		api.POST("/timeseries/retrieve", handleRetrieveInteractions)
	}
}

// createContext creates a context with metadata from HTTP headers
func createContext(c *gin.Context) context.Context {
	callerId := c.GetHeader(callerIdHeader)

	md := metadata.New(map[string]string{
		callerIdHeader: callerId,
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

func handlePersistClickInteractions(c *gin.Context) {
	ctx := createContext(c)

	var request timeseries.PersistClickDataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	interactionHandler := handler.InitInteractionHandler()
	result, err := interactionHandler.PersistClickData(ctx, &request)
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

func handlePersistOrderInteractions(c *gin.Context) {
	ctx := createContext(c)

	var request timeseries.PersistOrderDataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	interactionHandler := handler.InitInteractionHandler()
	result, err := interactionHandler.PersistOrderData(ctx, &request)
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

func handleRetrieveClickInteractions(c *gin.Context) {
	ctx := createContext(c)

	var request timeseries.RetrieveDataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	interactionHandler := handler.InitInteractionHandler()
	result, err := interactionHandler.RetrieveClickInteractions(ctx, &request)
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

func handleRetrieveOrderInteractions(c *gin.Context) {
	ctx := createContext(c)

	var request timeseries.RetrieveDataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	interactionHandler := handler.InitInteractionHandler()
	result, err := interactionHandler.RetrieveOrderInteractions(ctx, &request)
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

func handleRetrieveInteractions(c *gin.Context) {
	ctx := createContext(c)

	var request timeseries.RetrieveInteractionsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	interactionHandler := handler.InitInteractionHandler()
	result, err := interactionHandler.RetrieveInteractions(ctx, &request)
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
