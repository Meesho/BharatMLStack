package numerix

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/datatypeconverter/byteorder"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Initialize the byte order system for typeconverter to work
	byteorder.Init()
}

func float32ToBytes(f float32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, math.Float32bits(f))
	return buf
}

func float64ToBytes(f float64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(f))
	return buf
}

func int32ToBytes(i int32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(i))
	return buf
}

func Test_ConvertScoreDataFromFP32(t *testing.T) {
	adapter := &Adapter{}

	tests := []struct {
		name     string
		data     [][][]byte
		dataType string
		expected [][][]byte
	}{
		{
			name: "successful conversion from fp32 to fp16",
			data: [][][]byte{
				{
					{0, 0, 128, 63}, // fp32(1)
					{0, 0, 0, 64},   // fp32(2)
				},
			},
			dataType: "fp16",
			expected: [][][]byte{
				{
					{0, 60}, // fp16(1) - 2 bytes
					{0, 64}, // fp16(2) - 2 bytes
				},
			},
		},
		{
			name: "successful conversion from fp32 to int32",
			data: [][][]byte{
				{
					{0, 0, 128, 63}, // fp32(1.0)
					{0, 0, 0, 64},   // fp32(2.0)
				},
			},
			dataType: "int32",
			expected: [][][]byte{
				{
					{1, 0, 0, 0}, // int32(1) - 4 bytes
					{2, 0, 0, 0}, // int32(2) - 4 bytes
				},
			},
		},
		{
			name:     "empty data",
			data:     [][][]byte{},
			dataType: "fp16",
			expected: [][][]byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := adapter.ConvertScoreDataFromFP32(tt.data, tt.dataType)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, tt.data)
		})
	}
}
