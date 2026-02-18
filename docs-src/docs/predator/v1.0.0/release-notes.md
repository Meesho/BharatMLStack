---
title: Release Notes
sidebar_position: 3
---

# Predator - Release Notes

## Version 1.0.0

**Release Date**: June 2025  
**Status**: General Availability (GA)

First stable release of **Predator** — scalable model inference service built around **NVIDIA Triton Inference Server**, part of BharatMLStack. Serves Deep Learning and tree-based models with low latency in **Kubernetes**; integrates with **OnFS** and **Interflow**; clients use the **Helix client** over gRPC.

### What's New

- **Triton inference engine**: Unified runtime for DL and tree-based models on CPU/GPU; model repository via Init Container from GCS; gRPC API via Helix client.
- **Multi-backend support**: TensorRT, PyTorch, ONNX Runtime, TensorFlow, Python, FIL, DALI, Custom.
- **Dynamic batching & concurrency**: Configurable via `config.pbtxt`; model versioning and ensembles.
- **Kubernetes deployment**: Helm-based; Init Container + Triton container; custom Triton images from Artifact Registry; health probes; CPU/GPU autoscaling.
- **Observability**: Prometheus metrics, Grafana; warmup requests for cold-start avoidance.
