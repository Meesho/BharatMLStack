#!/bin/bash

# Setup script for pre-commit version checking
# This script installs the version check as a git pre-commit hook

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}🚀 Setting up pre-commit version check...${NC}"

# Check if we're in a git repository
if [ ! -d ".git" ]; then
    echo -e "${RED}❌ Error: Not in a git repository root${NC}"
    echo -e "${YELLOW}Please run this script from the root of your git repository${NC}"
    exit 1
fi

# Check if pre-commit-version-check.sh exists
if [ ! -f "pre-commit-version-check.sh" ]; then
    echo -e "${RED}❌ Error: pre-commit-version-check.sh not found${NC}"
    echo -e "${YELLOW}Please ensure the script is in the repository root${NC}"
    exit 1
fi

# Make sure the script is executable
chmod +x pre-commit-version-check.sh

# Create .git/hooks directory if it doesn't exist
mkdir -p .git/hooks

# Check if pre-commit hook already exists
if [ -f ".git/hooks/pre-commit" ]; then
    echo -e "${YELLOW}⚠️  Existing pre-commit hook found${NC}"
    echo -e "${BLUE}Backing up existing hook to .git/hooks/pre-commit.backup${NC}"
    cp .git/hooks/pre-commit .git/hooks/pre-commit.backup
fi

# Create the pre-commit hook
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash

# Git pre-commit hook with version checking
# This hook runs the version check script before allowing commits

# Run the version check script
./pre-commit-version-check.sh

# If version check script exits with non-zero, abort commit
if [ $? -ne 0 ]; then
    echo "Pre-commit version check failed. Commit aborted."
    exit 1
fi

# Continue with commit if version check passed
exit 0
EOF

# Make the hook executable
chmod +x .git/hooks/pre-commit

echo -e "${GREEN}✅ Pre-commit version check installed successfully!${NC}"
echo ""
echo -e "${BLUE}📖 How it works:${NC}"
echo "  • The hook automatically runs before every commit"
echo "  • It detects changes in directories with VERSION files"
echo "  • Prompts you to increment versions (major/minor/patch)"
echo "  • Updates VERSION files and stages them automatically"
echo ""
echo -e "${BLUE}📂 Monitored directories:${NC}"
echo "  • py-sdk/bharatml_commons"
echo "  • py-sdk/spark_feature_push_client"
echo "  • py-sdk/grpc_feature_client"
echo "  • horizon"
echo "  • go-sdk"
echo "  • online-feature-store"
echo "  • trufflebox-ui"
echo ""
echo -e "${YELLOW}💡 Usage tips:${NC}"
echo "  • Choose version increment type based on semantic versioning:"
echo "    - Major: Breaking changes (1.0.0 → 2.0.0)"
echo "    - Minor: New features (1.0.0 → 1.1.0)"
echo "    - Patch: Bug fixes (1.0.0 → 1.0.1)"
echo "  • Select 'Skip' if changes don't warrant a version bump"
echo "  • The script preserves 'v' prefix if present in VERSION files"
echo ""
echo -e "${GREEN}🎉 Setup complete! Try making a commit to test it out.${NC}" 