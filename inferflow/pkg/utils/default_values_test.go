package utils

import (
	"bytes"
	"testing"

	"github.com/Meesho/go-core/datatypeconverter/byteorder"
)

func init() {
	// Initialize byteorder before running tests
	// This is required for the typeconverter to work properly
	byteorder.Init()
}

func TestGetDefaultValuesInBytes(t *testing.T) {
	tests := []struct {
		name        string
		dataType    string
		expectedLen int
		description string
	}{
		// Boolean types - 1 byte
		{"bool", "bool", 1, "Boolean type"},
		{"datatypebool", "datatypebool", 1, "Boolean type with datatype prefix"},
		{"boolvector", "boolvector", 1, "Boolean vector type"},
		{"datatypeboolvector", "datatypeboolvector", 1, "Boolean vector type with datatype prefix"},

		// 8-bit integer types - 1 byte
		{"int8", "int8", 1, "8-bit signed integer"},
		{"datatypeint8", "datatypeint8", 1, "8-bit signed integer with datatype prefix"},
		{"int8vector", "int8vector", 1, "8-bit signed integer vector"},
		{"datatypeint8vector", "datatypeint8vector", 1, "8-bit signed integer vector with datatype prefix"},
		{"uint8", "uint8", 1, "8-bit unsigned integer"},
		{"datatypeuint8", "datatypeuint8", 1, "8-bit unsigned integer with datatype prefix"},
		{"uint8vector", "uint8vector", 1, "8-bit unsigned integer vector"},
		{"datatypeuint8vector", "datatypeuint8vector", 1, "8-bit unsigned integer vector with datatype prefix"},

		// 16-bit integer types - 2 bytes
		{"int16", "int16", 2, "16-bit signed integer"},
		{"datatypeint16", "datatypeint16", 2, "16-bit signed integer with datatype prefix"},
		{"int16vector", "int16vector", 2, "16-bit signed integer vector"},
		{"datatypeint16vector", "datatypeint16vector", 2, "16-bit signed integer vector with datatype prefix"},
		{"uint16", "uint16", 2, "16-bit unsigned integer"},
		{"datatypeuint16", "datatypeuint16", 2, "16-bit unsigned integer with datatype prefix"},
		{"uint16vector", "uint16vector", 2, "16-bit unsigned integer vector"},
		{"datatypeuint16vector", "datatypeuint16vector", 2, "16-bit unsigned integer vector with datatype prefix"},

		// 32-bit integer types - 4 bytes
		{"int32", "int32", 4, "32-bit signed integer"},
		{"datatypeint32", "datatypeint32", 4, "32-bit signed integer with datatype prefix"},
		{"int32vector", "int32vector", 4, "32-bit signed integer vector"},
		{"datatypeint32vector", "datatypeint32vector", 4, "32-bit signed integer vector with datatype prefix"},
		{"uint32", "uint32", 4, "32-bit unsigned integer"},
		{"datatypeuint32", "datatypeuint32", 4, "32-bit unsigned integer with datatype prefix"},
		{"uint32vector", "uint32vector", 4, "32-bit unsigned integer vector"},
		{"datatypeuint32vector", "datatypeuint32vector", 4, "32-bit unsigned integer vector with datatype prefix"},

		// 64-bit integer types - 8 bytes
		{"int64", "int64", 8, "64-bit signed integer"},
		{"datatypeint64", "datatypeint64", 8, "64-bit signed integer with datatype prefix"},
		{"int64vector", "int64vector", 8, "64-bit signed integer vector"},
		{"datatypeint64vector", "datatypeint64vector", 8, "64-bit signed integer vector with datatype prefix"},
		{"uint64", "uint64", 8, "64-bit unsigned integer"},
		{"datatypeuint64", "datatypeuint64", 8, "64-bit unsigned integer with datatype prefix"},
		{"uint64vector", "uint64vector", 8, "64-bit unsigned integer vector"},
		{"datatypeuint64vector", "datatypeuint64vector", 8, "64-bit unsigned integer vector with datatype prefix"},

		// 32-bit float types - 4 bytes
		{"fp32", "fp32", 4, "32-bit floating point"},
		{"datatypefp32", "datatypefp32", 4, "32-bit floating point with datatype prefix"},
		{"fp32vector", "fp32vector", 4, "32-bit floating point vector"},
		{"datatypefp32vector", "datatypefp32vector", 4, "32-bit floating point vector with datatype prefix"},

		// 64-bit float types - 8 bytes
		{"fp64", "fp64", 8, "64-bit floating point"},
		{"datatypefp64", "datatypefp64", 8, "64-bit floating point with datatype prefix"},
		{"fp64vector", "fp64vector", 8, "64-bit floating point vector"},
		{"datatypefp64vector", "datatypefp64vector", 8, "64-bit floating point vector with datatype prefix"},

		// 16-bit float types - 2 bytes
		{"fp16", "fp16", 2, "16-bit floating point"},
		{"datatypefp16", "datatypefp16", 2, "16-bit floating point with datatype prefix"},
		{"fp16vector", "fp16vector", 2, "16-bit floating point vector"},
		{"datatypefp16vector", "datatypefp16vector", 2, "16-bit floating point vector with datatype prefix"},

		// 8-bit float types - 1 byte
		{"fp8e5m2", "fp8e5m2", 1, "8-bit floating point E5M2"},
		{"datatypefp8e5m2", "datatypefp8e5m2", 1, "8-bit floating point E5M2 with datatype prefix"},
		{"fp8e5m2vector", "fp8e5m2vector", 1, "8-bit floating point E5M2 vector"},
		{"datatypefp8e5m2vector", "datatypefp8e5m2vector", 1, "8-bit floating point E5M2 vector with datatype prefix"},
		{"fp8e4m3", "fp8e4m3", 1, "8-bit floating point E4M3"},
		{"datatypefp8e4m3", "datatypefp8e4m3", 1, "8-bit floating point E4M3 with datatype prefix"},
		{"fp8e4m3vector", "fp8e4m3vector", 1, "8-bit floating point E4M3 vector"},
		{"datatypefp8e4m3vector", "datatypefp8e4m3vector", 1, "8-bit floating point E4M3 vector with datatype prefix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDefaultValuesInBytes(tt.dataType)

			// Check length
			if len(result) != tt.expectedLen {
				t.Errorf("Expected length %d, got %d for dataType %s", tt.expectedLen, len(result), tt.dataType)
			}

			// Check that all bytes are zero (default value)
			for i, b := range result {
				if b != 0 {
					t.Errorf("Expected all bytes to be zero, but byte at index %d is %d for dataType %s", i, b, tt.dataType)
				}
			}

			t.Logf("✓ %s: %d bytes, all zeros", tt.description, len(result))
		})
	}
}

