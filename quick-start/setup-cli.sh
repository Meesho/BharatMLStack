#!/bin/bash

# ONFS CLI Setup Script
# This script builds and installs the ONFS CLI tool for testing persist, retrieve, and retrieve-decoded operations

set -e

echo "🚀 Setting up ONFS CLI Tool..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Error: Go is not installed. Please install Go 1.22 or later.${NC}"
    echo "Visit: https://golang.org/doc/install"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.22"

if ! printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -C -V; then
    echo -e "${RED}❌ Error: Go version $GO_VERSION is too old. Required: $REQUIRED_VERSION or later.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Go version $GO_VERSION detected${NC}"



# Navigate to go-sdk directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_SDK_DIR="$SCRIPT_DIR/../go-sdk"

if [ ! -d "$GO_SDK_DIR" ]; then
    echo -e "${RED}❌ Error: go-sdk directory not found at $GO_SDK_DIR${NC}"
    exit 1
fi

cd "$GO_SDK_DIR"
echo -e "${BLUE}📁 Working in: $(pwd)${NC}"

# Download dependencies
echo -e "${YELLOW}📦 Downloading Go dependencies...${NC}"
go mod tidy
go mod download

# Build the CLI tool
echo -e "${YELLOW}🔨 Building ONFS CLI tool...${NC}"
go version
go build -o onfs-cli ./cmd/main.go

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Build successful${NC}"

# Create bin directory in user's home if it doesn't exist
BIN_DIR="$HOME/bin"
if [ ! -d "$BIN_DIR" ]; then
    mkdir -p "$BIN_DIR"
    echo -e "${BLUE}📁 Created $BIN_DIR directory${NC}"
fi

# Install the binary
echo -e "${YELLOW}📥 Installing onfs-cli to $BIN_DIR...${NC}"
cp onfs-cli "$BIN_DIR/"
chmod +x "$BIN_DIR/onfs-cli"

# Check if ~/bin is in PATH
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo -e "${YELLOW}⚠️  $BIN_DIR is not in your PATH${NC}"
    echo -e "${BLUE}Add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):${NC}"
    echo -e "${GREEN}export PATH=\"$BIN_DIR:\$PATH\"${NC}"
    echo -e "${BLUE}Then restart your terminal or run: source ~/.bashrc (or ~/.zshrc)${NC}"
fi

# Test the installation
echo -e "${YELLOW}🧪 Testing installation...${NC}"
if "$BIN_DIR/onfs-cli" --help > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Installation successful!${NC}"
else
    echo -e "${RED}❌ Installation test failed${NC}"
    exit 1
fi

# Create sample JSON files
SAMPLES_DIR="$SCRIPT_DIR/sample-data"
mkdir -p "$SAMPLES_DIR"

echo -e "${YELLOW}📝 Creating sample JSON files...${NC}"

# Create persist sample
cat > "$SAMPLES_DIR/persist-sample.json" << 'EOF'
{
  "entity_label": "user",
  "keys_schema": ["user_id"],
  "feature_groups": [
    {
      "label": "user_features",
      "feature_labels": ["age", "location", "subscription_type"]
    }
  ],
  "data": [
    {
      "key_values": ["user_123"],
      "feature_values": [
        {
          "values": {
            "int32_values": [28],
            "string_values": ["NYC", "premium"]
          }
        }
      ]
    },
    {
      "key_values": ["user_456"],
      "feature_values": [
        {
          "values": {
            "int32_values": [35],
            "string_values": ["SF", "basic"]
          }
        }
      ]
    }
  ]
}
EOF

# Create retrieve sample
cat > "$SAMPLES_DIR/retrieve-sample.json" << 'EOF'
{
  "entity_label": "user",
  "feature_groups": [
    {
      "label": "user_features",
      "feature_labels": ["age", "location", "subscription_type"]
    }
  ],
  "keys_schema": ["user_id"],
  "keys": [
    {
      "cols": ["user_123"]
    },
    {
      "cols": ["user_456"]
    }
  ]
}
EOF

echo -e "${GREEN}✅ Sample files created in $SAMPLES_DIR${NC}"

# Display usage information
echo -e "\n${BLUE}🎉 ONFS CLI Tool is ready to use!${NC}"
echo -e "\n${YELLOW}📖 Usage Examples:${NC}"
echo -e "${GREEN}# Show help${NC}"
echo -e "onfs-cli --help"
echo -e "\n${GREEN}# Persist features using JSON file${NC}"
echo -e "onfs-cli -operation persist -input $SAMPLES_DIR/persist-sample.json"
echo -e "\n${GREEN}# Retrieve features using JSON file${NC}"
echo -e "onfs-cli -operation retrieve -input $SAMPLES_DIR/retrieve-sample.json"
echo -e "\n${GREEN}# Retrieve decoded features${NC}"
echo -e "onfs-cli -operation retrieve-decoded -input $SAMPLES_DIR/retrieve-sample.json"
echo -e "\n${GREEN}# Interactive mode (no JSON file)${NC}"
echo -e "onfs-cli -operation persist"
echo -e "\n${GREEN}# Connect to different host/port${NC}"
echo -e "onfs-cli -operation retrieve -host localhost -port 8089 -input $SAMPLES_DIR/retrieve-sample.json"

echo -e "\n${YELLOW}📋 Prerequisites:${NC}"
echo -e "• Ensure ONFS service is running (docker-compose up -d)"
echo -e "• Service should be accessible at localhost:8089 (or specified host:port)"
echo -e "• Auth token 'test' is used by default (matches docker-compose setup)"

echo -e "\n${BLUE}📁 Sample Files Location:${NC}"
echo -e "$SAMPLES_DIR/"
echo -e "├── persist-sample.json    # Sample persist request"
echo -e "└── retrieve-sample.json   # Sample retrieve request"

echo -e "\n${GREEN}🎯 Ready to test your ONFS deployment!${NC}" 