package main

import (
	"flag"
	"fmt"
	"strings"
	"sync"
	"time"

	cachepkg "github.com/Meesho/BharatMLStack/flashring/internal/cache"
	"github.com/rs/zerolog"
)

func main() {
	// Flags to parameterize load tests
	var (
		mountPoint   string
		numShards    int
		keysPerShard int
		memtableMB   int
		fileSizeGB   int
		readWorkers  int
		writeWorkers int
		sampleSecs   int
		logStats     bool
	)

	flag.StringVar(&mountPoint, "mount", "/media/a0d00kc/trishul", "data directory for shard files")
	flag.IntVar(&numShards, "shards", 3, "number of shards")
	flag.IntVar(&keysPerShard, "keys-per-shard", 30_000_000, "keys per shard")
	flag.IntVar(&memtableMB, "memtable-mb", 128, "memtable size in MiB")
	flag.IntVar(&fileSizeGB, "file-gb", 50, "file size in GiB per shard")
	flag.IntVar(&readWorkers, "readers", 20, "number of read workers")
	flag.IntVar(&writeWorkers, "writers", 15, "number of write workers")
	flag.IntVar(&sampleSecs, "sample-secs", 30, "predictor sampling window in seconds")
	flag.BoolVar(&logStats, "log-stats", true, "periodically log cache stats")
	flag.Parse()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	cfg := cachepkg.WrapCacheConfig{
		NumShards:             numShards,
		KeysPerShard:          keysPerShard,
		FileSize:              int64(fileSizeGB) * 1024 * 1024 * 1024,
		MemtableSize:          int32(memtableMB) * 1024 * 1024,
		ReWriteScoreThreshold: 0.8,
		GridSearchEpsilon:     0.0001,
		SampleDuration:        time.Duration(sampleSecs) * time.Second,
	}

	pc, err := cachepkg.NewWrapCache(cfg, mountPoint, logStats)
	if err != nil {
		panic(err)
	}

	totalKeys := keysPerShard * numShards
	str1kb := strings.Repeat("a", 1024)
	str1kb = str1kb + "%d"

	// Prepopulate for read-only or read-heavy workloads: 80% of total keys
	if readWorkers > 0 && (writeWorkers == 0 || readWorkers >= int(1.2*float64(writeWorkers))) {
		preN := int(float64(totalKeys) * 0.8)
		for i := 0; i < preN; i++ {
			key := fmt.Sprintf("key%d", i)
			val := []byte(fmt.Sprintf(str1kb, i))
			if err := pc.Put(key, val, uint64(time.Now().Unix()+3600)); err != nil {
				panic(err)
			}
		}
	}

	var wg sync.WaitGroup

	// Spawn writers: each writer covers a disjoint partition of the keyspace
	if writeWorkers > 0 {
		wg.Add(writeWorkers)
		keysPerWriter := totalKeys / writeWorkers
		for w := 0; w < writeWorkers; w++ {
			start := w * keysPerWriter
			end := start + keysPerWriter
			// last worker takes any remainder
			if w == writeWorkers-1 {
				end = totalKeys
			}
			go func(wid, s, e int) {
				defer wg.Done()
				for i := s; i < e; i++ {
					key := fmt.Sprintf("key%d", i)
					val := []byte(fmt.Sprintf("value%d", i))
					if err := pc.Put(key, val, uint64(time.Now().Unix()+3600)); err != nil {
						panic(err)
					}
				}
			}(w, start, end)
		}
	}

	// Spawn readers: each reader covers a disjoint partition
	if readWorkers > 0 {
		wg.Add(readWorkers)
		readSpan := totalKeys
		// If we prepopulated, constrain readers to prepopulated range
		if writeWorkers == 0 || readWorkers >= int(1.2*float64(writeWorkers)) {
			readSpan = int(float64(totalKeys) * 0.8)
		}
		keysPerReader := readSpan / readWorkers
		for r := 0; r < readWorkers; r++ {
			start := r * keysPerReader
			end := start + keysPerReader
			if r == readWorkers-1 {
				end = readSpan
			}
			go func(rid, s, e int) {
				defer wg.Done()
				for i := s; i < e; i++ {
					key := fmt.Sprintf("key%d", i)
					val, found, expired := pc.Get(key)
					if !found {
						panic("key not found")
					}
					if expired {
						panic("key expired")
					}
					if string(val) != fmt.Sprintf("value%d", i) {
						panic("value mismatch")
					}
				}
			}(r, start, end)
		}
	}

	wg.Wait()
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
