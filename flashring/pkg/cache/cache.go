package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	"github.com/Meesho/BharatMLStack/flashring/internal/iouring"
	"github.com/Meesho/BharatMLStack/flashring/internal/maths"
	filecache "github.com/Meesho/BharatMLStack/flashring/internal/shard"
	"github.com/cespare/xxhash/v2"
	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
)

const (
	rounds             = 1
	maxKeysShard       = (1 << 26) // 67M
	blockSize          = 4096
	maxCoalescedReadSz = 65536 // must match the largest slab allocator size class
)

// Cache is the common interface for all cache backends.
type Cache interface {
	Put(key string, value []byte, ttl time.Duration) error
	Get(key string) (value []byte, found bool, expired bool)
	Close() error
}

// Config holds all parameters for creating a WrapCache.
type Config struct {
	NumShards             int
	KeysPerShard          int
	FileSize              int64
	MemtableSize          int32
	ReWriteScoreThreshold float32
	GridSearchEpsilon     float64
	SampleDuration        time.Duration
	FreqBands             []int
	RecencyBands          []int
}

var (
	ErrNumShardLessThan1            = fmt.Errorf("num shards must be greater than 0")
	ErrKeysPerShardLessThan1        = fmt.Errorf("keys per shard must be greater than 0")
	ErrKeysPerShardGreaterThan67M   = fmt.Errorf("keys per shard must be less than 67M")
	ErrMemtableSizeLessThan1        = fmt.Errorf("memtable size must be greater than 0")
	ErrMemtableSizeGreaterThan1GB   = fmt.Errorf("memtable size must be less than 1GB")
	ErrMemtableSizeNotMultipleOf4KB = fmt.Errorf("memtable size must be a multiple of 4KB")
	ErrFileSizeLessThan1            = fmt.Errorf("file size must be greater than 0")
	ErrFileSizeNotMultipleOf4KB     = fmt.Errorf("file size must be a multiple of 4KB")
)

func (c *Config) validate() error {
	checks := []struct {
		cond bool
		err  error
	}{
		{c.NumShards <= 0, ErrNumShardLessThan1},
		{c.KeysPerShard <= 0, ErrKeysPerShardLessThan1},
		{c.KeysPerShard > maxKeysShard, ErrKeysPerShardGreaterThan67M},
		{c.MemtableSize <= 0, ErrMemtableSizeLessThan1},
		{c.MemtableSize > 1<<30, ErrMemtableSizeGreaterThan1GB},
		{c.MemtableSize%blockSize != 0, ErrMemtableSizeNotMultipleOf4KB},
		{c.FileSize <= 0, ErrFileSizeLessThan1},
		{c.FileSize%blockSize != 0, ErrFileSizeNotMultipleOf4KB},
	}
	for _, ch := range checks {
		if ch.cond {
			return ch.err
		}
	}
	return nil
}

// WrapCache is the primary disk-backed NVMe cache.
type WrapCache struct {
	shards        []*filecache.ShardCache
	shardLocks    []sync.RWMutex
	predictor     *maths.Predictor
	iouringReader *iouring.ParallelBatchIoUringReader
	iouringWriter *iouring.IoUringWriter
	seed          uint64
}

var defaultWeights = []maths.WeightTuple{
	{WFreq: 0.1, WLA: 0.1},
	{WFreq: 0.45, WLA: 0.1},
	{WFreq: 0.9, WLA: 0.1},
	{WFreq: 0.1, WLA: 0.45},
	{WFreq: 0.45, WLA: 0.45},
	{WFreq: 0.9, WLA: 0.45},
	{WFreq: 0.1, WLA: 0.9},
	{WFreq: 0.45, WLA: 0.9},
	{WFreq: 0.9, WLA: 0.9},
}

