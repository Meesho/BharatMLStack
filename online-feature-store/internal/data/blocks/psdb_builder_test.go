package blocks

import (
	"testing"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestScalarNumericDataLength(t *testing.T) {
	tests := []struct {
		name        string
		dataType    types.DataType
		noOfFeature int
		want        int
		wantErr     bool
	}{
		{
			name:        "int32 scalar",
			dataType:    types.DataTypeInt32,
			noOfFeature: 3,
			want:        12, // 3 * 4 bytes
		},
		{
			name:        "float64 scalar",
			dataType:    types.DataTypeFP64,
			noOfFeature: 2,
			want:        16, // 2 * 8 bytes
		},
		{
			name:        "uint16 scalar",
			dataType:    types.DataTypeUint16,
			noOfFeature: 4,
			want:        8, // 4 * 2 bytes
		},
		{
			name:        "zero features",
			dataType:    types.DataTypeInt32,
			noOfFeature: 0,
			want:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scalarNumericDataLength(tt.dataType, tt.noOfFeature)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVectorNumericDataLength(t *testing.T) {
	tests := []struct {
		name          string
		dataType      types.DataType
		noOfFeature   int
		vectorLengths []uint16
		want          int
		wantErr       bool
	}{
		{
			name:          "int32 vector",
			dataType:      types.DataTypeInt32Vector,
			noOfFeature:   2,
			vectorLengths: []uint16{3, 2},
			want:          20, // (3+2) * 4 bytes
		},
		{
			name:          "float64 vector",
			dataType:      types.DataTypeFP64Vector,
			noOfFeature:   2,
			vectorLengths: []uint16{2, 3},
			want:          40, // (2+3) * 8 bytes
		},
		{
			name:          "mismatched lengths",
			dataType:      types.DataTypeInt32Vector,
			noOfFeature:   2,
			vectorLengths: []uint16{3},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := vectorNumericDataLength(tt.dataType, tt.noOfFeature, tt.vectorLengths)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScalarStringDataLength(t *testing.T) {
	tests := []struct {
		name          string
		stringLengths []uint16
		noOfFeature   int
		want          int
		wantErr       bool
	}{
		{
			name:          "basic strings",
			stringLengths: []uint16{5, 3, 4},
			noOfFeature:   3,
			want:          18, // 5 + 3 + 4 + (2*noOfFeatures)
		},
		{
			name:          "empty strings",
			stringLengths: []uint16{0, 0, 0},
			noOfFeature:   3,
			want:          6,
		},
		{
			name:          "mismatched lengths",
			stringLengths: []uint16{5, 3},
			noOfFeature:   3,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scalarStringDataLength(tt.stringLengths, tt.noOfFeature)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVectorStringDataLength(t *testing.T) {
	tests := []struct {
		name          string
		stringLengths []uint16
		noOfFeature   int
		vectorLengths []uint16
		want          int
		wantErr       bool
	}{
		{
			name:          "basic string vectors",
			stringLengths: []uint16{5, 3},
			noOfFeature:   2,
			vectorLengths: []uint16{2, 3},
			want:          29, // (5*2) + (3*3)
		},
		{
			name:          "empty strings",
			stringLengths: []uint16{0, 0},
			noOfFeature:   2,
			vectorLengths: []uint16{2, 2},
			want:          8,
		},
		{
			name:          "mismatched feature count",
			stringLengths: []uint16{5},
			noOfFeature:   2,
			vectorLengths: []uint16{2, 2},
			wantErr:       true,
		},
		{
			name:          "mismatched vector lengths",
			stringLengths: []uint16{5, 3},
			noOfFeature:   2,
			vectorLengths: []uint16{2},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := vectorStringDataLength(tt.stringLengths, tt.noOfFeature, tt.vectorLengths)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScalarBoolDataLength(t *testing.T) {
	tests := []struct {
		name        string
		noOfFeature int
		want        int
		wantErr     bool
	}{
		{
			name:        "single byte",
			noOfFeature: 7,
			want:        1, // ceil(7/8)
		},
		{
			name:        "multiple bytes",
			noOfFeature: 17,
			want:        3, // ceil(17/8)
		},
		{
			name:        "exact byte boundary",
			noOfFeature: 16,
			want:        2, // ceil(16/8)
		},
		{
			name:        "zero features",
			noOfFeature: 0,
			want:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scalarBoolDataLength(tt.noOfFeature)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVectorBoolDataLength(t *testing.T) {
	tests := []struct {
		name          string
		noOfFeature   int
		vectorLengths []uint16
		want          int
		wantErr       bool
	}{
		{
			name:          "single byte per vector",
			noOfFeature:   2,
			vectorLengths: []uint16{7, 6},
			want:          2, // ceil(7/8) + ceil(6/8)
		},
		{
			name:          "multiple bytes",
			noOfFeature:   2,
			vectorLengths: []uint16{9, 15},
			want:          3, // ceil(9/8 + 15/8)
		},
		{
			name:          "byte boundaries",
			noOfFeature:   2,
			vectorLengths: []uint16{8, 16},
			want:          3, // ceil(8/8) + ceil(16/8)
		},
		{
			name:          "mismatched lengths",
			noOfFeature:   2,
			vectorLengths: []uint16{8},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := vectorBoolDataLength(tt.noOfFeature, tt.vectorLengths)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
