package byteorder

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Initialize byte order before running tests to ensure c.ByteOrder is set for functions that rely on it
	Init()
	os.Exit(m.Run())
}

func TestBoolVector(t *testing.T) {
	// Desired bool slice: [true, false, true, true, false, false, true, false, true]
	bools := []bool{true, false, true, true, false, false, true, false, true}
	vectorLength := len(bools)

	// Create encoded bytes
	encoded := make([]byte, (vectorLength+7)/8) // enough bytes to hold all bits
	for i, v := range bools {
		if v {
			byteIndex := i / 8
			bitIndex := i % 8
			encoded[byteIndex] |= 1 << bitIndex
		}
	}

	decoded := ByteOrder.BoolVector(encoded, vectorLength)
	if len(decoded) != vectorLength {
		t.Fatalf("Expected length %d, got %d", vectorLength, len(decoded))
	}
	for i, v := range decoded {
		if v != bools[i] {
			t.Errorf("BoolVector mismatch at index %d: expected %v, got %v", i, bools[i], v)
		}
	}
}

func TestStringVector(t *testing.T) {
	vector := []string{"foo", "bar", "baz"}
	vectorLength := len(vector)
	stringLength := 3

	// Build encoded byte slice (concatenate, no padding needed since len==stringLength)
	encoded := []byte("foobarbaz")

	decoded := ByteOrder.StringVector(encoded, vectorLength, stringLength)
	if len(decoded) != vectorLength {
		t.Fatalf("Expected %d strings, got %d", vectorLength, len(decoded))
	}
	for i, v := range decoded {
		if v != vector[i] {
			t.Errorf("StringVector mismatch at index %d: expected %s, got %s", i, vector[i], v)
		}
	}
}
