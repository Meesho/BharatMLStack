package enums

type DataType string

const (
	DataTypeUnknown DataType = DataType(string(iota))
	DataTypeFP8E5M2
	DataTypeFP8E4M3
	DataTypeFP16
	DataTypeFP32
	DataTypeFP64
	DataTypeInt8
	DataTypeInt16
	DataTypeInt32
	DataTypeInt64
	DataTypeUint8
	DataTypeUint16
	DataTypeUint32
	DataTypeUint64
	DataTypeString
	DataTypeBool
	DataTypeFP8E5M2Vector
	DataTypeFP8E4M3Vector
	DataTypeFP16Vector
	DataTypeFP32Vector
	DataTypeFP64Vector
	DataTypeInt8Vector
	DataTypeInt16Vector
	DataTypeInt32Vector
	DataTypeInt64Vector
	DataTypeUint8Vector
	DataTypeUint16Vector
	DataTypeUint32Vector
	DataTypeUint64Vector
	DataTypeStringVector
	DataTypeBoolVector
)

func (d DataType) String() string {
	switch {
	case d == "DataTypeFP8E5M2":
		return "DataTypeFP8E5M2"
	case d == "DataTypeFP8E4M3":
		return "DataTypeFP8E4M3"
	case d == "DataTypeFP16":
		return "DataTypeFP16"
	case d == "DataTypeFP32":
		return "DataTypeFP32"
	case d == "DataTypeFP64":
		return "DataTypeFP64"
	case d == "DataTypeInt8":
		return "DataTypeInt8"
	case d == "DataTypeInt16":
		return "DataTypeInt16"
	case d == "DataTypeInt32":
		return "DataTypeInt32"
	case d == "DataTypeInt64":
		return "DataTypeInt64"
	case d == "DataTypeUint8":
		return "DataTypeUint8"
	case d == "DataTypeUint16":
		return "DataTypeUint16"
	case d == "DataTypeUint32":
		return "DataTypeUint32"
	case d == "DataTypeUint64":
		return "DataTypeUint64"
	case d == "DataTypeString":
		return "DataTypeString"
	case d == "DataTypeBool":
		return "DataTypeBool"
	case d == "DataTypeFP8E5M2Vector":
		return "DataTypeFP8E5M2Vector"
	case d == "DataTypeFP8E4M3Vector":
		return "DataTypeFP8E4M3Vector"
	case d == "DataTypeFP16Vector":
		return "DataTypeFP16Vector"
	case d == "DataTypeFP32Vector":
		return "DataTypeFP32Vector"
	case d == "DataTypeFP64Vector":
		return "DataTypeFP64Vector"
	case d == "DataTypeInt8Vector":
		return "DataTypeInt8Vector"
	case d == "DataTypeInt16Vector":
		return "DataTypeInt16Vector"
	case d == "DataTypeInt32Vector":
		return "DataTypeInt32Vector"
	case d == "DataTypeInt64Vector":
		return "DataTypeInt64Vector"
	case d == "DataTypeUint8Vector":
		return "DataTypeUint8Vector"
	case d == "DataTypeUint16Vector":
		return "DataTypeUint16Vector"
	case d == "DataTypeUint32Vector":
		return "DataTypeUint32Vector"
	case d == "DataTypeUint64Vector":
		return "DataTypeUint64Vector"
	case d == "DataTypeStringVector":
		return "DataTypeStringVector"
	case d == "DataTypeBoolVector":
		return "DataTypeBoolVector"
	default:
		return "Unknown"
	}
}

func (d DataType) Size() int {
	switch {
	case d == "DataTypeFP8E5M2":
		return 1
	case d == "DataTypeFP8E4M3":
		return 1
	case d == "DataTypeFP16":
		return 2
	case d == "DataTypeFP32":
		return 4
	case d == "DataTypeFP64":
		return 8
	case d == "DataTypeInt8":
		return 1
	case d == "DataTypeInt16":
		return 2
	case d == "DataTypeInt32":
		return 4
	case d == "DataTypeInt64":
		return 8
	case d == "DataTypeUint8":
		return 1
	case d == "DataTypeUint16":
		return 2
	case d == "DataTypeUint32":
		return 4
	case d == "DataTypeUint64":
		return 8
	case d == "DataTypeString":
		return 0
	case d == "DataTypeBool":
		return 0
	case d == "DataTypeFP8E5M2Vector":
		return 1
	case d == "DataTypeFP8E4M3Vector":
		return 1
	case d == "DataTypeFP16Vector":
		return 2
	case d == "DataTypeFP32Vector":
		return 4
	case d == "DataTypeFP64Vector":
		return 8
	case d == "DataTypeInt8Vector":
		return 1
	case d == "DataTypeInt16Vector":
		return 2
	case d == "DataTypeInt32Vector":
		return 4
	case d == "DataTypeInt64Vector":
		return 8
	case d == "DataTypeUint8Vector":
		return 1
	case d == "DataTypeUint16Vector":
		return 2
	case d == "DataTypeUint32Vector":
		return 4
	case d == "DataTypeUint64Vector":
		return 8
	case d == "DataTypeStringVector":
		return 0
	case d == "DataTypeBoolVector":
		return 0
	default:
		return 0
	}
}

func (d DataType) IsVector() bool {
	switch {
	case d == DataTypeFP8E5M2Vector:
		return true
	case d == DataTypeFP8E4M3Vector:
		return true
	case d == DataTypeFP16Vector:
		return true
	case d == DataTypeFP32Vector:
		return true
	case d == DataTypeFP64Vector:
		return true
	case d == DataTypeInt8Vector:
		return true
	case d == DataTypeInt16Vector:
		return true
	case d == DataTypeInt32Vector:
		return true
	case d == DataTypeInt64Vector:
		return true
	case d == DataTypeUint8Vector:
		return true
	case d == DataTypeUint16Vector:
		return true
	case d == DataTypeUint32Vector:
		return true
	case d == DataTypeUint64Vector:
		return true
	case d == DataTypeStringVector:
		return true
	case d == DataTypeBoolVector:
		return true
	default:
		return false
	}
}