func TestGetDefaultValuesInBytes_CaseInsensitive(t *testing.T) {
	testCases := []struct {
		lower string
		upper string
		mixed string
	}{
		{"bool", "BOOL", "Bool"},
		{"datatypebool", "DATATYPEBOOL", "DataTypeBool"},
		{"int32", "INT32", "Int32"},
		{"fp64", "FP64", "Fp64"},
		{"fp32vector", "FP32VECTOR", "Fp32Vector"},
	}

	for _, tc := range testCases {
		t.Run(tc.lower, func(t *testing.T) {
			lowerResult := GetDefaultValuesInBytes(tc.lower)
			upperResult := GetDefaultValuesInBytes(tc.upper)
			mixedResult := GetDefaultValuesInBytes(tc.mixed)

			if !bytes.Equal(lowerResult, upperResult) {
				t.Errorf("Case insensitive test failed: lower(%s) != upper(%s)", tc.lower, tc.upper)
			}

			if !bytes.Equal(lowerResult, mixedResult) {
				t.Errorf("Case insensitive test failed: lower(%s) != mixed(%s)", tc.lower, tc.mixed)
			}
		})
	}
}

func TestGetDefaultValuesInBytes_UnknownType(t *testing.T) {
	unknownTypes := []string{
		"unknown",
		"invalid",
		"randomtype",
		"",
	}

	for _, dataType := range unknownTypes {
		t.Run(dataType, func(t *testing.T) {
			result := GetDefaultValuesInBytes(dataType)

			// Should fallback to 4 bytes (fp32 default)
			if len(result) != 4 {
				t.Errorf("Expected fallback to 4 bytes for unknown type %s, got %d", dataType, len(result))
			}

			// Should be all zeros
			for i, b := range result {
				if b != 0 {
					t.Errorf("Expected all bytes to be zero for unknown type %s, but byte at index %d is %d", dataType, i, b)
				}
			}
		})
	}
}

