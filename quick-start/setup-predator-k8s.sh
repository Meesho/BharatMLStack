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
# Node name/selector used by deployable_metadata; kind names node <cluster>-control-plane, minikube we set explicitly
TARGET_NODE_NAME="bharatml-stack-control-plane"

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
    
    # Start minikube with node name matching deployable_metadata (nodeSelector)
    log_info "Starting minikube cluster with node name ${TARGET_NODE_NAME} (this may take a few minutes)..."
    if ! minikube start \
        --extra-config=kubeadm.node-name="${TARGET_NODE_NAME}" \
        --extra-config=kubelet.hostname-override="${TARGET_NODE_NAME}"; then
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
    log_info "Note: Docker Desktop node name stays 'docker-desktop'; script will label it ${TARGET_NODE_NAME} so pods can schedule."
    
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
    
    log_info "Node name: $NODE_NAME (label: dedicated=$TARGET_NODE_NAME)"
    
    # Label so pods with nodeSelector from deployable_metadata can schedule (see PREDATOR_SETUP.md troubleshooting)
    kubectl label node "$NODE_NAME" dedicated="$TARGET_NODE_NAME" --overwrite
    
    # Verify the label
    if kubectl get nodes --show-labels | grep -q "dedicated=$TARGET_NODE_NAME"; then
        log_success "Node labeled: dedicated=$TARGET_NODE_NAME"
    else
        log_warning "Node label verification failed, but continuing..."
    fi
}

install_contour_crds() {
    log_info "Installing Contour CRDs (required before ArgoCD can sync Contour)..."
    
    # Check if Contour CRDs are already installed
    if kubectl get crd httpproxies.projectcontour.io &> /dev/null; then
        log_warning "Contour CRDs already installed. Skipping..."
        return
    fi
    
    # Option 1: Install only Contour CRDs (needed before ArgoCD Application can sync)
    # Option 2 (fallback): Install full Contour deployment (includes CRDs + Contour + Envoy)
    # See PREDATOR_SETUP.md troubleshooting for manual commands.
    if ! kubectl apply -f https://raw.githubusercontent.com/projectcontour/contour/main/examples/contour/01-crds.yaml 2>/dev/null; then
        log_warning "Contour CRDs-only install failed. Trying full Contour deployment (includes CRDs)..."
        kubectl apply -f https://projectcontour.io/quickstart/contour.yaml
    fi
    
    # Verify HTTPProxy CRD
    if kubectl get crd httpproxies.projectcontour.io &> /dev/null; then
        log_success "Contour CRDs installed successfully"
    else
        log_error "Contour CRDs installation failed. HTTPProxy CRD not found. You can try manually:"
        echo "  # CRDs only:"
        echo "  kubectl apply -f https://raw.githubusercontent.com/projectcontour/contour/main/examples/contour/01-crds.yaml"
        echo "  # Or full deployment:"
        echo "  kubectl apply -f https://projectcontour.io/quickstart/contour.yaml"
        exit 1
    fi
}

install_contour() {
    log_info "Installing Contour via direct kubectl apply (fallback if ArgoCD not available)..."
    
    # Check if Contour is already installed
    if kubectl get namespace "$CONTOUR_NAMESPACE" &> /dev/null; then
        log_warning "Contour namespace already exists. Skipping Contour installation."
        # Still configure Envoy service if it exists
        configure_envoy_nodeport
        return
    fi
    
    # Install full Contour deployment
    kubectl apply -f https://projectcontour.io/quickstart/contour.yaml
    
    # Wait for Contour pods to be ready
    log_info "Waiting for Contour pods to be ready..."
    kubectl wait --for=condition=ready pod -l app=contour -n "$CONTOUR_NAMESPACE" --timeout=120s || true
    kubectl wait --for=condition=ready pod -l app=envoy -n "$CONTOUR_NAMESPACE" --timeout=120s || true
    
    # Wait for Envoy service to be created
    log_info "Waiting for Envoy service to be created..."
    local max_wait=30
    local waited=0
    while [ $waited -lt $max_wait ]; do
        if kubectl get svc envoy -n "$CONTOUR_NAMESPACE" &> /dev/null; then
            break
        fi
        sleep 2
        waited=$((waited + 2))
    done
    
    # Configure Envoy service as NodePort for local development (fixes stuck LoadBalancer)
    configure_envoy_nodeport
    
    # Verify HTTPProxy CRD
    if kubectl get crd httpproxies.projectcontour.io &> /dev/null; then
        log_success "Contour installed successfully"
    else
        log_error "Contour installation may have failed. HTTPProxy CRD not found."
        exit 1
    fi
}

