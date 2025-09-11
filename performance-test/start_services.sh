#!/bin/bash

# Performance Test Service Starter
# This script starts all three API services for performance testing
# Usage: ./start_services.sh [--console] [--limit-cpus[=CPUSET]]
#   --console              Start Rust with tokio-console support
#   --limit-cpus           Pin all services to CPUs 0-1 and tune threads (default)
#   --limit-cpus=CPUSET    Pin to explicit CPU set (e.g. 0-1,0,2)

set -e

# Parse flags
CONSOLE_MODE=false
LIMIT_CPUS=false
CPUSET="0-1"

for arg in "$@"; do
    case "$arg" in
        --console)
            CONSOLE_MODE=true
            echo "🔍 Console mode enabled for Rust service"
            ;;
        --limit-cpus)
            LIMIT_CPUS=true
            CPUSET="0-1"
            ;;
        --limit-cpus=*)
            LIMIT_CPUS=true
            CPUSET="${arg#*=}"
            ;;
    esac
done

# Derive CPU count from CPUSET when limiting CPUs
cpu_count_from_cpuset() {
    local cpuset="$1"
    local total=0
    local IFS=','
    read -ra parts <<< "$cpuset"
    for part in "${parts[@]}"; do
        if [[ "$part" =~ ^[0-9]+-[0-9]+$ ]]; then
            local IFS='-'
            read -r start end <<< "$part"
            if [[ -n "$start" && -n "$end" && $end -ge $start ]]; then
                total=$(( total + end - start + 1 ))
            fi
        elif [[ "$part" =~ ^[0-9]+$ ]]; then
            total=$(( total + 1 ))
        fi
    done
    echo "$total"
}

if [ "$LIMIT_CPUS" = true ]; then
    CPU_COUNT=$(cpu_count_from_cpuset "$CPUSET")
    if [ -z "$CPU_COUNT" ] || [ "$CPU_COUNT" -le 0 ]; then
        CPU_COUNT=2
    fi
    echo -e "${YELLOW}⚙️  CPU pinning enabled: cpuset=${CPUSET} (count=${CPU_COUNT})${NC}"

    # Decide pinning mechanism: prefer cgroups cpuset via cgcreate/cgexec, fallback to taskset
    CPU_PIN_PREFIX=""
    if command -v cgcreate >/dev/null 2>&1 && [ -d "/sys/fs/cgroup/cpuset" ]; then
        CGROUP_NAME="bench_${CPUSET//[^0-9a-zA-Z_-]/_}"
        echo -e "${YELLOW}🧰 Using cgroups cpuset via cgexec (cgroup=${CGROUP_NAME})${NC}"
        # Create cpuset cgroup and configure
        cgcreate -g cpuset:"$CGROUP_NAME" || true
        # Determine mems
        DEFAULT_MEMS="0"
        if [ -r "/sys/fs/cgroup/cpuset/cpuset.mems" ]; then
            DEFAULT_MEMS=$(cat /sys/fs/cgroup/cpuset/cpuset.mems)
        fi
        cgset -r cpuset.cpus="${CPUSET}" "$CGROUP_NAME" || true
        cgset -r cpuset.mems="${DEFAULT_MEMS}" "$CGROUP_NAME" || true
        CPU_PIN_PREFIX="cgexec -g cpuset:${CGROUP_NAME}"
    else
        if command -v taskset >/dev/null 2>&1; then
            echo -e "${YELLOW}🧰 Using taskset for CPU pinning${NC}"
            CPU_PIN_PREFIX="taskset -c $CPUSET"
        else
            echo -e "${YELLOW}⚠️  Neither cgcreate nor taskset found; CPU pinning will be skipped${NC}"
            CPU_PIN_PREFIX=""
        fi
    fi
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

JAVA_JVM_ARGS=""
if [ "$LIMIT_CPUS" = true ]; then
    JAVA_JVM_ARGS="-XX:ActiveProcessorCount=${CPU_COUNT} -XX:ParallelGCThreads=${CPU_COUNT} -XX:ConcGCThreads=${CPU_COUNT}"
fi

nohup $CPU_PIN_PREFIX java $JAVA_JVM_ARGS -jar target/java-caller-1.0.0.jar > ../performance-test/logs/java.log 2>&1 &
JAVA_PID=$!
echo "Java PID: $JAVA_PID"

# Start Rust service (Port 8080)
echo -e "${BLUE}🦀 Starting Rust Axum service (rust-caller-new)...${NC}"
cd ../rust-caller-new

if [ "$CONSOLE_MODE" = true ]; then
    echo -e "${YELLOW}🔍 Starting Rust with tokio-console support...${NC}"
    echo -e "${YELLOW}💡 Connect with: tokio-console${NC}"
    TOKIO_CONSOLE=1 RUSTFLAGS="--cfg tokio_unstable" nohup $CPU_PIN_PREFIX cargo run --release > ../performance-test/logs/rust_console.log 2>&1 &
    RUST_PID=$!
    echo "Rust PID: $RUST_PID (with console support on port 6669)"
else
    echo -e "${YELLOW}📊 Starting Rust in normal mode...${NC}"
    echo -e "${YELLOW}💡 For console mode, use: ./start_services.sh --console${NC}"
    RUST_ENV=""
    if [ "$LIMIT_CPUS" = true ]; then
        RUST_ENV="RAYON_NUM_THREADS=${CPU_COUNT} TOKIO_WORKER_THREADS=${CPU_COUNT}"
    fi
    nohup $CPU_PIN_PREFIX env $RUST_ENV cargo run --release > ../performance-test/logs/rust.log 2>&1 &
    RUST_PID=$!
    echo "Rust PID: $RUST_PID"
fi

# Start Go service (Port 8081)
echo -e "${BLUE}🐹 Starting Go Gin service...${NC}"
cd ../go-caller
GO_ENV=""
if [ "$LIMIT_CPUS" = true ]; then
    GO_ENV="GOMAXPROCS=${CPU_COUNT}"
fi
nohup $CPU_PIN_PREFIX env $GO_ENV go run main.go > ../performance-test/logs/go.log 2>&1 &
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
