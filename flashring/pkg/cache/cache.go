package cache

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	"github.com/Meesho/BharatMLStack/flashring/internal/index"
	"github.com/Meesho/BharatMLStack/flashring/internal/iouring"
	"github.com/Meesho/BharatMLStack/flashring/internal/maths"
	"github.com/Meesho/BharatMLStack/flashring/internal/memtables"
	filecache "github.com/Meesho/BharatMLStack/flashring/internal/shard"
	"github.com/cespare/xxhash/v2"
	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
)

const (
	rounds             = 1
	maxKeysShard       = (1 << 26) // 67M
	blockSize          = 4096
	maxCoalescedReadSz = 65536 // cap on a merged read; must not exceed the largest slab size class (128 KiB)
)

// Cache is the common interface for all cache backends. Get returns
// (value, found, expired): found reports whether the key resolved to a
// stored entry, and expired reports whether that entry's TTL has elapsed.
// An expired hit returns (nil, false, true) on backends that surface
// expiry; on backends that drop expired entries silently it returns
// (nil, false, false).
type Cache interface {
	Put(key string, value []byte, ttl time.Duration) error
	Get(key string) (value []byte, found bool, expired bool)
	Close() error
}

// Config holds all parameters for creating a WrapCache. Several fields
// encode hard invariants — see NewWrapCache for what is checked at
// construction time.
type Config struct {
	NumShards    int
	KeysPerShard int // capped at 67M (1<<26); ring buffer addressing is 32-bit.

	// FileSize and MemtableSize must both be multiples of 4096 — O_DIRECT
	// rejects unaligned I/O with EINVAL. MemtableSize is also capped at 1GB.
	FileSize     int64
	MemtableSize int32

	ReWriteScoreThreshold float32
	GridSearchEpsilon     float64
	SampleDuration        time.Duration

	// FreqBands holds three upper bounds in [cold, warm, hot] order.
	// RecencyBands holds three upper bounds in [hot, warm, cold] order
	// (reversed because lower last-access timestamps mean more recent).
	// Both must have exactly 3 elements.
	FreqBands    []int
	RecencyBands []int
}

var (
	ErrNumShardLessThan1                 = fmt.Errorf("num shards must be greater than 0")
	ErrKeysPerShardLessThan1             = fmt.Errorf("keys per shard must be greater than 0")
	ErrKeysPerShardGreaterThan67M        = fmt.Errorf("keys per shard must be less than 67M")
	ErrMemtableSizeLessThan1             = fmt.Errorf("memtable size must be greater than 0")
	ErrMemtableSizeGreaterThan1GB        = fmt.Errorf("memtable size must be less than 1GB")
	ErrMemtableSizeNotMultipleOf4KB      = fmt.Errorf("memtable size must be a multiple of 4KB")
	ErrFileSizeLessThan1                 = fmt.Errorf("file size must be greater than 0")
	ErrFileSizeNotMultipleOf4KB          = fmt.Errorf("file size must be a multiple of 4KB")
	ErrFileSizeNotMultipleOfMemtableSize = fmt.Errorf("file size must be a multiple of memtable size — a non-multiple leaves a tail region that a mid-flush wrap will straddle, splitting one memtable's bytes across MaxFileSize and offset 0")

	// ErrValueTooLarge is returned by Put when 4 + len(key) + len(value)
	// exceeds math.MaxUint16. The index entry's length field is 16-bit,
	// so oversized values would either wrap (silent truncation) or trip
	// downstream length checks on Get. Callers must chunk large payloads
	// themselves.
	ErrValueTooLarge = fmt.Errorf("total entry size (4 + len(key) + len(value)) must fit in uint16 (max 65535 bytes)")

	// ErrPreviousFlushFailed is returned by one Put after a background
	// memtable flush failed: that memtable's entries were lost. The Put
	// that reports it is not stored. Re-exported so callers outside this
	// module can match it with errors.Is.
	ErrPreviousFlushFailed = memtables.ErrPreviousFlushFailed

	ErrBandsNeedThreeValues = fmt.Errorf("FreqBands and RecencyBands must each have exactly 3 values")

	// ErrCacheClosed is returned by Put after Close.
	ErrCacheClosed = fmt.Errorf("cache is closed")
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
		// FileSize must be a whole-number multiple of MemtableSize.
		// The writer wraps at MaxFileSize, but flushes advance in
		// MemtableSize chunks — a non-multiple leaves a tail region
		// smaller than one memtable, and the flush that starts inside
		// that tail crosses MaxFileSize mid-batch. The wrap fires
		// between chunks so the memtable's bytes end up split between
		// the seam tail and offset 0, and any chunk written past
		// MaxFileSize fails isValidReadRegion on the way back out.
		{c.MemtableSize > 0 && c.FileSize%int64(c.MemtableSize) != 0, ErrFileSizeNotMultipleOfMemtableSize},
		{len(c.FreqBands) != 3 || len(c.RecencyBands) != 3, ErrBandsNeedThreeValues},
	}
	for _, ch := range checks {
		if ch.cond {
			return ch.err
		}
	}
	return nil
}

