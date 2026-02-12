package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

const (
	callerIdHeader = "interaction-store-caller-id"
)

// AuthMiddleware validates authentication headers
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for health check
		if c.Request.URL.Path == "/health/self" {
			c.Next()
			return
		}

		callerId := c.GetHeader(callerIdHeader)

		if callerId == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": callerIdHeader + " header is missing"})
			c.Abort()
			return
		}

		c.Next()
	}
}
