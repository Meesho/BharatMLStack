package inferflow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/components"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/matrix"
	pb "github.com/Meesho/BharatMLStack/inferflow/server/grpc/predict"
)

// adaptPointWiseRequest translates a PointWiseRequest into the existing ComponentRequest.
// Each Target becomes a row in the ComponentMatrix. Context features are broadcast to all rows.
func adaptPointWiseRequest(req *pb.PointWiseRequest, conf *config.Config, headers map[string]string) (*components.ComponentRequest, error) {
	numTargets := len(req.Targets)
	if numTargets == 0 {
		return nil, fmt.Errorf("pointwise request must have at least one target")
	}

	firstEntity := firstEntityFromConfig(conf)
	entities, entityIds, features := buildEntitiesFromTargets(req.Targets, req.ContextFeatures, req.TargetInputSchema, firstEntity)

	return &components.ComponentRequest{
		ComponentData:   &matrix.ComponentMatrix{},
		Entities:        &entities,
		EntityIds:       &entityIds,
		Features:        &features,
		ComponentConfig: &conf.ComponentConfig,
		ModelId:         req.ModelConfigId,
		Headers:         headers,
	}, nil
}

// adaptPairWiseRequest translates a PairWiseRequest into a slate-style ComponentRequest.
// A pair is a slate of exactly 2 targets. The adapter converts TargetPair entries into
// TargetSlate entries (target_indices = [first, second]) and delegates to the standard
// slate data builder. The downstream slate pipeline handles scoring without any changes.
func adaptPairWiseRequest(req *pb.PairWiseRequest, conf *config.Config, headers map[string]string) (*components.ComponentRequest, error) {
	if len(req.Pairs) == 0 {
		return nil, fmt.Errorf("pairwise request must have at least one pair")
	}
	if len(req.Targets) == 0 {
		return nil, fmt.Errorf("pairwise request must have at least one target")
	}
	if err := validatePairIndices(req.Pairs, len(req.Targets)); err != nil {
		return nil, err
	}

	firstEntity := firstEntityFromConfig(conf)

	// Main matrix: one row per target (for per-target feature fetch)
	entities, entityIds, features := buildEntitiesFromTargets(
		req.Targets, req.ContextFeatures, req.TargetInputSchema, firstEntity,
	)

	// Convert pairs → slates (each pair is a slate of size 2)
	slates := pairsToSlates(req.Pairs)

	// Build SlateData with slate_target_indices and pair-level features
	slateData := buildSlateData(req.Targets, slates, req.PairInputSchema, req.ContextFeatures)

	return &components.ComponentRequest{
		ComponentData:   &matrix.ComponentMatrix{},
		SlateData:       slateData,
		Entities:        &entities,
		EntityIds:       &entityIds,
		Features:        &features,
		ComponentConfig: &conf.ComponentConfig,
		ModelId:         req.ModelConfigId,
		Headers:         headers,
	}, nil
}

// pairsToSlates converts TargetPair entries to TargetSlate entries.
// Each pair becomes a slate with exactly 2 target indices.
func pairsToSlates(pairs []*pb.TargetPair) []*pb.TargetSlate {
	slates := make([]*pb.TargetSlate, len(pairs))
	for i, pair := range pairs {
		slates[i] = &pb.TargetSlate{
			TargetIndices: []int32{pair.FirstTargetIndex, pair.SecondTargetIndex},
			FeatureValues: pair.FeatureValues,
		}
	}
	return slates
}

