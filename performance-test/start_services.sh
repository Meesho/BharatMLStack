#!/bin/bash

# Performance Test Service Starter
# This script starts all three API services for performance testing
# Usage: ./start_services.sh [--console]
#   --console: Start Rust with tokio-console support

set -e

# Check for console mode
CONSOLE_MODE=false
if [[ "$1" == "--console" ]]; then
    CONSOLE_MODE=true
    echo "🔍 Console mode enabled for Rust service"
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Starting BharatMLStack Performance Test Services${NC}"
echo "================================================================"

# Function to check if port is available
check_port() {
    local port=$1
    local service=$2
    if lsof -i :$port > /dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  Port $port is already in use. Killing existing process...${NC}"
        lsof -ti :$port | xargs kill -9 2>/dev/null || true
        sleep 2
    fi
}

# Function to wait for service to be ready
wait_for_service() {
    local port=$1
    local service=$2
    local max_attempts=90
    local attempt=1
    
    echo -e "${BLUE}⏳ Waiting for $service to start on port $port...${NC}"
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s http://localhost:$port/retrieve-features -X POST > /dev/null 2>&1; then
            echo -e "${GREEN}✅ $service is ready!${NC}"
            return 0
        fi
        
        echo -n "."
        sleep 1
        attempt=$((attempt + 1))
    done
    
    echo -e "${RED}❌ $service failed to start within $max_attempts seconds${NC}"
    return 1
}

# Create logs directory
mkdir -p logs

# Kill any existing processes
echo -e "${YELLOW}🧹 Cleaning up existing processes...${NC}"
pkill -f "java-caller" 2>/dev/null || true
pkill -f "rust-caller" 2>/dev/null || true
pkill -f "rust-caller-new" 2>/dev/null || true
pkill -f "go-caller" 2>/dev/null || true
sleep 2

# Check and clear ports
check_port 8080 "Rust"
check_port 8081 "Go" 
check_port 8082 "Java"

echo

# Start Java service (Port 8082)
echo -e "${BLUE}🔵 Starting Java Spring Boot service...${NC}"
cd ../java-caller
if [ ! -f "target/java-caller-1.0.0.jar" ]; then
    echo -e "${YELLOW}📦 Building Java application...${NC}"
    mvn clean package -q
fi
nohup java -jar target/java-caller-1.0.0.jar > ../performance-test/logs/java.log 2>&1 &
JAVA_PID=$!
echo "Java PID: $JAVA_PID"

# Start Rust service (Port 8080)
echo -e "${BLUE}🦀 Starting Rust Axum service (rust-caller-new)...${NC}"
cd ../rust-caller-new

if [ "$CONSOLE_MODE" = true ]; then
    echo -e "${YELLOW}🔍 Starting Rust with tokio-console support...${NC}"
    echo -e "${YELLOW}💡 Connect with: tokio-console${NC}"
    TOKIO_CONSOLE=1 RUSTFLAGS="--cfg tokio_unstable" nohup cargo run --release > ../performance-test/logs/rust_console.log 2>&1 &
    RUST_PID=$!
    echo "Rust PID: $RUST_PID (with console support on port 6669)"
else
    echo -e "${YELLOW}📊 Starting Rust in normal mode...${NC}"
    echo -e "${YELLOW}💡 For console mode, use: ./start_services.sh --console${NC}"
    nohup cargo run --release > ../performance-test/logs/rust.log 2>&1 &
    RUST_PID=$!
    echo "Rust PID: $RUST_PID"
fi

# Start Go service (Port 8081)
echo -e "${BLUE}🐹 Starting Go Gin service...${NC}"
cd ../go-caller
nohup go run main.go > ../performance-test/logs/go.log 2>&1 &
GO_PID=$!
echo "Go PID: $GO_PID"

cd ../performance-test

echo
echo -e "${BLUE}⏳ Waiting for all services to become ready...${NC}"
echo

# Wait for all services
SERVICES_READY=true

if ! wait_for_service 8082 "Java"; then
    SERVICES_READY=false
fi

if ! wait_for_service 8080 "Rust"; then
    SERVICES_READY=false
fi

if ! wait_for_service 8081 "Go"; then
    SERVICES_READY=false
fi

echo

if [ "$SERVICES_READY" = true ]; then
    echo -e "${GREEN}🎉 All services are ready for performance testing!${NC}"
    echo
    echo -e "${BLUE}📊 Service URLs:${NC}"
    echo "  • Java:  http://localhost:8082/retrieve-features"
    echo "  • Rust:  http://localhost:8080/retrieve-features"
    echo "  • Go:    http://localhost:8081/retrieve-features"
    echo
    if [ "$CONSOLE_MODE" = true ]; then
        echo -e "${GREEN}🔍 Profiling & Monitoring:${NC}"
        echo "  • Rust tokio-console: tokio-console (port 6669)"
        echo "  • Go pprof server: http://localhost:6060/debug/pprof/"
        echo "  • Java actuator: http://localhost:8082/actuator/"
        echo
    fi
    echo -e "${BLUE}💾 Process IDs (for cleanup):${NC}"
    echo "  • Java PID: $JAVA_PID"
    echo "  • Rust PID: $RUST_PID"
    echo "  • Go PID: $GO_PID"
    echo
    echo -e "${YELLOW}📝 Logs are available in:${NC}"
    echo "  • Java: logs/java.log"
    if [ "$CONSOLE_MODE" = true ]; then
        echo "  • Rust: logs/rust_console.log"
    else
        echo "  • Rust: logs/rust.log"
    fi
    echo "  • Go: logs/go.log"
    echo
    echo -e "${GREEN}✨ Ready to run performance tests!${NC}"
    echo -e "${BLUE}Run: locust -f locustfile.py --host http://localhost${NC}"
else
    echo -e "${RED}❌ Some services failed to start. Check the logs for details.${NC}"
    exit 1
fi
