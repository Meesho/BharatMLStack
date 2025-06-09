package onfs

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/onfs/persist"
	pb "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/onfs/retrieve"
)

func TestConvertToQueriesProto(t *testing.T) {
	adapter := &Adapter{}

	tests := []struct {
		name      string
		query     *Query
		batchSize int
		want      []*pb.Query
		wantErr   bool
	}{
		{
			name:      "nil query",
			query:     nil,
			batchSize: 2,
			want:      nil,
			wantErr:   false,
		},
		{
			name: "valid query with batching",
			query: &Query{
				EntityLabel: "user",
				Keys: []Keys{
					{Cols: []string{"123", "456"}},
					{Cols: []string{"789", "012"}},
					{Cols: []string{"345", "678"}},
				},
				KeysSchema: []string{"user_id", "sscat_id"},
				FeatureGroups: []FeatureGroup{
					{
						Label:         "derived_string",
						FeatureLabels: []string{"feature1", "feature2"},
					},
					{
						Label:         "derived_int64",
						FeatureLabels: []string{"feature3", "feature4"},
					},
				},
			},
			batchSize: 2,
			want: []*pb.Query{
				{
					EntityLabel: "user",
					KeysSchema:  []string{"user_id", "sscat_id"},
					Keys: []*pb.Keys{
						{Cols: []string{"123", "456"}},
						{Cols: []string{"789", "012"}},
					},
					FeatureGroups: []*pb.FeatureGroup{
						{
							Label:         "derived_string",
							FeatureLabels: []string{"feature1", "feature2"},
						},
						{
							Label:         "derived_int64",
							FeatureLabels: []string{"feature3", "feature4"},
						},
					},
				},
				{
					EntityLabel: "user",
					KeysSchema:  []string{"user_id", "sscat_id"},
					Keys: []*pb.Keys{
						{Cols: []string{"345", "678"}},
					},
					FeatureGroups: []*pb.FeatureGroup{
						{
							Label:         "derived_string",
							FeatureLabels: []string{"feature1", "feature2"},
						},
						{
							Label:         "derived_int64",
							FeatureLabels: []string{"feature3", "feature4"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid batch size",
			query: &Query{
				EntityLabel: "user",
				Keys: []Keys{
					{Cols: []string{"123", "456"}},
				},
			},
			batchSize: 0,
			want:      nil,
			wantErr:   true,
		},
		{
			name: "empty keys",
			query: &Query{
				EntityLabel: "user",
				Keys:        []Keys{},
			},
			batchSize: 2,
			want:      []*pb.Query{},
			wantErr:   false,
		},
		{
			name: "empty feature groups",
			query: &Query{
				EntityLabel:   "user",
				Keys:          []Keys{{Cols: []string{"123", "456"}}},
				FeatureGroups: []FeatureGroup{},
			},
			batchSize: 2,
			want: []*pb.Query{
				{
					EntityLabel:   "user",
					Keys:          []*pb.Keys{{Cols: []string{"123", "456"}}},
					FeatureGroups: []*pb.FeatureGroup{},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := adapter.ConvertToQueriesProto(tt.query, tt.batchSize)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertToResult(t *testing.T) {
	adapter := &Adapter{}

	tests := []struct {
		name    string
		payload *pb.Result
		want    Result
	}{
		{
			name:    "nil payload",
			payload: nil,
			want:    Result{},
		},
		{
			name: "empty payload",
			payload: &pb.Result{
				Rows: []*pb.Row{},
			},
			want: Result{},
		},
		{
			name: "valid payload",
			payload: &pb.Result{
				EntityLabel: "",
				KeysSchema:  []string{"user_id", "sscat_id"},
				FeatureSchemas: []*pb.FeatureSchema{
					{
						FeatureGroupLabel: "derived_string",
						Features: []*pb.Feature{
							{
								Label:     "feature1",
								ColumnIdx: 0,
							},
							{
								Label:     "feature2",
								ColumnIdx: 1,
							},
						},
					},
				},
				Rows: []*pb.Row{
					{
						Keys:    []string{"123", "456"},
						Columns: [][]byte{[]byte("value1"), []byte("value2")},
					},
				},
			},
			want: Result{
				EntityLabel: "",
				KeysSchema:  []string{"user_id", "sscat_id"},
				FeatureSchemas: []FeatureSchema{
					{
						FeatureGroupLabel: "derived_string",
						Features: []Feature{
							{
								Label:     "feature1",
								ColumnIdx: 0,
							},
							{
								Label:     "feature2",
								ColumnIdx: 1,
							},
						},
					},
				},
				Rows: []Row{
					{
						Keys:    []string{"123", "456"},
						Columns: [][]byte{[]byte("value1"), []byte("value2")},
					},
				},
			},
		},
		{
			name: "empty feature schemas",
			payload: &pb.Result{
				EntityLabel:    "",
				KeysSchema:     []string{"user_id", "sscat_id"},
				FeatureSchemas: []*pb.FeatureSchema{},
				Rows: []*pb.Row{
					{
						Keys:    []string{"123", "456"},
						Columns: [][]byte{[]byte("value1"), []byte("value2")},
					},
				},
			},
			want: Result{
				EntityLabel:    "",
				KeysSchema:     []string{"user_id", "sscat_id"},
				FeatureSchemas: []FeatureSchema{},
				Rows: []Row{
					{
						Keys:    []string{"123", "456"},
						Columns: [][]byte{[]byte("value1"), []byte("value2")},
					},
				},
			},
		},
		{
			name: "empty keys schema",
			payload: &pb.Result{
				EntityLabel:    "",
				KeysSchema:     []string{},
				FeatureSchemas: []*pb.FeatureSchema{},
				Rows: []*pb.Row{
					{
						Keys:    []string{},
						Columns: [][]byte{},
					},
				},
			},
			want: Result{
				EntityLabel:    "",
				KeysSchema:     []string{},
				FeatureSchemas: []FeatureSchema{},
				Rows: []Row{
					{
						Keys:    []string{},
						Columns: [][]byte{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.ConvertToResult(tt.payload)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertToDecodedResult(t *testing.T) {
	adapter := &Adapter{}

	tests := []struct {
		name    string
		payload *pb.DecodedResult
		want    *DecodedResult
	}{
		{
			name:    "nil payload",
			payload: nil,
			want:    nil,
		},
		{
			name: "empty payload",
			payload: &pb.DecodedResult{
				Rows: []*pb.DecodedRow{},
			},
			want: nil,
		},
		{
			name: "valid payload",
			payload: &pb.DecodedResult{
				KeysSchema: []string{"user_id", "sscat_id"},
				FeatureSchemas: []*pb.FeatureSchema{
					{
						FeatureGroupLabel: "derived_string",
						Features: []*pb.Feature{
							{
								Label:     "feature1",
								ColumnIdx: 0,
							},
							{
								Label:     "feature2",
								ColumnIdx: 1,
							},
						},
					},
				},
				Rows: []*pb.DecodedRow{
					{
						Keys:    []string{"123", "456"},
						Columns: []string{"value1", "value2"},
					},
				},
			},
			want: &DecodedResult{
				KeysSchema: []string{"user_id", "sscat_id"},
				FeatureSchemas: []FeatureSchema{
					{
						FeatureGroupLabel: "derived_string",
						Features: []Feature{
							{
								Label:     "feature1",
								ColumnIdx: 0,
							},
							{
								Label:     "feature2",
								ColumnIdx: 1,
							},
						},
					},
				},
				Rows: []DecodedRow{
					{
						Keys:    []string{"123", "456"},
						Columns: []string{"value1", "value2"},
					},
				},
			},
		},
		{
			name: "empty feature schemas",
			payload: &pb.DecodedResult{
				KeysSchema:     []string{"user_id", "sscat_id"},
				FeatureSchemas: []*pb.FeatureSchema{},
				Rows: []*pb.DecodedRow{
					{
						Keys:    []string{"123", "456"},
						Columns: []string{"value1", "value2"},
					},
				},
			},
			want: &DecodedResult{
				KeysSchema:     []string{"user_id", "sscat_id"},
				FeatureSchemas: []FeatureSchema{},
				Rows: []DecodedRow{
					{
						Keys:    []string{"123", "456"},
						Columns: []string{"value1", "value2"},
					},
				},
			},
		},
		{
			name: "empty keys schema",
			payload: &pb.DecodedResult{
				KeysSchema:     []string{},
				FeatureSchemas: []*pb.FeatureSchema{},
				Rows: []*pb.DecodedRow{
					{
						Keys:    []string{},
						Columns: []string{},
					},
				},
			},
			want: &DecodedResult{
				KeysSchema:     []string{},
				FeatureSchemas: []FeatureSchema{},
				Rows: []DecodedRow{
					{
						Keys:    []string{},
						Columns: []string{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.ConvertToDecodedResult(tt.payload)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBatch(t *testing.T) {
	tests := []struct {
		name      string
		in        []int
		batchSize int
		want      [][]int
		wantErr   bool
	}{
		{
			name:      "empty input",
			in:        []int{},
			batchSize: 2,
			want:      nil,
			wantErr:   false,
		},
		{
			name:      "single batch",
			in:        []int{1, 2},
			batchSize: 2,
			want:      [][]int{{1, 2}},
			wantErr:   false,
		},
		{
			name:      "multiple batches",
			in:        []int{1, 2, 3, 4, 5},
			batchSize: 2,
			want:      [][]int{{1, 2}, {3, 4}, {5}},
			wantErr:   false,
		},
		{
			name:      "invalid batch size",
			in:        []int{1, 2},
			batchSize: 0,
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "negative batch size",
			in:        []int{1, 2},
			batchSize: -1,
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "batch size of 1",
			in:        []int{1, 2, 3},
			batchSize: 1,
			want:      [][]int{{1}, {2}, {3}},
			wantErr:   false,
		},
		{
			name:      "batch size larger than input",
			in:        []int{1, 2, 3},
			batchSize: 5,
			want:      [][]int{{1, 2, 3}},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := batch(tt.in, tt.batchSize)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertFeatures(t *testing.T) {
	tests := []struct {
		name          string
		protoFeatures []*pb.Feature
		want          []Feature
	}{
		{
			name:          "nil features",
			protoFeatures: nil,
			want:          []Feature{},
		},
		{
			name:          "empty features",
			protoFeatures: []*pb.Feature{},
			want:          []Feature{},
		},
		{
			name: "valid features",
			protoFeatures: []*pb.Feature{
				{
					Label:     "feature1",
					ColumnIdx: 0,
				},
				{
					Label:     "feature2",
					ColumnIdx: 1,
				},
			},
			want: []Feature{
				{
					Label:     "feature1",
					ColumnIdx: 0,
				},
				{
					Label:     "feature2",
					ColumnIdx: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertFeatures(tt.protoFeatures)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertRows(t *testing.T) {
	tests := []struct {
		name      string
		protoRows []*pb.Row
		want      []Row
	}{
		{
			name:      "nil rows",
			protoRows: nil,
			want:      []Row{},
		},
		{
			name:      "empty rows",
			protoRows: []*pb.Row{},
			want:      []Row{},
		},
		{
			name: "valid rows",
			protoRows: []*pb.Row{
				{
					Keys:    []string{"123", "456"},
					Columns: [][]byte{[]byte("value1"), []byte("value2")},
				},
				{
					Keys:    []string{"789", "012"},
					Columns: [][]byte{[]byte("value3"), []byte("value4")},
				},
			},
			want: []Row{
				{
					Keys:    []string{"123", "456"},
					Columns: [][]byte{[]byte("value1"), []byte("value2")},
				},
				{
					Keys:    []string{"789", "012"},
					Columns: [][]byte{[]byte("value3"), []byte("value4")},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertRows(tt.protoRows)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertDecodedRows(t *testing.T) {
	tests := []struct {
		name      string
		protoRows []*pb.DecodedRow
		want      []DecodedRow
	}{
		{
			name:      "nil rows",
			protoRows: nil,
			want:      []DecodedRow{},
		},
		{
			name:      "empty rows",
			protoRows: []*pb.DecodedRow{},
			want:      []DecodedRow{},
		},
		{
			name: "valid rows",
			protoRows: []*pb.DecodedRow{
				{
					Keys:    []string{"123", "456"},
					Columns: []string{"value1", "value2"},
				},
				{
					Keys:    []string{"789", "012"},
					Columns: []string{"value3", "value4"},
				},
			},
			want: []DecodedRow{
				{
					Keys:    []string{"123", "456"},
					Columns: []string{"value1", "value2"},
				},
				{
					Keys:    []string{"789", "012"},
					Columns: []string{"value3", "value4"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertDecodedRows(tt.protoRows)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertToPersistRequest(t *testing.T) {
	adapter := &Adapter{}

	tests := []struct {
		name    string
		request *PersistFeaturesRequest
		want    *persist.Query
	}{
		{
			name:    "nil request",
			request: nil,
			want:    nil,
		},
		{
			name: "valid request with all value types",
			request: &PersistFeaturesRequest{
				EntityLabel: "user_sscat",
				KeysSchema:  []string{"user_id", "sscat_id"},
				FeatureGroups: []FeatureGroupSchema{
					{
						Label:         "derived_int64",
						FeatureLabels: []string{"feature1", "feature2"},
					},
					{
						Label:         "derived_string",
						FeatureLabels: []string{"feature3", "feature4"},
					},
				},
				Data: []Data{
					{
						KeyValues: []string{"123", "456"},
						FeatureValues: []FeatureValues{
							{
								Values: Values{
									Fp32Values:   []float64{1.1, 2.2},
									Fp64Values:   []float64{3.3, 4.4},
									Int32Values:  []int32{5, 6},
									Int64Values:  []int64{7, 8},
									Uint32Values: []uint32{9, 10},
									Uint64Values: []uint64{11, 12},
									StringValues: []string{"a", "b"},
									BoolValues:   []bool{true, false},
									Vector: []Vector{
										{
											Values: Values{
												Int32Values: []int32{1, 2, 3},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: &persist.Query{
				EntityLabel: "user_sscat",
				KeysSchema:  []string{"user_id", "sscat_id"},
				FeatureGroupSchema: []*persist.FeatureGroupSchema{
					{
						Label:         "derived_int64",
						FeatureLabels: []string{"feature1", "feature2"},
					},
					{
						Label:         "derived_string",
						FeatureLabels: []string{"feature3", "feature4"},
					},
				},
				Data: []*persist.Data{
					{
						KeyValues: []string{"123", "456"},
						FeatureValues: []*persist.FeatureValues{
							{
								Values: &persist.Values{
									Fp32Values:   []float64{1.1, 2.2},
									Fp64Values:   []float64{3.3, 4.4},
									Int32Values:  []int32{5, 6},
									Int64Values:  []int64{7, 8},
									Uint32Values: []uint32{9, 10},
									Uint64Values: []uint64{11, 12},
									StringValues: []string{"a", "b"},
									BoolValues:   []bool{true, false},
									Vector: []*persist.Vector{
										{
											Values: &persist.Values{
												Int32Values: []int32{1, 2, 3},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "empty request",
			request: &PersistFeaturesRequest{
				EntityLabel:   "user_sscat",
				KeysSchema:    []string{},
				FeatureGroups: []FeatureGroupSchema{},
				Data:          []Data{},
			},
			want: &persist.Query{
				EntityLabel:        "user_sscat",
				KeysSchema:         []string{},
				FeatureGroupSchema: []*persist.FeatureGroupSchema{},
				Data:               []*persist.Data{},
			},
		},
		{
			name: "request with empty feature values",
			request: &PersistFeaturesRequest{
				EntityLabel: "user_sscat",
				KeysSchema:  []string{"user_id"},
				FeatureGroups: []FeatureGroupSchema{
					{
						Label:         "group1",
						FeatureLabels: []string{"feature1"},
					},
				},
				Data: []Data{
					{
						KeyValues:     []string{"123"},
						FeatureValues: []FeatureValues{},
					},
				},
			},
			want: &persist.Query{
				EntityLabel: "user_sscat",
				KeysSchema:  []string{"user_id"},
				FeatureGroupSchema: []*persist.FeatureGroupSchema{
					{
						Label:         "group1",
						FeatureLabels: []string{"feature1"},
					},
				},
				Data: []*persist.Data{
					{
						KeyValues:     []string{"123"},
						FeatureValues: []*persist.FeatureValues{},
					},
				},
			},
		},
		{
			name: "request with empty vector values",
			request: &PersistFeaturesRequest{
				EntityLabel: "user_sscat",
				KeysSchema:  []string{"user_id"},
				FeatureGroups: []FeatureGroupSchema{
					{
						Label:         "group1",
						FeatureLabels: []string{"feature1"},
					},
				},
				Data: []Data{
					{
						KeyValues: []string{"123"},
						FeatureValues: []FeatureValues{
							{
								Values: Values{
									Vector: []Vector{},
								},
							},
						},
					},
				},
			},
			want: &persist.Query{
				EntityLabel: "user_sscat",
				KeysSchema:  []string{"user_id"},
				FeatureGroupSchema: []*persist.FeatureGroupSchema{
					{
						Label:         "group1",
						FeatureLabels: []string{"feature1"},
					},
				},
				Data: []*persist.Data{
					{
						KeyValues: []string{"123"},
						FeatureValues: []*persist.FeatureValues{
							{
								Values: &persist.Values{
									Vector: []*persist.Vector{},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.ConvertToPersistRequest(tt.request)
			assert.Equal(t, tt.want, got)
		})
	}
}
