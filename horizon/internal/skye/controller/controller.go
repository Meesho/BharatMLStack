package controller

import (
	"strconv"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/skye/handler"
	"github.com/Meesho/BharatMLStack/horizon/pkg/api"
	"github.com/gin-gonic/gin"
)

type Config interface {
	RegisterStore(ctx *gin.Context)
	ApproveStoreRequest(ctx *gin.Context)
	GetStores(ctx *gin.Context)
	GetAllStoreRequests(ctx *gin.Context)
	RegisterEntity(ctx *gin.Context)
	ApproveEntityRequest(ctx *gin.Context)
	GetEntities(ctx *gin.Context)
	GetAllEntityRequests(ctx *gin.Context)
	RegisterModel(ctx *gin.Context)
	EditModel(ctx *gin.Context)
	ApproveModelRequest(ctx *gin.Context)
	ApproveModelEditRequest(ctx *gin.Context)
	GetModels(ctx *gin.Context)
	GetAllModelRequests(ctx *gin.Context)
	RegisterVariant(ctx *gin.Context)
	EditVariant(ctx *gin.Context)
	ApproveVariantRequest(ctx *gin.Context)
	ApproveVariantEditRequest(ctx *gin.Context)
	GetVariants(ctx *gin.Context)
	GetAllVariantRequests(ctx *gin.Context)
	RegisterFilter(ctx *gin.Context)
	ApproveFilterRequest(ctx *gin.Context)
	GetFilters(ctx *gin.Context)
	GetAllFilterRequests(ctx *gin.Context)
	RegisterJobFrequency(ctx *gin.Context)
	ApproveJobFrequencyRequest(ctx *gin.Context)
	GetJobFrequencies(ctx *gin.Context)
	GetAllJobFrequencyRequests(ctx *gin.Context)
	GetMQIdTopics(ctx *gin.Context)
	GetVariantsList(ctx *gin.Context)
	PromoteVariant(ctx *gin.Context)
	ApproveVariantPromotionRequest(ctx *gin.Context)
	OnboardVariant(ctx *gin.Context)
	ApproveVariantOnboardingRequest(ctx *gin.Context)
	GetAllVariantOnboardingRequests(ctx *gin.Context)
	GetVariantOnboardingTasks(ctx *gin.Context)
}

var (
	configController Config
	once             sync.Once
	emptyResponse    = ""
)

type V1 struct {
	Config handler.Config
}

func NewConfigController(appConfig configs.Configs) Config {
	if configController == nil {
		once.Do(func() {
			configController = &V1{
				Config: handler.NewConfigHandler(1, appConfig),
			}
		})
	}
	return configController
}

