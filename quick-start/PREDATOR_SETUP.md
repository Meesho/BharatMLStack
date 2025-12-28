# Predator Local Setup Guide

This guide will walk you through setting up Predator for local development, including ArgoCD configuration, GitHub App setup, and repository preparation.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Step 1: Set Up Local ArgoCD](#step-1-set-up-local-argocd)
3. [Step 2: Create a GitHub App](#step-2-create-a-github-app)
4. [Step 3: Configure GitHub App Permissions](#step-3-configure-github-app-permissions)
5. [Step 4: Install GitHub App to Your Repository](#step-4-install-github-app-to-your-repository)
6. [Step 5: Generate and Download Private Key](#step-5-generate-and-download-private-key)
7. [Step 6: Copy Predator Helm Chart to Repository](#step-6-copy-predator-helm-chart-to-repository)
8. [Step 7: Place GitHub Private Key in Horizon Configs](#step-7-place-github-private-key-in-horizon-configs)
9. [Step 8: Configure Docker Compose](#step-8-configure-docker-compose)
10. [Step 9: Verify Setup](#step-9-verify-setup)

---

## Prerequisites

- Docker and Docker Compose installed
- **Docker Desktop disk allocation: At least 100GB** (required for Triton server image ~15GB)
  - Check: Docker Desktop → Settings → Resources → Advanced → "Disk image size"
  - Increase if less than 100GB (see [Docker Disk Space troubleshooting](#issue-docker-disk-space---no-space-left-on-device))
- Kubernetes cluster running locally (e.g., minikube, kind, Docker Desktop Kubernetes)
- Argocd cli running locally to interact with argocd
- kubectl configured to access your local cluster
- A GitHub account
- A GitHub repository for storing Helm charts and ArgoCD applications

---

## Step 1: Set Up Local ArgoCD

### 1.0 Install Required CRDs and PriorityClass

The Predator Helm chart uses several Custom Resource Definitions (CRDs) and a PriorityClass that must be installed in your Kubernetes cluster before ArgoCD can deploy resources.

#### Install Flagger CRDs (Required for AlertProvider)

The `AlertProvider` resource in the Predator Helm chart requires Flagger CRDs to be installed. ArgoCD can deploy `AlertProvider` resources, but the CRD must exist in the cluster first.

```bash
# Install Flagger CRDs
kubectl apply -f https://raw.githubusercontent.com/fluxcd/flagger/main/artifacts/flagger/crd.yaml

# Verify installation
kubectl get crd | grep flagger
```

You should see CRDs like:
- `alertproviders.flagger.app`
- `canaries.flagger.app`
- `metrictemplates.flagger.app`

#### Install KEDA CRDs (Required for ScaledObject)

The `ScaledObject` resource requires KEDA CRDs to be installed. This is used for autoscaling based on custom metrics.

```bash
# Install KEDA CRDs
kubectl apply -f https://github.com/kedacore/keda/releases/download/v2.12.0/keda-2.12.0.yaml

# Or install only CRDs (lighter weight)
kubectl apply -f https://raw.githubusercontent.com/kedacore/keda/v2.12.0/config/crd/bases/keda.sh_scaledobjects.yaml
kubectl apply -f https://raw.githubusercontent.com/kedacore/keda/v2.12.0/config/crd/bases/keda.sh_scaledjobs.yaml
kubectl apply -f https://raw.githubusercontent.com/kedacore/keda/v2.12.0/config/crd/bases/keda.sh_triggerauthentications.yaml

# Verify installation
kubectl get crd | grep keda
```

You should see CRDs like:
- `scaledobjects.keda.sh`
- `scaledjobs.keda.sh`
- `triggerauthentications.keda.sh`

#### Install PriorityClass (Required for Pod Scheduling)

The Helm chart uses `priorityClassName: high-priority` for pod scheduling. You need to create this PriorityClass in your cluster.

```bash
# Create PriorityClass for high-priority pods
kubectl apply -f - <<EOF
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: high-priority
value: 1000
globalDefault: false
description: "High priority class for application pods"
EOF

# Verify installation
kubectl get priorityclass high-priority
```

**Note:** PriorityClass is a cluster-scoped resource, so you only need to create it once per cluster. Once created, ArgoCD will successfully deploy pods with this priority class.

**Note:** Once the CRDs and PriorityClass are installed, ArgoCD will successfully deploy these resources when syncing your application.

### 1.1 Install ArgoCD in Your Local Kubernetes Cluster

```bash
# Create ArgoCD namespace
kubectl create namespace argocd

# Install ArgoCD
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ArgoCD to be ready (this may take a few minutes)
kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n argocd
```

### 1.2 Port Forward ArgoCD Server

```bash
# Port forward ArgoCD server to access UI and API
kubectl port-forward svc/argocd-server -n argocd 8087:443
```

**Note:** Keep this terminal session running. In a new terminal, you can access:
- **ArgoCD UI**: https://localhost:8087 (accept the self-signed certificate warning)
- **ArgoCD API**: http://localhost:8087

### 1.3 Get ArgoCD Admin Password

```bash
# Get the initial admin password
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d && echo
```

**Default username:** `admin`

### 1.4 Generate ArgoCD API Token

1. Log in to ArgoCD UI at https://localhost:8087
2. Go to **User Info** (click on your username in the top right)
3. Click **Generate New Token**
4. Copy the generated token (you'll need this for `ARGOCD_TOKEN` in docker-compose.yml)

**Alternative (using CLI):**

```bash
# Install ArgoCD CLI (if not already installed)
# macOS
brew install argocd

# Login to ArgoCD
argocd login localhost:8087 --insecure

# Generate token
argocd account generate-token
```

### 1.5 Create ArgoCD Application (Alternative to Manual Creation)

If you prefer to create the ArgoCD Application via CLI instead of through the UI, you can use:

```bash
argocd app create prd-test \
  --repo https://github.com/<REPO_OWNER>/<REPO_NAME>.git \
  --revision main \
  --path 1.0.0 \
  --dest-server https://kubernetes.default.svc \
  --dest-namespace prd-test \
  --values ../prd/deployables/test/values.yaml \
  --sync-policy automated \
  --self-heal \
  --auto-prune \
  --sync-option CreateNamespace=true
```

**Important:** After creating the application, you **must** add the `app_name` label manually, as the CLI command doesn't support labels. The label is required for threshold updates to work correctly:

```bash
# Add the app_name label (use the actual app name without environment prefix)
kubectl label application prd-test app_name=test -n argocd --overwrite
```

Alternatively, you can use `kubectl patch`:

```bash
kubectl patch application prd-test -n argocd --type merge -p '{"metadata":{"labels":{"app_name":"test"}}}'
```

**Note:** Replace:
- `prd-test` with your application name (`{env}-{appName}`)
- `<REPO_OWNER>/<REPO_NAME>` with your GitHub repository
- `test` with your actual app name (without environment prefix) - this is what goes in the `app_name` label

**Important:** The `--values` path should be relative to the repository root. The path `prd/deployables/test/values.yaml` means the file is at the root of your repo under `prd/deployables/test/values.yaml`.

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
cp -r predator/1.0.0 YOUR_REPO_NAME/

# Or if you're already in the repo directory
cp -r ../predator/1.0.0 ./
```

### 6.3 Commit and Push

```bash
# Add the chart
git add predator/1.0.0

# Commit
git commit -m "Add Predator Helm chart 1.0.0"

# Push to main branch (or your default branch)
git push origin main
```

**Verify:** Check that `predator/1.0.0/` exists in your repository at the root level.

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
    - ARGOCD_TOKEN=<YOUR_ARGOCD_TOKEN>  # From Step 1.4
    - ARGOCD_NAMESPACE=argocd
    - ARGOCD_DESTINATION_NAME=in-cluster  # For local Kubernetes
    - ARGOCD_PROJECT=default
    - ARGOCD_HELMCHART_PATH=predator/1.0.0  # Path to Helm chart in your repo
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
```

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

#### For Docker Desktop Kubernetes:

Docker Desktop typically has access to `/Users` paths, so you can use your host path directly:
```yaml
- LOCAL_MODEL_PATH=/Users/adityakumargarg/Desktop/projects/OSS/BharatMLStack/horizon/configs/models
```

#### For minikube:

1. **SSH into minikube:**
   ```bash
   minikube ssh
   ```

2. **Copy models:**
   ```bash
   # From your host machine
   minikube cp /path/to/your/models /tmp/models
   ```

3. **Set `LOCAL_MODEL_PATH` in docker-compose.yml:**
   ```yaml
   - LOCAL_MODEL_PATH=/tmp/models
   ```

**Note:** If you update your models, you'll need to copy them again to the Kubernetes node.

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
2. You should see the application created (if onboarding was successful)
3. Or manually create an application pointing to your repository

### 9.5 Verify GitHub Repository

Check your GitHub repository:
- `{workingEnv}/deployables/{appName}/values.yaml` should exist
- `{workingEnv}/applications/{appName}.yaml` should exist
- Example: `prd/deployables/test/values.yaml` and `prd/applications/test.yaml`

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
- Verify `predator/1.0.0` exists in your repository
- Check that `ARGOCD_HELMCHART_PATH=predator/1.0.0` matches the path in your repo
- Ensure the repository is accessible (public or app has access)

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
- Install Flagger CRDs (see Step 1.0):
  ```bash
  kubectl apply -f https://raw.githubusercontent.com/fluxcd/flagger/main/artifacts/flagger/crd.yaml
  ```
- Once CRDs are installed, ArgoCD will automatically deploy `AlertProvider` resources when syncing

### Issue: Missing KEDA CRD Error

**Error Message:**
```
The Kubernetes API could not find keda.sh/ScaledObject for requested resource prd-test/prd-test.
```

**Solution:**
- Install KEDA CRDs (see Step 1.0):
  ```bash
  # Install KEDA CRDs (full installation)
  kubectl apply -f https://github.com/kedacore/keda/releases/download/v2.12.0/keda-2.12.0.yaml
  
  # Or install only CRDs (lighter weight, recommended)
  kubectl apply -f https://raw.githubusercontent.com/kedacore/keda/v2.12.0/config/crd/bases/keda.sh_scaledobjects.yaml
  kubectl apply -f https://raw.githubusercontent.com/kedacore/keda/v2.12.0/config/crd/bases/keda.sh_scaledjobs.yaml
  kubectl apply -f https://raw.githubusercontent.com/kedacore/keda/v2.12.0/config/crd/bases/keda.sh_triggerauthentications.yaml
  ```
- Verify installation:
  ```bash
  kubectl get crd | grep keda
  ```
- Once CRDs are installed, ArgoCD will successfully deploy `ScaledObject` resources when syncing

### Issue: PriorityClass Not Found Error

**Error Message:**
```
pods "prd-test-57ff5ffd59-" is forbidden: no PriorityClass with name high-priority was found
```

**Solution:**
- Create the PriorityClass (see Step 1.0):
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

The Helm chart uses `nodeSelector: dedicated: <value>` to schedule pods on specific nodes. The node label must match the nodeSelector value in your values.yaml.

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

- [ ] ArgoCD installed and running in local Kubernetes
- [ ] ArgoCD port-forwarded to localhost:8087
- [ ] ArgoCD admin password retrieved
- [ ] ArgoCD API token generated
- [ ] GitHub App created
- [ ] GitHub App permissions set (Contents: Read and write)
- [ ] GitHub App installed to repository
- [ ] Installation ID noted
- [ ] Private key generated and downloaded
- [ ] Private key placed in `horizon/configs/github.pem`
- [ ] Predator Helm chart copied to repository
- [ ] Repository changes committed and pushed
- [ ] docker-compose.yml updated with all configuration values
- [ ] Services started successfully
- [ ] Horizon logs show no errors
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

## Next Steps

After completing this setup:

1. **Test Onboarding**: Create a deployable through the Horizon API
2. **Monitor ArgoCD**: Watch applications sync automatically in ArgoCD UI
3. **Customize Values**: Modify Helm chart values in GitHub (ArgoCD will auto-sync)
4. **Add More Environments**: Configure additional working environments if needed

For more information, refer to:
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [GitHub Apps Documentation](https://docs.github.com/en/apps)
- [Predator Helm Chart](../predator/1.0.0/README.md)
