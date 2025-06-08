package onfs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/onfs/persist"
	"github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/onfs/retrieve"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"
)

const (
	// Header keys for authentication
	headerCallerID    = "ONLINE-FEATURE-STORE-CALLER-ID"
	headerCallerToken = "ONLINE-FEATURE-STORE-AUTH-TOKEN"

	// Default configuration values
	defaultBatchSize = 50
)

type Client interface {
	RetrieveFeatures(ctx context.Context, request *Query) (response *Result, err error)
	RetrieveDecodedFeatures(ctx context.Context, request *Query) (response *DecodedResult, err error)
	PersistFeatures(ctx context.Context, request *PersistFeaturesRequest) (response *PersistFeaturesResponse, err error)
}

// ClientV1 represents version 1 of the online-feature-store client
type ClientV1 struct {
	v1Client    *clientConfig
	adapter     Adapter
	batchSize   int
	callerId    string
	callerToken string
}

// clientConfig holds the gRPC client configuration
type clientConfig struct {
	client        retrieve.FeatureServiceClient
	persistClient persist.FeatureServiceClient
	deadline      int64
}

// NewClientV1 creates a new instance of the online-feature-store client (v1)
func NewClientV1(config *Config, timing func(name string, value time.Duration, tags []string), count func(name string, value int64, tags []string)) *ClientV1 {
	validateConfig(config)

	batchSize := defaultBatchSize
	if config.BatchSize > 0 {
		batchSize = config.BatchSize
	}

	conn := NewConnFromConfig(config, "online-feature-store", timing, count)

	return &ClientV1{
		v1Client: &clientConfig{
			client:        retrieve.NewFeatureServiceClient(conn),
			persistClient: persist.NewFeatureServiceClient(conn),
			deadline:      conn.DeadLine,
		},
		adapter:     Adapter{},
		batchSize:   batchSize,
		callerId:    config.CallerId,
		callerToken: config.CallerToken,
	}
}

// RetrieveFeatures retrieves features from online-feature-store service
func (c *ClientV1) RetrieveFeatures(ctx context.Context, request *Query) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	requests, err := c.adapter.ConvertToQueriesProto(request, c.batchSize)
	if err != nil {
		return nil, err
	}

	entityResult := c.fetchResponseFromServer(ctx, request, requests)
	if entityResult == nil {
		return nil, errors.New("error on retrieving features from online-feature-store")
	}
	return entityResult, nil
}

// RetrieveDecodedFeatures retrieves decoded features from online-feature-store service
func (c *ClientV1) RetrieveDecodedFeatures(ctx context.Context, request *Query) (*DecodedResult, error) {
	var backgroundContext context.Context
	if ctx == nil {
		backgroundContext = context.Background()
	}

	requests, err := c.adapter.ConvertToQueriesProto(request, c.batchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to convert query: %w", err)
	}

	decodedResult := c.fetchDecodedResponseFromServer(backgroundContext, request, requests)
	if decodedResult == nil {
		return nil, errors.New("error on retrieving decoded features from online-feature-store")
	}
	return decodedResult, nil
}

func (c *ClientV1) PersistFeatures(ctx context.Context, request *PersistFeaturesRequest) (response *PersistFeaturesResponse, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Convert request to proto format using adapter
	protoRequest := c.adapter.ConvertToPersistRequest(request)
	if protoRequest == nil {
		return nil, errors.New("failed to convert persist request to proto format")
	}

	protoResponse, err := c.contactPersistServer(ctx, protoRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to persist features: %w", err)
	}

	return &PersistFeaturesResponse{
		Message: protoResponse.Message,
	}, nil
}

// fetchResponseFromServer fetches responses from the server in parallel
func (c *ClientV1) fetchResponseFromServer(ctx context.Context, request *Query, requests []*retrieve.Query) *Result {
	responseChan := make(chan *Response, len(requests))
	var wg sync.WaitGroup

	for _, req := range requests {
		wg.Add(1)
		go func(req *retrieve.Query) {
			defer wg.Done()
			c.contactServer(ctx, req, responseChan)
		}(req)
	}

	// Close channel after all goroutines complete
	go func() {
		wg.Wait()
		close(responseChan)
	}()

	return c.handleResponsesFromChannel(request.EntityLabel, responseChan, len(requests))
}

// handleResponsesFromChannel processes responses from the channel
func (c *ClientV1) handleResponsesFromChannel(label string, responseChan chan *Response, batch int) *Result {
	result := &Result{
		EntityLabel:    label,
		KeysSchema:     make([]string, 0),
		FeatureSchemas: make([]FeatureSchema, 0),
		Rows:           make([]Row, 0),
	}

	for response := range responseChan {
		if len(response.Err) != 0 {
			log.Error().Msg(response.Err)
			continue
		}

		entityResult := c.adapter.ConvertToResult(response.Resp)
		if len(entityResult.KeysSchema) != 0 {
			result.KeysSchema = entityResult.KeysSchema
		}
		if len(entityResult.FeatureSchemas) != 0 {
			result.FeatureSchemas = entityResult.FeatureSchemas
		}
		result.Rows = append(result.Rows, entityResult.Rows...)
	}

	if len(result.Rows) == 0 {
		return nil
	}
	return result
}

