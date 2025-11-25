package typeconverter

import (
	"bytes"
	"math"
	"strconv"
	"testing"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/datatypeconverter/byteorder"
)

func TestMain(m *testing.M) {
	// Initialize the system before running tests
	byteorder.Init()
	// Run the tests
	m.Run()
}

// Helper functions to create test data
func createFloat32Bytes(f float32) []byte {
	buf := make([]byte, 4)
	byteorder.ByteOrder.PutFloat32(buf, f)
	return buf
}

func createFloat64Bytes(f float64) []byte {
	buf := make([]byte, 8)
	byteorder.ByteOrder.PutFloat64(buf, f)
	return buf
}

func createInt16Bytes(i int16) []byte {
	buf := make([]byte, 2)
	byteorder.ByteOrder.PutUint16(buf, uint16(i))
	return buf
}

func createInt32Bytes(i int32) []byte {
	buf := make([]byte, 4)
	byteorder.ByteOrder.PutUint32(buf, uint32(i))
	return buf
}

func createInt64Bytes(i int64) []byte {
	buf := make([]byte, 8)
	byteorder.ByteOrder.PutUint64(buf, uint64(i))
	return buf
}

func TestBytesToString(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		dataType string
		expected string
	}{
		{
			name:     "float32_simple",
			input:    createFloat32Bytes(3.14),
			dataType: "FP32",
			expected: "3.14",
		},
		{
			name:     "float64_pi",
			input:    createFloat64Bytes(math.Pi),
			dataType: "FP64",
			expected: "3.141592653589793",
		},
		// Integer scalars
		{
			name:     "int8_positive",
			input:    []byte{42},
			dataType: "Int8",
			expected: "42",
		},
		{
			name:     "int8_negative",
			input:    []byte{214},
			dataType: "Int8",
			expected: "-42",
		},
		{
			name:     "Int16",
			input:    createInt16Bytes(32000),
			dataType: "Int16",
			expected: "32000",
		},
		{
			name:     "Int32",
			input:    createInt32Bytes(-123456),
			dataType: "Int32",
			expected: "-123456",
		},
		{
			name:     "Int64",
			input:    createInt64Bytes(9223372036854775707),
			dataType: "Int64",
			expected: "9223372036854775707",
		},
		{
			name:     "Uint8",
			input:    []byte{255},
			dataType: "Uint8",
			expected: "255",
		},
		{
			name:     "Uint16",
			input:    func() []byte { b := make([]byte, 2); byteorder.ByteOrder.PutUint16(b, 65535); return b }(),
			dataType: "Uint16",
			expected: "65535",
		},
		// Floating-point special formats
		{
			name:     "FP16",
			input:    func() []byte { b := make([]byte, 2); byteorder.ByteOrder.PutFloat16FromFP32(b, 1.5); return b }(),
			dataType: "FP16",
			expected: "1.5",
		},
		{
			name:     "FP8E5M2",
			input:    func() []byte { b := []byte{0}; byteorder.ByteOrder.PutFloat8E5M2FromFP32(b, 0.03125); return b }(),
			dataType: "FP8E5M2",
			expected: "0.03125",
		},
		{
			name:     "FP8E4M3",
			input:    func() []byte { b := []byte{0}; byteorder.ByteOrder.PutFloat8E4M3FromFP32(b, 0.25); return b }(),
			dataType: "FP8E4M3",
			expected: "0.25",
		},
		{
			name:     "bool_true",
			input:    []byte{1},
			dataType: "Bool",
			expected: "true",
		},
		{
			name:     "bool_false",
			input:    []byte{0},
			dataType: "Bool",
			expected: "false",
		},
		{
			name:     "string_simple",
			input:    []byte("hello"),
			dataType: "String",
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BytesToString(tt.input, tt.dataType)
			if err != nil {
				t.Errorf("BytesToString() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("BytesToString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStringToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		dataType string
		validate func([]byte) bool
	}{
		{
			name:     "float32_conversion",
			input:    "3.14",
			dataType: "FP32",
			validate: func(b []byte) bool {
				if len(b) != 4 {
					return false
				}
				val := math.Float32frombits(byteorder.ByteOrder.Uint32(b))
				return math.Abs(float64(val-3.14)) < 0.0001
			},
		},
		{
			name:     "int8_conversion",
			input:    "42",
			dataType: "Int8",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 42
			},
		},
		{
			name:     "bool_true",
			input:    "true",
			dataType: "Bool",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 1
			},
		},
		{
			name:     "bool_false",
			input:    "false",
			dataType: "Bool",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 0
			},
		},
		{
			name:     "string_conversion",
			input:    "hello",
			dataType: "String",
			validate: func(b []byte) bool {
				return string(b) == "hello"
			},
		},
		{
			name:     "uint32_conversion",
			input:    "4000000000",
			dataType: "Uint32",
			validate: func(b []byte) bool {
				return len(b) == 4 && byteorder.ByteOrder.Uint32(b) == 4000000000
			},
		},
		{
			name:     "uint64_conversion",
			input:    "18446744073709551615",
			dataType: "Uint64",
			validate: func(b []byte) bool {
				return len(b) == 8 && byteorder.ByteOrder.Uint64(b) == 18446744073709551615
			},
		},
		{
			name:     "int16_conversion",
			input:    "-12345",
			dataType: "Int16",
			validate: func(b []byte) bool {
				return len(b) == 2 && byteorder.ByteOrder.Int16(b) == -12345
			},
		},
		{
			name:     "int32_conversion",
			input:    "-123456789",
			dataType: "Int32",
			validate: func(b []byte) bool {
				return len(b) == 4 && byteorder.ByteOrder.Int32(b) == -123456789
			},
		},
		{
			name:     "int64_conversion",
			input:    "9223372036854775707",
			dataType: "Int64",
			validate: func(b []byte) bool {
				return len(b) == 8 && byteorder.ByteOrder.Int64(b) == 9223372036854775707
			},
		},
		{
			name:     "uint8_conversion",
			input:    "255",
			dataType: "Uint8",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 255
			},
		},
		{
			name:     "uint16_conversion",
			input:    "65535",
			dataType: "Uint16",
			validate: func(b []byte) bool {
				return len(b) == 2 && byteorder.ByteOrder.Uint16(b) == 65535
			},
		},
		{
			name:     "fp64_conversion",
			input:    "3.141592653589793",
			dataType: "FP64",
			validate: func(b []byte) bool {
				if len(b) != 8 {
					return false
				}
				val := byteorder.ByteOrder.Float64(b)
				return math.Abs(val-3.141592653589793) < 1e-15
			},
		},
		{
			name:     "fp16_conversion",
			input:    "1.5",
			dataType: "FP16",
			validate: func(b []byte) bool {
				if len(b) != 2 {
					return false
				}
				val := byteorder.ByteOrder.Float16AsFP32(b)
				return math.Abs(float64(val-1.5)) < 0.01
			},
		},
		{
			name:     "fp8e5m2_conversion",
			input:    "0.03125",
			dataType: "FP8E5M2",
			validate: func(b []byte) bool {
				if len(b) != 1 {
					return false
				}
				val := byteorder.ByteOrder.Float8E5M2AsFP32(b)
				return math.Abs(float64(val-0.03125)) < 0.001
			},
		},
		{
			name:     "fp8e4m3_conversion",
			input:    "0.25",
			dataType: "FP8E4M3",
			validate: func(b []byte) bool {
				if len(b) != 1 {
					return false
				}
				val := byteorder.ByteOrder.Float8E4M3AsFP32(b)
				return math.Abs(float64(val-0.25)) < 0.001
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StringToBytes(tt.input, tt.dataType)
			if err != nil {
				t.Errorf("StringToBytes() error = %v", err)
				return
			}
			if !tt.validate(result) {
				t.Errorf("StringToBytes() validation failed for %v", tt.name)
			}
		})
	}
}

