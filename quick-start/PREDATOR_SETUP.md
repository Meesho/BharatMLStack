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

### 1.0 Create a Local Kubernetes Cluster

Before installing CRDs and ArgoCD, you need to have a Kubernetes cluster running locally. Choose one of the following options:

#### Option 1: Using kind (Kubernetes in Docker)

**kind** is a tool for running local Kubernetes clusters using Docker container "nodes".

1. **Install kind** (if not already installed):
   ```bash
   # macOS
   brew install kind
   
   # Linux
   curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
   chmod +x ./kind
   sudo mv ./kind /usr/local/bin/kind
   
   # Windows (using Chocolatey)
   choco install kind
   ```

2. **Create a kind cluster:**
   ```bash
   # Create a cluster with a custom name
   kind create cluster --name bharatml-stack
   
   # Or use the default name
   kind create cluster
   ```

3. **Verify the cluster is running:**
   ```bash
   kubectl cluster-info --context kind-bharatml-stack
   # Or for default cluster:
   kubectl cluster-info --context kind-kind
   
   # Check nodes
   kubectl get nodes
   ```

4. **Set kubectl context** (if needed):
   ```bash
   kubectl config use-context kind-bharatml-stack
   # Or for default:
   kubectl config use-context kind-kind
   ```

**Note:** The cluster name will be used in node labels. If you use a custom name like `bharatml-stack`, the node name will be `bharatml-stack-control-plane`.

#### Option 2: Using minikube

**minikube** runs a single-node Kubernetes cluster inside a VM on your local machine.

1. **Install minikube** (if not already installed):
   ```bash
   # macOS
   brew install minikube
   
   # Linux
   curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
   sudo install minikube-linux-amd64 /usr/local/bin/minikube
   
   # Windows
   choco install minikube
   ```

2. **Start minikube:**
   ```bash
   # Start with default settings
   minikube start
   
   # Or with specific Kubernetes version
   minikube start --kubernetes-version=v1.28.0
   
   # Or with more resources
   minikube start --memory=8192 --cpus=4
   ```

3. **Verify the cluster is running:**
   ```bash
   kubectl cluster-info
   kubectl get nodes
   ```

4. **Enable minikube addons** (optional but recommended):
   ```bash
   minikube addons enable ingress
   minikube addons enable metrics-server
   ```

#### Option 3: Using Docker Desktop Kubernetes

Docker Desktop includes a built-in Kubernetes option that can be enabled.

1. **Enable Kubernetes in Docker Desktop:**
   - Open Docker Desktop
   - Go to **Settings** (gear icon)
   - Navigate to **Kubernetes**
   - Check **"Enable Kubernetes"**
   - Click **"Apply & Restart"**

2. **Verify the cluster is running:**
   ```bash
   kubectl cluster-info
   kubectl get nodes
   ```

3. **Set kubectl context** (if needed):
   ```bash
   kubectl config use-context docker-desktop
   ```

**Note:** Docker Desktop Kubernetes typically has access to `/Users` paths on macOS, which can be useful for mounting local files.

#### Verify Your Cluster Setup

After creating your cluster, verify it's working:

```bash
# Check cluster connection
kubectl cluster-info

# Check nodes are ready
kubectl get nodes

# Verify kubectl is configured correctly
kubectl config current-context
```

**Important:** Make sure your `kubectl` is configured to use your local cluster before proceeding to the next steps. The context name will vary:
- kind: `kind-<cluster-name>` (e.g., `kind-bharatml-stack` or `kind-kind`)
- minikube: `minikube`
- Docker Desktop: `docker-desktop`

### 1.1 Label Kubernetes Node for Pod Scheduling

The Predator Helm chart uses `nodeSelector: dedicated: <value>` to schedule pods on specific nodes. To prevent pod scheduling failures, you need to label your Kubernetes node with a `dedicated` label that matches the `nodeSelectorValue` in your Helm values.

