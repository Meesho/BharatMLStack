package configutils

import (
	"reflect"
	"testing"
)

func TestNestedMapToPathMap(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]interface{}
		currentPath string
		expected    map[string]string
	}{
		{
			name:        "Empty map",
			input:       map[string]interface{}{},
			currentPath: "",
			expected:    map[string]string{},
		},
		{
			name: "Single level map with various types",
			input: map[string]interface{}{
				"string":   "value",
				"int":      42,
				"bool":     true,
				"float64":  3.14,
				"int64":    int64(12),
				"float32":  float32(1.5),
				"uint":     uint(10),
				"uint64":   uint64(15),
				"bytes":    []byte("byte array"),
				"nilValue": nil,
			},
			currentPath: "",
			expected: map[string]string{
				"/string":   "value",
				"/int":      "42",
				"/bool":     "true",
				"/float64":  "3.14",
				"/int64":    "12",
				"/float32":  "1.5",
				"/uint":     "10",
				"/uint64":   "15",
				"/bytes":    "byte array",
				"/nilValue": "",
			},
		},
		{
			name: "Nested map with multiple levels",
			input: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2a": "value",
					"level2b": map[string]interface{}{
						"level3": 123,
					},
				},
				"simple": "top-level",
			},
			currentPath: "",
			expected: map[string]string{
				"/level1/level2a":        "value",
				"/level1/level2b/level3": "123",
				"/simple":                "top-level",
			},
		},
		{
			name: "Map with custom path prefix",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"nested": "value2",
				},
			},
			currentPath: "/prefix",
			expected: map[string]string{
				"/prefix/key1":        "value1",
				"/prefix/key2/nested": "value2",
			},
		},
		{
			name: "Complex types",
			input: map[string]interface{}{
				"array":  []interface{}{1, "two", 3.0},
				"struct": struct{ Name string }{"test"},
				"map":    map[string]int{"one": 1, "two": 2},
			},
			currentPath: "",
			expected: map[string]string{
				"/array":  `[1,"two",3]`,
				"/struct": `{"Name":"test"}`,
				"/map":    `{"one":1,"two":2}`,
			},
		},
		{
			name: "Special values",
			input: map[string]interface{}{
				"complex": complex(1, 2),
			},
			currentPath: "",
			expected: map[string]string{
				"/complex": "(1+2i)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]string)
			NestedMapToPathMap(tt.input, tt.currentPath, result)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("NestedMapToPathMap() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNormalizePathMap(t *testing.T) {
	tests := []struct {
		name                   string
		originalData           map[string]string
		initialMeta            map[string]string
		expectedNormalizedData map[string]string
		expectedMeta           map[string]string
	}{
		{
			name:                   "EmptyMaps_ShouldRemainEmpty",
			originalData:           map[string]string{},
			initialMeta:            map[string]string{},
			expectedNormalizedData: map[string]string{},
			expectedMeta:           map[string]string{},
		},
		{
			name: "AlreadyNormalizedKeys_ShouldStayUnchanged",
			originalData: map[string]string{
				"lowercase": "value1",
				"nodashes":  "value2",
			},
			initialMeta: map[string]string{},
			expectedNormalizedData: map[string]string{
				"lowercase": "value1",
				"nodashes":  "value2",
			},
			expectedMeta: map[string]string{
				"lowercase": "lowercase",
				"nodashes":  "nodashes",
			},
		},
		{
			name: "UppercaseKeys_ShouldBeLowercased",
			originalData: map[string]string{
				"UPPERCASE": "value1",
				"CamelCase": "value2",
				"lowercase": "value3",
			},
			initialMeta: map[string]string{},
			expectedNormalizedData: map[string]string{
				"uppercase": "value1",
				"camelcase": "value2",
				"lowercase": "value3",
			},
			expectedMeta: map[string]string{
				"uppercase": "UPPERCASE",
				"camelcase": "CamelCase",
				"lowercase": "lowercase",
			},
		},
		{
			name: "KeysWithDashes_ShouldHaveDashesRemoved",
			originalData: map[string]string{
				"single-dash":      "value1",
				"multiple-dash-es": "value2",
				"nodashes":         "value3",
			},
			initialMeta: map[string]string{},
			expectedNormalizedData: map[string]string{
				"singledash":     "value1",
				"multipledashes": "value2",
				"nodashes":       "value3",
			},
			expectedMeta: map[string]string{
				"singledash":     "single-dash",
				"multipledashes": "multiple-dash-es",
				"nodashes":       "nodashes",
			},
		},
		{
			name: "MixedCaseAndDashes_ShouldBeFullyNormalized",
			originalData: map[string]string{
				"Mixed-Case":            "value1",
				"ALL-CAPS-WITH-DASHES":  "value2",
				"camelCase-with-Dashes": "value3",
			},
			initialMeta: map[string]string{},
			expectedNormalizedData: map[string]string{
				"mixedcase":           "value1",
				"allcapswithdashes":   "value2",
				"camelcasewithdashes": "value3",
			},
			expectedMeta: map[string]string{
				"mixedcase":           "Mixed-Case",
				"allcapswithdashes":   "ALL-CAPS-WITH-DASHES",
				"camelcasewithdashes": "camelCase-with-Dashes",
			},
		},
		{
			name: "ExistingMetaEntries_ShouldBePreserved",
			originalData: map[string]string{
				"New-Key":     "new value",
				"Another-Key": "another value",
			},
			initialMeta: map[string]string{
				"existingkey": "Original-Key",
				"anotherkey":  "Another-Original",
			},
			expectedNormalizedData: map[string]string{
				"newkey":     "new value",
				"anotherkey": "another value",
			},
			expectedMeta: map[string]string{
				"existingkey": "Original-Key",
				"anotherkey":  "Another-Key",
				"newkey":      "New-Key",
			},
		},
		{
			name: "ConfigurationPaths_ShouldBeNormalizedCorrectly",
			originalData: map[string]string{
				"/Service/Database/Host-Name":   "db.example.com",
				"/Service/Database/Port-Number": "5432",
				"/Service/API-Keys/Primary":     "secret-key",
			},
			initialMeta: map[string]string{},
			expectedNormalizedData: map[string]string{
				"/service/database/hostname":   "db.example.com",
				"/service/database/portnumber": "5432",
				"/service/apikeys/primary":     "secret-key",
			},
			expectedMeta: map[string]string{
				"/service/database/hostname":   "/Service/Database/Host-Name",
				"/service/database/portnumber": "/Service/Database/Port-Number",
				"/service/apikeys/primary":     "/Service/API-Keys/Primary",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualDataMap := copyStringMap(tc.originalData)
			actualMetaMap := copyStringMap(tc.initialMeta)

			NormalizePathMap(actualDataMap, actualMetaMap)

			if !reflect.DeepEqual(actualDataMap, tc.expectedNormalizedData) {
				t.Errorf("Data map normalization incorrect:\nGot:  %v\nWant: %v",
					actualDataMap, tc.expectedNormalizedData)
			}

			if !reflect.DeepEqual(actualMetaMap, tc.expectedMeta) {
				t.Errorf("Meta map mapping incorrect:\nGot:  %v\nWant: %v",
					actualMetaMap, tc.expectedMeta)
			}
		})
	}
}

func copyStringMap(original map[string]string) map[string]string {
	copy := make(map[string]string, len(original))
	for k, v := range original {
		copy[k] = v
	}
	return copy
}
