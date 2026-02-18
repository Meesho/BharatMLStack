---
title: Key Functionalities
sidebar_position: 2
---

# Predator - Key Functionalities

## Overview

Predator is a scalable, high-performance model inference service built as a wrapper around **NVIDIA Triton Inference Server**. It serves Deep Learning and tree-based models with low latency in **Kubernetes**, integrates with the **Online Feature Store (OnFS)** and uses **Interflow** for orchestration between clients, feature store, and inference engine. Clients send inference requests via the **Helix client** over gRPC.

---

## Core Capabilities

### Multi-Backend Inference

Predator leverages Triton's pluggable backends so you can serve a variety of model types from a single deployment:

| Backend | Use Case |
|---------|----------|
| **TensorRT** | GPU-optimized DL; serialized engines (FP16/INT8) |
| **PyTorch** | Native PyTorch via LibTorch |
| **ONNX Runtime** | Framework-agnostic ONNX with TensorRT/GPU |
| **TensorFlow** | SavedModel format |
| **Python** | Custom preprocessing, postprocessing, or unsupported models |
| **FIL** | Tree-based models (XGBoost, LightGBM, Random Forest) on GPU |
| **DALI** | GPU-accelerated data preprocessing (image, audio, video) |
| **Custom** | C++/Python backends for proprietary or specialized runtimes |

### Dynamic Batching

Triton combines multiple incoming requests into a single batch at runtime.

- Higher GPU utilization and improved throughput
- Reduced latency variance
- Configurable `preferred_batch_size` and `max_queue_delay_microseconds` in `config.pbtxt`

### Concurrent Model Execution

- Run multiple models simultaneously
- Run multiple instances of the same model
- Distribute load across GPUs via `instance_group` in model config

### Model Versioning & Ensembles

- **Versioning**: Multiple versions per model (e.g. `1/`, `2/` in the model repository)
- **Ensembles**: Define a pipeline of models as an ensemble; eliminates intermediate network hops and reduces latency

### Model Instance Scaling

- Deploy multiple copies of a model for parallel inference and load isolation
- Configured via `instance_group`

---

## Inference & API

### gRPC via Helix Client

Predator uses **gRPC** for efficient request/response handling. Client applications (e.g. Realestate, IOP) send inference requests through the **Helix client**, which talks to the Triton Inference Server inside the Predator pod.

### Model Repository

Models are stored in a local model repository. Predator materializes this via an **Init Container** that downloads artifacts from cloud storage (e.g. GCS) so Triton has no runtime dependency on remote storage during inference.

---

## Deployment & Operational Features

### Custom Triton Images

- Production uses **custom-built** Triton images (only required backends) for smaller size and faster startup
- Images built on GCP VM, pushed to **Artifact Registry**, and referenced in Helm deployments
- Optional **response caching** via custom cache plugins added at image build time

### Image Distribution

- **Secondary boot disk caching**: Triton image pre-cached on GPU node pool to reduce pod startup and scale-up latency
- **Image streaming**: Optionally used for faster time-to-readiness during scaling

### Health Probes

- Readiness and liveness use `/v2/health/ready`
- Triton receives traffic only after models are loaded; failed instances are restarted automatically

### Autoscaling

- CPU-based scaling for generic load
- GPU-based scaling using **DCGM** metrics (utilization, memory, power); custom queries drive scale-up/scale-down

---

## Observability

- **Prometheus metrics**: Latency, throughput, GPU utilization, and more
- Metrics emitted from the Triton Inference Container and visualized in **Grafana**
- **Warmup requests**: Configurable to preload kernels and avoid cold-start latency

---

## Contributing

We welcome contributions! See the [Contributing Guide](https://github.com/Meesho/BharatMLStack/blob/main/CONTRIBUTING.md).

## Community & Support

- **Discord**: [community chat](https://discord.gg/XkT7XsV2AU)
- **Issues**: [GitHub Issues](https://github.com/Meesho/BharatMLStack/issues)
- **Email**: [ml-oss@meesho.com](mailto:ml-oss@meesho.com)

## License

BharatMLStack is open-source under the [BharatMLStack Business Source License 1.1](https://github.com/Meesho/BharatMLStack/blob/main/LICENSE.md).

---

<div align="center"><strong>Built with ❤️ for the ML community from Meesho</strong></div>
<div align="center"><strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong></div>
