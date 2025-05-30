package blocks

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestSetupHeadersV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name    string
		block   *PermStorageDataBlock
		wantErr bool
	}{
		{
			name:    "nil block",
			block:   nil,
			wantErr: true,
		},
		{
			name: "buffer too small",
			block: &PermStorageDataBlock{
				buf: make([]byte, PSDBLayout1LengthBytes-1),
			},
			wantErr: true,
		},
		{
			name: "valid setup",
			block: &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				featureSchemaVersion: 1,
				layoutVersion:        2,
				dataType:             3,
				expiryAt:             uint64(time.Now().Add(24 * time.Hour).Unix()),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setupHeadersV2(tt.block)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetupFeatureSchemaVersion(t *testing.T) {
	system.Init()
	tests := []struct {
		name                 string
		featureSchemaVersion uint16
		want                 uint16
	}{
		{
			name:                 "zero version",
			featureSchemaVersion: 0,
			want:                 0,
		},
		{
			name:                 "max version",
			featureSchemaVersion: 65535, // 2^16 - 1
			want:                 65535,
		},
		{
			name:                 "typical version",
			featureSchemaVersion: 1234,
			want:                 1234,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				featureSchemaVersion: tt.featureSchemaVersion,
			}
			setupFeatureSchemaVersion(p)
			got := system.ByteOrder.Uint16(p.buf[0:2])
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSetupLayoutVersion(t *testing.T) {
	system.Init()
	tests := []struct {
		name          string
		layoutVersion uint8
		initialByte   uint8 // Initial value of byte 7
		want          uint8 // Expected value of byte 7
	}{
		{
			name:          "zero version preserves lower bits",
			layoutVersion: 0,
			initialByte:   0b00001111,
			want:          0b00001111,
		},
		{
			name:          "max version (0x0F)",
			layoutVersion: 0x0F,
			initialByte:   0b00001111,
			want:          0b11111111,
		},
		{
			name:          "typical version preserves lower bits",
			layoutVersion: 0x05,
			initialByte:   0b00001010,
			want:          0b01011010,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PermStorageDataBlock{
				buf:           make([]byte, PSDBLayout1LengthBytes),
				layoutVersion: tt.layoutVersion,
			}
			p.buf[7] = tt.initialByte
			setupLayoutVersion(p)
			assert.Equal(t, tt.want, p.buf[7])
		})
	}
}

func TestSetupCompressionType(t *testing.T) {
	system.Init()
	tests := []struct {
		name            string
		compressionType compression.Type
		initialByte     uint8 // Initial value of byte 7
		want            uint8 // Expected value of byte 7
	}{
		{
			name:            "none compression",
			compressionType: compression.TypeNone,
			initialByte:     0b11110001,
			want:            0b11110001,
		},
		{
			name:            "zstd compression",
			compressionType: compression.TypeZSTD,
			initialByte:     0b11110001,
			want:            0b11110011,
		},
		{
			name:            "preserve other bits",
			compressionType: compression.Type(0x03),
			initialByte:     0b11110001,
			want:            0b11110111,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PermStorageDataBlock{
				buf:             make([]byte, PSDBLayout1LengthBytes),
				compressionType: tt.compressionType,
			}
			p.buf[7] = tt.initialByte
			setupCompressionType(p)
			assert.Equal(t, tt.want, p.buf[7])
		})
	}
}

func TestSetupDataType(t *testing.T) {
	system.Init()
	tests := []struct {
		name         string
		dataType     types.DataType
		initialByte7 uint8
		initialByte8 uint8
		wantByte7    uint8
		wantByte8    uint8
	}{
		{
			name:         "DataTypeUnknown",
			dataType:     types.DataTypeUnknown, // 0
			initialByte7: 0b11111110,
			initialByte8: 0b00001111,
			wantByte7:    0b11111110,
			wantByte8:    0b00001111,
		},
		{
			name:         "DataTypeFP32",
			dataType:     types.DataTypeFP32, // 4
			initialByte7: 0b11111110,
			initialByte8: 0b00001111,
			wantByte7:    0b11111110,
			wantByte8:    0b01001111,
		},
		{
			name:         "DataTypeBoolVector",
			dataType:     types.DataTypeBoolVector, // 30 (0b11110)
			initialByte7: 0b11111110,
			initialByte8: 0b00001111,
			wantByte7:    0b11111111,
			wantByte8:    0b11101111,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PermStorageDataBlock{
				buf:      make([]byte, PSDBLayout1LengthBytes),
				dataType: tt.dataType,
			}
			p.buf[7] = tt.initialByte7
			p.buf[8] = tt.initialByte8
			setupDataType(p)
			assert.Equal(t, tt.wantByte7, p.buf[7])
			assert.Equal(t, tt.wantByte8, p.buf[8])
		})
	}
}

func TestSetupBoolDtypeLastIdx(t *testing.T) {
	system.Init()
	tests := []struct {
		name             string
		boolDtypeLastIdx uint8
		initialByte      uint8
		want             uint8
	}{
		{
			name:             "zero index",
			boolDtypeLastIdx: 0,
			initialByte:      0b11110000,
			want:             0b11110000,
		},
		{
			name:             "max index (15)",
			boolDtypeLastIdx: 15,
			initialByte:      0b11110000,
			want:             0b11111111,
		},
		{
			name:             "preserve upper bits",
			boolDtypeLastIdx: 0b0101,
			initialByte:      0b11110000,
			want:             0b11110101,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PermStorageDataBlock{
				buf: make([]byte, PSDBLayout1LengthBytes),
			}
			p.buf[8] = tt.initialByte
			setupBoolDtypeLastIdx(p, tt.boolDtypeLastIdx)
			assert.Equal(t, tt.want, p.buf[8])
		})
	}
}

func TestEncodeData(t *testing.T) {
	system.Init()
	tests := []struct {
		name           string
		originalData   []byte
		encoder        compression.Encoder
		wantCompressed bool
	}{
		{
			name:           "small data - no compression",
			originalData:   []byte("small"),
			encoder:        compression.NewZStdEncoder(),
			wantCompressed: false,
		},
		{
			name:           "compressible data",
			originalData:   []byte(string(make([]byte, 1000))), // highly compressible data
			encoder:        compression.NewZStdEncoder(),
			wantCompressed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PermStorageDataBlock{
				buf:             make([]byte, PSDBLayout1LengthBytes),
				originalData:    tt.originalData,
				originalDataLen: len(tt.originalData),
			}

			result, err := encodeData(p, tt.encoder)
			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.wantCompressed {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			} else {
				assert.Equal(t, compression.TypeNone, p.compressionType)
				assert.Equal(t, p.compressedDataLen, p.originalDataLen)
			}
		})
	}
}

