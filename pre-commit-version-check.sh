#!/bin/bash

# Smart Pre-commit Version Check Script
# This script intelligently checks for changes in directories with VERSION files
# and prompts the user to increment versions only when needed

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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

echo -e "${CYAN}🔄 Smart Version Check - Analyzing changes...${NC}\n"

# Function to get current branch name
get_current_branch() {
    git branch --show-current
}

# Function to get the actual parent branch from which current branch was forked
get_parent_branch() {
    local current_branch=$(get_current_branch)
    
    # If we're on master/main, there's no parent to compare against
    if [[ "$current_branch" == "master" || "$current_branch" == "main" ]]; then
        echo "HEAD~1"  # Compare against previous commit
        return
    fi
    
    # Method 1: Try to find the fork point using reflog
    local fork_point
    if fork_point=$(git reflog show --format="%H %gs" "$current_branch" 2>/dev/null | grep "branch: Created from" | head -1 | cut -d' ' -f1 2>/dev/null); then
        if [ -n "$fork_point" ]; then
            # Find which branch contains this commit
            local parent_branch
            if parent_branch=$(git branch --contains "$fork_point" 2>/dev/null | grep -v "^\*" | grep -v "$current_branch" | head -1 | sed 's/^[[:space:]]*//' 2>/dev/null); then
                if [ -n "$parent_branch" ]; then
                    echo "$parent_branch"
                    return
                fi
            fi
        fi
    fi
    
    # Method 2: Use git show-branch to find the most recent common ancestor
    local parent_candidates=()
    
    # Get all local branches except current
    while IFS= read -r branch; do
        branch=$(echo "$branch" | sed 's/^[[:space:]]*//' | sed 's/^* //')
        if [ "$branch" != "$current_branch" ] && [ -n "$branch" ]; then
            parent_candidates+=("$branch")
        fi
    done < <(git branch 2>/dev/null)
    
    # Get remote branches if no local candidates
    if [ ${#parent_candidates[@]} -eq 0 ]; then
        while IFS= read -r branch; do
            branch=$(echo "$branch" | sed 's/^[[:space:]]*//' | sed 's|^origin/||')
            if [ "$branch" != "$current_branch" ] && [ "$branch" != "HEAD" ] && [ -n "$branch" ]; then
                parent_candidates+=("origin/$branch")
            fi
        done < <(git branch -r 2>/dev/null | grep -v ' -> ')
    fi
    
    # Method 3: Find the branch with the most recent common ancestor
    local best_parent=""
    local best_distance=999999
    local best_commit_date=0
    
    for candidate in "${parent_candidates[@]}"; do
        if git show-ref --verify --quiet "refs/heads/$candidate" || git show-ref --verify --quiet "refs/remotes/$candidate"; then
            # Get merge base with candidate
            local merge_base
            if merge_base=$(git merge-base "$candidate" HEAD 2>/dev/null); then
                # Get commit date of merge base (more recent = better parent candidate)
                local commit_date
                if commit_date=$(git log -1 --format="%ct" "$merge_base" 2>/dev/null); then
                    # Get distance from merge base to current HEAD
                    local distance
                    if distance=$(git rev-list --count "$merge_base..HEAD" 2>/dev/null); then
                        # Prefer more recent common ancestor
                        if [ "$commit_date" -gt "$best_commit_date" ] || 
                           ([ "$commit_date" -eq "$best_commit_date" ] && [ "$distance" -lt "$best_distance" ]); then
                            best_parent="$candidate"
                            best_distance="$distance"
                            best_commit_date="$commit_date"
                        fi
                    fi
                fi
            fi
        fi
    done
    
    if [ -n "$best_parent" ]; then
        echo "$best_parent"
        return
    fi
    
    # Method 4: Fallback to common branch names with preference for most recent
    local fallback_candidates=("master" "main" "develop" "dev" "origin/master" "origin/main" "origin/develop" "origin/dev")
    
    for candidate in "${fallback_candidates[@]}"; do
        if git show-ref --verify --quiet "refs/heads/$candidate" || git show-ref --verify --quiet "refs/remotes/$candidate"; then
            if git merge-base --is-ancestor "$candidate" HEAD 2>/dev/null || git merge-base --is-ancestor HEAD "$candidate" 2>/dev/null; then
                echo "$candidate"
                return
            fi
        fi
    done
    
    # Ultimate fallback
    echo "HEAD~1"
}

# Function to get current version from VERSION file
get_current_version() {
    local dir="$1"
    if [ -f "$dir/VERSION" ]; then
        cat "$dir/VERSION" | tr -d '\n' | tr -d ' ' | sed 's/^v//'
    else
        echo "0.0.0"
    fi
}

# Function to check if directory has changes against parent branch
has_changes_against_parent() {
    local dir="$1"
    local parent_branch="$2"
    
    # Get list of changed files between parent branch and current HEAD
    local changed_files
    if ! changed_files=$(git diff --name-only "$parent_branch"...HEAD 2>/dev/null); then
        # Fallback if comparison fails
        echo -e "${YELLOW}   ⚠️  Could not compare against $parent_branch, checking against HEAD~1${NC}"
        changed_files=$(git diff --name-only HEAD~1...HEAD 2>/dev/null || echo "")
    fi
    
    # Check if any changed files are in this directory
    if echo "$changed_files" | grep -q "^$dir/"; then
        return 0  # Has changes
    fi
    
    # Also check for staged changes in current commit
    if git diff --cached --name-only | grep -q "^$dir/"; then
        return 0  # Has changes
    fi
    
    return 1  # No changes
}

# Function to show changes in directory
show_directory_changes() {
    local dir="$1"
    local parent_branch="$2"
    
    echo -e "${BLUE}   Changed files in $dir:${NC}"
    
    # Show files changed against parent
    local changed_files
    if changed_files=$(git diff --name-only "$parent_branch"...HEAD 2>/dev/null | grep "^$dir/" || true); then
        if [ -n "$changed_files" ]; then
            echo "$changed_files" | sed 's/^/     - /'
        fi
    fi
    
    # Show staged files
    local staged_files
    if staged_files=$(git diff --cached --name-only | grep "^$dir/" || true); then
        if [ -n "$staged_files" ]; then
            echo -e "${CYAN}   Staged files:${NC}"
            echo "$staged_files" | sed 's/^/     - /'
        fi
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

# Function to prompt for version increment
prompt_version_increment() {
    local dir="$1"
    local current_version="$2"
    
    echo -e "\n${YELLOW}📂 Directory: ${dir}${NC}"
    echo -e "${BLUE}   Current version: ${current_version}${NC}"
    
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

# Function to update VERSION file and pyproject.toml
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
    
    # Also update pyproject.toml if it exists (for Python packages)
    local pyproject_file="$dir/pyproject.toml"
    if [ -f "$pyproject_file" ]; then
        # Use sed to update the version line in pyproject.toml
        if sed -i '' "s/^version = [\"'].*[\"']/version = \"$new_version\"/" "$pyproject_file" 2>/dev/null; then
            echo -e "${GREEN}   ✅ Updated $pyproject_file to $new_version${NC}"
        else
            echo -e "${YELLOW}   ⚠️  Could not update $pyproject_file automatically${NC}"
            echo -e "${YELLOW}      Please manually update the version to $new_version${NC}"
        fi
    fi
}

# Main logic starts here
current_branch=$(get_current_branch)
parent_branch=$(get_parent_branch)

echo -e "${CYAN}Current branch: ${current_branch}${NC}"
echo -e "${CYAN}Detected fork parent: ${parent_branch}${NC}"

# Show additional context about the fork point
if [[ "$parent_branch" != "HEAD~1" ]]; then
    merge_base=$(git merge-base "$parent_branch" HEAD 2>/dev/null || echo "")
    if [ -n "$merge_base" ]; then
        commits_ahead=$(git rev-list --count "$merge_base..HEAD" 2>/dev/null || echo "unknown")
        echo -e "${CYAN}Fork point: ${merge_base:0:8} (${commits_ahead} commits ahead)${NC}"
    fi
fi
echo

# First, check if any versioned directories have changes
echo -e "${BLUE}🔍 Scanning for changes in versioned directories...${NC}"

changed_dirs=()
for dir in "${VERSION_DIRS[@]}"; do
    if [ -d "$dir" ] && has_changes_against_parent "$dir" "$parent_branch"; then
        changed_dirs+=("$dir")
        echo -e "${GREEN}   ✓ Changes detected in: ${dir}${NC}"
    fi
done

# If no changes detected in any versioned directory
if [ ${#changed_dirs[@]} -eq 0 ]; then
    echo -e "${GREEN}✅ No changes detected in versioned directories. No version updates needed.${NC}"
    exit 0
fi

echo -e "\n${YELLOW}📋 Found changes in ${#changed_dirs[@]} versioned director(ies):${NC}"
for dir in "${changed_dirs[@]}"; do
    echo -e "   • ${dir}"
done

# Show detailed changes for each directory
echo -e "\n${CYAN}📄 Details of changes:${NC}"
for dir in "${changed_dirs[@]}"; do
    echo -e "\n${BLUE}━━━ ${dir} ━━━${NC}"
    show_directory_changes "$dir" "$parent_branch"
done

# Ask user if they want to increment versions
echo -e "\n${YELLOW}❓ Do you want to increment versions for the changed directories? [y/N]: ${NC}"
read -r increment_versions </dev/tty

if [[ ! "$increment_versions" =~ ^[Yy]$ ]]; then
    echo -e "\n${BLUE}ℹ️  Version increments skipped. Proceeding with commit as-is.${NC}"
    exit 0
fi

# Process each changed directory for version updates
version_updates=()

echo -e "\n${CYAN}🔧 Processing version updates...${NC}"

for dir in "${changed_dirs[@]}"; do
    current_version=$(get_current_version "$dir")
    
    # Call the function and get result via global variable
    prompt_version_increment "$dir" "$current_version"
    increment_type="$INCREMENT_TYPE"
    
    if [ "$increment_type" != "skip" ]; then
        new_version=$(increment_version "$current_version" "$increment_type")
        version_updates+=("$dir:$new_version")
        echo -e "${BLUE}   📈 ${dir}: ${current_version} → ${new_version}${NC}"
    else
        echo -e "${YELLOW}   ⏭️  Skipped version update for ${dir}${NC}"
    fi
done

# Apply version updates if any
if [ ${#version_updates[@]} -gt 0 ]; then
    echo -e "\n${YELLOW}📋 Summary of version updates:${NC}"
    for update in "${version_updates[@]}"; do
        IFS=':' read -ra UPDATE_PARTS <<< "$update"
        dir="${UPDATE_PARTS[0]}"
        new_version="${UPDATE_PARTS[1]}"
        current_version=$(get_current_version "$dir")
        echo -e "   ${dir}: ${current_version} → ${new_version}"
    done
    
    echo -n -e "\n${YELLOW}Apply these version updates and stage the files? [y/N]: ${NC}"
    read -r confirm </dev/tty
    
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        # Apply updates
        echo -e "\n${CYAN}🔄 Applying version updates...${NC}"
        for update in "${version_updates[@]}"; do
            IFS=':' read -ra UPDATE_PARTS <<< "$update"
            dir="${UPDATE_PARTS[0]}"
            new_version="${UPDATE_PARTS[1]}"
            
            update_version_file "$dir" "$new_version"
            
            # Stage the VERSION file
            git add "$dir/VERSION"
            echo -e "${GREEN}   ✅ Staged $dir/VERSION${NC}"
            
            # Also stage pyproject.toml if it exists
            if [ -f "$dir/pyproject.toml" ]; then
                git add "$dir/pyproject.toml"
                echo -e "${GREEN}   ✅ Staged $dir/pyproject.toml${NC}"
            fi
        done
        
        echo -e "\n${GREEN}🎉 All version updates completed and staged!${NC}"
    else
        echo -e "\n${RED}❌ Version updates cancelled. Commit aborted.${NC}"
        exit 1
    fi
else
    echo -e "\n${BLUE}ℹ️  No version updates requested${NC}"
fi

echo -e "\n${GREEN}✅ Smart version check completed successfully!${NC}" 