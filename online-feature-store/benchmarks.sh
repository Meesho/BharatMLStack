#!/bin/bash

# Define the output directory
output_dir="bench"

# Create the output directory if it does not exist
mkdir -p "$output_dir"

# Function to get the next version number
get_next_version() {
    local prefix=$1
    local latest_version=0
    for file in "$output_dir"/"$prefix"_v*.txt "$output_dir"/"$prefix"_v*.prof; do
        [[ -e "$file" ]] || continue
        # Extract the version number from the filename
        version=$(echo "$file" | sed -E 's/.*_v([0-9]+)\..*/\1/')
        # Update latest_version if this version is greater
        if (( version > latest_version )); then
            latest_version=$version
        fi
    done
    # Return the next version number
    echo $((latest_version + 1))
}

# Get the next version numbers for benchmark and profile files
version=$(get_next_version "benchmark_results")
benchmark_file="$output_dir/benchmark_results_v${version}.txt"
pprof_file="$output_dir/cpu_profile_v${version}.prof"
memprof_file="$output_dir/mem_profile_v${version}.prof"

# Clear the benchmark results file
> "$benchmark_file"

# Check if "cpu" or "mem" arguments are passed
run_pprof=false
run_memprof=false

while [[ "$1" != "" ]]; do
    case "$1" in
        cpu)
            run_pprof=true
            echo "CPU profiling enabled. Profiles will be saved as $pprof_file"
            ;;
        mem)
            run_memprof=true
            echo "Memory profiling enabled. Profiles will be saved as $memprof_file"
            ;;
        *)
            break
            ;;
    esac
    shift
done

# Optional arguments for package and test name
pkg=${1:-}       # Package name, optional
test_name=${2:-} # Test name, optional

# Run benchmark for specific package and test if provided, otherwise run all
if [[ -n "$pkg" ]]; then
    echo "Running benchmarks for package: $pkg" | tee -a "$benchmark_file"
    if [[ -n "$test_name" ]]; then
        echo "Running specific test: $test_name" | tee -a "$benchmark_file"
    fi

    if $run_pprof && $run_memprof; then
        # Run benchmark with both CPU and memory profiling
        go test -run=^$ -bench="${test_name:-.}" -benchmem -cpuprofile="$pprof_file" -memprofile="$memprof_file" "$pkg" >> "$benchmark_file" 2>&1
        echo "CPU profile saved to $pprof_file and memory profile saved to $memprof_file for package $pkg" | tee -a "$benchmark_file"
    elif $run_pprof; then
        # Run benchmark with only CPU profiling
        go test -run=^$ -bench="${test_name:-.}" -benchmem -cpuprofile="$pprof_file" "$pkg" >> "$benchmark_file" 2>&1
        echo "CPU profile saved to $pprof_file for package $pkg" | tee -a "$benchmark_file"
    elif $run_memprof; then
        # Run benchmark with only memory profiling
        go test -run=^$ -bench="${test_name:-.}" -benchmem -memprofile="$memprof_file" "$pkg" >> "$benchmark_file" 2>&1
        echo "Memory profile saved to $memprof_file for package $pkg" | tee -a "$benchmark_file"
    else
        # Run benchmark without profiling
        go test -run=^$ -bench="${test_name:-.}" -benchmem "$pkg" >> "$benchmark_file" 2>&1
    fi
else
    # Run all packages and tests if no package is specified
    for pkg in $(go list ./...); do
        echo "Running benchmarks in package: $pkg" | tee -a "$benchmark_file"

        if $run_pprof && $run_memprof; then
            # Run benchmark with both CPU and memory profiling
            go test -run=^$ -bench=. -benchmem -cpuprofile="$pprof_file" -memprofile="$memprof_file" "$pkg" >> "$benchmark_file" 2>&1
            echo "CPU profile saved to $pprof_file and memory profile saved to $memprof_file for package $pkg" | tee -a "$benchmark_file"
        elif $run_pprof; then
            # Run benchmark with only CPU profiling
            go test -run=^$ -bench=. -benchmem -cpuprofile="$pprof_file" "$pkg" >> "$benchmark_file" 2>&1
            echo "CPU profile saved to $pprof_file for package $pkg" | tee -a "$benchmark_file"
        elif $run_memprof; then
            # Run benchmark with only memory profiling
            go test -run=^$ -bench=. -benchmem -memprofile="$memprof_file" "$pkg" >> "$benchmark_file" 2>&1
            echo "Memory profile saved to $memprof_file for package $pkg" | tee -a "$benchmark_file"
        else
            # Run benchmark without profiling
            go test -run=^$ -bench=. -benchmem "$pkg" >> "$benchmark_file" 2>&1
        fi

        echo "" >> "$benchmark_file"  # Add an empty line for separation
    done
fi

echo "Benchmark results saved to $benchmark_file"
