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

type CsvLogger struct {
	prevGetsTotal          uint64
	prevPutsTotal          uint64
	prevHitsTotal          uint64
	prevExpiredTotal       uint64
	prevReWritesTotal      uint64
	prevActiveEntriesTotal uint64

	samplesRthroguhput    []float64
	samplesWthroguhput    []float64
	samplesHitRate        []float64
	samplesActiveEntries  []float64
	samplesExpiredEntries []float64
	samplesReWrites       []float64
	samplesRP99           []time.Duration
	samplesRP50           []time.Duration
	samplesRP25           []time.Duration
	samplesWP99           []time.Duration
	samplesWP50           []time.Duration
	samplesWP25           []time.Duration

	totalSamples int

	metricsCollector *MetricsCollector
}

func (c *CsvLogger) collectMetrics() *time.Ticker {

	//tickered every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		shards := metricsCollector.Config.Metadata["shards"].(int)
		currentMetrics = metricsCollector.GetMetrics()

		getsTotal := uint64(0)
		putsTotal := uint64(0)
		hitsTotal := uint64(0)
		expiredTotal := uint64(0)
		reWritesTotal := uint64(0)
		activeEntriesTotal := uint64(0)

		rp99 := time.Duration(0)
		rp50 := time.Duration(0)
		rp25 := time.Duration(0)
		wp99 := time.Duration(0)
		wp50 := time.Duration(0)
		wp25 := time.Duration(0)

		for _, shard := range currentMetrics.ShardMetrics {
			getsTotal += uint64(shard.Gets)
			putsTotal += uint64(shard.Puts)
			hitsTotal += uint64(shard.Hits)
			expiredTotal += uint64(shard.ExpiredEntries)
			reWritesTotal += uint64(shard.Rewrites)
			activeEntriesTotal += uint64(shard.ActiveEntries)

			rp99 += shard.RP99
			rp50 += shard.RP50
			rp25 += shard.RP25
			wp99 += shard.WP99
			wp50 += shard.WP50
			wp25 += shard.WP25
		}

		rThroughput := float64(getsTotal-c.prevGetsTotal) / float64(30)
		wThroughput := float64(putsTotal-c.prevPutsTotal) / float64(30)
		hitRate := float64(hitsTotal-c.prevHitsTotal) / float64(getsTotal-c.prevGetsTotal)
		activeEntries := float64(activeEntriesTotal - c.prevActiveEntriesTotal)
		expiredEntries := float64(expiredTotal - c.prevExpiredTotal)
		reWrites := float64(reWritesTotal - c.prevReWritesTotal)

		rp99 = rp99 / time.Duration(shards)
		rp50 = rp50 / time.Duration(shards)
		rp25 = rp25 / time.Duration(shards)
		wp99 = wp99 / time.Duration(shards)
		wp50 = wp50 / time.Duration(shards)
		wp25 = wp25 / time.Duration(shards)

		c.samplesRthroguhput = append(c.samplesRthroguhput, rThroughput)
		c.samplesWthroguhput = append(c.samplesWthroguhput, wThroughput)
		c.samplesHitRate = append(c.samplesHitRate, hitRate)
		c.samplesActiveEntries = append(c.samplesActiveEntries, activeEntries)
		c.samplesExpiredEntries = append(c.samplesExpiredEntries, expiredEntries)
		c.samplesReWrites = append(c.samplesReWrites, reWrites)
		c.samplesRP99 = append(c.samplesRP99, rp99)
		c.samplesRP50 = append(c.samplesRP50, rp50)
		c.samplesRP25 = append(c.samplesRP25, rp25)
		c.samplesWP99 = append(c.samplesWP99, wp99)
		c.samplesWP50 = append(c.samplesWP50, wp50)
		c.samplesWP25 = append(c.samplesWP25, wp25)

		c.prevGetsTotal = getsTotal
		c.prevPutsTotal = putsTotal
		c.prevHitsTotal = hitsTotal
		c.prevExpiredTotal = expiredTotal
		c.prevReWritesTotal = reWritesTotal
		c.prevActiveEntriesTotal = activeEntriesTotal
	}

	return ticker

}

// RunCSVLoggerWaitForShutdown waits for shutdown signal and logs final metrics to CSV
func (c *CsvLogger) RunCSVLoggerWaitForShutdown() {

	ticker := c.collectMetrics()
	// --- Set up Signal Handling ---
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Program running. Press Ctrl+C to stop and log results to CSV...")

	// --- Wait for Stop Signal ---
	<-stopChan
	fmt.Println("\nTermination signal received. Stopping work and logging results...")

	// Stop the metrics collector
	if metricsCollector != nil {
		ticker.Stop()
		metricsCollector.Stop()
	}

	// --- Log Data to CSV ---
	if err := c.LogResultsToCSV(); err != nil {
		log.Fatalf("FATAL: Failed to log results to CSV: %v", err)
	}

	fmt.Printf("Successfully logged results to %s.\n", CSVFileName)

	// Exit the program since we're running in a goroutine
	os.Exit(0)
}

func (c *CsvLogger) LogResultsToCSV() error {
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

	metadata := c.metricsCollector.Config.Metadata
	timestamp := time.Now().In(time.FixedZone("IST", 5*60*60+30*60)).Format("2006-01-02 15:04:05")

	dataRow := []string{
		// Input Parameters
		strconv.Itoa(metadata["shards"].(int)),
		strconv.Itoa(metadata["keys_per_shard"].(int)),
		strconv.Itoa(metadata["read_workers"].(int)),
		strconv.Itoa(metadata["write_workers"].(int)),
		metadata["plan"].(string),

		// averaged observation parameters
		//sum sample and divide by total samples
		fmt.Sprintf("%v", averageFloat64(c.samplesRthroguhput)),
		fmt.Sprintf("%v", averageDuration(c.samplesRP99)),
		fmt.Sprintf("%v", averageDuration(c.samplesRP50)),
		fmt.Sprintf("%v", averageDuration(c.samplesRP25)),
		fmt.Sprintf("%v", averageFloat64(c.samplesWthroguhput)),
		fmt.Sprintf("%v", averageDuration(c.samplesWP99)),
		fmt.Sprintf("%v", averageDuration(c.samplesWP50)),
		fmt.Sprintf("%v", averageDuration(c.samplesWP25)),
		fmt.Sprintf("%v", averageFloat64(c.samplesHitRate)),
		fmt.Sprintf("%v", getCPUUsagePercent()),
		fmt.Sprintf("%v", getMemoryUsageMB()),
		timestamp,
	}

	if err := writer.Write(dataRow); err != nil {
		return fmt.Errorf("error writing CSV data row: %w", err)
	}

	return nil
}

func averageFloat64(samples []float64) float64 {
	sum := 0.0
	for _, sample := range samples {
		sum += sample
	}
	return sum / float64(len(samples))
}

func averageDuration(samples []time.Duration) time.Duration {
	sum := time.Duration(0)
	for _, sample := range samples {
		sum += sample
	}
	return sum / time.Duration(len(samples))
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
