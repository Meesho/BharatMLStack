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

			for {
				time.Sleep(sleepDuration)

				for i := 0; i < config.NumShards; i++ {

					getP25, getP50, getP99 := wc.stats[i].LatencyTracker.GetLatencyPercentiles()
					putP25, putP50, putP99 := wc.stats[i].LatencyTracker.PutLatencyPercentiles()

					shardGets := wc.stats[i].TotalGets.Load()
					shardPuts := wc.stats[i].TotalPuts.Load()
					shardHits := wc.stats[i].Hits.Load()
					shardExpired := wc.stats[i].Expired.Load()
					shardReWrites := wc.stats[i].ReWrites.Load()
					shardActiveEntries := wc.stats[i].ShardWiseActiveEntries.Load()

					wc.metricsCollector.RecordRP25(i, getP25)
					wc.metricsCollector.RecordRP50(i, getP50)
					wc.metricsCollector.RecordRP99(i, getP99)
					wc.metricsCollector.RecordWP25(i, putP25)
					wc.metricsCollector.RecordWP50(i, putP50)
					wc.metricsCollector.RecordWP99(i, putP99)

					wc.metricsCollector.RecordActiveEntries(i, int64(shardActiveEntries))
					wc.metricsCollector.RecordExpiredEntries(i, int64(shardExpired))
					wc.metricsCollector.RecordRewrites(i, int64(shardReWrites))
					wc.metricsCollector.RecordGets(i, int64(shardGets))
					wc.metricsCollector.RecordPuts(i, int64(shardPuts))
					wc.metricsCollector.RecordHits(i, int64(shardHits))

					//shard level index and rb data - actually send to metrics collector!
					wc.metricsCollector.RecordKeyNotFoundCount(i, wc.shards[i].Stats.KeyNotFoundCount.Load())
					wc.metricsCollector.RecordKeyExpiredCount(i, wc.shards[i].Stats.KeyExpiredCount.Load())
					wc.metricsCollector.RecordBadDataCount(i, wc.shards[i].Stats.BadDataCount.Load())
					wc.metricsCollector.RecordBadLengthCount(i, wc.shards[i].Stats.BadLengthCount.Load())
					wc.metricsCollector.RecordBadCR32Count(i, wc.shards[i].Stats.BadCR32Count.Load())
					wc.metricsCollector.RecordBadKeyCount(i, wc.shards[i].Stats.BadKeyCount.Load())
					wc.metricsCollector.RecordDeletedKeyCount(i, wc.shards[i].Stats.DeletedKeyCount.Load())

					//wrapAppendFilt stats
					wc.metricsCollector.RecordWriteCount(i, wc.shards[i].GetFileStat().WriteCount)
					wc.metricsCollector.RecordPunchHoleCount(i, wc.shards[i].GetFileStat().PunchHoleCount)

				}

				log.Error().Msgf("GridSearchActive: %v", wc.predictor.GridSearchEstimator.IsGridSearchActive())
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
	metrics.Timing(metrics.KEY_WRITE_LATENCY_STATSD, time.Since(start), metrics.BuildTag(metrics.NewTag(metrics.TAG_SHARD_IDX, strconv.Itoa(int(shardIdx)))))
	return op
}

func (wc *WrapCache) GetLL(key string) ([]byte, bool, bool) {
	h32 := wc.Hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	start := time.Now()

	// found, value, _, expired, needsSlowPath := wc.shards[shardIdx].GetFastPath(key)

	// if !needsSlowPath {
	// 	if found && !expired {
	// 		wc.stats[shardIdx].Hits.Add(1)
	// 	} else if expired {
	// 		wc.stats[shardIdx].Expired.Add(1)
	// 	}

	// 	wc.stats[shardIdx].TotalGets.Add(1)
	// 	wc.stats[shardIdx].LatencyTracker.RecordGet(time.Since(start))
	// 	return value, found, expired
	// }

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
	metrics.Timing(metrics.KEY_READ_LATENCY_STATSD, time.Since(start), metrics.BuildTag(metrics.NewTag(metrics.TAG_SHARD_IDX, strconv.Itoa(int(shardIdx)))))
	wc.stats[shardIdx].TotalGets.Add(1)

	return op.Data, op.Found, op.Expired
}

