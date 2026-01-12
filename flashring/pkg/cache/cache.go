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
	shards           []*filecache.ShardCache
	shardLocks       []sync.RWMutex
	predictor        *maths.Predictor
	stats            []*CacheStats
	metricsCollector *metrics.MetricsCollector
}

type CacheStats struct {
	Hits                   atomic.Uint64
	TotalGets              atomic.Uint64
	TotalPuts              atomic.Uint64
	ReWrites               atomic.Uint64
	Expired                atomic.Uint64
	ShardWiseActiveEntries atomic.Uint64
	LatencyTracker         *filecache.LatencyTracker
	BatchTracker           *filecache.BatchTracker
}

type WrapCacheConfig struct {
	NumShards             int
	KeysPerShard          int
	FileSize              int64
	MemtableSize          int32
	ReWriteScoreThreshold float32
	GridSearchEpsilon     float64
	SampleDuration        time.Duration

	// Batching reads
	EnableBatching    bool
	BatchWindowMicros int // in microseconds
	MaxBatchSize      int

	//lockless mode for PutLL/GetLL
	EnableLockless bool

	// Optional metrics recorder
	MetricsRecorder metrics.MetricsRecorder

	//Badger
	MountPoint string
}

func NewWrapCache(config WrapCacheConfig, mountPoint string, metricsCollector *metrics.MetricsCollector) (*WrapCache, error) {
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

	batchWindow := time.Duration(0)
	if config.EnableBatching && config.BatchWindowMicros > 0 {
		batchWindow = time.Duration(config.BatchWindowMicros) * time.Microsecond
	}
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

			//batching reads
			EnableBatching: config.EnableBatching,
			BatchWindow:    batchWindow,
			MaxBatchSize:   config.MaxBatchSize,

			//lockless mode for PutLL/GetLL
			EnableLockless: config.EnableLockless,
		}, &shardLocks[i])
	}

	stats := make([]*CacheStats, config.NumShards)
	for i := 0; i < config.NumShards; i++ {
		stats[i] = &CacheStats{LatencyTracker: filecache.NewLatencyTracker(), BatchTracker: filecache.NewBatchTracker()}
	}
	wc := &WrapCache{
		shards:           shards,
		shardLocks:       shardLocks,
		predictor:        predictor,
		stats:            stats,
		metricsCollector: metricsCollector,
	}

	if metricsCollector.Config.StatsEnabled {

		go func() {
			sleepDuration := 10 * time.Second
			perShardPrevTotalGets := make([]uint64, config.NumShards)
			perShardPrevTotalPuts := make([]uint64, config.NumShards)

			for {
				time.Sleep(sleepDuration)

				for i := 0; i < config.NumShards; i++ {
					total := wc.stats[i].TotalGets.Load()

					activeEntries := float64(wc.stats[i].ShardWiseActiveEntries.Load())

					perShardPrevTotalGets[i] = total
					perShardPrevTotalPuts[i] = wc.stats[i].TotalPuts.Load()

					getP25, getP50, getP99 := wc.stats[i].LatencyTracker.GetLatencyPercentiles()
					putP25, putP50, putP99 := wc.stats[i].LatencyTracker.PutLatencyPercentiles()

					shardGets := wc.stats[i].TotalGets.Load()
					shardPuts := wc.stats[i].TotalPuts.Load()
					shardHits := wc.stats[i].Hits.Load()

					// Calculate per-shard throughput
					rThroughput := float64(shardGets) / sleepDuration.Seconds()
					wThroughput := float64(shardPuts) / sleepDuration.Seconds()

					// Calculate per-shard hit rate
					shardHitRate := float64(0)
					if shardGets > 0 {
						shardHitRate = float64(shardHits) / float64(shardGets)
					}

					wc.metricsCollector.RecordRP25(i, getP25)
					wc.metricsCollector.RecordRP50(i, getP50)
					wc.metricsCollector.RecordRP99(i, getP99)
					wc.metricsCollector.RecordWP25(i, putP25)
					wc.metricsCollector.RecordWP50(i, putP50)
					wc.metricsCollector.RecordWP99(i, putP99)
					wc.metricsCollector.RecordRThroughput(i, rThroughput)
					wc.metricsCollector.RecordWThroughput(i, wThroughput)
					wc.metricsCollector.RecordHitRate(i, shardHitRate)
					wc.metricsCollector.RecordActiveEntries(i, activeEntries)

				}

				log.Info().Msgf("GridSearchActive: %v", wc.predictor.GridSearchEstimator.IsGridSearchActive())
			}
		}()
	}
	return wc, nil
}

