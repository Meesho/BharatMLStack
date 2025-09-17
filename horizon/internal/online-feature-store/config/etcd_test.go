package config

import (
	"testing"

	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config/enums"
	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEtcdInstance is a mock implementation that focuses on the methods we actually use
type MockEtcdInstance struct {
	mock.Mock
	etcd.Etcd // Embed the interface
}

func (m *MockEtcdInstance) GetConfigInstance() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *MockEtcdInstance) SetValues(paths map[string]interface{}) error {
	args := m.Called(paths)
	return args.Error(0)
}

func (m *MockEtcdInstance) CreateNodes(paths map[string]interface{}) error {
	args := m.Called(paths)
	return args.Error(0)
}

func (m *MockEtcdInstance) IsNodeExist(path string) (bool, error) {
	args := m.Called(path)
	return args.Bool(0), args.Error(1)
}

func (m *MockEtcdInstance) Delete(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

// Test helper functions
func createTestFeatureRegistry() *FeatureRegistry {
	return &FeatureRegistry{
		Source: map[string]string{},
		Entities: map[string]Entity{
			"test_entity": {
				Label: "test_entity",
				Keys: map[string]Key{
					"key1": {Sequence: 0, EntityLabel: "test_entity", ColumnLabel: "id"},
				},
				FeatureGroups: map[string]FeatureGroup{
					"test_fg": {
						Id:            1,
						ActiveVersion: "1",
						StoreId:       "1",
						DataType:      enums.DataTypeInt32,
						TtlInSeconds:  3600,
						JobId:         "test_job",
						Columns: map[string]Column{
							"seg_0": {Label: "seg_0", CurrentSizeInBytes: 100},
							"seg_1": {Label: "seg_1", CurrentSizeInBytes: 200},
						},
						Features: map[string]Feature{
							"1": {
								Labels:        "feature1,feature2",
								DefaultValues: "10,20",
								FeatureMeta: map[string]FeatureMeta{
									"feature1": {Sequence: 0, DefaultValuesInBytes: []byte{10, 0, 0, 0}, StringLength: 0, VectorLength: 0},
									"feature2": {Sequence: 1, DefaultValuesInBytes: []byte{20, 0, 0, 0}, StringLength: 0, VectorLength: 0},
								},
							},
						},
						InMemoryCacheEnabled:    true,
						DistributedCacheEnabled: true,
						LayoutVersion:           1,
					},
				},
			},
		},
		Storage: Storage{
			Stores: map[string]Store{
				"1": {
					DbType:               "postgres",
					ConfId:               1,
					Table:                "test_table",
					MaxColumnSizeInBytes: 1024,
					MaxRowSizeInBytes:    102400,
					PrimaryKeys:          []string{"id"},
					TableTtl:             86400,
				},
			},
		},
		Security: Security{
			Writer: map[string]Property{
				"test_job": {Token: "test_token"},
			},
			Reader: map[string]Property{},
		},
	}
}

func TestEtcd_AddFeatures(t *testing.T) {
	tests := []struct {
		name                 string
		entityLabel          string
		fgLabel              string
		labels               []string
		defaultValues        []string
		storageProvider      []string
		sourceBasePath       []string
		sourceDataPath       []string
		stringLength         []string
		vectorLength         []string
		setupMocks           func(*MockEtcdInstance)
		expectedColumnsToAdd []string
		expectError          bool
		expectedError        string
	}{
		{
			name:            "successful add features",
			entityLabel:     "test_entity",
			fgLabel:         "test_fg",
			labels:          []string{"feature3", "feature4"},
			defaultValues:   []string{"30", "40"},
			storageProvider: []string{"", ""},
			sourceBasePath:  []string{"", ""},
			sourceDataPath:  []string{"", ""},
			stringLength:    []string{"0", "0"},
			vectorLength:    []string{"0", "0"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/test_fg").Return(true, nil)
				mockEtcd.On("GetConfigInstance").Return(createTestFeatureRegistry())
				mockEtcd.On("SetValues", mock.AnythingOfType("map[string]interface {}")).Return(nil)
			},
			expectedColumnsToAdd: []string(nil),
			expectError:          false,
		},
		{
			name:        "entity not found",
			entityLabel: "nonexistent_entity",
			fgLabel:     "test_fg",
			labels:      []string{"feature3"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/nonexistent_entity").Return(false, nil)
			},
			expectError:   true,
			expectedError: "entity nonexistent_entity not found",
		},
		{
			name:        "feature group not found",
			entityLabel: "test_entity",
			fgLabel:     "nonexistent_fg",
			labels:      []string{"feature3"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/nonexistent_fg").Return(false, nil)
			},
			expectError:   true,
			expectedError: "feature group nonexistent_fg not found",
		},
		{
			name:            "duplicate feature labels in input",
			entityLabel:     "test_entity",
			fgLabel:         "test_fg",
			labels:          []string{"feature3", "feature3"},
			defaultValues:   []string{"30", "40"},
			storageProvider: []string{"", ""},
			sourceBasePath:  []string{"", ""},
			sourceDataPath:  []string{"", ""},
			stringLength:    []string{"0", "0"},
			vectorLength:    []string{"0", "0"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/test_fg").Return(true, nil)
			},
			expectError:   true,
			expectedError: "duplicate feature label in input: 'feature3'",
		},
		{
			name:            "feature already exists",
			entityLabel:     "test_entity",
			fgLabel:         "test_fg",
			labels:          []string{"feature1"}, // already exists in test data
			defaultValues:   []string{"30"},
			storageProvider: []string{""},
			sourceBasePath:  []string{""},
			sourceDataPath:  []string{""},
			stringLength:    []string{"0"},
			vectorLength:    []string{"0"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/test_fg").Return(true, nil)
				mockEtcd.On("GetConfigInstance").Return(createTestFeatureRegistry())
			},
			expectError:   true,
			expectedError: "feature label 'feature1' already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEtcd := new(MockEtcdInstance)
			tt.setupMocks(mockEtcd)

			etcdConfig := &Etcd{
				instance: mockEtcd,
				appName:  "test_app",
				env:      "test",
			}

			columnsToAdd, store, pathsToUpdate, paths, err := etcdConfig.AddFeatures(
				tt.entityLabel, tt.fgLabel, tt.labels, tt.defaultValues,
				tt.storageProvider, tt.sourceBasePath, tt.sourceDataPath,
				tt.stringLength, tt.vectorLength,
			)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedColumnsToAdd, columnsToAdd)
				assert.NotNil(t, store)
				assert.NotNil(t, pathsToUpdate)
				assert.NotNil(t, paths)
			}

			mockEtcd.AssertExpectations(t)
		})
	}
}