func (wc *WrapCache) Put(key string, value []byte, exptimeInMinutes uint16) error {

	h32 := wc.Hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	start := time.Now()
	defer func() {
		wc.stats[shardIdx].LatencyTracker.RecordPut(time.Since(start))
		metrics.Timing(metrics.KEY_WRITE_LATENCY_STATSD, time.Since(start), metrics.BuildTag(metrics.NewTag(metrics.TAG_SHARD_IDX, strconv.Itoa(int(shardIdx)))))
	}()

	start = time.Now()
	wc.shardLocks[shardIdx].Lock()
	metrics.Timing(metrics.LATENCY_WLOCK, time.Since(start), metrics.BuildTag(metrics.NewTag(metrics.TAG_SHARD_IDX, strconv.Itoa(int(shardIdx)))))
	defer wc.shardLocks[shardIdx].Unlock()

	err := wc.shards[shardIdx].Put(key, value, exptimeInMinutes)
	if err != nil {
		log.Error().Err(err).Msgf("Put failed for key: %s", key)
		return fmt.Errorf("put failed for key: %s", key)
	}
	wc.stats[shardIdx].TotalPuts.Add(1)
	if h32%100 < 10 {
		wc.stats[shardIdx].ShardWiseActiveEntries.Store(uint64(wc.shards[shardIdx].GetRingBufferActiveEntries()))
	}

	return nil
}

func (wc *WrapCache) Get(key string) ([]byte, bool, bool) {
	h32 := wc.Hash(key)
	shardIdx := h32 % uint32(len(wc.shards))

	start := time.Now()
	defer func() {
		wc.stats[shardIdx].LatencyTracker.RecordGet(time.Since(start))
		metrics.Timing(metrics.KEY_READ_LATENCY_STATSD, time.Since(start), metrics.BuildTag(metrics.NewTag(metrics.TAG_SHARD_IDX, strconv.Itoa(int(shardIdx)))))
	}()

	var keyFound bool
	var val []byte
	var valCopy []byte
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
		if shouldReWrite {
			valCopy = make([]byte, len(val))
			copy(valCopy, val)
		}
	} else {

		func(key string, shardIdx uint32) {
			start := time.Now()
			wc.shardLocks[shardIdx].RLock()
			metrics.Timing(metrics.LATENCY_RLOCK, time.Since(start), metrics.BuildTag(metrics.NewTag(metrics.TAG_SHARD_IDX, strconv.Itoa(int(shardIdx)))))
			defer wc.shardLocks[shardIdx].RUnlock()
			keyFound, val, remainingTTL, expired, shouldReWrite = wc.shards[shardIdx].Get(key)

			if shouldReWrite {
				//copy val into a safe variable because we are unlocking the shard
				// at the end of anon function execution
				valCopy = make([]byte, len(val))
				copy(valCopy, val)
				val = valCopy
			}
		}(key, shardIdx)

	}

	if keyFound && !expired {
		wc.stats[shardIdx].Hits.Add(1)
	}
	if expired {
		wc.stats[shardIdx].Expired.Add(1)
	}
	wc.stats[shardIdx].TotalGets.Add(1)
	if shouldReWrite {
		wc.stats[shardIdx].ReWrites.Add(1)
		wc.Put(key, valCopy, remainingTTL)
	}

	if time.Since(wc.stats[shardIdx].timeStarted) > 10*time.Second {
		//observing hit rate every call can be avoided because average remains the same
		hitRate := float64(wc.stats[shardIdx].Hits.Load()-wc.stats[shardIdx].PrevHits.Load()) / float64(wc.stats[shardIdx].TotalGets.Load()-wc.stats[shardIdx].PrevTotalGets.Load())
		wc.predictor.Observe(hitRate)

		wc.stats[shardIdx].timeStarted = time.Now()
		wc.stats[shardIdx].PrevHits.Store(wc.stats[shardIdx].Hits.Load())
		wc.stats[shardIdx].PrevTotalGets.Store(wc.stats[shardIdx].TotalGets.Load())
	}
	return val, keyFound, expired
}

func (wc *WrapCache) Hash(key string) uint32 {
	return uint32(xxhash.Sum64String(key) ^ Seed)
}

func (wc *WrapCache) GetShardCache(shardIdx int) *filecache.ShardCache {
	return wc.shards[shardIdx]
}
