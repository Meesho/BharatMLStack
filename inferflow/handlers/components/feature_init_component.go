//go:build !meesho

package components

import (
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/matrix"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
)

const (
	atTheRate     = "@"
	modelSuffix   = "_model"
	modelConfigId = "inferflow_config_id"
)

type FeatureInitComponent struct {
	ComponentName string
}

func (fiComponent *FeatureInitComponent) GetComponentName() string {
	return fiComponent.ComponentName
}

func (fiComponent *FeatureInitComponent) Run(request interface{}) {

	req, ok := request.(ComponentRequest)
	if !ok {
		return
	}

	metricTags := []string{modelId, req.ModelId, component, fiComponent.GetComponentName()}
	startTime := time.Now()
	metrics.Count("inferflow.component.execution.total", 1, metricTags)
	rowCount := len((*req.EntityIds)[0])
	stringColumnIndexMap := buildStringDataSchema(req)
	byteColumnIndexMap := buildByteDataSchema(req)
	componentMatrix := req.ComponentData
	componentMatrix.InitComponentMatrix(rowCount, stringColumnIndexMap, byteColumnIndexMap)
	populateStringData(componentMatrix, req)
	initSlateMatrix(req)

	metrics.Timing("inferflow.component.execution.latency", time.Since(startTime), metricTags)
}

func extractByteToStringSchema(stringColumnIndexMap map[string]matrix.Column, compConfig *config.ComponentConfig) {
	if compConfig.FeatureComponentConfig.Size() == 0 {
		return
	}
	idx := len(stringColumnIndexMap)
	for _, comp := range compConfig.FeatureComponentConfig.Values() {
		orionComp, ok := comp.(config.FeatureComponentConfig)
		if !ok {
			continue
		}
		for _, key := range orionComp.FSKeys {
			if _, exists := stringColumnIndexMap[key.Column]; !exists {
				stringColumnIndexMap[key.Column] = matrix.Column{
					Name:     key.Column,
					DataType: DataTypeString,
					Index:    idx,
				}
				idx++
			}
		}
	}
}

func buildStringDataSchema(req ComponentRequest) map[string]matrix.Column {
	stringColumnIndexMap := make(map[string]matrix.Column)
	extractEntityColumns(stringColumnIndexMap, req)
	extractFeatureColumns(stringColumnIndexMap, req)
	extractFsCompositeColumn(stringColumnIndexMap, req.ComponentConfig)
	extractByteToStringSchema(stringColumnIndexMap, req.ComponentConfig)
	extractCalibrationColumns(stringColumnIndexMap, req.ComponentConfig)
	return stringColumnIndexMap
}

func buildByteDataSchema(req ComponentRequest) map[string]matrix.Column {
	compConfig := req.ComponentConfig
	byteColumnIndexMap := make(map[string]matrix.Column)
	extractFsColumns(byteColumnIndexMap, compConfig)
	extractPredatorColumns(byteColumnIndexMap, compConfig)
	extractNumerixColumns(byteColumnIndexMap, compConfig)

	return byteColumnIndexMap
}

func populateStringData(matrix *matrix.ComponentMatrix, req ComponentRequest) {
	// Populate entity ID columns
	if req.EntityIds != nil && req.Entities != nil {
		for i := range *req.EntityIds {
			entityName := (*req.Entities)[i]
			entityIds := (*req.EntityIds)[i]

			if len(entityIds) == 1 {
				matrix.PopulateStringDataFromSingleValue(entityName, entityIds[0])
			} else {
				matrix.PopulateStringData(entityName, entityIds)
			}
		}
	}

	// Populate default feature columns
	rowCount := 0
	if req.EntityIds != nil && len(*req.EntityIds) > 0 {
		rowCount = len((*req.EntityIds)[0])
	}

	if req.Features != nil {
		for name, values := range *req.Features {
			if len(values) == rowCount {
				matrix.PopulateStringData(name, values)
			}
		}
	}

	// Populate calibration config if present
	if req.ComponentConfig.PredatorComponentConfig.Size() > 0 {
		calibration := false
		pCompMap := req.ComponentConfig.PredatorComponentConfig
		for _, iPredatorComp := range pCompMap.Values() {
			predatorComp, ok := iPredatorComp.(config.PredatorComponentConfig)
			if ok {
				if predatorComp.Calibration != "" {
					matrix.PopulateStringDataFromSingleValue(predatorComp.Calibration+modelSuffix, predatorComp.ModelName)
					calibration = true
				}
			}
		}
		if calibration {
			matrix.PopulateStringDataFromSingleValue(modelConfigId, req.ModelId)
		}
	}
}

