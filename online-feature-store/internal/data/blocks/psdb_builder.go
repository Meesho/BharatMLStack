package blocks

import (
	"fmt"
	"math"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
)

type PermStorageDataBlockBuilder struct {
	psdb *PermStorageDataBlock
}

func NewPermStorageDataBlockBuilder() *PermStorageDataBlockBuilder {
	return &PermStorageDataBlockBuilder{
		psdb: &PermStorageDataBlock{},
	}
}

func (p *PermStorageDataBlockBuilder) SetID(id uint) *PermStorageDataBlockBuilder {
	p.psdb.layoutVersion = uint8(id)
	return p
}

func (p *PermStorageDataBlockBuilder) SetVersion(version uint32) *PermStorageDataBlockBuilder {
	p.psdb.featureSchemaVersion = uint16(version)
	return p
}

func (p *PermStorageDataBlockBuilder) SetTTL(ttlInSec uint64) *PermStorageDataBlockBuilder {
	if ttlInSec == 0 { // lifetime ttl
		p.psdb.expiryAt = 0
		return p
	}
	p.psdb.expiryAt = uint64(time.Now().Unix()) + ttlInSec
	return p
}

func (p *PermStorageDataBlockBuilder) SetCompressionB(compressionType compression.Type) *PermStorageDataBlockBuilder {
	p.psdb.compressionType = compressionType
	return p
}

func (p *PermStorageDataBlockBuilder) SetDataType(dataType types.DataType) *PermStorageDataBlockBuilder {
	p.psdb.dataType = dataType
	return p
}

func (p *PermStorageDataBlockBuilder) SetStringValue(stringLens []uint16) *PermStorageDataBlockBuilder {
	p.psdb.stringLengths = stringLens
	return p
}

func (p *PermStorageDataBlockBuilder) SetScalarValues(featureValues interface{}, noOfFeatureValues int) *PermStorageDataBlockBuilder {
	p.psdb.Data = featureValues
	p.psdb.noOfFeatures = noOfFeatureValues
	return p
}

func (p *PermStorageDataBlockBuilder) SetVectorValues(featureValues interface{}, noOfFeatureValues int, vectorLengths []uint16) *PermStorageDataBlockBuilder {
	p.psdb.Data = featureValues
	p.psdb.noOfFeatures = noOfFeatureValues
	p.psdb.vectorLengths = vectorLengths
	return p
}

func (p *PermStorageDataBlockBuilder) SetStringScalarValues(featureValues interface{}, noOfFeatureValues int, stringLengths []uint16) *PermStorageDataBlockBuilder {
	p.psdb.Data = featureValues
	p.psdb.noOfFeatures = noOfFeatureValues
	p.psdb.stringLengths = stringLengths
	return p
}

func (p *PermStorageDataBlockBuilder) SetStringVectorValues(featureValues interface{}, noOfFeatureValues int, stringLengths []uint16, vectorLengths []uint16) *PermStorageDataBlockBuilder {
	p.psdb.Data = featureValues
	p.psdb.noOfFeatures = noOfFeatureValues
	p.psdb.stringLengths = stringLengths
	p.psdb.vectorLengths = vectorLengths
	return p
}

