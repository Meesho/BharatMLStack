package predator

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	"github.com/Meesho/helix-clients/pkg/clients/predator"
)

const (
	errorType              = "error-type"
	predatorApiError       = "predator-api-error"
	predatorClientNotFound = "predator-client-not-found"
)

var (
	predatorClientMap  map[string]*predator.Client
	predatorMetricTags = []string{"ext-service:predator", "endpoint:/ModelInfer"}
)

func InitPredatorHandler(configs *configs.AppConfigs) {
	predatorClientMap = make(map[string]*predator.Client)
	predatorConfigs := make([]*predator.Config, 0)

	configStr := configs.Configs.ExternalServicePredator

	if len(configStr) > 0 && configStr[0] == '\'' && configStr[len(configStr)-1] == '\'' {
		configStr = configStr[1 : len(configStr)-1]
	}

	configStr = convertToValidJSON(configStr)

	err := json.Unmarshal([]byte(configStr), &predatorConfigs)
	if err != nil {
		logger.Error(fmt.Sprintf("Error unmarshaling predator configs: %v", err), err)
		return
	}

	for i, value := range predatorConfigs {
		client := predator.InitClient(i, value)
		predatorClientMap[value.Host] = &client
	}
}

func convertToValidJSON(configStr string) string {
	properties := []string{"Host", "Port", "Deadline", "PlainText", "CallerId", "CallerToken", "BatchSize"}

	result := configStr
	for _, prop := range properties {
		unquoted := prop + ":"
		quoted := "\"" + prop + "\":"
		result = strings.ReplaceAll(result, unquoted, quoted)
	}

	return result
}

func GetPredatorResponse(predatorReq *predator.PredatorRequest, endPoint string, endPoints []config.ModelEndpoint, errLoggingPercent int, compMetricTags []string) *predator.PredatorResponse {
	metricTags := append(compMetricTags, predatorMetricTags...)

	predatorEndPoint := ""
	if len(endPoints) > 0 {
		randomNum := rand.Intn(100) + 1
		cumulativePercentage := 0

		for _, endPoint := range endPoints {
			cumulativePercentage += endPoint.RoutingPercentage
			if randomNum <= cumulativePercentage {
				predatorEndPoint = endPoint.EndPoint
				break
			}
		}

		if predatorEndPoint == "" {
			predatorEndPoint = endPoint
		}
	} else {
		predatorEndPoint = endPoint
	}

	predatorClientPtr, ok := predatorClientMap[predatorEndPoint]

	if !ok || predatorClientPtr == nil {
		logger.Error(fmt.Sprintf("Predator client not found for endpoint: %s", endPoint), nil)
		metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, predatorClientNotFound))
		return nil
	}
	predatorClient := *predatorClientPtr
	t := time.Now()
	metrics.Count("inferflow.external.api.request.total", 1, metricTags)
	res, err := predatorClient.GetInferenceScore(predatorReq)
	metrics.Timing("inferflow.external.api.request.latency", time.Since(t), metricTags)
	if err != nil {
		logger.PercentError(fmt.Sprintf("Error getting predator response for component %v", compMetricTags), err, errLoggingPercent)
		metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, predatorApiError))
	} else if res == nil {
		logger.PercentError(fmt.Sprintf("Found nil/Empty predator response for component %v", compMetricTags), err, errLoggingPercent)
		metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, predatorApiError))
	}
	return res
}
