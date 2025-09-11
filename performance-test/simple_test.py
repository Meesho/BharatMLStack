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
import os
import argparse
from datetime import datetime
import shutil

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

def _pid_listening_on_port(port: int):
    """Return PID listening on the given port: psutil (inet/tcp) then lsof fallback."""
    # Check IPv4/IPv6 via psutil
    for kind in ("inet", "tcp"):
        try:
            for conn in psutil.net_connections(kind=kind):
                try:
                    if conn.laddr and conn.laddr.port == port and conn.status == psutil.CONN_LISTEN and conn.pid:
                        return conn.pid
                except Exception:
                    continue
        except Exception:
            continue

    # Minimal fallback: lsof by port
    try:
        if shutil.which('lsof'):
            proc = subprocess.run(
                ['lsof', '-nP', f'-iTCP:{port}', '-sTCP:LISTEN', '-t'],
                stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True, check=False
            )
            for line in proc.stdout.splitlines():
                line = line.strip()
                if line.isdigit():
                    return int(line)
    except Exception:
        pass

    return None


def find_service_pids():
    """Find process IDs for monitoring CPU using listening ports (robust for 'go run')."""
    pids = {}
    port_map = {
        'java': 8082,
        'rust': 8080,
        'go': 8081,
    }

    for name, port in port_map.items():
        pid = _pid_listening_on_port(port)
        if pid:
            pids[name] = pid

    print("📍 Found service PIDs:")
    for service, pid in pids.items():
        try:
            cmd = ' '.join(psutil.Process(pid).cmdline())
        except Exception:
            cmd = "<unknown>"
        print(f"  {service.capitalize()}: {pid} | {cmd}")

    return pids

def run_locust_test(service_name, port, users=20, duration=60, spawn_rate=5):
    """Run individual service test"""
    print(f"\n🚀 Testing {service_name} (Port {port}) - {users} users, {duration}s")
    
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    output_prefix = f"results/{service_name}_{timestamp}"
    
    # Ensure results directory exists
    os.makedirs("results", exist_ok=True)
    
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
    
    # Start test
    print(f"⏳ Starting {duration}s test...")
    print(f"💾 Monitoring SYSTEM CPU usage every 5 seconds...")
    print(f"⚙️  Test config: {users} users, {spawn_rate} spawn rate")
    process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    # Monitor system-wide CPU and memory during test
    cpu_readings = []
    memory_readings = []

    # Prime CPU measurement
    psutil.cpu_percent(interval=None)
    time.sleep(0.1)

    for i in range(duration):
        try:
            cpu_percent = psutil.cpu_percent(interval=1.0)
            mem = psutil.virtual_memory()
            memory_mb = mem.used / (1024 * 1024)

            cpu_readings.append(cpu_percent)
            memory_readings.append(memory_mb)

            if i % 5 == 0:
                print(f"   {i:3d}s: CPU {cpu_percent:5.1f}%, Memory {memory_mb:6.1f}MB")
        except Exception as e:
            print(f"   ⚠️  Error monitoring system resources: {e}")
            break

    # Wait for locust to finish
    print(f"⏳ Waiting for Locust to finish and generate reports...")
    stdout, stderr = process.communicate()

    print(f"✅ Locust test completed successfully!")

    # Calculate averages
    avg_cpu = sum(cpu_readings) / len(cpu_readings) if cpu_readings else 0
    avg_memory = sum(memory_readings) / len(memory_readings) if memory_readings else 0

    print(f"📊 Processing results...")
    print(f"   Average CPU: {avg_cpu:.1f}% | Average Memory: {avg_memory:.1f}MB")

    return {
        'avg_cpu': avg_cpu,
        'avg_memory': avg_memory,
        'output_prefix': output_prefix
    }