**Get your node name and label it:**

```bash
# Get your node name
NODE_NAME=$(kubectl get nodes -o name | head -1 | sed 's|node/||')

# Display the node name (for reference)
echo "Node name: $NODE_NAME"

# Label the node with the dedicated label matching the node name
# This is the default pattern used in the Helm chart
kubectl label node $NODE_NAME dedicated=$NODE_NAME --overwrite

# Verify the label was added
kubectl get nodes --show-labels | grep dedicated
```

**What this does:**
- Labels your node with `dedicated: <node-name>` (e.g., `dedicated: bharatml-stack-control-plane` for a kind cluster)
- The Helm chart's default `nodeSelectorValue` typically matches the node name, so this ensures pods can be scheduled

**Note:** If you later customize the `nodeSelectorValue` in your Helm values.yaml, you'll need to update this label to match. The label format is `dedicated: <your-nodeSelector-value>`.

**Alternative approach:** If you prefer to use a custom value instead of the node name, you can label it with any value:

```bash
# Get your node name
NODE_NAME=$(kubectl get nodes -o name | head -1 | sed 's|node/||')

# Label with a custom value (e.g., "local-dev")
kubectl label node $NODE_NAME dedicated=local-dev --overwrite

# Then make sure your values.yaml uses: nodeSelectorValue: "local-dev"
```

**Verify the label:**
```bash
# Check that the label exists
kubectl get nodes --show-labels | grep dedicated
```

You should see output like: `dedicated=bharatml-stack-control-plane` (or your node name/custom value).

### 1.2 Install Required CRDs and PriorityClass

The Predator Helm chart uses several Custom Resource Definitions (CRDs) and a PriorityClass that must be installed in your Kubernetes cluster before ArgoCD can deploy resources.

#### Install Contour CRDs and IngressClass (Required for HTTPProxy)

The `HTTPProxy` resource in the Predator Helm chart requires Contour CRDs and IngressClass to be installed. ArgoCD can deploy `HTTPProxy` resources, but the CRD must exist in the cluster first.

**Install Full Contour (Required for HTTPProxy to Work)**

You need the full Contour deployment (not just CRDs) for HTTPProxy to be reconciled:

```bash
# Install full Contour deployment (includes CRDs + Contour + Envoy)
kubectl apply -f https://projectcontour.io/quickstart/contour.yaml
```

**Verify installation:**

```bash
# Check that HTTPProxy CRD is installed
kubectl get crd httpproxies.projectcontour.io

# Check that Contour pods are running
kubectl get pods -n projectcontour

# Create IngressClass for contour-internal (required for HTTPProxy reconciliation)
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: contour-internal
spec:
  controller: projectcontour.io/ingress-controller
EOF

# Verify IngressClass was created
kubectl get ingressclass contour-internal
```

**Important: Configure Contour to Watch for Your Ingress Class**

If your HTTPProxy uses a specific ingress class (e.g., `contour-internal`), you must configure Contour to watch for that class. By default, Contour watches all HTTPProxies, but when you specify an ingress class, Contour needs to be explicitly configured.

**Configure Contour for Specific Ingress Class (Only if Needed)**

If you need class-based isolation with `contour-internal`, configure Contour to watch for that class:

```bash
# Check current Contour configuration
kubectl -n projectcontour get deploy contour -o yaml | grep -A 10 "args:"

# If ingress-class-name is not set, add it:
kubectl -n projectcontour patch deploy contour --type='json' -p='[
  {
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--ingress-class-name=contour-internal"
  }
]'

# Restart Contour to apply changes
kubectl -n projectcontour rollout restart deploy contour

# Wait for rollout to complete
kubectl -n projectcontour rollout status deploy contour
```

