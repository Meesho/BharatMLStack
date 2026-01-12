package route

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/skye/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
)

func Init() {
	api := httpframework.Instance().Group("/api")
	{
		v1 := api.Group("/v1/horizon/skye")
		{
			// ==================== PLATFORM REGISTRATION OPERATIONS ====================

			// Store Registration and Management
			storeRequests := v1.Group("/requests/store")
			{
				storeRequests.POST("/register", controller.NewConfigController().RegisterStore)
				storeRequests.POST("/approve", controller.NewConfigController().ApproveStoreRequest)
			}

			storeData := v1.Group("/data/stores")
			{
				storeData.GET("", controller.NewConfigController().GetStores)
			}

			storeRequestsData := v1.Group("/data/store-requests")
			{
				storeRequestsData.GET("", controller.NewConfigController().GetAllStoreRequests)
			}

			// Entity Registration and Management
			entityRequests := v1.Group("/requests/entity")
			{
				entityRequests.POST("/register", controller.NewConfigController().RegisterEntity)
				entityRequests.POST("/approve", controller.NewConfigController().ApproveEntityRequest)
			}

			entityData := v1.Group("/data/entities")
			{
				entityData.GET("", controller.NewConfigController().GetEntities)
			}

			entityRequestsData := v1.Group("/data/entity-requests")
			{
				entityRequestsData.GET("", controller.NewConfigController().GetAllEntityRequests)
			}

			// Model Registration and Management
			modelRequests := v1.Group("/requests/model")
			{
				modelRequests.POST("/register", controller.NewConfigController().RegisterModel)
				modelRequests.POST("/edit", controller.NewConfigController().EditModel)
				modelRequests.POST("/approve", controller.NewConfigController().ApproveModelRequest)
				modelRequests.POST("/edit/approve", controller.NewConfigController().ApproveModelEditRequest)
			}

			modelData := v1.Group("/data/models")
			{
				modelData.GET("", controller.NewConfigController().GetModels)
			}

			modelRequestsData := v1.Group("/data/model-requests")
			{
				modelRequestsData.GET("", controller.NewConfigController().GetAllModelRequests)
			}

			// Variant Registration and Management
			variantRequests := v1.Group("/requests/variant")
			{
				variantRequests.POST("/register", controller.NewConfigController().RegisterVariant)
				variantRequests.POST("/edit", controller.NewConfigController().EditVariant)
				variantRequests.POST("/approve", controller.NewConfigController().ApproveVariantRequest)
				variantRequests.POST("/edit/approve", controller.NewConfigController().ApproveVariantEditRequest)
			}

			variantData := v1.Group("/data/variants")
			{
				variantData.GET("", controller.NewConfigController().GetVariants)
			}

			// Added new group for variant requests
			variantRequestsData := v1.Group("/data/variant-requests")
			{
				variantRequestsData.GET("", controller.NewConfigController().GetAllVariantRequests)
			}

			// Filter Registration and Management
			filterRequests := v1.Group("/requests/filter")
			{
				filterRequests.POST("/register", controller.NewConfigController().RegisterFilter)
				filterRequests.POST("/approve", controller.NewConfigController().ApproveFilterRequest)
			}

			filterData := v1.Group("/data/filters")
			{
				filterData.GET("", controller.NewConfigController().GetFilters)
			}

			// Added new group for filter requests
			filterRequestsData := v1.Group("/data/filter-requests")
			{
				filterRequestsData.GET("", controller.NewConfigController().GetAllFilterRequests)
			}

			// ==================== JOB FREQUENCY OPERATIONS ====================

			// Job Frequency Registration and Management
			jobFrequencyRequests := v1.Group("/requests/job-frequency")
			{
				jobFrequencyRequests.POST("/register", controller.NewConfigController().RegisterJobFrequency)
				jobFrequencyRequests.POST("/approve", controller.NewConfigController().ApproveJobFrequencyRequest)
			}

			jobFrequencyData := v1.Group("/data/job-frequencies")
			{
				jobFrequencyData.GET("", controller.NewConfigController().GetJobFrequencies)
			}

			jobFrequencyRequestsData := v1.Group("/data/job-frequency-requests")
			{
				jobFrequencyRequestsData.GET("", controller.NewConfigController().GetAllJobFrequencyRequests)
			}

			// ==================== DEPLOYMENT OPERATIONS ====================

			// Qdrant Cluster Operations
			qdrantRequests := v1.Group("/requests/qdrant")
			{
				qdrantRequests.POST("/create-cluster", controller.NewConfigController().CreateQdrantCluster)
				qdrantRequests.POST("/approve", controller.NewConfigController().ApproveQdrantClusterRequest)
			}

			qdrantData := v1.Group("/data/qdrant")
			{
				qdrantData.GET("/clusters", controller.NewConfigController().GetQdrantClusters)
			}

			// Variant Promotion Operations
			variantPromotionRequests := v1.Group("/requests/variant")
			{
				variantPromotionRequests.POST("/promote", controller.NewConfigController().PromoteVariant)
				variantPromotionRequests.POST("/promote/approve", controller.NewConfigController().ApproveVariantPromotionRequest)
			}

			// Variant Onboarding Operations
			variantOnboardingRequests := v1.Group("/requests/variant")
			{
				variantOnboardingRequests.POST("/onboard", controller.NewConfigController().OnboardVariant)
				variantOnboardingRequests.POST("/onboard/approve", controller.NewConfigController().ApproveVariantOnboardingRequest)
			}

			// Variant Onboarding Data Operations
			variantOnboardingData := v1.Group("/data/variant-onboarding")
			{
				variantOnboardingData.GET("/requests", controller.NewConfigController().GetAllVariantOnboardingRequests)
				variantOnboardingData.GET("/variants", controller.NewConfigController().GetOnboardedVariants)
			}
		}
	}
}
