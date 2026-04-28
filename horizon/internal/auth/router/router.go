package router

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/auth/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
	"github.com/gin-gonic/gin"
)

// Init expects http framework to be initialized before calling this function
func Init() {
	authController := controller.NewController()
	api := httpframework.Instance().Group("/")
	{
		api.POST("/register", authController.Register)
		api.POST("/login", authController.Login)
		api.POST("/logout", authController.Logout)
		api.GET("/users", authController.GetAllUsers)
		api.PUT("/update-user", authController.UpdateUserAccessAndRole)
		api.GET("/health", Health)

		// Session tracking
		api.POST("/track-session", authController.TrackSession)

		// SSO/OAuth routes (public endpoints)
		auth := api.Group("/auth")
		{
			auth.GET("/sso/status", authController.GetSSOStatus)
			auth.GET("/google/initiate", authController.InitiateGoogleOAuth)
			auth.GET("/google/callback", authController.GoogleOAuthCallback)
			auth.POST("/refresh", authController.RefreshToken)
		}
	}

	// Permission routes
	permissionController := controller.NewPermissionController()

	// Root level permission routes (used by PermissionManagement frontend)
	{
		api.GET("/permissions", permissionController.GetAllPermissions)
		api.POST("/permissions", permissionController.CreatePermission)
		api.PUT("/permissions/:id", permissionController.UpdatePermission)
		api.DELETE("/permissions/:id", permissionController.DeletePermission)
		api.PUT("/permissions/role/:role/bulk", permissionController.BulkUpdatePermissionsByRole)
	}

	// API v1 routes
	apiV1 := httpframework.Instance().Group("/api/v1/horizon")
	{
		// Get permissions for current user's role (used by frontend after login)
		apiV1.GET("/permission-by-role", permissionController.GetPermissionsByCurrentUserRole)

		// Permission management routes (super_admin only) - also available at root level
		apiV1.GET("/permissions", permissionController.GetAllPermissions)
		apiV1.GET("/permissions/:role", permissionController.GetPermissionsByRole)
		apiV1.POST("/permissions", permissionController.CreatePermission)
		apiV1.PUT("/permissions/:id", permissionController.UpdatePermission)
		apiV1.DELETE("/permissions/:id", permissionController.DeletePermission)
		apiV1.PUT("/permissions/role/:role", permissionController.BulkUpdatePermissionsByRole)
	}

	// Metadata routes
	metadataController := controller.NewMetadataController()
	metadata := httpframework.Instance().Group("/metadata")
	{
		// Service routes
		metadata.GET("/services", metadataController.GetAllServices)
		metadata.GET("/services/:id", metadataController.GetServiceByID)
		metadata.POST("/services", metadataController.CreateService)       // super_admin only
		metadata.PUT("/services/:id", metadataController.UpdateService)    // super_admin only
		metadata.DELETE("/services/:id", metadataController.DeleteService) // super_admin only

		// Screen type routes
		// Note: GetAllScreenTypes and GetScreenTypesByServiceID share the same route
		// The controller checks for service_id query parameter to determine which handler to use
		metadata.GET("/screen-types", func(ctx *gin.Context) {
			if ctx.Query("service_id") != "" {
				metadataController.GetScreenTypesByServiceID(ctx)
			} else {
				metadataController.GetAllScreenTypes(ctx)
			}
		})
		metadata.GET("/screen-types/:id", metadataController.GetScreenTypeByID)
		metadata.POST("/screen-types", metadataController.CreateScreenType)       // super_admin only
		metadata.PUT("/screen-types/:id", metadataController.UpdateScreenType)    // super_admin only
		metadata.DELETE("/screen-types/:id", metadataController.DeleteScreenType) // super_admin only

		// Action routes
		metadata.GET("/actions", metadataController.GetAllActions)
		metadata.GET("/actions/:id", metadataController.GetActionByID)
		metadata.POST("/actions", metadataController.CreateAction)       // super_admin only
		metadata.PUT("/actions/:id", metadataController.UpdateAction)    // super_admin only
		metadata.DELETE("/actions/:id", metadataController.DeleteAction) // super_admin only
		api.GET("/api/v1/horizon/permission-by-role", controller.NewController().GetPermissionByRole)
	}
}

func Health(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Application is up!!!"})
}
