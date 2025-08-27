#!/usr/bin/env python3
"""
Simple Performance Comparison
Just the key metrics for all 3 services in one clean output
"""

import subprocess
import time
import psutil
import json
import os
from datetime import datetime

def find_service_pids():
    """Find process IDs for monitoring"""
    pids = {}
    
    for proc in psutil.process_iter(['pid', 'name', 'cmdline']):
        try:
            cmdline = ' '.join(proc.info['cmdline'])
            
            if 'java-caller' in cmdline and 'java' in cmdline:
                pids['java'] = proc.info['pid']
            elif 'rust-caller' in cmdline:
                if 'target/release' in cmdline:
                    pids['rust'] = proc.info['pid']
                elif 'cargo' in cmdline and 'rust' not in pids:
                    pids['rust'] = proc.info['pid']
            elif '/go-build/' in cmdline and '/main' in cmdline:
                pids['go'] = proc.info['pid']
            elif 'go run main.go' in cmdline and 'go' not in pids:
                pids['go'] = proc.info['pid']
                
        except:
            continue
    
    return pids

def run_single_test(service_name, port, target_rps, users, duration=60):
    """Run test for one service and return key metrics"""
    print(f"🔥 Testing {service_name}...")
    
    # Find PID
    pids = find_service_pids()
    service_key = service_name.lower()
    
    if service_key not in pids:
        print(f"❌ {service_name} service not running")
        return None
    
    service_pid = pids[service_key]
    
    # Setup files
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    output_prefix = f"results/comparison_{service_name.lower()}_{timestamp}"
    os.makedirs("results", exist_ok=True)
    
    # Locust command
    spawn_rate = max(int(users * 0.1), 1)
    cmd = [
        "locust", "--headless",
        "--host", f"http://localhost:{port}",
        "--users", str(users),
        "--spawn-rate", str(spawn_rate),
        "--run-time", f"{duration}s",
        "--csv", output_prefix,
        "-f", "simple_locustfile.py"
    ]
    
    # Start test and monitor
    cpu_readings = []
    memory_readings = []
    
    start_time = time.time()
    process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    
    # Track progress
    last_update = 0
    print(f"⏳ Running {duration}s test with periodic updates...")
    
    while process.poll() is None and (time.time() - start_time) < duration + 10:
        try:
            proc = psutil.Process(service_pid)
            cpu_percent = proc.cpu_percent(interval=1.0)
            memory_mb = proc.memory_info().rss / (1024 * 1024)
            
            if cpu_percent > 0:
                cpu_readings.append(cpu_percent)
            memory_readings.append(memory_mb)
            
            # Show progress every 10 seconds
            elapsed = time.time() - start_time
            if elapsed - last_update >= 10:
                remaining = max(0, duration - elapsed)
                avg_cpu_so_far = sum(cpu_readings) / len(cpu_readings) if cpu_readings else 0
                avg_mem_so_far = sum(memory_readings) / len(memory_readings) if memory_readings else 0
                print(f"   📊 {elapsed:.0f}s | CPU: {cpu_percent:.1f}% (avg: {avg_cpu_so_far:.1f}%) | Memory: {memory_mb:.1f}MB (avg: {avg_mem_so_far:.1f}MB) | {remaining:.0f}s remaining")
                last_update = elapsed
            
        except:
            break
    
    process.communicate()
    
    print(f"✅ {service_name} test execution completed!")
    
    if process.returncode != 0:
        print(f"❌ {service_name} test failed")
        return None
    
    # Calculate averages
    avg_cpu = sum(cpu_readings) / len(cpu_readings) if cpu_readings else 0
    avg_memory = sum(memory_readings) / len(memory_readings) if memory_readings else 0
    
    # Parse results
    stats_file = f"{output_prefix}_stats.csv"
    
    try:
        with open(stats_file, 'r') as f:
            lines = f.readlines()
            if len(lines) > 1:
                # Find aggregated row
                for line in lines[1:]:
                    fields = line.strip().split(',')
                    if len(fields) > 10 and fields[1] == 'Aggregated':
                        data = fields
                        break
                else:
                    # Use first valid data row
                    for line in lines[1:]:
                        fields = line.strip().split(',')
                        if len(fields) > 10:
                            try:
                                float(fields[9])  # Test RPS parsing
                                data = fields
                                break
                            except:
                                continue
                    else:
                        print(f"❌ {service_name}: No valid data found")
                        return None
                
                # Extract key metrics
                actual_rps = float(data[9])
                total_requests = int(float(data[2]))
                failures = int(float(data[3]))
                avg_response = float(data[5])
                min_response = float(data[6])
                max_response = float(data[7])
                p95_response = float(data[15]) if len(data) > 15 and data[15] else 0
                p99_response = float(data[16]) if len(data) > 16 and data[16] else 0
                
                # Calculate derived metrics
                error_rate = (failures / total_requests * 100) if total_requests > 0 else 0
                rps_achievement = (actual_rps / target_rps * 100) if target_rps > 0 else 0
                cpu_efficiency = actual_rps / avg_cpu if avg_cpu > 0 else 0
                memory_efficiency = actual_rps / avg_memory if avg_memory > 0 else 0
                
                return {
                    'service': service_name,
                    'target_rps': target_rps,
                    'actual_rps': actual_rps,
                    'rps_achievement': rps_achievement,
                    'total_requests': total_requests,
                    'failures': failures,
                    'error_rate': error_rate,
                    'avg_response_ms': avg_response,
                    'p95_response_ms': p95_response,
                    'p99_response_ms': p99_response,
                    'min_response_ms': min_response,
                    'max_response_ms': max_response,
                    'avg_cpu': avg_cpu,
                    'avg_memory': avg_memory,
                    'cpu_efficiency': cpu_efficiency,
                    'memory_efficiency': memory_efficiency,
                    'users': users,
                    'duration': duration
                }
                
    except Exception as e:
        print(f"❌ {service_name}: Error parsing results - {e}")
        return None

