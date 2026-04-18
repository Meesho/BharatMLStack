package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	triton "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/predator/client/grpc"
)

type server struct {
	triton.UnimplementedGRPCInferenceServiceServer
}

// hashInputs creates a deterministic hash from input tensor contents
func hashInputs(inputs []*triton.ModelInferRequest_InferInputTensor) uint64 {
	h := sha256.New()

	for _, input := range inputs {
		// Hash the input name
		h.Write([]byte(input.Name))

		// Hash the contents based on data type
		if input.Contents != nil {
			// Hash FP32 contents
			for _, val := range input.Contents.Fp32Contents {
				b := make([]byte, 4)
				binary.LittleEndian.PutUint32(b, uint32(val))
				h.Write(b)
			}

			// Hash INT64 contents
			for _, val := range input.Contents.Int64Contents {
				b := make([]byte, 8)
				binary.LittleEndian.PutUint64(b, uint64(val))
				h.Write(b)
			}

			// Hash INT32 contents
			for _, val := range input.Contents.IntContents {
				b := make([]byte, 4)
				binary.LittleEndian.PutUint32(b, uint32(val))
				h.Write(b)
			}

			// Hash BYTES contents
			for _, val := range input.Contents.BytesContents {
				h.Write(val)
			}
		}
	}

	// Take first 8 bytes of hash as uint64
	hashBytes := h.Sum(nil)
	return binary.LittleEndian.Uint64(hashBytes[:8])
}

// generateScores generates deterministic scores (0-1) based on hash
func generateScores(hash uint64, numCatalogs int) []float32 {
	scores := make([]float32, numCatalogs)

	for i := 0; i < numCatalogs; i++ {
		// Generate different score for each catalog using hash + index
		// This ensures same input always gives same scores
		catalogHash := hash + uint64(i*7919) // Use prime number for better distribution
		score := float64(catalogHash) / float64(^uint64(0))

		// Ensure score is in [0, 1) range
		if score < 0 {
			score = -score
		}
		if score >= 1.0 {
			score = score - float64(int64(score))
		}

		scores[i] = float32(score)
	}

	return scores
}

// scoreFromHash converts a hash into a float32 in [0, 1).
func scoreFromHash(hash uint64) float32 {
	score := float64(hash) / float64(^uint64(0))
	if score < 0 {
		score = -score
	}
	if score >= 1.0 {
		score = score - float64(int64(score))
	}
	return float32(score)
}

// inferRowCount determines the batch size (number of rows) from the request.
// Falls back to 0 if shapes are unavailable, which tells callers to use the
// legacy catalog-based path.
func inferRowCount(req *triton.ModelInferRequest) int {
	for _, in := range req.Inputs {
		if len(in.Shape) > 0 && in.Shape[0] > 0 {
			return int(in.Shape[0])
		}
	}
	return 0
}

