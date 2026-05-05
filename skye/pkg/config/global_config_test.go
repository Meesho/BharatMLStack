package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type StaticConfig struct {
	Key1 string `yaml:"key1"`
	Key2 string `yaml:"key2"`
	Key3 string `yaml:"key3"`
}

type DynamicConfig struct {
	Key4 string `yaml:"key4"`
	Key5 string `yaml:"key5"`
	Key6 string `yaml:"key6"`
}

type MockGlobalConf struct {
	staticConfig  *StaticConfig
	dynamicConfig *DynamicConfig
}

func (m *MockGlobalConf) GetStaticConfig() interface{} {
	return m.staticConfig
}

func (m *MockGlobalConf) GetDynamicConfig() interface{} {
	return m.dynamicConfig
}

func TestInitGlobalConfig(t *testing.T) {
	tests := []struct {
		name                  string
		podNamespace          string
		envVars               map[string]string
		staticConfigContent   string
		dynamicConfigContent  string
		expectedStaticConfig  *StaticConfig
		expectedDynamicConfig *DynamicConfig
		shouldPanic           bool
		skipFileCreation      bool // when true, config files are not written (to test missing-file scenarios)
	}{
		{
			name:         "Static and Dynamic with deployable section",
			podNamespace: "dev-deployable",
			staticConfigContent: `
key1: value1
key2: value2
---
deployable-name: deployable
key2: override-value2
key3: value3
`,
			dynamicConfigContent: `
key4: value4
key5: value5
---
deployable-name: deployable
key5: override-value5
key6: value6
`,
			expectedStaticConfig: &StaticConfig{
				Key1: "value1",
				Key2: "override-value2",
				Key3: "value3",
			},
			expectedDynamicConfig: &DynamicConfig{
				Key4: "value4",
				Key5: "override-value5",
				Key6: "value6",
			},
			envVars: map[string]string{
				envConfigLocation: "/tmp/config",
			},
			shouldPanic: false,
		},
		{
			name:         "Static with deployable section, Dynamic without deployable section",
			podNamespace: "dev-deployable",
			staticConfigContent: `
key1: value1
key2: value2
---
deployable-name: deployable
key2: override-value2
key3: value3
`,
			dynamicConfigContent: `
key4: value4
key5: value5
`,
			expectedStaticConfig: &StaticConfig{
				Key1: "value1",
				Key2: "override-value2",
				Key3: "value3",
			},
			expectedDynamicConfig: &DynamicConfig{
				Key4: "value4",
				Key5: "value5",
				Key6: "",
			},
			envVars: map[string]string{
				envConfigLocation: "/tmp/config",
			},
			shouldPanic: false,
		},
		{
			name:         "Dynamic File With Invalid Yaml Content",
			podNamespace: "dev-deployable",
			staticConfigContent: `
key1: value1
key2: value2
---
deployable-name: deployable
key2: override-value2
key3: value3
`,
			dynamicConfigContent: `
key4
key5: value5
`,
			expectedStaticConfig:  &StaticConfig{},
			expectedDynamicConfig: &DynamicConfig{},
			envVars: map[string]string{
				envConfigLocation: "/tmp/config",
			},
			shouldPanic: true,
		},
		{
			name:         "Static File With Invalid Yaml Content",
			podNamespace: "dev-deployable",
			staticConfigContent: `
key1: value1
key2
---
deployable-name: deployable
key2: override-value2
key3: value3
`,
			dynamicConfigContent: `
key4 : value1
key5: value5
`,
			envVars: map[string]string{
				envConfigLocation: "/tmp/config",
			},
			expectedStaticConfig:  &StaticConfig{},
			expectedDynamicConfig: &DynamicConfig{},
			shouldPanic:           true,
		},
		{
			name:                 "No sections found in config file",
			podNamespace:         "dev-deployable",
			staticConfigContent:  "",
			dynamicConfigContent: "",
			envVars: map[string]string{
				envConfigLocation: "/tmp/config",
			},
			expectedStaticConfig:  &StaticConfig{},
			expectedDynamicConfig: &DynamicConfig{},
			shouldPanic:           false,
		},
		{
			name:                  "Static and Dynamic missing files",
			podNamespace:          "dev-deployable",
			staticConfigContent:   "",
			dynamicConfigContent:  "",
			expectedStaticConfig:  nil,
			expectedDynamicConfig: nil,
			shouldPanic:           true,
			skipFileCreation:      true,
		},
		{
			name:         "Setting no section",
			podNamespace: "dev-deployable",
			staticConfigContent: `
key1: value1
key2: value2
`,
			dynamicConfigContent: `
deployable-name: deployable-1
`,
			expectedStaticConfig: &StaticConfig{
				Key1: "value1",
				Key2: "value2",
			},
			envVars: map[string]string{
				envConfigLocation: "/tmp/config",
			},
			expectedDynamicConfig: &DynamicConfig{},
			shouldPanic:           false,
		},
		{
			name:         "Setting only deployable specific section",
			podNamespace: "dev-deployable",
			staticConfigContent: `
key1: value1
key2: value2
`,
			dynamicConfigContent: `
deployable-name: deployable
`,
			expectedStaticConfig: &StaticConfig{
				Key1: "value1",
				Key2: "value2",
			},
			envVars: map[string]string{
				envConfigLocation: "/tmp/config",
			},
			expectedDynamicConfig: &DynamicConfig{},
			shouldPanic:           false,
		},
		{
			name:         "Initialise dynamic config source etcd",
			podNamespace: "dev-deployable",
			staticConfigContent: `
key1: value1
key2: value2
---
deployable-name: deployable
key2: override-value2
key3: value3
dynamic-config:
  source: etcd

`,
			dynamicConfigContent: `
key4: value4
key5: value5
`,
			expectedStaticConfig: &StaticConfig{
				Key1: "value1",
				Key2: "override-value2",
				Key3: "value3",
			},
			expectedDynamicConfig: &DynamicConfig{
				Key4: "value4",
				Key5: "value5",
				Key6: "",
			},
			envVars: map[string]string{
				envConfigLocation: "/tmp/config",
			},
			shouldPanic: false,
		},
		{
			name:         "Initialise dynamic config source zookeeper",
			podNamespace: "dev-deployable",
			staticConfigContent: `
key1: value1
key2: value2
---
deployable-name: deployable
key2: override-value2
key3: value3
dynamic-config:
  source: zookeeper

`,
			dynamicConfigContent: `
key4: value4
key5: value5
`,
			expectedStaticConfig: &StaticConfig{
				Key1: "value1",
				Key2: "override-value2",
				Key3: "value3",
			},
			expectedDynamicConfig: &DynamicConfig{
				Key4: "value4",
				Key5: "value5",
				Key6: "",
			},
			envVars: map[string]string{
				envConfigLocation: "/tmp/config",
			},
			shouldPanic: true,
		},
		{
			name:         "Unmarshall static config source error",
			podNamespace: "dev-deployable",
			staticConfigContent: `
key1: value1
key2:
  nestedKey: 1
---
deployable-name: deployable
key3: value3
dynamic-config:
  source: etcd

`,
			dynamicConfigContent: `
key4: value4
key5: value5
`,
			expectedStaticConfig:  &StaticConfig{},
			expectedDynamicConfig: &DynamicConfig{},
			shouldPanic:           true,
			envVars: map[string]string{
				envConfigLocation: "/tmp/config",
			},
		},
		{
			name:         "Dynamic Config Source Soft Failure",
			podNamespace: "dev-deployable",
			staticConfigContent: `
key1: value1
key2: 1
---
deployable-name: deployable
key3: value3
dynamic-config:
  source: wrong-dynamic-source
  optional: true
`,
			dynamicConfigContent: `
key4: value4
key5: value5
`,
			expectedStaticConfig: &StaticConfig{
				Key1: "value1",
				Key2: "1",
				Key3: "value3",
			},
			expectedDynamicConfig: &DynamicConfig{
				Key4: "value4",
				Key5: "value5",
			},
			shouldPanic: false,
			envVars: map[string]string{
				envConfigLocation: "/tmp/config",
			},
		},
		{
			name:                  "Invalid POD_NAMESPACE format",
			podNamespace:          "invalidnamespace",
			staticConfigContent:   "",
			dynamicConfigContent:  "",
			expectedStaticConfig:  nil,
			expectedDynamicConfig: nil,
			shouldPanic:           true,
		},
		{
			name: "Environment and Deployable Name from Environment Variables",
			envVars: map[string]string{
				envEnvironment:    "dev",
				envDeployableName: "deployable",
				envConfigLocation: "/tmp/config",
			},
			staticConfigContent: `
key1: value1
key2: value2
---
deployable-name: deployable
key2: override-value2
key3: value3
`,
			dynamicConfigContent: `
key4: value4
key5: value5
---
deployable-name: deployable
key5: override-value5
key6: value6
`,
			expectedStaticConfig: &StaticConfig{
				Key1: "value1",
				Key2: "override-value2",
				Key3: "value3",
			},
			expectedDynamicConfig: &DynamicConfig{
				Key4: "value4",
				Key5: "override-value5",
				Key6: "value6",
			},
			shouldPanic: false,
		},
		{
			name: "No static and dynamic configs",
			envVars: map[string]string{
				envEnvironment:    "dev",
				envDeployableName: "deployable",
				envConfigLocation: "/tmp/config",
			},
			staticConfigContent:  ``,
			dynamicConfigContent: ``,
			expectedStaticConfig: &StaticConfig{
				Key1: "",
				Key2: "",
				Key3: "",
			},
			expectedDynamicConfig: &DynamicConfig{
				Key4: "",
				Key5: "",
				Key6: "",
			},
			shouldPanic: false,
		},
		{
			name: "Environment Variable Only",
			envVars: map[string]string{
				envEnvironment: "prod",
			},
			staticConfigContent: `
key1: value1
key2: value2
`,
			dynamicConfigContent: `
key4: value4
key5: value5
`,
			expectedStaticConfig:  &StaticConfig{},
			expectedDynamicConfig: &DynamicConfig{},
			shouldPanic:           true,
		},
		{
			name: "Deployable Name Variable Only",
			envVars: map[string]string{
				envDeployableName: "deployable",
			},
			staticConfigContent: `
key1: value1
key2: value2
---
deployable-name: deployable
key2: override-value2
key3: value3
`,
			dynamicConfigContent: `
key4: value4
key5: value5
---
deployable-name: deployable
key5: override-value5
key6: value6
`,
			expectedStaticConfig: &StaticConfig{
				Key1: "value1",
				Key2: "override-value2",
				Key3: "value3",
			},
			expectedDynamicConfig: &DynamicConfig{
				Key4: "value4",
				Key5: "override-value5",
				Key6: "value6",
			},
			shouldPanic: true,
		},
		{
			name:    "No Environment Variables",
			envVars: map[string]string{},
			staticConfigContent: `
key1: value1
key2: value2
`,
			dynamicConfigContent: `
key4: value4
key5: value5
`,
			expectedStaticConfig:  &StaticConfig{},
			expectedDynamicConfig: &DynamicConfig{},
			shouldPanic:           true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.RemoveAll("/tmp/config")
			os.RemoveAll("/opt/config")
			viper.Reset()
			if test.podNamespace != "" {
				viper.Set(envPodNamespace, test.podNamespace)
			}
			for key, value := range test.envVars {
				viper.Set(key, value)
			}
			mockConf := &MockGlobalConf{
				staticConfig:  &StaticConfig{},
				dynamicConfig: &DynamicConfig{},
			}
			directoryPath := "/opt/config"
			if viper.IsSet(envConfigLocation) {
				directoryPath = viper.GetString(envConfigLocation)
			}
			os.MkdirAll(directoryPath, os.ModePerm)
			if !test.skipFileCreation {
				os.WriteFile(directoryPath+"/application-dev.yml", []byte(test.staticConfigContent), os.ModePerm)
				os.WriteFile(directoryPath+"/application-dyn-dev.yml", []byte(test.dynamicConfigContent), os.ModePerm)
			}

			defer func() {
				if r := recover(); r != nil {
					if !test.shouldPanic {
						t.Errorf("unexpected panic: %v", r)
					}
				} else if test.shouldPanic {
					t.Error("expected panic but did not occur")
				}
			}()

			InitGlobalConfig(mockConf)

			if !test.shouldPanic {
				assert.Equal(t, *test.expectedStaticConfig, *mockConf.staticConfig)
				assert.Equal(t, *test.expectedDynamicConfig, *mockConf.dynamicConfig)
			}
		})
	}
}