func TestSerializeFP32AndLessV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		dataType  types.DataType
		data      []float32
		wantErr   bool
		checkComp bool // true if we want to verify compression happened
	}{
		{
			name:     "small fp32 data",
			dataType: types.DataTypeFP32,
			data:     []float32{1.0, 2.0},
			wantErr:  false,
		},
		{
			name:     "small fp16 data",
			dataType: types.DataTypeFP16,
			data:     []float32{1.0, 2.0},
			wantErr:  false,
		},
		{
			name:     "small fp8 data",
			dataType: types.DataTypeFP8E4M3,
			data:     []float32{1.0, 2.0},
			wantErr:  false,
		},
		{
			name:      "large fp32 data",
			dataType:  types.DataTypeFP32,
			data:      generateLargeFloat32Data(1000),
			wantErr:   false,
			checkComp: true,
		},
		{
			name:     "invalid container",
			dataType: types.DataTypeFP32,
			data:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             tt.dataType,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				originalData:         make([]byte, len(tt.data)*tt.dataType.Size()),
				originalDataLen:      len(tt.data) * tt.dataType.Size(),
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeFP32AndLessV2(p)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data can be decoded correctly
			if !tt.wantErr {
				var actualData []byte
				if p.compressionType == compression.TypeZSTD {
					dec := compression.NewZStdDecoder()
					actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
					assert.NoError(t, err)
				} else {
					actualData = result[PSDBLayout1LengthBytes:]
				}

				// Convert back to float32 and compare
				decoded := make([]float32, len(tt.data))
				getFloat, _ := system.GetFromByteFP32AndLess(tt.dataType)
				for i := 0; i < len(tt.data); i++ {
					decoded[i] = getFloat(actualData[i*tt.dataType.Size() : (i+1)*tt.dataType.Size()])
				}
				assert.InDeltaSlice(t, tt.data, decoded, 0.01)
			}
		})
	}
}

func TestSerializeInt32AndLessV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		dataType  types.DataType
		data      []int32
		wantErr   bool
		checkComp bool
	}{
		{
			name:     "small int32 data",
			dataType: types.DataTypeInt32,
			data:     []int32{1, 2},
			wantErr:  false,
		},
		{
			name:     "small int16 data",
			dataType: types.DataTypeInt16,
			data:     []int32{1, 2},
			wantErr:  false,
		},
		{
			name:     "small int8 data",
			dataType: types.DataTypeInt8,
			data:     []int32{1, 2},
			wantErr:  false,
		},
		{
			name:      "large int32 data",
			dataType:  types.DataTypeInt32,
			data:      generateLargeInt32Data(1000),
			wantErr:   false,
			checkComp: true,
		},
		{
			name:     "invalid container",
			dataType: types.DataTypeInt32,
			data:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             tt.dataType,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				originalData:         make([]byte, len(tt.data)*tt.dataType.Size()),
				originalDataLen:      len(tt.data) * tt.dataType.Size(),
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeInt32AndLessV2(p)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data can be decoded correctly
			if !tt.wantErr {
				var actualData []byte
				if p.compressionType == compression.TypeZSTD {
					dec := compression.NewZStdDecoder()
					actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
					assert.NoError(t, err)
				} else {
					actualData = result[PSDBLayout1LengthBytes:]
				}

				// Convert back to int32 and compare
				decoded := make([]int32, len(tt.data))
				getInt, _ := system.GetFromByteInt32AndLess(tt.dataType)
				for i := 0; i < len(tt.data); i++ {
					decoded[i] = getInt(actualData[i*tt.dataType.Size() : (i+1)*tt.dataType.Size()])
				}
				assert.Equal(t, tt.data, decoded)
			}
		})
	}
}

func TestSerializeUint32AndLessV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		dataType  types.DataType
		data      []uint32
		wantErr   bool
		checkComp bool
	}{
		{
			name:     "small uint32 data",
			dataType: types.DataTypeUint32,
			data:     []uint32{1, 2},
			wantErr:  false,
		},
		{
			name:     "small uint16 data",
			dataType: types.DataTypeUint16,
			data:     []uint32{1, 2},
			wantErr:  false,
		},
		{
			name:     "small uint8 data",
			dataType: types.DataTypeUint8,
			data:     []uint32{1, 2},
			wantErr:  false,
		},
		{
			name:      "large uint32 data",
			dataType:  types.DataTypeUint32,
			data:      generateLargeUint32Data(1000),
			wantErr:   false,
			checkComp: true,
		},
		{
			name:     "invalid container",
			dataType: types.DataTypeUint32,
			data:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             tt.dataType,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				originalData:         make([]byte, len(tt.data)*tt.dataType.Size()),
				originalDataLen:      len(tt.data) * tt.dataType.Size(),
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeUint32AndLessV2(p)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data can be decoded correctly
			if !tt.wantErr {
				var actualData []byte
				if p.compressionType == compression.TypeZSTD {
					dec := compression.NewZStdDecoder()
					actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
					assert.NoError(t, err)
				} else {
					actualData = result[PSDBLayout1LengthBytes:]
				}

				// Convert back to uint32 and compare
				decoded := make([]uint32, len(tt.data))
				getUint, _ := system.GetFromByteUint32AndLess(tt.dataType)
				for i := 0; i < len(tt.data); i++ {
					decoded[i] = getUint(actualData[i*tt.dataType.Size() : (i+1)*tt.dataType.Size()])
				}
				assert.Equal(t, tt.data, decoded)
			}
		})
	}
}

func TestSerializeFP64V2(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		data      []float64
		wantErr   bool
		checkComp bool // true if we want to verify compression happened
	}{
		{
			name:    "small data",
			data:    []float64{1.1, 2.2, 3.3},
			wantErr: false,
		},
		{
			name:      "large data",
			data:      generateLargeFloat64Data(1000),
			wantErr:   false,
			checkComp: true,
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []float64{},
			wantErr: true,
		},
		{
			name:    "special values",
			data:    []float64{math.Inf(1), math.Inf(-1), math.NaN(), math.MaxFloat64, math.SmallestNonzeroFloat64},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             types.DataTypeFP64,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				originalData:         make([]byte, len(tt.data)*8), // float64 = 8 bytes
				originalDataLen:      len(tt.data) * 8,
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeFP64V2(p)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data can be decoded correctly
			var actualData []byte
			if p.compressionType == compression.TypeZSTD {
				dec := compression.NewZStdDecoder()
				actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
				assert.NoError(t, err)
			} else {
				actualData = result[PSDBLayout1LengthBytes:]
			}

			// Convert back to float64 and compare
			decoded := make([]float64, len(tt.data))
			for i := 0; i < len(tt.data); i++ {
				decoded[i] = system.ByteOrder.Float64(actualData[i*8 : (i+1)*8])
			}

			// Special handling for NaN values
			for i := 0; i < len(tt.data); i++ {
				if math.IsNaN(tt.data[i]) {
					assert.True(t, math.IsNaN(decoded[i]))
				} else {
					assert.Equal(t, tt.data[i], decoded[i])
				}
			}
		})
	}
}

