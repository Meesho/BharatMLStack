#!/usr/bin/env python3
"""
Simple CPU + Latency Performance Test
Measures both CPU efficiency and latency for Java, Rust, and Go APIs
"""

import subprocess
import time
import psutil
import requests
import json
from datetime import datetime

def check_services():
    """Quick health check"""
    services = [
        ("Java", "http://localhost:8082/retrieve-features"),
        ("Rust", "http://localhost:8080/retrieve-features"),
        ("Go", "http://localhost:8081/retrieve-features")
    ]
    
    print("🔍 Checking services...")
    for name, url in services:
        try:
            response = requests.post(url, timeout=5)
            status = "✅" if response.status_code == 200 else "❌"
            print(f"{status} {name}: {response.status_code}")
        except:
            print(f"❌ {name}: Not responding")

def find_service_pids():
    """Find process IDs for monitoring CPU"""
    pids = {}
    
    # First pass: look for actual binaries (preferred)
    java_pids = []
    rust_pids = []
    go_pids = []
    
    for proc in psutil.process_iter(['pid', 'name', 'cmdline']):
        try:
            cmdline = ' '.join(proc.info['cmdline'])
            
            # Java: java -jar target/java-caller-1.0.0.jar
            if 'java-caller' in cmdline and 'java' in cmdline:
                java_pids.append(proc.info['pid'])
            
            # Rust: compiled binary (preferred) or cargo process
            elif 'rust-caller' in cmdline:
                if 'target/release' in cmdline:
                    rust_pids.insert(0, proc.info['pid'])  # Prefer compiled binary
                elif 'cargo' in cmdline:
                    rust_pids.append(proc.info['pid'])
            
            # Go: compiled binary (preferred) or go run wrapper
            elif '/go-build/' in cmdline and '/main' in cmdline:
                go_pids.insert(0, proc.info['pid'])  # Prefer compiled binary
            elif 'go run main.go' in cmdline:
                go_pids.append(proc.info['pid'])
                
        except:
            continue
    
    # Take the first (preferred) PID for each service
    if java_pids:
        pids['java'] = java_pids[0]
    if rust_pids:
        pids['rust'] = rust_pids[0]
    if go_pids:
        pids['go'] = go_pids[0]
    
    print("📍 Found service PIDs:")
    for service, pid in pids.items():
        print(f"  {service.capitalize()}: {pid}")
    
    return pids

def run_locust_test(service_name, port, users=20, duration=60, spawn_rate=5):
    """Run individual service test"""
    print(f"\n🚀 Testing {service_name} (Port {port}) - {users} users, {duration}s")
    
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    output_prefix = f"results/{service_name}_{timestamp}"
    
    # Create simple locust command for individual service
    cmd = [
        "locust", "-f", "simple_locustfile.py",
        "--host", f"http://localhost:{port}",
        "--users", str(users),
        "--spawn-rate", str(spawn_rate),
        "--run-time", f"{duration}s",
        "--headless",
        "--html", f"{output_prefix}.html",
        "--csv", f"{output_prefix}"
    ]
    
    # Monitor CPU during test
    pids = find_service_pids()
    service_pid = pids.get(service_name.lower())
    
    if service_pid:
        # Start test
        print(f"⏳ Starting {duration}s test...")
        process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        
        # Monitor CPU usage during test
        cpu_readings = []
        memory_readings = []
        
        # Initial CPU reading to "prime" the measurement
        try:
            proc = psutil.Process(service_pid)
            proc.cpu_percent()  # First call to initialize
            time.sleep(0.1)  # Small delay
        except:
            print(f"   ⚠️  Could not initialize CPU monitoring for PID {service_pid}")
        
        print(f"   💾 Monitoring CPU usage every 5 seconds...")
        
        for i in range(duration):
            try:
                proc = psutil.Process(service_pid)
                cpu_percent = proc.cpu_percent(interval=1.0)  # 1-second interval for accuracy
                memory_mb = proc.memory_info().rss / (1024 * 1024)
                
                if cpu_percent > 0:  # Only record non-zero CPU readings
                    cpu_readings.append(cpu_percent)
                memory_readings.append(memory_mb)
                
                if i % 5 == 0:  # Print every 5 seconds
                    print(f"   {i:3d}s: CPU {cpu_percent:5.1f}%, Memory {memory_mb:6.1f}MB")
                
                # Don't sleep here since cpu_percent(interval=1.0) already waits 1 second
            except Exception as e:
                print(f"   ⚠️  Error monitoring PID {service_pid}: {e}")
                break
        
        # Wait for locust to finish
        stdout, stderr = process.communicate()
        
        # Calculate averages
        avg_cpu = sum(cpu_readings) / len(cpu_readings) if cpu_readings else 0
        avg_memory = sum(memory_readings) / len(memory_readings) if memory_readings else 0
        
        return {
            'avg_cpu': avg_cpu,
            'avg_memory': avg_memory,
            'output_prefix': output_prefix
        }
    else:
        print(f"❌ Could not find PID for {service_name}")
        return None

