# Version Management

This repository uses `VERSION` files to manage package versions across all components.

## 📁 Version Files Location

```
BharatMLStack/
├── go-sdk/VERSION                              # Go SDK version
├── py-sdk/
│   ├── bharatml_commons/VERSION               # bharatml-commons package version  
│   ├── spark_feature_push_client/VERSION      # spark-feature-push-client package version
│   └── grpc_feature_client/VERSION            # grpc-feature-client package version
├── horizon/VERSION                             # Horizon service version
└── trufflebox-ui/VERSION                       # TruffleBox UI version
```

## 📝 Version Format

- **Go SDK**: `v0.1.0` (with 'v' prefix)
- **Python packages**: `v0.1.0` (will be stripped to `0.1.0` for Python)
- **Docker services**: `v0.1.20` (with 'v' prefix)

## 🚀 Release Process

### Automatic Releases (CI/CD)

**When you merge to `master` or `develop`:**

1. **Stable Release** (`master`):
   - Go SDK: Uses exact version from `go-sdk/VERSION` → `go-sdk/v0.1.0`
   - Python packages: Uses version without 'v' → `0.1.0`

2. **Pre-release** (`develop`):
   - Go SDK: Adds beta suffix → `go-sdk/v0.1.0-beta.abc1234`
   - Python packages: Adds beta suffix → `0.1.0b20250609130000.abc1234`

### Manual Version Updates

To release a new version:

1. **Update the VERSION file(s)**:
   ```bash
   # For Go SDK
   echo "v0.2.0" > go-sdk/VERSION
   
   # For Python packages (update individually)
   echo "v0.2.0" > py-sdk/bharatml_commons/VERSION
   echo "v0.2.0" > py-sdk/spark_feature_push_client/VERSION
   echo "v0.2.0" > py-sdk/grpc_feature_client/VERSION
   ```

2. **Commit and push to trigger release**:
   ```bash
   git add .
   git commit -m "Bump version to v0.2.0"
   git push origin master  # For stable release
   # OR
   git push origin develop # For pre-release
   ```

## 📦 Package Installation

### Go SDK
```bash
# Install specific version
go get github.com/Meesho/BharatMLStack/go-sdk@v0.1.0

# Install latest
go get github.com/Meesho/BharatMLStack/go-sdk@latest
```

### Python Packages
```bash
# Install from BharatML Stack registry
pip install --index-url https://meesho.github.io/BharatMLStack/pypi/simple/ bharatml-commons==0.1.0
pip install --index-url https://meesho.github.io/BharatMLStack/pypi/simple/ spark-feature-push-client==0.1.0
pip install --index-url https://meesho.github.io/BharatMLStack/pypi/simple/ grpc-feature-client==0.1.0
```

## 🔄 Workflow Triggers

- **Go SDK Release**: `.github/workflows/go-sdk-release.yml`
  - Triggers on: Push to `master`/`develop` with changes in `go-sdk/`
  - Reads: `go-sdk/VERSION`

- **Python SDK Publish**: `.github/workflows/py-sdk-publish.yml`
  - Triggers on: Push to `master`/`develop` with changes in `py-sdk/`
  - Reads: `py-sdk/*/VERSION` files

## ⚡ Quick Commands

```bash
# Check current versions
find . -name "VERSION" -exec sh -c 'echo "$1: $(cat $1)"' _ {} \;

# Update all Python package versions at once
NEW_VERSION="v0.2.0"
echo "$NEW_VERSION" > py-sdk/bharatml_commons/VERSION
echo "$NEW_VERSION" > py-sdk/spark_feature_push_client/VERSION  
echo "$NEW_VERSION" > py-sdk/grpc_feature_client/VERSION
``` 