func TestSerializeInt64V2(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		data      []int64
		wantErr   bool
		checkComp bool
	}{
		{
			name:    "small data",
			data:    []int64{1, -2, 3},
			wantErr: false,
		},
		{
			name:      "large data",
			data:      generateLargeInt64Data(1000),
			wantErr:   false,
			checkComp: true,
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []int64{},
			wantErr: true,
		},
		{
			name:    "edge values",
			data:    []int64{math.MaxInt64, math.MinInt64, 0},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             types.DataTypeInt64,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				originalData:         make([]byte, len(tt.data)*8),
				originalDataLen:      len(tt.data) * 8,
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeInt64V2(p)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data can be decoded correctly
			var actualData []byte
			if p.compressionType == compression.TypeZSTD {
				dec := compression.NewZStdDecoder()
				actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
				assert.NoError(t, err)
			} else {
				actualData = result[PSDBLayout1LengthBytes:]
			}

			// Convert back to int64 and compare
			decoded := make([]int64, len(tt.data))
			for i := 0; i < len(tt.data); i++ {
				decoded[i] = system.ByteOrder.Int64(actualData[i*8 : (i+1)*8])
			}
			assert.Equal(t, tt.data, decoded)
		})
	}
}

func TestSerializeUint64V2(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		data      []uint64
		wantErr   bool
		checkComp bool
	}{
		{
			name:    "small data",
			data:    []uint64{1, 2, 3},
			wantErr: false,
		},
		{
			name:      "large data",
			data:      generateLargeUint64Data(1000),
			wantErr:   false,
			checkComp: true,
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []uint64{},
			wantErr: true,
		},
		{
			name:    "edge values",
			data:    []uint64{0, math.MaxUint64},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             types.DataTypeUint64,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				originalData:         make([]byte, len(tt.data)*8),
				originalDataLen:      len(tt.data) * 8,
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeUint64V2(p)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data can be decoded correctly
			var actualData []byte
			if p.compressionType == compression.TypeZSTD {
				dec := compression.NewZStdDecoder()
				actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
				assert.NoError(t, err)
			} else {
				actualData = result[PSDBLayout1LengthBytes:]
			}

			// Convert back to uint64 and compare
			decoded := make([]uint64, len(tt.data))
			for i := 0; i < len(tt.data); i++ {
				decoded[i] = system.ByteOrder.Uint64(actualData[i*8 : (i+1)*8])
			}
			assert.Equal(t, tt.data, decoded)
		})
	}
}

func TestSerializeStringV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name          string
		data          []string
		stringLengths []uint16
		wantErr       bool
		errMsg        string
		checkComp     bool
	}{
		{
			name:          "basic strings",
			data:          []string{"hello", "world"},
			stringLengths: []uint16{5, 5},
			wantErr:       false,
		},
		{
			name:          "strings with different lengths",
			data:          []string{"short", "this is longer"},
			stringLengths: []uint16{5, 15},
			wantErr:       false,
		},
		{
			name:          "empty strings",
			data:          []string{"", ""},
			stringLengths: []uint16{1, 1},
			wantErr:       false,
		},
		{
			name:          "large compressible data",
			data:          generateRepetitiveStrings(1000, "test"),
			stringLengths: generateFixedLengths(1000, 4),
			wantErr:       false,
			checkComp:     true,
		},
		{
			name:          "string exceeds max length",
			data:          []string{"short", strings.Repeat("a", 65536)},
			stringLengths: []uint16{5, 65535},
			wantErr:       true,
			errMsg:        "exceeds max length of 65535",
		},
		{
			name:          "string exceeds booked length",
			data:          []string{"toolong", "ok"},
			stringLengths: []uint16{5, 5},
			wantErr:       true,
			errMsg:        "length 7 exceeds max length of 65535 or booked size 5",
		},
		{
			name:          "mismatched lengths",
			data:          []string{"one", "two", "three"},
			stringLengths: []uint16{5, 5},
			wantErr:       true,
			errMsg:        "mismatch in number of strings",
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
			errMsg:  "string data expected to come in string container",
		},
		{
			name:          "empty slice",
			data:          []string{},
			stringLengths: []uint16{},
			wantErr:       true,
			errMsg:        "string data expected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate total size needed for originalData
			totalSize := 0
			if tt.data != nil {
				totalSize = len(tt.data) * 2 // length storage space
				for _, s := range tt.data {
					totalSize += len(s) // string data space
				}
			}

			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             types.DataTypeString,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				stringLengths:        tt.stringLengths,
				originalData:         make([]byte, totalSize),
				originalDataLen:      totalSize,
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeStringV2(p)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data can be decoded correctly
			var actualData []byte
			if p.compressionType == compression.TypeZSTD {
				dec := compression.NewZStdDecoder()
				actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
				assert.NoError(t, err)
			} else {
				actualData = result[PSDBLayout1LengthBytes:]
			}

			// Verify string data
			strLenOffsetIdx := 0
			strDataOffsetIdx := len(tt.data) * 2 // Start of string data after all lengths

			for _, expectedStr := range tt.data {
				// Read length of current string
				strLen := system.ByteOrder.Uint16(actualData[strLenOffsetIdx : strLenOffsetIdx+2])
				assert.Equal(t, uint16(len(expectedStr)), strLen)

				// Extract and verify string
				actualStr := string(actualData[strDataOffsetIdx : strDataOffsetIdx+int(strLen)])
				assert.Equal(t, expectedStr, actualStr)

				// Update offsets
				strLenOffsetIdx += 2
				strDataOffsetIdx += int(strLen)
			}
		})
	}
}

