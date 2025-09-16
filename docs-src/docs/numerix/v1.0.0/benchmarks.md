---
title: Benchmarks
sidebar_position: 2
---

# Benchmarks (PoC)

## System Configuration
- **Processor**: Apple M3
- **Architecture**: ARMv8.6-A (64-bit)
- **SIMD Extension**: NEON
- **OS**: macOS Sonoma (14.x)
- **Rust Version**: rustc 1.76.0
- **Target Triple**: aarch64-apple-darwin

## Vector Addition (SIMD vs Scalar)

### SIMD (NEON f64)

| Elements | Time |
|---:|---:|
| 1,000 | 142.42 ns |
| 10,000 | 1.5365 µs |
| 1,000,000 | 215.55 µs |
| 10,000,000 | 2.2489 ms |

> Measurements are representative PoC numbers; actual performance depends on data locality, alignment, and CPU frequency scaling.

## Notes
- SIMD speedups grow with vectorizable workloads and aligned memory.
- fp32 often yields higher throughput due to lane width vs fp64.
- Ensure feature flags and target-cpu optimizations are enabled in release builds.


