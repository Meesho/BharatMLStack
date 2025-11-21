package handler

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	inferflow "github.com/Meesho/BharatMLStack/horizon/internal/inferflow"
	ofsHandler "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/handler"

	etcd "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/etcd"
	mapset "github.com/deckarep/golang-set/v2"
)

const (
	PARENT                      = "PARENT"
	DEFAULT_FEATURE             = "DEFAULT"
	ONLINE_FEATURE              = "ONLINE"
	MODEL_FEATURE               = "MODEL"
	OFFLINE_FEATURE             = "OFFLINE"
	CALIBRATION                 = "CALIBRATION"
	RTP_FEATURE                 = "RTP"
	PCTR_CALIBRATION            = "PCTR_CALIBRATION"
	PCVR_CALIBRATION            = "PCVR_CALIBRATION"
	PIPE_DELIMITER              = "|"
	UNDERSCORE_DELIMITER        = "_"
	COLON_DELIMITER             = ":"
	COMMA_DELIMITER             = ","
	featureClassOffline         = "offline"
	featureClassOnline          = "online"
	featureClassDefault         = "default"
	featureClassRtp             = "rtp"
	featureClassPCVRCalibration = "pcvr_calibration"
	featureClassPCTRCalibration = "pctr_calibration"
	featureClassInvalid         = "invalid"
	COMPONENT_NAME_PREFIX       = "composite_key_gen_"
	FEATURE_INITIALIZER         = "feature_initializer"
)

func (m *InferFlow) GetInferflowConfig(request InferflowOnboardRequest, token string) (InferflowConfig, error) {
	// RTP client is initialized in externalcall.Init()
	entityIDs := extractEntityIDs(request)

	featureList, featureToDataType, rtpFeatures, pcvrCalibrationFeatures, pctrCalibrationFeatures, predatorAndNumerixOutputsToDataType, offlineToOnlineMapping, err := GetFeatureList(request, m.EtcdConfig, token, entityIDs)
	if err != nil {
		return InferflowConfig{}, err
	}

	predatorComponents, err := GetPredatorComponents(request, offlineToOnlineMapping)
	if err != nil {
		return InferflowConfig{}, err
	}

	NumerixComponents, err := GetNumerixComponents(request, offlineToOnlineMapping, predatorAndNumerixOutputsToDataType, featureToDataType)
	if err != nil {
		return InferflowConfig{}, err
	}

	responseConfig, err := GetResponseConfigs(&request)
	if err != nil {
		return InferflowConfig{}, err
	}

	rtpComponents, err := GetRTPComponents(request, rtpFeatures, featureToDataType, m.EtcdConfig, token)
	if err != nil {
		return InferflowConfig{}, err
	}

	featureComponents, err := GetFeatureComponents(request, featureList, featureToDataType, pcvrCalibrationFeatures, pctrCalibrationFeatures, m.EtcdConfig, token, entityIDs)
	if err != nil {
		return InferflowConfig{}, err
	}

	componentConfig, err := GetComponentConfig(featureComponents, rtpComponents, NumerixComponents, predatorComponents)
	if err != nil {
		return InferflowConfig{}, err
	}

	dagExecutionConfig, err := GetDagExecutionConfig(request, featureComponents, rtpComponents, NumerixComponents, predatorComponents, m.EtcdConfig)
	if err != nil {
		return InferflowConfig{}, err
	}

	mpConfig := InferflowConfig{
		DagExecutionConfig: dagExecutionConfig,
		ComponentConfig:    componentConfig,
		ResponseConfig:     responseConfig,
	}

	return mpConfig, nil
}

// GetFeatureList extracts features from the request and fetches the component features from the etcd config
// and returns a set of features, a map of feature to data type, a map of offline feature to online feature
// and an error if any.
func GetFeatureList(request InferflowOnboardRequest, etcdConfig etcd.Manager, token string, entityIDs map[string]bool) (mapset.Set[string], map[string]string, mapset.Set[string], mapset.Set[string], mapset.Set[string], map[string]string, map[string]string, error) {
	initialFeatures, featureToDataType, predatorAndIrisOutputsToDataType := extractFeatures(request, entityIDs)

	offlineFeatures, onlineFeatures, _, rtpFeatures, pctrCalibrationFeatures, pcvrCalibrationFeatures, newFeatureToDataType, err := classifyFeatures(initialFeatures, featureToDataType)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	offlineToOnlineMapping, err := mapOfflineFeatures(offlineFeatures, token)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	for f, dtype := range newFeatureToDataType {
		featureToDataType[f] = dtype
	}

	features := mapset.NewSet[string]()
	for offlineFeature, onlineFeature := range offlineToOnlineMapping {
		features.Add(onlineFeature)
		featureToDataType[onlineFeature] = featureToDataType[offlineFeature]
		delete(featureToDataType, offlineFeature)
	}

	for _, f := range onlineFeatures.ToSlice() {
		features.Add(f)
	}

	// Fetch RTP registry once for classification
	rtpRegistry, err := GetRTPFeatureGroupDataTypeMap()
	if err != nil && inferflow.IsMeeshoEnabled {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to get RTP registry: %w", err)
	}

	// Fetch component features from RTP components
	rtpComponentFSFeatures, rtpComponentRTPFeatures, rtpComponentFeatureToDataType, err := fetchRTPComponentFeaturesWithClassification(rtpFeatures, etcdConfig, rtpRegistry)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	// Add FS features to the main features set
	for _, f := range rtpComponentFSFeatures.ToSlice() {
		features.Add(f)
	}

	// Add newly discovered RTP features to rtpFeatures set
	for _, f := range rtpComponentRTPFeatures.ToSlice() {
		rtpFeatures.Add(f)
	}

	for f, dtype := range rtpComponentFeatureToDataType {
		featureToDataType[f] = dtype
	}

	// Fetch component features from regular FS components
	componentFSFeatures, componentRTPFeatures, newfeatureToDataType, err := fetchComponentFeaturesWithClassification(features, pctrCalibrationFeatures, pcvrCalibrationFeatures, etcdConfig, request.Payload.RealEstate, token, rtpRegistry)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	// Add FS features to the main features set
	for _, f := range componentFSFeatures.ToSlice() {
		features.Add(f)
	}

	// Add newly discovered RTP features to rtpFeatures set
	for _, f := range componentRTPFeatures.ToSlice() {
		rtpFeatures.Add(f)
	}

	for f, dtype := range newfeatureToDataType {
		featureToDataType[f] = dtype
	}

	// for _, f := range defaultFeatures.ToSlice() {
	// 	features.Add(f)
	// }

	if err := fetchMissingDatatypes(featureToDataType, rtpFeatures, pctrCalibrationFeatures, pcvrCalibrationFeatures, onlineFeatures, token); err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	return features, featureToDataType, rtpFeatures, pctrCalibrationFeatures, pcvrCalibrationFeatures, predatorAndIrisOutputsToDataType, offlineToOnlineMapping, nil
}

func extractEntityIDs(request InferflowOnboardRequest) map[string]bool {
	entityIDs := make(map[string]bool)
	for _, ranker := range request.Payload.Rankers {
		entityID := strings.Join(ranker.EntityID, COLON_DELIMITER)
		entityIDs[entityID] = false
	}
	for _, reRanker := range request.Payload.ReRankers {
		entityID := strings.Join(reRanker.EntityID, COLON_DELIMITER)
		entityIDs[entityID] = false
	}
	return entityIDs
}

