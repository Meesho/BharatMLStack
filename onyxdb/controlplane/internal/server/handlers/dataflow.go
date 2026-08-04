package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// PutDataflow handles PUT /api/v1/tenants/:tenant/stores/:store/dataflow.
func (h *Handlers) PutDataflow(c *gin.Context) {
	tenant := c.Param("tenant")
	store := c.Param("store")

	var cfg model.DataflowConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.state.PutDataflow(c.Request.Context(), tenant, store, cfg); err != nil {
		if errors.Is(err, etcdstate.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant": tenant,
		"store":  store,
		"status": "dataflow config saved",
	})
}

// GetDataflow handles GET /api/v1/tenants/:tenant/stores/:store/dataflow.
func (h *Handlers) GetDataflow(c *gin.Context) {
	tenant := c.Param("tenant")
	store := c.Param("store")

	cfg, err := h.state.GetDataflow(c.Request.Context(), tenant, store)
	if err != nil {
		if errors.Is(err, etcdstate.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "dataflow config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cfg)
}
