package httpframework

import (
	"os"
	"sync"

	"github.com/Meesho/BharatMLStack/skye/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
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
		appName := viper.GetString("APP_NAME")
		if appName == "" {
			log.Fatal().Msg("APP_NAME cannot be empty!!!")
		}
		router = gin.New()
		middlewares = append(middlewares, otelgin.Middleware(appName), middleware.HTTPLogger(), middleware.HTTPRecovery())
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

// ResetForTesting resets the global state for testing purposes
// This function should only be used in tests
func ResetForTesting() {
	router = nil
	once = sync.Once{}
}
