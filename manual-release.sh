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
AVAILABLE_MODULES=("horizon" "trufflebox-ui" "online-feature-store" "go-sdk" "py-sdk")

# Python SDK subdirectories
PY_SDK_MODULES=("bharatml_commons" "grpc_feature_client" "spark_feature_push_client")

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



# Function to get next pre-release version (beta or alpha)
get_next_prerelease_version() {
    local base_version=$1
    local module=$2
    local release_type=$3
    
    # Remove 'v' prefix if present
    base_version=${base_version#v}
    
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
    
    local prerelease_num=1
    while check_prerelease_version_exists "$base_version" "$prerelease_num" "$module" "$prerelease_suffix"; do
        prerelease_num=$((prerelease_num + 1))
    done
    
    echo "${base_version}-${prerelease_suffix}.${prerelease_num}"
}

# Function to check if pre-release version exists
check_prerelease_version_exists() {
    local base_version=$1
    local prerelease_num=$2
    local module=$3
    local prerelease_suffix=$4
    
    # For Python packages, check PyPI
    if [[ "$module" == py-sdk/* ]]; then
        local package_name=$(basename "$module" | tr '_' '-')
        # Use pip index to check if version exists (this is a simplified check)
        # In practice, you might want to use PyPI API
        return 1  # For now, assume it doesn't exist
    fi
    
    # For Docker images, check registry
    if [[ "$module" == "horizon" ]] || [[ "$module" == "online-feature-store" ]] || [[ "$module" == "trufflebox-ui" ]]; then
        # Check if Docker tag exists (simplified)
        return 1  # For now, assume it doesn't exist
    fi
    
    # For Go SDK, check Git tags
    if [[ "$module" == "go-sdk" ]]; then
        git tag -l | grep -q "go-sdk/${base_version}-${prerelease_suffix}.${prerelease_num}" && return 0 || return 1
    fi
    
    return 1
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
    local workflow_file=""
    
    case "$module" in
        "horizon")
            workflow_file="release-horizon.yml"
            ;;
        "trufflebox-ui")
            workflow_file="release-trufflebox-ui.yml"
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
        gh workflow run "$workflow_file" -f version="$version" -f is_beta="$is_beta" -f is_alpha="$is_alpha"
        print_success "Release workflow $workflow_file triggered successfully"
        print_info "Monitor the workflow at: https://github.com/$(gh repo view --json owner,name -q '.owner.login + \"/\" + .name')/actions"
    else
        print_warning "GitHub CLI not found. Please install 'gh' or trigger workflow manually:"
        print_info "gh workflow run $workflow_file -f version=\"$version\" -f is_beta=\"$is_beta\" -f is_alpha=\"$is_alpha\""
        print_info "Or go to Actions tab in GitHub and manually trigger: $workflow_file"
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

# Function to select release type based on current branch
select_release_type() {
    local current_branch=$1
    echo ""
    print_header "Select Release Type"
    
    # Show available options based on current branch
    local available_options=()
    local option_map=()
    local option_num=1
    
    # Beta releases from develop
    if [[ "$current_branch" == "develop" ]]; then
        available_options+=("$option_num) Beta Release (x.y.z-beta.N) - from develop branch")
        option_map["$option_num"]="beta"
        option_num=$((option_num + 1))
    fi
    
    # Standard releases from main/master/release/*
    if [[ "$current_branch" == "main" || "$current_branch" == "master" || "$current_branch" =~ ^release/ ]]; then
        available_options+=("$option_num) Standard Release (x.y.z) - from $current_branch branch")
        option_map["$option_num"]="std-release"
        option_num=$((option_num + 1))
    fi
    
    # Alpha releases from feat/fix/feat-nbc branches
    if [[ "$current_branch" =~ ^(feat/|fix/|feat-nbc/) ]]; then
        available_options+=("$option_num) Alpha Release (x.y.z-alpha.N) - from $current_branch branch")
        option_map["$option_num"]="alpha"
        option_num=$((option_num + 1))
    fi
    
    if [[ ${#available_options[@]} -eq 0 ]]; then
        print_error "No valid release types available for branch '$current_branch'"
        print_info "Valid branches for releases:"
        print_info "  • develop: Beta releases"
        print_info "  • main/master/release/*: Standard releases"
        print_info "  • feat/*/fix/*/feat-nbc/*: Alpha releases"
        exit 1
    fi
    
    # Show available options
    for option in "${available_options[@]}"; do
        echo "$option"
    done
    echo ""
    
    # If only one option, auto-select it
    if [[ ${#available_options[@]} -eq 1 ]]; then
        print_info "Only one release type available - auto-selecting option 1"
        echo "${option_map[1]}"
        return
    fi
    
    while true; do
        read -p "Enter your choice (1-${#available_options[@]}): " choice
        if [[ -n "${option_map[$choice]}" ]]; then
            echo "${option_map[$choice]}"
            return
        else
            print_error "Invalid choice. Please enter a number between 1 and ${#available_options[@]}."
        fi
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
    read -p "Selection: " selection
    
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
    
    printf '%s\n' "${selected_modules[@]}"
}



# Function to process module release
process_module_release() {
    local module=$1
    local release_type=$2
    
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
                new_version=$(get_next_prerelease_version "$current_version" "py-sdk/$py_module" "$release_type")
                print_info "New version for $py_module: v$new_version"
                update_version_file "$version_file" "$new_version"
            else
                # Standard release - use existing version as-is
                new_version=${current_version#v}
                print_info "Using existing version for standard release: v$new_version"
            fi
            
            # Use the first module's version for the workflow trigger
            if [[ -z "$first_version" ]]; then
                first_version="v$new_version"
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
            new_version=$(get_next_prerelease_version "$current_version" "$module" "$release_type")
            print_info "New version for $module: v$new_version"
            update_version_file "$version_file" "$new_version"
        else
            # Standard release - use existing version as-is
            new_version=${current_version#v}
            print_info "Using existing version for standard release: v$new_version"
        fi
        final_version="v$new_version"
    fi
    
    # Trigger GitHub workflow
    trigger_workflow "$module" "$release_type" "$final_version"
    
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
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Release cancelled"
            exit 0
        fi
    fi
    
    # Select release type based on current branch
    local release_type=$(select_release_type "$current_branch")
    print_success "Selected release type: $release_type"
    
    # Validate branch for the selected release type (double-check)
    if ! validate_branch_for_release "$current_branch" "$release_type"; then
        exit 1
    fi
    
    # Note: For standard releases, we use the existing version as-is from VERSION/toml files
    # Only alpha/beta releases get auto-incremented with .N suffix
    
    # Select directories
    local selected_modules
    mapfile -t selected_modules < <(select_directories)
    
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
        print_info "Version Strategy: Auto-increment ${release_type}.N suffix"
    fi
    for module in "${selected_modules[@]}"; do
        print_info "Module: $module"
    done
    
    echo ""
    read -p "Proceed with release? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Release cancelled"
        exit 0
    fi
    
    # Process each module
    print_header "Starting Release Process"
    for module in "${selected_modules[@]}"; do
        process_module_release "$module" "$release_type"
    done
    
    # Final summary
    print_success "🎉 Release process completed successfully!"
    print_info "Don't forget to:"
    print_info "1. Review the triggered workflows in GitHub Actions"
    print_info "2. Commit the version file changes if not already committed"
    print_info "3. Create PR or push changes if needed"
}

# Run main function
main "$@" 