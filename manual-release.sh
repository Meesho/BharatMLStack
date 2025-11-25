#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Available modules for release
AVAILABLE_MODULES=("horizon" "trufflebox-ui" "numerix" "online-feature-store" "go-sdk" "py-sdk" "helix-client")

# Python SDK subdirectories
PY_SDK_MODULES=("bharatml_commons" "grpc_feature_client" "spark_feature_push_client")

# Global variable to store selected modules
SELECTED_MODULES=()

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_header() {
    echo -e "${CYAN}🚀 $1${NC}"
}

# Function to get current branch
get_current_branch() {
    git rev-parse --abbrev-ref HEAD
}

# Function to get version from VERSION file
get_version_from_file() {
    local version_file=$1
    if [[ -f "$version_file" ]]; then
        cat "$version_file" | tr -d '\n' | tr -d ' '
    else
        echo ""
    fi
}

# Function to get version from toml file
get_version_from_toml() {
    local toml_file=$1
    if [[ -f "$toml_file" ]]; then
        # Extract version from pyproject.toml using grep and sed
        grep -E "^version\s*=" "$toml_file" | sed -E 's/^version\s*=\s*"([^"]+)".*/\1/' | head -1
    else
        echo ""
    fi
}



# Function to get pre-release version with commit SHA (Docker/Go format)
get_prerelease_version_with_sha() {
    local base_version=$1
    local release_type=$2
    
    # Remove 'v' prefix if present
    base_version=${base_version#v}
  
    # Remove any existing pre-release suffixes to avoid double-suffixing
    base_version=$(echo "$base_version" | sed -E 's/-[a-z]+-[a-f0-9]+$//')
    base_version=$(echo "$base_version" | sed -E 's/-[a-z]+\.[0-9]+$//')
    # Get last commit SHA (6 characters)
    local commit_sha=$(git rev-parse --short=6 HEAD)
    
    local prerelease_suffix=""
    case "$release_type" in
        "beta")
            prerelease_suffix="beta"
            ;;
        "alpha")
            prerelease_suffix="alpha"
            ;;
        *)
            print_error "Invalid pre-release type: $release_type"
            return 1
            ;;
    esac
    
    echo "${base_version}-${prerelease_suffix}-${commit_sha}"
}

# Function to get Python PEP 440 pre-release version
get_python_prerelease_version() {
    local base_version=$1
    local release_type=$2
    
    # Remove 'v' prefix if present
    base_version=${base_version#v}
    
    # Remove any existing pre-release suffixes to avoid double-suffixing
    base_version=$(echo "$base_version" | sed -E 's/-[a-z]+-[a-f0-9]+$//')
    base_version=$(echo "$base_version" | sed -E 's/-[a-z]+\.[0-9]+$//')
    base_version=$(echo "$base_version" | sed -E 's/[a-z][0-9]+\+[a-f0-9]+$//')
    
    
    local prerelease_suffix=""
    case "$release_type" in
        "beta")
            prerelease_suffix="b0"
            ;;
        "alpha")
            prerelease_suffix="a0.dev"
            ;;
        *)
            print_error "Invalid pre-release type: $release_type"
            return 1
            ;;
    esac
    
    echo "${base_version}${prerelease_suffix}"
}





# Function to update VERSION file
update_version_file() {
    local version_file=$1
    local new_version=$2
    
    echo "v${new_version}" > "$version_file"
    print_success "Updated $version_file to v${new_version}"
}

# Function to trigger GitHub workflow
trigger_workflow() {
    local module=$1
    local release_type=$2
    local version=$3
    local branch=$4
    local workflow_file=""
    
    case "$module" in
        "horizon")
            workflow_file="release-horizon.yml"
            ;;
        "trufflebox-ui")
            workflow_file="release-trufflebox-ui.yml"
            ;;
        "numerix")
            workflow_file="release-numerix.yml"
            ;;
        "online-feature-store")
            workflow_file="release-online-feature-store.yml"
            ;;
        "go-sdk")
            workflow_file="release-go-sdk.yml"
            ;;
        "py-sdk")
            workflow_file="release-py-sdk.yml"
            ;;
        "helix-client")
            workflow_file="release-helix-client.yml"
            ;;
        *)
            print_error "Unknown module: $module"
            return 1
            ;;
    esac
    
    print_info "Triggering GitHub release workflow: $workflow_file for $module"
    
    # Determine release flags
    local is_beta="false"
    local is_alpha="false"
    case "$release_type" in
        "beta")
            is_beta="true"
            ;;
        "alpha")
            is_alpha="true"
            ;;
    esac
    
    # Use GitHub CLI to trigger workflow
    if command -v gh &> /dev/null; then
        print_info "Running: gh workflow run $workflow_file -f version=\"$version\" -f is_beta=\"$is_beta\" -f is_alpha=\"$is_alpha\""
        gh workflow run "$workflow_file" -f version="$version" -f is_beta="$is_beta" -f is_alpha="$is_alpha" -f branch="$branch"
        print_success "Release workflow $workflow_file triggered successfully"
        local repo_path=$(gh repo view --json owner,name -q '.owner.login + "/" + .name' 2>/dev/null || echo "your-repo")
        print_info "Monitor the workflow at: https://github.com/${repo_path}/actions"
    else
        print_warning "GitHub CLI not found. Please install 'gh' or trigger workflow manually:"
        print_info "gh workflow run $workflow_file -f version=\"$version\" -f is_beta=\"$is_beta\" -f is_alpha=\"$is_alpha\""
        print_info "Or go to GitHub Actions tab and manually trigger: $workflow_file"
    fi
}


