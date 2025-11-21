package numerix

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/clients/numerix/client/grpc"
	"github.com/Meesho/go-core/datatypeconverter/byteorder"
	"github.com/Meesho/go-core/grpcclient"
	"github.com/rs/zerolog/log"
	metadata "google.golang.org/grpc/metadata"
)

type ClientV1 struct {
	ClientConfigs *ClientConfig
	GrpcClient    *grpcclient.GRPCClient
	numerixClient grpc.NumerixClient
	Adapter       Adapter
}

var (
	client  *ClientV1
	once    sync.Once
	headers metadata.MD
)

const (
	V1Prefix          = "numerix_CLIENT_V1_"
	numerix_CALLER_ID = "numerix-CALLER-ID"
)

func InitV1Client(configBytes []byte) NumerixClient {
	if client == nil {
		once.Do(func() {
			byteorder.Init()

			clientConfig, err := getClientConfigs(configBytes)
			if err != nil {
				log.Panic().Err(err).Msgf("Invalid numerix client configs: %#v", clientConfig)
			}
			headers = metadata.New(map[string]string{
				numerix_CALLER_ID: clientConfig.CallerId,
			})

			grpcClient, grpcErr := getGrpcClient(clientConfig)
			if grpcErr != nil {
				log.Panic().Err(grpcErr).Msgf("Error creating numerix service grpc client, client: %#v", grpcClient)
			}

			numerixClient := grpc.NewNumerixClient(grpcClient)
			client = &ClientV1{
				ClientConfigs: clientConfig,
				GrpcClient:    grpcClient,
				numerixClient: numerixClient,
				Adapter:       Adapter{},
			}
		})
	}
	return client
}

func getGrpcClient(conf *ClientConfig) (*grpcclient.GRPCClient, error) {
	var client *grpcclient.GRPCClient
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic creating grpc client from prefix: %v", r)
		}
	}()
	client = grpcclient.NewConnFromConfig(&grpcclient.Config{
		Host:                conf.Host,
		Port:                conf.Port,
		DeadLine:            conf.DeadlineExceedMS,
		LoadBalancingPolicy: "round_robin",
		PlainText:           conf.PlainText,
	}, V1Prefix)
	return client, err
}

type BatchInfo struct {
	StartIndex int
	EndIndex   int
}

type batchResult struct {
	index    int
	response *grpc.NumerixResponseProto
	err      error
}

func (c *ClientV1) RetrieveScore(req *NumerixRequest) (*NumerixResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("numerix request cannot be nil")
	}

	err := validateRequest(req)
	if err != nil {
		return nil, err
	}

	numInputs := len(req.EntityScoreData.Data)
	batchSize := c.ClientConfigs.BatchSize

	protoReq := c.Adapter.MapRequestToProto(req)
	if protoReq == nil {
		return nil, fmt.Errorf("failed to map request to proto")
	}

	// If data is smaller than batch size, process directly
	if numInputs <= batchSize {
		return c.processRequest(protoReq)
	}

	batches := c.getBatchIndices(numInputs, batchSize)
	numBatches := len(batches)

	resultChan := make(chan batchResult, numBatches)

	finalResponse := &NumerixResponse{
		ComputationScoreData: ComputationScoreData{
			Schema:     protoReq.EntityScoreData.Schema,
			Data:       make([][][]byte, numInputs),
			StringData: make([][]string, 0),
		},
	}

	wg := sync.WaitGroup{}
	// Process each batch in parallel
	for i, batch := range batches {
		wg.Add(1)
		go func(batch BatchInfo, batchIndex int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Warn().
						Interface("panic", r).
						Int("batch_index", batchIndex).
						Msg("Panic occurred while processing numerix batch")
					resultChan <- batchResult{index: batchIndex, response: nil, err: fmt.Errorf("panic in batch processing: %v", r)}
				}
			}()

			protoResp, err := c.processBatchByIndex(protoReq, batch, batchIndex)
			if err != nil {
				resultChan <- batchResult{index: batchIndex, response: nil, err: err}
				return
			}

			resultChan <- batchResult{index: batchIndex, response: protoResp, err: nil}
		}(batch, i)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		if result.err != nil {
			log.Warn().Err(result.err).Msgf("Error processing batch %d", result.index)
		}
		batch := batches[result.index]
		c.fillResponseFromBatch(finalResponse, result.response, batch)
	}

	return finalResponse, nil
}

