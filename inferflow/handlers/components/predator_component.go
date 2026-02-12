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
			metrics.Timing("modelproxy.component.execution.latency", time.Since(t), metricTags)
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
// For each slate it makes a **separate inference request**:
//  1. Read slate_target_indices from SlateData (comma-separated row indices into the target matrix).
//  2. Build a per-slate matrix view containing only that slate's target rows.
//  3. Run the standard predator flow (init builder → gather features → score → collect output).
//  4. Write the per-slate scores into the corresponding SlateData row.
func (pComponent *PredatorComponent) runSlate(
	req ComponentRequest,
	pConfig config.PredatorComponentConfig,
	targetMatrix *matrix.ComponentMatrix,
	errLoggingPercent int,
	metricTags []string,
) {
	slateMatrix := req.SlateData
	slateRows := slateMatrix.Rows
	numSlates := len(slateRows)
	if numSlates == 0 {
		return
	}

	indicesCol, ok := slateMatrix.StringColumnIndexMap[SlateTargetIndicesColumn]
	if !ok {
		logger.Error("slate_target_indices column not found in SlateData", nil)
		metrics.Count("modelproxy.component.execution.error", 1, append(metricTags, errorType, compConfigErr))
		return
	}

	// Pre-allocate score accumulators: one [][]byte per output score, length = numSlates
	totalScoreCount := 0
	for _, out := range pConfig.Outputs {
		totalScoreCount += len(out.ModelScores)
	}
	slateScoresByOutput := make([][][]byte, totalScoreCount)
	for i := range slateScoresByOutput {
		slateScoresByOutput[i] = make([][]byte, numSlates)
	}

	// Process each slate independently in parallel.
	var wg sync.WaitGroup
	for s, slateRow := range slateRows {
		wg.Add(1)
		go func(s int, slateRow matrix.Row) {
			defer wg.Done()
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error(fmt.Sprintf("panic recovered in slate predator goroutine for model-id %s slate %d: %v", req.ModelId, s, rec), nil)
					metrics.Count("modelproxy.component.execution.error", 1, append(metricTags, errorType, "slate-predator-goroutine-panic"))
				}
			}()

			// Parse target indices for this slate
			idxStr := slateRow.StringData[indicesCol.Index]
			parts := strings.Split(idxStr, ",")
			targetRows := make([]matrix.Row, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				idx, err := strconv.Atoi(p)
				if err != nil || idx < 0 || idx >= len(targetMatrix.Rows) {
					continue
				}
				targetRows = append(targetRows, targetMatrix.Rows[idx])
			}
			if len(targetRows) == 0 {
				return
			}

			// Per-slate matrix view (reuses target column maps, only this slate's target rows)
			perSlateMatrix := &matrix.ComponentMatrix{
				StringColumnIndexMap: targetMatrix.StringColumnIndexMap,
				ByteColumnIndexMap:   targetMatrix.ByteColumnIndexMap,
				Rows:                 targetRows,
			}

			// Standard predator flow on the per-slate matrix
			builder := &models.PredatorComponentBuilder{}
			initializePredatorComponentBuilder(builder, &pConfig, perSlateMatrix)

			t1 := time.Now()
			GetMatrixOfColumnSliceWithDataType(&pConfig, perSlateMatrix, builder, metricTags)
			metrics.Timing("modelproxy.datatype.conversion.execution.latency", time.Since(t1), metricTags)

			t2 := time.Now()
			pComponent.populateScores(req.ModelId, pConfig, builder, errLoggingPercent)
			metrics.Timing("modelproxy.component.predator.execution.latency", time.Since(t2), metricTags)

			// Collect per-slate scores (model returns one score set for this slate)
			for i := range slateScoresByOutput {
				if i < len(builder.Scores) && builder.Scores[i] != nil && len(builder.Scores[i]) > 0 {
					// Take the first output row as the slate-level score
					slateScoresByOutput[i][s] = builder.Scores[i][0]
				}
			}
		}(s, slateRow)
	}
	wg.Wait()

	// Write accumulated slate scores to SlateData
	counter := 0
	for _, out := range pConfig.Outputs {
		for _, scoreName := range out.ModelScores {
			slateMatrix.PopulateByteAndStringData(scoreName, slateScoresByOutput[counter])
			counter++
		}
	}
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