func TestSerializeBoolV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name        string
		data        []uint8
		wantErr     bool
		errMsg      string
		checkComp   bool
		wantLastIdx uint8
	}{
		{
			name:        "single byte of bools",
			data:        []uint8{1, 0, 1, 1, 0, 0, 1}, // 7 bits
			wantErr:     false,
			wantLastIdx: 1, // last bit at position 1
		},
		{
			name:        "complete byte of bools",
			data:        []uint8{1, 0, 1, 1, 0, 0, 1, 0}, // 8 bits
			wantErr:     false,
			wantLastIdx: 0, // last bit at position 0
		},
		{
			name:        "multiple bytes of bools",
			data:        []uint8{1, 1, 1, 1, 1, 1, 1, 1, 0, 1}, // 10 bits
			wantErr:     false,
			wantLastIdx: 6, // last bit at position 6
		},
		{
			name:        "large bool array",
			data:        generateBoolData(1000),
			wantErr:     false,
			checkComp:   true,
			wantLastIdx: uint8(1000 % 8), // last bit position for 1000 bits
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
			errMsg:  "bool Data expected to come in uin8 container",
		},
		{
			name:    "empty data",
			data:    []uint8{},
			wantErr: true,
			errMsg:  "bool Data expected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate bytes needed: ceil(len(data)/8)
			byteCount := (len(tt.data) + 7) / 8

			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             types.DataTypeBool,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				originalData:         make([]byte, byteCount),
				originalDataLen:      byteCount,
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeBoolV2(p)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Check last bit index is correctly set
			lastIdx := result[8] & 0x0F // Extract lower 4 bits of byte 8
			assert.Equal(t, tt.wantLastIdx, lastIdx)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data can be decoded correctly
			var actualData []byte
			if p.compressionType == compression.TypeZSTD {
				dec := compression.NewZStdDecoder()
				actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
				assert.NoError(t, err)
			} else {
				actualData = result[PSDBLayout1LengthBytes:]
			}

			// Convert bit-packed data back to bool array
			decoded := make([]uint8, len(tt.data))
			byteIdx := 0
			bitIdx := 7
			for i := 0; i < len(tt.data); i++ {
				decoded[i] = (actualData[byteIdx] >> bitIdx) & 1
				bitIdx--
				if bitIdx < 0 {
					bitIdx = 7
					byteIdx++
				}
			}

			assert.Equal(t, tt.data, decoded)
		})
	}
}

// Note: Test values are chosen to be exactly representable in their target formats.
// Real-world data might come in FP32 containers but should contain values that are
// actually FP16/FP8 values to avoid precision loss during conversion. If that is not the case,
// the test will fail, but tested and precision loss is acceptable.
func TestSerializeFP32VectorAndLessV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name          string
		data          [][]float32
		vectorLengths []uint16
		dataType      types.DataType
		wantErr       bool
		errMsg        string
		checkComp     bool
	}{
		{
			name: "basic fp32 vectors",
			data: [][]float32{
				{1.1, 2.2, 3.3},
				{4.4, 5.5, 6.6},
			},
			vectorLengths: []uint16{3, 3},
			dataType:      types.DataTypeFP32Vector,
			wantErr:       false,
		},
		{
			name: "fp16 vectors",
			data: [][]float32{
				{1.0, 2.0},  // exact values representable in FP16
				{-4.0, 8.0}, // exact values representable in FP16
			},
			vectorLengths: []uint16{2, 2},
			dataType:      types.DataTypeFP16Vector,
			wantErr:       false,
		},
		{
			name: "fp8 vectors (E4M3)",
			data: [][]float32{
				{1.0, 2.0},  // exact values representable in FP8E4M3
				{4.0, -8.0}, // exact values representable in FP8E4M3
			},
			vectorLengths: []uint16{2, 2},
			dataType:      types.DataTypeFP8E4M3Vector,
			wantErr:       false,
		},
		{
			name: "fp8 vectors (E5M2)",
			data: [][]float32{
				{1.0, 2.0},   // exact values representable in FP8E5M2
				{4.0, -16.0}, // exact values representable in FP8E5M2
			},
			vectorLengths: []uint16{2, 2},
			dataType:      types.DataTypeFP8E5M2Vector,
			wantErr:       false,
		},
		{
			name:          "large vectors for compression",
			data:          generateLargeFloat32Vectors(100, 10), // 100 vectors of length 10
			vectorLengths: generateFixedVectorLengths(100, 10),
			dataType:      types.DataTypeFP32Vector,
			wantErr:       false,
			checkComp:     true,
		},
		{
			name: "mismatched vector length",
			data: [][]float32{
				{1.1, 2.2, 3.3},
				{4.4, 5.5}, // shorter vector
			},
			vectorLengths: []uint16{3, 3},
			dataType:      types.DataTypeFP32Vector,
			wantErr:       true,
			errMsg:        "mismatch in vector length at index 1",
		},
		{
			name: "mismatched number of vectors",
			data: [][]float32{
				{1.1, 2.2},
				{3.3, 4.4},
			},
			vectorLengths: []uint16{2}, // fewer lengths than vectors
			dataType:      types.DataTypeFP32Vector,
			wantErr:       true,
			errMsg:        "mismatch in number of vectors",
		},
		{
			name:     "nil data",
			data:     nil,
			dataType: types.DataTypeFP32Vector,
			wantErr:  true,
			errMsg:   "fp32 vector Data expected to come in fp32 vector container",
		},
		{
			name:          "empty vectors",
			data:          [][]float32{},
			vectorLengths: []uint16{},
			dataType:      types.DataTypeFP32Vector,
			wantErr:       true,
			errMsg:        "fp32 vector Data expected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate total size needed
			totalSize := 0
			if tt.data != nil {
				for _, vec := range tt.data {
					totalSize += len(vec) * tt.dataType.Size()
				}
			}

			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             tt.dataType,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				vectorLengths:        tt.vectorLengths,
				originalData:         make([]byte, totalSize),
				originalDataLen:      totalSize,
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeFP32VectorAndLessV2(p)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data can be decoded correctly
			var actualData []byte
			if p.compressionType == compression.TypeZSTD {
				dec := compression.NewZStdDecoder()
				actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
				assert.NoError(t, err)
			} else {
				actualData = result[PSDBLayout1LengthBytes:]
			}

			// Convert back to float32 vectors
			decoded := make([][]float32, len(tt.data))
			offset := 0
			getFloat, err := system.GetFromByteFP32AndLess(tt.dataType)
			assert.NoError(t, err)

			for i, vec := range tt.data {
				decoded[i] = make([]float32, len(vec))
				for j := range vec {
					decoded[i][j] = getFloat(actualData[offset : offset+tt.dataType.Size()])
					offset += tt.dataType.Size()
				}
			}

			// Compare vectors with appropriate precision based on data type
			for i, vec := range tt.data {
				for j, val := range vec {
					switch tt.dataType {
					case types.DataTypeFP32Vector:
						assert.InDelta(t, val, decoded[i][j], 1e-6)
					case types.DataTypeFP16Vector:
						assert.InDelta(t, val, decoded[i][j], 1e-3)
					case types.DataTypeFP8E4M3Vector, types.DataTypeFP8E5M2Vector:
						assert.InDelta(t, val, decoded[i][j], 1e-1)
					}
				}
			}
		})
	}
}

