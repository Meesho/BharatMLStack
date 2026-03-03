package controller

import (
	"net/http"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/internal/featurecomputeengine/handler"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Controller struct {
	handler *handler.Handler
}

func New(h *handler.Handler) *Controller {
	return &Controller{handler: h}
}

func (c *Controller) RegisterAssets(ctx *gin.Context) {
	var req handler.RegisterAssetsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("invalid register assets request")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Assets) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "assets list must not be empty"})
		return
	}

	count, err := c.handler.RegisterAssets(ctx.Request.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("failed to register assets")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"registered": count})
}

func (c *Controller) GetExecutionPlan(ctx *gin.Context) {
	var req handler.GetExecutionPlanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("invalid execution plan request")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Notebook == "" || req.TriggerType == "" || req.Partition == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "notebook, trigger_type, and partition are required"})
		return
	}

	plan, err := c.handler.GetExecutionPlan(ctx.Request.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get execution plan")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, plan)
}

func (c *Controller) ReportAssetReady(ctx *gin.Context) {
	var req handler.ReportAssetReadyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("invalid report asset ready request")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.AssetName == "" || req.Partition == "" || req.ComputeKey == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "asset_name, partition, and compute_key are required"})
		return
	}

	actions, err := c.handler.ReportAssetReady(ctx.Request.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("failed to report asset ready")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"trigger_actions": actions})
}

func (c *Controller) ReportAssetFailed(ctx *gin.Context) {
	var req handler.ReportAssetFailedRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("invalid report asset failed request")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.AssetName == "" || req.Partition == "" || req.Error == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "asset_name, partition, and error are required"})
		return
	}

	if err := c.handler.ReportAssetFailed(ctx.Request.Context(), req); err != nil {
		log.Error().Err(err).Msg("failed to report asset failure")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "asset failure recorded"})
}

func (c *Controller) GetAllNecessity(ctx *gin.Context) {
	necessity, err := c.handler.GetAllNecessity(ctx.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get necessity map")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, necessity)
}

func (c *Controller) GetNecessity(ctx *gin.Context) {
	assetName := ctx.Param("asset_name")
	if assetName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "asset_name is required"})
		return
	}

	nec, err := c.handler.GetNecessity(ctx.Request.Context(), assetName)
	if err != nil {
		log.Error().Err(err).Msg("failed to get necessity")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	if nec == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"asset_name": assetName, "necessity": *nec})
}

func (c *Controller) SetServingOverride(ctx *gin.Context) {
	var req handler.SetServingOverrideRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("invalid serving override request")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.AssetName == "" || req.Reason == "" || req.UpdatedBy == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "asset_name, reason, and updated_by are required"})
		return
	}

	if err := c.handler.SetServingOverride(ctx.Request.Context(), req); err != nil {
		log.Error().Err(err).Msg("failed to set serving override")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "serving override set"})
}

func (c *Controller) ClearServingOverride(ctx *gin.Context) {
	assetName := ctx.Param("asset_name")
	if assetName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "asset_name is required"})
		return
	}

	if err := c.handler.ClearServingOverride(ctx.Request.Context(), assetName); err != nil {
		log.Error().Err(err).Msg("failed to clear serving override")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "serving override cleared"})
}

func (c *Controller) ResolveNecessity(ctx *gin.Context) {
	necessity, err := c.handler.ResolveNecessity(ctx.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to resolve necessity")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, necessity)
}

func (c *Controller) GetFullLineage(ctx *gin.Context) {
	lineage, err := c.handler.GetFullLineage(ctx.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get lineage")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, lineage)
}

func (c *Controller) GetUpstream(ctx *gin.Context) {
	assetName := ctx.Param("asset_name")
	if assetName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "asset_name is required"})
		return
	}

	upstream, err := c.handler.GetUpstream(ctx.Request.Context(), assetName)
	if err != nil {
		if isNotFoundErr(err) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		log.Error().Err(err).Msg("failed to get upstream")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"asset_name": assetName, "upstream": upstream})
}

func (c *Controller) GetDownstream(ctx *gin.Context) {
	assetName := ctx.Param("asset_name")
	if assetName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "asset_name is required"})
		return
	}

	downstream, err := c.handler.GetDownstream(ctx.Request.Context(), assetName)
	if err != nil {
		if isNotFoundErr(err) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		log.Error().Err(err).Msg("failed to get downstream")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"asset_name": assetName, "downstream": downstream})
}

func (c *Controller) ComputePlan(ctx *gin.Context) {
	var req handler.ProposePlanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("invalid plan request")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Assets) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "assets list must not be empty"})
		return
	}

	result, err := c.handler.ComputePlan(ctx.Request.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("failed to compute plan")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) ApplyPlan(ctx *gin.Context) {
	var req handler.ProposePlanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("invalid apply plan request")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Assets) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "assets list must not be empty"})
		return
	}

	if err := c.handler.ApplyPlan(ctx.Request.Context(), req); err != nil {
		log.Error().Err(err).Msg("failed to apply plan")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "plan applied"})
}

func (c *Controller) MarkDatasetReady(ctx *gin.Context) {
	var req handler.MarkDatasetReadyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("invalid dataset ready request")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.DatasetName == "" || req.Partition == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "dataset_name and partition are required"})
		return
	}

	actions, err := c.handler.MarkDatasetReady(ctx.Request.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("failed to mark dataset ready")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"trigger_actions": actions})
}

func (c *Controller) GetDatasetPartitions(ctx *gin.Context) {
	datasetName := ctx.Param("dataset_name")
	if datasetName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "dataset_name is required"})
		return
	}

	partitions, err := c.handler.GetDatasetPartitions(ctx.Request.Context(), datasetName)
	if err != nil {
		log.Error().Err(err).Msg("failed to get dataset partitions")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, partitions)
}

func isNotFoundErr(err error) bool {
	return strings.Contains(err.Error(), "not found")
}
