package inferflow

import (
	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/datatypeconverter/typeconverter"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/matrix"
	pb "github.com/Meesho/BharatMLStack/inferflow/server/grpc/predict"
)

// buildPointWiseResponse extracts scores from the ComponentMatrix into a PointWiseResponse.
// Each matrix row maps to one TargetScore. Output columns are determined by ResponseConfig.
func buildPointWiseResponse(componentMatrix *matrix.ComponentMatrix, conf *config.Config) *pb.PointWiseResponse {
	outputColumns := conf.ResponseConfig.Features
	outputSchema := buildOutputSchema(outputColumns, componentMatrix)

	scores := make([]*pb.TargetScore, len(componentMatrix.Rows))
	for i, row := range componentMatrix.Rows {
		scores[i] = &pb.TargetScore{
			OutputValues: extractRowOutputBytes(row, outputColumns, outputSchema, componentMatrix),
		}
	}

	return &pb.PointWiseResponse{
		TargetOutputSchema: outputSchema,
		TargetScores:       scores,
	}
}

// buildPairWiseResponse extracts target scores from ComponentData and pair scores from SlateData.
// A pair is a slate of size 2, so pair-level outputs come from SlateData (slate component outputs)
// and target-level outputs come from the main ComponentMatrix.
func buildPairWiseResponse(componentMatrix *matrix.ComponentMatrix, slateData *matrix.ComponentMatrix, conf *config.Config) *pb.PairWiseResponse {
	resp := &pb.PairWiseResponse{}

	// Target-level scores from the main matrix (one per target row)
	targetOutputColumns := conf.ResponseConfig.Features
	if len(targetOutputColumns) > 0 && len(componentMatrix.Rows) > 0 {
		targetOutputSchema := buildOutputSchema(targetOutputColumns, componentMatrix)
		targetScores := make([]*pb.TargetScore, len(componentMatrix.Rows))
		for i, row := range componentMatrix.Rows {
			targetScores[i] = &pb.TargetScore{
				OutputValues: extractRowOutputBytes(row, targetOutputColumns, targetOutputSchema, componentMatrix),
			}
		}
		resp.TargetScores = targetScores
		resp.TargetOutputSchema = targetOutputSchema
	}

	// Pair-level scores from SlateData (one per pair/slate row)
	if slateData != nil && len(slateData.Rows) > 0 {
		pairOutputColumns := slateOutputColumnNames(conf)
		if len(pairOutputColumns) > 0 {
			pairOutputSchema := buildOutputSchema(pairOutputColumns, slateData)
			pairScores := make([]*pb.PairScore, len(slateData.Rows))
			for i, row := range slateData.Rows {
				pairScores[i] = &pb.PairScore{
					OutputValues: extractRowOutputBytes(row, pairOutputColumns, pairOutputSchema, slateData),
				}
			}
			resp.PairScores = pairScores
			resp.PairOutputSchema = pairOutputSchema
		}
	}

	return resp
}

// buildSlateWiseResponse extracts slate scores from SlateData and target scores from the main
// ComponentData. Slate output columns come from slate predator/iris configs; target output
// columns come from ResponseConfig.Features.
func buildSlateWiseResponse(componentMatrix *matrix.ComponentMatrix, slateData *matrix.ComponentMatrix, conf *config.Config) *pb.SlateWiseResponse {
	resp := &pb.SlateWiseResponse{}

	// Target-level scores from the main matrix (one per target row)
	targetOutputColumns := conf.ResponseConfig.Features
	if len(targetOutputColumns) > 0 && len(componentMatrix.Rows) > 0 {
		targetOutputSchema := buildOutputSchema(targetOutputColumns, componentMatrix)
		targetScores := make([]*pb.TargetScore, len(componentMatrix.Rows))
		for i, row := range componentMatrix.Rows {
			targetScores[i] = &pb.TargetScore{
				OutputValues: extractRowOutputBytes(row, targetOutputColumns, targetOutputSchema, componentMatrix),
			}
		}
		resp.TargetScores = targetScores
		resp.TargetOutputSchema = targetOutputSchema
	}

	// Slate-level scores from SlateData (one per slate row)
	if slateData != nil && len(slateData.Rows) > 0 {
		slateOutputColumns := slateOutputColumnNames(conf)
		if len(slateOutputColumns) > 0 {
			slateOutputSchema := buildOutputSchema(slateOutputColumns, slateData)
			slateScores := make([]*pb.SlateScore, len(slateData.Rows))
			for i, row := range slateData.Rows {
				slateScores[i] = &pb.SlateScore{
					OutputValues: extractRowOutputBytes(row, slateOutputColumns, slateOutputSchema, slateData),
				}
			}
			resp.SlateScores = slateScores
			resp.SlateOutputSchema = slateOutputSchema
		}
	}

	return resp
}

