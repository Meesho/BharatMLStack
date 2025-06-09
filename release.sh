#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Directories with release.sh scripts
RELEASE_DIRS=("horizon" "trufflebox-ui" "online-feature-store")

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

# Function to check if directory has changes
has_changes() {
    local dir=$1
    local base_ref=${2:-"HEAD~1"}  # Default to last commit
    
    if [ ! -d "$dir" ]; then
        print_warning "Directory $dir does not exist"
        return 1
    fi
    
    # Check if there are changes in the directory compared to base reference
    local changes=$(git diff --name-only "$base_ref" HEAD -- "$dir/")
    
    if [ -n "$changes" ]; then
        print_info "Changes detected in $dir:"
        echo "$changes" | sed 's/^/  - /'
        return 0
    else
        return 1
    fi
}

# Function to run release script in directory
run_release() {
    local dir=$1
    
    if [ ! -f "$dir/release.sh" ]; then
        print_error "Release script not found in $dir/"
        return 1
    fi
    
    if [ ! -x "$dir/release.sh" ]; then
        print_info "Making $dir/release.sh executable"
        chmod +x "$dir/release.sh"
    fi
    
    print_info "Running release script in $dir/"
    echo "=========================================="
    
    # Change to directory and run release script
    (cd "$dir" && ./release.sh)
    
    if [ $? -eq 0 ]; then
        print_success "Release completed successfully for $dir"
    else
        print_error "Release failed for $dir"
        return 1
    fi
    
    echo "=========================================="
}

# Main execution
main() {
    print_info "BharatMLStack Release Manager"
    echo ""
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_error "Not in a git repository"
        exit 1
    fi
    
    # Parse command line arguments
    BASE_REF="HEAD~1"
    FORCE_ALL=false
    DRY_RUN=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --base-ref)
                BASE_REF="$2"
                shift 2
                ;;
            --force-all)
                FORCE_ALL=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --base-ref REF    Compare changes against REF (default: HEAD~1)"
                echo "  --force-all       Run release for all directories regardless of changes"
                echo "  --dry-run         Show what would be released without actually running"
                echo "  --help           Show this help message"
                echo ""
                echo "Examples:"
                echo "  $0                          # Release changed directories since last commit"
                echo "  $0 --base-ref main         # Release changed directories since main branch"
                echo "  $0 --force-all             # Release all directories"
                echo "  $0 --dry-run               # Show what would be released"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
    
    print_info "Checking for changes since: $BASE_REF"
    echo ""
    
    # Track directories that need release
    DIRS_TO_RELEASE=()
    
    # Check each directory for changes
    for dir in "${RELEASE_DIRS[@]}"; do
        if [ "$FORCE_ALL" = true ]; then
            print_info "Force mode: Adding $dir to release queue"
            DIRS_TO_RELEASE+=("$dir")
        elif has_changes "$dir" "$BASE_REF"; then
            DIRS_TO_RELEASE+=("$dir")
        else
            print_info "No changes in $dir - skipping"
        fi
        echo ""
    done
    
    # Show summary
    if [ ${#DIRS_TO_RELEASE[@]} -eq 0 ]; then
        print_info "No directories need to be released"
        exit 0
    fi
    
    print_info "Directories to be released:"
    for dir in "${DIRS_TO_RELEASE[@]}"; do
        echo "  - $dir"
    done
    echo ""
    
    if [ "$DRY_RUN" = true ]; then
        print_info "Dry run mode - no actual releases will be performed"
        exit 0
    fi
    
    # Confirm before proceeding
    if [ "$FORCE_ALL" != true ]; then
        read -p "Proceed with releases? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Release cancelled by user"
            exit 0
        fi
    fi
    
    # Run releases
    FAILED_RELEASES=()
    for dir in "${DIRS_TO_RELEASE[@]}"; do
        print_info "Starting release for $dir"
        if ! run_release "$dir"; then
            FAILED_RELEASES+=("$dir")
        fi
        echo ""
    done
    
    # Final summary
    echo "=========================================="
    if [ ${#FAILED_RELEASES[@]} -eq 0 ]; then
        print_success "All releases completed successfully!"
    else
        print_error "Some releases failed:"
        for dir in "${FAILED_RELEASES[@]}"; do
            echo "  - $dir"
        done
        exit 1
    fi
}

# Run main function with all arguments
main "$@" 