func NewWrapCache(config Config, mountPoint string) (*WrapCache, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	files, err := os.ReadDir(mountPoint)
	if err != nil {
		return nil, fmt.Errorf("read mount point: %w", err)
	}
	for _, file := range files {
		os.Remove(filepath.Join(mountPoint, file.Name()))
	}

	maxMemTableCount := config.FileSize / int64(config.MemtableSize)
	predictor := maths.NewPredictor(maths.PredictorConfig{
		ReWriteScoreThreshold: config.ReWriteScoreThreshold,
		Weights:               defaultWeights,
		SampleDuration:        config.SampleDuration,
		MaxMemTableCount:      uint32(maxMemTableCount),
		GridSearchEpsilon:     config.GridSearchEpsilon,
		FreqBands:             maths.FreqBands{Cold: uint64(config.FreqBands[0]), Warm: uint64(config.FreqBands[1]), Hot: uint64(config.FreqBands[2])},
		RecencyBands:          maths.RecencyBands{Hot: uint64(config.RecencyBands[0]), Warm: uint64(config.RecencyBands[1]), Cold: uint64(config.RecencyBands[2])},
	})

	readRing, err := iouring.NewParallelBatchIoUringReader(iouring.BatchIoUringConfig{
		RingDepth:   512,
		MaxBatch:    512,
		MaxInflight: 512,
		QueueSize:   2048,
		Window:      0,
		SQPoll:      true,
	}, 4)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to create batched io_uring reader")
	}

	writeRing, err := iouring.NewIoUringWriter(256, 0)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to create io_uring write ring")
	}

	seed := xxhash.Sum64String(strconv.Itoa(int(time.Now().UnixNano())))

	metrics.BuildShardTags(config.NumShards)
	shardLocks := make([]sync.RWMutex, config.NumShards)
	shards := make([]*filecache.ShardCache, config.NumShards)

	// Stagger each shard's first memtable fill level so flushes are spread
	// evenly over time instead of all firing at once. Shard i starts with
	// i/N of its memtable already "used", so it fills sooner by that fraction.
	// After the first cycle the stagger is self-sustaining.
	staggerStep := (int(config.MemtableSize) / config.NumShards) &^ (blockSize - 1) // block-align

	for i := 0; i < config.NumShards; i++ {
		shards[i], err = filecache.NewShardCache(filecache.ShardCacheConfig{
			MemtableSize:        config.MemtableSize,
			Rounds:              rounds,
			RbInitial:           config.KeysPerShard,
			RbMax:               config.KeysPerShard,
			DeleteAmortizedStep: 10000,
			MaxFileSize:         config.FileSize,
			BlockSize:           blockSize,
			Directory:           mountPoint,
			Predictor:           predictor,
			IoUringReader:       readRing,
			IoUringWriter:       writeRing,
			FlushStaggerOffset:  i * staggerStep,
		}, &shardLocks[i])
		if err != nil {
			for j := 0; j < i; j++ {
				shards[j].Close()
			}
			readRing.Close()
			writeRing.Close()
			return nil, fmt.Errorf("create shard %d: %w", i, err)
		}
	}

	return &WrapCache{
		shards:        shards,
		shardLocks:    shardLocks,
		predictor:     predictor,
		iouringReader: readRing,
		iouringWriter: writeRing,
		seed:          seed,
	}, nil
}

func (wc *WrapCache) Put(key string, value []byte, ttl time.Duration) error {
	h32 := wc.hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	start := time.Now()
	defer func() {
		metrics.Timing(metrics.KEY_PUT_LATENCY, time.Since(start), metrics.GetShardTag(shardIdx))
	}()

	wc.shardLocks[shardIdx].Lock()
	metrics.Timing(metrics.LATENCY_WLOCK, time.Since(start), []string{})
	defer wc.shardLocks[shardIdx].Unlock()

	ttlMinutes := uint16(ttl.Minutes())
	if ttlMinutes == 0 && ttl > 0 {
		ttlMinutes = 1
	}

	if err := wc.shards[shardIdx].Put(key, value, ttlMinutes); err != nil {
		return fmt.Errorf("put failed for key %s: %w", key, err)
	}
	metrics.Incr(metrics.KEY_PUTS, metrics.GetShardTag(shardIdx))
	if h32%100 < 10 {
		metrics.Incr(metrics.KEY_RINGBUFFER_ACTIVE_ENTRIES, metrics.GetShardTag(shardIdx))
	}
	return nil
}

