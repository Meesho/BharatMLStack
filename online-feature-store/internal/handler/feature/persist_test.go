package feature

import (
	"fmt"
	"os"
	"testing"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/blocks"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config/enums"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/persist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	system.Init()
	os.Exit(m.Run())
}

func TestPersist(t *testing.T) {
	tests := []struct {
		name           string
		query          *persist.Query
		setupMock      func(*config.MockConfigManager)
		validateResult func(*testing.T, *PersistData)
		expectedError  string
	}{
		{
			name: "Fail - invalid entityLabel",
			query: &persist.Query{
				EntityLabel: "invalid_entity",
				FeatureGroupSchema: []*persist.FeatureGroupSchema{
					{
						Label:         "derived_int64",
						FeatureLabels: []string{"feature1"},
					},
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				m.On("GetEntity", "invalid_entity").Return(nil, fmt.Errorf("entity not found"))
			},
			expectedError: "invalid entity invalid_entity: entity not found",
		},
		{
			name: "Fail - Invalid Feature Group",
			query: &persist.Query{
				EntityLabel: "user_sscat",
				FeatureGroupSchema: []*persist.FeatureGroupSchema{
					{
						Label:         "invalid_fg",
						FeatureLabels: []string{"feature1"},
					},
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				m.On("GetEntity", "user_sscat").Return(&config.Entity{
					Label: "user_sscat",
					FeatureGroups: map[string]config.FeatureGroup{
						"derived_int64": {
							Id:       1,
							DataType: enums.DataTypeInt64,
						},
					},
				}, nil)
				m.On("GetFeatureGroup", "user_sscat", "invalid_fg").Return(nil, fmt.Errorf("feature group not found"))
			},
			expectedError: "failed to get feature group invalid_fg: feature group not found",
		},
		{
			name: "Happy path - single feature group, all features",
			query: &persist.Query{
				EntityLabel: "user_sscat",
				FeatureGroupSchema: []*persist.FeatureGroupSchema{
					{
						Label: "derived_int64",
						FeatureLabels: []string{
							"clicks_7day",
							"orders_7day",
							"user_sscat__days_since_last_order",
							"user_sscat__platform_clicks_14_days",
							"user_sscat__platform_orders_28_days",
							"user_sscat__platform_orders_56_days",
							"views_7day",
						},
					},
				},
				KeysSchema: []string{"user_id", "sscat_id"},
				Data: []*persist.Data{
					{
						KeyValues: []string{"-2", "-3"},
						FeatureValues: []*persist.FeatureValues{
							{
								Values: &persist.Values{
									Int64Values: []int64{12, 5, 10, 25, 8, 15, 30},
								},
							},
						},
					},
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				// Mock GetEntity
				m.On("GetEntity", "user_sscat").Return(&config.Entity{
					Label: "user_sscat",
					FeatureGroups: map[string]config.FeatureGroup{
						"derived_int64": {
							Id:            1,
							ActiveVersion: "1",
							DataType:      enums.DataTypeInt64,
							StoreId:       "1",
							LayoutVersion: 1,
							TtlInSeconds:  604800,
							Features: map[string]config.FeatureSchema{
								"1": {
									Labels: "",
									FeatureMeta: map[string]config.FeatureMeta{
										"clicks_7day": {
											Sequence:             2,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
										},
										"orders_7day": {
											Sequence:             3,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
										},
										"user_sscat__days_since_last_order": {
											Sequence:             0,
											DefaultValuesInBytes: []byte{255, 255, 255, 255, 255, 255, 255, 255},
										},
										"user_sscat__platform_clicks_14_days": {
											Sequence:             4,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
										},
										"user_sscat__platform_orders_28_days": {
											Sequence:             6,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
										},
										"user_sscat__platform_orders_56_days": {
											Sequence:             5,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
										},
										"views_7day": {
											Sequence:             1,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
										},
									},
								},
							},
							DistributedCacheEnabled: true,
							InMemoryCacheEnabled:    true,
						},
					},
					DistributedCache: config.Cache{Enabled: true, TtlInSeconds: 600},
					InMemoryCache:    config.Cache{Enabled: true, TtlInSeconds: 120},
				}, nil)

				// Mock GetFeatureGroup
				m.On("GetFeatureGroup", "user_sscat", "derived_int64").Return(&config.FeatureGroup{
					Id:                      1,
					DataType:                enums.DataTypeInt64,
					StoreId:                 "1",
					LayoutVersion:           1,
					TtlInSeconds:            604800,
					DistributedCacheEnabled: true,
					InMemoryCacheEnabled:    true,
				}, nil)

				// Mock GetStore
				m.On("GetStore", "1").Return(&config.Store{
					DbType:               "scylla",
					ConfId:               1,
					MaxColumnSizeInBytes: 1024,
					MaxRowSizeInBytes:    102400,
				}, nil)

				// Mock GetNumOfFeatures
				m.On("GetNumOfFeatures", "user_sscat", 1, mock.Anything).Return(7, nil)

				// Mock GetStringLengths
				m.On("GetStringLengths", "user_sscat", 1, mock.Anything).Return([]uint16{0}, nil)

				// Mock GetVectorLengths
				m.On("GetVectorLengths", "user_sscat", 1, mock.Anything).Return([]uint16{0}, nil)

				// Mock GetActiveVersion
				m.On("GetActiveVersion", "user_sscat", 1).Return(1, nil)
			},
			validateResult: func(t *testing.T, persistData *PersistData) {
				assert.NotNil(t, persistData.AllFGIdToFgConf)
				assert.NotNil(t, persistData.AllFGLabelToFGId)
				assert.True(t, persistData.IsAnyFGDistCached)
				assert.True(t, persistData.IsAnyFGInMemCached)

				// Verify StoreIdToRows
				assert.NotNil(t, persistData.StoreIdToRows)
				rows := persistData.StoreIdToRows["1"]
				assert.Equal(t, 1, len(rows), "Should have 1 row")

				row := rows[0]
				// Verify primary keys
				assert.Equal(t, map[string]string{
					"user_id":  "-2",
					"sscat_id": "-3",
				}, row.PkMap)

				// Verify PSDB block exists
				assert.Contains(t, row.FgIdToPsDb, 1, "Should have PSDB block for derived_int64")

				// Verify derived_int64 PSDB block
				psdbInt64 := row.FgIdToPsDb[1]
				int64Values, ok := psdbInt64.Data.([]int64)
				assert.True(t, ok, "Int64 block should contain []int64 data")
				assert.Equal(t, []int64{10, 30, 12, 5, 25, 15, 8}, int64Values, "Int64 values should match")
			},
		},
		{
			name: "Happy path - single feature group, missing features",
			query: &persist.Query{
				EntityLabel: "user_sscat",
				FeatureGroupSchema: []*persist.FeatureGroupSchema{
					{
						Label: "derived_int64",
						FeatureLabels: []string{
							"clicks_7day",
							"orders_7day",
							"views_7day",
						},
					},
				},
				KeysSchema: []string{"user_id", "sscat_id"},
				Data: []*persist.Data{
					{
						KeyValues: []string{"-2", "-3"},
						FeatureValues: []*persist.FeatureValues{
							{
								Values: &persist.Values{
									Int64Values: []int64{12, 5, 30},
								},
							},
						},
					},
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				// Reuse the same mock setup as the previous test case but with different validation
				m.On("GetEntity", "user_sscat").Return(&config.Entity{
					Label: "user_sscat",
					FeatureGroups: map[string]config.FeatureGroup{
						"derived_int64": {
							Id:            1,
							ActiveVersion: "1",
							DataType:      enums.DataTypeInt64,
							StoreId:       "1",
							LayoutVersion: 1,
							TtlInSeconds:  604800,
							Features: map[string]config.FeatureSchema{
								"1": {
									Labels: "",
									FeatureMeta: map[string]config.FeatureMeta{
										"clicks_7day": {
											Sequence:             2,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"orders_7day": {
											Sequence:             3,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"user_sscat__days_since_last_order": {
											Sequence:             0,
											DefaultValuesInBytes: []byte{255, 255, 255, 255, 255, 255, 255, 255},
											StringLength:         0,
											VectorLength:         0,
										},
										"user_sscat__platform_clicks_14_days": {
											Sequence:             4,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"user_sscat__platform_orders_28_days": {
											Sequence:             6,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"user_sscat__platform_orders_56_days": {
											Sequence:             5,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"views_7day": {
											Sequence:             1,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
									},
								},
							},
							DistributedCacheEnabled: true,
							InMemoryCacheEnabled:    true,
						},
					},
					DistributedCache: config.Cache{Enabled: true, TtlInSeconds: 600},
					InMemoryCache:    config.Cache{Enabled: true, TtlInSeconds: 120},
				}, nil)

				// Mock GetFeatureGroup
				m.On("GetFeatureGroup", "user_sscat", "derived_int64").Return(&config.FeatureGroup{
					Id:                      1,
					DataType:                enums.DataTypeInt64,
					StoreId:                 "1",
					LayoutVersion:           1,
					TtlInSeconds:            604800,
					DistributedCacheEnabled: true,
					InMemoryCacheEnabled:    true,
				}, nil)

				// Mock GetStore
				m.On("GetStore", "1").Return(&config.Store{
					DbType:               "scylla",
					ConfId:               1,
					MaxColumnSizeInBytes: 1024,
					MaxRowSizeInBytes:    102400,
				}, nil)

				// Mock GetNumOfFeatures
				m.On("GetNumOfFeatures", "user_sscat", 1, mock.Anything).Return(3, nil)

				// Mock GetStringLengths
				m.On("GetStringLengths", "user_sscat", 1, mock.Anything).Return([]uint16{0}, nil)

				// Mock GetVectorLengths
				m.On("GetVectorLengths", "user_sscat", 1, mock.Anything).Return([]uint16{0}, nil)

				// Mock GetActiveVersion
				m.On("GetActiveVersion", "user_sscat", 1).Return(1, nil)
			},
			validateResult: func(t *testing.T, persistData *PersistData) {
				assert.NotNil(t, persistData.AllFGIdToFgConf)
				assert.NotNil(t, persistData.AllFGLabelToFGId)
				assert.True(t, persistData.IsAnyFGDistCached)
				assert.True(t, persistData.IsAnyFGInMemCached)

				// Verify StoreIdToRows
				assert.NotNil(t, persistData.StoreIdToRows)
				rows := persistData.StoreIdToRows["1"]
				assert.Equal(t, 1, len(rows), "Should have 1 row")

				row := rows[0]
				// Verify primary keys
				assert.Equal(t, map[string]string{
					"user_id":  "-2",
					"sscat_id": "-3",
				}, row.PkMap)

				// Verify PSDB block exists
				assert.Contains(t, row.FgIdToPsDb, 1, "Should have PSDB block for derived_int64")

				// Verify derived_int64 PSDB block
				psdbInt64 := row.FgIdToPsDb[1]
				int64Values, ok := psdbInt64.Data.([]int64)
				assert.True(t, ok, "Int64 block should contain []int64 data")
				assert.Equal(t, []int64{-1, 30, 12, 5, 0, 0, 0}, int64Values, "Int64 values should match")
			},
		},
		{
			name: "Happy path - Multiple feature groups , Missing Features",
			query: &persist.Query{
				EntityLabel: "user_sscat",
				FeatureGroupSchema: []*persist.FeatureGroupSchema{
					{
						Label: "derived_int64",
						FeatureLabels: []string{
							"user_sscat__platform_clicks_14_days",
							"clicks_7day",
							"user_sscat__platform_orders_28_days",
						},
					},
					{
						Label: "derived_string",
						FeatureLabels: []string{
							"attribute_value_1_click_28days",
							"attribute_value_1_orders_56days",
							"attribute_value_2_click_28days",
							"attribute_value_2_orders_56days",
							"attribute_value_3_click_28days",
							"attribute_value_3_orders_56days",
							"attribute_value_1_orders_28days",
							"attribute_value_2_orders_28days",
						},
					},
				},
				KeysSchema: []string{"user_id", "sscat_id"},
				Data: []*persist.Data{
					{
						KeyValues: []string{"-2", "-3"},
						FeatureValues: []*persist.FeatureValues{
							{
								Values: &persist.Values{
									Int64Values: []int64{25, 12, 5},
								},
							},
							{
								Values: &persist.Values{
									StringValues: []string{
										"high", "medium", "low", "high",
										"medium", "low", "high", "medium",
									},
									Vector: []*persist.Vector{
										{
											Values: &persist.Values{
												Int32Values: []int32{0, 1, 2, 3, 4, 5},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				// Mock GetEntity
				m.On("GetEntity", "user_sscat").Return(&config.Entity{
					Label: "user_sscat",
					FeatureGroups: map[string]config.FeatureGroup{
						"derived_int64": {
							Id:            1,
							ActiveVersion: "1",
							DataType:      enums.DataTypeInt64,
							StoreId:       "1",
							LayoutVersion: 1,
							TtlInSeconds:  604800,
							Columns: map[string]config.Column{
								"seg_1": {
									Label:              "seg_1",
									CurrentSizeInBytes: 56,
								},
							},
							Features: map[string]config.FeatureSchema{
								"1": {
									Labels: "",
									FeatureMeta: map[string]config.FeatureMeta{
										"clicks_7day": {
											Sequence:             2,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"orders_7day": {
											Sequence:             3,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"user_sscat__days_since_last_order": {
											Sequence:             0,
											DefaultValuesInBytes: []byte{255, 255, 255, 255, 255, 255, 255, 255},
											StringLength:         0,
											VectorLength:         0,
										},
										"user_sscat__platform_clicks_14_days": {
											Sequence:             4,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"user_sscat__platform_orders_28_days": {
											Sequence:             6,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"user_sscat__platform_orders_56_days": {
											Sequence:             5,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"views_7day": {
											Sequence:             1,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
									},
								},
							},
							DistributedCacheEnabled: true,
							InMemoryCacheEnabled:    true,
						},
						"derived_string": {
							Id:            2,
							ActiveVersion: "1",
							DataType:      enums.DataTypeString,
							StoreId:       "1",
							LayoutVersion: 1,
							TtlInSeconds:  604800,
							Columns: map[string]config.Column{
								"seg_2": {
									Label:              "seg_2",
									CurrentSizeInBytes: 0,
								},
							},
							Features: map[string]config.FeatureSchema{
								"1": {
									Labels: "",
									FeatureMeta: map[string]config.FeatureMeta{
										"attribute_value_1_click_28days": {
											Sequence:             0,
											DefaultValuesInBytes: []byte("other"),
											StringLength:         90,
											VectorLength:         0,
										},
										"attribute_value_1_orders_28days": {
											Sequence:             6,
											DefaultValuesInBytes: []byte("other"),
											StringLength:         80,
											VectorLength:         0,
										},
										"attribute_value_1_orders_56days": {
											Sequence:             1,
											DefaultValuesInBytes: []byte("other"),
											StringLength:         80,
											VectorLength:         0,
										},
										"attribute_value_2_click_28days": {
											Sequence:             2,
											DefaultValuesInBytes: []byte("other"),
											StringLength:         140,
											VectorLength:         0,
										},
										"attribute_value_2_orders_28days": {
											Sequence:             7,
											DefaultValuesInBytes: []byte("other"),
											StringLength:         140,
											VectorLength:         0,
										},
										"attribute_value_2_orders_56days": {
											Sequence:             3,
											DefaultValuesInBytes: []byte("other"),
											StringLength:         140,
											VectorLength:         0,
										},
										"attribute_value_3_click_28days": {
											Sequence:             4,
											DefaultValuesInBytes: []byte("other"),
											StringLength:         80,
											VectorLength:         0,
										},
										"attribute_value_3_orders_28days": {
											Sequence:             8,
											DefaultValuesInBytes: []byte("other"),
											StringLength:         80,
											VectorLength:         0,
										},
										"attribute_value_3_orders_56days": {
											Sequence:             5,
											DefaultValuesInBytes: []byte("other"),
											StringLength:         80,
											VectorLength:         0,
										},
									},
								},
							},
							DistributedCacheEnabled: true,
							InMemoryCacheEnabled:    true,
						},
					},
					DistributedCache: config.Cache{Enabled: true, TtlInSeconds: 600},
					InMemoryCache:    config.Cache{Enabled: true, TtlInSeconds: 120},
				}, nil)

				// Mock GetFeatureGroup for both groups
				m.On("GetFeatureGroup", "user_sscat", "derived_int64").Return(&config.FeatureGroup{
					Id:                      1,
					DataType:                enums.DataTypeInt64,
					StoreId:                 "1",
					LayoutVersion:           1,
					TtlInSeconds:            604800,
					DistributedCacheEnabled: true,
					InMemoryCacheEnabled:    true,
				}, nil)

				m.On("GetFeatureGroup", "user_sscat", "derived_string").Return(&config.FeatureGroup{
					Id:                      2,
					DataType:                enums.DataTypeString,
					StoreId:                 "1",
					LayoutVersion:           1,
					TtlInSeconds:            604800,
					DistributedCacheEnabled: true,
					InMemoryCacheEnabled:    true,
				}, nil)

				// Mock GetStore
				m.On("GetStore", "1").Return(&config.Store{
					DbType:               "scylla",
					ConfId:               1,
					MaxColumnSizeInBytes: 1024,
					MaxRowSizeInBytes:    102400,
				}, nil)

				// Mock GetNumOfFeatures
				m.On("GetNumOfFeatures", "user_sscat", 1, mock.Anything).Return(7, nil)
				m.On("GetNumOfFeatures", "user_sscat", 2, mock.Anything).Return(9, nil)

				// Mock GetStringLengths
				m.On("GetStringLengths", "user_sscat", 1, mock.Anything).Return([]uint16{0}, nil)
				m.On("GetStringLengths", "user_sscat", 2, mock.Anything).Return([]uint16{90, 80, 80, 140, 140, 140, 80, 80, 80}, nil)

				// Mock GetVectorLengths
				m.On("GetVectorLengths", "user_sscat", 1, mock.Anything).Return([]uint16{0}, nil)
				m.On("GetVectorLengths", "user_sscat", 2, mock.Anything).Return([]uint16{0}, nil)

				// Mock GetActiveVersion
				m.On("GetActiveVersion", "user_sscat", 1).Return(1, nil)
				m.On("GetActiveVersion", "user_sscat", 2).Return(1, nil)
			},
			validateResult: func(t *testing.T, persistData *PersistData) {
				assert.NotNil(t, persistData.AllFGIdToFgConf)
				assert.NotNil(t, persistData.AllFGLabelToFGId)
				assert.True(t, persistData.IsAnyFGDistCached)
				assert.True(t, persistData.IsAnyFGInMemCached)

				// Verify StoreIdToRows
				assert.NotNil(t, persistData.StoreIdToRows)
				rows := persistData.StoreIdToRows["1"]
				assert.Equal(t, 1, len(rows), "Should have 1 row")

				row := rows[0]
				// Verify primary keys
				assert.Equal(t, map[string]string{
					"user_id":  "-2",
					"sscat_id": "-3",
				}, row.PkMap)

				// Verify PSDB blocks exist for both feature groups
				assert.Contains(t, row.FgIdToPsDb, 1, "Should have PSDB block for derived_int64")
				assert.Contains(t, row.FgIdToPsDb, 2, "Should have PSDB block for derived_string")

				// Verify derived_int64 PSDB block
				psdbInt64 := row.FgIdToPsDb[1]
				int64Values, ok := psdbInt64.Data.([]int64)
				assert.True(t, ok, "Int64 block should contain []int64 data")
				assert.Equal(t, []int64{-1, 0, 12, 0, 25, 0, 5}, int64Values, "Int64 values should match")

				// Verify derived_string PSDB block
				psdbString := row.FgIdToPsDb[2]
				stringValues, ok := psdbString.Data.([]string)
				assert.True(t, ok, "String block should contain []string data")
				assert.Equal(t, []string{"high", "medium", "low", "high", "medium", "low", "high", "medium", "other"}, stringValues, "String values should match")
			},
		},
		{
			name: "Happy path - single feature group, extra features",
			query: &persist.Query{
				EntityLabel: "user_sscat",
				FeatureGroupSchema: []*persist.FeatureGroupSchema{
					{
						Label: "derived_int64",
						FeatureLabels: []string{
							"clicks_7day",
							"orders_7day",
							"views_7day",
							"extra_feature_1",
							"extra_feature_2",
						},
					},
				},
				KeysSchema: []string{"user_id", "sscat_id"},
				Data: []*persist.Data{
					{
						KeyValues: []string{"-2", "-3"},
						FeatureValues: []*persist.FeatureValues{
							{
								Values: &persist.Values{
									Int64Values: []int64{12, 5, 30, 40, 50},
								},
							},
						},
					},
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				m.On("GetEntity", "user_sscat").Return(&config.Entity{
					Label: "user_sscat",
					FeatureGroups: map[string]config.FeatureGroup{
						"derived_int64": {
							Id:            1,
							ActiveVersion: "1",
							DataType:      enums.DataTypeInt64,
							StoreId:       "1",
							LayoutVersion: 1,
							TtlInSeconds:  604800,
							Features: map[string]config.FeatureSchema{
								"1": {
									Labels: "",
									FeatureMeta: map[string]config.FeatureMeta{
										"clicks_7day": {
											Sequence:             2,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"orders_7day": {
											Sequence:             3,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"user_sscat__days_since_last_order": {
											Sequence:             0,
											DefaultValuesInBytes: []byte{255, 255, 255, 255, 255, 255, 255, 255},
											StringLength:         0,
											VectorLength:         0,
										},
										"user_sscat__platform_clicks_14_days": {
											Sequence:             4,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"user_sscat__platform_orders_28_days": {
											Sequence:             6,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"user_sscat__platform_orders_56_days": {
											Sequence:             5,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
										"views_7day": {
											Sequence:             1,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         0,
										},
									},
								},
							},
							DistributedCacheEnabled: true,
							InMemoryCacheEnabled:    true,
						},
					},
					DistributedCache: config.Cache{Enabled: true, TtlInSeconds: 600},
					InMemoryCache:    config.Cache{Enabled: true, TtlInSeconds: 120},
				}, nil)

				// Mock GetFeatureGroup
				m.On("GetFeatureGroup", "user_sscat", "derived_int64").Return(&config.FeatureGroup{
					Id:                      1,
					DataType:                enums.DataTypeInt64,
					StoreId:                 "1",
					LayoutVersion:           1,
					TtlInSeconds:            604800,
					DistributedCacheEnabled: true,
					InMemoryCacheEnabled:    true,
				}, nil)

				// Mock GetStore
				m.On("GetStore", "1").Return(&config.Store{
					DbType:               "scylla",
					ConfId:               1,
					MaxColumnSizeInBytes: 1024,
					MaxRowSizeInBytes:    102400,
				}, nil)

				// Mock GetNumOfFeatures
				m.On("GetNumOfFeatures", "user_sscat", 1, mock.Anything).Return(7, nil)

				// Mock GetStringLengths
				m.On("GetStringLengths", "user_sscat", 1, mock.Anything).Return([]uint16{0}, nil)

				// Mock GetVectorLengths
				m.On("GetVectorLengths", "user_sscat", 1, mock.Anything).Return([]uint16{0}, nil)

				// Mock GetActiveVersion
				m.On("GetActiveVersion", "user_sscat", 1).Return(1, nil)
			},
			validateResult: func(t *testing.T, persistData *PersistData) {
				assert.NotNil(t, persistData.AllFGIdToFgConf)
				assert.NotNil(t, persistData.AllFGLabelToFGId)
				assert.True(t, persistData.IsAnyFGDistCached)
				assert.True(t, persistData.IsAnyFGInMemCached)

				// Verify StoreIdToRows
				assert.NotNil(t, persistData.StoreIdToRows)
				rows := persistData.StoreIdToRows["1"]
				assert.Equal(t, 1, len(rows), "Should have 1 row")

				row := rows[0]
				// Verify primary keys
				assert.Equal(t, map[string]string{
					"user_id":  "-2",
					"sscat_id": "-3",
				}, row.PkMap)

				// Verify PSDB block exists
				assert.Contains(t, row.FgIdToPsDb, 1, "Should have PSDB block for derived_int64")

				// Verify derived_int64 PSDB block
				psdbInt64 := row.FgIdToPsDb[1]
				int64Values, ok := psdbInt64.Data.([]int64)
				assert.True(t, ok, "Int64 block should contain []int64 data")
				// Only the valid features should be included, extra features should be ignored
				assert.Equal(t, []int64{-1, 30, 12, 5, 0, 0, 0}, int64Values, "Int64 values should match")
			},
		},
		{
			name: "Happy path - single feature group, vector features",
			query: &persist.Query{
				EntityLabel: "user_sscat",
				FeatureGroupSchema: []*persist.FeatureGroupSchema{
					{
						Label: "derived_int64_vector",
						FeatureLabels: []string{
							"embedding_10",
							"embedding_20",
						},
					},
				},
				KeysSchema: []string{"user_id", "sscat_id"},
				Data: []*persist.Data{
					{
						KeyValues: []string{"-2", "-3"},
						FeatureValues: []*persist.FeatureValues{
							{
								Values: &persist.Values{
									Vector: []*persist.Vector{
										{
											Values: &persist.Values{
												Int64Values: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
											},
										},
										{
											Values: &persist.Values{
												Int64Values: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				m.On("GetEntity", "user_sscat").Return(&config.Entity{
					Label: "user_sscat",
					FeatureGroups: map[string]config.FeatureGroup{
						"derived_int64_vector": {
							Id:            3,
							ActiveVersion: "1",
							DataType:      enums.DataTypeInt64Vector,
							StoreId:       "1",
							LayoutVersion: 1,
							TtlInSeconds:  604800,
							Features: map[string]config.FeatureSchema{
								"1": {
									Labels: "",
									FeatureMeta: map[string]config.FeatureMeta{
										"embedding_10": {
											Sequence:             0,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         10,
										},
										"embedding_20": {
											Sequence: 1,
											DefaultValuesInBytes: []byte{
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 0
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 1
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 2
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 3
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 4
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 5
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 6
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 7
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 8
												255, 255, 255, 255, 255, 255, 255, 255, // -1 at index 9
												255, 255, 255, 255, 255, 255, 255, 255, // -1 at index 10
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 11
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 12
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 13
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 14
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 15
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 16
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 17
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 18
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 19
											},
											StringLength: 0,
											VectorLength: 20,
										},
									},
								},
							},
							DistributedCacheEnabled: true,
							InMemoryCacheEnabled:    true,
						},
					},
					DistributedCache: config.Cache{Enabled: true, TtlInSeconds: 600},
					InMemoryCache:    config.Cache{Enabled: true, TtlInSeconds: 120},
				}, nil)

				// Mock GetFeatureGroup
				m.On("GetFeatureGroup", "user_sscat", "derived_int64_vector").Return(&config.FeatureGroup{
					Id:                      3,
					DataType:                enums.DataTypeInt64Vector,
					StoreId:                 "1",
					LayoutVersion:           1,
					TtlInSeconds:            604800,
					DistributedCacheEnabled: true,
					InMemoryCacheEnabled:    true,
				}, nil)

				// Mock GetStore
				m.On("GetStore", "1").Return(&config.Store{
					DbType:               "scylla",
					ConfId:               1,
					MaxColumnSizeInBytes: 1024,
					MaxRowSizeInBytes:    102400,
				}, nil)

				// Mock GetNumOfFeatures
				m.On("GetNumOfFeatures", "user_sscat", 3, mock.Anything).Return(2, nil)

				// Mock GetStringLengths
				m.On("GetStringLengths", "user_sscat", 3, mock.Anything).Return([]uint16{0}, nil)

				// Mock GetVectorLengths
				m.On("GetVectorLengths", "user_sscat", 3, mock.Anything).Return([]uint16{10, 20}, nil)

				// Mock GetActiveVersion
				m.On("GetActiveVersion", "user_sscat", 3).Return(1, nil)
			},
			validateResult: func(t *testing.T, persistData *PersistData) {
				assert.NotNil(t, persistData.AllFGIdToFgConf)
				assert.NotNil(t, persistData.AllFGLabelToFGId)
				assert.True(t, persistData.IsAnyFGDistCached)
				assert.True(t, persistData.IsAnyFGInMemCached)

				// Verify StoreIdToRows
				assert.NotNil(t, persistData.StoreIdToRows)
				rows := persistData.StoreIdToRows["1"]
				assert.Equal(t, 1, len(rows), "Should have 1 row")

				row := rows[0]
				// Verify primary keys
				assert.Equal(t, map[string]string{
					"user_id":  "-2",
					"sscat_id": "-3",
				}, row.PkMap)

				// Verify PSDB block exists
				assert.Contains(t, row.FgIdToPsDb, 3, "Should have PSDB block for derived_int64_vector")

				// Verify derived_int64_vector PSDB block
				psdbInt64Vector := row.FgIdToPsDb[3]
				vectorValues, ok := psdbInt64Vector.Data.([][]int64)
				assert.True(t, ok, "Int64Vector block should contain [][]int64 data")
				assert.Equal(t, 2, len(vectorValues), "Should have 2 vectors")

				// Verify first vector (length 10)
				assert.Equal(t, 10, len(vectorValues[0]), "First vector should have length 10")
				assert.Equal(t, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, vectorValues[0], "First vector values should match")

				// Verify second vector (length 20)
				assert.Equal(t, 20, len(vectorValues[1]), "Second vector should have length 20")
				assert.Equal(t, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}, vectorValues[1], "Second vector values should match")
			},
		},
		{
			name: "Happy path - single feature group, vector features, missing features",
			query: &persist.Query{
				EntityLabel: "user_sscat",
				FeatureGroupSchema: []*persist.FeatureGroupSchema{
					{
						Label: "derived_int64_vector",
						FeatureLabels: []string{
							"embedding_10", // Only requesting the first feature
						},
					},
				},
				KeysSchema: []string{"user_id", "sscat_id"},
				Data: []*persist.Data{
					{
						KeyValues: []string{"-2", "-3"},
						FeatureValues: []*persist.FeatureValues{
							{
								Values: &persist.Values{
									Vector: []*persist.Vector{
										{
											Values: &persist.Values{
												Int64Values: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			setupMock: func(m *config.MockConfigManager) {
				m.On("GetEntity", "user_sscat").Return(&config.Entity{
					Label: "user_sscat",
					FeatureGroups: map[string]config.FeatureGroup{
						"derived_int64_vector": {
							Id:            3,
							ActiveVersion: "1",
							DataType:      enums.DataTypeInt64Vector,
							StoreId:       "1",
							LayoutVersion: 1,
							TtlInSeconds:  604800,
							Features: map[string]config.FeatureSchema{
								"1": {
									Labels: "",
									FeatureMeta: map[string]config.FeatureMeta{
										"embedding_10": {
											Sequence:             0,
											DefaultValuesInBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0},
											StringLength:         0,
											VectorLength:         10,
										},
										"embedding_20": {
											Sequence: 1,
											DefaultValuesInBytes: []byte{
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 0
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 1
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 2
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 3
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 4
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 5
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 6
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 7
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 8
												255, 255, 255, 255, 255, 255, 255, 255, // -1 at index 9
												255, 255, 255, 255, 255, 255, 255, 255, // -1 at index 10
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 11
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 12
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 13
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 14
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 15
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 16
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 17
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 18
												0, 0, 0, 0, 0, 0, 0, 0, // 0 at index 19
											},
											StringLength: 0,
											VectorLength: 20,
										},
									},
								},
							},
							DistributedCacheEnabled: true,
							InMemoryCacheEnabled:    true,
						},
					},
					DistributedCache: config.Cache{Enabled: true, TtlInSeconds: 600},
					InMemoryCache:    config.Cache{Enabled: true, TtlInSeconds: 120},
				}, nil)

				// Mock GetFeatureGroup
				m.On("GetFeatureGroup", "user_sscat", "derived_int64_vector").Return(&config.FeatureGroup{
					Id:                      3,
					DataType:                enums.DataTypeInt64Vector,
					StoreId:                 "1",
					LayoutVersion:           1,
					TtlInSeconds:            604800,
					DistributedCacheEnabled: true,
					InMemoryCacheEnabled:    true,
				}, nil)

				// Mock GetStore
				m.On("GetStore", "1").Return(&config.Store{
					DbType:               "scylla",
					ConfId:               1,
					MaxColumnSizeInBytes: 1024,
					MaxRowSizeInBytes:    102400,
				}, nil)

				// Mock GetNumOfFeatures
				m.On("GetNumOfFeatures", "user_sscat", 3, mock.Anything).Return(2, nil)

				// Mock GetStringLengths
				m.On("GetStringLengths", "user_sscat", 3, mock.Anything).Return([]uint16{0}, nil)

				// Mock GetVectorLengths
				m.On("GetVectorLengths", "user_sscat", 3, mock.Anything).Return([]uint16{10, 20}, nil)

				// Mock GetActiveVersion
				m.On("GetActiveVersion", "user_sscat", 3).Return(1, nil)
			},
			validateResult: func(t *testing.T, persistData *PersistData) {
				assert.NotNil(t, persistData.AllFGIdToFgConf)
				assert.NotNil(t, persistData.AllFGLabelToFGId)
				assert.True(t, persistData.IsAnyFGDistCached)
				assert.True(t, persistData.IsAnyFGInMemCached)

				// Verify StoreIdToRows
				assert.NotNil(t, persistData.StoreIdToRows)
				rows := persistData.StoreIdToRows["1"]
				assert.Equal(t, 1, len(rows), "Should have 1 row")

				row := rows[0]
				// Verify primary keys
				assert.Equal(t, map[string]string{
					"user_id":  "-2",
					"sscat_id": "-3",
				}, row.PkMap)

				// Verify PSDB block exists
				assert.Contains(t, row.FgIdToPsDb, 3, "Should have PSDB block for derived_int64_vector")

				// Verify derived_int64_vector PSDB block
				psdbInt64Vector := row.FgIdToPsDb[3]
				vectorValues, ok := psdbInt64Vector.Data.([][]int64)
				assert.True(t, ok, "Int64Vector block should contain [][]int64 data")
				assert.Equal(t, 2, len(vectorValues), "Should have 2 vectors")

				// Verify first vector (length 10) - provided in request
				assert.Equal(t, 10, len(vectorValues[0]), "First vector should have length 10")
				assert.Equal(t, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, vectorValues[0], "First vector values should match")

				// Verify second vector (length 20) - should be default values since it was missing
				assert.Equal(t, 20, len(vectorValues[1]), "Second vector should have length 20")
				expectedDefaultVector := []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, -1, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0} // Default values with -1 at positions 9 and 10
				assert.Equal(t, expectedDefaultVector, vectorValues[1], "Second vector should be default values")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConfig := new(config.MockConfigManager)
			tt.setupMock(mockConfig)

			handler := &PersistHandler{
				config: mockConfig,
			}

			persistData := &PersistData{
				Query:       tt.query,
				EntityLabel: tt.query.EntityLabel,
			}

			err := handler.preProcessRequest(persistData)
			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
				return
			}
			assert.NoError(t, err)

			// Add preparePersistData call
			err = handler.preparePersistData(persistData)
			assert.NoError(t, err)

			tt.validateResult(t, persistData)

			mockConfig.AssertExpectations(t)
		})
	}
}

func TestRestructureRowsByPrimaryKeys(t *testing.T) {
	handler := &PersistHandler{}

	tests := []struct {
		name           string
		inputRows      []Row
		expectedRows   []Row
		expectedCount  int
		description    string
		validateMerged func(*testing.T, []Row)
	}{
		{
			name: "Single row - no restructuring needed",
			inputRows: []Row{
				{
					PkMap: map[string]string{"catalog_id": "123"},
					FgIdToPsDb: map[int]*blocks.PermStorageDataBlock{
						1: &blocks.PermStorageDataBlock{},
					},
				},
			},
			expectedCount: 1,
			description:   "Single row should remain unchanged",
			validateMerged: func(t *testing.T, result []Row) {
				assert.Equal(t, 1, len(result))
				row := result[0]
				assert.Equal(t, map[string]string{"catalog_id": "123"}, row.PkMap)
				assert.Equal(t, 1, len(row.FgIdToPsDb))
				assert.Contains(t, row.FgIdToPsDb, 1)
			},
		},
		{
			name: "Multiple rows with same primary key - should merge",
			inputRows: []Row{
				{
					PkMap: map[string]string{"catalog_id": "123"},
					FgIdToPsDb: map[int]*blocks.PermStorageDataBlock{
						1: &blocks.PermStorageDataBlock{},
					},
				},
				{
					PkMap: map[string]string{"catalog_id": "123"},
					FgIdToPsDb: map[int]*blocks.PermStorageDataBlock{
						2: &blocks.PermStorageDataBlock{},
					},
				},
			},
			expectedCount: 1,
			description:   "Two rows with same catalog_id should be merged into one",
			validateMerged: func(t *testing.T, result []Row) {
				assert.Equal(t, 1, len(result))
				row := result[0]
				assert.Equal(t, map[string]string{"catalog_id": "123"}, row.PkMap)
				assert.Equal(t, 2, len(row.FgIdToPsDb), "Should have 2 feature groups after merging")
				assert.Contains(t, row.FgIdToPsDb, 1, "Should contain feature group 1")
				assert.Contains(t, row.FgIdToPsDb, 2, "Should contain feature group 2")
			},
		},
		{
			name: "Multiple rows with different primary keys - no merging",
			inputRows: []Row{
				{
					PkMap: map[string]string{"catalog_id": "123"},
					FgIdToPsDb: map[int]*blocks.PermStorageDataBlock{
						1: &blocks.PermStorageDataBlock{},
					},
				},
				{
					PkMap: map[string]string{"catalog_id": "456"},
					FgIdToPsDb: map[int]*blocks.PermStorageDataBlock{
						2: &blocks.PermStorageDataBlock{},
					},
				},
			},
			expectedCount: 2,
			description:   "Two rows with different catalog_ids should remain separate",
			validateMerged: func(t *testing.T, result []Row) {
				assert.Equal(t, 2, len(result))

				// Find the row with catalog_id:123
				var row123, row456 *Row
				for i := range result {
					if result[i].PkMap["catalog_id"] == "123" {
						row123 = &result[i]
					} else if result[i].PkMap["catalog_id"] == "456" {
						row456 = &result[i]
					}
				}

				assert.NotNil(t, row123, "Should have row with catalog_id:123")
				assert.NotNil(t, row456, "Should have row with catalog_id:456")

				assert.Equal(t, 1, len(row123.FgIdToPsDb))
				assert.Contains(t, row123.FgIdToPsDb, 1)

				assert.Equal(t, 1, len(row456.FgIdToPsDb))
				assert.Contains(t, row456.FgIdToPsDb, 2)
			},
		},
		{
			name: "Cross entity primary keys - should merge correctly",
			inputRows: []Row{
				{
					PkMap: map[string]string{"campaign_id": "123", "catalog_id": "456"},
					FgIdToPsDb: map[int]*blocks.PermStorageDataBlock{
						1: &blocks.PermStorageDataBlock{},
					},
				},
				{
					PkMap: map[string]string{"catalog_id": "456", "campaign_id": "123"}, // Same keys, different order
					FgIdToPsDb: map[int]*blocks.PermStorageDataBlock{
						2: &blocks.PermStorageDataBlock{},
					},
				},
			},
			expectedCount: 1,
			description:   "Cross entity rows with same keys in different order should be merged",
			validateMerged: func(t *testing.T, result []Row) {
				assert.Equal(t, 1, len(result))
				row := result[0]
				expectedPkMap := map[string]string{"campaign_id": "123", "catalog_id": "456"}
				assert.Equal(t, expectedPkMap, row.PkMap)
				assert.Equal(t, 2, len(row.FgIdToPsDb), "Should have 2 feature groups after merging")
				assert.Contains(t, row.FgIdToPsDb, 1, "Should contain feature group 1")
				assert.Contains(t, row.FgIdToPsDb, 2, "Should contain feature group 2")
			},
		},
		{
			name: "Complex scenario - multiple merges",
			inputRows: []Row{
				{
					PkMap: map[string]string{"catalog_id": "123"},
					FgIdToPsDb: map[int]*blocks.PermStorageDataBlock{
						1: &blocks.PermStorageDataBlock{},
					},
				},
				{
					PkMap: map[string]string{"catalog_id": "123"},
					FgIdToPsDb: map[int]*blocks.PermStorageDataBlock{
						2: &blocks.PermStorageDataBlock{},
					},
				},
				{
					PkMap: map[string]string{"catalog_id": "456"},
					FgIdToPsDb: map[int]*blocks.PermStorageDataBlock{
						3: &blocks.PermStorageDataBlock{},
					},
				},
				{
					PkMap: map[string]string{"catalog_id": "123"},
					FgIdToPsDb: map[int]*blocks.PermStorageDataBlock{
						4: &blocks.PermStorageDataBlock{},
					},
				},
			},
			expectedCount: 2,
			description:   "Should merge 3 rows with catalog_id=123 into 1, keep 1 row with catalog_id=456",
			validateMerged: func(t *testing.T, result []Row) {
				assert.Equal(t, 2, len(result))

				// Find the merged row with catalog_id:123
				var row123, row456 *Row
				for i := range result {
					if result[i].PkMap["catalog_id"] == "123" {
						row123 = &result[i]
					} else if result[i].PkMap["catalog_id"] == "456" {
						row456 = &result[i]
					}
				}

				assert.NotNil(t, row123, "Should have merged row with catalog_id:123")
				assert.NotNil(t, row456, "Should have row with catalog_id:456")

				// Check that catalog_id:123 row has all 3 feature groups merged
				assert.Equal(t, 3, len(row123.FgIdToPsDb), "Should have 3 feature groups after merging")
				assert.Contains(t, row123.FgIdToPsDb, 1, "Should contain feature group 1")
				assert.Contains(t, row123.FgIdToPsDb, 2, "Should contain feature group 2")
				assert.Contains(t, row123.FgIdToPsDb, 4, "Should contain feature group 4")

				// Check that catalog_id:456 row has its single feature group
				assert.Equal(t, 1, len(row456.FgIdToPsDb))
				assert.Contains(t, row456.FgIdToPsDb, 3)
			},
		},
		{
			name:          "Empty input - should return empty",
			inputRows:     []Row{},
			expectedCount: 0,
			description:   "Empty input should return empty output",
			validateMerged: func(t *testing.T, result []Row) {
				assert.Equal(t, 0, len(result))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.restructureRowsByPrimaryKeys(tt.inputRows)

			assert.Equal(t, tt.expectedCount, len(result), tt.description)

			// Verify that all feature groups are preserved
			totalFgCount := 0
			for _, row := range tt.inputRows {
				totalFgCount += len(row.FgIdToPsDb)
			}

			resultFgCount := 0
			for _, row := range result {
				resultFgCount += len(row.FgIdToPsDb)
			}

			assert.Equal(t, totalFgCount, resultFgCount, "Total feature group count should be preserved")

			// Run the specific validation for this test case
			tt.validateMerged(t, result)
		})
	}
}
