#!/bin/bash

# Predator Kubernetes Setup Script
# This script automates the basic Kubernetes setup for Predator local development
# Based on PREDATOR_SETUP.md Step 1 (Set Up Local ArgoCD)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
ARGOCD_NAMESPACE="argocd"
ARGOCD_PORT="8087"
CONTOUR_NAMESPACE="projectcontour"
KEDA_VERSION="v2.12.0"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        log_info "Install kubectl: https://kubernetes.io/docs/tasks/tools/"
        exit 1
    fi
    
    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_warning "Cannot connect to Kubernetes cluster."
        log_info "No active cluster detected. Let's create one..."
        create_cluster
    else
        log_success "Connected to existing Kubernetes cluster"
    fi
    
    log_success "Prerequisites check passed"
}

detect_cluster_tools() {
    local available_tools=()
    
    # Check for kind
    if command -v kind &> /dev/null 2>&1; then
        available_tools+=("kind")
    fi
    
    # Check for minikube
    if command -v minikube &> /dev/null 2>&1; then
        available_tools+=("minikube")
    fi
    
    # Check for Docker Desktop Kubernetes
    if command -v docker &> /dev/null 2>&1; then
        if docker info &> /dev/null 2>&1; then
            # Check if Docker Desktop is running (macOS or Linux)
            if [[ "$OSTYPE" == "darwin"* ]] || [[ "$OSTYPE" == "linux-gnu"* ]]; then
                available_tools+=("docker-desktop")
            fi
        fi
    fi
    
    echo "${available_tools[@]}"
}

create_cluster_kind() {
    log_info "Creating Kubernetes cluster using kind..."
    
    local cluster_name="bharatml-stack"
    
    # Check if cluster already exists
    if kind get clusters 2>/dev/null | grep -q "^${cluster_name}$"; then
        log_warning "kind cluster '${cluster_name}' already exists."
        read -p "Do you want to delete and recreate it? (y/n): " recreate
        if [[ "$recreate" =~ ^[Yy]$ ]]; then
            log_info "Deleting existing cluster..."
            kind delete cluster --name "$cluster_name" || true
        else
            log_info "Using existing cluster..."
            kubectl config use-context "kind-${cluster_name}" 2>/dev/null || true
            return
        fi
    fi
    
    # Create cluster
    log_info "Creating kind cluster '${cluster_name}' (this may take a few minutes)..."
    if ! kind create cluster --name "$cluster_name"; then
        log_error "Failed to create kind cluster"
        exit 1
    fi
    
    # Set kubectl context
    kubectl config use-context "kind-${cluster_name}" || {
        log_error "Failed to set kubectl context"
        exit 1
    }
    
    # Wait for cluster to be ready
    log_info "Waiting for cluster to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=300s || {
        log_warning "Cluster nodes may not be fully ready, but continuing..."
    }
    
    log_success "kind cluster created successfully"
}

create_cluster_minikube() {
    log_info "Creating Kubernetes cluster using minikube..."
    
    # Check if minikube is already running
    if minikube status &> /dev/null 2>&1; then
        log_warning "minikube cluster is already running."
        read -p "Do you want to delete and recreate it? (y/n): " recreate
        if [[ "$recreate" =~ ^[Yy]$ ]]; then
            log_info "Stopping and deleting existing cluster..."
            minikube delete || true
        else
            log_info "Using existing minikube cluster..."
            return
        fi
    fi
    
    # Start minikube
    log_info "Starting minikube cluster (this may take a few minutes)..."
    if ! minikube start; then
        log_error "Failed to start minikube cluster"
        exit 1
    fi
    
    # Enable recommended addons
    log_info "Enabling minikube addons..."
    minikube addons enable ingress 2>/dev/null || true
    minikube addons enable metrics-server 2>/dev/null || true
    
    log_success "minikube cluster created successfully"
}

create_cluster_docker_desktop() {
    log_info "Setting up Docker Desktop Kubernetes..."
    
    log_warning "Docker Desktop Kubernetes must be enabled manually."
    echo ""
    log_info "To enable Kubernetes in Docker Desktop:"
    echo "  1. Open Docker Desktop"
    echo "  2. Go to Settings (gear icon)"
    echo "  3. Navigate to Kubernetes"
    echo "  4. Check 'Enable Kubernetes'"
    echo "  5. Click 'Apply & Restart'"
    echo ""
    
    read -p "Have you enabled Kubernetes in Docker Desktop? (y/n): " ENABLED
    if [[ ! "$ENABLED" =~ ^[Yy]$ ]]; then
        log_error "Please enable Kubernetes in Docker Desktop and run this script again."
        exit 1
    fi
    
    # Set kubectl context
    kubectl config use-context docker-desktop || {
        log_error "Failed to set docker-desktop context. Please check Docker Desktop Kubernetes is enabled."
        exit 1
    }
    
    # Wait for cluster to be ready
    log_info "Waiting for cluster to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=300s || true
    
    log_success "Docker Desktop Kubernetes is ready"
}

