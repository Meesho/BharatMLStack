package blocks

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
)

// PSDBBlock is the interface for deserialized Permanent Storage Data Blocks.
// Layout V1 (DeserializedPSDB) and Layout V2 (DeserializedPSDBLayout2) each
// implement this interface with their own feature extraction logic.
type PSDBBlock interface {
	GetLayoutVersion() uint8
	GetOriginalData() []byte
	GetCompressedData() []byte
	GetHeader() []byte
	GetDataType() types.DataType
	GetCompressionType() compression.Type
	GetFeatureSchemaVersion() uint16
	GetExpiryAt() uint64
	IsNegativeCache() bool
	IsExpired() bool
	SetNegativeCache(val bool)
	CopyBlock() PSDBBlock

	// Feature extraction — V1 and V2 each implement their own logic
	GetNumericScalarFeature(pos int, numFeatures int, defaultValue []byte) ([]byte, error)
	GetNumericVectorFeature(pos int, vectorLengths []uint16, defaultValue []byte) ([]byte, error)
	GetStringScalarFeature(pos int, noOfFeatures int, defaultValue []byte) ([]byte, error)
	GetStringVectorFeature(pos int, noOfFeatures int, vectorLengths []uint16, defaultValue []byte) ([]byte, error)
	GetBoolScalarFeature(pos int) ([]byte, error)
	GetBoolVectorFeature(pos int, vectorLengths []uint16, defaultValue []byte) ([]byte, error)
}
