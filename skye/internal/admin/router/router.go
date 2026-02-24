package router

import (
	"github.com/Meesho/BharatMLStack/skye/internal/admin/controller"
	"github.com/Meesho/BharatMLStack/skye/pkg/httpframework"
)

const (
	HeathCheckPath = "/health"
)

// Init expects http framework to be initialized before calling this function
func Init() {
	api := httpframework.Instance().Group("/api")
	{
		v1 := api.Group("/v1")
		model := v1.Group("/model")
		{
			model.POST("/register-model", controller.NewRegistryController().RegisterModel)
			model.POST("/register-variant", controller.NewRegistryController().RegisterVariant)
			model.POST("/register-store", controller.NewRegistryController().RegisterStore)
			model.POST("/register-frequency", controller.NewRegistryController().RegisterFrequency)
			model.POST("/register-entity", controller.NewRegistryController().RegisterEntity)
		}
		{
			qdrant := v1.Group("/qdrant")
			qdrant.POST("/create-collection", controller.NewQdrantController().CreateCollection)
			qdrant.POST("/process-model", controller.NewQdrantController().ProcessModel)
			qdrant.POST("/process-multi-variant", controller.NewQdrantController().ProcessMultiVariant)
			qdrant.POST("/process-models-with-freq", controller.NewQdrantController().ProcessModelsWithFrequency)
			qdrant.POST("/promote-variant", controller.NewQdrantController().PromoteVariant)
			qdrant.POST("/trigger-indexing", controller.NewQdrantController().TriggerIndexing)
			qdrant.POST("/process-multi-variant-force-reset", controller.NewQdrantController().ProcessMultiVariantForceReset)
		}
	}

	// Init health check
	httpframework.Instance().GET(HeathCheckPath, controller.Health)
}
