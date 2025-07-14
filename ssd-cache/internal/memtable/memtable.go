package memtable

import (
	"fmt"
	"os"
	"syscall"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/allocator"
	"github.com/rs/zerolog/log"
)

type Memtable struct {
	file          *os.File
	writeFD       int
	fileOffset    int64
	page          *allocator.Page
	capacity      int32
	size          int32
	readyForFlush bool
	flushCount    int64
	allocator     *allocator.AlignedPageAllocator
	Next          *Memtable
	Id            int64
}

func NewMemtable(writeFD int, fileOffset int64, capacity int32, allocator *allocator.AlignedPageAllocator, idx int64) *Memtable {
	page, _ := allocator.Get()
	return &Memtable{
		writeFD:       writeFD,
		fileOffset:    fileOffset,
		page:          page,
		capacity:      capacity,
		size:          0,
		readyForFlush: false,
		flushCount:    0,
		allocator:     allocator,
		Id:            idx,
	}
}

func (m *Memtable) Get(offset, length int32) []byte {
	return m.page.Buf[offset : offset+length]
}

func (m *Memtable) Put(buf []byte) (int32, int32) {
	offset := m.size
	if offset+int32(len(buf)) > m.capacity {
		m.readyForFlush = true
		return -1, -1
	}
	copy(m.page.Buf[offset:], buf)
	m.size += int32(len(buf))
	return offset, int32(len(buf))
}

func (m *Memtable) FlushV2() error {
	if !m.readyForFlush {
		return fmt.Errorf("memtable not ready for flush")
	}
	//log.Info().Msgf("Flushing memtable, memId:%d fileDescriptor: %d, fileOffset: %d, capacity: %d, flushCount: %d, afterFlushFileOffset: %d", m.Id, m.writeFD, m.fileOffset, m.capacity, m.flushCount, m.capacity+m.fileOffset)
	m.readyForFlush = false
	n, err := syscall.Pwrite(m.writeFD, m.page.Buf, m.fileOffset)
	if err != nil {
		log.Error().Msgf("Failed to flush, memId:%d, fileDescriptor: %d, fileOffset: %d, capacity: %d, flushCount: %d, err: %v", m.Id, m.writeFD, m.fileOffset, m.capacity, m.flushCount, err)
		return err
	}
	if n != int(m.capacity) {
		return fmt.Errorf("write failed")
	}
	m.fileOffset += int64(m.capacity)
	m.size = 0
	m.flushCount++
	return nil
}

func (m *Memtable) Discard() error {
	m.allocator.Put(m.page)
	return nil
}
