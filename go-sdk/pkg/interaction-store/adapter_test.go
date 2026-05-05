package interactionstore

import (
	"testing"

	pb "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/interaction-store/timeseries"
	"github.com/stretchr/testify/assert"
)

func TestAdapter_ConvertToPersistClickDataProto(t *testing.T) {
	adapter := Adapter{}

	tests := []struct {
		name    string
		request *PersistClickDataRequest
		want    *pb.PersistClickDataRequest
	}{
		{
			name:    "nil request",
			request: nil,
			want:    nil,
		},
		{
			name: "valid request",
			request: &PersistClickDataRequest{
				UserId: "user123",
				Data: []ClickData{
					{CatalogId: 100, ProductId: 200, Timestamp: 1704067200000, Metadata: "meta1"},
					{CatalogId: 101, ProductId: 201, Timestamp: 1704067201000, Metadata: "meta2"},
				},
			},
			want: &pb.PersistClickDataRequest{
				UserId: "user123",
				Data: []*pb.ClickData{
					{CatalogId: 100, ProductId: 200, Timestamp: 1704067200000, Metadata: "meta1"},
					{CatalogId: 101, ProductId: 201, Timestamp: 1704067201000, Metadata: "meta2"},
				},
			},
		},
		{
			name: "empty data",
			request: &PersistClickDataRequest{
				UserId: "user123",
				Data:   []ClickData{},
			},
			want: &pb.PersistClickDataRequest{
				UserId: "user123",
				Data:   []*pb.ClickData{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.ConvertToPersistClickDataProto(tt.request)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.want.UserId, got.UserId)
				assert.Len(t, got.Data, len(tt.want.Data))
			}
		})
	}
}

func TestAdapter_ConvertToPersistOrderDataProto(t *testing.T) {
	adapter := Adapter{}

	tests := []struct {
		name    string
		request *PersistOrderDataRequest
		want    *pb.PersistOrderDataRequest
	}{
		{
			name:    "nil request",
			request: nil,
			want:    nil,
		},
		{
			name: "valid request",
			request: &PersistOrderDataRequest{
				UserId: "user123",
				Data: []OrderData{
					{CatalogId: 100, ProductId: 200, SubOrderNum: "SUB001", Timestamp: 1704067200000, Metadata: "meta1"},
				},
			},
			want: &pb.PersistOrderDataRequest{
				UserId: "user123",
				Data: []*pb.OrderData{
					{CatalogId: 100, ProductId: 200, SubOrderNum: "SUB001", Timestamp: 1704067200000, Metadata: "meta1"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.ConvertToPersistOrderDataProto(tt.request)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.want.UserId, got.UserId)
				assert.Len(t, got.Data, len(tt.want.Data))
			}
		})
	}
}

func TestAdapter_ConvertToRetrieveDataProto(t *testing.T) {
	adapter := Adapter{}

	tests := []struct {
		name    string
		request *RetrieveDataRequest
		want    *pb.RetrieveDataRequest
	}{
		{
			name:    "nil request",
			request: nil,
			want:    nil,
		},
		{
			name: "valid request",
			request: &RetrieveDataRequest{
				UserId:         "user123",
				StartTimestamp: 1704067200000,
				EndTimestamp:   1704153600000,
				Limit:          100,
			},
			want: &pb.RetrieveDataRequest{
				UserId:         "user123",
				StartTimestamp: 1704067200000,
				EndTimestamp:   1704153600000,
				Limit:          100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.ConvertToRetrieveDataProto(tt.request)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.want.UserId, got.UserId)
				assert.Equal(t, tt.want.StartTimestamp, got.StartTimestamp)
				assert.Equal(t, tt.want.EndTimestamp, got.EndTimestamp)
				assert.Equal(t, tt.want.Limit, got.Limit)
			}
		})
	}
}

func TestAdapter_ConvertToRetrieveInteractionsProto(t *testing.T) {
	adapter := Adapter{}

	tests := []struct {
		name    string
		request *RetrieveInteractionsRequest
		want    *pb.RetrieveInteractionsRequest
	}{
		{
			name:    "nil request",
			request: nil,
			want:    nil,
		},
		{
			name: "valid request with multiple interaction types",
			request: &RetrieveInteractionsRequest{
				UserId:           "user123",
				InteractionTypes: []InteractionType{InteractionTypeClick, InteractionTypeOrder},
				StartTimestamp:   1704067200000,
				EndTimestamp:     1704153600000,
				Limit:            100,
			},
			want: &pb.RetrieveInteractionsRequest{
				UserId:           "user123",
				InteractionTypes: []pb.InteractionType{pb.InteractionType_CLICK, pb.InteractionType_ORDER},
				StartTimestamp:   1704067200000,
				EndTimestamp:     1704153600000,
				Limit:            100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.ConvertToRetrieveInteractionsProto(tt.request)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.want.UserId, got.UserId)
				assert.Len(t, got.InteractionTypes, len(tt.want.InteractionTypes))
			}
		})
	}
}

