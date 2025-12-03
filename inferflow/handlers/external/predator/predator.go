package predator

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/clients/predator"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/etcd"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
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

	InferflowConfig := etcd.Instance().GetConfigInstance().(*config.ModelConfig)
	model_endpoints := make(map[string]bool, 0)
	i := 0
	for _, modelConfig := range InferflowConfig.ConfigMap {
		predatorConfigs := modelConfig.ComponentConfig.PredatorComponentConfig.Values()
		for _, predatorConfig := range predatorConfigs {
			predatorConfigMap := predatorConfig.(config.PredatorComponentConfig)
			deadline := configs.Configs.ExternalServicePredator_Deadline
			logger.Info(fmt.Sprintf("Deadline: %d", deadline))
			if predatorConfigMap.ModelEndpoint != "" && !model_endpoints[predatorConfigMap.ModelEndpoint] {
				model_endpoints[predatorConfigMap.ModelEndpoint] = true
				config := &predator.Config{
					Host:        predatorConfigMap.ModelEndpoint,
					Port:        configs.Configs.ExternalServicePredator_Port,
					PlainText:   configs.Configs.ExternalServicePredator_GrpcPlainText,
					CallerId:    configs.Configs.ExternalServicePredator_CallerId,
					CallerToken: configs.Configs.ExternalServicePredator_CallerToken,
					DeadLine:    deadline,
				}
				client := predator.InitClient(i, config)
				predatorClientMap[predatorConfigMap.ModelEndpoint] = &client
				i++
			}
			if len(predatorConfigMap.ModelEndPoints) != 0 {
				for _, modelEndPoint := range predatorConfigMap.ModelEndPoints {
					if !model_endpoints[modelEndPoint.EndPoint] {
						model_endpoints[modelEndPoint.EndPoint] = true
						config := &predator.Config{
							Host:        modelEndPoint.EndPoint,
							Port:        configs.Configs.ExternalServicePredator_Port,
							PlainText:   configs.Configs.ExternalServicePredator_GrpcPlainText,
							CallerId:    configs.Configs.ExternalServicePredator_CallerId,
							CallerToken: configs.Configs.ExternalServicePredator_CallerToken,
							DeadLine:    deadline,
						}
						client := predator.InitClient(i, config)
						predatorClientMap[modelEndPoint.EndPoint] = &client
						i++
					}
				}

			}
		}
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