func extractFsColumns(byteColumnIndexMap map[string]matrix.Column, compConfig *config.ComponentConfig) {
	if compConfig.FeatureComponentConfig.Size() == 0 {
		return
	}

	index := 0
	var builder strings.Builder

	for _, comp := range compConfig.FeatureComponentConfig.Values() {
		orionComp, ok := comp.(config.FeatureComponentConfig)
		if !ok {
			continue
		}
		label := orionComp.FSRequest.Label
		for _, group := range orionComp.FSRequest.FeatureGroups {
			for _, feature := range group.Features {
				// Efficient string building
				builder.Reset()
				builder.WriteString(orionComp.ColNamePrefix)
				builder.WriteString(label)
				builder.WriteByte(':')
				builder.WriteString(group.Label)
				builder.WriteByte(':')
				builder.WriteString(feature)

				col := builder.String()

				// Check for '@' to extract data type info
				if idx := strings.LastIndex(col, atTheRate); idx != -1 {
					name := col[:idx]
					byteColumnIndexMap[name] = matrix.Column{
						Name:     name,
						DataType: col[idx+1:],
						Index:    index,
					}
				} else {
					byteColumnIndexMap[col] = matrix.Column{
						Name:     col,
						DataType: group.DataType,
						Index:    index,
					}
				}
				index++
			}
		}
	}
}

func extractFsCompositeColumn(stringColumnIndexMap map[string]matrix.Column, compConfig *config.ComponentConfig) {
	if compConfig.FeatureComponentConfig.Size() == 0 {
		return
	}

	idx := len(stringColumnIndexMap)
	for _, comp := range compConfig.FeatureComponentConfig.Values() {
		orionComp, ok := comp.(config.FeatureComponentConfig)
		if !ok {
			continue
		}
		if orionComp.CompositeId {
			stringColumnIndexMap[orionComp.ComponentId] = matrix.Column{Name: orionComp.ComponentId, DataType: DataTypeString, Index: idx}
			idx++
		}
	}
}

func extactPredatorOutputDataType(output *config.ModelOutput, dim []int) string {
	if len(dim) == 1 && dim[0] == 1 {
		return output.DataType
	} else {
		return "DataType" + output.DataType + "Vector"
	}
}

func extractCalibrationColumns(stringColumnIndexMap map[string]matrix.Column, compConfig *config.ComponentConfig) {
	if compConfig.PredatorComponentConfig.Size() == 0 {
		return
	}

	calibration := false
	index := len(stringColumnIndexMap)
	for _, comp := range compConfig.PredatorComponentConfig.Values() {
		predatorComp, ok := comp.(config.PredatorComponentConfig)
		if !ok {
			continue
		}
		if predatorComp.Calibration != "" {
			if _, exists := stringColumnIndexMap[predatorComp.Calibration+modelSuffix]; exists {
				continue
			}
			stringColumnIndexMap[predatorComp.Calibration+modelSuffix] = matrix.Column{
				Name:     predatorComp.Calibration + modelSuffix,
				DataType: DataTypeString,
				Index:    index,
			}
			index++
			calibration = true
		}
	}
	if calibration {
		if _, exists := stringColumnIndexMap[modelConfigId]; exists {
			return
		}
		stringColumnIndexMap[modelConfigId] = matrix.Column{
			Name:     modelConfigId,
			DataType: DataTypeString,
			Index:    index,
		}
		index++
	}
}

