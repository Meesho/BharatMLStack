package enum

import (
	"errors"
)

type DataType uint8

const (
	DataTypeUnknown DataType = iota
	DataTypeFP32Vector
	DataTypeFP64Vector
	DataTypeInt32Vector
	DataTypeInt64Vector
	DataTypeStringVector
	DataTypeBoolVector
)

func (d DataType) String() string {
	switch {
	case d == DataTypeFP32Vector:
		return "DataTypeFP32Vector"
	case d == DataTypeFP64Vector:
		return "DataTypeFP64Vector"
	case d == DataTypeInt32Vector:
		return "DataTypeInt32Vector"
	case d == DataTypeInt64Vector:
		return "DataTypeInt64Vector"
	case d == DataTypeStringVector:
		return "DataTypeStringVector"
	case d == DataTypeBoolVector:
		return "DataTypeBoolVector"
	default:
		return "Unknown"
	}
}

func (d DataType) Size() int {
	switch {
	case d == DataTypeFP32Vector:
		return 4
	case d == DataTypeFP64Vector:
		return 8
	case d == DataTypeInt32Vector:
		return 4
	case d == DataTypeInt64Vector:
		return 8
	case d == DataTypeStringVector:
		return 0
	case d == DataTypeBoolVector:
		return 0
	default:
		return 0
	}
}

func (d DataType) IsVector() bool {
	switch {
	case d == DataTypeFP32Vector:
		return true
	case d == DataTypeFP64Vector:
		return true
	case d == DataTypeInt32Vector:
		return true
	case d == DataTypeInt64Vector:
		return true
	case d == DataTypeStringVector:
		return true
	case d == DataTypeBoolVector:
		return true
	default:
		return false
	}
}

func ParseDataType(value string) (DataType, error) {
	switch value {
	case "DataTypeFP32Vector":
		return DataTypeFP32Vector, nil
	case "DataTypeFP64Vector":
		return DataTypeFP64Vector, nil
	case "DataTypeInt32Vector":
		return DataTypeInt32Vector, nil
	case "DataTypeInt64Vector":
		return DataTypeInt64Vector, nil
	case "DataTypeStringVector":
		return DataTypeStringVector, nil
	case "DataTypeBoolVector":
		return DataTypeBoolVector, nil
	default:
		return DataTypeUnknown, errors.New("invalid data type string")
	}
}
