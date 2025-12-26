package controller

import (
	"fmt"
	"net/http"
	"sync"

	workflowHandler "github.com/Meesho/BharatMLStack/horizon/internal/workflow/handler"
	"github.com/gin-gonic/gin"
)

type Config interface {
	GetWorkflowStatus(ctx *gin.Context)
}

var (
	workflow         Config
	workflowInitOnce sync.Once
)

type V1 struct {
	handler workflowHandler.Handler
}

func NewController() Config {
	if workflow == nil {
		workflowInitOnce.Do(func() {
			handler := workflowHandler.GetWorkflowHandler()
			if handler == nil {
				panic("Failed to initialize workflow handler")
			}
			workflow = &V1{
				handler: handler,
			}
		})
	}
	return workflow
}

// GetWorkflowStatus retrieves the status of a workflow by workflow ID
// GET /api/v1/horizon/workflow/status?workflowId=<workflow-id>
func (w *V1) GetWorkflowStatus(ctx *gin.Context) {
	workflowID := ctx.Query("workflowId")
	if workflowID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "workflowId query parameter is required",
			"data":  nil,
		})
		return
	}

	status, err := w.handler.GetWorkflowStatus(workflowID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Error fetching workflow status: %v", err),
			"data":  nil,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"error": nil,
		"data":  status,
	})
}