create_cluster() {
    log_info "Detecting available cluster creation tools..."
    
    local available_tools_str
    available_tools_str=$(detect_cluster_tools)
    
    if [ -z "$available_tools_str" ]; then
        log_error "No Kubernetes cluster tools found."
        echo ""
        log_info "Please install one of the following:"
        echo "  - kind:     brew install kind (macOS) or see https://kind.sigs.k8s.io/"
        echo "  - minikube: brew install minikube (macOS) or see https://minikube.sigs.k8s.io/"
        echo "  - Docker Desktop: Enable Kubernetes in Docker Desktop settings"
        echo ""
        exit 1
    fi
    
    # Convert string to array
    local available_tools=()
    IFS=' ' read -r -a available_tools <<< "$available_tools_str"
    
    echo ""
    log_info "Available cluster tools: ${available_tools[*]}"
    echo ""
    
    local tool
    if [ ${#available_tools[@]} -eq 1 ]; then
        tool="${available_tools[0]}"
        log_info "Using the only available tool: $tool"
    else
        echo "Select a tool to create the cluster:"
        local i
        for i in "${!available_tools[@]}"; do
            echo "  $((i+1)). ${available_tools[$i]}"
        done
        echo ""
        read -p "Enter your choice (1-${#available_tools[@]}): " choice
        
        if ! [[ "$choice" =~ ^[0-9]+$ ]] || [ "$choice" -lt 1 ] || [ "$choice" -gt ${#available_tools[@]} ]; then
            log_error "Invalid choice"
            exit 1
        fi
        
        tool="${available_tools[$((choice-1))]}"
    fi
    
    echo ""
    case "$tool" in
        kind)
            create_cluster_kind
            ;;
        minikube)
            create_cluster_minikube
            ;;
        docker-desktop)
            create_cluster_docker_desktop
            ;;
        *)
            log_error "Unknown tool: $tool"
            exit 1
            ;;
    esac
    
    # Verify cluster is accessible
    if ! kubectl cluster-info &> /dev/null 2>&1; then
        log_error "Failed to connect to cluster after creation"
        exit 1
    fi
    
    log_success "Cluster is ready and accessible"
}

label_node() {
    log_info "Labeling Kubernetes node for pod scheduling..."
    
    # Get the first node name
    NODE_NAME=$(kubectl get nodes -o name | head -1 | sed 's|node/||')
    
    if [ -z "$NODE_NAME" ]; then
        log_error "No nodes found in the cluster"
        exit 1
    fi
    
    log_info "Node name: $NODE_NAME"
    
    # Label the node with dedicated label matching the node name
    kubectl label node "$NODE_NAME" dedicated="$NODE_NAME" --overwrite
    
    # Verify the label
    if kubectl get nodes --show-labels | grep -q "dedicated=$NODE_NAME"; then
        log_success "Node labeled successfully: dedicated=$NODE_NAME"
    else
        log_warning "Node label verification failed, but continuing..."
    fi
}

install_contour() {
    log_info "Installing Contour (includes CRDs, Contour, and Envoy)..."
    
    # Check if Contour is already installed
    if kubectl get namespace "$CONTOUR_NAMESPACE" &> /dev/null; then
        log_warning "Contour namespace already exists. Skipping Contour installation."
        return
    fi
    
    # Install full Contour deployment
    kubectl apply -f https://projectcontour.io/quickstart/contour.yaml
    
    # Wait for Contour pods to be ready
    log_info "Waiting for Contour pods to be ready..."
    kubectl wait --for=condition=ready pod -l app=contour -n "$CONTOUR_NAMESPACE" --timeout=120s || true
    kubectl wait --for=condition=ready pod -l app=envoy -n "$CONTOUR_NAMESPACE" --timeout=120s || true
    
    # Verify HTTPProxy CRD
    if kubectl get crd httpproxies.projectcontour.io &> /dev/null; then
        log_success "Contour installed successfully"
    else
        log_error "Contour installation may have failed. HTTPProxy CRD not found."
        exit 1
    fi
}

create_ingress_class() {
    log_info "Creating IngressClass for contour-internal..."
    
    # Check if IngressClass already exists
    if kubectl get ingressclass contour-internal &> /dev/null; then
        log_warning "IngressClass 'contour-internal' already exists. Skipping..."
        return
    fi
    
    # Create IngressClass
    kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: contour-internal
spec:
  controller: projectcontour.io/ingress-controller
EOF
    
    if kubectl get ingressclass contour-internal &> /dev/null; then
        log_success "IngressClass 'contour-internal' created successfully"
    else
        log_error "Failed to create IngressClass"
        exit 1
    fi
}

configure_contour_ingress_class() {
    log_info "Configuring Contour to watch for contour-internal ingress class..."
    
    # Check if Contour is already configured
    if kubectl -n "$CONTOUR_NAMESPACE" get deploy contour -o jsonpath='{.spec.template.spec.containers[0].args}' 2>/dev/null | grep -q "ingress-class-name"; then
        log_warning "Contour is already configured for ingress class. Skipping..."
        return
    fi
    
    # Add ingress-class-name argument to Contour deployment
    kubectl -n "$CONTOUR_NAMESPACE" patch deploy contour --type='json' -p='[
      {
        "op": "add",
        "path": "/spec/template/spec/containers/0/args/-",
        "value": "--ingress-class-name=contour-internal"
      }
    ]' || {
        log_warning "Failed to patch Contour deployment. It may already be configured."
    }
    
    # Restart Contour to apply changes
    log_info "Restarting Contour deployment..."
    kubectl -n "$CONTOUR_NAMESPACE" rollout restart deploy contour
    kubectl -n "$CONTOUR_NAMESPACE" rollout status deploy contour --timeout=120s || true
    
    log_success "Contour configured to watch for contour-internal ingress class"
}

