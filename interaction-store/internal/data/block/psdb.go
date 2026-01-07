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
	DataLength        uint16
	OriginalData      []byte
	OriginalDataLen   int
	CompressedData    []byte
	CompressedDataLen int
	Data              any
	InteractionType   enum.InteractionType
	Buf               []byte
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
	if len(p.Buf) > PSDBLayout1HeaderLength {
		p.Buf = p.Buf[:PSDBLayout1HeaderLength]
	}
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
	if err := setHeader(p); err != nil {
		return nil, err
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
	enc, err := compression.GetEncoder(p.CompressionType)
	if err != nil {
		return nil, err
	}
	return encodeData(p, enc)
}

func (p *PermanentStorageDataBlock) serializeClickEvents() error {
	clickEvents := p.Data.([]model.ClickEvent)
	p.DataLength = uint16(len(clickEvents))
	catalogIds := make([]int32, p.DataLength)
	productIds := make([]int32, p.DataLength)
	timestamps := make([]int64, p.DataLength)
	for i, event := range clickEvents {
		catalogIds[i] = event.ClickEventData.Payload.CatalogId
		productIds[i] = event.ClickEventData.Payload.ProductId
		timestamps[i] = event.ClickEventData.Payload.ClickedAt
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
	return nil
}

func (p *PermanentStorageDataBlock) serializeOrderEvents() error {
	orderEvents := p.Data.([]model.FlatOrderEvent)
	p.DataLength = uint16(len(orderEvents))
	catalogIds := make([]int32, p.DataLength)
	productIds := make([]int32, p.DataLength)
	timestamps := make([]int64, p.DataLength)
	subOrderNums := make([]string, p.DataLength)
	for i, event := range orderEvents {
		catalogIds[i] = event.CatalogID
		productIds[i] = event.ProductID
		timestamps[i] = event.OrderedAt
		subOrderNums[i] = event.SubOrderNum
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
	compression := (uint8(compression.TypeZSTD) & 0x07) << 4
	lengthMSB := (uint8(p.DataLength>>8) & 0x01) << 7
	p.Buf[0] = version | compression | lengthMSB
	p.Buf[1] = uint8(p.DataLength & 0xFF)
	return nil
}

func encodeData(p *PermanentStorageDataBlock, enc compression.Encoder) ([]byte, error) {
	p.OriginalDataLen = len(p.OriginalData)
	enc.Encode(p.OriginalData, &p.CompressedData)
	p.CompressedDataLen = len(p.CompressedData)
	p.Buf = append(p.Buf, p.CompressedData...)
	return p.Buf, nil
}

// func serializeFP32Vector(values []float32, p *PermanentStorageDataBlock) error {
// 	unitSize := enum.DataTypeFP32Vector.Size()
// 	if len(values) == 0 {
// 		return fmt.Errorf("fp32 vector data is empty")
// 	}
// 	if len(values) != int(p.DataLength) {
// 		return fmt.Errorf("mismatch in number of elements (%d) and defined data length (%d)", len(values), p.DataLength)
// 	}
// 	idx := len(p.OriginalData)
// 	for _, v := range values {
// 		utils.ByteOrder.PutFloat32(p.OriginalData[idx:idx+unitSize], v)
// 		idx += unitSize
// 	}
// 	return nil
// }

func serializeInt32Vector(values []int32, p *PermanentStorageDataBlock) error {
	unitSize := enum.DataTypeInt32Vector.Size()
	if len(values) == 0 {
		return fmt.Errorf("int32 vector data is empty")
	}
	if len(values) != int(p.DataLength) {
		return fmt.Errorf("mismatch in number of elements (%d) and defined data length (%d)", len(values), p.DataLength)
	}
	idx := len(p.OriginalData)
	for _, v := range values {
		utils.ByteOrder.PutInt32(p.OriginalData[idx:idx+unitSize], v)
		idx += unitSize
	}
	return nil
}

// func serializeFP64Vector(values []float64, p *PermanentStorageDataBlock) error {
// 	unitSize := enum.DataTypeFP64Vector.Size()
// 	if len(values) == 0 {
// 		return fmt.Errorf("fp64 vector data is empty")
// 	}
// 	if len(values) != int(p.DataLength) {
// 		return fmt.Errorf("mismatch in number of elements (%d) and defined data length (%d)", len(values), p.DataLength)
// 	}
// 	idx := len(p.OriginalData)
// 	for _, v := range values {
// 		utils.ByteOrder.PutFloat64(p.OriginalData[idx:idx+unitSize], v)
// 		idx += unitSize
// 	}
// 	return nil
// }

func serializeInt64Vector(values []int64, p *PermanentStorageDataBlock) error {
	unitSize := enum.DataTypeInt64Vector.Size()
	if len(values) == 0 {
		return fmt.Errorf("int64 vector data is empty")
	}
	if len(values) != int(p.DataLength) {
		return fmt.Errorf("mismatch in number of elements (%d) and defined data length (%d)", len(values), p.DataLength)
	}
	idx := len(p.OriginalData)
	for _, v := range values {
		utils.ByteOrder.PutInt64(p.OriginalData[idx:idx+unitSize], v)
		idx += unitSize
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
// 	idx := len(p.OriginalData)
// 	for _, v := range values {
// 		if v {
// 			p.OriginalData[idx] = 1
// 		} else {
// 			p.OriginalData[idx] = 0
// 		}
// 		idx++
// 	}
// 	return nil
// }
