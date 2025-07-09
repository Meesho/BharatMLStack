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
	capacity      int64
	size          int64
	readyForFlush bool
	flushCount    int64
	allocator     *allocator.AlignedPageAllocator
	Next          *Memtable
	Id            int64
}

func NewMemtable(file *os.File, fileOffset int64, capacity int64) *Memtable {
	allocator := allocator.NewAlignedPageAllocator(allocator.AlignedPageAllocatorConfig{
		PageSizeAlignement: 4096,
		Multiplier:         int(capacity / 4096),
		MaxPages:           1000,
	})
	page, _ := allocator.Get()
	return &Memtable{
		file:          file,
		fileOffset:    fileOffset,
		page:          page,
		capacity:      capacity,
		size:          0,
		readyForFlush: false,
		flushCount:    0,
		allocator:     allocator,
	}
}

func NewMemtableV2(file *os.File, fileOffset int64, capacity int64, allocator *allocator.AlignedPageAllocator, idx int64) *Memtable {
	page, _ := allocator.Get()
	return &Memtable{
		file:          file,
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

func NewMemtableV3(writeFD int, fileOffset int64, capacity int64, allocator *allocator.AlignedPageAllocator, idx int64) *Memtable {
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

func (m *Memtable) Get(offset, length int64) []byte {
	return m.page.Buf[offset : offset+length]
}

func (m *Memtable) Put(buf []byte) (int64, int64) {
	offset := m.size
	if offset+int64(len(buf)) > m.capacity {
		m.readyForFlush = true
		return -1, -1
	}
	copy(m.page.Buf[offset:], buf)
	m.size += int64(len(buf))
	return offset, int64(len(buf))
}

func (m *Memtable) Flush() error {
	if !m.readyForFlush {
		return fmt.Errorf("memtable not ready for flush")
	}
	m.readyForFlush = false
	n, err := m.file.Write(m.page.Buf)
	if err != nil {
		return err
	}
	if n != int(m.capacity) {
		return fmt.Errorf("write failed")
	}
	m.fileOffset += m.capacity
	m.size = 0
	m.flushCount++
	return nil
}

func (m *Memtable) FlushV2() error {
	if !m.readyForFlush {
		return fmt.Errorf("memtable not ready for flush")
	}
	m.readyForFlush = false
	n, err := syscall.Pwrite(m.writeFD, m.page.Buf, m.fileOffset)
	if err != nil {
		log.Error().Msgf("Failed to flush, fileDescriptor: %d, fileOffset: %d, capacity: %d, flushCount: %d, err: %v", m.writeFD, m.fileOffset, m.capacity, m.flushCount, err)
		return err
	}
	if n != int(m.capacity) {
		return fmt.Errorf("write failed")
	}
	m.fileOffset += m.capacity
	m.size = 0
	m.flushCount++
	return nil
}

func (m *Memtable) Discard() error {
	m.allocator.Put(m.page)
	return nil
}
