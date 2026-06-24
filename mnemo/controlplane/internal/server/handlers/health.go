package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// readyTimeout bounds the etcd round-trip the readiness probe makes.
const readyTimeout = time.Second

// Ready handles GET /api/v1/ready — the readiness probe. It pings etcd; if etcd
// is unreachable it returns 503 so Kubernetes takes the pod out of the Service
// endpoints (vs liveness /api/v1/health, which stays 200 so the pod isn't killed
// for a transient etcd blip).
func (h *Handlers) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), readyTimeout)
	defer cancel()
	if err := h.state.Health(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": "etcd unreachable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
