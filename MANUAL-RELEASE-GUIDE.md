# Manual Release Guide

This guide explains how to use the new manual release system for BharatMLStack components.

## Overview

The automated release system has been replaced with a manual script-based approach that gives you full control over when and what to release. The new system uses the `manual-release.sh` script.

## Features

✅ **Branch Context**: Shows current branch information with branch-based release restrictions  
✅ **Release Type Selection**: Choose between alpha, beta, and standard releases based on current branch  
✅ **Module Selection**: Pick which modules to release  
✅ **Version Management**: Automatic version handling with proper incrementing  
✅ **GitHub Workflow Integration**: Automatically triggers appropriate workflows  
✅ **Smart Pre-release Versioning**: Handles `x.y.z-alpha.1`, `x.y.z-beta.1`, etc.  
✅ **Branch Validation**: Enforces release type restrictions based on branch naming conventions  

## Prerequisites

1. **GitHub CLI (Optional but Recommended)**:
   ```bash
   # Install GitHub CLI for automatic workflow triggering
   brew install gh
   # Or download from https://cli.github.com/
   
   # Authenticate
   gh auth login
   ```

2. **Git Repository**: Must be run from within the BharatMLStack git repository

## Usage

### Quick Start

```bash
./manual-release.sh
```

The script will guide you through an interactive process:

1. **Branch Check**: Shows current branch and checks for uncommitted changes
2. **Release Type**: Choose from available release types based on your current branch
   - **Alpha**: Available from `feat/`, `fix/`, `feat-nbc/` branches - Auto-increments `.alpha.N`
   - **Beta**: Available from `develop` branch - Auto-increments `.beta.N`
   - **Standard**: Available from `main`, `master`, `release/*` branches - Uses existing versions as-is
3. **Module Selection**: Select which modules to release
4. **Confirmation**: Review and confirm the release plan
5. **Execution**: Updates versions (for alpha/beta only) and triggers workflows

### Available Modules

- **horizon**: Container orchestration service
- **trufflebox-ui**: Web interface
- **online-feature-store**: Feature store service  
- **go-sdk**: Go software development kit
- **py-sdk**: Python SDK (includes bharatml_commons, grpc_feature_client, spark_feature_push_client)

## Version Management

### Version File Locations

**All Modules** (use VERSION files):
- `horizon/VERSION`
- `trufflebox-ui/VERSION`
- `online-feature-store/VERSION`
- `go-sdk/VERSION`
- `py-sdk/bharatml_commons/VERSION`
- `py-sdk/grpc_feature_client/VERSION`
- `py-sdk/spark_feature_push_client/VERSION`

**Note**: Python packages use VERSION files which are automatically read by pyproject.toml via hatch configuration.

### Version Format

**All VERSION files**: `v0.1.20`

### Release Types & Branch Restrictions

#### Standard Release (`x.y.z`) - **Production Ready**
- **Allowed Branches**: `main`, `master`, `release/*`
- **Version Strategy**: Uses existing version from files as-is (no auto-increment)
- **Manual Process**: Update version in files manually before running release script
- **Publishing**: PyPI (production), Docker with `latest` tag, full GitHub releases

#### Beta Release (`x.y.z-beta.N`) - **Testing Ready**
- **Allowed Branches**: `develop` only
- **First Beta**: `v0.1.20` → `v0.1.20-beta.1`
- **Next Beta**: `v0.1.20-beta.1` → `v0.1.20-beta.2`
- **Sequential**: Automatically finds next available beta number
- **Publishing**: TestPyPI, Docker without `latest` tag, GitHub pre-releases

#### Alpha Release (`x.y.z-alpha.N`) - **Development/Experimental**
- **Allowed Branches**: `feat/*`, `fix/*`, `feat-nbc/*`
- **First Alpha**: `v0.1.20` → `v0.1.20-alpha.1`
- **Next Alpha**: `v0.1.20-alpha.1` → `v0.1.20-alpha.2`
- **Sequential**: Automatically finds next available alpha number
- **Publishing**: TestPyPI, Docker without `latest` tag, GitHub pre-releases

## Module-Specific Behavior

### Python SDK (py-sdk)
- **Version Source**: Reads from `VERSION` files (pyproject.toml configured to read from VERSION files via hatch)
- **Standard Releases**: Uses existing version in VERSION file as-is
- **Alpha/Beta Releases**: Auto-increments with `.alpha.N` or `.beta.N` suffix and updates VERSION file
- **All Packages Updated**: bharatml_commons, grpc_feature_client, spark_feature_push_client
- **Production releases**: Published to PyPI
- **Alpha/Beta releases**: Published to TestPyPI
- Triggers `release-py-sdk.yml` workflow

### Docker-based Modules (horizon, trufflebox-ui, online-feature-store)
- **Version Source**: Reads from `VERSION` files
- **Standard Releases**: Uses existing version in VERSION file as-is
- **Alpha/Beta Releases**: Auto-increments with `.alpha.N` or `.beta.N` suffix and updates VERSION file
- **Container Registry**: Pushes to `ghcr.io` (GitHub Container Registry)
- **Production releases**: Tagged with version + `latest`
- **Alpha/Beta releases**: Tagged with version only (no `latest`)
- **online-feature-store**: Builds two images (`onfs-api-server`, `onfs-consumer`)
- Triggers respective release workflows

