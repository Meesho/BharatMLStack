![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/go-sdk.yml/badge.svg)
![Static Badge](https://img.shields.io/badge/release-v1.0.0-blue?style=flat)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)

# BharatMLStack Go SDK

A Go SDK for interacting with BharatMLStack components, providing easy-to-use client libraries for the Online Feature Store and Interaction Store services.

## Features

- **Online Feature Store Client**: Complete gRPC client for feature retrieval and persistence
- **Interaction Store Client**: Complete gRPC client for user interaction tracking (clicks, orders)
- **Multiple API Methods**: Support for feature operations and interaction persistence/retrieval
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

### Online Feature Store (ONFS) Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Host` | string | Yes | Server hostname (e.g., "localhost", "feature-store.example.com") |
| `Port` | string | Yes | Server port (e.g., "8080", "443") |
| `CallerId` | string | Yes | Unique identifier for your service/application |
| `CallerToken` | string | Yes | Authentication token for API access |
| `DeadLine` | int | No | Request timeout in milliseconds (default: 5000) |
| `PlainText` | bool | No | Use plaintext connection instead of TLS (default: false) |
| `BatchSize` | int | No | Maximum batch size for bulk operations (default: 50) |

### Interaction Store Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Host` | string | Yes | Server hostname (e.g., "localhost", "interaction-store.example.com") |
| `Port` | string | Yes | Server port (e.g., "8080", "443") |
| `CallerId` | string | Yes | Unique identifier for your service/application |
| `DeadLine` | int | No | Request timeout in milliseconds (default: 5000) |
| `PlainText` | bool | No | Use plaintext connection instead of TLS (default: false) |

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

## Interaction Store Client

The Interaction Store client provides APIs for tracking and retrieving user interactions like clicks and orders.

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    interactionstore "github.com/Meesho/BharatMLStack/go-sdk/pkg/interaction-store"
)

func main() {
    config := &interactionstore.Config{
        Host:      "localhost",
        Port:      "8090",
        PlainText: true, // For local development
        CallerId:  "my-service",
    }

    // Initialize client (timing and count can be nil)
    client := interactionstore.NewClientV1(config, nil, nil)
    
    // Your interaction operations here...
}
```

### Complete Example

```go
package main

import (
    "context"
    "log"
    "time"
    
    interactionstore "github.com/Meesho/BharatMLStack/go-sdk/pkg/interaction-store"
)

func main() {
    // Create configuration
    config := &interactionstore.Config{
        Host:      "localhost",
        Port:      "8090",
        DeadLine:  5000, // 5 seconds timeout in milliseconds
        PlainText: true, // Use plaintext connection for local development
        CallerId:  "your-service-id",
    }

    // Timing and count functions (can be nil for basic usage)
    timing := func(name string, value time.Duration, tags []string) {
        log.Printf("Timing: %s took %v with tags %v", name, value, tags)
    }
    count := func(name string, value int64, tags []string) {
        log.Printf("Count: %s = %d with tags %v", name, value, tags)
    }

    // Initialize the client
    client := interactionstore.InitClient(interactionstore.Version1, config, timing, count)

    ctx := context.Background()

    // Example: Persist click data
    clickRequest := &interactionstore.PersistClickDataRequest{
        UserId: "user123",
        Data: []interactionstore.ClickData{
            {
                CatalogId: 100,
                ProductId: 200,
                Timestamp: time.Now().UnixMilli(),
                Metadata:  `{"source": "homepage"}`,
            },
        },
    }

    clickResponse, err := client.PersistClickData(ctx, clickRequest)
    if err != nil {
        log.Fatalf("Failed to persist click data: %v", err)
    }
    log.Printf("Persist click result: %s", clickResponse.Message)

    // Example: Persist order data
    orderRequest := &interactionstore.PersistOrderDataRequest{
        UserId: "user123",
        Data: []interactionstore.OrderData{
            {
                CatalogId:   100,
                ProductId:   200,
                SubOrderNum: "SUB001",
                Timestamp:   time.Now().UnixMilli(),
                Metadata:    `{"payment_method": "upi"}`,
            },
        },
    }

    orderResponse, err := client.PersistOrderData(ctx, orderRequest)
    if err != nil {
        log.Fatalf("Failed to persist order data: %v", err)
    }
    log.Printf("Persist order result: %s", orderResponse.Message)

    // Example: Retrieve click interactions
    retrieveRequest := &interactionstore.RetrieveDataRequest{
        UserId:         "user123",
        StartTimestamp: time.Now().Add(-24 * time.Hour).UnixMilli(),
        EndTimestamp:   time.Now().UnixMilli(),
        Limit:          100,
    }

    clicks, err := client.RetrieveClickInteractions(ctx, retrieveRequest)
    if err != nil {
        log.Fatalf("Failed to retrieve clicks: %v", err)
    }
    log.Printf("Retrieved %d click events", len(clicks.Data))

    // Example: Retrieve order interactions
    orders, err := client.RetrieveOrderInteractions(ctx, retrieveRequest)
    if err != nil {
        log.Fatalf("Failed to retrieve orders: %v", err)
    }
    log.Printf("Retrieved %d order events", len(orders.Data))

    // Example: Retrieve multiple interaction types at once
    multiRequest := &interactionstore.RetrieveInteractionsRequest{
        UserId:           "user123",
        InteractionTypes: []interactionstore.InteractionType{
            interactionstore.InteractionTypeClick,
            interactionstore.InteractionTypeOrder,
        },
        StartTimestamp: time.Now().Add(-24 * time.Hour).UnixMilli(),
        EndTimestamp:   time.Now().UnixMilli(),
        Limit:          100,
    }

    interactions, err := client.RetrieveInteractions(ctx, multiRequest)
    if err != nil {
        log.Fatalf("Failed to retrieve interactions: %v", err)
    }
    for key, data := range interactions.Data {
        log.Printf("Interaction type %s: %d clicks, %d orders", 
            key, len(data.ClickEvents), len(data.OrderEvents))
    }
}
```

### Interaction Store API Reference

#### Persist Methods

| Method | Description |
|--------|-------------|
| `PersistClickData(ctx, request)` | Persist click interaction data for a user |
| `PersistOrderData(ctx, request)` | Persist order interaction data for a user |

#### Retrieve Methods

| Method | Description |
|--------|-------------|
| `RetrieveClickInteractions(ctx, request)` | Retrieve click interactions for a user within a time range |
| `RetrieveOrderInteractions(ctx, request)` | Retrieve order interactions for a user within a time range |
| `RetrieveInteractions(ctx, request)` | Retrieve multiple interaction types for a user |

#### Data Types

| Type | Fields |
|------|--------|
| `ClickData` | `CatalogId`, `ProductId`, `Timestamp`, `Metadata` |
| `OrderData` | `CatalogId`, `ProductId`, `SubOrderNum`, `Timestamp`, `Metadata` |
| `InteractionType` | `InteractionTypeClick (0)`, `InteractionTypeOrder (1)` |

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
go test -v ./pkg/interaction-store

# Run with race detection
go test -race ./...
```


## Contributing

We welcome contributions from the community! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to get started.

## Community & Support

- 💬 **Discord**: Join our [community chat](https://discord.gg/XkT7XsV2AU)
- 🐛 **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/Meesho/BharatMLStack/issues)
- 📧 **Email**: Contact us at [ml-oss@meesho.com](mailto:ml-oss@meesho.com )

## License

BharatMLStack is open-source software licensed under the [BharatMLStack Business Source License 1.1](LICENSE.md).

---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
<div align="center">
  <strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong>
</div>