func (p *PermStorageDataBlockBuilder) Build() (*PermStorageDataBlock, error) {
	p.psdb.Builder = p
	if p.psdb.buf == nil {
		p.psdb.buf = make([]byte, 24)
	} else {
		p.psdb.buf = p.psdb.buf[:PSDBLayout1LengthBytes]
	}
	var err error
	switch p.psdb.dataType {
	case types.DataTypeBool:
		p.psdb.originalDataLen, err = scalarBoolDataLength(p.psdb.noOfFeatures)
	case types.DataTypeBoolVector:
		p.psdb.originalDataLen, err = vectorBoolDataLength(p.psdb.noOfFeatures, p.psdb.vectorLengths)
	case types.DataTypeString:
		p.psdb.originalDataLen, err = scalarStringDataLength(p.psdb.stringLengths, p.psdb.noOfFeatures)
	case types.DataTypeStringVector:
		p.psdb.originalDataLen, err = vectorStringDataLength(p.psdb.stringLengths, p.psdb.noOfFeatures, p.psdb.vectorLengths)
	default:
		{
			if p.psdb.dataType.IsVector() {
				p.psdb.originalDataLen, err = vectorNumericDataLength(p.psdb.dataType, p.psdb.noOfFeatures, p.psdb.vectorLengths)
			} else {
				p.psdb.originalDataLen, err = scalarNumericDataLength(p.psdb.dataType, p.psdb.noOfFeatures)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if p.psdb.originalData == nil {
		p.psdb.originalData = make([]byte, p.psdb.originalDataLen)
	} else if len(p.psdb.originalData) < p.psdb.originalDataLen {
		p.psdb.originalData = append(p.psdb.originalData, make([]byte, p.psdb.originalDataLen-len(p.psdb.originalData))...)
	} else {
		p.psdb.originalData = p.psdb.originalData[:p.psdb.originalDataLen]
	}
	p.psdb.compressedDataLen = len(p.psdb.compressedData)
	if p.psdb.compressedData == nil {
		p.psdb.compressedData = make([]byte, 0, p.psdb.originalDataLen)
		p.psdb.compressedData = p.psdb.compressedData[:0]
		p.psdb.compressedDataLen = 0
	} else if p.psdb.compressedDataLen < p.psdb.originalDataLen {
		p.psdb.compressedData = append(p.psdb.compressedData, make([]byte, p.psdb.originalDataLen-p.psdb.compressedDataLen)...)
		p.psdb.compressedData = p.psdb.compressedData[:0]
		p.psdb.compressedDataLen = 0
	} else {
		p.psdb.compressedData = p.psdb.compressedData[:0]
		p.psdb.compressedDataLen = 0
	}

	if p.psdb.buf == nil {
		p.psdb.buf = make([]byte, PSDBLayout1LengthBytes)
	} else {
		p.psdb.buf = p.psdb.buf[:PSDBLayout1LengthBytes]
	}

	return p.psdb, nil
}

func scalarNumericDataLength(dataType types.DataType, noOfFeature int) (int, error) {
	return noOfFeature * dataType.Size(), nil
}

func vectorNumericDataLength(dataType types.DataType, noOfFeature int, vectorLengths []uint16) (int, error) {
	if noOfFeature != len(vectorLengths) {
		return 0, fmt.Errorf("no of feature and vector lengths mismatch")
	}
	totalLen := 0
	for _, vectorLen := range vectorLengths {
		totalLen += int(vectorLen) * dataType.Size()
	}
	return totalLen, nil
}

func scalarStringDataLength(stringLengths []uint16, noOfFeature int) (int, error) {
	if noOfFeature != len(stringLengths) {
		return 0, fmt.Errorf("no of feature and string lengths mismatch")
	}
	totalLen := 0
	for _, stringLen := range stringLengths {
		totalLen += int(stringLen)
	}
	totalLen += 2 * noOfFeature
	return totalLen, nil
}

func vectorStringDataLength(stringLengths []uint16, noOfFeature int, vectorLengths []uint16) (int, error) {
	if noOfFeature != len(vectorLengths) {
		return 0, fmt.Errorf("no of feature and vector lengths mismatch")
	}
	if noOfFeature != len(stringLengths) {
		return 0, fmt.Errorf("no of feature and string lengths mismatch")
	}
	totalLen := 0
	strCount := 0
	for i, vectorLen := range vectorLengths {
		totalLen += int(vectorLen) * int(stringLengths[i])
		strCount += int(vectorLen)
	}
	totalLen += 2 * strCount
	return totalLen, nil
}

func scalarBoolDataLength(noOfFeature int) (int, error) {
	totalLen := int(math.Ceil(float64(noOfFeature) / 8))
	return totalLen, nil
}

func vectorBoolDataLength(noOfFeature int, vectorLengths []uint16) (int, error) {
	if noOfFeature != len(vectorLengths) {
		return 0, fmt.Errorf("no of feature and vector lengths mismatch")
	}
	totalLen := 0
	for _, vectorLen := range vectorLengths {
		totalLen += int(vectorLen)
	}
	totalLen = int(math.Ceil(float64(totalLen) / 8))
	return totalLen, nil
}
