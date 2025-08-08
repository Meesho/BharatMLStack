package internal

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/filecache"
	"github.com/cespare/xxhash/v2"
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
	shards     []*filecache.FileCache
	shardLocks []sync.RWMutex
}

type WrapCacheConfig struct {
	NumShards    int
	KeysPerShard int
	FileSize     int64
	MemtableSize int32
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
	shards := make([]*filecache.FileCache, config.NumShards)
	for i := 0; i < config.NumShards; i++ {
		shards[i] = filecache.NewFileCache(filecache.FileCacheConfig{
			MemtableSize:        config.MemtableSize,
			Rounds:              ROUNDS,
			RbInitial:           config.KeysPerShard,
			RbMax:               config.KeysPerShard,
			DeleteAmortizedStep: 1000,
			MaxFileSize:         int64(config.FileSize),
			BlockSize:           BLOCK_SIZE,
			Directory:           mountPoint,
		})
	}
	shardLocks := make([]sync.RWMutex, config.NumShards)
	return &WrapCache{
		shards:     shards,
		shardLocks: shardLocks,
	}, nil
}

func (wc *WrapCache) Put(key string, value []byte, exptime uint64) error {
	shardIdx := hash(key) % uint32(len(wc.shards))
	wc.shardLocks[shardIdx].Lock()
	defer wc.shardLocks[shardIdx].Unlock()
	wc.shards[shardIdx].Put(key, value, exptime)
	return nil
}

func (wc *WrapCache) Get(key string) ([]byte, bool, bool) {
	shardIdx := hash(key) % uint32(len(wc.shards))
	wc.shardLocks[shardIdx].RLock()
	val, exptime, keyFound, expired, shouldReWrite := wc.shards[shardIdx].Get(key)
	wc.shardLocks[shardIdx].RUnlock()
	if shouldReWrite {
		wc.Put(key, val, exptime)
	}
	return val, keyFound, expired
}

func hash(key string) uint32 {
	nKey := key + Seed
	return uint32(xxhash.Sum64String(nKey))
}