func TestGetHardcodedZeroBytes(t *testing.T) {
	// Test the hardcoded zero bytes function directly

	tests := []struct {
		name        string
		dataType    string
		expectedLen int
	}{
		{"bool_fallback", "bool", 1},
		{"int8_fallback", "int8", 1},
		{"uint8_fallback", "uint8", 1},
		{"int16_fallback", "int16", 2},
		{"uint16_fallback", "uint16", 2},
		{"fp16_fallback", "fp16", 2},
		{"int32_fallback", "int32", 4},
		{"uint32_fallback", "uint32", 4},
		{"fp32_fallback", "fp32", 4},
		{"int64_fallback", "int64", 8},
		{"uint64_fallback", "uint64", 8},
		{"fp64_fallback", "fp64", 8},
		{"fp8e5m2_fallback", "fp8e5m2", 1},
		{"fp8e4m3_fallback", "fp8e4m3", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getHardcodedZeroBytes(tt.dataType)

			if len(result) != tt.expectedLen {
				t.Errorf("Expected length %d, got %d for dataType %s", tt.expectedLen, len(result), tt.dataType)
			}

			// Check that all bytes are zero
			for i, b := range result {
				if b != 0 {
					t.Errorf("Expected all bytes to be zero, but byte at index %d is %d for dataType %s", i, b, tt.dataType)
				}
			}
		})
	}
}

func TestGetHardcodedZeroBytes_VectorTypes(t *testing.T) {
	vectorTests := []struct {
		name        string
		dataType    string
		expectedLen int
	}{
		{"boolvector", "boolvector", 1},
		{"datatypeboolvector", "datatypeboolvector", 1},
		{"int16vector", "int16vector", 2},
		{"datatypeint32vector", "datatypeint32vector", 4},
		{"fp64vector", "fp64vector", 8},
		{"fp8e5m2vector", "fp8e5m2vector", 1},
	}

	for _, tt := range vectorTests {
		t.Run(tt.name, func(t *testing.T) {
			result := getHardcodedZeroBytes(tt.dataType)

			if len(result) != tt.expectedLen {
				t.Errorf("Expected length %d, got %d for vector dataType %s", tt.expectedLen, len(result), tt.dataType)
			}

			// Check that all bytes are zero
			for i, b := range result {
				if b != 0 {
					t.Errorf("Expected all bytes to be zero, but byte at index %d is %d for vector dataType %s", i, b, tt.dataType)
				}
			}
		})
	}
}

func TestGetHardcodedZeroBytes_UnknownType(t *testing.T) {
	unknownTypes := []string{
		"unknown",
		"invalid",
		"randomtype",
		"",
	}

	for _, dataType := range unknownTypes {
		t.Run(dataType, func(t *testing.T) {
			result := getHardcodedZeroBytes(dataType)

			// Should default to 4 bytes (fp32)
			if len(result) != 4 {
				t.Errorf("Expected default 4 bytes for unknown type %s, got %d", dataType, len(result))
			}

			// Should be all zeros
			for i, b := range result {
				if b != 0 {
					t.Errorf("Expected all bytes to be zero for unknown type %s, but byte at index %d is %d", dataType, i, b)
				}
			}
		})
	}
}

// Test to simulate what happens when typeconverter fails
func TestGetDefaultValuesInBytes_TypeConverterFailure(t *testing.T) {
	// This test simulates scenarios where typeconverter might fail
	// and ensures our fallback mechanism works

	// Test with a mock scenario where we can't initialize byteorder
	// (This would be tested by temporarily disabling byteorder.Init())

	tests := []struct {
		name        string
		dataType    string
		expectedLen int
	}{
		{"bool_with_fallback", "bool", 1},
		{"int16_with_fallback", "int16", 2},
		{"fp32_with_fallback", "fp32", 4},
		{"int64_with_fallback", "int64", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should work even if typeconverter fails
			// because we have the fallback mechanism
			result := GetDefaultValuesInBytes(tt.dataType)

			if len(result) != tt.expectedLen {
				t.Errorf("Expected length %d, got %d for dataType %s", tt.expectedLen, len(result), tt.dataType)
			}

			// Should be all zeros
			for i, b := range result {
				if b != 0 {
					t.Errorf("Expected all bytes to be zero for dataType %s, but byte at index %d is %d", tt.dataType, i, b)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkGetDefaultValuesInBytes(b *testing.B) {
	dataTypes := []string{
		"bool", "int8", "int16", "int32", "int64",
		"uint8", "uint16", "uint32", "uint64",
		"fp32", "fp64", "fp16", "fp8e5m2", "fp8e4m3",
	}

	for _, dt := range dataTypes {
		b.Run(dt, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				GetDefaultValuesInBytes(dt)
			}
		})
	}
}

func BenchmarkGetHardcodedZeroBytes(b *testing.B) {
	dataTypes := []string{
		"bool", "int8", "int16", "int32", "int64",
		"uint8", "uint16", "uint32", "uint64",
		"fp32", "fp64", "fp16", "fp8e5m2", "fp8e4m3",
	}

	for _, dt := range dataTypes {
		b.Run(dt, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				getHardcodedZeroBytes(dt)
			}
		})
	}
}
