package components

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/clients/predator"
	extPredator "github.com/Meesho/BharatMLStack/inferflow/handlers/external/predator"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/models"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/datatypeconverter/typeconverter"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/matrix"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/utils"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
)

const (
	DataTypePrefix = "DataType"
	DataTypeBytes  = "BYTES"
	dataTypeString = "DataTypeString"
)

type PredatorComponent struct {
	ComponentName string
}

func (pComponent *PredatorComponent) GetComponentName() string {
	return pComponent.ComponentName
}

func (pComponent *PredatorComponent) Run(request interface{}) {

	componentRequest, ok := request.(ComponentRequest)
	if ok {
		modelID := componentRequest.ModelId
		metricTags := []string{modelId, modelID, component, pComponent.GetComponentName()}
		t := time.Now()
		metrics.Count("inferflow.component.execution.total", 1, metricTags)
		matrixUtil := *componentRequest.ComponentData
		errLoggingPercent := componentRequest.ComponentConfig.ErrorLoggingPercent

		// get config for component
		iPConfig, ok := (*componentRequest.ComponentConfig).PredatorComponentConfig.Get(pComponent.GetComponentName())
		if !ok {
			logger.Error(fmt.Sprintf("Config not found for component %s ", pComponent.GetComponentName()), nil)
			metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, compConfigErr))
			return
		}
		pConfig, ok := iPConfig.(config.PredatorComponentConfig)
		if !ok {
			logger.Error(fmt.Sprintf("Error casting interface to config for component %s ", pComponent.GetComponentName()), nil)
			metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, compConfigErr))
			return
		}
		isValid := validatePredatorConfig(&pConfig)
		if !isValid {
			metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, compConfigErr))
			logger.Error(fmt.Sprintf("Invalid component request for model-id %s and component %s ", modelID, pComponent.GetComponentName()), nil)
			return
		}

		if pConfig.SlateComponent && componentRequest.SlateData != nil {
			pComponent.runSlate(componentRequest, pConfig, &matrixUtil, errLoggingPercent, metricTags)
			metrics.Timing("inferflow.component.execution.latency", time.Since(t), metricTags)
			return
		}

		// get payload for model
		predatorComponentBuilder := &models.PredatorComponentBuilder{}
		initializePredatorComponentBuilder(predatorComponentBuilder, &pConfig, &matrixUtil)
		t1 := time.Now()
		GetMatrixOfColumnSliceWithDataType(&pConfig, &matrixUtil, predatorComponentBuilder, metricTags)
		metrics.Timing("inferflow.datatype.conversion.execution.latency", time.Since(t1), metricTags)
		t2 := time.Now()
		pComponent.populateScores(modelID, pConfig, predatorComponentBuilder, errLoggingPercent)
		metrics.Timing("inferflow.component.predator.execution.latency", time.Since(t2), metricTags)
		counter := 0
		for _, out := range pConfig.Outputs {
			for _, scoreName := range out.ModelScores {
				matrixUtil.PopulateByteData(scoreName, predatorComponentBuilder.Scores[counter])
				counter++
			}
		}
		metrics.Timing("inferflow.component.execution.latency", time.Since(t), metricTags)
	}
}

