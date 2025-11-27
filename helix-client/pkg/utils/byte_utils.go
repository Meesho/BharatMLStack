package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/system"
	"github.com/rs/zerolog/log"
)

func BytesToInt(in []byte, ii *int) error {
	if len(in) != 8 {
		return errors.New("int demands 8-byte length")
	}
	var i int64
	reader := bytes.NewReader(in)
	err := binary.Read(reader, system.ByteOrder, &i)
	if err != nil {
		return err
	}
	*ii = int(i)
	return nil
}

func BytesToInt64(in []byte, i *int64) error {
	if len(in) != 8 {
		return errors.New("int64 demands 8-byte length")
	}
	reader := bytes.NewReader(in)
	err := binary.Read(reader, system.ByteOrder, i)
	if err != nil {
		return err
	}
	return nil
}

func BytesToUint64(in []byte, i *uint64) error {
	if len(in) != 8 {
		return errors.New("uint64 demands 8-byte length")
	}
	reader := bytes.NewReader(in)
	err := binary.Read(reader, system.ByteOrder, i)
	if err != nil {
		return err
	}
	return nil
}

func BytesToInt32(in []byte, i *int32) error {
	if len(in) != 4 {
		return errors.New("int32 demands 4-byte length")
	}
	reader := bytes.NewReader(in)
	err := binary.Read(reader, system.ByteOrder, i)
	if err != nil {
		return err
	}
	return nil
}

func BytesToUint32(in []byte, i *uint32) error {
	if len(in) != 4 {
		return errors.New("uint32 demands 4-byte length")
	}
	reader := bytes.NewReader(in)
	err := binary.Read(reader, system.ByteOrder, i)
	if err != nil {
		return err
	}
	return nil
}

func BytesToInt16(in []byte, i *int16) error {
	if len(in) != 2 {
		return errors.New("int16 demands 2-byte length")
	}
	reader := bytes.NewReader(in)
	err := binary.Read(reader, system.ByteOrder, i)
	if err != nil {
		return err
	}
	return nil
}

func BytesToUint16(in []byte, i *uint16) error {
	if len(in) != 2 {
		return errors.New("uint16 demands 2-byte length")
	}
	reader := bytes.NewReader(in)
	err := binary.Read(reader, system.ByteOrder, i)
	if err != nil {
		return err
	}
	return nil
}

func BytesToInt8(in []byte, i *int8) error {
	if len(in) != 1 {
		return errors.New("int8 demands 1-byte length")
	}
	reader := bytes.NewReader(in)
	err := binary.Read(reader, system.ByteOrder, i)
	if err != nil {
		return err
	}
	return nil
}

func BytesToUint8(in []byte, i *uint8) error {
	if len(in) != 1 {
		return errors.New("uint8 demands 1-byte length")
	}
	reader := bytes.NewReader(in)
	err := binary.Read(reader, system.ByteOrder, i)
	if err != nil {
		return err
	}
	return nil
}

func IntToBytes(value interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	switch v := value.(type) {
	case int:
		err := binary.Write(buf, system.ByteOrder, int64(v)) // Convert int to int64
		return buf.Bytes(), err
	case uint:
		err := binary.Write(buf, system.ByteOrder, uint64(v))
		return buf.Bytes(), err
	case int8, int16, int32, int64, uint8, uint16, uint32, uint64:
		err := binary.Write(buf, system.ByteOrder, v)
		return buf.Bytes(), err
	default:
		log.Error().Msgf("unsupported type: %s", reflect.TypeOf(value).Name())
		return nil, errors.New("unsupported type: " + reflect.TypeOf(value).Name())
	}
}