# Function to validate branch for release type
validate_branch_for_release() {
    local branch=$1
    local release_type=$2
    
    case "$release_type" in
        "beta")
            if [[ "$branch" != "develop" ]]; then
                print_error "Beta releases can only be made from 'develop' branch. Current branch: $branch"
                return 1
            fi
            ;;
        "std-release")
            if [[ "$branch" != "main" && "$branch" != "master" && ! "$branch" =~ ^release/ ]]; then
                print_error "Standard releases can only be made from 'main', 'master', or 'release/' branches. Current branch: $branch"
                return 1
            fi
            ;;
        "alpha")
            if [[ ! "$branch" =~ ^(feat/|fix/|feat-nbc/) ]]; then
                print_error "Alpha releases can only be made from 'feat/', 'fix/', or 'feat-nbc/' branches. Current branch: $branch"
                return 1
            fi
            ;;
        *)
            print_error "Unknown release type: $release_type"
            return 1
            ;;
    esac
    return 0
}

# Function to select release type
select_release_type() {
    echo "" >&2
    print_header "Select Release Type" >&2
    
    echo "1) Alpha Release (Python: x.y.za0+<commit-sha>, Docker/Go: x.y.z-alpha-<commit-sha>)" >&2
    echo "2) Beta Release (Python: x.y.zb0+<commit-sha>, Docker/Go: x.y.z-beta-<commit-sha>)" >&2
    echo "3) Standard Release (x.y.z)" >&2
    echo "" >&2
    
    while true; do
        echo -n "Enter your choice (1-3): " >&2
        read choice
        case $choice in
            1)
                echo "alpha"
                return
                ;;
            2)
                echo "beta"
                return
                ;;
            3)
                echo "std-release"
                return
                ;;
            *)
                print_error "Invalid choice. Please enter 1, 2, or 3." >&2
                ;;
        esac
    done
}

