---
slug: post-five
title: "LLM Inference Optimization Techniques: Engineering Sub-Second Latency at Scale"
authors: [jaya]
date: 2025-6-2
tags: [llm, vllm, tensorrt-llm, mlplatform, meesho, bharatmlstack]
---

![BharatMLStack](./bms.png)
## LLM Inference Optimization Techniques: Engineering Sub-Second Latency at Scale

Raw execution of Large Language Models is inherently expensive and memory-intensive. To achieve sub-second latency and high throughput, we implement a multi-layered optimization strategy that targets the entire inference stack—from memory management to kernel execution.

## 1. Advanced Memory Management: Paged & Prefix KV Caching

The most significant bottleneck in LLM inference is not always compute, but memory bandwidth—specifically managing the Key-Value (KV) cache.

### Paged KV caching

Standard caching suffers from fragmentation. We use **Paged KV caching**, which operates similarly to an operating system's virtual memory: the KV cache is divided into non-contiguous blocks. This lets us serve larger batch sizes without running out of memory.

### KV cache quantization

To further maximize available memory, we implement **KV cache quantization** (e.g., FP8). By compressing stored attention keys and values from 16-bit to 8-bit, we nearly double the effective context window capacity of the GPU, allowing longer conversations or larger batches without materially degrading quality.

### Prefix caching (the "voice bot" optimizer)

For use cases like GenAI voice bots where the system prompt (e.g., "You are a helpful assistant...") is static across thousands of requests, we enable **prefix caching**.

- **Impact**: By reusing pre-computed KV states for common prefixes, we achieve a cache hit rate of ~90%. This reduces **Time To First Token (TTFT)** by skipping redundant computation of the system prompt.

## 2. Aggressive Quantization (INT4 AWQ & FP8)

Running models in their native 16-bit precision (BF16) restricts maximum batch size and throughput. We use quantization to shrink model weights without sacrificing accuracy.

### INT4 AWQ (Activation-aware Weight Quantization)

For the Llama 3 family, we use **AWQ** to compress weights to 4 bits. This reduces model size by ~75%, allowing larger models to fit into L4 GPU memory and significantly improving token generation speed.

### FP8 precision

For NVIDIA Hopper (H100) architectures, we are exploring **FP8 quantization**, leveraging native FP8 tensor cores to accelerate matrix multiplications while maintaining a higher dynamic range than integer quantization.

- **Verification**: We validate quantized models by comparing dot-product similarity of embeddings against the FP16 baseline, consistently achieving **>99% similarity**.

## 3. Kernel Fusion & Custom Plugins

To minimize overhead from launching thousands of small GPU operations, we fuse them into monolithic kernels using NVIDIA TensorRT plugins.

- **Flash attention & FMHA**: We enable **Fused Multi-Head Attention (FMHA)** combined with flash attention to reduce memory reads/writes.
- **GEMM plugins**: We use specialized **GEMM** plugins to accelerate transformer linear layers.
- **Removing input padding**: Instead of padding short sequences to match the longest, we remove input padding so the GPU processes only valid tokens.

## 4. Inflight (Continuous) Batching

Traditional static batching waits for all requests in a batch to finish before returning results—so one long response delays everyone else.

We implement **inflight batching**: as soon as one request completes, its slot is freed and filled by a new request from the queue. This keeps GPUs saturated and decouples latency of short queries from long ones.

## 5. Parallelism Strategies: Scaling Beyond One GPU

For large models (e.g., 70B+ parameters) that cannot fit into the VRAM of a single GPU, we use parallelism strategies.

- **Tensor parallelism (TP)**: Split weight matrices across multiple GPUs (e.g., 4× L4 or 8× A100). Each GPU computes a shard and outputs are reduced at every layer.
- **Pipeline parallelism (PP)**: Split model layers across GPUs to pipeline compute (e.g., while one GPU computes later layers for Request A, another starts early layers for Request B).

## 6. Speculative Decoding