def print_service_result(result):
    """Print immediate results after each service completes"""
    if not result:
        return
    
    r = result
    print(f"\n📋 {r['service'].upper()} TEST COMPLETED")
    print("=" * 60)
    print(f"🎯 Target: {r['target_rps']} RPS | ✅ Achieved: {r['actual_rps']:.1f} RPS ({r['rps_achievement']:.1f}%)")
    print(f"📊 Requests: {r['total_requests']} total | ❌ Failures: {r['failures']} ({r['error_rate']:.1f}%)")
    print(f"⏱️  Latency: Avg {r['avg_response_ms']:.1f}ms | P95 {r['p95_response_ms']:.1f}ms | P99 {r['p99_response_ms']:.1f}ms")
    print(f"💻 Resources: CPU {r['avg_cpu']:.1f}% | Memory {r['avg_memory']:.1f}MB")
    print(f"⚡ Efficiency: {r['cpu_efficiency']:.2f} RPS/CPU% | {r['memory_efficiency']:.2f} RPS/MB")
    print("=" * 60)

def print_comparison(results, target_rps):
    """Print comprehensive comparison table"""
    print(f"\n{'='*140}")
    print(f"🎯 COMPREHENSIVE PERFORMANCE COMPARISON - {target_rps} RPS TARGET")
    print(f"{'='*140}")
    
    # Main metrics table
    print(f"{'Service':<8} | {'Target':<6} | {'Actual':<6} | {'Achievement':<11} | {'Errors%':<7} | {'Requests':<8} | {'Users':<5} | {'Duration':<8}")
    print("-" * 80)
    
    # Sort by actual RPS
    results.sort(key=lambda x: x['actual_rps'], reverse=True)
    
    for r in results:
        print(f"{r['service']:<8} | {r['target_rps']:6.0f} | {r['actual_rps']:6.1f} | {r['rps_achievement']:9.1f}% | {r['error_rate']:6.1f} | {r['total_requests']:8.0f} | {r['users']:5.0f} | {r['duration']:8.0f}s")
    
    # Latency details
    print(f"\n📊 LATENCY BREAKDOWN (milliseconds)")
    print("-" * 100)
    print(f"{'Service':<8} | {'Min':<6} | {'Avg':<6} | {'P95':<6} | {'P99':<6} | {'Max':<8} | {'Range':<8}")
    print("-" * 100)
    
    for r in results:
        latency_range = r['max_response_ms'] - r['min_response_ms']
        print(f"{r['service']:<8} | {r['min_response_ms']:6.1f} | {r['avg_response_ms']:6.1f} | {r['p95_response_ms']:6.1f} | {r['p99_response_ms']:6.1f} | {r['max_response_ms']:8.1f} | {latency_range:8.1f}")
    
    # Resource efficiency
    print(f"\n⚡ RESOURCE EFFICIENCY")
    print("-" * 80)
    print(f"{'Service':<8} | {'CPU%':<6} | {'Memory(MB)':<10} | {'RPS/CPU%':<8} | {'RPS/MB':<8} | {'Efficiency':<10}")
    print("-" * 80)
    
    for r in results:
        efficiency = "Excellent" if r['memory_efficiency'] > 1.0 else "Good" if r['memory_efficiency'] > 0.5 else "Fair"
        print(f"{r['service']:<8} | {r['avg_cpu']:6.1f} | {r['avg_memory']:10.1f} | {r['cpu_efficiency']:8.2f} | {r['memory_efficiency']:8.2f} | {efficiency:<10}")
    
    # Winner analysis
    best_rps = max(results, key=lambda x: x['actual_rps'])
    best_latency = min(results, key=lambda x: x['p95_response_ms'])
    best_memory = max(results, key=lambda x: x['memory_efficiency'])
    best_cpu = max(results, key=lambda x: x['cpu_efficiency'])
    lowest_p99 = min(results, key=lambda x: x['p99_response_ms'])
    
    print(f"\n🏆 PERFORMANCE LEADERS:")
    print(f"   🚀 Highest Throughput: {best_rps['service']} ({best_rps['actual_rps']:.1f} RPS)")
    print(f"   ⚡ Best P95 Latency: {best_latency['service']} ({best_latency['p95_response_ms']:.1f}ms)")
    print(f"   🎯 Best P99 Latency: {lowest_p99['service']} ({lowest_p99['p99_response_ms']:.1f}ms)")
    print(f"   💾 Memory Champion: {best_memory['service']} ({best_memory['memory_efficiency']:.2f} RPS/MB)")
    print(f"   🔥 CPU Champion: {best_cpu['service']} ({best_cpu['cpu_efficiency']:.2f} RPS/CPU%)")
    
    print(f"\n{'='*140}")

