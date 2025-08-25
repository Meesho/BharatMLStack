#!/bin/bash

# Local Performance Test Runner with Virtual Environment
# This script handles the virtual environment activation automatically

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔬 BharatMLStack Local Performance Test${NC}"
echo "=================================================="

# Check if virtual environment exists
if [ ! -d "venv" ]; then
    echo -e "${YELLOW}📦 Creating virtual environment...${NC}"
    python3 -m venv venv
    source venv/bin/activate
    pip install psutil requests locust
else
    echo -e "${GREEN}✅ Using existing virtual environment${NC}"
    source venv/bin/activate
fi

# Parse RPS argument
if [ $# -eq 0 ]; then
    RPS=50
    echo -e "${BLUE}🎯 Using default target: ${RPS} RPS${NC}"
else
    RPS=$1
    echo -e "${BLUE}🎯 Target RPS: ${RPS}${NC}"
fi

echo -e "${BLUE}📊 Starting local performance test...${NC}"
echo

# Run the test
python3 simple_test.py $RPS

echo
echo -e "${GREEN}🎉 Test completed!${NC}"
echo -e "${YELLOW}💡 To run again: ./run_local_test.sh [RPS]${NC}"
echo -e "${YELLOW}   Examples:${NC}"
echo -e "${YELLOW}   ./run_local_test.sh 30   # Light load${NC}"
echo -e "${YELLOW}   ./run_local_test.sh 100  # Medium load${NC}"
echo -e "${YELLOW}   ./run_local_test.sh 200  # Higher load${NC}"