func TestSerializeInt32VectorAndLessV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name          string
		data          [][]int32
		vectorLengths []uint16
		dataType      types.DataType
		wantErr       bool
		errMsg        string
		checkComp     bool
	}{
		{
			name: "int32 vectors",
			data: [][]int32{
				{-2147483648, 2147483647}, // min/max int32
				{0, 42},
			},
			vectorLengths: []uint16{2, 2},
			dataType:      types.DataTypeInt32Vector,
			wantErr:       false,
		},
		{
			name: "int16 vectors",
			data: [][]int32{
				{-32768, 32767}, // min/max int16
				{0, 42},
			},
			vectorLengths: []uint16{2, 2},
			dataType:      types.DataTypeInt16Vector,
			wantErr:       false,
		},
		{
			name: "int8 vectors",
			data: [][]int32{
				{-128, 127}, // min/max int8
				{0, 42},
			},
			vectorLengths: []uint16{2, 2},
			dataType:      types.DataTypeInt8Vector,
			wantErr:       false,
		},
		{
			name:          "large vectors for compression",
			data:          generateLargeInt32Vectors(100, 10), // 100 vectors of length 10
			vectorLengths: generateFixedVectorLengths(100, 10),
			dataType:      types.DataTypeInt32Vector,
			wantErr:       false,
			checkComp:     true,
		},
		{
			name: "mismatched vector length",
			data: [][]int32{
				{1, 2, 3},
				{4, 5}, // shorter vector
			},
			vectorLengths: []uint16{3, 3},
			dataType:      types.DataTypeInt32Vector,
			wantErr:       true,
			errMsg:        "mismatch in vector length at index 1",
		},
		{
			name:     "nil data",
			data:     nil,
			dataType: types.DataTypeInt32Vector,
			wantErr:  true,
			errMsg:   "int32 vector Data expected to come in int32 vector container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalSize := 0
			if tt.data != nil {
				for _, vec := range tt.data {
					totalSize += len(vec) * tt.dataType.Size()
				}
			}

			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             tt.dataType,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				vectorLengths:        tt.vectorLengths,
				originalData:         make([]byte, totalSize),
				originalDataLen:      totalSize,
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeInt32VectorAndLessV2(p)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data
			var actualData []byte
			if p.compressionType == compression.TypeZSTD {
				dec := compression.NewZStdDecoder()
				actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
				assert.NoError(t, err)
			} else {
				actualData = result[PSDBLayout1LengthBytes:]
			}

			// Convert back to int32 vectors
			decoded := make([][]int32, len(tt.data))
			offset := 0
			getInt, err := system.GetFromByteInt32AndLess(tt.dataType)
			assert.NoError(t, err)

			for i, vec := range tt.data {
				decoded[i] = make([]int32, len(vec))
				for j := range vec {
					decoded[i][j] = getInt(actualData[offset : offset+tt.dataType.Size()])
					offset += tt.dataType.Size()
				}
			}

			assert.Equal(t, tt.data, decoded)
		})
	}
}

func TestSerializeUint32VectorAndLessV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name          string
		data          [][]uint32
		vectorLengths []uint16
		dataType      types.DataType
		wantErr       bool
		errMsg        string
		checkComp     bool
	}{
		{
			name: "uint32 vectors",
			data: [][]uint32{
				{0, 4294967295}, // min/max uint32
				{42, 1000000},
			},
			vectorLengths: []uint16{2, 2},
			dataType:      types.DataTypeUint32Vector,
			wantErr:       false,
		},
		{
			name: "uint16 vectors",
			data: [][]uint32{
				{0, 65535}, // min/max uint16
				{42, 1000},
			},
			vectorLengths: []uint16{2, 2},
			dataType:      types.DataTypeUint16Vector,
			wantErr:       false,
		},
		{
			name: "uint8 vectors",
			data: [][]uint32{
				{0, 255}, // min/max uint8
				{42, 100},
			},
			vectorLengths: []uint16{2, 2},
			dataType:      types.DataTypeUint8Vector,
			wantErr:       false,
		},
		{
			name:          "large vectors for compression",
			data:          generateLargeUint32Vectors(100, 10), // 100 vectors of length 10
			vectorLengths: generateFixedVectorLengths(100, 10),
			dataType:      types.DataTypeUint32Vector,
			wantErr:       false,
			checkComp:     true,
		},
		{
			name: "mismatched vector length",
			data: [][]uint32{
				{1, 2, 3},
				{4, 5}, // shorter vector
			},
			vectorLengths: []uint16{3, 3},
			dataType:      types.DataTypeUint32Vector,
			wantErr:       true,
			errMsg:        "mismatch in vector length at index 1",
		},
		{
			name:     "nil data",
			data:     nil,
			dataType: types.DataTypeUint32Vector,
			wantErr:  true,
			errMsg:   "uint32 vector Data expected to come in uint32 vector container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalSize := 0
			if tt.data != nil {
				for _, vec := range tt.data {
					totalSize += len(vec) * tt.dataType.Size()
				}
			}

			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             tt.dataType,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				vectorLengths:        tt.vectorLengths,
				originalData:         make([]byte, totalSize),
				originalDataLen:      totalSize,
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeUint32VectorAndLessV2(p)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data
			var actualData []byte
			if p.compressionType == compression.TypeZSTD {
				dec := compression.NewZStdDecoder()
				actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
				assert.NoError(t, err)
			} else {
				actualData = result[PSDBLayout1LengthBytes:]
			}

			// Convert back to uint32 vectors
			decoded := make([][]uint32, len(tt.data))
			offset := 0
			getUint, err := system.GetFromByteUint32AndLess(tt.dataType)
			assert.NoError(t, err)

			for i, vec := range tt.data {
				decoded[i] = make([]uint32, len(vec))
				for j := range vec {
					decoded[i][j] = getUint(actualData[offset : offset+tt.dataType.Size()])
					offset += tt.dataType.Size()
				}
			}

			assert.Equal(t, tt.data, decoded)
		})
	}
}