def analyze_results(service_name, resource_data, output_prefix):
    """Analyze results and show CPU efficiency + latency"""
    stats_file = f"{output_prefix}_stats.csv"
    
    try:
        # Read locust results
        with open(stats_file, 'r') as f:
            lines = f.readlines()
            if len(lines) > 1:
                # Parse the main results line (skip header)
                data = lines[1].split(',')
                
                rps = float(data[9])  # Requests/s (column 9)
                avg_response = float(data[5])  # Average Response Time
                p95_response = float(data[15]) if len(data) > 15 else 0  # 95%
                max_response = float(data[7])  # Max Response Time
                
                # Calculate efficiency
                cpu_efficiency = rps / resource_data['avg_cpu'] if resource_data['avg_cpu'] > 0 else 0
                
                return {
                    'service': service_name,
                    'rps': rps,
                    'avg_cpu': resource_data['avg_cpu'],
                    'avg_memory': resource_data['avg_memory'],
                    'cpu_efficiency': cpu_efficiency,
                    'avg_response_ms': avg_response,
                    'p95_response_ms': p95_response,
                    'max_response_ms': max_response
                }
    except Exception as e:
        print(f"Error analyzing {service_name}: {e}")
    
    return None

def calculate_load_config(target_rps):
    """Calculate users and spawn rate for target RPS"""
    # Assumption: Average wait time between requests is ~0.3 seconds
    # RPS ≈ Users / Average_Wait_Time
    # Therefore: Users ≈ RPS * Average_Wait_Time
    
    avg_wait_time = 0.3  # seconds (from locustfile wait_time)
    users = max(int(target_rps * avg_wait_time), 1)
    
    # Set spawn rate as 10% of users (reasonable ramp-up)
    spawn_rate = max(int(users * 0.1), 1)
    
    # Duration based on RPS level
    if target_rps <= 50:
        duration = 60      # 1 minute for low RPS
    elif target_rps <= 200:
        duration = 90      # 1.5 minutes for medium RPS
    else:
        duration = 120     # 2 minutes for high RPS
    
    return {
        'users': users,
        'duration': duration,
        'spawn_rate': spawn_rate,
        'target_rps': target_rps
    }

