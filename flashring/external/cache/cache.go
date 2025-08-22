package internal

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/external/maths"
	filecache "github.com/Meesho/BharatMLStack/flashring/external/shard"
	"github.com/cespare/xxhash/v2"
	"github.com/rs/zerolog/log"
)

/*
 Each shard can keep 67M keys
 With Round = 1, expected collision (67M)^2/(2*2^62) = 4.87×10^-4
*/

const (
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
	Seed                            = strconv.Itoa(int(time.Now().UnixNano()))
)

type WrapCache struct {
	shards          []*filecache.ShardCache
	shardLocks      []sync.RWMutex
	predictor       *maths.Predictor
	stats           []*CacheStats
	readSemaphore   chan int
	writeSemaphores []chan int // Per-shard write semaphores
}

type CacheStats struct {
	Hits                   atomic.Uint64
	TotalGets              atomic.Uint64
	TotalPuts              atomic.Uint64
	ReWrites               atomic.Uint64
	Expired                atomic.Uint64
	ShardWiseActiveEntries atomic.Uint64
}

type WrapCacheConfig struct {
	NumShards             int
	KeysPerShard          int
	FileSize              int64
	MemtableSize          int32
	ReWriteScoreThreshold float32
	GridSearchEpsilon     float64
	SampleDuration        time.Duration
	MaxConcurrentReads    int64 // Maximum concurrent read operations
	Rounds                int
}

func NewWrapCache(config WrapCacheConfig, mountPoint string, logStats bool) (*WrapCache, error) {
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
	shards := make([]*filecache.ShardCache, config.NumShards)
	for i := 0; i < config.NumShards; i++ {
		shards[i] = filecache.NewShardCache(filecache.ShardCacheConfig{
			MemtableSize:        config.MemtableSize,
			Rounds:              config.Rounds,
			RbInitial:           config.KeysPerShard,
			RbMax:               config.KeysPerShard,
			DeleteAmortizedStep: 1000,
			MaxFileSize:         int64(config.FileSize),
			BlockSize:           BLOCK_SIZE,
			Directory:           mountPoint,
			Predictor:           predictor,
		})
	}
	shardLocks := make([]sync.RWMutex, config.NumShards)
	stats := make([]*CacheStats, config.NumShards)
	for i := 0; i < config.NumShards; i++ {
		stats[i] = &CacheStats{}
	}

	// Initialize semaphores using channels
	maxReads := config.MaxConcurrentReads
	if maxReads <= 0 {
		maxReads = int64(config.NumShards * 10) // Default: 10x the number of shards
	}
	readSemaphore := make(chan int, maxReads)

	// Create per-shard write semaphores - 1 concurrent write per shard
	writeSemaphores := make([]chan int, config.NumShards)
	for i := 0; i < config.NumShards; i++ {
		writeSemaphores[i] = make(chan int, 1) // Only 1 concurrent write per shard
	}

	wc := &WrapCache{
		shards:          shards,
		shardLocks:      shardLocks,
		predictor:       predictor,
		stats:           stats,
		readSemaphore:   readSemaphore,
		writeSemaphores: writeSemaphores,
	}
	if logStats {
		go func() {
			sleepDuration := 10 * time.Second
			perShardPrevTotalGets := make([]uint64, config.NumShards)
			perShardPrevTotalPuts := make([]uint64, config.NumShards)
			for {
				time.Sleep(sleepDuration)
				for i := 0; i < config.NumShards; i++ {
					log.Info().Msgf("Shard %d has %d active entries", i, wc.stats[i].ShardWiseActiveEntries.Load())
					total := wc.stats[i].TotalGets.Load()
					hits := wc.stats[i].Hits.Load()
					hitRate := float64(0)
					if total > 0 {
						hitRate = float64(hits) / float64(total)
					}
					log.Info().Msgf("Shard %d HitRate: %v", i, hitRate)
					log.Info().Msgf("Shard %d ReWrites: %v", i, wc.stats[i].ReWrites.Load())
					log.Info().Msgf("Shard %d Expired: %v", i, wc.stats[i].Expired.Load())
					log.Info().Msgf("Shard %d Total: %v", i, total)
					log.Info().Msgf("Gets/sec: %v", float64(total-perShardPrevTotalGets[i])/float64(sleepDuration.Seconds()))
					log.Info().Msgf("Puts/sec: %v", float64(wc.stats[i].TotalPuts.Load()-perShardPrevTotalPuts[i])/float64(sleepDuration.Seconds()))
					perShardPrevTotalGets[i] = total
					perShardPrevTotalPuts[i] = wc.stats[i].TotalPuts.Load()
				}
				log.Info().Msgf("GridSearchActive: %v", wc.predictor.GridSearchEstimator.IsGridSearchActive())
			}
		}()
	}
	return wc, nil
}

func (wc *WrapCache) Put(key string, value []byte, exptime uint64) error {
	h32 := hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	// Acquire per-shard write semaphore to limit concurrent writes per shard
	select {
	case wc.writeSemaphores[shardIdx] <- 1:
		// Successfully acquired write semaphore for this shard
	default:
		// Write semaphore full for this shard, block until available
		wc.writeSemaphores[shardIdx] <- 1
	}
	defer func() { <-wc.writeSemaphores[shardIdx] }() // Release write semaphore for this shard

	// Write semaphore acquired, proceed with write operation

	wc.shardLocks[shardIdx].Lock()
	defer wc.shardLocks[shardIdx].Unlock()
	wc.shards[shardIdx].Put(key, value, exptime)
	wc.stats[shardIdx].TotalPuts.Add(1)
	if h32%100 < 10 {
		wc.stats[shardIdx].ShardWiseActiveEntries.Store(uint64(wc.shards[shardIdx].GetRingBufferActiveEntries()))
	}
	return nil
}

func (wc *WrapCache) Get(key string) ([]byte, bool, bool) {
	h32 := hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	// Acquire read semaphore to limit concurrent reads
	select {
	case wc.readSemaphore <- 1:
		// Successfully acquired read semaphore
	default:
		// Read semaphore full, block until available
		wc.readSemaphore <- 1
	}
	defer func() { <-wc.readSemaphore }() // Release read semaphore

	// Use RLock for read operations - this will naturally wait for any exclusive writes to complete
	wc.shardLocks[shardIdx].RLock()
	val, exptime, keyFound, expired, shouldReWrite := wc.shards[shardIdx].Get(key)
	wc.shardLocks[shardIdx].RUnlock()

	if keyFound && !expired {
		wc.stats[shardIdx].Hits.Add(1)
	}
	if expired {
		wc.stats[shardIdx].Expired.Add(1)
	}
	wc.stats[shardIdx].TotalGets.Add(1)
	if shouldReWrite {
		wc.stats[shardIdx].ReWrites.Add(1)
		wc.Put(key, val, exptime)
	}
	if h32%100 < 10 {
		wc.predictor.Observe(float64(wc.stats[shardIdx].Hits.Load()) / float64(wc.stats[shardIdx].TotalGets.Load()))
	}
	return val, keyFound, expired
}

func hash(key string) uint32 {
	nKey := key + Seed
	return uint32(xxhash.Sum64String(nKey))
}
