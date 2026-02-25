package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	infrastructurehandler "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	workflowHandler "github.com/Meesho/BharatMLStack/horizon/internal/workflow/handler"
	"github.com/Meesho/BharatMLStack/horizon/pkg/argocd"
	"github.com/gin-gonic/gin"
)

// getWorkingEnv extracts workingEnv from Gin context (set by middleware)
// This matches RingMaster's pattern where middleware validates and injects workingEnv into context
func getWorkingEnv(ctx *gin.Context) string {
	workingEnv, exists := ctx.Get("workingEnv")
	if !exists {
		// This should not happen if middleware is properly configured
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "workingEnv not found in context"})
		return ""
	}
	workingEnvStr, ok := workingEnv.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "workingEnv has invalid type"})
		return ""
	}
	return workingEnvStr
}

type InfrastructureController struct {
	handler         infrastructurehandler.InfrastructureHandler
	workflowHandler workflowHandler.Handler
}

func NewController() *InfrastructureController {
	return &InfrastructureController{
		handler:         infrastructurehandler.InitInfrastructureHandler(),
		workflowHandler: workflowHandler.GetWorkflowHandler(),
	}
}

type HPAConfigRequest struct {
	AppName string `json:"appName" form:"appName"`
}

type RestartDeploymentRequest struct {
	IsCanary bool `json:"isCanary" form:"isCanary"`
}

type UpdateThresholdRequest struct {
	CPUThreshold string `json:"cpuThreshold" binding:"required"`
}

type UpdateGPUThresholdRequest struct {
	GPUThreshold string `json:"gpuThreshold" binding:"required"`
}

type UpdateSharedMemoryRequest struct {
	Size string `json:"size" binding:"required"` // Kubernetes memory size (e.g., "1Gi", "2Gi")
}

type UpdatePodAnnotationsRequest struct {
	Annotations map[string]string `json:"annotations" binding:"required"` // Pod annotations as key-value pairs
}

type UpdateAutoscalingTriggersRequest struct {
	Triggers []interface{} `json:"triggers" binding:"required"` // Array of trigger objects (CPU, cron, prometheus, etc.)
}

// ApplicationLogsQuery holds query params for GET /applications/:appName/logs
type ApplicationLogsQuery struct {
	PodName      string `form:"podName"`
	Container    string `form:"container"`
	Follow       bool   `form:"follow"`
	Previous     bool   `form:"previous"`
	SinceSeconds string `form:"sinceSeconds"` // string int64, default "0"
	TailLines    string `form:"tailLines"`    // string int64, default "1000"
	Filter       string `form:"filter"`
}

func (c *InfrastructureController) GetHPAConfig(ctx *gin.Context) {
	appName := ctx.Param("appName")
	workingEnv := getWorkingEnv(ctx)
	if workingEnv == "" {
		return // Error already handled in getWorkingEnv
	}

	if appName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "appName is required"})
		return
	}

	hpaConfig, err := c.handler.GetHPAProperties(appName, workingEnv)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, hpaConfig)
}

func (c *InfrastructureController) GetResourceDetail(ctx *gin.Context) {
	appName := ctx.Query("appName")
	workingEnv := getWorkingEnv(ctx)
	if workingEnv == "" {
		return // Error already handled in getWorkingEnv
	}

	if appName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "appName query parameter is required"})
		return
	}

	resourceDetail, err := c.handler.GetResourceDetail(appName, workingEnv)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, resourceDetail)
}