// ModelInfer implements the ModelInfer RPC.
//
// Two execution modes:
//  1. Per-row mode (preferred): When req.Inputs[0].Shape[0] > 0 AND the request
//     carries RawInputContents, we hash the input bytes for each row separately
//     and emit exactly one score per row per requested output. This makes the
//     dummy's output actually depend on feature values — different features,
//     different scores.
//  2. Legacy catalog mode: If no shape is available, fall back to the original
//     behaviour (hash of input tensor metadata → 10 deterministic scores).
func (s *server) ModelInfer(ctx context.Context, req *triton.ModelInferRequest) (*triton.ModelInferResponse, error) {
	rowCount := inferRowCount(req)
	hasRawInputs := len(req.RawInputContents) > 0

	response := &triton.ModelInferResponse{
		ModelName:         req.ModelName,
		ModelVersion:      req.ModelVersion,
		Id:                req.Id,
		Outputs:           make([]*triton.ModelInferResponse_InferOutputTensor, 0),
		RawOutputContents: make([][]byte, 0),
	}

	if rowCount > 0 && hasRawInputs {
		// Per-row hashing: chunk each input tensor's raw bytes by row and hash.
		rowHashes := make([]uint64, rowCount)
		for inputIdx, in := range req.Inputs {
			if inputIdx >= len(req.RawInputContents) {
				break
			}
			raw := req.RawInputContents[inputIdx]
			if len(raw) == 0 || rowCount == 0 {
				continue
			}
			bytesPerRow := len(raw) / rowCount
			if bytesPerRow == 0 {
				continue
			}
			for row := 0; row < rowCount; row++ {
				start := row * bytesPerRow
				end := start + bytesPerRow
				if end > len(raw) {
					end = len(raw)
				}
				h := sha256.New()
				h.Write([]byte(in.Name))
				h.Write(raw[start:end])
				sum := h.Sum(nil)
				// Mix with any previous input's hash for this row so multi-input
				// tensors compose.
				mixed := binary.LittleEndian.Uint64(sum[:8])
				rowHashes[row] = rowHashes[row] ^ mixed
			}
		}

		outs := req.Outputs
		if len(outs) == 0 {
			outs = []*triton.ModelInferRequest_InferRequestedOutputTensor{{Name: "output"}}
		}

		// One FP32 score per row per requested output. The helix-client adapter
		// reads RawOutputContents[outputIdx] as rowCount*4 bytes.
		for outIdx, outReq := range outs {
			buf := make([]byte, rowCount*4)
			for row := 0; row < rowCount; row++ {
				// Salt the hash with outIdx so multiple outputs differ.
				score := scoreFromHash(rowHashes[row] + uint64(outIdx)*7919)
				binary.LittleEndian.PutUint32(buf[row*4:(row+1)*4], math.Float32bits(score))
			}
			response.Outputs = append(response.Outputs, &triton.ModelInferResponse_InferOutputTensor{
				Name:     outReq.Name,
				Datatype: "FP32",
				Shape:    []int64{int64(rowCount), 1},
			})
			response.RawOutputContents = append(response.RawOutputContents, buf)
		}
		return response, nil
	}

	// Legacy fallback: single catalog-style output (10 deterministic scores).
	hash := hashInputs(req.Inputs)
	numCatalogs := 10
	scores := make([]float32, numCatalogs)
	for i := 0; i < numCatalogs; i++ {
		scores[i] = scoreFromHash(hash + uint64(i*7919))
	}
	rawOutputBytes := make([]byte, len(scores)*4)
	for i, score := range scores {
		binary.LittleEndian.PutUint32(rawOutputBytes[i*4:(i+1)*4], math.Float32bits(score))
	}
	if len(req.Outputs) == 0 {
		response.Outputs = []*triton.ModelInferResponse_InferOutputTensor{{Name: "output", Datatype: "FP32", Shape: []int64{int64(len(scores))}}}
		response.RawOutputContents = [][]byte{rawOutputBytes}
	} else {
		for _, outputReq := range req.Outputs {
			response.Outputs = append(response.Outputs, &triton.ModelInferResponse_InferOutputTensor{Name: outputReq.Name, Datatype: "FP32", Shape: []int64{int64(len(scores))}})
			response.RawOutputContents = append(response.RawOutputContents, rawOutputBytes)
		}
	}
	return response, nil
}

// ServerLive implements health check
func (s *server) ServerLive(ctx context.Context, req *triton.ServerLiveRequest) (*triton.ServerLiveResponse, error) {
	return &triton.ServerLiveResponse{Live: true}, nil
}

// ServerReady implements readiness check
func (s *server) ServerReady(ctx context.Context, req *triton.ServerReadyRequest) (*triton.ServerReadyResponse, error) {
	return &triton.ServerReadyResponse{Ready: true}, nil
}

// ModelReady implements model readiness check
func (s *server) ModelReady(ctx context.Context, req *triton.ModelReadyRequest) (*triton.ModelReadyResponse, error) {
	return &triton.ModelReadyResponse{Ready: true}, nil
}

// ServerMetadata implements server metadata
func (s *server) ServerMetadata(ctx context.Context, req *triton.ServerMetadataRequest) (*triton.ServerMetadataResponse, error) {
	return &triton.ServerMetadataResponse{
		Name:       "predator-dummy-server",
		Version:    "1.0.0",
		Extensions: []string{"classification", "sequence"},
	}, nil
}

