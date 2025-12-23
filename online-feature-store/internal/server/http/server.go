package http

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	router *gin.Engine
	once   sync.Once
)

func Init() {
	once.Do(func() {
		env := viper.GetString("APP_ENV")
		if env == "prod" || env == "production" {
			gin.SetMode(gin.ReleaseMode)
		}
		router = gin.New()

		router.Use(gin.Recovery())
		router.Use(gin.Logger())
		router.Use(AuthMiddleware())

		router.GET("/health/self", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "true"})
		})

		// Register API routes
		RegisterRoutes(router)
	})
}

func Instance() *gin.Engine {
	if router == nil {
		log.Fatal().Msg("Router not initialized")
	}
	return router
}