func (wc *WrapCache) PutLL(key string, value []byte, exptimeInMinutes uint16) error {

	h32 := wc.Hash(key)
	shardIdx := h32 % uint32(len(wc.shards))
	start := time.Now()

	result := filecache.ErrorPool.Get().(chan error)

	wc.shards[shardIdx].WriteCh <- &filecache.WriteRequestV2{
		Key:              key,
		Value:            value,
		ExptimeInMinutes: exptimeInMinutes,
		Result:           result,
	}

	if h32%100 < 10 {
		wc.stats[shardIdx].ShardWiseActiveEntries.Store(uint64(wc.shards[shardIdx].GetRingBufferActiveEntries()))
	}

	op := <-result
	filecache.ErrorPool.Put(result)
	wc.stats[shardIdx].TotalPuts.Add(1)
	wc.stats[shardIdx].LatencyTracker.RecordPut(time.Since(start))
	return op
}

func (wc *WrapCache) GetLL(key string) ([]byte, bool, bool) {
	h32 := wc.Hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	start := time.Now()

	found, value, _, expired, needsSlowPath := wc.shards[shardIdx].GetFastPath(key)

	if !needsSlowPath {
		if found && !expired {
			wc.stats[shardIdx].Hits.Add(1)
		} else if expired {
			wc.stats[shardIdx].Expired.Add(1)
		}

		wc.stats[shardIdx].TotalGets.Add(1)
		wc.stats[shardIdx].LatencyTracker.RecordGet(time.Since(start))
		return value, found, expired
	}

	result := filecache.ReadResultPool.Get().(chan filecache.ReadResultV2)

	req := filecache.ReadRequestPool.Get().(*filecache.ReadRequestV2)
	req.Key = key
	req.Result = result

	wc.shards[shardIdx].ReadCh <- req
	op := <-result

	filecache.ReadResultPool.Put(result)
	filecache.ReadRequestPool.Put(req)

	if op.Found && !op.Expired {
		wc.stats[shardIdx].Hits.Add(1)
	}
	if op.Expired {
		wc.stats[shardIdx].Expired.Add(1)
	}
	wc.stats[shardIdx].LatencyTracker.RecordGet(time.Since(start))
	wc.stats[shardIdx].TotalGets.Add(1)

	return op.Data, op.Found, op.Expired
}

func (wc *WrapCache) Put(key string, value []byte, exptimeInMinutes uint16) error {

	h32 := wc.Hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	start := time.Now()
	defer func() {
		wc.stats[shardIdx].LatencyTracker.RecordPut(time.Since(start))
	}()

	wc.shardLocks[shardIdx].Lock()
	defer wc.shardLocks[shardIdx].Unlock()
	wc.putLocked(shardIdx, h32, key, value, exptimeInMinutes)
	return nil
}

func (wc *WrapCache) putLocked(shardIdx uint32, h32 uint32, key string, value []byte, exptimeInMinutes uint16) {
	wc.shards[shardIdx].Put(key, value, exptimeInMinutes)
	wc.stats[shardIdx].TotalPuts.Add(1)
	if h32%100 < 10 {
		wc.stats[shardIdx].ShardWiseActiveEntries.Store(uint64(wc.shards[shardIdx].GetRingBufferActiveEntries()))
	}
}

func (wc *WrapCache) Get(key string) ([]byte, bool, bool) {
	h32 := wc.Hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	start := time.Now()
	defer func() {
		wc.stats[shardIdx].LatencyTracker.RecordGet(time.Since(start))
	}()

	var keyFound bool
	var val []byte
	var remainingTTL uint16
	var expired bool
	var shouldReWrite bool
	if wc.shards[shardIdx].BatchReader != nil {
		reqChan := make(chan filecache.ReadResultV2, 1)
		wc.shards[shardIdx].BatchReader.Requests <- &filecache.ReadRequestV2{
			Key:    key,
			Result: reqChan,
		}
		result := <-reqChan

		keyFound, val, remainingTTL, expired, shouldReWrite = result.Found, result.Data, result.TTL, result.Expired, result.ShouldRewrite
	} else {
		wc.shardLocks[shardIdx].RLock()
		defer wc.shardLocks[shardIdx].RUnlock()
		keyFound, val, remainingTTL, expired, shouldReWrite = wc.shards[shardIdx].Get(key)
	}

	if keyFound && !expired {
		wc.stats[shardIdx].Hits.Add(1)
	}
	if expired {
		wc.stats[shardIdx].Expired.Add(1)
	}
	wc.stats[shardIdx].TotalGets.Add(1)
	if false && shouldReWrite {
		wc.stats[shardIdx].ReWrites.Add(1)
		wc.putLocked(shardIdx, h32, key, val, remainingTTL)
	}
	wc.predictor.Observe(float64(wc.stats[shardIdx].Hits.Load()) / float64(wc.stats[shardIdx].TotalGets.Load()))
	return val, keyFound, expired
}

func (wc *WrapCache) Hash(key string) uint32 {
	return uint32(xxhash.Sum64String(key) ^ Seed)
}

func (wc *WrapCache) GetShardCache(shardIdx int) *filecache.ShardCache {
	return wc.shards[shardIdx]
}
