package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/etcdstate"
)

// GetTopology handles GET /api/v1/tenants/:tenant/stores/:store/topology.
func (h *Handlers) GetTopology(c *gin.Context) {
	tenant := c.Param("tenant")
	store := c.Param("store")

	topo, err := h.state.GetTopology(c.Request.Context(), tenant, store)
	if err != nil {
		if errors.Is(err, etcdstate.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, topo)
}
