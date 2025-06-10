package router

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
)

// Init expects http framework to be initialized before calling this function
func Init() {
	api := httpframework.Instance().Group("/api")
	{
		v1 := api.Group("/v1")
		model := v1.Group("/online-feature-store")
		{
			// Register/Edit APIs
			model.POST("/register-store", controller.NewConfigController().RegisterStore)
			model.POST("/register-entity", controller.NewConfigController().RegisterEntity)
			model.POST("/edit-entity", controller.NewConfigController().EditEntity)
			model.POST("/register-feature-group", controller.NewConfigController().RegisterFeatureGroup)
			model.POST("/edit-feature-group", controller.NewConfigController().EditFeatureGroup)
			model.POST("/add-features", controller.NewConfigController().AddFeatures)
			model.POST("/edit-features", controller.NewConfigController().EditFeatures)
			model.POST("/register-job", controller.NewConfigController().RegisterJob)

			// Get APIs
			model.GET("/get-entites", controller.NewConfigController().GetEntities)
			model.GET("/get-config", controller.NewConfigController().GetConfig)
			model.GET("/get-store", controller.NewConfigController().GetStores)
			model.GET("/get-jobs", controller.NewConfigController().GetJobs)
			model.GET("/get-feature-groups", controller.NewConfigController().GetFeatureGroup)
			model.GET("/retrieve-entities", controller.NewConfigController().RetrieveEntities)
			model.GET("/retrieve-feature-groups", controller.NewConfigController().RetrieveFeatureGroups)

			// Process APIs
			model.POST("/process-store", controller.NewConfigController().ProcessStore)
			model.POST("/process-entity", controller.NewConfigController().ProcessEntity)
			model.POST("/process-feature-group", controller.NewConfigController().ProcessFeatureGroup)
			model.POST("/process-job", controller.NewConfigController().ProcessJob)
			model.POST("/process-add-features", controller.NewConfigController().ProcessAddFeatures)

			// Get Requests APIs
			model.GET("/get-store-requests", controller.NewConfigController().GetAllStoresRequestForUser)
			model.GET("/get-entity-requests", controller.NewConfigController().GetAllEntitiesRequestForUser)
			model.GET("/get-feature-group-requests", controller.NewConfigController().GetAllFeatureGroupsRequestForUser)
			model.GET("/get-job-requests", controller.NewConfigController().GetAllJobsRequestForUser)
			model.GET("/get-add-features-requests", controller.NewConfigController().GetAllFeaturesRequestForUser)
			model.POST("/get-source-mapping", controller.NewConfigController().RetrieveSourceMapping)
			model.POST("/get-online-features-mapping", controller.NewConfigController().GetOnlineFeatureMapping)
		}
	}
}
