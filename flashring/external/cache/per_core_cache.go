package internal

import (
	"fmt"
	"runtime"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/external/maths"
	filecache "github.com/Meesho/BharatMLStack/flashring/external/shard"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

// PerCoreWrapCache is a variant of WrapCache that dedicates one OS thread per shard
// and pins that thread to a specific CPU core for the lifetime of the process.
// All operations for a shard are serialized through a pinned worker goroutine.
type Stats struct {
	Hits                   uint64
	TotalGets              uint64
	TotalPuts              uint64
	ReWrites               uint64
	Expired                uint64
	ShardWiseActiveEntries uint64
}

type PerCoreWrapCache struct {
	shards       []*filecache.ShardCache
	predictor    *maths.Predictor
	stats        []*Stats
	workerInputs []shardChannels
	stopChans    []chan struct{}
}

type shardChannels struct {
	putCh chan putRequest
	getCh chan getRequest
}

type putRequest struct {
	key     string
	value   []byte
	exptime uint64
}

type getRequest struct {
	key  string
	resp chan getResponse
}

type getResponse struct {
	value    []byte
	keyFound bool
	expired  bool
}

var (
	ErrNumShardsExceedsCPUs = fmt.Errorf("num shards must be <= number of CPUs - 1 for PerCoreWrapCache")
)

// NewPerCoreWrapCache constructs a PerCoreWrapCache with one pinned worker per shard.
func NewPerCoreWrapCache(config WrapCacheConfig, mountPoint string, logStats bool) (*PerCoreWrapCache, error) {
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

	// // Each shard must map to a dedicated CPU core, leaving one CPU for general runtime/OS.
	// if config.NumShards > runtime.NumCPU()-1 {
	// 	return nil, ErrNumShardsExceedsCPUs
	// }

	weights := []maths.WeightTuple{
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
			DeleteAmortizedStep: 1000,
			MaxFileSize:         int64(config.FileSize),
			BlockSize:           BLOCK_SIZE,
			Directory:           mountPoint,
			Predictor:           predictor,
		})
	}

	// Initialize stats including per-shard active entries counters.
	stats := make([]*Stats, config.NumShards)
	for i := 0; i < config.NumShards; i++ {
		stats[i] = &Stats{}
		stats[i].ShardWiseActiveEntries = 0
	}

	pc := &PerCoreWrapCache{
		shards:       shards,
		predictor:    predictor,
		stats:        stats,
		workerInputs: make([]shardChannels, config.NumShards),
		stopChans:    make([]chan struct{}, config.NumShards),
	}

	// Create channels and start one pinned worker per shard.
	const channelBufferSize = 4096
	for shardIdx := 0; shardIdx < config.NumShards; shardIdx++ {
		pc.workerInputs[shardIdx] = shardChannels{
			putCh: make(chan putRequest, channelBufferSize),
			getCh: make(chan getRequest, channelBufferSize),
		}
		pc.stopChans[shardIdx] = make(chan struct{})
		cpuID := shardIdx // one-to-one mapping using lower CPU indices when NumShards <= NumCPU-1
		go pc.runPinnedWorker(shardIdx, cpuID)
	}

	if logStats {
		go func() {
			sleepDuration := 5 * time.Second
			perShardPrevTotalGets := make([]uint64, config.NumShards)
			perShardPrevTotalPuts := make([]uint64, config.NumShards)
			for {
				time.Sleep(sleepDuration)
				for i := 0; i < config.NumShards; i++ {
					log.Info().Msgf("Shard %d has %d active entries", i, pc.stats[i].ShardWiseActiveEntries)
					total := pc.stats[i].TotalGets
					hits := pc.stats[i].Hits
					hitRate := float64(0)
					if total > 0 {
						hitRate = float64(hits) / float64(total)
					}
					log.Info().Msgf("Shard %d HitRate: %v", i, hitRate)
					log.Info().Msgf("Shard %d ReWrites: %v", i, pc.stats[i].ReWrites)
					log.Info().Msgf("Shard %d Expired: %v", i, pc.stats[i].Expired)
					log.Info().Msgf("Shard %d Total: %v", i, total)
					log.Info().Msgf("Gets/sec: %v", float64(total-perShardPrevTotalGets[i])/float64(sleepDuration.Seconds()))
					log.Info().Msgf("Puts/sec: %v", float64(pc.stats[i].TotalPuts-perShardPrevTotalPuts[i])/float64(sleepDuration.Seconds()))
					perShardPrevTotalGets[i] = total
					perShardPrevTotalPuts[i] = pc.stats[i].TotalPuts
				}
				log.Info().Msgf("GridSearchActive: %v", pc.predictor.GridSearchEstimator.IsGridSearchActive())
			}
		}()
	}

	return pc, nil
}

// Put routes the write to the appropriate shard worker.
func (pc *PerCoreWrapCache) Put(key string, value []byte, exptime uint64) error {
	h32 := hash(key)
	shardIdx := h32 % uint32(len(pc.shards))
	pc.workerInputs[shardIdx].putCh <- putRequest{key: key, value: value, exptime: exptime}
	// No error path from underlying shard cache currently; return nil for API parity.
	return nil
}

// Get synchronously queries the appropriate shard worker and returns the value.
func (pc *PerCoreWrapCache) Get(key string) ([]byte, bool, bool) {
	h32 := hash(key)
	shardIdx := h32 % uint32(len(pc.shards))
	respCh := make(chan getResponse, 1)
	pc.workerInputs[shardIdx].getCh <- getRequest{key: key, resp: respCh}
	resp := <-respCh
	return resp.value, resp.keyFound, resp.expired
}

// runPinnedWorker executes all operations for a given shard on a dedicated OS thread
// pinned to the provided cpuID.
func (pc *PerCoreWrapCache) runPinnedWorker(shardIdx int, cpuID int) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var cpuset unix.CPUSet
	cpuset.Zero()
	cpuset.Set(cpuID)
	if err := unix.SchedSetaffinity(0, &cpuset); err != nil {
		log.Error().Err(err).Msgf("failed to set CPU affinity for shard %d to CPU %d", shardIdx, cpuID)
		// Continue without affinity rather than crashing.
	}

	sc := pc.shards[shardIdx]
	ch := pc.workerInputs[shardIdx]
	stop := pc.stopChans[shardIdx]

	for {
		select {
		case req := <-ch.putCh:
			// Process write on this shard's dedicated worker
			sc.Put(req.key, req.value, req.exptime)
			if hash(req.key)%100 < 10 {
				pc.stats[shardIdx].ShardWiseActiveEntries = uint64(sc.GetRingBufferActiveEntries())
			}
			pc.stats[shardIdx].TotalPuts++

		case req := <-ch.getCh:
			val, exptime, keyFound, expired, shouldReWrite := sc.Get(req.key)
			if keyFound && !expired {
				pc.stats[shardIdx].Hits++
			}
			if expired {
				pc.stats[shardIdx].Expired++
			}
			pc.stats[shardIdx].TotalGets++
			if shouldReWrite {
				pc.stats[shardIdx].ReWrites++
				// Re-write directly via the same shard cache to avoid deadlocks
				sc.Put(req.key, val, exptime)
			}
			if hash(req.key)%100 < 10 {
				total := pc.stats[shardIdx].TotalGets
				hits := pc.stats[shardIdx].Hits
				if total > 0 {
					pc.predictor.Observe(float64(hits) / float64(total))
				}
			}
			req.resp <- getResponse{value: val, keyFound: keyFound, expired: expired}

		case <-stop:
			return
		}
	}
}