// contactServer makes a gRPC call to retrieve features
func (c *ClientV1) contactServer(ctx context.Context, request *retrieve.Query, responseChan chan<- *Response) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("Recovered from panic in contactServer")
			responseChan <- &Response{
				Err: fmt.Sprintf("Panic occurred: %v", r),
			}
		}
	}()

	ctx = c.withAuthMetadata(ctx)
	response, err := c.v1Client.client.RetrieveFeatures(ctx, request)

	if err != nil {
		responseChan <- &Response{
			Err:  fmt.Sprintf("error retrieving features: %v", err),
			Resp: nil,
		}
		return
	}

	if response == nil {
		responseChan <- &Response{
			Err:  "empty response from server",
			Resp: nil,
		}
		return
	}

	responseChan <- &Response{
		Err:  "",
		Resp: response,
	}
}

// withAuthMetadata adds authentication metadata to context
func (c *ClientV1) withAuthMetadata(ctx context.Context) context.Context {
	md := metadata.New(map[string]string{
		headerCallerID:    c.callerId,
		headerCallerToken: c.callerToken,
	})
	return metadata.NewOutgoingContext(ctx, md)
}

// withTimeout adds timeout to context
func (c *ClientV1) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := time.Duration(c.v1Client.deadline) * time.Millisecond
	return context.WithTimeout(ctx, timeout)
}

// contactPersistServer makes a gRPC call to persist features
func (c *ClientV1) contactPersistServer(ctx context.Context, request *persist.Query) (*persist.Result, error) {
	ctx = c.withAuthMetadata(ctx)
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	response, err := c.v1Client.persistClient.PersistFeatures(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to persist features: %w", err)
	}

	if response == nil {
		return nil, errors.New("empty response from server")
	}

	return response, nil
}

func (c *ClientV1) fetchDecodedResponseFromServer(ctx context.Context, request *Query, requests []*retrieve.Query) *DecodedResult {
	responseChan := make(chan *DecodedResponse, len(requests))
	defer close(responseChan)
	for _, req := range requests {
		go c.contactDecodedServer(ctx, req, responseChan)
	}

	decodedResult := c.handleDecodedResponsesFromChannel(request.EntityLabel, responseChan, len(requests))

	return decodedResult
}

func (c *ClientV1) handleDecodedResponsesFromChannel(_ string, responseChan chan *DecodedResponse, batch int) *DecodedResult {
	result := &DecodedResult{
		KeysSchema:     make([]string, 0),
		FeatureSchemas: make([]FeatureSchema, 0),
		Rows:           make([]DecodedRow, 0),
	}

	for i := 0; i < batch; i++ {
		response := <-responseChan
		if len(response.Err) != 0 {

			continue
		}
		decodedResult := c.adapter.ConvertToDecodedResult(response.Resp)
		if len(decodedResult.KeysSchema) != 0 {
			result.KeysSchema = decodedResult.KeysSchema
		}
		if len(decodedResult.FeatureSchemas) != 0 {
			result.FeatureSchemas = decodedResult.FeatureSchemas
		}
		result.Rows = append(result.Rows, decodedResult.Rows...)
	}
	if len(result.Rows) == 0 {
		return nil
	}
	return result
}

func (c *ClientV1) contactDecodedServer(
	ctx context.Context,
	request *retrieve.Query,
	responseChan chan *DecodedResponse,
) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("Recovered from panic in fetchDecodedResponseFromServer goroutine")
			responseChan <- &DecodedResponse{
				Err: fmt.Sprintf("Panic occurred: %v", r),
			}
		}
	}()
	if ctx == nil {
		ctx = context.Background()
	}

	var ctxWithTimeout context.Context
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctxWithTimeout, cancel = c.withTimeout(ctx)
		defer cancel()
	} else {
		ctxWithTimeout = ctx
	}

	//set metadata
	finalCtx := c.withAuthMetadata(ctxWithTimeout)

	//Contact server
	response, err := c.v1Client.client.RetrieveDecodedResult(finalCtx, request)

	//Dispatch response
	if err != nil {
		res := &DecodedResponse{
			Err:  "error in getting decoded features from online-feature-store",
			Resp: nil,
		}
		log.Error().Msg(res.Err)
		responseChan <- res
	} else if response == nil {
		res := &DecodedResponse{
			Err:  "empty response from online-feature-store",
			Resp: nil,
		}
		log.Error().Msg(res.Err)
		responseChan <- res
	} else {
		res := &DecodedResponse{
			Err:  "",
			Resp: response,
		}
		responseChan <- res
	}
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
	if len(config.CallerToken) == 0 {
		panic("Configuration error: Caller token is empty. Please provide a valid caller token.")
	}
}
