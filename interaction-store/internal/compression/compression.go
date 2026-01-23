package compression

import "fmt"

type Type uint8

const (
	TypeNone Type = iota
	TypeZSTD
)

type Encoder interface {
	Encode(data []byte, outputBuffer *[]byte)
}

type Decoder interface {
	Decode(compressedData []byte) ([]byte, error)
}

func GetEncoder(compressionType Type) (Encoder, error) {
	switch compressionType {
	case TypeZSTD:
		return NewZStdEncoder()
	default:
		return nil, fmt.Errorf("unsupported compression type: %d", compressionType)
	}
}

func GetDecoder(compressionType Type) (Decoder, error) {
	switch compressionType {
	case TypeZSTD:
		return NewZStdDecoder()
	default:
		return nil, fmt.Errorf("unsupported compression type: %d", compressionType)
	}
}
