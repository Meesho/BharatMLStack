package controller

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/application/handler"
	"github.com/Meesho/BharatMLStack/horizon/pkg/api"
	"github.com/gin-gonic/gin"
)

type Config interface {
	Onboard(ctx *gin.Context)
	GetAll(ctx *gin.Context)
	Edit(ctx *gin.Context)
}

var (
	configController Config
	once             sync.Once
	emptyResponse    = ""
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
	var request handler.OnboardRequest
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

func (c *V1) GetAll(ctx *gin.Context) {
	response, err := c.Config.GetAll()
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
	var request handler.EditRequest
	token := ctx.Param("token")
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(api.NewBadRequestError(err.Error()).StatusCode, handler.Response{
			Error: err.Error(),
			Data:  handler.Message{Message: emptyResponse},
		})
		return
	}
	request.Payload.AppToken = token
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