// WrapCache is the primary disk-backed NVMe cache: a fixed set of
// shards each owning an in-memory index, a double-buffered memtable,
// and an O_DIRECT append-only file. Reads and writes are batched
// through shared io_uring rings. WrapCache is safe for concurrent use;
// per-shard locking serializes writers within a shard.
type WrapCache struct {
	shards        []*filecache.ShardCache
	shardLocks    []sync.RWMutex
	predictor     *maths.Predictor
	iouringReader *iouring.ParallelBatchIoUringReader
	iouringWriter *iouring.IoUringWriter
	seed          uint64
	closeOnce     sync.Once
	closed        atomic.Bool // set by Close; checked under the shard lock
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

// NewWrapCache constructs a sharded WrapCache backed by files under
// mountPoint. Linux only — the io_uring and O_DIRECT paths use
// linux-specific syscalls.
//
// Destructive: every existing entry under mountPoint is deleted before
// the shards are created. Pass a directory that the caller owns
// exclusively.
//
// Allocates two io_uring rings (one reader, one writer) shared across
// all shards and a Predictor for hot-data rewrite scoring. Shard
// memtables are pre-staggered by FlushStaggerOffset so flushes spread
// across the SampleDuration window instead of all firing on the same
// tick. The caller must invoke Close on the returned WrapCache to
// release the rings and open files.
//
// Returns ErrNumShardLessThan1, ErrKeysPerShardGreaterThan67M, the
// alignment errors, etc., from Config validation; otherwise wraps any
// shard-construction error and tears down rings + already-built shards
// before returning. Panics inside io_uring setup are reported via
// log.Panic — the kernel features (io_uring + SQPOLL) must be
// available.
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
	}, 1)
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
			ShardIdx:            uint32(i),
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

// Put stores value under key with the given TTL. TTL is encoded with
// minute granularity: a positive ttl smaller than one minute is rounded
// up to one minute. A ttl of zero is encoded as zero minutes, which
// expires the entry immediately on the next read. Acquires the shard's
// write lock; concurrent Puts to the same shard serialize.
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

	// Reject oversized values before touching the shard. The 16-bit
	// length field in the index entry cannot represent values where
	// 4 + len(key) + len(value) > math.MaxUint16 — the cast would wrap
	// and cause silent truncation on Put + panic on Get.
	if 4+len(key)+len(value) > math.MaxUint16 {
		metrics.Incr(metrics.KEY_VALUE_TOO_LARGE_COUNT, metrics.GetShardTag(shardIdx))
		return ErrValueTooLarge
	}
	if wc.closed.Load() {
		return ErrCacheClosed
	}

	// Consume the async-flush health flag exactly once per user-visible
	// Put. This is intentionally NOT inside ShardCache.Put — the
	// background rewrite goroutine (WrapCache.rewrite → wc.Put) would
	// otherwise consume the flag and silently discard the error, so
	// the next user-visible Put would never learn a flush had failed.
	// rewrite bypasses this check by calling shard.Put directly below.
	if wc.shards[shardIdx].MemtableManager().ConsumeFlushErr() {
		return fmt.Errorf("put failed for key(len=%d): %w", len(key), ErrPreviousFlushFailed)
	}

	// Clamp before narrowing. The index holds at most index.MaxTTLMinutes
	// (about 42 days); a longer TTL is shortened to that, never wrapped. A
	// negative TTL expires at once, like zero.
	minutes := ttl.Minutes()
	if minutes > index.MaxTTLMinutes {
		minutes = index.MaxTTLMinutes
	}
	if minutes < 0 {
		minutes = 0
	}
	ttlMinutes := uint16(minutes)
	if ttlMinutes == 0 && ttl > 0 {
		ttlMinutes = 1
	}

	// The delete manager returns index.ErrTrimNeededRetry after punching a
	// hole at the file head to make room. The retry lands in the
	// delete-in-progress branch and completes cleanly. One retry always
	// suffices because ExecuteDeleteIfNeeded won't re-trip the boundary
	// on the immediately following call.
	err := wc.shards[shardIdx].Put(key, value, ttlMinutes)
	if err != nil && errors.Is(err, index.ErrTrimNeededRetry) {
		err = wc.shards[shardIdx].Put(key, value, ttlMinutes)
	}
	if err != nil {
		// Never embed the raw caller-supplied key in the error message —
		// keys may contain user identifiers (see CLAUDE.md NEVER DO and
		// docs/SECURITY.md).
		return fmt.Errorf("put failed for key(len=%d): %w", len(key), err)
	}
	metrics.Incr(metrics.KEY_PUTS, metrics.GetShardTag(shardIdx))
	if h32%100 < 10 {
		metrics.Gauge(metrics.KEY_RINGBUFFER_ACTIVE_ENTRIES, float64(wc.shards[shardIdx].GetRingBufferActiveEntries()), metrics.GetShardTag(shardIdx))
	}
	return nil
}