# Function to select directories
select_directories() {
    echo ""
    print_header "Select Modules to Release"
    
    local selected_modules=()
    local i=1
    
    echo "Available modules:"
    for module in "${AVAILABLE_MODULES[@]}"; do
        echo "$i) $module"
        i=$((i + 1))
    done
    echo ""
    
    print_info "Enter module numbers separated by spaces (e.g., '1 3 5') or 'all' for all modules:"
    echo -n "Selection: "
    read selection
    
    if [[ "$selection" == "all" ]]; then
        selected_modules=("${AVAILABLE_MODULES[@]}")
    else
        for num in $selection; do
            if [[ "$num" =~ ^[0-9]+$ ]] && [[ "$num" -ge 1 ]] && [[ "$num" -le ${#AVAILABLE_MODULES[@]} ]]; then
                selected_modules+=("${AVAILABLE_MODULES[$((num-1))]}")
            else
                print_warning "Invalid selection: $num"
            fi
        done
    fi
    
    if [[ ${#selected_modules[@]} -eq 0 ]]; then
        print_error "No valid modules selected"
        exit 1
    fi
    
    # Store results in a global variable instead of stdout
    SELECTED_MODULES=("${selected_modules[@]}")
}



# Function to process module release
process_module_release() {
    local module=$1
    local release_type=$2
    local branch=$3
    
    print_header "Processing $module"
    
    local final_version=""
    
    if [[ "$module" == "py-sdk" ]]; then
        # Handle Python SDK modules - use VERSION files (pyproject.toml reads from them via hatch)
        local first_version=""
        for py_module in "${PY_SDK_MODULES[@]}"; do
            local version_file="py-sdk/$py_module/VERSION"
            local current_version=$(get_version_from_file "$version_file")
            local new_version=""
            
            if [[ -z "$current_version" ]]; then
                print_error "Could not read version from $version_file"
                continue
            fi
            
            print_info "Current version for $py_module: $current_version"
            
            if [[ "$release_type" == "beta" || "$release_type" == "alpha" ]]; then
                new_version=$(get_python_prerelease_version "$current_version" "$release_type")
                print_info "Calculated release version for $py_module: $new_version"
                print_info "VERSION file remains unchanged: $current_version"
            else
                # Standard release - use existing version as-is
                new_version=${current_version#v}
                print_info "Using existing version for standard release: $new_version"
            fi
            
            # Use the first module's version for the workflow trigger
            if [[ -z "$first_version" ]]; then
                first_version="$new_version"
            fi
        done
        final_version="$first_version"
    else
        # Handle other modules
        local version_file="$module/VERSION"
        local current_version=$(get_version_from_file "$version_file")
        local new_version=""
        
        if [[ -z "$current_version" ]]; then
            print_error "Could not read version from $version_file"
            return 1
        fi
        
        print_info "Current version for $module: $current_version"
        
        if [[ "$release_type" == "beta" || "$release_type" == "alpha" ]]; then
            new_version=$(get_prerelease_version_with_sha "$current_version" "$release_type")
            print_info "Calculated release version for $module: v$new_version"
            print_info "VERSION file remains unchanged: $current_version"
        else
            # Standard release - use existing version as-is
            new_version=${current_version#v}
            print_info "Using existing version for standard release: v$new_version"
        fi
        final_version="v$new_version"
    fi
    
    # Trigger GitHub workflow
    trigger_workflow "$module" "$release_type" "$final_version" "$branch"
    
    print_success "Completed processing $module"
    echo ""
}

# Main function
main() {
    print_header "BharatMLStack Manual Release Tool"
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_error "Not in a git repository"
        exit 1
    fi
    
    # Show current branch context
    local current_branch=$(get_current_branch)
    print_info "Current branch: $current_branch"
    
    # Check if branch is clean
    if [[ -n $(git status --porcelain) ]]; then
        print_warning "Working directory is not clean. Consider committing changes before release."
        echo -n "Continue anyway? (y/N): "
        read -n 1 -r REPLY
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Release cancelled"
            exit 0
        fi
    fi
    
    # Select release type
    local release_type=$(select_release_type)
    print_success "Selected release type: $release_type"
    
    # Validate branch for the selected release type
    if ! validate_branch_for_release "$current_branch" "$release_type"; then
        exit 1
    fi
    
    # Note: For standard releases, we use the existing version as-is from VERSION files
    # Alpha/beta releases calculate -<type>-<commit-sha> suffix but don't update VERSION files
    
    # Select directories
    select_directories
    local selected_modules=("${SELECTED_MODULES[@]}")
    
    print_success "Selected modules:"
    for module in "${selected_modules[@]}"; do
        echo "  - $module"
    done
    
    # Confirmation
    echo ""
    print_warning "Release Summary:"
    print_info "Branch: $current_branch"
    print_info "Release Type: $release_type"
    if [[ "$release_type" == "std-release" ]]; then
        print_info "Version Strategy: Use existing versions from files as-is"
    else
        print_info "Version Strategy: Python uses PEP 440 format (x.y.z${release_type:0:1}0+<sha>), Docker/Go uses -${release_type}-<sha>"
        print_info "Note: VERSION files remain unchanged for alpha/beta releases"
    fi
    for module in "${selected_modules[@]}"; do
        print_info "Module: $module"
    done
    
    echo ""
    echo -n "Proceed with release? (y/N): "
    read -n 1 -r REPLY
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Release cancelled"
        exit 0
    fi
    
    # Process each module
    print_header "Starting Release Process"
    for module in "${selected_modules[@]}"; do
        process_module_release "$module" "$release_type" "$current_branch"
    done
    
    # Final summary
    print_success "🎉 Release process completed successfully!"
    print_info "Don't forget to:"
    print_info "1. Review the triggered workflows in GitHub Actions"
    if [[ "$release_type" == "std-release" ]]; then
        print_info "2. Commit any version file changes if you updated them manually"
        print_info "3. Create PR or push changes if needed"
    else
        print_info "2. VERSION files remain unchanged for alpha/beta releases"
        print_info "3. Release artifacts will use calculated ${release_type} versions with commit SHA"
    fi
}

# Run main function
main "$@" 