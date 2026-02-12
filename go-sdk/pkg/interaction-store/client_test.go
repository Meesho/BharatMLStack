package interactionstore

import (
	"context"
	"errors"
	"testing"

	pb "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/interaction-store/timeseries"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockInteractionStoreClient is a mock implementation of InteractionStoreTimeSeriesServiceClient
type MockInteractionStoreClient struct {
	mock.Mock
}

func (m *MockInteractionStoreClient) PersistClickData(ctx context.Context, in *pb.PersistClickDataRequest, opts ...grpc.CallOption) (*pb.PersistDataResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.PersistDataResponse), args.Error(1)
}

func (m *MockInteractionStoreClient) PersistOrderData(ctx context.Context, in *pb.PersistOrderDataRequest, opts ...grpc.CallOption) (*pb.PersistDataResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.PersistDataResponse), args.Error(1)
}

func (m *MockInteractionStoreClient) RetrieveClickInteractions(ctx context.Context, in *pb.RetrieveDataRequest, opts ...grpc.CallOption) (*pb.RetrieveClickDataResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.RetrieveClickDataResponse), args.Error(1)
}

func (m *MockInteractionStoreClient) RetrieveOrderInteractions(ctx context.Context, in *pb.RetrieveDataRequest, opts ...grpc.CallOption) (*pb.RetrieveOrderDataResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.RetrieveOrderDataResponse), args.Error(1)
}

func (m *MockInteractionStoreClient) RetrieveInteractions(ctx context.Context, in *pb.RetrieveInteractionsRequest, opts ...grpc.CallOption) (*pb.RetrieveInteractionsResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.RetrieveInteractionsResponse), args.Error(1)
}

func TestNewClientV1(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name: "success",
			config: &Config{
				Host:      "localhost",
				Port:      "50051",
				DeadLine:  1000,
				PlainText: true,
				CallerId:  "test-caller",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewClientV1(tt.config, nil, nil)
			assert.Equal(t, tt.config.CallerId, got.callerId)
			assert.NotNil(t, got.grpcClient)
			assert.NotNil(t, got.client)
		})
	}
}

func TestClientV1_PersistClickData(t *testing.T) {
	mockClient := &MockInteractionStoreClient{}
	client := &ClientV1{
		grpcClient: &GRPCClient{DeadLine: 1000},
		client:     mockClient,
		adapter:    Adapter{},
		callerId:   "test-caller",
	}

	tests := []struct {
		name       string
		request    *PersistClickDataRequest
		mockResult *pb.PersistDataResponse
		mockError  error
		wantErr    bool
	}{
		{
			name: "success",
			request: &PersistClickDataRequest{
				UserId: "user123",
				Data: []ClickData{
					{CatalogId: 100, ProductId: 200, Timestamp: 1704067200000},
				},
			},
			mockResult: &pb.PersistDataResponse{Message: "success"},
			mockError:  nil,
			wantErr:    false,
		},
		{
			name: "server error",
			request: &PersistClickDataRequest{
				UserId: "user123",
				Data: []ClickData{
					{CatalogId: 100, ProductId: 200, Timestamp: 1704067200000},
				},
			},
			mockResult: nil,
			mockError:  errors.New("server error"),
			wantErr:    true,
		},
		{
			name:       "nil request",
			request:    nil,
			mockResult: nil,
			mockError:  nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.request != nil {
				mockClient.On("PersistClickData", mock.Anything, mock.Anything, mock.Anything).
					Return(tt.mockResult, tt.mockError).Once()
			}

			result, err := client.PersistClickData(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.mockResult.Message, result.Message)
			}
		})
	}
}

