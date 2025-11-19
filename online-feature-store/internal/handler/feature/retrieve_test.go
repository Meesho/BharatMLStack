package feature

import (
	"fmt"
	"testing"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config/enums"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/ds"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
	"github.com/stretchr/testify/assert"
)

func TestPreProcessFGs(t *testing.T) {
	tests := []struct {
		name           string
		retrieveData   *RetrieveData
		setupMock      func(*config.MockConfigManager)
		expectedError  string
		validateResult func(*testing.T, *RetrieveData)
	}{
		{
			name: "Happy path - single feature group",
			retrieveData: &RetrieveData{
				EntityLabel: "user",
				Query: &retrieve.Query{
					EntityLabel: "user",
					FeatureGroups: []*retrieve.FeatureGroup{
						{
							Label:         "user_profile",
							FeatureLabels: []string{"age", "gender"},
						},
					},
				},
				Result: &retrieve.Result{},
			},
			setupMock: func(m *config.MockConfigManager) {
				// Mock GetEntity
				m.On("GetEntity", "user").Return(&config.Entity{
					Label: "user",
					FeatureGroups: map[string]config.FeatureGroup{
						"user_profile": {
							Id:       1,
							DataType: enums.DataTypeInt32,
						},
					},
				}, nil)

				// Mock GetActiveFeatureSchema
				m.On("GetActiveFeatureSchema", "user", "user_profile").Return(&config.FeatureSchema{
					FeatureMeta: map[string]config.FeatureMeta{
						"age": {
							Sequence:             0,
							DefaultValuesInBytes: []byte{0, 0, 0, 1},
						},
						"gender": {
							Sequence:             1,
							DefaultValuesInBytes: []byte{0, 0, 0, 8},
						},
					},
				}, nil)

				// Mock GetFeatureGroup
				m.On("GetFeatureGroup", "user", "user_profile").Return(&config.FeatureGroup{
					Id:       1,
					DataType: enums.DataTypeInt32,
				}, nil)
			},
			validateResult: func(t *testing.T, rd *RetrieveData) {
				// Validate FeatureSchemas
				assert.Len(t, rd.Result.FeatureSchemas, 1, "Should have one feature schema")
				schema := rd.Result.FeatureSchemas[0]
				assert.Equal(t, "user_profile", schema.FeatureGroupLabel)
				assert.Len(t, schema.Features, 2, "Should have two features")

				// Validate feature ordering and column indices
				features := make(map[string]int32)
				for _, f := range schema.Features {
					features[f.Label] = f.ColumnIdx
				}
				assert.Contains(t, features, "age")
				assert.Contains(t, features, "gender")
				assert.True(t, features["age"] < features["gender"], "Age should have lower column index")

				// Validate ReqFGIds
				assert.Equal(t, 1, rd.ReqFGIds.Size(), "Should have one feature group ID")
				assert.True(t, rd.ReqFGIds.Has(1), "Should contain feature group ID 1")

				// Validate ReqFGIdToFeatureLabels
				featureLabels := rd.ReqFGIdToFeatureLabels[1]
				assert.Equal(t, 2, featureLabels.Size(), "Should have two feature labels")
				assert.True(t, featureLabels.Has("age"))
				assert.True(t, featureLabels.Has("gender"))

				// Validate column indices in metadata
				ageIdx := featureLabels.GetMeta("age").(int)
				genderIdx := featureLabels.GetMeta("gender").(int)
				assert.Equal(t, int(features["age"]), ageIdx)
				assert.Equal(t, int(features["gender"]), genderIdx)

				// Validate total column count
				assert.Equal(t, 2, rd.ReqColumnCount, "Should have two columns")
			},
		},
		{
			name: "Multiple feature groups with different data types",
			retrieveData: &RetrieveData{
				EntityLabel: "user",
				Query: &retrieve.Query{
					EntityLabel: "user",
					FeatureGroups: []*retrieve.FeatureGroup{
						{
							Label:         "user_profile",
							FeatureLabels: []string{"age"},
						},
						{
							Label:         "user_preferences",
							FeatureLabels: []string{"favorite_color", "interests"},
						},
					},
				},
				Result: &retrieve.Result{},
			},
			setupMock: func(m *config.MockConfigManager) {
				// Mock GetEntity
				m.On("GetEntity", "user").Return(&config.Entity{
					Label: "user",
					FeatureGroups: map[string]config.FeatureGroup{
						"user_profile": {
							Id:       1,
							DataType: enums.DataTypeInt32,
						},
						"user_preferences": {
							Id:       2,
							DataType: enums.DataTypeInt32,
						},
					},
				}, nil)

				// Mock schemas for both feature groups
				m.On("GetActiveFeatureSchema", "user", "user_profile").Return(&config.FeatureSchema{
					FeatureMeta: map[string]config.FeatureMeta{
						"age": {
							Sequence:             0,
							DefaultValuesInBytes: []byte{0, 0, 0, 1},
						},
					},
				}, nil)

				m.On("GetActiveFeatureSchema", "user", "user_preferences").Return(&config.FeatureSchema{
					FeatureMeta: map[string]config.FeatureMeta{
						"favorite_color": {
							Sequence:             0,
							DefaultValuesInBytes: []byte{0, 0, 0, 8},
						},
						"interests": {
							Sequence:             1,
							DefaultValuesInBytes: []byte{0, 0, 0, 16},
						},
					},
				}, nil)

				// Mock GetFeatureGroup for both groups
				m.On("GetFeatureGroup", "user", "user_profile").Return(&config.FeatureGroup{
					Id:       1,
					DataType: enums.DataTypeInt32,
				}, nil)

				m.On("GetFeatureGroup", "user", "user_preferences").Return(&config.FeatureGroup{
					Id:       2,
					DataType: enums.DataTypeInt32,
				}, nil)
			},
			validateResult: func(t *testing.T, rd *RetrieveData) {
				// Validate feature schemas
				assert.Len(t, rd.Result.FeatureSchemas, 2, "Should have two feature schemas")

				// Validate ReqFGIds
				assert.Equal(t, 2, rd.ReqFGIds.Size(), "Should have two feature group IDs")
				assert.True(t, rd.ReqFGIds.Has(1))
				assert.True(t, rd.ReqFGIds.Has(2))

				// Validate column indices are sequential
				totalFeatures := 0
				for _, schema := range rd.Result.FeatureSchemas {
					totalFeatures += len(schema.Features)
					for _, feature := range schema.Features {
						assert.Less(t, feature.ColumnIdx, int32(totalFeatures),
							"Column index should be less than total features")
					}
				}

				// Validate total column count
				assert.Equal(t, 3, rd.ReqColumnCount, "Should have three columns total")
			},
		},
		{
			name: "Error - feature group not found",
			retrieveData: &RetrieveData{
				EntityLabel: "user",
				Query: &retrieve.Query{
					EntityLabel: "user",
					FeatureGroups: []*retrieve.FeatureGroup{
						{
							Label:         "nonexistent_fg",
							FeatureLabels: []string{"feature1"},
						},
					},
				},
				Result: &retrieve.Result{},
			},
			setupMock: func(m *config.MockConfigManager) {
				m.On("GetEntity", "user").Return(&config.Entity{}, nil)
				m.On("GetActiveFeatureSchema", "user", "nonexistent_fg").Return(nil,
					fmt.Errorf("feature group not found"))
			},
			expectedError: "failed to get active schema for feature group nonexistent_fg: feature group not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConfig := new(config.MockConfigManager)
			tt.setupMock(mockConfig)

			err := preProcessFGs(tt.retrieveData, mockConfig)

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				tt.validateResult(t, tt.retrieveData)
			}

			mockConfig.AssertExpectations(t)
		})
	}
}

