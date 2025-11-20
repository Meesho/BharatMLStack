package internal

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/maths"
	filecache "github.com/Meesho/BharatMLStack/flashring/internal/shard"
	"github.com/cespare/xxhash/v2"
	"github.com/rs/zerolog/log"
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
	Seed                            = strconv.Itoa(int(time.Now().UnixNano()))
)

type WrapCache struct {
	shards     []*filecache.ShardCache
	shardLocks []sync.RWMutex
	predictor  *maths.Predictor
	stats      []*CacheStats
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
			Rounds:              ROUNDS,
			RbInitial:           config.KeysPerShard,
			RbMax:               config.KeysPerShard,
			DeleteAmortizedStep: 10000,
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
	wc := &WrapCache{
		shards:     shards,
		shardLocks: shardLocks,
		predictor:  predictor,
		stats:      stats,
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

					getP25, getP50, getP99 := wc.shards[i].Stats.GetLatencyPercentiles()
					putP25, putP50, putP99 := wc.shards[i].Stats.PutLatencyPercentiles()
					log.Info().Msgf("Get Latencies - P25: %v, P50: %v, P99: %v", getP25, getP50, getP99)
					log.Info().Msgf("Put Latencies - P25: %v, P50: %v, P99: %v", putP25, putP50, putP99)

				}
				log.Info().Msgf("GridSearchActive: %v", wc.predictor.GridSearchEstimator.IsGridSearchActive())
			}
		}()
	}
	return wc, nil
}

func (wc *WrapCache) Put(key string, value []byte, exptimeInMinutes uint16) error {
	h32 := hash(key)
	shardIdx := h32 % uint32(len(wc.shards))
	wc.shardLocks[shardIdx].Lock()
	defer wc.shardLocks[shardIdx].Unlock()
	wc.shards[shardIdx].Put(key, value, exptimeInMinutes)
	wc.stats[shardIdx].TotalPuts.Add(1)
	if h32%100 < 10 {
		wc.stats[shardIdx].ShardWiseActiveEntries.Store(uint64(wc.shards[shardIdx].GetRingBufferActiveEntries()))
	}
	return nil
}

func (wc *WrapCache) Get(key string) ([]byte, bool, bool) {
	h32 := hash(key)
	shardIdx := h32 % uint32(len(wc.shards))
	wc.shardLocks[shardIdx].Lock()
	defer wc.shardLocks[shardIdx].Unlock()
	keyFound, val, remainingTTL, expired, shouldReWrite := wc.shards[shardIdx].Get(key)
	if keyFound && !expired {
		wc.stats[shardIdx].Hits.Add(1)
	}
	if expired {
		wc.stats[shardIdx].Expired.Add(1)
	}
	wc.stats[shardIdx].TotalGets.Add(1)
	if shouldReWrite {
		wc.stats[shardIdx].ReWrites.Add(1)
		wc.Put(key, val, remainingTTL)
	}
	wc.predictor.Observe(float64(wc.stats[shardIdx].Hits.Load()) / float64(wc.stats[shardIdx].TotalGets.Load()))
	return val, keyFound, expired
}

func hash(key string) uint32 {
	nKey := key + Seed
	return uint32(xxhash.Sum64String(nKey))
}