install_contour_via_argocd() {
    log_info "Installing Contour via ArgoCD Application (production-like setup)..."
    
    # Check if Contour Application already exists
    if kubectl get application contour -n "$ARGOCD_NAMESPACE" &> /dev/null 2>&1; then
        log_warning "Contour ArgoCD Application already exists. Skipping..."
        return
    fi
    
    # Check if ArgoCD is installed
    if ! kubectl get namespace "$ARGOCD_NAMESPACE" &> /dev/null; then
        log_warning "ArgoCD not installed yet. Contour will be installed via ArgoCD after ArgoCD setup."
        return
    fi
    
    # Create Contour Application in ArgoCD (using official projectcontour repo)
    log_info "Creating Contour ArgoCD Application (official Contour manifests)..."
    kubectl apply -f - <<EOF
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: contour
  namespace: $ARGOCD_NAMESPACE
spec:
  project: default
  source:
    repoURL: https://github.com/projectcontour/contour
    targetRevision: v1.30.0
    path: examples/contour
  destination:
    server: https://kubernetes.default.svc
    namespace: $CONTOUR_NAMESPACE
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
EOF
    
    if kubectl get application contour -n "$ARGOCD_NAMESPACE" &> /dev/null; then
        log_success "Contour ArgoCD Application created successfully"
        log_info "Waiting for ArgoCD to sync Contour..."
        sleep 5
        
        # Wait for Contour namespace to be created
        local max_wait=60
        local waited=0
        while [ $waited -lt $max_wait ]; do
            if kubectl get namespace "$CONTOUR_NAMESPACE" &> /dev/null; then
                break
            fi
            sleep 2
            waited=$((waited + 2))
        done
        
        # Wait for Contour pods to be ready
        if kubectl get namespace "$CONTOUR_NAMESPACE" &> /dev/null; then
            log_info "Waiting for Contour pods to be ready..."
            kubectl wait --for=condition=ready pod -l app=contour -n "$CONTOUR_NAMESPACE" --timeout=180s || true
            kubectl wait --for=condition=ready pod -l app=envoy -n "$CONTOUR_NAMESPACE" --timeout=180s || true
            
            # Wait for Envoy service to be created
            log_info "Waiting for Envoy service to be created..."
            local max_wait=30
            local waited=0
            while [ $waited -lt $max_wait ]; do
                if kubectl get svc envoy -n "$CONTOUR_NAMESPACE" &> /dev/null; then
                    break
                fi
                sleep 2
                waited=$((waited + 2))
            done
            
            # Disable Envoy hostPorts for local development (enables port-forward)
            disable_envoy_hostports
            
            # Configure Envoy service as NodePort for local access
            configure_envoy_nodeport
        fi
    else
        log_error "Failed to create Contour ArgoCD Application"
        exit 1
    fi
}

disable_envoy_hostports() {
    log_info "Disabling Envoy hostPorts for local development (enables port-forward)..."
    
    # Check if Envoy DaemonSet exists
    if ! kubectl get ds envoy -n "$CONTOUR_NAMESPACE" &> /dev/null; then
        log_warning "Envoy DaemonSet not found. It may not be ready yet."
        return 0
    fi
    
    # Check if hostPorts are already disabled
    local has_hostport
    has_hostport=$(kubectl get ds envoy -n "$CONTOUR_NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].ports[*].hostPort}' 2>/dev/null || echo "")
    
    if [ -z "$has_hostport" ]; then
        log_success "Envoy hostPorts are already disabled"
        return 0
    fi
    
    log_info "Removing hostPorts from Envoy DaemonSet (fixes port-forward issues)..."
    
    # Patch the DaemonSet to remove hostPort from all ports
    kubectl -n "$CONTOUR_NAMESPACE" patch ds envoy --type='json' -p='[
      {"op": "remove", "path": "/spec/template/spec/containers/0/ports/0/hostPort"},
      {"op": "remove", "path": "/spec/template/spec/containers/0/ports/1/hostPort"}
    ]' 2>/dev/null || {
        log_warning "Failed to remove hostPorts. They may already be removed or have different indices."
        # Try alternative: get current ports and rebuild without hostPort
        log_info "Trying alternative method to remove hostPorts..."
        kubectl -n "$CONTOUR_NAMESPACE" get ds envoy -o json | \
        jq '.spec.template.spec.containers[0].ports |= map(del(.hostPort))' | \
        kubectl apply -f - 2>/dev/null || {
            log_warning "Alternative method also failed. Continuing anyway..."
            return 0
        }
    }
    
    # Wait for DaemonSet to roll out
    log_info "Waiting for Envoy DaemonSet to roll out..."
    sleep 3
    kubectl -n "$CONTOUR_NAMESPACE" rollout status ds envoy --timeout=120s || true
    
    # Verify hostPorts are removed
    has_hostport=$(kubectl get ds envoy -n "$CONTOUR_NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].ports[*].hostPort}' 2>/dev/null || echo "")
    
    if [ -z "$has_hostport" ]; then
        log_success "Envoy hostPorts disabled successfully (port-forward will now work)"
    else
        log_warning "Envoy hostPorts may still be present. Port-forward might fail."
    fi
}

