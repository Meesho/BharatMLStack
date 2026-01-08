#!/bin/bash

set -e

# Get the script directory (parent directory where this script lives)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

get_available_folders() {
    local folders=()
    for dir in "$SCRIPT_DIR"/*; do
        if [ -d "$dir" ] && [ -f "$dir/go.mod" ]; then
            local folder_name=$(basename "$dir")
            folders+=("$folder_name")
        fi
    done
    echo "${folders[@]}"
}

interactive_select_folder() {
    local available_folders=($(get_available_folders))
    
    if [ ${#available_folders[@]} -eq 0 ]; then
        log_error "No folders with go.mod found in $SCRIPT_DIR"
        exit 1
    fi
    
    echo "" >&2
    echo "==========================================" >&2
    echo "  Step 1: Select Folder(s)" >&2
    echo "==========================================" >&2
    echo "" >&2
    echo "Available folders:" >&2
    local index=1
    for folder in "${available_folders[@]}"; do
        echo "  [$index] $folder" >&2
        ((index++))
    done
    echo "" >&2
    echo "You can select:" >&2
    echo "  • Single folder: Enter a number (e.g., 1)" >&2
    echo "  • Multiple folders: Enter numbers separated by comma or space (e.g., 1,2 or 1 2 3)" >&2
    echo "" >&2
    
    while true; do
        read -p "Select folder(s): " selection
        
        # Remove any extra spaces
        selection=$(echo "$selection" | tr -s ' ' | tr ',' ' ')
        
        # Validate all selections
        local valid=true
        local selected_folders=()
        local IFS=' '
        for num in $selection; do
            if [[ "$num" =~ ^[0-9]+$ ]] && [ "$num" -ge 1 ] && [ "$num" -le ${#available_folders[@]} ]; then
                selected_folders+=("${available_folders[$((num-1))]}")
            else
                valid=false
                break
            fi
        done
        
        if [ "$valid" = true ] && [ ${#selected_folders[@]} -gt 0 ]; then
            # Remove duplicates
            local unique_folders=()
            for folder in "${selected_folders[@]}"; do
                local is_duplicate=false
                for unique in "${unique_folders[@]}"; do
                    if [ "$folder" = "$unique" ]; then
                        is_duplicate=true
                        break
                    fi
                done
                if [ "$is_duplicate" = false ]; then
                    unique_folders+=("$folder")
                fi
            done
            
            # Output folders separated by space (will be captured as array) to stdout
            echo "${unique_folders[@]}"
            return 0
        else
            echo "Invalid selection. Please enter number(s) between 1 and ${#available_folders[@]}." >&2
            echo "  Examples: 1  or  1,2  or  1 2 3" >&2
        fi
    done
}

interactive_select_command() {
    echo "" >&2
    echo "==========================================" >&2
    echo "  Step 2: Select Command" >&2
    echo "==========================================" >&2
    echo "" >&2
    echo "Available commands:" >&2
    echo "  [1] enable   - Enable development mode (clone internal repo, copy files, update go.mod)" >&2
    echo "  [2] disable  - Disable development mode (remove copied files and go.mod changes)" >&2
    echo "  [3] status   - Show current development mode status" >&2
    echo "  [4] update   - Update internal configs (pull latest from internal repo)" >&2
    echo "" >&2
    
    while true; do
        read -p "Select command (1-4): " selection
        case "$selection" in
            1)
                echo "enable"
                return 0
                ;;
            2)
                echo "disable"
                return 0
                ;;
            3)
                echo "status"
                return 0
                ;;
            4)
                echo "update"
                return 0
                ;;
            *)
                echo "Invalid selection. Please enter a number between 1 and 4." >&2
                ;;
        esac
    done
}

select_internal_repo_branch() {
    local provided_branch="${1:-}"
    local branch=""

    # 1) explicit arg wins
    if [ -n "$provided_branch" ]; then
        branch="$provided_branch"
    # 2) env var next
    elif [ -n "${INTERNAL_REPO_BRANCH:-}" ]; then
        branch="$INTERNAL_REPO_BRANCH"
    # 3) interactive prompt if we have a TTY
    elif [ -t 0 ]; then
        echo "" >&2
        echo "==========================================" >&2
        echo "  Step 3: Select Internal Configs Branch" >&2
        echo "==========================================" >&2
        echo "" >&2
        echo "Choose the branch to use from internal configs repo:" >&2
        echo "  [1] develop (default)" >&2
        echo "  [2] main" >&2
        echo "  [3] enter a custom branch name (e.g., feature/my-change)" >&2
        echo "" >&2

        while true; do
            read -p "Select branch (1-2 or name): " selection
            selection="$(echo "${selection:-}" | tr '[:upper:]' '[:lower:]' | tr -d '[:space:]')"
            case "$selection" in
                ""|1|develop)
                    branch="develop"
                    break
                    ;;
                2|main)
                    branch="main"
                    break
                    ;;
                3)
                    read -p "Enter branch name: " custom
                    branch="${custom:-}"
                    break
                    ;;
                *)
                    # Treat anything else as a branch name
                    branch="$selection"
                    break
                    ;;
            esac
        done
    else
        # 4) non-interactive default
        branch="develop"
    fi

    branch="$(echo "${branch:-}" | tr -d '\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
    if [ -z "$branch" ]; then
        log_error "Internal configs branch cannot be empty"
        exit 1
    fi

    # Validate branch name as a proper git branch ref (safe for checkout/clone)
    if command -v git >/dev/null 2>&1; then
        if ! git check-ref-format --branch "$branch" >/dev/null 2>&1; then
            log_error "Invalid internal configs branch name: '$branch'"
            exit 1
        fi
    fi

    echo "$branch"
    return 0
}

print_usage() {
    echo "Usage: $0 <folder-name> <command> [branch]"
    echo ""
    echo "Parameters:"
    echo "  1. folder-name  - Name of the folder to operate on"
    echo ""
    
    # Detect and show available folders
    local available_folders=($(get_available_folders))
    if [ ${#available_folders[@]} -gt 0 ]; then
        echo "     Available folders:"
        for folder in "${available_folders[@]}"; do
            echo "       • $folder"
        done
    else
        echo "     Available folders: (none found with go.mod)"
    fi
    
    echo ""
    echo "  2. command      - Action to perform"
    echo ""
    echo "     Available commands:"
    echo "       • enable   - Enable development mode (clone internal repo, copy files, update go.mod)"
    echo "       • disable  - Disable development mode (remove copied files and go.mod changes)"
    echo "       • status   - Show current development mode status"
    echo "       • update   - Update internal configs (pull latest from internal repo)"
    echo ""
    echo "  3. branch       - (Optional) Internal configs repo branch to use (default: develop)"
    echo "                   Examples: develop, main, feature/my-change"
    echo "                   You can also set INTERNAL_REPO_BRANCH env var."
    echo ""
    echo "Examples:"
    if [ ${#available_folders[@]} -gt 0 ]; then
        local first_folder="${available_folders[0]}"
        echo "  $0 $first_folder enable"
        echo "  $0 $first_folder enable develop"
        echo "  $0 $first_folder enable main"
        if [ ${#available_folders[@]} -gt 1 ]; then
            local second_folder="${available_folders[1]}"
            echo "  $0 $second_folder status"
        fi
        echo "  $0 $first_folder disable"
        echo "  $0 $first_folder update"
    else
        echo "  $0 online-feature-store enable"
        echo "  $0 horizon status"
        echo "  $0 online-feature-store disable"
    fi
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

# Validate folder name and set up paths
validate_and_setup() {
    FOLDER_NAME="${1:-}"
    COMMAND="${2:-}"
    
    if [ -z "$FOLDER_NAME" ]; then
        log_error "Folder name is required"
        print_usage
    fi
    
    if [ -z "$COMMAND" ]; then
        log_error "Command is required"
        print_usage
    fi
    
    # Set up paths based on folder name
    TARGET_DIR="$SCRIPT_DIR/$FOLDER_NAME"
    STATE_FILE="$TARGET_DIR/.dev-toggle-state"
    GO_MOD_FILE="$TARGET_DIR/go.mod"
    GO_MOD_APPEND_FILE="$TARGET_DIR/.go.mod.appended"
    
    INTERNAL_REPO_URL="https://github.com/Meesho/BharatMLStack-internal-configs"
    INTERNAL_REPO_DIR="$TARGET_DIR/.internal-configs"
    INTERNAL_FOLDER_DIR="$INTERNAL_REPO_DIR/$FOLDER_NAME"
    
    # Validate that target directory exists
    if [ ! -d "$TARGET_DIR" ]; then
        log_error "Target directory does not exist: $TARGET_DIR"
        exit 1
    fi
    
    # Validate that go.mod exists in target directory
    if [ ! -f "$GO_MOD_FILE" ]; then
        log_error "go.mod not found in target directory: $TARGET_DIR"
        log_error "This script is designed for Go projects with go.mod files"
        exit 1
    fi
    
    log_debug "Target directory: $TARGET_DIR"
    log_debug "State file: $STATE_FILE"
    log_debug "Internal repo directory: $INTERNAL_REPO_DIR"
    log_debug "Internal folder directory: $INTERNAL_FOLDER_DIR"
}

check_status() {
    log_info "=========================================="
    log_info "Development Mode Status for: $FOLDER_NAME"
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
        echo "Run '$0 $FOLDER_NAME enable' to enable development mode."
        echo ""
        return 1
    fi
}

clone_or_update_internal_repo() {
    local target_branch="${INTERNAL_REPO_BRANCH:-develop}"

    if [ -d "$INTERNAL_REPO_DIR/.git" ]; then
        log_info "Internal configs repo already exists, updating..."
        log_debug "Repository path: $INTERNAL_REPO_DIR"
        cd "$INTERNAL_REPO_DIR"
        
        # Fetch latest changes
        log_info "Fetching latest changes from origin..."
        git fetch origin
        log_debug "Fetch completed"
        
        log_info "Using branch: $target_branch"

        # Ensure the remote branch exists (and fetch it explicitly if needed)
        if ! git show-ref --verify --quiet "refs/remotes/origin/$target_branch"; then
            log_info "Remote branch origin/$target_branch not found locally; fetching it..."
            git fetch origin "$target_branch" || true
        fi
        if ! git show-ref --verify --quiet "refs/remotes/origin/$target_branch"; then
            log_error "Branch not found in internal configs repo: origin/$target_branch"
            exit 1
        fi

        # Switch to the requested branch, tracking origin/<branch>
        local current_branch
        current_branch=$(git rev-parse --abbrev-ref HEAD)
        log_debug "Current branch: $current_branch"
        if [ "$current_branch" != "$target_branch" ]; then
            log_info "Switching from $current_branch to $target_branch"
        fi
        git checkout -B "$target_branch" "origin/$target_branch"
        log_debug "Branch checkout completed"
        
        # Reset to latest
        log_info "Resetting to origin/$target_branch..."
        git reset --hard "origin/$target_branch"
        local commit_hash=$(git rev-parse --short HEAD)
        log_info "Updated to commit: $commit_hash"
        
        cd "$TARGET_DIR"
    else
        log_info "Internal configs repo not found, cloning..."
        log_debug "Cloning from: $INTERNAL_REPO_URL"
        log_debug "Destination: $INTERNAL_REPO_DIR"
        log_debug "Branch: $target_branch"
        rm -rf "$INTERNAL_REPO_DIR"
        git clone -b "$target_branch" "$INTERNAL_REPO_URL" "$INTERNAL_REPO_DIR"
        local commit_hash=$(cd "$INTERNAL_REPO_DIR" && git rev-parse --short HEAD)
        log_info "Clone completed at commit: $commit_hash"
    fi
}

enable_dev_mode() {
    log_info "=========================================="
    log_info "Starting development mode enablement for: $FOLDER_NAME"
    log_info "=========================================="
    
    if [ -f "$STATE_FILE" ]; then
        log_warn "Development mode is already enabled!"
        echo "Run '$0 $FOLDER_NAME disable' first to revert, or delete $STATE_FILE if state is corrupted."
        exit 1
    fi

    log_debug "State file: $STATE_FILE"
    log_debug "Working directory: $TARGET_DIR"

    # Clone or update the internal repo
    log_info "Step 1: Fetching internal configurations..."
    clone_or_update_internal_repo

    # Check if folder directory exists in internal repo
    log_info "Step 2: Validating internal repository structure..."
    log_debug "Looking for: $INTERNAL_FOLDER_DIR"
    if [ ! -d "$INTERNAL_FOLDER_DIR" ]; then
        log_error "$FOLDER_NAME directory not found in internal configs repo!"
        log_error "Expected path: $INTERNAL_FOLDER_DIR"
        exit 1
    fi
    log_debug "✓ $FOLDER_NAME directory found"

    # Initialize state file
    log_info "Step 3: Initializing state tracking..."
    echo "# Dev toggle state - DO NOT EDIT MANUALLY" > "$STATE_FILE"
    echo "# Generated on $(date)" >> "$STATE_FILE"
    echo "# Folder: $FOLDER_NAME" >> "$STATE_FILE"
    log_debug "Created state file: $STATE_FILE"

    # Find and copy all .go files from internal repo
    log_info "Step 4: Searching for .go files to copy..."
    log_debug "Scanning directory: $INTERNAL_FOLDER_DIR"
    
    local copied_count=0
    local file_count=$(find "$INTERNAL_FOLDER_DIR" -name "*.go" -type f | wc -l)
    log_info "Found $file_count .go file(s) to copy"
    
    while IFS= read -r -d '' src_file; do
        # Get relative path from INTERNAL_FOLDER_DIR
        rel_path="${src_file#$INTERNAL_FOLDER_DIR/}"
        dest_file="$TARGET_DIR/$rel_path"
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
    done < <(find "$INTERNAL_FOLDER_DIR" -name "*.go" -type f -print0)

    log_info "Successfully copied $copied_count file(s)"

    # Check if there's a go.mod file in internal repo with replace directives
    log_info "Step 5: Processing go.mod modifications..."
    if [ -f "$INTERNAL_FOLDER_DIR/go.mod" ]; then
        log_debug "Found go.mod in internal configs: $INTERNAL_FOLDER_DIR/go.mod"
        
        # Extract require/replace directives from internal go.mod.
        # Supports both single-line directives and block forms:
        #   require ( ... )
        #   replace ( ... )
        # We normalize block entries into single-line statements to keep the appended section simple.
        local extracted_lines
        extracted_lines="$(
            awk '
                function ltrim(s) { sub(/^[ \t]+/, "", s); return s }
                BEGIN { in_req=0; in_rep=0 }
                {
                    line=$0

                    if (line ~ /^require[ \t]*\(/) { in_req=1; next }
                    if (line ~ /^replace[ \t]*\(/) { in_rep=1; next }

                    if (in_req) {
                        if (line ~ /^\)/) { in_req=0; next }
                        trimmed=ltrim(line)
                        if (trimmed=="" || trimmed ~ /^\/\//) next
                        print "require " trimmed
                        next
                    }

                    if (in_rep) {
                        if (line ~ /^\)/) { in_rep=0; next }
                        trimmed=ltrim(line)
                        if (trimmed=="" || trimmed ~ /^\/\//) next
                        print "replace " trimmed
                        next
                    }

                    if (line ~ /^require[ \t]+/ && line !~ /^require[ \t]*\(/) { print line; next }
                    if (line ~ /^replace[ \t]+/ && line !~ /^replace[ \t]*\(/) { print line; next }
                }
            ' "$INTERNAL_FOLDER_DIR/go.mod" | sed '/^[[:space:]]*$/d'
        )"

        if [ -n "$extracted_lines" ]; then
            printf "%s\n" "$extracted_lines" > "$GO_MOD_APPEND_FILE"

            local extracted_count
            extracted_count=$(wc -l < "$GO_MOD_APPEND_FILE")
            log_info "Found $extracted_count directive(s) (require/replace) to append"
            log_debug "Directives to append:"
            while IFS= read -r line; do
                log_debug "  $line"
            done < "$GO_MOD_APPEND_FILE"

            # Append to current go.mod
            log_info "Appending to $GO_MOD_FILE..."
            echo "" >> "$GO_MOD_FILE"
            echo "// Added by dev-toggle-go.sh - DO NOT EDIT" >> "$GO_MOD_FILE"
            cat "$GO_MOD_APPEND_FILE" >> "$GO_MOD_FILE"

            log_info "✓ Successfully appended require/replace directives to go.mod"
        else
            log_warn "No require/replace directives found in internal go.mod"
            rm -f "$GO_MOD_APPEND_FILE"
        fi
    else
        log_warn "No go.mod found in internal configs at: $INTERNAL_FOLDER_DIR/go.mod"
    fi

    # Run go mod tidy
    log_info "Step 6: Running go mod tidy..."
    log_debug "Changing to directory: $TARGET_DIR"
    cd "$TARGET_DIR"
    log_debug "Executing: go mod tidy"
    go mod tidy
    log_info "✓ go mod tidy completed successfully"

    log_info "=========================================="
    log_info "✓ Development mode enabled successfully!"
    log_info "=========================================="
    echo ""
    echo "Summary:"
    echo "  Folder: $FOLDER_NAME"
    echo "  Files copied: $copied_count"
    echo "  go.mod updated: $([ -f "$GO_MOD_APPEND_FILE" ] && echo "YES" || echo "NO")"
    echo "  State file: $STATE_FILE"
}

disable_dev_mode() {
    log_info "=========================================="
    log_info "Starting development mode disablement for: $FOLDER_NAME"
    log_info "=========================================="
    
    if [ ! -f "$STATE_FILE" ]; then
        log_error "Development mode is not enabled or state file is missing!"
        log_error "Expected state file: $STATE_FILE"
        echo "Nothing to disable."
        exit 1
    fi

    log_debug "State file found: $STATE_FILE"
    log_debug "Working directory: $TARGET_DIR"

    # Remove copied files
    log_info "Step 1: Removing copied files..."
    local file_count=$(grep -c "^FILE:" "$STATE_FILE" || echo "0")
    log_info "Found $file_count file(s) to remove"
    
    local removed_count=0
    while IFS= read -r line; do
        if [[ "$line" =~ ^FILE:(.+)$ ]]; then
            rel_path="${BASH_REMATCH[1]}"
            file="$TARGET_DIR/$rel_path"
            
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
        if grep -q "// Added by dev-toggle-go.sh - DO NOT EDIT" "$GO_MOD_FILE"; then
            log_debug "Found marker comment in go.mod"
            local lines_before=$(wc -l < "$GO_MOD_FILE")
            
            # Use sed to delete from the marker line to end of file
            log_debug "Removing lines from marker to end of file..."
            sed -i.bak '/\/\/ Added by dev-toggle-go.sh - DO NOT EDIT/,$d' "$GO_MOD_FILE"
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
    log_debug "Changing to directory: $TARGET_DIR"
    cd "$TARGET_DIR"
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
    log_info "Updating internal configurations for: $FOLDER_NAME"
    log_info "=========================================="
    
    if [ ! -f "$STATE_FILE" ]; then
        log_error "Development mode is not enabled!"
        log_error "Expected state file: $STATE_FILE"
        echo "Run '$0 $FOLDER_NAME enable' first."
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
FOLDER_NAME_INPUT="${1:-}"
COMMAND="${2:-}"
BRANCH_INPUT="${3:-}"

# Interactive mode: prompt for missing parameters
if [ -z "$FOLDER_NAME_INPUT" ]; then
    echo "=========================================="
    echo "  Development Toggle for Go Projects"
    echo "=========================================="
    FOLDER_NAME_INPUT=$(interactive_select_folder)
fi

if [ -z "$COMMAND" ]; then
    COMMAND=$(interactive_select_command)
fi

# If command needs internal configs, choose the branch once (applies to all selected folders)
if [ "$COMMAND" = "enable" ] || [ "$COMMAND" = "update" ]; then
    INTERNAL_REPO_BRANCH="$(select_internal_repo_branch "$BRANCH_INPUT")"
    log_debug "Internal configs branch: $INTERNAL_REPO_BRANCH"
fi

# Parse folder names (support multiple folders)
FOLDER_NAMES=($FOLDER_NAME_INPUT)

# If only one folder, process it directly
if [ ${#FOLDER_NAMES[@]} -eq 1 ]; then
    FOLDER_NAME="${FOLDER_NAMES[0]}"
    
    # Validate and set up paths
    validate_and_setup "$FOLDER_NAME" "$COMMAND"
    
    log_debug "Script started with folder: $FOLDER_NAME, command: $COMMAND"
    log_debug "Script directory: $SCRIPT_DIR"
    log_debug "Target directory: $TARGET_DIR"
    echo ""
    
    case "$COMMAND" in
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
            log_error "Invalid command: $COMMAND"
            print_usage
            ;;
    esac
else
    # Multiple folders selected - process each one
    echo ""
    log_info "Processing ${#FOLDER_NAMES[@]} folder(s) with command: $COMMAND"
    echo ""
    
    for FOLDER_NAME in "${FOLDER_NAMES[@]}"; do
        echo ""
        log_info "=========================================="
        log_info "Processing: $FOLDER_NAME"
        log_info "=========================================="
        echo ""
        
        # Validate and set up paths for this folder
        validate_and_setup "$FOLDER_NAME" "$COMMAND"
        
        case "$COMMAND" in
            enable)
                enable_dev_mode
                ;;
            disable)
                disable_dev_mode
                ;;
            status)
                check_status
                ;;
            update)
                update_internal_configs
                ;;
            *)
                log_error "Invalid command: $COMMAND"
                print_usage
                exit 1
                ;;
        esac
    done
    
    echo ""
    log_info "=========================================="
    log_info "✓ Completed processing all ${#FOLDER_NAMES[@]} folder(s)"
    log_info "=========================================="
fi

