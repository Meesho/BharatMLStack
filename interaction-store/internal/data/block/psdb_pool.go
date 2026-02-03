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
				psdb := &PermanentStorageDataBlock{
					Buf: make([]byte, PSDBLayout1HeaderLength),
				}
				psdb.Builder = &PermanentStorageDataBlockBuilder{psdb: psdb}
				return psdb
			},
		},
	}
}

func (p *PSDBPool) Get() *PermanentStorageDataBlock {
	return p.pool.Get().(*PermanentStorageDataBlock)
}

func (p *PSDBPool) Put(b *PermanentStorageDataBlock) {
	b.Clear()
	p.pool.Put(b)
}
