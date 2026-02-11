package memtables

import (
	"errors"
	"runtime"
	"strconv"

	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
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
	Id            uint32
	capacity      int
	currentOffset int
	file          *fs.WrapAppendFile
	page          *fs.AlignedPage
	readyForFlush bool
	next          *Memtable
	prev          *Memtable
}

type MemtableConfig struct {
	capacity int
	id       uint32
	page     *fs.AlignedPage
	file     *fs.WrapAppendFile
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
		Id:            config.id,
		capacity:      config.capacity,
		currentOffset: 0,
		file:          config.file,
		page:          config.page,
		readyForFlush: false,
	}, nil
}

func (m *Memtable) Get(offset int, length uint16) ([]byte, error) {
	if offset+int(length) > m.capacity {
		return nil, ErrOffsetOutOfBounds
	}
	return m.page.Buf[offset : offset+int(length)], nil
}

func (m *Memtable) Put(buf []byte) (offset int, length uint16, readyForFlush bool) {
	offset = m.currentOffset
	if offset+len(buf) > m.capacity {
		m.readyForFlush = true
		return -1, 0, true
	}
	copy(m.page.Buf[offset:], buf)
	m.currentOffset += len(buf)
	return offset, uint16(len(buf)), false
}

// Efforts to make zero copy
func (m *Memtable) GetBufForAppend(size uint16) (bbuf []byte, offset int, length uint16, readyForFlush bool) {
	offset = m.currentOffset
	if offset+int(size) > m.capacity {
		m.readyForFlush = true
		return nil, -1, 0, true
	}
	bbuf = m.page.Buf[offset : offset+int(size)]
	m.currentOffset += int(size)
	return bbuf, offset, size, false
}

func (m *Memtable) GetBufForRead(offset int, length uint16) (bbuf []byte, exists bool) {
	if offset+int(length) > m.capacity {
		return nil, false
	}
	return m.page.Buf[offset : offset+int(length)], true
}

func (m *Memtable) Flush() (n int, fileOffset int64, err error) {
	if !m.readyForFlush {
		return 0, 0, ErrMemtableNotReadyForFlush
	}

	chunkSize := 32 * fs.BLOCK_SIZE
	totalWritten := 0

	for totalWritten < len(m.page.Buf) {
		metrics.Count(metrics.KEY_MEMTABLE_FLUSH_COUNT, 1, []string{"memtable_id", strconv.Itoa(int(m.Id))})
		chunk := m.page.Buf[totalWritten : totalWritten+chunkSize]

		if err != nil {
			return 0, 0, err
		}
		totalWritten += chunkSize
		fileOffset, err = m.file.Pwrite(chunk)
		if err != nil {
			return 0, 0, err
		}

		runtime.Gosched()
	}
	m.currentOffset = 0
	m.readyForFlush = false
	return totalWritten, fileOffset, nil
}

func (m *Memtable) Discard() {
	m.file = nil
	m.page = nil
}