// extractFeatures extracts features from the request and returns a set of features and a map of
// feature to data type, the features in it will have atleast two parts after split by PIPE_DELIMITER
// and the feature should not be the entity_id
func extractFeatures(request InferflowOnboardRequest, entityIDs map[string]bool) (mapset.Set[string], map[string]string, map[string]string) {
	features := mapset.NewSet[string]()
	featureToDataType := make(map[string]string)
	outputFeatures := mapset.NewSet[string]()
	predatorOutputs := mapset.NewSet[string]()
	predatorAndNumerixOutputsToDataType := make(map[string]string)

	for _, ranker := range request.Payload.Rankers {
		for _, output := range ranker.Outputs {
			for _, ms := range output.ModelScores {
				predatorOutputs.Add(ms)
				predatorAndNumerixOutputsToDataType[ms] = output.DataType
			}
		}
	}

	addFeature := func(feature, dtype string) {
		if _, ok := entityIDs[feature]; ok {
			return
		}
		if parts := strings.Split(feature, PIPE_DELIMITER); len(parts) >= 2 && !predatorOutputs.Contains(feature) {
			features.Add(feature)
			featureToDataType[feature] = dtype
		}
	}

	for _, ranker := range request.Payload.Rankers {
		for _, input := range ranker.Inputs {
			for _, feature := range input.Features {
				addFeature(feature, input.DataType)
			}
		}

		for _, output := range ranker.Outputs {
			outputFeatures.Add(output.Name)
		}
	}

	for _, reRanker := range request.Payload.ReRankers {
		for _, feature := range reRanker.EqVariables {
			if _, ok := entityIDs[feature]; ok {
				continue
			}
			parts := strings.Split(feature, PIPE_DELIMITER)
			if len(parts) < 2 {
				continue
			}
			featureName := parts[1]
			if predatorOutputs.Contains(feature) || predatorOutputs.Contains(featureName) {
				continue
			}
			features.Add(feature)
			featureToDataType[feature] = ""
		}
		predatorAndNumerixOutputsToDataType[reRanker.Score] = reRanker.DataType
	}

	return features, featureToDataType, predatorAndNumerixOutputsToDataType
}

func fetchMissingDatatypes(
	featureToDataType map[string]string,
	rtpFeatures mapset.Set[string],
	pctrCalibrationFeatures mapset.Set[string],
	pcvrCalibrationFeatures mapset.Set[string],
	onlineFeatures mapset.Set[string],
	token string,
) error {
	hasEmptyDatatype := false
	for _, dtype := range featureToDataType {
		if dtype == "" {
			hasEmptyDatatype = true
			break
		}
	}
	if !hasEmptyDatatype {
		return nil
	}

	horizonFeatures := make(map[string]struct{ label, group string })
	rtpFeaturesToFetch := mapset.NewSet[string]()

	for feature, dtype := range featureToDataType {
		if dtype != "" {
			continue
		}

		parts := strings.Split(feature, COLON_DELIMITER)

		// Calibration features
		if pctrCalibrationFeatures.Contains(feature) {
			if len(parts) >= 3 {
				label := parts[1]
				group := parts[2]
				horizonFeatures[feature] = struct{ label, group string }{label, group}
			}
			continue
		}
		if pcvrCalibrationFeatures.Contains(feature) {
			if len(parts) >= 3 {
				label := parts[1]
				group := parts[2]
				horizonFeatures[feature] = struct{ label, group string }{label, group}
			}
			continue
		}

		// Check if it's an RTP feature
		if rtpFeatures.Contains(feature) {
			rtpFeaturesToFetch.Add(feature)
			continue
		}

		// Check if it's an online feature
		if onlineFeatures.Contains(feature) {
			if len(parts) >= 2 {
				if strings.HasPrefix(feature, "parent:") && len(parts) == 2 {
					continue
				}
				label := parts[0]
				group := parts[1]
				horizonFeatures[feature] = struct{ label, group string }{label, group}
			}
			continue
		}
	}

	// Query RTP API once for all RTP datatypes
	if rtpFeaturesToFetch.Cardinality() > 0 {
		rtpDataTypeMap, err := GetRTPFeatureGroupDataTypeMap()
		if err == nil {
			for _, feature := range rtpFeaturesToFetch.ToSlice() {
				if dataType, exists := rtpDataTypeMap[feature]; exists {
					featureToDataType[feature] = dataType
					continue
				}

				parts := strings.Split(feature, COLON_DELIMITER)
				if len(parts) == 4 {
					withoutPrefix := strings.Join(parts[1:], COLON_DELIMITER)
					if dataType, exists := rtpDataTypeMap[withoutPrefix]; exists {
						featureToDataType[feature] = dataType
					}
				}
			}
		}
	}

	// Query Horizon API for remaining features grouped by label
	if len(horizonFeatures) > 0 {
		labelToGroups := make(map[string]mapset.Set[string])

		for _, labelGroup := range horizonFeatures {
			if labelToGroups[labelGroup.label] == nil {
				labelToGroups[labelGroup.label] = mapset.NewSet[string]()
			}
			labelToGroups[labelGroup.label].Add(labelGroup.group)
		}

		labelToGroupDataType := make(map[string]map[string]string)
		for label := range labelToGroups {
			featureGroupDataTypeMap, err := GetFeatureGroupDataTypeMap(label, token)
			if err != nil {
				continue
			}
			labelToGroupDataType[label] = featureGroupDataTypeMap
		}

		for feature, labelGroup := range horizonFeatures {
			if featureToDataType[feature] != "" {
				continue
			}

			if groupMap, exists := labelToGroupDataType[labelGroup.label]; exists {
				if dataType, exists := groupMap[labelGroup.group]; exists {
					featureToDataType[feature] = dataType
				}
			}
		}
	}

	return nil
}

// classifyFeatures classifies features into offline, online and default features
// and returns a set of features for each class and a map of feature to data type
func classifyFeatures(
	featureList mapset.Set[string],
	featureDataTypes map[string]string,
) (mapset.Set[string], mapset.Set[string], mapset.Set[string], mapset.Set[string], mapset.Set[string], mapset.Set[string], map[string]string, error) {
	defaultFeatures := mapset.NewSet[string]()
	onlineFeatures := mapset.NewSet[string]()
	offlineFeatures := mapset.NewSet[string]()
	rtpFeatures := mapset.NewSet[string]()
	pctrCalibrationFeatures := mapset.NewSet[string]()
	pcvrCalibrationFeatures := mapset.NewSet[string]()
	newFeatureToDataType := make(map[string]string)

	add := func(name, originalFeature string, featureType string) error {
		if err := AddFeatureToSet(&defaultFeatures, &onlineFeatures, &offlineFeatures, &rtpFeatures, &pctrCalibrationFeatures, &pcvrCalibrationFeatures, name, featureType); err != nil {
			return fmt.Errorf("error classifying feature: %w", err)
		}
		newFeatureToDataType[name] = featureDataTypes[originalFeature]
		return nil
	}

	for feature := range featureList.Iter() {
		transformedFeature, featureType, err := transformFeature(feature)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, err
		}

		if err := add(transformedFeature, feature, featureType); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, err
		}
	}

	return offlineFeatures, onlineFeatures, defaultFeatures, rtpFeatures, pctrCalibrationFeatures, pcvrCalibrationFeatures, newFeatureToDataType, nil
}

