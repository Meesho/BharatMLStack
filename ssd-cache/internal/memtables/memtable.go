package memtables

import (
	"errors"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/fs"
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
	file          *fs.RollingAppendFile
	page          *fs.AlignedPage
	readyForFlush bool
}

type MemtableConfig struct {
	capacity int
	id       uint32
	page     *fs.AlignedPage
	file     *fs.RollingAppendFile
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

func (m *Memtable) Flush() (n int, fileOffset int64, err error) {
	if !m.readyForFlush {
		return 0, 0, ErrMemtableNotReadyForFlush
	}
	fileOffset, err = m.file.Pwrite(m.page.Buf)
	if err != nil {
		return 0, 0, err
	}
	m.readyForFlush = false
	return len(m.page.Buf), fileOffset, nil
}

func (m *Memtable) Discard() {
	m.file = nil
	m.page = nil
}
