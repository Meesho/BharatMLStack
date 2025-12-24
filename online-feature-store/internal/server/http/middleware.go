package http

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/gin-gonic/gin"
)

const (
	callerIdHeader  = "online-feature-store-caller-id"
	AuthTokenHeader = "online-feature-store-auth-token"
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
		authToken := c.GetHeader(AuthTokenHeader)

		if callerId == "" {
			c.JSON(400, gin.H{"error": callerIdHeader + " header is missing"})
			c.Abort()
			return
		}

		if authToken == "" {
			c.JSON(400, gin.H{"error": AuthTokenHeader + " header is missing"})
			c.Abort()
			return
		}

		// Validate token using existing config (same as gRPC)
		configManager := config.Instance(config.DefaultVersion)
		registeredClients := configManager.GetAllRegisteredClients()
		expectedToken, ok := registeredClients[callerId]

		if !ok || expectedToken != authToken {
			c.JSON(401, gin.H{"error": "Invalid auth token"})
			c.Abort()
			return
		}

		c.Next()
	}
}