install_flagger_crds() {
    log_info "Installing Flagger CRDs..."
    
    # Check if Flagger CRDs are already installed
    if kubectl get crd alertproviders.flagger.app &> /dev/null; then
        log_warning "Flagger CRDs already installed. Skipping..."
        return
    fi
    
    # Install Flagger CRDs
    kubectl apply -f https://raw.githubusercontent.com/fluxcd/flagger/main/artifacts/flagger/crd.yaml
    
    # Verify installation
    if kubectl get crd | grep -q flagger; then
        log_success "Flagger CRDs installed successfully"
    else
        log_error "Flagger CRDs installation may have failed"
        exit 1
    fi
}

install_keda_crds() {
    log_info "Installing KEDA CRDs..."
    
    # Check if KEDA CRDs are already installed
    if kubectl get crd scaledobjects.keda.sh &> /dev/null; then
        log_warning "KEDA CRDs already installed. Skipping..."
        return
    fi
    
    # Install only ScaledObject CRD (required for Predator)
    # Note: We skip ScaledJob CRD as it has oversized annotations that exceed Kubernetes limits
    # and is not required for Predator setup
    log_info "Installing ScaledObject CRD (required for autoscaling)..."
    if ! kubectl apply -f https://raw.githubusercontent.com/kedacore/keda/${KEDA_VERSION}/config/crd/bases/keda.sh_scaledobjects.yaml; then
        log_error "Failed to install ScaledObject CRD"
        exit 1
    fi
    
    # Verify installation
    if kubectl get crd scaledobjects.keda.sh &> /dev/null; then
        log_success "KEDA ScaledObject CRD installed successfully"
    else
        log_error "KEDA ScaledObject CRD installation verification failed"
        exit 1
    fi
}

install_priority_class() {
    log_info "Installing PriorityClass 'high-priority'..."
    
    # Check if PriorityClass already exists
    if kubectl get priorityclass high-priority &> /dev/null; then
        log_warning "PriorityClass 'high-priority' already exists. Skipping..."
        return
    fi
    
    # Create PriorityClass
    kubectl apply -f - <<EOF
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: high-priority
value: 1000
globalDefault: false
description: "High priority class for application pods"
EOF
    
    if kubectl get priorityclass high-priority &> /dev/null; then
        log_success "PriorityClass 'high-priority' created successfully"
    else
        log_error "Failed to create PriorityClass"
        exit 1
    fi
}

install_argocd() {
    log_info "Installing ArgoCD..."
    
    # Check if ArgoCD namespace already exists
    if kubectl get namespace "$ARGOCD_NAMESPACE" &> /dev/null; then
        log_warning "ArgoCD namespace already exists. Checking if ArgoCD is installed..."
        
        if kubectl get deployment argocd-server -n "$ARGOCD_NAMESPACE" &> /dev/null; then
            log_warning "ArgoCD appears to be already installed. Skipping installation."
            return
        fi
    else
        # Create ArgoCD namespace
        kubectl create namespace "$ARGOCD_NAMESPACE"
    fi
    
    # Install ArgoCD
    kubectl apply -n "$ARGOCD_NAMESPACE" -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
    
    # Wait for ArgoCD to be ready
    log_info "Waiting for ArgoCD server to be ready (this may take a few minutes)..."
    kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n "$ARGOCD_NAMESPACE" || {
        log_warning "ArgoCD server may not be fully ready yet. Continuing..."
    }
    
    log_success "ArgoCD installed successfully"
}

enable_argocd_api_key() {
    log_info "Enabling API key permissions for ArgoCD admin account..."
    
    # Check if API key is already enabled
    if kubectl -n "$ARGOCD_NAMESPACE" get configmap argocd-cm -o jsonpath='{.data.accounts\.admin}' 2>/dev/null | grep -q "apiKey"; then
        log_warning "API key permissions already enabled. Skipping..."
        return
    fi
    
    # Get current config
    CURRENT_CONFIG=$(kubectl -n "$ARGOCD_NAMESPACE" get configmap argocd-cm -o jsonpath='{.data.accounts\.admin}' 2>/dev/null || echo "")
    
    if [ -z "$CURRENT_CONFIG" ]; then
        # Add accounts.admin if it doesn't exist
        kubectl -n "$ARGOCD_NAMESPACE" patch configmap argocd-cm --type='json' -p='[
          {
            "op": "add",
            "path": "/data/accounts.admin",
            "value": "apiKey, login"
          }
        ]'
    else
        # Update existing config to include apiKey
        if [[ ! "$CURRENT_CONFIG" =~ "apiKey" ]]; then
            NEW_CONFIG="${CURRENT_CONFIG}, apiKey"
            kubectl -n "$ARGOCD_NAMESPACE" patch configmap argocd-cm --type='json' -p="[
              {
                \"op\": \"replace\",
                \"path\": \"/data/accounts.admin\",
                \"value\": \"${NEW_CONFIG}\"
              }
            ]"
        fi
    fi
    
    # Restart ArgoCD server to apply changes
    log_info "Restarting ArgoCD server to apply configuration..."
    kubectl -n "$ARGOCD_NAMESPACE" rollout restart deployment argocd-server
    kubectl -n "$ARGOCD_NAMESPACE" rollout status deployment argocd-server --timeout=120s || true
    
    log_success "API key permissions enabled for ArgoCD admin account"
}

