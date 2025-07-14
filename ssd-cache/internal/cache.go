package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/allocator"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/freecache"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/index"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/memtable"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/pool"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

const (
	// File flags for testing
	O_DIRECT          = 0x4000
	O_WRONLY          = syscall.O_WRONLY
	O_RDONLY          = syscall.O_RDONLY
	O_APPEND          = syscall.O_APPEND
	O_CREAT           = syscall.O_CREAT
	O_DSYNC           = syscall.O_DSYNC
	FILE_MODE         = 0644
	BLOCK_SIZE        = 4096
	INDEX_BUFFER_SIZE = 16
)

type Cache struct {
	writeFD            int
	readFD             int
	memtableSize       int32
	filePunchHoleSize  int32
	fileStartOffset    int64
	fileCurrentSize    int64
	fileMaxSize        int64
	fromDiskCount      int64
	fromMemtableCount  int64
	fromLruCacheCount  int64
	punchHoleCount     int64
	punchHoleMissCount int64
	writeFile          *os.File
	readFile           *os.File
	index              *index.Index
	lruCache           *freecache.Cache
	memtableManager    *memtable.MemtableManager
	writePageAllocator *allocator.AlignedPageAllocator
	readPageAllocator  *allocator.SlabAlignedPageAllocator
}

type CacheConfig struct {
	MemtableCapacity       int32
	BlockSizeMultipliers   []int
	LRUCacheSize           int
	FileMaxSize            int64
	IndexBufferPoolMinSize int
}

func NewCache(config CacheConfig) *Cache {
	memtableCapacity := config.MemtableCapacity
	if memtableCapacity%BLOCK_SIZE != 0 {
		memtableCapacity = (memtableCapacity/BLOCK_SIZE + 1) * BLOCK_SIZE
	}
	maxPages := make([]int, len(config.BlockSizeMultipliers))
	for i := range config.BlockSizeMultipliers {
		maxPages[i] = 1
	}
	writePageAllocator := allocator.NewAlignedPageAllocator(allocator.AlignedPageAllocatorConfig{
		PageSizeAlignement: BLOCK_SIZE,
		Multiplier:         int(memtableCapacity / BLOCK_SIZE),
		MaxPages:           1000,
	})

	readPageAllocator := allocator.NewSlabAlignedPageAllocator(allocator.SlabAlignedPageAllocatorConfig{
		PageSizeAlignement: BLOCK_SIZE,
		Multipliers:        config.BlockSizeMultipliers,
		MaxPages:           maxPages,
	})
	bufferPool := pool.NewLeakyPool(config.IndexBufferPoolMinSize, INDEX_BUFFER_SIZE, func() interface{} {
		return make([]byte, INDEX_BUFFER_SIZE)
	})
	filename := fmt.Sprintf("test_memtable_%d.dat", time.Now().UnixNano())
	filename = filepath.Join(".", filename)

	writeFd, writeFile, err := createWriteFileDescriptor(filename)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create write file descriptor")
	}
	readFd, readFile, err := createReadFileDescriptor(filename)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create read file descriptor")
	}
	memtableManager := memtable.NewMemtableManager(writeFd, memtableCapacity, writePageAllocator)
	index := index.NewIndexV2(bufferPool, int32(memtableCapacity))
	lruCache := freecache.NewCache(config.LRUCacheSize)

	return &Cache{
		memtableManager:    memtableManager,
		writePageAllocator: writePageAllocator,
		readPageAllocator:  readPageAllocator,
		lruCache:           lruCache,
		index:              index,
		writeFD:            writeFd,
		writeFile:          writeFile,
		readFD:             readFd,
		readFile:           readFile,
		fileStartOffset:    0,
		filePunchHoleSize:  memtableCapacity,
		fileCurrentSize:    0,
		fileMaxSize:        config.FileMaxSize,
		fromDiskCount:      0,
		fromMemtableCount:  0,
		memtableSize:       memtableCapacity,
		punchHoleCount:     0,
		punchHoleMissCount: 0,
	}
}

func (c *Cache) Put(key string, value []byte) {
	memtable, memtableId, _ := c.memtableManager.GetMemtable()
	data := append([]byte(key), value...)
	offset, length := memtable.Put(data)
	if offset == -1 && length == -1 {
		c.memtableManager.Flush()
		c.fileCurrentSize += int64(c.memtableSize)

		if c.fileCurrentSize > c.fileMaxSize {
			err := unix.Fallocate(c.writeFD, unix.FALLOC_FL_PUNCH_HOLE|unix.FALLOC_FL_KEEP_SIZE, c.fileStartOffset, int64(c.filePunchHoleSize))
			if err != nil {
				log.Error().Msgf("Failed to punch hole: %v", err)
			} else {
				log.Info().Msgf("Punched hole: %d", c.fileStartOffset)
			}
			c.fileStartOffset += int64(c.filePunchHoleSize)
			c.fileCurrentSize -= int64(c.filePunchHoleSize)
			c.punchHoleCount++
		}
		memtable, memtableId, _ = c.memtableManager.GetMemtable()
		offset, length = memtable.Put(data)
	}
	c.index.Put(key, offset, length, memtableId)
}

func (c *Cache) Get(key string) []byte {
	if value, err := c.lruCache.Get([]byte(key)); err == nil {
		c.fromLruCacheCount++
		return value
	}
	offset, length, fileOffset, id, ok := c.index.Get(key)
	if !ok || fileOffset < c.fileStartOffset {
		return nil
	}
	memtable := c.memtableManager.GetMemtableById(id)
	var data []byte
	if memtable != nil {
		data = memtable.Get(offset, length)
		c.fromMemtableCount++
	} else {
		data = make([]byte, length)
		c.ReadFromDisk(fileOffset, length, data)
		c.lruCache.Set([]byte(key), data[len(key):], 0)
		c.fromDiskCount++
	}
	gotKey := string(data[:len(key)])
	if gotKey != key {
		return nil
	}
	return data[len(key):]
}

func (c *Cache) Discard() {
	// TODO: implement cleanup for memtables
	//Wait for flush to complete
	time.Sleep(1 * time.Second)
	syscall.Close(c.writeFD)
	syscall.Close(c.readFD)
	os.Remove(c.writeFile.Name())
	os.Remove(c.readFile.Name())
}

func (c *Cache) ReadFromDisk(fileOffset int64, length int32, buf []byte) error {
	if fileOffset < c.fileStartOffset {
		c.punchHoleMissCount++
		return fmt.Errorf("fileOffset is less than fileStartOffset")
	}
	alignedStartOffset := (fileOffset / BLOCK_SIZE) * BLOCK_SIZE
	endndOffset := fileOffset + int64(length)
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
	copy(buf, page.Buf[start:start+int64(length)])
	log.Debug().Msgf("Read data: %s", string(buf))
	c.readPageAllocator.Put(page)
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
