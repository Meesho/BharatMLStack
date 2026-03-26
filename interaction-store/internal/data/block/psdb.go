package blocks

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/compression"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/enum"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/utils"
)

// LAYOUT : 4 bits layout version, 3 bits compression, 9 bits data length (total header 2 bytes)
type PermanentStorageDataBlock struct {
	LayoutVersion     uint8
	CompressionType   compression.Type
	DataLength        uint16 // total number of elements in the data
	OriginalData      []byte
	OriginalDataLen   int
	CompressedData    []byte
	CompressedDataLen int
	Data              any
	InteractionType   enum.InteractionType
	Buf               []byte
	Builder           *PermanentStorageDataBlockBuilder
}

func (p *PermanentStorageDataBlock) Clear() {
	p.LayoutVersion = 0
	p.CompressionType = compression.TypeNone
	p.DataLength = 0
	if len(p.OriginalData) > 0 {
		p.OriginalData = p.OriginalData[:0]
	}
	p.OriginalDataLen = 0
	if len(p.CompressedData) > 0 {
		p.CompressedData = p.CompressedData[:0]
	}
	p.CompressedDataLen = 0
	p.Data = nil
	if len(p.Buf) > PSDBLayout1HeaderLength {
		p.Buf = p.Buf[:PSDBLayout1HeaderLength]
	}
}

func (p *PermanentStorageDataBlock) Serialize() ([]byte, error) {
	switch p.LayoutVersion {
	case 1:
		return p.serializeLayout1()
	default:
		return nil, fmt.Errorf("unsupported Layout Version: %d", p.LayoutVersion)
	}
}

func (p *PermanentStorageDataBlock) serializeLayout1() ([]byte, error) {
	if p.Data == nil {
		return nil, fmt.Errorf("data is nil")
	}
	switch p.InteractionType {
	case enum.InteractionTypeClick:
		err := p.serializeClickEvents()
		if err != nil {
			return nil, err
		}
	case enum.InteractionTypeOrder:
		err := p.serializeOrderEvents()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported interaction type: %s", p.InteractionType)
	}
	if err := setHeader(p); err != nil {
		return nil, err
	}
	return compressAndAppendData(p)
}

func (p *PermanentStorageDataBlock) serializeClickEvents() error {
	clickEvents, ok := p.Data.([]model.ClickEvent)
	if !ok {
		return fmt.Errorf("unexpected data type for click events: got %T, want []model.ClickEvent", p.Data)
	}
	p.DataLength = uint16(len(clickEvents))
	catalogIds := make([]int32, p.DataLength)
	productIds := make([]int32, p.DataLength)
	timestamps := make([]int64, p.DataLength)
	metadataList := make([]string, p.DataLength)
	for i, event := range clickEvents {
		catalogIds[i] = event.ClickEventData.Payload.CatalogId
		productIds[i] = event.ClickEventData.Payload.ProductId
		timestamps[i] = event.ClickEventData.Payload.ClickedAt
		metadataList[i] = event.ClickEventData.Payload.Metadata
	}
	if err := serializeInt32Vector(catalogIds, p); err != nil {
		return err
	}
	if err := serializeInt32Vector(productIds, p); err != nil {
		return err
	}
	if err := serializeInt64Vector(timestamps, p); err != nil {
		return err
	}
	if err := serializeStringVector(metadataList, p); err != nil {
		return err
	}
	return nil
}

