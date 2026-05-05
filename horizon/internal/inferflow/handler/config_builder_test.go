package handler

import (
	"testing"

	"github.com/Meesho/BharatMLStack/horizon/internal/inferflow/etcd"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEtcdManager struct {
	components map[string]*etcd.ComponentData
}

func (f *mockEtcdManager) GetComponentData(componentName string) *etcd.ComponentData {
	if f.components == nil {
		return nil
	}
	return f.components[componentName]
}

func (f *mockEtcdManager) CreateConfig(serviceName string, ConfigId string, InferflowConfig etcd.InferflowConfig) error {
	return nil
}

func (f *mockEtcdManager) UpdateConfig(serviceName string, ConfigId string, InferflowConfig etcd.InferflowConfig) error {
	return nil
}

func (f *mockEtcdManager) DeleteConfig(serviceName string, ConfigId string) error {
	return nil
}

func (f *mockEtcdManager) GetConfiguredEndpoints(serviceDeployableName string) mapset.Set[string] {
	return mapset.NewSet[string]()
}

func TestExtractEntityIDs(t *testing.T) {
	request := InferflowOnboardRequest{
		Payload: OnboardPayload{
			Rankers: []Ranker{
				{EntityID: []string{"user", "item"}},
				{EntityID: []string{"user"}},
			},
			ReRankers: []ReRanker{
				{EntityID: []string{"session"}},
			},
		},
	}
	got := extractEntityIDs(request)
	// extractEntityIDs sets each entityID to false; keys indicate which entity IDs were seen
	assert.Contains(t, got, "user:item")
	assert.Contains(t, got, "user")
	assert.Contains(t, got, "session")
	assert.Len(t, got, 3)
	assert.False(t, got["user:item"])
	assert.False(t, got["user"])
	assert.False(t, got["session"])
}

func TestExtractEntityIDs_Empty(t *testing.T) {
	request := InferflowOnboardRequest{Payload: OnboardPayload{}}
	got := extractEntityIDs(request)
	assert.Empty(t, got)
}

func TestTransformFeature(t *testing.T) {
	tests := []struct {
		name        string
		feature     string
		wantFeature string
		wantType    string
		wantErr     bool
	}{
		{"invalid - single part", "onlyone", "", featureClassInvalid, true},
		{"default feature", "DEFAULT|foo", "foo", featureClassDefault, false},
		{"model feature", "MODEL|bar", "bar", featureClassModel, false},
		{"online feature", "ONLINE|baz", "baz", featureClassOnline, false},
		{"offline feature", "OFFLINE|qux", "qux", featureClassOffline, false},
		{"pctr_calibration", "PCTR_CALIBRATION|pctr", "pctr_calibration:pctr", featureClassPCTRCalibration, false},
		{"pcvr_calibration", "PCVR_CALIBRATION|pcvr", "pcvr_calibration:pcvr", featureClassPCVRCalibration, false},
		{"parent default", "PARENT_DEFAULT_FEATURE|pf", "parent:pf", featureClassDefault, false},
		{"parent online", "PARENT_ONLINE_FEATURE|po", "parent:po", featureClassOnline, false},
		{"fallback default", "UNKNOWN|x", "x", featureClassDefault, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFeature, gotType, err := transformFeature(tt.feature)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantFeature, gotFeature)
			assert.Equal(t, tt.wantType, gotType)
		})
	}
}