func (wc *WrapCache) Get(key string) ([]byte, bool, bool) {
	h32 := wc.hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	start := time.Now()
	defer func() {
		metrics.Timing(metrics.KEY_GET_LATENCY, time.Since(start), metrics.GetShardTag(shardIdx))
	}()

	keyFound, val, remainingTTL, expired, shouldReWrite := wc.shards[shardIdx].Get(key)

	if keyFound && !expired {
		metrics.Incr(metrics.KEY_HITS, metrics.GetShardTag(shardIdx))
	}
	if expired {
		metrics.Incr(metrics.KEY_EXPIRED_ENTRIES, metrics.GetShardTag(shardIdx))
	}
	metrics.Incr(metrics.KEY_GETS, metrics.GetShardTag(shardIdx))

	if shouldReWrite {
		metrics.Incr(metrics.KEY_REWRITES, metrics.GetShardTag(shardIdx))
		valCopy := make([]byte, len(val))
		copy(valCopy, val)
		go wc.rewrite(key, valCopy, remainingTTL)
	}

	return val, keyFound, expired
}

// Delete removes the key from the index only. The data remains on disk
// but becomes unreachable via Get. Debug use only.
func (wc *WrapCache) Delete(key string) bool {
	h32 := wc.hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	wc.shardLocks[shardIdx].Lock()
	defer wc.shardLocks[shardIdx].Unlock()

	return wc.shards[shardIdx].DeleteKey(key)
}

// rewrite puts the value back into the cache asynchronously to move
// hot data closer to the write head.
func (wc *WrapCache) rewrite(key string, value []byte, remainingTTLMinutes uint16) {
	wc.Put(key, value, time.Duration(remainingTTLMinutes)*time.Minute)
}

func (wc *WrapCache) Close() error {
	for i := range wc.shards {
		wc.shardLocks[i].Lock()
		wc.shards[i].Flush()
		wc.shards[i].Close()
		wc.shardLocks[i].Unlock()
	}
	wc.iouringReader.Close()
	wc.iouringWriter.Close()
	return nil
}

// MGetResult holds the result for a single key in a batch get.
type MGetResult struct {
	Value   []byte
	Found   bool
	Expired bool
}

