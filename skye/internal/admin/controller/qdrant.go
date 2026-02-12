package controller

import (
	"net/http"
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/admin/handler/qdrant"
	"github.com/Meesho/BharatMLStack/skye/pkg/api"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Qdrant interface {
	CreateCollection(ctx *gin.Context)
	TriggerIndexing(ctx *gin.Context)
	ProcessModel(ctx *gin.Context)
	ProcessMultiVariant(ctx *gin.Context)
	ProcessModelsWithFrequency(ctx *gin.Context)
	PromoteVariant(ctx *gin.Context)
	ProcessMultiVariantForceReset(ctx *gin.Context)
}

var (
	qdrantController Qdrant
	onceQdrant       sync.Once
)

type QdrantController struct {
	QdrantHandler qdrant.Db
}

func NewQdrantController() Qdrant {
	if qdrantController == nil {
		onceQdrant.Do(func() {
			qdrantController = &QdrantController{
				QdrantHandler: qdrant.NewHandler(qdrant.DefaultVersion),
			}
		})
	}
	return qdrantController
}

func (q *QdrantController) CreateCollection(ctx *gin.Context) {
	var request qdrant.CreateCollectionRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if apiError := q.QdrantHandler.CreateCollection(&request); apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Collection created successfully"})
}

func (q *QdrantController) TriggerIndexing(ctx *gin.Context) {
	var request qdrant.TriggerIndexingRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	go q.QdrantHandler.TriggerIndexing(&request)
	ctx.JSON(http.StatusOK, gin.H{"message": "Indexing Triggered successfully"})
}

func (q *QdrantController) ProcessModel(ctx *gin.Context) {
	var request qdrant.ProcessModelRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response, apiError := q.QdrantHandler.ProcessModel(&request)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, response)
}

func (q *QdrantController) ProcessMultiVariant(ctx *gin.Context) {
	var request qdrant.ProcessMultiVariantRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response, apiError := q.QdrantHandler.ProcessMultiVariant(&request)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, response)
}

func (q *QdrantController) ProcessModelsWithFrequency(ctx *gin.Context) {
	var request qdrant.ProcessModelsWithFrequencyRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response, apiError := q.QdrantHandler.ProcessModelsWithFrequency(&request)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, response)
}

func (q *QdrantController) PromoteVariant(ctx *gin.Context) {
	var request qdrant.PromoteVariantRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response, apiError := q.QdrantHandler.PromoteVariant(&request)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, response)
}

func (q *QdrantController) ProcessMultiVariantForceReset(ctx *gin.Context) {
	var request qdrant.ProcessMultiVariantRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response, apiError := q.QdrantHandler.ProcessMultiVariantForceReset(&request)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, response)
}
