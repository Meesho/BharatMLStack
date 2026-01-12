package metrics

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// --- CSV Configuration ---
const CSVFileName = "performance_results.csv"

// RunCSVLoggerWaitForShutdown waits for shutdown signal and logs final metrics to CSV
func RunCSVLoggerWaitForShutdown() {
	// --- Set up Signal Handling ---
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Program running. Press Ctrl+C to stop and log results to CSV...")

	// --- Wait for Stop Signal ---
	<-stopChan
	fmt.Println("\nTermination signal received. Stopping work and logging results...")

	// Stop the metrics collector
	if metricsCollector != nil {
		metricsCollector.Stop()

		// Get final averaged metrics
		currentMetrics = metricsCollector.GetMetrics()
	}

	// --- Log Data to CSV ---
	if err := LogResultsToCSV(metricsCollector.Config.Metadata); err != nil {
		log.Fatalf("FATAL: Failed to log results to CSV: %v", err)
	}

	fmt.Printf("Successfully logged results to %s.\n", CSVFileName)

	// Exit the program since we're running in a goroutine
	os.Exit(0)
}

func LogResultsToCSV(metadata map[string]any) error {
	// 1. Check if the file exists to determine if we need a header row.
	file, err := os.OpenFile(CSVFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush() // Crucial to ensure data is written to the file before exiting.

	// The list of all your column headers (per-shard metrics)
	header := []string{
		"SHARDS", "KEYS_PER_SHARD", "READ_WORKERS", "WRITE_WORKERS", "PLAN",
		"R_THROUGHPUT", "R_P99", "R_P50", "R_P25", "W_THROUGHPUT", "W_P99", "W_P50", "W_P25",
		"HIT_RATE", "CPU", "MEMORY", "TIME",
	}

	// Determine if the file is new (or empty) and needs the header
	fileInfo, _ := file.Stat()
	if fileInfo.Size() == 0 {
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("error writing CSV header: %w", err)
		}
	}

	timestamp := time.Now().In(time.FixedZone("IST", 5*60*60+30*60)).Format("2006-01-02 15:04:05")

	dataRow := []string{
		// Input Parameters
		strconv.Itoa(metadata["shards"].(int)),
		strconv.Itoa(metadata["keys_per_shard"].(int)),
		strconv.Itoa(metadata["read_workers"].(int)),
		strconv.Itoa(metadata["write_workers"].(int)),
		metadata["plan"].(string),

		// averaged observation parameters
		fmt.Sprintf("%v", currentMetrics.AveragedMetrics.RThroughput),
		fmt.Sprintf("%v", currentMetrics.AveragedMetrics.RP99),
		fmt.Sprintf("%v", currentMetrics.AveragedMetrics.RP50),
		fmt.Sprintf("%v", currentMetrics.AveragedMetrics.RP25),
		fmt.Sprintf("%v", currentMetrics.AveragedMetrics.WThroughput),
		fmt.Sprintf("%v", currentMetrics.AveragedMetrics.WP99),
		fmt.Sprintf("%v", currentMetrics.AveragedMetrics.WP50),
		fmt.Sprintf("%v", currentMetrics.AveragedMetrics.WP25),
		fmt.Sprintf("%v", currentMetrics.AveragedMetrics.HitRate),
		fmt.Sprintf("%v", getCPUUsagePercent()),
		fmt.Sprintf("%v", getMemoryUsageMB()),
		timestamp,
	}

	if err := writer.Write(dataRow); err != nil {
		return fmt.Errorf("error writing CSV data row: %w", err)
	}

	return nil
}

// getMemoryUsageMB returns the current memory usage of this process in MB
func getMemoryUsageMB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// Alloc is bytes of allocated heap objects
	return float64(m.Alloc) / 1024 / 1024
}

// getCPUUsagePercent returns the CPU usage percentage for this process
// It measures CPU usage over a short interval
func getCPUUsagePercent() float64 {
	// Read initial CPU stats
	idle1, total1 := getCPUStats()
	time.Sleep(100 * time.Millisecond)
	// Read CPU stats again
	idle2, total2 := getCPUStats()

	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)

	if totalDelta == 0 {
		return 0
	}

	cpuUsage := (1.0 - idleDelta/totalDelta) * 100.0
	return cpuUsage
}

// getCPUStats reads /proc/stat and returns idle and total CPU time
func getCPUStats() (idle, total uint64) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return 0, 0
			}
			// fields: cpu user nice system idle iowait irq softirq steal guest guest_nice
			var values []uint64
			for _, field := range fields[1:] {
				val, err := strconv.ParseUint(field, 10, 64)
				if err != nil {
					continue
				}
				values = append(values, val)
				total += val
			}
			if len(values) >= 4 {
				idle = values[3] // idle is the 4th value
			}
			break
		}
	}
	return idle, total
}
