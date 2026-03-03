package main

import (
	"flag"
	"fmt"
	"math/bits"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cachepkg "github.com/Meesho/BharatMLStack/flashring/pkg/cache"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const histBuckets = 32 // bucket i covers [2^i, 2^(i+1)) nanoseconds

type opMetrics struct {
	count   atomic.Int64
	totalNs atomic.Int64
	minNs   atomic.Int64
	maxNs   atomic.Int64
	hist    [histBuckets]atomic.Int64
}

func (m *opMetrics) record(d time.Duration) {
	ns := d.Nanoseconds()
	if ns <= 0 {
		ns = 1
	}
	m.count.Add(1)
	m.totalNs.Add(ns)

	bucket := bits.Len64(uint64(ns)) - 1
	if bucket >= histBuckets {
		bucket = histBuckets - 1
	}
	m.hist[bucket].Add(1)

	for {
		cur := m.minNs.Load()
		if cur != 0 && cur <= ns {
			break
		}
		if m.minNs.CompareAndSwap(cur, ns) {
			break
		}
	}
	for {
		cur := m.maxNs.Load()
		if cur >= ns {
			break
		}
		if m.maxNs.CompareAndSwap(cur, ns) {
			break
		}
	}
}

func (m *opMetrics) percentile(p float64) time.Duration {
	total := m.count.Load()
	if total == 0 {
		return 0
	}
	threshold := int64(float64(total)*p/100.0 + 0.5)
	var cumulative int64
	for i := 0; i < histBuckets; i++ {
		cumulative += m.hist[i].Load()
		if cumulative >= threshold {
			return time.Duration(int64(1) << i)
		}
	}
	return time.Duration(m.maxNs.Load())
}

func (m *opMetrics) snapshot() (count int64, avg, min, max, p50, p99 time.Duration) {
	count = m.count.Load()
	if count == 0 {
		return
	}
	avg = time.Duration(m.totalNs.Load() / count)
	min = time.Duration(m.minNs.Load())
	max = time.Duration(m.maxNs.Load())
	p50 = m.percentile(50)
	p99 = m.percentile(99)
	return
}

type loadMetrics struct {
	getMetrics            opMetrics
	putMetrics            opMetrics
	prepopulatePutMetrics opMetrics
	getHits               atomic.Int64
	getMisses             atomic.Int64
	getExpired            atomic.Int64
}

func printOpLine(name string, m *opMetrics) {
	count, avg, min, max, p50, p99 := m.snapshot()
	fmt.Printf("%-5s count=%-12d\n", name, count)
	if count > 0 {
		fmt.Printf("      avg=%-14s  min=%-14s  max=%-14s  p50=%-14s  p99=%-14s\n", avg, min, max, p50, p99)
	}
}

func (lm *loadMetrics) printStats(label string) {
	gc, _, _, _, _, _ := lm.getMetrics.snapshot()

	fmt.Printf("\n===== %s =====\n", label)
	fmt.Printf("GET  count=%-12d  hits=%-12d  misses=%-12d  expired=%-12d\n",
		gc, lm.getHits.Load(), lm.getMisses.Load(), lm.getExpired.Load())
	if gc > 0 {
		printOpLine("GET", &lm.getMetrics)
	}
	printOpLine("PUT", &lm.putMetrics)
	printOpLine("PREPOP", &lm.prepopulatePutMetrics)
	fmt.Println()
}

func planReadthroughGaussian() {
	var (
		mountPoint         string
		numShards          int
		keysPerShard       int
		memtableMB         int
		fileSizeMultiplier float64
		readWorkers        int
		writeWorkers       int
		sampleSecs         int
		iterations         int64
		aVal               float64
		logStats           bool
		memProfile         string
		cpuProfile         string
	)

	flag.StringVar(&mountPoint, "mount", "/mnt/disks/nvme/", "data directory for shard files")
	flag.IntVar(&numShards, "shards", 10, "number of shards")
	flag.IntVar(&keysPerShard, "keys-per-shard", 6_00_000, "keys per shard")
	flag.IntVar(&memtableMB, "memtable-mb", 8, "memtable size in MiB")
	flag.Float64Var(&fileSizeMultiplier, "file-size-multiplier", 0.25, "file size in GiB per shard")
	flag.IntVar(&readWorkers, "readers", 16, "number of read workers")
	flag.IntVar(&writeWorkers, "writers", 16, "number of write workers")
	flag.IntVar(&sampleSecs, "sample-secs", 30, "predictor sampling window in seconds")
	flag.Int64Var(&iterations, "iterations", 100_000_000, "number of iterations")
	flag.Float64Var(&aVal, "a", 0.4, "a value for the predictor")
	flag.BoolVar(&logStats, "log-stats", true, "periodically log cache stats")
	flag.StringVar(&memProfile, "memprofile", "mem.prof", "write memory profile to this file")
	flag.StringVar(&cpuProfile, "cpuprofile", "", "write cpu profile to this file")
	flag.Parse()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	go func() {
		log.Info().Msg("Starting pprof server on :8080")
		log.Info().Msg("Access profiles at: http://localhost:8080/debug/pprof/")
		log.Info().Msg("Memory profile: http://localhost:8080/debug/pprof/heap")
		log.Info().Msg("Goroutine profile: http://localhost:8080/debug/pprof/goroutine")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Error().Err(err).Msg("pprof server failed")
		}
	}()

	// CPU profiling
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			log.Fatal().Err(err).Msg("could not create CPU profile")
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal().Err(err).Msg("could not start CPU profile")
		}
		defer pprof.StopCPUProfile()
	}

	//remove all files inside the mount point
	files, err := os.ReadDir(mountPoint)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		os.Remove(filepath.Join(mountPoint, file.Name()))
	}

	memtableSizeInBytes := int32(memtableMB) * 1024 * 1024
	fileSizeInBytes := int64(float64(fileSizeMultiplier) * 1024 * 1024 * 1024) // fileSizeMultiplier in GiB

	cfg := cachepkg.WrapCacheConfig{
		NumShards:             numShards,
		KeysPerShard:          keysPerShard,
		FileSize:              fileSizeInBytes,
		MemtableSize:          memtableSizeInBytes,
		ReWriteScoreThreshold: 0.8,
		GridSearchEpsilon:     0.0001,
		SampleDuration:        time.Duration(sampleSecs) * time.Second,
	}

	pc, err := cachepkg.NewWrapCache(cfg, mountPoint)
	if err != nil {
		panic(err)
	}

	MULTIPLIER := 300

	missedKeyChanList := make([]chan int, writeWorkers)
	for i := 0; i < writeWorkers; i++ {
		missedKeyChanList[i] = make(chan int)
	}

	totalKeys := 10_000_000
	str1kb := strings.Repeat("a", 1024)
	str1kb = "%d" + str1kb

	var wg sync.WaitGroup
	var writeWg sync.WaitGroup

	var metrics loadMetrics

	// periodic stats reporter
	stopReporter := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				metrics.printStats("PERIODIC")
			case <-stopReporter:
				return
			}
		}
	}()

	//prepopulate 70% keys
	fmt.Printf("----------------------------------------------prepopulating keys\n")
	for k := 0; k < int(totalKeys); k++ {

		if rand.Intn(100) < 30 {
			continue
		}

		key := fmt.Sprintf("key%d", k)
		val := []byte(fmt.Sprintf(str1kb, k))
		start := time.Now()
		if err := pc.Put(key, val, 60); err != nil {
			log.Error().Err(err).Msgf("error putting key %s", key)
		}
		metrics.prepopulatePutMetrics.record(time.Since(start))
		if k%5000000 == 0 {
			fmt.Printf("----------------------------------------------prepopulated %d keys\n", k)
		}
	}

	if writeWorkers > 0 {
		fmt.Printf("----------------------------------------------starting write workers\n")
		writeWg.Add(writeWorkers)

		for w := 0; w < writeWorkers; w++ {
			go func(workerID int) {
				defer writeWg.Done()

				for mk := range missedKeyChanList[workerID] {
					key := fmt.Sprintf("key%d", mk)
					val := []byte(fmt.Sprintf(str1kb, mk))
					start := time.Now()
					if err := pc.Put(key, val, 60); err != nil {
						log.Error().Err(err).Msgf("error putting key %s", key)
					}
					metrics.putMetrics.record(time.Since(start))
				}
			}(w)
		}
	}

	if readWorkers > 0 {
		fmt.Printf("----------------------------------------------reading keys\n")
		wg.Add(readWorkers)

		for r := 0; r < readWorkers; r++ {
			go func(workerID int) {
				defer wg.Done()
				for k := 0; k < totalKeys*MULTIPLIER; k += 1 {
					randomval := normalDistIntPartitioned(workerID, readWorkers, totalKeys)
					key := fmt.Sprintf("key%d", randomval)
					start := time.Now()
					val, found, expired := pc.Get(key)
					metrics.getMetrics.record(time.Since(start))

					if !found {
						metrics.getMisses.Add(1)
						writeWorkerid := randomval % writeWorkers
						missedKeyChanList[writeWorkerid] <- randomval
					} else {
						metrics.getHits.Add(1)
					}

					if expired {
						metrics.getExpired.Add(1)
						log.Error().Msgf("key %s expired", key)
					}
					if found && string(val) != fmt.Sprintf(str1kb, randomval) {
						panic("value mismatch")
					}
					if k%50000 == 0 {
						fmt.Printf("----------------------------------------------read %d keys %d readerid\n", k, workerID)
					}
				}
			}(r)
		}
	}

	// Start pprof HTTP server for runtime profiling

	wg.Wait()
	close(stopReporter)
	metrics.printStats("FINAL")
	log.Info().Msgf("done putting")

	// Memory profiling
	if memProfile != "" {
		runtime.GC() // get up-to-date statistics
		f, err := os.Create(memProfile)
		if err != nil {
			log.Fatal().Err(err).Msg("could not create memory profile")
		}
		defer f.Close()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal().Err(err).Msg("could not write memory profile")
		}
		log.Info().Msgf("Memory profile written to %s", memProfile)
	}

	// Print memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Info().
		Str("alloc", fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024)).
		Str("total_alloc", fmt.Sprintf("%.2f MB", float64(m.TotalAlloc)/1024/1024)).
		Str("sys", fmt.Sprintf("%.2f MB", float64(m.Sys)/1024/1024)).
		Uint32("num_gc", m.NumGC).
		Msg("Memory statistics")
}
