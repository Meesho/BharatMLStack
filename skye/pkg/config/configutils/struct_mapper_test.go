package configutils

import (
	"reflect"
	"testing"
)

func TestMapToStruct_SimpleStruct(t *testing.T) {
	resetHandledPrefix()
	type SimpleConfig struct {
		Host    string
		Port    int
		Enabled bool
	}

	dataMap := map[string]string{
		"/host":    "localhost",
		"/port":    "8080",
		"/enabled": "true",
	}
	metaMap := map[string]string{
		"/host":    "/host",
		"/port":    "/port",
		"/enabled": "/enabled",
	}

	config := SimpleConfig{}
	err := MapToStruct(&dataMap, &metaMap, &config, "")

	if err != nil {
		t.Fatalf("MapToStruct returned error: %v", err)
	}

	if config.Host != "localhost" {
		t.Errorf("Expected Host to be 'localhost', got '%s'", config.Host)
	}
	if config.Port != 8080 {
		t.Errorf("Expected Port to be 8080, got %d", config.Port)
	}
	if !config.Enabled {
		t.Errorf("Expected Enabled to be true, got %v", config.Enabled)
	}
	resetHandledPrefix()
}

func TestMapToStruct_NestedStruct(t *testing.T) {
	type DatabaseConfig struct {
		Username string
		Password string
		Host     string
		Port     int
	}

	type AppConfig struct {
		Name     string
		Version  string
		Database DatabaseConfig
	}

	dataMap := map[string]string{
		"/name":              "app",
		"/version":           "1",
		"/database/username": "admin",
		"/database/password": "secret",
		"/database/host":     "db.example.com",
		"/database/port":     "5432",
	}
	metaMap := map[string]string{
		"/name":              "/name",
		"/version":           "/version",
		"/database/username": "/Database/username",
		"/database/password": "/Database/password",
		"/database/host":     "/Database/host",
		"/database/port":     "/Database/port",
	}

	config := AppConfig{}
	err := MapToStruct(&dataMap, &metaMap, &config, "")

	if err != nil {
		t.Fatalf("MapToStruct returned error: %v", err)
	}

	if config.Name != "app" {
		t.Errorf("Expected Name to be 'app', got '%s'", config.Name)
	}
	if config.Version != "1" {
		t.Errorf("Expected Version to be '1', got '%s'", config.Version)
	}
	if config.Database.Username != "admin" {
		t.Errorf("Expected Database.Username to be 'admin', got '%s'", config.Database.Username)
	}
	if config.Database.Port != 5432 {
		t.Errorf("Expected Database.Port to be 5432, got %d", config.Database.Port)
	}
	resetHandledPrefix()
}

func TestMapToStruct_WithMap(t *testing.T) {
	type ConfigWithMap struct {
		Name       string
		Properties map[string]string
	}

	dataMap := map[string]string{
		"/name":       "MapTest",
		"/properties": `{"key1":"value1","key2":"value2"}`,
	}
	metaMap := map[string]string{
		"/name":       "/name",
		"/properties": "/properties",
	}

	config := ConfigWithMap{}
	err := MapToStruct(&dataMap, &metaMap, &config, "")

	if err != nil {
		t.Fatalf("MapToStruct returned error: %v", err)
	}

	if config.Name != "MapTest" {
		t.Errorf("Expected Name to be 'MapTest', got '%s'", config.Name)
	}

	expectedMap := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	if !reflect.DeepEqual(config.Properties, expectedMap) {
		t.Errorf("Expected Properties to be %v, got %v", expectedMap, config.Properties)
	}
	resetHandledPrefix()
}

func TestMapToStruct_WithNestedMap(t *testing.T) {
	type ConfigWithNestedMap struct {
		Settings map[string]map[string]string
	}

	dataMap := map[string]string{
		"/settings/database/host": "localhost",
		"/settings/database/port": "5432",
		"/settings/app/name":      "app",
	}
	metaMap := map[string]string{
		"/settings/database/host": "/settings/database/host",
		"/settings/database/port": "/settings/database/port",
		"/settings/app/name":      "/settings/app/name",
	}

	config := ConfigWithNestedMap{
		Settings: make(map[string]map[string]string),
	}
	err := MapToStruct(&dataMap, &metaMap, &config, "")

	if err != nil {
		t.Fatalf("MapToStruct returned error: %v", err)
	}

	if config.Settings["database"] == nil {
		t.Fatalf("Expected Settings to have 'database' key")
	}
	if config.Settings["database"]["host"] != "localhost" {
		t.Errorf("Expected database host to be 'localhost', got '%s'", config.Settings["database"]["host"])
	}
	if config.Settings["database"]["port"] != "5432" {
		t.Errorf("Expected database port to be '5432', got '%s'", config.Settings["database"]["port"])
	}
	if config.Settings["app"] == nil {
		t.Fatalf("Expected Settings to have 'app' key")
	}
	if config.Settings["app"]["name"] != "app" {
		t.Errorf("Expected app name to be 'app', got '%s'", config.Settings["app"]["name"])
	}
	resetHandledPrefix()
}