func extractPredatorColumns(byteColumnIndexMap map[string]matrix.Column, compConfig *config.ComponentConfig) {
	if compConfig.PredatorComponentConfig.Size() == 0 {
		return
	}
	index := len(byteColumnIndexMap)
	for _, comp := range compConfig.PredatorComponentConfig.Values() {
		predatorComp, ok := comp.(config.PredatorComponentConfig)
		if !ok {
			continue
		}
		if predatorComp.SlateComponent {
			continue
		}
		for _, output := range predatorComp.Outputs {
			for scoreIdx, modelScore := range output.ModelScores {
				dataType := output.DataType
				if len(output.ModelScoresDims) > scoreIdx && len(output.ModelScoresDims[scoreIdx]) > 0 {
					dataType = extactPredatorOutputDataType(&output, output.ModelScoresDims[scoreIdx])
				}
				byteColumnIndexMap[modelScore] = matrix.Column{
					Name:     modelScore,
					DataType: dataType,
					Index:    index,
				}
				index++
			}
		}
	}
}

func extractNumerixColumns(byteColumnIndexMap map[string]matrix.Column, compConfig *config.ComponentConfig) {
	if compConfig.NumerixComponentConfig.Size() == 0 {
		return
	}
	index := len(byteColumnIndexMap)
	for _, comp := range compConfig.NumerixComponentConfig.Values() {
		numerixComp, ok := comp.(config.NumerixComponentConfig)
		if ok {

			if numerixComp.SlateComponent {
				continue
			}
			byteColumnIndexMap[numerixComp.ScoreColumn] = matrix.Column{
				Name:     numerixComp.ScoreColumn,
				DataType: numerixComp.DataType,
				Index:    index,
			}
			index++
		}
	}
}

func extractFeatureColumns(stringColumnIndexMap map[string]matrix.Column, req ComponentRequest) {
	if req.Features != nil {
		idx := len(stringColumnIndexMap)
		for name := range *req.Features {
			stringColumnIndexMap[name] = matrix.Column{Name: name, DataType: DataTypeString, Index: idx}
			idx++
		}
	}
}

func extractEntityColumns(stringColumnIndexMap map[string]matrix.Column, req ComponentRequest) {
	if req.Entities == nil {
		return
	}
	for idx, entity := range *req.Entities {
		stringColumnIndexMap[entity] = matrix.Column{Name: entity, DataType: DataTypeString, Index: idx}
	}
}

func initSlateMatrix(req ComponentRequest) {
	if req.SlateData == nil {
		return
	}
	slateRowCount := len(req.SlateData.Rows)
	if slateRowCount == 0 {
		return
	}

	// Preserve adapter-populated slate string features before matrix re-init.
	// InitComponentMatrix reallocates rows, so we need to restore these columns.
	preservedSlateStrings := make(map[string][]string, len(req.SlateData.StringColumnIndexMap))
	for colName, col := range req.SlateData.StringColumnIndexMap {
		values := make([]string, slateRowCount)
		for i := 0; i < slateRowCount && i < len(req.SlateData.Rows); i++ {
			if col.Index < len(req.SlateData.Rows[i].StringData) {
				values[i] = req.SlateData.Rows[i].StringData[col.Index]
			}
		}
		preservedSlateStrings[colName] = values
	}
	// Preserve adapter-populated slate byte features as well.
	preservedSlateBytes := make(map[string][][]byte, len(req.SlateData.ByteColumnIndexMap))
	for colName, col := range req.SlateData.ByteColumnIndexMap {
		values := make([][]byte, slateRowCount)
		for i := 0; i < slateRowCount && i < len(req.SlateData.Rows); i++ {
			if col.Index < len(req.SlateData.Rows[i].ByteData) {
				values[i] = req.SlateData.Rows[i].ByteData[col.Index]
			}
		}
		preservedSlateBytes[colName] = values
	}

	slateByteColumns := buildSlateByteDataSchema(req.ComponentConfig, req.SlateData)
	slateStringColumns := buildSlateStringDataSchema(req.SlateData)

	req.SlateData.InitComponentMatrix(slateRowCount, slateStringColumns, slateByteColumns)

	// Restore adapter-populated slate string features (including slate_target_indices and
	// any additional slate features passed in the request).
	for colName, values := range preservedSlateStrings {
		req.SlateData.PopulateStringData(colName, values)
	}
	for colName, values := range preservedSlateBytes {
		req.SlateData.PopulateByteData(colName, values)
	}
}

