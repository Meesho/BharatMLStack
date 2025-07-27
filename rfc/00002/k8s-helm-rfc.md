
# RFC: Kubernetes Deployment with Helm Charts for BharatMLStack

## Metadata
- **Title**: Kubernetes Deployment Strategy using Helm Charts for BharatMLStack
- **Author**: Raju Gupta
- **Status**: Draft
- **Created**: 2025-07-26
- **Last Updated**: 2025-07-26
- **Repository**: [Meesho/BharatMLStack](https://github.com/Meesho/BharatMLStack)
- **Target Release:** v1.0.0
- **Discussion Channel:** #infra-dev

## 🎯 Motivation

BharatMLStack powers core ML infrastructure — including **Online Feature Store**, **Horizon (control plane backend)**, and **Trufflebox UI** — to support high-throughput inference, training, and real-time feature retrieval.

The current deployment methods are manual and component-specific, making it hard to:
- Standardize deployment patterns across components.
- Onboard new contributors or operators quickly.
- Maintain consistent security and observability standards.

A **Helm-based deployment approach** is needed to:
- **Simplify deployment** for ML engineers and data scientists.
- **Enable consistent configuration as code** across all environments.
- **Support production-grade scaling** (HPA, PDB, Gateway routing).
- **Adopt cloud-native best practices** from the start.

## ✅ Goals

- Provide **modular Helm charts** for core components:  
  **Online Feature Store**, **Horizon**, **Trufflebox UI**, and optional **SDK utilities**.
- Support **Ingress** (default) and **Gateway API** (production-ready routing).
- Embed **security & observability best practices** (RBAC, NetworkPolicy, ServiceMonitor).
- Enable environment-specific overrides (`values-dev.yaml`, `values-prod.yaml`).
- Provide a **contributor-friendly structure** (clear templates, tests, CI-ready).

## 🚫 Non-Goals

- Provisioning Kubernetes clusters or cloud infrastructure.
- Managing third-party services (Redis, Scylla, Postgres) beyond optional values.
- Providing GitOps or CI/CD pipelines (only chart testing and linting).
- Combining components into a single “umbrella chart” (initial phase is modular).

## 🧱 Proposed Directory Structure

The Helm charts are **modularized per core component** for independent development, deployment, and scaling.  
Each component explicitly supports **Ingress** and **Gateway API** (Gateway + HTTPRoute).

```
bharatmlstack/helm/
├── online-feature-store/                  # Core real-time feature store
│   ├── Chart.yaml
│   ├── values.yaml
│   ├── values-dev.yaml
│   ├── values-prod.yaml
│   └── templates/
│       ├── deployment.yaml
│       ├── service.yaml
│       ├── ingress.yaml
│       ├── gateway.yaml
│       ├── httproute.yaml
│       ├── configmap.yaml
│       ├── secret.yaml
│       ├── hpa.yaml
│       ├── networkpolicy.yaml
│       ├── servicemonitor.yaml
│       ├── pdb.yaml
│       └── tests/
│           ├── latency-test.yaml
│           └── api-test.yaml

├── horizon/                               # Control plane backend
│   ├── Chart.yaml
│   ├── values.yaml
│   ├── values-dev.yaml
│   ├── values-prod.yaml
│   └── templates/
│       ├── deployment.yaml
│       ├── service.yaml
│       ├── ingress.yaml
│       ├── gateway.yaml
│       ├── httproute.yaml
│       ├── configmap.yaml
│       ├── secret.yaml
│       ├── cronjob.yaml
│       ├── rbac.yaml
│       ├── networkpolicy.yaml
│       ├── servicemonitor.yaml
│       └── tests/
│           └── api-test.yaml

├── trufflebox-ui/                         # Management console
│   ├── Chart.yaml
│   ├── values.yaml
│   ├── values-dev.yaml
│   ├── values-prod.yaml
│   └── templates/
│       ├── deployment.yaml
│       ├── service.yaml
│       ├── ingress.yaml
│       ├── gateway.yaml
│       ├── httproute.yaml
│       ├── configmap.yaml
│       ├── secret.yaml
│       ├── pdb.yaml
│       └── tests/
│           └── ui-availability.yaml

└── sdk-common/ (optional shared utilities)
    ├── Chart.yaml
    ├── values.yaml
    └── templates/
        ├── job.yaml
        └── cronjob.yaml
```

## 🌐 Ingress and Gateway API Support

### Why Both?
- **Ingress (NGINX/Traefik)** → Default, widely supported, ideal for dev/local.
- **Gateway API** → Kubernetes future standard, ideal for production-grade routing, traffic-splitting, and multi-tenant use cases.

### Example Values
```yaml
ingress:
  enabled: true
  className: "nginx"
  host: bharatmlstack.local

gateway:
  enabled: false
  className: "istio"
  host: bharatmlstack.prod.com
  tls:
    enabled: true
    secretName: bharatmlstack-tls
```

## 📝 Contribution Workflow

- Fork the repository and work under `bharatmlstack/helm/<component>/`.
- Run `helm lint` and `helm template` before raising PRs.
- Update/add Helm test hooks in `templates/tests/`.
- Ensure changes work with KinD or Minikube (CI will validate).
- Update `values-*.yaml` for new configs.