func TestMapToStruct_DifferentTypes(t *testing.T) {
	type AllTypesConfig struct {
		StringVal  string
		BoolVal    bool
		Int8Val    int8
		Int16Val   int16
		Int32Val   int32
		Int64Val   int64
		UintVal    uint
		Uint8Val   uint8
		Uint16Val  uint16
		Uint32Val  uint32
		Uint64Val  uint64
		Float32Val float32
		Float64Val float64
	}

	dataMap := map[string]string{
		"/stringval":  "test",
		"/boolval":    "true",
		"/int8val":    "42",
		"/int16val":   "1000",
		"/int32val":   "70000",
		"/int64val":   "9000000000",
		"/uintval":    "100",
		"/uint8val":   "200",
		"/uint16val":  "50000",
		"/uint32val":  "3000000000",
		"/uint64val":  "9000000000",
		"/float32val": "3.14",
		"/float64val": "2.71828",
	}
	metaMap := map[string]string{
		"/stringval":  "/StringVal",
		"/boolval":    "/BoolVal",
		"/int8val":    "/Int8Val",
		"/int16val":   "/Int16Val",
		"/int32val":   "/Int32Val",
		"/int64val":   "/Int64Val",
		"/uintval":    "/UintVal",
		"/uint8val":   "/Uint8Val",
		"/uint16val":  "/Uint16Val",
		"/uint32val":  "/Uint32Val",
		"/uint64val":  "/Uint64Val",
		"/float32val": "/Float32Val",
		"/float64val": "/Float64Val",
	}

	config := AllTypesConfig{}
	err := MapToStruct(&dataMap, &metaMap, &config, "")

	if err != nil {
		t.Fatalf("MapToStruct returned error: %v", err)
	}

	if config.StringVal != "test" {
		t.Errorf("Expected StringVal to be 'test', got '%s'", config.StringVal)
	}
	if !config.BoolVal {
		t.Errorf("Expected BoolVal to be true, got %v", config.BoolVal)
	}
	if config.Int8Val != 42 {
		t.Errorf("Expected Int8Val to be 42, got %d", config.Int8Val)
	}
	if config.Int16Val != 1000 {
		t.Errorf("Expected Int16Val to be 1000, got %d", config.Int16Val)
	}
	if config.Int32Val != 70000 {
		t.Errorf("Expected Int32Val to be 70000, got %d", config.Int32Val)
	}
	if config.Int64Val != 9000000000 {
		t.Errorf("Expected Int64Val to be 9000000000, got %d", config.Int64Val)
	}
	if config.UintVal != 100 {
		t.Errorf("Expected UintVal to be 100, got %d", config.UintVal)
	}
	if config.Uint8Val != 200 {
		t.Errorf("Expected Uint8Val to be 200, got %d", config.Uint8Val)
	}
	if config.Uint16Val != 50000 {
		t.Errorf("Expected Uint16Val to be 50000, got %d", config.Uint16Val)
	}
	if config.Uint32Val != 3000000000 {
		t.Errorf("Expected Uint32Val to be 3000000000, got %d", config.Uint32Val)
	}
	if config.Uint64Val != 9000000000 {
		t.Errorf("Expected Uint64Val to be 9000000000, got %d", config.Uint64Val)
	}
	if config.Float32Val != 3.14 {
		t.Errorf("Expected Float32Val to be 3.14, got %f", config.Float32Val)
	}
	if config.Float64Val != 2.71828 {
		t.Errorf("Expected Float64Val to be 2.71828, got %f", config.Float64Val)
	}
	resetHandledPrefix()
}