configure_envoy_nodeport() {
    log_info "Configuring Envoy service as NodePort for local access..."
    
    # Check if Envoy service exists
    if ! kubectl get svc envoy -n "$CONTOUR_NAMESPACE" &> /dev/null; then
        log_warning "Envoy service not found. It may not be ready yet."
        log_info "You can configure it later by running this function again"
        return 0
    fi
    
    # Check current service type
    local current_type
    current_type=$(kubectl get svc envoy -n "$CONTOUR_NAMESPACE" -o jsonpath='{.spec.type}' 2>/dev/null || echo "")
    
    if [ "$current_type" = "NodePort" ]; then
        log_success "Envoy service is already configured as NodePort"
        return 0
    fi
    
    # Get existing ports to preserve them (Envoy typically has http:80 and https:443)
    local existing_ports_json
    existing_ports_json=$(kubectl get svc envoy -n "$CONTOUR_NAMESPACE" -o jsonpath='{.spec.ports}' 2>/dev/null || echo "[]")
    
    # Extract port information and convert to NodePort format
    # We'll preserve all ports but ensure http port (80) gets nodePort 30080
    log_info "Preserving existing ports and converting to NodePort..."
    
    # Patch Envoy service to NodePort, preserving all existing ports
    # Use strategic merge patch to preserve ports we don't modify
    kubectl patch svc envoy -n "$CONTOUR_NAMESPACE" --type='merge' -p='{
        "spec": {
            "type": "NodePort",
            "ports": [
                {
                    "name": "http",
                    "port": 80,
                    "targetPort": 8080,
                    "nodePort": 30080,
                    "protocol": "TCP"
                }
            ]
        }
    }' || {
        # Fallback: use JSON patch if merge fails
        log_info "Trying alternative patch method..."
        kubectl patch svc envoy -n "$CONTOUR_NAMESPACE" --type='json' -p='[
            {"op": "replace", "path": "/spec/type", "value": "NodePort"}
        ]' || {
            log_error "Failed to patch Envoy service to NodePort"
            log_info "You can manually fix it with:"
            echo "  kubectl patch svc envoy -n $CONTOUR_NAMESPACE --type='merge' -p='{\"spec\":{\"type\":\"NodePort\"}}'"
            return 1
        }
    }
    
    # Wait a moment for the change to propagate
    sleep 2
    
    # Verify the change
    local updated_type
    updated_type=$(kubectl get svc envoy -n "$CONTOUR_NAMESPACE" -o jsonpath='{.spec.type}' 2>/dev/null || echo "")
    
    if [ "$updated_type" = "NodePort" ]; then
        log_success "Envoy service configured as NodePort"
        
        # Get the actual nodePort assigned (might be different from 30080 if auto-assigned)
        local node_port
        node_port=$(kubectl get svc envoy -n "$CONTOUR_NAMESPACE" -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}' 2>/dev/null || echo "")
        
        if [ -n "$node_port" ]; then
            log_info "Envoy NodePort: $node_port (accessible on localhost for Docker Desktop/minikube)"
        fi
        
        local cluster_type
        cluster_type=$(detect_cluster_type)
        if [ "$cluster_type" = "kind" ]; then
            log_info "Note: With kind clusters, NodePort is not accessible from localhost"
            log_info "Use port-forward instead: kubectl port-forward -n $CONTOUR_NAMESPACE svc/envoy 30080:80"
        fi
    else
        log_error "Failed to verify Envoy NodePort configuration"
        return 1
    fi
}

