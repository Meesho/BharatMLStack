package blocks

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/compression"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/enum"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/utils"
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
	if len(data) == 0 {
		return nil, fmt.Errorf("data is missing")
	}
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
	// Data length MSB is in bit 7 of the first byte, LSB is in the second byte
	lengthMSB := uint16((data[0] & 0x80) >> 7)
	lengthLSB := uint16(data[1] & 0xFF)
	return (lengthMSB << 8) | lengthLSB
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

func (dPsdb *DeserializedPSDB) RetrieveEventData() (any, error) {
	switch dPsdb.InteractionType {
	case enum.InteractionTypeClick:
		return dPsdb.retrieveClickEventData()
	case enum.InteractionTypeOrder:
		return dPsdb.retrieveOrderEventData()
	default:
		return nil, fmt.Errorf("unsupported interaction type: %s", dPsdb.InteractionType)
	}
}

func (dPsdb *DeserializedPSDB) retrieveClickEventData() ([]model.ClickEvent, error) {
	idx := 0
	deltaLength := int(dPsdb.DataLength) * enum.DataTypeInt32Vector.Size()
	catalogIds := utils.ByteOrder.Int32Vector(dPsdb.OriginalData[idx : idx+deltaLength])
	idx += deltaLength
	deltaLength = int(dPsdb.DataLength) * enum.DataTypeInt32Vector.Size()
	productIds := utils.ByteOrder.Int32Vector(dPsdb.OriginalData[idx : idx+deltaLength])
	idx += deltaLength
	deltaLength = int(dPsdb.DataLength) * enum.DataTypeInt64Vector.Size()
	timestamps := utils.ByteOrder.Int64Vector(dPsdb.OriginalData[idx : idx+deltaLength])
	events := make([]model.ClickEvent, dPsdb.DataLength)
	if dPsdb.DataLength != uint16(len(catalogIds)) || dPsdb.DataLength != uint16(len(productIds)) || dPsdb.DataLength != uint16(len(timestamps)) {
		return nil, fmt.Errorf("data length mismatch")
	}
	for i := 0; i < int(dPsdb.DataLength); i++ {
		events[i] = model.ClickEvent{
			ClickEventData: model.ClickEventData{
				Payload: model.ClickEventPayload{
					CatalogId: catalogIds[i],
					ProductId: productIds[i],
					ClickedAt: timestamps[i],
				},
			},
		}
	}
	return events, nil
}

func (dPsdb *DeserializedPSDB) retrieveOrderEventData() ([]model.FlatOrderEvent, error) {
	idx := 0
	deltaLength := int(dPsdb.DataLength) * enum.DataTypeInt32Vector.Size()
	catalogIds := utils.ByteOrder.Int32Vector(dPsdb.OriginalData[idx : idx+deltaLength])
	idx += deltaLength
	deltaLength = int(dPsdb.DataLength) * enum.DataTypeInt32Vector.Size()
	productIds := utils.ByteOrder.Int32Vector(dPsdb.OriginalData[idx : idx+deltaLength])
	idx += deltaLength
	deltaLength = int(dPsdb.DataLength) * enum.DataTypeInt64Vector.Size()
	timestamps := utils.ByteOrder.Int64Vector(dPsdb.OriginalData[idx : idx+deltaLength])
	idx += deltaLength

	subOrderNums, err := deserializeStringVector(dPsdb.OriginalData[idx:], int(dPsdb.DataLength))
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize sub_order_num: %w", err)
	}

	if dPsdb.DataLength != uint16(len(catalogIds)) || dPsdb.DataLength != uint16(len(productIds)) || dPsdb.DataLength != uint16(len(timestamps)) {
		return nil, fmt.Errorf("data length mismatch")
	}

	events := make([]model.FlatOrderEvent, dPsdb.DataLength)
	for i := 0; i < int(dPsdb.DataLength); i++ {
		events[i] = model.FlatOrderEvent{
			CatalogID:   catalogIds[i],
			ProductID:   productIds[i],
			OrderedAt:   timestamps[i],
			SubOrderNum: subOrderNums[i],
		}
	}
	return events, nil
}

// deserializeStringVector reads pascal-encoded strings: [len1][len2]...[lenN][str1][str2]...[strN]
func deserializeStringVector(data []byte, count int) ([]string, error) {
	if count == 0 {
		return []string{}, nil
	}
	lengthsSize := count * 2
	if len(data) < lengthsSize {
		return nil, fmt.Errorf("data too short to contain string lengths")
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
			return nil, fmt.Errorf("data too short to contain string at index %d", i)
		}
		result[i] = string(data[strOffset : strOffset+strLen])
		strOffset += strLen
	}
	return result, nil
}
