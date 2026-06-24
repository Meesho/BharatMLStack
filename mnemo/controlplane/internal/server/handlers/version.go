package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/placement"
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
)

// PublishVersionRequest is the request body for publishing a new version.
type PublishVersionRequest struct {
	Date string `json:"date" binding:"required"`
	Run  string `json:"run"  binding:"required"`
}

// PublishVersion handles POST /api/v1/tenants/:tenant/stores/:store/versions/:vId/publish.
func (h *Handlers) PublishVersion(c *gin.Context) {
	tenant := c.Param("tenant")
	store := c.Param("store")
	vID := c.Param("vId")

	var req PublishVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	storeState, err := h.state.GetStore(c.Request.Context(), tenant, store)
	if err != nil {
		if errors.Is(err, etcdstate.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	meta := model.VersionMeta{
		Date:       req.Date,
		Run:        req.Run,
		ShardCount: storeState.Config.ShardCount,
		Status:     model.StatusReady,
	}

	if err := h.state.PublishVersion(c.Request.Context(), tenant, store, vID, meta); err != nil {
		if errors.Is(err, etcdstate.ErrAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "version already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"tenant":    tenant,
		"store":     store,
		"versionId": vID,
		"status":    string(model.StatusReady),
	})
}

// PromoteVersion handles POST /api/v1/tenants/:tenant/stores/:store/versions/:vId/promote.
func (h *Handlers) PromoteVersion(c *gin.Context) {
	tenant := c.Param("tenant")
	store := c.Param("store")
	vID := c.Param("vId")
	ctx := c.Request.Context()

	storeState, err := h.state.GetStore(ctx, tenant, store)
	if err != nil {
		if errors.Is(err, etcdstate.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	pods, err := h.state.ListPods(ctx, tenant, store)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	assignment := placement.DeriveAssignment(storeState.Config.ShardCount, pods, vID)

	var missing []string
	for i := 0; i < storeState.Config.ShardCount; i++ {
		sid := strconv.Itoa(i)
		if len(assignment[sid]) == 0 {
			missing = append(missing, sid)
		}
	}
	sort.Strings(missing)

	if len(missing) > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":   fmt.Sprintf("coverage incomplete: %d/%d shards warm", storeState.Config.ShardCount-len(missing), storeState.Config.ShardCount),
			"missing": missing,
		})
		return
	}

	if err := h.state.PromoteVersion(ctx, tenant, store, vID, assignment); err != nil {
		if errors.Is(err, etcdstate.ErrCASConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "topology version conflict, retry"})
			return
		}
		if errors.Is(err, etcdstate.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant":    tenant,
		"store":     store,
		"versionId": vID,
		"status":    string(model.StatusActive),
	})
}

// Rollback handles POST /api/v1/tenants/:tenant/stores/:store/rollback.
func (h *Handlers) Rollback(c *gin.Context) {
	tenant := c.Param("tenant")
	store := c.Param("store")

	newActive, err := h.state.RollbackStore(c.Request.Context(), tenant, store)
	if err != nil {
		if errors.Is(err, etcdstate.ErrNoRollback) {
			c.JSON(http.StatusConflict, gin.H{"error": "no rollback version available"})
			return
		}
		if errors.Is(err, etcdstate.ErrCASConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "topology version conflict, retry"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant":           tenant,
		"store":            store,
		"newActiveVersion": newActive,
	})
}

// RetireVersion handles POST /api/v1/tenants/:tenant/stores/:store/versions/:vId/retire.
func (h *Handlers) RetireVersion(c *gin.Context) {
	tenant := c.Param("tenant")
	store := c.Param("store")
	vID := c.Param("vId")

	if err := h.state.RetireVersion(c.Request.Context(), tenant, store, vID); err != nil {
		if errors.Is(err, etcdstate.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant":    tenant,
		"store":     store,
		"versionId": vID,
		"status":    string(model.StatusRetiring),
	})
}