func (p *PermanentStorageDataBlock) serializeOrderEvents() error {
	orderEvents, ok := p.Data.([]model.FlattenedOrderEvent)
	if !ok {
		return fmt.Errorf("unexpected data type for order events: got %T, want []model.FlattenedOrderEvent", p.Data)
	}
	p.DataLength = uint16(len(orderEvents))
	catalogIds := make([]int32, p.DataLength)
	productIds := make([]int32, p.DataLength)
	timestamps := make([]int64, p.DataLength)
	subOrderNums := make([]string, p.DataLength)
	metadataList := make([]string, p.DataLength)
	for i, event := range orderEvents {
		catalogIds[i] = event.CatalogID
		productIds[i] = event.ProductID
		timestamps[i] = event.OrderedAt
		subOrderNums[i] = event.SubOrderNum
		metadataList[i] = event.Metadata
	}
	if err := serializeInt32Vector(catalogIds, p); err != nil {
		return err
	}
	if err := serializeInt32Vector(productIds, p); err != nil {
		return err
	}
	if err := serializeInt64Vector(timestamps, p); err != nil {
		return err
	}
	if err := serializeStringVector(subOrderNums, p); err != nil {
		return err
	}
	if err := serializeStringVector(metadataList, p); err != nil {
		return err
	}
	return nil
}

func setHeader(p *PermanentStorageDataBlock) error {
	if p == nil {
		return fmt.Errorf("permanent storage data block is nil")
	}
	if len(p.Buf) < PSDBLayout1HeaderLength {
		return fmt.Errorf("buffer too small: required=%d, actual=%d", PSDBLayout1HeaderLength, len(p.Buf))
	}
	version := p.LayoutVersion & 0x0F
	compression := (uint8(p.CompressionType) & 0x07) << 4
	lengthMSB := (uint8(p.DataLength>>8) & 0x01) << 7
	p.Buf[0] = version | compression | lengthMSB
	p.Buf[1] = uint8(p.DataLength & 0xFF)
	return nil
}

func setCompressionTypeInHeader(p *PermanentStorageDataBlock) error {
	if len(p.Buf) < PSDBLayout1HeaderLength {
		return fmt.Errorf("buffer too small: required=%d, actual=%d", PSDBLayout1HeaderLength, len(p.Buf))
	}
	// Clear compression bits (4-6) and set new compression type
	p.Buf[0] = (p.Buf[0] & 0x8F) | ((uint8(p.CompressionType) & 0x07) << 4)
	return nil
}

func compressAndAppendData(p *PermanentStorageDataBlock) ([]byte, error) {
	// If no compression requested, append original data directly
	if p.CompressionType == compression.TypeNone {
		p.Buf = append(p.Buf, p.OriginalData...)
		return p.Buf, nil
	}

	// Get encoder for the specified compression type
	enc, err := compression.GetEncoder(p.CompressionType)
	if err != nil {
		return nil, err
	}

	// Attempt compression and check if it's beneficial
	return encodeData(p, enc)
}

func encodeData(p *PermanentStorageDataBlock, enc compression.Encoder) ([]byte, error) {
	p.OriginalDataLen = len(p.OriginalData)
	enc.Encode(p.OriginalData, &p.CompressedData)
	p.CompressedDataLen = len(p.CompressedData)

	// If compression doesn't reduce size, fall back to uncompressed data
	if p.CompressedDataLen >= p.OriginalDataLen {
		// Reset compression type to None
		p.CompressionType = compression.TypeNone
		// Update header with new compression type
		if err := setCompressionTypeInHeader(p); err != nil {
			return nil, err
		}
		// Use original data instead of compressed data
		p.Buf = append(p.Buf, p.OriginalData...)
		return p.Buf, nil
	}

	p.Buf = append(p.Buf, p.CompressedData...)
	return p.Buf, nil
}

// func serializeFP32Vector(values []float32, p *PermanentStorageDataBlock) error {
// 	if len(values) == 0 {
// 		return fmt.Errorf("fp32 vector data is empty")
// 	}
// 	if len(values) != int(p.DataLength) {
// 		return fmt.Errorf("mismatch in number of elements (%d) and defined data length (%d)", len(values), p.DataLength)
// 	}
// 	unitSize := enum.DataTypeFP32Vector.Size()
// 	totalSize := len(values) * unitSize
// 	startIdx := len(p.OriginalData)
// 	p.OriginalData = append(p.OriginalData, make([]byte, totalSize)...)
// 	for _, v := range values {
// 		utils.ByteOrder.PutFloat32(p.OriginalData[startIdx:startIdx+unitSize], v)
// 		startIdx += unitSize
// 	}
// 	return nil
// }

