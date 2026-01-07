package blocks

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeserializePSDBV2(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		buildFunc func() (*PermStorageDataBlock, error)
		wantErr   bool
		validate  func(*testing.T, *DeserializedPSDB)
	}{
		{
			name: "scalar int32",
			buildFunc: func() (*PermStorageDataBlock, error) {
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt32).
					SetScalarValues([]int32{1, 2, 3}, 3).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, uint16(1), d.FeatureSchemaVersion)
				assert.Equal(t, uint8(1), d.LayoutVersion)
				assert.Equal(t, types.DataTypeInt32, d.DataType)
				assert.Equal(t, compression.TypeNone, d.CompressionType)
				assert.Equal(t, 12, len(d.OriginalData)) // 3 * 4 bytes
				assert.False(t, d.Expired)
				assert.False(t, d.NegativeCache)
			},
		},
		{
			name: "small vector small string with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeStringVector).
					SetStringVectorValues(
						[][]string{
							{"abc", "def"},
							{"ghi", "jkl", "mno"},
						},
						2,
						[]uint16{5, 5}, // max string lengths
						[]uint16{2, 3}, // vector lengths
					).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, types.DataTypeStringVector, d.DataType)
				assert.Equal(t, compression.TypeNone, d.CompressionType)
				assert.NotNil(t, d.CompressedData)
				assert.NotNil(t, d.OriginalData)
				assert.False(t, d.Expired)
				assert.False(t, d.NegativeCache)
			},
		},
		{
			name: "bool vector",
			buildFunc: func() (*PermStorageDataBlock, error) {
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeBoolVector).
					SetVectorValues(
						[][]uint8{
							{1, 0, 1},
							{0, 1, 1, 1},
						},
						2,
						[]uint16{3, 4},
					).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, types.DataTypeBoolVector, d.DataType)
				assert.Equal(t, compression.TypeNone, d.CompressionType)
				// 7 bits total, rounded up to 1 byte
				assert.Equal(t, 1, len(d.OriginalData))
				assert.False(t, d.Expired)
				assert.False(t, d.NegativeCache)
			},
		},
		//{
		//	name: "expired data",
		//	buildFunc: func() (*PermStorageDataBlock, error) {
		//		return NewPermStorageDataBlockBuilder().
		//			SetID(1).
		//			SetVersion(1).
		//			SetTTL(0). // immediate expiry in nanosec
		//			SetCompressionB(compression.TypeNone).
		//			SetDataType(types.DataTypeInt32).
		//			SetScalarValues([]int32{1}, 1).
		//			Build()
		//	},
		//	validate: func(t *testing.T, d *DeserializedPSDB) {
		//		assert.True(t, d.Expired)
		//	},
		//},
		{
			name: "invalid layout version",
			buildFunc: func() (*PermStorageDataBlock, error) {
				return NewPermStorageDataBlockBuilder().
					SetID(2). // invalid layout version
					SetVersion(1).
					SetTTL(3600).
					SetDataType(types.DataTypeInt32).
					SetScalarValues([]int32{1}, 1).
					Build()
			},
			wantErr: true,
		},
		{
			name: "scalar float32",
			buildFunc: func() (*PermStorageDataBlock, error) {
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeFP32).
					SetScalarValues([]float32{1.1, 2.2, 3.3}, 3).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, types.DataTypeFP32, d.DataType)
				assert.Equal(t, compression.TypeNone, d.CompressionType)
				assert.Equal(t, 12, len(d.OriginalData)) // 3 * 4 bytes
			},
		},
		{
			name: "scalar float64",
			buildFunc: func() (*PermStorageDataBlock, error) {
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeFP64).
					SetScalarValues([]float64{1.1, 2.2, 3.3}, 3).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, types.DataTypeFP64, d.DataType)
				assert.Equal(t, compression.TypeNone, d.CompressionType)
				assert.Equal(t, 24, len(d.OriginalData)) // 3 * 8 bytes
			},
		},
		{
			name: "scalar int8",
			buildFunc: func() (*PermStorageDataBlock, error) {
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt8).
					SetScalarValues([]int32{1, 2, 3}, 3).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, types.DataTypeInt8, d.DataType)
				assert.Equal(t, compression.TypeNone, d.CompressionType)
				assert.Equal(t, 3, len(d.OriginalData)) // 3 * 1 byte
			},
		},
		{
			name: "scalar int16",
			buildFunc: func() (*PermStorageDataBlock, error) {
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt16).
					SetScalarValues([]int32{1, 2, 3}, 3).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, types.DataTypeInt16, d.DataType)
				assert.Equal(t, compression.TypeNone, d.CompressionType)
				assert.Equal(t, 6, len(d.OriginalData)) // 3 * 2 bytes
			},
		},
		{
			name: "scalar int64",
			buildFunc: func() (*PermStorageDataBlock, error) {
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt64).
					SetScalarValues([]int64{1, 2, 3}, 3).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, types.DataTypeInt64, d.DataType)
				assert.Equal(t, compression.TypeNone, d.CompressionType)
				assert.Equal(t, 24, len(d.OriginalData)) // 3 * 8 bytes
			},
		},
		{
			name: "float32 vector",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := [][]float32{
					{1.1, 2.2},
					{3.3, 4.4, 5.5},
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeFP32Vector).
					SetVectorValues(values, len(values), []uint16{2, 3}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, types.DataTypeFP32Vector, d.DataType)
				assert.Equal(t, compression.TypeNone, d.CompressionType)
				assert.Equal(t, 20, len(d.OriginalData)) // (2+3) * 4 bytes
			},
		},
		{
			name: "float64 vector",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := [][]float64{
					{1.1, 2.2},
					{3.3, 4.4, 5.5},
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeFP64Vector).
					SetVectorValues(values, len(values), []uint16{2, 3}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, types.DataTypeFP64Vector, d.DataType)
				assert.Equal(t, compression.TypeNone, d.CompressionType)
				assert.Equal(t, 40, len(d.OriginalData)) // (2+3) * 8 bytes
			},
		},
		{
			name: "int8 vector",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := [][]int32{
					{1, 2},
					{3, 4, 5},
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt8Vector).
					SetVectorValues(values, len(values), []uint16{2, 3}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, types.DataTypeInt8Vector, d.DataType)
				assert.Equal(t, compression.TypeNone, d.CompressionType)
				assert.Equal(t, 5, len(d.OriginalData)) // (2+3) * 1 byte
			},
		},
		{
			name: "int16 vector",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := [][]int32{
					{1, 2},
					{3, 4, 5},
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt16Vector).
					SetVectorValues(values, len(values), []uint16{2, 3}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, types.DataTypeInt16Vector, d.DataType)
				assert.Equal(t, compression.TypeNone, d.CompressionType)
				assert.Equal(t, 10, len(d.OriginalData)) // (2+3) * 2 bytes
			},
		},
		{
			name: "int64 vector",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := [][]int64{
					{1, 2},
					{3, 4, 5},
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt64Vector).
					SetVectorValues(values, len(values), []uint16{2, 3}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, types.DataTypeInt64Vector, d.DataType)
				assert.Equal(t, compression.TypeNone, d.CompressionType)
				assert.Equal(t, 40, len(d.OriginalData)) // (2+3) * 8 bytes
			},
		},
		// {
		// 	name: "too short data",
		// 	buildFunc: func() (*PermStorageDataBlock, error) {
		// 		// Return a truncated/invalid data block
		// 		return &PermStorageDataBlock{
		// 			buf: make([]byte, 8), // Less than minimum required header size
		// 		}, nil
		// 	},
		// 	wantErr: true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			psdb, err := tt.buildFunc()
			require.NoError(t, err)

			serialized, err := psdb.Serialize()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			deserialized, err := DeserializePSDB(serialized)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, deserialized)

			if tt.validate != nil {
				tt.validate(t, deserialized)
			}
		})
	}
}

