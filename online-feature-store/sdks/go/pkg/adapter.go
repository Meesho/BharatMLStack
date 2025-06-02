package gosdk

import (
	"errors"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/persist"
	pb "github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
)

type IAdapter interface {
	ConvertToResult(payload *pb.Result) Result
	ConvertToQueriesProto(query *Query, batchSize int) ([]*pb.Query, error)
	ConvertToDecodedResult(payload *pb.DecodedResult) *DecodedResult
	ConvertToPersistRequest(request *PersistFeaturesRequest) *persist.Query
}

type Adapter struct {
}

func (a *Adapter) ConvertToQueriesProto(query *Query, batchSize int) ([]*pb.Query, error) {
	if query == nil {
		return nil, nil
	}

	protoFeatureGroups := createFeatureGroups(query.FeatureGroups)
	protoKeys := query.Keys

	batchedKeys, err := batch(protoKeys, batchSize)
	if err != nil {
		return nil, err
	}

	return createQueries(query.EntityLabel, protoFeatureGroups, query.KeysSchema, batchedKeys), nil
}

func createFeatureGroups(featureGroups []FeatureGroup) []*pb.FeatureGroup {
	pbFeatureGroups := make([]*pb.FeatureGroup, len(featureGroups))
	for groupIdx, featureGroup := range featureGroups {
		pbFeatureGroups[groupIdx] = &pb.FeatureGroup{
			Label:         featureGroup.Label,
			FeatureLabels: featureGroup.FeatureLabels,
		}
	}
	return pbFeatureGroups
}

func createQueries(entityLabel string, featureGroups []*pb.FeatureGroup, keysSchema []string, batchedKeys [][]Keys) []*pb.Query {
	queries := make([]*pb.Query, 0, len(batchedKeys))
	for _, batchKeys := range batchedKeys {
		pbKeys := make([]*pb.Keys, len(batchKeys))
		for i, key := range batchKeys {
			pbKeys[i] = &pb.Keys{
				Cols: key.Cols,
			}
		}
		query := &pb.Query{
			EntityLabel:   entityLabel,
			FeatureGroups: featureGroups,
			KeysSchema:    keysSchema,
			Keys:          pbKeys,
		}
		queries = append(queries, query)
	}
	return queries
}

func (a *Adapter) ConvertToResult(payload *pb.Result) Result {
	if payload == nil || len(payload.Rows) == 0 {
		return Result{}
	}

	result := Result{
		EntityLabel:    payload.EntityLabel,
		KeysSchema:     payload.KeysSchema,
		FeatureSchemas: convertFeatureSchemas(payload.FeatureSchemas),
		Rows:           convertRows(payload.Rows),
	}
	return result
}

func (a *Adapter) ConvertToDecodedResult(payload *pb.DecodedResult) *DecodedResult {
	if payload == nil || len(payload.Rows) == 0 {
		return nil
	}

	result := &DecodedResult{
		KeysSchema:     payload.KeysSchema,
		FeatureSchemas: convertFeatureSchemas(payload.FeatureSchemas),
		Rows:           convertDecodedRows(payload.Rows),
	}
	return result
}

func convertFeatureSchemas(protoSchemas []*pb.FeatureSchema) []FeatureSchema {
	if len(protoSchemas) == 0 {
		return []FeatureSchema{}
	}

	schemas := make([]FeatureSchema, len(protoSchemas))
	for schemaIndex, protoSchema := range protoSchemas {
		schemas[schemaIndex] = FeatureSchema{
			FeatureGroupLabel: protoSchema.FeatureGroupLabel,
			Features:          convertFeatures(protoSchema.Features),
		}
	}
	return schemas
}

func convertFeatures(protoFeatures []*pb.Feature) []Feature {
	if len(protoFeatures) == 0 {
		return []Feature{}
	}

	features := make([]Feature, len(protoFeatures))
	for featureIndex, protoFeature := range protoFeatures {
		features[featureIndex] = Feature{
			Label:     protoFeature.Label,
			ColumnIdx: protoFeature.ColumnIdx,
		}
	}
	return features
}

