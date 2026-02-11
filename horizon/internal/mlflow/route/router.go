package route

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/mlflow/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
)

var initMLFlowRouterOnce sync.Once

// Init initializes the MLFlow proxy routes
// Expects http framework to be initialized before calling this function
func Init() {
	initMLFlowRouterOnce.Do(func() {
		api := httpframework.Instance().Group("/api/v1/mlflow")
		{
			// Use wildcard route to catch all MLFlow API paths
			// This will forward all requests like /api/v1/mlflow/* to the MLFlow backend
			api.Any("/*mlflowpath", controller.NewMLFlowController().ProxyToMLFlow)
		}
	})
}
