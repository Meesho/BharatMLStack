# Online-Feature-Store Go Client

A Go client library for interacting with the online-feature-store service. This client provides a simple interface to retrieve and persist features from the online-feature-store service.

## Prerequisites

- Go 1.16 or higher
- Access to an Online-Feature-Store service instance
- Required authentication credentials (CallerId and CallerToken)

## Features

- Retrieve features from online-feature-store service
- Retrieve decoded features
- Persist features to online-feature-store service
- Support for versioned client implementations
- Configurable connection settings
- Batch processing support
- TLS support
- Connection timeout handling
- Automatic retries for failed requests
- Context-based request cancellation
- Proper error handling and propagation

## Installation

```bash
go get github.com/Meesho/BharatMLStack/online-feature/store/sdks/go
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/Meesho/BharatMLStack/online-feature-store/sdks/go"
    "github.com/Meesho/BharatMLStack/online-feature-store/sdks/go/pkg/models"
)

func main() {
    // Create configuration
    config := &models.Config{
        Host:        "your-online-feature-store-host",
        Port:        "your-online-feature-store-port",
        DeadLine:    30,              // Deadline in seconds
        PlainText:   false,           // Use TLS
        BatchSize:   100,             // Batch size for operations
        CallerId:    "your-caller-id",
        CallerToken: "your-caller-token",
    }

    // Initialize the client
    client := onlineFS.InitClient(onlineFSClient.Version1, config)
    
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Example: Retrieve features
    query := &models.Query{
        // Configure your query parameters
    }
    result, err := client.RetrieveFeatures(ctx, query)
    if err != nil {
        log.Fatalf("Failed to retrieve features: %v", err)
    }
    log.Printf("Retrieved features: %+v", result)
}
```

## Usage

### Initializing the Client

```go
import (
    "github.com/Meesho/BharatMLStack/online-feature-store/sdks/go"
    "github.com/Meesho/BharatMLStack/online-feature-store/sdks/go/pkg/models"
)

// Create configuration
config := &models.Config{
    Host:        "your-online-feature-store-host",
    Port:        "your-online-feature-store-port",
    DeadLine:    30,              // Deadline in seconds
    PlainText:   false,           // Use TLS
    BatchSize:   100,             // Batch size for operations (defaults to 50 if not specified)
    CallerId:    "your-caller-id",
    CallerToken: "your-caller-token",
}

// Initialize the client
client := onlineFS.InitClient(onlineFSClient.Version1, config)
```

### Using the Client

All client methods now require a context parameter for better control over timeouts and cancellation:

```go
// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Retrieve features
query := &models.Query{
    // Configure your query parameters
}
result, err := client.RetrieveFeatures(ctx, query)

// Retrieve decoded features
decodedResult, err := client.RetrieveDecodedFeatures(ctx, query)

// Persist features
persistRequest := &models.PersistFeaturesRequest{
    // Configure your persist request
}
persistResponse, err := client.PersistFeatures(ctx, persistRequest)
```

## Configuration

The client can be configured using the following parameters:

- `Host`: online-feature-store service host address (required)
- `Port`: online-feature-store service port (required)
- `DeadLine`: Connection deadline in seconds (required, must be positive)
- `PlainText`: Whether to use plain text (non-TLS) connection
- `BatchSize`: Size of batch operations (defaults to 50 if not specified)
- `CallerId`: Identifier for the caller (required)
- `CallerToken`: Authentication token for the caller (required)

## Error Handling

The client now provides better error handling with specific error types and detailed error messages:

```go
result, err := client.RetrieveFeatures(ctx, query)
if err != nil {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        log.Printf("Request timed out: %v", err)
    case errors.Is(err, context.Canceled):
        log.Printf("Request was canceled: %v", err)
    default:
        log.Printf("Error retrieving features: %v", err)
    }
    return
}
```

## Running Tests

To run the tests:

```bash
go test ./... -v
```

For coverage report:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Troubleshooting

Common issues and their solutions:

1. Connection Timeouts
   - Check if the online-feature-store service is accessible
   - Verify network connectivity
   - Adjust the DeadLine parameter if needed
   - Check context timeout settings

2. Authentication Failures
   - Verify CallerId and CallerToken are correct
   - Check if credentials have expired
   - Ensure proper configuration initialization

3. TLS Issues
   - Ensure proper TLS certificates are configured
   - Verify PlainText setting matches your environment