To reduce inter-token latency (ITL), we explore **speculative decoding**.

- **Mechanism**: A smaller, faster "draft" model speculatively generates a short token sequence (e.g., 5 tokens).
- **Verification**: The larger target model verifies those tokens in one parallel forward pass. If correct, we effectively generate multiple tokens per large-model step; if not, we discard and regenerate. This is effective for predictable text, improving perceived generation speed.

## Few Benchmarks

Below are a couple of representative use cases and performance numbers.

### Search query rewriting

- **LLM**: Fine-tuned llama-3.2-1B
- **Input & output token length**: ~10–20
- **Response type**: Non-streaming

| Inference runtime | Hardware                 | Max requests/sec | Max p99 latency |
| --- | --- | ---: | ---: |
| TensorRT-LLM      | 4 × L4 GPUs (multi-GPU)  | 1000             | 95 ms           |
| TensorRT-LLM      | 1 × A100 40 GB GPU       | 1000             | 69 ms           |

### Voice bot query

- **LLM**: Llama-3.1-8B
- **Input token length**: ~1900–2000
- **Output token length**: ~200
- **Response type**: Streaming

| Inference runtime | Concurrency | p99 TTFT (ms) | p99 ITL (ms) | Token throughput (tokens/sec) | Request throughput (req/sec) | Hardware |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| TensorRT-LLM | 1   | 36.27   | 22.78  | 45.66    | 0.23  | L4   |
| TensorRT-LLM | 2   | 49.81   | 23.21  | 89.37    | 0.45  | L4   |
| TensorRT-LLM | 4   | 55.33   | 36.62  | 153.39   | 0.78  | L4   |
| TensorRT-LLM | 8   | 66.5    | 39.11  | 279.88   | 1.47  | L4   |
| TensorRT-LLM | 16  | 131.8   | 30.39  | 547.8    | 2.77  | L4   |
| TensorRT-LLM | 32  | 277.22  | 48.02  | 925.7    | 4.78  | L4   |
| TensorRT-LLM | 64  | 498.52  | 71.62  | 1,164.40 | 6.2   | L4   |
| TensorRT-LLM | 128 | 677.31  | 120.37 | 1,445.18 | 7.69  | L4   |
| TensorRT-LLM | 256 | 1,926.31 | 216.88 | 1,600.81 | 8.52  | L4   |
| TensorRT-LLM | 1   | 21.17   | 9.24   | 130.05   | 0.68  | A100 |
| TensorRT-LLM | 2   | 25.78   | 9.21   | 264.5    | 1.35  | A100 |
| TensorRT-LLM | 4   | 28.52   | 10.99  | 437.69   | 2.27  | A100 |
| TensorRT-LLM | 8   | 34.4    | 12.61  | 760.49   | 3.96  | A100 |
| TensorRT-LLM | 16  | 68.03   | 14.32  | 1,343.80 | 7.01  | A100 |
| TensorRT-LLM | 32  | 185.96  | 16.82  | 2,287.30 | 11.92 | A100 |
| TensorRT-LLM | 64  | 136.87  | 21.17  | 3,625.22 | 18.89 | A100 |
| TensorRT-LLM | 128 | 463.78  | 34.15  | 4,456.51 | 23.24 | A100 |
| TensorRT-LLM | 256 | 890.12  | 59.18  | 5,188.24 | 27.05 | A100 |

## Conclusion

High-performance LLM inference is fundamentally a systems engineering problem: memory efficiency, kernel execution, batching strategy, and parallelism determine real-world latency and throughput. Techniques such as paged KV caching, aggressive quantization, kernel fusion, and inflight batching improve GPU utilization while reducing latency and memory pressure.

These optimizations enable the platform to deliver sub-second responses, sustain high concurrency, and efficiently serve both lightweight and long-context workloads. By continuously optimizing across the full inference stack, we keep LLM serving scalable, cost-efficient, and production-ready for real-time AI applications.