func (c *InfrastructureController) GetApplicationLogs(ctx *gin.Context) {
	appName := strings.TrimSpace(ctx.Param("appName"))
	workingEnv := getWorkingEnv(ctx)
	if workingEnv == "" {
		return
	}
	if appName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"message": "appName is required",
		}})
		return
	}

	var q ApplicationLogsQuery
	if err := ctx.ShouldBindQuery(&q); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"message": "invalid query parameters",
		}})
		return
	}

	container := strings.TrimSpace(q.Container)
	if container == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "container is required",
			},
		})
		return
	}

	if q.Follow {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "streaming (follow=true) is not supported",
			},
		})
		return
	}

	sinceSeconds := int64(0)
	if q.SinceSeconds != "" {
		var err error
		sinceSeconds, err = strconv.ParseInt(q.SinceSeconds, 10, 64)
		if err != nil || sinceSeconds < 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"message": "sinceSeconds must be a non-negative integer",
			}})
			return
		}
	}
	tailLines := int64(1000)
	if q.TailLines != "" {
		var err error
		tailLines, err = strconv.ParseInt(q.TailLines, 10, 64)
		if err != nil || tailLines < 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"message": "tailLines must be a non-negative integer",
			}})
			return
		}
	}

	opts := &argocd.ApplicationLogsOptions{
		PodName:      q.PodName,
		Container:    container,
		Previous:     q.Previous,
		SinceSeconds: sinceSeconds,
		TailLines:    tailLines,
		Filter:       q.Filter,
	}

	entries, err := c.handler.GetApplicationLogs(appName, workingEnv, opts)
	if err != nil {
		status := http.StatusInternalServerError
		var apiErr *argocd.ArgoCDAPIError
		if errors.As(err, &apiErr) {
			status = apiErr.StatusCode
		}
		ctx.JSON(status, gin.H{
			"error": gin.H{
				"message": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, entries)
}

func (c *InfrastructureController) RestartDeployment(ctx *gin.Context) {
	appName := ctx.Param("appName")
	workingEnv := getWorkingEnv(ctx)
	if workingEnv == "" {
		return // Error already handled in getWorkingEnv
	}

	if appName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "appName is required"})
		return
	}

	var request RestartDeploymentRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get user email from context if available, otherwise use default
	userEmail := "horizon-system"
	if email, exists := ctx.Get("userEmail"); exists {
		if emailStr, ok := email.(string); ok && emailStr != "" {
			userEmail = emailStr
		}
	}

	// Create workflow payload
	payload := map[string]interface{}{
		"appName":  appName,
		"isCanary": fmt.Sprintf("%v", request.IsCanary),
		"email":    userEmail,
	}

	// Start workflow and return workflowId
	workflowID, err := c.workflowHandler.StartRestartDeploymentWorkflow(payload, userEmail, workingEnv)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"workflowId": workflowID,
		"message":    "Restart deployment workflow started successfully",
	})
}

func (c *InfrastructureController) UpdateCPUThreshold(ctx *gin.Context) {
	appName := ctx.Param("appName")
	workingEnv := getWorkingEnv(ctx)
	if workingEnv == "" {
		return // Error already handled in getWorkingEnv
	}

	if appName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "appName is required"})
		return
	}

	var request UpdateThresholdRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get user email from context if available, otherwise use default
	userEmail := "horizon-system"
	if email, exists := ctx.Get("userEmail"); exists {
		if emailStr, ok := email.(string); ok && emailStr != "" {
			userEmail = emailStr
		}
	}

	// Create workflow payload
	payload := map[string]interface{}{
		"appName":      appName,
		"cpuThreshold": request.CPUThreshold,
		"email":        userEmail,
	}

	// Start workflow and return workflowId
	workflowID, err := c.workflowHandler.StartCPUThresholdUpdateWorkflow(payload, userEmail, workingEnv)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"workflowId": workflowID,
		"message":    "CPU threshold update workflow started successfully",
	})
}

