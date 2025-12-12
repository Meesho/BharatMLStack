package main

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
	"sync"
	"syscall"
	"time"
)

// Define your parameter structure
type RunMetrics struct {
	// Input Parameters
	Shards       int
	KeysPerShard int
	ReadWorkers  int
	WriteWorkers int
	Plan         string

	// Observation Parameters
	RP99        time.Duration
	RP50        time.Duration
	RP25        time.Duration
	WP99        time.Duration
	WP50        time.Duration
	WP25        time.Duration
	RThroughput float64
	WThroughput float64
	HitRate     float64
	CPUUsage    float64
	MemoryUsage float64
}

// MetricChannels holds separate channels for each metric type
type MetricChannels struct {
	RP99        chan time.Duration
	RP50        chan time.Duration
	RP25        chan time.Duration
	WP99        chan time.Duration
	WP50        chan time.Duration
	WP25        chan time.Duration
	RThroughput chan float64
	WThroughput chan float64
	HitRate     chan float64
	CPUUsage    chan float64
	MemoryUsage chan float64
}

// MetricAverager maintains running averages for a metric
type MetricAverager struct {
	mu    sync.RWMutex
	sum   float64
	count int64
}

func (ma *MetricAverager) Add(value float64) {
	if value == 0 {
		return // Ignore zero values
	}
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.sum += value
	ma.count++
}

func (ma *MetricAverager) AddDuration(value time.Duration) {
	if value == 0 {
		return // Ignore zero values
	}
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.sum += float64(value)
	ma.count++
}

func (ma *MetricAverager) Average() float64 {
	ma.mu.RLock()
	defer ma.mu.RUnlock()
	if ma.count == 0 {
		return 0
	}
	return ma.sum / float64(ma.count)
}

func (ma *MetricAverager) Reset() {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.sum = 0
	ma.count = 0
}

// MetricsCollector collects and averages all metrics
type MetricsCollector struct {
	channels  MetricChannels
	averagers map[string]*MetricAverager
	stopCh    chan struct{}
	wg        sync.WaitGroup

	// Input parameters (set once)
	Shards       int
	KeysPerShard int
	ReadWorkers  int
	WriteWorkers int
	Plan         string
}

// NewMetricsCollector creates a new metrics collector with channels
func NewMetricsCollector(bufferSize int) *MetricsCollector {
	mc := &MetricsCollector{
		channels: MetricChannels{
			RP99:        make(chan time.Duration, bufferSize),
			RP50:        make(chan time.Duration, bufferSize),
			RP25:        make(chan time.Duration, bufferSize),
			WP99:        make(chan time.Duration, bufferSize),
			WP50:        make(chan time.Duration, bufferSize),
			WP25:        make(chan time.Duration, bufferSize),
			RThroughput: make(chan float64, bufferSize),
			WThroughput: make(chan float64, bufferSize),
			HitRate:     make(chan float64, bufferSize),
			CPUUsage:    make(chan float64, bufferSize),
			MemoryUsage: make(chan float64, bufferSize),
		},
		averagers: make(map[string]*MetricAverager),
		stopCh:    make(chan struct{}),
	}

	// Initialize averagers for each metric
	metricNames := []string{"RP99", "RP50", "RP25", "WP99", "WP50", "WP25", "RThroughput", "WThroughput", "HitRate", "CPUUsage", "MemoryUsage"}
	for _, name := range metricNames {
		mc.averagers[name] = &MetricAverager{}
	}

	return mc
}

// Start begins collecting metrics from all channels
func (mc *MetricsCollector) Start() {
	// Start a goroutine for each metric channel
	mc.wg.Add(11)

	go mc.collectMetricDuration(mc.channels.RP99, "RP99")
	go mc.collectMetricDuration(mc.channels.RP50, "RP50")
	go mc.collectMetricDuration(mc.channels.RP25, "RP25")
	go mc.collectMetricDuration(mc.channels.WP99, "WP99")
	go mc.collectMetricDuration(mc.channels.WP50, "WP50")
	go mc.collectMetricDuration(mc.channels.WP25, "WP25")
	go mc.collectMetric(mc.channels.RThroughput, "RThroughput")
	go mc.collectMetric(mc.channels.WThroughput, "WThroughput")
	go mc.collectMetric(mc.channels.HitRate, "HitRate")
	go mc.collectMetric(mc.channels.CPUUsage, "CPUUsage")
	go mc.collectMetric(mc.channels.MemoryUsage, "MemoryUsage")
}

