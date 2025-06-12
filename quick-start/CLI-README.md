# ONFS CLI Tool

A command-line interface for testing ONFS (Online Feature Store) operations including persist, retrieve, and retrieve-decoded functionality.

## Features

- **Persist Features**: Store feature data in the ONFS service
- **Retrieve Features**: Get feature data in raw binary format
- **Retrieve Decoded Features**: Get feature data in human-readable string format
- **JSON Input Support**: Use JSON files for complex data structures
- **Interactive Mode**: Enter data interactively when no input file is provided
- **Flexible Configuration**: Configure host, port, authentication, and other parameters

## Quick Setup

1. **Install the CLI tool:**
   ```bash
   cd quick-start
   ./setup-cli.sh
   ```

2. **Add to PATH (if not already):**
   ```bash
   export PATH="$HOME/bin:$PATH"
   # Add this to your ~/.bashrc or ~/.zshrc for persistence
   ```

3. **Verify installation:**
   ```bash
   onfs-cli --help
   ```

## Prerequisites

- Go 1.22 or later installed
- ONFS service running (via docker-compose or standalone)
- Network access to ONFS service (default: localhost:8089)

## Usage

### Command Line Options

```bash
onfs-cli -operation <persist|retrieve|retrieve-decoded> [options]

Options:
  -host           ONFS service host (default: localhost)
  -port           ONFS service port (default: 8089)
  -caller-id      Caller ID for authentication (default: onfs-cli)
  -caller-token   Caller token for authentication (default: test)
  -operation      Operation: persist, retrieve, retrieve-decoded
  -input          Input JSON file path (optional, uses interactive mode if not provided)
  -plaintext      Use plaintext connection (default: true)
  -timeout        Request timeout in milliseconds (default: 30000)
  -batch-size     Batch size for requests (default: 50)
```

### Basic Examples

1. **Persist features using JSON file:**
   ```bash
   onfs-cli -operation persist -input sample-data/persist-sample.json
   ```

2. **Retrieve features:**
   ```bash
   onfs-cli -operation retrieve -input sample-data/retrieve-sample.json
   ```

3. **Retrieve decoded features:**
   ```bash
   onfs-cli -operation retrieve-decoded -input sample-data/retrieve-sample.json
   ```

4. **Interactive mode:**
   ```bash
   onfs-cli -operation persist
   # Follow the prompts to enter data
   ```

5. **Custom host and port:**
   ```bash
   onfs-cli -operation retrieve -host myserver.com -port 9090 -input my-query.json
   ```

## JSON Input Formats

### Persist JSON Format

```json
{
  "entity_label": "user",
  "keys_schema": ["user_id"],
  "feature_groups": [
    {
      "label": "user_features",
      "feature_labels": ["age", "location", "subscription_type"]
    }
  ],
  "data": [
    {
      "key_values": ["user_123"],
      "feature_values": [
        {
          "values": {
            "int32_values": [28],
            "string_values": ["NYC", "premium"]
          }
        }
      ]
    }
  ]
}
```

### Retrieve JSON Format

```json
{
  "entity_label": "user",
  "feature_groups": [
    {
      "label": "user_features",
      "feature_labels": ["age", "location", "subscription_type"]
    }
  ],
  "keys_schema": ["user_id"],
  "keys": [
    {
      "cols": ["user_123"]
    },
    {
      "cols": ["user_456"]
    }
  ]
}
```

## Supported Data Types

The CLI tool supports all ONFS data types:

- **Numeric Types**: `int32_values`, `int64_values`, `uint32_values`, `uint64_values`
- **Floating Point**: `fp32_values`, `fp64_values`
- **Text**: `string_values`
- **Boolean**: `bool_values`
- **Vectors**: `vector` (nested values)

### Example with Multiple Data Types

```json
{
  "entity_label": "product",
  "keys_schema": ["product_id"],
  "feature_groups": [
    {
      "label": "product_features",
      "feature_labels": ["price", "rating", "category", "is_available", "embedding"]
    }
  ],
  "data": [
    {
      "key_values": ["prod_001"],
      "feature_values": [
        {
          "values": {
            "fp64_values": [29.99, 4.5],
            "string_values": ["electronics"],
            "bool_values": [true],
            "vector": [
              {
                "values": {
                  "fp32_values": [0.1, 0.2, 0.3, 0.4, 0.5]
                }
              }
            ]
          }
        }
      ]
    }
  ]
}
```

## Output Examples

### Persist Operation Output

```bash
$ onfs-cli -operation persist -input persist-sample.json
Persisting features for entity: user
[METRIC] onfs.grpc.invoke.latency: 45.2ms (tags: [communication_protocol:grpc external_service:online-feature-store method:/persist.FeatureService/PersistFeatures grpc_status_code:0])
[METRIC] onfs.grpc.invoke.count: 1 (tags: [communication_protocol:grpc external_service:online-feature-store method:/persist.FeatureService/PersistFeatures grpc_status_code:0])
Persist successful: Features persisted successfully
```