func TestVectorConversions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		dataType string
		expected string // Expected output after bytes to string conversion
		validate func([]byte) bool
	}{
		{
			name:     "float32_vector",
			input:    "1.0:2.0:3.0",
			dataType: "FP32Vector",
			expected: "1,2,3", // Float32 values are formatted without decimal places when they are whole numbers
			validate: func(b []byte) bool {
				if len(b) != 12 {
					return false
				}
				vals := make([]float32, 3)
				for i := 0; i < 3; i++ {
					vals[i] = math.Float32frombits(byteorder.ByteOrder.Uint32(b[i*4 : (i+1)*4]))
				}
				return vals[0] == 1.0 && vals[1] == 2.0 && vals[2] == 3.0
			},
		},
		{
			name:     "int8_vector",
			input:    "1:2:3",
			dataType: "Int8Vector",
			expected: "1,2,3",
			validate: func(b []byte) bool {
				return bytes.Equal(b, []byte{1, 2, 3})
			},
		},
		{
			name:     "bool_vector",
			input:    "true:false:true",
			dataType: "BoolVector",
			expected: "true,false,true",
			validate: func(b []byte) bool {
				return bytes.Equal(b, []byte{1, 0, 1})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test string to bytes conversion
			bytes, err := StringToBytes(tt.input, tt.dataType)
			if err != nil {
				t.Errorf("StringToBytes() error = %v", err)
				return
			}
			if !tt.validate(bytes) {
				t.Errorf("StringToBytes() validation failed for %v", tt.name)
			}

			// Test bytes to string conversion
			str, err := BytesToString(bytes, tt.dataType)
			if err != nil {
				t.Errorf("BytesToString() error = %v", err)
				return
			}
			// Compare with expected string
			if str != tt.expected {
				t.Errorf("BytesToString() = %v, want %v", str, tt.expected)
			}
		})
	}
}

func TestConvertBytesToBytes(t *testing.T) {
	tests := []struct {
		name       string
		input      []byte
		inputType  string
		outputType string
		validate   func([]byte) bool
	}{
		{
			name:       "int8_to_float32",
			input:      []byte{42},
			inputType:  "Int8",
			outputType: "FP32",
			validate: func(b []byte) bool {
				if len(b) != 4 {
					return false
				}
				val := math.Float32frombits(byteorder.ByteOrder.Uint32(b))
				return val == 42.0
			},
		},
		{
			name: "float32_to_int8",
			input: func() []byte {
				buf := make([]byte, 4)
				byteorder.ByteOrder.PutFloat32(buf, 42.7) // Should truncate to 42
				return buf
			}(),
			inputType:  "FP32",
			outputType: "Int8",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 42
			},
		},
		{
			name:       "bool_to_int8",
			input:      []byte{1},
			inputType:  "Bool",
			outputType: "Int8",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertBytesToBytes(tt.input, tt.inputType, tt.outputType)
			if err != nil {
				t.Errorf("ConvertBytesToBytes() error = %v", err)
				return
			}
			if !tt.validate(result) {
				t.Errorf("ConvertBytesToBytes() validation failed for %v", tt.name)
			}
		})
	}
}

func TestErrorCases(t *testing.T) {
	// Test nil input
	t.Run("nil_input", func(t *testing.T) {
		_, err := BytesToString(nil, "int8")
		if err == nil {
			t.Error("BytesToString() should return error for nil input")
		}
	})

	// Test invalid data type
	t.Run("invalid_data_type", func(t *testing.T) {
		_, err := BytesToString([]byte{1}, "invalid_type")
		if err == nil {
			t.Error("BytesToString() should return error for invalid data type")
		}
	})

	// Test insufficient bytes
	t.Run("insufficient_bytes", func(t *testing.T) {
		_, err := BytesToString([]byte{1}, "FP32") // Need 4 bytes for float32
		if err == nil {
			t.Error("BytesToString() should return error for insufficient bytes")
		}
	})

	// Test invalid string input
	t.Run("invalid_string_input", func(t *testing.T) {
		_, err := StringToBytes("not_a_number", "FP32")
		if err == nil {
			t.Error("StringToBytes() should return error for invalid string input")
		}
	})
}

func TestSpecialValues(t *testing.T) {
	tests := []struct {
		name     string
		value    float32
		dataType string
	}{
		{
			name:     "positive_infinity",
			value:    float32(math.Inf(1)),
			dataType: "FP32",
		},
		{
			name:     "negative_infinity",
			value:    float32(math.Inf(-1)),
			dataType: "FP32",
		},
		{
			name:     "nan",
			value:    float32(math.NaN()),
			dataType: "FP32",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes := createFloat32Bytes(tt.value)
			str, err := BytesToString(bytes, tt.dataType)
			if err != nil {
				t.Errorf("BytesToString() error = %v", err)
				return
			}
			if str == "" {
				t.Error("BytesToString() returned empty string for special value")
			}
		})
	}
}

func TestStringToFP32Conversion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float32
	}{
		{
			name:     "integer_string",
			input:    "1232",
			expected: 1232.0,
		},
		{
			name:     "decimal_string",
			input:    "0.1324321",
			expected: 0.1324321,
		},
		{
			name:     "negative_string",
			input:    "-123.456",
			expected: -123.456,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert string to fp32 bytes
			bytes, err := StringToBytes(tt.input, "FP32")
			if err != nil {
				t.Errorf("StringToBytes() error = %v", err)
				return
			}

			// Verify the bytes contain the correct float32 value
			if len(bytes) != 4 {
				t.Errorf("Expected 4 bytes for float32, got %d", len(bytes))
				return
			}

			// Convert bytes back to float32
			value := math.Float32frombits(byteorder.ByteOrder.Uint32(bytes))
			if math.Abs(float64(value-tt.expected)) > 1e-6 {
				t.Errorf("Got %v, want %v", value, tt.expected)
			}
		})
	}
}