get_argocd_password() {
    log_info "Retrieving ArgoCD admin password..."
    
    # Wait for secret to be available
    for i in {1..30}; do
        if kubectl -n "$ARGOCD_NAMESPACE" get secret argocd-initial-admin-secret &> /dev/null; then
            break
        fi
        log_info "Waiting for ArgoCD secret to be created... ($i/30)"
        sleep 2
    done
    
    if kubectl -n "$ARGOCD_NAMESPACE" get secret argocd-initial-admin-secret &> /dev/null; then
        PASSWORD=$(kubectl -n "$ARGOCD_NAMESPACE" get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d 2>/dev/null || base64 -D 2>/dev/null)
        echo ""
        log_success "ArgoCD Admin Credentials:"
        echo -e "  ${GREEN}Username:${NC} admin"
        echo -e "  ${GREEN}Password:${NC} $PASSWORD"
        echo ""
        # Store password for later use
        export ARGOCD_PASSWORD="$PASSWORD"
    else
        log_warning "ArgoCD initial admin secret not found yet. You may need to wait a bit longer."
        log_info "To get the password later, run:"
        echo "  kubectl -n $ARGOCD_NAMESPACE get secret argocd-initial-admin-secret -o jsonpath=\"{.data.password}\" | base64 -d && echo"
    fi
}

check_argocd_cli() {
    if command -v argocd &> /dev/null; then
        return 0
    fi
    
    log_warning "ArgoCD CLI is not installed."
    log_info "Attempting to install ArgoCD CLI automatically..."
    
    # Detect OS
    local os_type
    os_type=$(uname -s)
    
    case "$os_type" in
        Darwin)
            # macOS - use Homebrew
            if command -v brew &> /dev/null; then
                log_info "Installing ArgoCD CLI using Homebrew..."
                if brew install argocd; then
                    # Verify installation
                    if command -v argocd &> /dev/null; then
                        log_success "ArgoCD CLI installed successfully"
                        return 0
                    else
                        log_warning "ArgoCD CLI installed but not found in PATH. You may need to restart your terminal."
                        # Try to find it
                        local brew_prefix
                        brew_prefix=$(brew --prefix)
                        if [ -f "${brew_prefix}/bin/argocd" ]; then
                            export PATH="${brew_prefix}/bin:$PATH"
                            log_success "ArgoCD CLI found and added to PATH"
                            return 0
                        fi
                    fi
                else
                    log_error "Failed to install ArgoCD CLI using Homebrew"
                fi
            else
                log_error "Homebrew is not installed. Please install Homebrew first:"
                echo "  /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
            fi
            ;;
        Linux)
            # Linux - download binary
            log_info "Installing ArgoCD CLI from GitHub releases..."
            local arch
            arch=$(uname -m)
            case "$arch" in
                x86_64) arch="amd64" ;;
                aarch64|arm64) arch="arm64" ;;
                *) arch="amd64" ;;
            esac
            
            local version="v2.10.0"
            local download_url="https://github.com/argoproj/argo-cd/releases/download/${version}/argocd-linux-${arch}"
            
            # Try user-local installation first (no sudo required)
            local install_path="$HOME/.local/bin/argocd"
            local install_dir="$HOME/.local/bin"
            
            # Create directory if it doesn't exist
            mkdir -p "$install_dir"
            
            log_info "Downloading ArgoCD CLI ${version} for Linux ${arch}..."
            if curl -L -o "$install_path" "$download_url" 2>/dev/null; then
                chmod +x "$install_path"
                # Add to PATH if not already there
                if [[ ":$PATH:" != *":$install_dir:"* ]]; then
                    export PATH="$install_dir:$PATH"
                    log_info "Added $install_dir to PATH for this session"
                    log_info "To make it permanent, add to your ~/.bashrc or ~/.zshrc:"
                    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
                fi
                # Verify installation
                if [ -f "$install_path" ] && [ -x "$install_path" ]; then
                    # Verify it works
                    if "$install_path" version --client &> /dev/null; then
                        log_success "ArgoCD CLI installed and verified successfully"
                        return 0
                    else
                        log_warning "ArgoCD CLI installed but verification failed. Continuing anyway..."
                        return 0
                    fi
                else
                    log_error "ArgoCD CLI installation verification failed"
                fi
            else
                log_error "Failed to download ArgoCD CLI"
                # Try system-wide installation as fallback (requires sudo)
                log_info "Attempting system-wide installation (may require sudo)..."
                if sudo curl -L -o "/usr/local/bin/argocd" "$download_url" 2>/dev/null; then
                    sudo chmod +x "/usr/local/bin/argocd"
                    if command -v argocd &> /dev/null && argocd version --client &> /dev/null; then
                        log_success "ArgoCD CLI installed and verified successfully"
                        return 0
                    else
                        log_warning "ArgoCD CLI installed but verification failed. Continuing anyway..."
                        return 0
                    fi
                else
                    log_error "Failed to install ArgoCD CLI system-wide"
                fi
            fi
            ;;
        *)
            log_error "Unsupported OS: $os_type"
            log_info "Please install ArgoCD CLI manually:"
            echo "  See: https://argo-cd.readthedocs.io/en/stable/cli_installation/"
            ;;
    esac
    
    # If installation failed, ask user if they want to continue
    echo ""
    read -p "ArgoCD CLI installation failed. Do you want to continue without it? (y/n): " continue_without_cli
    if [[ "$continue_without_cli" =~ ^[Yy]$ ]]; then
        log_warning "Continuing without ArgoCD CLI. Repository authentication will be skipped."
        return 1
    else
        log_info "Please install ArgoCD CLI manually and run this script again."
        exit 0
    fi
}

