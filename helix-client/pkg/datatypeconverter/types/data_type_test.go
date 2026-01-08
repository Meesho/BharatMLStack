package types

import "testing"

func TestDataTypeString(t *testing.T) {
	tests := []struct {
		dtype    DataType
		expected string
	}{
		{DataTypeFP8E5M2, "DataTypeFP8E5M2"},
		{DataTypeFP8E4M3, "DataTypeFP8E4M3"},
		{DataTypeFP16, "DataTypeFP16"},
		{DataTypeFP32, "DataTypeFP32"},
		{DataTypeFP64, "DataTypeFP64"},
		{DataTypeInt8, "DataTypeInt8"},
		{DataTypeInt16, "DataTypeInt16"},
		{DataTypeInt32, "DataTypeInt32"},
		{DataTypeInt64, "DataTypeInt64"},
		{DataTypeUint8, "DataTypeUint8"},
		{DataTypeUint16, "DataTypeUint16"},
		{DataTypeUint32, "DataTypeUint32"},
		{DataTypeUint64, "DataTypeUint64"},
		{DataTypeString, "DataTypeString"},
		{DataTypeBool, "DataTypeBool"},
		{DataTypeFP8E5M2Vector, "DataTypeFP8E5M2Vector"},
		{DataTypeFP8E4M3Vector, "DataTypeFP8E4M3Vector"},
		{DataTypeFP16Vector, "DataTypeFP16Vector"},
		{DataTypeFP32Vector, "DataTypeFP32Vector"},
		{DataTypeFP64Vector, "DataTypeFP64Vector"},
		{DataTypeInt8Vector, "DataTypeInt8Vector"},
		{DataTypeInt16Vector, "DataTypeInt16Vector"},
		{DataTypeInt32Vector, "DataTypeInt32Vector"},
		{DataTypeInt64Vector, "DataTypeInt64Vector"},
		{DataTypeUint8Vector, "DataTypeUint8Vector"},
		{DataTypeUint16Vector, "DataTypeUint16Vector"},
		{DataTypeUint32Vector, "DataTypeUint32Vector"},
		{DataTypeUint64Vector, "DataTypeUint64Vector"},
		{DataTypeStringVector, "DataTypeStringVector"},
		{DataTypeBoolVector, "DataTypeBoolVector"},
		{DataTypeUnknown, "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.dtype.String(); got != tt.expected {
			t.Errorf("DataType.String() for %v = %s, want %s", tt.dtype, got, tt.expected)
		}
	}
}

func TestDataTypeSize(t *testing.T) {
	tests := []struct {
		dtype    DataType
		expected int
	}{
		{DataTypeFP8E5M2, 1},
		{DataTypeFP8E4M3, 1},
		{DataTypeFP16, 2},
		{DataTypeFP32, 4},
		{DataTypeFP64, 8},
		{DataTypeInt8, 1},
		{DataTypeInt16, 2},
		{DataTypeInt32, 4},
		{DataTypeInt64, 8},
		{DataTypeUint8, 1},
		{DataTypeUint16, 2},
		{DataTypeUint32, 4},
		{DataTypeUint64, 8},
		{DataTypeString, 0},
		{DataTypeBool, 0},
		{DataTypeFP8E5M2Vector, 1},
		{DataTypeFP8E4M3Vector, 1},
		{DataTypeFP16Vector, 2},
		{DataTypeFP32Vector, 4},
		{DataTypeFP64Vector, 8},
		{DataTypeInt8Vector, 1},
		{DataTypeInt16Vector, 2},
		{DataTypeInt32Vector, 4},
		{DataTypeInt64Vector, 8},
		{DataTypeUint8Vector, 1},
		{DataTypeUint16Vector, 2},
		{DataTypeUint32Vector, 4},
		{DataTypeUint64Vector, 8},
		{DataTypeStringVector, 0},
		{DataTypeBoolVector, 0},
		{DataTypeUnknown, 0},
	}

	for _, tt := range tests {
		if got := tt.dtype.Size(); got != tt.expected {
			t.Errorf("DataType.Size() for %v = %d, want %d", tt.dtype, got, tt.expected)
		}
	}
}

