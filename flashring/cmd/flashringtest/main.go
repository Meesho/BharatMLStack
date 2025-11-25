package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	_ "net/http/pprof"

	cachepkg "github.com/Meesho/BharatMLStack/flashring/internal/cache"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// normalDistInt returns an integer in [0, max) following a normal distribution
// centered at max/2 with standard deviation = max/6 (so ~99.7% values are in range)
func normalDistInt(max int) int {
	if max <= 0 {
		return 0
	}

	mean := float64(max) / 2.0
	stdDev := float64(max) / 6.0

	for {
		val := rand.NormFloat64()*stdDev + mean

		if val >= 0 && val < float64(max) {
			return int(val)
		}
	}
}

func main() {
	// Flags to parameterize load tests
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
	flag.IntVar(&numShards, "shards", 1, "number of shards")
	flag.IntVar(&keysPerShard, "keys-per-shard", 20_000_000, "keys per shard")
	flag.IntVar(&memtableMB, "memtable-mb", 16, "memtable size in MiB")
	flag.IntVar(&fileSizeMultiplier, "file-size-multiplier", 40, "file size in GiB per shard")
	flag.IntVar(&readWorkers, "readers", 1, "number of read workers")
	flag.IntVar(&writeWorkers, "writers", 1, "number of write workers")
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
	fileSizeInBytes := int64(fileSizeMultiplier) * int64(memtableSizeInBytes)

	cfg := cachepkg.WrapCacheConfig{
		NumShards:             numShards,
		KeysPerShard:          keysPerShard,
		FileSize:              fileSizeInBytes,
		MemtableSize:          memtableSizeInBytes,
		ReWriteScoreThreshold: 0.8,
		GridSearchEpsilon:     0.0001,
		SampleDuration:        time.Duration(sampleSecs) * time.Second,
	}

	pc, err := cachepkg.NewWrapCache(cfg, mountPoint, logStats)
	if err != nil {
		panic(err)
	}

	MULTIPLIER := 300

	totalKeys := keysPerShard * numShards
	str1kb := strings.Repeat("a", 1024)
	str1kb = "%d" + str1kb

	var wg sync.WaitGroup

	if writeWorkers > 0 {
		fmt.Printf("----------------------------------------------writing keys\n")
		wg.Add(writeWorkers)

		for w := 0; w < writeWorkers; w++ {
			go func(workerID int) {
				defer wg.Done()
				for k := 0; k < totalKeys*MULTIPLIER; k += 1 {
					randomval := normalDistInt(totalKeys)
					key := fmt.Sprintf("key%d", randomval)

					val := []byte(fmt.Sprintf(str1kb, randomval))
					if err := pc.Put(key, val, 60); err != nil {
						panic(err)
					}

					if k%5000000 == 0 {
						fmt.Printf("----------------------------------------------wrote %d keys %d writerid\n", k, workerID)
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
					val, found, expired := pc.Get(key)

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

func BucketsByWidth(a float64, n int) []float64 {
	if n <= 0 {
		return []float64{0}
	}
	b := make([]float64, n+1)
	b[0] = 0
	if math.Abs(a) < 1e-12 {
		// a ~ 0 => uniform
		for i := 1; i <= n; i++ {
			b[i] = float64(i) / float64(n)
		}
		return b
	}
	s := math.Expm1(a) / float64(n) // (e^a - 1)/n (stable)
	ia := 1.0 / a
	for i := 0; i <= n; i++ {
		b[i] = ia * math.Log1p(s*float64(i)) // ln(1 + s*i)
	}
	return b
}

func rand32(rng uint32) uint32 {
	r := rng
	r ^= r << 13
	r ^= r >> 17
	r ^= r << 5
	return r
}

// 	var (
// 		mountPoint string
// 		writers    int
// 		readers    int
// 		putCount   int64
// 		getCount   int64
// 		valSize    int
// 		logEvery   time.Duration
// 	)
// 	flag.StringVar(&mountPoint, "mount", "/tmp/ssd-cache", "data directory for shard files")
// 	flag.IntVar(&writers, "writers", 4, "number of writer goroutines")
// 	flag.IntVar(&readers, "readers", 8, "number of reader goroutines")
// 	flag.Int64Var(&putCount, "puts", 100_000_000, "total puts")
// 	flag.Int64Var(&getCount, "gets", 500_000_000, "total gets")
// 	flag.IntVar(&valSize, "valsize", 64, "value size in bytes")
// 	flag.DurationVar(&logEvery, "log-every", 5*time.Second, "progress log interval")
// 	flag.Parse()

// 	runtime.GOMAXPROCS(runtime.NumCPU())
// 	debug.SetGCPercent(100)

// 	cfg := cachepkg.WrapCacheConfig{
// 		NumShards:             2,
// 		KeysPerShard:          50_000_000,
// 		FileSize:              5 * 1024 * 1024 * 1024, // 5G
// 		MemtableSize:          5 * 1024 * 1024,        // 5MB
// 		ReWriteScoreThreshold: 0.8,
// 		GridSearchEpsilon:     0.0001,
// 		SampleDuration:        30 * time.Second,
// 	}

// 	pc, err := cachepkg.NewPerCoreWrapCache(cfg, mountPoint, false)
// 	if err != nil {
// 		panic(err)
// 	}

// 	// Metrics
// 	var putDone int64
// 	var getDone int64
// 	var putBytes int64
// 	var getBytes int64
// 	var getHits int64

// 	// Payload generator (deterministic, small)
// 	baseVal := make([]byte, valSize)
// 	for i := range baseVal {
// 		baseVal[i] = byte(i)
// 	}

// 	// Writer workers
// 	var wg sync.WaitGroup
// 	start := time.Now()

// 	wg.Add(writers)
// 	for w := 0; w < writers; w++ {
// 		go func(id int) {
// 			defer wg.Done()
// 			// Interleave ranges per worker to avoid contention on counters
// 			for i := int64(id); i < putCount; i += int64(writers) {
// 				key := fmt.Sprintf("key_%d", i)
// 				if err := pc.Put(key, baseVal, 0); err == nil {
// 					atomic.AddInt64(&putDone, 1)
// 					atomic.AddInt64(&putBytes, int64(len(baseVal)))
// 				}
// 			}
// 		}(w)
// 	}

// 	// Reader workers
// 	wg.Add(readers)
// 	for r := 0; r < readers; r++ {
// 		go func(id int) {
// 			defer wg.Done()
// 			// Spread key space; avoid keeping keys in memory
// 			// Readers probe uniformly in [0, putCount)
// 			rnd := newXorShift64(uint64(0x9e3779b97f4a7c15) + uint64(id))
// 			for i := int64(id); i < getCount; i += int64(readers) {
// 				k := int64(rnd.next() % uint64(putCount))
// 				key := fmt.Sprintf("key_%d", k)
// 				val, found, expired := pc.Get(key)
// 				if found && !expired {
// 					atomic.AddInt64(&getHits, 1)
// 					atomic.AddInt64(&getBytes, int64(len(val)))
// 				}
// 				atomic.AddInt64(&getDone, 1)
// 			}
// 		}(r)
// 	}

// 	// Progress logger
// 	doneCh := make(chan struct{})
// 	go func() {
// 		ticker := time.NewTicker(logEvery)
// 		defer ticker.Stop()
// 		for {
// 			select {
// 			case <-ticker.C:
// 				elapsed := time.Since(start)
// 				pd := atomic.LoadInt64(&putDone)
// 				gd := atomic.LoadInt64(&getDone)
// 				pb := atomic.LoadInt64(&putBytes)
// 				gb := atomic.LoadInt64(&getBytes)
// 				gh := atomic.LoadInt64(&getHits)
// 				putNsOp := float64(0)
// 				getNsOp := float64(0)
// 				if pd > 0 {
// 					putNsOp = float64(elapsed.Nanoseconds()) / float64(pd)
// 				}
// 				if gd > 0 {
// 					getNsOp = float64(elapsed.Nanoseconds()) / float64(gd)
// 				}
// 				sec := elapsed.Seconds()
// 				putBps := float64(pb) / sec
// 				getBps := float64(gb) / sec
// 				hitRate := float64(0)
// 				if gd > 0 {
// 					hitRate = float64(gh) / float64(gd)
// 				}
// 				fmt.Printf("prog: puts=%d gets=%d hitRate=%.4f put_ns/op=%.0f get_ns/op=%.0f put_MBps=%.2f get_MBps=%.2f\n",
// 					pd, gd, hitRate, putNsOp, getNsOp, putBps/1e6, getBps/1e6)
// 			case <-doneCh:
// 				return
// 			}
// 		}
// 	}()

// 	wg.Wait()
// 	close(doneCh)

// 	elapsed := time.Since(start)
// 	pd := atomic.LoadInt64(&putDone)
// 	gd := atomic.LoadInt64(&getDone)
// 	pb := atomic.LoadInt64(&putBytes)
// 	gb := atomic.LoadInt64(&getBytes)
// 	gh := atomic.LoadInt64(&getHits)
// 	putNsOp := float64(0)
// 	getNsOp := float64(0)
// 	if pd > 0 {
// 		putNsOp = float64(elapsed.Nanoseconds()) / float64(pd)
// 	}
// 	if gd > 0 {
// 		getNsOp = float64(elapsed.Nanoseconds()) / float64(gd)
// 	}
// 	sec := elapsed.Seconds()
// 	putBps := float64(pb) / sec
// 	getBps := float64(gb) / sec
// 	hitRate := float64(0)
// 	if gd > 0 {
// 		hitRate = float64(gh) / float64(gd)
// 	}
// 	fmt.Printf("done: took=%s puts=%d gets=%d hitRate=%.4f put_ns/op=%.0f get_ns/op=%.0f put_MBps=%.2f get_MBps=%.2f\n",
// 		elapsed.String(), pd, gd, hitRate, putNsOp, getNsOp, putBps/1e6, getBps/1e6)
// }

// // Simple lock-free PRNG for readers
// type xorShift64 struct{ x uint64 }

// func newXorShift64(seed uint64) *xorShift64 {
// 	if seed == 0 {
// 		seed = 1
// 	}
// 	return &xorShift64{x: seed}
// }
// func (r *xorShift64) next() uint64 {
// 	x := r.x
// 	x ^= x << 13
// 	x ^= x >> 7
// 	x ^= x << 17
// 	r.x = x
// 	return x
// }
