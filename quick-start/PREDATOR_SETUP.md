# Predator Local Setup Guide

This guide will walk you through setting up Predator for local development, including ArgoCD configuration, GitHub App setup, and repository preparation.

## Table of Contents

1. [Quick Start: Automated Kubernetes Setup](#quick-start-automated-kubernetes-setup)
2. [Prerequisites](#prerequisites)
3. [Step 1: Set Up Local ArgoCD](#step-1-set-up-local-argocd)
4. [Manual Repository Setup](#manual-repository-setup)
5. [Step 2: Create a GitHub App](#step-2-create-a-github-app)
6. [Step 3: Configure GitHub App Permissions](#step-3-configure-github-app-permissions)
7. [Step 4: Install GitHub App to Your Repository](#step-4-install-github-app-to-your-repository)
8. [Step 5: Generate and Download Private Key](#step-5-generate-and-download-private-key)
9. [Step 6: Copy Predator Helm Chart to Repository](#step-6-copy-predator-helm-chart-to-repository)
10. [Step 7: Place GitHub Private Key in Horizon Configs](#step-7-place-github-private-key-in-horizon-configs)
11. [Step 8: Configure Docker Compose](#step-8-configure-docker-compose)
12. [Step 9: Verify Setup](#step-9-verify-setup)
13. [Step 10: Access Predator Service Through Contour HTTPProxy](#step-10-access-predator-service-through-contour-httpproxy)

---

## Quick Start: Automated Kubernetes Setup

**🚀 For a faster setup, use the automated script that handles all basic Kubernetes configuration:**

The `setup-predator-k8s.sh` script automates the following setup steps:
- ✅ **Create Kubernetes cluster** (if not exists) - supports kind, minikube, or Docker Desktop
- ✅ Label Kubernetes node for pod scheduling
- ✅ Install Contour (CRDs, Contour, and Envoy)
- ✅ Create IngressClass for contour-internal
- ✅ Configure Contour to watch for contour-internal
- ✅ Install Flagger CRDs (for AlertProvider)
- ✅ Install KEDA CRDs (for ScaledObject)
- ✅ Install PriorityClass (high-priority)
- ✅ Install ArgoCD
- ✅ Enable API key permissions for ArgoCD admin
- ✅ Retrieve ArgoCD admin password
- ✅ **Add GitHub repository to ArgoCD with GitHub App authentication**
- ✅ Optionally set up automated ArgoCD application onboarding (prd-applications)

### Usage

1. **Run the setup script** (it will detect and create a cluster if needed):
   ```bash
   cd quick-start
   ./setup-predator-k8s.sh
   ```

   The script will:
   - Check if a Kubernetes cluster is already running
   - If not, detect available tools (kind, minikube, or Docker Desktop)
   - Prompt you to select a tool (or use the only available one)
   - Create the cluster automatically
   - Continue with all setup steps

3. **Follow the script prompts:**
   - The script will automatically install all required components
   - When prompted, provide your GitHub repository details to add it to ArgoCD with GitHub App authentication:
     - GitHub App ID
     - GitHub App Installation ID
     - GitHub App Private Key path (e.g., `horizon/configs/github.pem`)
   - Optionally set up automated application onboarding (prd-applications)

4. **After the script completes:**
   - Access ArgoCD UI at https://localhost:8087 (port-forward is already running in background)
   - Generate ArgoCD API token (see [Step 1.7](#17-generate-argocd-api-token))
   - Continue with remaining setup steps:
     - [Step 6](#step-6-copy-predator-helm-chart-to-repository): Copy Predator Helm chart to repository
     - [Step 7](#step-7-place-github-private-key-in-horizon-configs): Place GitHub private key in Horizon configs
     - [Step 8](#step-8-configure-docker-compose): Configure Docker Compose

**Note:** The script automatically handles GitHub repository authentication with GitHub App. If you skipped this step, you can add the repository manually (see [Manual Repository Setup](#manual-repository-setup) below).

### What the Script Does

The script performs all the basic Kubernetes setup steps from [Step 1](#step-1-set-up-local-argocd) automatically:
- **Step 1.0**: Creates a Kubernetes cluster (if not exists) - supports kind, minikube, or Docker Desktop
- **Step 1.1**: Labels the Kubernetes node
- **Step 1.2**: Installs all required CRDs and PriorityClass
- **Step 1.3**: Installs ArgoCD
- **Step 1.6**: Enables API key permissions
- **Step 1.8**: Adds GitHub repository to ArgoCD with GitHub App authentication
- **Step 1.9**: Optionally sets up automated application onboarding (prd-applications)

**Note:** 
- For Docker Desktop, you'll need to manually enable Kubernetes in Docker Desktop settings first. The script will guide you through this.
- The script automatically installs ArgoCD CLI if not present:
  - **macOS**: Uses Homebrew (`brew install argocd`)
  - **Linux**: Downloads binary from GitHub releases to `~/.local/bin/argocd` (or `/usr/local/bin/argocd` with sudo)
  - If installation fails, you can continue without it (repository authentication will be skipped)
- The script uses GitHub App authentication (not PAT or SSH) to match the Horizon service configuration

### Manual Setup Alternative

If you prefer to set up manually or need to customize the installation, you can follow the detailed steps in [Step 1: Set Up Local ArgoCD](#step-1-set-up-local-argocd) and subsequent sections.

---

## Prerequisites

- Docker and Docker Compose installed
- **Docker Desktop disk allocation: At least 100GB** (required for Triton server image ~15GB)
  - Check: Docker Desktop → Settings → Resources → Advanced → "Disk image size"
  - Increase if less than 100GB (see [Docker Disk Space troubleshooting](#issue-docker-disk-space---no-space-left-on-device))
- kubectl installed and configured
- One of the following Kubernetes tools (optional - script will create cluster if needed):
  - kind (recommended for local development)
  - minikube
  - Docker Desktop Kubernetes
- A GitHub account
- A GitHub repository for storing Helm charts and ArgoCD applications

**Note:** 
- ArgoCD CLI is automatically installed by the script if not present
- The script will create a Kubernetes cluster automatically if one doesn't exist
- For Docker Desktop, you'll need to manually enable Kubernetes in settings first (script will guide you)

---

## Step 1: Set Up Local ArgoCD

**✅ All Kubernetes setup steps are automated by the script.** Simply run:

```bash
cd quick-start
./setup-predator-k8s.sh
```

The script automatically handles:
- Creating Kubernetes cluster (if needed)
- Labeling nodes
- Installing all required CRDs and PriorityClass
- Installing and configuring ArgoCD
- Enabling API key permissions
- Adding GitHub repository with GitHub App authentication
- Setting up prd-applications (optional)

### 1.7 Generate ArgoCD API Token

After running the script, you need to generate an ArgoCD API token for use in docker-compose.yml:

1. Log in to ArgoCD UI at https://localhost:8087
2. Go to **User Info** (click on your username in the top right)
3. Click **Generate New Token**
4. Copy the generated token (you'll need this for `ARGOCD_TOKEN` in docker-compose.yml)

**Alternative (using CLI):**

```bash
# Login to ArgoCD (ArgoCD CLI is automatically installed by the script)
argocd login localhost:8087 --insecure

# Generate token
argocd account generate-token
```

**Note:** The script automatically handles repository authentication with GitHub App and can optionally set up prd-applications. If you skipped those steps during script execution, see [Manual Repository Setup](#manual-repository-setup) below.

---

## Manual Repository Setup

If you need to manually add or update the GitHub repository in ArgoCD (e.g., if the script failed or you skipped that step):

### Using GitHub App (Recommended)

```bash
# Login to ArgoCD
argocd login localhost:8087 --insecure

# Add repository with GitHub App authentication
argocd repo add https://github.com/<YOUR_USERNAME>/<YOUR_REPO_NAME>.git \
  --name <YOUR_REPO_NAME> \
  --type git \
  --github-app-id <GITHUB_APP_ID> \
  --github-app-installation-id <GITHUB_INSTALLATION_ID> \
  --github-app-private-key-path <PATH_TO_PRIVATE_KEY>
```

### Verify Repository Access

```bash
# Test repository connection
argocd repo get https://github.com/<YOUR_USERNAME>/<YOUR_REPO_NAME>.git

# List all repositories
argocd repo list
```

---

## Step 2: Create a GitHub App

### 2.1 Navigate to GitHub App Settings

1. Go to https://github.com/settings/apps
2. Click **New GitHub App** (top right)

### 2.2 Fill in Basic Information

- **GitHub App name**: `horizon-bot` (or any name you prefer)
- **Homepage URL**: `https://github.com` (required, can be any valid URL)
- **User authorization callback URL**: Leave empty (not needed for this use case)
- **Webhook URL**: Leave empty (optional)
- **Webhook secret**: Leave empty (optional)

### 2.3 Configure Permissions

Set the following permissions:

- **Repository permissions:**
  - **Contents**: `Read and write` ⚠️ **REQUIRED**
  - **Metadata**: `Read-only` (automatically set)
  - **Pull requests**: `Read-only` (optional, for PR-based workflows)

- **Account permissions:**
  - Leave all as `No access` (not needed)

### 2.4 Configure Where App Can Be Installed

- Select **Only on this account** (for personal account) or **Any account** (for organization)

### 2.5 Create the GitHub App

Click **Create GitHub App** at the bottom of the page.

---

## Step 3: Configure GitHub App Permissions

After creating the app, you'll see the app's settings page. Note down:

- **App ID**: Found at the top of the page (e.g., `2546855`)
- **Client ID**: Not needed for this setup
- **Client secret**: Not needed for this setup

**Important:** The app is created but not yet installed. You need to install it to your repository in the next step.

---

## Step 4: Install GitHub App to Your Repository

### 4.1 Install the App

1. On the GitHub App settings page, scroll down to **Install App** section
2. Click **Install** next to your account/organization name
3. Select the repository where you want to store Helm charts (e.g., `onboarding-test`)
4. Click **Install**

### 4.2 Note the Installation ID

After installation, you'll be redirected to the installation page. The URL will look like:
```
https://github.com/settings/installations/100732634
```

The number at the end (`100732634`) is your **Installation ID**. Note this down.

**Alternative way to find Installation ID:**
- Go to your repository settings
- Click **Integrations** → **GitHub Apps**
- Find your app and click **Configure**
- The Installation ID is in the URL

---

## Step 5: Generate and Download Private Key

### 5.1 Generate Private Key

1. On your GitHub App settings page, scroll to **Private keys** section
2. Click **Generate a private key**
3. A `.pem` file will be downloaded automatically

**⚠️ Important:** 
- This key is only shown once. Save it securely.
- If you lose it, you'll need to generate a new one.

### 5.2 Save the Key

Save the downloaded file as `github.pem` (or any name you prefer). You'll place this in the Horizon configs directory in the next step.

---

## Step 6: Copy Predator Helm Chart to Repository

The Predator Helm chart needs to be available in your GitHub repository for ArgoCD to deploy applications.

### 6.1 Clone Your Repository

```bash
# Navigate to your workspace
cd ~/Desktop/projects/OSS/BharatMLStack

# Clone your repository (if not already cloned)
git clone https://github.com/YOUR_USERNAME/YOUR_REPO_NAME.git
cd YOUR_REPO_NAME
```

### 6.2 Copy Predator Chart

```bash
# From the BharatMLStack root directory
# Copy the predator/1.0.0 directory to your repository
# The chart should be at the root level as 1.0.0/ (to match ARGOCD_HELMCHART_PATH=1.0.0)
cp -r predator/1.0.0 YOUR_REPO_NAME/1.0.0

# Or if you're already in the repo directory
cp -r ../predator/1.0.0 ./1.0.0
```

**Note:** The chart path in your repo should match `ARGOCD_HELMCHART_PATH` in `docker-compose.yml`:
- If `ARGOCD_HELMCHART_PATH=1.0.0`, the chart should be at `1.0.0/` in your repo root

### 6.3 Commit and Push

```bash
# Add the chart
git add 1.0.0

# Commit
git commit -m "Add Predator Helm chart 1.0.0"

# Push to main branch (or your default branch)
git push origin main
```

**Verify:** Check that `1.0.0/` exists in your repository at the root level

---

## Step 7: Place GitHub Private Key in Horizon Configs

### 7.1 Locate Horizon Configs Directory

The Horizon service expects the GitHub private key at:
```
horizon/configs/github.pem
```

### 7.2 Copy the Private Key

```bash
# From BharatMLStack root directory
# Copy your downloaded github.pem file to horizon/configs/
cp /path/to/your/downloaded/github.pem horizon/configs/github.pem

# Verify it's there
ls -la horizon/configs/github.pem
```

**Note:** The `quick-start/start.sh` script automatically copies `horizon/configs/` to `workspace/configs/` during setup, which is then mounted into the Horizon container.

---

## Step 8: Configure Docker Compose

### 8.1 Update docker-compose.yml

Edit `quick-start/docker-compose.yml` and update the following environment variables in the `horizon` service:

```yaml
horizon:
  environment:
    # ArgoCD Configuration
    - ARGOCD_API=http://host.docker.internal:8087
    - ARGOCD_TOKEN=<YOUR_ARGOCD_TOKEN>  # From Step 1.7
    - ARGOCD_NAMESPACE=argocd
    - ARGOCD_DESTINATION_NAME=in-cluster  # For local Kubernetes
    - ARGOCD_PROJECT=default
    - ARGOCD_HELMCHART_PATH=1.0.0  # Path to Helm chart in your repo (should match chart location in repo)
    - ARGOCD_SYNC_POLICY_OPTIONS=CreateNamespace=true
    - ARGOCD_INSECURE=true
    
    # Local Development: Model Path (only used when GCS fields are "NA")
    # IMPORTANT: This must be an absolute path accessible from your Kubernetes node
    # - Docker Desktop: Use /Users/... paths (e.g., /Users/adityakumargarg/models)
    # - kind/minikube: Path must exist on the VM/node (see Step 8.2 for copying models)
    # - This path will be mounted as hostPath volume in the pod
    - LOCAL_MODEL_PATH=/tmp/models  # For kind: use /tmp/models (see Step 8.2)
    
    # GitHub Configuration
    - REPOSITORY_NAME=onboarding-test  # Your repository name
    - BRANCH_NAME=main  # Your default branch
    - GITHUB_APP_ID=<YOUR_APP_ID>  # From Step 3 (e.g., 2546855)
    - GITHUB_INSTALLATION_ID=<YOUR_INSTALLATION_ID>  # From Step 4.2 (e.g., 101432634)
    - GITHUB_PRIVATE_KEY_PATH=/app/configs/github.pem  # Path inside container
    - GITHUB_OWNER=<YOUR_GITHUB_USERNAME>  # Your GitHub username or org
    - GITHUB_COMMIT_AUTHOR=horizon-bot  # Name for git commits
    - GITHUB_COMMIT_EMAIL=your-email@example.com  # Email for git commits
    
    # GCS Configuration (for model operations)
    - GCS_ENABLED=true  # Set to false to disable GCS operations
    - GCS_MODEL_BUCKET=your-gcs-bucket-name  # GCS bucket for models
    - GCS_MODEL_BASE_PATH=your-base-path  # Base path in bucket
    - CLOUDSDK_CONFIG=/root/.config/gcloud  # Path to gcloud config inside container
  volumes:
    - ./configs:/app/configs:ro
    # Mount gcloud credentials for Application Default Credentials (ADC)
    # This allows the container to use credentials from 'gcloud auth application-default login'
    - ~/.config/gcloud:/root/.config/gcloud:ro
```

**Important:** For GCS authentication using Application Default Credentials (ADC):

1. **Authenticate on your host machine first:**
   ```bash
   # Run this on your host (not inside the container)
   gcloud auth application-default login
   ```
   This will create credentials at `~/.config/gcloud/application_default_credentials.json`

2. **Verify credentials exist:**
   ```bash
   ls -la ~/.config/gcloud/application_default_credentials.json
   ```

3. **The docker-compose.yml mounts your host's `~/.config/gcloud` directory into the container:**
   - Host path: `~/.config/gcloud`
   - Container path: `/root/.config/gcloud`
   - The Go GCS client will automatically find and use these credentials

4. **Set the correct GCP project (if needed):**
   ```bash
   gcloud config set project your-gcp-project-id
   ```

The `CLOUDSDK_CONFIG` environment variable tells gcloud SDK (if used) where to find the config, and the Go client library will automatically discover the ADC credentials at the standard location.

### 8.2 Copy Models to Kubernetes Node (for kind/minikube)

**Important:** For local Kubernetes clusters (kind/minikube), the models must exist on the Kubernetes node, not just on your host machine. The `hostPath` volume mounts from the node's filesystem.

#### For kind Clusters:

1. **Identify your kind node name:**
   ```bash
   kubectl get nodes
   # Example output: bharatml-stack-control-plane
   ```

2. **Copy your models into the kind node:**
   ```bash
   # Get your kind node name
   NODE_NAME=$(kubectl get nodes -o name | head -1 | sed 's|node/||')
   
   # Create models directory in the node
   docker exec $NODE_NAME mkdir -p /tmp/models
   
   # Copy models from your host to the kind node
   # Replace with your actual models directory path
   tar -czf - -C /path/to/your/models . 2>/dev/null | \
     docker exec -i $NODE_NAME tar -xzf - -C /tmp/models
   
   # Verify models were copied
   docker exec $NODE_NAME ls -la /tmp/models/
   ```

3. **Example with actual path:**
   ```bash
   # If your models are at: /Users/adityakumargarg/Desktop/projects/OSS/BharatMLStack/horizon/configs/models/
   NODE_NAME=$(kubectl get nodes -o name | head -1 | sed 's|node/||')
   docker exec $NODE_NAME mkdir -p /tmp/models
   tar -czf - -C /Users/adityakumargarg/Desktop/projects/OSS/BharatMLStack/horizon/configs/models . 2>/dev/null | \
     docker exec -i $NODE_NAME tar -xzf - -C /tmp/models
   docker exec $NODE_NAME ls -laR /tmp/models/
   ```

4. **Set `LOCAL_MODEL_PATH` in docker-compose.yml:**
   ```yaml
   - LOCAL_MODEL_PATH=/tmp/models  # Path inside the kind node
   ```

### 8.3 Example Configuration

```yaml
# Example with actual values
- ARGOCD_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
- GITHUB_APP_ID=2540855
- GITHUB_INSTALLATION_ID=100332634
- GITHUB_OWNER=Adit2607
- REPOSITORY_NAME=onboarding-test
- BRANCH_NAME=main
- LOCAL_MODEL_PATH=/tmp/models  # For kind clusters
```

---

## Step 9: Verify Setup

### 9.1 Start the Services

```bash
cd quick-start
./start.sh
```

### 9.2 Check Horizon Logs

```bash
# Check if GitHub client initialized successfully
docker-compose logs horizon | grep -i "github\|InitGitHubClient"

# Check for any errors
docker-compose logs horizon | grep -i error
```

### 9.3 Test Onboarding

1. Access Horizon API at http://localhost:8082
2. Create a new deployable/onboarding request
3. Check logs to ensure:
   - GitHub client initializes successfully
   - Files are created in GitHub repository
   - ArgoCD Application YAML is generated

### 9.4 Verify in ArgoCD UI

1. Open ArgoCD UI at https://localhost:8087
2. You should see:
   - The `prd-applications` Application (watches `prd/applications` directory)
   - Any applications automatically created from onboarding (e.g., `prd-test`)
3. Applications are automatically synced when Horizon creates YAML files in your GitHub repo

### 9.5 Verify GitHub Repository

Check your GitHub repository:
- `{workingEnv}/deployables/{appName}/values.yaml` should exist
- `{workingEnv}/applications/{appName}.yaml` should exist
- Example: `prd/deployables/test/values.yaml` and `prd/applications/test.yaml`

**Automated Workflow:**
1. When you onboard a deployable through Horizon, it creates:
   - `prd/deployables/{appName}/values.yaml` - Helm values for the deployment
   - `prd/applications/{appName}.yaml` - ArgoCD Application definition
2. The `prd-applications` ArgoCD Application (created in Step 1.9) watches the `prd/applications` directory
3. ArgoCD automatically detects the new application YAML and creates the ArgoCD Application
4. The application is automatically synced, creating the namespace and deploying the service

---

## Troubleshooting

### Issue: GitHub API 404 Errors

**Solution:**
- Verify the GitHub App has **Contents: Read and write** permission
- Ensure the app is installed to your repository
- Check that `GITHUB_OWNER` matches your GitHub username/org exactly
- Verify `REPOSITORY_NAME` matches the repository name exactly

### Issue: GitHub API 403 Errors

**Solution:**
- Ensure the GitHub App has **Contents: Read and write** permission (not just Read)
- Reinstall the app to your repository if permissions were changed
- Verify the Installation ID is correct

### Issue: ArgoCD Cannot Find Helm Chart

**Solution:**
- **First, verify the repository is added to ArgoCD with proper authentication** (see Step 1.8):
  ```bash
  # Check if repository is added
  argocd repo list
  
  # Test repository connection
  argocd repo get https://github.com/<YOUR_USERNAME>/<YOUR_REPO_NAME>.git
  ```
  If the repository is not listed or connection fails, add it following Step 1.8.
- Verify the Helm chart exists in your repository at the path specified in `ARGOCD_HELMCHART_PATH`
- Check that `ARGOCD_HELMCHART_PATH` in `docker-compose.yml` matches the actual path in your repo
  - Default: `ARGOCD_HELMCHART_PATH=1.0.0` (chart at `1.0.0/` in repo root)
  - Alternative: `ARGOCD_HELMCHART_PATH=predator/1.0.0` (chart at `predator/1.0.0/` in repo)
- Ensure the repository is accessible (public or app has access)
- Verify the application YAML file in `prd/applications/{appName}.yaml` references the correct chart path

### Issue: Repository Authentication Failed

**Error Messages:**
```
repository not found
authentication failed
permission denied
```

**Solution:**

1. **Verify the repository is added to ArgoCD:**
   ```bash
   argocd repo list
   ```
   If your repository is not in the list, add it following Step 1.8.

2. **Test repository connection:**
   ```bash
   argocd repo get https://github.com/<YOUR_USERNAME>/<YOUR_REPO_NAME>.git
   ```
   This should show repository details without errors.

3. **If using Personal Access Token (PAT):**
   - Verify the token has `repo` scope
   - Check if the token has expired
   - Regenerate the token if needed and update ArgoCD:
     ```bash
     argocd repo remove https://github.com/<YOUR_USERNAME>/<YOUR_REPO_NAME>.git
     argocd repo add https://github.com/<YOUR_USERNAME>/<YOUR_REPO_NAME>.git \
       --name <YOUR_REPO_NAME> \
       --type git \
       --username <YOUR_GITHUB_USERNAME> \
       --password <NEW_TOKEN>
     ```

4. **If using SSH:**
   - Verify the SSH key is added to your GitHub account
   - Check SSH key permissions:
     ```bash
     ls -la ~/.ssh/argocd_key
     # Should be readable (600 permissions)
     ```
   - Test SSH connection:
     ```bash
     ssh -T -i ~/.ssh/argocd_key git@github.com
     ```

5. **Check repository access:**
   - Ensure the repository exists and is accessible
   - For private repositories, verify your credentials have access
   - Check if the repository URL is correct (HTTPS vs SSH)

### Issue: Applications Not Appearing in ArgoCD After Onboarding

**Symptoms:**
- Horizon creates files in GitHub repo successfully
- But no ArgoCD Application appears in ArgoCD UI

**Solution:**

1. **Verify `prd-applications` Application exists:**
   ```bash
   kubectl get application prd-applications -n argocd
   ```

2. **Check if `prd-applications` is synced:**
   ```bash
   kubectl get application prd-applications -n argocd -o yaml | grep -A 5 sync
   ```
   - If not synced, manually sync it: `argocd app sync prd-applications`

3. **Verify the application YAML was created in GitHub:**
   - Check `prd/applications/{appName}.yaml` exists in your repo
   - Verify the YAML structure is correct (should be a valid ArgoCD Application resource)

4. **Check ArgoCD logs for errors:**
   ```bash
   kubectl logs -n argocd -l app.kubernetes.io/name=argocd-application-controller --tail=50
   ```

5. **Manually trigger a refresh:**
   ```bash
   argocd app get prd-applications --refresh
   ```

### Issue: Namespace Not Found Error

**Error Message:**
```
namespaces "prd-test" not found
```

**Solution:**

**Namespaces are created automatically!** The Horizon workflow creates ArgoCD Applications with `CreateNamespace=true` in sync options. When ArgoCD syncs the application, it automatically creates the namespace.

**If you see this error, check:**

1. **Verify the Application has CreateNamespace=true:**
   ```bash
   # Check the Application resource
   kubectl get application prd-test -n argocd -o yaml | grep -A 5 syncOptions
   ```
   
   Should show:
   ```yaml
   syncOptions:
   - CreateNamespace=true
   ```

2. **Trigger a sync in ArgoCD:**
   - Go to ArgoCD UI → Your Application
   - Click **Sync** button
   - The namespace will be created automatically during sync

3. **Check ArgoCD RBAC permissions:**
   - ArgoCD needs permission to create namespaces
   - For local development, ArgoCD should have cluster-admin or namespace creation permissions

**Note:** The namespace format is `{env}-{appName}` (e.g., `prd-test`). With `CreateNamespace=true`, ArgoCD creates it automatically - no manual steps needed!

### Issue: Missing Flagger CRD Error

**Error Message:**
```
The Kubernetes API could not find flagger.app/AlertProvider for requested resource prd-test/flagger-status.
```

**Solution:**
- Install Flagger CRDs (see Step 1.2):
  ```bash
  kubectl apply -f https://raw.githubusercontent.com/fluxcd/flagger/main/artifacts/flagger/crd.yaml
  ```
- Once CRDs are installed, ArgoCD will automatically deploy `AlertProvider` resources when syncing

### Issue: HTTPProxy Resource Not Found - Contour CRD Missing

**Error Message:**
```
Resource not found in cluster: projectcontour.io/v1/HTTPProxy:prd-predator-test
The Kubernetes API could not find projectcontour.io/HTTPProxy for requested resource prd-predator-test/prd-predator-test. Make sure the "HTTPProxy" CRD is installed on the destination cluster.
```

**Solution:**
1. **Install Contour CRDs** (see Step 1.2):
   ```bash
   # Install full Contour deployment (includes CRDs + Contour + Envoy)
   kubectl apply -f https://projectcontour.io/quickstart/contour.yaml
   
   # Or install only CRDs if you have Contour already running:
   kubectl apply -f https://raw.githubusercontent.com/projectcontour/contour/main/examples/contour/01-crds.yaml
   ```

2. **Verify installation:**
   ```bash
   kubectl get crd httpproxies.projectcontour.io
   ```

3. **Check your values.yaml** to ensure HTTPProxy conditions are met:
   - `ingress.enabled: true` ✓
   - `createContourGateway: true` ✓
   - `ingressClassName: "contour-internal"` (or `contour-external`, `contour-internal-0`, etc.) ✓
   - `ingress.hosts` is set (host should be generated as `<appname>.<domain>`) ✓

4. **Once CRDs are installed, trigger ArgoCD sync:**
   ```bash
   argocd app sync prd-predator-test
   ```
   Or wait for automatic sync (if enabled)

**Note:** The HTTPProxy template requires all three conditions to be true. If any condition fails, the HTTPProxy won't be rendered in the Helm chart output.

### Issue: Missing KEDA CRD Error

**Error Message:**
```
The Kubernetes API could not find keda.sh/ScaledObject for requested resource prd-test/prd-test.
```

**Solution:**
- Install KEDA ScaledObject CRD (see Step 1.2):
  ```bash
  # Install only ScaledObject CRD (required for Predator)
  # Note: ScaledJob CRD is skipped due to oversized annotations that exceed Kubernetes limits
  kubectl apply -f https://raw.githubusercontent.com/kedacore/keda/v2.12.0/config/crd/bases/keda.sh_scaledobjects.yaml
  ```
- Verify installation:
  ```bash
  kubectl get crd scaledobjects.keda.sh
  ```
- Once CRDs are installed, ArgoCD will successfully deploy `ScaledObject` resources when syncing

### Issue: KEDA ScaledJob CRD Annotation Size Error

**Error Message:**
```
The CustomResourceDefinition "scaledjobs.keda.sh" is invalid: metadata.annotations: Too long: may not be more than 262144 bytes
```

**Solution:**
- This error occurs because the `scaledjobs.keda.sh` CRD has oversized annotations that exceed Kubernetes limits
- **Predator only requires `ScaledObject` CRD, not `ScaledJob`**
- The setup script automatically installs only the required `ScaledObject` CRD
- If you manually installed `ScaledJob` CRD and see this error, you can safely ignore it or delete the CRD:
  ```bash
  kubectl delete crd scaledjobs.keda.sh
  ```
- Only install `ScaledObject` CRD for Predator:
  ```bash
  kubectl apply -f https://raw.githubusercontent.com/kedacore/keda/v2.12.0/config/crd/bases/keda.sh_scaledobjects.yaml
  ```

### Issue: PriorityClass Not Found Error

**Error Message:**
```
pods "prd-test-57ff5ffd59-" is forbidden: no PriorityClass with name high-priority was found
```

**Solution:**
- Create the PriorityClass (see Step 1.2):
  ```bash
  kubectl apply -f - <<EOF
  apiVersion: scheduling.k8s.io/v1
  kind: PriorityClass
  metadata:
    name: high-priority
  value: 1000
  globalDefault: false
  description: "High priority class for application pods"
  EOF
  ```
- Verify installation:
  ```bash
  kubectl get priorityclass high-priority
  ```
- Once created, ArgoCD will successfully deploy pods with this priority class
- **Note:** PriorityClass is cluster-scoped, so you only need to create it once per cluster

### Issue: Docker Disk Space - "no space left on device"

**Error Message:**
```
failed to pull and unpack image: no space left on device
```

**Solution:**

The Triton server full image (`25.06-py3`) is **~15GB+**. You **must increase Docker Desktop's disk allocation** - cleaning up space alone won't be sufficient.

**Option 1: Increase Docker Disk Space (REQUIRED for Full Image)**

**For Docker Desktop on macOS:**

1. **Open Docker Desktop**
2. Click the **Settings** (gear icon) in the top right
3. Go to **Resources** → **Advanced**
4. Find **"Disk image size"** (or "Disk image location")
5. **Increase the size** to at least **100GB** (recommended: 120-150GB to have buffer)
   - Current default is often 60GB, which is insufficient
   - The Triton image alone needs ~15GB, plus your existing containers/volumes
6. Click **"Apply & Restart"**
   - Docker Desktop will restart and resize the disk image
   - This may take a few minutes

**For Docker Desktop on Windows:**

1. Open Docker Desktop
2. Go to **Settings** → **Resources** → **Advanced**
3. Increase **"Disk image size"** to at least **100GB**
4. Click **"Apply & Restart"**

**Verify disk space after restart:**
```bash
docker system df
```

**Option 2: Clean Up Docker Resources (Do This First)**

Before increasing disk size, clean up unused resources:

```bash
# Check current disk usage
docker system df

# Remove unused containers, networks, images, and build cache
docker system prune -a -f

# Remove unused volumes (be careful - this removes all unused volumes)
# Only run this if you don't need any stopped containers' data
docker volume prune -f

# Remove specific unused images
docker image prune -a -f
```

**Option 3: Use Minimal Image (If You Can't Increase Disk Space)**

If you cannot increase Docker's disk allocation, use the minimal image variant:
- Change `triton_image_tags` from `25.06-py3` to `25.06-py3-min` in your database
- Note: The minimal image may have limitations and may not include `tritonserver` in PATH

**After increasing disk space, update the database:**

```bash
# Connect to MySQL and ensure the image tag is set to full image
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  UPDATE deployable_metadata 
  SET value = '25.06-py3' 
  WHERE \`key\` = 'triton_image_tags' AND id = 6;
"
```

Then re-run the onboarding workflow to use the full image.

Then re-run the onboarding workflow to use the full image.

### Issue: Node Affinity/Selector Not Matching

**Error Message:**
```
0/1 nodes are available: 1 node(s) didn't match Pod's node affinity/selector.
no new claims to deallocate, preemption: 0/1 nodes are available: 
1 Preemption is not helpful for scheduling.
```

**Solution:**

**First, ensure you completed Step 1.1** (Label Kubernetes Node for Pod Scheduling). This step should have labeled your node correctly. If you skipped it or the label was removed, follow Step 1.1 to label your node.

If you've already completed Step 1.1 and still see this error, the node label may not match the `nodeSelectorValue` in your Helm values.yaml. The Helm chart uses `nodeSelector: dedicated: <value>` to schedule pods on specific nodes. The node label must match the nodeSelector value in your values.yaml.

**Step 1: Check what the pod is requesting:**
```bash
kubectl get pod -n <namespace> -o jsonpath='{.items[0].spec.nodeSelector}'
```

**Step 2: Check current node labels:**
```bash
kubectl get nodes --show-labels | grep dedicated
```

**Step 3: Update the node label to match:**
```bash
# Get your node name
NODE_NAME=$(kubectl get nodes -o name | head -1 | sed 's|node/||')

# Update the label to match your nodeSelector value
# Replace <your-nodeSelector-value> with the value from your values.yaml
kubectl label node $NODE_NAME dedicated=<your-nodeSelector-value> --overwrite

# Verify the label
kubectl get nodes --show-labels | grep dedicated
```

**Example:**
If your values.yaml has `nodeSelectorValue: "bharatml-stack-control-plane"`, then:
```bash
NODE_NAME=$(kubectl get nodes -o name | head -1 | sed 's|node/||')
kubectl label node $NODE_NAME dedicated=bharatml-stack-control-plane --overwrite
```

**Alternative: Remove nodeSelector for local development**

If you want to remove nodeSelector requirements for local development, edit your `prd/deployables/{appName}/values.yaml` in GitHub:
```yaml
nodeSelectorValue: ""  # Empty value will prevent nodeSelector from being applied
```
Then sync the ArgoCD application to pick up the change.

**Note:** Labeling the node to match your nodeSelector is recommended as it matches production behavior without modifying Helm values.

### Issue: ArgoCD Token Expired

**Solution:**
- Generate a new token from ArgoCD UI (User Info → Generate New Token)
- Update `ARGOCD_TOKEN` in docker-compose.yml
- Restart Horizon service: `./restart.sh horizon`

### Issue: Config File Not Found Error

**Error Message:**
```
config.yaml not found for environment 'prd' and service 'predator' 
(expected: configs/services/predator/prd/config.yaml): 
failed to read service config file at /app/configs/services/predator/prd/config.yaml
```

**Solution:**

The `start.sh` script automatically copies `horizon/configs` to `workspace/configs` during setup. If you see this error:

1. **Verify config file exists in source:**
   ```bash
   ls -la horizon/configs/services/predator/prd/config.yaml
   ```

2. **Verify configs were copied to workspace:**
   ```bash
   ls -la quick-start/workspace/configs/services/predator/prd/config.yaml
   ```

3. **If configs are missing in workspace, re-run start.sh:**
   ```bash
   cd quick-start
   ./start.sh
   ```
   The `start.sh` script will copy the configs directory automatically.

4. **Check volume mount in docker-compose.yml:**
   - Should have: `- ./configs:/app/configs:ro` (relative to workspace directory)
   - And: `SERVICE_CONFIG_PATH=/app/configs`

5. **If still not working, manually copy configs:**
   ```bash
   cd quick-start
   cp -r ../horizon/configs workspace/
   cd workspace
   docker-compose restart horizon
   ```

**Note:** The `start.sh` script automatically copies `horizon/configs` to `workspace/configs` during initial setup. If you see this error, it usually means the workspace wasn't set up properly or the configs weren't copied.

### Issue: Private Key Not Found

**Solution:**
- Verify `github.pem` exists in `horizon/configs/github.pem`
- Ensure `start.sh` copied the configs directory (it should do this automatically)
- Check container logs: `docker-compose logs horizon | grep github.pem`

---

## Summary Checklist

### Automated by Script (✅ Done automatically)
- [x] Kubernetes cluster created and running (if not exists)
- [x] Node labeled with `dedicated` label
- [x] Required CRDs and PriorityClass installed
- [x] ArgoCD installed and running in local Kubernetes
- [x] ArgoCD admin password retrieved
- [x] API key permissions enabled for admin account
- [x] GitHub repository added to ArgoCD with GitHub App authentication
- [x] prd-applications ArgoCD Application created (optional)

### Manual Steps (Still Required)
- [ ] Run the setup script: `cd quick-start && ./setup-predator-k8s.sh`
- [ ] Generate ArgoCD API token (see Step 1.7) - Access ArgoCD UI at https://localhost:8087
- [ ] Create GitHub App (Step 2-5)
- [ ] Place GitHub App private key in `horizon/configs/github.pem` (Step 7)
- [ ] Copy Predator Helm chart to repository (Step 6)
- [ ] Commit and push repository changes
- [ ] Configure docker-compose.yml with all values (Step 8)
- [ ] Start services: `cd quick-start && ./start.sh`
- [ ] Verify Horizon logs show no errors
- [ ] Test onboarding works

---

## Automated Workflow

Once setup is complete, everything happens automatically via GitOps:

1. **Onboarding Request** → Horizon API receives request
2. **GitHub Push** → Horizon automatically creates:
   - `{env}/deployables/{appName}/values.yaml`
   - `{env}/applications/{appName}.yaml` (with `CreateNamespace=true`)
3. **ArgoCD Auto-Sync** → ArgoCD automatically:
   - Detects new Application YAML in GitHub
   - **Creates namespace automatically** (via `CreateNamespace=true`)
   - Syncs Helm chart and deploys all resources
   - Deploys `AlertProvider` (if Flagger CRDs are installed)

**No manual namespace creation needed!** Everything is automated.

---

## Step 10: Access Predator Service Through Contour HTTPProxy

Once the HTTPProxy is created and Contour is running, you can access your Predator service through Contour's Envoy proxy.

### 10.1 Check HTTPProxy Status

```bash
# List all HTTPProxies (find your namespace)
kubectl get httpproxy -A

# Check HTTPProxy status in your namespace
kubectl get httpproxy -n <your-namespace>

# Get HTTPProxy details including FQDN
kubectl get httpproxy -n <your-namespace> <httpproxy-name> -o yaml | grep -A 5 "virtualhost:"

# Get FQDN directly
kubectl get httpproxy -n <your-namespace> <httpproxy-name> -o jsonpath='{.spec.virtualhost.fqdn}'
```

The HTTPProxy should show `Valid` status once Contour controller reconciles it.

### 10.2 Access via Port Forward (Recommended for Local Development)

For local development, especially with kind clusters, port-forward is the most reliable method:

**Prerequisites:** Ensure Contour is installed (see Step 1.0)

```bash
# Step 1: Verify Contour is installed
kubectl get namespace projectcontour
kubectl get pods -n projectcontour

# If namespace doesn't exist, install Contour first:
kubectl apply -f https://projectcontour.io/quickstart/contour.yaml

# Wait for Contour pods to be ready
kubectl wait --for=condition=ready pod -l app=contour -n projectcontour --timeout=120s
kubectl wait --for=condition=ready pod -l app=envoy -n projectcontour --timeout=120s

# Create IngressClass and configure Contour (see Step 1.0 for details)
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: contour-internal
spec:
  controller: projectcontour.io/ingress-controller
EOF

# Configure Contour to watch for contour-internal
# First check if the argument already exists
if ! kubectl -n projectcontour get deploy contour -o jsonpath='{.spec.template.spec.containers[0].args}' | grep -q "ingress-class-name"; then
  kubectl -n projectcontour patch deploy contour --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--ingress-class-name=contour-internal"}]'
else
  echo "Contour already configured for ingress class"
fi

# Restart Contour to apply changes
kubectl -n projectcontour rollout restart deploy contour
kubectl -n projectcontour rollout status deploy contour --timeout=120s

# Verify the configuration (should show --ingress-class-name=contour-internal only once)
kubectl -n projectcontour get deploy contour -o jsonpath='{.spec.template.spec.containers[0].args}' | grep ingress-class-name

# Step 2: Port forward Envoy service (Contour's ingress proxy)
# Run this in a terminal and keep it running, or run in background:
# Format: kubectl port-forward -n projectcontour svc/envoy <LOCAL_PORT>:<SERVICE_PORT>
# Envoy service exposes port 80 (HTTP) and 443 (HTTPS)

# Option A: Use port 8080 (if available)
kubectl port-forward -n projectcontour svc/envoy 8080:80


# Step 2: Get your HTTPProxy FQDN
# Automatically find the first HTTPProxy with an FQDN (works for most cases)
FQDN=$(kubectl get httpproxy -A -o jsonpath='{range .items[*]}{.metadata.namespace}{"\t"}{.metadata.name}{"\t"}{.spec.virtualhost.fqdn}{"\n"}{end}' | grep -v "^$" | awk '{print $3}' | head -1)
echo "FQDN: $FQDN"

# Alternative: List all HTTPProxies to find yours
kubectl get httpproxy -A -o custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name,FQDN:.spec.virtualhost.fqdn

# Step 3: Access your service using the FQDN as Host header
curl -H "Host: $FQDN" http://localhost:8080/

# Example for health check (adjust path based on your service):
curl -H "Host: $FQDN" http://localhost:8080/v2/self/health
# or
curl -H "Host: $FQDN" http://localhost:8080/self/health

# Note: If you see "grpc-status" in the response, the endpoint is gRPC-only
# Use a gRPC client (like grpcurl) instead of curl for gRPC endpoints
```

### 10.3 Access via LoadBalancer or NodePort

**Check Envoy Service Configuration:**

```bash
# Check Envoy service type and status
kubectl get svc -n projectcontour envoy

# You'll see one of:
# - LoadBalancer with EXTERNAL-IP (cloud clusters)
# - LoadBalancer with <pending> (local clusters - use NodePort or port-forward)
# - NodePort with assigned port (if configured)
```

**For Local Clusters (kind/minikube/Docker Desktop):**

Local clusters typically show `<pending>` for LoadBalancer. Use one of these methods:

**Option A: Port Forward (Recommended - Simplest)**

```bash
# Port forward Envoy service to localhost
kubectl port-forward -n projectcontour svc/envoy 8080:80

# In another terminal, access your service with Host header
curl -H "Host: predator-test.prd.meesho.int" http://localhost:8080/self/health

# Or for other endpoints:
curl -H "Host: predator-test.prd.meesho.int" http://localhost:8080/
```

**Option B: Use NodePort (If Available)**

NodePort is automatically assigned when using LoadBalancer type in local clusters. Access methods vary by cluster type:

```bash
# Get the NodePort assigned to Envoy
NODEPORT=$(kubectl get svc -n projectcontour envoy -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}')
echo "NodePort: $NODEPORT"

# Get your HTTPProxy FQDN
FQDN=$(kubectl get httpproxy -n prd-predator-test prd-predator-test -o jsonpath='{.spec.virtualhost.fqdn}' 2>/dev/null || echo "predator-test.prd.meesho.int")
echo "FQDN: $FQDN"
```

**For kind clusters:**

⚠️ **Important:** kind does NOT automatically expose NodePort on localhost. NodePort access will fail with "Empty reply from server" or connection errors.

**✅ Use Port-Forward (Recommended and Only Working Method for kind)**

This is the ONLY reliable method for kind clusters. NodePort will NOT work on localhost.

```bash
# Step 1: Port forward Envoy service to localhost (run in background or separate terminal)
kubectl port-forward -n projectcontour svc/envoy 8080:80

# Step 2: Get your HTTPProxy FQDN
# Automatically get the first HTTPProxy FQDN (works for most cases)
FQDN=$(kubectl get httpproxy -A -o jsonpath='{range .items[*]}{.metadata.namespace}{"\t"}{.metadata.name}{"\t"}{.spec.virtualhost.fqdn}{"\n"}{end}' | grep -v "^$" | awk '{print $3}' | head -1)
echo "Using FQDN: $FQDN"

# Alternative: List all HTTPProxies to find yours
kubectl get httpproxy -A -o custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name,FQDN:.spec.virtualhost.fqdn

# Step 3: Access your service with Host header
# Replace 8080 with your actual local port if you used a different one
LOCAL_PORT=8080  # Change to 8081, 9090, etc. if 8080 is in use

# For HTTP endpoints:
curl -H "Host: $FQDN" http://localhost:$LOCAL_PORT/

# For health check (adjust path based on your service):
curl -H "Host: $FQDN" http://localhost:$LOCAL_PORT/v2/self/health
# or
curl -H "Host: $FQDN" http://localhost:$LOCAL_PORT/self/health

# Note: If you get "grpc-status: 2" or "Bad method header", the endpoint might be gRPC-only
# Use a gRPC client instead of curl for gRPC endpoints
```

**Why NodePort Doesn't Work on kind:**

When you try to access NodePort on kind:
```bash
NODEPORT=$(kubectl get svc -n projectcontour envoy -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}')
curl -H "Host: $FQDN" http://localhost:$NODEPORT/v2/self/health
# Result: curl: (52) Empty reply from server
```

This happens because:
- kind runs nodes in Docker containers
- NodePort services are only accessible from within the Docker network
- localhost doesn't have access to the kind node's network namespace
- Port-forward creates a tunnel that bridges this gap

**Solution: Always use port-forward for kind clusters**

**Alternative: Configure kind with Port Mapping (Not Recommended for Existing Clusters)**

If you really need NodePort access, you must recreate your kind cluster with port mapping:

```bash
# Create kind cluster with port mapping (example)
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 32558  # Your NodePort (must match exactly)
    hostPort: 32558
    protocol: TCP
EOF

# Then access via localhost
curl -H "Host: $FQDN" http://localhost:32558/v2/self/health
```

**⚠️ Warning:** This requires:
- Destroying your existing cluster
- Recreating all resources (ArgoCD, Contour, applications, etc.)
- Reconfiguring everything

**Recommendation:** Use port-forward instead - it's simpler and works immediately without cluster recreation.

**For minikube:**

```bash
# Method 1: Use minikube service command (automatically sets up tunnel)
minikube service -n projectcontour envoy --url
# This will output a URL like: http://192.168.49.2:32558
# Use it with Host header:
curl -H "Host: $FQDN" http://192.168.49.2:$NODEPORT/self/health

# Method 2: Get minikube IP and use NodePort directly
MINIKUBE_IP=$(minikube ip)
curl -H "Host: $FQDN" http://$MINIKUBE_IP:$NODEPORT/self/health
```

**For Docker Desktop Kubernetes:**

```bash
# Docker Desktop exposes NodePort on localhost
curl -H "Host: $FQDN" http://localhost:$NODEPORT/self/health
```

**Note:** NodePort access may require firewall rules or port forwarding depending on your setup. Port-forward (Option A) is more reliable for local development.

**For Cloud Clusters (GKE/EKS/AKS) with LoadBalancer:**

```bash
# Wait for external IP to be assigned (may take 1-2 minutes)
kubectl wait --for=condition=ready svc/envoy -n projectcontour --timeout=5m

# Get the external IP
EXTERNAL_IP=$(kubectl get svc -n projectcontour envoy -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
# Or for hostname-based load balancers:
# EXTERNAL_IP=$(kubectl get svc -n projectcontour envoy -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')

echo "External IP: $EXTERNAL_IP"

# Access using the external IP with Host header
curl -H "Host: predator-test.prd.meesho.int" http://$EXTERNAL_IP/self/health

# Example for other endpoints:
curl -H "Host: predator-test.prd.meesho.int" http://$EXTERNAL_IP/
```

**Quick Test Command:**

```bash
# Get your HTTPProxy FQDN
FQDN=$(kubectl get httpproxy -n prd-predator-test prd-predator-test -o jsonpath='{.spec.virtualhost.fqdn}')
echo "FQDN: $FQDN"

# Test with port-forward (if using local cluster)
kubectl port-forward -n projectcontour svc/envoy 8080:80 &
sleep 2
curl -H "Host: $FQDN" http://localhost:8080/self/health
kill %1 2>/dev/null || true
```

**Summary for Local Clusters:**

| Cluster Type | Recommended Method | NodePort Access |
|-------------|-------------------|-----------------|
| **kind** | Port-forward (Option A) | ❌ Not available on localhost (requires cluster recreation with port mapping) |
| **minikube** | `minikube service` or port-forward | ✅ Available via `minikube service` or minikube IP |
| **Docker Desktop** | Port-forward or NodePort | ✅ Available on localhost |

**Important Notes:**
- For local development, **port-forward (Option A) is the simplest and most reliable method** for all cluster types
- LoadBalancer services in local clusters (kind/minikube) often show `<pending>` status indefinitely
- The Host header **must match** the FQDN in your HTTPProxy's `virtualhost.fqdn`
- You can verify your FQDN with: `kubectl get httpproxy -n <namespace> <name> -o jsonpath='{.spec.virtualhost.fqdn}'`
- For kind clusters, NodePort is NOT accessible on localhost unless you configure port mapping during cluster creation

### 10.4 Access via /etc/hosts (Alternative for Local Testing)

For easier local testing without Host headers:

```bash
# Add to /etc/hosts (replace <EXTERNAL-IP> with Envoy external IP or use 127.0.0.1 if port-forwarding)
echo "127.0.0.1 predator-test.prd.meesho.int" | sudo tee -a /etc/hosts

# Then access directly (if using port-forward on port 8080)
curl http://predator-test.prd.meesho.int:8080/
```

### 10.5 Verify Service is Accessible

```bash
# Check if the service is running
kubectl get pods -n prd-predator-test

# Check service endpoints
kubectl get endpoints -n prd-predator-test

# Test a health check endpoint (if available)
curl -H "Host: predator-test.prd.meesho.int" http://localhost:8080/health
```

### 10.6 Troubleshooting HTTPProxy Access

**Issue: HTTPProxy shows "NotReconciled" or "Invalid" status**

```bash
# Check Contour controller logs
kubectl logs -n projectcontour -l app=contour | tail -50

# Check HTTPProxy status details
kubectl describe httpproxy -n prd-predator-test

# Verify the service exists and has endpoints
kubectl get svc -n prd-predator-test
kubectl get endpoints -n prd-predator-test
```

**Issue: 404 Not Found when accessing**

- Verify the FQDN matches what's in HTTPProxy: `kubectl get httpproxy -n prd-predator-test -o yaml | grep fqdn`
- Ensure you're using the correct Host header
- Check that the service name in HTTPProxy matches your actual service: `kubectl get svc -n prd-predator-test`

**Issue: Connection refused**

- Ensure Envoy is running: `kubectl get pods -n projectcontour -l app=envoy`
- Check Envoy logs: `kubectl logs -n projectcontour -l app=envoy | tail -50`

**Issue: Port 8080 Already in Use**

If you get `bind: address already in use` when trying to port-forward:

```bash
# Find what's using port 8080
lsof -i :8080

# Use a different local port (8081, 9090, 8888, etc.)
kubectl port-forward -n projectcontour svc/envoy 8081:80

# Then access using the new port
curl -H "Host: $FQDN" http://localhost:8081/
```

**Note:** The Envoy service exposes port 80 (HTTP) and 443 (HTTPS). Always use `:80` as the service port in port-forward commands.

**Issue: HTTPProxy Shows "NotReconciled" / "Waiting for controller" Status**

This is the most common issue with HTTPProxy. The root cause is usually that Contour is not configured to watch for your specific ingress class. To resolve:

1. **Most Common Cause: Contour Not Configured for Your Ingress Class**

   If your HTTPProxy uses `ingressClassName: contour-internal`, Contour must be configured to watch for that class:
   
   ```bash
   # Check if Contour is configured for your ingress class
   kubectl -n projectcontour get deploy contour -o yaml | grep ingress-class-name
   
   # If nothing is returned, Contour is not watching for contour-internal
   # Configure Contour to watch for contour-internal:
   kubectl -n projectcontour patch deploy contour --type='json' -p='[
     {
       "op": "add",
       "path": "/spec/template/spec/containers/0/args/-",
       "value": "--ingress-class-name=contour-internal"
     }
   ]'
   
   # Restart Contour
   kubectl -n projectcontour rollout restart deploy contour
   kubectl -n projectcontour rollout status deploy contour
   
   # Verify HTTPProxy status (should change to "valid" after a few seconds)
   kubectl get httpproxy -n prd-predator-test-2
   ```

2. **Verify Contour CRD is installed:**
   ```bash
   kubectl get crd httpproxies.projectcontour.io
   ```

3. **Verify IngressClass exists:**
   ```bash
   kubectl get ingressclass contour-internal
   # If missing, create it:
   kubectl apply -f - <<EOF
   apiVersion: networking.k8s.io/v1
   kind: IngressClass
   metadata:
     name: contour-internal
   spec:
     controller: projectcontour.io/ingress-controller
   EOF
   ```

4. **Verify HTTPProxy has virtualhost.fqdn:**
   ```bash
   kubectl get httpproxy -n prd-predator-test -o yaml | grep -A 3 "virtualhost:"
   # If missing, add it:
   kubectl patch httpproxy -n prd-predator-test prd-predator-test --type=merge -p '{"spec":{"virtualhost":{"fqdn":"predator-test.prd.meesho.int"}}}'
   ```

5. **Check Contour controller is running:**
   ```bash
   kubectl get pods -n projectcontour -l app=contour
   ```

**Note:** After configuring Contour with `--ingress-class-name=contour-internal`, HTTPProxy status should change from "NotReconciled" to "valid" within a few seconds.

---

## Next Steps

After completing this setup:

1. **Test Onboarding**: Create a deployable through the Horizon API
2. **Monitor ArgoCD**: Watch applications sync automatically in ArgoCD UI
3. **Access Services**: Use Contour HTTPProxy to access your Predator services
4. **Customize Values**: Modify Helm chart values in GitHub (ArgoCD will auto-sync)
5. **Add More Environments**: Configure additional working environments if needed

For more information, refer to:
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [Contour Documentation](https://projectcontour.io/docs/)
- [GitHub Apps Documentation](https://docs.github.com/en/apps)
- [Predator Helm Chart](../predator/1.0.0/README.md)
