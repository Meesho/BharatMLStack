package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
)

// PutClientConfig handles PUT /api/v1/tenants/:tenant/stores/:store/clientConfig.
func (h *Handlers) PutClientConfig(c *gin.Context) {
	tenant := c.Param("tenant")
	store := c.Param("store")

	var cfg model.ClientConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.state.SetClientConfig(c.Request.Context(), tenant, store, cfg); err != nil {
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
		"status": "client config saved",
	})
}

// GetClientConfig handles GET /api/v1/tenants/:tenant/stores/:store/clientConfig.
func (h *Handlers) GetClientConfig(c *gin.Context) {
	tenant := c.Param("tenant")
	store := c.Param("store")

	cfg, err := h.state.GetClientConfig(c.Request.Context(), tenant, store)
	if err != nil {
		if errors.Is(err, etcdstate.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "client config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cfg)
}