func AddFeatureToSet(defaultFeatures, onlineFeatures, offlineFeatures, rtpFeatures, pctrCalibrationFeatures, pcvrCalibrationFeatures *mapset.Set[string], feature string, featureType string) error {
	allSets := map[string]*mapset.Set[string]{
		featureClassDefault:         defaultFeatures,
		featureClassOnline:          onlineFeatures,
		featureClassOffline:         offlineFeatures,
		featureClassRtp:             rtpFeatures,
		featureClassPCTRCalibration: pctrCalibrationFeatures,
		featureClassPCVRCalibration: pcvrCalibrationFeatures,
	}

	for setType, set := range allSets {
		if setType != featureType && (*set).Contains(feature) {
			return fmt.Errorf("feature '%s' already exists in %s features, cannot add to %s features", feature, setType, featureType)
		}
	}

	targetSet, exists := allSets[featureType]
	if !exists {
		return fmt.Errorf("invalid feature type '%s' for feature '%s'", featureType, feature)
	}

	(*targetSet).Add(feature)
	return nil
}

// transformFeature transforms the feature to either online, offline or default feature
// and returns the transformed feature and the feature type. The feature can be of these given types:
// 1. PARENT_OFFLINE_FEATURE|FEATURE
// 2. DEFAULT_FEATURE|FEATURE
// 3. ONLINE_FEATURE|FEATURE
// 4. OFFLINE_FEATURE|FEATURE
// 5. PARENT_DEFAULT_FEATURE|FEATURE
// 6. PARENT_ONLINE_FEATURE|FEATURE
// 7. MODEL_FEATURE|FEATURE
// 8. CALIBRATION|FEATURE
// 9. RTP_FEATURE|FEATURE
// 10. PARENT_RTP_FEATURE|FEATURE
func transformFeature(feature string) (string, string, error) {
	parts := strings.Split(feature, PIPE_DELIMITER)
	if len(parts) < 2 {
		return "", featureClassInvalid, fmt.Errorf("feature %s is invalid", feature)
	}

	featureTypes := strings.Split(parts[0], UNDERSCORE_DELIMITER)
	featureName := parts[1]

	if parts[0] == PCTR_CALIBRATION {
		return strings.ToLower(PCTR_CALIBRATION) + COLON_DELIMITER + featureName, featureClassPCTRCalibration, nil
	}
	if parts[0] == PCVR_CALIBRATION {
		return strings.ToLower(PCVR_CALIBRATION) + COLON_DELIMITER + featureName, featureClassPCVRCalibration, nil
	}

	if featureTypes[0] == PARENT {
		if len(featureTypes) > 1 {
			newFeature := strings.ToLower(featureTypes[0]) + COLON_DELIMITER + featureName
			switch featureTypes[1] {
			case DEFAULT_FEATURE:
				return newFeature, featureClassDefault, nil
			case ONLINE_FEATURE, CALIBRATION:
				return newFeature, featureClassOnline, nil
			case OFFLINE_FEATURE:
				return newFeature, featureClassOffline, nil
			case RTP_FEATURE:
				return newFeature, featureClassRtp, nil
			case PCVR_CALIBRATION:
				return newFeature, featureClassPCVRCalibration, nil
			case PCTR_CALIBRATION:
				return newFeature, featureClassPCTRCalibration, nil
			}
		}
	}

	switch featureTypes[0] {
	case DEFAULT_FEATURE:
		return featureName, featureClassDefault, nil
	case ONLINE_FEATURE, CALIBRATION:
		return featureName, featureClassOnline, nil
	case OFFLINE_FEATURE:
		return featureName, featureClassOffline, nil
	case RTP_FEATURE:
		return featureName, featureClassRtp, nil
	case PCVR_CALIBRATION:
		return featureName, featureClassPCVRCalibration, nil
	case PCTR_CALIBRATION:
		return featureName, featureClassPCTRCalibration, nil
	}

	return featureName, featureClassDefault, nil
}

// mapOfflineFeatures maps the offline features to the online features
// and returns a map of offline feature to online feature
func mapOfflineFeatures(offlineFeatureList mapset.Set[string], token string) (map[string]string, error) {
	return GetOnlineFeatureMapping(offlineFeatureList, token)
}

func fetchRTPComponentFeaturesWithClassification(rtpFeatures mapset.Set[string], etcdConfig etcd.Manager, rtpRegistry map[string]string) (mapset.Set[string], mapset.Set[string], map[string]string, error) {
	componentList := getComponentList(rtpFeatures, nil, nil)
	fsFeatures := mapset.NewSet[string]()
	newRTPFeatures := mapset.NewSet[string]()
	featureToDataType := make(map[string]string)

	for _, component := range componentList.ToSlice() {
		componentData := etcdConfig.GetComponentData(component)
		if componentData == nil {
			return nil, nil, nil, fmt.Errorf("RTP Component: componentData for '%s' not found in registry", component)
		}

		for _, pair := range componentData.FSIdSchemaToValueColumns {
			if strings.Contains(pair.ValueCol, COLON_DELIMITER) {
				// Check if this is an RTP feature or FS feature
				isRTPFeature := false

				// Check direct match in RTP registry
				if _, exists := rtpRegistry[pair.ValueCol]; exists {
					isRTPFeature = true
				} else {
					// Check with prefix removed (for features like "parent:entity:group:feature")
					parts := strings.Split(pair.ValueCol, COLON_DELIMITER)
					if len(parts) == 4 {
						withoutPrefix := strings.Join(parts[1:], COLON_DELIMITER)
						if _, exists := rtpRegistry[withoutPrefix]; exists {
							isRTPFeature = true
						}
					}
				}

				if isRTPFeature {
					newRTPFeatures.Add(pair.ValueCol)
				} else {
					fsFeatures.Add(pair.ValueCol)
				}
				featureToDataType[pair.ValueCol] = pair.DataType
			}
		}
	}

	return fsFeatures, newRTPFeatures, featureToDataType, nil
}

// fetchComponentFeaturesWithClassification fetches the component features from the etcd config
// and classifies them as RTP or FS features, returns both sets and a map of feature to data type
func fetchComponentFeaturesWithClassification(features mapset.Set[string], pctrCalibrationFeatures mapset.Set[string], pcvrCalibrationFeatures mapset.Set[string], etcdConfig etcd.Manager, realEstate string, token string, rtpRegistry map[string]string) (mapset.Set[string], mapset.Set[string], map[string]string, error) {
	componentList := getComponentList(features, pctrCalibrationFeatures, pcvrCalibrationFeatures)
	fsFeatures := mapset.NewSet[string]()
	newRTPFeatures := mapset.NewSet[string]()
	featureToDataType := make(map[string]string)

	for _, component := range componentList.ToSlice() {
		componentData := etcdConfig.GetComponentData(component)
		if componentData == nil {
			return nil, nil, nil, fmt.Errorf("Component Data: ComponentData for '%s' not found in registry.\nPlease Contact MLP Team to onboard the component.", component)
		}

		for _, pair := range componentData.FSIdSchemaToValueColumns {
			if strings.Contains(pair.ValueCol, COLON_DELIMITER) {
				// Check if this is an RTP feature or FS feature
				isRTPFeature := false

				// Check direct match in RTP registry
				if _, exists := rtpRegistry[pair.ValueCol]; exists {
					isRTPFeature = true
				} else {
					// Check with prefix removed (for features like "parent:entity:group:feature")
					parts := strings.Split(pair.ValueCol, COLON_DELIMITER)
					if len(parts) == 4 {
						withoutPrefix := strings.Join(parts[1:], COLON_DELIMITER)
						if _, exists := rtpRegistry[withoutPrefix]; exists {
							isRTPFeature = true
						}
					}
				}

				if isRTPFeature {
					newRTPFeatures.Add(pair.ValueCol)
				} else {
					fsFeatures.Add(pair.ValueCol)
				}
				featureToDataType[pair.ValueCol] = pair.DataType
			}
		}

		if override, hasOverride := componentData.Overridecomponent[realEstate]; hasOverride {
			// Override components are always FS features
			fsFeatures.Add(override.ComponentId)
			parts := strings.Split(override.ComponentId, COLON_DELIMITER)
			var label, group string

			if len(parts) == 3 {
				label, group = parts[0], parts[1]
			} else if len(parts) == 4 {
				label, group = parts[1], parts[2]
			} else {
				return nil, nil, nil, fmt.Errorf("Component Data: invalid override component id: %s", override.ComponentId)
			}

			featureGroupDataTypeMap, err := GetFeatureGroupDataTypeMap(label, token)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("Component Data: error getting feature group data type map: %w", err)
			}

			if dataType, exists := featureGroupDataTypeMap[group]; exists {
				featureToDataType[override.ComponentId] = dataType
			} else {
				return nil, nil, nil, fmt.Errorf("Component Data: feature group data type not found for %s: %s", override.ComponentId, group)
			}
		}
	}

	return fsFeatures, newRTPFeatures, featureToDataType, nil
}