func TestMapToStruct_ErrorCases(t *testing.T) {
	dataMap := map[string]string{"key": "value"}
	metaMap := map[string]string{"key": "Key"}

	var nonPtr struct{}
	err := MapToStruct(&dataMap, &metaMap, nonPtr, "")
	if err == nil {
		t.Error("Expected error for non-pointer output, got nil")
	}

	var nilPtr *struct{}
	err = MapToStruct(&dataMap, &metaMap, nilPtr, "")
	if err == nil {
		t.Error("Expected error for nil pointer output, got nil")
	}

	var nonStruct string
	err = MapToStruct(&dataMap, &metaMap, &nonStruct, "")
	if err == nil {
		t.Error("Expected error for non-struct pointer output, got nil")
	}

	type TypeConversionErrors struct {
		BoolField  bool
		IntField   int
		FloatField float64
	}

	invalidDataMap := map[string]string{
		"/boolfield": "not-a-bool",
	}
	invalidMetaMap := map[string]string{
		"/boolfield": "/BoolField",
	}

	config := TypeConversionErrors{}
	err = MapToStruct(&invalidDataMap, &invalidMetaMap, &config, "")
	if err == nil {
		t.Error("Expected error for invalid bool conversion, got nil")
	}

	invalidDataMap = map[string]string{
		"/intfield": "not-an-int",
	}
	invalidMetaMap = map[string]string{
		"/intfield": "/IntField",
	}

	config = TypeConversionErrors{}
	err = MapToStruct(&invalidDataMap, &invalidMetaMap, &config, "")
	if err == nil {
		t.Error("Expected error for invalid int conversion, got nil")
	}

	invalidDataMap = map[string]string{
		"/floatfield": "not-a-float",
	}
	invalidMetaMap = map[string]string{
		"/floatfield": "/FloatField",
	}

	config = TypeConversionErrors{}
	err = MapToStruct(&invalidDataMap, &invalidMetaMap, &config, "")
	if err == nil {
		t.Error("Expected error for invalid float conversion, got nil")
	}

	type ConfigWithStruct struct {
		SubStruct struct {
			Field string
		}
	}

	invalidJSONMap := map[string]string{
		"/substruct": "{invalid-json}}",
	}
	jsonMetaMap := map[string]string{
		"/substruct": "/SubStruct",
	}

	structConfig := ConfigWithStruct{}
	err = MapToStruct(&invalidJSONMap, &jsonMetaMap, &structConfig, "")
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	type IntConfig struct {
		SmallInt int8
	}

	overflowMap := map[string]string{
		"/smallint": "1000",
	}
	overflowMetaMap := map[string]string{
		"/smallint": "/SmallInt",
	}

	intConfig := IntConfig{}
	err = MapToStruct(&overflowMap, &overflowMetaMap, &intConfig, "")
	if err == nil {
		t.Error("Expected error for int8 overflow, got nil")
	}

	type MapConfig struct {
		Properties map[string]string
	}

	invalidMapJSON := map[string]string{
		"/properties": "{not-valid-json}",
	}
	mapMetaMap := map[string]string{
		"/properties": "/Properties",
	}

	mapConfig := MapConfig{}
	err = MapToStruct(&invalidMapJSON, &mapMetaMap, &mapConfig, "")
	if err == nil {
		t.Error("Expected error for invalid map JSON, got nil")
	}
	resetHandledPrefix()
}

func TestMapToStruct_MapWithStructValues(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	type ConfigWithMapOfStructs struct {
		People map[string]Person
	}

	dataMap := map[string]string{
		"/people/john": `{"Name":"John Doe","Age":30}`,
		"/people/jane": `{"Name":"Jane Smith","Age":25}`,
	}
	metaMap := map[string]string{
		"/people/john": "/people/john",
		"/people/jane": "/people/jane",
	}

	config := ConfigWithMapOfStructs{
		People: make(map[string]Person),
	}
	err := MapToStruct(&dataMap, &metaMap, &config, "")

	if err != nil {
		t.Fatalf("MapToStruct returned error: %v", err)
	}

	if person, exists := config.People["john"]; !exists {
		t.Errorf("Expected People map to have 'john' key")
	} else {
		if person.Name != "John Doe" {
			t.Errorf("Expected john's Name to be 'John Doe', got '%s'", person.Name)
		}
		if person.Age != 30 {
			t.Errorf("Expected john's Age to be 30, got %d", person.Age)
		}
	}

	if person, exists := config.People["jane"]; !exists {
		t.Errorf("Expected People map to have 'jane' key")
	} else {
		if person.Name != "Jane Smith" {
			t.Errorf("Expected jane's Name to be 'Jane Smith', got '%s'", person.Name)
		}
		if person.Age != 25 {
			t.Errorf("Expected jane's Age to be 25, got %d", person.Age)
		}
	}
	resetHandledPrefix()
}

