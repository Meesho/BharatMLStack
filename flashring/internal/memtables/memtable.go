package memtables

import (
	"errors"
	"sync/atomic"

	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
)

var (
	ErrCapacityNotAligned         = errors.New("capacity must be aligned to block size")
	ErrPageNotProvided            = errors.New("page must be provided")
	ErrFileNotProvided            = errors.New("file must be provided")
	ErrPageBufferCapacityMismatch = errors.New("page buffer must be provided and must be of size capacity")
	ErrOffsetOutOfBounds          = errors.New("offset out of bounds")
	ErrMemtableNotReadyForFlush   = errors.New("memtable not ready for flush")
)

type Memtable struct {
	currentOffset int64  // atomic; must be first for 64-bit alignment on 32-bit platforms
	flushStarted  uint32 // atomic; 0 = not started, 1 = flush triggered
	Id            uint32
	capacity      int
	file          *fs.WrapAppendFile
	page          *fs.AlignedPage
	next          *Memtable
	prev          *Memtable
	ShardIdx      uint32
}

type MemtableConfig struct {
	capacity int
	id       uint32
	page     *fs.AlignedPage
	file     *fs.WrapAppendFile
	shardIdx uint32
}

func NewMemtable(config MemtableConfig) (*Memtable, error) {
	if config.capacity%fs.BLOCK_SIZE != 0 {
		return nil, ErrCapacityNotAligned
	}
	if config.page == nil {
		return nil, ErrPageNotProvided
	}
	if config.file == nil {
		return nil, ErrFileNotProvided
	}
	if config.page.Buf == nil || len(config.page.Buf) != config.capacity {
		return nil, ErrPageBufferCapacityMismatch
	}
	return &Memtable{
		Id:       config.id,
		ShardIdx: config.shardIdx,
		capacity: config.capacity,
		file:     config.file,
		page:     config.page,
	}, nil
}

func (m *Memtable) Get(offset int, length uint16) ([]byte, error) {
	if offset+int(length) > m.capacity {
		return nil, ErrOffsetOutOfBounds
	}
	return m.page.Buf[offset : offset+int(length)], nil
}

func (m *Memtable) Put(buf []byte) (offset int, length uint16, readyForFlush bool) {
	sz := int64(len(buf))
	newOffset := atomic.AddInt64(&m.currentOffset, sz)
	start := newOffset - sz

	if newOffset > int64(m.capacity) {
		if atomic.CompareAndSwapUint32(&m.flushStarted, 0, 1) {
			return -1, 0, true
		}
		return -1, 0, false
	}
	copy(m.page.Buf[start:start+sz], buf)
	return int(start), uint16(len(buf)), false
}

func (m *Memtable) GetBufForAppend(size uint16) (bbuf []byte, offset int, length uint16, readyForFlush bool) {
	sz := int64(size)
	newOffset := atomic.AddInt64(&m.currentOffset, sz)
	start := newOffset - sz

	if newOffset > int64(m.capacity) {
		if atomic.CompareAndSwapUint32(&m.flushStarted, 0, 1) {
			return nil, -1, 0, true
		}
		return nil, -1, 0, false
	}
	return m.page.Buf[start:newOffset], int(start), size, false
}

func (m *Memtable) GetBufForRead(offset int, length uint16) (bbuf []byte, exists bool) {
	if offset+int(length) > m.capacity {
		return nil, false
	}
	return m.page.Buf[offset : offset+int(length)], true
}

func (m *Memtable) Flush() (n int, fileOffset int64, err error) {
	if atomic.LoadUint32(&m.flushStarted) == 0 {
		return 0, 0, ErrMemtableNotReadyForFlush
	}

	chunkSize := fs.BLOCK_SIZE
	numChunks := len(m.page.Buf) / chunkSize
	if len(m.page.Buf)%chunkSize != 0 {
		numChunks++
	}

	// PwriteBatch submits all chunks in one io_uring_enter when WriteRing is
	// set, otherwise falls back to sequential pwrite internally.
	totalWritten, fileOffset, err := m.file.PwriteBatch(m.page.Buf, chunkSize)
	if err != nil {
		return 0, 0, err
	}

	atomic.StoreInt64(&m.currentOffset, 0)
	atomic.StoreUint32(&m.flushStarted, 0)
	return totalWritten, fileOffset, nil
}

func (m *Memtable) Discard() {
	m.file = nil
	m.page = nil
}