// slateOutputColumnNames collects output column names from slate predator and iris configs.
func slateOutputColumnNames(conf *config.Config) []string {
	var cols []string
	for _, comp := range conf.ComponentConfig.PredatorComponentConfig.Values() {
		pComp, ok := comp.(config.PredatorComponentConfig)
		if !ok || !pComp.SlateComponent {
			continue
		}
		for _, out := range pComp.Outputs {
			cols = append(cols, out.ModelScores...)
		}
	}
	for _, comp := range conf.ComponentConfig.NumerixComponentConfig.Values() {
		iComp, ok := comp.(config.NumerixComponentConfig)
		if !ok || !iComp.SlateComponent {
			continue
		}
		cols = append(cols, iComp.ScoreColumn)
	}
	return cols
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// buildOutputSchema creates FeatureSchema entries for the output columns.
// It looks up each column in the byte column map first (typed), then string column map.
func buildOutputSchema(columns []string, m *matrix.ComponentMatrix) []*pb.FeatureSchema {
	schema := make([]*pb.FeatureSchema, len(columns))
	for i, col := range columns {
		schema[i] = &pb.FeatureSchema{
			Name:     col,
			DataType: resolveDataType(col, m),
		}
	}
	return schema
}

// resolveDataType determines the proto DataType for a column by checking the matrix column maps.
func resolveDataType(colName string, m *matrix.ComponentMatrix) pb.DataType {
	if byteCol, ok := m.ByteColumnIndexMap[colName]; ok {
		return mapDataTypeString(byteCol.DataType)
	}
	if stringCol, ok := m.StringColumnIndexMap[colName]; ok {
		return mapDataTypeString(stringCol.DataType)
	}
	return pb.DataType_DataTypeString // default fallback
}

// mapDataTypeString converts the internal DataType string to the proto enum.
func mapDataTypeString(dt string) pb.DataType {
	switch dt {
	case "DataTypeFP32", "fp32":
		return pb.DataType_DataTypeFP32
	case "DataTypeFP64", "fp64":
		return pb.DataType_DataTypeFP64
	case "DataTypeFP16", "fp16":
		return pb.DataType_DataTypeFP16
	case "DataTypeInt8", "int8":
		return pb.DataType_DataTypeInt8
	case "DataTypeInt16", "int16":
		return pb.DataType_DataTypeInt16
	case "DataTypeInt32", "int32":
		return pb.DataType_DataTypeInt32
	case "DataTypeInt64", "int64":
		return pb.DataType_DataTypeInt64
	case "DataTypeUint8", "uint8":
		return pb.DataType_DataTypeUint8
	case "DataTypeUint16", "uint16":
		return pb.DataType_DataTypeUint16
	case "DataTypeUint32", "uint32":
		return pb.DataType_DataTypeUint32
	case "DataTypeUint64", "uint64":
		return pb.DataType_DataTypeUint64
	case "DataTypeBool", "bool":
		return pb.DataType_DataTypeBool
	case "DataTypeFP8E5M2", "fp8e5m2":
		return pb.DataType_DataTypeFP8E5M2
	case "DataTypeFP8E4M3", "fp8e4m3":
		return pb.DataType_DataTypeFP8E4M3
	case "DataTypeFP32Vector", "fp32vector":
		return pb.DataType_DataTypeFP32Vector
	case "DataTypeFP64Vector", "fp64vector":
		return pb.DataType_DataTypeFP64Vector
	case "DataTypeInt32Vector", "int32vector":
		return pb.DataType_DataTypeInt32Vector
	case "DataTypeInt64Vector", "int64vector":
		return pb.DataType_DataTypeInt64Vector
	case "DataTypeStringVector", "stringvector":
		return pb.DataType_DataTypeStringVector
	default:
		return pb.DataType_DataTypeString
	}
}

// extractRowOutputBytes extracts the output column values from a single matrix row as [][]byte.
// It prefers byte columns (typed, zero-copy). For string columns, it converts the string value
// to typed bytes using the output schema's data type so the response carries actual typed values.
func extractRowOutputBytes(row matrix.Row, columns []string, outputSchema []*pb.FeatureSchema, m *matrix.ComponentMatrix) [][]byte {
	values := make([][]byte, len(columns))
	for i, col := range columns {
		if byteCol, ok := m.ByteColumnIndexMap[col]; ok {
			// Byte column: already typed, use directly
			if byteCol.Index < len(row.ByteData) {
				values[i] = row.ByteData[byteCol.Index]
			}
		} else if stringCol, ok := m.StringColumnIndexMap[col]; ok {
			if stringCol.Index < len(row.StringData) {
				strVal := row.StringData[stringCol.Index]
				// Convert string value to typed bytes using the output schema's data type
				if i < len(outputSchema) {
					targetDt := outputSchema[i].DataType.String()
					if converted, err := typeconverter.StringToBytes(strVal, targetDt); err == nil {
						values[i] = converted
						continue
					}
				}
				// Fallback: raw string bytes if conversion fails or no schema
				values[i] = []byte(strVal)
			}
		}
	}
	return values
}
