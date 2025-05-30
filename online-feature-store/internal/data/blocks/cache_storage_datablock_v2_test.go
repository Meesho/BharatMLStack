package blocks

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/ds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializeForInMemoryInt32(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		buildFunc func() (*CacheStorageDataBlock, error)
		validate  func(*testing.T, []byte)
	}{
		{
			name: "int32 scalar data",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetDataType(types.DataTypeInt32).
					SetScalarValues([]int32{1, 2, 3}, 3).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeInt32, ddb.DataType)

				// Verify all values
				for i, expected := range []int32{1, 2, 3} {
					feature, err := ddb.GetNumericScalarFeature(i)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt32(feature)
					require.NoError(t, err)
					assert.Equal(t, expected, value)
				}
			},
		},
		{
			name: "int32 scalar large data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				values := make([]int32, 10000)
				for i := range values {
					values[i] = int32(i * 100)
				}
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt32).
					SetScalarValues(values, len(values)).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeInt32, ddb.DataType)

				// Test random positions
				testPositions := []int{0, 42, 1000, 5000, 9999}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericScalarFeature(pos)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt32(feature)
					require.NoError(t, err)
					assert.Equal(t, int32(pos*100), value)
				}
			},
		},
		{
			name: "int32 vector data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				numVectors := 1000
				values := make([][]int32, numVectors)
				vectorLengths := make([]uint16, numVectors)

				for i := range values {
					vecLen := 10 + (i % 5) // varying lengths between 10 and 14
					values[i] = make([]int32, vecLen)
					for j := range values[i] {
						values[i][j] = int32(i*100 + j)
					}
					vectorLengths[i] = uint16(vecLen)
				}

				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt32Vector).
					SetVectorValues(values, len(values), vectorLengths).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeInt32Vector, ddb.DataType)

				vectorLengths := make([]uint16, 1000)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 5))
				}

				// Test random positions
				testPositions := []int{0, 42, 123, 456, 789, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureInt32ToConcatenatedString(feature)
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 5)
					assert.Equal(t, vecLen, len(values))

					for j, valStr := range values {
						var intVal int32
						fmt.Sscanf(valStr, "%d", &intVal)
						expected := int32(pos*100 + j)
						assert.Equal(t, expected, intVal)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csdb, err := tt.buildFunc()
			require.NoError(t, err)

			data, err := csdb.SerializeForInMemory()
			require.NoError(t, err)
			require.NotNil(t, data)

			tt.validate(t, data)
		})
	}
}

func TestSerializeForInMemoryInt8(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		buildFunc func() (*CacheStorageDataBlock, error)
		validate  func(*testing.T, []byte)
	}{
		{
			name: "int8 scalar data",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetDataType(types.DataTypeInt8).
					SetScalarValues([]int32{1, 2, 3}, 3).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeInt8, ddb.DataType)

				// Verify all values
				for i, expected := range []int8{1, 2, 3} {
					feature, err := ddb.GetNumericScalarFeature(i)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt8(feature)
					require.NoError(t, err)
					assert.Equal(t, expected, value)
				}
			},
		},
		{
			name: "int8 scalar large data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				values := make([]int32, 1000)
				for i := range values {
					values[i] = int32(i % 128) // Stay within int8 range but as int32
				}
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt8).
					SetScalarValues(values, len(values)).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeInt8, ddb.DataType)

				// Test random positions
				testPositions := []int{0, 42, 100, 500, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericScalarFeature(pos)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt8(feature)
					require.NoError(t, err)
					assert.Equal(t, int8(pos%128), value)
				}
			},
		},
		{
			name: "int8 vector data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				numVectors := 1000
				values := make([][]int32, numVectors)
				vectorLengths := make([]uint16, numVectors)

				for i := range values {
					vecLen := 10 + (i % 5) // varying lengths between 10 and 14
					values[i] = make([]int32, vecLen)
					for j := range values[i] {
						values[i][j] = int32((i*10 + j) % 128) // Stay within int8 range but as int32
					}
					vectorLengths[i] = uint16(vecLen)
				}

				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt8Vector).
					SetVectorValues(values, len(values), vectorLengths).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeInt8Vector, ddb.DataType)

				vectorLengths := make([]uint16, 1000)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 5))
				}

				// Test random positions
				testPositions := []int{0, 42, 123, 456, 789, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureInt8ToConcatenatedString(feature)
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 5)
					assert.Equal(t, vecLen, len(values))

					for j, valStr := range values {
						var intVal int8
						fmt.Sscanf(valStr, "%d", &intVal)
						expected := int8((pos*10 + j) % 128)
						assert.Equal(t, expected, intVal,
							"Mismatch at position %d, index %d", pos, j)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csdb, err := tt.buildFunc()
			require.NoError(t, err)

			data, err := csdb.SerializeForInMemory()
			require.NoError(t, err)
			require.NotNil(t, data)

			tt.validate(t, data)
		})
	}
}

