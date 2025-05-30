package quantization

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"reflect"
	"testing"
)

func TestQuantize_ValidInput(t *testing.T) {
	system.Init()
	matrix := [][][]byte{
		{{37, 142, 129, 59}, {0, 0, 96, 60}, {0x00, 0xc0, 0x7f, 0x38}, {229, 7, 79, 55}, {207, 0}, {253, 94}, {97, 160, 224, 196, 120, 245, 166, 72}, {97, 160, 224, 196, 120, 245, 166, 72}, {97, 160, 224, 196, 120, 245, 166, 72}, {97, 160, 224, 196, 120, 245, 166, 72}},
		{{0, 0, 128, 56}, {205, 204, 76, 57}, {0x00, 0x20, 0x80, 0x7f}, {63, 210, 166, 70}, {55, 117}, {201, 216}, {0, 0, 0, 0, 0, 0, 236, 64}, {0, 0, 0, 0, 0, 0, 236, 64}, {245, 121, 117, 0, 0, 0, 16, 63}, {0, 0, 0, 0, 0, 0, 124, 64}},
		{{0, 80, 195, 71}, {0, 0, 224, 67}, {0x00, 0xf0, 0xff, 0xc4}, {33, 32, 128, 199}, {0, 252}, {0, 252}, {225, 163, 164, 64, 225, 122, 180, 62}, {225, 163, 164, 64, 225, 122, 180, 62}, {0, 0, 0, 0, 0, 0, 8, 63}, {197, 180, 197, 107, 185, 154, 191, 63}},
		{{0, 0, 128, 55}, {0, 0, 0, 59}, {0xdb, 0x0f, 0x49, 0x40}, {0, 0, 0, 0}, {0, 0}, {183, 11}, {0, 0, 0, 0, 0, 106, 248, 64}, {0, 0, 0, 0, 0, 106, 248, 64}, {0, 0, 0, 0, 0, 0, 240, 62}, {0, 0, 0, 0, 0, 0, 240, 62}},
	}

	columnSourceDatatypeMap := map[string]types.DataType{
		"col1":  types.DataTypeFP32,
		"col2":  types.DataTypeFP32,
		"col3":  types.DataTypeFP32,
		"col4":  types.DataTypeFP32,
		"col5":  types.DataTypeFP16,
		"col6":  types.DataTypeFP16,
		"col7":  types.DataTypeFP64,
		"col8":  types.DataTypeFP64,
		"col9":  types.DataTypeFP64,
		"col10": types.DataTypeFP64,
	}
	columnRequestedDatatypeMap := map[string]types.DataType{
		"col1":  types.DataTypeFP8E5M2,
		"col2":  types.DataTypeFP8E4M3,
		"col3":  types.DataTypeFP16,
		"col4":  types.DataTypeFP16,
		"col5":  types.DataTypeFP8E5M2,
		"col6":  types.DataTypeFP8E4M3,
		"col7":  types.DataTypeFP32,
		"col8":  types.DataTypeFP16,
		"col9":  types.DataTypeFP8E5M2,
		"col10": types.DataTypeFP8E4M3,
	}
	matrixColumnIndexMap := map[string]int{
		"col1":  0,
		"col2":  1,
		"col3":  2,
		"col4":  3,
		"col5":  4,
		"col6":  5,
		"col7":  6,
		"col8":  7,
		"col9":  8,
		"col10": 9,
	}

	quantize(matrix, columnSourceDatatypeMap, columnRequestedDatatypeMap, matrixColumnIndexMap)

	expected := [][][]byte{
		{{28}, {7}, {0xff, 0x03}, {207, 0}, {1}, {126}, {0, 0, 128, 127}, {0, 124}, {124}, {127}},
		{{4}, {0}, {0x01, 0x7e}, {55, 117}, {117}, {242}, {0, 0, 96, 71}, {0, 123}, {4}, {126}},
		{{124}, {126}, {0x00, 0xe8}, {0, 252}, {252}, {255}, {10, 215, 163, 53}, {20, 0}, {3}, {32}},
		{{1}, {1}, {0x48, 0x42}, {0, 0}, {0}, {0}, {0, 80, 195, 71}, {0, 124}, {1}, {0}},
	}

	if !reflect.DeepEqual(matrix, expected) {
		t.Errorf("Expected %v, got %v", expected, matrix)
	}
}
