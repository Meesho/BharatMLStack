![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/go-sdk.yml/badge.svg)
![Static Badge](https://img.shields.io/badge/release-v1.0.0-blue?style=flat)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)

# BharatMLStack Go SDK

A Go SDK for interacting with BharatMLStack components, providing easy-to-use client libraries for the Online Feature Store and other services.

## Features

- **Online Feature Store Client**: Complete gRPC client for feature retrieval and persistence
- **Multiple API Methods**: Support for `RetrieveFeatures`, `RetrieveDecodedFeatures`, and `PersistFeatures`
- **Protocol Buffer Support**: Generated clients from proto definitions with full type safety
- **Batch Processing**: Configurable batch sizes for efficient bulk operations
- **Authentication**: Built-in support for caller ID and token-based authentication
- **Connection Management**: Configurable timeouts, TLS, and connection pooling
- **Metrics Integration**: Built-in timing and count metrics for monitoring
- **Type-Safe API**: Strongly typed Go interfaces and data structures
- **Test Coverage**: Comprehensive test suite with mocking support

## Installation

```bash
go get github.com/Meesho/BharatMLStack/go-sdk
```

## Configuration

The SDK requires a configuration object with the following fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Host` | string | Yes | Server hostname (e.g., "localhost", "feature-store.example.com") |
| `Port` | string | Yes | Server port (e.g., "8080", "443") |
| `CallerId` | string | Yes | Unique identifier for your service/application |
| `CallerToken` | string | Yes | Authentication token for API access |
| `DeadLine` | int | No | Request timeout in milliseconds (default: 5000) |
| `PlainText` | bool | No | Use plaintext connection instead of TLS (default: false) |
| `BatchSize` | int | No | Maximum batch size for bulk operations (default: 50) |

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/Meesho/BharatMLStack/go-sdk/pkg/onfs"
)

func main() {
    config := &onfs.Config{
        Host:        "localhost",
        Port:        "8080",
        PlainText:   true, // For local development
        CallerId:    "my-service",
        CallerToken: "my-token",
    }

    // Initialize client (timing and count can be nil)
    client := onfs.NewClientV1(config, nil, nil)
    
    // Your feature operations here...
}
```

### Complete Example

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/Meesho/BharatMLStack/go-sdk/pkg/onfs"
)

func main() {
    // Create configuration
    config := &onfs.Config{
        Host:        "localhost",
        Port:        "8080",
        DeadLine:    5000, // 5 seconds timeout in milliseconds
        PlainText:   true, // Use plaintext connection for local development
        BatchSize:   50,   // Optional: batch size for requests
        CallerId:    "your-service-id",
        CallerToken: "your-auth-token",
    }

    // Timing and count functions (can be nil for basic usage)
    timing := func(name string, value time.Duration, tags []string) {
        log.Printf("Timing: %s took %v with tags %v", name, value, tags)
    }
    count := func(name string, value int64, tags []string) {
        log.Printf("Count: %s = %d with tags %v", name, value, tags)
    }

    // Initialize the client
    client := onfs.InitClient(onfs.Version1, config, timing, count)
    // Or alternatively use: client := onfs.NewClientV1(config, timing, count)

    ctx := context.Background()

    // Example: Retrieve features
    query := &onfs.Query{
        EntityLabel: "user",
        FeatureGroups: []onfs.FeatureGroup{
            {
                Label:         "user_features",
                FeatureLabels: []string{"age", "location", "preferences"},
            },
        },
        KeysSchema: []string{"user_id"},
        Keys: []onfs.Keys{
            {Cols: []string{"12345"}},
            {Cols: []string{"67890"}},
        },
    }

    result, err := client.RetrieveFeatures(ctx, query)
    if err != nil {
        log.Fatalf("Failed to retrieve features: %v", err)
    }

    log.Printf("Retrieved %d rows for entity %s", len(result.Rows), result.EntityLabel)

    // Example: Retrieve decoded features (string values)
    decodedResult, err := client.RetrieveDecodedFeatures(ctx, query)
    if err != nil {
        log.Fatalf("Failed to retrieve decoded features: %v", err)
    }

    log.Printf("Retrieved %d decoded rows", len(decodedResult.Rows))

    // Example: Persist features
    persistRequest := &onfs.PersistFeaturesRequest{
        EntityLabel: "user",
        KeysSchema:  []string{"user_id"},
        FeatureGroups: []onfs.FeatureGroupSchema{
            {
                Label:         "user_features",
                FeatureLabels: []string{"age", "location"},
            },
        },
        Data: []onfs.Data{
            {
                KeyValues: []string{"12345"},
                FeatureValues: []onfs.FeatureValues{
                    {
                        Values: onfs.Values{
                            Int32Values:  []int32{25},
                            StringValues: []string{"New York"},
                        },
                    },
                },
            },
        },
    }

    persistResponse, err := client.PersistFeatures(ctx, persistRequest)
    if err != nil {
        log.Fatalf("Failed to persist features: %v", err)
    }

    log.Printf("Persist result: %s", persistResponse.Message)
}
```

## Development

### Prerequisites

- Go 1.22 or later (as specified in go.mod)

### Building

```bash
# Build all packages
go build ./...

# Run tests
go test ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Testing

```bash
# Run all tests
go test -v ./...

# Run specific package tests
go test -v ./pkg/onfs

# Run with race detection
go test -race ./...
```


## License

See [LICENSE.md](../LICENSE.md) for details. 