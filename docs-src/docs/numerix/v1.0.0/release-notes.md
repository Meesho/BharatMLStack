---
title: Release Notes
sidebar_position: 5
---

# Numerix - Release Notes

## Version 1.0.0 🚀
**Release Date**: September 2025  
**Status**: General Availability (GA)

The first stable release of **Numerix** — a Rust-based compute service for evaluating mathematical expressions over feature matrices with very low latency. Numerix executes postfix expressions from an etcd-backed registry using a stack-based evaluator and compiler-assisted SIMD.

---

## 🎯 What's New

### Core Engine
- **Postfix Expression Execution**: `compute_id → postfix` mapping in etcd; parser-free request path.
- **Stack-Based Evaluator**: Linear-time execution over aligned vectors for predictable latency.
- **Compiler-Assisted SIMD**: Relies on LLVM autovectorization (NEON/SVE on ARM; SSE/AVX on x86); no handwritten intrinsics.
- **Typed Evaluation**: Internal conversion to `fp32`/`fp64` for consistent performance/precision.

### API Surface
- **gRPC**: Single RPC — `numerix.Numerix/Compute`.
- **Input Formats**: Strings for ease, bytes for performance; both map to vectorized math internally.

### Observability
- **Datadog/DogStatsD** metrics: Latency (P50/P95/P99), RPS, error rate.
- Minimal HTTP diagnostics: `/health` (and optional `/metrics`).

---

## 🚀 Performance & Optimization

- **Autovectorized Loops**: Tight loops over contiguous memory enable the compiler to emit SIMD instructions automatically.
- **ARM Focus Option**: Excellent results with AArch64; builds can enable NEON/SVE/SVE2:
```bash
RUSTFLAGS="-C target-feature=+neon,+sve,+sve2" \
cargo build --release --target aarch64-unknown-linux-gnu
```
- **Deterministic Runtime**: No dynamic parsing in hot path; O(n) across tokens with vectorized inner ops.

---

## 🛠️ APIs

### gRPC
```protobuf
service Numerix {
  rpc Compute(NumerixRequestProto) returns (NumerixResponseProto);
}
```

Example call:
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

---

## 🏗️ Deployment & Configuration

### Environment
```bash
APPLICATION_PORT=8083
APP_ENV=prd
APP_LOG_LEVEL=ERROR
APP_NAME=numerix

# Performance
CHANNEL_BUFFER_SIZE=10000

# etcd
ETCD_SERVERS=127.0.0.1:2379

# Metrics
METRIC_SAMPLING_RATE=1
TELEGRAF_UDP_HOST=127.0.0.1
TELEGRAF_UDP_PORT=8125
```

### Containers
- Multi-arch images: `linux/amd64`, `linux/arm64`.
- Build targets example: `x86_64-unknown-linux-gnu`, `aarch64-unknown-linux-gnu`.

---

## 🔄 Compatibility

- **Clients**: Any language with gRPC + generated stubs.
- **Architectures**: amd64 and arm64; ARM builds can enable NEON/SVE/SVE2.

---

## 🐛 Known Issues

1. Introduce a configurable log sampling rate to reduce pod memory usage during computation errors.

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



