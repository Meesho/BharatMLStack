package blocks

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/compression"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/enum"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/utils"
	"github.com/rs/zerolog/log"
)

type DeserializedPSDB struct {
	LayoutVersion   uint8
	CompressionType compression.Type
	DataLength      uint16
	Header          []byte
	OriginalData    []byte
	CompressedData  []byte
	InteractionType enum.InteractionType
}

func DeserializePSDB(data []byte, interactionType enum.InteractionType) (*DeserializedPSDB, error) {
	layoutVersion, err := extractLayoutVersionFromHeader(data)
	if err != nil {
		return nil, err
	}
	var dPsdb *DeserializedPSDB
	switch layoutVersion {
	case 1:
		dPsdb, err = deserializePSDBForLayout1(data)
	default:
		err = fmt.Errorf("unsupported layout version: %d", layoutVersion)
	}
	if err == nil {
		dPsdb.LayoutVersion = layoutVersion
		dPsdb.InteractionType = interactionType
	}
	return dPsdb, err
}

func extractLayoutVersionFromHeader(data []byte) (uint8, error) {
	if len(data) < PSDBLayout1HeaderLength {
		return 0, fmt.Errorf("header is too short to contain layout version")
	}
	// Layout version is stored in bits 0-3 (lower 4 bits) of the first byte
	return data[0] & 0x0F, nil
}

func extractCompressionType(data []byte) compression.Type {
	// Compression type is stored in bits 4-6 (3 bits) of the first byte
	return compression.Type((data[0] & 0x70) >> 4)
}

func extractDataLength(data []byte) uint16 {
	// Data length Most Significant Bit is in bit 7 of the first byte, Least Significant Bit is in the second byte
	lengthMostSignificantBit := uint16((data[0] & 0x80) >> 7)
	lengthLeastSignificantBit := uint16(data[1] & 0xFF)
	return (lengthMostSignificantBit << 8) | lengthLeastSignificantBit
}

func deserializePSDBForLayout1(data []byte) (*DeserializedPSDB, error) {
	if len(data) < PSDBLayout1HeaderLength {
		return nil, fmt.Errorf("data is too short to contain a valid psdb header")
	}
	compressionType := extractCompressionType(data[0:1])
	dataLength := extractDataLength(data[0:2])
	header := data[0:PSDBLayout1HeaderLength]
	var originalData []byte
	var compressedData []byte
	if compressionType == compression.TypeNone {
		originalData = data[PSDBLayout1HeaderLength:]
		compressedData = data[PSDBLayout1HeaderLength:]
	} else {
		dec, err := compression.GetDecoder(compressionType)
		if err != nil {
			return nil, err
		}
		compressedData = data[PSDBLayout1HeaderLength:]
		originalData, err = dec.Decode(compressedData)
		if err != nil {
			return nil, err
		}
	}
	return &DeserializedPSDB{
		CompressionType: compressionType,
		DataLength:      dataLength,
		Header:          header,
		OriginalData:    originalData,
		CompressedData:  compressedData,
	}, nil
}

func (dPsdb *DeserializedPSDB) RetrieveEventData(userId string) (any, error) {
	switch dPsdb.InteractionType {
	case enum.InteractionTypeClick:
		return dPsdb.retrieveClickEventData(userId)
	case enum.InteractionTypeOrder:
		return dPsdb.retrieveOrderEventData(userId)
	default:
		return nil, fmt.Errorf("unsupported interaction type: %s", dPsdb.InteractionType)
	}
}

// extractCommonEventFields extracts catalog IDs, product IDs, and timestamps from the original data
// Returns the extracted vectors, the current index position, and any error encountered
func (dPsdb *DeserializedPSDB) extractCommonEventFields(userId string) (catalogIds, productIds []int32, timestamps []int64, idx int, err error) {
	idx = 0
	int32Size := enum.DataTypeInt32Vector.Size()
	int64Size := enum.DataTypeInt64Vector.Size()

	// Calculate total required bytes for all three vectors:
	// catalogIds (int32) + productIds (int32) + timestamps (int64)
	requiredBytes := int(dPsdb.DataLength) * (int32Size + int32Size + int64Size)
	availableBytes := len(dPsdb.OriginalData)

	if availableBytes < requiredBytes {
		log.Error().Msgf("data corruption detected (psdb): original data too short for declared data length, dataLength=%d, requiredBytes=%d, availableBytes=%d, compressionType=%d, interactionType=%s, userId=%s", dPsdb.DataLength, requiredBytes, availableBytes, uint8(dPsdb.CompressionType), string(dPsdb.InteractionType), userId)

		return nil, nil, nil, 0, fmt.Errorf("insufficient data in psdb: original data too short for declared data length")
	}

	// Extract catalogIds
	deltaLength := int(dPsdb.DataLength) * int32Size
	catalogIds = utils.ByteOrder.Int32Vector(dPsdb.OriginalData[idx : idx+deltaLength])
	idx += deltaLength

	// Extract productIds
	deltaLength = int(dPsdb.DataLength) * int32Size
	productIds = utils.ByteOrder.Int32Vector(dPsdb.OriginalData[idx : idx+deltaLength])
	idx += deltaLength

	// Extract timestamps
	deltaLength = int(dPsdb.DataLength) * int64Size
	timestamps = utils.ByteOrder.Int64Vector(dPsdb.OriginalData[idx : idx+deltaLength])
	idx += deltaLength

	return catalogIds, productIds, timestamps, idx, nil
}