func TestStringBytesToFP32Conversion(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected float32
	}{
		{
			name:     "integer_string_as_bytes",
			input:    []byte("1232"),
			expected: 1232.0,
		},
		{
			name:     "decimal_string_as_bytes",
			input:    []byte("0.1324321"),
			expected: 0.1324321,
		},
		{
			name:     "negative_string_as_bytes",
			input:    []byte("-123.456"),
			expected: -123.456,
		},
		{
			name:     "scientific_notation_as_bytes",
			input:    []byte("1.23e-4"),
			expected: 0.000123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert string bytes to fp32 bytes
			bytes, err := ConvertBytesToBytes(tt.input, "String", "FP32")
			if err != nil {
				t.Errorf("ConvertBytesToBytes() error = %v", err)
				return
			}

			// Verify the bytes contain the correct float32 value
			if len(bytes) != 4 {
				t.Errorf("Expected 4 bytes for float32, got %d", len(bytes))
				return
			}

			// Convert bytes back to float32
			value := math.Float32frombits(byteorder.ByteOrder.Uint32(bytes))
			if math.Abs(float64(value-tt.expected)) > 1e-6 {
				t.Errorf("Got %v, want %v", value, tt.expected)
			}
		})
	}
}

func BenchmarkConversions(b *testing.B) {
	float32Data := createFloat32Bytes(3.14159)
	int8Data := []byte{42}
	vectorData := make([]byte, 20) // 5 float32s
	for i := 0; i < 5; i++ {
		copy(vectorData[i*4:(i+1)*4], createFloat32Bytes(float32(i)))
	}

	b.Run("BytesToString_float32", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = BytesToString(float32Data, "FP32")
		}
	})

	b.Run("BytesToString_int8", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = BytesToString(int8Data, "Int8")
		}
	})

	b.Run("StringToBytes_float32", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = StringToBytes("3.14159", "FP32")
		}
	})

	b.Run("StringToBytes_vector", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = StringToBytes("1.0:2.0:3.0:4.0:5.0", "FP32Vector")
		}
	})

	b.Run("ConvertBytesToBytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = ConvertBytesToBytes(float32Data, "FP32", "Int8")
		}
	})
}

// =============================
// ADDITIONAL COVERAGE MERGED FROM EXTRA FILE
// =============================

