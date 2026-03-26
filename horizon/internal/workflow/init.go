package workflow

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/workflow/activities"
)

var (
	initWorkflowOnce sync.Once
	WorkflowAppName  string
	AppEnv           string
)

func Init(config configs.Configs) {
	initWorkflowOnce.Do(func() {
		WorkflowAppName = "horizon" // Use "horizon" as app name for workflow storage
		AppEnv = config.AppEnv

		// Initialize activities configuration (for test mode)
		activities.InitActivitiesConfig(config)
	})
}