func TestSerializeForInMemoryInt16(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		buildFunc func() (*CacheStorageDataBlock, error)
		validate  func(*testing.T, []byte)
	}{
		{
			name: "int16 scalar data",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetDataType(types.DataTypeInt16).
					SetScalarValues([]int32{1000, 2000, 3000}, 3).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeInt16, ddb.DataType)

				// Verify all values
				for i, expected := range []int16{1000, 2000, 3000} {
					feature, err := ddb.GetNumericScalarFeature(i)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt16(feature)
					require.NoError(t, err)
					assert.Equal(t, expected, value)
				}
			},
		},
		{
			name: "int16 scalar large data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				values := make([]int32, 1000)
				for i := range values {
					values[i] = int32(i * 30) // Stay within int16 range
				}
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt16).
					SetScalarValues(values, len(values)).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeInt16, ddb.DataType)

				// Test random positions
				testPositions := []int{0, 42, 100, 500, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericScalarFeature(pos)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt16(feature)
					require.NoError(t, err)
					assert.Equal(t, int16(pos*30), value)
				}
			},
		},
		{
			name: "int16 vector data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				numVectors := 1000
				values := make([][]int32, numVectors)
				vectorLengths := make([]uint16, numVectors)

				for i := range values {
					vecLen := 10 + (i % 5) // varying lengths between 10 and 14
					values[i] = make([]int32, vecLen)
					for j := range values[i] {
						values[i][j] = int32((i*100 + j) * 20) // Stay within int16 range
					}
					vectorLengths[i] = uint16(vecLen)
				}

				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt16Vector).
					SetVectorValues(values, len(values), vectorLengths).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeInt16Vector, ddb.DataType)

				vectorLengths := make([]uint16, 1000)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 5))
				}

				// Test random positions
				testPositions := []int{0, 42, 123, 456, 789, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureInt16ToConcatenatedString(feature)
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 5)
					assert.Equal(t, vecLen, len(values))

					for j, valStr := range values {
						var intVal int16
						fmt.Sscanf(valStr, "%d", &intVal)
						expected := int16((pos*100 + j) * 20)
						assert.Equal(t, expected, intVal,
							"Mismatch at position %d, index %d", pos, j)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csdb, err := tt.buildFunc()
			require.NoError(t, err)

			data, err := csdb.SerializeForInMemory()
			require.NoError(t, err)
			require.NotNil(t, data)

			tt.validate(t, data)
		})
	}
}