// getComponentList gets the component list from the features
// and returns a set of component names. The component name can be of these given types:
// 1. parent_<label>
// 2. pctr_calibration_<label>
// 3. pcvr_calibration_<label>
func getComponentList(features mapset.Set[string], pctrCalibrationFeatures mapset.Set[string], pcvrCalibrationFeatures mapset.Set[string]) mapset.Set[string] {
	componentList := mapset.NewSet[string]()

	for _, feature := range features.ToSlice() {
		parts := strings.Split(feature, COLON_DELIMITER)
		component := parts[0]
		if component == strings.ToLower(PARENT) {
			component = component + UNDERSCORE_DELIMITER + parts[1]
		}
		componentList.Add(component)
	}

	if pctrCalibrationFeatures != nil {
		for _, feature := range pctrCalibrationFeatures.ToSlice() {
			parts := strings.Split(feature, COLON_DELIMITER)
			component := parts[0] + UNDERSCORE_DELIMITER + parts[1]
			componentList.Add(component)
		}
	}

	if pcvrCalibrationFeatures != nil {
		for _, feature := range pcvrCalibrationFeatures.ToSlice() {
			parts := strings.Split(feature, COLON_DELIMITER)
			component := parts[0] + UNDERSCORE_DELIMITER + parts[1]
			componentList.Add(component)
		}
	}

	return componentList
}

func GetOnlineFeatureMapping(offlineFeatureList mapset.Set[string], token string) (map[string]string, error) {
	// Use internal handler instead of HTTP request
	handler := ofsHandler.NewConfigHandler(1)
	if handler == nil {
		return nil, fmt.Errorf("failed to initialize online feature store handler")
	}

	requestBody := ofsHandler.GetOnlineFeatureMappingRequest{
		OfflineFeatureList: offlineFeatureList.ToSlice(),
	}

	response, err := handler.GetOnlineFeatureMapping(requestBody)
	if err != nil {
		return nil, err
	}

	if response.Error != "" {
		return nil, fmt.Errorf("error from GetOnlineFeatureMapping: %s", response.Error)
	}

	// response.Data is already a map[string]string mapping offline to online features
	// Just return it directly
	return response.Data, nil
}

func GetFeatureComponents(request InferflowOnboardRequest, featureList mapset.Set[string], featureToDataType map[string]string, pcvrCalibrationFeatures mapset.Set[string], pctrCalibrationFeatures mapset.Set[string], etcdConfig etcd.Manager, token string, entityIDs map[string]bool) ([]FeatureComponent, error) {
	featureComponents := make([]FeatureComponent, 0, featureList.Cardinality()+pcvrCalibrationFeatures.Cardinality()+pctrCalibrationFeatures.Cardinality())

	featureComponentsMap := GetFeatureLabelToPrefixToFeatureGroupToFeatureMap(featureList.ToSlice())
	pcvrCalibrationFeaturesMap := GetFeatureLabelToPrefixToFeatureGroupToFeatureMap(pcvrCalibrationFeatures.ToSlice())
	pctrCalibrationFeaturesMap := GetFeatureLabelToPrefixToFeatureGroupToFeatureMap(pctrCalibrationFeatures.ToSlice())

	err := FillFeatureComponentFromComponentMap(request, featureComponentsMap, &featureComponents, token, etcdConfig)
	if err != nil {
		return nil, err
	}

	err = FillFeatureComponentFromComponentMap(request, pcvrCalibrationFeaturesMap, &featureComponents, token, etcdConfig)
	if err != nil {
		return nil, err
	}

	err = FillFeatureComponentFromComponentMap(request, pctrCalibrationFeaturesMap, &featureComponents, token, etcdConfig)
	if err != nil {
		return nil, err
	}

	compositeKeysGenerated := 0
	for _, ranker := range request.Payload.Rankers {
		if len(ranker.EntityID) == 1 {
			continue
		}
		entityID := strings.Join(ranker.EntityID, COLON_DELIMITER)
		if complete, ok := entityIDs[entityID]; complete && ok {
			continue
		}
		fsIds := make([]FSKey, 0, len(ranker.EntityID))
		for _, entityID := range ranker.EntityID {
			fsIds = append(fsIds, FSKey{
				Schema: entityID,
				Col:    entityID,
			})
		}
		featureComponents = append(featureComponents, FeatureComponent{
			Component:        COMPONENT_NAME_PREFIX + strconv.Itoa(compositeKeysGenerated+1),
			CompositeID:      len(ranker.EntityID) > 1,
			ComponentID:      entityID,
			CompCacheEnabled: false,
			FSKeys:           fsIds,
			FSRequest: &FSRequest{
				Label:         "dummy",
				FeatureGroups: []FSFeatureGroup{},
			},
			FSFlattenRespKeys: []string{"dummy"},
		})
		compositeKeysGenerated++
		entityIDs[entityID] = true
	}

	for _, reRanker := range request.Payload.ReRankers {
		if len(reRanker.EntityID) == 1 {
			continue
		}
		entityID := strings.Join(reRanker.EntityID, COLON_DELIMITER)
		if complete, ok := entityIDs[entityID]; complete && ok {
			continue
		}
		fsIds := make([]FSKey, 0, len(reRanker.EntityID))
		for _, entityID := range reRanker.EntityID {
			fsIds = append(fsIds, FSKey{
				Schema: entityID,
				Col:    entityID,
			})
		}
		featureComponents = append(featureComponents, FeatureComponent{
			Component:        COMPONENT_NAME_PREFIX + strconv.Itoa(compositeKeysGenerated+1),
			CompositeID:      len(reRanker.EntityID) > 1,
			ComponentID:      entityID,
			CompCacheEnabled: false,
			FSKeys:           fsIds,
			FSRequest: &FSRequest{
				Label:         "dummy",
				FeatureGroups: []FSFeatureGroup{},
			},
			FSFlattenRespKeys: []string{"dummy"},
		})
		entityIDs[entityID] = true
		compositeKeysGenerated++
	}

	return featureComponents, nil
}

