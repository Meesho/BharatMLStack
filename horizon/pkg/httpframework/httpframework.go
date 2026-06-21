package httpframework

import (
	"os"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
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
		// By default gin trusts all proxies, which makes c.ClientIP() honour a
		// client-supplied X-Forwarded-For header. That lets a caller spoof their
		// IP and bypass per-IP protections such as the /login rate limiter. Only
		// trust the proxies explicitly listed in TRUSTED_PROXIES (comma
		// separated CIDRs/IPs); if unset, trust none so ClientIP() falls back to
		// the real remote address.
		var trusted []string
		if tp := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES")); tp != "" {
			for _, p := range strings.Split(tp, ",") {
				if p = strings.TrimSpace(p); p != "" {
					trusted = append(trusted, p)
				}
			}
		}
		if err := router.SetTrustedProxies(trusted); err != nil {
			log.Error().Err(err).Msg("Failed to set trusted proxies; defaulting to trust none")
			_ = router.SetTrustedProxies(nil)
		}
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
