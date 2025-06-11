# 🔄 Automated Version Management

This repository includes an automated version management system that helps maintain consistent versioning across all components.

## 🚀 Quick Setup

```bash
# Run the setup script to install the pre-commit hook
./setup-version-check.sh
```

## 📖 How It Works

### Pre-commit Hook
- **Automatic Detection**: Scans for changes in directories containing `VERSION` files
- **Interactive Prompts**: Asks you to choose version increment type for each changed directory
- **Auto-staging**: Updates `VERSION` files and stages them before commit
- **Semantic Versioning**: Follows [SemVer](https://semver.org/) principles

### Monitored Directories
```
📂 Repository Structure
├── py-sdk/bharatml_commons/VERSION
├── py-sdk/spark_feature_push_client/VERSION  
├── py-sdk/grpc_feature_client/VERSION
├── horizon/VERSION
├── go-sdk/VERSION
├── online-feature-store/VERSION
└── trufflebox-ui/VERSION
```

## 🎯 Version Increment Types

| Type | When to Use | Example | Description |
|------|-------------|---------|-------------|
| **Major** | Breaking changes | `1.2.3 → 2.0.0` | API changes, incompatible updates |
| **Minor** | New features | `1.2.3 → 1.3.0` | Backward-compatible functionality |
| **Patch** | Bug fixes | `1.2.3 → 1.2.4` | Backward-compatible bug fixes |
| **Skip** | No version change | `1.2.3 → 1.2.3` | Documentation, tests, internal changes |

## 🔄 CI/CD Integration

### Release Versioning
- **Master branch**: Uses exact version from `VERSION` files (e.g., `1.2.3`)
- **Develop branch**: Appends `-beta` suffix (e.g., `1.2.3-beta`)

### Supported Release Types
- **Python SDK**: PEP 440 compliant versions
- **Go SDK**: Go module compatible versions  
- **Docker Images**: Container registry tags
- **All Components**: Consistent versioning strategy

## 💡 Usage Examples

### Example 1: Adding a New Feature
```bash
# Make changes to py-sdk/bharatml_commons/
git add py-sdk/bharatml_commons/src/new_feature.py
git commit -m "Add new feature"

# Pre-commit hook will prompt:
# 📂 Directory: py-sdk/bharatml_commons
# Current version: 1.2.3
# Changes detected in this directory
# How would you like to increment the version?
# [1] Major (breaking changes)
# [2] Minor (new features)      ← Choose this
# [3] Patch (bug fixes)
# [s] Skip (no version change)
# Choice [1/2/3/s]: 2

# Result: VERSION updated to 1.3.0 and staged
```

### Example 2: Bug Fix
```bash
# Fix a bug in horizon/
git add horizon/src/bugfix.go
git commit -m "Fix authentication bug"

# Pre-commit hook will prompt:
# 📂 Directory: horizon  
# Current version: 2.1.5
# How would you like to increment the version?
# Choice [1/2/3/s]: 3  ← Choose patch for bug fix

# Result: VERSION updated to 2.1.6 and staged
```

### Example 3: Multiple Directory Changes
```bash
# Changes in multiple directories
git add go-sdk/ horizon/
git commit -m "Update SDK and API"  

# Pre-commit hook will prompt for EACH directory:
# 📂 Directory: go-sdk
# Current version: 1.0.0
# Choice [1/2/3/s]: 2  (minor update)

# 📂 Directory: horizon
# Current version: 2.1.6  
# Choice [1/2/3/s]: 3  (patch update)

# Summary shown before applying:
# go-sdk: 1.0.0 → 1.1.0
# horizon: 2.1.6 → 2.1.7
# Proceed with these version updates? [y/N]: y
```

## 🛠️ Manual Operations

### Testing the Hook
```bash
# Test without committing
./pre-commit-version-check.sh
```

### Bypass Version Check (Not Recommended)
```bash
# Skip pre-commit hooks entirely
git commit --no-verify -m "Emergency fix"
```

### Uninstall Hook
```bash
# Remove the pre-commit hook
rm .git/hooks/pre-commit

# Restore backup if exists
mv .git/hooks/pre-commit.backup .git/hooks/pre-commit
```

## 🏗️ Architecture

### Files Structure
```
📁 Root/
├── 📄 pre-commit-version-check.sh    # Main version checking script
├── 📄 setup-version-check.sh         # Installation script  
├── 📄 VERSION-MANAGEMENT.md          # This documentation
└── 📁 .git/hooks/
    └── 📄 pre-commit                 # Git hook (auto-generated)
```

### Script Features
- ✅ **Change Detection**: Git diff analysis for staged/unstaged changes
- ✅ **Version Parsing**: Handles both `v1.2.3` and `1.2.3` formats
- ✅ **Interactive UI**: Colored output and clear prompts
- ✅ **Error Handling**: Validation and rollback on errors
- ✅ **Batch Operations**: Handle multiple directory changes efficiently

## 🔧 Troubleshooting

### Hook Not Running
```bash
# Check if hook exists and is executable
ls -la .git/hooks/pre-commit

# Reinstall if needed
./setup-version-check.sh
```

### Permission Issues
```bash
# Fix permissions
chmod +x pre-commit-version-check.sh
chmod +x setup-version-check.sh
chmod +x .git/hooks/pre-commit
```

### Version Format Issues
- Script handles both `v1.2.3` and `1.2.3` formats automatically
- Preserves original format in VERSION files
- Validates semantic versioning format

## 🎯 Best Practices

1. **Commit Frequently**: Small, focused commits make version decisions easier
2. **Use Semantic Versioning**: Follow SemVer guidelines for consistency
3. **Document Changes**: Link version increments to changelog entries
4. **Test Before Release**: Verify version increments don't break CI/CD
5. **Coordinate Teams**: Communicate major version changes across teams

## 🔗 Related Documentation

- [Semantic Versioning Specification](https://semver.org/)
- [Git Hooks Documentation](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks)
- [Python PEP 440 - Version Identification](https://peps.python.org/pep-0440/)
- [Go Modules Version Numbers](https://go.dev/doc/modules/version-numbers) 