func TestDataTypeIsVector(t *testing.T) {
	vectorTypes := []DataType{
		DataTypeFP8E5M2Vector,
		DataTypeFP8E4M3Vector,
		DataTypeFP16Vector,
		DataTypeFP32Vector,
		DataTypeFP64Vector,
		DataTypeInt8Vector,
		DataTypeInt16Vector,
		DataTypeInt32Vector,
		DataTypeInt64Vector,
		DataTypeUint8Vector,
		DataTypeUint16Vector,
		DataTypeUint32Vector,
		DataTypeUint64Vector,
		DataTypeStringVector,
		DataTypeBoolVector,
	}

	for _, vt := range vectorTypes {
		if !vt.IsVector() {
			t.Errorf("Expected %v to be vector type", vt)
		}
	}

	nonVectorTypes := []DataType{
		DataTypeFP8E5M2,
		DataTypeFP8E4M3,
		DataTypeFP16,
		DataTypeFP32,
		DataTypeFP64,
		DataTypeInt8,
		DataTypeInt16,
		DataTypeInt32,
		DataTypeInt64,
		DataTypeUint8,
		DataTypeUint16,
		DataTypeUint32,
		DataTypeUint64,
		DataTypeString,
		DataTypeBool,
	}

	for _, nt := range nonVectorTypes {
		if nt.IsVector() {
			t.Errorf("Expected %v to be non-vector type", nt)
		}
	}
}

func TestParseDataType(t *testing.T) {
	validTests := []struct {
		str      string
		expected DataType
	}{
		{"DataTypeFP8E5M2", DataTypeFP8E5M2},
		{"DataTypeFP8E4M3", DataTypeFP8E4M3},
		{"DataTypeFP16", DataTypeFP16},
		{"DataTypeFP32", DataTypeFP32},
		{"DataTypeFP64", DataTypeFP64},
		{"DataTypeInt8", DataTypeInt8},
		{"DataTypeInt16", DataTypeInt16},
		{"DataTypeInt32", DataTypeInt32},
		{"DataTypeInt64", DataTypeInt64},
		{"DataTypeUint8", DataTypeUint8},
		{"DataTypeUint16", DataTypeUint16},
		{"DataTypeUint32", DataTypeUint32},
		{"DataTypeUint64", DataTypeUint64},
		{"DataTypeString", DataTypeString},
		{"DataTypeBool", DataTypeBool},
		{"DataTypeFP8E5M2Vector", DataTypeFP8E5M2Vector},
		{"DataTypeFP8E4M3Vector", DataTypeFP8E4M3Vector},
		{"DataTypeFP16Vector", DataTypeFP16Vector},
		{"DataTypeFP32Vector", DataTypeFP32Vector},
		{"DataTypeFP64Vector", DataTypeFP64Vector},
		{"DataTypeInt8Vector", DataTypeInt8Vector},
		{"DataTypeInt16Vector", DataTypeInt16Vector},
		{"DataTypeInt32Vector", DataTypeInt32Vector},
		{"DataTypeInt64Vector", DataTypeInt64Vector},
		{"DataTypeUint8Vector", DataTypeUint8Vector},
		{"DataTypeUint16Vector", DataTypeUint16Vector},
		{"DataTypeUint32Vector", DataTypeUint32Vector},
		{"DataTypeUint64Vector", DataTypeUint64Vector},
		{"DataTypeStringVector", DataTypeStringVector},
		{"DataTypeBoolVector", DataTypeBoolVector},
	}

	for _, tt := range validTests {
		if got, err := ParseDataType(tt.str); err != nil || got != tt.expected {
			t.Errorf("ParseDataType(%s) = %v, %v; want %v, nil", tt.str, got, err, tt.expected)
		}
	}

	// Invalid case
	if _, err := ParseDataType("InvalidType"); err == nil {
		t.Error("Expected error for invalid data type string, got nil")
	}
}