def main():
    """Run the complete test"""
    import sys
    
    # Parse command line arguments for target RPS
    if len(sys.argv) > 1:
        try:
            target_rps = int(sys.argv[1])
            if target_rps <= 0:
                raise ValueError("RPS must be positive")
        except ValueError:
            print("❌ Error: Please provide a valid RPS number")
            print("📖 Usage examples:")
            print("   python3 simple_test.py 50    # Target 50 RPS")
            print("   python3 simple_test.py 200   # Target 200 RPS") 
            print("   python3 simple_test.py 500   # Target 500 RPS")
            print("   python3 simple_test.py       # Default 30 RPS")
            return
    else:
        target_rps = 30  # Default target RPS
    
    # Calculate load configuration
    test_config = calculate_load_config(target_rps)
    
    print(f"🔬 Simple CPU + Latency Performance Test")
    print(f"🎯 Target: {test_config['target_rps']} RPS")
    print(f"📊 Configuration: {test_config['users']} users, {test_config['duration']}s, spawn rate {test_config['spawn_rate']}/s")
    print("=" * 80)
    
    # Check services
    check_services()
    
    # Create results directory
    subprocess.run(["mkdir", "-p", "results"], check=False)
    
    # Test each service
    services = [
        ("Java", 8082),
        ("Rust", 8080), 
        ("Go", 8081)
    ]
    
    results = []
    
    for i, (service_name, port) in enumerate(services):
        # Add 1-minute delay between services (except for the first one)
        if i > 0:
            print(f"\n⏰ Waiting 60 seconds before testing {service_name} (cooldown period)...")
            for countdown in range(60, 0, -10):
                print(f"   ⏳ {countdown} seconds remaining...")
                time.sleep(10)
            print(f"   ✅ Starting {service_name} test now!\n")
        resource_data = run_locust_test(
            service_name, port, 
            users=test_config['users'], 
            duration=test_config['duration'],
            spawn_rate=test_config['spawn_rate']
        )
        if resource_data:
            analysis = analyze_results(service_name, resource_data, resource_data['output_prefix'])
            if analysis:
                results.append(analysis)
    
    # Show comparison
    if results:
        print("\n" + "=" * 80)
        print("📊 PERFORMANCE COMPARISON - CPU EFFICIENCY + LATENCY")
        print("=" * 80)
        
        print(f"{'Service':<8} | {'RPS':<6} | {'CPU%':<6} | {'RPS/CPU%':<8} | {'Avg(ms)':<8} | {'P95(ms)':<8} | {'Memory':<8}")
        print("-" * 80)
        
        # Sort by CPU efficiency
        results.sort(key=lambda x: x['cpu_efficiency'], reverse=True)
        
        for r in results:
            print(f"{r['service']:<8} | {r['rps']:5.1f} | {r['avg_cpu']:5.1f} | "
                  f"{r['cpu_efficiency']:6.2f} | {r['avg_response_ms']:6.1f} | "
                  f"{r['p95_response_ms']:6.1f} | {r['avg_memory']:6.1f}MB")
        
        # Summary  
        best_cpu = results[0]
        best_latency = min(results, key=lambda x: x['avg_response_ms'])
        
        print(f"\n🏆 WINNERS at {test_config['target_rps']} RPS target:")
        print(f"  CPU Efficiency: {best_cpu['service']} ({best_cpu['cpu_efficiency']:.2f} RPS/CPU%)")
        print(f"  Lowest Latency: {best_latency['service']} ({best_latency['avg_response_ms']:.1f}ms avg)")
        
        print(f"\n💡 CPU EFFICIENCY INSIGHTS:")
        print(f"  • At 10% CPU: {best_cpu['service']} could handle ~{best_cpu['cpu_efficiency'] * 10:.0f} RPS")
        print(f"  • At 20% CPU: {best_cpu['service']} could handle ~{best_cpu['cpu_efficiency'] * 20:.0f} RPS")
        print(f"  • At 50% CPU: {best_cpu['service']} could handle ~{best_cpu['cpu_efficiency'] * 50:.0f} RPS")
        
        print(f"\n📈 SCALING RECOMMENDATIONS:")
        for r in results:
            max_sustainable_rps = r['cpu_efficiency'] * 70  # Assuming 70% CPU max for stability
            print(f"  • {r['service']}: Max sustainable ~{max_sustainable_rps:.0f} RPS (at 70% CPU)")
        
        print(f"\n🎯 TO TEST DIFFERENT LOADS:")
        print(f"  python3 simple_test.py 100   # Test at 100 RPS")
        print(f"  python3 simple_test.py 500   # Test at 500 RPS")
        print(f"  python3 simple_test.py 1000  # Test at 1000 RPS")

if __name__ == "__main__":
    main()
