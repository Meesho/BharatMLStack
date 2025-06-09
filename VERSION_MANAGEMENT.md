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
├── online-feature-store/VERSION               # Online Feature Store services version
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
   - Docker images: Tagged with VERSION file content → `ghcr.io/meesho/horizon:v0.1.20`

2. **Pre-release** (`develop`):
   - Go SDK: Adds beta suffix → `go-sdk/v0.1.0-beta.abc1234`
   - Python packages: Adds beta suffix → `0.1.0b20250609130000.abc1234`
   - Docker images: Uses version-beta format → `ghcr.io/meesho/horizon:v0.1.20-beta`

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
   
   # For Docker services
   echo "v0.2.0" > horizon/VERSION
   echo "v0.2.0" > online-feature-store/VERSION
   echo "v0.2.0" > trufflebox-ui/VERSION
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

### Docker Images
```bash
# Pull from GitHub Container Registry
docker pull ghcr.io/meesho/horizon:v0.1.20
docker pull ghcr.io/meesho/onfs-api-server:v0.1.0
docker pull ghcr.io/meesho/onfs-consumer:v0.1.0
docker pull ghcr.io/meesho/trufflebox-ui:v0.1.6

# Or pull latest versions
docker pull ghcr.io/meesho/horizon:latest
docker pull ghcr.io/meesho/onfs-api-server:latest
docker pull ghcr.io/meesho/onfs-consumer:latest
docker pull ghcr.io/meesho/trufflebox-ui:latest
```

## 🔄 Workflow Triggers

- **Go SDK Release**: `.github/workflows/go-sdk-release.yml`
  - Triggers on: Push to `master`/`develop` with changes in `go-sdk/`
  - Reads: `go-sdk/VERSION`

- **Python SDK Publish**: `.github/workflows/py-sdk-publish.yml`
  - Triggers on: Push to `master`/`develop` with changes in `py-sdk/`
  - Reads: `py-sdk/*/VERSION` files

- **Docker Image Build**: `.github/workflows/build-and-push-images.yml`
  - Triggers on: Push to `master`/`develop` or version tags
  - Reads: `horizon/VERSION`, `online-feature-store/VERSION`, `trufflebox-ui/VERSION`
  - Publishes to: GitHub Container Registry (`ghcr.io/meesho/*`)

## 🐳 Docker Image Tags

**Published Images:**
- `ghcr.io/meesho/horizon` (from `horizon/VERSION`)
- `ghcr.io/meesho/onfs-api-server` (from `online-feature-store/VERSION`)
- `ghcr.io/meesho/onfs-consumer` (from `online-feature-store/VERSION`)  
- `ghcr.io/meesho/trufflebox-ui` (from `trufflebox-ui/VERSION`)

**Tag Strategy:**

**Master Branch (Stable Release):**
- **Version tag**: `v0.1.20` (exact VERSION file content)
- **Latest tag**: `latest`
- **Branch tag**: `master`

**Develop Branch (Beta Release):**
- **Beta version tag**: `v0.1.20-beta` (VERSION file + `-beta` suffix)

**Pull Requests:**
- **PR tag**: `pr-123` (for validation only, not pushed)

## ⚡ Quick Commands

```bash
# Check current versions
find . -name "VERSION" -exec sh -c 'echo "$1: $(cat $1)"' _ {} \;

# Update all Python package versions at once
NEW_VERSION="v0.2.0"
echo "$NEW_VERSION" > py-sdk/bharatml_commons/VERSION
echo "$NEW_VERSION" > py-sdk/spark_feature_push_client/VERSION  
echo "$NEW_VERSION" > py-sdk/grpc_feature_client/VERSION

# Update all Docker service versions at once
echo "$NEW_VERSION" > horizon/VERSION
echo "$NEW_VERSION" > online-feature-store/VERSION
echo "$NEW_VERSION" > trufflebox-ui/VERSION
```

## 🏷️ Example Docker Tags After Merge

**After merge to `master` (horizon/VERSION = `v0.1.20`):**
```bash
ghcr.io/meesho/horizon:v0.1.20    # Exact version from VERSION file
ghcr.io/meesho/horizon:latest     # Latest stable version
ghcr.io/meesho/horizon:master     # Master branch tag
```

**After merge to `develop` (horizon/VERSION = `v0.1.20`):**
```bash
ghcr.io/meesho/horizon:v0.1.20-beta    # Version + beta suffix
```

**During Pull Request:**
```bash
ghcr.io/meesho/horizon:pr-123     # PR number (built but not pushed)
``` 