func TestBytesToString_NumericScalarsAdditional(t *testing.T) {
	tests := []struct {
		name     string
		dtype    string
		bytes    []byte
		expected string
	}{
		{"int16_pos", "DataTypeInt16", func() []byte { b := make([]byte, 2); byteorder.ByteOrder.PutUint16(b, uint16(int16(1234))); return b }(), "1234"},
		{"int32_neg", "DataTypeInt32", func() []byte { b := make([]byte, 4); byteorder.ByteOrder.PutInt32(b, int32(-5678)); return b }(), "-5678"},
		{"int64", "DataTypeInt64", func() []byte {
			b := make([]byte, 8)
			byteorder.ByteOrder.PutUint64(b, uint64(int64(1234567890)))
			return b
		}(), "1234567890"},
		{"uint8", "DataTypeUint8", []byte{200}, "200"},
		{"uint16", "DataTypeUint16", func() []byte { b := make([]byte, 2); byteorder.ByteOrder.PutUint16(b, 65530); return b }(), "65530"},
		{"uint32", "DataTypeUint32", func() []byte { b := make([]byte, 4); byteorder.ByteOrder.PutUint32(b, 4294960000); return b }(), "4294960000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BytesToString(tt.bytes, tt.dtype)
			if err != nil {
				t.Fatalf("BytesToString error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("BytesToString got %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestBytesToString_FloatingPointSpecial(t *testing.T) {
	fp32Val := float32(6.2831) // 2π approx

	fp16Buf := make([]byte, 2)
	byteorder.ByteOrder.PutFloat16FromFP32(fp16Buf, fp32Val)
	s, err := BytesToString(fp16Buf, "FP16")
	if err != nil {
		t.Fatalf("BytesToString fp16 error: %v", err)
	}
	parsed, _ := strconv.ParseFloat(s, 32)
	if math.Abs(float64(float32(parsed)-fp32Val)) > 0.01 {
		t.Errorf("fp16 round-trip difference too large: got %s", s)
	}

	fp8Buf := []byte{0}
	byteorder.ByteOrder.PutFloat8E5M2FromFP32(fp8Buf, fp32Val)
	s2, err := BytesToString(fp8Buf, "FP8E5M2")
	if err != nil {
		t.Fatalf("BytesToString fp8 error: %v", err)
	}
	if s2 == "" {
		t.Errorf("expected non-empty string for fp8 value")
	}
}

func TestStringToBytes_NumericVectors(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		dtype  string
		verify func([]byte) bool
	}{
		{
			name:  "int16_vector",
			input: "10:-20:30",
			dtype: "Int16Vector",
			verify: func(b []byte) bool {
				if len(b) != 6 {
					return false
				}
				vals := byteorder.ByteOrder.Int16Vector(b)
				return vals[0] == 10 && vals[1] == -20 && vals[2] == 30
			},
		},
		{
			name:  "uint32_vector",
			input: "1:4294967295:1234",
			dtype: "Uint32Vector",
			verify: func(b []byte) bool {
				if len(b) != 12 {
					return false
				}
				vals := byteorder.ByteOrder.Uint32Vector(b)
				return vals[0] == 1 && vals[1] == 4294967295 && vals[2] == 1234
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := StringToBytes(tt.input, tt.dtype)
			if err != nil {
				t.Fatalf("StringToBytes error: %v", err)
			}
			if !tt.verify(out) {
				t.Errorf("verification failed for %s", tt.name)
			}
		})
	}
}

func TestStringToBytes_ErrorVectorLength(t *testing.T) {
	_, err := StringToBytes("not_an_int", "int32vector")
	if err == nil {
		t.Error("expected error for invalid int32vector element")
	}
}

func TestStringToBytes_AllVectorTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		dataType string
		validate func([]byte) bool
	}{
		{
			name:     "int64_vector",
			input:    "100:200:300",
			dataType: "Int64Vector",
			validate: func(b []byte) bool {
				if len(b) != 24 {
					return false
				}
				vals := byteorder.ByteOrder.Int64Vector(b)
				return vals[0] == 100 && vals[1] == 200 && vals[2] == 300
			},
		},
		{
			name:     "uint8_vector",
			input:    "1:2:255",
			dataType: "Uint8Vector",
			validate: func(b []byte) bool {
				return len(b) == 3 && b[0] == 1 && b[1] == 2 && b[2] == 255
			},
		},
		{
			name:     "uint16_vector",
			input:    "1000:2000:65535",
			dataType: "Uint16Vector",
			validate: func(b []byte) bool {
				if len(b) != 6 {
					return false
				}
				vals := byteorder.ByteOrder.Uint16AsUint32Vector(b)
				return vals[0] == 1000 && vals[1] == 2000 && vals[2] == 65535
			},
		},
		{
			name:     "uint64_vector",
			input:    "1:2:18446744073709551615",
			dataType: "Uint64Vector",
			validate: func(b []byte) bool {
				if len(b) != 24 {
					return false
				}
				vals := byteorder.ByteOrder.Uint64Vector(b)
				return vals[0] == 1 && vals[1] == 2 && vals[2] == 18446744073709551615
			},
		},
		{
			name:     "fp64_vector",
			input:    "1.1:2.2:3.3",
			dataType: "FP64Vector",
			validate: func(b []byte) bool {
				if len(b) != 24 {
					return false
				}
				vals := byteorder.ByteOrder.Float64Vector(b)
				return math.Abs(vals[0]-1.1) < 1e-10 &&
					math.Abs(vals[1]-2.2) < 1e-10 &&
					math.Abs(vals[2]-3.3) < 1e-10
			},
		},
		{
			name:     "fp16_vector",
			input:    "1.0:2.0:3.0",
			dataType: "FP16Vector",
			validate: func(b []byte) bool {
				if len(b) != 6 {
					return false
				}
				vals := byteorder.ByteOrder.FP16Vector(b)
				return math.Abs(float64(vals[0]-1.0)) < 0.01 &&
					math.Abs(float64(vals[1]-2.0)) < 0.01 &&
					math.Abs(float64(vals[2]-3.0)) < 0.01
			},
		},
		{
			name:     "fp8e5m2_vector",
			input:    "0.03125:0.0625:0.09375",
			dataType: "FP8E5M2Vector",
			validate: func(b []byte) bool {
				if len(b) != 3 {
					return false
				}
				vals := byteorder.ByteOrder.FP8E5M2Vector(b)
				return math.Abs(float64(vals[0]-0.03125)) < 0.001 &&
					math.Abs(float64(vals[1]-0.0625)) < 0.001 &&
					math.Abs(float64(vals[2]-0.09375)) < 0.001
			},
		},
		{
			name:     "fp8e4m3_vector",
			input:    "0.25:0.5:0.75",
			dataType: "FP8E4M3Vector",
			validate: func(b []byte) bool {
				if len(b) != 3 {
					return false
				}
				vals := byteorder.ByteOrder.FP8E4M3Vector(b)
				return math.Abs(float64(vals[0]-0.25)) < 0.001 &&
					math.Abs(float64(vals[1]-0.5)) < 0.001 &&
					math.Abs(float64(vals[2]-0.75)) < 0.001
			},
		},
		{
			name:     "string_vector",
			input:    "hello:world:test",
			dataType: "StringVector",
			validate: func(b []byte) bool {
				return string(b) == "hello,world,test"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StringToBytes(tt.input, tt.dataType)
			if err != nil {
				t.Fatalf("StringToBytes error: %v", err)
			}
			if !tt.validate(result) {
				t.Errorf("validation failed for %s", tt.name)
			}
		})
	}
}

// =============================================
// EDGE- and ERROR-PATH TESTS TO PUSH COVERAGE
// =============================================

func TestBytesToString_ErrorPaths(t *testing.T) {
	t.Run("unsupported_type", func(t *testing.T) {
		if _, err := BytesToString([]byte{1}, "unknown_type"); err == nil {
			t.Error("expected error for unsupported data type")
		}
	})

	t.Run("nil_input", func(t *testing.T) {
		if _, err := BytesToString(nil, "Int8"); err == nil {
			t.Error("expected error for nil slice")
		}
	})

	t.Run("wrong_length_scalar", func(t *testing.T) {
		// fp16 requires 2 bytes but we supply 3
		if _, err := BytesToString([]byte{0xAA, 0xBB, 0xCC}, "FP16"); err == nil {
			t.Error("expected length error for fp16")
		}
	})

	t.Run("wrong_length_vector", func(t *testing.T) {
		// int16vector requires even number of bytes
		if _, err := BytesToString([]byte{1, 2, 3}, "Int16Vector"); err == nil {
			t.Error("expected vector length error for int16vector")
		}
	})
}

func TestStringToBytes_ErrorPaths(t *testing.T) {
	t.Run("empty_value", func(t *testing.T) {
		if _, err := StringToBytes("", "Int8"); err == nil {
			t.Error("expected error for empty value string")
		}
	})

	t.Run("invalid_bool", func(t *testing.T) {
		if _, err := StringToBytes("maybe", "Bool"); err == nil {
			t.Error("expected parse error for invalid bool literal")
		}
	})

	t.Run("unsupported_type", func(t *testing.T) {
		if _, err := StringToBytes("42", "notatype"); err == nil {
			t.Error("expected error for unsupported data type")
		}
	})

	t.Run("vector_invalid_token", func(t *testing.T) {
		if _, err := StringToBytes("1:x:3", "Int8Vector"); err == nil {
			t.Error("expected parse error for bad vector token")
		}
	})

	t.Run("unsigned_negative", func(t *testing.T) {
		if _, err := StringToBytes("-1", "Uint16"); err == nil {
			t.Error("expected error for negative unsigned literal")
		}
	})
}

func TestConvertBytesToBytes_ErrorPaths(t *testing.T) {
	// nil input
	if _, err := ConvertBytesToBytes(nil, "Int8", "FP32"); err == nil {
		t.Error("expected error for nil input slice")
	}

	// identity convert should return same pointer
	original := []byte{0x2A}
	out, err := ConvertBytesToBytes(original, "Int8", "Int8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if &out[0] != &original[0] {
		t.Error("identity conversion should return original slice pointer")
	}

	// insufficient bytes inside bytesToNativeType (fp32 expects 4)
	if _, err := ConvertBytesToBytes([]byte{0xAA, 0xBB}, "FP32", "Int8"); err == nil {
		t.Error("expected error for insufficient bytes to parse fp32")
	}

	// unsupported output type triggers nativeTypeToBytes error
	if _, err := ConvertBytesToBytes([]byte{42}, "Int8", "StringVector"); err == nil {
		t.Error("expected error for unsupported output type")
	}
}

// =============================================
// COMPREHENSIVE TEST CASES FOR MISSING FUNCTIONALITY
// =============================================

// Test all data type aliases for BytesToString
func TestBytesToString_DataTypeAliases(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		dataType string
		expected string
	}{
		// Test all datatypeXXX aliases
		{
			name:     "datatypestring",
			input:    []byte("test"),
			dataType: "DataTypeString",
			expected: "test",
		},
		{
			name:     "datatypebool_true",
			input:    []byte{1},
			dataType: "DataTypeBool",
			expected: "true",
		},
		{
			name:     "datatypeint8",
			input:    []byte{127},
			dataType: "DataTypeInt8",
			expected: "127",
		},
		{
			name:     "datatypeint16",
			input:    createInt16Bytes(12345),
			dataType: "DataTypeInt16",
			expected: "12345",
		},
		{
			name:     "datatypeint32",
			input:    createInt32Bytes(-987654),
			dataType: "DataTypeInt32",
			expected: "-987654",
		},
		{
			name:     "datatypeint64",
			input:    createInt64Bytes(1234567890123),
			dataType: "DataTypeInt64",
			expected: "1234567890123",
		},
		{
			name:     "datatypeuint8",
			input:    []byte{255},
			dataType: "DataTypeUint8",
			expected: "255",
		},
		{
			name:     "datatypeuint16",
			input:    func() []byte { b := make([]byte, 2); byteorder.ByteOrder.PutUint16(b, 12345); return b }(),
			dataType: "DataTypeUint16",
			expected: "12345",
		},
		{
			name:     "datatypeuint32",
			input:    func() []byte { b := make([]byte, 4); byteorder.ByteOrder.PutUint32(b, 1234567890); return b }(),
			dataType: "DataTypeUint32",
			expected: "1234567890",
		},
		{
			name:     "datatypeuint64",
			input:    func() []byte { b := make([]byte, 8); byteorder.ByteOrder.PutUint64(b, 9876543210123456789); return b }(),
			dataType: "DataTypeUint64",
			expected: "9876543210123456789",
		},
		{
			name:     "datatypefp32",
			input:    createFloat32Bytes(2.718),
			dataType: "DataTypeFP32",
			expected: "2.718",
		},
		{
			name:     "datatypefp64",
			input:    createFloat64Bytes(2.718281828),
			dataType: "DataTypeFP64",
			expected: "2.718281828",
		},
		{
			name:     "datatypefp16",
			input:    func() []byte { b := make([]byte, 2); byteorder.ByteOrder.PutFloat16FromFP32(b, 2.5); return b }(),
			dataType: "DataTypeFP16",
			expected: "2.5",
		},
		{
			name:     "datatypefp8e5m2",
			input:    func() []byte { b := []byte{0}; byteorder.ByteOrder.PutFloat8E5M2FromFP32(b, 0.125); return b }(),
			dataType: "DataTypeFP8E5M2",
			expected: "0.125",
		},
		{
			name:     "datatypefp8e4m3",
			input:    func() []byte { b := []byte{0}; byteorder.ByteOrder.PutFloat8E4M3FromFP32(b, 0.5); return b }(),
			dataType: "DataTypeFP8E4M3",
			expected: "0.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BytesToString(tt.input, tt.dataType)
			if err != nil {
				t.Errorf("BytesToString() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("BytesToString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test all vector type aliases for BytesToString
func TestBytesToString_VectorTypeAliases(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		dataType string
		expected string
	}{
		{
			name:     "DataTypeBoolVector",
			input:    []byte{1, 0, 1, 0},
			dataType: "DataTypeBoolVector",
			expected: "true,false,true,false",
		},
		{
			name:     "DataTypeInt8Vector",
			input:    []byte{1, 2, 3, 4},
			dataType: "DataTypeInt8Vector",
			expected: "1,2,3,4",
		},
		{
			name: "DataTypeInt16Vector",
			input: func() []byte {
				b := make([]byte, 8)
				byteorder.ByteOrder.PutUint16(b[0:2], uint16(100))
				byteorder.ByteOrder.PutUint16(b[2:4], uint16(200))
				byteorder.ByteOrder.PutUint16(b[4:6], uint16(300))
				byteorder.ByteOrder.PutUint16(b[6:8], uint16(400))
				return b
			}(),
			dataType: "DataTypeInt16Vector",
			expected: "100,200,300,400",
		},
		{
			name: "DataTypeInt32Vector",
			input: func() []byte {
				b := make([]byte, 12)
				byteorder.ByteOrder.PutUint32(b[0:4], uint32(1000))
				byteorder.ByteOrder.PutUint32(b[4:8], uint32(2000))
				byteorder.ByteOrder.PutUint32(b[8:12], uint32(3000))
				return b
			}(),
			dataType: "DataTypeInt32Vector",
			expected: "1000,2000,3000",
		},
		{
			name: "DataTypeInt64Vector",
			input: func() []byte {
				b := make([]byte, 16)
				byteorder.ByteOrder.PutUint64(b[0:8], uint64(10000))
				byteorder.ByteOrder.PutUint64(b[8:16], uint64(20000))
				return b
			}(),
			dataType: "DataTypeInt64Vector",
			expected: "10000,20000",
		},
		{
			name:     "DataTypeUint8Vector",
			input:    []byte{100, 200, 255},
			dataType: "DataTypeUint8Vector",
			expected: "100,200,255",
		},
		{
			name: "DataTypeUint16Vector",
			input: func() []byte {
				b := make([]byte, 6)
				byteorder.ByteOrder.PutUint16(b[0:2], 1000)
				byteorder.ByteOrder.PutUint16(b[2:4], 2000)
				byteorder.ByteOrder.PutUint16(b[4:6], 3000)
				return b
			}(),
			dataType: "DataTypeUint16Vector",
			expected: "1000,2000,3000",
		},
		{
			name: "DataTypeUint32Vector",
			input: func() []byte {
				b := make([]byte, 8)
				byteorder.ByteOrder.PutUint32(b[0:4], 100000)
				byteorder.ByteOrder.PutUint32(b[4:8], 200000)
				return b
			}(),
			dataType: "DataTypeUint32Vector",
			expected: "100000,200000",
		},
		{
			name: "DataTypeUint64Vector",
			input: func() []byte {
				b := make([]byte, 16)
				byteorder.ByteOrder.PutUint64(b[0:8], 1000000)
				byteorder.ByteOrder.PutUint64(b[8:16], 2000000)
				return b
			}(),
			dataType: "DataTypeUint64Vector",
			expected: "1000000,2000000",
		},
		{
			name: "DataTypeFP32Vector",
			input: func() []byte {
				b := make([]byte, 12)
				byteorder.ByteOrder.PutFloat32(b[0:4], 1.5)
				byteorder.ByteOrder.PutFloat32(b[4:8], 2.5)
				byteorder.ByteOrder.PutFloat32(b[8:12], 3.5)
				return b
			}(),
			dataType: "DataTypeFP32Vector",
			expected: "1.5,2.5,3.5",
		},
		{
			name: "DataTypeFP64Vector",
			input: func() []byte {
				b := make([]byte, 16)
				byteorder.ByteOrder.PutFloat64(b[0:8], 1.123456789)
				byteorder.ByteOrder.PutFloat64(b[8:16], 2.987654321)
				return b
			}(),
			dataType: "DataTypeFP64Vector",
			expected: "1.123456789,2.987654321",
		},
		{
			name: "DataTypeFP16Vector",
			input: func() []byte {
				b := make([]byte, 6)
				byteorder.ByteOrder.PutFloat16FromFP32(b[0:2], 1.0)
				byteorder.ByteOrder.PutFloat16FromFP32(b[2:4], 2.0)
				byteorder.ByteOrder.PutFloat16FromFP32(b[4:6], 3.0)
				return b
			}(),
			dataType: "DataTypeFP16Vector",
			expected: "1,2,3",
		},
		{
			name: "DataTypeFP8E5M2Vector",
			input: func() []byte {
				b := make([]byte, 3)
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(b[0:1], 0.03125)
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(b[1:2], 0.0625)
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(b[2:3], 0.125)
				return b
			}(),
			dataType: "DataTypeFP8E5M2Vector",
			expected: "0.03125,0.0625,0.125",
		},
		{
			name: "DataTypeFP8E4M3Vector",
			input: func() []byte {
				b := make([]byte, 3)
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(b[0:1], 0.25)
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(b[1:2], 0.5)
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(b[2:3], 0.75)
				return b
			}(),
			dataType: "DataTypeFP8E4M3Vector",
			expected: "0.25,0.5,0.75",
		},
		{
			name:     "DataTypeStringVector",
			input:    []byte("hello,world,test"),
			dataType: "DataTypeStringVector",
			expected: "hello,world,test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BytesToString(tt.input, tt.dataType)
			if err != nil {
				t.Errorf("BytesToString() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("BytesToString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test all data type aliases for StringToBytes
func TestStringToBytes_DataTypeAliases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		dataType string
		validate func([]byte) bool
	}{
		{
			name:     "datatypestring",
			input:    "hello",
			dataType: "DataTypeString",
			validate: func(b []byte) bool {
				return string(b) == "hello"
			},
		},
		{
			name:     "datatypebool_true",
			input:    "true",
			dataType: "DataTypeBool",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 1
			},
		},
		{
			name:     "datatypebool_false",
			input:    "false",
			dataType: "DataTypeBool",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 0
			},
		},
		{
			name:     "datatypeint8",
			input:    "127",
			dataType: "DataTypeInt8",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 127
			},
		},
		{
			name:     "datatypeint16",
			input:    "12345",
			dataType: "DataTypeInt16",
			validate: func(b []byte) bool {
				return len(b) == 2 && byteorder.ByteOrder.Int16(b) == 12345
			},
		},
		{
			name:     "datatypeint32",
			input:    "-987654",
			dataType: "DataTypeInt32",
			validate: func(b []byte) bool {
				return len(b) == 4 && byteorder.ByteOrder.Int32(b) == -987654
			},
		},
		{
			name:     "datatypeint64",
			input:    "1234567890123",
			dataType: "DataTypeInt64",
			validate: func(b []byte) bool {
				return len(b) == 8 && byteorder.ByteOrder.Int64(b) == 1234567890123
			},
		},
		{
			name:     "datatypeuint8",
			input:    "255",
			dataType: "DataTypeUint8",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 255
			},
		},
		{
			name:     "datatypeuint16",
			input:    "12345",
			dataType: "DataTypeUint16",
			validate: func(b []byte) bool {
				return len(b) == 2 && byteorder.ByteOrder.Uint16(b) == 12345
			},
		},
		{
			name:     "datatypeuint32",
			input:    "1234567890",
			dataType: "DataTypeUint32",
			validate: func(b []byte) bool {
				return len(b) == 4 && byteorder.ByteOrder.Uint32(b) == 1234567890
			},
		},
		{
			name:     "datatypeuint64",
			input:    "9876543210123456789",
			dataType: "DataTypeUint64",
			validate: func(b []byte) bool {
				return len(b) == 8 && byteorder.ByteOrder.Uint64(b) == 9876543210123456789
			},
		},
		{
			name:     "datatypefp32",
			input:    "2.718",
			dataType: "DataTypeFP32",
			validate: func(b []byte) bool {
				if len(b) != 4 {
					return false
				}
				val := byteorder.ByteOrder.Float32(b)
				return math.Abs(float64(val-2.718)) < 0.0001
			},
		},
		{
			name:     "datatypefp64",
			input:    "2.718281828",
			dataType: "DataTypeFP64",
			validate: func(b []byte) bool {
				if len(b) != 8 {
					return false
				}
				val := byteorder.ByteOrder.Float64(b)
				return math.Abs(val-2.718281828) < 1e-9
			},
		},
		{
			name:     "datatypefp16",
			input:    "2.5",
			dataType: "DataTypeFP16",
			validate: func(b []byte) bool {
				if len(b) != 2 {
					return false
				}
				val := byteorder.ByteOrder.Float16AsFP32(b)
				return math.Abs(float64(val-2.5)) < 0.01
			},
		},
		{
			name:     "datatypefp8e5m2",
			input:    "0.125",
			dataType: "DataTypeFP8E5M2",
			validate: func(b []byte) bool {
				if len(b) != 1 {
					return false
				}
				val := byteorder.ByteOrder.Float8E5M2AsFP32(b)
				return math.Abs(float64(val-0.125)) < 0.001
			},
		},
		{
			name:     "datatypefp8e4m3",
			input:    "0.5",
			dataType: "DataTypeFP8E4M3",
			validate: func(b []byte) bool {
				if len(b) != 1 {
					return false
				}
				val := byteorder.ByteOrder.Float8E4M3AsFP32(b)
				return math.Abs(float64(val-0.5)) < 0.001
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StringToBytes(tt.input, tt.dataType)
			if err != nil {
				t.Errorf("StringToBytes() error = %v", err)
				return
			}
			if !tt.validate(result) {
				t.Errorf("StringToBytes() validation failed for %v", tt.name)
			}
		})
	}
}

// Test all vector type aliases for StringToBytes
func TestStringToBytes_VectorTypeAliases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		dataType string
		validate func([]byte) bool
	}{
		{
			name:     "DataTypeBoolVector",
			input:    "true:false:true:false",
			dataType: "DataTypeBoolVector",
			validate: func(b []byte) bool {
				return bytes.Equal(b, []byte{1, 0, 1, 0})
			},
		},
		{
			name:     "DataTypeInt8Vector",
			input:    "1:2:3:4",
			dataType: "DataTypeInt8Vector",
			validate: func(b []byte) bool {
				return bytes.Equal(b, []byte{1, 2, 3, 4})
			},
		},
		{
			name:     "DataTypeInt16Vector",
			input:    "100:200:300:400",
			dataType: "DataTypeInt16Vector",
			validate: func(b []byte) bool {
				if len(b) != 8 {
					return false
				}
				return byteorder.ByteOrder.Int16(b[0:2]) == 100 &&
					byteorder.ByteOrder.Int16(b[2:4]) == 200 &&
					byteorder.ByteOrder.Int16(b[4:6]) == 300 &&
					byteorder.ByteOrder.Int16(b[6:8]) == 400
			},
		},
		{
			name:     "DataTypeInt32Vector",
			input:    "1000:2000:3000",
			dataType: "DataTypeInt32Vector",
			validate: func(b []byte) bool {
				if len(b) != 12 {
					return false
				}
				return byteorder.ByteOrder.Int32(b[0:4]) == 1000 &&
					byteorder.ByteOrder.Int32(b[4:8]) == 2000 &&
					byteorder.ByteOrder.Int32(b[8:12]) == 3000
			},
		},
		{
			name:     "DataTypeInt64Vector",
			input:    "10000:20000",
			dataType: "DataTypeInt64Vector",
			validate: func(b []byte) bool {
				if len(b) != 16 {
					return false
				}
				return byteorder.ByteOrder.Int64(b[0:8]) == 10000 &&
					byteorder.ByteOrder.Int64(b[8:16]) == 20000
			},
		},
		{
			name:     "DataTypeUint8Vector",
			input:    "100:200:255",
			dataType: "DataTypeUint8Vector",
			validate: func(b []byte) bool {
				return bytes.Equal(b, []byte{100, 200, 255})
			},
		},
		{
			name:     "DataTypeUint16Vector",
			input:    "1000:2000:3000",
			dataType: "DataTypeUint16Vector",
			validate: func(b []byte) bool {
				if len(b) != 6 {
					return false
				}
				return byteorder.ByteOrder.Uint16(b[0:2]) == 1000 &&
					byteorder.ByteOrder.Uint16(b[2:4]) == 2000 &&
					byteorder.ByteOrder.Uint16(b[4:6]) == 3000
			},
		},
		{
			name:     "DataTypeUint32Vector",
			input:    "100000:200000",
			dataType: "DataTypeUint32Vector",
			validate: func(b []byte) bool {
				if len(b) != 8 {
					return false
				}
				return byteorder.ByteOrder.Uint32(b[0:4]) == 100000 &&
					byteorder.ByteOrder.Uint32(b[4:8]) == 200000
			},
		},
		{
			name:     "DataTypeUint64Vector",
			input:    "1000000:2000000",
			dataType: "DataTypeUint64Vector",
			validate: func(b []byte) bool {
				if len(b) != 16 {
					return false
				}
				return byteorder.ByteOrder.Uint64(b[0:8]) == 1000000 &&
					byteorder.ByteOrder.Uint64(b[8:16]) == 2000000
			},
		},
		{
			name:     "DataTypeFP32Vector",
			input:    "1.5:2.5:3.5",
			dataType: "DataTypeFP32Vector",
			validate: func(b []byte) bool {
				if len(b) != 12 {
					return false
				}
				return math.Abs(float64(byteorder.ByteOrder.Float32(b[0:4])-1.5)) < 0.0001 &&
					math.Abs(float64(byteorder.ByteOrder.Float32(b[4:8])-2.5)) < 0.0001 &&
					math.Abs(float64(byteorder.ByteOrder.Float32(b[8:12])-3.5)) < 0.0001
			},
		},
		{
			name:     "DataTypeFP64Vector",
			input:    "1.123456789:2.987654321",
			dataType: "DataTypeFP64Vector",
			validate: func(b []byte) bool {
				if len(b) != 16 {
					return false
				}
				return math.Abs(byteorder.ByteOrder.Float64(b[0:8])-1.123456789) < 1e-9 &&
					math.Abs(byteorder.ByteOrder.Float64(b[8:16])-2.987654321) < 1e-9
			},
		},
		{
			name:     "DataTypeFP16Vector",
			input:    "1.0:2.0:3.0",
			dataType: "DataTypeFP16Vector",
			validate: func(b []byte) bool {
				if len(b) != 6 {
					return false
				}
				return math.Abs(float64(byteorder.ByteOrder.Float16AsFP32(b[0:2])-1.0)) < 0.01 &&
					math.Abs(float64(byteorder.ByteOrder.Float16AsFP32(b[2:4])-2.0)) < 0.01 &&
					math.Abs(float64(byteorder.ByteOrder.Float16AsFP32(b[4:6])-3.0)) < 0.01
			},
		},
		{
			name:     "DataTypeFP8E5M2Vector",
			input:    "0.03125:0.0625:0.09375",
			dataType: "DataTypeFP8E5M2Vector",
			validate: func(b []byte) bool {
				if len(b) != 3 {
					return false
				}
				return math.Abs(float64(byteorder.ByteOrder.Float8E5M2AsFP32(b[0:1])-0.03125)) < 0.001 &&
					math.Abs(float64(byteorder.ByteOrder.Float8E5M2AsFP32(b[1:2])-0.0625)) < 0.001 &&
					math.Abs(float64(byteorder.ByteOrder.Float8E5M2AsFP32(b[2:3])-0.09375)) < 0.001
			},
		},
		{
			name:     "DataTypeFP8E4M3Vector",
			input:    "0.25:0.5:0.75",
			dataType: "DataTypeFP8E4M3Vector",
			validate: func(b []byte) bool {
				if len(b) != 3 {
					return false
				}
				return math.Abs(float64(byteorder.ByteOrder.Float8E4M3AsFP32(b[0:1])-0.25)) < 0.001 &&
					math.Abs(float64(byteorder.ByteOrder.Float8E4M3AsFP32(b[1:2])-0.5)) < 0.001 &&
					math.Abs(float64(byteorder.ByteOrder.Float8E4M3AsFP32(b[2:3])-0.75)) < 0.001
			},
		},
		{
			name:     "DataTypeStringVector",
			input:    "hello:world:test",
			dataType: "DataTypeStringVector",
			validate: func(b []byte) bool {
				return string(b) == "hello,world,test"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StringToBytes(tt.input, tt.dataType)
			if err != nil {
				t.Errorf("StringToBytes() error = %v", err)
				return
			}
			if !tt.validate(result) {
				t.Errorf("StringToBytes() validation failed for %v", tt.name)
			}
		})
	}
}

// Test comprehensive type conversions for ConvertBytesToBytes
func TestConvertBytesToBytes_ComprehensiveConversions(t *testing.T) {
	tests := []struct {
		name       string
		input      []byte
		inputType  string
		outputType string
		validate   func([]byte) bool
	}{
		// Bool conversions
		{
			name:       "bool_to_uint8",
			input:      []byte{1},
			inputType:  "Bool",
			outputType: "Uint8",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 1
			},
		},
		{
			name:       "bool_to_int16",
			input:      []byte{0},
			inputType:  "Bool",
			outputType: "Int16",
			validate: func(b []byte) bool {
				return len(b) == 2 && byteorder.ByteOrder.Int16(b) == 0
			},
		},
		{
			name:       "bool_to_fp32",
			input:      []byte{1},
			inputType:  "Bool",
			outputType: "FP32",
			validate: func(b []byte) bool {
				return len(b) == 4 && byteorder.ByteOrder.Float32(b) == 1.0
			},
		},
		{
			name:       "bool_to_fp64",
			input:      []byte{0},
			inputType:  "Bool",
			outputType: "FP64",
			validate: func(b []byte) bool {
				return len(b) == 8 && byteorder.ByteOrder.Float64(b) == 0.0
			},
		},
		// Int8 conversions
		{
			name:       "int8_to_bool",
			input:      []byte{42},
			inputType:  "Int8",
			outputType: "Bool",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 1
			},
		},
		{
			name:       "int8_to_uint16",
			input:      []byte{42},
			inputType:  "Int8",
			outputType: "Uint16",
			validate: func(b []byte) bool {
				return len(b) == 2 && byteorder.ByteOrder.Uint16(b) == 42
			},
		},
		{
			name:       "int8_to_int64",
			input:      []byte{42},
			inputType:  "Int8",
			outputType: "Int64",
			validate: func(b []byte) bool {
				return len(b) == 8 && byteorder.ByteOrder.Int64(b) == 42
			},
		},
		{
			name:       "int8_to_fp16",
			input:      []byte{42},
			inputType:  "Int8",
			outputType: "FP16",
			validate: func(b []byte) bool {
				return len(b) == 2 && math.Abs(float64(byteorder.ByteOrder.Float16AsFP32(b))-42.0) < 0.01
			},
		},
		{
			name:       "int8_to_fp8e5m2",
			input:      []byte{1},
			inputType:  "Int8",
			outputType: "FP8E5M2",
			validate: func(b []byte) bool {
				return len(b) == 1 && math.Abs(float64(byteorder.ByteOrder.Float8E5M2AsFP32(b))-1.0) < 0.01
			},
		},
		{
			name:       "int8_to_fp8e4m3",
			input:      []byte{1},
			inputType:  "Int8",
			outputType: "FP8E4M3",
			validate: func(b []byte) bool {
				return len(b) == 1 && math.Abs(float64(byteorder.ByteOrder.Float8E4M3AsFP32(b))-1.0) < 0.01
			},
		},
		// Float32 conversions
		{
			name: "fp32_to_bool",
			input: func() []byte {
				b := make([]byte, 4)
				byteorder.ByteOrder.PutFloat32(b, 3.14)
				return b
			}(),
			inputType:  "FP32",
			outputType: "Bool",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 1
			},
		},
		{
			name: "fp32_to_uint32",
			input: func() []byte {
				b := make([]byte, 4)
				byteorder.ByteOrder.PutFloat32(b, 42.7)
				return b
			}(),
			inputType:  "FP32",
			outputType: "Uint32",
			validate: func(b []byte) bool {
				return len(b) == 4 && byteorder.ByteOrder.Uint32(b) == 42
			},
		},
		{
			name: "fp32_to_fp64",
			input: func() []byte {
				b := make([]byte, 4)
				byteorder.ByteOrder.PutFloat32(b, 3.14)
				return b
			}(),
			inputType:  "FP32",
			outputType: "FP64",
			validate: func(b []byte) bool {
				return len(b) == 8 && math.Abs(byteorder.ByteOrder.Float64(b)-3.14) < 0.01
			},
		},
		// String conversions (numbers as strings)
		{
			name:       "string_to_int32",
			input:      []byte("12345"),
			inputType:  "DataTypeString",
			outputType: "Int32",
			validate: func(b []byte) bool {
				return len(b) == 4 && byteorder.ByteOrder.Int32(b) == 12345
			},
		},
		{
			name:       "string_to_fp32",
			input:      []byte("3.14159"),
			inputType:  "DataTypeString",
			outputType: "FP32",
			validate: func(b []byte) bool {
				return len(b) == 4 && math.Abs(float64(byteorder.ByteOrder.Float32(b))-3.14159) < 0.0001
			},
		},
		{
			name:       "string_to_uint64",
			input:      []byte("9876543210"),
			inputType:  "DataTypeString",
			outputType: "Uint64",
			validate: func(b []byte) bool {
				return len(b) == 8 && byteorder.ByteOrder.Uint64(b) == 9876543210
			},
		},
		// Cross-type conversions
		{
			name: "fp64_to_int8",
			input: func() []byte {
				b := make([]byte, 8)
				byteorder.ByteOrder.PutFloat64(b, 127.9)
				return b
			}(),
			inputType:  "FP64",
			outputType: "Int8",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 127
			},
		},
		{
			name: "uint64_to_fp16",
			input: func() []byte {
				b := make([]byte, 8)
				byteorder.ByteOrder.PutUint64(b, 42)
				return b
			}(),
			inputType:  "Uint64",
			outputType: "FP16",
			validate: func(b []byte) bool {
				return len(b) == 2 && math.Abs(float64(byteorder.ByteOrder.Float16AsFP32(b))-42.0) < 0.01
			},
		},
		{
			name: "fp16_to_uint8",
			input: func() []byte {
				b := make([]byte, 2)
				byteorder.ByteOrder.PutFloat16FromFP32(b, 200.5)
				return b
			}(),
			inputType:  "FP16",
			outputType: "Uint8",
			validate: func(b []byte) bool {
				return len(b) == 1 && b[0] == 200
			},
		},
		{
			name: "fp8e5m2_to_int32",
			input: func() []byte {
				b := make([]byte, 1)
				byteorder.ByteOrder.PutFloat8E5M2FromFP32(b, 2.0)
				return b
			}(),
			inputType:  "FP8E5M2",
			outputType: "Int32",
			validate: func(b []byte) bool {
				return len(b) == 4 && byteorder.ByteOrder.Int32(b) == 2
			},
		},
		{
			name: "fp8e4m3_to_fp64",
			input: func() []byte {
				b := make([]byte, 1)
				byteorder.ByteOrder.PutFloat8E4M3FromFP32(b, 1.5)
				return b
			}(),
			inputType:  "FP8E4M3",
			outputType: "FP64",
			validate: func(b []byte) bool {
				return len(b) == 8 && math.Abs(byteorder.ByteOrder.Float64(b)-1.5) < 0.01
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertBytesToBytes(tt.input, tt.inputType, tt.outputType)
			if err != nil {
				t.Errorf("ConvertBytesToBytes() error = %v", err)
				return
			}
			if !tt.validate(result) {
				t.Errorf("ConvertBytesToBytes() validation failed for %v", tt.name)
			}
		})
	}
}

