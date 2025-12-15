package http

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates authentication headers (mirrors gRPC ServerInterceptor)
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for health check
		if c.Request.URL.Path == "/health/self" {
			c.Next()
			return
		}

		callerId := c.GetHeader("online-feature-store-caller-id")
		authToken := c.GetHeader("online-feature-store-auth-token")

		if callerId == "" {
			c.JSON(400, gin.H{"error": "online-feature-store-caller-id header is missing"})
			c.Abort()
			return
		}

		if authToken == "" {
			c.JSON(400, gin.H{"error": "online-feature-store-auth-token header is missing"})
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