func (mc *MetricsCollector) collectMetric(ch chan float64, name string) {
	defer mc.wg.Done()
	for {
		select {
		case <-mc.stopCh:
			return
		case value, ok := <-ch:
			if !ok {
				return
			}
			mc.averagers[name].Add(value)
		}
	}
}

func (mc *MetricsCollector) collectMetricDuration(ch chan time.Duration, name string) {
	defer mc.wg.Done()
	for {
		select {
		case <-mc.stopCh:
			return
		case value, ok := <-ch:
			if !ok {
				return
			}
			mc.averagers[name].AddDuration(value)
		}
	}
}

// RecordRP99 sends a value to the RP99 channel
func (mc *MetricsCollector) RecordRP99(value time.Duration) {
	select {
	case mc.channels.RP99 <- value:
	default: // Don't block if channel is full
	}
}

// RecordRP50 sends a value to the RP50 channel
func (mc *MetricsCollector) RecordRP50(value time.Duration) {
	select {
	case mc.channels.RP50 <- value:
	default:
	}
}

// RecordRP25 sends a value to the RP25 channel
func (mc *MetricsCollector) RecordRP25(value time.Duration) {
	select {
	case mc.channels.RP25 <- value:
	default:
	}
}

// RecordWP99 sends a value to the WP99 channel
func (mc *MetricsCollector) RecordWP99(value time.Duration) {
	select {
	case mc.channels.WP99 <- value:
	default:
	}
}

// RecordWP50 sends a value to the WP50 channel
func (mc *MetricsCollector) RecordWP50(value time.Duration) {
	select {
	case mc.channels.WP50 <- value:
	default:
	}
}

// RecordWP25 sends a value to the WP25 channel
func (mc *MetricsCollector) RecordWP25(value time.Duration) {
	select {
	case mc.channels.WP25 <- value:
	default:
	}
}

// RecordRThroughput sends a value to the RThroughput channel
func (mc *MetricsCollector) RecordRThroughput(value float64) {
	select {
	case mc.channels.RThroughput <- value:
	default:
	}
}

// RecordWThroughput sends a value to the WThroughput channel
func (mc *MetricsCollector) RecordWThroughput(value float64) {
	select {
	case mc.channels.WThroughput <- value:
	default:
	}
}

// RecordHitRate sends a value to the HitRate channel
func (mc *MetricsCollector) RecordHitRate(value float64) {
	select {
	case mc.channels.HitRate <- value:
	default:
	}
}

// GetAveragedMetrics returns the current averaged metrics
func (mc *MetricsCollector) GetAveragedMetrics() RunMetrics {
	return RunMetrics{
		Shards:       mc.Shards,
		KeysPerShard: mc.KeysPerShard,
		ReadWorkers:  mc.ReadWorkers,
		WriteWorkers: mc.WriteWorkers,
		Plan:         mc.Plan,
		RP99:         time.Duration(mc.averagers["RP99"].Average()),
		RP50:         time.Duration(mc.averagers["RP50"].Average()),
		RP25:         time.Duration(mc.averagers["RP25"].Average()),
		WP99:         time.Duration(mc.averagers["WP99"].Average()),
		WP50:         time.Duration(mc.averagers["WP50"].Average()),
		WP25:         time.Duration(mc.averagers["WP25"].Average()),
		RThroughput:  mc.averagers["RThroughput"].Average(),
		WThroughput:  mc.averagers["WThroughput"].Average(),
		HitRate:      mc.averagers["HitRate"].Average(),
		CPUUsage:     mc.averagers["CPUUsage"].Average(),
		MemoryUsage:  mc.averagers["MemoryUsage"].Average(),
	}
}

// ResetAverages resets all averagers to start fresh
func (mc *MetricsCollector) ResetAverages() {
	for _, avg := range mc.averagers {
		avg.Reset()
	}
}

// Stop stops all collector goroutines
func (mc *MetricsCollector) Stop() {
	close(mc.stopCh)
	mc.wg.Wait()
}

// SetShards sets the number of shards (input parameter)
func (mc *MetricsCollector) SetShards(value int) {
	mc.Shards = value
}