func serializeInt32Vector(values []int32, p *PermanentStorageDataBlock) error {
	if len(values) == 0 {
		return fmt.Errorf("int32 vector data is empty")
	}
	if len(values) != int(p.DataLength) {
		return fmt.Errorf("mismatch in number of elements (%d) and defined data length (%d)", len(values), p.DataLength)
	}
	unitSize := enum.DataTypeInt32Vector.Size()
	totalSize := len(values) * unitSize
	startIdx := len(p.OriginalData)
	p.OriginalData = append(p.OriginalData, make([]byte, totalSize)...)
	for _, v := range values {
		utils.ByteOrder.PutInt32(p.OriginalData[startIdx:startIdx+unitSize], v)
		startIdx += unitSize
	}
	return nil
}

// func serializeFP64Vector(values []float64, p *PermanentStorageDataBlock) error {
// 	if len(values) == 0 {
// 		return fmt.Errorf("fp64 vector data is empty")
// 	}
// 	if len(values) != int(p.DataLength) {
// 		return fmt.Errorf("mismatch in number of elements (%d) and defined data length (%d)", len(values), p.DataLength)
// 	}
// 	unitSize := enum.DataTypeFP64Vector.Size()
// 	totalSize := len(values) * unitSize
// 	startIdx := len(p.OriginalData)
// 	p.OriginalData = append(p.OriginalData, make([]byte, totalSize)...)
// 	for _, v := range values {
// 		utils.ByteOrder.PutFloat64(p.OriginalData[startIdx:startIdx+unitSize], v)
// 		startIdx += unitSize
// 	}
// 	return nil
// }

func serializeInt64Vector(values []int64, p *PermanentStorageDataBlock) error {
	if len(values) == 0 {
		return fmt.Errorf("int64 vector data is empty")
	}
	if len(values) != int(p.DataLength) {
		return fmt.Errorf("mismatch in number of elements (%d) and defined data length (%d)", len(values), p.DataLength)
	}
	unitSize := enum.DataTypeInt64Vector.Size()
	totalSize := len(values) * unitSize
	startIdx := len(p.OriginalData)
	p.OriginalData = append(p.OriginalData, make([]byte, totalSize)...)
	for _, v := range values {
		utils.ByteOrder.PutInt64(p.OriginalData[startIdx:startIdx+unitSize], v)
		startIdx += unitSize
	}
	return nil
}

// serializeStringVector serializes string vector data into a byte slice, using pascal string encoding.
// Format: [len1][len2]...[lenN][str1][str2]...[strN]
func serializeStringVector(values []string, p *PermanentStorageDataBlock) error {
	if len(values) != int(p.DataLength) {
		return fmt.Errorf("mismatch in number of elements (%d) and defined data length (%d)",
			len(values), p.DataLength)
	}
	for _, v := range values {
		strLen := uint16(len(v))
		lenBytes := make([]byte, 2)
		utils.ByteOrder.PutUint16(lenBytes, strLen)
		p.OriginalData = append(p.OriginalData, lenBytes...)
	}
	for _, v := range values {
		p.OriginalData = append(p.OriginalData, []byte(v)...)
	}
	return nil
}

// func serializeBoolVector(values []bool, p *PermanentStorageDataBlock) error {
// 	if len(values) == 0 {
// 		return fmt.Errorf("bool vector data is empty")
// 	}
// 	if len(values) != int(p.DataLength) {
// 		return fmt.Errorf("mismatch in number of elements (%d) and defined data length (%d)",
// 			len(values), p.DataLength)
// 	}
// 	unitSize := enum.DataTypeBoolVector.Size()
// 	totalSize := len(values) * unitSize
// 	startIdx := len(p.OriginalData)
// 	p.OriginalData = append(p.OriginalData, make([]byte, totalSize)...)
// 	for _, v := range values {
// 		if v {
// 			p.OriginalData[startIdx] = 1
// 		} else {
// 			p.OriginalData[startIdx] = 0
// 		}
// 		startIdx += unitSize
// 	}
// 	return nil
// }
