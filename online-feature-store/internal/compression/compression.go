package compression

import "fmt"

type Type uint8

const (
	TypeNone Type = iota
	TypeZSTD
)

type Encoder interface {
	Encode(data []byte) []byte
	EncodeV2(data []byte, outputBuffer *[]byte)
	EncoderType() Type
}

type Decoder interface {
	Decode(cdata []byte) ([]byte, error)
	DecoderType() Type
}

type NoOpEncoder struct{}

func (e *NoOpEncoder) Encode(data []byte) []byte {
	return data
}

func (e *NoOpEncoder) EncodeV2(data []byte, outputBuffer *[]byte) {
	*outputBuffer = make([]byte, len(data))
	copy(*outputBuffer, data)
}

func (e *NoOpEncoder) EncoderType() Type {
	return TypeNone
}

type NoOpDecoder struct{}

func (d *NoOpDecoder) Decode(cdata []byte) ([]byte, error) {
	return cdata, nil
}

func (d *NoOpDecoder) DecoderType() Type {
	return TypeNone
}

func GetEncoder(compressionType Type) (Encoder, error) {
	switch compressionType {
	case TypeZSTD:
		return NewZStdEncoder(), nil
	case TypeNone:
		return &NoOpEncoder{}, nil
	default:
		return nil, fmt.Errorf("unsupported compression type: %d", compressionType)
	}
}

func GetDecoder(compressionType Type) (Decoder, error) {
	switch compressionType {
	case TypeZSTD:
		return NewZStdDecoder(), nil
	case TypeNone:
		return &NoOpDecoder{}, nil
	default:
		return nil, fmt.Errorf("unsupported compression type: %d", compressionType)
	}
}