func FillFeatureComponentFromComponentMap(request InferflowOnboardRequest, featureComponentsMap map[string]map[string]map[string]mapset.Set[string], featureComponents *[]FeatureComponent, token string, etcdConfig etcd.Manager) error {
	for label, prefixToFeatureGroupToFeatureMap := range featureComponentsMap {
		featureGroupDataTypeMap, err := GetFeatureGroupDataTypeMap(label, token)
		if err != nil {
			return err
		}
		for prefix, featureGroupToFeatureMap := range prefixToFeatureGroupToFeatureMap {
			componentName := label
			colNamePrefix := emptyResponse

			if prefix != emptyResponse {
				componentName = prefix + UNDERSCORE_DELIMITER + label
				colNamePrefix = prefix + COLON_DELIMITER
			}

			featureGroups := make([]FSFeatureGroup, 0, len(featureGroupToFeatureMap))

			for featureGroup, featureSet := range featureGroupToFeatureMap {
				featureGroupDataType := featureGroupDataTypeMap[featureGroup]
				featureGroupData := FSFeatureGroup{
					Label:    featureGroup,
					Features: featureSet.ToSlice(),
					DataType: featureGroupDataType,
				}
				featureGroups = append(featureGroups, featureGroupData)
			}

			componentData := etcdConfig.GetComponentData(componentName)
			if componentData == nil {
				return fmt.Errorf("Feature Components: componentData for '%s' not found in registry", componentName)
			}

			componentID := componentData.ComponentID
			overrideComponentID := ""
			if realEstate := request.Payload.RealEstate; realEstate != "" {
				if override, exists := componentData.Overridecomponent[realEstate]; exists {
					overrideComponentID = override.ComponentId
					componentID = override.ComponentId
				}
			}

			// Build FSKeys in a deterministic order. Since map iteration order is random in Go,
			// we first collect the map keys, sort them, and then append the entries in that order.
			idKeys := make([]string, 0, len(componentData.FSIdSchemaToValueColumns))
			for k := range componentData.FSIdSchemaToValueColumns {
				idKeys = append(idKeys, k)
			}
			sort.Strings(idKeys)
			fsKeys := make([]FSKey, 0, len(idKeys))
			for _, k := range idKeys {
				pair := componentData.FSIdSchemaToValueColumns[k]
				col := pair.ValueCol
				if overrideComponentID != "" {
					col = overrideComponentID
				}
				fsKeys = append(fsKeys, FSKey{
					Schema: pair.Schema,
					Col:    col,
				})
			}

			// Build FSFlattenRespKeys in deterministic order as well.
			flattenKeys := make([]string, 0, len(componentData.FSFlattenResKeys))
			for k := range componentData.FSFlattenResKeys {
				flattenKeys = append(flattenKeys, k)
			}
			sort.Strings(flattenKeys)
			fsFlattenRespKeys := make([]string, 0, len(flattenKeys))
			for _, k := range flattenKeys {
				fsFlattenRespKeys = append(fsFlattenRespKeys, componentData.FSFlattenResKeys[k])
			}

			featureComponent := FeatureComponent{
				Component:        componentName,
				ComponentID:      componentID,
				CompCacheEnabled: activeTrue,
				FSKeys:           fsKeys,
				FSRequest: &FSRequest{
					Label:         label,
					FeatureGroups: featureGroups,
				},
				FSFlattenRespKeys: fsFlattenRespKeys,
			}

			if prefix != emptyResponse {
				featureComponent.ColNamePrefix = colNamePrefix
			}

			*featureComponents = append(*featureComponents, featureComponent)
		}
	}
	return nil
}

func GetFeatureLabelToPrefixToFeatureGroupToFeatureMap(featureStrings []string) map[string]map[string]map[string]mapset.Set[string] {
	featuresMap := make(map[string]map[string]map[string]mapset.Set[string])

	if len(featureStrings) == 0 {
		return featuresMap
	}

	sort.Strings(featureStrings)

	for _, input := range featureStrings {
		parts := strings.Split(input, COLON_DELIMITER)
		if len(parts) != 3 && len(parts) != 4 {
			continue
		}

		var (
			prefix  string
			label   string
			group   string
			feature string
		)

		if len(parts) == 4 {
			prefix, label, group, feature = parts[0], parts[1], parts[2], parts[3]
		} else {
			prefix = ""
			label, group, feature = parts[0], parts[1], parts[2]
		}

		if _, ok := featuresMap[label]; !ok {
			featuresMap[label] = make(map[string]map[string]mapset.Set[string])
		}
		if _, ok := featuresMap[label][prefix]; !ok {
			featuresMap[label][prefix] = make(map[string]mapset.Set[string])
		}
		if _, ok := featuresMap[label][prefix][group]; !ok {
			featuresMap[label][prefix][group] = mapset.NewSet[string]()
		}

		featuresMap[label][prefix][group].Add(feature)
	}

	return featuresMap
}

func GetFeatureGroupDataTypeMap(label string, token string) (map[string]string, error) {
	featureGroupDataTypeMap := make(map[string]string)

	// Use internal handler instead of HTTP request
	ofsHandler := ofsHandler.NewConfigHandler(1)
	if ofsHandler == nil {
		return nil, fmt.Errorf("failed to initialize online feature store handler")
	}

	featureGroups, err := ofsHandler.RetrieveFeatureGroups(label)
	if err != nil {
		return nil, err
	}

	for _, fg := range *featureGroups {
		featureGroupDataTypeMap[fg.FeatureGroupLabel] = string(fg.DataType)
	}

	return featureGroupDataTypeMap, nil
}

func GetRTPComponents(request InferflowOnboardRequest, rtpFeatures mapset.Set[string], featureToDataTypeMap map[string]string, etcdConfig etcd.Manager, token string) ([]RTPComponent, error) {
	rtpComponents := make([]RTPComponent, 0)

	if rtpFeatures.Cardinality() == 0 {
		return rtpComponents, nil
	}

	featureDataTypeMap, err := GetRTPFeatureGroupDataTypeMap()
	if err != nil && inferflow.IsMeeshoEnabled {
		return rtpComponents, nil
	}
	rtpFeatureComponentsMap := GetRTPFeatureLabelToPrefixToFeatureGroupToFeatureMap(rtpFeatures.ToSlice())
	for label, prefixToFeatureGroupToFeatureMap := range rtpFeatureComponentsMap {
		if err != nil {
			return nil, err
		}

		for prefix, featureGroupToFeatureMap := range prefixToFeatureGroupToFeatureMap {
			componentName := label
			colNamePrefix := ""

			if prefix != "" {
				componentName = prefix + UNDERSCORE_DELIMITER + label
				colNamePrefix = prefix + COLON_DELIMITER
			}

			featureGroupsByDataType := make(map[string]map[string][]string) // [featureGroupName][dataType][]features

			for featureGroupName, featureSet := range featureGroupToFeatureMap {
				featureGroupsByDataType[featureGroupName] = make(map[string][]string)

				for _, feature := range featureSet.ToSlice() {
					featureDataType := featureDataTypeMap[strings.Join([]string{label, featureGroupName, feature}, COLON_DELIMITER)]
					if featureDataType == "" {
						return nil, fmt.Errorf("RTP Components: no data type found for feature %s", feature)
					}

					if featureGroupsByDataType[featureGroupName][featureDataType] == nil {
						featureGroupsByDataType[featureGroupName][featureDataType] = make([]string, 0)
					}
					featureGroupsByDataType[featureGroupName][featureDataType] = append(featureGroupsByDataType[featureGroupName][featureDataType], feature)
				}
			}

			featureGroups := make([]FSFeatureGroup, 0)
			for featureGroupName, dataTypeToFeatures := range featureGroupsByDataType {
				for dataType, features := range dataTypeToFeatures {
					sort.Strings(features)

					featureGroupData := FSFeatureGroup{
						Label:    featureGroupName,
						Features: features,
						DataType: dataType,
					}
					featureGroups = append(featureGroups, featureGroupData)
				}
			}

			componentData := etcdConfig.GetComponentData(componentName)
			if componentData == nil {
				return nil, fmt.Errorf("RTP Components: componentData for '%s' not found in registry", componentName)
			}

			componentID := componentData.ComponentID
			overrideComponentID := ""
			if realEstate := request.Payload.RealEstate; realEstate != "" {
				if override, exists := componentData.Overridecomponent[realEstate]; exists {
					overrideComponentID = override.ComponentId
					componentID = override.ComponentId
				}
			}

			idKeys := make([]string, 0, len(componentData.FSIdSchemaToValueColumns))
			for k := range componentData.FSIdSchemaToValueColumns {
				idKeys = append(idKeys, k)
			}
			sort.Strings(idKeys)
			fsKeys := make([]FSKey, 0, len(idKeys))
			for _, k := range idKeys {
				pair := componentData.FSIdSchemaToValueColumns[k]
				col := pair.ValueCol
				if overrideComponentID != "" {
					col = overrideComponentID
				}
				fsKeys = append(fsKeys, FSKey{
					Schema: pair.Schema,
					Col:    col,
				})
			}

			rtpComponent := RTPComponent{
				Component:         componentName,
				ComponentID:       componentID,
				CompCacheEnabled:  false,
				FSKeys:            fsKeys,
				FSFlattenRespKeys: []string{componentData.FSFlattenResKeys[label]},
				FeatureRequest: &FSRequest{
					Label:         label,
					FeatureGroups: featureGroups,
				},
				ColNamePrefix: colNamePrefix,
			}

			rtpComponents = append(rtpComponents, rtpComponent)
		}
	}

	return rtpComponents, nil
}

