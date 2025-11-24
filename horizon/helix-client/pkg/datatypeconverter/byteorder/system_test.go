package byteorder

import (
	"math"
	"testing"
)

// Helper function to create a buffer
func createBuffer(size int) []byte {
	return make([]byte, size)
}

func TestCustomByteOrder_PutUint8FromUint32(t *testing.T) {
	buf := createBuffer(1)
	val := uint32(255)

	ByteOrder.PutUint8FromUint32(buf, val)
	result := ByteOrder.Uint8(buf)

	if result != uint8(val) {
		t.Errorf("PutUint8FromUint32/Uin8 failed. Expected %v, got %v", uint8(val), result)
	}
}

func TestCustomByteOrder_PutUint16FromUint32(t *testing.T) {
	buf := createBuffer(2)
	val := uint32(65535)

	ByteOrder.PutUint16FromUint32(buf, val)
	result := ByteOrder.Uint16AsUint32(buf)

	if result != uint32(uint16(val)) {
		t.Errorf("PutUint16FromUint32/Uint16AsUint32 failed. Expected %v, got %v", uint32(uint16(val)), result)
	}
}

func TestCustomByteOrder_PutInt8FromInt32(t *testing.T) {
	buf := createBuffer(1)
	val := int32(-127)

	ByteOrder.PutInt8FromInt32(buf, val)
	result := ByteOrder.Int8(buf)

	if result != int8(val) {
		t.Errorf("PutInt8FromInt32/Int8 failed. Expected %v, got %v", int8(val), result)
	}
}

func TestCustomByteOrder_PutInt16FromInt32(t *testing.T) {
	buf := createBuffer(2)
	val := int32(-32767)

	ByteOrder.PutInt16FromInt32(buf, val)
	result := ByteOrder.Int16AsInt32(buf)

	if result != int32(int16(val)) {
		t.Errorf("PutInt16FromInt32/Int16AsInt32 failed. Expected %v, got %v", int32(int16(val)), result)
	}
}

func TestCustomByteOrder_PutFloat32(t *testing.T) {
	buf := createBuffer(4)
	val := float32(3.14)

	ByteOrder.PutFloat32(buf, val)
	result := ByteOrder.Float32(buf)

	if result != val {
		t.Errorf("PutFloat32/Float32 failed. Expected %v, got %v", val, result)
	}
}

func TestCustomByteOrder_PutFloat64(t *testing.T) {
	buf := createBuffer(8)
	val := float64(3.1415926535)

	ByteOrder.PutFloat64(buf, val)
	result := ByteOrder.Float64(buf)

	if result != val {
		t.Errorf("PutFloat64/Float64 failed. Expected %v, got %v", val, result)
	}
}

func TestCustomByteOrder_PutFloat16FromFP32(t *testing.T) {
	buf := createBuffer(2)
	val := float32(3.14)

	ByteOrder.PutFloat16FromFP32(buf, val)
	result := ByteOrder.Float16AsFP32(buf)

	// Allowing a small margin of error due to precision loss
	if math.Abs(float64(result-val)) > 0.01 {
		t.Errorf("PutFloat16FromFP32/Float16AsFP32 failed. Expected %v, got %v", val, result)
	}
}

func TestCustomByteOrder_PutFloat8E5M2FromFP32(t *testing.T) {
	tests := []struct {
		val      float32
		expected float32
	}{
		{0.00783292, 0.0078125},                        // random positive value within range
		{-0.00005, -4.5776367e-05},                     // random negative value within range
		{57344, 57344},                                 // normal Max FP8 value
		{6.103515625e-05, 0.00006103515625},            // normal Min FP8 value
		{4.57763671875e-05, 0.0000457763671875},        // subnormal Max FP8 value
		{1.52587890625e-05, 0.0000152587890625},        // subnormal Min FP8 value
		{1000000, float32(math.Inf(1))},                // overflow
		{0.0000012207031, 0},                           // underflow
		{float32(math.Inf(1)), float32(math.Inf(1))},   // +Inf
		{float32(math.Inf(-1)), float32(math.Inf(-1))}, // -Inf
		{float32(math.NaN()), float32(math.NaN())},     // NaN
		{0, 0}, // Zero
	}

	for _, test := range tests {
		buf := createBuffer(1)
		ByteOrder.PutFloat8E5M2FromFP32(buf, test.val)
		result := ByteOrder.Float8E5M2AsFP32(buf)

		if !(math.IsNaN(float64(result)) && math.IsNaN(float64(test.expected)) || result == test.expected) {
			t.Errorf("PutFloat8E5M2FromFP32/Float8E5M2AsFP32 failed. Expected %v, got %v", test.expected, result)
		}
	}
}

func TestCustomByteOrder_PutFloat8E4M3FromFP32(t *testing.T) {
	tests := []struct {
		val      float32
		expected float32
	}{
		{12.45766435432, 12},                         // random positive value within range
		{-3.14159265359, -3.25},                      // random negative value within range
		{448, 448},                                   // normal Max FP8 value
		{0.015625, 0.015625},                         // normal Min FP8 value
		{0.013671875, 0.013671875},                   // subnormal Max FP8 value
		{0.001953125, 0.001953125},                   // subnormal Min FP8 value
		{5000, float32(math.NaN())},                  // overflow
		{0.0001953125, 0},                            // underflow
		{float32(math.NaN()), float32(math.NaN())},   // NaN
		{float32(math.Inf(1)), float32(math.NaN())},  // +Inf
		{float32(math.Inf(-1)), float32(math.NaN())}, // -Inf
		{0, 0}, // Zero
	}

	for _, test := range tests {
		buf := createBuffer(1)
		ByteOrder.PutFloat8E4M3FromFP32(buf, test.val)
		result := ByteOrder.Float8E4M3AsFP32(buf)

		if !(math.IsNaN(float64(result)) && math.IsNaN(float64(test.expected)) || result == test.expected) {
			t.Errorf("PutFloat8E4M3FromFP32/Float8E4M3AsFP32 failed. Expected %v, got %v", test.expected, result)
		}
	}
}