func TestPreProcessEntity(t *testing.T) {
	system.Init()
	tests := []struct {
		name         string
		retrieveData *RetrieveData
		setupMock    func(*config.MockConfigManager)
		validate     func(*testing.T, *RetrieveData)
		wantErr      bool
	}{
		{
			name: "happy path - mixed cache settings",
			retrieveData: &RetrieveData{
				EntityLabel: "user",
				ReqFGIds:    ds.NewOrderedSetWithCapacity[int](2).Add(1).Add(2),
			},
			setupMock: func(m *config.MockConfigManager) {
				m.On("GetEntity", "user").Return(&config.Entity{
					Label: "user",
					InMemoryCache: config.Cache{
						Enabled: true,
					},
					DistributedCache: config.Cache{
						Enabled: true,
					},
					FeatureGroups: map[string]config.FeatureGroup{
						"fg1": {
							Id:                      1,
							DataType:                "DataTypeFP64",
							StoreId:                 "store1",
							InMemoryCacheEnabled:    true,
							DistributedCacheEnabled: false,
						},
						"fg2": {
							Id:                      2,
							DataType:                "DataTypeInt32",
							StoreId:                 "store2",
							InMemoryCacheEnabled:    false,
							DistributedCacheEnabled: true,
						},
						"fg3": { // Not in request
							Id:                      3,
							DataType:                "DataTypeString",
							StoreId:                 "store3",
							DistributedCacheEnabled: true,
						},
					},
				}, nil)
			},
			validate: func(t *testing.T, rd *RetrieveData) {
				// Validate all mappings
				assert.Equal(t, 3, len(rd.AllFGLabelToFGId))
				assert.Equal(t, 3, len(rd.AllFGIdToStoreId))
				assert.Equal(t, 3, len(rd.AllFGIdToDataType))

				// Validate FG mappings
				assert.Equal(t, 1, rd.AllFGLabelToFGId["fg1"])
				assert.Equal(t, "store1", rd.AllFGIdToStoreId[1])
				assert.Equal(t, types.DataTypeFP64, rd.AllFGIdToDataType[1])

				// Validate cache sets
				assert.True(t, rd.ReqInMemCachedFGIds.Has(1))
				assert.True(t, rd.ReqDistCachedFGIds.Has(2))
				assert.True(t, rd.AllDistCachedFGIds.Has(2))
				assert.True(t, rd.AllDistCachedFGIds.Has(3)) // Non-requested but dist-cacheable

				// Validate sizes
				assert.Equal(t, 1, rd.ReqInMemCachedFGIds.Size())
				assert.Equal(t, 1, rd.ReqDistCachedFGIds.Size())
				assert.Equal(t, 2, rd.AllDistCachedFGIds.Size())
				assert.Equal(t, 0, rd.ReqDbFGIds.Size())
			},
		},
		{
			name: "entity not found",
			retrieveData: &RetrieveData{
				EntityLabel: "nonexistent",
			},
			setupMock: func(m *config.MockConfigManager) {
				m.On("GetEntity", "nonexistent").Return(nil,
					fmt.Errorf("entity not found"))
			},
			wantErr: true,
		},
		{
			name: "invalid data type",
			retrieveData: &RetrieveData{
				EntityLabel: "user",
				ReqFGIds:    ds.NewOrderedSetWithCapacity[int](1).Add(1),
			},
			setupMock: func(m *config.MockConfigManager) {
				m.On("GetEntity", "user").Return(&config.Entity{
					Label: "user",
					FeatureGroups: map[string]config.FeatureGroup{
						"fg1": {
							Id:       1,
							DataType: "Invali",
						},
					},
				}, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConfig := new(config.MockConfigManager)
			tt.setupMock(mockConfig)

			err := preProcessEntity(tt.retrieveData, mockConfig)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			tt.validate(t, tt.retrieveData)
			mockConfig.AssertExpectations(t)
		})
	}
}

func TestPreProcessForKeys(t *testing.T) {
	tests := []struct {
		name           string
		retrieveData   *RetrieveData
		setupMock      func(*config.MockConfigManager)
		expectedError  string
		validateResult func(*testing.T, *RetrieveData)
	}{
		{
			name: "Happy path - numeric features with duplicate keys",
			retrieveData: &RetrieveData{
				EntityLabel:    "user",
				ReqColumnCount: 2,
				Query: &retrieve.Query{
					Keys: []*retrieve.Keys{
						{Cols: []string{"user1", "region1"}},
						{Cols: []string{"user2", "region2"}},
						{Cols: []string{"user1", "region1"}}, // Duplicate key
					},
				},
				Result: &retrieve.Result{
					FeatureSchemas: []*retrieve.FeatureSchema{
						{
							FeatureGroupLabel: "user_profile",
							Features: []*retrieve.Feature{
								{Label: "age", ColumnIdx: 0},
								{Label: "income", ColumnIdx: 1},
							},
						},
					},
				},
				AllFGLabelToFGId: map[string]int{
					"user_profile": 1,
				},
				AllFGIdToDataType: map[int]types.DataType{
					1: types.DataTypeInt32,
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				m.On("GetFeatureGroup", "user", "user_profile").Return(&config.FeatureGroup{
					Id:            1,
					ActiveVersion: "1",
				}, nil)

				m.On("GetDefaultValueByte", "user", 1, 1, "age").Return([]byte{0, 0, 0, 25}, nil)
				m.On("GetDefaultValueByte", "user", 1, 1, "income").Return([]byte{0, 0, 0, 50}, nil)
			},
			validateResult: func(t *testing.T, rd *RetrieveData) {
				// Check unique keys
				assert.Len(t, rd.UniqueKeys, 2, "Should have 2 unique keys")
				assert.Len(t, rd.Result.Rows, 3, "Should have 3 total rows")

				// Validate KeyToOriginalIndices
				indices := rd.KeyToOriginalIndices["user1|region1"]
				assert.Len(t, indices, 2, "Should have 2 indices for duplicate key")
				assert.Contains(t, indices, 0)
				assert.Contains(t, indices, 2)

				// Check ReqKeyToIdx mapping
				assert.Equal(t, 0, rd.ReqKeyToIdx["user1|region1"])
				assert.Equal(t, 1, rd.ReqKeyToIdx["user2|region2"])

				// Validate all rows have correct default values
				for i := 0; i < 3; i++ {
					row := rd.Result.Rows[i]
					assert.Equal(t, []byte{0, 0, 0, 25}, row.Columns[0], "First column should have default age value")
					assert.Equal(t, []byte{0, 0, 0, 50}, row.Columns[1], "Second column should have default income value")
				}
			},
		},
		{
			name: "Happy path - string features with duplicate keys",
			retrieveData: &RetrieveData{
				EntityLabel:    "user",
				ReqColumnCount: 1,
				Query: &retrieve.Query{
					Keys: []*retrieve.Keys{
						{Cols: []string{"user1"}},
						{Cols: []string{"user1"}},
						{Cols: []string{"user2"}},
					},
				},
				Result: &retrieve.Result{
					FeatureSchemas: []*retrieve.FeatureSchema{
						{
							FeatureGroupLabel: "user_info",
							Features: []*retrieve.Feature{
								{Label: "name", ColumnIdx: 0},
							},
						},
					},
				},
				AllFGLabelToFGId: map[string]int{
					"user_info": 1,
				},
				AllFGIdToDataType: map[int]types.DataType{
					1: types.DataTypeString,
				},
			},
			setupMock: func(m *config.MockConfigManager) {},
			validateResult: func(t *testing.T, rd *RetrieveData) {
				// Check unique keys
				assert.Len(t, rd.UniqueKeys, 2, "Should have 2 unique keys")
				assert.Len(t, rd.Result.Rows, 3, "Should have 3 total rows")

				// Validate KeyToOriginalIndices
				indices := rd.KeyToOriginalIndices["user1"]
				assert.Len(t, indices, 2, "Should have 2 indices for duplicate key")
				assert.Contains(t, indices, 0)
				assert.Contains(t, indices, 1)

				// Check ReqKeyToIdx mapping
				assert.Equal(t, 0, rd.ReqKeyToIdx["user1"])
				assert.Equal(t, 2, rd.ReqKeyToIdx["user2"])

				// Validate all rows have nil columns initially
				for i := 0; i < 3; i++ {
					row := rd.Result.Rows[i]
					assert.Nil(t, row.Columns[0], "String column should be nil initially")
				}
			},
		},
		{
			name: "Happy path - numeric features",
			retrieveData: &RetrieveData{
				EntityLabel:    "user",
				ReqColumnCount: 2,
				Query: &retrieve.Query{
					Keys: []*retrieve.Keys{
						{Cols: []string{"user1", "region1"}},
					},
				},
				Result: &retrieve.Result{
					FeatureSchemas: []*retrieve.FeatureSchema{
						{
							FeatureGroupLabel: "user_profile",
							Features: []*retrieve.Feature{
								{Label: "age", ColumnIdx: 0},
								{Label: "income", ColumnIdx: 1},
							},
						},
					},
				},
				AllFGLabelToFGId: map[string]int{
					"user_profile": 1,
				},
				AllFGIdToDataType: map[int]types.DataType{
					1: types.DataTypeInt32,
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				// Initial feature group properties check
				m.On("GetFeatureGroup", "user", "user_profile").Return(&config.FeatureGroup{
					Id:            1,
					ActiveVersion: "1",
				}, nil)

				// Default values
				m.On("GetDefaultValueByte", "user", 1, 1, "age").Return([]byte{0, 0, 0, 25}, nil)
				m.On("GetDefaultValueByte", "user", 1, 1, "income").Return([]byte{0, 0, 0, 50}, nil)
			},
			validateResult: func(t *testing.T, rd *RetrieveData) {
				assert.Len(t, rd.Result.Rows, 1)
				row := rd.Result.Rows[0]
				assert.Equal(t, []string{"user1", "region1"}, row.Keys)
				assert.Equal(t, []byte{0, 0, 0, 25}, row.Columns[0], "First column should have default age value")
				assert.Equal(t, []byte{0, 0, 0, 50}, row.Columns[1], "Second column should have default income value")
				assert.Equal(t, 0, rd.ReqKeyToIdx["user1|region1"])
			},
		},
		{
			name: "Happy path - string features",
			retrieveData: &RetrieveData{
				EntityLabel:    "user",
				ReqColumnCount: 1,
				Query: &retrieve.Query{
					Keys: []*retrieve.Keys{
						{Cols: []string{"user1"}},
					},
				},
				Result: &retrieve.Result{
					FeatureSchemas: []*retrieve.FeatureSchema{
						{
							FeatureGroupLabel: "user_info",
							Features: []*retrieve.Feature{
								{Label: "name", ColumnIdx: 0},
							},
						},
					},
				},
				AllFGLabelToFGId: map[string]int{
					"user_info": 1,
				},
				AllFGIdToDataType: map[int]types.DataType{
					1: types.DataTypeString,
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				// Mock GetFeatureGroup for vector check only (string type skips default value)
				//m.On("GetFeatureGroup", "user", "user_info").Return(&config.FeatureGroup{
				//	Id:            1,
				//	ActiveVersion: "1",
				//}, nil).Once()
			},
			validateResult: func(t *testing.T, rd *RetrieveData) {
				assert.Len(t, rd.Result.Rows, 1)
				row := rd.Result.Rows[0]
				assert.Equal(t, []string{"user1"}, row.Keys)
				assert.Nil(t, row.Columns[0], "String column should be nil initially")
			},
		},
		{
			name: "Vector feature type",
			retrieveData: &RetrieveData{
				EntityLabel:    "user",
				ReqColumnCount: 1,
				Query: &retrieve.Query{
					Keys: []*retrieve.Keys{
						{Cols: []string{"user1"}},
					},
				},
				Result: &retrieve.Result{
					FeatureSchemas: []*retrieve.FeatureSchema{
						{
							FeatureGroupLabel: "user_embeddings",
							Features: []*retrieve.Feature{
								{Label: "embedding", ColumnIdx: 0},
							},
						},
					},
				},
				AllFGLabelToFGId: map[string]int{
					"user_embeddings": 1,
				},
				AllFGIdToDataType: map[int]types.DataType{
					1: types.DataTypeFP32Vector,
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				// Mock GetFeatureGroup for both vector check and version check
				m.On("GetFeatureGroup", "user", "user_embeddings").Return(&config.FeatureGroup{
					Id:            1,
					ActiveVersion: "1",
				}, nil).Once()

				// Mock GetDefaultValueByte for vector feature
				m.On("GetDefaultValueByte", "user", 1, 1, "embedding").Return([]byte{0, 0, 0, 0, 0, 0, 0, 0}, nil).Once()
			},
			validateResult: func(t *testing.T, rd *RetrieveData) {
				assert.Len(t, rd.Result.Rows, 1)
				row := rd.Result.Rows[0]
				assert.Equal(t, []string{"user1"}, row.Keys)
				assert.Equal(t, []byte{0, 0, 0, 0, 0, 0, 0, 0}, row.Columns[0], "Vector column should have default value")
			},
		},
		{
			name: "Error - GetFeatureGroup fails",
			retrieveData: &RetrieveData{
				EntityLabel:    "user",
				ReqColumnCount: 1,
				Query: &retrieve.Query{
					Keys: []*retrieve.Keys{
						{Cols: []string{"user1"}},
					},
				},
				Result: &retrieve.Result{
					FeatureSchemas: []*retrieve.FeatureSchema{
						{
							FeatureGroupLabel: "nonexistent",
							Features: []*retrieve.Feature{
								{Label: "feature1", ColumnIdx: 0},
							},
						},
					},
				},
				AllFGLabelToFGId: map[string]int{
					"nonexistent": 1,
				},
				AllFGIdToDataType: map[int]types.DataType{
					1: types.DataTypeInt32,
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				m.On("GetFeatureGroup", "user", "nonexistent").Return(nil,
					fmt.Errorf("feature group not found")).Once()
			},
			expectedError: "failed to get active schema: feature group not found",
		},
		{
			name: "Error - GetDefaultValueByte fails",
			retrieveData: &RetrieveData{
				EntityLabel:    "user",
				ReqColumnCount: 1,
				Query: &retrieve.Query{
					Keys: []*retrieve.Keys{
						{Cols: []string{"user1"}},
					},
				},
				Result: &retrieve.Result{
					FeatureSchemas: []*retrieve.FeatureSchema{
						{
							FeatureGroupLabel: "user_profile",
							Features: []*retrieve.Feature{
								{Label: "nonexistent_feature", ColumnIdx: 0},
							},
						},
					},
				},
				AllFGLabelToFGId: map[string]int{
					"user_profile": 1,
				},
				AllFGIdToDataType: map[int]types.DataType{
					1: types.DataTypeInt32,
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				m.On("GetFeatureGroup", "user", "user_profile").Return(&config.FeatureGroup{
					Id:            1,
					ActiveVersion: "1",
				}, nil).Once()
				m.On("GetDefaultValueByte", "user", 1, 1, "nonexistent_feature").Return(nil,
					fmt.Errorf("feature not found")).Once()
			},
			expectedError: "failed to get default value for feature nonexistent_feature: feature not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConfig := new(config.MockConfigManager)
			tt.setupMock(mockConfig)

			err := preProcessForKeys(tt.retrieveData, mockConfig)

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				tt.validateResult(t, tt.retrieveData)
			}

			mockConfig.AssertExpectations(t)
		})
	}
}