wait_for_argocd_server() {
    log_info "Waiting for ArgoCD server to be ready..."
    
    # Wait for ArgoCD server deployment to exist
    for i in {1..30}; do
        if kubectl -n "$ARGOCD_NAMESPACE" get deployment argocd-server &> /dev/null; then
            break
        fi
        sleep 2
    done
    
    # Wait for deployment to be available
    if kubectl -n "$ARGOCD_NAMESPACE" wait --for=condition=available --timeout=300s deployment/argocd-server &> /dev/null; then
        log_success "ArgoCD server deployment is available"
    else
        log_warning "ArgoCD server deployment may not be fully available"
    fi
    
    # Wait for pods to be ready
    log_info "Waiting for ArgoCD server pods to be ready..."
    if kubectl -n "$ARGOCD_NAMESPACE" wait --for=condition=ready pod -l app.kubernetes.io/name=argocd-server --timeout=300s &> /dev/null; then
        log_success "ArgoCD server pods are ready"
        # Give it a few more seconds to fully initialize
        log_info "Waiting additional 5 seconds for ArgoCD server to fully initialize..."
        sleep 5
        return 0
    else
        log_warning "ArgoCD server pods may not be fully ready, but continuing..."
        return 0
    fi
}

verify_port_forward() {
    log_info "Verifying port-forward connection..."
    
    # Try to connect to ArgoCD server via port-forward
    for i in {1..10}; do
        if curl -k -s -o /dev/null -w "%{http_code}" "https://localhost:$ARGOCD_PORT" 2>/dev/null | grep -q "200\|401\|403"; then
            log_success "Port-forward is working"
            return 0
        fi
        if [ $i -lt 10 ]; then
            sleep 1
        fi
    done
    
    log_warning "Port-forward connection verification failed"
    return 1
}

setup_port_forward() {
    log_info "Setting up port forwarding for ArgoCD server..."
    
    # Check if port is already in use
    if lsof -Pi :$ARGOCD_PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
        log_warning "Port $ARGOCD_PORT is already in use. Verifying connection..."
        if verify_port_forward; then
            log_success "Existing port-forward is working"
            return 0
        else
            log_warning "Existing port-forward may not be working. Attempting to create a new one..."
            # Kill existing process on the port (if it's a kubectl port-forward)
            local existing_pid
            existing_pid=$(lsof -Pi :$ARGOCD_PORT -sTCP:LISTEN -t 2>/dev/null | head -1)
            if [ -n "$existing_pid" ]; then
                log_info "Stopping existing process on port $ARGOCD_PORT (PID: $existing_pid)..."
                kill "$existing_pid" 2>/dev/null || true
                sleep 2
            fi
        fi
    fi
    
    # Start port-forward in background
    log_info "Starting port-forward on port $ARGOCD_PORT (running in background)..."
    kubectl port-forward svc/argocd-server -n "$ARGOCD_NAMESPACE" $ARGOCD_PORT:443 > /dev/null 2>&1 &
    local pf_pid=$!
    
    # Wait a moment for port-forward to establish
    sleep 3
    
    # Check if port-forward is still running
    if ! kill -0 $pf_pid 2>/dev/null; then
        log_error "Port-forward failed to start"
        return 1
    fi
    
    # Verify the connection works
    if verify_port_forward; then
        log_success "Port-forward established and verified on port $ARGOCD_PORT"
        log_info "Note: Port-forward will run in background. You may need to restart it manually if it stops."
        return 0
    else
        log_error "Port-forward started but connection verification failed"
        kill $pf_pid 2>/dev/null || true
        return 1
    fi
}

