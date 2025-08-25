#!/bin/bash

# Performance Test Service Stopper
# This script stops all running API services

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🛑 Stopping BharatMLStack Performance Test Services${NC}"
echo "================================================================"

# Function to kill process by port
kill_by_port() {
    local port=$1
    local service=$2
    
    echo -e "${YELLOW}🔍 Checking for $service on port $port...${NC}"
    
    if lsof -i :$port > /dev/null 2>&1; then
        echo -e "${BLUE}⏹️  Stopping $service...${NC}"
        lsof -ti :$port | xargs kill -9 2>/dev/null
        echo -e "${GREEN}✅ $service stopped${NC}"
    else
        echo -e "${YELLOW}ℹ️  $service was not running${NC}"
    fi
}

# Kill services by port
kill_by_port 8082 "Java service"
kill_by_port 8080 "Rust service"
kill_by_port 8081 "Go service"

# Also kill by process name (backup)
echo -e "${YELLOW}🧹 Cleaning up any remaining processes...${NC}"

pkill -f "java-caller" 2>/dev/null && echo -e "${GREEN}✅ Java processes cleaned${NC}" || true
pkill -f "rust-caller" 2>/dev/null && echo -e "${GREEN}✅ Rust processes cleaned${NC}" || true
pkill -f "go-caller" 2>/dev/null && echo -e "${GREEN}✅ Go processes cleaned${NC}" || true

echo
echo -e "${GREEN}🎉 All services stopped successfully!${NC}"
echo -e "${BLUE}📝 Logs are preserved in the logs/ directory${NC}"
