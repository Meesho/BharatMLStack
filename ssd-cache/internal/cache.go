package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/allocator"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/index"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/memtable"
	"github.com/rs/zerolog/log"
)

const (
	// File flags for testing
	O_DIRECT   = 0x4000
	O_WRONLY   = syscall.O_WRONLY
	O_RDONLY   = syscall.O_RDONLY
	O_APPEND   = syscall.O_APPEND
	O_CREAT    = syscall.O_CREAT
	O_DSYNC    = syscall.O_DSYNC
	FILE_MODE  = 0644
	BLOCK_SIZE = 4096
)

type Cache struct {
	memtableManager    *memtable.MemtableManager
	writePageAllocator *allocator.AlignedPageAllocator
	readPageAllocator  *allocator.SlabAlignedPageAllocator
	index              *index.Index
	writeFD            int
	readFD             int
	writeFile          *os.File
	readFile           *os.File
}

func NewCache(memtableCapacity int64) *Cache {
	if memtableCapacity%BLOCK_SIZE != 0 {
		memtableCapacity = (memtableCapacity/BLOCK_SIZE + 1) * BLOCK_SIZE
	}
	writePageAllocator := allocator.NewAlignedPageAllocator(allocator.AlignedPageAllocatorConfig{
		PageSizeAlignement: BLOCK_SIZE,
		Multiplier:         int(memtableCapacity / BLOCK_SIZE),
		MaxPages:           1000,
	})

	readPageAllocator := allocator.NewSlabAlignedPageAllocator(allocator.SlabAlignedPageAllocatorConfig{
		PageSizeAlignement: BLOCK_SIZE,
		Multipliers:        []int{1, 2, 4, 8},
		MaxPages:           []int{1, 1, 1, 1},
	})

	filename := filepath.Join(".", "test_memtable.dat")

	writeFd, writeFile, err := createWriteFileDescriptor(filename)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create write file descriptor")
	}
	readFd, readFile, err := createReadFileDescriptor(filename)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create read file descriptor")
	}
	memtableManager := memtable.NewMemtableManagerV2(writeFd, memtableCapacity, writePageAllocator)
	index := index.NewIndex()

	return &Cache{
		memtableManager:    memtableManager,
		writePageAllocator: writePageAllocator,
		readPageAllocator:  readPageAllocator,
		index:              index,
		writeFD:            writeFd,
		writeFile:          writeFile,
		readFD:             readFd,
		readFile:           readFile,
	}
}

func (c *Cache) Put(key string, value []byte) {
	memtable, memtableId, fileOffset := c.memtableManager.GetMemtable()
	data := append([]byte(key), value...)
	offset, length := memtable.Put(data)
	if offset == -1 && length == -1 {
		c.memtableManager.Flush()
		memtable, memtableId, fileOffset = c.memtableManager.GetMemtable()
		offset, length = memtable.Put(data)
	}
	c.index.Put(key, offset, length, fileOffset, memtableId)
}

func (c *Cache) Get(key string) []byte {
	offset, length, fileOffset, id, ok := c.index.Get(key)
	if !ok {
		return nil
	}
	memtable := c.memtableManager.GetMemtableById(id)
	var data []byte
	if memtable != nil {
		data = memtable.Get(offset, length)
	} else {
		data = make([]byte, length)
		c.ReadFromDisk(fileOffset+offset, length, data)
	}
	gotKey := string(data[:len(key)])
	if gotKey != key {
		return nil
	}
	return data[len(key):]
}

func (c *Cache) Discard() {
	// TODO: implement cleanup for memtables
	syscall.Close(c.writeFD)
	syscall.Close(c.readFD)
	os.Remove(c.writeFile.Name())
	os.Remove(c.readFile.Name())
}

func (c *Cache) ReadFromDisk(fileOffset int64, length int64, buf []byte) error {
	alignedStartOffset := (fileOffset / BLOCK_SIZE) * BLOCK_SIZE
	endndOffset := fileOffset + length
	endAlignedOffset := ((endndOffset + BLOCK_SIZE - 1) / BLOCK_SIZE) * BLOCK_SIZE
	alignedReadSize := endAlignedOffset - alignedStartOffset

	page, crossBound := c.readPageAllocator.Get(int(alignedReadSize))

	if crossBound {
		log.Warn().Msg("Cache: Crossed bound")
	}
	log.Debug().Msgf("Read params: fileOffset: %d, length: %d, alignedStartOffset: %d, endAlignedOffset: %d, alignedReadSize: %d", fileOffset, length, alignedStartOffset, endAlignedOffset, alignedReadSize)
	n, err := syscall.Pread(c.readFD, page.Buf, alignedStartOffset)
	if err != nil {
		return err
	}
	if n < int(alignedReadSize) {
		return fmt.Errorf("read size mismatch: %d != %d", n, alignedReadSize)
	}
	start := fileOffset - alignedStartOffset
	copy(buf, page.Buf[start:start+length])
	log.Debug().Msgf("Read data: %s", string(buf))
	return nil
}

func createWriteFileDescriptor(filename string) (int, *os.File, error) {

	// Open file with DIRECT_IO, WRITE_ONLY, CREAT flags
	flags := O_DIRECT | O_WRONLY | O_CREAT | O_DSYNC
	fd, err := syscall.Open(filename, flags, FILE_MODE)
	if err != nil {
		// If DIRECT_IO is not supported, fall back to regular flags
		log.Warn().Msgf("DIRECT_IO not supported, falling back to regular flags: %v", err)
		flags = O_WRONLY | O_CREAT | O_DSYNC
		fd, err = syscall.Open(filename, flags, FILE_MODE)
		if err != nil {
			return 0, nil, err
		}
	}
	file := os.NewFile(uintptr(fd), filename)
	if file == nil {
		return 0, nil, fmt.Errorf("failed to create file from fd")
	}

	return fd, file, nil
}

func createReadFileDescriptor(filename string) (int, *os.File, error) {
	flags := O_DIRECT | O_RDONLY
	fd, err := syscall.Open(filename, flags, 0)
	if err != nil {
		return 0, nil, err
	}
	file := os.NewFile(uintptr(fd), filename)
	if file == nil {
		return 0, nil, fmt.Errorf("failed to create file from fd")
	}

	return fd, file, nil
}
