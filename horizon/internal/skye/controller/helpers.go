package controller

import (
	"net/http"
	"strconv"

	"github.com/Meesho/BharatMLStack/horizon/internal/skye/handler"
	"github.com/Meesho/BharatMLStack/horizon/pkg/api"
	"github.com/gin-gonic/gin"
)

// respondError sends an error response with the given status code
func respondError(ctx *gin.Context, statusCode int, message string) {
	ctx.JSON(statusCode, handler.Response{
		Error: message,
		Data:  handler.Message{Message: ""},
	})
}

// respondBadRequest sends a 400 Bad Request error response
func respondBadRequest(ctx *gin.Context, message string) {
	respondError(ctx, api.NewBadRequestError(message).StatusCode, message)
}

// respondInternalError sends a 500 Internal Server Error response
func respondInternalError(ctx *gin.Context, err error) {
	respondError(ctx, api.NewInternalServerError(err.Error()).StatusCode, err.Error())
}

// respondSuccess sends a successful JSON response
func respondSuccess(ctx *gin.Context, data interface{}) {
	ctx.JSON(http.StatusOK, data)
}

// handleSimpleGet handles endpoints that just call a handler function with no parameters
func handleSimpleGet[T any](ctx *gin.Context, fn func() (T, error)) {
	response, err := fn()
	if err != nil {
		respondInternalError(ctx, err)
		return
	}
	respondSuccess(ctx, response)
}

// handleRegister handles registration endpoints that bind JSON and call a handler
func handleRegister[Req any, Resp any](ctx *gin.Context, fn func(Req) (Resp, error)) {
	var request Req
	if err := ctx.ShouldBindJSON(&request); err != nil {
		respondBadRequest(ctx, err.Error())
		return
	}
	response, err := fn(request)
	if err != nil {
		respondInternalError(ctx, err)
		return
	}
	respondSuccess(ctx, response)
}

// handleApprove handles approval endpoints that parse request_id, bind JSON approval, and call handler
func handleApprove(ctx *gin.Context, fn func(int, handler.ApprovalRequest) (handler.ApprovalResponse, error)) {
	requestID, ok := parseRequestID(ctx)
	if !ok {
		return
	}
	var approval handler.ApprovalRequest
	if err := ctx.ShouldBindJSON(&approval); err != nil {
		respondBadRequest(ctx, err.Error())
		return
	}
	response, err := fn(requestID, approval)
	if err != nil {
		respondInternalError(ctx, err)
		return
	}
	respondSuccess(ctx, response)
}

// parseRequestID extracts and validates the request_id query parameter
func parseRequestID(ctx *gin.Context) (int, bool) {
	requestIDStr := ctx.Query("request_id")
	if requestIDStr == "" {
		respondBadRequest(ctx, "request_id query parameter is required")
		return 0, false
	}
	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		respondBadRequest(ctx, "invalid request_id format")
		return 0, false
	}
	return requestID, true
}

// handleTest handles test endpoints that bind JSON and call a handler
func handleTest[Req any, Resp any](ctx *gin.Context, fn func(Req) (Resp, error)) {
	var request Req
	if err := ctx.ShouldBindJSON(&request); err != nil {
		respondBadRequest(ctx, err.Error())
		return
	}
	response, err := fn(request)
	if err != nil {
		respondInternalError(ctx, err)
		return
	}
	respondSuccess(ctx, response)
}
