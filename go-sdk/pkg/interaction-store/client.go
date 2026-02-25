package interactionstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	pb "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/interaction-store/timeseries"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"
)

const (
	headerCallerID = "interaction-store-caller-id"
)

// Client defines the interface for interaction-store operations
type Client interface {
	// PersistClickData persists click interaction data
	PersistClickData(ctx context.Context, request *PersistClickDataRequest) (*PersistDataResponse, error)
	// PersistOrderData persists order interaction data
	PersistOrderData(ctx context.Context, request *PersistOrderDataRequest) (*PersistDataResponse, error)
	// RetrieveClickInteractions retrieves click interactions for a user
	RetrieveClickInteractions(ctx context.Context, request *RetrieveDataRequest) (*RetrieveClickDataResponse, error)
	// RetrieveOrderInteractions retrieves order interactions for a user
	RetrieveOrderInteractions(ctx context.Context, request *RetrieveDataRequest) (*RetrieveOrderDataResponse, error)
	// RetrieveInteractions retrieves multiple interaction types for a user
	RetrieveInteractions(ctx context.Context, request *RetrieveInteractionsRequest) (*RetrieveInteractionsResponse, error)
}

// ClientV1 represents version 1 of the interaction-store client
type ClientV1 struct {
	grpcClient *GRPCClient
	client     pb.InteractionStoreTimeSeriesServiceClient
	adapter    Adapter
	callerId   string
}

// NewClientV1 creates a new instance of the interaction-store client (v1)
func NewClientV1(config *Config, timing func(name string, value time.Duration, tags []string), count func(name string, value int64, tags []string)) *ClientV1 {
	validateConfig(config)

	conn := NewConnFromConfig(config, "interaction-store", timing, count)

	return &ClientV1{
		grpcClient: conn,
		client:     pb.NewInteractionStoreTimeSeriesServiceClient(conn),
		adapter:    Adapter{},
		callerId:   config.CallerId,
	}
}

// PersistClickData persists click interaction data
func (c *ClientV1) PersistClickData(ctx context.Context, request *PersistClickDataRequest) (*PersistDataResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	protoRequest := c.adapter.ConvertToPersistClickDataProto(request)
	if protoRequest == nil {
		return nil, errors.New("failed to convert persist click data request to proto format")
	}

	ctx = c.withAuthMetadata(ctx)
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	response, err := c.client.PersistClickData(ctx, protoRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to persist click data: %w", err)
	}

	if response == nil {
		return nil, errors.New("empty response from server")
	}

	return c.adapter.ConvertFromPersistDataResponse(response), nil
}

// PersistOrderData persists order interaction data
func (c *ClientV1) PersistOrderData(ctx context.Context, request *PersistOrderDataRequest) (*PersistDataResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	protoRequest := c.adapter.ConvertToPersistOrderDataProto(request)
	if protoRequest == nil {
		return nil, errors.New("failed to convert persist order data request to proto format")
	}

	ctx = c.withAuthMetadata(ctx)
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	response, err := c.client.PersistOrderData(ctx, protoRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to persist order data: %w", err)
	}

	if response == nil {
		return nil, errors.New("empty response from server")
	}

	return c.adapter.ConvertFromPersistDataResponse(response), nil
}

// RetrieveClickInteractions retrieves click interactions for a user
func (c *ClientV1) RetrieveClickInteractions(ctx context.Context, request *RetrieveDataRequest) (*RetrieveClickDataResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	protoRequest := c.adapter.ConvertToRetrieveDataProto(request)
	if protoRequest == nil {
		return nil, errors.New("failed to convert retrieve click data request to proto format")
	}

	ctx = c.withAuthMetadata(ctx)
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	response, err := c.client.RetrieveClickInteractions(ctx, protoRequest)
	if err != nil {
		log.Error().Err(err).Msg("failed to retrieve click interactions")
		return nil, fmt.Errorf("failed to retrieve click interactions: %w", err)
	}

	return c.adapter.ConvertFromRetrieveClickDataResponse(response), nil
}

// RetrieveOrderInteractions retrieves order interactions for a user
func (c *ClientV1) RetrieveOrderInteractions(ctx context.Context, request *RetrieveDataRequest) (*RetrieveOrderDataResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	protoRequest := c.adapter.ConvertToRetrieveDataProto(request)
	if protoRequest == nil {
		return nil, errors.New("failed to convert retrieve order data request to proto format")
	}

	ctx = c.withAuthMetadata(ctx)
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	response, err := c.client.RetrieveOrderInteractions(ctx, protoRequest)
	if err != nil {
		log.Error().Err(err).Msg("failed to retrieve order interactions")
		return nil, fmt.Errorf("failed to retrieve order interactions: %w", err)
	}

	return c.adapter.ConvertFromRetrieveOrderDataResponse(response), nil
}

// RetrieveInteractions retrieves multiple interaction types for a user
func (c *ClientV1) RetrieveInteractions(ctx context.Context, request *RetrieveInteractionsRequest) (*RetrieveInteractionsResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	protoRequest := c.adapter.ConvertToRetrieveInteractionsProto(request)
	if protoRequest == nil {
		return nil, errors.New("failed to convert retrieve interactions request to proto format")
	}

	ctx = c.withAuthMetadata(ctx)
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	response, err := c.client.RetrieveInteractions(ctx, protoRequest)
	if err != nil {
		log.Error().Err(err).Msg("failed to retrieve interactions")
		return nil, fmt.Errorf("failed to retrieve interactions: %w", err)
	}

	return c.adapter.ConvertFromRetrieveInteractionsResponse(response), nil
}

// withAuthMetadata adds authentication metadata to context
func (c *ClientV1) withAuthMetadata(ctx context.Context) context.Context {
	md := metadata.New(map[string]string{
		headerCallerID: c.callerId,
	})
	return metadata.NewOutgoingContext(ctx, md)
}

// withTimeout adds timeout to context
func (c *ClientV1) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := time.Duration(c.grpcClient.DeadLine) * time.Millisecond
	return context.WithTimeout(ctx, timeout)
}

func validateConfig(config *Config) {
	if config == nil {
		panic("Configuration is nil. Please provide a valid config.")
	}
	if len(config.Host) == 0 {
		panic("Configuration error: Host is empty. Please provide a valid host.")
	}
	if len(config.Port) == 0 {
		panic("Configuration error: Port is empty. Please provide a valid port.")
	}
	if len(config.CallerId) == 0 {
		panic("Configuration error: Caller ID is empty. Please provide a valid caller ID.")
	}
}
