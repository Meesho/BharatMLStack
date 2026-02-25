package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	"github.com/Meesho/BharatMLStack/flashring/internal/maths"
	filecache "github.com/Meesho/BharatMLStack/flashring/internal/shard"
	"github.com/cespare/xxhash/v2"
	"github.com/rs/zerolog/log"

	metrics "github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
)

/*
 Each shard can keep 67M keys
 With Round = 1, expected collision (67M)^2/(2*2^62) = 4.87×10^-4
*/

const (
	ROUNDS         = 1
	KEYS_PER_SHARD = (1 << 26)
	BLOCK_SIZE     = 4096
)

var (
	ErrNumShardLessThan1            = fmt.Errorf("num shards must be greater than 0")
	ErrKeysPerShardLessThan1        = fmt.Errorf("keys per shard must be greater than 0")
	ErrKeysPerShardGreaterThan67M   = fmt.Errorf("keys per shard must be less than 67M")
	ErrMemtableSizeLessThan1        = fmt.Errorf("memtable size must be greater than 0")
	ErrMemtableSizeGreaterThan1GB   = fmt.Errorf("memtable size must be less than 1GB")
	ErrMemtableSizeNotMultipleOf4KB = fmt.Errorf("memtable size must be a multiple of 4KB")
	ErrFileSizeLessThan1            = fmt.Errorf("file size must be greater than 0")
	ErrFileSizeNotMultipleOf4KB     = fmt.Errorf("file size must be a multiple of 4KB")
	Seed                            = xxhash.Sum64String(strconv.Itoa(int(time.Now().UnixNano())))
)

type WrapCache struct {
	shards      []*filecache.ShardCache
	shardLocks  []sync.RWMutex
	predictor   *maths.Predictor
	batchReader *fs.ParallelBatchIoUringReader // global batched io_uring reader
}

type WrapCacheConfig struct {
	NumShards             int
	KeysPerShard          int
	FileSize              int64
	MemtableSize          int32
	ReWriteScoreThreshold float32
	GridSearchEpsilon     float64
	SampleDuration        time.Duration
}

func NewWrapCache(config WrapCacheConfig, mountPoint string) (*WrapCache, error) {
	if config.NumShards <= 0 {
		return nil, ErrNumShardLessThan1
	}
	if config.KeysPerShard <= 0 {
		return nil, ErrKeysPerShardLessThan1
	}
	if config.KeysPerShard > KEYS_PER_SHARD {
		return nil, ErrKeysPerShardGreaterThan67M
	}
	if config.MemtableSize <= 0 {
		return nil, ErrMemtableSizeLessThan1
	}
	if config.MemtableSize > 1024*1024*1024 {
		return nil, ErrMemtableSizeGreaterThan1GB
	}
	if config.MemtableSize%BLOCK_SIZE != 0 {
		return nil, ErrMemtableSizeNotMultipleOf4KB
	}
	if config.FileSize <= 0 {
		return nil, ErrFileSizeLessThan1
	}
	if config.FileSize%BLOCK_SIZE != 0 {
		return nil, ErrFileSizeNotMultipleOf4KB
	}

	//clear existing data
	files, err := os.ReadDir(mountPoint)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read directory")
		panic(err)
	}
	for _, file := range files {
		os.Remove(filepath.Join(mountPoint, file.Name()))
	}

	weights := []maths.WeightTuple{
		{
			WFreq: 0.1,
			WLA:   0.1,
		},
		{
			WFreq: 0.45,
			WLA:   0.1,
		},
		{
			WFreq: 0.9,
			WLA:   0.1,
		},
		{
			WFreq: 0.1,
			WLA:   0.45,
		},
		{
			WFreq: 0.45,
			WLA:   0.45,
		},
		{
			WFreq: 0.9,
			WLA:   0.45,
		},
		{
			WFreq: 0.1,
			WLA:   0.9,
		},
		{
			WFreq: 0.45,
			WLA:   0.9,
		},
		{
			WFreq: 0.9,
			WLA:   0.9,
		},
	}
	MaxMemTableCount := config.FileSize / int64(config.MemtableSize)
	predictor := maths.NewPredictor(maths.PredictorConfig{
		ReWriteScoreThreshold: config.ReWriteScoreThreshold,
		Weights:               weights,
		SampleDuration:        config.SampleDuration,
		MaxMemTableCount:      uint32(MaxMemTableCount),
		GridSearchEpsilon:     config.GridSearchEpsilon,
	})

	// Create a single global batched io_uring reader shared across all shards.
	// All disk reads funnel into one channel; the background goroutine collects
	// them for up to 1ms and submits them in a single io_uring_enter call.
	batchReader, err := fs.NewParallelBatchIoUringReader(fs.BatchIoUringConfig{
		RingDepth: 256,
		MaxBatch:  256,
		Window:    time.Millisecond,
		QueueSize: 1024,
	}, 2)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create batched io_uring reader, falling back to per-shard rings")
		batchReader = nil
	}

	// Separate io_uring ring dedicated to batched writes (memtable flushes).
	// Kept separate from the read ring to avoid mutex contention between the
	// read batch loop and concurrent flushes.
	writeRing, err := fs.NewIoUring(256, 0)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create io_uring write ring, falling back to sequential pwrite")
		writeRing = nil
	}

	metrics.BuildShardTags(config.NumShards)
	shardLocks := make([]sync.RWMutex, config.NumShards)
	shards := make([]*filecache.ShardCache, config.NumShards)
	for i := 0; i < config.NumShards; i++ {
		shards[i] = filecache.NewShardCache(filecache.ShardCacheConfig{
			MemtableSize:        config.MemtableSize,
			Rounds:              ROUNDS,
			RbInitial:           config.KeysPerShard,
			RbMax:               config.KeysPerShard,
			DeleteAmortizedStep: 10000,
			MaxFileSize:         int64(config.FileSize),
			BlockSize:           BLOCK_SIZE,
			Directory:           mountPoint,
			Predictor:           predictor,
			BatchIoUringReader:  batchReader,
			WriteRing:           writeRing,
		}, &shardLocks[i])
	}

	wc := &WrapCache{
		shards:      shards,
		shardLocks:  shardLocks,
		predictor:   predictor,
		batchReader: batchReader,
	}

	return wc, nil
}

