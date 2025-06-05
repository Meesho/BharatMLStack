package httpframework

import (
	"github.com/Meesho/BharatMLStack/horizon/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"os"
	"sync"
)

var (
	router *gin.Engine
	once   sync.Once
)

// Init initializes gin engine with the given middlewares
// It sets the gin mode to release if the environment is production and use the middleware logger and recovery
func Init(middlewares ...gin.HandlerFunc) {
	once.Do(func() {
		env := os.Getenv("APP_ENV")
		if env == "prod" || env == "production" {
			gin.SetMode(gin.ReleaseMode)
		}
		router = gin.New()
		middlewares = append(middlewares, middleware.HTTPLogger(), middleware.HTTPRecovery())
		router.Use(middlewares...)
	})
}

// Instance returns the httpframework instance
func Instance() *gin.Engine {
	if router == nil {
		log.Fatal().Msg("Router not initialized")
	}
	return router
}
