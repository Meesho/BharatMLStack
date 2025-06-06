# BharatMLStack

![CI Build and Test](https://github.com/YOUR_USERNAME/BharatMLStack/workflows/CI%20Build%20and%20Test/badge.svg)

A comprehensive ML infrastructure stack with feature store, model serving, and UI components.

## Build Status

| Component | Build Status | Version | Directory |
|-----------|--------------|---------|-----------|
| **Horizon** | ![Build](https://img.shields.io/badge/build-unknown-lightgrey) | ![Version](https://img.shields.io/badge/version-unknown-lightgrey) | [horizon/](./horizon) |
| **Trufflebox UI** | ![Build](https://img.shields.io/badge/build-unknown-lightgrey) | ![Version](https://img.shields.io/badge/version-unknown-lightgrey) | [trufflebox-ui/](./trufflebox-ui) |
| **Online Feature Store** | ![Build](https://img.shields.io/badge/build-unknown-lightgrey) | ![Version](https://img.shields.io/badge/version-unknown-lightgrey) | [online-feature-store/](./online-feature-store) |

## Components

### 🚀 Horizon
Model serving and inference service built with Go. Provides high-performance model serving capabilities.

### 🎨 Trufflebox UI
Modern web interface for managing ML models and features. Built with React and provides an intuitive user experience.

### 🗄️ Online Feature Store
High-performance feature store for real-time ML inference. Supports both batch and streaming feature processing.

## Quick Start

```bash
# Check what would be released
./release.sh --dry-run

# Release components with changes
./release.sh

# Force release all components
./release.sh --force-all
```

## Development

Each component has its own build and release process:

- **Horizon**: Go-based service with Docker support
- **Trufflebox UI**: Node.js/React application with Docker support  
- **Online Feature Store**: Go-based service with gRPC API

See individual README files in each directory for detailed development instructions.

## CI/CD

This repository uses GitHub Actions for continuous integration:

- **Automatic builds** on PRs to master/develop branches
- **Smart change detection** - only builds components that changed
- **Automatic badge updates** showing build status
- **Multi-platform Docker builds** for releases

## License

See [LICENSE.md](LICENSE.md) for details.
