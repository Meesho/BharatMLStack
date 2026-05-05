package router

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/workflow/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
)

var (
	initWorkflowRouterOnce sync.Once
)

// Init expects http framework to be initialized before calling this function
func Init() {
	initWorkflowRouterOnce.Do(func() {
		workflowAPI := httpframework.Instance().Group("/api/v1/horizon/workflow")
		{
			// Workflow status query endpoint
			workflowAPI.GET("/status", controller.NewController().GetWorkflowStatus)
		}
	})
}

