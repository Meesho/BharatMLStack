package controller

import (
	"strconv"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/numerix/handler"
	"github.com/Meesho/BharatMLStack/horizon/pkg/api"
	"github.com/gin-gonic/gin"
)

type Config interface {
	Onboard(ctx *gin.Context)
	Promote(ctx *gin.Context)
	GetAll(ctx *gin.Context)
	GenerateExpression(ctx *gin.Context)
	GetExpressionVariables(ctx *gin.Context)
	ReviewRequest(ctx *gin.Context)
	Edit(ctx *gin.Context)
	CancelRequest(ctx *gin.Context)
	GetAllRequests(ctx *gin.Context)
	GenerateFuncitonalTestRequest(ctx *gin.Context)
	ExecuteFuncitonalTestRequest(ctx *gin.Context)
}

var (
	configController Config
	once             sync.Once
)

type V1 struct {
	Config handler.Config
}

func NewConfigController() Config {
	if configController == nil {
		once.Do(func() {
			configController = &V1{
				Config: handler.NewConfigHandler(1),
			}
		})
	}
	return configController
}

func (c *V1) ReviewRequest(ctx *gin.Context) {
	var request handler.ReviewRequestConfigRequest
	var emptyResponse string
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.ReviewRequest(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) Edit(ctx *gin.Context) {
	var request handler.EditConfigRequest
	var emptyResponse string
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.Edit(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) CancelRequest(ctx *gin.Context) {
	var request handler.CancelConfigRequest
	var emptyResponse string
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.CancelRequest(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) GetAllRequests(ctx *gin.Context) {
	role := ctx.GetString("role")
	email := ctx.GetString("email")

	var emptyResponse []handler.RequestConfig

	request := handler.GetAllRequestConfigsRequest{
		Email: email,
		Role:  role,
	}

	response, err := c.Config.GetAllRequests(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.GetAllRequestConfigsResponse{
			Error: err.Error(),
			Data:  emptyResponse,
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) GenerateExpression(ctx *gin.Context) {
	var request handler.ExpressionGenerateRequest
	var emptyResponse string
	if err := ctx.BindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.ExpressionGenerateResponse{
			Error: err.Error(),
			Data:  emptyResponse,
		})
		return
	}
	response, err := c.Config.GenerateExpression(request)
	if err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.ExpressionGenerateResponse{
			Error: err.Error(),
			Data:  emptyResponse,
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) GetExpressionVariables(ctx *gin.Context) {
	var request handler.ExpressionVariablesRequest
	var emptyResponse []string
	if err := ctx.ShouldBindUri(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.ExpressionVariablesResponse{
			Error: err.Error(),
			Data:  emptyResponse,
		})
		return
	}
	response, err := c.Config.GetExpressionVariables(request)
	if err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.ExpressionVariablesResponse{
			Error: err.Error(),
			Data:  emptyResponse,
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) Onboard(ctx *gin.Context) {
	var request handler.OnboardConfigRequest
	var emptyResponse string
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.Onboard(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) Promote(ctx *gin.Context) {
	var request handler.PromoteConfigRequest
	var emptyResponse string
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.Promote(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) GetAll(ctx *gin.Context) {
	var emptyResponse handler.GetAllConfigsResponse

	page := 0
	pageSize := 0
	configID := 0

	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		} else {
			ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.GetAllConfigsResponse{
				Error: err.Error(),
				Data:  emptyResponse.Data,
			})
			return
		}
	}

	if limitStr := ctx.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			pageSize = l
		} else {
			ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.GetAllConfigsResponse{
				Error: err.Error(),
				Data:  emptyResponse.Data,
			})
			return
		}
	}

	if configIDStr := ctx.Query("config_id"); configIDStr != "" {
		if s, err := strconv.Atoi(configIDStr); err == nil {
			configID = s
		} else {
			ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.GetAllConfigsResponse{
				Error: err.Error(),
				Data:  emptyResponse.Data,
			})
			return
		}
	}

	request := handler.GetAllConfigsRequest{
		Page:     page,
		PageSize: pageSize,
		ConfigID: configID,
	}

	response, err := c.Config.GetAll(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.GetAllConfigsResponse{
			Error: err.Error(),
			Data:  emptyResponse.Data,
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) GenerateFuncitonalTestRequest(ctx *gin.Context) {
	var request handler.RequestGenerationRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.ErrorResponse{
			Error: err.Error(),
		})
		return
	}
	response, err := c.Config.GenerateFuncitonalTestRequest(request)

	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.ErrorResponse{
			Error: err.Error(),
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) ExecuteFuncitonalTestRequest(ctx *gin.Context) {
	var request handler.ExecuteRequestFunctionalRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	response, err := c.Config.ExecuteFuncitonalTestRequest(request)

	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(200, response)
}