func TestDeserializePSDBV2_Features(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		buildFunc func() (*PermStorageDataBlock, error)
		validate  func(*testing.T, *DeserializedPSDB)
	}{
		{
			name: "int32 scalar feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := []int32{1, 2, 3}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt32).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				// Test each position
				for pos := 0; pos < 3; pos++ {
					feature, err := d.GetNumericScalarFeature(pos, 3, []byte{0, 0, 0})
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt32(feature)
					require.NoError(t, err)
					assert.Equal(t, int32(pos+1), value)
				}
			},
		},
		{
			name: "int32 vector feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := [][]int32{
					{1, 2},
					{3, 4, 5},
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt32Vector).
					SetVectorValues(values, len(values), []uint16{2, 3}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				// Test each vector
				vectorLengths := []uint16{2, 3}
				expectedValues := [][]int32{{1, 2}, {3, 4, 5}}

				for pos := 0; pos < 2; pos++ {
					feature, err := d.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					values, err := HelperVectorFeatureToTypeInt32(feature)
					require.NoError(t, err)
					assert.Equal(t, expectedValues[pos], values)
				}
			},
		},
		{
			name: "string scalar feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := []string{"abc", "def", "ghi"}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeString).
					SetStringScalarValues(values, len(values), []uint16{3, 3, 3}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				expectedValues := []string{"abc", "def", "ghi"}
				for pos := 0; pos < 3; pos++ {
					feature, err := d.GetStringScalarFeature(pos, 3)
					require.NoError(t, err)

					value, err := HelperScalarFeatureToTypeString(feature)
					require.NoError(t, err)
					assert.Equal(t, expectedValues[pos], value)
				}
			},
		},
		{
			name: "string vector feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := [][]string{
					{"abc", "def"},
					{"ghi", "jkl", "mno"},
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeStringVector).
					SetStringVectorValues(values, len(values), []uint16{3, 3}, []uint16{2, 3}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				vectorLengths := []uint16{2, 3}
				expectedValues := [][]string{{"abc", "def"}, {"ghi", "jkl", "mno"}}

				for pos := 0; pos < 2; pos++ {
					feature, err := d.GetStringVectorFeature(pos, 2, vectorLengths)
					require.NoError(t, err)

					result, err := HelperVectorFeatureStringToConcatenatedString(feature, int(vectorLengths[pos]))
					require.NoError(t, err)
					expected := strings.Join(expectedValues[pos], ":")
					assert.Equal(t, expected, result)
				}
			},
		},
		{
			name: "bool scalar feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := []uint8{1, 0, 1, 1}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeBool).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				expectedValues := []bool{true, false, true, true}
				for pos := 0; pos < 4; pos++ {
					feature, err := d.GetBoolScalarFeature(pos)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeBool(feature)
					require.NoError(t, err)
					assert.Equal(t, expectedValues[pos], value)
				}
			},
		},
		{
			name: "bool vector feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := [][]uint8{
					{1, 0, 1},
					{0, 1, 1, 1},
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeBoolVector).
					SetVectorValues(values, len(values), []uint16{3, 4}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				vectorLengths := []uint16{3, 4}
				expectedValues := [][]bool{{true, false, true}, {false, true, true, true}}
				fmt.Printf("d.OriginalData %v\n", d.OriginalData)
				for pos := 0; pos < 2; pos++ {
					feature, err := d.GetBoolVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					fmt.Printf("feature %v\n", feature)
					result, err := HelperVectorFeatureBoolToConcatenatedString(feature)
					require.NoError(t, err)
					expected := make([]string, len(expectedValues[pos]))
					for i, val := range expectedValues[pos] {
						if val {
							expected[i] = "1"
						} else {
							expected[i] = "0"
						}
					}
					assert.Equal(t, strings.Join(expected, ":"), result)
				}
			},
		},
		{
			name: "float32 scalar feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := []float32{1.1, 2.2, 3.3}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeFP32).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				expectedValues := []float32{1.1, 2.2, 3.3}
				for pos := 0; pos < 3; pos++ {
					feature, err := d.GetNumericScalarFeature(pos, 3, []byte{0, 0, 0})
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeFloat32(feature)
					require.NoError(t, err)
					assert.InDelta(t, expectedValues[pos], value, 0.0001)
				}
			},
		},
		{
			name: "float64 scalar feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := []float64{1.1, 2.2, 3.3}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeFP64).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				expectedValues := []float64{1.1, 2.2, 3.3}
				for pos := 0; pos < 3; pos++ {
					feature, err := d.GetNumericScalarFeature(pos, 3, []byte{0, 0, 0})
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeFloat64(feature)
					require.NoError(t, err)
					assert.InDelta(t, expectedValues[pos], value, 0.0001)
				}
			},
		},
		{
			name: "int8 scalar feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := []int32{1, 2, 3}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt8).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				expectedValues := []int8{1, 2, 3}
				for pos := 0; pos < 3; pos++ {
					feature, err := d.GetNumericScalarFeature(pos, 3, []byte{0, 0, 0})
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt8(feature)
					require.NoError(t, err)
					assert.Equal(t, expectedValues[pos], value)
				}
			},
		},
		{
			name: "int16 scalar feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := []int32{1, 2, 3}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt16).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				expectedValues := []int16{1, 2, 3}
				for pos := 0; pos < 3; pos++ {
					feature, err := d.GetNumericScalarFeature(pos, 3, []byte{0, 0, 0})
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt16(feature)
					require.NoError(t, err)
					assert.Equal(t, expectedValues[pos], value)
				}
			},
		},
		{
			name: "int64 scalar feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := []int64{1, 2, 3}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt64).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				expectedValues := []int64{1, 2, 3}
				for pos := 0; pos < 3; pos++ {
					feature, err := d.GetNumericScalarFeature(pos, 3, []byte{0, 0, 0})
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt64(feature)
					require.NoError(t, err)
					assert.Equal(t, expectedValues[pos], value)
				}
			},
		},
		{
			name: "float32 vector feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := [][]float32{
					{1.1, 2.2},
					{3.3, 4.4, 5.5},
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeFP32Vector).
					SetVectorValues(values, len(values), []uint16{2, 3}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				vectorLengths := []uint16{2, 3}
				expectedValues := [][]float32{{1.1, 2.2}, {3.3, 4.4, 5.5}}

				for pos := 0; pos < 2; pos++ {
					feature, err := d.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureFp32ToConcatenatedString(feature)
					require.NoError(t, err)
					values := strings.Split(result, ":")
					for i, v := range values {
						var floatVal float32
						fmt.Sscanf(v, "%f", &floatVal)
						assert.InDelta(t, expectedValues[pos][i], floatVal, 0.0001)
					}
				}
			},
		},
		{
			name: "float64 vector feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := [][]float64{
					{1.1, 2.2},
					{3.3, 4.4, 5.5},
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeFP64Vector).
					SetVectorValues(values, len(values), []uint16{2, 3}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				vectorLengths := []uint16{2, 3}
				expectedValues := [][]float64{{1.1, 2.2}, {3.3, 4.4, 5.5}}

				for pos := 0; pos < 2; pos++ {
					feature, err := d.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureFp64ToConcatenatedString(feature)
					require.NoError(t, err)
					values := strings.Split(result, ":")
					for i, v := range values {
						var floatVal float64
						fmt.Sscanf(v, "%f", &floatVal)
						assert.InDelta(t, expectedValues[pos][i], floatVal, 0.0001)
					}
				}
			},
		},
		{
			name: "int8 vector feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := [][]int32{
					{1, 2},
					{3, 4, 5},
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt8Vector).
					SetVectorValues(values, len(values), []uint16{2, 3}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				vectorLengths := []uint16{2, 3}
				expectedValues := [][]int8{{1, 2}, {3, 4, 5}}

				for pos := 0; pos < 2; pos++ {
					feature, err := d.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureInt8ToConcatenatedString(feature)
					require.NoError(t, err)
					expected := make([]string, len(expectedValues[pos]))
					for i, v := range expectedValues[pos] {
						expected[i] = fmt.Sprintf("%d", v)
					}
					assert.Equal(t, strings.Join(expected, ":"), result)
				}
			},
		},
		{
			name: "int16 vector feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := [][]int32{
					{1, 2},
					{3, 4, 5},
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt16Vector).
					SetVectorValues(values, len(values), []uint16{2, 3}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				vectorLengths := []uint16{2, 3}
				expectedValues := [][]int16{{1, 2}, {3, 4, 5}}

				for pos := 0; pos < 2; pos++ {
					feature, err := d.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureInt16ToConcatenatedString(feature)
					require.NoError(t, err)
					expected := make([]string, len(expectedValues[pos]))
					for i, v := range expectedValues[pos] {
						expected[i] = fmt.Sprintf("%d", v)
					}
					assert.Equal(t, strings.Join(expected, ":"), result)
				}
			},
		},
		{
			name: "int64 vector feature",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := [][]int64{
					{1, 2},
					{3, 4, 5},
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeNone).
					SetDataType(types.DataTypeInt64Vector).
					SetVectorValues(values, len(values), []uint16{2, 3}).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				vectorLengths := []uint16{2, 3}
				expectedValues := [][]int64{{1, 2}, {3, 4, 5}}

				for pos := 0; pos < 2; pos++ {
					feature, err := d.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureInt64ToConcatenatedString(feature)
					require.NoError(t, err)
					expected := make([]string, len(expectedValues[pos]))
					for i, v := range expectedValues[pos] {
						expected[i] = fmt.Sprintf("%d", v)
					}
					assert.Equal(t, strings.Join(expected, ":"), result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build and serialize
			psdb, err := tt.buildFunc()
			require.NoError(t, err)

			serialized, err := psdb.Serialize()
			require.NoError(t, err)

			// Deserialize
			deserialized, err := DeserializePSDB(serialized)
			require.NoError(t, err)
			require.NotNil(t, deserialized)

			// Validate features
			tt.validate(t, deserialized)
		})
	}
}