**⚠️ Note:** 
- Without Contour CRDs, ArgoCD will fail to sync with error: `The Kubernetes API could not find projectcontour.io/HTTPProxy for requested resource...`
- Without the IngressClass, HTTPProxy will show "NotReconciled" status even if Contour is running
- If Contour is not configured to watch your ingress class, HTTPProxy will remain "NotReconciled"

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

### 1.3 Install ArgoCD in Your Local Kubernetes Cluster

```bash
# Create ArgoCD namespace
kubectl create namespace argocd

# Install ArgoCD
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ArgoCD to be ready (this may take a few minutes)
kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n argocd
```

### 1.4 Port Forward ArgoCD Server

```bash
# Port forward ArgoCD server to access UI and API
kubectl port-forward svc/argocd-server -n argocd 8087:443
```

**Note:** Keep this terminal session running. In a new terminal, you can access:
- **ArgoCD UI**: https://localhost:8087 (accept the self-signed certificate warning)
- **ArgoCD API**: http://localhost:8087

### 1.5 Get ArgoCD Admin Password

```bash
# Get the initial admin password
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d && echo
```

**Default username:** `admin`

### 1.6 Enable API Key Permissions

By default, ArgoCD does not allow API key generation. You need to enable this permission for the admin account before you can generate API tokens.

```bash
# Edit the ArgoCD ConfigMap to enable API key permissions
kubectl -n argocd edit configmap argocd-cm
```

In the editor that opens, add the following under the `data` section:

```yaml
data:
  accounts.admin: apiKey, login
```

**Note:** If the `data` section doesn't exist, create it. The `apiKey, login` value enables both API key generation and login capabilities for the admin account.

After saving and closing the editor, restart the ArgoCD server to apply the changes:

```bash
# Restart ArgoCD server to apply the configuration
kubectl -n argocd rollout restart deployment argocd-server

# Wait for the server to be ready
kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n argocd
```

**Verify the change:**
```bash
# Check that the configmap was updated
kubectl -n argocd get configmap argocd-cm -o yaml | grep accounts.admin
```

You should see `accounts.admin: apiKey, login` in the output.

### 1.7 Generate ArgoCD API Token

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

### 1.8 Add GitHub Repository to ArgoCD

Before creating ArgoCD Applications, you must add your GitHub repository to ArgoCD with proper authentication. ArgoCD needs credentials to access your repository to sync applications and Helm charts.

#### Option 1: Using Personal Access Token (Recommended)