### Go SDK (go-sdk)
- **Version Source**: Reads from `VERSION` file
- **Standard Releases**: Uses existing version in VERSION file as-is
- **Alpha/Beta Releases**: Auto-increments with `.alpha.N` or `.beta.N` suffix and updates VERSION file
- **Git Tags**: Creates `go-sdk/v{version}` tags
- **GitHub Releases**: Automatic release notes and installation instructions
- **Alpha/Beta releases**: Marked as pre-releases
- **Validation**: Runs tests, builds, and go vet before release
- Triggers `release-go-sdk.yml` workflow

## GitHub Workflows

The script automatically triggers the appropriate release workflow for each module:

| Module | Workflow File | Purpose |
|--------|---------------|---------|
| horizon | release-horizon.yml | Build and push Docker images to registry |
| trufflebox-ui | release-trufflebox-ui.yml | Build and push UI Docker images |
| online-feature-store | release-online-feature-store.yml | Build and push feature store Docker images (api-server + consumer) |
| go-sdk | release-go-sdk.yml | Create Git tags and GitHub releases |
| py-sdk | release-py-sdk.yml | Build and publish Python packages to PyPI/TestPyPI |

**Note**: These are separate release workflows, not the existing CI workflows (`horizon.yml`, `py-sdk.yml`, etc.) which only run tests and builds.

## Examples

### Alpha Release from Feature Branch
```bash
git checkout feat/new-feature
./manual-release.sh
# Available: 1) Alpha Release (x.y.z-alpha.N) - from feat/new-feature branch
# Select: 1) Alpha Release
# Select: 2) horizon
# Result: horizon gets v0.1.20-alpha.1, published to TestPyPI/Docker registry
```

### Beta Release from Develop
```bash
git checkout develop
./manual-release.sh
# Available: 1) Beta Release (x.y.z-beta.N) - from develop branch
# Select: 1) Beta Release
# Select: 1 3 5 (horizon, online-feature-store, py-sdk)
# Result: All selected modules get v0.1.20-beta.1
```

### Standard Release from Main
```bash
git checkout main
./manual-release.sh
# Available: 1) Standard Release (x.y.z) - from main branch
# Select: 1) Standard Release
# Select: all
# Result: All modules use existing versions from their files (no version changes)
```

### Branch Restriction Examples
```bash
# This will show an error - can't do standard release from feature branch
git checkout feat/my-feature
./manual-release.sh
# Error: "No valid release types available for branch 'feat/my-feature'"

# This will work - alpha release from feature branch
git checkout feat/my-feature
./manual-release.sh
# Shows: 1) Alpha Release (x.y.z-alpha.N) - from feat/my-feature branch
```

## Post-Release Steps

After running the script:

1. **Monitor Workflows**: Check GitHub Actions for workflow status
2. **Commit Changes**: The script updates VERSION files - commit these changes:
   ```bash
   git add .
   git commit -m "chore: bump versions for release"
   git push
   ```
3. **Verify Releases**: Check that packages/images are published correctly
4. **Create Release Notes**: Update CHANGELOG or create GitHub releases as needed

## Branch-Based Release Strategy

The release system enforces a strict branch-based strategy to ensure proper release management:

### Branch Types & Purposes

| Branch Pattern | Release Type | Purpose | Publishing Target |
|----------------|--------------|---------|-------------------|
| `feat/*`, `fix/*`, `feat-nbc/*` | Alpha | Development/Experimental builds | TestPyPI, Docker (no latest) |
| `develop` | Beta | Testing and integration | TestPyPI, Docker (no latest) |
| `main`, `master`, `release/*` | Standard | Production releases | PyPI, Docker (with latest) |

### Why These Restrictions?

- **Alpha (feat/fix branches)**: Experimental features that need early testing
- **Beta (develop)**: Integrated features ready for broader testing before production
- **Standard (main/release)**: Stable, production-ready code

## Troubleshooting

### Branch Validation Errors
```bash
# Error: Beta releases can only be made from 'develop' branch
# Solution: Switch to develop or use alpha release
git checkout develop  # For beta releases
# OR
# Use alpha release from current feature branch
```

### No Valid Release Types Available
```bash
# Error: "No valid release types available for branch 'random-branch'"
# Solution: Use proper branch naming convention
git checkout -b feat/my-feature    # For alpha releases
git checkout develop               # For beta releases  
git checkout main                  # For standard releases
```

### GitHub CLI Not Found
If you don't have GitHub CLI installed, the script will show manual commands to trigger workflows.

### Version File Missing
The script will error if a VERSION file is missing. Create one with the current version:
```bash
echo "v0.1.0" > module-name/VERSION
```

### Workflow Not Found
Ensure the workflow files exist in `.github/workflows/` directory.

## Migration from Automated System

The previous automated release system (`smart-release.yml`) has been removed. Key differences:

- **Manual Control**: You decide when to release
- **Selective Releases**: Choose specific modules instead of releasing everything
- **Better Beta Handling**: Proper beta version incrementing
- **Branch Flexibility**: Release from any branch

## Support

If you encounter issues with the release script:

1. Check that you're in the correct git repository
2. Ensure VERSION files exist for all modules
3. Verify GitHub workflows exist
4. Check GitHub CLI authentication if using workflow triggering

For questions or improvements, create an issue in the repository. 