func (c *ClientV1) fillResponseFromBatch(finalResponse *NumerixResponse, batchProtoResp *grpc.NumerixResponseProto, batch BatchInfo) {
	if errProto := batchProtoResp.GetError(); errProto != nil {
		log.Warn().Msgf("Received error in proto response error: %s", errProto.GetMessage())
		return
	}

	compDataProto := batchProtoResp.GetComputationScoreData()
	if compDataProto == nil {
		return
	}

	computationScores := compDataProto.GetComputationScores()
	batchSize := batch.EndIndex - batch.StartIndex

	for i, scoreProto := range computationScores {
		if i >= batchSize {
			break
		}

		targetIndex := batch.StartIndex + i
		switch T := scoreProto.GetMatrixFormat().(type) {
		case *grpc.Score_ByteData:
			if T.ByteData != nil && T.ByteData.GetValues() != nil {
				finalResponse.ComputationScoreData.Data[targetIndex] = T.ByteData.GetValues()
			}

		case *grpc.Score_StringData:
			if T.StringData != nil && T.StringData.GetValues() != nil {
				finalResponse.ComputationScoreData.StringData = append(finalResponse.ComputationScoreData.StringData, T.StringData.GetValues())
			}
		}

	}
}

func (c *ClientV1) getBatchIndices(numElements, batchSize int) []BatchInfo {
	numBatches := (numElements + batchSize - 1) / batchSize
	batches := make([]BatchInfo, numBatches)

	for i := 0; i < numElements; i += batchSize {
		end := i + batchSize
		if end > numElements {
			end = numElements
		}
		batches[i/batchSize] = BatchInfo{i, end}
	}
	return batches
}

func (c *ClientV1) processBatchByIndex(protoReq *grpc.NumerixRequestProto, batch BatchInfo, batchIndex int) (*grpc.NumerixResponseProto, error) {
	batchReq := &grpc.NumerixRequestProto{
		EntityScoreData: &grpc.EntityScoreData{
			ComputeId:    protoReq.EntityScoreData.ComputeId,
			DataType:     protoReq.EntityScoreData.DataType,
			Schema:       protoReq.EntityScoreData.Schema,
			EntityScores: protoReq.EntityScoreData.EntityScores[batch.StartIndex:batch.EndIndex],
		},
	}

	response, err := c.callGrpcService(batchReq)
	if err != nil {
		log.Warn().Err(err).
			Int("batch_index", batchIndex).
			Str("compute_id", batchReq.EntityScoreData.ComputeId).
			Msg("Failed to get score from numerix service")
		return nil, err
	}

	return response, nil
}

func (c *ClientV1) processRequest(protoReq *grpc.NumerixRequestProto) (*NumerixResponse, error) {
	response, err := c.callGrpcService(protoReq)
	if err != nil {
		return nil, err
	}

	resp := c.Adapter.MapProtoToResponse(response)
	if resp == nil {
		return nil, fmt.Errorf("failed to map proto to response")
	}
	return resp, nil
}

func validateRequest(req *NumerixRequest) error {
	if len(req.EntityScoreData.Schema) == 0 {
		return fmt.Errorf("schema is required")
	}
	if (req.EntityScoreData.Data == nil && req.EntityScoreData.StringData == nil) || (len(req.EntityScoreData.Data) == 0 && len(req.EntityScoreData.StringData) == 0) {
		return fmt.Errorf("data is required")
	}
	if req.EntityScoreData.ComputeID == "" {
		return fmt.Errorf("compute_id is required")
	}
	if req.EntityScoreData.DataType == "" {
		return fmt.Errorf("data_type is required")
	}
	return nil
}

func (c *ClientV1) callGrpcService(req *grpc.NumerixRequestProto) (*grpc.NumerixResponseProto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.ClientConfigs.DeadlineExceedMS)*time.Millisecond)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, headers)
	response, err := c.numerixClient.Compute(ctx, req)
	if err != nil {
		return nil, err
	}
	return response, nil
}
