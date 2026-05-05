package cache

import (
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

type Badger struct {
	cache *badger.DB
}

func NewBadger(config Config, dir string) (*Badger, error) {
	options := badger.DefaultOptions(dir)
	options.MetricsEnabled = false
	options.BlockCacheSize = 1024 << 20
	options.IndexCacheSize = 512 << 20
	options.NumMemtables = 40
	options.MemTableSize = 1024 << 20
	options.ValueThreshold = 1024
	options.SyncWrites = false

	db, err := badger.Open(options)
	if err != nil {
		return nil, err
	}
	return &Badger{cache: db}, nil
}

func (b *Badger) Put(key string, value []byte, ttl time.Duration) error {
	return b.cache.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), value).WithTTL(ttl)
		return txn.SetEntry(entry)
	})
}

func (b *Badger) Get(key string) ([]byte, bool, bool) {
	var val []byte
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
