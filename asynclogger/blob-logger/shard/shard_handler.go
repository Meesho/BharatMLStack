package shard

import (
	"math/rand/v2"
	"runtime"
	"time"
)

type SwapShard struct {
	shards    []*ShardV2
	active    uint64
	semaphore chan int
}

func NewSwapShard(shards []*ShardV2) *SwapShard {
	return &SwapShard{
		shards:    shards,
		active:    0,
		semaphore: make(chan int, 1),
	}
}

func (s *SwapShard) Swap() {
	s.shards[s.active].readyForFlush = true
	s.active = (s.active + 1) % uint64(len(s.shards))
}

func (s *SwapShard) GetActive() *ShardV2 {
	return s.shards[s.active]
}

func (s *SwapShard) GetInactive() *ShardV2 {
	return s.shards[(s.active+1)%uint64(len(s.shards))]
}

type ShardHandler struct {
	swapShards map[uint32]*SwapShard
}

func NewShardHandler(shardCount int, capacity int) *ShardHandler {
	swapShards := make(map[uint32]*SwapShard)
	j := 0
	for i := 0; i < shardCount; i += 2 {
		shardA := NewShardV2(capacity, uint32(i))
		shardB := NewShardV2(capacity, uint32(i+1))
		swapShards[uint32(j)] = NewSwapShard([]*ShardV2{shardA, shardB})
		j++
	}
	return &ShardHandler{
		swapShards: swapShards,
	}
}

func (s *ShardHandler) Write(p []byte) bool {
	randomNumber := rand.IntN(10000)
	swapShardId := uint32(randomNumber % len(s.swapShards))
	swapShard := s.swapShards[swapShardId]
	_, id, full, _ := swapShard.GetActive().Write(p)
	if !full {
		return true
	}
	swapShard.semaphore <- 1
	currShard := swapShard.GetActive()
	if id == currShard.id {
		s.flush(swapShardId, id)
		swapShard.Swap()
		_, _, full, _ := swapShard.GetActive().Write(p)
		<-swapShard.semaphore
		return !full
	} else {
		<-swapShard.semaphore
		_, _, full, _ := currShard.Write(p)
		return !full
	}
}

func (s *ShardHandler) flush(swapShardId, shardId uint32) {
	swapShard := s.swapShards[swapShardId]
	var flushShard *ShardV2
	if shardId == swapShard.GetActive().id {
		flushShard = swapShard.GetActive()
	} else {
		flushShard = swapShard.GetInactive()
	}

	go func() {
		for flushShard.inflight.Load() != 0 {
			runtime.Gosched()
		}
		//TODO: Do actual flush
		time.Sleep(10 * time.Millisecond)
		flushShard.Reset()
	}()
}
