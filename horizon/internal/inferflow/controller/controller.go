package controller

import (
	"sync"

	"strings"

	handler "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/handler"
	"github.com/Meesho/BharatMLStack/horizon/pkg/api"
	"github.com/gin-gonic/gin"
)

type Config interface {
	Onboard(ctx *gin.Context)
	Promote(ctx *gin.Context)
	Edit(ctx *gin.Context)
	Clone(ctx *gin.Context)
	Delete(ctx *gin.Context)
	ScaleUp(ctx *gin.Context)
	Cancel(ctx *gin.Context)
	Review(ctx *gin.Context)
	GetAll(ctx *gin.Context)
	GetAllRequests(ctx *gin.Context)
	ValidateRequest(ctx *gin.Context)
	GenerateFunctionalTestRequest(ctx *gin.Context)
	ExecuteFuncitonalTestRequest(ctx *gin.Context)
	GetLatestRequest(ctx *gin.Context)
	GetLoggingTTL(ctx *gin.Context)
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

func (c *V1) Onboard(ctx *gin.Context) {
	var request handler.InferflowOnboardRequest
	var emptyResponse string
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	token := ctx.GetHeader("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	response, err := c.Config.Onboard(request, token)
	if err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
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
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) Edit(ctx *gin.Context) {
	var request handler.EditConfigOrCloneConfigRequest
	var emptyResponse string
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	token := ctx.GetHeader("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	response, err := c.Config.Edit(request, token)
	if err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) Clone(ctx *gin.Context) {
	var request handler.EditConfigOrCloneConfigRequest
	var emptyResponse string
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	token := ctx.GetHeader("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	response, err := c.Config.Clone(request, token)
	if err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) Delete(ctx *gin.Context) {
	var request handler.DeleteConfigRequest
	var emptyResponse string
	id := ctx.Query("id")
	request.ConfigID = id
	request.CreatedBy = ctx.GetString("email")
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.Delete(request)
	if err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) ScaleUp(ctx *gin.Context) {
	var request handler.ScaleUpConfigRequest
	var emptyResponse string
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.ScaleUp(request)
	if err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) Cancel(ctx *gin.Context) {
	var request handler.CancelConfigRequest
	var emptyResponse string
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.Cancel(request)
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
	var emptyResponse []handler.ConfigTable
	response, err := c.Config.GetAll()
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.GetAllResponse{
			Error: err.Error(),
			Data:  emptyResponse,
		})
		return
	}

	ctx.JSON(200, response)
}

func (c *V1) Review(ctx *gin.Context) {
	var request handler.ReviewRequest
	var emptyResponse string
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	response, err := c.Config.Review(request)
	if err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
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
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.GetAllRequestConfigsResponse{
			Error: err.Error(),
			Data:  emptyResponse,
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) ValidateRequest(ctx *gin.Context) {

	var emptyResponse string
	requestID := ctx.Param("request_id")
	if requestID == "" {
		ctx.JSON(api.NewBadRequestError("request_id is required").StatusCode, handler.Response{
			Error: "request_id is required",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}

	var request handler.ValidateRequest = handler.ValidateRequest{
		ConfigID: requestID,
	}

	token := ctx.GetHeader("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	response, err := c.Config.ValidateRequest(request, token)
	if err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) GenerateFunctionalTestRequest(ctx *gin.Context) {
	var request handler.GenerateRequestFunctionalTestingRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.GenerateRequestFunctionalTestingResponse{
			Error: err.Error(),
		})
		return
	}

	response, err := c.Config.GenerateFunctionalTestRequest(request)
	if err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, response)
	}
	ctx.JSON(200, response)
}

func (c *V1) ExecuteFuncitonalTestRequest(ctx *gin.Context) {
	var request handler.ExecuteRequestFunctionalTestingRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.ExecuteRequestFunctionalTestingResponse{
			Error: err.Error(),
		})
		return
	}

	response, err := c.Config.ExecuteFuncitonalTestRequest(request)
	if err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, response)
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) GetLatestRequest(ctx *gin.Context) {
	var emptyResponse string
	requestID := ctx.Param("config_id")
	if requestID == "" {
		ctx.JSON(api.NewBadRequestError("config_id is required").StatusCode, handler.Response{
			Error: "config_id is required",
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	response, err := c.Config.GetLatestRequest(requestID)
	if err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	ctx.JSON(200, response)
}

func (c *V1) GetLoggingTTL(ctx *gin.Context) {
	var emptyResponse []int
	response, err := c.Config.GetLoggingTTL()
	if err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.GetLoggingTTLResponse{
			Data: emptyResponse,
		})
	}
	ctx.JSON(200, response)
}
