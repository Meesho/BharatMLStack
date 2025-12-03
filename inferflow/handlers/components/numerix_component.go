package components

import (
	"fmt"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/models"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/matrix"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/clients/numerix"
	extNumerix "github.com/Meesho/BharatMLStack/inferflow/handlers/external/numerix"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
)

type NumerixComponent struct {
	ComponentName string
}

func (iComponent *NumerixComponent) GetComponentName() string {
	return iComponent.ComponentName
}

func (iComponent *NumerixComponent) Run(request interface{}) {

	componentRequest, ok := request.(ComponentRequest)
	if ok {
		modelID := componentRequest.ModelId
		metricTags := []string{modelId, modelID, component, iComponent.GetComponentName()}
		t := time.Now()
		metrics.Count("inferflow.component.execution.total", 1, metricTags)
		matrixUtil := *componentRequest.ComponentData
		errLoggingPercent := componentRequest.ComponentConfig.ErrorLoggingPercent

		// get config for component
		iIConfig, ok := (*componentRequest.ComponentConfig).NumerixComponentConfig.Get(iComponent.GetComponentName())
		if !ok {
			logger.Error(fmt.Sprintf("Config not found for component %s ", iComponent.GetComponentName()), nil)
			metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, compConfigErr))
			return
		}
		iConfig, ok := iIConfig.(config.NumerixComponentConfig)
		if !ok {
			logger.Error(fmt.Sprintf("Error casting interface to config for component %s ", iComponent.GetComponentName()), nil)
			metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, compConfigErr))
			return
		}
		isValid := validateNumerixConfig(&iConfig)
		if !isValid {
			metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, compConfigErr))
			logger.Error(fmt.Sprintf("Invalid component request for model-id %s and component %s ", modelID, iComponent.GetComponentName()), nil)
			return
		}

		numerixComponentBuilder := &models.NumerixComponentBuilder{}
		initializeNumerixComponentBuilder(numerixComponentBuilder, &iConfig, &matrixUtil)

		numerixComponentBuilder.MatrixColumns = append(numerixComponentBuilder.MatrixColumns, iConfig.ComponentId)
		numerixComponentBuilder.Schema = append(numerixComponentBuilder.Schema, iConfig.ComponentId)
		for key, value := range iConfig.ScoreMapping {
			numerixComponentBuilder.Schema = append(numerixComponentBuilder.Schema, key)
			numerixComponentBuilder.MatrixColumns = append(numerixComponentBuilder.MatrixColumns, value)
		}

		matrixUtil.PopulateMatrixOfColumnSlice(numerixComponentBuilder)
		iComponent.populateScoreMap(modelID, iConfig, numerixComponentBuilder.Schema, numerixComponentBuilder, errLoggingPercent)
		// populate 2D matrix
		matrixUtil.PopulateByteData(iConfig.ScoreColumn, numerixComponentBuilder.Scores)
		metrics.Timing("inferflow.component.execution.latency", time.Duration(time.Since(t)), metricTags)
	}
}

func initializeNumerixComponentBuilder(numerixComponentBuilder *models.NumerixComponentBuilder, iConfig *config.NumerixComponentConfig, matrixUtil *matrix.ComponentMatrix) {
	numRow := len(matrixUtil.Rows)
	numCols := len(iConfig.ScoreMapping) + 1

	numerixComponentBuilder.Schema = make([]string, 0, numCols)
	numerixComponentBuilder.MatrixColumns = make([]string, 0, numCols)

	buffer := make([][]byte, numRow*numCols)
	numerixComponentBuilder.Matrix = make([][][]byte, numRow)
	for i := 0; i < numRow; i++ {
		start := i * numCols
		numerixComponentBuilder.Matrix[i] = buffer[start:(start + numCols)]
	}
}

func (iComponent *NumerixComponent) populateScoreMap(modelID string, config config.NumerixComponentConfig, schema []string, numerixComponentBuilder *models.NumerixComponentBuilder, errLoggingPercent int) {
	if len(numerixComponentBuilder.Matrix) > 0 {
		metricTags := []string{modelId, modelID, component, iComponent.GetComponentName()}
		numerixRequest := createNumerixRequest(config, schema, numerixComponentBuilder.Matrix)
		numerixResponse := extNumerix.GetNumerixResponse(numerixRequest, errLoggingPercent, metricTags)
		if numerixResponse != nil && numerixResponse.ComputationScoreData.Data != nil && len(numerixResponse.ComputationScoreData.Data) > 0 {
			numerixComponentBuilder.Scores = make([][]byte, len(numerixResponse.ComputationScoreData.Data))
			for i, data := range numerixResponse.ComputationScoreData.Data {
				if data != nil && len(data) > 1 && data[1] != nil {
					numerixComponentBuilder.Scores[i] = data[1]
				}
			}
		}
	}
}

func createNumerixRequest(config config.NumerixComponentConfig, schema []string, payload [][][]byte) numerix.NumerixRequest {
	return numerix.NumerixRequest{
		EntityScoreData: numerix.EntityScoreData{
			Schema:     schema,
			Data:       payload,
			StringData: nil,
			ComputeID:  config.ComputeId,
			DataType:   config.DataType,
		},
	}
}

func validateNumerixConfig(iConfig *config.NumerixComponentConfig) bool {

	if iConfig == nil ||
		iConfig.Component == "" ||
		iConfig.ComponentId == "" ||
		iConfig.ScoreMapping == nil ||
		iConfig.ScoreColumn == "" ||
		iConfig.ComputeId == "" {
		return false
	}
	return true
}
