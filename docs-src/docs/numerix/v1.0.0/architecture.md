---
title: Architecture
sidebar_position: 1
---

# Numerix Service - Architecture

## Problem Statement
We are currently operating a Go-based service that performs computationally intensive matrix calculations and returns the final results. This service runs on multiple pods, leading to significant infrastructure costs.

## Objective of Re-Architecture
To optimize performance and reduce operational expenses, we aim to rewrite this service in Rust, leveraging SIMD (Single Instruction, Multiple Data) to maximize efficiency and parallel computation capabilities.

### Goals
- **Performance Optimization**: Utilize Rust’s low-level control and memory efficiency to enhance computational speed.
- **Cost Reduction**: Reduce the number of pods required by improving processing efficiency.
- **Parallel Computation**: Implement SIMD to execute matrix operations faster by processing multiple data points simultaneously.
- **Reliability and Safety**: Benefit from Rust’s strong memory safety guarantees while maintaining or improving system stability.

This rewrite will involve analyzing the existing Go implementation and implementing an optimized Rust version that efficiently utilizes SIMD instructions for matrix computations.

## High-Level Design (HLD)

Numerix is a gRPC service written in Rust (Tonic + Prost) that exposes matrix computation primitives. It fetches computation expressions from `etcd`, validates requests, and executes SIMD-accelerated operations.

### Components
- **gRPC Service** (Tonic): API layer for `Compute` requests.
- **Controller/Handler**: Request validation and orchestration.
- **ConfigStore (etcd)**: Stores expressions and configs.
- **Rust Matrix Frame**: SIMD-optimized math engine (fp32/fp64).
- **Error Handler**: Consistent error mapping and propagation.
- **Observability**: Metrics and traces for latency, RPS, and errors.

### Request Flow (Sequence)
1. Client calls gRPC → `Numerix/Compute`.
2. Controller delegates to Handler.
3. Handler fetches expression from `etcd` by id.
4. Request validation; on failure → error returned.
5. Handler invokes Matrix Frame with expression + scores.
6. On success → result returned; on failure → error propagated.

### Rust Matrix Frame Flow
1. Parse expression.
2. Determine data type (fp32/fp64).
3. Execute SIMD-optimized computation path (fp32/fp64).
4. Return result or structured error.

### Tech Choices
- **Rust**: Low latency, memory safety, predictable performance.
- **Tonic**: Native async gRPC server/client.
- **Prost**: Protobuf types generation.
- **etcd client**: Config and expression store.
- **config-rs**: Layered configuration.
- **OpenTelemetry** (optional): Tracing/metrics integration.

### gRPC Schema (excerpt)
```protobuf
service Numerix {
  rpc Compute(NumerixRequestProto) returns (NumerixResponseProto);
}
```

### Example gRPC Call (grpcurl)
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

---

## Why SIMD?

**SIMD (Single Instruction, Multiple Data)** is a parallel computing technique where a single instruction operates on multiple data points simultaneously, enabling significant speedups in matrix operations.

### Scalar vs SIMD Example (Rust)
```rust
fn add_arrays(a: &[i32], b: &[i32]) -> Vec<i32> {
    let mut result = Vec::new();
    for i in 0..a.len() {
        result.push(a[i] + b[i]);
    }
    result
}
```

```rust
#[cfg(target_arch = "aarch64")]
use core::arch::aarch64::*;
fn add_simd_in_place(a: &[f64], b: &[f64], result: &mut [f64]) {
    let step = 2;
    let simd_end = (a.len() / step) * step;
    unsafe {
        for i in (0..simd_end).step_by(step) {
            let a_vec = vld1q_f64(a.as_ptr().add(i));
            let b_vec = vld1q_f64(b.as_ptr().add(i));
            let sum = vaddq_f64(a_vec, b_vec);
            vst1q_f64(result.as_mut_ptr().add(i), sum);
        }
    }
    for i in simd_end..a.len() {
        result[i] = a[i] + b[i];
    }
}
```

### Architecture SIMD Support
- x86_64: SSE/AVX/AVX2/AVX-512 via `std::arch::x86_64::*`.
- AArch64: NEON/SVE via `std::arch::aarch64::*`.

---

## Observability & Monitoring
- Latency per Compute ID (P99.9/P99/P95/P50)
- Total Latency (P99.9/P99/P95/P50)
- Error Rate (Computation Failures)
- Requests Per Second (RPS)
- Service 5xx Count
- ETCD 5xx Count
- ETCD Latency (P99.9/P99/P95/P50)

---

## Scalability
The service is deployed on K8s (e.g., GKE) with CPU-based autoscaling.

---

## Libraries
- Rust core + `std::simd`
- Tonic (gRPC)
- Prost (Protobuf)
- etcd client
- config-rs
- Actix-web (optional HTTP endpoints)
- OpenTelemetry (optional)