func TestEtcd_EditFeatures(t *testing.T) {
	tests := []struct {
		name            string
		entityLabel     string
		fgLabel         string
		featureLabels   []string
		defaultValues   []string
		storageProvider []string
		sourceBasePath  []string
		sourceDataPath  []string
		stringLength    []string
		vectorLength    []string
		setupMocks      func(*MockEtcdInstance)
		expectError     bool
		expectedError   string
	}{
		{
			name:            "successful edit features",
			entityLabel:     "test_entity",
			fgLabel:         "test_fg",
			featureLabels:   []string{"feature1"},
			defaultValues:   []string{"100"},
			storageProvider: []string{"s3"},
			sourceBasePath:  []string{"/base/path"},
			sourceDataPath:  []string{"/data/path"},
			stringLength:    []string{"0"},
			vectorLength:    []string{"0"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/test_fg").Return(true, nil)
				mockEtcd.On("GetConfigInstance").Return(createTestFeatureRegistry())
				mockEtcd.On("SetValues", mock.AnythingOfType("map[string]interface {}")).Return(nil)
			},
			expectError: false,
		},
		{
			name:          "entity not found",
			entityLabel:   "nonexistent_entity",
			fgLabel:       "test_fg",
			featureLabels: []string{"feature1"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/nonexistent_entity").Return(false, nil)
			},
			expectError:   true,
			expectedError: "entity nonexistent_entity not found",
		},
		{
			name:          "feature group not found",
			entityLabel:   "test_entity",
			fgLabel:       "nonexistent_fg",
			featureLabels: []string{"feature1"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/nonexistent_fg").Return(false, nil)
			},
			expectError:   true,
			expectedError: "feature group nonexistent_fg not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEtcd := new(MockEtcdInstance)
			tt.setupMocks(mockEtcd)

			etcdConfig := &Etcd{
				instance: mockEtcd,
				appName:  "test_app",
				env:      "test",
			}

			_, _, err := etcdConfig.EditFeatures(
				tt.entityLabel, tt.fgLabel, tt.featureLabels, tt.defaultValues,
				tt.storageProvider, tt.sourceBasePath, tt.sourceDataPath,
				tt.stringLength, tt.vectorLength,
			)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockEtcd.AssertExpectations(t)
		})
	}
}

