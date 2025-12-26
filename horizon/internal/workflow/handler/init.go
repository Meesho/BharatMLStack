package handler

import (
	"sync"

	workflowetcd "github.com/Meesho/BharatMLStack/horizon/internal/workflow/etcd"
	"github.com/Meesho/BharatMLStack/horizon/internal/workflow"
	"github.com/rs/zerolog/log"
)

var (
	initWorkflowHandlerOnce sync.Once
	workflowHandler         Handler
)

// InitV1WorkflowHandler initializes the workflow handler (same pattern as numerix InitV1ConfigHandler)
func InitV1WorkflowHandler() Handler {
	initWorkflowHandlerOnce.Do(func() {
		// Create etcd instance (same pattern as numerix)
		etcdRepo := workflowetcd.NewEtcdInstance()
		
		// Create orchestrator with etcd repo (avoids import cycle)
		orchestrator := workflow.NewOrchestrator(etcdRepo)
		worker := workflow.NewWorker(orchestrator, 5) // 5 workers by default
		
		// Start worker pool
		worker.Start()
		
		workflowHandler = &WorkflowHandler{
			orchestrator: orchestrator,
			worker:       worker,
		}
		
		log.Info().Msg("Workflow handler initialized")
	})
	return workflowHandler
}

// GetWorkflowHandler returns the initialized workflow handler
func GetWorkflowHandler() Handler {
	if workflowHandler == nil {
		return InitV1WorkflowHandler()
	}
	return workflowHandler
}