func TestMapToMap_Basic(t *testing.T) {
	dataMap := map[string]string{
		"/settings/database/host": "localhost",
		"/settings/database/port": "5432",
		"/settings/app/name":      "myapp",
	}
	metaMap := map[string]string{
		"/settings/database/host": "/settings/database/host",
		"/settings/database/port": "/settings/database/port",
		"/settings/app/name":      "/settings/app/name",
	}

	var result interface{} = make(map[string]map[string]string)
	err := mapToMap(&dataMap, &metaMap, &result, "/settings")

	if err != nil {
		t.Fatalf("mapToMap returned error: %v", err)
	}

	resultMap := result.(map[string]map[string]string)

	resetHandledPrefix()

	if database, exists := resultMap["database"]; !exists {
		t.Error("Expected 'database' key in resultMap")
	} else {
		if database["host"] != "localhost" {
			t.Errorf("Expected database/host to be 'localhost', got '%s'", database["host"])
		}
		if database["port"] != "5432" {
			t.Errorf("Expected database/port to be '5432', got '%s'", database["port"])
		}
	}

	if app, exists := resultMap["app"]; !exists {
		t.Error("Expected 'app' key in resultMap")
	} else {
		if app["name"] != "myapp" {
			t.Errorf("Expected app/name to be 'myapp', got '%s'", app["name"])
		}
	}
}

func TestMapToMap_DifferentValueTypes(t *testing.T) {
	for k := range handledPrefix {
		delete(handledPrefix, k)
	}

	t.Run("SuccessfulParsing", func(t *testing.T) {

		stringData := map[string]string{"/config/string": "test"}
		stringMeta := map[string]string{"/config/string": "/config/string"}
		var stringResult interface{} = make(map[string]string)
		err := mapToMap(&stringData, &stringMeta, &stringResult, "/config")
		if err != nil {
			t.Fatalf("mapToMap for string map returned error: %v", err)
		}
		stringMap := stringResult.(map[string]string)
		if stringMap["string"] != "test" {
			t.Errorf("Expected string value 'test', got '%s'", stringMap["string"])
		}

		resetHandledPrefix()

		boolData := map[string]string{"/config/bool": "true"}
		boolMeta := map[string]string{"/config/bool": "/config/bool"}
		var boolResult interface{} = make(map[string]bool)
		err = mapToMap(&boolData, &boolMeta, &boolResult, "/config")
		if err != nil {
			t.Fatalf("mapToMap for bool map returned error: %v", err)
		}
		boolMap := boolResult.(map[string]bool)
		if !boolMap["bool"] {
			t.Errorf("Expected bool value true, got %v", boolMap["bool"])
		}

		resetHandledPrefix()

		intData := map[string]string{"/config/int": "42"}
		intMeta := map[string]string{"/config/int": "/config/int"}
		var intResult interface{} = make(map[string]int64)
		err = mapToMap(&intData, &intMeta, &intResult, "/config")
		if err != nil {
			t.Fatalf("mapToMap for int map returned error: %v", err)
		}
		intMap := intResult.(map[string]int64)
		if intMap["int"] != 42 {
			t.Errorf("Expected int value 42, got %d", intMap["int"])
		}

		resetHandledPrefix()

		floatData := map[string]string{"/config/float": "3.14"}
		floatMeta := map[string]string{"/config/float": "/config/float"}
		var floatResult interface{} = make(map[string]float64)
		err = mapToMap(&floatData, &floatMeta, &floatResult, "/config")
		if err != nil {
			t.Fatalf("mapToMap for float map returned error: %v", err)
		}
		floatMap := floatResult.(map[string]float64)
		if floatMap["float"] != 3.14 {
			t.Errorf("Expected float value 3.14, got %f", floatMap["float"])
		}

		resetHandledPrefix()
	})

	t.Run("ParsingFailures", func(t *testing.T) {
		invalidBoolData := map[string]string{"/config/bool": "not-a-bool"}
		invalidBoolMeta := map[string]string{"/config/bool": "/config/bool"}
		var boolResult interface{} = make(map[string]bool)
		err := mapToMap(&invalidBoolData, &invalidBoolMeta, &boolResult, "/config")
		if err == nil {
			t.Error("Expected error when parsing invalid bool, got nil")
		}

		resetHandledPrefix()

		invalidIntData := map[string]string{"/config/int": "not-an-int"}
		invalidIntMeta := map[string]string{"/config/int": "/config/int"}
		var intResult interface{} = make(map[string]int)
		err = mapToMap(&invalidIntData, &invalidIntMeta, &intResult, "/config")
		if err == nil {
			t.Error("Expected error when parsing invalid int, got nil")
		}

		resetHandledPrefix()

		invalidFloatData := map[string]string{"/config/float": "not-a-float"}
		invalidFloatMeta := map[string]string{"/config/float": "/config/float"}
		var floatResult interface{} = make(map[string]float64)
		err = mapToMap(&invalidFloatData, &invalidFloatMeta, &floatResult, "/config")
		if err == nil {
			t.Error("Expected error when parsing invalid float, got nil")
		}

		resetHandledPrefix()

		overflowInt8Data := map[string]string{"/config/int8": "300"} // Overflows int8
		overflowInt8Meta := map[string]string{"/config/int8": "/config/int8"}
		var int8Result interface{} = make(map[string]int8)
		err = mapToMap(&overflowInt8Data, &overflowInt8Meta, &int8Result, "/config")
		if err == nil {
			t.Error("Expected error when int8 overflows, got nil")
		}

		resetHandledPrefix()

		negativeUintData := map[string]string{"/config/uint": "-10"}
		negativeUintMeta := map[string]string{"/config/uint": "/config/uint"}
		var uintResult interface{} = make(map[string]uint)
		err = mapToMap(&negativeUintData, &negativeUintMeta, &uintResult, "/config")
		if err == nil {
			t.Error("Expected error when parsing negative value as uint, got nil")
		}

		resetHandledPrefix()
	})
}

