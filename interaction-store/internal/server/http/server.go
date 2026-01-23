package http

import (
	"net/http"
	"sync"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

var (
	router *gin.Engine
	once   sync.Once
)

func Init(config config.Configs) {
	once.Do(func() {
		env := config.AppEnv
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