func TestSerializeFP64VectorV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name          string
		data          [][]float64
		vectorLengths []uint16
		wantErr       bool
		errMsg        string
		checkComp     bool
	}{
		{
			name: "basic vectors",
			data: [][]float64{
				{1.0, -2.0, math.Pi},
				{math.MaxFloat64, -math.MaxFloat64},
			},
			vectorLengths: []uint16{3, 2},
			wantErr:       false,
		},
		{
			name: "special values",
			data: [][]float64{
				{math.Inf(1), math.Inf(-1)},
				{math.NaN(), 0.0},
			},
			vectorLengths: []uint16{2, 2},
			wantErr:       false,
		},
		{
			name:          "large vectors for compression",
			data:          generateLargeFP64Vectors(100, 10),
			vectorLengths: generateFixedVectorLengths(100, 10),
			wantErr:       false,
			checkComp:     true,
		},
		{
			name: "mismatched vector length",
			data: [][]float64{
				{1.0, 2.0, 3.0},
				{4.0, 5.0}, // shorter vector
			},
			vectorLengths: []uint16{3, 3},
			wantErr:       true,
			errMsg:        "mismatch in vector length",
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
			errMsg:  "fp64 vector Data expected to come in fp64 vector container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalSize := 0
			if tt.data != nil {
				for _, vec := range tt.data {
					totalSize += len(vec) * 8 // float64 size
				}
			}

			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             types.DataTypeFP64Vector,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				vectorLengths:        tt.vectorLengths,
				originalData:         make([]byte, totalSize),
				originalDataLen:      totalSize,
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeFP64VectorV2(p)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data
			var actualData []byte
			if p.compressionType == compression.TypeZSTD {
				dec := compression.NewZStdDecoder()
				actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
				assert.NoError(t, err)
			} else {
				actualData = result[PSDBLayout1LengthBytes:]
			}

			// Convert back to float64 vectors
			decoded := make([][]float64, len(tt.data))
			offset := 0
			for i, vec := range tt.data {
				decoded[i] = make([]float64, len(vec))
				for j := range vec {
					decoded[i][j] = system.ByteOrder.Float64(actualData[offset : offset+8])
					offset += 8
				}
			}

			// Special handling for NaN comparison
			for i, vec := range tt.data {
				for j, val := range vec {
					if math.IsNaN(val) {
						assert.True(t, math.IsNaN(decoded[i][j]))
					} else {
						assert.Equal(t, val, decoded[i][j])
					}
				}
			}
		})
	}
}

func TestSerializeInt64VectorV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name          string
		data          [][]int64
		vectorLengths []uint16
		wantErr       bool
		errMsg        string
		checkComp     bool
	}{
		{
			name: "basic vectors",
			data: [][]int64{
				{math.MinInt64, math.MaxInt64}, // min/max int64
				{0, 42},
			},
			vectorLengths: []uint16{2, 2},
			wantErr:       false,
		},
		{
			name:          "large vectors for compression",
			data:          generateLargeInt64Vectors(100, 10),
			vectorLengths: generateFixedVectorLengths(100, 10),
			wantErr:       false,
			checkComp:     true,
		},
		{
			name: "mismatched vector length",
			data: [][]int64{
				{1, 2, 3},
				{4, 5}, // shorter vector
			},
			vectorLengths: []uint16{3, 3},
			wantErr:       true,
			errMsg:        "mismatch in vector length",
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
			errMsg:  "int64 vector Data expected to come in int64 vector container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalSize := 0
			if tt.data != nil {
				for _, vec := range tt.data {
					totalSize += len(vec) * 8 // int64 size
				}
			}

			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             types.DataTypeInt64Vector,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				vectorLengths:        tt.vectorLengths,
				originalData:         make([]byte, totalSize),
				originalDataLen:      totalSize,
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeInt64VectorV2(p)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data
			var actualData []byte
			if p.compressionType == compression.TypeZSTD {
				dec := compression.NewZStdDecoder()
				actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
				assert.NoError(t, err)
			} else {
				actualData = result[PSDBLayout1LengthBytes:]
			}

			// Convert back to int64 vectors
			decoded := make([][]int64, len(tt.data))
			offset := 0
			for i, vec := range tt.data {
				decoded[i] = make([]int64, len(vec))
				for j := range vec {
					decoded[i][j] = system.ByteOrder.Int64(actualData[offset : offset+8])
					offset += 8
				}
			}

			assert.Equal(t, tt.data, decoded)
		})
	}
}

func TestSerializeUint64VectorV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name          string
		data          [][]uint64
		vectorLengths []uint16
		wantErr       bool
		errMsg        string
		checkComp     bool
	}{
		{
			name: "basic vectors",
			data: [][]uint64{
				{0, math.MaxUint64}, // min/max uint64
				{42, 1000000},
			},
			vectorLengths: []uint16{2, 2},
			wantErr:       false,
		},
		{
			name:          "large vectors for compression",
			data:          generateLargeUint64Vectors(100, 10),
			vectorLengths: generateFixedVectorLengths(100, 10),
			wantErr:       false,
			checkComp:     true,
		},
		{
			name: "mismatched vector length",
			data: [][]uint64{
				{1, 2, 3},
				{4, 5}, // shorter vector
			},
			vectorLengths: []uint16{3, 3},
			wantErr:       true,
			errMsg:        "mismatch in vector length",
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
			errMsg:  "uint64 vector Data expected to come in uint64 vector container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalSize := 0
			if tt.data != nil {
				for _, vec := range tt.data {
					totalSize += len(vec) * 8 // uint64 size
				}
			}

			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             types.DataTypeUint64Vector,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				vectorLengths:        tt.vectorLengths,
				originalData:         make([]byte, totalSize),
				originalDataLen:      totalSize,
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeUint64VectorV2(p)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data
			var actualData []byte
			if p.compressionType == compression.TypeZSTD {
				dec := compression.NewZStdDecoder()
				actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
				assert.NoError(t, err)
			} else {
				actualData = result[PSDBLayout1LengthBytes:]
			}

			// Convert back to uint64 vectors
			decoded := make([][]uint64, len(tt.data))
			offset := 0
			for i, vec := range tt.data {
				decoded[i] = make([]uint64, len(vec))
				for j := range vec {
					decoded[i][j] = system.ByteOrder.Uint64(actualData[offset : offset+8])
					offset += 8
				}
			}

			assert.Equal(t, tt.data, decoded)
		})
	}
}