// runSlate handles the slate-aware predator path.
//
// For each slate:
//  1. Parse slate_target_indices → pick target rows from the filled target matrix.
//  2. Build a per-slate matrix view with those rows.
//  3. Same predator flow: init builder → gather features → call predator.
//  4. Model returns 1 score for N inputs → write that score directly
//     into SlateData row [s], score column.
func (pComponent *PredatorComponent) runSlate(
	req ComponentRequest,
	pConfig config.PredatorComponentConfig,
	targetMatrix *matrix.ComponentMatrix,
	errLoggingPercent int,
	metricTags []string,
) {
	slateMatrix := req.SlateData
	numSlates := len(slateMatrix.Rows)
	if numSlates == 0 {
		return
	}

	indicesCol, ok := slateMatrix.StringColumnIndexMap[SlateTargetIndicesColumn]
	if !ok {
		logger.Error(fmt.Sprintf("slate_target_indices column not found in SlateData for component %s", pComponent.GetComponentName()), nil)
		metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, compConfigErr))
		return
	}

	// --- Process each slate (parallel) ---
	var wg sync.WaitGroup
	for s := 0; s < numSlates; s++ {
		wg.Add(1)
		go func(s int) {
			defer wg.Done()
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error(fmt.Sprintf("panic in slate %d for model-id %s component %s: %v",
						s, req.ModelId, pComponent.GetComponentName(), rec), nil)
				}
			}()

			// 1. Parse target indices for this slate
			targetRows := parseSlateTargetRows(slateMatrix.Rows[s], indicesCol.Index, targetMatrix)
			if len(targetRows) == 0 {
				return
			}

			// 2. Build per-slate matrix view (same columns, subset of rows)
			perSlateMatrix := &matrix.ComponentMatrix{
				StringColumnIndexMap: targetMatrix.StringColumnIndexMap,
				ByteColumnIndexMap:   targetMatrix.ByteColumnIndexMap,
				Rows:                 targetRows,
			}

			// 3. Same predator flow: init → gather features → call predator
			builder := &models.PredatorComponentBuilder{}
			initializePredatorComponentBuilder(builder, &pConfig, perSlateMatrix)
			GetMatrixOfColumnSliceWithDataType(&pConfig, perSlateMatrix, builder, metricTags)
			pComponent.populateScores(req.ModelId, pConfig, builder, errLoggingPercent)

			// 4. Write the score directly into SlateData row [s], for each score column.
			//    Model returns 1 score (builder.Scores[i][0]) for this slate.
			counter := 0

			for _, out := range pConfig.Outputs {
				for _, scoreName := range out.ModelScores {
					if counter < len(builder.Scores) && len(builder.Scores[counter]) > 0 {
						if col, ok := slateMatrix.ByteColumnIndexMap[scoreName]; ok {
							slateMatrix.Rows[s].ByteData[col.Index] = builder.Scores[counter][0]
						}
					}
					counter++
				}
			}
		}(s)
	}
	wg.Wait()
}

// parseSlateTargetRows reads the comma-separated target indices from a slate row
// and returns the corresponding rows from the target matrix.
func parseSlateTargetRows(slateRow matrix.Row, indicesColIdx int, targetMatrix *matrix.ComponentMatrix) []matrix.Row {
	idxStr := slateRow.StringData[indicesColIdx]
	parts := strings.Split(idxStr, ",")
	rows := make([]matrix.Row, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		idx, err := strconv.Atoi(p)
		if err != nil || idx < 0 || idx >= len(targetMatrix.Rows) {
			continue
		}
		rows = append(rows, targetMatrix.Rows[idx])
	}
	return rows
}

