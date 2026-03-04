package internal

import (
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

type Badger struct {
	cache *badger.DB
}

func NewBadger(config WrapCacheConfig, logStats bool) (*Badger, error) {
	options := badger.DefaultOptions("/mnt/disks/nvme")
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
	}

	return bc, nil
}

func (b *Badger) Put(key string, value []byte, exptimeInMinutes uint16) error {

	err := b.cache.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), value).WithTTL(time.Duration(exptimeInMinutes) * time.Minute)
		err := txn.SetEntry(entry)
		return err
	})
	return err
}

func (b *Badger) Get(key string) ([]byte, bool, bool) {

	val := make([]byte, 0)
	err := b.cache.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(val)

		return err
	})
	return val, err != badger.ErrKeyNotFound, false
}

func (b *Badger) Close() error {
	return b.cache.Close()
}