func TestClientV1_PersistOrderData(t *testing.T) {
	mockClient := &MockInteractionStoreClient{}
	client := &ClientV1{
		grpcClient: &GRPCClient{DeadLine: 1000},
		client:     mockClient,
		adapter:    Adapter{},
		callerId:   "test-caller",
	}

	tests := []struct {
		name       string
		request    *PersistOrderDataRequest
		mockResult *pb.PersistDataResponse
		mockError  error
		wantErr    bool
	}{
		{
			name: "success",
			request: &PersistOrderDataRequest{
				UserId: "user123",
				Data: []OrderData{
					{CatalogId: 100, ProductId: 200, SubOrderNum: "SUB001", Timestamp: 1704067200000},
				},
			},
			mockResult: &pb.PersistDataResponse{Message: "success"},
			mockError:  nil,
			wantErr:    false,
		},
		{
			name: "server error",
			request: &PersistOrderDataRequest{
				UserId: "user123",
				Data: []OrderData{
					{CatalogId: 100, ProductId: 200, SubOrderNum: "SUB001", Timestamp: 1704067200000},
				},
			},
			mockResult: nil,
			mockError:  errors.New("server error"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.On("PersistOrderData", mock.Anything, mock.Anything, mock.Anything).
				Return(tt.mockResult, tt.mockError).Once()

			result, err := client.PersistOrderData(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestClientV1_RetrieveClickInteractions(t *testing.T) {
	mockClient := &MockInteractionStoreClient{}
	client := &ClientV1{
		grpcClient: &GRPCClient{DeadLine: 1000},
		client:     mockClient,
		adapter:    Adapter{},
		callerId:   "test-caller",
	}

	tests := []struct {
		name       string
		request    *RetrieveDataRequest
		mockResult *pb.RetrieveClickDataResponse
		mockError  error
		wantErr    bool
	}{
		{
			name: "success",
			request: &RetrieveDataRequest{
				UserId:         "user123",
				StartTimestamp: 1704067200000,
				EndTimestamp:   1704153600000,
				Limit:          100,
			},
			mockResult: &pb.RetrieveClickDataResponse{
				Data: []*pb.ClickEvent{
					{CatalogId: 100, ProductId: 200, Timestamp: 1704067200000},
				},
			},
			mockError: nil,
			wantErr:   false,
		},
		{
			name: "server error",
			request: &RetrieveDataRequest{
				UserId:         "user123",
				StartTimestamp: 1704067200000,
				EndTimestamp:   1704153600000,
				Limit:          100,
			},
			mockResult: nil,
			mockError:  errors.New("server error"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.On("RetrieveClickInteractions", mock.Anything, mock.Anything, mock.Anything).
				Return(tt.mockResult, tt.mockError).Once()

			result, err := client.RetrieveClickInteractions(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result.Data, len(tt.mockResult.Data))
			}
		})
	}
}

func TestClientV1_RetrieveOrderInteractions(t *testing.T) {
	mockClient := &MockInteractionStoreClient{}
	client := &ClientV1{
		grpcClient: &GRPCClient{DeadLine: 1000},
		client:     mockClient,
		adapter:    Adapter{},
		callerId:   "test-caller",
	}

	tests := []struct {
		name       string
		request    *RetrieveDataRequest
		mockResult *pb.RetrieveOrderDataResponse
		mockError  error
		wantErr    bool
	}{
		{
			name: "success",
			request: &RetrieveDataRequest{
				UserId:         "user123",
				StartTimestamp: 1704067200000,
				EndTimestamp:   1704153600000,
				Limit:          100,
			},
			mockResult: &pb.RetrieveOrderDataResponse{
				Data: []*pb.OrderEvent{
					{CatalogId: 100, ProductId: 200, SubOrderNum: "SUB001", Timestamp: 1704067200000},
				},
			},
			mockError: nil,
			wantErr:   false,
		},
		{
			name: "server error",
			request: &RetrieveDataRequest{
				UserId:         "user123",
				StartTimestamp: 1704067200000,
				EndTimestamp:   1704153600000,
				Limit:          100,
			},
			mockResult: nil,
			mockError:  errors.New("server error"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.On("RetrieveOrderInteractions", mock.Anything, mock.Anything, mock.Anything).
				Return(tt.mockResult, tt.mockError).Once()

			result, err := client.RetrieveOrderInteractions(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestClientV1_RetrieveInteractions(t *testing.T) {
	mockClient := &MockInteractionStoreClient{}
	client := &ClientV1{
		grpcClient: &GRPCClient{DeadLine: 1000},
		client:     mockClient,
		adapter:    Adapter{},
		callerId:   "test-caller",
	}

	tests := []struct {
		name       string
		request    *RetrieveInteractionsRequest
		mockResult *pb.RetrieveInteractionsResponse
		mockError  error
		wantErr    bool
	}{
		{
			name: "success",
			request: &RetrieveInteractionsRequest{
				UserId:           "user123",
				InteractionTypes: []InteractionType{InteractionTypeClick, InteractionTypeOrder},
				StartTimestamp:   1704067200000,
				EndTimestamp:     1704153600000,
				Limit:            100,
			},
			mockResult: &pb.RetrieveInteractionsResponse{
				Data: map[string]*pb.InteractionData{
					"clicks": {
						ClickEvents: []*pb.ClickEvent{
							{CatalogId: 100, ProductId: 200, Timestamp: 1704067200000},
						},
					},
				},
			},
			mockError: nil,
			wantErr:   false,
		},
		{
			name: "server error",
			request: &RetrieveInteractionsRequest{
				UserId:           "user123",
				InteractionTypes: []InteractionType{InteractionTypeClick},
				StartTimestamp:   1704067200000,
				EndTimestamp:     1704153600000,
				Limit:            100,
			},
			mockResult: nil,
			mockError:  errors.New("server error"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.On("RetrieveInteractions", mock.Anything, mock.Anything, mock.Anything).
				Return(tt.mockResult, tt.mockError).Once()

			result, err := client.RetrieveInteractions(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		shouldPanic bool
	}{
		{
			name:        "nil config",
			config:      nil,
			shouldPanic: true,
		},
		{
			name: "empty host",
			config: &Config{
				Host:     "",
				Port:     "50051",
				CallerId: "test",
			},
			shouldPanic: true,
		},
		{
			name: "empty port",
			config: &Config{
				Host:     "localhost",
				Port:     "",
				CallerId: "test",
			},
			shouldPanic: true,
		},
		{
			name: "empty caller id",
			config: &Config{
				Host:     "localhost",
				Port:     "50051",
				CallerId: "",
			},
			shouldPanic: true,
		},
		{
			name: "valid config",
			config: &Config{
				Host:     "localhost",
				Port:     "50051",
				CallerId: "test",
			},
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				assert.Panics(t, func() { validateConfig(tt.config) })
			} else {
				assert.NotPanics(t, func() { validateConfig(tt.config) })
			}
		})
	}
}