// Get returns (value, found, expired). When the predictor flags an
// entry as hot enough to be moved closer to the write head, Get
// launches a background goroutine that copies value and re-Puts it
// under the same key with the remaining TTL — callers must not assume
// Get is side-effect-free or strictly read-only.
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
// hot data closer to the write head. Deliberately bypasses WrapCache.Put
// to avoid consuming the memtable-manager's flush-error flag — rewrites
// are background best-effort operations that discard their error return,
// so consuming the flag here would silently swallow the error that the
// next user-visible Put should have seen. Skipping WrapCache.Put also
// skips its trim-retry loop; a rewrite that hits a trim boundary is
// dropped rather than retried, which is acceptable for a hot-data
// optimisation.
func (wc *WrapCache) rewrite(key string, value []byte, remainingTTLMinutes uint16) {
	h32 := wc.hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	wc.shardLocks[shardIdx].Lock()
	defer wc.shardLocks[shardIdx].Unlock()

	// A rewrite started by a Get can outlive Close; the shard's fds are
	// gone by then.
	if wc.closed.Load() {
		return
	}
	_ = wc.shards[shardIdx].Put(key, value, remainingTTLMinutes)
}

// Close waits for each shard's in-flight flush, closes the per-shard
// files, and tears down the shared io_uring reader and writer rings. The
// active memtable is not flushed: the files are scratch and are wiped by
// the next NewWrapCache. Safe to call more than once. Always returns nil
// today; the error in the signature exists to satisfy the Cache interface
// and reserve future reporting. Not safe to call concurrently with Put,
// Get, or MGet.
func (wc *WrapCache) Close() error {
	wc.closeOnce.Do(wc.close)
	return nil
}

func (wc *WrapCache) close() {
	wc.closed.Store(true)
	for i := range wc.shards {
		wc.shardLocks[i].Lock()
		wc.shards[i].Close()
		wc.shardLocks[i].Unlock()
	}
	wc.iouringReader.Close()
	wc.iouringWriter.Close()
	// Tear down the predictor last — its background goroutine consumes
	// hit-rate samples that the shards' Get/MGet path may still be
	// producing until the shards are fully closed above. Predictor.Close
	// closes hitRateCh, which lets the goroutine's `for range` drain
	// and exit. Idempotent via sync.Once.
	wc.predictor.Close()
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
			// TODO(test): drive this defensive guard from a test (per
			// nilesh.solanki on PR #1031 — the guard is provably
			// unreachable today but stays here intentionally for the
			// "real-corruption" debug path; the path needs a test before
			// any future deletion).
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

// Hash returns the 32-bit shard-routing hash for key, using the same
// per-instance seed that Put, Get, and MGet apply internally. Exposed
// so that sharding-aware callers (e.g. batch loaders that pre-group
// keys per shard) can route keys without re-hashing through Get.
func (wc *WrapCache) Hash(key string) uint32 {
	return wc.hash(key)
}