func TestEtcd_DeleteFeatures(t *testing.T) {
	tests := []struct {
		name          string
		entityLabel   string
		fgLabel       string
		featureLabels []string
		setupMocks    func(*MockEtcdInstance)
		expectError   bool
		expectedError string
	}{
		{
			name:          "successful delete features",
			entityLabel:   "test_entity",
			fgLabel:       "test_fg",
			featureLabels: []string{"feature1"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/test_fg").Return(true, nil)
				mockEtcd.On("GetConfigInstance").Return(createTestFeatureRegistry())
				mockEtcd.On("SetValues", mock.AnythingOfType("map[string]interface {}")).Return(nil)
				mockEtcd.On("Delete", "/config/test_app/source/test_entity|test_fg|feature1").Return(nil)
			},
			expectError: false,
		},
		{
			name:          "entity not found",
			entityLabel:   "nonexistent_entity",
			fgLabel:       "test_fg",
			featureLabels: []string{"feature1"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/nonexistent_entity").Return(false, nil)
			},
			expectError:   true,
			expectedError: "entity nonexistent_entity not found",
		},
		{
			name:          "feature group not found",
			entityLabel:   "test_entity",
			fgLabel:       "nonexistent_fg",
			featureLabels: []string{"feature1"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/nonexistent_fg").Return(false, nil)
			},
			expectError:   true,
			expectedError: "feature group nonexistent_fg not found",
		},
		{
			name:          "feature not found in active version",
			entityLabel:   "test_entity",
			fgLabel:       "test_fg",
			featureLabels: []string{"nonexistent_feature"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/test_fg").Return(true, nil)
				mockEtcd.On("GetConfigInstance").Return(createTestFeatureRegistry())
			},
			expectError:   true,
			expectedError: "feature label 'nonexistent_feature' does not exist in the active version of feature group",
		},
		{
			name:          "multiple features with some not found",
			entityLabel:   "test_entity",
			fgLabel:       "test_fg",
			featureLabels: []string{"feature1", "nonexistent_feature"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/test_fg").Return(true, nil)
				mockEtcd.On("GetConfigInstance").Return(createTestFeatureRegistry())
			},
			expectError:   true,
			expectedError: "feature label 'nonexistent_feature' does not exist in the active version of feature group",
		},
		{
			name:          "delete all features",
			entityLabel:   "test_entity",
			fgLabel:       "test_fg",
			featureLabels: []string{"feature1", "feature2"},
			setupMocks: func(mockEtcd *MockEtcdInstance) {
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
				mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/test_fg").Return(true, nil)
				mockEtcd.On("GetConfigInstance").Return(createTestFeatureRegistry())
				mockEtcd.On("SetValues", mock.AnythingOfType("map[string]interface {}")).Return(nil)
				mockEtcd.On("Delete", "/config/test_app/source/test_entity|test_fg|feature1").Return(nil)
				mockEtcd.On("Delete", "/config/test_app/source/test_entity|test_fg|feature2").Return(nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEtcd := new(MockEtcdInstance)
			tt.setupMocks(mockEtcd)

			etcdConfig := &Etcd{
				instance: mockEtcd,
				appName:  "test_app",
				env:      "test",
			}

			err := etcdConfig.DeleteFeatures(tt.entityLabel, tt.fgLabel, tt.featureLabels)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockEtcd.AssertExpectations(t)
		})
	}
}

func TestEtcd_AddFeatures_WithVectorDataTypes(t *testing.T) {
	tests := []struct {
		name          string
		dataType      enums.DataType
		labels        []string
		defaultValues []string
		vectorLength  []string
		expectError   bool
	}{
		{
			name:          "add features with int32 vector",
			dataType:      enums.DataTypeInt32Vector,
			labels:        []string{"vector_feature"},
			defaultValues: []string{"[1,2,3,4]"},
			vectorLength:  []string{"4"},
			expectError:   false,
		},
		{
			name:          "add features with string vector",
			dataType:      enums.DataTypeStringVector,
			labels:        []string{"string_vector_feature"},
			defaultValues: []string{"[hello,world]"},
			vectorLength:  []string{"2"},
			expectError:   false,
		},
		{
			name:          "add features with float32 vector",
			dataType:      enums.DataTypeFP32Vector,
			labels:        []string{"float_vector_feature"},
			defaultValues: []string{"[1.0,2.5,3.7]"},
			vectorLength:  []string{"3"},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEtcd := new(MockEtcdInstance)

			// Create test registry with vector data type
			registry := createTestFeatureRegistry()

			// Extract the nested structs, modify, and reassign
			entity := registry.Entities["test_entity"]
			fg := entity.FeatureGroups["test_fg"]
			fg.DataType = tt.dataType
			entity.FeatureGroups["test_fg"] = fg
			registry.Entities["test_entity"] = entity

			mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
			mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/test_fg").Return(true, nil)
			mockEtcd.On("GetConfigInstance").Return(registry)

			if !tt.expectError {
				mockEtcd.On("SetValues", mock.AnythingOfType("map[string]interface {}")).Return(nil)
			}

			etcdConfig := &Etcd{
				instance: mockEtcd,
				appName:  "test_app",
				env:      "test",
			}

			_, _, _, _, err := etcdConfig.AddFeatures(
				"test_entity", "test_fg", tt.labels, tt.defaultValues,
				[]string{""}, []string{""}, []string{""},
				[]string{"0"}, tt.vectorLength,
			)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockEtcd.AssertExpectations(t)
		})
	}
}

