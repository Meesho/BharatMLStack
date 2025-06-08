package onfs

import (
	"context"
	"errors"
	"testing"

	"github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/onfs/persist"
	pb "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/onfs/retrieve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type mockRetrieveServiceClient struct {
	mock.Mock
}

func (m *mockRetrieveServiceClient) RetrieveFeatures(ctx context.Context, in *pb.Query, opts ...grpc.CallOption) (*pb.Result, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.Result), args.Error(1)
}

func (m *mockRetrieveServiceClient) RetrieveDecodedResult(ctx context.Context, in *pb.Query, opts ...grpc.CallOption) (*pb.DecodedResult, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.DecodedResult), args.Error(1)
}

// MockFeatureServiceClient is a mock implementation of persist.FeatureServiceClient
type MockFeatureServiceClient struct {
	mock.Mock
}

func (m *MockFeatureServiceClient) PersistFeatures(ctx context.Context, in *persist.Query, opts ...grpc.CallOption) (*persist.Result, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*persist.Result), args.Error(1)
}

func TestNewClientV1(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   *ClientV1
	}{
		{
			name: "success",
			config: &Config{
				Host:        "localhost",
				Port:        "50051",
				DeadLine:    1000,
				PlainText:   true,
				BatchSize:   100,
				CallerId:    "test-caller",
				CallerToken: "test-token",
			},
			want: &ClientV1{
				batchSize:   100,
				callerId:    "test-caller",
				callerToken: "test-token",
			},
		},
		{
			name: "default batch size",
			config: &Config{
				Host:        "localhost",
				Port:        "50051",
				DeadLine:    1000,
				PlainText:   true,
				CallerId:    "test-caller",
				CallerToken: "test-token",
			},
			want: &ClientV1{
				batchSize:   50, // default value
				callerId:    "test-caller",
				callerToken: "test-token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewClientV1(tt.config, nil, nil)
			assert.Equal(t, tt.want.batchSize, got.batchSize)
			assert.Equal(t, tt.want.callerId, got.callerId)
			assert.Equal(t, tt.want.callerToken, got.callerToken)
			assert.NotNil(t, got.v1Client)
			assert.NotNil(t, got.adapter)
		})
	}
}

