package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/sizing"
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
)

// CreateStoreRequest is the request body for onboarding a new store.
type CreateStoreRequest struct {
	Name       string `json:"name"       binding:"required"`
	EntityKey  string `json:"entityKey"  binding:"required"`
	ShardCount int    `json:"shardCount" binding:"required,min=1"`
}

// CreateStoreResponse is the response body after creating a store.
type CreateStoreResponse struct {
	Tenant     string        `json:"tenant"`
	Store      string        `json:"store"`
	EntityKey  string        `json:"entityKey"`
	ShardCount int           `json:"shardCount"`
	Sizing     sizing.Output `json:"sizing"`
}

// OnboardStore handles POST /api/v1/tenants/:tenant/stores.
func (h *Handlers) OnboardStore(c *gin.Context) {
	tenant := c.Param("tenant")

	var req CreateStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg := model.StoreConfig{
		Tenant:     tenant,
		Store:      req.Name,
		EntityKey:  req.EntityKey,
		ShardCount: req.ShardCount,
	}

	if err := h.state.CreateStore(c.Request.Context(), cfg); err != nil {
		if errors.Is(err, etcdstate.ErrAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "store already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rec := sizing.Compute(sizing.Input{DatasetSizeGB: 0, TargetRPS: 0})
	c.JSON(http.StatusCreated, CreateStoreResponse{
		Tenant:     tenant,
		Store:      req.Name,
		EntityKey:  req.EntityKey,
		ShardCount: req.ShardCount,
		Sizing:     rec,
	})
}

// GetStore handles GET /api/v1/tenants/:tenant/stores/:store.
func (h *Handlers) GetStore(c *gin.Context) {
	tenant := c.Param("tenant")
	store := c.Param("store")

	state, err := h.state.GetStore(c.Request.Context(), tenant, store)
	if err != nil {
		if errors.Is(err, etcdstate.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, state)
}