func TestMergeMaps(t *testing.T) {
	tests := []struct {
		name     string
		source   map[string]interface{}
		target   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "Simple merge",
			source: map[string]interface{}{
				"key1": "value1",
			},
			target: map[string]interface{}{
				"key2": "value2",
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "Overwrite existing key",
			source: map[string]interface{}{
				"key1": "newValue1",
			},
			target: map[string]interface{}{
				"key1": "oldValue1",
			},
			expected: map[string]interface{}{
				"key1": "newValue1",
			},
		},
		{
			name: "Nested maps",
			source: map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"nestedKey1": "nestedValue1",
				},
			},
			target: map[string]interface{}{
				"key2": map[string]interface{}{
					"nestedKey1": "oldValue",
					"nestedKey2": "nestedValue2",
				},
				"key3": "value3",
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"nestedKey1": "nestedValue1",
					"nestedKey2": "nestedValue2",
				},
				"key3": "value3",
			},
		},
		{
			name:   "Source map is empty",
			source: map[string]interface{}{},
			target: map[string]interface{}{
				"key1": "value1",
			},
			expected: map[string]interface{}{
				"key1": "value1",
			},
		},
		{
			name: "Target map is empty",
			source: map[string]interface{}{
				"key1": "value1",
			},
			target: map[string]interface{}{},
			expected: map[string]interface{}{
				"key1": "value1",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := mergeMaps(test.source, test.target)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestTransformKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "Simple keys",
			input: map[string]interface{}{
				"Key-One": "value1",
				"Key-Two": "value2",
			},
			expected: map[string]interface{}{
				"keyone": "value1",
				"keytwo": "value2",
			},
		},
		{
			name: "Nested keys",
			input: map[string]interface{}{
				"Key-One": map[string]interface{}{
					"Nested-Key": "nestedValue",
				},
			},
			expected: map[string]interface{}{
				"keyone": map[string]interface{}{
					"nestedkey": "nestedValue",
				},
			},
		},
		{
			name: "Mixed keys",
			input: map[string]interface{}{
				"Key-One": "value1",
				"Key-Two": map[string]interface{}{
					"Nested-Key": "nestedValue",
				},
			},
			expected: map[string]interface{}{
				"keyone": "value1",
				"keytwo": map[string]interface{}{
					"nestedkey": "nestedValue",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := normaliseKeys(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestSetFlattenedKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "Simple keys",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expected: map[string]interface{}{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
		{
			name: "Nested keys",
			input: map[string]interface{}{
				"key1": map[string]interface{}{
					"nestedKey": "nestedValue",
				},
			},
			expected: map[string]interface{}{
				"KEY1_NESTEDKEY": "nestedValue",
			},
		},
		{
			name: "Mixed keys",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"nestedKey": "nestedValue",
				},
			},
			expected: map[string]interface{}{
				"KEY1":           "value1",
				"KEY2_NESTEDKEY": "nestedValue",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			viper.Reset()
			setFlattenedKeys(test.input, "")
			for key, expectedValue := range test.expected {
				assert.Equal(t, expectedValue, viper.Get(key))
			}
		})
	}
}