func TestEtcd_SegmentSizeManagement(t *testing.T) {
	t.Run("add features requiring new segments", func(t *testing.T) {
		mockEtcd := new(MockEtcdInstance)

		// Create registry with small max column size to force segment creation
		registry := createTestFeatureRegistry()
		store := registry.Storage.Stores["1"]
		store.MaxColumnSizeInBytes = 10 // Very small to force new segments
		registry.Storage.Stores["1"] = store

		mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
		mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/test_fg").Return(true, nil)
		mockEtcd.On("GetConfigInstance").Return(registry)
		mockEtcd.On("CreateNodes", mock.AnythingOfType("map[string]interface {}")).Return(nil).Once()

		etcdConfig := &Etcd{
			instance: mockEtcd,
			appName:  "test_app",
			env:      "test",
		}

		// Add features that will require multiple segments
		columnsToAdd, _, _, _, err := etcdConfig.AddFeatures(
			"test_entity", "test_fg",
			[]string{"big_feature1", "big_feature2"},
			[]string{"123456789012345", "987654321098765"}, // Large values to exceed max column size
			[]string{"", ""}, []string{"", ""}, []string{"", ""},
			[]string{"0", "0"}, []string{"0", "0"},
		)

		assert.NoError(t, err)
		assert.NotEmpty(t, columnsToAdd) // Should create new segments
		mockEtcd.AssertExpectations(t)
	})
}

// Benchmark tests for performance
func BenchmarkEtcd_AddFeatures(b *testing.B) {
	mockEtcd := new(MockEtcdInstance)
	mockEtcd.On("IsNodeExist", mock.AnythingOfType("string")).Return(true, nil)
	mockEtcd.On("GetConfigInstance").Return(createTestFeatureRegistry())
	mockEtcd.On("SetValues", mock.AnythingOfType("map[string]interface {}")).Return(nil)

	etcdConfig := &Etcd{
		instance: mockEtcd,
		appName:  "test_app",
		env:      "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _, _ = etcdConfig.AddFeatures(
			"test_entity", "test_fg",
			[]string{"benchmark_feature"},
			[]string{"100"},
			[]string{""}, []string{""}, []string{""},
			[]string{"0"}, []string{"0"},
		)
	}
}

