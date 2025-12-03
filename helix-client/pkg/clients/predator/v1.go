package predator

import (
	"context"
	"fmt"
	"time"

	triton "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/predator/client/grpc"
	"github.com/Meesho/BharatMLStack/helix-client/pkg/grpcclient"
	"github.com/Meesho/BharatMLStack/helix-client/pkg/metric"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"
)

const (
	v1Prefix = "externalservicepredator_"

	// Header keys for authentication
	headerCallerID      = "PREDATOR-CALLER-ID"
	headerCallerToken   = "PREDATOR-AUTH-TOKEN"
	predatorServiceName = "predator"
)

type ClientV1 struct {
	adapter     Adapter
	callerId    string
	callerToken string
	conn        *grpcclient.GRPCClient
	grpcClient  triton.GRPCInferenceServiceClient
}

// NewClientV1 creates a new instance of the Orion client (v1)
func NewClientV1(config *Config) *ClientV1 {
	validateConfig(config)

	conn := grpcclient.NewConnFromConfig(&grpcclient.Config{
		Host:      config.Host,
		Port:      config.Port,
		DeadLine:  config.DeadLine,
		PlainText: config.PlainText,
	}, v1Prefix)

	// Create the gRPC client once during initialization
	grpcClient := triton.NewGRPCInferenceServiceClient(conn)

	return &ClientV1{
		adapter:     Adapter{},
		callerId:    config.CallerId,
		callerToken: config.CallerToken,
		conn:        conn,
		grpcClient:  grpcClient,
	}
}

func validateConfig(config *Config) {
	if config == nil {
		log.Panic().Msg("Configuration is nil. Please provide a valid config.")
		return
	}
	if len(config.Host) == 0 {
		log.Panic().Msg("Configuration error: Host is empty. Please provide a valid host.")
	}
	if len(config.Port) == 0 {
		log.Panic().Msg("Configuration error: Port is empty. Please provide a valid port.")
	}
	if len(config.CallerId) == 0 {
		log.Panic().Msg("Configuration error: Caller ID is empty. Please provide a valid caller ID.")
	}
	if len(config.CallerToken) == 0 {
		log.Panic().Msg("Configuration error: Caller token is empty. Please provide a valid caller token.")
	}
}

func getRequestDataType(req *PredatorRequest) PredatorDataType {
	if len(req.Inputs[0].StringData) > 0 {
		return PredatorDataTypeString
	}
	if len(req.Inputs[0].Data) > 0 {
		return PredatorDataTypeBytes
	}
	return PredatorDataTypeNone
}

type BatchInfo struct {
	StartIndex int
	EndIndex   int
}

type preparedRequest struct {
	requestedOutputs []Output
	predatorDataType PredatorDataType
	deadline         int64
	numInputs        int
	batches          []BatchInfo
	batchProtoReqs   []*triton.ModelInferRequest
}

type batchResult struct {
	index    int
	response *triton.ModelInferResponse
	err      error
}

// prepareInferenceRequest prepares the initial request components
func (c *ClientV1) prepareInferenceRequest(req *PredatorRequest) (*preparedRequest, error) {
	err := validatePredatorRequest(req)
	if err != nil {
		return nil, err
	}

	requestedOutputs := req.Outputs
	predatorDataType := getRequestDataType(req)
	batchSize := req.BatchSize
	deadline := req.Deadline

	var numInputs int
	if predatorDataType == PredatorDataTypeString {
		numInputs = len(req.Inputs[0].StringData)
	} else {
		numInputs = len(req.Inputs[0].Data)
	}

	batches := c.getBatchIndices(numInputs, batchSize)
	batchProtoReqs := c.adapter.MapPredatorRequestToProto(req, predatorDataType, batches)

	if batchProtoReqs == nil {
		return nil, fmt.Errorf("failed to map request to proto")
	}

	return &preparedRequest{
		requestedOutputs: requestedOutputs,
		predatorDataType: predatorDataType,
		deadline:         deadline,
		numInputs:        numInputs,
		batches:          batches,
		batchProtoReqs:   batchProtoReqs,
	}, nil
}

// processBatchInParallel processes all batches in parallel and returns results
func (c *ClientV1) processBatchInParallel(req *preparedRequest) []batchResult {
	resultChan := make(chan batchResult, len(req.batches))
	results := make([]batchResult, len(req.batches))

	for i := range req.batches {
		go c.processSingleBatch(i, req.batchProtoReqs[i], req.deadline, resultChan)
	}

	for i := 0; i < len(req.batches); i++ {
		indexedResult := <-resultChan
		results[indexedResult.index] = indexedResult
	}

	return results
}