func TestAddFeatureToSet(t *testing.T) {
	defaultFeatures := mapset.NewSet[string]()
	modelFeatures := mapset.NewSet[string]()
	onlineFeatures := mapset.NewSet[string]()
	offlineFeatures := mapset.NewSet[string]()
	pctrCalibrationFeatures := mapset.NewSet[string]()
	pcvrCalibrationFeatures := mapset.NewSet[string]()

	err := AddFeatureToSet(&defaultFeatures, &modelFeatures, &onlineFeatures, &offlineFeatures, &pctrCalibrationFeatures, &pcvrCalibrationFeatures, "f1", featureClassDefault)
	require.NoError(t, err)
	assert.True(t, defaultFeatures.Contains("f1"))

	err = AddFeatureToSet(&defaultFeatures, &modelFeatures, &onlineFeatures, &offlineFeatures, &pctrCalibrationFeatures, &pcvrCalibrationFeatures, "f2", featureClassOnline)
	require.NoError(t, err)
	assert.True(t, onlineFeatures.Contains("f2"))

	// duplicate in same set is allowed (Add is idempotent); adding to different set with same feature name should error
	err = AddFeatureToSet(&defaultFeatures, &modelFeatures, &onlineFeatures, &offlineFeatures, &pctrCalibrationFeatures, &pcvrCalibrationFeatures, "f1", featureClassModel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// invalid feature type
	err = AddFeatureToSet(&defaultFeatures, &modelFeatures, &onlineFeatures, &offlineFeatures, &pctrCalibrationFeatures, &pcvrCalibrationFeatures, "x", "invalid_type")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid feature type")
}

func TestGetComponentList(t *testing.T) {
	features := mapset.NewSet[string]()
	features.Add("parent_label:group1:feat1")
	features.Add("online:group2:feat2")
	pctr := mapset.NewSet[string]()
	pctr.Add("pctr_calibration:label1:g1")
	pcvr := mapset.NewSet[string]()
	pcvr.Add("pcvr_calibration:label2:g2")

	got := getComponentList(features, pctr, pcvr)
	assert.True(t, got.Contains("parent_label"))
	assert.True(t, got.Contains("online"))
	assert.True(t, got.Contains("pctr_calibration_label1"))
	assert.True(t, got.Contains("pcvr_calibration_label2"))
}

func TestGetComponentList_Empty(t *testing.T) {
	got := getComponentList(mapset.NewSet[string](), nil, nil)
	assert.True(t, got.IsEmpty())
}

func TestGetComponentDataOrDefault_MissingComponent_UsesDefaults(t *testing.T) {
	manager := &mockEtcdManager{
		components: map[string]*etcd.ComponentData{},
	}

	got := getComponentDataOrDefault(manager, "campaign")
	require.NotNil(t, got)

	assert.Equal(t, "campaign_id", got.ComponentID)
	assert.False(t, got.CompositeID)
	assert.Equal(t, FEATURE_INITIALIZER, got.ExecutionDependency)
	assert.Equal(t, map[string]string{"0": "campaign_id"}, got.FSFlattenResKeys)
	require.Contains(t, got.FSIdSchemaToValueColumns, "0")
	assert.Equal(t, etcd.FSIdSchemaToValueColumnPair{
		Schema:   "campaign_id",
		ValueCol: "campaign_id",
		DataType: "FP32",
	}, got.FSIdSchemaToValueColumns["0"])
	assert.Empty(t, got.Overridecomponent)
}

func TestGetComponentDataOrDefault_ExistingComponent_PreservesDataAndInitializesMaps(t *testing.T) {
	manager := &mockEtcdManager{
		components: map[string]*etcd.ComponentData{
			"catalog": {
				ComponentID:         "catalog_id",
				ExecutionDependency: "upstream_component",
				FSFlattenResKeys:    nil,
				FSIdSchemaToValueColumns: map[string]etcd.FSIdSchemaToValueColumnPair{
					"0": {
						Schema:   "catalog_id",
						ValueCol: "catalog_id",
						DataType: "INT64",
					},
				},
				Overridecomponent: nil,
			},
		},
	}

	got := getComponentDataOrDefault(manager, "catalog")
	require.NotNil(t, got)

	assert.Equal(t, "catalog_id", got.ComponentID)
	assert.Equal(t, "upstream_component", got.ExecutionDependency)
	require.NotNil(t, got.FSFlattenResKeys)
	require.NotNil(t, got.FSIdSchemaToValueColumns)
	require.NotNil(t, got.Overridecomponent)
	assert.Equal(t, "INT64", got.FSIdSchemaToValueColumns["0"].DataType)
}

func TestGetResponseConfigs(t *testing.T) {
	request := &InferflowOnboardRequest{
		Payload: OnboardPayload{
			Response: ResponseConfig{
				PrismLoggingPerc:                   10,
				RankerSchemaFeaturesInResponsePerc: 20,
				ResponseFeatures:                   []string{"f1"},
				LogSelectiveFeatures:               true,
				LogBatchSize:                       100,
				LoggingTTL:                         30,
			},
		},
	}
	got, err := GetResponseConfigs(request)
	require.NoError(t, err)
	assert.Equal(t, 10, got.LoggingPerc)
	assert.Equal(t, 20, got.ModelSchemaPerc)
	assert.Equal(t, []string{"f1"}, got.Features)
	assert.True(t, got.LogSelectiveFeatures)
	assert.Equal(t, 100, got.LogBatchSize)
	assert.Equal(t, 30, got.LoggingTTL)
}

func TestGetComponentConfig(t *testing.T) {
	featureComponents := []FeatureComponent{{Component: "fc1"}}
	rtpComponents := []RTPComponent{}
	seenScoreComponents := []SeenScoreComponent{}
	numerixComponents := []NumerixComponent{{Component: "i1"}}
	predatorComponents := []PredatorComponent{{Component: "p1"}}

	got, err := GetComponentConfig(featureComponents, rtpComponents, seenScoreComponents, numerixComponents, predatorComponents)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.CacheEnabled)
	assert.Equal(t, 300, got.CacheTTL)
	assert.Equal(t, 1, got.CacheVersion)
	assert.Len(t, got.FeatureComponents, 1)
	assert.Len(t, got.PredatorComponents, 1)
	assert.Len(t, got.NumerixComponents, 1)
}

