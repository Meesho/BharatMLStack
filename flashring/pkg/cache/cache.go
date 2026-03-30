package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/iouring"
	"github.com/Meesho/BharatMLStack/flashring/internal/maths"
	filecache "github.com/Meesho/BharatMLStack/flashring/internal/shard"
	"github.com/cespare/xxhash/v2"
	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
)

const (
	rounds       = 1
	maxKeysShard = (1 << 26) // 67M
	blockSize    = 4096
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
	})

	readRing, err := iouring.NewParallelBatchIoUringReader(iouring.BatchIoUringConfig{
		RingDepth:   256,
		MaxBatch:    100,
		MaxInflight: 100,
		QueueSize:   1024,
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

func (wc *WrapCache) hash(key string) uint32 {
	return uint32(xxhash.Sum64String(key) ^ wc.seed)
}

func (wc *WrapCache) Hash(key string) uint32 {
	return wc.hash(key)
}
