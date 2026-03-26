package controller

import (
	"net/http"
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/admin/handler/registry"
	"github.com/Meesho/BharatMLStack/skye/pkg/api"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Registry interface {
	RegisterFrequency(ctx *gin.Context)
	RegisterStore(ctx *gin.Context)
	RegisterEntity(ctx *gin.Context)
	RegisterModel(ctx *gin.Context)
	RegisterVariant(ctx *gin.Context)
}

var (
	registryController Registry
	once               sync.Once
)

type RegistryController struct {
	RegistryHandler registry.Manager
}

func NewRegistryController() Registry {
	if registryController == nil {
		once.Do(func() {
			registryController = &RegistryController{
				RegistryHandler: registry.NewHandler(registry.DefaultVersion),
			}
		})
	}
	return registryController
}

func (r *RegistryController) RegisterFrequency(ctx *gin.Context) {
	var request registry.CreateFrequencyRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if apiError := r.RegistryHandler.RegisterFrequency(&request); apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Frequency Created successfully"})
}

func (r *RegistryController) RegisterStore(ctx *gin.Context) {
	var request registry.CreateStoreRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if apiError := r.RegistryHandler.RegisterStore(&request); apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Store Created successfully"})
}

func (r *RegistryController) RegisterEntity(ctx *gin.Context) {
	var request registry.RegisterEntityRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if apiError := r.RegistryHandler.RegisterEntity(&request); apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Entity registered successfully"})
}

func (r *RegistryController) RegisterModel(ctx *gin.Context) {
	var request registry.RegisterModelRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if apiError := r.RegistryHandler.RegisterModel(&request); apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Model registered successfully"})
}

func (r *RegistryController) RegisterVariant(ctx *gin.Context) {
	var request registry.RegisterVariantRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if apiError := r.RegistryHandler.RegisterVariant(&request); apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Variant registered successfully"})
}