// buildSlateByteDataSchema creates byte columns for slate-level predator and iris outputs.
func buildSlateByteDataSchema(compConfig *config.ComponentConfig, slateData *matrix.ComponentMatrix) map[string]matrix.Column {
	byteColumnIndexMap := make(map[string]matrix.Column)
	index := 0
	// Preserve adapter-provided slate byte feature columns first.
	if slateData != nil {
		for colName, col := range slateData.ByteColumnIndexMap {
			byteColumnIndexMap[colName] = matrix.Column{
				Name:     colName,
				DataType: col.DataType,
				Index:    index,
			}
			index++
		}
	}

	// Slate predator output columns
	for _, comp := range compConfig.PredatorComponentConfig.Values() {
		predatorComp, ok := comp.(config.PredatorComponentConfig)
		if !ok || !predatorComp.SlateComponent {
			continue
		}
		for _, output := range predatorComp.Outputs {
			for scoreIdx, modelScore := range output.ModelScores {
				dataType := output.DataType
				if len(output.ModelScoresDims) > scoreIdx && len(output.ModelScoresDims[scoreIdx]) > 0 {
					dataType = extactPredatorOutputDataType(&output, output.ModelScoresDims[scoreIdx])
				}
				if _, exists := byteColumnIndexMap[modelScore]; exists {
					continue
				}
				byteColumnIndexMap[modelScore] = matrix.Column{
					Name:     modelScore,
					DataType: dataType,
					Index:    index,
				}
				index++
			}
		}
	}

	// Slate numerix output columns
	for _, comp := range compConfig.NumerixComponentConfig.Values() {
		irisComp, ok := comp.(config.NumerixComponentConfig)
		if !ok || !irisComp.SlateComponent {
			continue
		}
		if _, exists := byteColumnIndexMap[irisComp.ScoreColumn]; exists {
			continue
		}
		byteColumnIndexMap[irisComp.ScoreColumn] = matrix.Column{
			Name:     irisComp.ScoreColumn,
			DataType: irisComp.DataType,
			Index:    index,
		}
		index++
	}

	return byteColumnIndexMap
}

// buildSlateStringDataSchema creates string columns for SlateData.
// It always includes slate_target_indices and preserves any adapter-provided slate feature columns.
func buildSlateStringDataSchema(slateData *matrix.ComponentMatrix) map[string]matrix.Column {
	stringColumnIndexMap := make(map[string]matrix.Column, len(slateData.StringColumnIndexMap)+1)
	stringColumnIndexMap[SlateTargetIndicesColumn] = matrix.Column{
		Name:     SlateTargetIndicesColumn,
		DataType: DataTypeString,
		Index:    0,
	}
	index := 1
	for colName := range slateData.StringColumnIndexMap {
		if colName == SlateTargetIndicesColumn {
			continue
		}
		stringColumnIndexMap[colName] = matrix.Column{
			Name:     colName,
			DataType: DataTypeString,
			Index:    index,
		}
		index++
	}
	return stringColumnIndexMap
}
