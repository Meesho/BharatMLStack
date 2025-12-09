package numerix

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/clients/numerix"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
)

const (
	errorType       = "error-type"
	numerixApiError = "numerix-api-error"
)

var (
	numerixClient     numerix.NumerixClient
	numerixMetricTags = []string{"ext-service:numerix_service", "endpoint:/NumerixService/Compute"}
)

func InitNumerixHandler(configs *configs.AppConfigs) {
	numerixClient = numerix.GetNumerixClient(1, getNumerixConfigBytes(configs))
	logger.Info("Numerix handler initialized")
}

func getNumerixConfigBytes(configs *configs.AppConfigs) []byte {
	numerixConfig := numerix.ClientConfig{
		Host:             configs.Configs.NumerixClientV1_Host,
		Port:             strconv.Itoa(configs.Configs.NumerixClientV1_Port),
		DeadlineExceedMS: configs.Configs.NumerixClientV1_Deadline,
		BatchSize:        configs.Configs.NumerixClientV1_BatchSize,
		PlainText:        configs.Configs.NumerixClientV1_GrpcPlainText,
		AuthToken:        configs.Configs.NumerixClientV1_AuthToken,
		CallerId:         configs.Configs.ApplicationName,
	}
	numerixConfigBytes, err := json.Marshal(numerixConfig)
	if err != nil {
		logger.Error("Error marshalling numerix config", err)
		return nil
	}
	return numerixConfigBytes
}

func GetNumerixResponse(numerixReq numerix.NumerixRequest, errLoggingPercent int, compMetricTags []string) *numerix.NumerixResponse {
	t := time.Now()
	numerixTags := append(compMetricTags, numerixMetricTags...)
	metrics.Count("inferflow.external.api.request.total", 1, numerixTags)
	numerixRes, err := numerixClient.RetrieveScore(&numerixReq)
	if err != nil {
		logger.PercentError(fmt.Sprintf("Error getting Numerix response for component %v", compMetricTags), err, errLoggingPercent)
		metrics.Count("inferflow.component.execution.error", 1, append(numerixTags, errorType, numerixApiError))
	} else if numerixRes == nil || numerixRes.ComputationScoreData.Data == nil {
		logger.PercentError(fmt.Sprintf("Found nil/Empty Numerix response for component %v", compMetricTags), nil, errLoggingPercent)
		metrics.Count("inferflow.component.execution.error", 1, append(numerixTags, errorType, numerixApiError))
	}
	metrics.Timing("inferflow.external.api.request.latency", time.Since(t), numerixTags)
	return numerixRes
}
