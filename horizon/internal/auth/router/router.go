package router

import (
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/auth/controller"
	"github.com/Meesho/BharatMLStack/horizon/internal/middleware"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// loginRateLimiter throttles login attempts per client IP to slow down
// brute-force / credential-stuffing attacks. Allows ~5 attempts per minute
// with a small burst; idle IP entries are evicted after 10 minutes.
var loginRateLimiter = middleware.NewIPRateLimiter(rate.Every(12*time.Second), 5, 10*time.Minute)

// Init expects http framework to be initialized before calling this function
func Init() {
	api := httpframework.Instance().Group("/")
	{
		api.POST("/register", controller.NewController().Register)
		api.POST("/login", loginRateLimiter.Middleware(), controller.NewController().Login)
		api.POST("/logout", controller.NewController().Logout)
		api.GET("/users", controller.NewController().GetAllUsers)
		api.PUT("/update-user", controller.NewController().UpdateUserAccessAndRole)
		api.GET("/health", Health)
		api.GET("/api/v1/horizon/permission-by-role", controller.NewController().GetPermissionByRole)
	}
}

func Health(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Application is up!!!"})
}
