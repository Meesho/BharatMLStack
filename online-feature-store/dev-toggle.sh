#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_FILE="$SCRIPT_DIR/.dev-toggle-state"
GO_MOD_FILE="$SCRIPT_DIR/go.mod"
GO_MOD_APPEND_FILE="$SCRIPT_DIR/.go.mod.appended"

INTERNAL_REPO_URL="https://github.com/Meesho/BharatMLStack-internal-configs"
INTERNAL_REPO_DIR="$SCRIPT_DIR/.internal-configs"
INTERNAL_OFS_DIR="$INTERNAL_REPO_DIR/online-feature-store"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_usage() {
    echo "Usage: $0 [enable|disable|status|update]"
    echo ""
    echo "Commands:"
    echo "  enable   - Enable development mode (clone internal repo, copy files, update go.mod)"
    echo "  disable  - Disable development mode (remove copied files and go.mod changes)"
    echo "  status   - Show current development mode status"
    echo "  update   - Update internal configs (pull latest from internal repo)"
    echo ""
    exit 1
}

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

check_status() {
    log_info "=========================================="
    log_info "Development Mode Status"
    log_info "=========================================="
    
    if [ -f "$STATE_FILE" ]; then
        echo ""
        echo "Status: ENABLED"
        echo ""
        
        # Count files
        local file_count=$(grep -c "^FILE:" "$STATE_FILE" || echo "0")
        echo "Files copied: $file_count"
        
        # Show files
        if [ "$file_count" -gt 0 ]; then
            echo ""
            echo "Copied files:"
            grep "^FILE:" "$STATE_FILE" | sed 's/^FILE:/  ✓ /'
        fi
        
        echo ""
        echo "go.mod modified: $([ -f "$GO_MOD_APPEND_FILE" ] && echo "YES" || echo "NO")"
        
        # Show replace directives if they exist
        if [ -f "$GO_MOD_APPEND_FILE" ]; then
            local replace_count=$(wc -l < "$GO_MOD_APPEND_FILE")
            echo "Replace directives: $replace_count"
        fi
        
        # Show internal repo status
        if [ -d "$INTERNAL_REPO_DIR/.git" ]; then
            echo ""
            echo "Internal repo:"
            local commit_hash=$(cd "$INTERNAL_REPO_DIR" && git rev-parse --short HEAD 2>/dev/null || echo "unknown")
            local branch=$(cd "$INTERNAL_REPO_DIR" && git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
            echo "  Branch: $branch"
            echo "  Commit: $commit_hash"
        fi
        
        echo ""
        echo "State file: $STATE_FILE"
        echo ""
        
        return 0
    else
        echo ""
        echo "Status: DISABLED"
        echo ""
        echo "No active development mode configuration found."
        echo "Run './dev-toggle.sh enable' to enable development mode."
        echo ""
        return 1
    fi
}

clone_or_update_internal_repo() {
    if [ -d "$INTERNAL_REPO_DIR/.git" ]; then
        log_info "Internal configs repo already exists, updating..."
        log_debug "Repository path: $INTERNAL_REPO_DIR"
        cd "$INTERNAL_REPO_DIR"
        
        # Fetch latest changes
        log_info "Fetching latest changes from origin..."
        git fetch origin
        log_debug "Fetch completed"
        
        # Determine the default branch (main or master)
        log_debug "Determining default branch..."
        default_branch=$(git remote show origin | grep 'HEAD branch' | cut -d' ' -f5)
        if [ -z "$default_branch" ]; then
            log_debug "Could not detect default branch via remote show, trying fallback..."
            # Fallback: try main first, then master
            if git show-ref --verify --quiet refs/remotes/origin/main; then
                default_branch="main"
                log_debug "Found refs/remotes/origin/main"
            elif git show-ref --verify --quiet refs/remotes/origin/master; then
                default_branch="master"
                log_debug "Found refs/remotes/origin/master"
            else
                log_error "Could not determine default branch (main or master not found)"
                cd "$SCRIPT_DIR"
                exit 1
            fi
        fi
        
        log_info "Default branch detected: $default_branch"
        
        # Checkout the default branch if we're on a different branch
        current_branch=$(git rev-parse --abbrev-ref HEAD)
        log_debug "Current branch: $current_branch"
        if [ "$current_branch" != "$default_branch" ]; then
            log_info "Switching from $current_branch to $default_branch"
            git checkout "$default_branch"
            log_debug "Branch switch completed"
        else
            log_debug "Already on $default_branch, no branch switch needed"
        fi
        
        # Reset to latest
        log_info "Resetting to origin/$default_branch..."
        git reset --hard "origin/$default_branch"
        local commit_hash=$(git rev-parse --short HEAD)
        log_info "Updated to commit: $commit_hash"
        
        cd "$SCRIPT_DIR"
    else
        log_info "Internal configs repo not found, cloning..."
        log_debug "Cloning from: $INTERNAL_REPO_URL"
        log_debug "Destination: $INTERNAL_REPO_DIR"
        rm -rf "$INTERNAL_REPO_DIR"
        git clone "$INTERNAL_REPO_URL" "$INTERNAL_REPO_DIR"
        local commit_hash=$(cd "$INTERNAL_REPO_DIR" && git rev-parse --short HEAD)
        log_info "Clone completed at commit: $commit_hash"
    fi
}

enable_dev_mode() {
    log_info "=========================================="
    log_info "Starting development mode enablement"
    log_info "=========================================="
    
    if [ -f "$STATE_FILE" ]; then
        log_warn "Development mode is already enabled!"
        echo "Run '$0 disable' first to revert, or delete $STATE_FILE if state is corrupted."
        exit 1
    fi

    log_debug "State file: $STATE_FILE"
    log_debug "Working directory: $SCRIPT_DIR"

    # Clone or update the internal repo
    log_info "Step 1: Fetching internal configurations..."
    clone_or_update_internal_repo

    # Check if online-feature-store directory exists in internal repo
    log_info "Step 2: Validating internal repository structure..."
    log_debug "Looking for: $INTERNAL_OFS_DIR"
    if [ ! -d "$INTERNAL_OFS_DIR" ]; then
        log_error "online-feature-store directory not found in internal configs repo!"
        log_error "Expected path: $INTERNAL_OFS_DIR"
        exit 1
    fi
    log_debug "✓ online-feature-store directory found"

    # Initialize state file
    log_info "Step 3: Initializing state tracking..."
    echo "# Dev toggle state - DO NOT EDIT MANUALLY" > "$STATE_FILE"
    echo "# Generated on $(date)" >> "$STATE_FILE"
    log_debug "Created state file: $STATE_FILE"

    # Find and copy all .go files from internal repo
    log_info "Step 4: Searching for .go files to copy..."
    log_debug "Scanning directory: $INTERNAL_OFS_DIR"
    
    local copied_count=0
    local file_count=$(find "$INTERNAL_OFS_DIR" -name "*.go" -type f | wc -l)
    log_info "Found $file_count .go file(s) to copy"
    
    while IFS= read -r -d '' src_file; do
        # Get relative path from INTERNAL_OFS_DIR
        rel_path="${src_file#$INTERNAL_OFS_DIR/}"
        dest_file="$SCRIPT_DIR/$rel_path"
        dest_dir="$(dirname "$dest_file")"

        # Create destination directory if it doesn't exist
        if [ ! -d "$dest_dir" ]; then
            log_debug "Creating directory: $dest_dir"
            mkdir -p "$dest_dir"
        fi

        # Check if file already exists
        if [ -f "$dest_file" ]; then
            log_warn "File already exists, will be overwritten: $rel_path"
        fi

        # Copy the file
        log_debug "Copying: $src_file -> $dest_file"
        cp "$src_file" "$dest_file"
        log_info "✓ Copied: $rel_path"
        
        # Record in state file
        echo "FILE:$rel_path" >> "$STATE_FILE"
        ((copied_count++))
    done < <(find "$INTERNAL_OFS_DIR" -name "*.go" -type f -print0)

    log_info "Successfully copied $copied_count file(s)"

    # Check if there's a go.mod file in internal repo with replace directives
    log_info "Step 5: Processing go.mod modifications..."
    if [ -f "$INTERNAL_OFS_DIR/go.mod" ]; then
        log_debug "Found go.mod in internal configs: $INTERNAL_OFS_DIR/go.mod"
        
        # Extract lines that start with "replace " from internal go.mod
        if grep -E "^replace " "$INTERNAL_OFS_DIR/go.mod" > "$GO_MOD_APPEND_FILE"; then
            local replace_count=$(wc -l < "$GO_MOD_APPEND_FILE")
            log_info "Found $replace_count replace directive(s) to append"
            log_debug "Replace directives:"
            while IFS= read -r line; do
                log_debug "  $line"
            done < "$GO_MOD_APPEND_FILE"
            
            # Append to current go.mod
            log_info "Appending to $GO_MOD_FILE..."
            echo "" >> "$GO_MOD_FILE"
            echo "// Added by dev-toggle.sh - DO NOT EDIT" >> "$GO_MOD_FILE"
            cat "$GO_MOD_APPEND_FILE" >> "$GO_MOD_FILE"
            
            log_info "✓ Successfully appended replace directives to go.mod"
        else
            log_warn "No replace directives found in internal go.mod"
            rm -f "$GO_MOD_APPEND_FILE"
        fi
    else
        log_warn "No go.mod found in internal configs at: $INTERNAL_OFS_DIR/go.mod"
    fi

    # Run go mod tidy
    log_info "Step 6: Running go mod tidy..."
    log_debug "Changing to directory: $SCRIPT_DIR"
    cd "$SCRIPT_DIR"
    log_debug "Executing: go mod tidy"
    go mod tidy
    log_info "✓ go mod tidy completed successfully"

    log_info "=========================================="
    log_info "✓ Development mode enabled successfully!"
    log_info "=========================================="
    echo ""
    echo "Summary:"
    echo "  Files copied: $copied_count"
    echo "  go.mod updated: $([ -f "$GO_MOD_APPEND_FILE" ] && echo "YES" || echo "NO")"
    echo "  State file: $STATE_FILE"
}

disable_dev_mode() {
    log_info "=========================================="
    log_info "Starting development mode disablement"
    log_info "=========================================="
    
    if [ ! -f "$STATE_FILE" ]; then
        log_error "Development mode is not enabled or state file is missing!"
        log_error "Expected state file: $STATE_FILE"
        echo "Nothing to disable."
        exit 1
    fi

    log_debug "State file found: $STATE_FILE"
    log_debug "Working directory: $SCRIPT_DIR"

    # Remove copied files
    log_info "Step 1: Removing copied files..."
    local file_count=$(grep -c "^FILE:" "$STATE_FILE" || echo "0")
    log_info "Found $file_count file(s) to remove"
    
    local removed_count=0
    while IFS= read -r line; do
        if [[ "$line" =~ ^FILE:(.+)$ ]]; then
            rel_path="${BASH_REMATCH[1]}"
            file="$SCRIPT_DIR/$rel_path"
            
            log_debug "Attempting to remove: $file"
            if [ -f "$file" ]; then
                rm "$file"
                log_info "✓ Removed: $rel_path"
                ((removed_count++))
            else
                log_warn "File not found (may have been manually deleted): $rel_path"
            fi
        fi
    done < "$STATE_FILE"

    log_info "Successfully removed $removed_count file(s)"

    # Revert go.mod changes if they were made
    log_info "Step 2: Reverting go.mod changes..."
    if [ -f "$GO_MOD_APPEND_FILE" ]; then
        log_debug "Found append file: $GO_MOD_APPEND_FILE"
        
        # Find the marker line and remove everything from that line onwards
        if grep -q "// Added by dev-toggle.sh - DO NOT EDIT" "$GO_MOD_FILE"; then
            log_debug "Found marker comment in go.mod"
            local lines_before=$(wc -l < "$GO_MOD_FILE")
            
            # Use sed to delete from the marker line to end of file
            log_debug "Removing lines from marker to end of file..."
            sed -i.bak '/\/\/ Added by dev-toggle.sh - DO NOT EDIT/,$d' "$GO_MOD_FILE"
            rm -f "${GO_MOD_FILE}.bak"
            
            # Remove any trailing empty lines that might be left
            log_debug "Cleaning up trailing empty lines..."
            while [ -s "$GO_MOD_FILE" ]; do
                last_line=$(tail -n 1 "$GO_MOD_FILE")
                if [ -z "$last_line" ] || [ "$last_line" = $'\n' ]; then
                    # Remove last line if empty
                    sed -i.bak '$ d' "$GO_MOD_FILE" 2>/dev/null || {
                        # Fallback for systems without sed -i
                        head -n -1 "$GO_MOD_FILE" > "${GO_MOD_FILE}.tmp"
                        mv "${GO_MOD_FILE}.tmp" "$GO_MOD_FILE"
                    }
                    rm -f "${GO_MOD_FILE}.bak"
                else
                    break
                fi
            done
            
            local lines_after=$(wc -l < "$GO_MOD_FILE")
            local lines_removed=$((lines_before - lines_after))
            log_info "✓ Removed $lines_removed line(s) from go.mod"
        else
            log_warn "Marker comment not found in go.mod (may have been manually edited)"
        fi
        
        log_debug "Removing append file: $GO_MOD_APPEND_FILE"
        rm -f "$GO_MOD_APPEND_FILE"
    else
        log_debug "No append file found, skipping go.mod revert"
    fi

    # Run go mod tidy
    log_info "Step 3: Running go mod tidy..."
    log_debug "Changing to directory: $SCRIPT_DIR"
    cd "$SCRIPT_DIR"
    log_debug "Executing: go mod tidy"
    go mod tidy
    log_info "✓ go mod tidy completed successfully"

    # Clean up state file
    log_info "Step 4: Cleaning up state file..."
    log_debug "Removing: $STATE_FILE"
    rm -f "$STATE_FILE"
    log_info "✓ State file removed"

    # Remove internal configs directory
    log_info "Step 5: Removing internal configs directory..."
    if [ -d "$INTERNAL_REPO_DIR" ]; then
        log_debug "Removing directory: $INTERNAL_REPO_DIR"
        rm -rf "$INTERNAL_REPO_DIR"
        log_info "✓ Internal configs directory removed"
    else
        log_debug "Internal configs directory not found, skipping"
    fi

    log_info "=========================================="
    log_info "✓ Development mode disabled successfully!"
    log_info "=========================================="
}

update_internal_configs() {
    log_info "=========================================="
    log_info "Updating internal configurations"
    log_info "=========================================="
    
    if [ ! -f "$STATE_FILE" ]; then
        log_error "Development mode is not enabled!"
        log_error "Expected state file: $STATE_FILE"
        echo "Run '$0 enable' first."
        exit 1
    fi

    log_info "Current state: Development mode is ENABLED"
    log_info "Update strategy: Disable -> Re-enable with latest configs"
    echo ""
    
    # Disable current dev mode
    log_info "Phase 1: Disabling current development mode..."
    disable_dev_mode
    
    echo ""
    log_info "Phase 2: Re-enabling development mode with latest configs..."
    # Re-enable with fresh configs
    enable_dev_mode
    
    echo ""
    log_info "=========================================="
    log_info "✓ Internal configs updated successfully!"
    log_info "=========================================="
}

# Main script logic
log_debug "Script started with command: ${1:-<none>}"
log_debug "Script directory: $SCRIPT_DIR"
log_debug "Internal repo URL: $INTERNAL_REPO_URL"
echo ""

case "${1:-}" in
    enable)
        log_debug "Command: enable"
        enable_dev_mode
        ;;
    disable)
        log_debug "Command: disable"
        disable_dev_mode
        ;;
    status)
        log_debug "Command: status"
        check_status
        ;;
    update)
        log_debug "Command: update"
        update_internal_configs
        ;;
    *)
        log_error "Invalid or missing command"
        print_usage
        ;;
esac