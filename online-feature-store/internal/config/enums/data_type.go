package enums

type DataType string

const (
	DataTypeUnknown       DataType = "DataTypeUnknown"
	DataTypeFP8E5M2       DataType = "DataTypeFP8E5M2"
	DataTypeFP8E4M3       DataType = "DataTypeFP8E4M3"
	DataTypeFP16          DataType = "DataTypeFP16"
	DataTypeFP32          DataType = "DataTypeFP32"
	DataTypeFP64          DataType = "DataTypeFP64"
	DataTypeInt8          DataType = "DataTypeInt8"
	DataTypeInt16         DataType = "DataTypeInt16"
	DataTypeInt32         DataType = "DataTypeInt32"
	DataTypeInt64         DataType = "DataTypeInt64"
	DataTypeUint8         DataType = "DataTypeUint8"
	DataTypeUint16        DataType = "DataTypeUint16"
	DataTypeUint32        DataType = "DataTypeUint32"
	DataTypeUint64        DataType = "DataTypeUint64"
	DataTypeString        DataType = "DataTypeString"
	DataTypeBool          DataType = "DataTypeBool"
	DataTypeFP8E5M2Vector DataType = "DataTypeFP8E5M2Vector"
	DataTypeFP8E4M3Vector DataType = "DataTypeFP8E4M3Vector"
	DataTypeFP16Vector    DataType = "DataTypeFP16Vector"
	DataTypeFP32Vector    DataType = "DataTypeFP32Vector"
	DataTypeFP64Vector    DataType = "DataTypeFP64Vector"
	DataTypeInt8Vector    DataType = "DataTypeInt8Vector"
	DataTypeInt16Vector   DataType = "DataTypeInt16Vector"
	DataTypeInt32Vector   DataType = "DataTypeInt32Vector"
	DataTypeInt64Vector   DataType = "DataTypeInt64Vector"
	DataTypeUint8Vector   DataType = "DataTypeUint8Vector"
	DataTypeUint16Vector  DataType = "DataTypeUint16Vector"
	DataTypeUint32Vector  DataType = "DataTypeUint32Vector"
	DataTypeUint64Vector  DataType = "DataTypeUint64Vector"
	DataTypeStringVector  DataType = "DataTypeStringVector"
	DataTypeBoolVector    DataType = "DataTypeBoolVector"
)

func (d DataType) String() string {
	switch d {
	case DataTypeFP8E5M2:
		return "DataTypeFP8E5M2"
	case DataTypeFP8E4M3:
		return "DataTypeFP8E4M3"
	case DataTypeFP16:
		return "DataTypeFP16"
	case DataTypeFP32:
		return "DataTypeFP32"
	case DataTypeFP64:
		return "DataTypeFP64"
	case DataTypeInt8:
		return "DataTypeInt8"
	case DataTypeInt16:
		return "DataTypeInt16"
	case DataTypeInt32:
		return "DataTypeInt32"
	case DataTypeInt64:
		return "DataTypeInt64"
	case DataTypeUint8:
		return "DataTypeUint8"
	case DataTypeUint16:
		return "DataTypeUint16"
	case DataTypeUint32:
		return "DataTypeUint32"
	case DataTypeUint64:
		return "DataTypeUint64"
	case DataTypeString:
		return "DataTypeString"
	case DataTypeBool:
		return "DataTypeBool"
	case DataTypeFP8E5M2Vector:
		return "DataTypeFP8E5M2Vector"
	case DataTypeFP8E4M3Vector:
		return "DataTypeFP8E4M3Vector"
	case DataTypeFP16Vector:
		return "DataTypeFP16Vector"
	case DataTypeFP32Vector:
		return "DataTypeFP32Vector"
	case DataTypeFP64Vector:
		return "DataTypeFP64Vector"
	case DataTypeInt8Vector:
		return "DataTypeInt8Vector"
	case DataTypeInt16Vector:
		return "DataTypeInt16Vector"
	case DataTypeInt32Vector:
		return "DataTypeInt32Vector"
	case DataTypeInt64Vector:
		return "DataTypeInt64Vector"
	case DataTypeUint8Vector:
		return "DataTypeUint8Vector"
	case DataTypeUint16Vector:
		return "DataTypeUint16Vector"
	case DataTypeUint32Vector:
		return "DataTypeUint32Vector"
	case DataTypeUint64Vector:
		return "DataTypeUint64Vector"
	case DataTypeStringVector:
		return "DataTypeStringVector"
	case DataTypeBoolVector:
		return "DataTypeBoolVector"
	default:
		return "DataTypeUnknown"
	}
}

func (d DataType) Size() int {
	switch d {
	case DataTypeFP8E5M2:
		return 1
	case DataTypeFP8E4M3:
		return 1
	case DataTypeFP16:
		return 2
	case DataTypeFP32:
		return 4
	case DataTypeFP64:
		return 8
	case DataTypeInt8:
		return 1
	case DataTypeInt16:
		return 2
	case DataTypeInt32:
		return 4
	case DataTypeInt64:
		return 8
	case DataTypeUint8:
		return 1
	case DataTypeUint16:
		return 2
	case DataTypeUint32:
		return 4
	case DataTypeUint64:
		return 8
	case DataTypeString:
		return 0
	case DataTypeBool:
		return 0
	case DataTypeFP8E5M2Vector:
		return 1
	case DataTypeFP8E4M3Vector:
		return 1
	case DataTypeFP16Vector:
		return 2
	case DataTypeFP32Vector:
		return 4
	case DataTypeFP64Vector:
		return 8
	case DataTypeInt8Vector:
		return 1
	case DataTypeInt16Vector:
		return 2
	case DataTypeInt32Vector:
		return 4
	case DataTypeInt64Vector:
		return 8
	case DataTypeUint8Vector:
		return 1
	case DataTypeUint16Vector:
		return 2
	case DataTypeUint32Vector:
		return 4
	case DataTypeUint64Vector:
		return 8
	case DataTypeStringVector:
		return 0
	case DataTypeBoolVector:
		return 0
	default:
		return 0
	}
}

func (d DataType) IsVector() bool {
	switch {
	case d == "DataTypeFP8E5M2Vector":
		return true
	case d == "DataTypeFP8E4M3Vector":
		return true
	case d == "DataTypeFP16Vector":
		return true
	case d == "DataTypeFP32Vector":
		return true
	case d == "DataTypeFP64Vector":
		return true
	case d == "DataTypeInt8Vector":
		return true
	case d == "DataTypeInt16Vector":
		return true
	case d == "DataTypeInt32Vector":
		return true
	case d == "DataTypeInt64Vector":
		return true
	case d == "DataTypeUint8Vector":
		return true
	case d == "DataTypeUint16Vector":
		return true
	case d == "DataTypeUint32Vector":
		return true
	case d == "DataTypeUint64Vector":
		return true
	case d == "DataTypeStringVector":
		return true
	case d == "DataTypeBoolVector":
		return true
	default:
		return false
	}
}