func TestMapToMap_MapValues(t *testing.T) {
	dataMap := map[string]string{
		"/config/map1": `{"key1":1,"key2":2}`,
		"/config/map2": `{"key3":3,"key4":4}`,
	}
	metaMap := map[string]string{
		"/config/map1": "/config/map1",
		"/config/map2": "/config/map2",
	}

	var result interface{} = make(map[string]map[string]int)
	err := mapToMap(&dataMap, &metaMap, &result, "/config")

	if err != nil {
		t.Fatalf("mapToMap returned error: %v", err)
	}

	resultMap := result.(map[string]map[string]int)

	expectedMap1 := map[string]int{"key1": 1, "key2": 2}
	expectedMap2 := map[string]int{"key3": 3, "key4": 4}

	if !reflect.DeepEqual(resultMap["map1"], expectedMap1) {
		t.Errorf("Expected map1 to be %v, got %v", expectedMap1, resultMap["map1"])
	}

	if !reflect.DeepEqual(resultMap["map2"], expectedMap2) {
		t.Errorf("Expected map2 to be %v, got %v", expectedMap2, resultMap["map2"])
	}
	resetHandledPrefix()
}

func TestMapToMap_ErrorCases(t *testing.T) {
	dataMap := map[string]string{
		"/config/key": "value",
	}
	metaMap := map[string]string{
		"/config/key": "/config/key",
	}

	resetHandledPrefix()

	var nonPtr map[string]string
	err := mapToMap(&dataMap, &metaMap, nonPtr, "/config")
	if err == nil {
		t.Error("Expected error for non-pointer output, got nil")
	}

	resetHandledPrefix()
	var nilPtr *map[string]string
	err = mapToMap(&dataMap, &metaMap, nilPtr, "/config")
	if err == nil {
		t.Error("Expected error for nil pointer, got nil")
	}

	resetHandledPrefix()
	nonInterface := "string"
	err = mapToMap(&dataMap, &metaMap, &nonInterface, "/config")
	if err == nil {
		t.Error("Expected error for non-interface pointer, got nil")
	}

	resetHandledPrefix()
	invalidBoolMap := map[string]string{
		"/config/bool": "not-a-bool",
	}
	invalidMetaMap := map[string]string{
		"/config/bool": "/config/bool",
	}

	var boolResult interface{} = make(map[string]bool)
	err = mapToMap(&invalidBoolMap, &invalidMetaMap, &boolResult, "/config")
	if err == nil {
		t.Error("Expected error for invalid bool conversion, got nil")
	}

	resetHandledPrefix()
	invalidIntMap := map[string]string{
		"/config/int": "not-an-int",
	}
	invalidMetaMap = map[string]string{
		"/config/int": "/config/int",
	}

	var intResult interface{} = make(map[string]int)
	err = mapToMap(&invalidIntMap, &invalidMetaMap, &intResult, "/config")
	if err == nil {
		t.Error("Expected error for invalid int conversion, got nil")
	}

	resetHandledPrefix()
	invalidFloatMap := map[string]string{
		"/config/float": "not-a-float",
	}
	invalidMetaMap = map[string]string{
		"/config/float": "/config/float",
	}

	var floatResult interface{} = make(map[string]float64)
	err = mapToMap(&invalidFloatMap, &invalidMetaMap, &floatResult, "/config")
	if err == nil {
		t.Error("Expected error for invalid float conversion, got nil")
	}

	invalidJSONMap := map[string]string{
		"/config/struct": "{invalid-json}",
	}
	invalidMetaMap = map[string]string{
		"/config/struct": "/config/struct",
	}

	type TestStruct struct {
		Field string
	}
	var structResult interface{} = make(map[string]TestStruct)
	err = mapToMap(&invalidJSONMap, &invalidMetaMap, &structResult, "/config")
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	incompleteMetaMap := map[string]string{}
	var result interface{} = make(map[string]string)
	err = mapToMap(&dataMap, &incompleteMetaMap, &result, "/config")
	if err == nil {
		t.Error("Expected error for missing metaMap entry, got nil")
	}
}