verify_envoy_connectivity() {
    log_info "Verifying Envoy connectivity to Contour..."
    
    # Wait for Envoy pods to be ready
    if ! kubectl wait --for=condition=ready pod -l app=envoy -n "$CONTOUR_NAMESPACE" --timeout=120s 2>/dev/null; then
        log_warning "Envoy pods may not be ready yet"
        return 1
    fi
    
    # Check for TLS certificate errors in Envoy logs
    log_info "Checking for TLS certificate errors in Envoy logs..."
    local envoy_pod
    envoy_pod=$(kubectl get pods -n "$CONTOUR_NAMESPACE" -l app=envoy -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$envoy_pod" ]; then
        log_warning "No Envoy pod found"
        return 1
    fi
    
    # Check for TLS errors in the last 50 lines of logs
    local tls_errors
    tls_errors=$(kubectl logs -n "$CONTOUR_NAMESPACE" "$envoy_pod" -c envoy --tail=50 2>/dev/null | grep -iE "(TLS_error|certificate.*fail|CERTIFICATE_VERIFY_FAILED)" || true)
    
    if [ -n "$tls_errors" ]; then
        log_warning "TLS certificate errors detected in Envoy logs"
        log_info "Restarting Envoy pod to fix certificate connectivity issues..."
        
        # Delete the Envoy pod to force restart
        kubectl delete pod "$envoy_pod" -n "$CONTOUR_NAMESPACE" --wait=false
        
        # Wait for new pod to be ready
        log_info "Waiting for Envoy pod to restart..."
        sleep 5
        if kubectl wait --for=condition=ready pod -l app=envoy -n "$CONTOUR_NAMESPACE" --timeout=120s 2>/dev/null; then
            log_success "Envoy pod restarted successfully"
            
            # Wait a bit more for Envoy to fully initialize
            sleep 5
            
            # Check again for TLS errors
            local new_envoy_pod
            new_envoy_pod=$(kubectl get pods -n "$CONTOUR_NAMESPACE" -l app=envoy -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
            local new_tls_errors
            new_tls_errors=$(kubectl logs -n "$CONTOUR_NAMESPACE" "$new_envoy_pod" -c envoy --tail=30 2>/dev/null | grep -iE "(TLS_error|certificate.*fail|CERTIFICATE_VERIFY_FAILED)" || true)
            
            if [ -z "$new_tls_errors" ]; then
                log_success "TLS errors resolved after restart"
            else
                log_warning "TLS errors may still be present. This might be transient during startup."
            fi
        else
            log_warning "Envoy pod restart may have failed or is still starting"
        fi
    else
        log_success "No TLS certificate errors detected in Envoy logs"
    fi
    
    # Verify Envoy is initialized
    log_info "Verifying Envoy initialization..."
    local envoy_ready
    envoy_ready=$(kubectl logs -n "$CONTOUR_NAMESPACE" "$envoy_pod" -c envoy --tail=20 2>/dev/null | grep -iE "(all clusters initialized|all dependencies initialized|starting workers)" || true)
    
    if [ -n "$envoy_ready" ]; then
        log_success "Envoy is initialized and ready"
    else
        log_warning "Envoy may still be initializing"
    fi
}

detect_cluster_type() {
    # Detect if we're running on kind, minikube, or other
    if kubectl get nodes -o jsonpath='{.items[0].metadata.name}' 2>/dev/null | grep -q "kind"; then
        echo "kind"
    elif kubectl get nodes -o jsonpath='{.items[0].metadata.labels}' 2>/dev/null | grep -q "minikube"; then
        echo "minikube"
    elif command -v minikube &> /dev/null && minikube status &> /dev/null; then
        echo "minikube"
    else
        echo "unknown"
    fi
}

create_ingress_class() {
    log_info "Creating IngressClass for contour-internal..."
    
    # Check if IngressClass already exists
    if kubectl get ingressclass contour-internal &> /dev/null; then
        log_warning "IngressClass 'contour-internal' already exists. Skipping..."
        return
    fi
    
    # Create IngressClass (contour-internal)
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
    
    # Ensure Contour namespace exists before patching
    if ! kubectl get namespace "$CONTOUR_NAMESPACE" &> /dev/null; then
        log_warning "Contour namespace '$CONTOUR_NAMESPACE' not found. Skipping Contour ingress-class patch (Contour not installed yet?)."
        log_info "Contour will be patched automatically on the next run once the namespace exists."
        return
    fi
    
    # Check if Contour is already configured with contour-internal
    if kubectl -n "$CONTOUR_NAMESPACE" get deploy contour -o jsonpath='{.spec.template.spec.containers[0].args}' 2>/dev/null | grep -q "ingress-class-name=contour-internal"; then
        log_warning "Contour is already configured for contour-internal. Skipping..."
        return
    fi
    
    # Remove any existing ingress-class-name argument first (to avoid duplicates)
    local existing_args
    existing_args=$(kubectl -n "$CONTOUR_NAMESPACE" get deploy contour -o jsonpath='{.spec.template.spec.containers[0].args}' 2>/dev/null || echo "[]")
    
    # Add ingress-class-name argument to Contour deployment (contour-internal)
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

install_keda() {
    log_info "Installing KEDA operator..."
    
    # Check if KEDA is already installed
    if kubectl get deployment keda-operator -n keda &> /dev/null 2>&1 || \
       kubectl get deployment keda-operator -n keda-system &> /dev/null 2>&1; then
        log_warning "KEDA operator already installed. Skipping..."
        return
    fi
    
    # Install KEDA (full operator installation)
    log_info "Installing KEDA operator (version ${KEDA_VERSION})..."
    log_info "Note: ScaledJob CRD may show an error due to oversized annotations - this is expected and can be ignored"
    
    # Install KEDA (ScaledJob CRD error is expected and can be ignored)
    # We capture output and filter out the ScaledJob error, then verify installation succeeded
    local install_output
    # Asset filename is keda-2.12.0.yaml (no 'v' in filename), tag is v2.12.0
    local keda_yaml="keda-${KEDA_VERSION#v}.yaml"
    install_output=$(kubectl apply -f "https://github.com/kedacore/keda/releases/download/${KEDA_VERSION}/${keda_yaml}" 2>&1) || {
        # Installation may have failed, but check if it's just the ScaledJob CRD error
        if echo "$install_output" | grep -q "scaledjobs.keda.sh.*Too long"; then
            log_warning "ScaledJob CRD installation failed (expected - not needed for Predator)"
        else
            log_error "KEDA installation failed with unexpected error:"
            echo "$install_output"
            exit 1
        fi
    }
    
    # Show output excluding ScaledJob errors
    echo "$install_output" | grep -v "scaledjobs.keda.sh.*Too long" || true
    
    # Verify that KEDA operator was actually installed (even if ScaledJob failed)
    if ! kubectl get deployment keda-operator -n keda &> /dev/null 2>&1 && \
       ! kubectl get deployment keda-operator -n keda-system &> /dev/null 2>&1; then
        log_error "KEDA operator deployment not found after installation"
        exit 1
    fi
    
    # Wait for KEDA namespace to be created (it might be 'keda' or 'keda-system')
    log_info "Waiting for KEDA namespace to be ready..."
    sleep 3
    
    # Check which namespace KEDA was installed in
    local keda_namespace=""
    if kubectl get namespace keda &> /dev/null 2>&1; then
        keda_namespace="keda"
    elif kubectl get namespace keda-system &> /dev/null 2>&1; then
        keda_namespace="keda-system"
    else
        log_error "KEDA namespace not found after installation"
        exit 1
    fi
    
    log_info "Waiting for KEDA operator pods to be ready..."
    if kubectl wait --for=condition=ready pod -l app=keda-operator -n ${keda_namespace} --timeout=300s 2>/dev/null; then
        log_success "KEDA operator installed and ready"
    else
        log_warning "KEDA operator pods may still be starting. Checking status..."
        kubectl get pods -n ${keda_namespace} || true
        log_info "KEDA installation completed. Pods may take a few more minutes to be fully ready."
    fi
    
    # Verify KEDA components
    if kubectl get deployment keda-operator -n ${keda_namespace} &> /dev/null; then
        log_success "KEDA operator deployment verified"
    else
        log_error "KEDA operator deployment verification failed"
        exit 1
    fi
}

ensure_predator_service_grpc() {
    log_info "Ensuring Predator Service has correct gRPC configuration..."
    
    # Predator namespace (standard naming: prd-predator)
    local PREDATOR_NAMESPACE="prd-predator"
    local PREDATOR_SERVICE="prd-predator"
    
    # Check if namespace exists
    if ! kubectl get namespace "$PREDATOR_NAMESPACE" &> /dev/null; then
        log_warning "Predator namespace '$PREDATOR_NAMESPACE' does not exist yet."
        log_info "This function will be called again after Predator is deployed."
        return 0
    fi
    
    # Check if service exists
    if ! kubectl -n "$PREDATOR_NAMESPACE" get svc "$PREDATOR_SERVICE" &> /dev/null; then
        log_warning "Predator service '$PREDATOR_SERVICE' does not exist yet."
        log_info "This function will be called again after Predator is deployed."
        return 0
    fi
    
    # Check current port configuration
    local current_port_name
    current_port_name=$(kubectl -n "$PREDATOR_NAMESPACE" get svc "$PREDATOR_SERVICE" -o jsonpath='{.spec.ports[0].name}' 2>/dev/null || echo "")
    
    # Check if port name is already 'grpc' (preferred method)
    if [ "$current_port_name" = "grpc" ]; then
        log_success "Predator Service already has port name 'grpc' (correct configuration)"
        return 0
    fi
    
    # Check if h2c annotation exists (fallback method)
    local h2c_annotation
    h2c_annotation=$(kubectl -n "$PREDATOR_NAMESPACE" get svc "$PREDATOR_SERVICE" -o jsonpath='{.metadata.annotations.projectcontour\.io/upstream-protocol\.h2c}' 2>/dev/null || echo "")
    
    if [ -n "$h2c_annotation" ]; then
        log_warning "Predator Service uses h2c annotation (less reliable than port name)"
        log_info "Updating to use port name 'grpc' instead (more reliable)..."
    fi
    
    # Get current port configuration
    local current_port
    local current_target_port
    current_port=$(kubectl -n "$PREDATOR_NAMESPACE" get svc "$PREDATOR_SERVICE" -o jsonpath='{.spec.ports[0].port}' 2>/dev/null || echo "80")
    current_target_port=$(kubectl -n "$PREDATOR_NAMESPACE" get svc "$PREDATOR_SERVICE" -o jsonpath='{.spec.ports[0].targetPort}' 2>/dev/null || echo "8001")
    
    # Patch service to use port name 'grpc' (preferred method - more reliable than annotation)
    log_info "Patching Predator Service to use port name 'grpc'..."
    kubectl -n "$PREDATOR_NAMESPACE" patch svc "$PREDATOR_SERVICE" --type='json' -p="[
      {
        \"op\": \"replace\",
        \"path\": \"/spec/ports/0/name\",
        \"value\": \"grpc\"
      }
    ]" || {
        log_error "Failed to patch Predator Service port name"
        return 1
    }
    
    # Remove h2c annotation if it exists (port name is sufficient)
    if [ -n "$h2c_annotation" ]; then
        log_info "Removing h2c annotation (port name 'grpc' is sufficient)..."
        kubectl -n "$PREDATOR_NAMESPACE" annotate svc "$PREDATOR_SERVICE" projectcontour.io/upstream-protocol.h2c- 2>/dev/null || true
    fi
    
    # Verify the change
    local updated_port_name
    updated_port_name=$(kubectl -n "$PREDATOR_NAMESPACE" get svc "$PREDATOR_SERVICE" -o jsonpath='{.spec.ports[0].name}' 2>/dev/null || echo "")
    
    if [ "$updated_port_name" = "grpc" ]; then
        log_success "Predator Service configured with port name 'grpc' (enables HTTP/2 upstream for gRPC)"
        log_info "Service port: $current_port, targetPort: $current_target_port"
    else
        log_error "Failed to verify Predator Service port name update"
        return 1
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
    
    # Install ArgoCD (ApplicationSet CRD may fail with "metadata.annotations: Too long" - optional for core Argo CD)
    local argocd_output
    argocd_output=$(kubectl apply -n "$ARGOCD_NAMESPACE" -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml 2>&1) || {
        if echo "$argocd_output" | grep -qE "applicationsets\.argoproj\.io.*(invalid|Too long)|metadata\.annotations: Too long"; then
            log_warning "ApplicationSet CRD installation failed (expected - annotations exceed limit; not required for core Argo CD)"
        else
            log_error "ArgoCD installation failed:"
            echo "$argocd_output"
            exit 1
        fi
    }
    echo "$argocd_output" | grep -vE "applicationsets\.argoproj\.io.*(invalid|Too long)|metadata\.annotations: Too long" || true

    # Verify core Argo CD resources were created
    if ! kubectl get deployment argocd-server -n "$ARGOCD_NAMESPACE" &> /dev/null; then
        log_error "ArgoCD server deployment not found after installation"
        exit 1
    fi

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
        # Store repo URL and GitHub App credentials for Step 6 (copy Predator chart to repo)
        export ARGOCD_REPO_URL="$repo_url"
        export GITHUB_APP_ID="$github_app_id"
        export GITHUB_INSTALLATION_ID="$github_installation_id"
        export GITHUB_APP_PRIVATE_KEY_PATH="$github_private_key_path"
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

# get_github_installation_token generates a GitHub App installation access token (for git clone/push).
# Requires: jq, openssl, curl. Outputs the token to stdout; returns 0 on success.
get_github_installation_token() {
    local app_id="$1"
    local installation_id="$2"
    local private_key_path="$3"
    local now
    now=$(date +%s)
    local exp=$((now + 600))
    local header='{"alg":"RS256","typ":"JWT"}'
    local payload
    payload=$(jq -n --arg iat "$now" --arg exp "$exp" --arg iss "$app_id" '{iat: ($iat|tonumber), exp: ($exp|tonumber), iss: $iss}')
    local header_b64 payload_b64
    header_b64=$(echo -n "$header" | base64 2>/dev/null | tr -d '\n' | tr '+/' '-_' | tr -d '=')
    payload_b64=$(echo -n "$payload" | base64 2>/dev/null | tr -d '\n' | tr '+/' '-_' | tr -d '=')
    local signed_content="${header_b64}.${payload_b64}"
    local sig_b64
    sig_b64=$(echo -n "$signed_content" | openssl dgst -sha256 -sign "$private_key_path" 2>/dev/null | base64 2>/dev/null | tr -d '\n' | tr '+/' '-_' | tr -d '=')
    local jwt="${signed_content}.${sig_b64}"
    local token
    token=$(curl -s -X POST \
        -H "Authorization: Bearer ${jwt}" \
        -H "Accept: application/vnd.github+json" \
        "https://api.github.com/app/installations/${installation_id}/access_tokens" | jq -r '.token')
    if [ -z "$token" ] || [ "$token" = "null" ]; then
        return 1
    fi
    echo -n "$token"
}

# copy_predator_chart_to_repo implements PREDATOR_SETUP.md Step 6: copy Predator Helm chart to repository.
# Uses GitHub App installation token to clone, add 1.0.0, commit and push.
copy_predator_chart_to_repo() {
    local repo_url="${ARGOCD_REPO_URL:-}"
    local branch="${REPO_BRANCH:-}"
    if [ -z "$branch" ]; then
        read -p "Enter your repository branch (default: main): " branch
        branch=${branch:-main}
    fi
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local chart_source="${script_dir}/../predator/1.0.0"
    local configs_services_source="${script_dir}/../horizon/configs/services"
    if [ -z "$repo_url" ]; then
        log_error "No repository URL set. Add the repository to ArgoCD first."
        return 1
    fi
    if [ ! -d "$chart_source" ]; then
        log_error "Predator Helm chart not found at: $chart_source"
        log_info "Run this script from the BharatMLStack repo (quick-start/)."
        return 1
    fi
    if [ -z "${GITHUB_APP_ID:-}" ] || [ -z "${GITHUB_INSTALLATION_ID:-}" ] || [ -z "${GITHUB_APP_PRIVATE_KEY_PATH:-}" ]; then
        log_error "GitHub App credentials not set. Add the repository with GitHub App first."
        return 1
    fi
    if ! command -v jq &> /dev/null; then
        log_error "jq is required for Step 6 (GitHub App token). Install with: brew install jq (macOS) or apt install jq (Linux)"
        return 1
    fi
    log_info "Step 6: Copying Predator Helm chart to repository..."
    local token
    token=$(get_github_installation_token "$GITHUB_APP_ID" "$GITHUB_INSTALLATION_ID" "$GITHUB_APP_PRIVATE_KEY_PATH") || {
        log_error "Failed to get GitHub App installation token. Check App ID, Installation ID, and private key."
        return 1
    }
    # Build HTTPS URL with token for clone/push (github.com only)
    local repo_url_authed
    if [[ "$repo_url" =~ ^https://github\.com/(.+)$ ]]; then
        repo_url_authed="https://x-access-token:${token}@github.com/${BASH_REMATCH[1]}"
    elif [[ "$repo_url" =~ ^https://(.+)$ ]]; then
        log_warning "Repository is not github.com; using URL as-is (push may require existing credentials)"
        repo_url_authed="$repo_url"
    else
        log_error "Unsupported repository URL for automated push: $repo_url"
        return 1
    fi
    local tmpdir
    tmpdir=$(mktemp -d 2>/dev/null || mktemp -d -t predator-step6)
    trap "rm -rf '$tmpdir'" RETURN
    log_info "Cloning repository (branch: $branch)..."
    if ! git clone --branch "$branch" --depth 1 "$repo_url_authed" "$tmpdir/repo" 2>/dev/null; then
        log_warning "Clone failed (branch may not exist). Trying default branch..."
        git clone --depth 1 "$repo_url_authed" "$tmpdir/repo" 2>/dev/null || {
            log_error "Failed to clone repository."
            return 1
        }
        cd "$tmpdir/repo"
        branch=$(git rev-parse --abbrev-ref HEAD)
        cd - >/dev/null
    fi
    local repo_dir="$tmpdir/repo"
    if [ -d "$repo_dir/1.0.0" ]; then
        log_info "1.0.0/ already exists in repository; updating contents..."
        rm -rf "$repo_dir/1.0.0"
    fi
    cp -r "$chart_source" "$repo_dir/1.0.0"

    # Create prd/applications/ (blank folder tracked with .gitkeep)
    mkdir -p "$repo_dir/prd/applications"
    if [ ! -f "$repo_dir/prd/applications/.gitkeep" ]; then
        touch "$repo_dir/prd/applications/.gitkeep"
    fi

    # Copy horizon/configs/services as configs/services/ in repo
    if [ -d "$configs_services_source" ]; then
        log_info "Copying configs/services (from horizon/configs/services)..."
        mkdir -p "$repo_dir/configs"
        if [ -d "$repo_dir/configs/services" ]; then
            rm -rf "$repo_dir/configs/services"
        fi
        cp -r "$configs_services_source" "$repo_dir/configs/services"
    else
        log_warning "horizon/configs/services not found at $configs_services_source; skipping configs/services copy"
    fi

    cd "$repo_dir"
    git config user.email "horizon-bot@predator-setup.local"
    git config user.name "Predator Setup"
    git add 1.0.0 prd/applications
    [ -d configs/services ] && git add configs/services
    if git diff --cached --quiet; then
        log_info "No changes to commit (repo already up to date)."
        return 0
    fi
    git commit -m "Add Predator Helm chart 1.0.0, prd/applications, configs/services (via setup-predator-k8s.sh Step 6)"
    if ! git push origin "$branch" 2>/dev/null; then
        log_warning "Push failed. You can push manually from your repo:"
        echo "  cd <your-repo> && git push origin $branch"
        return 1
    fi
    cd - >/dev/null
    log_success "Step 6 complete: 1.0.0/, prd/applications/, configs/services/ copied to repository and pushed to $branch"
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

setup_local_dns() {
    log_info "Setting up local DNS for Predator service..."
    
    local fqdn="predator.prd.meesho.int"
    local hosts_entry="127.0.0.1 $fqdn"
    
    # Check if entry already exists
    if grep -q "$fqdn" /etc/hosts 2>/dev/null; then
        log_warning "DNS entry for $fqdn already exists in /etc/hosts"
        return
    fi
    
    log_info "To enable local DNS resolution, add this to /etc/hosts:"
    echo ""
    echo "  $hosts_entry"
    echo ""
    read -p "Do you want to add this entry to /etc/hosts now? (requires sudo) (y/n): " ADD_DNS
    
    if [[ "$ADD_DNS" =~ ^[Yy]$ ]]; then
        if sudo sh -c "echo '$hosts_entry' >> /etc/hosts"; then
            log_success "DNS entry added to /etc/hosts"
        else
            log_warning "Failed to add DNS entry. Please add it manually:"
            echo "  sudo sh -c \"echo '$hosts_entry' >> /etc/hosts\""
        fi
    else
        log_info "Skipping DNS setup. Add this manually to /etc/hosts:"
        echo "  $hosts_entry"
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
    echo "2. Access Predator service directly via port-forward:"
    echo "   kubectl -n prd-predator port-forward svc/prd-predator 8090:80"
    echo "   Then test with:"
    echo "   grpcurl -plaintext -import-path helix-client/pkg/clients/predator/client/proto \\"
    echo "     -proto grpc_service.proto -d '{}' \\"
    echo "     localhost:8090 inference.GRPCInferenceService/ServerLive"
    echo ""
    echo "3. Set up local DNS (optional - for FQDN-based access):"
    echo "   Add to /etc/hosts: 127.0.0.1 predator.prd.meesho.int"
    echo "   Or run: sudo sh -c \"echo '127.0.0.1 predator.prd.meesho.int' >> /etc/hosts\""
    echo ""
    echo "4. Generate ArgoCD API token:"
    echo "   - Log in to ArgoCD UI"
    echo "   - Go to User Info → Generate New Token"
    echo "   - Or use CLI: argocd login localhost:$ARGOCD_PORT --insecure"
    echo "                 argocd account generate-token"
    echo ""
    echo "5. If you didn't add GitHub repository during setup, add it now (see PREDATOR_SETUP.md Manual Repository Setup):"
    echo "   argocd repo add <YOUR_REPO_URL> --name <REPO_NAME> --type git \\"
    echo "     --github-app-id <APP_ID> --github-app-installation-id <INSTALLATION_ID> \\"
    echo "     --github-app-private-key-path <PATH_TO_KEY>"
    echo ""
    echo "6. Important: Predator Service gRPC configuration (automatically enforced):"
    echo "   - Service port: 80 (not 8001)"
    echo "   - Target port: 8001 (maps to container port 8001)"
    echo "   - Port name: 'grpc' (REQUIRED for HTTP/2 upstream - automatically set by script)"
    echo "   This enables HTTP/2 upstream protocol for Contour routing to Triton gRPC"
    echo "   Note: Port name 'grpc' enables HTTP/2 upstream for gRPC (production best practice)"
    echo ""
    echo "7. Note: Contour is disabled for local development (simplified setup)"
    echo "   Access services directly via port-forward to their K8s services"
    echo "   For production, enable Contour in the script (uncomment Contour functions in main())"
    echo ""
    echo "8. Continue with remaining setup steps in PREDATOR_SETUP.md:"
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

    # Contour setup (commented out - use direct port-forward for local dev)
    # For production, uncomment these lines:
    # install_contour_crds
    # create_ingress_class
    # install_contour_via_argocd
    # configure_contour_ingress_class
    # verify_envoy_connectivity
    
    install_flagger_crds
    install_keda_crds
    install_keda
    install_priority_class
    install_argocd
    enable_argocd_api_key
    get_argocd_password
    
    # Ensure Predator Service has correct gRPC configuration (if it exists)
    ensure_predator_service_grpc
    
    # Ask if user wants to add GitHub repository to ArgoCD
    echo ""
    read -p "Do you want to add a GitHub repository to ArgoCD with authentication? (y/n): " ADD_REPO
    if [[ "$ADD_REPO" =~ ^[Yy]$ ]]; then
        if add_github_repo_to_argocd; then
            log_success "GitHub repository authentication configured"
            # Step 6: Copy Predator Helm chart to repository (PREDATOR_SETUP.md)
            echo ""
            read -p "Do you want to copy the Predator Helm chart to your repository (Step 6)? (y/n): " COPY_CHART
            if [[ "$COPY_CHART" =~ ^[Yy]$ ]]; then
                if copy_predator_chart_to_repo; then
                    log_success "Step 6 complete: Predator Helm chart is in your repo at 1.0.0/"
                else
                    log_warning "Step 6 failed or was skipped. You can do it manually (see PREDATOR_SETUP.md Step 6)"
                fi
            else
                log_info "Skipping Step 6. You can copy the chart manually (see PREDATOR_SETUP.md Step 6)"
            fi
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
    
    # Ask if user wants to set up local DNS
    echo ""
    read -p "Do you want to set up local DNS for Predator service? (y/n): " SETUP_DNS
    if [[ "$SETUP_DNS" =~ ^[Yy]$ ]]; then
        setup_local_dns
    else
        log_info "Skipping DNS setup. You can add it manually to /etc/hosts later."
    fi
    
    print_summary
}

# Run main function
main