def save_summary(results, target_rps):
    """Save simple summary to file"""
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    filename = f"results/summary_{target_rps}rps_{timestamp}.json"
    
    summary = {
        'test_info': {
            'target_rps': target_rps,
            'timestamp': timestamp,
            'test_type': 'comparison'
        },
        'results': results
    }
    
    with open(filename, 'w') as f:
        json.dump(summary, f, indent=2)
    
    print(f"📁 Results saved to: {filename}")

def main():
    import argparse
    parser = argparse.ArgumentParser(description='Simple performance comparison')
    parser.add_argument('target_rps', type=int, help='Target RPS for all services')
    parser.add_argument('--duration', type=int, default=60, help='Test duration in seconds (default: 60)')
    parser.add_argument('--cooldown', type=int, default=120, help='Cooldown between services in seconds (default: 120)')
    
    args = parser.parse_args()
    
    # Test configuration
    target_rps = args.target_rps
    duration = args.duration
    cooldown = args.cooldown
    users = max(int(target_rps * 0.3), 3)  # Simple user calculation
    
    print(f"🚀 Starting comparison test: {target_rps} RPS target, {duration}s duration, {users} users, {cooldown}s cooldown")
    
    services = [
        ("Java", 8082),
        ("Rust", 8080), 
        ("Go", 8081)
    ]
    
    results = []
    
    for i, (service_name, port) in enumerate(services):
        result = run_single_test(service_name, port, target_rps, users, duration)
        if result:
            results.append(result)
            print_service_result(result)  # Show immediate results
        
        # Brief pause between services
        if i < len(services) - 1:  # Don't pause after last service
            next_service = services[i + 1][0]
            print(f"\n⏳ {cooldown}s cooldown between services...")
            for countdown in range(cooldown, 0, -30):
                print(f"   ⏱️  {countdown}s remaining...")
                time.sleep(min(30, countdown))
            print(f"🔜 Starting {next_service} test next...")
    
    if results:
        print(f"\n🎉 ALL TESTS COMPLETED! Tested {len(results)}/{len(services)} services")
        print_comparison(results, target_rps)
        save_summary(results, target_rps)
        
        # Final summary
        best_service = max(results, key=lambda x: x['actual_rps'])
        total_requests = sum(r['total_requests'] for r in results)
        total_failures = sum(r['failures'] for r in results)
        print(f"\n📈 FINAL SUMMARY:")
        print(f"   🚀 Best performer: {best_service['service']} with {best_service['actual_rps']:.1f} RPS")
        print(f"   📊 Total requests processed: {total_requests}")
        print(f"   ❌ Total failures: {total_failures} ({(total_failures/total_requests*100) if total_requests > 0 else 0:.1f}%)")
        print(f"   ⏱️  Total test time: ~{len(services) * duration + (len(services)-1) * cooldown}s")
    else:
        print("❌ No results to compare")

if __name__ == "__main__":
    main()