func (dPsdb *DeserializedPSDB) retrieveClickEventData(userId string) ([]model.ClickEvent, error) {
	catalogIds, productIds, timestamps, idx, err := dPsdb.extractCommonEventFields(userId)
	if err != nil {
		return nil, fmt.Errorf("failed to extract common event fields for click events: %w", err)
	}

	metadata, err := deserializeStringVector(dPsdb.OriginalData[idx:], int(dPsdb.DataLength))
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize metadata for click events: %w", err)
	}

	if dPsdb.DataLength != uint16(len(catalogIds)) || dPsdb.DataLength != uint16(len(productIds)) || dPsdb.DataLength != uint16(len(timestamps)) || dPsdb.DataLength != uint16(len(metadata)) {
		return nil, fmt.Errorf("data length mismatch while deserializing click events")
	}
	events := make([]model.ClickEvent, dPsdb.DataLength)
	for i := 0; i < int(dPsdb.DataLength); i++ {
		events[i] = model.ClickEvent{
			ClickEventData: model.ClickEventData{
				Payload: model.ClickEventPayload{
					CatalogId: catalogIds[i],
					ProductId: productIds[i],
					ClickedAt: timestamps[i],
					Metadata:  metadata[i],
				},
			},
		}
	}
	return events, nil
}

func (dPsdb *DeserializedPSDB) retrieveOrderEventData(userId string) ([]model.FlattenedOrderEvent, error) {
	catalogIds, productIds, timestamps, idx, err := dPsdb.extractCommonEventFields(userId)
	if err != nil {
		return nil, fmt.Errorf("failed to extract common event fields for order events: %w", err)
	}

	subOrderNums, bytesConsumed, err := deserializeStringVectorWithOffset(dPsdb.OriginalData[idx:], int(dPsdb.DataLength))
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize sub_order_num: %w", err)
	}
	idx += bytesConsumed

	metadata, err := deserializeStringVector(dPsdb.OriginalData[idx:], int(dPsdb.DataLength))
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
	}

	if dPsdb.DataLength != uint16(len(catalogIds)) || dPsdb.DataLength != uint16(len(productIds)) || dPsdb.DataLength != uint16(len(timestamps)) || dPsdb.DataLength != uint16(len(subOrderNums)) || dPsdb.DataLength != uint16(len(metadata)) {
		return nil, fmt.Errorf("data length mismatch while deserializing order events")
	}

	events := make([]model.FlattenedOrderEvent, dPsdb.DataLength)
	for i := 0; i < int(dPsdb.DataLength); i++ {
		events[i] = model.FlattenedOrderEvent{
			CatalogID:   catalogIds[i],
			ProductID:   productIds[i],
			OrderedAt:   timestamps[i],
			SubOrderNum: subOrderNums[i],
			Metadata:    metadata[i],
		}
	}
	return events, nil
}

// deserializeStringVector reads pascal-encoded strings: [len1][len2]...[lenN][str1][str2]...[strN]
func deserializeStringVector(data []byte, count int) ([]string, error) {
	result, _, err := deserializeStringVectorWithOffset(data, count)
	return result, err
}

// deserializeStringVectorWithOffset reads pascal-encoded strings and returns the bytes consumed
func deserializeStringVectorWithOffset(data []byte, count int) ([]string, int, error) {
	if count == 0 {
		return []string{}, 0, nil
	}
	lengthsSize := count * 2
	if len(data) < lengthsSize {
		return nil, 0, fmt.Errorf("data too short to contain string lengths")
	}

	lengths := make([]uint16, count)
	for i := 0; i < count; i++ {
		lengths[i] = utils.ByteOrder.Uint16(data[i*2 : i*2+2])
	}

	result := make([]string, count)
	strOffset := lengthsSize
	for i := 0; i < count; i++ {
		strLen := int(lengths[i])
		if strOffset+strLen > len(data) {
			return nil, 0, fmt.Errorf("data too short to contain string at index %d", i)
		}
		result[i] = string(data[strOffset : strOffset+strLen])
		strOffset += strLen
	}
	return result, strOffset, nil
}