func TestGetPredatorComponents_Simple(t *testing.T) {
	request := InferflowOnboardRequest{
		Payload: OnboardPayload{
			Rankers: []Ranker{
				{
					ModelName: "m1",
					EndPoint:  "ep1",
					EntityID:  []string{"user"},
					Inputs:    []Input{{Name: "in1", Features: []string{"ONLINE|f1"}}},
					Outputs:   []Output{{Name: "out1", ModelScores: []string{"score1"}, ModelScoresDims: [][]int{{1}}, DataType: "Float"}},
				},
			},
		},
	}
	offlineToOnlineMapping := map[string]string{}

	got, err := GetPredatorComponents(request, offlineToOnlineMapping)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "p1", got[0].Component)
	assert.Equal(t, "user", got[0].ComponentID)
	assert.Equal(t, "m1", got[0].ModelName)
	assert.Equal(t, "ep1", got[0].ModelEndPoint)
	assert.Len(t, got[0].Inputs, 1)
	assert.Len(t, got[0].Outputs, 1)
}

func TestGetPredatorComponents_RoutingConfig_Invalid(t *testing.T) {
	request := InferflowOnboardRequest{
		Payload: OnboardPayload{
			Rankers: []Ranker{
				{
					ModelName:     "m1",
					EndPoint:      "ep1",
					EntityID:      []string{"user"},
					RoutingConfig: []RoutingConfig{{ModelName: "", ModelEndpoint: "ep"}},
				},
			},
		},
	}
	_, err := GetPredatorComponents(request, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "routing config")
}

func TestGetFeatureLabelToPrefixToFeatureGroupToFeatureMap(t *testing.T) {
	features := []string{
		"label1:group1:feat1",
		"label1:group1:feat2",
		"parent_label:group2:feat3",
	}
	got := GetFeatureLabelToPrefixToFeatureGroupToFeatureMap(features)
	require.Contains(t, got, "label1")
	require.Contains(t, got["label1"], "")
	assert.True(t, got["label1"][""]["group1"].Contains("feat1"))
	assert.True(t, got["label1"][""]["group1"].Contains("feat2"))
	require.Contains(t, got, "parent_label")
	assert.True(t, got["parent_label"][""]["group2"].Contains("feat3"))
}