func TestSerializeForInMemoryInt64(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		buildFunc func() (*CacheStorageDataBlock, error)
		validate  func(*testing.T, []byte)
	}{
		{
			name: "int64 scalar data",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetDataType(types.DataTypeInt64).
					SetScalarValues([]int64{1000000000000, 2000000000000, 3000000000000}, 3).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeInt64, ddb.DataType)

				// Verify all values
				for i, expected := range []int64{1000000000000, 2000000000000, 3000000000000} {
					feature, err := ddb.GetNumericScalarFeature(i)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt64(feature)
					require.NoError(t, err)
					assert.Equal(t, expected, value)
				}
			},
		},
		{
			name: "int64 scalar large data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				values := make([]int64, 1000)
				for i := range values {
					values[i] = int64(i) * 1000000000000 // Large values that need int64
				}
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt64).
					SetScalarValues(values, len(values)).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeInt64, ddb.DataType)

				// Test random positions
				testPositions := []int{0, 42, 100, 500, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericScalarFeature(pos)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeInt64(feature)
					require.NoError(t, err)
					assert.Equal(t, int64(pos)*1000000000000, value)
				}
			},
		},
		{
			name: "int64 vector data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				numVectors := 500
				values := make([][]int64, numVectors)
				vectorLengths := make([]uint16, numVectors)

				for i := range values {
					vecLen := 10 + (i % 5) // varying lengths between 10 and 14
					values[i] = make([]int64, vecLen)
					for j := range values[i] {
						values[i][j] = int64(i*10+j) * 1000000000000 // Large values that need int64
					}
					vectorLengths[i] = uint16(vecLen)
				}

				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeInt64Vector).
					SetVectorValues(values, len(values), vectorLengths).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeInt64Vector, ddb.DataType)

				vectorLengths := make([]uint16, 1000)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 5))
				}

				// Test random positions
				testPositions := []int{0, 42, 123, 456, 499}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureInt64ToConcatenatedString(feature)
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 5)
					assert.Equal(t, vecLen, len(values))

					for j, value := range values {
						expected := int64(pos*10+j) * 1000000000000
						actual, _ := strconv.ParseInt(value, 10, 64)
						assert.Equal(t, expected, actual,
							"Mismatch at position %d, index %d", pos, j)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csdb, err := tt.buildFunc()
			require.NoError(t, err)

			data, err := csdb.SerializeForInMemory()
			require.NoError(t, err)
			require.NotNil(t, data)

			tt.validate(t, data)
		})
	}
}

func TestSerializeForInMemoryFP8(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		buildFunc func() (*CacheStorageDataBlock, error)
		validate  func(*testing.T, []byte)
	}{
		{
			name: "fp8 scalar data",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetDataType(types.DataTypeFP8E4M3).
					SetScalarValues([]float32{1.0, 2.0, 4.0}, 3). // Powers of 2 are exactly representable in FP8E4M3
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeFP8E4M3, ddb.DataType)

				// Verify all values
				for i, expected := range []float32{1.0, 2.0, 4.0} {
					feature, err := ddb.GetNumericScalarFeature(i)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeFP8E4M3(feature)
					require.NoError(t, err)
					assert.InDelta(t, expected, value, 0.01) // Tighter delta for exact values
				}
			},
		},
		{
			name: "fp8 scalar large data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				values := make([]float32, 1000)
				for i := range values {
					if i%2 == 0 {
						values[i] = 2.5
					} else {
						values[i] = 0.75
					}
				}
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeFP8E4M3).
					SetScalarValues(values, len(values)).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeFP8E4M3, ddb.DataType)

				// Test random positions
				testPositions := []int{0, 42, 100, 500, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericScalarFeature(pos)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeFP8E4M3(feature)
					require.NoError(t, err)
					if pos%2 == 0 {
						assert.Equal(t, float32(2.5), value)
					} else {
						assert.Equal(t, float32(0.75), value)
					}
				}
			},
		},
		{
			name: "fp8 vector data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				numVectors := 1000
				values := make([][]float32, numVectors)
				vectorLengths := make([]uint16, numVectors)

				for i := range values {
					vecLen := 10 + (i % 5) // varying lengths between 10 and 14
					values[i] = make([]float32, vecLen)
					for j := range values[i] {
						if j%2 == 0 {
							values[i][j] = 2.5
						} else {
							values[i][j] = 0.75
						}
					}
					vectorLengths[i] = uint16(vecLen)
				}

				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeFP8E4M3Vector).
					SetVectorValues(values, len(values), vectorLengths).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeFP8E4M3Vector, ddb.DataType)

				vectorLengths := make([]uint16, 1000)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 5))
				}

				// Test random positions
				testPositions := []int{0, 42, 123, 456, 789, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureFp8E4M3ToConcatenatedString(feature)
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 5)
					assert.Equal(t, vecLen, len(values))

					for j, valStr := range values {
						var floatVal float32
						fmt.Sscanf(valStr, "%f", &floatVal)
						if j%2 == 0 {
							expected := float32(2.5)
							assert.InDelta(t, expected, floatVal, 0.001)
						} else {
							expected := float32(0.75)
							assert.InDelta(t, expected, floatVal, 0.001)
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csdb, err := tt.buildFunc()
			require.NoError(t, err)

			data, err := csdb.SerializeForInMemory()
			require.NoError(t, err)
			require.NotNil(t, data)

			tt.validate(t, data)
		})
	}
}