// ModelMetadata implements model metadata
func (s *server) ModelMetadata(ctx context.Context, req *triton.ModelMetadataRequest) (*triton.ModelMetadataResponse, error) {
	return &triton.ModelMetadataResponse{
		Name:     req.Name,
		Versions: []string{"1"},
		Platform: "dummy",
		Inputs: []*triton.ModelMetadataResponse_TensorMetadata{
			{
				Name:     "input",
				Datatype: "FP32",
				Shape:    []int64{-1},
			},
		},
		Outputs: []*triton.ModelMetadataResponse_TensorMetadata{
			{
				Name:     "output",
				Datatype: "FP32",
				Shape:    []int64{-1},
			},
		},
	}, nil
}

// ModelConfig implements model config
func (s *server) ModelConfig(ctx context.Context, req *triton.ModelConfigRequest) (*triton.ModelConfigResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "ModelConfig not implemented")
}

// ModelStatistics implements model statistics
func (s *server) ModelStatistics(ctx context.Context, req *triton.ModelStatisticsRequest) (*triton.ModelStatisticsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "ModelStatistics not implemented")
}

// RepositoryIndex implements repository index
func (s *server) RepositoryIndex(ctx context.Context, req *triton.RepositoryIndexRequest) (*triton.RepositoryIndexResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "RepositoryIndex not implemented")
}

// RepositoryModelLoad implements model load
func (s *server) RepositoryModelLoad(ctx context.Context, req *triton.RepositoryModelLoadRequest) (*triton.RepositoryModelLoadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "RepositoryModelLoad not implemented")
}

// RepositoryModelUnload implements model unload
func (s *server) RepositoryModelUnload(ctx context.Context, req *triton.RepositoryModelUnloadRequest) (*triton.RepositoryModelUnloadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "RepositoryModelUnload not implemented")
}

// SystemSharedMemoryStatus implements shared memory status
func (s *server) SystemSharedMemoryStatus(ctx context.Context, req *triton.SystemSharedMemoryStatusRequest) (*triton.SystemSharedMemoryStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "SystemSharedMemoryStatus not implemented")
}

// SystemSharedMemoryRegister implements shared memory register
func (s *server) SystemSharedMemoryRegister(ctx context.Context, req *triton.SystemSharedMemoryRegisterRequest) (*triton.SystemSharedMemoryRegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "SystemSharedMemoryRegister not implemented")
}

// SystemSharedMemoryUnregister implements shared memory unregister
func (s *server) SystemSharedMemoryUnregister(ctx context.Context, req *triton.SystemSharedMemoryUnregisterRequest) (*triton.SystemSharedMemoryUnregisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "SystemSharedMemoryUnregister not implemented")
}

// CudaSharedMemoryStatus implements CUDA shared memory status
func (s *server) CudaSharedMemoryStatus(ctx context.Context, req *triton.CudaSharedMemoryStatusRequest) (*triton.CudaSharedMemoryStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "CudaSharedMemoryStatus not implemented")
}

// CudaSharedMemoryRegister implements CUDA shared memory register
func (s *server) CudaSharedMemoryRegister(ctx context.Context, req *triton.CudaSharedMemoryRegisterRequest) (*triton.CudaSharedMemoryRegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "CudaSharedMemoryRegister not implemented")
}

// CudaSharedMemoryUnregister implements CUDA shared memory unregister
func (s *server) CudaSharedMemoryUnregister(ctx context.Context, req *triton.CudaSharedMemoryUnregisterRequest) (*triton.CudaSharedMemoryUnregisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "CudaSharedMemoryUnregister not implemented")
}

// TraceSetting implements trace setting
func (s *server) TraceSetting(ctx context.Context, req *triton.TraceSettingRequest) (*triton.TraceSettingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "TraceSetting not implemented")
}

// LogSettings implements log settings
func (s *server) LogSettings(ctx context.Context, req *triton.LogSettingsRequest) (*triton.LogSettingsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "LogSettings not implemented")
}

// ModelStreamInfer implements streaming inference (not implemented for dummy server)
func (s *server) ModelStreamInfer(stream triton.GRPCInferenceService_ModelStreamInferServer) error {
	return status.Errorf(codes.Unimplemented, "ModelStreamInfer not implemented")
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	triton.RegisterGRPCInferenceServiceServer(s, &server{})

	log.Printf("Predator dummy gRPC server listening on port %s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