// adaptSlateWiseRequest translates a SlateWiseRequest into a ComponentRequest.
// The main matrix has one row per target (so feature store can fill per-target features).
// SlateData has one row per slate with slate_target_indices pointing into the main matrix,
// plus any slate-level features. Slate components read from main matrix and write to SlateData.
func adaptSlateWiseRequest(req *pb.SlateWiseRequest, conf *config.Config, headers map[string]string) (*components.ComponentRequest, error) {
	numSlates := len(req.Slates)
	if numSlates == 0 {
		return nil, fmt.Errorf("slatewise request must have at least one slate")
	}
	if len(req.Targets) == 0 {
		return nil, fmt.Errorf("slatewise request must have at least one target")
	}
	if err := validateSlateIndices(req.Slates, len(req.Targets)); err != nil {
		return nil, err
	}

	firstEntity := firstEntityFromConfig(conf)

	// Main matrix: one row per target (for per-target feature fetch)
	entities, entityIds, features := buildEntitiesFromTargets(
		req.Targets, req.ContextFeatures, req.TargetInputSchema, firstEntity,
	)

	// Build SlateData with slate_target_indices and slate-level features
	slateData := buildSlateData(req.Targets, req.Slates, req.SlateInputSchema, req.ContextFeatures)

	return &components.ComponentRequest{
		ComponentData:   &matrix.ComponentMatrix{},
		SlateData:       slateData,
		Entities:        &entities,
		EntityIds:       &entityIds,
		Features:        &features,
		ComponentConfig: &conf.ComponentConfig,
		ModelId:         req.ModelConfigId,
		Headers:         headers,
	}, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// firstEntityFromConfig returns the first entity/key column name from the ETCD DAG response config.
// It uses ResponseConfig.Features[0] when set; otherwise the first feature component's first FSKey.Column.
func firstEntityFromConfig(conf *config.Config) string {
	if conf != nil && len(conf.ResponseConfig.Features) > 0 {
		return conf.ResponseConfig.Features[0]
	}
	return "id" // fallback for legacy configs
}

// buildEntitiesFromTargets creates the entity/feature structures for pointwise.
// The first entity dimension is the target ID (entity name from config); target and context features are added to the features map.
func buildEntitiesFromTargets(targets []*pb.Target, contextFeatures []*pb.ContextFeature, targetInputSchema []*pb.FeatureSchema, firstEntity string) ([]string, [][]string, map[string][]string) {
	entities := []string{firstEntity}
	targetIds := make([]string, len(targets))
	for i, t := range targets {
		targetIds[i] = t.Id
	}
	entityIds := [][]string{targetIds}

	features := make(map[string][]string)

	// Target-level features (one column per schema field, values from each target's feature_values)
	for schemaIdx, schema := range targetInputSchema {
		vals := make([]string, len(targets))
		for i, t := range targets {
			if schemaIdx < len(t.FeatureValues) {
				vals[i] = string(t.FeatureValues[schemaIdx])
			}
		}
		features[schema.Name] = vals
	}

	// Broadcast context features to all rows
	for _, cf := range contextFeatures {
		vals := make([]string, len(targets))
		strVal := string(cf.Value)
		for i := range vals {
			vals[i] = strVal
		}
		features[cf.Name] = vals
	}

	return entities, entityIds, features
}

// buildSlateData creates a pre-populated SlateData ComponentMatrix for slatewise requests.
// Each row represents one slate. The slate_target_indices string column holds comma-separated
// row indices into the main target matrix. Slate-level features are stored as string columns.
// feature_init will later call InitComponentMatrix to set up byte columns for slate outputs.
func buildSlateData(
	targets []*pb.Target,
	slates []*pb.TargetSlate,
	slateSchema []*pb.FeatureSchema,
	contextFeatures []*pb.ContextFeature,
) *matrix.ComponentMatrix {
	numSlates := len(slates)

	// Build slate_target_indices: comma-separated target row indices
	slateTargetIndices := make([]string, numSlates)
	for i, slate := range slates {
		parts := make([]string, len(slate.TargetIndices))
		for j, idx := range slate.TargetIndices {
			parts[j] = strconv.Itoa(int(idx))
		}
		slateTargetIndices[i] = strings.Join(parts, ",")
	}

	// Build typed string/byte column schemas and value buffers.
	stringCols := make(map[string]matrix.Column)
	byteCols := make(map[string]matrix.Column)
	stringValues := make(map[string][]string)
	byteValues := make(map[string][][]byte)
	stringColIdx := 0
	byteColIdx := 0

	addStringCol := func(name string) {
		if _, exists := stringCols[name]; exists {
			return
		}
		stringCols[name] = matrix.Column{
			Name:     name,
			DataType: "string",
			Index:    stringColIdx,
		}
		stringColIdx++
	}
	addByteCol := func(name string, dt pb.DataType) {
		if _, exists := byteCols[name]; exists {
			return
		}
		byteCols[name] = matrix.Column{
			Name:     name,
			DataType: dt.String(),
			Index:    byteColIdx,
		}
		byteColIdx++
	}

	// Always keep slate_target_indices as string column.
	addStringCol(components.SlateTargetIndicesColumn)
	stringValues[components.SlateTargetIndicesColumn] = slateTargetIndices

	// Slate-level features from TargetSlate.feature_values using slate_input_schema data types.
	for schemaIdx, schema := range slateSchema {
		if schema.GetDataType() == pb.DataType_DataTypeString {
			addStringCol(schema.Name)
			vals := make([]string, numSlates)
			for i, slate := range slates {
				if schemaIdx < len(slate.FeatureValues) {
					vals[i] = string(slate.FeatureValues[schemaIdx])
				}
			}
			stringValues[schema.Name] = vals
			continue
		}

		addByteCol(schema.Name, schema.GetDataType())
		vals := make([][]byte, numSlates)
		for i, slate := range slates {
			if schemaIdx < len(slate.FeatureValues) {
				vals[i] = slate.FeatureValues[schemaIdx]
			}
		}
		byteValues[schema.Name] = vals
	}

	// Context features broadcast to all slate rows, typed by context feature schema.
	for _, cf := range contextFeatures {
		if cf.GetDataType() == pb.DataType_DataTypeString {
			addStringCol(cf.Name)
			vals := make([]string, numSlates)
			strVal := string(cf.Value)
			for i := range vals {
				vals[i] = strVal
			}
			stringValues[cf.Name] = vals
			continue
		}

		addByteCol(cf.Name, cf.GetDataType())
		vals := make([][]byte, numSlates)
		for i := range vals {
			vals[i] = cf.Value
		}
		byteValues[cf.Name] = vals
	}

	// Allocate rows with both string and byte schemas.
	slateMatrix := &matrix.ComponentMatrix{}
	slateMatrix.InitComponentMatrix(numSlates, stringCols, byteCols)

	// Populate preserved string features.
	for colName, vals := range stringValues {
		slateMatrix.PopulateStringData(colName, vals)
	}

	// Populate typed byte features.
	for colName, vals := range byteValues {
		slateMatrix.PopulateByteData(colName, vals)
	}

	return slateMatrix
}

func validatePairIndices(pairs []*pb.TargetPair, targetCount int) error {
	for i, pair := range pairs {
		if pair == nil {
			return fmt.Errorf("pairwise request has nil pair at index %d", i)
		}

		firstIndex := int(pair.FirstTargetIndex)
		if firstIndex < 0 || firstIndex >= targetCount {
			return fmt.Errorf(
				"pairwise request has out-of-range first_target_index=%d at pair index %d (target count=%d)",
				pair.FirstTargetIndex, i, targetCount,
			)
		}

		secondIndex := int(pair.SecondTargetIndex)
		if secondIndex < 0 || secondIndex >= targetCount {
			return fmt.Errorf(
				"pairwise request has out-of-range second_target_index=%d at pair index %d (target count=%d)",
				pair.SecondTargetIndex, i, targetCount,
			)
		}
	}
	return nil
}

func validateSlateIndices(slates []*pb.TargetSlate, targetCount int) error {
	for slateIdx, slate := range slates {
		if slate == nil {
			return fmt.Errorf("slatewise request has nil slate at index %d", slateIdx)
		}

		for targetIdxPos, targetIdx := range slate.TargetIndices {
			index := int(targetIdx)
			if index < 0 || index >= targetCount {
				return fmt.Errorf(
					"slatewise request has out-of-range target index=%d at slates[%d].target_indices[%d] (target count=%d)",
					targetIdx, slateIdx, targetIdxPos, targetCount,
				)
			}
		}
	}
	return nil
}
