package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

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
	shards     []*filecache.ShardCache
	shardLocks []sync.RWMutex
	predictor  *maths.Predictor
}

type CacheStats struct {
	Hits                   atomic.Uint64
	TotalGets              atomic.Uint64
	TotalPuts              atomic.Uint64
	ReWrites               atomic.Uint64
	Expired                atomic.Uint64
	ShardWiseActiveEntries atomic.Uint64

	PrevHits      atomic.Uint64
	PrevTotalGets atomic.Uint64
	timeStarted   time.Time
}

type WrapCacheConfig struct {
	NumShards             int
	KeysPerShard          int
	FileSize              int64
	MemtableSize          int32
	ReWriteScoreThreshold float32
	GridSearchEpsilon     float64
	SampleDuration        time.Duration
	MountPoint            string
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

	weights := []maths.WeightTuple{{WFreq: 0.1, WLA: 0.1},
		{WFreq: 0.45, WLA: 0.1},
		{WFreq: 0.9, WLA: 0.1},
		{WFreq: 0.1, WLA: 0.45},
		{WFreq: 0.45, WLA: 0.45},
		{WFreq: 0.9, WLA: 0.45},
		{WFreq: 0.1, WLA: 0.9},
		{WFreq: 0.45, WLA: 0.9},
		{WFreq: 0.9, WLA: 0.9}}
	MaxMemTableCount := config.FileSize / int64(config.MemtableSize)
	predictor := maths.NewPredictor(maths.PredictorConfig{
		ReWriteScoreThreshold: config.ReWriteScoreThreshold,
		Weights:               weights,
		SampleDuration:        config.SampleDuration,
		MaxMemTableCount:      uint32(MaxMemTableCount),
		GridSearchEpsilon:     config.GridSearchEpsilon,
	})

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
		}, &shardLocks[i], i)
	}
	wc := &WrapCache{
		shards:     shards,
		shardLocks: shardLocks,
		predictor:  predictor,
	}
	return wc, nil
}

func (wc *WrapCache) Put(key string, value []byte, exptimeInMinutes uint16) error {
	t := time.Now()

	h32 := wc.Hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	defer func() {
		metrics.Timing("flashring.put.latency", time.Since(t), []string{"shard_id", strconv.Itoa(int(shardIdx))})
	}()

	// Measure lock acquisition time
	wc.shardLocks[shardIdx].Lock()
	defer wc.shardLocks[shardIdx].Unlock()

	// Measure work time inside lock
	err := wc.shards[shardIdx].Put(key, value, exptimeInMinutes)
	if err != nil {
		log.Error().Err(err).Msgf("Put failed for key: %s", key)
		return fmt.Errorf("put failed for key: %s", key)
	}
	metrics.Count("flashring.put.count", 1, []string{"shard_id", strconv.Itoa(int(shardIdx))})
	if h32%100 < 10 {
		metrics.Gauge("flashring.active.entries.gauge", float64(wc.shards[shardIdx].GetRingBufferActiveEntries()), []string{"shard_id", strconv.Itoa(int(shardIdx))})
	}

	return nil
}

func (wc *WrapCache) Get(key string) ([]byte, bool, bool) {
	t := time.Now()
	h32 := wc.Hash(key)
	shardIdx := h32 % uint32(len(wc.shards))
	shardTag := []string{"shard_id", strconv.Itoa(int(shardIdx))}

	defer func() {
		metrics.Timing("flashring.get.latency", time.Since(t), shardTag)
	}()

	// Phase 1: Acquire lock and get metadata
	lockStart := time.Now()
	wc.shardLocks[shardIdx].RLock()
	metrics.Timing("flashring.get.lock_acquire.latency", time.Since(lockStart), shardTag)

	metadataStart := time.Now()
	found, isInMemtable, memtableBuf, length, memId, offset, remainingTTL, expired, shouldReWrite := wc.shards[shardIdx].GetMetadata(key)
	metrics.Timing("flashring.get.metadata.latency", time.Since(metadataStart), shardTag)

	// Handle not found / expired cases
	if !found {
		wc.shardLocks[shardIdx].RUnlock()
		metrics.Count("flashring.get.not_found.count", 1, shardTag)
		metrics.Count("flashring.get.total.count", 1, shardTag)
		return nil, false, expired
	}

	if expired {
		wc.shardLocks[shardIdx].RUnlock()
		metrics.Count("flashring.get.expired.count", 1, shardTag)
		metrics.Count("flashring.get.total.count", 1, shardTag)
		return nil, false, true
	}

	// If data is in memtable, validate and return (fast path, keep lock)
	if isInMemtable {
		valid, val := wc.shards[shardIdx].ValidateBuffer(key, memtableBuf, length)
		wc.shardLocks[shardIdx].RUnlock()

		metrics.Count("flashring.get.total.count", 1, shardTag)
		metrics.Count("flashring.get.source", 1, []string{"source", "ram", "shard_id", strconv.Itoa(int(shardIdx))})

		if !valid {
			return nil, false, false
		}

		metrics.Count("flashring.get.hit.count", 1, shardTag)

		if shouldReWrite {
			metrics.Count("flashring.get.rewrite.count", 1, shardTag)
			valCopy := make([]byte, len(val))
			copy(valCopy, val)
			go wc.Put(key, valCopy, remainingTTL)
			return valCopy, true, false
		}
		return val, true, false
	}

	// Phase 2: Release lock BEFORE slow SSD read
	wc.shardLocks[shardIdx].RUnlock()
	metrics.Count("flashring.get.source", 1, []string{"source", "ssd", "shard_id", strconv.Itoa(int(shardIdx))})

	// SSD read without holding lock
	ssdStart := time.Now()
	buf := wc.shards[shardIdx].ReadFromSSD(memId, offset, length)
	metrics.Timing("flashring.get.ssd_read.latency", time.Since(ssdStart), shardTag)

	if buf == nil {
		metrics.Count("flashring.get.total.count", 1, shardTag)
		return nil, false, false
	}

	// Phase 3: Validate CRC (ensures data integrity after lock-free read)
	valid, val := wc.shards[shardIdx].ValidateBuffer(key, buf, length)
	metrics.Count("flashring.get.total.count", 1, shardTag)

	if !valid {
		metrics.Count("flashring.get.crc_fail.count", 1, shardTag)
		return nil, false, false
	}

	metrics.Count("flashring.get.hit.count", 1, shardTag)

	if shouldReWrite {
		metrics.Count("flashring.get.rewrite.count", 1, shardTag)
		go wc.Put(key, val, remainingTTL)
	}
	return val, true, false
}

func (wc *WrapCache) Hash(key string) uint32 {
	return uint32(xxhash.Sum64String(key) ^ Seed)
}

func (wc *WrapCache) GetShardCache(shardIdx int) *filecache.ShardCache {
	return wc.shards[shardIdx]
}