func TestClientV1_RetrieveFeatures(t *testing.T) {
	mockClient := &mockRetrieveServiceClient{}
	client := &ClientV1{
		v1Client: &clientConfig{
			client:   mockClient,
			deadline: 1000,
		},
		adapter:     Adapter{},
		batchSize:   2,
		callerId:    "test-caller",
		callerToken: "test-token",
	}

	tests := []struct {
		name       string
		request    *Query
		mockResult *pb.Result
		mockError  error
		wantErr    bool
		validateFn func(*testing.T, *Result)
	}{
		{
			name: "success with multiple feature groups",
			request: &Query{
				EntityLabel: "test_entity",
				Keys: []Keys{
					{Cols: []string{"key1", "key2"}},
				},
				KeysSchema: []string{"id", "type"},
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
			mockResult: &pb.Result{
				KeysSchema: []string{"id", "type"},
				FeatureSchemas: []*pb.FeatureSchema{
					{
						FeatureGroupLabel: "derived_string",
						Features: []*pb.Feature{
							{Label: "feature1", ColumnIdx: 0},
							{Label: "feature2", ColumnIdx: 1},
						},
					},
					{
						FeatureGroupLabel: "derived_int64",
						Features: []*pb.Feature{
							{Label: "feature3", ColumnIdx: 2},
							{Label: "feature4", ColumnIdx: 3},
						},
					},
				},
				Rows: []*pb.Row{
					{
						Keys:    []string{"key1", "key2"},
						Columns: [][]byte{[]byte("value1"), []byte("value2"), []byte("123"), []byte("456")},
					},
				},
			},
			mockError: nil,
			wantErr:   false,
			validateFn: func(t *testing.T, result *Result) {
				assert.Equal(t, []string{"id", "type"}, result.KeysSchema)
				assert.Len(t, result.FeatureSchemas, 2)
				assert.Equal(t, "derived_string", result.FeatureSchemas[0].FeatureGroupLabel)
				assert.Equal(t, "derived_int64", result.FeatureSchemas[1].FeatureGroupLabel)
				assert.Len(t, result.Rows, 1)
				assert.Equal(t, []string{"key1", "key2"}, result.Rows[0].Keys)
				assert.Equal(t, [][]byte{[]byte("value1"), []byte("value2"), []byte("123"), []byte("456")}, result.Rows[0].Columns)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.On("RetrieveFeatures", mock.Anything, mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockError)

			result, err := client.RetrieveFeatures(context.Background(), tt.request)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			tt.validateFn(t, result)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestClientV1_RetrieveDecodedFeatures(t *testing.T) {
	mockClient := &mockRetrieveServiceClient{}
	client := &ClientV1{
		v1Client: &clientConfig{
			client:   mockClient,
			deadline: 1000,
		},
		adapter:     Adapter{},
		batchSize:   2,
		callerId:    "test-caller",
		callerToken: "test-token",
	}

	tests := []struct {
		name       string
		request    *Query
		mockResult *pb.DecodedResult
		mockError  error
		wantErr    bool
		validateFn func(*testing.T, *DecodedResult)
	}{
		{
			name: "success with multiple feature groups",
			request: &Query{
				EntityLabel: "test_entity",
				Keys: []Keys{
					{Cols: []string{"key1", "key2"}},
				},
				KeysSchema: []string{"id", "type"},
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
			mockResult: &pb.DecodedResult{
				KeysSchema: []string{"id", "type"},
				FeatureSchemas: []*pb.FeatureSchema{
					{
						FeatureGroupLabel: "derived_string",
						Features: []*pb.Feature{
							{Label: "feature1", ColumnIdx: 0},
							{Label: "feature2", ColumnIdx: 1},
						},
					},
					{
						FeatureGroupLabel: "derived_int64",
						Features: []*pb.Feature{
							{Label: "feature3", ColumnIdx: 2},
							{Label: "feature4", ColumnIdx: 3},
						},
					},
				},
				Rows: []*pb.DecodedRow{
					{
						Keys:    []string{"key1", "key2"},
						Columns: []string{"value1", "value2", "123", "456"},
					},
				},
			},
			mockError: nil,
			wantErr:   false,
			validateFn: func(t *testing.T, result *DecodedResult) {
				assert.Equal(t, []string{"id", "type"}, result.KeysSchema)
				assert.Len(t, result.FeatureSchemas, 2)
				assert.Equal(t, "derived_string", result.FeatureSchemas[0].FeatureGroupLabel)
				assert.Equal(t, "derived_int64", result.FeatureSchemas[1].FeatureGroupLabel)
				assert.Len(t, result.Rows, 1)
				assert.Equal(t, []string{"key1", "key2"}, result.Rows[0].Keys)
				assert.Equal(t, []string{"value1", "value2", "123", "456"}, result.Rows[0].Columns)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.On("RetrieveDecodedResult", mock.Anything, mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockError)

			result, err := client.RetrieveDecodedFeatures(context.Background(), tt.request)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			tt.validateFn(t, result)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestPersistFeatures(t *testing.T) {
	tests := []struct {
		name          string
		request       *PersistFeaturesRequest
		mockResponse  *persist.Result
		mockError     error
		wantResponse  *PersistFeaturesResponse
		wantError     bool
		errorContains string
	}{
		{
			name: "successful persist",
			request: &PersistFeaturesRequest{
				EntityLabel: "user_sscat",
				KeysSchema:  []string{"user_id", "sscat_id"},
				FeatureGroups: []FeatureGroupSchema{
					{
						Label:         "derived_int64",
						FeatureLabels: []string{"feature1", "feature2"},
					},
				},
				Data: []Data{
					{
						KeyValues: []string{"123", "456"},
						FeatureValues: []FeatureValues{
							{
								Values: Values{
									Int64Values: []int64{1, 2},
								},
							},
						},
					},
				},
			},
			mockResponse: &persist.Result{
				Message: "Features persisted successfully",
			},
			mockError: nil,
			wantResponse: &PersistFeaturesResponse{
				Message: "Features persisted successfully",
			},
			wantError: false,
		},
		{
			name:    "nil request",
			request: nil,
			mockResponse: &persist.Result{
				Message: "Features persisted successfully",
			},
			mockError:     nil,
			wantResponse:  nil,
			wantError:     true,
			errorContains: "failed to convert persist request to proto format",
		},
		{
			name: "server error",
			request: &PersistFeaturesRequest{
				EntityLabel: "user_sscat",
				KeysSchema:  []string{"user_id", "sscat_id"},
				FeatureGroups: []FeatureGroupSchema{
					{
						Label:         "derived_int64",
						FeatureLabels: []string{"feature1", "feature2"},
					},
				},
				Data: []Data{
					{
						KeyValues: []string{"123", "456"},
						FeatureValues: []FeatureValues{
							{
								Values: Values{
									Int64Values: []int64{1, 2},
								},
							},
						},
					},
				},
			},
			mockResponse:  nil,
			mockError:     errors.New("server error"),
			wantResponse:  nil,
			wantError:     true,
			errorContains: "server error",
		},
		{
			name: "empty response from server",
			request: &PersistFeaturesRequest{
				EntityLabel: "user_sscat",
				KeysSchema:  []string{"user_id", "sscat_id"},
				FeatureGroups: []FeatureGroupSchema{
					{
						Label:         "derived_int64",
						FeatureLabels: []string{"feature1", "feature2"},
					},
				},
				Data: []Data{
					{
						KeyValues: []string{"123", "456"},
						FeatureValues: []FeatureValues{
							{
								Values: Values{
									Int64Values: []int64{1, 2},
								},
							},
						},
					},
				},
			},
			mockResponse:  nil,
			mockError:     nil,
			wantResponse:  nil,
			wantError:     true,
			errorContains: "empty response from server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := new(MockFeatureServiceClient)

			// Set up expectations
			if tt.request != nil {
				mockClient.On("PersistFeatures", mock.Anything, mock.Anything, mock.Anything).
					Return(tt.mockResponse, tt.mockError)
			}

			// Create client with mock
			client := &ClientV1{
				v1Client: &clientConfig{
					persistClient: mockClient,
					deadline:      5000,
				},
				adapter:     Adapter{},
				callerId:    "test-caller",
				callerToken: "test-token",
			}

			// Call the method
			got, err := client.PersistFeatures(context.Background(), tt.request)

			// Check results
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantResponse, got)

			// Verify mock expectations
			mockClient.AssertExpectations(t)
		})
	}
}