func GetRTPFeatureLabelToPrefixToFeatureGroupToFeatureMap(featureStrings []string) map[string]map[string]map[string]mapset.Set[string] {
	featuresMap := make(map[string]map[string]map[string]mapset.Set[string])

	if len(featureStrings) == 0 {
		return featuresMap
	}

	sort.Strings(featureStrings)

	for _, input := range featureStrings {
		parts := strings.Split(input, COLON_DELIMITER)
		if len(parts) != 3 && len(parts) != 4 {
			continue
		}

		var (
			prefix  string
			label   string
			group   string
			feature string
		)

		if len(parts) == 4 {
			prefix, label, group, feature = parts[0], parts[1], parts[2], parts[3]
		} else {
			prefix = ""
			label, group, feature = parts[0], parts[1], parts[2]
		}

		if _, ok := featuresMap[label]; !ok {
			featuresMap[label] = make(map[string]map[string]mapset.Set[string])
		}
		if _, ok := featuresMap[label][prefix]; !ok {
			featuresMap[label][prefix] = make(map[string]mapset.Set[string])
		}
		if _, ok := featuresMap[label][prefix][group]; !ok {
			featuresMap[label][prefix][group] = mapset.NewSet[string]()
		}

		featuresMap[label][prefix][group].Add(feature)
	}

	return featuresMap
}

func GetRTPFeatureGroupDataTypeMap() (map[string]string, error) {
	return externalcall.PricingClient.GetFeatureGroupDataTypeMap()
}

func GetPredatorComponents(request InferflowOnboardRequest, offlineToOnlineMapping map[string]string) ([]PredatorComponent, error) {
	predatorComponents := make([]PredatorComponent, 0, len(request.Payload.Rankers))

	for i, ranker := range request.Payload.Rankers {

		// validate routing config
		if len(ranker.RoutingConfig) > 0 {
			totalRoutingPercentage := float32(0)
			defaultEndPointFallback := false
			for _, route := range ranker.RoutingConfig {
				if route.ModelName == "" || route.ModelEndpoint == "" {
					return nil, fmt.Errorf("Predator Components: model name or model endpoint is missing for routing config")
				}
				if route.RoutingPercentage < 0 {
					return nil, fmt.Errorf("Predator Components: routing percentage is less than 0 for routing config")
				}
				totalRoutingPercentage += route.RoutingPercentage
				if route.ModelEndpoint == ranker.EndPoint {
					defaultEndPointFallback = true
				}
			}
			if defaultEndPointFallback {
				if totalRoutingPercentage != 100.0 {
					return nil, fmt.Errorf("Default endpoint included but total routing percentage is not 100")
				}
			} else {
				if totalRoutingPercentage > 100.0 {
					return nil, fmt.Errorf("Total routing percentage is greater than 100")
				}
			}
		}

		predatorComponent := PredatorComponent{
			Component:     "p" + strconv.Itoa(i+1),
			ComponentID:   strings.Join(ranker.EntityID, COLON_DELIMITER),
			ModelName:     ranker.ModelName,
			ModelEndPoint: ranker.EndPoint,
			Deadline:      ranker.Deadline,
			BatchSize:     ranker.BatchSize,
			Inputs:        make([]PredatorInput, 0, len(ranker.Inputs)),
			Outputs:       make([]PredatorOutput, 0, len(ranker.Outputs)),
			RoutingConfig: ranker.RoutingConfig,
		}

		if ranker.Calibration != "" {
			predatorComponent.Calibration = ranker.Calibration
		}

		for _, input := range ranker.Inputs {
			featureList, err := getPredatorInputFeaturesList(input.Features, offlineToOnlineMapping)
			if err != nil {
				return nil, err
			}
			predatorComponent.Inputs = append(predatorComponent.Inputs, PredatorInput{
				Name:     input.Name,
				Features: featureList,
				Dims:     input.Dims,
				DataType: input.DataType,
			})
		}
		for _, output := range ranker.Outputs {
			predatorComponent.Outputs = append(predatorComponent.Outputs, PredatorOutput{
				Name:            output.Name,
				ModelScores:     output.ModelScores,
				ModelScoresDims: output.ModelScoresDims,
				DataType:        output.DataType,
			})
		}
		predatorComponents = append(predatorComponents, predatorComponent)
	}
	return predatorComponents, nil
}

func getPredatorInputFeaturesList(features []string, offlineToOnlineMapping map[string]string) ([]string, error) {
	finalFeatures := make([]string, 0, len(features))

	for _, feature := range features {
		transformedFeature, featureType, err := transformFeature(feature)
		if err != nil {
			return nil, fmt.Errorf("Predator Input: error transforming feature %s: %w", feature, err)
		}

		var featureToAdd string
		switch featureType {
		case featureClassOffline:
			if onlineFeature, ok := offlineToOnlineMapping[transformedFeature]; ok {
				featureToAdd = onlineFeature
			} else {
				return nil, fmt.Errorf("Predator Input: offlineToOnlineMapping for '%s' not found", transformedFeature)
			}
		default:
			featureToAdd = transformedFeature
		}

		finalFeatures = append(finalFeatures, featureToAdd)
	}

	return finalFeatures, nil
}

func GetNumerixComponents(request InferflowOnboardRequest, offlineToOnlineMapping map[string]string, predatorAndNumerixOutputsToDataType map[string]string, featureToDataType map[string]string) ([]NumerixComponent, error) {
	NumerixComponents := make([]NumerixComponent, 0, len(request.Payload.ReRankers))
	for i, reRanker := range request.Payload.ReRankers {
		scoremap, err := getNumerixScoreMapping(reRanker.EqVariables, offlineToOnlineMapping, predatorAndNumerixOutputsToDataType, featureToDataType)
		if err != nil {
			return nil, err
		}
		NumerixComponent := NumerixComponent{
			Component:    "i" + strconv.Itoa(i+1),
			ComponentID:  strings.Join(reRanker.EntityID, COLON_DELIMITER),
			ScoreCol:     reRanker.Score,
			ComputeID:    strconv.Itoa(reRanker.EqID),
			ScoreMapping: scoremap,
			DataType:     reRanker.DataType,
		}
		NumerixComponents = append(NumerixComponents, NumerixComponent)
	}

	return NumerixComponents, nil
}

