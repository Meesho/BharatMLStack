package shard

import "sync/atomic"

type Shard struct {
	data     []byte
	offset   atomic.Int32
	capacity int32
	id       uint32
}

func NewShard(capacity int, id uint32) *Shard {
	return &Shard{
		data:     make([]byte, capacity),
		offset:   atomic.Int32{},
		capacity: int32(capacity),
		id:       id,
	}
}

func (b *Shard) Write(p []byte) (int32, bool) {
	size := int32(len(p))
	if size == 0 {
		return 0, false
	}

	newOffset := b.offset.Add(size)

	if newOffset > b.capacity {
		return 0, true
	}

	copy(b.data[newOffset-size:newOffset], p)

	return size, false
}