### Retrieve Operation Output

```bash
$ onfs-cli -operation retrieve -input retrieve-sample.json
Retrieving features for entity: user

=== Retrieve Result ===
Entity Label: user
Keys Schema: [user_id]

Feature Schemas:
  Feature Group: user_features
    - age (column 0)
    - location (column 1)
    - subscription_type (column 2)

Rows (2 total):
  Row 1:
    Keys: [user_123]
    Columns: 3 bytes total
      Column 0: [28 0 0 0]... (4 bytes)
      Column 1: [78 89 67]... (3 bytes)
      Column 2: [112 114 101 109 105 117 109]... (7 bytes)
  Row 2:
    Keys: [user_456]
    Columns: 3 bytes total
      Column 0: [35 0 0 0]... (4 bytes)
      Column 1: [83 70]... (2 bytes)
      Column 2: [98 97 115 105 99]... (5 bytes)
```

### Retrieve Decoded Operation Output

```bash
$ onfs-cli -operation retrieve-decoded -input retrieve-sample.json
Retrieving decoded features for entity: user

=== Decoded Retrieve Result ===
Keys Schema: [user_id]

Feature Schemas:
  Feature Group: user_features
    - age (column 0)
    - location (column 1)
    - subscription_type (column 2)

Decoded Rows (2 total):
  Row 1:
    Keys: [user_123]
    Columns: [28 NYC premium]
  Row 2:
    Keys: [user_456]
    Columns: [35 SF basic]
```

## Testing Workflow

1. **Start ONFS services:**
   ```bash
   cd quick-start
   docker-compose up -d
   ```

2. **Install CLI tool:**
   ```bash
   ./setup-cli.sh
   ```

3. **Persist some test data:**
   ```bash
   onfs-cli -operation persist -input sample-data/persist-sample.json
   ```

4. **Retrieve the data:**
   ```bash
   onfs-cli -operation retrieve -input sample-data/retrieve-sample.json
   ```

5. **Retrieve decoded data:**
   ```bash
   onfs-cli -operation retrieve-decoded -input sample-data/retrieve-sample.json
   ```

## Troubleshooting

### Common Issues

1. **Connection refused:**
   - Ensure ONFS service is running: `docker-compose ps`
   - Check if port 8089 is accessible: `curl http://localhost:8089/health/self`

2. **Authentication errors:**
   - Verify caller token matches service configuration
   - Default token is `test` for docker-compose setup

3. **Build errors:**
   - Ensure Go 1.22+ is installed: `go version`
   - Run `go mod tidy` in the go-sdk directory

4. **Binary not found:**
   - Check if `$HOME/bin` is in PATH: `echo $PATH`
   - Add to PATH: `export PATH="$HOME/bin:$PATH"`

### Debug Mode

Enable verbose output by checking service logs:

```bash
# Check ONFS service logs
docker logs onfs-api-server

# Check all service logs
docker-compose logs -f
```

## Sample Data Files

The setup script creates sample JSON files in `quick-start/sample-data/`:

- `persist-sample.json` - Example persist request with user data
- `retrieve-sample.json` - Example retrieve request for the same data

You can modify these files or create your own following the JSON format specifications above.

## Advanced Usage

### Batch Processing

For large datasets, adjust the batch size:

```bash
onfs-cli -operation retrieve -batch-size 100 -input large-query.json
```

### Custom Timeout

For slow networks or large requests:

```bash
onfs-cli -operation persist -timeout 60000 -input large-persist.json
```

### Different Environment

Connect to staging or production:

```bash
onfs-cli -operation retrieve \
  -host staging.example.com \
  -port 443 \
  -plaintext=false \
  -caller-token "your-production-token" \
  -input production-query.json
```

## Integration

The CLI tool can be integrated into scripts or CI/CD pipelines:

```bash
#!/bin/bash
# Example integration script

# Persist test data
if onfs-cli -operation persist -input test-data.json; then
    echo "✅ Data persisted successfully"
else
    echo "❌ Failed to persist data"
    exit 1
fi

# Verify data can be retrieved
if onfs-cli -operation retrieve -input test-query.json > /dev/null; then
    echo "✅ Data retrieval successful"
else
    echo "❌ Failed to retrieve data"
    exit 1
fi
```

## Contributing

To extend the CLI tool:

1. Modify `go-sdk/cmd/main.go`
2. Add new command line flags or operations
3. Test with `go run ./cmd/main.go -operation your-test`
4. Build and install: `go build -o onfs-cli ./cmd/main.go`

The CLI tool uses the ONFS Go SDK (`go-sdk/pkg/onfs`) for all gRPC communication, so any SDK improvements automatically benefit the CLI tool. 