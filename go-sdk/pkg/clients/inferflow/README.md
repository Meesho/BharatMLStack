# Inferflow Client

A Go client library for interacting with the Inferflow Predict service, supporting PointWise, PairWise, and SlateWise inference APIs.

## Features

- **Three Inference Patterns**: PointWise (per-target scoring), PairWise (pair-level ranking), SlateWise (group-level scoring)
- **gRPC Communication**: Efficient binary protocol via gRPC
- **Authentication**: Built-in caller ID and token-based auth via metadata
- **Configurable Timeouts**: Per-client deadline configuration
- **Singleton Pattern**: Thread-safe client initialization with `sync.Once`

## Quick Start

### 1. Configuration

Set environment variables with the `INFERFLOW_CLIENT_V1_` prefix:

```bash
export INFERFLOW_CLIENT_V1_HOST=inferflow.svc
export INFERFLOW_CLIENT_V1_PORT=8080
export INFERFLOW_CLIENT_V1_DEADLINE_MS=500
export INFERFLOW_CLIENT_V1_PLAINTEXT=true
export INFERFLOW_CLIENT_V1_AUTH_TOKEN=your-token
export APP_NAME=my-service
```

### 2. Initialize Client

```go
import "github.com/Meesho/BharatMLStack/go-sdk/pkg/clients/inferflow"

// From environment variables
client := inferflow.GetInferflowClient(1)

// Or from explicit config
client := inferflow.GetInferflowClientFromConfig(1, inferflow.ClientConfig{
    Host:             "inferflow.svc",
    Port:             "8080",
    DeadlineExceedMS: 500,
    PlainText:        true,
    AuthToken:        "your-token",
}, "my-service")
```

### 3. PointWise Inference

Score each target independently against context features.

**Use cases:** CTR prediction, fraud scoring, relevance ranking.

```go
import grpc "github.com/Meesho/BharatMLStack/go-sdk/pkg/clients/inferflow/client/grpc"

resp, err := client.InferPointWise(&grpc.PointWiseRequest{
    ModelConfigId: "ranking_model_v1",
    TrackingId:    "req-123",
    TenantId:      "tenant-1",
    TargetInputSchema: []*grpc.FeatureSchema{
        {Name: "price", DataType: grpc.DataType_DataTypeFP32},
    },
    Targets: []*grpc.Target{
        {Id: "product-1", FeatureValues: [][]byte{priceBytes}},
        {Id: "product-2", FeatureValues: [][]byte{priceBytes}},
    },
    ContextFeatures: []*grpc.ContextFeature{
        {Name: "user_segment", Value: segmentBytes, DataType: grpc.DataType_DataTypeString},
    },
})
// resp.TargetScores contains per-target scores
```

### 4. PairWise Inference

Score pairs of targets relative to each other.

**Use cases:** Preference learning, comparison-based ranking.

```go
resp, err := client.InferPairWise(&grpc.PairWiseRequest{
    ModelConfigId:     "pairwise_model_v1",
    TrackingId:        "req-456",
    TenantId:          "tenant-1",
    TargetInputSchema: targetSchema,
    PairInputSchema:   pairSchema,
    Targets:           targets,
    Pairs: []*grpc.TargetPair{
        {FirstTargetIndex: 0, SecondTargetIndex: 1, FeatureValues: pairFeatures},
    },
})
// resp.PairScores contains per-pair scores
// resp.TargetScores contains optional per-target scores
```

### 5. SlateWise Inference

Score groups (slates) of targets together, capturing inter-item effects.

**Use cases:** Whole-page optimization, slate-level reranking, diversity-aware scoring.

```go
resp, err := client.InferSlateWise(&grpc.SlateWiseRequest{
    ModelConfigId:     "slate_model_v1",
    TrackingId:        "req-789",
    TenantId:          "tenant-1",
    TargetInputSchema: targetSchema,
    SlateInputSchema:  slateSchema,
    Targets:           targets,
    Slates: []*grpc.TargetSlate{
        {TargetIndices: []int32{0, 1, 2}, FeatureValues: slateFeatures},
    },
})
// resp.SlateScores contains per-slate scores
// resp.TargetScores contains optional per-target scores
```

## Configuration Options

| Option | Env Var Suffix | Type | Description | Default |
|--------|---------------|------|-------------|---------|
| `Host` | `HOST` | string | Inferflow service hostname | Required |
| `Port` | `PORT` | string | Inferflow service port | `8080` |
| `DeadlineExceedMS` | `DEADLINE_MS` | int | Request timeout (ms) | `200` |
| `PlainText` | `PLAINTEXT` | bool | Use plaintext connection | `true` |
| `AuthToken` | `AUTH_TOKEN` | string | Authentication token | `""` |

## API Reference

### InferflowClient Interface

```go
type InferflowClient interface {
    InferPointWise(request *grpc.PointWiseRequest) (*grpc.PointWiseResponse, error)
    InferPairWise(request *grpc.PairWiseRequest) (*grpc.PairWiseResponse, error)
    InferSlateWise(request *grpc.SlateWiseRequest) (*grpc.SlateWiseResponse, error)
}
```

## Testing

```bash
go test -v ./pkg/clients/inferflow/...
```

## Dependencies

- `google.golang.org/grpc` — gRPC framework
- `google.golang.org/protobuf` — Protocol Buffers
- `github.com/rs/zerolog` — Structured logging
- `github.com/spf13/viper` — Configuration management