func (c *InfrastructureController) UpdateGPUThreshold(ctx *gin.Context) {
	appName := ctx.Param("appName")
	workingEnv := getWorkingEnv(ctx)
	if workingEnv == "" {
		return // Error already handled in getWorkingEnv
	}

	if appName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "appName is required"})
		return
	}

	var request UpdateGPUThresholdRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get user email from context if available, otherwise use default
	userEmail := "horizon-system"
	if email, exists := ctx.Get("userEmail"); exists {
		if emailStr, ok := email.(string); ok && emailStr != "" {
			userEmail = emailStr
		}
	}

	// Create workflow payload
	payload := map[string]interface{}{
		"appName":      appName,
		"gpuThreshold": request.GPUThreshold,
		"email":        userEmail,
	}

	// Start workflow and return workflowId
	workflowID, err := c.workflowHandler.StartGPUThresholdUpdateWorkflow(payload, userEmail, workingEnv)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"workflowId": workflowID,
		"message":    "GPU threshold update workflow started successfully",
	})
}

func (c *InfrastructureController) UpdateSharedMemory(ctx *gin.Context) {
	appName := ctx.Param("appName")
	workingEnv := getWorkingEnv(ctx)
	if workingEnv == "" {
		return // Error already handled in getWorkingEnv
	}

	if appName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "appName is required"})
		return
	}

	var request UpdateSharedMemoryRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get user email from context if available, otherwise use default
	userEmail := "horizon-system"
	if email, exists := ctx.Get("userEmail"); exists {
		if emailStr, ok := email.(string); ok && emailStr != "" {
			userEmail = emailStr
		}
	}

	// Create workflow payload
	payload := map[string]interface{}{
		"appName": appName,
		"size":    request.Size,
		"email":   userEmail,
	}

	// Start workflow and return workflowId
	workflowID, err := c.workflowHandler.StartSharedMemoryUpdateWorkflow(payload, userEmail, workingEnv)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"workflowId": workflowID,
		"message":    "Shared memory update workflow started successfully",
	})
}

func (c *InfrastructureController) UpdatePodAnnotations(ctx *gin.Context) {
	appName := ctx.Param("appName")
	workingEnv := getWorkingEnv(ctx)
	if workingEnv == "" {
		return // Error already handled in getWorkingEnv
	}

	if appName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "appName is required"})
		return
	}

	var request UpdatePodAnnotationsRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate annotations
	if len(request.Annotations) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "annotations are required and cannot be empty"})
		return
	}

	// Get user email from context if available, otherwise use default
	userEmail := "horizon-system"
	if email, exists := ctx.Get("userEmail"); exists {
		if emailStr, ok := email.(string); ok && emailStr != "" {
			userEmail = emailStr
		}
	}

	// Create workflow payload
	payload := map[string]interface{}{
		"appName":     appName,
		"annotations": request.Annotations,
		"email":       userEmail,
	}

	// Start workflow and return workflowId
	workflowID, err := c.workflowHandler.StartPodAnnotationsUpdateWorkflow(payload, userEmail, workingEnv)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"workflowId": workflowID,
		"message":    "Pod annotations update workflow started successfully",
	})
}

func (c *InfrastructureController) UpdateAutoscalingTriggers(ctx *gin.Context) {
	appName := ctx.Param("appName")
	workingEnv := getWorkingEnv(ctx)
	if workingEnv == "" {
		return // Error already handled in getWorkingEnv
	}

	if appName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "appName is required"})
		return
	}

	var request UpdateAutoscalingTriggersRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate triggers
	if len(request.Triggers) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "triggers array is required and cannot be empty"})
		return
	}

	// Get user email from context if available, otherwise use default
	userEmail := "horizon-system"
	if email, exists := ctx.Get("userEmail"); exists {
		if emailStr, ok := email.(string); ok && emailStr != "" {
			userEmail = emailStr
		}
	}

	// Create workflow payload
	payload := map[string]interface{}{
		"appName":  appName,
		"triggers": request.Triggers,
		"email":    userEmail,
	}

	// Start workflow and return workflowId
	workflowID, err := c.workflowHandler.StartAutoscalingTriggersUpdateWorkflow(payload, userEmail, workingEnv)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"workflowId": workflowID,
		"message":    "Autoscaling triggers update workflow started successfully",
	})
}