def analyze_results(service_name, resource_data, output_prefix):
    """Analyze results and show comprehensive throughput + efficiency metrics"""
    stats_file = f"{output_prefix}_stats.csv"
    
    try:
        # Read locust results
        with open(stats_file, 'r') as f:
            lines = f.readlines()
            if len(lines) > 1:
                # Find the data line (skip headers and endpoint-specific lines)
                data_line = None
                for line in lines[1:]:  # Skip header
                    fields = line.strip().split(',')
                    if len(fields) > 10 and fields[1] == 'Aggregated':  # Look for Aggregated row (Name column)
                        data_line = fields
                        break
                
                if not data_line:
                    # If no Aggregated row found, use the first non-header line with valid data
                    for line in lines[1:]:
                        fields = line.strip().split(',')
                        if len(fields) > 10:
                            try:
                                # Test if we can parse key numeric fields
                                float(fields[9])  # RPS
                                float(fields[5])  # Avg response time
                                data_line = fields
                                break
                            except (ValueError, IndexError):
                                continue
                
                if not data_line:
                    print(f"❌ Could not find valid data row in {stats_file}")
                    return None
                
                print(f"✅ Using CSV row: {data_line[1]} ({data_line[0]})")
                data = data_line
                
                # Core throughput metrics
                rps = float(data[9])  # Requests/s (column 9)
                total_requests = int(float(data[2]))  # Request Count
                failures = int(float(data[3]))  # Failure Count
                
                # Latency metrics
                avg_response = float(data[5])  # Average Response Time
                min_response = float(data[6])  # Min Response Time
                max_response = float(data[7])  # Max Response Time
                p95_response = float(data[15]) if len(data) > 15 and data[15] else 0  # 95%
                p99_response = float(data[16]) if len(data) > 16 and data[16] else 0  # 99%
                
                # Calculate throughput quality metrics
                error_rate = (failures / total_requests * 100) if total_requests > 0 else 0
                
                # Calculate efficiency ratios
                cpu_efficiency = rps / resource_data['avg_cpu'] if resource_data['avg_cpu'] > 0 else 0
                memory_efficiency = rps / resource_data['avg_memory'] if resource_data['avg_memory'] > 0 else 0
                
                # Throughput assessment
                throughput_quality = "Excellent" if error_rate < 1 and p95_response < 500 else \
                                   "Good" if error_rate < 5 and p95_response < 1000 else \
                                   "Poor" if error_rate > 10 or p95_response > 2000 else "Fair"
                
                return {
                    'service': service_name,
                    'rps': rps,
                    'total_requests': total_requests,
                    'failures': failures,
                    'error_rate': error_rate,
                    'avg_cpu': resource_data['avg_cpu'],
                    'avg_memory': resource_data['avg_memory'],
                    'cpu_efficiency': cpu_efficiency,
                    'memory_efficiency': memory_efficiency,
                    'avg_response_ms': avg_response,
                    'min_response_ms': min_response,
                    'max_response_ms': max_response,
                    'p95_response_ms': p95_response,
                    'p99_response_ms': p99_response,
                    'throughput_quality': throughput_quality
                }
    except Exception as e:
        print(f"❌ Error analyzing {service_name}: {e}")
        # Debug: Show the contents of the CSV file
        if os.path.exists(stats_file):
            print(f"📄 Debug - CSV contents of {stats_file}:")
            with open(stats_file, 'r') as f:
                lines = f.readlines()
                for i, line in enumerate(lines[:5]):  # Show first 5 lines
                    print(f"  Line {i}: {line.strip()}")
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
    parser = argparse.ArgumentParser(description='Simple CPU + Latency Performance Test')
    parser.add_argument('rps', nargs='?', type=int, default=30, 
                       help='Target RPS (default: 30)')
    parser.add_argument('--service', choices=['java', 'rust', 'go'], 
                       help='Test only a specific service (default: test all)')
    parser.add_argument('--duration', type=int, 
                       help='Override test duration in seconds')
    
    args = parser.parse_args()
    
    # Validate RPS
    if args.rps <= 0:
        print("❌ Error: RPS must be positive")
        print("📖 Usage examples:")
        print("   python3 simple_test.py 50                    # Target 50 RPS (all services)")
        print("   python3 simple_test.py 200 --service rust    # Target 200 RPS (Rust only)")
        print("   python3 simple_test.py 500 --service java    # Target 500 RPS (Java only)") 
        print("   python3 simple_test.py 100 --duration 120    # Target 100 RPS, 120s duration")
        return
    
    target_rps = args.rps
    
    # Calculate load configuration
    test_config = calculate_load_config(target_rps)
    
    # Override duration if specified
    if args.duration:
        test_config['duration'] = args.duration
    
    # Filter services based on --service flag
    all_services = [
        ("Java", 8082),
        ("Rust", 8080), 
        ("Go", 8081)
    ]
    
    if args.service:
        service_map = {"java": ("Java", 8082), "rust": ("Rust", 8080), "go": ("Go", 8081)}
        services = [service_map[args.service]]
        test_mode = f"Single Service ({args.service.capitalize()})"
    else:
        services = all_services
        test_mode = "All Services"
    
    print(f"🔬 Simple CPU + Latency Performance Test")
    print(f"🎯 Target: {test_config['target_rps']} RPS")
    print(f"🔧 Mode: {test_mode}")
    print(f"📊 Configuration: {test_config['users']} users, {test_config['duration']}s, spawn rate {test_config['spawn_rate']}/s")
    print("=" * 80)
    
    # Check services
    check_services()
    
    # Create results directory
    subprocess.run(["mkdir", "-p", "results"], check=False)
    
    results = []
    
    for i, (service_name, port) in enumerate(services):
        # Add 1-minute delay between services (except for the first one and single service mode)
        if i > 0 and not args.service:
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
        print("\n" + "=" * 90)
        if args.service:
            print(f"📊 SINGLE SERVICE PERFORMANCE ANALYSIS - {args.service.upper()}")
        else:
            print("📊 COMPREHENSIVE THROUGHPUT & EFFICIENCY ANALYSIS")
        print("=" * 90)
        
        # Throughput Quality Overview
        print(f"\n🎯 THROUGHPUT QUALITY at {test_config['target_rps']} RPS target:")
        print("-" * 65)
        for r in results:
            quality_emoji = {"Excellent": "🟢", "Good": "🟡", "Fair": "🟠", "Poor": "🔴"}
            emoji = quality_emoji.get(r['throughput_quality'], "⚪")
            print(f"   {r['service']:<6}: {emoji} {r['throughput_quality']:<9} | "
                  f"{r['rps']:5.1f} RPS | {r['error_rate']:4.1f}% errors | "
                  f"P95: {r['p95_response_ms']:5.1f}ms")
        
        # Main Performance Table
        print(f"\n📈 PERFORMANCE METRICS")
        print("-" * 90)
        print(f"{'Service':<8} | {'RPS':<6} | {'Errors%':<7} | {'Avg(ms)':<7} | {'P95(ms)':<7} | {'Mem(MB)':<7} | {'RPS/MB':<6}")
        print("-" * 90)
        
        # Sort by effective throughput (RPS adjusted for errors)
        results.sort(key=lambda x: x['rps'] * (1 - x['error_rate']/100), reverse=True)
        
        for r in results:
            print(f"{r['service']:<8} | {r['rps']:5.1f} | {r['error_rate']:6.1f} | "
                  f"{r['avg_response_ms']:6.1f} | {r['p95_response_ms']:6.1f} | "
                  f"{r['avg_memory']:6.1f} | {r['memory_efficiency']:5.2f}")
        
        # Resource Efficiency
        print(f"\n⚡ RESOURCE EFFICIENCY")
        print("-" * 70)
        print(f"{'Service':<8} | {'CPU%':<6} | {'RPS/CPU%':<8} | {'Memory(MB)':<10} | {'CPU Efficiency':<12}")
        print("-" * 70)
        
        for r in results:
            print(f"{r['service']:<8} | {r['avg_cpu']:5.1f} | {r['cpu_efficiency']:7.2f} | "
                  f"{r['avg_memory']:9.1f} | {'High' if r['cpu_efficiency'] > 1.0 else 'Medium' if r['cpu_efficiency'] > 0.5 else 'Low':<12}")
        
        # Winners Analysis (only for multi-service comparison)
        if len(results) > 1:
            best_throughput = max(results, key=lambda x: x['rps'])
            best_latency = min(results, key=lambda x: x['p95_response_ms'])
            best_memory = max(results, key=lambda x: x['memory_efficiency'])
            best_quality = max(results, key=lambda x: {"Excellent": 4, "Good": 3, "Fair": 2, "Poor": 1}[r['throughput_quality']])
            
            print(f"\n🏆 PERFORMANCE LEADERS:")
            print(f"   🚀 Highest Throughput: {best_throughput['service']} ({best_throughput['rps']:.1f} RPS)")
            print(f"   ⚡ Best Latency (P95): {best_latency['service']} ({best_latency['p95_response_ms']:.1f}ms)")
            print(f"   💾 Memory Champion: {best_memory['service']} ({best_memory['memory_efficiency']:.2f} RPS/MB)")
            print(f"   🎯 Best Overall Quality: {best_quality['service']} ({best_quality['throughput_quality']})")
        else:
            # Single service summary
            r = results[0]
            print(f"\n🎯 {r['service'].upper()} PERFORMANCE SUMMARY:")
            print(f"   🚀 Throughput: {r['rps']:.1f} RPS")
            print(f"   ⚡ Latency (P95): {r['p95_response_ms']:.1f}ms")
            print(f"   💾 Memory Efficiency: {r['memory_efficiency']:.2f} RPS/MB")
            print(f"   🎯 Quality: {r['throughput_quality']}")
            print(f"   🔥 CPU Usage: {r['avg_cpu']:.1f}%")
            print(f"   📈 CPU Efficiency: {r['cpu_efficiency']:.2f} RPS/CPU%")
        
        # I/O Efficiency insights (adjusted for single vs multi-service)
        if len(results) > 1:
            print(f"\n💡 I/O EFFICIENCY INSIGHTS:")
            best_cpu = max(results, key=lambda x: x['cpu_efficiency'])
            best_memory = max(results, key=lambda x: x['memory_efficiency'])
            print(f"   • {best_cpu['service']} shows best CPU efficiency for I/O workloads")
            print(f"   • Memory winner ({best_memory['service']}) uses {best_memory['avg_memory']:.0f}MB vs others")
            print(f"   • At 50% CPU load: {best_cpu['service']} could handle ~{best_cpu['cpu_efficiency'] * 50:.0f} RPS")
        else:
            r = results[0]
            print(f"\n💡 I/O EFFICIENCY INSIGHTS:")
            print(f"   • {r['service']} CPU efficiency: {r['cpu_efficiency']:.2f} RPS per CPU%")
            print(f"   • Memory usage: {r['avg_memory']:.0f}MB for {r['rps']:.1f} RPS")
            print(f"   • Scaling potential: ~{r['cpu_efficiency'] * 50:.0f} RPS at 50% CPU")
        
        print(f"\n📈 SCALING RECOMMENDATIONS FOR I/O WORKLOADS:")
        for r in results:
            if r['memory_efficiency'] > 1.0 and r['error_rate'] < 1:
                rec = "🚀 Ideal for high-scale I/O (efficient + reliable)"
            elif r['memory_efficiency'] > 0.5 and r['p95_response_ms'] < 500:
                rec = "⚡ Good for medium-scale, low-latency I/O"
            elif r['error_rate'] < 1:
                rec = "✅ Reliable for production I/O workloads"
            else:
                rec = "⚠️  Needs optimization before I/O scaling"
            print(f"   • {r['service']}: {rec}")
        
        print(f"\n🎯 FOR ADDITIONAL TESTING:")
        if args.service:
            # Single service mode suggestions
            print(f"   python3 simple_test.py 200 --service {args.service}     # Test different load")
            print(f"   python3 simple_test.py 500 --service {args.service}     # Test higher load") 
            print(f"   python3 throughput_test.py --single-rps 100 --services {args.service.capitalize()}  # Throughput curve")
            print(f"   python3 simple_test.py 100                # Compare all services")
        else:
            # Multi-service mode suggestions
            print(f"   python3 throughput_test.py              # Full load curve analysis")
            print(f"   python3 throughput_test.py --single-rps 100   # Test specific RPS")
            print(f"   python3 simple_test.py 500 --service rust    # Test single service")
            print(f"   python3 simple_test.py 500             # Test higher load")

if __name__ == "__main__":
    main()
