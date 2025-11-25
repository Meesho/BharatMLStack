# Predator Client

A Go client library for interacting with Triton Inference Server through the Predator service.

## Features

- **Batch Processing**: Automatically splits large requests into configurable batch sizes
- **Concurrent Execution**: Processes batches in parallel using goroutines
- **Multiple Data Types**: Supports FP32, INT32, INT64, BYTES, and other Triton data types
- **Authentication**: Built-in support for caller ID and token-based authentication
- **Error Handling**: Comprehensive error handling with detailed logging
- **Response Merging**: Automatically merges batch responses into a single response

## Quick Start

### 1. Configuration

```go
config := &predator.Config{
    Host:        "localhost",          // Triton server host
    Port:        "8001",              // Triton server port (gRPC)
    DeadLine:    30000,               // Timeout in milliseconds
    PlainText:   true,                // Use plaintext connection (set false for TLS)
    BatchSize:   200,                 // Maximum batch size for splitting requests
    CallerId:    "your-caller-id",    // Authentication caller ID
    CallerToken: "your-auth-token",   // Authentication token
}
```

### 2. Initialize Client

```go
// Initialize the client (singleton pattern)
client := predator.InitClient(predator.Version1, config)

// Or get existing instance
client := predator.GetInstance(predator.Version1)
```

### 3. Create Request

```go
request := &predator.PredatorRequest{
    ModelName:    "your_model_name",
    ModelVersion: "1",
    Inputs: []predator.Input{
        {
            Name:     "input_tensor",
            DataType: "FP32",
            Dims:     []int{4},           // Shape of each sample
            Data:     inputData,          // [][]byte containing your data
        },
    },
    Outputs: []predator.Output{
        {
            Name:        "output_scores",
            ModelScores: []string{"score1", "score2"},
        },
    },
}
```

### 4. Make Inference Call

```go
response, err := client.GetInferenceScore(request)
if err != nil {
    log.Printf("Error: %v", err)
    return
}

// Process response
for i, output := range response.Outputs {
    fmt.Printf("Output %d: %s, Shape: %v, Data length: %d\n", 
        i+1, output.Name, output.Shape, len(output.Data))
}
```

## Data Type Conversion

### Float32 Data
```go
func createFloat32Data(data [][]float32) [][]byte {
    result := make([][]byte, len(data))
    for i, row := range data {
        bytes := make([]byte, len(row)*4)
        for j, val := range row {
            binary.LittleEndian.PutUint32(bytes[j*4:(j+1)*4], math.Float32bits(val))
        }
        result[i] = bytes
    }
    return result
}

// Usage
inputData := createFloat32Data([][]float32{{1.0, 2.0, 3.0, 4.0}})
```

### Int32 Data
```go
func createInt32Data(data [][]int32) [][]byte {
    result := make([][]byte, len(data))
    for i, row := range data {
        bytes := make([]byte, len(row)*4)
        for j, val := range row {
            binary.LittleEndian.PutUint32(bytes[j*4:(j+1)*4], uint32(val))
        }
        result[i] = bytes
    }
    return result
}
```

### String Data (BYTES type)
```go
func createStringData(data [][]string) [][]byte {
    result := make([][]byte, len(data))
    for i, row := range data {
        var bytes []byte
        for _, str := range row {
            strBytes := []byte(str)
            // 4-byte length prefix + string content
            lengthBytes := make([]byte, 4)
            binary.LittleEndian.PutUint32(lengthBytes, uint32(len(strBytes)))
            bytes = append(bytes, lengthBytes...)
            bytes = append(bytes, strBytes...)
        }
        result[i] = bytes
    }
    return result
}
```

## Running the Examples

### 1. Run the Main Test Program

```bash
# From the predator directory
cd cmd
go run main.go
```

The main program includes several test scenarios:
- Simple single request
- Batch request (tests automatic batching)
- Multiple input types (FP32, INT32, BYTES)
- Large data test (tests performance)

### 2. Run Unit Tests

```bash
# Run all tests
go test -v

# Run specific test
go test -v -run TestConfigValidation

# Run benchmarks
go test -bench=.

# Run tests with coverage
go test -cover
```

### 3. Run Integration Tests

```bash
# Set up environment and run integration tests
INTEGRATION=1 go test -v -run TestIntegration
```

## Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `Host` | string | Triton server hostname | Required |
| `Port` | string | Triton server port | Required |
| `DeadLine` | int | Request timeout (ms) | Required |
| `PlainText` | bool | Use plaintext connection | `false` |
| `BatchSize` | int | Max batch size | `200` |
| `CallerId` | string | Authentication caller ID | Required |
| `CallerToken` | string | Authentication token | Required |

## Batch Processing

The client automatically splits large requests into smaller batches:

```go
// If you send 1000 samples with BatchSize=200
// The client will:
// 1. Split into 5 batches of 200 samples each
// 2. Process all batches concurrently using goroutines  
// 3. Merge responses back into a single response
// 4. Return the merged result

request := &predator.PredatorRequest{
    // ... your request with 1000 samples
}

// This handles batching automatically
response, err := client.GetInferenceScore(request)
```

## Performance Tips

1. **Optimal Batch Size**: Tune `BatchSize` based on your model and server capacity
2. **Connection Reuse**: The client reuses gRPC connections for better performance
3. **Concurrent Processing**: Batches are processed in parallel
4. **Memory Management**: Consider implementing object pooling for high-throughput scenarios

## Error Handling

```go
response, err := client.GetInferenceScore(request)
if err != nil {
    // Handle different error types
    switch {
    case strings.Contains(err.Error(), "timeout"):
        log.Printf("Request timed out: %v", err)
    case strings.Contains(err.Error(), "connection"):
        log.Printf("Connection error: %v", err)
    default:
        log.Printf("Inference error: %v", err)
    }
    return
}
```

## Monitoring and Debugging

The client uses structured logging with zerolog:

```go
// Enable debug logging
log.Logger = log.Logger.Level(zerolog.DebugLevel)

// Logs include:
// - Request/response details
// - Batch processing information
// - Error details with context
// - Performance metrics
```

## Dependencies

- `github.com/rs/zerolog` - Structured logging
- `google.golang.org/grpc` - gRPC framework

## Best Practices

1. **Initialize Once**: Use the singleton pattern for client initialization
2. **Reuse Clients**: Don't create new clients for each request
3. **Handle Timeouts**: Set appropriate deadlines based on your model's inference time
4. **Monitor Memory**: Watch memory usage with large batch sizes
5. **Error Recovery**: Implement retry logic for transient failures

## Examples Directory

See `cmd/main.go` for comprehensive examples covering:
- Basic usage patterns
- Different data types
- Batch processing
- Error handling
- Performance testing

## Testing

The package includes comprehensive tests:
- Unit tests for all core functionality
- Benchmark tests for performance analysis
- Integration tests (require running Triton server)
- Example usage patterns

Run tests before deployment:
```bash
go test -v -cover ./...
``` 