// MGet fetches multiple keys in a single call, batching disk I/O through
// io_uring for significantly lower and more consistent latency than issuing
// individual Get calls (even concurrently via goroutines).
//
// The operation runs in four phases on a single goroutine:
//  1. Index lookups + memtable checks for every key (in-memory, fast).
//  2. Coalesce: sort pending reads by (shard, offset), merge overlapping
//     aligned ranges so nearby keys share a single io_uring pread.
//  3. Submit one pread per coalesced group in a tight loop.
//  4. Collect completions, scatter to individual keys, validate CRC32.
func (wc *WrapCache) MGet(keys []string) []MGetResult {
	results := make([]MGetResult, len(keys))

	type pendingEntry struct {
		keyIdx   int
		key      string
		shardIdx uint32
		meta     filecache.MGetMeta
	}

	var diskReads []pendingEntry

	// ── Phase 1: index lookups + memtable checks (sequential, all in-memory) ──
	for i, key := range keys {
		h32 := wc.hash(key)
		shardIdx := h32 % uint32(len(wc.shards))

		meta := wc.shards[shardIdx].GetMetaForMGet(key)
		metrics.Incr(metrics.KEY_GETS, metrics.GetShardTag(shardIdx))

		if meta.Expired {
			metrics.Incr(metrics.KEY_EXPIRED_ENTRIES, metrics.GetShardTag(shardIdx))
			results[i] = MGetResult{Expired: true}
			continue
		}

		if !meta.Found {
			continue
		}

		// Memtable hit — validate and return inline.
		if meta.Value != nil {
			val, ok := wc.shards[shardIdx].ValidateAndExtract(meta.Value, key, meta.Length)
			if ok {
				metrics.Incr(metrics.KEY_HITS, metrics.GetShardTag(shardIdx))
				results[i] = MGetResult{Value: val, Found: true}
			}
			if meta.ShouldReWrite && ok {
				metrics.Incr(metrics.KEY_REWRITES, metrics.GetShardTag(shardIdx))
				valCopy := make([]byte, len(val))
				copy(valCopy, val)
				go wc.rewrite(key, valCopy, meta.RemainingTTL)
			}
			continue
		}

		// Needs disk read — collect for coalescing.
		if meta.NeedsDiskRead {
			diskReads = append(diskReads, pendingEntry{
				keyIdx:   i,
				key:      key,
				shardIdx: shardIdx,
				meta:     meta,
			})
		}
	}

	if len(diskReads) == 0 {
		return results
	}

	// ── Phase 2: coalesce nearby disk reads ──
	// Sort by (shard, file offset) so keys that map to overlapping or
	// adjacent 4KB-aligned blocks end up next to each other.
	sort.Slice(diskReads, func(i, j int) bool {
		if diskReads[i].shardIdx != diskReads[j].shardIdx {
			return diskReads[i].shardIdx < diskReads[j].shardIdx
		}
		return diskReads[i].meta.FileOffset < diskReads[j].meta.FileOffset
	})

	type coalescedGroup struct {
		shardIdx     uint32
		alignedStart int64
		alignedEnd   int64 // exclusive
		members      []int // indices into diskReads
		pending      *filecache.CoalescedPendingRead
	}

	groups := make([]coalescedGroup, 0, len(diskReads))
	for i, dr := range diskReads {
		aStart, aSize := fs.AlignRange(dr.meta.FileOffset, int(dr.meta.Length), fs.BLOCK_SIZE)
		aEnd := aStart + aSize

		if len(groups) > 0 {
			last := &groups[len(groups)-1]
			// Merge if same shard, overlapping/adjacent, and the result
			// still fits within the slab allocator's largest size class.
			mergedEnd := last.alignedEnd
			if aEnd > mergedEnd {
				mergedEnd = aEnd
			}
			if dr.shardIdx == last.shardIdx && aStart <= last.alignedEnd &&
				mergedEnd-last.alignedStart <= maxCoalescedReadSz {
				last.alignedEnd = mergedEnd
				last.members = append(last.members, i)
				continue
			}
		}

		groups = append(groups, coalescedGroup{
			shardIdx:     dr.shardIdx,
			alignedStart: aStart,
			alignedEnd:   aEnd,
			members:      []int{i},
		})
	}

	// ── Phase 3: submit one io_uring pread per coalesced group ──
	for g := range groups {
		size := int(groups[g].alignedEnd - groups[g].alignedStart)
		pr, err := wc.shards[groups[g].shardIdx].SubmitCoalescedReadAsync(
			groups[g].alignedStart, size)
		if err != nil {
			continue
		}
		groups[g].pending = pr
	}

	// ── Phase 4: collect completions, scatter to individual keys ──
	for _, grp := range groups {
		if grp.pending == nil {
			continue
		}

		coalescedBuf := wc.shards[grp.shardIdx].CollectCoalescedRead(grp.pending)
		if coalescedBuf == nil {
			continue
		}

		for _, memberIdx := range grp.members {
			dr := diskReads[memberIdx]
			bufOffset := int(dr.meta.FileOffset - grp.alignedStart)
			if bufOffset < 0 || bufOffset+int(dr.meta.Length) > len(coalescedBuf) {
				continue
			}

			keyBuf := coalescedBuf[bufOffset : bufOffset+int(dr.meta.Length)]
			val, ok := wc.shards[dr.shardIdx].ValidateAndExtract(keyBuf, dr.key, dr.meta.Length)
			if ok {
				metrics.Incr(metrics.KEY_HITS, metrics.GetShardTag(dr.shardIdx))
				results[dr.keyIdx] = MGetResult{Value: val, Found: true}
			}
			if dr.meta.ShouldReWrite && ok {
				metrics.Incr(metrics.KEY_REWRITES, metrics.GetShardTag(dr.shardIdx))
				valCopy := make([]byte, len(val))
				copy(valCopy, val)
				go wc.rewrite(dr.key, valCopy, dr.meta.RemainingTTL)
			}
		}
	}

	return results
}

func (wc *WrapCache) hash(key string) uint32 {
	return uint32(xxhash.Sum64String(key) ^ wc.seed)
}

func (wc *WrapCache) Hash(key string) uint32 {
	return wc.hash(key)
}