// Test edge cases for string handling
func TestStringHandling_EdgeCases(t *testing.T) {
	// Test string that doesn't represent a number: should error
	t.Run("non_numeric_string", func(t *testing.T) {
		_, err := ConvertBytesToBytes([]byte("hello"), "DataTypeString", "Int32")
		if err == nil {
			t.Error("Expected error for non-numeric string conversion")
		}
	})

	// Test empty string
	t.Run("empty_string_bytes", func(t *testing.T) {
		_, err := ConvertBytesToBytes([]byte{}, "DataTypeString", "Int32")
		if err == nil {
			t.Error("Expected error for empty string conversion")
		}
	})
}

// Test case sensitivity
func TestCaseSensitivity(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		input    []byte
		expected bool // whether conversion should succeed
	}{
		{
			name:     "uppercase_FP32",
			dataType: "DataTypeFP32",
			input:    createFloat32Bytes(3.14),
			expected: true, // Should succeed (case-insensitive)
		},
		{
			name:     "mixed_case_Int8",
			dataType: "DataTypeInt8",
			input:    []byte{42},
			expected: true, // Should succeed
		},
		{
			name:     "correct_case_fp32",
			dataType: "DataTypeFP32",
			input:    createFloat32Bytes(3.14),
			expected: true, // Should succeed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BytesToString(tt.input, tt.dataType)
			if tt.expected && err != nil {
				t.Errorf("BytesToString() should succeed but got error: %v", err)
			}
			if !tt.expected && err == nil {
				t.Errorf("BytesToString() should fail but succeeded")
			}
		})
	}
}

