# BharatMLStack Performance Testing Suite

## Overview
Compares performance between Java (Spring Boot), Rust (Axum), and Go (Gin) services for the feature store API.

## Files
- **`simple_test.py`** - Main performance testing script with CPU/memory monitoring 
- **`start_services.sh`** - Starts all three API services
- **`stop_services.sh`** - Stops all API services
- **`requirements.txt`** - Python dependencies

## Usage

### Manual Usage
```bash
# 1. Start services
./start_services.sh

# 2. Run performance test
python3 simple_test.py 50

# 3. Stop services
./stop_services.sh
```

## What it measures
- **RPS** (Requests Per Second)
- **CPU Usage** (%)
- **CPU Efficiency** (RPS per 1% CPU)
- **Latency** (Average & P95 response times)
- **Memory Usage** (MB)

## Output
Provides comparison table showing which service wins for:
- CPU Efficiency
- Lowest Latency
- Scaling Recommendations

## Requirements
- Python 3.8+
- All three services running (Java, Rust, Go)