func TestAdapter_ConvertFromPersistDataResponse(t *testing.T) {
	adapter := Adapter{}

	tests := []struct {
		name     string
		response *pb.PersistDataResponse
		want     *PersistDataResponse
	}{
		{
			name:     "nil response",
			response: nil,
			want:     nil,
		},
		{
			name: "valid response",
			response: &pb.PersistDataResponse{
				Message: "success",
			},
			want: &PersistDataResponse{
				Message: "success",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.ConvertFromPersistDataResponse(tt.response)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.want.Message, got.Message)
			}
		})
	}
}

func TestAdapter_ConvertFromRetrieveClickDataResponse(t *testing.T) {
	adapter := Adapter{}

	tests := []struct {
		name     string
		response *pb.RetrieveClickDataResponse
		want     *RetrieveClickDataResponse
	}{
		{
			name:     "nil response",
			response: nil,
			want:     &RetrieveClickDataResponse{Data: []ClickEvent{}},
		},
		{
			name:     "empty data",
			response: &pb.RetrieveClickDataResponse{Data: nil},
			want:     &RetrieveClickDataResponse{Data: []ClickEvent{}},
		},
		{
			name: "valid response",
			response: &pb.RetrieveClickDataResponse{
				Data: []*pb.ClickEvent{
					{CatalogId: 100, ProductId: 200, Timestamp: 1704067200000, Metadata: "meta1"},
					{CatalogId: 101, ProductId: 201, Timestamp: 1704067201000, Metadata: "meta2"},
				},
			},
			want: &RetrieveClickDataResponse{
				Data: []ClickEvent{
					{CatalogId: 100, ProductId: 200, Timestamp: 1704067200000, Metadata: "meta1"},
					{CatalogId: 101, ProductId: 201, Timestamp: 1704067201000, Metadata: "meta2"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.ConvertFromRetrieveClickDataResponse(tt.response)
			assert.Len(t, got.Data, len(tt.want.Data))
			for i, e := range got.Data {
				assert.Equal(t, tt.want.Data[i].CatalogId, e.CatalogId)
				assert.Equal(t, tt.want.Data[i].ProductId, e.ProductId)
			}
		})
	}
}

func TestAdapter_ConvertFromRetrieveOrderDataResponse(t *testing.T) {
	adapter := Adapter{}

	tests := []struct {
		name     string
		response *pb.RetrieveOrderDataResponse
		want     *RetrieveOrderDataResponse
	}{
		{
			name:     "nil response",
			response: nil,
			want:     &RetrieveOrderDataResponse{Data: []OrderEvent{}},
		},
		{
			name: "valid response",
			response: &pb.RetrieveOrderDataResponse{
				Data: []*pb.OrderEvent{
					{CatalogId: 100, ProductId: 200, SubOrderNum: "SUB001", Timestamp: 1704067200000, Metadata: "meta1"},
				},
			},
			want: &RetrieveOrderDataResponse{
				Data: []OrderEvent{
					{CatalogId: 100, ProductId: 200, SubOrderNum: "SUB001", Timestamp: 1704067200000, Metadata: "meta1"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.ConvertFromRetrieveOrderDataResponse(tt.response)
			assert.Len(t, got.Data, len(tt.want.Data))
		})
	}
}

func TestAdapter_ConvertFromRetrieveInteractionsResponse(t *testing.T) {
	adapter := Adapter{}

	tests := []struct {
		name     string
		response *pb.RetrieveInteractionsResponse
		wantLen  int
	}{
		{
			name:     "nil response",
			response: nil,
			wantLen:  0,
		},
		{
			name:     "empty data",
			response: &pb.RetrieveInteractionsResponse{Data: nil},
			wantLen:  0,
		},
		{
			name: "valid response",
			response: &pb.RetrieveInteractionsResponse{
				Data: map[string]*pb.InteractionData{
					"clicks": {
						ClickEvents: []*pb.ClickEvent{
							{CatalogId: 100, ProductId: 200, Timestamp: 1704067200000},
						},
					},
				},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.ConvertFromRetrieveInteractionsResponse(tt.response)
			assert.Len(t, got.Data, tt.wantLen)
		})
	}
}