// Test validation functionality for string and vector features
func TestEtcd_AddFeatures_ValidationTests(t *testing.T) {
	tests := []struct {
		name          string
		dataType      enums.DataType
		labels        []string
		defaultValues []string
		stringLength  []string
		vectorLength  []string
		expectError   bool
		expectedError string
	}{
		{
			name:          "string length validation - value exceeds configured length",
			dataType:      enums.DataTypeString,
			labels:        []string{"string_feature"},
			defaultValues: []string{"this_is_a_very_long_string"},
			stringLength:  []string{"10"}, // Configured length is 10, but value is longer
			vectorLength:  []string{"0"},
			expectError:   true,
			expectedError: "default value length (26) exceeds configured string length (10)",
		},
		{
			name:          "string length validation - value within configured length",
			dataType:      enums.DataTypeString,
			labels:        []string{"string_feature"},
			defaultValues: []string{"short"},
			stringLength:  []string{"10"}, // Configured length is 10, value is 5
			vectorLength:  []string{"0"},
			expectError:   false,
		},
		{
			name:          "string vector validation - vector size exceeds configured length",
			dataType:      enums.DataTypeStringVector,
			labels:        []string{"string_vector_feature"},
			defaultValues: []string{"[hello,world,test,extra,item]"},
			stringLength:  []string{"10"},
			vectorLength:  []string{"3"}, // Configured vector length is 3, but we have 5 items
			expectError:   true,
			expectedError: "vector size (5) exceeds configured vector length (3)",
		},
		{
			name:          "string vector validation - individual string exceeds length",
			dataType:      enums.DataTypeStringVector,
			labels:        []string{"string_vector_feature"},
			defaultValues: []string{"[hello,very_long_string_exceeding_lmt]"}, // This is exactly 30 chars
			stringLength:  []string{"10"},                                     // Each string should be max 10 chars
			vectorLength:  []string{"3"},
			expectError:   true,
			expectedError: "string at position 1 (length 30) exceeds configured string length (10)", // Fixed to 30
		},
		{
			name:          "string vector validation - valid configuration",
			dataType:      enums.DataTypeStringVector,
			labels:        []string{"string_vector_feature"},
			defaultValues: []string{"[hello,world]"},
			stringLength:  []string{"10"},
			vectorLength:  []string{"5"},
			expectError:   false,
		},
		{
			name:          "multiple features with mixed validation results",
			dataType:      enums.DataTypeString,
			labels:        []string{"new_feature1", "new_feature2"},                 // Use new feature names to avoid existing feature conflict
			defaultValues: []string{"valid", "this_string_is_too_long_for_the_cfg"}, // Fixed length to 35 chars
			stringLength:  []string{"10", "10"},
			vectorLength:  []string{"0", "0"},
			expectError:   true,
			expectedError: "default value length (35) exceeds configured string length (10) for feature at index 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEtcd := new(MockEtcdInstance)

			// Create test registry with the specified data type
			registry := createTestFeatureRegistry()
			entity := registry.Entities["test_entity"]
			fg := entity.FeatureGroups["test_fg"]
			fg.DataType = tt.dataType
			entity.FeatureGroups["test_fg"] = fg
			registry.Entities["test_entity"] = entity

			mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity").Return(true, nil)
			mockEtcd.On("IsNodeExist", "/config/test_app/entities/test_entity/feature-groups/test_fg").Return(true, nil)
			mockEtcd.On("GetConfigInstance").Return(registry)

			if !tt.expectError {
				mockEtcd.On("SetValues", mock.AnythingOfType("map[string]interface {}")).Return(nil)
			}

			etcdConfig := &Etcd{
				instance: mockEtcd,
				appName:  "test_app",
				env:      "test",
			}

			_, _, _, _, err := etcdConfig.AddFeatures(
				"test_entity", "test_fg", tt.labels, tt.defaultValues,
				make([]string, len(tt.labels)), make([]string, len(tt.labels)), make([]string, len(tt.labels)),
				tt.stringLength, tt.vectorLength,
			)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockEtcd.AssertExpectations(t)
		})
	}
}
