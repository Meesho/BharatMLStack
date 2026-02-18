package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/Meesho/BharatMLStack/go-sdk/pkg/api"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// HTTPRecovery handles context errors/panics and sets response code accordingly
func HTTPRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if len(c.Errors) > 0 {
				err := c.Errors.Last().Err
				if apiErr, ok := err.(*api.Error); ok {
					c.JSON(apiErr.StatusCode, gin.H{"error": apiErr.Message})
					c.Abort()
				}
			}
			if err := recover(); err != nil {
				log.Error().Msgf("Panic occurred: %v\n%s", err, debug.Stack())
				errorMsg := fmt.Sprintf("%v", err)
				c.JSON(500, gin.H{"error": errorMsg})
				c.Abort()
			}
		}()
		c.Next()
	}
}
