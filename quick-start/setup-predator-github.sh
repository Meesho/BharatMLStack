#!/bin/bash
# setup-predator-github.sh — GitHub repo population and optional ArgoCD repo registration
# No Docker/Kubernetes/ArgoCD installation. Requires: repo URL, GitHub App ID, Installation ID, private key path.
# Optionally: ArgoCD server URL + token to add the repository to ArgoCD.
# See GITHUB_REPO_SETUP.md for prerequisites (GitHub App creation).

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_SOURCE="${SCRIPT_DIR}/../predator/1.0.0"
CONFIGS_SERVICES_SOURCE="${SCRIPT_DIR}/../horizon/configs/services"

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Runs interactively by default: prompts for repo URL, GitHub App ID, Installation ID,"
    echo "and private key path (one by one). Optional flags pre-fill or skip prompts."
    echo ""
    echo "Optional (pre-fill prompts):"
    echo "  --repo-url <URL>              GitHub repo URL (e.g. https://github.com/username/repo.git)"
    echo "  --app-id <ID>                 GitHub App ID"
    echo "  --installation-id <ID>        GitHub App Installation ID"
    echo "  --private-key <PATH>          Path to GitHub App private key (.pem)"
    echo "  --branch <BRANCH>             Branch to push to (default: main)"
    echo "  --repo-name <NAME>            ArgoCD repo name (default: derived from URL)"
    echo "  --argocd-server <URL>         ArgoCD server URL (e.g. https://localhost:8087)"
    echo "  --argocd-token <TOKEN>        ArgoCD auth token (to add repo to ArgoCD)"
    echo "  -h, --help                    Show this help"
    exit 0
}

# Parse arguments
REPO_URL=""
GITHUB_APP_ID=""
GITHUB_INSTALLATION_ID=""
GITHUB_PRIVATE_KEY_PATH=""
BRANCH="main"
REPO_NAME=""
ARGOCD_SERVER=""
ARGOCD_TOKEN=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --repo-url)         REPO_URL="$2"; shift 2 ;;
        --app-id)           GITHUB_APP_ID="$2"; shift 2 ;;
        --installation-id)  GITHUB_INSTALLATION_ID="$2"; shift 2 ;;
        --private-key)      GITHUB_PRIVATE_KEY_PATH="$2"; shift 2 ;;
        --branch)           BRANCH="$2"; shift 2 ;;
        --repo-name)        REPO_NAME="$2"; shift 2 ;;
        --argocd-server)    ARGOCD_SERVER="$2"; shift 2 ;;
        --argocd-token)     ARGOCD_TOKEN="$2"; shift 2 ;;
        -h|--help)          usage ;;
        *)                  log_error "Unknown option: $1"; echo ""; usage ;;
    esac
done

# Prompt for any missing required values (one by one, like setup-predator-k8s.sh)
echo ""
log_info "Enter GitHub repository and App details (see GITHUB_REPO_SETUP.md for creating a GitHub App):"
echo ""

if [ -z "$REPO_URL" ]; then
    read -p "Repository URL (e.g., https://github.com/username/repo.git): " REPO_URL
fi
if [ -z "$REPO_URL" ]; then
    log_error "Repository URL is required."
    exit 1
fi

if [ -z "$GITHUB_APP_ID" ]; then
    read -p "GitHub App ID: " GITHUB_APP_ID
fi
if [ -z "$GITHUB_APP_ID" ]; then
    log_error "GitHub App ID is required."
    exit 1
fi

if [ -z "$GITHUB_INSTALLATION_ID" ]; then
    read -p "GitHub App Installation ID: " GITHUB_INSTALLATION_ID
fi
if [ -z "$GITHUB_INSTALLATION_ID" ]; then
    log_error "GitHub App Installation ID is required."
    exit 1
fi

if [ -z "$GITHUB_PRIVATE_KEY_PATH" ]; then
    read -p "GitHub App Private Key path (e.g., horizon/configs/github.pem): " GITHUB_PRIVATE_KEY_PATH
fi
if [ -z "$GITHUB_PRIVATE_KEY_PATH" ]; then
    log_error "GitHub App Private Key path is required."
    exit 1
fi

read -p "Branch to push to (default: main): " BRANCH_INPUT
BRANCH="${BRANCH_INPUT:-$BRANCH}"
BRANCH="${BRANCH:-main}"