func GetMatrixOfColumnSliceWithDataType(
	pConfig *config.PredatorComponentConfig,
	matrixUtil *matrix.ComponentMatrix,
	predatorComponentBuilder *models.PredatorComponentBuilder,
	metricTags []string,
) {
	var wg sync.WaitGroup
	var byteToByteConversionCounter int64
	var cacheMutex sync.RWMutex

	for inputIdx, input := range pConfig.Inputs {
		for i, row := range matrixUtil.Rows {
			wg.Add(1)
			go func(inputIdx int, input config.ModelInput, i int, row matrix.Row) {
				defer wg.Done()
				localByteToByteConversionCounter := 0

				for j, col := range input.Features {
					var err error
					byteCol, hasByteData := matrixUtil.ByteColumnIndexMap[col]
					stringCol, hasStringData := matrixUtil.StringColumnIndexMap[col]

					switch {
					case hasByteData && row.ByteData[byteCol.Index] != nil:
						switch {
						case "DataType"+input.DataType == byteCol.DataType:
							predatorComponentBuilder.FeaturePayloads[inputIdx][i][j] = row.ByteData[byteCol.Index]
						case input.DataType == DataTypeBytes:
							if byteCol.DataType == dataTypeString {
								predatorComponentBuilder.FeaturePayloads[inputIdx][i][j] = row.ByteData[byteCol.Index]
							} else {
								localByteToByteConversionCounter++
								if valStr, convertErr := typeconverter.BytesToString(row.ByteData[byteCol.Index], byteCol.DataType); convertErr != nil {
								} else if convertedVal, convertErr := typeconverter.StringToBytes(valStr, input.DataType); convertErr != nil {
								} else {
									predatorComponentBuilder.FeaturePayloads[inputIdx][i][j] = convertedVal
								}
							}
						default:
							localByteToByteConversionCounter++
							isVector := strings.Contains(byteCol.DataType, "Vector")
							if isVector {
								inputDataType := strings.Replace(byteCol.DataType, "Vector", "", 1)
								outputDataType := "DataType" + input.DataType

								bytesData, convertErr := typeconverter.ConvertVectorBytesToBytesLessGcOverhead(
									row.ByteData[byteCol.Index],
									inputDataType,
									outputDataType,
								)
								if convertErr != nil {
								} else {
									predatorComponentBuilder.FeaturePayloads[inputIdx][i][j] = bytesData
								}
							} else {
								predatorComponentBuilder.FeaturePayloads[inputIdx][i][j], err = typeconverter.ConvertBytesToBytes(
									row.ByteData[byteCol.Index],
									byteCol.DataType,
									input.DataType,
								)
								if err != nil {
								}
							}
						}
					case hasStringData:
						stringVal := row.StringData[stringCol.Index]

						cacheMutex.RLock()
						cached := predatorComponentBuilder.StringToByteCache[stringVal]
						cacheMutex.RUnlock()
						var converted []byte

						if cached != nil {
							predatorComponentBuilder.FeaturePayloads[inputIdx][i][j] = cached
						} else {
							switch input.DataType {
							case DataTypeBytes:
								converted, err = typeconverter.StringToBytes(stringVal, input.DataType)
							default:
								isVector := false
								if input.Shape[0] > 1 && len(input.Features) == 1 {
									isVector = true
								}
								predatorInputDataType := input.DataType
								if !strings.HasPrefix(predatorInputDataType, "DataType") {
									predatorInputDataType = "DataType" + predatorInputDataType
								}
								if isVector {
									if !strings.HasSuffix(predatorInputDataType, "Vector") {
										predatorInputDataType = predatorInputDataType + "Vector"
									}
								}
								converted, err = typeconverter.StringToBytes(stringVal, predatorInputDataType)
							}
							if err != nil {
							} else {
								cacheMutex.Lock()
								predatorComponentBuilder.StringToByteCache[stringVal] = converted
								cacheMutex.Unlock()

								predatorComponentBuilder.FeaturePayloads[inputIdx][i][j] = converted
							}
						}
					}

					// Set default if nothing set
					if len(predatorComponentBuilder.FeaturePayloads[inputIdx][i][j]) == 0 {
						if hasByteData {
							predatorComponentBuilder.FeaturePayloads[inputIdx][i][j] = utils.GetDefaultValuesInBytesForVector(input.DataType, byteCol.DataType, input.Shape)
						} else {
							predatorComponentBuilder.FeaturePayloads[inputIdx][i][j] = utils.GetDefaultValuesInBytesForVector(input.DataType, stringCol.DataType, input.Shape)
						}
					}
				}

				// Thread-safe increment
				atomic.AddInt64(&byteToByteConversionCounter, int64(localByteToByteConversionCounter))
			}(inputIdx, input, i, row)
		}
	}

	wg.Wait()

	metrics.Count("inferflow.matrix.byte.to.byte.conversion.count", atomic.LoadInt64(&byteToByteConversionCounter), metricTags)
}

func initializePredatorComponentBuilder(predatorComponentBuilder *models.PredatorComponentBuilder, config *config.PredatorComponentConfig, matrixUtil *matrix.ComponentMatrix) {
	rowsCount := len(matrixUtil.Rows)
	inputCount := len(config.Inputs)

	thirdDimensionLength := 0
	for _, input := range config.Inputs {
		for range matrixUtil.Rows {
			thirdDimensionLength += len(input.Features)
		}
	}

	predatorComponentBuilder.Inputs = make([]predator.Input, len(config.Inputs))
	predatorComponentBuilder.Outputs = make([]predator.Output, len(config.Outputs))
	thirdDimensionBuffer := make([][]byte, thirdDimensionLength)
	predatorComponentBuilder.StringToByteCache = make(map[string][]byte, len(matrixUtil.StringColumnIndexMap)*len(matrixUtil.Rows))

	secondDimensionBuffer := make([][][]byte, inputCount*rowsCount)

	for _, modelOutput := range config.Outputs {
		predatorComponentBuilder.ModelScoreCount += len(modelOutput.ModelScores)
	}

	predatorComponentBuilder.Scores = make([][][]byte, predatorComponentBuilder.ModelScoreCount)
	predatorComponentBuilder.FeaturePayloads = make([][][][]byte, inputCount)

	thirdIndex := 0
	for i, input := range config.Inputs {
		secondDimensionForInput := secondDimensionBuffer[i*rowsCount : (i+1)*rowsCount]

		for j := 0; j < rowsCount; j++ {
			featureCount := len(input.Features)
			secondDimensionForInput[j] = thirdDimensionBuffer[thirdIndex : thirdIndex+featureCount]
			thirdIndex += featureCount
		}

		predatorComponentBuilder.FeaturePayloads[i] = secondDimensionForInput
	}
}

