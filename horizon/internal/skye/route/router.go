package route

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/skye/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
)

func Init(appConfig configs.Configs) {
	api := httpframework.Instance().Group("/api")
	ctrl := controller.NewConfigController(appConfig)
	v1 := api.Group("/v1/horizon/skye")

	storeRequests := v1.Group("/requests/store")
	storeRequests.POST("/register", ctrl.RegisterStore)
	storeRequests.POST("/approve", ctrl.ApproveStoreRequest)

	v1.GET("/data/stores", ctrl.GetStores)
	v1.GET("/data/store-requests", ctrl.GetAllStoreRequests)

	entityRequests := v1.Group("/requests/entity")
	entityRequests.POST("/register", ctrl.RegisterEntity)
	entityRequests.POST("/approve", ctrl.ApproveEntityRequest)

	v1.GET("/data/entities", ctrl.GetEntities)
	v1.GET("/data/entity-requests", ctrl.GetAllEntityRequests)

	modelRequests := v1.Group("/requests/model")
	modelRequests.POST("/register", ctrl.RegisterModel)
	modelRequests.POST("/approve", ctrl.ApproveModelRequest)

	v1.GET("/data/models", ctrl.GetModels)
	v1.GET("/data/model-requests", ctrl.GetAllModelRequests)

	variantRequests := v1.Group("/requests/variant")
	variantRequests.POST("/register", ctrl.RegisterVariant)
	variantRequests.POST("/approve", ctrl.ApproveVariantRequest)
	variantRequests.POST("/scaleup", ctrl.ScaleUpVariant)
	variantRequests.POST("/scaleup/approve", ctrl.ApproveVariantScaleUpRequest)

	v1.GET("/data/variants", ctrl.GetVariants)
	v1.GET("/data/variant-requests", ctrl.GetAllVariantRequests)

	filterRequests := v1.Group("/requests/filter")
	filterRequests.POST("/register", ctrl.RegisterFilter)
	filterRequests.POST("/approve", ctrl.ApproveFilterRequest)

	v1.GET("/data/filters", ctrl.GetFilters)
	v1.GET("/data/all-filters", ctrl.GetAllFilters)
	v1.GET("/data/filter-requests", ctrl.GetAllFilterRequests)

	jobFrequencyRequests := v1.Group("/requests/job-frequency")
	jobFrequencyRequests.POST("/register", ctrl.RegisterJobFrequency)
	jobFrequencyRequests.POST("/approve", ctrl.ApproveJobFrequencyRequest)

	v1.GET("/data/job-frequencies", ctrl.GetJobFrequencies)
	v1.GET("/data/job-frequency-requests", ctrl.GetAllJobFrequencyRequests)

	v1.GET("/data/mq-id-topics", ctrl.GetMQIdTopics)
	v1.GET("/data/variants-list", ctrl.GetVariantsList)

	variantOnboardingData := v1.Group("/data/variant-onboarding")
	// variantOnboardingData.GET("/requests", ctrl.GetAllVariantOnboardingRequests)
	variantOnboardingData.GET("/tasks", ctrl.GetVariantOnboardingTasks)

	variantScaleUpData := v1.Group("/data/variant-scaleup")
	variantScaleUpData.GET("/requests", ctrl.GetAllVariantScaleUpRequests)
	variantScaleUpData.GET("/tasks", ctrl.GetVariantScaleUpTasks)

	v1.POST("/test/variant/execute-request", ctrl.TestVariant)
	v1.POST("/test/variant/generate-request", ctrl.GenerateTestRequest)
}
