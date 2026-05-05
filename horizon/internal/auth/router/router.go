package router

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/auth/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
	"github.com/gin-gonic/gin"
)

// Init expects http framework to be initialized before calling this function
func Init() {
	api := httpframework.Instance().Group("/")
	{
		api.POST("/register", controller.NewController().Register)
		api.POST("/login", controller.NewController().Login)
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