func TestDeserializePSDBV2LargeData(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		buildFunc func() (*PermStorageDataBlock, error)
		wantErr   bool
		validate  func(*testing.T, *DeserializedPSDB)
	}{
		{
			name: "large scalar int32 with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([]int32, 10000)
				for i := range values {
					values[i] = int32(i)
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt32).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, compression.TypeZSTD, d.CompressionType)
				assert.NotNil(t, d.CompressedData)
				assert.Equal(t, 40000, len(d.OriginalData)) // 10000 * 4 bytes
			},
		},
		{
			name: "large string vector with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([][]string, 1000)
				maxLens := make([]uint16, 1000)
				vecLens := make([]uint16, 1000)
				for i := range values {
					vecLen := 5 + (i % 5) // varying vector lengths between 5 and 9
					values[i] = make([]string, vecLen)
					for j := range values[i] {
						values[i][j] = fmt.Sprintf("str_%d_%d", i, j)
					}
					maxLens[i] = 10 // max string length
					vecLens[i] = uint16(vecLen)
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeStringVector).
					SetStringVectorValues(values, len(values), maxLens, vecLens).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, compression.TypeZSTD, d.CompressionType)
				assert.NotNil(t, d.CompressedData)
				assert.NotNil(t, d.OriginalData)
			},
		},
		{
			name: "large bool vector with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([][]uint8, 1000)
				vecLens := make([]uint16, 1000)
				for i := range values {
					vecLen := 8 + (i % 8) // varying vector lengths between 8 and 15
					values[i] = make([]uint8, vecLen)
					for j := range values[i] {
						values[i][j] = uint8(j % 2) // alternating true/false
					}
					vecLens[i] = uint16(vecLen)
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeBoolVector).
					SetVectorValues(values, len(values), vecLens).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				assert.Equal(t, compression.TypeZSTD, d.CompressionType)
				assert.NotNil(t, d.CompressedData)
				assert.NotNil(t, d.OriginalData)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			psdb, err := tt.buildFunc()
			require.NoError(t, err)

			serialized, err := psdb.Serialize()
			require.NoError(t, err)

			deserialized, err := DeserializePSDB(serialized)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, deserialized)

			if tt.validate != nil {
				tt.validate(t, deserialized)
			}
		})
	}
}

