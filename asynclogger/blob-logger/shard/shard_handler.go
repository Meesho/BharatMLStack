package shard

import (
	"math/rand/v2"
	"runtime"

	"github.com/Meesho/BharatMLStack/asynclogger/blob-logger/ssdio"
)

type SwapShard struct {
	shards    []*ShardV2
	active    uint64
	writer    *ssdio.SSDWriter
	semaphore chan int
}

func NewSwapShard(shards []*ShardV2, writer *ssdio.SSDWriter) *SwapShard {

	return &SwapShard{
		shards:    shards,
		active:    0,
		writer:    writer,
		semaphore: make(chan int, 1),
	}
}

func (s *SwapShard) Swap() {
	s.shards[s.active].readyForFlush.Store(1)
	s.active = (s.active + 1) % uint64(len(s.shards))
}
func (s *SwapShard) SwapBlocking() {
	s.shards[s.active].readyForFlush.Store(1)
	next := (s.active + 1) % uint64(len(s.shards))
	nextShard := s.shards[next]

	for nextShard.readyForFlush.Load() == 1 {
		runtime.Gosched()
	}

	s.active = next
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

func NewShardHandler(shardCount int, capacity int, fileSize int, dirPath string) *ShardHandler {
	swapShards := make(map[uint32]*SwapShard)
	j := 0
	for i := 0; i < shardCount; i += 2 {
		shardA := NewShardV2(capacity, uint32(i))
		shardB := NewShardV2(capacity, uint32(i+1))
		writer, err := ssdio.NewSSDWriter(fileSize, dirPath, j)
		if err != nil {
			panic(err)
		}
		swapShards[uint32(j)] = NewSwapShard([]*ShardV2{shardA, shardB}, writer)
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
		swapShard.SwapBlocking()
		_, _, full, _ := swapShard.GetActive().Write(p)
		<-swapShard.semaphore
		if !full {
			return true
		} else {
			return false
		}
	} else {
		<-swapShard.semaphore
		_, _, full, _ := currShard.Write(p)
		if !full {
			return true
		} else {
			return false
		}
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
			//fmt.Printf("waiting for inflight to be 0 %d %d\n", swapShardId, shardId)
			runtime.Gosched()
		}
		swapShard.writer.Write(flushShard.data)
		flushShard.Reset()
	}()
}
