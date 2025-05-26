package blocks

import (
	"sync"
)

var (
	pooledPSDB = newPSDBPool()
)

func GetPSDBPool() *PSDBPool {
	return pooledPSDB
}

type PSDBPool struct {
	pool sync.Pool
}

func newPSDBPool() *PSDBPool {
	return &PSDBPool{
		pool: sync.Pool{
			New: func() interface{} {
				psdb := &PermStorageDataBlock{}
				psdb.Builder = &PermStorageDataBlockBuilder{psdb: psdb}
				return psdb
			},
		},
	}
}

func (p *PSDBPool) Get() *PermStorageDataBlock {
	return p.pool.Get().(*PermStorageDataBlock)
}

func (p *PSDBPool) Put(b *PermStorageDataBlock) {
	b.Clear()
	p.pool.Put(b)
}
