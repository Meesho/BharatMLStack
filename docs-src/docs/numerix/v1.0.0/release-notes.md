---
title: Release Notes
sidebar_position: 4
---

# Numerix - Release Notes

## v1.0.0

### Highlights
- Initial Rust-based Numerix service with gRPC API (`Compute`).
- SIMD-optimized matrix operations (fp32/fp64) with NEON/AVX support.
- `etcd`-backed expression/config retrieval.
- Basic observability: latency histograms, error rates, RPS.

### Breaking Changes
- HTTP `/compute` endpoint removed; gRPC is the primary interface.

### Known Issues
- OpenTelemetry exporter integration TBD.
- Further kernel-tuned optimizations planned for large matrix shapes.

### Upgrade Notes
- Update clients to use gRPC (`numerix.Numerix/Compute`).
- Ensure proto compatibility and generated client code via Prost/Tonic.


