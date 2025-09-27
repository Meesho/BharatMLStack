---
title: Key Functionalities
sidebar_position: 3
---

# Numerix — Key Functionalities

## Overview

Numerix evaluates mathematical expressions over feature matrices with a simple, low-latency gRPC surface. Each request references a `compute_id`; Numerix resolves a postfix expression, maps variables to input columns, and evaluates it over fp32/fp64 vectors with compiler-assisted SIMD.

## 🚀 Core Capabilities

### Expression Evaluation
- **Postfix execution**: Linear-time, stack-based evaluation over aligned vectors.
- **Vectorized math**: Compiler autovectorization (NEON/SVE on ARM, SSE/AVX on x86) — no handwritten intrinsics.
- **Typed compute**: Inputs converted to `fp32` or `fp64` for predictable performance.

### Input Formats
- **Strings**: Easy-to-produce feature values (converted internally).
- **Bytes**: Efficient wire format for high-throughput paths.

### Request Patterns
- **Single entity or batch**: Multiple `entity_scores` per call.
- **Schema-driven**: Column order in `schema` drives variable mapping in expressions.

## 🎯 Developer Experience

- **gRPC API**: Simple single RPC — `Numerix/Compute`.
- **Protobuf schema**: Language-agnostic client generation.
- **Deterministic behavior**: No parsing at request time; expression resolved from registry.

### gRPC Service
```protobuf
service Numerix {
  rpc Compute(NumerixRequestProto) returns (NumerixResponseProto);
}
```

### Example Call (grpcurl)
```bash
grpcurl -plaintext \
  -import-path ./numerix/src/protos/proto \
  -proto numerix.proto \
  -d '{
    "entityScoreData": {
      "schema": ["feature1", "feature2"],
      "entityScores": [ { "stringData": { "values": ["1.0", "2.0"] } } ],
      "computeId": "1001",
      "dataType": "fp32"
    }
  }' \
  localhost:8080 numerix.Numerix/Compute
```

## 📊 Observability

- **Datadog/DogStatsD**: Latency (P50/P95/P99), RPS, error rate via UDP client.
- Optional `/metrics` endpoint for local/adhoc debugging.

## ⚙️ Configuration & Registry

- **etcd registry**: `compute_id (int) → postfix expression` mapping.
- **Environment-driven config**: endpoints, timeouts, sampling rate.

## 🧪 Example Scenarios

### Batched evaluation
- Submit multiple entities in one call to reduce RPC overhead; evaluate the same `compute_id` across all rows.

### Mixed input formats
- Start with string inputs for ease; migrate to bytes for performance without changing the expression or API.

## 🔧 Tuning Knobs

- **Data type**: choose `fp32` (speed) vs `fp64` (precision).
- **Batch size**: tune number of entities per call for your p99 vs throughput goals.
- **Sampling rate**: adjust Datadog metric sampling to balance signal vs cost.

---

## Contributing

We welcome contributions from the community! Please see our [Contributing Guide](https://github.com/Meesho/BharatMLStack/blob/main/CONTRIBUTING.md) for details on how to get started.

## Community & Support

- 💬 **Discord**: Join our [community chat](https://discord.gg/XkT7XsV2AU)
- 🐛 **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/Meesho/BharatMLStack/issues)
- 📧 **Email**: Contact us at [ml-oss@meesho.com](mailto:ml-oss@meesho.com )

## License

BharatMLStack is open-source software licensed under the [BharatMLStack Business Source License 1.1](https://github.com/Meesho/BharatMLStack/blob/main/LICENSE.md).

---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
<div align="center">
  <strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong>
</div>

