package route

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/skye/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
)

func Init(appConfig configs.Configs) {
	api := httpframework.Instance().Group("/api")
	controller := controller.NewConfigController(appConfig)
	{
		v1 := api.Group("/v1/horizon/skye")
		{
			storeRequests := v1.Group("/requests/store")
			{
				storeRequests.POST("/register", controller.RegisterStore)
				storeRequests.POST("/approve", controller.ApproveStoreRequest)
			}
			storeData := v1.Group("/data/stores")
			{
				storeData.GET("", controller.GetStores)
			}
			storeRequestsData := v1.Group("/data/store-requests")
			{
				storeRequestsData.GET("", controller.GetAllStoreRequests)
			}
			entityRequests := v1.Group("/requests/entity")
			{
				entityRequests.POST("/register", controller.RegisterEntity)
				entityRequests.POST("/approve", controller.ApproveEntityRequest)
			}

			entityData := v1.Group("/data/entities")
			{
				entityData.GET("", controller.GetEntities)
			}

			entityRequestsData := v1.Group("/data/entity-requests")
			{
				entityRequestsData.GET("", controller.GetAllEntityRequests)
			}

			// Model Registration and Management
			modelRequests := v1.Group("/requests/model")
			{
				modelRequests.POST("/register", controller.RegisterModel)
				modelRequests.POST("/edit", controller.EditModel)
				modelRequests.POST("/approve", controller.ApproveModelRequest)
				modelRequests.POST("/edit/approve", controller.ApproveModelEditRequest)
			}

			modelData := v1.Group("/data/models")
			{
				modelData.GET("", controller.GetModels)
			}

			modelRequestsData := v1.Group("/data/model-requests")
			{
				modelRequestsData.GET("", controller.GetAllModelRequests)
			}

			// Variant Registration and Management
			variantRequests := v1.Group("/requests/variant")
			{
				variantRequests.POST("/register", controller.RegisterVariant)
				variantRequests.POST("/edit", controller.EditVariant)
				variantRequests.POST("/approve", controller.ApproveVariantRequest)
				variantRequests.POST("/edit/approve", controller.ApproveVariantEditRequest)
			}

			variantData := v1.Group("/data/variants")
			{
				variantData.GET("", controller.GetVariants)
			}

			// Added new group for variant requests
			variantRequestsData := v1.Group("/data/variant-requests")
			{
				variantRequestsData.GET("", controller.GetAllVariantRequests)
			}

			// Filter Registration and Management
			filterRequests := v1.Group("/requests/filter")
			{
				filterRequests.POST("/register", controller.RegisterFilter)
				filterRequests.POST("/approve", controller.ApproveFilterRequest)
			}

			filterData := v1.Group("/data/filters")
			{
				filterData.GET("", controller.GetFilters)
			}

			// Added new group for filter requests
			filterRequestsData := v1.Group("/data/filter-requests")
			{
				filterRequestsData.GET("", controller.GetAllFilterRequests)
			}

			// ==================== JOB FREQUENCY OPERATIONS ====================

			// Job Frequency Registration and Management
			jobFrequencyRequests := v1.Group("/requests/job-frequency")
			{
				jobFrequencyRequests.POST("/register", controller.RegisterJobFrequency)
				jobFrequencyRequests.POST("/approve", controller.ApproveJobFrequencyRequest)
			}

			jobFrequencyData := v1.Group("/data/job-frequencies")
			{
				jobFrequencyData.GET("", controller.GetJobFrequencies)
			}

			jobFrequencyRequestsData := v1.Group("/data/job-frequency-requests")
			{
				jobFrequencyRequestsData.GET("", controller.GetAllJobFrequencyRequests)
			}

			// ==================== MQ ID TO TOPICS OPERATIONS ====================

			mqIdTopicsData := v1.Group("/data/mq-id-topics")
			{
				mqIdTopicsData.GET("", controller.GetMQIdTopics)
			}

			variantsListData := v1.Group("/data/variants-list")
			{
				variantsListData.GET("", controller.GetVariantsList)
			}
			// Variant Promotion Operations
			variantPromotionRequests := v1.Group("/requests/variant")
			{
				variantPromotionRequests.POST("/promote", controller.PromoteVariant)
				variantPromotionRequests.POST("/promote/approve", controller.ApproveVariantPromotionRequest)
			}

			// Variant Onboarding Operations
			variantOnboardingRequests := v1.Group("/requests/variant")
			{
				variantOnboardingRequests.POST("/onboard", controller.OnboardVariant)
				variantOnboardingRequests.POST("/onboard/approve", controller.ApproveVariantOnboardingRequest)
			}

			// Variant Onboarding Data Operations
			variantOnboardingData := v1.Group("/data/variant-onboarding")
			{
				variantOnboardingData.GET("/requests", controller.GetAllVariantOnboardingRequests)
				variantOnboardingData.GET("/tasks", controller.GetVariantOnboardingTasks)
			}
		}
	}
}
