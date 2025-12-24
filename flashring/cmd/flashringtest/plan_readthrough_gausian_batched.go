package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	cachepkg "github.com/Meesho/BharatMLStack/flashring/pkg/cache"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func planReadthroughGaussianBatched() {
	var (
		mountPoint         string
		numShards          int
		keysPerShard       int
		memtableMB         int
		fileSizeMultiplier int
		readWorkers        int
		writeWorkers       int
		sampleSecs         int
		iterations         int64
		aVal               float64
		logStats           bool
		memProfile         string
		cpuProfile         string

		//batching reads
		enableBatching    bool
		batchWindowMicros int // in microseconds
		maxBatchSize      int
	)

	flag.StringVar(&mountPoint, "mount", "/media/a0d00kc/trishul/", "data directory for shard files")
	flag.IntVar(&numShards, "shards", 200, "number of shards")
	flag.IntVar(&keysPerShard, "keys-per-shard", 10_00_00, "keys per shard")
	flag.IntVar(&memtableMB, "memtable-mb", 16, "memtable size in MiB")
	flag.IntVar(&fileSizeMultiplier, "file-size-multiplier", 10, "file size in GiB per shard")
	flag.IntVar(&readWorkers, "readers", 8, "number of read workers")
	flag.IntVar(&writeWorkers, "writers", 8, "number of write workers")
	flag.IntVar(&sampleSecs, "sample-secs", 30, "predictor sampling window in seconds")
	flag.Int64Var(&iterations, "iterations", 100_000_000, "number of iterations")
	flag.Float64Var(&aVal, "a", 0.4, "a value for the predictor")
	flag.BoolVar(&logStats, "log-stats", true, "periodically log cache stats")
	flag.StringVar(&memProfile, "memprofile", "mem.prof", "write memory profile to this file")
	flag.StringVar(&cpuProfile, "cpuprofile", "", "write cpu profile to this file")

	flag.BoolVar(&enableBatching, "enable-batching", true, "enable read batching")
	flag.IntVar(&batchWindowMicros, "batch-window-us", 1, "batch window in microseconds")
	flag.IntVar(&maxBatchSize, "max-batch", 200, "max batch size")
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
	fileSizeInBytes := int64(fileSizeMultiplier) * int64(memtableSizeInBytes)

	cfg := cachepkg.WrapCacheConfig{
		NumShards:             numShards,
		KeysPerShard:          keysPerShard,
		FileSize:              fileSizeInBytes,
		MemtableSize:          memtableSizeInBytes,
		ReWriteScoreThreshold: 0.8,
		GridSearchEpsilon:     0.0001,
		SampleDuration:        time.Duration(sampleSecs) * time.Second,

		//batching reads
		EnableBatching:    enableBatching,
		BatchWindowMicros: batchWindowMicros,
		MaxBatchSize:      maxBatchSize,

		// Pass the metrics collector to record cache metrics
		MetricsRecorder: InitMetricsCollector(),
	}

	// Set additional input parameters that the cache doesn't know about
	metricsCollector.SetShards(numShards)
	metricsCollector.SetKeysPerShard(keysPerShard)
	metricsCollector.SetReadWorkers(readWorkers)
	metricsCollector.SetWriteWorkers(writeWorkers)
	metricsCollector.SetPlan("readthrough-batched")

	// Start background goroutine to wait for shutdown signal and export CSV
	go RunmetricsWaitForShutdown()

	pc, err := cachepkg.NewWrapCache(cfg, mountPoint, logStats)
	if err != nil {
		panic(err)
	}

	MULTIPLIER := 300

	missedKeyChanList := make([]chan int, writeWorkers)
	for i := 0; i < writeWorkers; i++ {
		missedKeyChanList[i] = make(chan int)
	}

	totalKeys := keysPerShard * numShards
	str1kb := strings.Repeat("a", 1024)
	str1kb = "%d" + str1kb

	var wg sync.WaitGroup
	var writeWg sync.WaitGroup

	//prepopulate 70% keys
	fmt.Printf("----------------------------------------------prepopulating keys\n")
	for k := 0; k < int(totalKeys); k++ {

		if rand.Intn(100) < 30 {
			continue
		}

		key := fmt.Sprintf("key%d", k)
		val := []byte(fmt.Sprintf(str1kb, k))
		if err := pc.Put(key, val, 60); err != nil {
			panic(err)
		}
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
					if err := pc.Put(key, val, 60); err != nil {
						panic(err)
					}
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
					// Each worker samples from its own partition of the key space
					randomval := normalDistIntPartitioned(workerID, readWorkers, totalKeys)
					key := fmt.Sprintf("key%d", randomval)
					val, found, expired := pc.Get(key)

					if !found {
						writeWorkerid := randomval % writeWorkers
						missedKeyChanList[writeWorkerid] <- randomval
					}

					if expired {
						panic("key expired")

					}
					if found && string(val) != fmt.Sprintf(str1kb, randomval) {
						panic("value mismatch")
					}
					if k%5000000 == 0 {
						fmt.Printf("----------------------------------------------read %d keys %d readerid\n", k, workerID)
					}
				}
			}(r)
		}
	}

	// Start pprof HTTP server for runtime profiling

	wg.Wait()
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