// SetKeysPerShard sets the keys per shard (input parameter)
func (mc *MetricsCollector) SetKeysPerShard(value int) {
	mc.KeysPerShard = value
}

// SetReadWorkers sets the number of read workers (input parameter)
func (mc *MetricsCollector) SetReadWorkers(value int) {
	mc.ReadWorkers = value
}

// SetWriteWorkers sets the number of write workers (input parameter)
func (mc *MetricsCollector) SetWriteWorkers(value int) {
	mc.WriteWorkers = value
}

// SetPlan sets the plan name (input parameter)
func (mc *MetricsCollector) SetPlan(value string) {
	mc.Plan = value
}

// Global variable to hold runtime data
var currentMetrics RunMetrics
var metricsCollector *MetricsCollector

// --- CSV Configuration ---
const CSVFileName = "performance_results.csv"

// InitMetricsCollector creates and starts the metrics collector, returning it
// so it can be passed to other components (e.g., cache config)
func InitMetricsCollector() *MetricsCollector {
	metricsCollector = NewMetricsCollector(100)
	metricsCollector.Start()
	return metricsCollector
}

// RunmetricsWaitForShutdown waits for shutdown signal and logs final metrics to CSV
func RunmetricsWaitForShutdown() {
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
		currentMetrics = metricsCollector.GetAveragedMetrics()
	}

	// Get memory usage and CPU usage at this instant
	currentMetrics.MemoryUsage = getMemoryUsageMB()
	currentMetrics.CPUUsage = getCPUUsagePercent()

	// --- Log Data to CSV ---
	if err := logResultsToCSV(); err != nil {
		log.Fatalf("FATAL: Failed to log results to CSV: %v", err)
	}

	fmt.Printf("Successfully logged results to %s.\n", CSVFileName)

	// Exit the program since we're running in a goroutine
	os.Exit(0)
}

// RunmetricsInit initializes metrics and waits for shutdown (convenience function)
func RunmetricsInit() {
	InitMetricsCollector()
	RunmetricsWaitForShutdown()
}

func logResultsToCSV() error {
	// 1. Check if the file exists to determine if we need a header row.
	file, err := os.OpenFile(CSVFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush() // Crucial to ensure data is written to the file before exiting.

	// The list of all your column headers
	header := []string{
		"SHARDS", "KEYS_PER_SHARD", "READ_WORKERS", "WRITE_WORKERS", "PLAN",
		"R_P99", "R_P50", "R_P25", "W_P99", "W_P50", "W_P25",
		"R_THROUGHPUT", "W_THROUGHPUT", "HIT_RATE", "CPU", "MEMORY", "TIME",
	}

	// Determine if the file is new (or empty) and needs the header
	fileInfo, _ := file.Stat()
	if fileInfo.Size() == 0 {
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("error writing CSV header: %w", err)
		}
	}

	// Convert your struct fields into a slice of strings for the CSV writer
	dataRow := []string{
		// Input Parameters
		strconv.Itoa(currentMetrics.Shards),
		strconv.Itoa(currentMetrics.KeysPerShard),
		strconv.Itoa(currentMetrics.ReadWorkers), // Convert int to string
		strconv.Itoa(currentMetrics.WriteWorkers),
		currentMetrics.Plan,

		// Observation Parameters (convert floats to strings)
		fmt.Sprintf("%v", currentMetrics.RP99),
		fmt.Sprintf("%v", currentMetrics.RP50),
		fmt.Sprintf("%v", currentMetrics.RP25),
		fmt.Sprintf("%v", currentMetrics.WP99),
		fmt.Sprintf("%v", currentMetrics.WP50),
		fmt.Sprintf("%v", currentMetrics.WP25),
		fmt.Sprintf("%v", currentMetrics.RThroughput),
		fmt.Sprintf("%v", currentMetrics.WThroughput),
		fmt.Sprintf("%v", currentMetrics.HitRate),
		fmt.Sprintf("%v", currentMetrics.CPUUsage),
		fmt.Sprintf("%v", currentMetrics.MemoryUsage),
		fmt.Sprintf("%v", time.Now().In(time.FixedZone("IST", 5*60*60+30*60)).Format("2006-01-02 15:04:05")),
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

// getSystemMemoryUsageMB returns the total system memory used by this process in MB
func getSystemMemoryUsageMB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// Sys is the total bytes of memory obtained from the OS
	return float64(m.Sys) / 1024 / 1024
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
