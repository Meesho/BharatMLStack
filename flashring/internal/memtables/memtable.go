package memtables

import (
	"errors"

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
	Id            uint32
	capacity      int
	currentOffset int
	file          *fs.WrapAppendFile
	page          *fs.AlignedPage
	readyForFlush bool
	next          *Memtable
	prev          *Memtable
	ShardIdx      uint32

	// flushStartOffset is the byte offset within the page where real data
	// begins. On the first (staggered) memtable this equals the stagger
	// offset so Flush() skips the uninitialized region. Reset to 0 after
	// the first flush.
	flushStartOffset int
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
		Id:            config.id,
		ShardIdx:      config.shardIdx,
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

// Put copies buf into the memtable at the current write head. When the
// remaining capacity is too small, the write is rejected (returns
// offset=-1) and the memtable is marked readyForFlush so the manager
// rotates buffers — callers must retry against the new active memtable.
func (m *Memtable) Put(buf []byte) (offset int, length uint16, readyForFlush bool) {
	m.dropStaggerIfEmpty(len(buf))
	offset = m.currentOffset
	if offset+len(buf) > m.capacity {
		m.readyForFlush = true
		return -1, 0, true
	}
	copy(m.page.Buf[offset:], buf)
	m.currentOffset += len(buf)
	return offset, uint16(len(buf)), false
}

// GetBufForAppend reserves a size-byte slice from the memtable and
// hands the caller a writable view into the underlying mmap page —
// avoids the copy that Put performs. Same overflow contract: when the
// remaining capacity is too small, returns nil/offset=-1 and marks the
// memtable readyForFlush. The returned slice must be fully written
// before the next memtable rotation, since it is read directly from
// the page on subsequent Gets.
func (m *Memtable) GetBufForAppend(size uint16) (bbuf []byte, offset int, length uint16, readyForFlush bool) {
	m.dropStaggerIfEmpty(int(size))
	offset = m.currentOffset
	if offset+int(size) > m.capacity {
		m.readyForFlush = true
		return nil, -1, 0, true
	}
	bbuf = m.page.Buf[offset : offset+int(size)]
	m.currentOffset += int(size)
	return bbuf, offset, size, false
}

// Seal makes the memtable refuse further appends, so the next one rotates
// it out and flushes it as if it were full.
func (m *Memtable) Seal() {
	m.currentOffset = m.capacity
}

// dropStaggerIfEmpty rewinds a staggered memtable that holds no entries
// when the first entry does not fit in the room the stagger left. Flushing
// it empty instead would leave a memtable id with no index entries, which
// the delete manager cannot step over.
func (m *Memtable) dropStaggerIfEmpty(size int) {
	if m.flushStartOffset > 0 && m.currentOffset == m.flushStartOffset && m.currentOffset+size > m.capacity {
		m.currentOffset = 0
		m.flushStartOffset = 0
	}
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

	chunkSize := fs.BLOCK_SIZE

	// When the memtable has a stagger offset (first cycle only), skip the
	// uninitialized region: advance the file's write pointer past it, then
	// write only the real data. Total file advancement = flushStartOffset +
	// len(usedBuf) = capacity, preserving the memId*capacity layout.
	startOff := m.flushStartOffset
	if startOff > 0 {
		m.file.AdvanceWriteOffset(int64(startOff))
	}

	buf := m.page.Buf[startOff:]

	// PwriteBatch submits all chunks via io_uring.
	totalWritten, fileOffset, err := m.file.PwriteBatch(buf, chunkSize)

	// Reset on failure too: the manager recycles this memtable under a new
	// id either way, and PwriteBatch has already consumed its file slot.
	m.currentOffset = 0
	m.readyForFlush = false
	m.flushStartOffset = 0 // subsequent flushes write the full page
	if err != nil {
		return 0, 0, err
	}
	return startOff + totalWritten, fileOffset, nil
}

func (m *Memtable) Discard() {
	m.file = nil
	m.page = nil
}