1. **Create a GitHub Personal Access Token:**
   - Go to https://github.com/settings/tokens
   - Click **Generate new token** → **Generate new token (classic)**
   - Give it a name (e.g., `argocd-repo-access`)
   - Select scopes: **`repo`** (Full control of private repositories)
   - Click **Generate token**
   - **Copy the token immediately** (you won't be able to see it again)

2. **Add the repository to ArgoCD using CLI:**
   ```bash
   # Login to ArgoCD (if not already logged in)
   argocd login localhost:8087 --insecure
   
   # Add the repository
   argocd repo add https://github.com/<YOUR_USERNAME>/<YOUR_REPO_NAME>.git \
     --name <YOUR_REPO_NAME> \
     --type git \
     --username <YOUR_GITHUB_USERNAME> \
     --password <YOUR_PERSONAL_ACCESS_TOKEN>
   ```

   **Replace:**
   - `<YOUR_USERNAME>` with your GitHub username or organization
   - `<YOUR_REPO_NAME>` with your repository name (e.g., `onboarding-test`)
   - `<YOUR_GITHUB_USERNAME>` with your GitHub username
   - `<YOUR_PERSONAL_ACCESS_TOKEN>` with the token you just created

3. **Verify the repository was added:**
   ```bash
   # List repositories
   argocd repo list
   
   # You should see your repository in the list
   ```

#### Option 2: Using SSH Key

1. **Generate an SSH key (if you don't have one):**
   ```bash
   # Generate a new SSH key for ArgoCD
   ssh-keygen -t ed25519 -C "argocd@yourdomain.com" -f ~/.ssh/argocd_key
   
   # Don't set a passphrase (or ArgoCD won't be able to use it automatically)
   ```

2. **Add the SSH public key to GitHub:**
   - Copy your public key:
     ```bash
     cat ~/.ssh/argocd_key.pub
     ```
   - Go to https://github.com/settings/keys
   - Click **New SSH key**
   - Paste the public key and save

3. **Add the repository to ArgoCD using SSH:**
   ```bash
   # Login to ArgoCD (if not already logged in)
   argocd login localhost:8087 --insecure
   
   # Add the repository using SSH URL
   argocd repo add git@github.com:<YOUR_USERNAME>/<YOUR_REPO_NAME>.git \
     --name <YOUR_REPO_NAME> \
     --type git \
     --ssh-private-key-path ~/.ssh/argocd_key
   ```

   **Replace:**
   - `<YOUR_USERNAME>` with your GitHub username or organization
   - `<YOUR_REPO_NAME>` with your repository name

4. **Verify the repository was added:**
   ```bash
   # List repositories
   argocd repo list
   ```

#### Option 3: Using ArgoCD UI

1. **Log in to ArgoCD UI** at https://localhost:8087

2. **Navigate to Settings → Repositories**

3. **Click "Connect Repo"**

4. **Fill in the repository details:**
   - **Type**: Git
   - **Project**: default
   - **Repository URL**: `https://github.com/<YOUR_USERNAME>/<YOUR_REPO_NAME>.git`
   - **Username**: Your GitHub username
   - **Password**: Your Personal Access Token (if using HTTPS)
   - Or select **SSH** and provide your SSH private key (if using SSH)

5. **Click "Connect"**

6. **Verify connection** - The repository should appear in the list with a green checkmark

#### Verify Repository Access

After adding the repository, verify ArgoCD can access it:

```bash
# Test repository connection
argocd repo get https://github.com/<YOUR_USERNAME>/<YOUR_REPO_NAME>.git

# Or using the repository name
argocd repo get <YOUR_REPO_NAME>
```

You should see repository details without errors. If there are authentication issues, check:
- Token has `repo` scope (for PAT)
- SSH key is added to GitHub
- Repository URL is correct
- Repository is accessible (not private without proper access)

**Note:** This repository configuration is required before creating any ArgoCD Applications that reference this repository. Without it, ArgoCD will fail to sync applications with errors like "repository not found" or "authentication failed".

### 1.9 Set Up Automated ArgoCD Application Onboarding

To enable automatic ArgoCD application creation when you onboard new deployables, set up an ArgoCD Application that watches your GitHub repository's `prd/applications` directory.

#### 1.9.1 Update the prd-applications.yaml File

1. **Navigate to the predator folder:**
   ```bash
   cd /Users/adityakumargarg/Desktop/projects/OSS/BharatMLStack/predator
   ```

2. **Edit `prd-applications.yaml` and update the repository URL:**
   ```yaml
   apiVersion: argoproj.io/v1alpha1
   kind: Application
   metadata:
     name: prd-applications
     namespace: argocd
   spec:
     project: default
     source:
       repoURL: https://github.com/<YOUR_USERNAME>/<YOUR_REPO_NAME>.git  # Update this
       targetRevision: main  # Update if using a different branch
       path: prd/applications  # This directory will be watched for new applications
     destination:
       server: https://kubernetes.default.svc
       namespace: argocd
     syncPolicy:
       automated:
         prune: true
         selfHeal: true
   ```

   **Replace:**
   - `<YOUR_USERNAME>` with your GitHub username or organization
   - `<YOUR_REPO_NAME>` with your repository name (e.g., `onboarding-test`)
   - `main` with your default branch if different

#### 1.9.2 Apply the Application

```bash
# From the predator directory
kubectl apply -f prd-applications.yaml
```

**What this does:**
- Creates an ArgoCD Application named `prd-applications` that watches the `prd/applications` directory in your GitHub repo
- When Horizon creates new application YAML files in `prd/applications/` (e.g., `prd/applications/test.yaml`), ArgoCD automatically detects and creates the corresponding ArgoCD Application
- All applications are automatically synced with `prune` and `selfHeal` enabled

#### 1.9.3 Verify the Setup

```bash
# Check that the prd-applications Application is created
kubectl get application prd-applications -n argocd

# Check its status
kubectl get application prd-applications -n argocd -o yaml | grep -A 10 status
```

**Note:** After onboarding a new deployable through Horizon, the application YAML will be created in your GitHub repo at `prd/applications/{appName}.yaml`, and ArgoCD will automatically create the corresponding ArgoCD Application. No manual steps required!

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
- If `ARGOCD_HELMCHART_PATH=predator/1.0.0`, the chart should be at `predator/1.0.0/` in your repo

### 6.3 Commit and Push

```bash
# Add the chart
git add 1.0.0

# Commit
git commit -m "Add Predator Helm chart 1.0.0"

# Push to main branch (or your default branch)
git push origin main
```

**Verify:** Check that `1.0.0/` exists in your repository at the root level (or `predator/1.0.0/` if using that path).

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
1. **Install Contour CRDs** (see Step 1.0):
   ```bash
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
- Install KEDA CRDs (see Step 1.2):
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

- [ ] Kubernetes cluster created and running
- [ ] Node labeled with `dedicated` label (Step 1.1)
- [ ] Required CRDs and PriorityClass installed
- [ ] ArgoCD installed and running in local Kubernetes
- [ ] ArgoCD port-forwarded to localhost:8087
- [ ] ArgoCD admin password retrieved
- [ ] API key permissions enabled for admin account
- [ ] ArgoCD API token generated
- [ ] GitHub repository added to ArgoCD with authentication
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

## Step 10: Access Predator Service Through Contour HTTPProxy

Once the HTTPProxy is created and Contour is running, you can access your Predator service through Contour's Envoy proxy.

### 10.1 Check HTTPProxy Status

```bash
# Check HTTPProxy status
kubectl get httpproxy -n prd-predator-test

# Get HTTPProxy details including FQDN
kubectl get httpproxy -n prd-predator-test -o yaml | grep -A 5 "virtualhost:"
```

The HTTPProxy should show `Valid` status once Contour controller reconciles it.

### 10.2 Access via Port Forward (Local Development)

For local development, the easiest way is to port-forward Contour's Envoy service:

```bash
# Port forward Envoy service (Contour's ingress proxy)
kubectl port-forward -n projectcontour svc/envoy 8080:80

# In another terminal, access your service using the FQDN as Host header
# Replace <fqdn> with your actual FQDN (e.g., predator-test.prd.meesho.int)
curl -H "Host: <fqdn>" http://localhost:8080/

# Example for predator-test:
curl -H "Host: predator-test.prd.meesho.int" http://localhost:8080/
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

⚠️ **Important:** kind does NOT automatically expose NodePort on localhost. You have two options:

**Option 1: Use Port-Forward (Recommended for kind)**
```bash
# This is the simplest and most reliable method for kind
kubectl port-forward -n projectcontour svc/envoy 8080:80
# Then in another terminal:
curl -H "Host: $FQDN" http://localhost:8080/self/health
```

**Option 2: Configure kind with Port Mapping (Advanced)**

If you want to use NodePort directly, you need to configure port mapping when creating the kind cluster:

```bash
# Create kind cluster with port mapping (example)
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 32558  # Your NodePort
    hostPort: 32558
    protocol: TCP
EOF

# Then access via localhost
curl -H "Host: $FQDN" http://localhost:$NODEPORT/self/health
```

**Note:** For existing kind clusters, port-forward (Option 1) is much simpler than recreating the cluster with port mapping.

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