func TestSerializeForInMemoryFP32(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		buildFunc func() (*CacheStorageDataBlock, error)
		validate  func(*testing.T, []byte)
	}{
		{
			name: "fp32 scalar data",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetDataType(types.DataTypeFP32).
					SetScalarValues([]float32{1.234, 2.345, 3.456}, 3).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeFP32, ddb.DataType)

				// Verify all values
				for i, expected := range []float32{1.234, 2.345, 3.456} {
					feature, err := ddb.GetNumericScalarFeature(i)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeFloat32(feature)
					require.NoError(t, err)
					assert.InDelta(t, expected, value, 0.0001)
				}
			},
		},
		{
			name: "fp32 scalar large data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				values := make([]float32, 1000)
				for i := range values {
					values[i] = float32(i) * 1.234 // Using non-trivial floating point values
				}
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeFP32).
					SetScalarValues(values, len(values)).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeFP32, ddb.DataType)

				// Test random positions
				testPositions := []int{0, 42, 100, 500, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericScalarFeature(pos)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeFloat32(feature)
					require.NoError(t, err)
					assert.InDelta(t, float32(pos)*1.234, value, 0.0001)
				}
			},
		},
		{
			name: "fp32 vector data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				numVectors := 1000
				values := make([][]float32, numVectors)
				vectorLengths := make([]uint16, numVectors)

				for i := range values {
					vecLen := 10 + (i % 5) // varying lengths between 10 and 14
					values[i] = make([]float32, vecLen)
					for j := range values[i] {
						values[i][j] = float32(i*10+j) * 1.234 // Non-trivial floating point values
					}
					vectorLengths[i] = uint16(vecLen)
				}

				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeFP32Vector).
					SetVectorValues(values, len(values), vectorLengths).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeFP32Vector, ddb.DataType)

				vectorLengths := make([]uint16, 1000)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 5))
				}

				// Test random positions
				testPositions := []int{0, 42, 123, 456, 789, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureFp32ToConcatenatedString(feature)
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 5)
					assert.Equal(t, vecLen, len(values))

					for j, valStr := range values {
						var floatVal float32
						fmt.Sscanf(valStr, "%f", &floatVal)
						expected := float32(pos*10+j) * 1.234
						assert.InDelta(t, expected, floatVal, 0.0001,
							"Mismatch at position %d, index %d", pos, j)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csdb, err := tt.buildFunc()
			require.NoError(t, err)

			data, err := csdb.SerializeForInMemory()
			require.NoError(t, err)
			require.NotNil(t, data)

			tt.validate(t, data)
		})
	}
}