// processSingleBatch processes a single batch request
func (c *ClientV1) processSingleBatch(batchIndex int, batchProtoReq *triton.ModelInferRequest, deadline int64, resultChan chan batchResult) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn().
				Interface("panic", r).
				Int("batch_index", batchIndex).
				Msg("Panic occurred while processing batch")
			resultChan <- batchResult{index: batchIndex, response: nil, err: fmt.Errorf("panic in batch processing: %v", r)}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(deadline)*time.Millisecond)
	defer cancel()

	md := getMetadata(c.callerId, c.callerToken)
	ctx = metadata.NewOutgoingContext(ctx, md)
	client := c.grpcClient
	response, err := client.ModelInfer(ctx, batchProtoReq)
	if err != nil {
		log.Warn().Err(err).
			Int("batch_index", batchIndex).
			Str("model_name", batchProtoReq.ModelName).
			Str("model_version", batchProtoReq.ModelVersion).
			Msg("Failed to get inference from Triton server")
		resultChan <- batchResult{index: batchIndex, response: nil, err: err}
		return
	}

	resultChan <- batchResult{index: batchIndex, response: response, err: nil}
}

func (c *ClientV1) GetInferenceScore(req *PredatorRequest) (*PredatorResponse, error) {
	// Prepare request components
	preparedReq, err := c.prepareInferenceRequest(req)
	if err != nil {
		return nil, err
	}

	// Process batches in parallel
	results := c.processBatchInParallel(preparedReq)

	// Convert each proto response to PredatorResponse and merge
	finalResponse, err := c.convertAndMergeResponses(req, preparedReq, results)
	if err != nil {
		return nil, err
	}

	return finalResponse, nil
}

func (c *ClientV1) GetInferenceScoreV2(req *PredatorRequest) (*triton.ModelInferResponse, error) {
	// Prepare request components
	preparedReq, err := c.prepareInferenceRequest(req)
	if err != nil {
		return nil, err
	}

	// Process batches in parallel
	results := c.processBatchInParallel(preparedReq)

	// Convert each proto response to PredatorResponse and merge
	finalResponse, err := c.convertAndMergeResponsesV2(req, preparedReq, results)
	if err != nil {
		return nil, err
	}

	return finalResponse, nil
}

// convertAndMergeResponses converts proto responses to PredatorResponses and merges them
func (c *ClientV1) convertAndMergeResponses(predatorReq *PredatorRequest, preparedReq *preparedRequest, results []batchResult) (*PredatorResponse, error) {
	var predatorResponses []*PredatorResponse
	errorCount := 0

	// Convert each batch result to PredatorResponse
	for i, result := range results {
		if result.err != nil {
			errorCount++
			log.Warn().Err(result.err).
				Int("batch_index", i).
				Msg("Batch processing failed")
			continue
		}

		if result.response == nil {
			errorCount++
			log.Warn().Int("batch_index", i).Msg("Batch returned nil response")
			continue
		}

		// Use adapter to convert proto response to PredatorResponse
		predatorResp := c.adapter.MapProtoToPredatorResponse(result.response, preparedReq.requestedOutputs, preparedReq.predatorDataType)
		if predatorResp == nil {
			errorCount++
			log.Error().Int("batch_index", i).Msg("Failed to map triton response to predator response")
			continue
		}

		predatorResponses = append(predatorResponses, predatorResp)
	}

	// If all batches failed, return an error
	if len(predatorResponses) == 0 {
		if errorCount > 0 {
			return nil, fmt.Errorf("all batch processing failed: %v", errorCount)
		}
		return nil, fmt.Errorf("all batch processing failed")
	}

	// If some batches failed, log warning but continue with successful ones
	if errorCount > 0 {
		log.Warn().
			Int("failed_batches", errorCount).
			Int("successful_batches", len(predatorResponses)).
			Int("total_batches", len(results)).
			Msg("Some batches failed but continuing with successful results")
		metricTags := c.buildMetricTags(predatorReq.ModelName, "batch_processing_error")
		metric.Count(metric.ExternalApiRequestCount, int64(errorCount), metricTags)
	}

	// Merge all successful responses
	mergedResponse := c.mergeResponses(predatorResponses)
	if mergedResponse == nil {
		return nil, fmt.Errorf("failed to merge batch responses")
	}

	return mergedResponse, nil
}

