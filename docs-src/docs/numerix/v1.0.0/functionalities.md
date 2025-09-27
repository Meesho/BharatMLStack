---
title: Key Functionalities
sidebar_position: 3
---

# Key Functionalities

## gRPC API

### Service
```protobuf
service Numerix {
  rpc Compute(NumerixRequestProto) returns (NumerixResponseProto);
}
```

### Example Call
```bash
grpcurl -plaintext \
  -import-path ./numerix/src/protos/proto \
  -proto numerix.proto \
  -d '{
    "entityScoreData": {
      "schema": ["feature1", "feature2"],
      "entityScores": [ { "stringData": { "values": ["1.0", "2.0"] } } ],
      "computeId": "test_computation",
      "dataType": "fp32"
    }
  }' \
  localhost:8080 numerix.Numerix/Compute
```

## SIMD-Accelerated Matrix Ops
- Vectorized fp32/fp64 arithmetic using NEON (AArch64) and SSE/AVX (x86-64).
- Expression parsing and efficient execution plans.

## Config & Storage
- `etcd` used as configuration and expression source.
- Layered configuration via `config-rs`.

## Observability
- Metrics for latency (P50/P95/P99/P99.9) and RPS.
- Error rate and etcd call metrics.

## Libraries
- Tonic (gRPC), Prost (Protobuf), etcd client, config-rs, OpenTelemetry (optional).