func TestSerializeForInMemoryFP64(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		buildFunc func() (*CacheStorageDataBlock, error)
		validate  func(*testing.T, []byte)
	}{
		{
			name: "fp64 scalar data",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetDataType(types.DataTypeFP64).
					SetScalarValues([]float64{1.23456789, 2.34567890, 3.45678901}, 3).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeFP64, ddb.DataType)

				// Verify all values
				for i, expected := range []float64{1.23456789, 2.34567890, 3.45678901} {
					feature, err := ddb.GetNumericScalarFeature(i)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeFloat64(feature)
					require.NoError(t, err)
					assert.InDelta(t, expected, value, 0.00000001)
				}
			},
		},
		{
			name: "fp64 scalar large data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				values := make([]float64, 1000)
				for i := range values {
					values[i] = float64(i) * 1.23456789 // Using high-precision values
				}
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeFP64).
					SetScalarValues(values, len(values)).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeFP64, ddb.DataType)

				// Test random positions
				testPositions := []int{0, 42, 100, 500, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericScalarFeature(pos)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeFloat64(feature)
					require.NoError(t, err)
					assert.InDelta(t, float64(pos)*1.23456789, value, 0.00000001)
				}
			},
		},
		{
			name: "fp64 vector data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				numVectors := 500
				values := make([][]float64, numVectors)
				vectorLengths := make([]uint16, numVectors)

				for i := range values {
					vecLen := 10 + (i % 5) // varying lengths between 10 and 14
					values[i] = make([]float64, vecLen)
					for j := range values[i] {
						values[i][j] = float64(i*10+j) * 1.23456789 // High-precision values
					}
					vectorLengths[i] = uint16(vecLen)
				}

				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeFP64Vector).
					SetVectorValues(values, len(values), vectorLengths).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeFP64Vector, ddb.DataType)

				vectorLengths := make([]uint16, 500)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 5))
				}

				// Test random positions
				testPositions := []int{0, 42, 123, 399}
				for _, pos := range testPositions {
					feature, err := ddb.GetNumericVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureFp64ToConcatenatedString(feature)
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 5)
					assert.Equal(t, vecLen, len(values))

					for j, value := range values {
						expected := float64(pos*10+j) * 1.23456789
						floatVal, err := strconv.ParseFloat(value, 64)
						require.NoError(t, err)
						assert.InDelta(t, expected, floatVal, 0.00000001,
							"Mismatch at position %d, index %d", pos, j)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csdb, err := tt.buildFunc()
			require.NoError(t, err)

			data, err := csdb.SerializeForInMemory()
			require.NoError(t, err)
			require.NotNil(t, data)

			tt.validate(t, data)
		})
	}
}

func TestSerializeForInMemoryString(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		buildFunc func() (*CacheStorageDataBlock, error)
		validate  func(*testing.T, []byte)
	}{
		{
			name: "string scalar data",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetDataType(types.DataTypeString).
					SetStringScalarValues([]string{"hello", "world", "test"}, 3, []uint16{10, 10, 10}).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeString, ddb.DataType)

				// Verify all values
				for i, expected := range []string{"hello", "world", "test"} {
					feature, err := ddb.GetStringScalarFeature(i, 3)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeString(feature)
					require.NoError(t, err)
					assert.Equal(t, expected, value)
				}
			},
		},
		{
			name: "string scalar large data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				values := make([]string, 1000)
				stringLengths := make([]uint16, 1000)
				for i := range values {
					values[i] = fmt.Sprintf("string_%d", i)
					stringLengths[i] = uint16(10)
				}
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeString).
					SetStringScalarValues(values, len(values), stringLengths).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeString, ddb.DataType)

				// Test random positions
				testPositions := []int{0, 42, 100, 500, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetStringScalarFeature(pos, 1000)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeString(feature)
					require.NoError(t, err)
					assert.Equal(t, fmt.Sprintf("string_%d", pos), value)
				}
			},
		},
		{
			name: "string vector data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				numVectors := 400
				values := make([][]string, numVectors)
				vectorLengths := make([]uint16, numVectors)
				stringLengths := make([]uint16, numVectors)
				for i := range values {
					vecLen := 10 + (i % 2) // varying lengths between 10 and 14
					values[i] = make([]string, vecLen)
					for j := range values[i] {
						values[i][j] = fmt.Sprintf("vec_%d_str_%d", i, j)
					}
					vectorLengths[i] = uint16(vecLen)
					stringLengths[i] = uint16(25)
				}

				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeStringVector).
					SetStringVectorValues(values, len(values), stringLengths, vectorLengths).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeStringVector, ddb.DataType)

				vectorLengths := make([]uint16, 400)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 2))
				}

				// Test random positions
				testPositions := []int{0, 42, 123, 399}
				for _, pos := range testPositions {
					feature, err := ddb.GetStringVectorFeature(pos, 400, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureStringToConcatenatedString(feature, int(vectorLengths[pos]))
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 2)
					assert.Equal(t, vecLen, len(values))

					for j, value := range values {
						expected := fmt.Sprintf("vec_%d_str_%d", pos, j)
						assert.Equal(t, expected, value,
							"Mismatch at position %d, index %d", pos, j)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csdb, err := tt.buildFunc()
			require.NoError(t, err)

			data, err := csdb.SerializeForInMemory()
			require.NoError(t, err)
			require.NotNil(t, data)

			tt.validate(t, data)
		})
	}
}

