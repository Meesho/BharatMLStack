# GitHub Actions Workflows

This directory contains automated workflows for the BharatMLStack project. The workflows provide continuous integration, automated testing, and build status management.

## Workflows Overview

### 1. CI Build and Test (`ci.yml`)

**Triggers:**
- Pull requests to `master` or `develop` branches
- Direct pushes to `master` or `develop` branches
- Pull request target events (opened, synchronized, reopened)

**Features:**
- **Smart Change Detection**: Only builds components that have changed
- **Multi-language Support**: Handles Go (Horizon, Online Feature Store) and Node.js (Trufflebox UI)
- **Comprehensive Testing**: Runs tests, linting, and static analysis
- **Docker Build Verification**: Ensures Docker images can be built successfully
- **Build Summary**: Provides a summary of all build results

**Jobs:**
- `detect-changes`: Identifies which directories have changes
- `build-horizon`: Builds and tests the Horizon Go service
- `build-trufflebox-ui`: Builds and tests the Trufflebox UI React app
- `build-online-feature-store`: Builds and tests the Online Feature Store Go service
- `summary`: Provides a summary of all build results

### 2. Update Build Badges (`update-badges.yml`)

**Triggers:**
- Automatically runs after the CI workflow completes

**Features:**
- **Automatic Badge Updates**: Updates build status badges in README files
- **Dynamic Badge Colors**: Green for passing, yellow for skipped, red for failing
- **Version Badge Updates**: Shows current version from VERSION files
- **Multi-README Updates**: Updates both root and individual component READMEs

### 3. Manual Badge Update (`manual-badge-update.yml`)

**Triggers:**
- Manual workflow dispatch with custom inputs

**Features:**
- **Manual Control**: Allows manual override of build status badges
- **Custom Status Selection**: Choose from passing, failing, skipped, or unknown
- **Individual Component Control**: Set status for each component independently

## Setup Instructions

### 1. Repository Setup

1. **Fork or clone** this repository
2. **Update repository references** in the README files:
   - Replace `Meesho` in badge URLs with your GitHub username
   - Update the repository name if different

### 2. GitHub Token Permissions

Ensure your repository has the necessary permissions:

1. Go to **Settings → Actions → General**
2. Under **Workflow permissions**, select:
   - ✅ **Read and write permissions**
   - ✅ **Allow GitHub Actions to create and approve pull requests**

### 3. Branch Protection (Optional but Recommended)

Set up branch protection for `master` and `develop`:

1. Go to **Settings → Branches**
2. Add protection rules:
   - ✅ **Require status checks to pass before merging**
   - ✅ **Require branches to be up to date before merging**
   - ✅ **Include administrators**

## Usage

### Automatic CI Builds

The CI workflow runs automatically when:

```bash
# Creating a pull request
git checkout -b feature/my-feature
git commit -am "Add new feature"
git push origin feature/my-feature
# Create PR to master or develop

# Pushing to protected branches
git push origin master
git push origin develop
```

### Manual Badge Updates

To manually update build status badges:

1. Go to **Actions** tab in GitHub
2. Select **Manual Badge Update** workflow
3. Click **Run workflow**
4. Select desired status for each component:
   - `passing` - Green badge
   - `failing` - Red badge
   - `skipped` - Yellow badge
   - `unknown` - Gray badge
5. Click **Run workflow**

### Viewing Build Results

**In Pull Requests:**
- Build status checks appear at the bottom of PR
- Click "Details" to view full build logs
- Failed builds prevent merging (if branch protection enabled)

**In Actions Tab:**
- View all workflow runs
- See detailed logs for each job
- Download build artifacts (if any)

**In README Files:**
- Build status badges show current status
- Version badges show current versions
- Click badges to view latest workflow runs

## Badge Examples

The workflows create several types of badges:

### Overall CI Status
```markdown
![CI Build and Test](https://github.com/USERNAME/REPO/workflows/CI%20Build%20and%20Test/badge.svg)
```

### Component Build Status
```markdown
![Build](https://img.shields.io/badge/build-passing-brightgreen)
![Build](https://img.shields.io/badge/build-failing-red)
![Build](https://img.shields.io/badge/build-skipped-yellow)
![Build](https://img.shields.io/badge/build-unknown-lightgrey)
```

### Version Badges
```markdown
![Version](https://img.shields.io/badge/version-v1.2.3-blue)
```

## Troubleshooting

### Common Issues

**1. Workflow not triggering:**
- Check branch protection settings
- Ensure correct branch names (master/develop)
- Verify workflow file syntax

**2. Permission errors:**
- Check repository token permissions
- Ensure `GITHUB_TOKEN` has write access

**3. Build failures:**
- Check individual job logs
- Verify dependencies are correct
- Ensure Docker builds work locally

**4. Badge not updating:**
- Check if badge update workflow completed
- Verify README file syntax
- Clear browser cache

### Getting Help

1. **Check workflow logs** in the Actions tab
2. **Validate YAML syntax** using online validators
3. **Test locally** before pushing changes
4. **Review GitHub Actions documentation** for specific action issues

## Extending the Workflows

### Adding New Components

To add a new component to the CI pipeline:

1. **Update `ci.yml`:**
   ```yaml
   # Add to detect-changes job
   new-component:
     - 'new-component/**'
   
   # Add new build job
   build-new-component:
     needs: detect-changes
     if: needs.detect-changes.outputs.new-component-changed == 'true'
     # ... build steps
   ```

2. **Update badge workflows** to include the new component

3. **Add README** for the new component with badge placeholders

### Customizing Build Steps

Each build job can be customized:

```yaml
# Example: Add code coverage
- name: Run tests with coverage
  run: go test -v -coverprofile=coverage.out ./...

- name: Upload coverage
  uses: codecov/codecov-action@v3
  with:
    file: ./coverage.out
```

### Adding Notifications

Add Slack/Discord notifications:

```yaml
- name: Notify on failure
  if: failure()
  uses: 8398a7/action-slack@v3
  with:
    status: failure
    webhook_url: ${{ secrets.SLACK_WEBHOOK }}
```

## Security Considerations

1. **Use `pull_request` instead of `pull_request_target`** when possible
2. **Limit permissions** to minimum required
3. **Review external actions** before using
4. **Protect sensitive secrets** in repository settings
5. **Use specific action versions** instead of `@main`

## Performance Optimization

1. **Cache dependencies** (Go modules, npm packages)
2. **Use job matrices** for parallel builds
3. **Skip unnecessary steps** with conditionals
4. **Optimize Docker builds** with multi-stage builds 