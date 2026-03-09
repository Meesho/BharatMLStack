package main

import (
	"flag"
	"fmt"
	"math/bits"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"sync/atomic"
	"time"

	_ "net/http/pprof"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// normalDistInt returns an integer in [0, max) following a normal distribution
// centered at max/2 with standard deviation = max/8
func normalDistInt(max int) int {
	if max <= 0 {
		return 0
	}
	mean := float64(max) / 2.0
	stdDev := float64(max) / 8.0
	for {
		val := rand.NormFloat64()*stdDev + mean
		if val >= 0 && val < float64(max) {
			return int(val)
		}
	}
}

// normalDistIntPartitioned returns an integer following a normal distribution
// constrained to a specific worker's partition of the total key space.
func normalDistIntPartitioned(workerID, numWorkers, totalKeys int) int {
	if totalKeys <= 0 || numWorkers <= 0 {
		return 0
	}
	partitionSize := totalKeys / numWorkers
	partitionStart := workerID * partitionSize
	partitionEnd := partitionStart + partitionSize
	if workerID == numWorkers-1 {
		partitionEnd = totalKeys
	}
	mean := float64(totalKeys) / 2.0
	stdDev := float64(totalKeys) / 8.0
	for {
		val := rand.NormFloat64()*stdDev + mean
		if val >= float64(partitionStart) && val < float64(partitionEnd) {
			return int(val)
		}
	}
}

// ---- Shared metrics & profiling infrastructure ----

const histBuckets = 32

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

// commonFlags holds the shared flags across all plans.
type commonFlags struct {
	mountPoint         string
	numShards          int
	keysPerShard       int
	memtableMB         int
	fileSizeMultiplier float64
	readWorkers        int
	writeWorkers       int
	sampleSecs         int
	iterations         int64
	logStats           bool
	memProfile         string
	cpuProfile         string
}

func (f *commonFlags) register(fs *flag.FlagSet, defaults commonFlags) {
	fs.StringVar(&f.mountPoint, "mount", defaults.mountPoint, "data directory for shard files")
	fs.IntVar(&f.numShards, "shards", defaults.numShards, "number of shards")
	fs.IntVar(&f.keysPerShard, "keys-per-shard", defaults.keysPerShard, "keys per shard")
	fs.IntVar(&f.memtableMB, "memtable-mb", defaults.memtableMB, "memtable size in MiB")
	fs.Float64Var(&f.fileSizeMultiplier, "file-size-multiplier", defaults.fileSizeMultiplier, "file size in GiB per shard")
	fs.IntVar(&f.readWorkers, "readers", defaults.readWorkers, "number of read workers")
	fs.IntVar(&f.writeWorkers, "writers", defaults.writeWorkers, "number of write workers")
	fs.IntVar(&f.sampleSecs, "sample-secs", defaults.sampleSecs, "predictor sampling window in seconds")
	fs.Int64Var(&f.iterations, "iterations", defaults.iterations, "number of iterations")
	fs.BoolVar(&f.logStats, "log-stats", defaults.logStats, "periodically log cache stats")
	fs.StringVar(&f.memProfile, "memprofile", defaults.memProfile, "write memory profile to this file")
	fs.StringVar(&f.cpuProfile, "cpuprofile", defaults.cpuProfile, "write cpu profile to this file")
}

func (f *commonFlags) memtableSizeBytes() int32 {
	return int32(f.memtableMB) * 1024 * 1024
}

func (f *commonFlags) fileSizeBytes() int64 {
	return int64(f.fileSizeMultiplier * 1024 * 1024 * 1024)
}

// setupProfiling starts pprof, CPU profiling and returns a teardown function
// that writes the memory profile.
func setupProfiling(flags commonFlags) func() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	go func() {
		log.Info().Msg("Starting pprof server on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Error().Err(err).Msg("pprof server failed")
		}
	}()

	if flags.cpuProfile != "" {
		f, err := os.Create(flags.cpuProfile)
		if err != nil {
			log.Fatal().Err(err).Msg("could not create CPU profile")
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			f.Close()
			log.Fatal().Err(err).Msg("could not start CPU profile")
		}
	}

	return func() {
		pprof.StopCPUProfile()

		if flags.memProfile != "" {
			runtime.GC()
			f, err := os.Create(flags.memProfile)
			if err != nil {
				log.Fatal().Err(err).Msg("could not create memory profile")
			}
			defer f.Close()
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal().Err(err).Msg("could not write memory profile")
			}
			log.Info().Msgf("Memory profile written to %s", flags.memProfile)
		}

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		log.Info().
			Str("alloc", fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024)).
			Str("total_alloc", fmt.Sprintf("%.2f MB", float64(m.TotalAlloc)/1024/1024)).
			Str("sys", fmt.Sprintf("%.2f MB", float64(m.Sys)/1024/1024)).
			Uint32("num_gc", m.NumGC).
			Msg("Memory statistics")
	}
}

// ---- Plan registry ----

type plan func()

var plans = map[string]plan{
	"freecache":           planFreecache,
	"readthrough":         planReadthroughGaussian,
	"random":              planRandomGaussian,
	"readthrough-batched": planReadthroughGaussianBatched,
	"badger":              planBadger,
}

func main() {
	name := os.Getenv("PLAN")
	p, ok := plans[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown plan %q, available: ", name)
		for k := range plans {
			fmt.Fprintf(os.Stderr, "%s ", k)
		}
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
	p()
}