func getNumerixScoreMapping(eqVariables map[string]string, offlineToOnlineMapping map[string]string, predatorAndNumerixOutputsToDataType map[string]string, featureToDataType map[string]string) (map[string]string, error) {
	scoremap := make(map[string]string)
	for key, feature := range eqVariables {
		transformedFeature, featureType, err := transformFeature(feature)
		if err != nil {
			return nil, fmt.Errorf("Numerix Score Mapping: error transforming feature %s: %w", feature, err)
		}
		keyDataType := featureToDataType[transformedFeature]
		if keyDataType == "" {
			keyDataType = predatorAndNumerixOutputsToDataType[transformedFeature]
		}
		if !strings.Contains(keyDataType, "DataType") {
			key = key + "@DataType" + keyDataType
		} else {
			key = key + "@" + keyDataType
		}

		switch featureType {
		case featureClassOffline:
			if onlineFeature, ok := offlineToOnlineMapping[transformedFeature]; ok {
				scoremap[key] = onlineFeature
			} else {
				return nil, fmt.Errorf("Numerix Score Mapping: offlineToOnlineMapping for '%s' not found", transformedFeature)
			}
		case featureClassOnline, featureClassDefault, featureClassRtp:
			scoremap[key] = transformedFeature
		default:
			scoremap[key] = transformedFeature
		}

	}
	return scoremap, nil
}

func GetResponseConfigs(request *InferflowOnboardRequest) (*FinalResponseConfig, error) {
	responseConfigs := &FinalResponseConfig{
		LoggingPerc:          request.Payload.Response.PrismLoggingPerc,
		ModelSchemaPerc:      request.Payload.Response.RankerSchemaFeaturesInResponsePerc,
		Features:             request.Payload.Response.ResponseFeatures,
		LogSelectiveFeatures: request.Payload.Response.LogSelectiveFeatures,
		LogBatchSize:         request.Payload.Response.LogBatchSize,
	}

	return responseConfigs, nil
}

func GetComponentConfig(featureComponents []FeatureComponent, rtpComponents []RTPComponent, NumerixComponents []NumerixComponent, predatorComponents []PredatorComponent) (*ComponentConfig, error) {
	componentConfig := &ComponentConfig{
		CacheEnabled:       true,
		CacheTTL:           300,
		CacheVersion:       1,
		FeatureComponents:  featureComponents,
		RTPComponents:      rtpComponents,
		NumerixComponents:  NumerixComponents,
		PredatorComponents: predatorComponents,
	}

	return componentConfig, nil
}

func GetDagExecutionConfig(request InferflowOnboardRequest, featureComponents []FeatureComponent, rtpComponents []RTPComponent, NumerixComponents []NumerixComponent, predatorComponents []PredatorComponent, etcdConfig etcd.Manager) (*DagExecutionConfig, error) {
	dagExecutionConfig := &DagExecutionConfig{
		ComponentDependency: make(map[string][]string),
	}

	for _, component := range featureComponents {
		componentName := component.Component

		specificDependencies := findSpecificFeatureDependencies(component, featureComponents, rtpComponents)

		if len(specificDependencies) > 0 {
			dagExecutionConfig.ComponentDependency[componentName] = append(dagExecutionConfig.ComponentDependency[componentName], specificDependencies...)
		} else {
			dagExecutionConfig.ComponentDependency[componentName] = append(dagExecutionConfig.ComponentDependency[componentName], FEATURE_INITIALIZER)
		}
	}

	for _, component := range rtpComponents {
		componentName := component.Component

		specificDependencies := findSpecificRTPDependencies(component, rtpComponents, featureComponents)

		if len(specificDependencies) > 0 {
			dagExecutionConfig.ComponentDependency[componentName] = append(dagExecutionConfig.ComponentDependency[componentName], specificDependencies...)
		} else {
			dagExecutionConfig.ComponentDependency[componentName] = append(dagExecutionConfig.ComponentDependency[componentName], FEATURE_INITIALIZER)
		}
	}

	for _, component := range predatorComponents {
		componentName := component.Component

		specificDependencies := findSpecificPredatorDependencies(component, featureComponents, rtpComponents, NumerixComponents)

		if len(specificDependencies) > 0 {
			dagExecutionConfig.ComponentDependency[componentName] = append(dagExecutionConfig.ComponentDependency[componentName], specificDependencies...)
		} else {
			dagExecutionConfig.ComponentDependency[componentName] = append(dagExecutionConfig.ComponentDependency[componentName], FEATURE_INITIALIZER)
		}
	}

	for _, component := range NumerixComponents {
		componentName := component.Component

		specificDependencies := findSpecificNumerixDependencies(component, featureComponents, rtpComponents, predatorComponents, NumerixComponents)

		if len(specificDependencies) > 0 {
			dagExecutionConfig.ComponentDependency[componentName] = append(dagExecutionConfig.ComponentDependency[componentName], specificDependencies...)
		} else {
			dagExecutionConfig.ComponentDependency[componentName] = append(dagExecutionConfig.ComponentDependency[componentName], FEATURE_INITIALIZER)
		}
	}

	addPredatorDependencies(predatorComponents, dagExecutionConfig)

	return dagExecutionConfig, nil
}

func findSpecificFeatureDependencies(featureComp FeatureComponent, featureComponents []FeatureComponent, rtpComponents []RTPComponent) []string {
	var dependencies []string

	requiredInputs := make(map[string]struct{})
	completedComponents := make(map[string]bool)
	for _, key := range featureComp.FSKeys {
		requiredInputs[key.Col] = struct{}{}
	}

	for _, otherComp := range featureComponents {
		if done, ok := completedComponents[otherComp.Component]; ok && done {
			continue
		}
		compPrefix := otherComp.ColNamePrefix
		for _, featureGroup := range otherComp.FSRequest.FeatureGroups {
			for _, feature := range featureGroup.Features {
				featureKey := compPrefix + otherComp.FSRequest.Label + COLON_DELIMITER + featureGroup.Label + COLON_DELIMITER + feature
				if _, required := requiredInputs[featureKey]; required && !completedComponents[otherComp.Component] {
					dependencies = append(dependencies, otherComp.Component)
					completedComponents[otherComp.Component] = true
					break
				}
			}
		}
	}

	for _, rtpComp := range rtpComponents {
		if done, ok := completedComponents[rtpComp.Component]; ok && done {
			continue
		}
		colNamePrefix := rtpComp.ColNamePrefix
		for _, featureGroup := range rtpComp.FeatureRequest.FeatureGroups {
			for _, feature := range featureGroup.Features {
				featureKey := colNamePrefix + rtpComp.FeatureRequest.Label + COLON_DELIMITER + featureGroup.Label + COLON_DELIMITER + feature
				if _, required := requiredInputs[featureKey]; required && !completedComponents[rtpComp.Component] {
					dependencies = append(dependencies, rtpComp.Component)
					completedComponents[rtpComp.Component] = true
					break
				}
			}
		}
	}

	return dependencies
}

