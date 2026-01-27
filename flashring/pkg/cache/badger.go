package internal

import (
	"sync/atomic"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog/log"
)

type Badger struct {
	cache *badger.DB
	stats *CacheStats
}

func NewBadger(config WrapCacheConfig, logStats bool) (*Badger, error) {
	options := badger.DefaultOptions(config.MountPoint)
	options.MetricsEnabled = false

	// 1. PRIMARY CACHE (1GB)
	// This caches the data blocks themselves.
	options.BlockCacheSize = 1024 << 20

	// 2. INDEX CACHE (512MB)
	// This keeps the keys and the structure of the LSM tree in RAM.
	// This is the most critical setting for read latency.
	options.IndexCacheSize = 512 << 20

	// 3. WRITE BUFFERS (Memtables)
	// We use 3 tables of 64MB each. This allows Badger to handle
	// write spikes without blocking. (~192MB total)
	options.NumMemtables = 40
	options.MemTableSize = 1024 << 20

	options.ValueThreshold = 1024
	options.SyncWrites = false

	cache, err := badger.Open(options)
	if err != nil {
		return nil, err
	}
	bc := &Badger{
		cache: cache,
		stats: &CacheStats{
			Hits:                   atomic.Uint64{},
			TotalGets:              atomic.Uint64{},
			TotalPuts:              atomic.Uint64{},
			ReWrites:               atomic.Uint64{},
			Expired:                atomic.Uint64{},
			ShardWiseActiveEntries: atomic.Uint64{},
		},
	}

	if logStats {
		go func() {
			sleepDuration := 10 * time.Second
			var prevTotalGets, prevTotalPuts uint64
			for {
				time.Sleep(sleepDuration)

				totalGets := bc.stats.TotalGets.Load()
				totalPuts := bc.stats.TotalPuts.Load()
				getsPerSec := float64(totalGets-prevTotalGets) / sleepDuration.Seconds()
				putsPerSec := float64(totalPuts-prevTotalPuts) / sleepDuration.Seconds()

				log.Info().Msgf("Shard %d HitRate: %v", 0, cache.BlockCacheMetrics().Hits())
				log.Info().Msgf("Shard %d Expired: %v", 0, cache.BlockCacheMetrics().Misses())
				log.Info().Msgf("Shard %d Total: %v", 0, cache.BlockCacheMetrics().KeysEvicted())
				log.Info().Msgf("Gets/sec: %v", getsPerSec)
				log.Info().Msgf("Puts/sec: %v", putsPerSec)

				log.Info().Msgf("Get Count: %v", totalGets)
				log.Info().Msgf("Put Count: %v", totalPuts)

				prevTotalGets = totalGets
				prevTotalPuts = totalPuts
			}
		}()
	}

	return bc, nil
}

func (b *Badger) Put(key string, value []byte, exptimeInMinutes uint16) error {
	b.stats.TotalPuts.Add(1)
	err := b.cache.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), value).WithTTL(time.Duration(exptimeInMinutes) * time.Minute)
		err := txn.SetEntry(entry)
		return err
	})
	return err
}

func (b *Badger) Get(key string) ([]byte, bool, bool) {
	b.stats.TotalGets.Add(1)

	val := make([]byte, 0)
	err := b.cache.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(val)

		if err != nil {
			b.stats.Hits.Add(1)
		}

		return err
	})
	return val, err != badger.ErrKeyNotFound, false
}

func (b *Badger) Close() error {
	return b.cache.Close()
}