func TestSerializeForInMemoryBool(t *testing.T) {
	system.Init()
	tests := []struct {
		name      string
		buildFunc func() (*CacheStorageDataBlock, error)
		validate  func(*testing.T, []byte)
	}{
		{
			name: "bool scalar data",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetDataType(types.DataTypeBool).
					SetScalarValues([]uint8{1, 0, 1}, 3).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeBool, ddb.DataType)

				// Verify all values
				for i, expected := range []bool{true, false, true} {
					feature, err := ddb.GetBoolScalarFeature(i)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeBool(feature)
					require.NoError(t, err)
					assert.Equal(t, expected, value)
				}
			},
		},
		{
			name: "bool scalar large data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				values := make([]uint8, 1000)
				for i := range values {
					values[i] = uint8(i % 2) // Alternating true/false
				}
				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeBool).
					SetScalarValues(values, len(values)).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeBool, ddb.DataType)

				// Test random positions
				testPositions := []int{0, 42, 100, 500, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetBoolScalarFeature(pos)
					require.NoError(t, err)
					value, err := HelperScalarFeatureToTypeBool(feature)
					require.NoError(t, err)
					expected := !(pos%2 == 0)
					assert.Equal(t, expected, value)
				}
			},
		},
		{
			name: "bool vector data with compression",
			buildFunc: func() (*CacheStorageDataBlock, error) {
				numVectors := 1000
				values := make([][]uint8, numVectors)
				vectorLengths := make([]uint16, numVectors)

				for i := range values {
					vecLen := 10 + (i % 5) // varying lengths between 10 and 14
					values[i] = make([]uint8, vecLen)
					for j := range values[i] {
						values[i][j] = uint8((i + j) % 2) // Pattern of alternating true/false
					}
					vectorLengths[i] = uint16(vecLen)
				}

				psdb, err := NewPermStorageDataBlockBuilder().
					SetID(1).
					SetVersion(1).
					SetTTL(3600).
					SetCompressionB(compression.TypeZSTD).
					SetDataType(types.DataTypeBoolVector).
					SetVectorValues(values, len(values), vectorLengths).
					Build()
				if err != nil {
					return nil, err
				}

				serializedPSDB, err := psdb.Serialize()
				if err != nil {
					return nil, err
				}

				ddb, err := DeserializePSDB(serializedPSDB)
				if err != nil {
					return nil, err
				}

				csdb := NewCacheStorageDataBlock(1)
				err = csdb.AddFGIdToDDB(1, ddb)
				return csdb, err
			},
			validate: func(t *testing.T, data []byte) {
				csdb, err := CreateCSDBForInMemory(data)
				require.NoError(t, err)
				fgIds := ds.NewOrderedSet[int]()
				fgIds.Add(1)
				ddbMap, err := csdb.GetDeserializedPSDBForFGIds(fgIds)
				require.NoError(t, err)
				require.NotNil(t, ddbMap)

				ddb := ddbMap[1]
				require.NotNil(t, ddb)
				assert.Equal(t, types.DataTypeBoolVector, ddb.DataType)

				vectorLengths := make([]uint16, 1000)
				for i := range vectorLengths {
					vectorLengths[i] = uint16(10 + (i % 5))
				}

				// Test random positions
				testPositions := []int{0, 42, 123, 456, 789, 999}
				for _, pos := range testPositions {
					feature, err := ddb.GetBoolVectorFeature(pos, vectorLengths)
					require.NoError(t, err)
					result, err := HelperVectorFeatureBoolToConcatenatedString(feature)
					require.NoError(t, err)

					values := strings.Split(result, ":")
					vecLen := 10 + (pos % 5)
					assert.Equal(t, vecLen, len(values))

					for j, valStr := range values {
						expected := !((pos+j)%2 == 0)
						var boolVal bool
						if valStr == "1" {
							boolVal = true
						} else {
							boolVal = false
						}
						assert.Equal(t, expected, boolVal,
							"Mismatch at position %d, index %d", pos, j)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csdb, err := tt.buildFunc()
			require.NoError(t, err)

			data, err := csdb.SerializeForInMemory()
			require.NoError(t, err)
			require.NotNil(t, data)

			tt.validate(t, data)
		})
	}
}