func TestGetFeatureLabelToPrefixToFeatureGroupToFeatureMap_Empty(t *testing.T) {
	got := GetFeatureLabelToPrefixToFeatureGroupToFeatureMap(nil)
	assert.Empty(t, got)
	got = GetFeatureLabelToPrefixToFeatureGroupToFeatureMap([]string{})
	assert.Empty(t, got)
}

func TestGetFeatureLabelToPrefixToFeatureGroupToFeatureMap_SkipsInvalidParts(t *testing.T) {
	features := []string{
		"onlytwo",
		"a:b:c:d", // 4 parts -> prefix, label, group, feature
	}
	got := GetFeatureLabelToPrefixToFeatureGroupToFeatureMap(features)
	// "onlytwo" has 1 part after split by :, skipped (len != 3 && != 4)
	assert.Contains(t, got, "b") // 4 parts: prefix=a, label=b, group=c, feature=d
	assert.True(t, got["b"]["a"]["c"].Contains("d"))
}

func TestGetNumerixComponents_Simple(t *testing.T) {
	request := InferflowOnboardRequest{
		Payload: OnboardPayload{
			ReRankers: []ReRanker{
				{
					EqID:     1,
					Score:    "score1",
					DataType: "Float",
					EntityID: []string{"user"},
					EqVariables: map[string]string{
						"x": "ONLINE|feat1",
					},
				},
			},
		},
	}
	offlineMapping := map[string]string{}
	predatorOutputsToDataType := map[string]string{}
	featureToDataType := map[string]string{"feat1": "Float"}

	got, err := GetNumerixComponents(request, offlineMapping, predatorOutputsToDataType, featureToDataType)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "i1", got[0].Component)
	assert.Equal(t, "score1", got[0].ScoreCol)
	assert.Equal(t, "1", got[0].ComputeID)
	assert.Contains(t, got[0].ScoreMapping, "x@DataTypeFloat")
}

func TestGetNumerixScoreMapping_OfflineNotFound(t *testing.T) {
	eqVariables := map[string]string{"k": "OFFLINE|off_feat"}
	offlineMapping := map[string]string{}                       // no mapping for off_feat
	featureToDataType := map[string]string{"off_feat": "Float"} // set so we reach offline-mapping check
	_, err := getNumerixScoreMapping(eqVariables, offlineMapping, nil, featureToDataType)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "offlineToOnlineMapping")
}

func TestGetNumerixScoreMapping_DataTypeNotFound(t *testing.T) {
	eqVariables := map[string]string{"k": "ONLINE|feat1"}
	featureToDataType := map[string]string{} // no dtype for feat1
	predatorOutputsToDataType := map[string]string{}
	_, err := getNumerixScoreMapping(eqVariables, nil, predatorOutputsToDataType, featureToDataType)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data type")
}

func TestGetPredatorInputFeaturesList_InvalidFeature(t *testing.T) {
	_, err := getPredatorInputFeaturesList([]string{"invalid_single_part"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transforming feature")
}

func TestGetPredatorInputFeaturesList_OfflineMapping(t *testing.T) {
	features := []string{"OFFLINE|off_feat"}
	mapping := map[string]string{"off_feat": "online_feat"}
	got, err := getPredatorInputFeaturesList(features, mapping)
	require.NoError(t, err)
	assert.Equal(t, []string{"online_feat"}, got)
}

func TestExtractFeatures(t *testing.T) {
	entityIDs := map[string]bool{"user:item": true}
	request := InferflowOnboardRequest{
		Payload: OnboardPayload{
			Rankers: []Ranker{
				{
					Inputs:  []Input{{Features: []string{"ONLINE|f1", "DEFAULT|f2"}}},
					Outputs: []Output{{Name: "o1", ModelScores: []string{"ms1"}, DataType: "Float"}},
				},
			},
		},
	}
	features, featureToDataType, predatorOutputsToDataType := extractFeatures(request, entityIDs)
	assert.True(t, features.Contains("ONLINE|f1"))
	assert.True(t, features.Contains("DEFAULT|f2"))
	assert.Contains(t, featureToDataType, "ONLINE|f1")
	assert.Contains(t, predatorOutputsToDataType, "ms1")
}