// convertAndMergeResponsesV2 merges proto responses directly and returns merged ModelInferResponse
func (c *ClientV1) convertAndMergeResponsesV2(predatorReq *PredatorRequest, preparedReq *preparedRequest, results []batchResult) (*triton.ModelInferResponse, error) {
	var successfulResponses []*triton.ModelInferResponse
	errorCount := 0
	successfulIndices := make([]int, 0)

	// Collect successful batch responses
	for i, result := range results {
		if result.err != nil {
			errorCount++
			log.Warn().Err(result.err).
				Int("batch_index", i).
				Msg("Batch processing failed")
			continue
		}

		if result.response == nil {
			errorCount++
			log.Warn().Int("batch_index", i).Msg("Batch returned nil response")
			continue
		}

		successfulResponses = append(successfulResponses, result.response)
		successfulIndices = append(successfulIndices, i)
	}

	// If all batches failed, return an error
	if len(successfulResponses) == 0 {
		if errorCount > 0 {
			return nil, fmt.Errorf("all batch processing failed: %v", errorCount)
		}
		return nil, fmt.Errorf("all batch processing failed")
	}

	// If some batches failed, log warning but continue with successful ones
	if errorCount > 0 {
		log.Warn().
			Int("failed_batches", errorCount).
			Int("successful_batches", len(successfulResponses)).
			Int("total_batches", len(results)).
			Msg("Some batches failed but continuing with successful results")
		metricTags := c.buildMetricTags(predatorReq.ModelName, "batch_processing_error")
		metric.Count(metric.ExternalApiRequestCount, int64(errorCount), metricTags)
	}

	// If there's only one successful response, return it directly
	if len(successfulResponses) == 1 {
		return successfulResponses[0], nil
	}

	// Create final proto response using first successful response as base
	firstResponse := successfulResponses[0]
	finalProto := &triton.ModelInferResponse{
		Outputs:           make([]*triton.ModelInferResponse_InferOutputTensor, len(firstResponse.Outputs)),
		RawOutputContents: make([][]byte, len(firstResponse.RawOutputContents)),
	}

	// Initialize output tensor structures
	for i := range firstResponse.Outputs {
		finalProto.Outputs[i] = &triton.ModelInferResponse_InferOutputTensor{}
	}

	// Pre-allocate sizes using first batch
	c.preAllocateFinalSizes(finalProto, firstResponse, preparedReq.numInputs, preparedReq.batches)

	// Set metadata from first batch
	c.setProtoMetadataFromBatch(finalProto, firstResponse, preparedReq.numInputs)

	// Set output names (not set by setProtoMetadataFromBatch)
	for i, output := range firstResponse.Outputs {
		if i < len(finalProto.Outputs) {
			finalProto.Outputs[i].Name = output.Name
		}
	}

	// Copy data from all successful batches
	for idx, response := range successfulResponses {
		batchIndex := successfulIndices[idx]
		if batchIndex < len(preparedReq.batches) {
			c.copyBatchResponseToProto(finalProto, response, preparedReq.batches[batchIndex])
		}
	}

	return finalProto, nil
}

// mergeResponses combines multiple PredatorResponse objects into a single response
func (c *ClientV1) mergeResponses(responses []*PredatorResponse) *PredatorResponse {
	if len(responses) == 0 {
		log.Warn().Msg("Attempting to merge empty response list")
		return nil
	}

	// If there's only one response, return it directly
	if len(responses) == 1 {
		return responses[0]
	}

	// Create merged response using the first response as base
	mergedResp := &PredatorResponse{
		ModelName:    responses[0].ModelName,
		ModelVersion: responses[0].ModelVersion,
		Outputs:      make([]ResponseOutput, len(responses[0].Outputs)),
	}

	// Merge output data for each output
	for i, output := range responses[0].Outputs {
		var combinedData [][]byte
		var combinedStringData [][]string

		// Calculate total shape for the merged output
		totalShape := make([]int64, len(output.Shape))
		copy(totalShape, output.Shape)
		if len(totalShape) > 0 {
			totalShape[0] = 0 // Will be calculated based on combined data
		}

		for j, resp := range responses {
			if resp == nil {
				log.Warn().
					Int("response_index", j).
					Msg("Encountered nil response during merge")
				continue
			}

			if len(resp.Outputs) <= i {
				log.Warn().
					Int("response_index", j).
					Int("output_index", i).
					Int("available_outputs", len(resp.Outputs)).
					Msg("Response has fewer outputs than expected")
				continue
			}

			// Combine byte data
			if len(resp.Outputs[i].Data) > 0 {
				combinedData = append(combinedData, resp.Outputs[i].Data...)
				if len(totalShape) > 0 {
					totalShape[0] += resp.Outputs[i].Shape[0]
				}
			}
			// Combine string data
			if len(resp.Outputs[i].StringData) > 0 {
				combinedStringData = append(combinedStringData, resp.Outputs[i].StringData...)
			}
		}

		mergedResp.Outputs[i] = ResponseOutput{
			Name:       output.Name,
			DataType:   output.DataType,
			Shape:      totalShape,
			Data:       combinedData,
			StringData: combinedStringData,
		}
	}

	return mergedResp
}