func TestSerializeStringVectorV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name          string
		data          [][]string
		vectorLengths []uint16
		stringLengths []uint16
		wantErr       bool
		errMsg        string
		checkComp     bool
	}{
		{
			name: "basic vectors",
			data: [][]string{
				{"abc", "def"},          // vector1: strings <= 5 chars
				{"12345", "123", "456"}, // vector2: strings <= 6 chars
			},
			vectorLengths: []uint16{2, 3},
			stringLengths: []uint16{5, 6},
			wantErr:       false,
		},
		{
			name: "empty strings",
			data: [][]string{
				{"", ""},
				{"", "", ""},
			},
			vectorLengths: []uint16{2, 3},
			stringLengths: []uint16{5, 5},
			wantErr:       false,
		},
		{
			name: "max length strings",
			data: [][]string{
				{"12345", "12345"},          // exactly 5 chars
				{"123456", "123", "123456"}, // exactly 6 chars
			},
			vectorLengths: []uint16{2, 3},
			stringLengths: []uint16{5, 6},
			wantErr:       false,
		},
		{
			name: "string too long for vector",
			data: [][]string{
				{"123456", "12345"}, // first string > 5 chars
				{"12345", "123"},
			},
			vectorLengths: []uint16{2, 2},
			stringLengths: []uint16{5, 6},
			wantErr:       true,
			errMsg:        "exceeds max length",
		},
		{
			name: "mismatched vector length",
			data: [][]string{
				{"123", "123"},
				{"123", "123", "123"}, // expected 2 strings
			},
			vectorLengths: []uint16{2, 2},
			stringLengths: []uint16{5, 5},
			wantErr:       true,
			errMsg:        "mismatch in vector length",
		},
		{
			name: "mismatched stringLengths",
			data: [][]string{
				{"123", "123"},
				{"123", "123"},
			},
			vectorLengths: []uint16{2, 2},
			stringLengths: []uint16{5}, // missing second vector's string length
			wantErr:       true,
			errMsg:        "mismatch in number of vectors",
		},
		{
			name:          "large vectors for compression",
			data:          generateLargeStringVectors(100, 10, 5), // 100 vectors, 10 strings each, 5 chars
			vectorLengths: generateFixedVectorLengths(100, 10),
			stringLengths: generateFixedVectorLengths(100, 5),
			wantErr:       false,
			checkComp:     true,
		},
		{
			name: "unicode strings",
			data: [][]string{
				{"世界", "你好"},    // Chinese
				{"🌟", "⭐", "✨"}, // Emojis
			},
			vectorLengths: []uint16{2, 3},
			stringLengths: []uint16{6, 4}, // UTF-8 bytes, not runes
			wantErr:       false,
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
			errMsg:  "string vector data expected to come in string vector container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate total size needed
			totalStrings := 0
			totalBytes := 0
			if tt.data != nil {
				for _, vec := range tt.data {
					totalStrings += len(vec)
					for _, str := range vec {
						totalBytes += len(str)
					}
				}
			}

			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             types.DataTypeStringVector,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				vectorLengths:        tt.vectorLengths,
				stringLengths:        tt.stringLengths,
				originalData:         make([]byte, totalStrings*2+totalBytes), // length prefixes + string data
				originalDataLen:      totalStrings*2 + totalBytes,
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeStringVectorV2(p)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data
			var actualData []byte
			if p.compressionType == compression.TypeZSTD {
				dec := compression.NewZStdDecoder()
				actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
				assert.NoError(t, err)
			} else {
				actualData = result[PSDBLayout1LengthBytes:]
			}

			// Convert back to string vectors
			decoded := make([][]string, len(tt.data))
			strLenOffsetIdx := 0
			strDataOffsetIdx := totalStrings * 2 // Start after all length prefixes

			for i, vec := range tt.data {
				decoded[i] = make([]string, len(vec))
				for j := range vec {
					// Read string length
					strLen := system.ByteOrder.Uint16(actualData[strLenOffsetIdx:])
					strLenOffsetIdx += 2

					// Read string data
					decoded[i][j] = string(actualData[strDataOffsetIdx : strDataOffsetIdx+int(strLen)])
					strDataOffsetIdx += int(strLen)
				}
			}

			assert.Equal(t, tt.data, decoded)
		})
	}
}

func TestSerializeBoolVectorV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name          string
		data          [][]uint8
		vectorLengths []uint16
		wantErr       bool
		errMsg        string
		checkComp     bool
	}{
		{
			name: "basic vectors",
			data: [][]uint8{
				{1, 0, 1},    // first vector
				{0, 0, 1, 1}, // second vector
			},
			vectorLengths: []uint16{3, 4},
			wantErr:       false,
		},
		{
			name: "all true",
			data: [][]uint8{
				{1, 1, 1},
				{1, 1, 1, 1},
			},
			vectorLengths: []uint16{3, 4},
			wantErr:       false,
		},
		{
			name: "all false",
			data: [][]uint8{
				{0, 0, 0},
				{0, 0, 0, 0},
			},
			vectorLengths: []uint16{3, 4},
			wantErr:       false,
		},
		{
			name: "invalid bool value",
			data: [][]uint8{
				{1, 0, 1},
				{0, 2, 1}, // 2 is invalid
			},
			vectorLengths: []uint16{3, 3},
			wantErr:       true,
			errMsg:        "invalid bool value",
		},
		{
			name: "mismatched vector length",
			data: [][]uint8{
				{1, 0, 1},
				{0, 1}, // expected 3
			},
			vectorLengths: []uint16{3, 3},
			wantErr:       true,
			errMsg:        "mismatch in vector length",
		},
		{
			name:          "large vectors for compression",
			data:          generateLargeBoolVectors(100, 10), // 100 vectors of 10 bools each
			vectorLengths: generateFixedVectorLengths(100, 10),
			wantErr:       false,
			checkComp:     true,
		},
		{
			name: "byte boundary edge case",
			data: [][]uint8{
				{1, 0, 1, 0, 1, 0, 1, 0}, // fills exactly one byte
				{1, 1},                   // partial byte
			},
			vectorLengths: []uint16{8, 2},
			wantErr:       false,
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
			errMsg:  "bool v Data expected to come in [][]uint8 container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate total size needed (in bytes)
			totalBits := 0
			if tt.data != nil {
				for _, vec := range tt.data {
					totalBits += len(vec)
				}
			}
			totalBytes := (totalBits + 7) / 8 // Round up to nearest byte

			p := &PermStorageDataBlock{
				buf:                  make([]byte, PSDBLayout1LengthBytes),
				dataType:             types.DataTypeBoolVector,
				compressionType:      compression.TypeZSTD,
				Data:                 tt.data,
				vectorLengths:        tt.vectorLengths,
				originalData:         make([]byte, totalBytes),
				originalDataLen:      totalBytes,
				layoutVersion:        1,
				featureSchemaVersion: 1,
				expiryAt:             uint64(time.Now().Add(time.Hour).Unix()),
			}

			result, err := serializeBoolVectorV2(p)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkComp {
				assert.Equal(t, compression.TypeZSTD, p.compressionType)
				assert.Less(t, p.compressedDataLen, p.originalDataLen)
			}

			// Verify the data
			var actualData []byte
			if p.compressionType == compression.TypeZSTD {
				dec := compression.NewZStdDecoder()
				actualData, err = dec.Decode(result[PSDBLayout1LengthBytes:])
				assert.NoError(t, err)
			} else {
				actualData = result[PSDBLayout1LengthBytes:]
			}

			// Convert back to bool vectors
			decoded := make([][]uint8, len(tt.data))
			byteIdx := 0
			shift := 7

			for i, vec := range tt.data {
				decoded[i] = make([]uint8, len(vec))
				for j := range vec {
					// Extract bit
					decoded[i][j] = (actualData[byteIdx] >> shift) & 1
					shift--
					if shift < 0 {
						shift = 7
						byteIdx++
					}
				}
			}

			assert.Equal(t, tt.data, decoded)

			// Verify last byte's unused bits are zero
			if !tt.wantErr && tt.data != nil {
				lastByte := actualData[len(actualData)-1]
				unusedBits := (8 - ((totalBits-1)%8 + 1)) % 8
				mask := byte(0xFF << unusedBits)
				assert.Equal(t, byte(0), lastByte&^mask, "unused bits should be zero")
			}
		})
	}
}