func convertRows(protoRows []*pb.Row) []Row {
	if len(protoRows) == 0 {
		return []Row{}
	}

	rows := make([]Row, len(protoRows))
	for rowIndex, protoRow := range protoRows {
		rows[rowIndex] = Row{
			Keys:    protoRow.Keys,
			Columns: protoRow.Columns,
		}
	}
	return rows
}

func convertDecodedRows(protoRows []*pb.DecodedRow) []DecodedRow {
	if len(protoRows) == 0 {
		return []DecodedRow{}
	}

	rows := make([]DecodedRow, len(protoRows))
	for rowIndex, protoRow := range protoRows {
		rows[rowIndex] = DecodedRow{
			Keys:    protoRow.Keys,
			Columns: protoRow.Columns,
		}
	}
	return rows
}

func batch[T any](in []T, batchSize int) ([][]T, error) {
	if batchSize <= 0 {
		return nil, errors.New("batch size must be positive")
	}

	if len(in) == 0 {
		return nil, nil
	}

	numBatches := (len(in) + batchSize - 1) / batchSize
	batches := make([][]T, numBatches)

	for i := 0; i < numBatches; i++ {
		start := i * batchSize
		end := start + batchSize
		if end > len(in) {
			end = len(in)
		}
		batches[i] = in[start:end]
	}

	return batches, nil
}

func (a *Adapter) ConvertToPersistRequest(request *PersistFeaturesRequest) *persist.Query {
	if request == nil {
		return nil
	}

	protoRequest := &persist.Query{
		EntityLabel:        request.EntityLabel,
		KeysSchema:         request.KeysSchema,
		FeatureGroupSchema: make([]*persist.FeatureGroupSchema, len(request.FeatureGroups)),
		Data:               make([]*persist.Data, len(request.Data)),
	}

	// Convert FeatureGroups
	for i, fg := range request.FeatureGroups {
		protoRequest.FeatureGroupSchema[i] = &persist.FeatureGroupSchema{
			Label:         fg.Label,
			FeatureLabels: fg.FeatureLabels,
		}
	}

	// Convert Data
	for i, d := range request.Data {
		protoRequest.Data[i] = &persist.Data{
			KeyValues:     d.KeyValues,
			FeatureValues: make([]*persist.FeatureValues, len(d.FeatureValues)),
		}

		// Convert FeatureValues
		for j, fv := range d.FeatureValues {
			protoRequest.Data[i].FeatureValues[j] = &persist.FeatureValues{
				Values: &persist.Values{
					Fp32Values:   fv.Values.Fp32Values,
					Fp64Values:   fv.Values.Fp64Values,
					Int32Values:  fv.Values.Int32Values,
					Int64Values:  fv.Values.Int64Values,
					Uint32Values: fv.Values.Uint32Values,
					Uint64Values: fv.Values.Uint64Values,
					StringValues: fv.Values.StringValues,
					BoolValues:   fv.Values.BoolValues,
					Vector:       make([]*persist.Vector, len(fv.Values.Vector)),
				},
			}

			// Convert Vector values
			for k, v := range fv.Values.Vector {
				protoRequest.Data[i].FeatureValues[j].Values.Vector[k] = &persist.Vector{
					Values: &persist.Values{
						Fp32Values:   v.Values.Fp32Values,
						Fp64Values:   v.Values.Fp64Values,
						Int32Values:  v.Values.Int32Values,
						Int64Values:  v.Values.Int64Values,
						Uint32Values: v.Values.Uint32Values,
						Uint64Values: v.Values.Uint64Values,
						StringValues: v.Values.StringValues,
						BoolValues:   v.Values.BoolValues,
					},
				}
			}
		}
	}

	return protoRequest
}