func (c *ClientV1) getBatchIndices(numElements, batchSize int) []BatchInfo {
	numBatches := (numElements + batchSize - 1) / batchSize
	batches := make([]BatchInfo, numBatches)

	for i := 0; i < numElements; i += batchSize {
		end := i + batchSize
		if end > numElements {
			end = numElements
		}
		batches[i/batchSize] = BatchInfo{StartIndex: i, EndIndex: end}
	}
	return batches
}

func (c *ClientV1) setProtoMetadataFromBatch(finalProto, batchProto *triton.ModelInferResponse, totalInputs int) {
	finalProto.ModelName = batchProto.ModelName
	finalProto.ModelVersion = batchProto.ModelVersion
	finalProto.Id = batchProto.Id
	finalProto.Parameters = batchProto.Parameters

	for i, batchOutput := range batchProto.Outputs {
		if i < len(finalProto.Outputs) {
			finalProto.Outputs[i].Datatype = batchOutput.Datatype
			finalProto.Outputs[i].Shape = make([]int64, len(batchOutput.Shape))
			copy(finalProto.Outputs[i].Shape, batchOutput.Shape)

			if len(finalProto.Outputs[i].Shape) > 0 {
				finalProto.Outputs[i].Shape[0] = int64(totalInputs)
			}

			finalProto.Outputs[i].Parameters = batchOutput.Parameters
		}
	}
}

func (c *ClientV1) preAllocateFinalSizes(finalProto, batchProto *triton.ModelInferResponse, totalInputs int, batches []BatchInfo) {
	for i := range batchProto.Outputs {
		if i < len(finalProto.RawOutputContents) {
			batchSize := batches[0].EndIndex - batches[0].StartIndex
			if len(batchProto.RawOutputContents) > i {
				sizePerElement := len(batchProto.RawOutputContents[i]) / batchSize
				totalExpectedSize := sizePerElement * totalInputs
				finalProto.RawOutputContents[i] = make([]byte, totalExpectedSize)
			}
		}
	}
}

func (c *ClientV1) copyBatchResponseToProto(finalProto, batchProto *triton.ModelInferResponse, batch BatchInfo) {
	batchSize := batch.EndIndex - batch.StartIndex

	for i, batchRawOutput := range batchProto.RawOutputContents {
		if i < len(finalProto.RawOutputContents) {
			if len(batchRawOutput) > 0 && batchSize > 0 {
				bytesPerElement := len(batchRawOutput) / batchSize
				startBytePos := batch.StartIndex * bytesPerElement
				endBytePos := batch.EndIndex * bytesPerElement

				if endBytePos <= len(finalProto.RawOutputContents[i]) {
					copy(finalProto.RawOutputContents[i][startBytePos:endBytePos], batchRawOutput)
				} else {
					finalProto.RawOutputContents[i] = append(finalProto.RawOutputContents[i], batchRawOutput...)
				}
			}
		}
	}
}

// Keep the existing mergeResponses method as is
// Remove the old splitIntoBatches method since we're using getBatchIndices now

func getMetadata(callerId string, callerToken string) metadata.MD {
	md := metadata.New(nil)
	md.Set(headerCallerID, callerId)
	md.Set(headerCallerToken, callerToken)
	return md
}

func validatePredatorRequest(req *PredatorRequest) error {
	if req == nil {
		err := fmt.Errorf("predator request cannot be nil")
		log.Error().Msg("Received nil predator request")
		return err
	}

	if len(req.Inputs) == 0 {
		err := fmt.Errorf("predator request must contain at least one input")
		log.Error().Msg("Predator request contains no inputs")
		return err
	}

	// Validate required request parameters
	if req.BatchSize <= 0 {
		err := fmt.Errorf("batch_size must be greater than 0, got: %d", req.BatchSize)
		log.Error().Int("batch_size", req.BatchSize).Msg("Invalid batch size in request")
		return err
	}

	if req.Deadline <= 0 {
		err := fmt.Errorf("deadline must be greater than 0, got: %d", req.Deadline)
		log.Error().Int64("deadline", req.Deadline).Msg("Invalid deadline in request")
		return err
	}

	return nil
}

func (c *ClientV1) buildMetricTags(modelName string, errorType string) []string {
	return metric.BuildTag(
		metric.NewTag("model-name", modelName),
		metric.NewTag("caller-id", c.callerId),
		metric.NewTag("error_type", errorType),
	)
}