# Repo name for ArgoCD (when adding to ArgoCD)
if [ -z "$REPO_NAME" ]; then
    REPO_NAME=$(echo "$REPO_URL" | sed -E 's|.*github.com[:/]([^/]+)/([^/]+)(\.git)?$|\2|' | sed 's|\.git$||')
fi

# Resolve private key path
GITHUB_PRIVATE_KEY_PATH="${GITHUB_PRIVATE_KEY_PATH/#\~/$HOME}"
if [[ "$GITHUB_PRIVATE_KEY_PATH" != /* ]]; then
    if [ -f "$SCRIPT_DIR/$GITHUB_PRIVATE_KEY_PATH" ]; then
        GITHUB_PRIVATE_KEY_PATH="$SCRIPT_DIR/$GITHUB_PRIVATE_KEY_PATH"
    elif [ -f "$GITHUB_PRIVATE_KEY_PATH" ]; then
        GITHUB_PRIVATE_KEY_PATH="$(cd "$(dirname "$GITHUB_PRIVATE_KEY_PATH")" && pwd)/$(basename "$GITHUB_PRIVATE_KEY_PATH")"
    fi
fi
if [ ! -f "$GITHUB_PRIVATE_KEY_PATH" ]; then
    log_error "Private key not found: $GITHUB_PRIVATE_KEY_PATH"
    exit 1
fi

# Prerequisites: jq
if ! command -v jq &>/dev/null; then
    log_error "jq is required. Install with: brew install jq (macOS) or apt install jq (Linux)"
    exit 1
fi

if [ ! -d "$CHART_SOURCE" ]; then
    log_error "Predator chart not found at: $CHART_SOURCE (run from BharatMLStack repo quick-start/)"
    exit 1
fi

# Generate GitHub App installation token (JWT + API)
get_github_installation_token() {
    local app_id="$1"
    local installation_id="$2"
    local key_path="$3"
    local now exp header payload header_b64 payload_b64 signed_content sig_b64 jwt token
    now=$(date +%s)
    exp=$((now + 600))
    header='{"alg":"RS256","typ":"JWT"}'
    payload=$(jq -n --arg iat "$now" --arg exp "$exp" --arg iss "$app_id" '{iat: ($iat|tonumber), exp: ($exp|tonumber), iss: $iss}')
    header_b64=$(echo -n "$header" | base64 2>/dev/null | tr -d '\n' | tr '+/' '-_' | tr -d '=')
    payload_b64=$(echo -n "$payload" | base64 2>/dev/null | tr -d '\n' | tr '+/' '-_' | tr -d '=')
    signed_content="${header_b64}.${payload_b64}"
    sig_b64=$(echo -n "$signed_content" | openssl dgst -sha256 -sign "$key_path" 2>/dev/null | base64 2>/dev/null | tr -d '\n' | tr '+/' '-_' | tr -d '=')
    jwt="${signed_content}.${sig_b64}"
    token=$(curl -s -X POST \
        -H "Authorization: Bearer ${jwt}" \
        -H "Accept: application/vnd.github+json" \
        "https://api.github.com/app/installations/${installation_id}/access_tokens" | jq -r '.token')
    if [ -z "$token" ] || [ "$token" = "null" ]; then
        return 1
    fi
    echo -n "$token"
}

# 1) Populate GitHub repo: clone, copy 1.0.0 + prd/applications + configs/services, commit & push
log_info "Fetching GitHub App installation token..."
TOKEN=$(get_github_installation_token "$GITHUB_APP_ID" "$GITHUB_INSTALLATION_ID" "$GITHUB_PRIVATE_KEY_PATH") || {
    log_error "Failed to get installation token. Check App ID, Installation ID, and private key."
    exit 1
}

REPO_URL_AUTHED=""
if [[ "$REPO_URL" =~ ^https://github\.com/(.+)$ ]]; then
    REPO_URL_AUTHED="https://x-access-token:${TOKEN}@github.com/${BASH_REMATCH[1]}"
else
    log_error "Unsupported repo URL (expected https://github.com/...): $REPO_URL"
    exit 1
fi

TMPDIR=$(mktemp -d 2>/dev/null || mktemp -d -t predator-github)
trap "rm -rf '$TMPDIR'" EXIT

log_info "Cloning repository (branch: $BRANCH)..."
if ! git clone --branch "$BRANCH" --depth 1 "$REPO_URL_AUTHED" "$TMPDIR/repo" 2>/dev/null; then
    log_warning "Clone failed (branch may not exist). Trying default branch..."
    git clone --depth 1 "$REPO_URL_AUTHED" "$TMPDIR/repo" 2>/dev/null || {
        log_error "Failed to clone repository."
        exit 1
    }
    cd "$TMPDIR/repo"
    BRANCH=$(git rev-parse --abbrev-ref HEAD)
    cd - >/dev/null
fi

REPO_DIR="$TMPDIR/repo"
if [ -d "$REPO_DIR/1.0.0" ]; then
    log_info "1.0.0/ already exists; updating..."
    rm -rf "$REPO_DIR/1.0.0"
fi
cp -r "$CHART_SOURCE" "$REPO_DIR/1.0.0"

mkdir -p "$REPO_DIR/prd/applications"
[ ! -f "$REPO_DIR/prd/applications/.gitkeep" ] && touch "$REPO_DIR/prd/applications/.gitkeep"

if [ -d "$CONFIGS_SERVICES_SOURCE" ]; then
    log_info "Copying configs/services..."
    mkdir -p "$REPO_DIR/configs"
    [ -d "$REPO_DIR/configs/services" ] && rm -rf "$REPO_DIR/configs/services"
    cp -r "$CONFIGS_SERVICES_SOURCE" "$REPO_DIR/configs/services"
else
    log_warning "horizon/configs/services not found; skipping."
fi

cd "$REPO_DIR"
git config user.email "horizon-bot@predator-setup.local"
git config user.name "Predator Setup"
git add 1.0.0 prd/applications
[ -d configs/services ] && git add configs/services
if git diff --cached --quiet; then
    log_info "No changes to commit (repo already up to date)."
else
    git commit -m "Add Predator Helm chart 1.0.0, prd/applications, configs/services (via setup-predator-github.sh)"
    if git push origin "$BRANCH" 2>/dev/null; then
        log_success "Pushed to $REPO_URL (branch: $BRANCH)."
    else
        log_warning "Push failed. Push manually: cd <repo> && git push origin $BRANCH"
        exit 1
    fi
fi
cd - >/dev/null

log_success "GitHub repo populated: 1.0.0/, prd/applications/, configs/services/"

# 2) Optional: add repository to ArgoCD (via flags or interactive prompt)
if [ -z "$ARGOCD_SERVER" ] || [ -z "$ARGOCD_TOKEN" ]; then
    echo ""
    read -p "Add this repository to ArgoCD? (y/n): " add_argocd
    if [[ "$add_argocd" =~ ^[Yy]$ ]]; then
        read -p "ArgoCD server URL (e.g. https://localhost:8087 or localhost:8087): " ARGOCD_SERVER
        read -p "ArgoCD auth token: " ARGOCD_TOKEN
    fi
fi

if [ -n "$ARGOCD_SERVER" ] && [ -n "$ARGOCD_TOKEN" ]; then
    if ! command -v argocd &>/dev/null; then
        log_warning "ArgoCD CLI not found; skipping ArgoCD repo add. Install argocd and run:"
        echo "  argocd repo add $REPO_URL --name $REPO_NAME --type git \\"
        echo "    --github-app-id $GITHUB_APP_ID --github-app-installation-id $GITHUB_INSTALLATION_ID \\"
        echo "    --github-app-private-key-path $GITHUB_PRIVATE_KEY_PATH --insecure"
        exit 0
    fi
    # Normalize server (argocd login expects host:port or URL)
    ARGOCD_HOST="$ARGOCD_SERVER"
    [[ "$ARGOCD_HOST" =~ ^https?:// ]] && ARGOCD_HOST="${ARGOCD_HOST#*://}"
    log_info "Adding repository to ArgoCD at $ARGOCD_HOST..."
    if argocd login "$ARGOCD_HOST" --auth-token "$ARGOCD_TOKEN" --insecure 2>/dev/null; then
        if argocd repo add "$REPO_URL" \
            --name "$REPO_NAME" \
            --type git \
            --github-app-id "$GITHUB_APP_ID" \
            --github-app-installation-id "$GITHUB_INSTALLATION_ID" \
            --github-app-private-key-path "$GITHUB_PRIVATE_KEY_PATH" \
            --insecure; then
            log_success "Repository added to ArgoCD."
        else
            log_warning "argocd repo add failed. Add manually (see GITHUB_REPO_SETUP.md)."
        fi
    else
        log_warning "ArgoCD login failed. Add repo manually with token (see GITHUB_REPO_SETUP.md)."
    fi
else
    log_info "To add this repo to ArgoCD later, run with --argocd-server and --argocd-token or run again and choose 'y' when prompted."
fi
