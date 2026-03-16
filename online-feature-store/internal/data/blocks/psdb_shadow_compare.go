package blocks

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"github.com/rs/zerolog/log"
)

// ShadowSerializeAsLayout builds a PSDB with the given layout version, serializes it,
// returns the serialized size, then returns the PSDB to the pool.
// Panics are recovered so shadow path never fails the primary write.
func ShadowSerializeAsLayout(
	layoutVersion uint,
	dataType types.DataType,
	featureData interface{},
	featureBitmap []byte,
	compressionType compression.Type,
	ttlInSeconds uint64,
	activeVersion uint32,
	numOfFeatures int,
	stringLengths []uint16,
	vectorLengths []uint16,
) (serializedSize int) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn().Msgf("shadow serialize panic recovered: %v", r)
			serializedSize = -1
		}
	}()

	pool := GetPSDBPool()
	pooled := pool.Get()
	builder := pooled.Builder.
		SetID(layoutVersion).
		SetDataType(dataType).
		SetCompressionB(compressionType).
		SetTTL(ttlInSeconds).
		SetVersion(activeVersion).
		SetBitmap(featureBitmap)

	if layoutVersion == 2 && len(featureBitmap) > 0 {
		builder = builder.SetupBitmapMeta(numOfFeatures)
	}

	var psdb *PermStorageDataBlock
	var err error

	switch dataType.String() {
	case "DataTypeString":
		psdb, err = builder.
			SetStringValue(stringLengths).
			SetScalarValues(featureData, numOfFeatures).
			Build()
	case "DataTypeStringVector":
		psdb, err = builder.
			SetStringValue(stringLengths).
			SetVectorValues(featureData, numOfFeatures, vectorLengths).
			Build()
	default:
		if dataType.IsVector() {
			psdb, err = builder.
				SetVectorValues(featureData, numOfFeatures, vectorLengths).
				Build()
		} else {
			psdb, err = builder.
				SetScalarValues(featureData, numOfFeatures).
				Build()
		}
	}

	if err != nil {
		log.Warn().Err(err).Msg("shadow serialize: failed to build PSDB")
		return -1
	}

	serialized, err := psdb.Serialize()
	if err != nil {
		log.Warn().Err(err).Msg("shadow serialize: failed to serialize PSDB")
		psdb.Clear()
		pool.Put(psdb)
		return -1
	}

	size := len(serialized)
	psdb.Clear()
	pool.Put(psdb)
	return size
}
