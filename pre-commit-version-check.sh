#!/bin/bash

# Pre-commit Version Check Script
# This script checks for changes in directories with VERSION files
# and prompts the user to increment versions before committing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global variable for function communication
INCREMENT_TYPE=""

# Directories with VERSION files (relative to repo root)
VERSION_DIRS=(
    "py-sdk/bharatml_commons"
    "py-sdk/spark_feature_push_client" 
    "py-sdk/grpc_feature_client"
    "horizon"
    "go-sdk"
    "online-feature-store"
    "trufflebox-ui"
)

echo -e "${BLUE}🔍 Checking for changes in versioned directories...${NC}"

# Function to get current version from VERSION file
get_current_version() {
    local dir="$1"
    if [ -f "$dir/VERSION" ]; then
        cat "$dir/VERSION" | tr -d '\n' | tr -d ' ' | sed 's/^v//'
    else
        echo "0.0.0"
    fi
}

# Function to increment version
increment_version() {
    local version="$1"
    local type="$2"
    
    # Parse version (remove 'v' prefix if present)
    version=$(echo "$version" | sed 's/^v//')
    
    # Split version into parts
    IFS='.' read -ra VERSION_PARTS <<< "$version"
    local major="${VERSION_PARTS[0]:-0}"
    local minor="${VERSION_PARTS[1]:-0}"
    local patch="${VERSION_PARTS[2]:-0}"
    
    case "$type" in
        "major")
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        "minor")
            minor=$((minor + 1))
            patch=0
            ;;
        "patch")
            patch=$((patch + 1))
            ;;
        *)
            echo "Invalid version type: $type"
            return 1
            ;;
    esac
    
    echo "$major.$minor.$patch"
}

# Function to check if directory has changes
has_changes() {
    local dir="$1"
    
    # Check if there are staged changes in this directory
    if git diff --cached --name-only | grep -q "^$dir/"; then
        return 0  # Has changes
    fi
    
    # Check if there are unstaged changes in this directory
    if git diff --name-only | grep -q "^$dir/"; then
        return 0  # Has changes
    fi
    
    return 1  # No changes
}

# Function to prompt for version increment
prompt_version_increment() {
    local dir="$1"
    local current_version="$2"
    echo "Prompting for version increment"
    
    echo -e "\n${YELLOW}📂 Directory: ${dir}${NC}"
    echo -e "${BLUE}   Current version: ${current_version}${NC}"
    echo -e "${GREEN}   Changes detected in this directory${NC}"
    
    while true; do
        echo -e "${YELLOW}   How would you like to increment the version?${NC}"
        echo "   [1] Major (breaking changes)"
        echo "   [2] Minor (new features)"
        echo "   [3] Patch (bug fixes)"
        echo "   [s] Skip (no version change)"
        echo -n "   Choice [1/2/3/s]: "
        
        read -r choice </dev/tty
        
        case "$choice" in
            1|major)
                INCREMENT_TYPE="major"
                return 0
                ;;
            2|minor)
                INCREMENT_TYPE="minor"
                return 0
                ;;
            3|patch)
                INCREMENT_TYPE="patch"
                return 0
                ;;
            s|skip)
                INCREMENT_TYPE="skip"
                return 0
                ;;
            *)
                echo -e "${RED}   Invalid choice. Please enter 1, 2, 3, or s${NC}"
                ;;
        esac
    done
}

# Function to update VERSION file
update_version_file() {
    local dir="$1"
    local new_version="$2"
    local version_file="$dir/VERSION"
    
    # Preserve the 'v' prefix if it existed in the original file
    local original_content=""
    if [ -f "$version_file" ]; then
        original_content=$(cat "$version_file")
    fi
    
    if [[ "$original_content" == v* ]]; then
        echo "v$new_version" > "$version_file"
    else
        echo "$new_version" > "$version_file"
    fi
    
    echo -e "${GREEN}   ✅ Updated $version_file to $new_version${NC}"
}

# Main logic
changed_dirs=()
version_updates=()

# Check each versioned directory for changes
for dir in "${VERSION_DIRS[@]}"; do
    echo "Checking $dir"
    if [ -d "$dir" ] && has_changes "$dir"; then
        changed_dirs+=("$dir")
        current_version=$(get_current_version "$dir")
        echo "Current version: $current_version"
        
        # Call the function and get result via global variable
        prompt_version_increment "$dir" "$current_version"
        increment_type="$INCREMENT_TYPE"
        echo "Increment type: $increment_type"
        
        if [ "$increment_type" != "skip" ]; then
            new_version=$(increment_version "$current_version" "$increment_type")
            version_updates+=("$dir:$new_version")
            echo "Version updates: $version_updates"
            echo -e "${BLUE}   New version will be: $new_version${NC}"
        else
            echo -e "${YELLOW}   Skipping version update for $dir${NC}"
        fi
    fi
done

# If no changes detected
if [ ${#changed_dirs[@]} -eq 0 ]; then
    echo -e "${GREEN}✅ No changes detected in versioned directories${NC}"
    exit 0
fi

# Confirm all updates
if [ ${#version_updates[@]} -gt 0 ]; then
    echo -e "\n${YELLOW}📋 Summary of version updates:${NC}"
    for update in "${version_updates[@]}"; do
        IFS=':' read -ra UPDATE_PARTS <<< "$update"
        dir="${UPDATE_PARTS[0]}"
        new_version="${UPDATE_PARTS[1]}"
        current_version=$(get_current_version "$dir")
        echo -e "   ${dir}: ${current_version} → ${new_version}"
    done
    
    echo -n -e "\n${YELLOW}Proceed with these version updates? [y/N]: ${NC}"
    read -r confirm </dev/tty
    
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        # Apply updates
        for update in "${version_updates[@]}"; do
            IFS=':' read -ra UPDATE_PARTS <<< "$update"
            dir="${UPDATE_PARTS[0]}"
            new_version="${UPDATE_PARTS[1]}"
            
            update_version_file "$dir" "$new_version"
            
            # Stage the VERSION file
            git add "$dir/VERSION"
            echo -e "${GREEN}   ✅ Staged $dir/VERSION${NC}"
        done
        
        echo -e "\n${GREEN}🎉 All version updates completed and staged!${NC}"
    else
        echo -e "\n${RED}❌ Version updates cancelled. Commit aborted.${NC}"
        exit 1
    fi
else
    echo -e "\n${BLUE}ℹ️  No version updates requested${NC}"
fi

echo -e "\n${GREEN}✅ Pre-commit version check completed${NC}" 