add_github_repo_to_argocd() {
    log_info "Adding GitHub repository to ArgoCD..."
    
    # Check if ArgoCD CLI is available
    if ! check_argocd_cli; then
        log_warning "Skipping repository authentication setup (ArgoCD CLI not available)"
        log_info "You can add the repository manually later (see PREDATOR_SETUP.md Step 1.8)"
        return 1
    fi
    
    # Wait for ArgoCD server
    wait_for_argocd_server
    
    # Set up port forwarding
    if ! setup_port_forward; then
        log_warning "Failed to set up port-forward. You may need to set it up manually."
        log_info "Run: kubectl port-forward svc/argocd-server -n $ARGOCD_NAMESPACE $ARGOCD_PORT:443"
        return 1
    fi
    
    # Get ArgoCD password if not already set
    if [ -z "${ARGOCD_PASSWORD:-}" ]; then
        if kubectl -n "$ARGOCD_NAMESPACE" get secret argocd-initial-admin-secret &> /dev/null; then
            ARGOCD_PASSWORD=$(kubectl -n "$ARGOCD_NAMESPACE" get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d 2>/dev/null || base64 -D 2>/dev/null)
        else
            log_error "Cannot retrieve ArgoCD password"
            return 1
        fi
    fi
    
    # Verify ArgoCD server is accessible before attempting login
    log_info "Verifying ArgoCD server is accessible..."
    if ! curl -k -s -o /dev/null "https://localhost:$ARGOCD_PORT" 2>/dev/null; then
        log_error "Cannot connect to ArgoCD server on port $ARGOCD_PORT"
        log_info "Please ensure port-forward is running:"
        echo "  kubectl port-forward svc/argocd-server -n $ARGOCD_NAMESPACE $ARGOCD_PORT:443"
        return 1
    fi
    
    # Wait a bit more for ArgoCD to be fully ready (sometimes needs extra time after port-forward)
    log_info "Waiting for ArgoCD server to be fully ready..."
    sleep 5
    
    # Try to get session token via API first (alternative method)
    log_info "Attempting to authenticate with ArgoCD API..."
    local api_token=""
    for i in {1..5}; do
        api_token=$(curl -k -s -X POST "https://localhost:$ARGOCD_PORT/api/v1/session" \
            -H "Content-Type: application/json" \
            -d "{\"username\":\"admin\",\"password\":\"$ARGOCD_PASSWORD\"}" 2>/dev/null | grep -o '"token":"[^"]*' | cut -d'"' -f4)
        
        if [ -n "$api_token" ]; then
            log_success "Successfully authenticated via API"
            # Store token for later use if needed
            export ARGOCD_API_TOKEN="$api_token"
            break
        fi
        if [ $i -lt 5 ]; then
            sleep 3
        fi
    done
    
    # Login to ArgoCD CLI with retry logic and better error handling
    log_info "Logging in to ArgoCD CLI (this may take a few attempts)..."
    local login_success=false
    local last_error=""
    
    # Check ArgoCD CLI version to determine which method to use
    local argocd_version
    argocd_version=$(argocd version --client --short 2>/dev/null | head -1 || echo "unknown")
    log_info "ArgoCD CLI version: $argocd_version"
    
    for i in {1..10}; do
        local error_output
        local exit_code
        
        # Try different login methods based on attempt number
        if [ $i -le 3 ]; then
            # First 3 attempts: try with --password flag (works with older versions)
            error_output=$(argocd login localhost:$ARGOCD_PORT --username admin --password "$ARGOCD_PASSWORD" --insecure 2>&1)
            exit_code=$?
        elif [ $i -le 6 ]; then
            # Next 3 attempts: try with --password-stdin (newer versions)
            error_output=$(echo "$ARGOCD_PASSWORD" | argocd login localhost:$ARGOCD_PORT --username admin --password-stdin --insecure 2>&1)
            exit_code=$?
        elif [ $i -le 8 ]; then
            # Next 2 attempts: try with --grpc-web and --password
            error_output=$(argocd login localhost:$ARGOCD_PORT --username admin --password "$ARGOCD_PASSWORD" --insecure --grpc-web 2>&1)
            exit_code=$?
        else
            # Last 2 attempts: try with environment variable
            export ARGOCD_PASSWORD_ENV="$ARGOCD_PASSWORD"
            error_output=$(argocd login localhost:$ARGOCD_PORT --username admin --password "$ARGOCD_PASSWORD_ENV" --insecure 2>&1)
            exit_code=$?
            unset ARGOCD_PASSWORD_ENV
        fi
        
        if [ $exit_code -eq 0 ]; then
            login_success=true
            break
        fi
        
        # Store last error for debugging
        last_error="$error_output"
        
        if [ $i -lt 10 ]; then
            log_info "Login attempt $i failed, retrying in 5 seconds..."
            # Show error on first, middle, and last attempt for debugging
            if [ $i -eq 1 ] || [ $i -eq 5 ] || [ $i -eq 9 ]; then
                log_info "Error: ${error_output:0:200}"  # Show first 200 chars
            fi
            sleep 5
        fi
    done
    
    if [ "$login_success" = false ]; then
        log_error "Failed to login to ArgoCD after 10 attempts"
        log_info "Last error: $last_error"
        echo ""
        log_info "Troubleshooting steps:"
        echo "  1. Verify ArgoCD server is ready:"
        echo "     kubectl get pods -n $ARGOCD_NAMESPACE -l app.kubernetes.io/name=argocd-server"
        echo ""
        echo "  2. Check ArgoCD server logs:"
        echo "     kubectl logs -n $ARGOCD_NAMESPACE -l app.kubernetes.io/name=argocd-server --tail=50"
        echo ""
        echo "  3. Verify port-forward is working:"
        echo "     curl -k https://localhost:$ARGOCD_PORT"
        echo ""
        echo "  4. Try logging in manually:"
        echo "     argocd login localhost:$ARGOCD_PORT --insecure"
        echo "     (Username: admin, Password: $ARGOCD_PASSWORD)"
        echo ""
        echo "  5. Check if ArgoCD server needs more time:"
        echo "     kubectl wait --for=condition=ready pod -n $ARGOCD_NAMESPACE -l app.kubernetes.io/name=argocd-server --timeout=300s"
        echo ""
        log_info "You can continue without repository authentication and add it manually later."
        read -p "Do you want to continue without repository authentication? (y/n): " continue_without_auth
        if [[ "$continue_without_auth" =~ ^[Yy]$ ]]; then
            log_warning "Continuing without repository authentication"
            return 1
        else
            return 1
        fi
    fi
    
    log_success "Logged in to ArgoCD"
    
    # Prompt for repository details
    echo ""
    log_info "Enter GitHub repository details:"
    read -p "Repository URL (e.g., https://github.com/username/repo.git): " repo_url
    read -p "Repository name (for ArgoCD, default: extracted from URL): " repo_name
    
    if [ -z "$repo_url" ]; then
        log_warning "No repository URL provided. Skipping repository addition."
        return 1
    fi
    
    # Extract repo name from URL if not provided
    if [ -z "$repo_name" ]; then
        repo_name=$(echo "$repo_url" | sed -E 's|.*github.com[:/]([^/]+)/([^/]+)(\.git)?$|\2|' | sed 's|\.git$||')
    fi
    
    # GitHub App authentication
    echo ""
    log_info "GitHub App Authentication Setup"
    log_info "You need GitHub App credentials (App ID, Installation ID, and Private Key)"
    echo ""
    log_info "If you haven't created a GitHub App yet, see PREDATOR_SETUP.md Step 2-5"
    echo ""
    read -p "GitHub App ID: " github_app_id
    read -p "GitHub App Installation ID: " github_installation_id
    read -p "GitHub App Private Key path (e.g., horizon/configs/github.pem): " github_private_key_path
    
    if [ -z "$github_app_id" ] || [ -z "$github_installation_id" ] || [ -z "$github_private_key_path" ]; then
        log_error "GitHub App ID, Installation ID, and Private Key path are required"
        return 1
    fi
    
    # Expand ~ to home directory and handle relative paths
    github_private_key_path="${github_private_key_path/#\~/$HOME}"
    
    # If relative path, try to resolve from script directory
    if [[ "$github_private_key_path" != /* ]]; then
        local script_dir
        script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        local abs_path="${script_dir}/${github_private_key_path}"
        if [ -f "$abs_path" ]; then
            github_private_key_path="$abs_path"
        fi
    fi
    
    # Check if private key file exists
    if [ ! -f "$github_private_key_path" ]; then
        log_error "GitHub App private key not found at: $github_private_key_path"
        log_info "Please ensure the private key file exists. You can generate it from:"
        echo "  https://github.com/settings/apps/<YOUR_APP_NAME>"
        echo ""
        log_info "The private key should be saved (e.g., to horizon/configs/github.pem)"
        return 1
    fi
    
    log_info "Adding repository to ArgoCD with GitHub App authentication..."
    if argocd repo add "$repo_url" \
        --name "$repo_name" \
        --type git \
        --github-app-id "$github_app_id" \
        --github-app-installation-id "$github_installation_id" \
        --github-app-private-key-path "$github_private_key_path" \
        --insecure; then
        log_success "Repository added successfully with GitHub App authentication"
        # Store repo URL for later use
        export ARGOCD_REPO_URL="$repo_url"
    else
        log_error "Failed to add repository"
        log_info "Troubleshooting:"
        echo "  1. Verify GitHub App ID and Installation ID are correct"
        echo "  2. Ensure the private key file is valid and readable"
        echo "  3. Check that the GitHub App is installed to the repository"
        echo "  4. Verify the GitHub App has 'Contents: Read and write' permission"
        echo "  5. Check ArgoCD CLI version supports GitHub App: argocd version --client"
        return 1
    fi
    
    # Verify repository was added
    log_info "Verifying repository access..."
    # Try to verify with the URL used for adding (should be HTTPS)
    if argocd repo get "$repo_url" --insecure &> /dev/null 2>&1; then
        log_success "Repository verified and accessible"
        
        # Check repository type to ensure it's using GitHub App, not SSH
        local repo_type
        repo_type=$(argocd repo get "$repo_url" --insecure -o json 2>/dev/null | grep -o '"type":"[^"]*' | cut -d'"' -f4 || echo "")
        if [ "$repo_type" = "git" ]; then
            log_info "Repository type: git (GitHub App authentication)"
        else
            log_warning "Repository type: $repo_type (may not be using GitHub App authentication)"
        fi
        return 0
    else
        log_warning "Repository was added but verification failed. It may still work."
        log_info "If you see SSH agent errors, the repository may have been added with SSH authentication."
        log_info "Remove and re-add the repository with GitHub App authentication:"
        echo "  argocd repo remove $repo_url --insecure"
        echo "  # Then re-run this script or add manually with GitHub App"
        return 0
    fi
}

setup_prd_applications() {
    log_info "Setting up automated ArgoCD application onboarding..."
    
    # Use repository URL from previous step if available
    if [ -n "${ARGOCD_REPO_URL:-}" ]; then
        REPO_URL="$ARGOCD_REPO_URL"
        log_info "Using repository: $REPO_URL"
        read -p "Enter your repository branch (default: main): " REPO_BRANCH
        REPO_BRANCH=${REPO_BRANCH:-main}
    else
        read -p "Enter your GitHub repository URL (e.g., https://github.com/username/repo.git): " REPO_URL
        read -p "Enter your repository branch (default: main): " REPO_BRANCH
        REPO_BRANCH=${REPO_BRANCH:-main}
    fi
    
    if [ -z "$REPO_URL" ]; then
        log_warning "No repository URL provided. Skipping prd-applications setup."
        log_info "You can set this up manually later using:"
        echo "  kubectl apply -f predator/prd-applications.yaml"
        return
    fi
    
    # Create prd-applications Application
    log_info "Creating prd-applications ArgoCD Application..."
    log_info "Note: This watches 'prd/applications' directory where Horizon creates ArgoCD Application YAML files"
    log_info "      The Helm chart is at '1.0.0/' (ARGOCD_HELMCHART_PATH) - these are different paths"
    
    kubectl apply -f - <<EOF
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: prd-applications
  namespace: $ARGOCD_NAMESPACE
spec:
  project: default
  source:
    repoURL: $REPO_URL
    targetRevision: $REPO_BRANCH
    path: prd/applications
  destination:
    server: https://kubernetes.default.svc
    namespace: $ARGOCD_NAMESPACE
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
EOF
    
    if kubectl get application prd-applications -n "$ARGOCD_NAMESPACE" &> /dev/null; then
        log_success "prd-applications ArgoCD Application created successfully"
    else
        log_error "Failed to create prd-applications Application"
    fi
}

print_summary() {
    echo ""
    log_success "=========================================="
    log_success "Kubernetes Setup Complete!"
    log_success "=========================================="
    echo ""
    log_info "Next Steps:"
    echo ""
    echo "1. Access ArgoCD UI (port-forward is already running in background):"
    echo "   https://localhost:$ARGOCD_PORT"
    echo "   (Accept the self-signed certificate warning)"
    echo ""
    echo "2. Generate ArgoCD API token:"
    echo "   - Log in to ArgoCD UI"
    echo "   - Go to User Info → Generate New Token"
    echo "   - Or use CLI: argocd login localhost:$ARGOCD_PORT --insecure"
    echo "                 argocd account generate-token"
    echo ""
    echo "3. If you didn't add GitHub repository during setup, add it now (see PREDATOR_SETUP.md Manual Repository Setup):"
    echo "   argocd repo add <YOUR_REPO_URL> --name <REPO_NAME> --type git \\"
    echo "     --github-app-id <APP_ID> --github-app-installation-id <INSTALLATION_ID> \\"
    echo "     --github-app-private-key-path <PATH_TO_KEY>"
    echo ""
    echo "4. Continue with remaining setup steps in PREDATOR_SETUP.md:"
    echo "   - Step 2-7: GitHub App setup"
    echo "   - Step 8: Configure Docker Compose"
    echo ""
    log_info "For detailed instructions, see: quick-start/PREDATOR_SETUP.md"
    echo ""
}

# Main execution
main() {
    echo ""
    log_info "=========================================="
    log_info "Predator Kubernetes Setup Script"
    log_info "=========================================="
    echo ""
    
    check_prerequisites
    label_node
    install_contour
    create_ingress_class
    configure_contour_ingress_class
    install_flagger_crds
    install_keda_crds
    install_priority_class
    install_argocd
    enable_argocd_api_key
    get_argocd_password
    
    # Ask if user wants to add GitHub repository to ArgoCD
    echo ""
    read -p "Do you want to add a GitHub repository to ArgoCD with authentication? (y/n): " ADD_REPO
    if [[ "$ADD_REPO" =~ ^[Yy]$ ]]; then
        if add_github_repo_to_argocd; then
            log_success "GitHub repository authentication configured"
        else
            log_warning "Repository authentication setup failed or was skipped"
        fi
    else
        log_info "Skipping repository authentication. You can add it manually later (see PREDATOR_SETUP.md Step 1.8)"
    fi
    
    # Ask if user wants to set up prd-applications
    echo ""
    read -p "Do you want to set up automated ArgoCD application onboarding (prd-applications)? (y/n): " SETUP_APPS
    if [[ "$SETUP_APPS" =~ ^[Yy]$ ]]; then
        setup_prd_applications
    else
        log_info "Skipping prd-applications setup. You can set this up later."
    fi
    
    print_summary
}

# Run main function
main