func TestMapToStruct_MapInsideStructAsJSON(t *testing.T) {
	for k := range handledPrefix {
		delete(handledPrefix, k)
	}

	type ConfigWithMapField struct {
		Name       string
		Properties map[string]int
	}

	type OuterConfig struct {
		Settings ConfigWithMapField
	}

	structWithMapJSON := `{"Name":"TestConfig","Properties":{"key1":1,"key2":2}}`

	dataMap := map[string]string{
		"/settings": structWithMapJSON,
	}

	metaMap := map[string]string{
		"/settings": "/settings",
	}

	config := OuterConfig{}
	err := MapToStruct(&dataMap, &metaMap, &config, "")

	if err != nil {
		t.Fatalf("MapToStruct returned error: %v", err)
	}

	if config.Settings.Name != "TestConfig" {
		t.Errorf("Expected Settings.Name to be 'TestConfig', got '%s'", config.Settings.Name)
	}

	expectedMap := map[string]int{
		"key1": 1,
		"key2": 2,
	}

	if !reflect.DeepEqual(config.Settings.Properties, expectedMap) {
		t.Errorf("Expected Settings.Properties to be %v, got %v", expectedMap, config.Settings.Properties)
	}

	resetHandledPrefix()
}

func TestMapToMap_StructWithMapAsJSON(t *testing.T) {
	for k := range handledPrefix {
		delete(handledPrefix, k)
	}

	type ConfigWithMapField struct {
		Name       string
		Properties map[string]string
	}

	structWithMapJSON := `{"Name":"TestConfig","Properties":{"key1":"value1","key2":"value2"}}`

	dataMap := map[string]string{
		"/config/item1": structWithMapJSON,
		"/config/item2": `{"Name":"AnotherConfig","Properties":{"key3":"value3","key4":"value4"}}`,
	}

	metaMap := map[string]string{
		"/config/item1": "/config/item1",
		"/config/item2": "/config/item2",
	}

	var result interface{} = make(map[string]ConfigWithMapField)
	err := mapToMap(&dataMap, &metaMap, &result, "/config")

	if err != nil {
		t.Fatalf("mapToMap returned error: %v", err)
	}

	resultMap := result.(map[string]ConfigWithMapField)

	if item, exists := resultMap["item1"]; !exists {
		t.Error("Expected 'item1' key in resultMap")
	} else {
		if item.Name != "TestConfig" {
			t.Errorf("Expected item1.Name to be 'TestConfig', got '%s'", item.Name)
		}

		expectedProperties := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		if !reflect.DeepEqual(item.Properties, expectedProperties) {
			t.Errorf("Expected item1.Properties to be %v, got %v", expectedProperties, item.Properties)
		}
	}

	if item, exists := resultMap["item2"]; !exists {
		t.Error("Expected 'item2' key in resultMap")
	} else {
		if item.Name != "AnotherConfig" {
			t.Errorf("Expected item2.Name to be 'AnotherConfig', got '%s'", item.Name)
		}

		expectedProperties := map[string]string{
			"key3": "value3",
			"key4": "value4",
		}

		if !reflect.DeepEqual(item.Properties, expectedProperties) {
			t.Errorf("Expected item2.Properties to be %v, got %v", expectedProperties, item.Properties)
		}
	}

	invalidJSONMap := map[string]string{
		"/config/invalid": "{invalid-json-format}",
	}
	invalidMetaMap := map[string]string{
		"/config/invalid": "/config/invalid",
	}

	var invalidResult interface{} = make(map[string]ConfigWithMapField)
	err = mapToMap(&invalidJSONMap, &invalidMetaMap, &invalidResult, "/config")

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	resetHandledPrefix()
}

