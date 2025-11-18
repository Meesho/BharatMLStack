package controller

import (
	feature_registry "github.com/Meesho/BharatMLStack/horizon/internal/feature_store/handler"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type FsConfigController interface {
	GetFsConfigs(ctx *gin.Context)
}

type FsConfigControllerV1 struct {
	FsConfigHandler feature_registry.FsConfigHandler
}

func NewFsConfigControllerGenerator(handler feature_registry.FsConfigHandler) *FsConfigControllerV1 {
	if handler == nil {
		log.Panic().Msg("application config handler cannot be nil")
	}
	return &FsConfigControllerV1{
		FsConfigHandler: handler,
	}
}

func (f *FsConfigControllerV1) GetFsConfigs(ctx *gin.Context) {
	if apiError := f.FsConfigHandler.GetFsConfigs(ctx); apiError != nil {
		ctx.Error(apiError)
		return
	}
}
