package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/Meesho/BharatMLStack/horizon/internal/featurereview/handler"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Controller struct {
	handler *handler.Handler
}

func New(h *handler.Handler) *Controller {
	return &Controller{handler: h}
}

func (c *Controller) HandleWebhook(ctx *gin.Context) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	signature := ctx.GetHeader("X-Hub-Signature-256")
	if !c.handler.VerifyWebhookSignature(body, signature) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	event := ctx.GetHeader("X-GitHub-Event")
	if event != "pull_request" {
		ctx.JSON(http.StatusOK, gin.H{"message": "event ignored", "event": event})
		return
	}

	var payload handler.WebhookPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		// Body was already read; re-parse from the raw bytes
		if err := bindJSON(body, &payload); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
	}

	if err := c.handler.HandleWebhook(ctx.Request.Context(), payload); err != nil {
		log.Error().Err(err).Msg("webhook processing failed")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "webhook processing failed"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "webhook processed"})
}

func (c *Controller) ListReviews(ctx *gin.Context) {
	status := ctx.Query("status")
	repo := ctx.Query("repo")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(ctx.DefaultQuery("per_page", "20"))

	result, err := c.handler.ListReviews(ctx.Request.Context(), status, repo, page, perPage)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) GetReview(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid review ID"})
		return
	}

	result, err := c.handler.GetReview(ctx.Request.Context(), id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "review not found"})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) ComputePlan(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid review ID"})
		return
	}

	result, err := c.handler.ComputePlan(ctx.Request.Context(), id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) Approve(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid review ID"})
		return
	}

	var req handler.ApproveRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		req = handler.ApproveRequest{}
	}

	reviewedBy := ctx.GetHeader("X-User-Email")
	if reviewedBy == "" {
		reviewedBy = "admin"
	}

	result, err := c.handler.Approve(ctx.Request.Context(), id, reviewedBy, req.Comment)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) Reject(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid review ID"})
		return
	}

	var req handler.RejectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.Comment == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "comment is required for rejection"})
		return
	}

	reviewedBy := ctx.GetHeader("X-User-Email")
	if reviewedBy == "" {
		reviewedBy = "admin"
	}

	if err := c.handler.Reject(ctx.Request.Context(), id, reviewedBy, req.Comment); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "review rejected", "review_id": id, "status": "rejected"})
}

func (c *Controller) MergePR(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid review ID"})
		return
	}

	mergedBy := ctx.GetHeader("X-User-Email")
	if mergedBy == "" {
		mergedBy = "admin"
	}

	if err := c.handler.MergePR(ctx.Request.Context(), id, mergedBy); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "PR merge initiated", "review_id": id})
}

func (c *Controller) GetStats(ctx *gin.Context) {
	result, err := c.handler.GetStats(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func bindJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