func TestMapToMap_StructInsideMap(t *testing.T) {
	resetHandledPrefix()

	type ServerConfig struct {
		Host     string
		Port     int
		Username string
		Password string
		Active   bool
	}

	dataMap := map[string]string{
		"/servers/prod/host":     "prod.example.com",
		"/servers/prod/port":     "443",
		"/servers/prod/username": "admin",
		"/servers/prod/password": "prodpass",
		"/servers/prod/active":   "true",
		"/servers/dev/host":      "dev.example.com",
		"/servers/dev/port":      "8080",
		"/servers/dev/username":  "developer",
		"/servers/dev/password":  "devpass",
		"/servers/dev/active":    "false",
	}

	metaMap := map[string]string{
		"/servers/prod/host":     "/servers/prod/host",
		"/servers/prod/port":     "/servers/prod/port",
		"/servers/prod/username": "/servers/prod/username",
		"/servers/prod/password": "/servers/prod/password",
		"/servers/prod/active":   "/servers/prod/active",
		"/servers/dev/host":      "/servers/dev/host",
		"/servers/dev/port":      "/servers/dev/port",
		"/servers/dev/username":  "/servers/dev/username",
		"/servers/dev/password":  "/servers/dev/password",
		"/servers/dev/active":    "/servers/dev/active",
	}

	var result interface{} = make(map[string]ServerConfig)

	err := mapToMap(&dataMap, &metaMap, &result, "/servers")

	if err != nil {
		t.Fatalf("mapToMap returned error: %v", err)
	}

	resultMap := result.(map[string]ServerConfig)

	if server, exists := resultMap["prod"]; !exists {
		t.Error("Expected 'prod' key in resultMap")
	} else {
		if server.Host != "prod.example.com" {
			t.Errorf("Expected prod.Host to be 'prod.example.com', got '%s'", server.Host)
		}
		if server.Port != 443 {
			t.Errorf("Expected prod.Port to be 443, got %d", server.Port)
		}
		if server.Username != "admin" {
			t.Errorf("Expected prod.Username to be 'admin', got '%s'", server.Username)
		}
		if server.Password != "prodpass" {
			t.Errorf("Expected prod.Password to be 'prodpass', got '%s'", server.Password)
		}
		if !server.Active {
			t.Errorf("Expected prod.Active to be true, got %v", server.Active)
		}
	}

	if server, exists := resultMap["dev"]; !exists {
		t.Error("Expected 'dev' key in resultMap")
	} else {
		if server.Host != "dev.example.com" {
			t.Errorf("Expected dev.Host to be 'dev.example.com', got '%s'", server.Host)
		}
		if server.Port != 8080 {
			t.Errorf("Expected dev.Port to be 8080, got %d", server.Port)
		}
		if server.Username != "developer" {
			t.Errorf("Expected dev.Username to be 'developer', got '%s'", server.Username)
		}
		if server.Password != "devpass" {
			t.Errorf("Expected dev.Password to be 'devpass', got '%s'", server.Password)
		}
		if server.Active {
			t.Errorf("Expected dev.Active to be false, got %v", server.Active)
		}
	}

	resetHandledPrefix()
}

func resetHandledPrefix() {
	for k := range handledPrefix {
		delete(handledPrefix, k)
	}
}
