# BharatMLStack Feature Store APIs

This document describes how to run the REST APIs for all three callers that expose the feature store functionality.

## Overview

All three callers now expose the same REST API endpoint:
- **Endpoint**: `POST /retrieve-features`
- **Response**: JSON with success status, data, and message
- **Function**: Retrieves features from the BharatML feature store using the same parameters as the original main methods

## Services and Ports

| Service | Technology | Port | URL |
|---------|------------|------|-----|
| Java Caller | Spring Boot | 8082 | http://localhost:8082/retrieve-features |
| Rust Caller | Axum | 8080 | http://localhost:8080/retrieve-features |
| Go Caller | Gin | 8081 | http://localhost:8081/retrieve-features |

## Running the APIs

### Java Caller (Port 8082)
```bash
cd java-caller
mvn clean package
java -jar target/feature-store-java-1.0.0.jar
```

### Rust Caller (Port 8080)
```bash
cd rust-caller
cargo run
```

### Go Caller (Port 8081)
```bash
cd go-caller
go run main.go
```

## Testing the APIs

You can test any of the APIs using curl:

```bash
# Test Java API
curl -X POST http://localhost:8082/retrieve-features

# Test Rust API
curl -X POST http://localhost:8080/retrieve-features

# Test Go API
curl -X POST http://localhost:8081/retrieve-features
```

## API Response Format

All APIs return the same JSON structure:

```json
{
    "success": true,
    "data": "feature_store_result_data",
    "message": "Features retrieved successfully"
}
```

On error:
```json
{
    "success": false,
    "error": "error_message",
    "message": "Failed to retrieve features"
}
```

## Dependencies Added

### Java (Spring Boot)
- `spring-boot-starter-web`: Web framework
- `spring-boot-starter`: Core Spring Boot

### Rust (Axum)
- `axum`: Web framework
- `serde`: JSON serialization
- `tower-http`: CORS support

### Go (Gin)
- `gin-gonic/gin`: Web framework with built-in CORS support

## Architecture

Each API service:
1. Wraps the original feature store client functionality
2. Exposes it via HTTP REST endpoint
3. Returns standardized JSON responses
4. Includes error handling and CORS support
5. Maintains the same authentication and query parameters as the original implementations
