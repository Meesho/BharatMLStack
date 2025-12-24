package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	"github.com/Meesho/BharatMLStack/horizon/internal/deployable/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/gin-gonic/gin"
)

type Config interface {
	GetMetaData(ctx *gin.Context)
	CreateDeployable(ctx *gin.Context)
	UpdateDeployable(ctx *gin.Context)
	GetDeployablesByService(ctx *gin.Context)
	RefreshDeployable(ctx *gin.Context)
	TuneThresholds(ctx *gin.Context)
}

var (
	deployable         Config
	deployableInitOnce sync.Once
)

type V1 struct {
	config handler.Config
}

func NewConfigController() Config {
	if deployable == nil {
		deployableInitOnce.Do(func() {
			config, err := handler.NewDeployable(1)
			if err != nil {
				panic(fmt.Sprintf("Failed to initialize deployable config: %v", err))
			}
			if config == nil {
				panic("Deployable config is nil after initialization")
			}
			deployable = &V1{
				config: config,
			}
		})
	}
	return deployable
}

func (d *V1) GetMetaData(ctx *gin.Context) {
	metaData, err := d.config.GetMetaData()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{constant.Error: "Failed to fetch metadata"})
		return
	}
	ctx.JSON(http.StatusOK, metaData)
}

func (d *V1) CreateDeployable(ctx *gin.Context) {
	var request handler.DeployableRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"data":  nil,
		})
		return
	}

	if err := d.config.CreateDeployable(&request); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Error registering deployable: %v", err),
			"data":  nil,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"error": nil,
		"data": gin.H{
			"message": "Deployable registered is in progress.",
		},
	})
}

func (d *V1) UpdateDeployable(ctx *gin.Context) {
	var request handler.DeployableRequest // Using same request structure as create
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"data":  nil,
		})
		return
	}

	if err := d.config.UpdateDeployable(&request); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Error updating deployable: %v", err),
			"data":  nil,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"error": nil,
		"data": gin.H{
			"message": "Deployable update is in progress.",
		},
	})
}

func (d *V1) GetDeployablesByService(ctx *gin.Context) {
	serviceName := ctx.Query("service_name")
	if serviceName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "service_name query parameter is required",
			"data":  nil,
		})
		return
	}

	deployables, err := d.config.GetDeployablesByService(serviceName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Error fetching deployables: %v", err),
			"data":  nil,
		})
		return
	}

	if len(deployables) == 0 {
		ctx.JSON(http.StatusOK, gin.H{
			"error": nil,
			"data":  []gin.H{},
		})
		return
	}

	// Use channels for concurrent processing
	type deployableResult struct {
		index int
		data  gin.H
		err   error
	}

	results := make(chan deployableResult, len(deployables))
	var wg sync.WaitGroup

	// Process each deployable concurrently
	for i, deployable := range deployables {
		wg.Add(1)
		go func(idx int, dep servicedeployableconfig.ServiceDeployableConfig) {
			defer wg.Done()

			// Parse config JSON
			var configMap map[string]interface{}
			if err := json.Unmarshal(dep.Config, &configMap); err != nil {
				results <- deployableResult{index: idx, err: fmt.Errorf("error parsing deployable config: %w", err)}
				return
			}

			// Create base response
			deployableResponse := gin.H{
				"id":                     dep.ID,
				"name":                   dep.Name,
				"host":                   dep.Host,
				"service":                dep.Service,
				"active":                 dep.Active,
				"created_by":             dep.CreatedBy,
				"updated_by":             dep.UpdatedBy,
				"created_at":             dep.CreatedAt,
				"updated_at":             dep.UpdatedAt,
				"monitoring_url":         dep.MonitoringUrl,
				"deployable_workflow_id": dep.DeployableWorkFlowId,
				"deployment_run_id":      dep.DeploymentRunID,
				"deployable_health":      dep.DeployableHealth,
				"workflow_status":        dep.WorkFlowStatus,
			}

			// Add config fields (excluding replica fields)
			for key, value := range configMap {
				if key != "min_replica" && key != "max_replica" {
					deployableResponse[key] = value
				}
			}

			// Get Ring Master config if needed
			if dep.DeployableWorkFlowId != "" && dep.DeploymentRunID != "" {
				ringMasterConfig := d.config.GetRingMasterConfig(dep.Name, dep.DeployableWorkFlowId, dep.DeploymentRunID)
				deployableResponse["min_replica"] = ringMasterConfig.MinReplica
				deployableResponse["max_replica"] = ringMasterConfig.MaxReplica
				deployableResponse["deployable_running_status"] = ringMasterConfig.RunningStatus == "true"
			}

			results <- deployableResult{index: idx, data: deployableResponse}
		}(i, deployable)
	}

	// Close results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and handle errors
	response := make([]gin.H, len(deployables))
	for result := range results {
		if result.err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": result.err.Error(),
				"data":  nil,
			})
			return
		}
		response[result.index] = result.data
	}

	ctx.JSON(http.StatusOK, gin.H{
		"error": nil,
		"data":  response,
	})
}

func (d *V1) RefreshDeployable(ctx *gin.Context) {
	appName := ctx.Query("app_name")
	serviceType := ctx.Query("service_type")

	if appName == "" || serviceType == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "app_name and service_type query parameters are required",
			"data":  nil,
		})
		return
	}

	deployable, err := d.config.RefreshDeployable(appName, serviceType)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Error refreshing deployable: %v", err),
			"data":  nil,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"error": nil,
		"data": gin.H{
			"deployable_running_status": deployable.DeployableRunningStatus,
			"deployable_health":         deployable.DeployableHealth,
			"workflow_status":           deployable.WorkFlowStatus,
		},
	})
}

func (d *V1) TuneThresholds(ctx *gin.Context) {
	var request handler.TuneThresholdsRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"data":  nil,
		})
		return
	}

	if err := d.config.TuneThresholds(&request); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Error updating deployable thresholds: %v", err),
			"data":  nil,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"error": nil,
		"data": gin.H{
			"message": "Deployable Thresholds Updated Started.",
		},
	})
}
