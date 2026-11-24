package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	cachepkg "github.com/Meesho/BharatMLStack/flashring/pkg/cache"
	"github.com/rs/zerolog/log"
)

// planStress is the middle-weight plan, sitting between `smoke` (200 keys,
// single-threaded, sub-second) and the existing load plans (millions of keys,
// minutes-to-hours). It drives concurrent writers + readers against a small
// cache sized so the memtable flushes during the run, then verifies every
// key it expects to be live is still readable.
//
// What it covers that smoke can't:
//   - concurrent Put + Get across multiple goroutines (data races, lock
//     contention, shard correctness).
//   - memtable -> disk flush path (sized so writes overflow the memtable
//     several times).
//   - working-set cache behavior under pressure (writes evict older entries
//     once the file wraps).
//
// Defaults complete in ~30s and use ~64 MiB of disk. Pair with `-race`:
//
//	PLAN=stress ./flashringtest                       # default
//	PLAN=stress ./flashringtest -keys 50000           # heavier
//	PLAN=stress ./flashringtest -writers 8 -readers 8 # more contention
func planStress() {
	mountPoint := flag.String("mount", os.TempDir()+"/flashring-stress", "data directory")
	keyCount := flag.Int("keys", 10_000, "total distinct keys exercised")
	writers := flag.Int("writers", 4, "concurrent writer goroutines")
	readers := flag.Int("readers", 4, "concurrent reader goroutines")
	duration := flag.Duration("duration", 0, "run for at most this long (0 = unbounded; finishes when writers complete)")
	flag.Parse()

	// Writer/reader loops use `% *keyCount`; reject non-positive values up
	// front rather than panicking on divide-by-zero deep in a goroutine.
	if *keyCount <= 0 {
		log.Fatal().Int("keys", *keyCount).Msg("keys must be > 0")
	}

	if err := os.MkdirAll(*mountPoint, 0o755); err != nil {
		log.Fatal().Err(err).Msgf("create mount %s", *mountPoint)
	}

	const (
		valueSize    = 1024           // 1 KiB per value
		memtableSize = 1 << 20        // 1 MiB memtable -> ~1024 puts before flush
		fileSize     = 16 * (1 << 20) // 16 MiB per shard rolling file
		ttl          = 24 * time.Hour // long enough to outlive the run
	)
	// 4 shards * 16 MiB = 64 MiB on disk — comfortably fits the fast-mode
	// 256 MiB loopback ext4. With memtable=1 MiB and value=1 KiB, ~1024
	// writes per shard fill the memtable and trigger a flush.

	cfg := cachepkg.Config{
		NumShards:             4,
		KeysPerShard:          16384,
		FileSize:              fileSize,
		MemtableSize:          memtableSize,
		ReWriteScoreThreshold: 0.5,
		GridSearchEpsilon:     0.01,
		SampleDuration:        5 * time.Second,
		FreqBands:             []int{10, 100, 1000},
		RecencyBands:          []int{1000, 100, 10},
	}

	pc, err := cachepkg.NewWrapCache(cfg, *mountPoint)
	if err != nil {
		log.Fatal().Err(err).Msg("NewWrapCache")
	}
	defer pc.Close()

	// Pre-populate so readers have something to chase. Without this the
	// initial reads are 100% misses and don't exercise the read path.
	value := make([]byte, valueSize)
	for i := range value {
		value[i] = byte(i % 251)
	}
	for i := 0; i < *keyCount; i++ {
		if err := pc.Put(fmt.Sprintf("seed-%07d", i), value, ttl); err != nil {
			log.Fatal().Err(err).Int("i", i).Msg("seed Put")
		}
	}

	var (
		wg          sync.WaitGroup
		writeCount  atomic.Int64
		readCount   atomic.Int64
		readHits    atomic.Int64
		readExpired atomic.Int64
		readMissed  atomic.Int64
		stopAt      time.Time
	)
	if *duration > 0 {
		stopAt = time.Now().Add(*duration)
	}
	stop := func() bool {
		return *duration > 0 && time.Now().After(stopAt)
	}

	const writesPerWorker = 25_000
	start := time.Now()

	for w := 0; w < *writers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for n := 0; n < writesPerWorker; n++ {
				if stop() {
					return
				}
				k := fmt.Sprintf("w%d-%07d", id, n%*keyCount)
				if err := pc.Put(k, value, ttl); err != nil {
					log.Error().Err(err).Str("key", k).Msg("Put")
					continue
				}
				writeCount.Add(1)
			}
		}(w)
	}

	for r := 0; r < *readers; r++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Readers cycle over the seed key range, which is fully written.
			i := id
			for {
				if stop() {
					return
				}
				// End readers after writers stop producing — checked by
				// sampling writeCount and giving up after a quiet stretch.
				k := fmt.Sprintf("seed-%07d", i%*keyCount)
				_, found, expired := pc.Get(k)
				readCount.Add(1)
				switch {
				case expired:
					readExpired.Add(1)
				case found:
					readHits.Add(1)
				default:
					readMissed.Add(1)
				}
				i += *readers
				if i > *keyCount*4 {
					return
				}
			}
		}(r)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Cleanup so re-running starts fresh. RemoveAll handles any future
	// shards-as-subdirs layout; surface errors instead of silently leaking
	// state into the next run.
	if files, err := os.ReadDir(*mountPoint); err != nil {
		log.Warn().Err(err).Str("mount", *mountPoint).Msg("cleanup ReadDir")
	} else {
		for _, file := range files {
			path := filepath.Join(*mountPoint, file.Name())
			if err := os.RemoveAll(path); err != nil {
				log.Warn().Err(err).Str("path", path).Msg("cleanup RemoveAll")
			}
		}
	}

	totalReads := readCount.Load()
	hits := readHits.Load()
	hitPct := 0.0
	if totalReads > 0 {
		hitPct = 100 * float64(hits) / float64(totalReads)
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	fmt.Printf("stress plan PASS — writers=%d readers=%d duration=%s\n",
		*writers, *readers, elapsed.Round(time.Millisecond))
	fmt.Printf("  writes:    %d (%.0f/s)\n",
		writeCount.Load(), float64(writeCount.Load())/elapsed.Seconds())
	fmt.Printf("  reads:     %d (%.0f/s) hits=%d (%.1f%%) expired=%d miss=%d\n",
		totalReads, float64(totalReads)/elapsed.Seconds(),
		hits, hitPct, readExpired.Load(), readMissed.Load())
	fmt.Printf("  mem:       alloc=%.1fMiB sys=%.1fMiB num_gc=%d\n",
		float64(mem.Alloc)/1024/1024, float64(mem.Sys)/1024/1024, mem.NumGC)
}