// Test boundary values
func TestBoundaryValues(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		dataType  string
		shouldErr bool
	}{
		{
			name:      "int8_max",
			input:     "127",
			dataType:  "DataTypeInt8",
			shouldErr: false,
		},
		{
			name:      "int8_min",
			input:     "-128",
			dataType:  "DataTypeInt8",
			shouldErr: false,
		},
		{
			name:      "int8_overflow",
			input:     "128",
			dataType:  "DataTypeInt8",
			shouldErr: true, // Should error, value out of range
		},
		{
			name:      "uint8_max",
			input:     "255",
			dataType:  "DataTypeUint8",
			shouldErr: false,
		},
		{
			name:      "uint8_negative",
			input:     "-1",
			dataType:  "DataTypeUint8",
			shouldErr: true, // Should fail for negative uint
		},
		{
			name:      "float32_infinity",
			input:     "Inf",
			dataType:  "DataTypeFP32",
			shouldErr: false,
		},
		{
			name:      "float32_negative_infinity",
			input:     "-Inf",
			dataType:  "DataTypeFP32",
			shouldErr: false,
		},
		{
			name:      "float32_nan",
			input:     "NaN",
			dataType:  "DataTypeFP32",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := StringToBytes(tt.input, tt.dataType)
			if tt.shouldErr && err == nil {
				t.Errorf("StringToBytes() should fail but succeeded")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("StringToBytes() should succeed but got error: %v", err)
			}
		})
	}
}