func findSpecificRTPDependencies(rtpComp RTPComponent, rtpComponents []RTPComponent, featureComponents []FeatureComponent) []string {
	var dependencies []string
	completedComponents := make(map[string]bool)
	requiredInputs := make(map[string]struct{})
	for _, key := range rtpComp.FSKeys {
		requiredInputs[key.Col] = struct{}{}
	}

	for _, otherComp := range rtpComponents {
		if done, ok := completedComponents[otherComp.Component]; ok && done {
			continue
		}
		colNamePrefix := otherComp.ColNamePrefix
		for _, featureGroup := range otherComp.FeatureRequest.FeatureGroups {
			for _, feature := range featureGroup.Features {
				featureKey := colNamePrefix + otherComp.FeatureRequest.Label + COLON_DELIMITER + featureGroup.Label + COLON_DELIMITER + feature
				if _, required := requiredInputs[featureKey]; required && !completedComponents[otherComp.Component] {
					dependencies = append(dependencies, otherComp.Component)
					completedComponents[otherComp.Component] = true
					break
				}
			}
		}
	}

	for _, featureComp := range featureComponents {
		if done, ok := completedComponents[featureComp.Component]; ok && done {
			continue
		}
		colNamePrefix := featureComp.ColNamePrefix
		for _, featureGroup := range featureComp.FSRequest.FeatureGroups {
			for _, feature := range featureGroup.Features {
				featureKey := colNamePrefix + featureComp.FSRequest.Label + COLON_DELIMITER + featureGroup.Label + COLON_DELIMITER + feature
				if _, required := requiredInputs[featureKey]; required && !completedComponents[featureComp.Component] {
					dependencies = append(dependencies, featureComp.Component)
					completedComponents[featureComp.Component] = true
					break
				}
			}
		}
	}
	return dependencies
}

func findSpecificPredatorDependencies(predatorComp PredatorComponent, featureComponents []FeatureComponent, rtpComponents []RTPComponent, NumerixComponents []NumerixComponent) []string {
	var dependencies []string

	requiredInputs := make(map[string]struct{})
	completedComponents := make(map[string]bool)
	for _, input := range predatorComp.Inputs {
		for _, feature := range input.Features {
			requiredInputs[feature] = struct{}{}
		}
	}

	for _, featureComp := range featureComponents {
		if done, ok := completedComponents[featureComp.Component]; ok && done {
			continue
		}
		colNamePrefix := featureComp.ColNamePrefix
		for _, featureGroup := range featureComp.FSRequest.FeatureGroups {
			for _, feature := range featureGroup.Features {
				featureKey := colNamePrefix + featureComp.FSRequest.Label + COLON_DELIMITER + featureGroup.Label + COLON_DELIMITER + feature
				if _, required := requiredInputs[featureKey]; required && !completedComponents[featureComp.Component] {
					dependencies = append(dependencies, featureComp.Component)
					completedComponents[featureComp.Component] = true
					break
				}
			}
		}
	}

	for _, rtpComp := range rtpComponents {
		if done, ok := completedComponents[rtpComp.Component]; ok && done {
			continue
		}
		colNamePrefix := rtpComp.ColNamePrefix
		for _, featureGroup := range rtpComp.FeatureRequest.FeatureGroups {
			for _, feature := range featureGroup.Features {
				featureKey := colNamePrefix + rtpComp.FeatureRequest.Label + COLON_DELIMITER + featureGroup.Label + COLON_DELIMITER + feature
				if _, required := requiredInputs[featureKey]; required && !completedComponents[rtpComp.Component] {
					dependencies = append(dependencies, rtpComp.Component)
					completedComponents[rtpComp.Component] = true
					break
				}
			}
		}
	}

	for _, NumerixComp := range NumerixComponents {
		if _, required := requiredInputs[NumerixComp.ScoreCol]; required && !completedComponents[NumerixComp.Component] {
			completedComponents[NumerixComp.Component] = true
			dependencies = append(dependencies, NumerixComp.Component)
			continue
		}
	}

	return dependencies
}

func findSpecificNumerixDependencies(NumerixComp NumerixComponent, featureComponents []FeatureComponent, rtpComponents []RTPComponent, predatorComponents []PredatorComponent, NumerixComponents []NumerixComponent) []string {
	var dependencies []string

	requiredInputs := make(map[string]struct{})
	for _, input := range NumerixComp.ScoreMapping {
		requiredInputs[input] = struct{}{}
	}

	completedComponents := make(map[string]bool)

	for _, featureComp := range featureComponents {
		if done, ok := completedComponents[featureComp.Component]; ok && done {
			continue
		}
		colNamePrefix := featureComp.ColNamePrefix
		for _, featureGroup := range featureComp.FSRequest.FeatureGroups {
			for _, feature := range featureGroup.Features {
				featureKey := colNamePrefix + featureComp.FSRequest.Label + COLON_DELIMITER + featureGroup.Label + COLON_DELIMITER + feature
				if _, required := requiredInputs[featureKey]; required && !completedComponents[featureComp.Component] {
					dependencies = append(dependencies, featureComp.Component)
					completedComponents[featureComp.Component] = true
					break
				}
			}
		}
	}

	for _, rtpComp := range rtpComponents {
		if done, ok := completedComponents[rtpComp.Component]; ok && done {
			continue
		}
		colNamePrefix := rtpComp.ColNamePrefix
		for _, featureGroup := range rtpComp.FeatureRequest.FeatureGroups {
			for _, feature := range featureGroup.Features {
				featureKey := colNamePrefix + rtpComp.FeatureRequest.Label + COLON_DELIMITER + featureGroup.Label + COLON_DELIMITER + feature
				if _, required := requiredInputs[featureKey]; required && !completedComponents[rtpComp.Component] {
					dependencies = append(dependencies, rtpComp.Component)
					completedComponents[rtpComp.Component] = true
					break
				}
			}
		}
	}

	for _, predatorComp := range predatorComponents {
		if done, ok := completedComponents[predatorComp.Component]; ok && done {
			continue
		}
		for _, output := range predatorComp.Outputs {
			for _, modelScore := range output.ModelScores {
				if _, required := requiredInputs[modelScore]; required && !completedComponents[predatorComp.Component] {
					dependencies = append(dependencies, predatorComp.Component)
					completedComponents[predatorComp.Component] = true
					break
				}
			}
		}
	}

	for _, i := range NumerixComponents {
		if done, ok := completedComponents[NumerixComp.Component]; ok && done {
			continue
		}
		if _, required := requiredInputs[i.ScoreCol]; required && !completedComponents[i.Component] {
			dependencies = append(dependencies, i.Component)
			completedComponents[i.Component] = true
		}
	}

	return dependencies
}

func addPredatorDependencies(predatorComponents []PredatorComponent, dagExecutionConfig *DagExecutionConfig) {
	// Build a map of what each predator produces
	prod := make(map[string]map[string]struct{})
	for _, p := range predatorComponents {
		s := make(map[string]struct{})
		for _, o := range p.Outputs {
			s[o.Name] = struct{}{}
			for _, ms := range o.ModelScores {
				s[ms] = struct{}{}
			}
		}
		prod[p.Component] = s
	}

	for _, p2 := range predatorComponents {
		inputSet := make(map[string]struct{})
		for _, in := range p2.Inputs {
			for _, f := range in.Features {
				inputSet[f] = struct{}{}
			}
		}
		for p1Name, outSet := range prod {
			if p1Name == p2.Component {
				continue
			}
			for f := range inputSet {
				if _, ok := outSet[f]; ok {
					dagExecutionConfig.ComponentDependency[p2.Component] = append(dagExecutionConfig.ComponentDependency[p2.Component], p1Name)
					break
				}
			}
		}
	}
}
