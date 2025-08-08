package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type LoadTestConfig struct {
	Duration       time.Duration
	NumWorkers     int
	WriteRatio     float64 // 0.0 = all reads, 1.0 = all writes
	KeyRange       int     // keys will be key_0 to key_(KeyRange-1)
	ValueSizeBytes int
	ExpTime        uint64
}

type LoadTestMetrics struct {
	totalReads     int64
	totalWrites    int64
	totalHits      int64
	totalMisses    int64
	readLatencyNs  int64
	writeLatencyNs int64
}

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Cache configuration
	cacheConfig := internal.WrapCacheConfig{
		NumShards:    4,
		KeysPerShard: 1000000,
		FileSize:     1 * 1024 * 1024 * 1024,
		MemtableSize: 1 * 1024 * 1024,
	}

	// Load test configuration
	loadTestConfig := LoadTestConfig{
		Duration:       30 * time.Second,
		NumWorkers:     10,
		WriteRatio:     0.3, // 30% writes, 70% reads
		KeyRange:       100000,
		ValueSizeBytes: 1024,
		ExpTime:        uint64(time.Now().Add(time.Hour).Unix()),
	}

	cache, err := internal.NewWrapCache(cacheConfig, "/tmp")
	if err != nil {
		log.Error().Msgf("Error creating cache: %v\n", err)
		return
	}

	log.Info().Msg("Starting load test...")
	runLoadTest(cache, loadTestConfig)
}

func runLoadTest(cache *internal.WrapCache, config LoadTestConfig) {
	metrics := &LoadTestMetrics{}
	var wg sync.WaitGroup

	startTime := time.Now()
	endTime := startTime.Add(config.Duration)

	// Start worker goroutines
	for i := 0; i < config.NumWorkers; i++ {
		wg.Add(1)
		go worker(cache, config, metrics, endTime, &wg, i)
	}

	// Monitor progress
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			elapsed := time.Since(startTime)
			if elapsed >= config.Duration {
				return
			}
			printProgress(metrics, elapsed)
		}
	}()

	// Wait for all workers to complete
	wg.Wait()

	elapsed := time.Since(startTime)
	printFinalResults(metrics, elapsed)
}

func worker(cache *internal.WrapCache, config LoadTestConfig, metrics *LoadTestMetrics, endTime time.Time, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))
	value := make([]byte, config.ValueSizeBytes)
	for i := range value {
		value[i] = byte(rng.Intn(256))
	}

	for time.Now().Before(endTime) {
		// Generate probabilistic key
		keyID := rng.Intn(config.KeyRange)
		key := fmt.Sprintf("key_%d", keyID)

		// Decide operation type based on write ratio
		if rng.Float64() < config.WriteRatio {
			// Write operation
			start := time.Now()
			err := cache.Put(key, value, config.ExpTime)
			latency := time.Since(start).Nanoseconds()

			if err != nil {
				log.Error().Msgf("Write error for key %s: %v", key, err)
			} else {
				atomic.AddInt64(&metrics.totalWrites, 1)
				atomic.AddInt64(&metrics.writeLatencyNs, latency)
			}
		} else {
			// Read operation
			start := time.Now()
			_, found, expired := cache.Get(key)
			latency := time.Since(start).Nanoseconds()

			atomic.AddInt64(&metrics.totalReads, 1)
			atomic.AddInt64(&metrics.readLatencyNs, latency)

			if found && !expired {
				atomic.AddInt64(&metrics.totalHits, 1)
			} else {
				atomic.AddInt64(&metrics.totalMisses, 1)
			}
		}
	}
}

func printProgress(metrics *LoadTestMetrics, elapsed time.Duration) {
	reads := atomic.LoadInt64(&metrics.totalReads)
	writes := atomic.LoadInt64(&metrics.totalWrites)
	hits := atomic.LoadInt64(&metrics.totalHits)

	total := reads + writes
	if total == 0 {
		return
	}

	opsPerSec := float64(total) / elapsed.Seconds()
	hitRate := float64(hits) / float64(reads) * 100

	log.Info().Msgf("Progress [%v]: Total ops: %d, Ops/sec: %.2f, Reads: %d, Writes: %d, Hit rate: %.2f%%",
		elapsed.Truncate(time.Second), total, opsPerSec, reads, writes, hitRate)
}

func printFinalResults(metrics *LoadTestMetrics, elapsed time.Duration) {
	reads := atomic.LoadInt64(&metrics.totalReads)
	writes := atomic.LoadInt64(&metrics.totalWrites)
	hits := atomic.LoadInt64(&metrics.totalHits)
	misses := atomic.LoadInt64(&metrics.totalMisses)
	readLatencyNs := atomic.LoadInt64(&metrics.readLatencyNs)
	writeLatencyNs := atomic.LoadInt64(&metrics.writeLatencyNs)

	total := reads + writes

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Println("LOAD TEST RESULTS")
	fmt.Printf("%s\n", strings.Repeat("=", 60))
	fmt.Printf("Test Duration: %v\n", elapsed.Truncate(time.Millisecond))
	fmt.Printf("Total Operations: %d\n", total)
	fmt.Printf("Operations/sec: %.2f\n", float64(total)/elapsed.Seconds())
	fmt.Println()

	fmt.Printf("Read Operations: %d\n", reads)
	fmt.Printf("Write Operations: %d\n", writes)
	fmt.Printf("Cache Hits: %d\n", hits)
	fmt.Printf("Cache Misses: %d\n", misses)

	if reads > 0 {
		hitRate := float64(hits) / float64(reads) * 100
		avgReadLatency := time.Duration(readLatencyNs / reads)
		fmt.Printf("Hit Rate: %.2f%%\n", hitRate)
		fmt.Printf("Average Read Latency: %v\n", avgReadLatency)
	}

	if writes > 0 {
		avgWriteLatency := time.Duration(writeLatencyNs / writes)
		fmt.Printf("Average Write Latency: %v\n", avgWriteLatency)
	}

	fmt.Printf("%s\n", strings.Repeat("=", 60))
}