func TestDeserializePSDBV2_FeaturesLargeData(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		buildFunc func() (*PermStorageDataBlock, error)
		validate  func(*testing.T, *DeserializedPSDB)
	}{
		{
			name: "large int32 scalar feature with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([]int32, 10000)
				for i := range values {
					values[i] = int32(i * 2)
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt32).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				// Test random positions
				for _, pos := range []int{0, 100, 1000, 9999} {
					feature, err := d.GetNumericScalarFeature(pos, 3, []byte{0, 0, 0})
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt32(feature)
					require.NoError(t, err)
					assert.Equal(t, int32(pos*2), value)
				}
			},
		},
		{
			name: "large string scalar feature with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([]string, 5000)
				maxLens := make([]uint16, 5000)
				for i := range values {
					values[i] = fmt.Sprintf("test_string_%d", i)
					maxLens[i] = 20
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeString).
					SetStringScalarValues(values, len(values), maxLens).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				// Test random positions
				for _, pos := range []int{0, 50, 500, 4999} {
					feature, err := d.GetStringScalarFeature(pos, 5000)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeString(feature)
					require.NoError(t, err)
					assert.Equal(t, fmt.Sprintf("test_string_%d", pos), value)
				}
			},
		},
		{
			name: "large bool vector feature with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([][]uint8, 1000)
				vecLens := make([]uint16, 1000)
				for i := range values {
					vecLen := 16 // fixed length for easier testing
					values[i] = make([]uint8, vecLen)
					for j := range values[i] {
						values[i][j] = uint8((i + j) % 2) // pattern based on position
					}
					vecLens[i] = uint16(vecLen)
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeBoolVector).
					SetVectorValues(values, len(values), vecLens).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				vectorLengths := make([]uint16, 1000)
				for i := range vectorLengths {
					vectorLengths[i] = 16
				}

				// Test random positions
				for _, pos := range []int{0, 50, 500, 999} {
					feature, err := d.GetBoolVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureBoolToConcatenatedString(feature)
					require.NoError(t, err)

					// Verify the pattern
					valuesArray := strings.Split(result, ":")
					for j, valStr := range valuesArray {
						expected := (pos+j)%2 == 1
						expectedStr := "0"
						if expected {
							expectedStr = "1"
						}
						assert.Equal(t, expectedStr, valStr,
							"Mismatch at position %d, index %d", pos, j)
					}
				}
			},
		},
		{
			name: "large float32 scalar feature with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([]float32, 10000)
				for i := range values {
					values[i] = float32(i) * 1.5
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeFP32).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				// Test random positions
				for _, pos := range []int{0, 100, 1000, 9999} {
					feature, err := d.GetNumericScalarFeature(pos, 3, []byte{0, 0, 0})
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeFloat32(feature)
					require.NoError(t, err)
					assert.InDelta(t, float32(pos)*1.5, value, 0.0001)
				}
			},
		},
		{
			name: "large float64 scalar feature with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([]float64, 10000)
				for i := range values {
					values[i] = float64(i) * 2.5
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeFP64).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				// Test random positions
				for _, pos := range []int{0, 100, 1000, 9999} {
					feature, err := d.GetNumericScalarFeature(pos, 3, []byte{0, 0, 0})
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeFloat64(feature)
					require.NoError(t, err)
					assert.InDelta(t, float64(pos)*2.5, value, 0.0001)
				}
			},
		},
		{
			name: "large int8 scalar feature with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([]int32, 1000)
				for i := range values {
					values[i] = int32(i % 128) // Stay within int8 range
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt8).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				// Test random positions
				for _, pos := range []int{0, 50, 100, 999} {
					feature, err := d.GetNumericScalarFeature(pos, 3, []byte{0, 0, 0})
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt8(feature)
					require.NoError(t, err)
					assert.Equal(t, int8(pos%128), value)
				}
			},
		},
		{
			name: "large int16 scalar feature with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([]int32, 5000)
				for i := range values {
					values[i] = int32(i)
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt16).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				// Test random positions
				for _, pos := range []int{0, 100, 1000, 4999} {
					feature, err := d.GetNumericScalarFeature(pos, 3, []byte{0, 0, 0})
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt16(feature)
					require.NoError(t, err)
					assert.Equal(t, int16(pos), value)
				}
			},
		},
		{
			name: "large int64 scalar feature with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([]int64, 10000)
				for i := range values {
					values[i] = int64(i) * 1000000
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt64).
					SetScalarValues(values, len(values)).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				// Test random positions
				for _, pos := range []int{0, 100, 1000, 9999} {
					feature, err := d.GetNumericScalarFeature(pos, 3, []byte{0, 0, 0})
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt64(feature)
					require.NoError(t, err)
					assert.Equal(t, int64(pos)*1000000, value)
				}
			},
		},
		{
			name: "large float32 vector feature with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([][]float32, 1000)
				vecLens := make([]uint16, 1000)
				for i := range values {
					vecLen := 10 + (i % 5) // varying vector lengths between 10 and 14
					values[i] = make([]float32, vecLen)
					for j := range values[i] {
						values[i][j] = float32(i*100+j) * 1.5
					}
					vecLens[i] = uint16(vecLen)
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeFP32Vector).
					SetVectorValues(values, len(values), vecLens).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				vectorLengths := make([]uint16, 1000)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 5))
				}

				// Test random positions
				for _, pos := range []int{0, 50, 500, 999} {
					feature, err := d.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureFp32ToConcatenatedString(feature)
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 5)
					assert.Equal(t, vecLen, len(values))
					for j, v := range values {
						var floatVal float32
						fmt.Sscanf(v, "%f", &floatVal)
						expected := float32(pos*100+j) * 1.5
						assert.InDelta(t, expected, floatVal, 0.0001)
					}
				}
			},
		},
		{
			name: "large float64 vector feature with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([][]float64, 1000)
				vecLens := make([]uint16, 1000)
				for i := range values {
					vecLen := 10 + (i % 5) // varying vector lengths between 10 and 14
					values[i] = make([]float64, vecLen)
					for j := range values[i] {
						values[i][j] = float64(i*100+j) * 2.5
					}
					vecLens[i] = uint16(vecLen)
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeFP64Vector).
					SetVectorValues(values, len(values), vecLens).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				vectorLengths := make([]uint16, 1000)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 5))
				}

				// Test random positions
				for _, pos := range []int{0, 50, 500, 999} {
					feature, err := d.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureFp64ToConcatenatedString(feature)
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 5)
					assert.Equal(t, vecLen, len(values))
					for j, v := range values {
						var floatVal float64
						fmt.Sscanf(v, "%f", &floatVal)
						expected := float64(pos*100+j) * 2.5
						assert.InDelta(t, expected, floatVal, 0.0001)
					}
				}
			},
		},
		{
			name: "large int8 vector feature with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([][]int32, 1000)
				vecLens := make([]uint16, 1000)
				for i := range values {
					vecLen := 10 + (i % 5) // varying vector lengths between 10 and 14
					values[i] = make([]int32, vecLen)
					for j := range values[i] {
						values[i][j] = int32((i + j) % 128) // Stay within int8 range
					}
					vecLens[i] = uint16(vecLen)
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt8Vector).
					SetVectorValues(values, len(values), vecLens).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				vectorLengths := make([]uint16, 1000)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 5))
				}

				// Test random positions
				for _, pos := range []int{0, 50, 500, 999} {
					feature, err := d.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureInt8ToConcatenatedString(feature)
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 5)
					assert.Equal(t, vecLen, len(values))
					for j, v := range values {
						var intVal int
						fmt.Sscanf(v, "%d", &intVal)
						expected := int8((pos + j) % 128)
						assert.Equal(t, int(expected), intVal)
					}
				}
			},
		},
		{
			name: "large int16 vector feature with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([][]int32, 1000)
				vecLens := make([]uint16, 1000)
				for i := range values {
					vecLen := 10 + (i % 5) // varying vector lengths between 10 and 14
					values[i] = make([]int32, vecLen)
					for j := range values[i] {
						values[i][j] = int32(i*100 + j)
					}
					vecLens[i] = uint16(vecLen)
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt16Vector).
					SetVectorValues(values, len(values), vecLens).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				vectorLengths := make([]uint16, 1000)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 5))
				}

				// Test random positions
				for _, pos := range []int{0, 50, 500, 999} {
					feature, err := d.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureInt16ToConcatenatedString(feature)
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 5)
					assert.Equal(t, vecLen, len(values))
					for j, v := range values {
						var intVal int
						fmt.Sscanf(v, "%d", &intVal)
						expected := int16(pos*100 + j)
						assert.Equal(t, int(expected), intVal)
					}
				}
			},
		},
		{
			name: "large int64 vector feature with compression",
			buildFunc: func() (*PermStorageDataBlock, error) {
				values := make([][]int64, 1000)
				vecLens := make([]uint16, 1000)
				for i := range values {
					vecLen := 10 + (i % 5) // varying vector lengths between 10 and 14
					values[i] = make([]int64, vecLen)
					for j := range values[i] {
						values[i][j] = int64(i*100+j) * 1000000
					}
					vecLens[i] = uint16(vecLen)
				}
				return NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt64Vector).
					SetVectorValues(values, len(values), vecLens).
					Build()
			},
			validate: func(t *testing.T, d *DeserializedPSDB) {
				vectorLengths := make([]uint16, 1000)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 5))
				}

				// Test random positions
				for _, pos := range []int{0, 50, 500, 999} {
					feature, err := d.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureInt64ToConcatenatedString(feature)
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 5)
					assert.Equal(t, vecLen, len(values))
					for j, v := range values {
						var intVal int64
						fmt.Sscanf(v, "%d", &intVal)
						expected := int64(pos*100+j) * 1000000
						assert.Equal(t, expected, intVal)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			psdb, err := tt.buildFunc()
			require.NoError(t, err)

			serialized, err := psdb.Serialize()
			require.NoError(t, err)

			deserialized, err := DeserializePSDB(serialized)
			require.NoError(t, err)
			require.NotNil(t, deserialized)

			tt.validate(t, deserialized)
		})
	}
}