// Helper function to generate test data
func generateLargeBoolVectors(numVectors, vectorLength int) [][]uint8 {
	vectors := make([][]uint8, numVectors)
	for i := range vectors {
		vectors[i] = make([]uint8, vectorLength)
		for j := range vectors[i] {
			// Alternate between 0 and 1
			vectors[i][j] = uint8((i + j) % 2)
		}
	}
	return vectors
}

// Helper functions for generating test data
func generateLargeFP64Vectors(numVectors, vectorLength int) [][]float64 {
	vectors := make([][]float64, numVectors)
	for i := range vectors {
		vectors[i] = make([]float64, vectorLength)
		for j := range vectors[i] {
			vectors[i][j] = float64(i) + float64(j)/10.0
		}
	}
	return vectors
}

func generateLargeInt64Vectors(numVectors, vectorLength int) [][]int64 {
	vectors := make([][]int64, numVectors)
	for i := range vectors {
		vectors[i] = make([]int64, vectorLength)
		for j := range vectors[i] {
			vectors[i][j] = int64(i*vectorLength + j)
		}
	}
	return vectors
}

func generateLargeUint64Vectors(numVectors, vectorLength int) [][]uint64 {
	vectors := make([][]uint64, numVectors)
	for i := range vectors {
		vectors[i] = make([]uint64, vectorLength)
		for j := range vectors[i] {
			vectors[i][j] = uint64(i*vectorLength + j)
		}
	}
	return vectors
}

// Helper functions
func generateLargeInt32Vectors(numVectors, vectorLength int) [][]int32 {
	vectors := make([][]int32, numVectors)
	for i := range vectors {
		vectors[i] = make([]int32, vectorLength)
		for j := range vectors[i] {
			vectors[i][j] = int32(i*vectorLength + j)
		}
	}
	return vectors
}

func generateLargeUint32Vectors(numVectors, vectorLength int) [][]uint32 {
	vectors := make([][]uint32, numVectors)
	for i := range vectors {
		vectors[i] = make([]uint32, vectorLength)
		for j := range vectors[i] {
			vectors[i][j] = uint32(i*vectorLength + j)
		}
	}
	return vectors
}

func generateLargeFloat32Vectors(numVectors, vectorLength int) [][]float32 {
	vectors := make([][]float32, numVectors)
	for i := range vectors {
		vectors[i] = make([]float32, vectorLength)
		for j := range vectors[i] {
			vectors[i][j] = float32(i) + float32(j)/10.0
		}
	}
	return vectors
}

func generateFixedVectorLengths(numVectors int, length uint16) []uint16 {
	lengths := make([]uint16, numVectors)
	for i := range lengths {
		lengths[i] = length
	}
	return lengths
}

// Helper functions for generating test data
func generateLargeFloat32Data(size int) []float32 {
	data := make([]float32, size)
	for i := 0; i < size; i++ {
		data[i] = float32(i)
	}
	return data
}

func generateLargeInt32Data(size int) []int32 {
	data := make([]int32, size)
	for i := 0; i < size; i++ {
		data[i] = int32(i)
	}
	return data
}

func generateLargeUint32Data(size int) []uint32 {
	data := make([]uint32, size)
	for i := 0; i < size; i++ {
		data[i] = uint32(i)
	}
	return data
}

func generateLargeFloat64Data(size int) []float64 {
	data := make([]float64, size)
	for i := 0; i < size; i++ {
		data[i] = float64(i) + 0.5
	}
	return data
}

func generateLargeInt64Data(size int) []int64 {
	data := make([]int64, size)
	for i := 0; i < size; i++ {
		data[i] = int64(i)
	}
	return data
}

func generateLargeUint64Data(size int) []uint64 {
	data := make([]uint64, size)
	for i := 0; i < size; i++ {
		data[i] = uint64(i)
	}
	return data
}

// Helper functions to generate test data
func generateRepetitiveStrings(count int, base string) []string {
	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = base
	}
	return result
}

func generateFixedLengths(count int, length uint16) []uint16 {
	result := make([]uint16, count)
	for i := 0; i < count; i++ {
		result[i] = length
	}
	return result
}

// Helper function to generate bool test data
func generateBoolData(size int) []uint8 {
	data := make([]uint8, size)
	for i := 0; i < size; i++ {
		if i%3 == 0 { // Create some pattern
			data[i] = 1
		}
	}
	return data
}

// Helper function to generate test data
func generateLargeStringVectors(numVectors, vectorLength int, strLength uint16) [][]string {
	vectors := make([][]string, numVectors)
	for i := range vectors {
		vectors[i] = make([]string, vectorLength)
		for j := range vectors[i] {
			// Generate string of specified length
			str := make([]byte, strLength)
			for k := range str {
				str[k] = byte('a' + (i+j)%26) // cycle through alphabet
			}
			vectors[i][j] = string(str)
		}
	}
	return vectors
}
