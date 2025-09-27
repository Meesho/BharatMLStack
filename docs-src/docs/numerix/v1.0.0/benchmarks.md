---
title: Benchmarks
sidebar_position: 2
---

# Benchmarks (PoC)

This PoC measures the performance of **vector addition** in Rust **with and without compiler SIMD optimizations**. Requests consist of repeated fixed-size vector addition operations processed in parallel by the CPU. These results provide perspective on **how much faster SIMD makes vectorized computations**, and similar improvements are expected for other vectorized operations in Numerix.

## System Configuration

- **Instance Type**: c4a-highcpu-16  
- **Processor**: Google Axion (ARMv9, 64-bit)  
- **SIMD Extension**: SVE2  
- **OS**: Linux (Ubuntu 22.04)  
- **Rust Version**: rustc 1.80.0  
- **Target Triple**: aarch64-unknown-linux-gnu  


## Vector Addition Performance

### With SIMD

| Vector Dim | ns per op | Iterations       | Throughput (GiB/s) | Total CPU (raw) | Total CPU (normalized) |
|-----------:|----------:|----------------:|-------------------:|----------------:|-----------------------:|
| 10         | 0.39626 ns| 170,057,457,941 | 376.04             | 1564%           | 97.75%                 |
| 50         | 0.6641 ns | 94,342,709,095  | 1121.9             | 1590%           | 99.38%                 |
| 100        | 1.1522 ns | 51,705,835,397  | 1286.9             | 1560%           | 97.50%                 |
| 500        | 5.0649 ns | 12,061,753,661  | 1471               | 1538%           | 96.12%                 |
| 1000       | 9.648 ns  | 6,488,848,705   | 1544.5             | 1570%           | 98.12%                 |
| 5000       | 52.925 ns | 1,169,316,813   | 1407.8             | 1590%           | 99.38%                 |
| 10000      | 114.68 ns | 555,779,981     | 1299.4             | 1592%           | 99.50%                 |
| 50000      | 644.60 ns | 94,372,153      | 1155.9             | 1560%           | 97.50%                 |
| 100000     | 1.4530 µs | 42,502,201      | 1025.5             | 1526%           | 95.38%                 |

### Without SIMD

| Vector Dim | ns per op | Iterations      | Throughput (GiB/s) | Total CPU (raw) | Total CPU (normalized) |
|-----------:|----------:|---------------:|-------------------:|----------------:|-----------------------:|
| 10         | 3.196 ns  | 1,000,000,000  | 25.03              | 1313%           | 82.06%                 |
| 50         | 3.866 ns  | 1,000,000,000  | 103.46             | 1417%           | 88.56%                 |
| 100        | 5.867 ns  | 1,000,000,000  | 136.35             | 1495%           | 93.44%                 |
| 500        | 19.25 ns  | 1,000,000,000  | 207.81             | 1600%           | 100.00%                |
| 1000       | 33.91 ns  | 1,000,000,000  | 235.92             | 1600%           | 100.00%                |
| 5000       | 162.1 ns  | 448,785,386    | 246.71             | 1600%           | 100.00%                |
| 10000      | 332.0 ns  | 208,428,151    | 240.94             | 1600%           | 100.00%                |
| 50000      | 1,740 ns  | 39,247,646     | 229.93             | 1600%           | 100.00%                |
| 100000     | 3,401 ns  | 19,598,293     | 235.24             | 1600%           | 100.00%                |

> Normalization: Total CPU (normalized) = Total CPU (raw) / 16, since 1600% equals full utilization on a 16‑core machine.

## Observations
- **SIMD provides large speedups across all vector sizes**: overall throughput improvements range from roughly **4–15×** versus Without SIMD.
- For small vectors (10–100), throughput gains are about **9–15×**, with ns/op reduced proportionally.
- For larger vectors (500–100000), speedups stabilize around **~4–7×** as memory bandwidth pressure increases.
- **CPU saturation**: Without SIMD reaches 100% normalized CPU at and beyond 500 elements, whereas With SIMD typically operates at ~95–99% normalized CPU yet delivers substantially higher throughput at similar CPU.
- **Per‑CPU efficiency**: With SIMD, throughput per unit of CPU is much higher, reflecting better vector unit utilization and fewer instructions per element.
- Absolute values depend on hardware and load; the relative differential reflects the benefit of compiler SIMD optimizations.

> ⚠ Note: Absolute numbers depend on CPU frequency, memory locality, and system load. These results are meant 
to show relative SIMD benefits.