func (pComponent *PredatorComponent) populateScores(modelID string, config config.PredatorComponentConfig, predatorComponentBuilder *models.PredatorComponentBuilder, errLoggingPercent int) {
	if len(predatorComponentBuilder.FeaturePayloads) > 0 {
		metricTags := []string{modelId, modelID, component, pComponent.GetComponentName()}
		predatorRequest := createPredatorRequest(config, predatorComponentBuilder)
		predatorResponse := extPredator.GetPredatorResponse(predatorRequest, config.ModelEndpoint, config.ModelEndPoints, errLoggingPercent, metricTags)
		if predatorResponse != nil && validatePredatorResponse(metricTags, predatorResponse, predatorComponentBuilder) {
			for i, out := range predatorResponse.Outputs {
				predatorComponentBuilder.Scores[i] = out.Data
			}
		}
	}
}

func createPredatorRequest(config config.PredatorComponentConfig, predatorComponentBuilder *models.PredatorComponentBuilder) *predator.PredatorRequest {

	for i, in := range config.Inputs {
		predatorComponentBuilder.Inputs[i] = predator.Input{
			Name:     in.Name,
			DataType: in.DataType,
			Dims:     in.Shape,
			Data:     predatorComponentBuilder.FeaturePayloads[i],
		}
	}

	for i, out := range config.Outputs {
		predatorComponentBuilder.Outputs[i] = predator.Output{
			Name:        out.Name,
			ModelScores: out.ModelScores,
			Dims:        out.ModelScoresDims,
		}
	}

	return &predator.PredatorRequest{
		ModelName:    config.ModelName,
		ModelVersion: "1",
		Inputs:       predatorComponentBuilder.Inputs,
		Outputs:      predatorComponentBuilder.Outputs,
		Deadline:     int64(config.Deadline),
		BatchSize:    config.BatchSize,
	}
}

func validatePredatorResponse(metricTags []string, predatorResponse *predator.PredatorResponse, predatorComponentBuilder *models.PredatorComponentBuilder) bool {
	predatorResponseCount := len(predatorResponse.Outputs)
	if predatorResponseCount != predatorComponentBuilder.ModelScoreCount {
		metrics.Count("inferflow.component.execution.error", int64(predatorComponentBuilder.ModelScoreCount-predatorResponseCount), append(metricTags, errorType, compConfigErr))
		return false
	}

	return true
}

func validatePredatorConfig(rConfig *config.PredatorComponentConfig) bool {
	if rConfig == nil ||
		rConfig.ComponentId == "" ||
		len(rConfig.Inputs) == 0 ||
		len(rConfig.Outputs) == 0 ||
		rConfig.ModelName == "" {
		return false
	}

	if len(rConfig.ModelEndPoints) == 0 && rConfig.ModelEndpoint == "" {
		return false
	}

	if len(rConfig.ModelEndPoints) > 0 {
		totalPercentage := 0
		for _, endPoint := range rConfig.ModelEndPoints {
			if endPoint.EndPoint == "" {
				return false
			}
			totalPercentage += endPoint.RoutingPercentage
		}
		if totalPercentage != 100 && totalPercentage != 0 {
			return false
		}
	}

	for _, output := range rConfig.Outputs {
		if len(output.ModelScoresDims) != len(output.ModelScores) {
			return false
		}
	}
	return true
}