func (c *V1) RegisterStore(ctx *gin.Context) {
	var request handler.StoreRegisterRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.RegisterStore(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) ApproveStoreRequest(ctx *gin.Context) {
	requestIDStr := ctx.Query("request_id")
	if requestIDStr == "" {
		ctx.JSON(api.NewBadRequestError("request_id query parameter is required").StatusCode, handler.Response{
			Error: "request_id query parameter is required",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		ctx.JSON(api.NewBadRequestError("invalid request_id format").StatusCode, handler.Response{
			Error: "invalid request_id format",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	var approval handler.ApprovalRequest
	if err := ctx.ShouldBindJSON(&approval); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.ApproveStoreRequest(requestID, approval)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetStores(ctx *gin.Context) {
	response, err := c.Config.GetStores()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetAllStoreRequests(ctx *gin.Context) {
	response, err := c.Config.GetAllStoreRequests()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) RegisterEntity(ctx *gin.Context) {
	var request handler.EntityRegisterRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.RegisterEntity(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) ApproveEntityRequest(ctx *gin.Context) {
	requestIDStr := ctx.Query("request_id")
	if requestIDStr == "" {
		ctx.JSON(api.NewBadRequestError("request_id query parameter is required").StatusCode, handler.Response{
			Error: "request_id query parameter is required",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		ctx.JSON(api.NewBadRequestError("invalid request_id format").StatusCode, handler.Response{
			Error: "invalid request_id format",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	var approval handler.ApprovalRequest
	if err := ctx.ShouldBindJSON(&approval); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.ApproveEntityRequest(requestID, approval)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetEntities(ctx *gin.Context) {
	response, err := c.Config.GetEntities()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetAllEntityRequests(ctx *gin.Context) {
	response, err := c.Config.GetAllEntityRequests()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) RegisterModel(ctx *gin.Context) {
	var request handler.ModelRegisterRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.RegisterModel(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) EditModel(ctx *gin.Context) {
	var request handler.ModelEditRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.EditModel(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) ApproveModelRequest(ctx *gin.Context) {
	requestIDStr := ctx.Query("request_id")
	if requestIDStr == "" {
		ctx.JSON(api.NewBadRequestError("request_id query parameter is required").StatusCode, handler.Response{
			Error: "request_id query parameter is required",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		ctx.JSON(api.NewBadRequestError("invalid request_id format").StatusCode, handler.Response{
			Error: "invalid request_id format",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	var approval handler.ApprovalRequest
	if err := ctx.ShouldBindJSON(&approval); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.ApproveModelRequest(requestID, approval)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) ApproveModelEditRequest(ctx *gin.Context) {
	requestIDStr := ctx.Query("request_id")
	if requestIDStr == "" {
		ctx.JSON(api.NewBadRequestError("request_id query parameter is required").StatusCode, handler.Response{
			Error: "request_id query parameter is required",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		ctx.JSON(api.NewBadRequestError("invalid request_id format").StatusCode, handler.Response{
			Error: "invalid request_id format",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	var approval handler.ApprovalRequest
	if err := ctx.ShouldBindJSON(&approval); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.ApproveModelEditRequest(requestID, approval)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetModels(ctx *gin.Context) {
	response, err := c.Config.GetModels()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetAllModelRequests(ctx *gin.Context) {
	response, err := c.Config.GetAllModelRequests()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) RegisterVariant(ctx *gin.Context) {
	var request handler.VariantRegisterRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.RegisterVariant(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) EditVariant(ctx *gin.Context) {
	var request handler.VariantEditRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.EditVariant(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) ApproveVariantRequest(ctx *gin.Context) {
	requestIDStr := ctx.Query("request_id")
	if requestIDStr == "" {
		ctx.JSON(api.NewBadRequestError("request_id query parameter is required").StatusCode, handler.Response{
			Error: "request_id query parameter is required",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		ctx.JSON(api.NewBadRequestError("invalid request_id format").StatusCode, handler.Response{
			Error: "invalid request_id format",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	var approval handler.ApprovalRequest
	if err := ctx.ShouldBindJSON(&approval); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.ApproveVariantRequest(requestID, approval)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) ApproveVariantEditRequest(ctx *gin.Context) {
	requestIDStr := ctx.Query("request_id")
	if requestIDStr == "" {
		ctx.JSON(api.NewBadRequestError("request_id query parameter is required").StatusCode, handler.Response{
			Error: "request_id query parameter is required",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		ctx.JSON(api.NewBadRequestError("invalid request_id format").StatusCode, handler.Response{
			Error: "invalid request_id format",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	var approval handler.ApprovalRequest
	if err := ctx.ShouldBindJSON(&approval); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.ApproveVariantEditRequest(requestID, approval)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetVariants(ctx *gin.Context) {
	entity := ctx.Query("entity")
	model := ctx.Query("model")
	response, err := c.Config.GetVariants(entity, model)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetAllVariantRequests(ctx *gin.Context) {
	response, err := c.Config.GetAllVariantRequests()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) RegisterFilter(ctx *gin.Context) {
	var request handler.FilterRegisterRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.RegisterFilter(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) ApproveFilterRequest(ctx *gin.Context) {
	requestIDStr := ctx.Query("request_id")
	if requestIDStr == "" {
		ctx.JSON(api.NewBadRequestError("request_id query parameter is required").StatusCode, handler.Response{
			Error: "request_id query parameter is required",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		ctx.JSON(api.NewBadRequestError("invalid request_id format").StatusCode, handler.Response{
			Error: "invalid request_id format",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	var approval handler.ApprovalRequest
	if err := ctx.ShouldBindJSON(&approval); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.ApproveFilterRequest(requestID, approval)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetFilters(ctx *gin.Context) {
	entity := ctx.Query("entity")
	response, err := c.Config.GetFilters(entity)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetAllFilterRequests(ctx *gin.Context) {
	response, err := c.Config.GetAllFilterRequests()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) RegisterJobFrequency(ctx *gin.Context) {
	var request handler.JobFrequencyRegisterRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.RegisterJobFrequency(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) ApproveJobFrequencyRequest(ctx *gin.Context) {
	requestIDStr := ctx.Query("request_id")
	if requestIDStr == "" {
		ctx.JSON(api.NewBadRequestError("request_id query parameter is required").StatusCode, handler.Response{
			Error: "request_id query parameter is required",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		ctx.JSON(api.NewBadRequestError("invalid request_id format").StatusCode, handler.Response{
			Error: "invalid request_id format",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	var approval handler.ApprovalRequest
	if err := ctx.ShouldBindJSON(&approval); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.ApproveJobFrequencyRequest(requestID, approval)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetJobFrequencies(ctx *gin.Context) {
	response, err := c.Config.GetJobFrequencies()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetAllJobFrequencyRequests(ctx *gin.Context) {
	response, err := c.Config.GetAllJobFrequencyRequests()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetMQIdTopics(ctx *gin.Context) {
	response, err := c.Config.GetMQIdTopics()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetVariantsList(ctx *gin.Context) {
	response, err := c.Config.GetVariantsList()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) PromoteVariant(ctx *gin.Context) {
	var request handler.VariantPromotionRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.PromoteVariant(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) ApproveVariantPromotionRequest(ctx *gin.Context) {
	requestIDStr := ctx.Query("request_id")
	if requestIDStr == "" {
		ctx.JSON(api.NewBadRequestError("request_id query parameter is required").StatusCode, handler.Response{
			Error: "request_id query parameter is required",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		ctx.JSON(api.NewBadRequestError("invalid request_id format").StatusCode, handler.Response{
			Error: "invalid request_id format",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	var approval handler.ApprovalRequest
	if err := ctx.ShouldBindJSON(&approval); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.ApproveVariantPromotionRequest(requestID, approval)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) OnboardVariant(ctx *gin.Context) {
	var request handler.VariantOnboardingRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.OnboardVariant(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) ApproveVariantOnboardingRequest(ctx *gin.Context) {
	requestIDStr := ctx.Query("request_id")
	if requestIDStr == "" {
		ctx.JSON(api.NewBadRequestError("request_id query parameter is required").StatusCode, handler.Response{
			Error: "request_id query parameter is required",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		ctx.JSON(api.NewBadRequestError("invalid request_id format").StatusCode, handler.Response{
			Error: "invalid request_id format",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	var approval handler.ApprovalRequest
	if err := ctx.ShouldBindJSON(&approval); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.ApproveVariantOnboardingRequest(requestID, approval)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetAllVariantOnboardingRequests(ctx *gin.Context) {
	response, err := c.Config.GetAllVariantOnboardingRequests()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) GetVariantOnboardingTasks(ctx *gin.Context) {
	response, err := c.Config.GetVariantOnboardingTasks()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	ctx.JSON(200, response)
}
