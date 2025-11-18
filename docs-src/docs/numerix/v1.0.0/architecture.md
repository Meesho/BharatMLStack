---
title: Architecture
sidebar_position: 1
---

# BharatMLStack - Numerix

---

Numerix is a Rust-based compute service in **BharatMLStack** designed for low-latency evaluation of mathematical expressions over feature matrices. Each request carries a compute_id and a matrix of features; Numerix fetches the corresponding postfix expression, maps variables to feature columns (treated as vectors), and evaluates the expression with a stack-based SIMD-optimized runtime.

---

## High-Level Components

- **Tonic gRPC server (Rust)**: exposes `Numerix/Compute` for low-latency requests.
  - Accepts feature data as strings (for ease of use) or byte arrays (for efficient transmission).
  - All input data is converted internally to fp32 or fp64 vectors for evaluation.
- **Compute Registry (etcd)**: stores `compute_id (int) → postfix expression` mappings.
- **Stack-based Evaluator**: Runs postfix expressions in linear time using a stack based approach over aligned vectors.
- **Vectorized Math Runtime**: No handwritten SIMD intrinsics; relies on **LLVM autovectorization**.  
  - Operations are intentionally simple and memory-aligned.  
  - Compiler emits SIMD instructions automatically.  
  - Portable across CPU architectures (ARM & AMD). 
- **Metrics and Health**  
  - Latency, RPS, and error rates via **Datadog/DogStatsD** UDP client.  
  - Minimal HTTP endpoints (`/health`, optional `/metrics`) for diagnostics.

---

## What is SIMD?

SIMD (Single Instruction, Multiple Data) is a CPU feature that allows a single instruction to operate on multiple data points at once. In Numerix, this means that operations on feature vectors can be executed in parallel, making evaluation of mathematical expressions faster and more predictable.

## Why SIMD Matters for Numerix

- Postfix expressions operate on vectors (columns of the input matrix).
- SIMD allows multiple elements of these vectors to be processed in one CPU instruction, rather than element-by-element.
- This results in low-latency, high-throughput computation without the need for handwritten intrinsics — the compiler handles the vectorization automatically.

---

## Why ARM, Why LLVM

During design exploration, we tested SIMD on different architectures and found **ARM (AArch64)** with NEON/SVE/SVE2 provided excellent performance for our workloads.  

Instead of writing custom intrinsics, Numerix **compiles with SIMD flags** and lets LLVM handle vectorization:  

```bash
RUSTFLAGS="-C target-feature=+neon,+sve,+sve2" \
cargo build --release --target aarch64-unknown-linux-gnu
```

- This approach works well because operations are straightforward, data is aligned, and compiler auto-vectorization is reliable.

- AMD/x86 builds are equally supported — enabling their SIMD extensions is just a matter of changing build flags.

## Request Model and Flow

1. Client calls gRPC `numerix.Numerix/Compute` with:
   - `schema`: ordered feature names
   - `entity_scores`: per-entity vectors (string or bytes)
   - `compute_id`: integer identifier for the expression
   - `data_type` (optional): e.g., `fp32` or `fp64`
2. Service fetches the postfix expression for `compute_id` which was pre-fetched from `etcd`.
3. Request is validated for schema and data shape.
4. The stack-based evaluator executes the expression in O(n) over tokens, with vectorized inner operations.
5. Response returns `computation_score_data` or a structured `error`.

---

## Why Postfix Expressions

- **Stored in etcd** as postfix (Reverse Polish) notation.
- Postfix makes evaluation parser-free and linear time.
- Execution uses a stack machine:
  - Push operands (feature vectors).
  - Pop, compute, and push results for each operator.
- Benefits: predictable runtime, compiler-friendly loops, cache efficiency.

## gRPC Interface

- **Service:** `numerix.Numerix`
- **RPC:** `Compute(NumerixRequestProto) → NumerixResponseProto`
- **Request fields:** `schema`, `entity_scores`, `compute_id`, optional `data_type`
- **Response fields:** `computation_score_data` or `error`

Example (grpcurl):
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

## Observability

- **Datadog (DogStatsD)** metrics publication via UDP client:
  - Latency (P50/P95/P99), error rate, RPS, internal failures
  - Configurable sampling rate via environment variables
- Optional `/metrics` HTTP endpoint can be enabled for local debugging.

---

## Environments

- Kubernetes (K8s), including GKE and EKS
- Multi-arch builds: amd64, arm64.
- ARM builds ship with NEON/SVE/SVE2 enabled.

---

## Key Takeaways

- Minimal service surface: **gRPC + etcd**.
- **No custom intrinsics** — portable across **ARM & AMD** via compiler flags.
- Supports both string and byte input, internally converted to aligned fp32/fp64 vectors.
- **Stack-based postfix evaluation** : linear time, cache-friendly.
- Predictable, ultra-low-latency performance.

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

