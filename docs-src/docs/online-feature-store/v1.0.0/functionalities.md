---
title: Key Functionalities
sidebar_position: 4
---

# Online Feature Store - Key Functionalities

## Overview

The BharatML Online Feature Store is a high-performance, production-ready system designed to serve machine learning features with **sub-10ms P99 latency** and **1M+ RPS capacity**. It bridges the gap between offline feature engineering and real-time model inference.

## 🚀 Core Capabilities

### **Real-Time Feature Serving**
- **Ultra-Low Latency**: Sub-10ms P99 response times
- **High Throughput**: Tested at 1M+ requests per second with 100 IDs per request
- **Batch Retrieval**: Fetch multiple features for multiple entities in a single request
- **Point-in-Time Consistency**: Ensure feature consistency across model predictions

### **Multi-Format Data Support**
Supports all common ML data types with optimized serialization:

| Data Type | Support | Use Cases |
|-----------|---------|-----------|
| **Integers** | int8, int16, int32, int64 | User IDs, counts, categorical encodings |
| **Floats** | float16, float32, float64 | Continuous features, embeddings, scores |
| **Strings** | Variable length | Categories, text features, metadata |
| **Booleans** | Bit-packed | Feature flags, binary indicators |
| **Vectors** | All above types | Embeddings, feature arrays, time series |

### **Multi-Database Backend**
Flexible storage options for different deployment needs:

- **🔥 Scylla DB**: Ultra-high performance NoSQL (recommended for production)
- **⚡ Dragonfly**: Modern Redis alternative with better memory efficiency  
- **📊 Redis**: Standard in-memory store for development and small-scale deployments

## 🎯 Key Features

### **Performance Optimizations**
- **Custom PSDB Format**: Proprietary serialization format optimized for ML features
- **Object Pooling**: Memory-efficient resource reuse
- **Connection Pooling**: Optimized database connection management
- **Compression Support**: Multiple algorithms (LZ4, Snappy, ZSTD) with intelligent fallback

### **Data Management**
- **TTL Support**: Automatic feature expiration with configurable time-to-live
- **Versioning**: Multiple feature schema versions with backward compatibility
- **Batch Operations**: Efficient bulk read/write operations
- **Feature Groups**: Logical grouping of related features for better organization

### **Developer Experience**
- **gRPC API**: High-performance, language-agnostic interface
- **Go SDK**: Native Go client with connection pooling and error handling
- **Python SDK**: ML-friendly Python bindings for data scientists
- **RESTful Interface**: HTTP API for web applications and testing

### **Production Ready**
- **Health Checks**: Built-in monitoring and health endpoints
- **Metrics Integration**: DataDog, Prometheus-compatible metrics
- **Structured Logging**: JSON-formatted logs with configurable levels
- **Graceful Shutdown**: Clean resource cleanup on termination

## 📊 Use Cases

### **Real-Time ML Inference**
```go
// Fetch user features for recommendation model
query := &onfs.Query{
    EntityLabel: "user",
    FeatureGroups: []onfs.FeatureGroup{
        {
            Label:         "demographics",
            FeatureLabels: []string{"age", "location", "income"},
        },
        {
            Label:         "behavior", 
            FeatureLabels: []string{"click_rate", "purchase_history"},
        },
    },
    KeysSchema: []string{"user_id"},
    Keys: []onfs.Keys{
        {Cols: []string{"user_123"}},
    },
}

result, err := client.RetrieveFeatures(ctx, query)
```

### **Batch Feature Serving**
```go
// Bulk feature retrieval for model training
query := &onfs.Query{
    EntityLabel: "transaction",
    FeatureGroups: []onfs.FeatureGroup{
        {
            Label:         "transaction_history",
            FeatureLabels: []string{"amount", "frequency", "merchant_type"},
        },
        {
            Label:         "risk_scores",
            FeatureLabels: []string{"fraud_score", "credit_score"},
        },
    },
    KeysSchema: []string{"transaction_id"},
    Keys: []onfs.Keys{
        {Cols: []string{"txn_001"}},
        {Cols: []string{"txn_002"}},
        // ... 100s of transaction IDs
    },
}

result, err := client.RetrieveFeatures(ctx, query)
```

### **A/B Testing Support**
```go
// Version-aware feature retrieval with decoded values
query := &onfs.Query{
    EntityLabel: "experiment",
    FeatureGroups: []onfs.FeatureGroup{
        {
            Label:         "model_features_v2", // Specific version
            FeatureLabels: []string{"feature_a", "feature_b", "feature_c"},
        },
    },
    KeysSchema: []string{"user_id"},
    Keys: []onfs.Keys{
        {Cols: []string{"user_123"}},
    },
}

// Get string-decoded values for easier debugging/analysis
decodedResult, err := client.RetrieveDecodedFeatures(ctx, query)
```

## 🎛️ Configuration Options

### **Performance Tuning**
- **Cache TTL**: Configure feature freshness requirements

### **Storage Configuration**
- **Database Selection**: Choose backend based on scale and requirements
- **Replication Factor**: Set availability vs consistency trade-offs
- **Partition Strategy**: Optimize data distribution
- **Backup Frequency**: Configure data durability requirements

### **Monitoring & Observability**
- **Metrics Collection**: Request rates, latencies, error rates
- **Custom Dashboards**: Feature-specific monitoring views

## 📈 Production Deployment

### **Recommended Architecture**
- **Load Balancer**: Distribute traffic across multiple instances
- **Feature Store Cluster**: 3+ instances for high availability
- **Database Cluster**: Replicated backend with automatic failover
- **Monitoring Stack**: Metrics, logs, and alerting infrastructure

### **Scaling Guidelines**
- **Horizontal Scaling**: Add more feature store instances
- **Database Scaling**: Increase partition count or upgrade hardware
- **Cache Warming**: Pre-load frequently accessed features
- **Connection Tuning**: Optimize pool sizes for your traffic patterns

---

*The Online Feature Store is designed to be the high-performance bridge between your offline feature engineering and real-time ML inference needs.*
