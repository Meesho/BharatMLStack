package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"sync"

	cachepkg "github.com/Meesho/BharatMLStack/flashring/internal/cache"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func planFreecache() {

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
	)

	flag.StringVar(&mountPoint, "mount", "/media/a0d00kc/trishul/", "data directory for shard files")
	flag.IntVar(&numShards, "shards", 10, "number of shards")
	flag.IntVar(&keysPerShard, "keys-per-shard", 2_000_000, "keys per shard")
	flag.IntVar(&memtableMB, "memtable-mb", 16, "memtable size in MiB")
	flag.IntVar(&fileSizeMultiplier, "file-size-multiplier", 1, "file size in GiB per shard")
	flag.IntVar(&readWorkers, "readers", 4, "number of read workers")
	flag.IntVar(&writeWorkers, "writers", 4, "number of write workers")
	flag.IntVar(&sampleSecs, "sample-secs", 30, "predictor sampling window in seconds")
	flag.Int64Var(&iterations, "iterations", 100_000_000, "number of iterations")
	flag.Float64Var(&aVal, "a", 0.4, "a value for the predictor")
	flag.BoolVar(&logStats, "log-stats", true, "periodically log cache stats")
	flag.StringVar(&memProfile, "memprofile", "mem.prof", "write memory profile to this file")
	flag.StringVar(&cpuProfile, "cpuprofile", "", "write cpu profile to this file")
	flag.Parse()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	cfg := cachepkg.WrapCacheConfig{
		KeysPerShard: keysPerShard,
		FileSize:     1 * 1024 * 1024 * 1024,
	}

	cache, err := cachepkg.NewFreecache(cfg, logStats)
	if err != nil {
		panic(err)
	}
	debug.SetGCPercent(20)

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

		if rand.Intn(100) < 5 {
			continue
		}

		key := fmt.Sprintf("key%d", k)
		val := []byte(fmt.Sprintf(str1kb, k))
		if err := cache.Put(key, val, 60*60); err != nil {
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
					if err := cache.Put(key, val, 60*60); err != nil {
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
					randomval := normalDistInt(totalKeys)
					key := fmt.Sprintf("key%d", randomval)
					_, found, expired := cache.Get(key)

					if !found {
						writeWorkerid := randomval % writeWorkers
						missedKeyChanList[writeWorkerid] <- randomval
					}

					if expired {
						panic("key expired")
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