func (wc *WrapCache) Put(key string, value []byte, exptimeInMinutes uint16) error {

	h32 := wc.Hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	var start = time.Now()
	defer func() {
		metrics.Timing(metrics.KEY_PUT_LATENCY, time.Since(start), metrics.GetShardTag(shardIdx))
	}()

	wc.shardLocks[shardIdx].Lock()
	metrics.Timing(metrics.LATENCY_WLOCK, time.Since(start), []string{})
	defer wc.shardLocks[shardIdx].Unlock()

	err := wc.shards[shardIdx].Put(key, value, exptimeInMinutes)
	if err != nil {
		log.Error().Err(err).Msgf("Put failed for key: %s", key)
		return fmt.Errorf("put failed for key: %s", key)
	}
	metrics.Incr(metrics.KEY_PUTS, metrics.GetShardTag(shardIdx))
	if h32%100 < 10 {
		metrics.Incr(metrics.KEY_RINGBUFFER_ACTIVE_ENTRIES, metrics.GetShardTag(shardIdx))
	}

	return nil
}

func (wc *WrapCache) Get(key string) ([]byte, bool, bool) {
	h32 := wc.Hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	var start = time.Now()
	defer func() {
		metrics.Timing(metrics.KEY_GET_LATENCY, time.Since(start), metrics.GetShardTag(shardIdx))
	}()

	var keyFound bool
	var val []byte
	var valCopy []byte
	var remainingTTL uint16
	var expired bool
	var shouldReWrite bool

	func(key string, shardIdx uint32) {

		keyFound, val, remainingTTL, expired, shouldReWrite = wc.shards[shardIdx].Get(key)

		if shouldReWrite {
			//copy val into a safe variable because we are unlocking the shard
			// at the end of anon function execution
			valCopy = make([]byte, len(val))
			copy(valCopy, val)
			val = valCopy
		}
	}(key, shardIdx)

	if keyFound && !expired {
		metrics.Incr(metrics.KEY_HITS, metrics.GetShardTag(shardIdx))
	}
	if expired {
		metrics.Incr(metrics.KEY_EXPIRED_ENTRIES, metrics.GetShardTag(shardIdx))
	}
	metrics.Incr(metrics.KEY_GETS, metrics.GetShardTag(shardIdx))
	if shouldReWrite {
		metrics.Incr(metrics.KEY_REWRITES, metrics.GetShardTag(shardIdx))
	}
	if shouldReWrite {
		wc.Put(key, valCopy, remainingTTL)
	}

	//todo: track hit rate here using
	// wc.predictor.Observe(hitRate)
	return val, keyFound, expired
}

func (wc *WrapCache) Hash(key string) uint32 {
	return uint32(xxhash.Sum64String(key) ^ Seed)
}

func (wc *WrapCache) GetShardCache(shardIdx int) *filecache.ShardCache {
	return wc.shards[shardIdx]
}
