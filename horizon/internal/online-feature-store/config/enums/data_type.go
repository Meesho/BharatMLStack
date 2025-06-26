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

var dataTypeSizes = map[DataType]int{
	DataTypeFP8E5M2:       1,
	DataTypeFP8E4M3:       1,
	DataTypeFP16:          2,
	DataTypeFP32:          4,
	DataTypeFP64:          8,
	DataTypeInt8:          1,
	DataTypeInt16:         2,
	DataTypeInt32:         4,
	DataTypeInt64:         8,
	DataTypeUint8:         1,
	DataTypeUint16:        2,
	DataTypeUint32:        4,
	DataTypeUint64:        8,
	DataTypeString:        0,
	DataTypeBool:          0,
	DataTypeFP8E5M2Vector: 1,
	DataTypeFP8E4M3Vector: 1,
	DataTypeFP16Vector:    2,
	DataTypeFP32Vector:    4,
	DataTypeFP64Vector:    8,
	DataTypeInt8Vector:    1,
	DataTypeInt16Vector:   2,
	DataTypeInt32Vector:   4,
	DataTypeInt64Vector:   8,
	DataTypeUint8Vector:   1,
	DataTypeUint16Vector:  2,
	DataTypeUint32Vector:  4,
	DataTypeUint64Vector:  8,
	DataTypeStringVector:  0,
	DataTypeBoolVector:    0,
}

var vectorTypes = map[DataType]bool{
	DataTypeFP8E5M2Vector: true,
	DataTypeFP8E4M3Vector: true,
	DataTypeFP16Vector:    true,
	DataTypeFP32Vector:    true,
	DataTypeFP64Vector:    true,
	DataTypeInt8Vector:    true,
	DataTypeInt16Vector:   true,
	DataTypeInt32Vector:   true,
	DataTypeInt64Vector:   true,
	DataTypeUint8Vector:   true,
	DataTypeUint16Vector:  true,
	DataTypeUint32Vector:  true,
	DataTypeUint64Vector:  true,
	DataTypeStringVector:  true,
	DataTypeBoolVector:    true,
}

func (d DataType) String() string {
	if _, ok := dataTypeSizes[d]; ok {
		return string(d)
	}
	return "Unknown"
}

func (d DataType) Size() int {
	if size, ok := dataTypeSizes[d]; ok {
		return size
	}
	return 0
}

func (d DataType) IsVector() bool {
	return vectorTypes[d]
}
