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
		}, &shardLocks[i])
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

	wc.shardLocks[shardIdx].Lock()
	defer wc.shardLocks[shardIdx].Unlock()

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

	defer func() {
		metrics.Timing("flashring.get.latency", time.Since(t), []string{"shard_id", strconv.Itoa(int(shardIdx))})
	}()

	wc.shardLocks[shardIdx].RLock()
	keyFound, val, remainingTTL, expired, shouldReWrite := wc.shards[shardIdx].Get(key)

	if keyFound && !expired {
		metrics.Count("flashring.get.hit.count", 1, []string{"shard_id", strconv.Itoa(int(shardIdx))})
	}
	if expired {
		metrics.Count("flashring.get.expired.count", 1, []string{"shard_id", strconv.Itoa(int(shardIdx))})
	}
	metrics.Count("flashring.get.total.count", 1, []string{"shard_id", strconv.Itoa(int(shardIdx))})
	if shouldReWrite {
		metrics.Count("flashring.get.rewrite.count", 1, []string{"shard_id", strconv.Itoa(int(shardIdx))})
		valToWrite := make([]byte, len(val))
		copy(valToWrite, val)
		wc.shardLocks[shardIdx].RUnlock()
		wc.Put(key, valToWrite, remainingTTL)
		return valToWrite, keyFound, expired
	}
	wc.shardLocks[shardIdx].RUnlock()
	return val, keyFound, expired
}

func (wc *WrapCache) Hash(key string) uint32 {
	return uint32(xxhash.Sum64String(key) ^ Seed)
}

func (wc *WrapCache) GetShardCache(shardIdx int) *filecache.ShardCache {
	return wc.shards[shardIdx]
}
