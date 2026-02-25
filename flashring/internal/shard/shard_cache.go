package filecache

import (
	"fmt"
	"hash/crc32"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/allocators"
	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	indices "github.com/Meesho/BharatMLStack/flashring/internal/indicesV3"
	"github.com/Meesho/BharatMLStack/flashring/internal/maths"
	"github.com/Meesho/BharatMLStack/flashring/internal/memtables"
	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
	"github.com/rs/zerolog/log"
)

type ShardCache struct {
	keyIndex          *indices.Index
	file              *fs.WrapAppendFile
	ioFile            *fs.IOUringFile
	batchReader       *fs.ParallelBatchIoUringReader // global batched io_uring reader (shared across shards)
	mm                *memtables.MemtableManager
	readPageAllocator *allocators.SlabAlignedPageAllocator
	dm                *indices.DeleteManager
	predictor         *maths.Predictor
	startAt           int64
	ShardIdx          uint32
}

type ShardCacheConfig struct {
	Rounds              int
	RbInitial           int
	RbMax               int
	DeleteAmortizedStep int
	MemtableSize        int32
	MaxFileSize         int64
	BlockSize           int
	Directory           string
	AsyncReadWorkers    int
	AsyncQueueDepth     int
	Predictor           *maths.Predictor

	// Global batched io_uring reader (shared across all shards).
	// When set, disk reads go through this instead of the per-shard IOUringFile.
	BatchIoUringReader *fs.ParallelBatchIoUringReader

	// Dedicated io_uring ring for batched writes (shared across all shards).
	WriteRing *fs.IoUring
}

func NewShardCache(config ShardCacheConfig, sl *sync.RWMutex) *ShardCache {
	filename := fmt.Sprintf("%s/%d.bin", config.Directory, time.Now().UnixNano())
	punchHoleSize := config.MemtableSize
	fsConf := fs.FileConfig{
		Filename:          filename,
		MaxFileSize:       config.MaxFileSize,
		FilePunchHoleSize: int64(punchHoleSize),
		BlockSize:         config.BlockSize,
	}
	file, err := fs.NewWrapAppendFile(fsConf)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to create file")
	}
	memtableManager, err := memtables.NewMemtableManager(file, config.MemtableSize)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to create memtable manager")
	}
	ki := indices.NewIndex(0, config.RbInitial, config.RbMax, config.DeleteAmortizedStep, sl)
	sizeClasses := make([]allocators.SizeClass, 0)
	i := fs.BLOCK_SIZE
	iMax := (1 << 16)
	for i < iMax {
		sizeClasses = append(sizeClasses, allocators.SizeClass{Size: i, MinCount: 20})
		i *= 2
	}
	readPageAllocator, err := allocators.NewSlabAlignedPageAllocator(allocators.SlabAlignedPageAllocatorConfig{SizeClasses: sizeClasses})
	if err != nil {
		log.Panic().Err(err).Msg("Failed to create read page allocator")
	}
	dm := indices.NewDeleteManager(ki, file, config.DeleteAmortizedStep)

	// Attach the dedicated write ring so memtable flushes use batched io_uring.
	if config.WriteRing != nil {
		file.WriteRing = config.WriteRing
	}

	sc := &ShardCache{
		keyIndex:          ki,
		mm:                memtableManager,
		file:              file,
		readPageAllocator: readPageAllocator,
		dm:                dm,
		predictor:         config.Predictor,
		startAt:           time.Now().Unix(),
	}

	if config.BatchIoUringReader != nil {
		// Use the global batched io_uring reader (shared across all shards).
		sc.batchReader = config.BatchIoUringReader
	} else {
		log.Panic().Msg("BatchIoUringReader is required")
	}

	return sc
}

func (fc *ShardCache) Put(key string, value []byte, ttlMinutes uint16) error {
	size := 4 + len(key) + len(value)
	mt, mtId, _ := fc.mm.GetMemtable()
	err := fc.dm.ExecuteDeleteIfNeeded()
	if err != nil {
		return err
	}
	buf, offset, length, readyForFlush := mt.GetBufForAppend(uint16(size))
	if readyForFlush {
		fc.mm.Flush()
		mt, mtId, _ = fc.mm.GetMemtable()
		buf, offset, length, _ = mt.GetBufForAppend(uint16(size))
	}
	copy(buf[4:], key)
	copy(buf[4+len(key):], value)
	crc := crc32.ChecksumIEEE(buf[4:])
	indices.ByteOrder.PutUint32(buf[0:4], crc)
	fc.keyIndex.Put(key, length, ttlMinutes, mtId, uint32(offset))
	fc.dm.IncMemtableKeyCount(mtId)
	return nil
}

func (fc *ShardCache) Get(key string) (bool, []byte, uint16, bool, bool) {
	length, lastAccess, remainingTTL, freq, memId, offset, status := fc.keyIndex.Get(key)
	if status == indices.StatusNotFound {
		if metrics.Enabled() {
			metrics.Incr(metrics.KEY_KEY_NOT_FOUND_COUNT, metrics.GetShardTag(fc.ShardIdx))
		}
		return false, nil, 0, false, false
	}

	if metrics.Enabled() {
		metrics.Timing(metrics.KEY_DATA_LENGTH, time.Duration(length), metrics.GetShardTag(fc.ShardIdx))
	}

	if status == indices.StatusExpired {
		if metrics.Enabled() {
			metrics.Incr(metrics.KEY_KEY_EXPIRED_COUNT, metrics.GetShardTag(fc.ShardIdx))
		}
		return false, nil, 0, true, false
	}

	_, currMemId, _ := fc.mm.GetMemtable()
	shouldReWrite := fc.predictor.Predict(uint64(freq), uint64(lastAccess), memId, currMemId)

	exists := true
	var buf []byte
	memtableExists := true
	mt := fc.mm.GetMemtableById(memId)
	if mt == nil {
		memtableExists = false
	}
	if !memtableExists {
		if metrics.Enabled() {
			metrics.Incr(metrics.KEY_MEMTABLE_MISS, metrics.GetShardTag(fc.ShardIdx))
		}
		buf = make([]byte, length)
		fileOffset := uint64(memId)*uint64(fc.mm.Capacity) + uint64(offset)
		n := fc.readFromDiskAsync(int64(fileOffset), length, buf)
		if n != int(length) {
			if metrics.Enabled() {
				metrics.Incr(metrics.KEY_BAD_LENGTH_COUNT, metrics.GetShardTag(fc.ShardIdx))
			}
			return false, nil, 0, false, shouldReWrite
		}
	} else {
		if metrics.Enabled() {
			metrics.Incr(metrics.KEY_MEMTABLE_HIT, metrics.GetShardTag(fc.ShardIdx))
		}
		buf, exists = mt.GetBufForRead(int(offset), length)
		if !exists {
			panic("memtable exists but buf not found")
		}
	}
	gotCR32 := indices.ByteOrder.Uint32(buf[0:4])
	computedCR32 := crc32.ChecksumIEEE(buf[4:length])
	gotKey := string(buf[4 : 4+len(key)])
	if gotCR32 != computedCR32 {
		if metrics.Enabled() {
			metrics.Incr(metrics.KEY_BAD_CR32_COUNT, append(metrics.GetShardTag(fc.ShardIdx), metrics.GetMemtableTag(memId)...))
		}
		return false, nil, 0, false, shouldReWrite
	}
	if gotKey != key {
		if metrics.Enabled() {
			metrics.Incr(metrics.KEY_BAD_KEY_COUNT, append(metrics.GetShardTag(fc.ShardIdx), metrics.GetMemtableTag(memId)...))
		}
		return false, nil, 0, false, shouldReWrite
	}
	valLen := int(length) - 4 - len(key)
	return true, buf[4+len(key) : 4+len(key)+valLen], remainingTTL, false, shouldReWrite
}

func (fc *ShardCache) readFromDisk(fileOffset int64, length uint16, buf []byte) int {

	alignedStartOffset := (fileOffset / fs.BLOCK_SIZE) * fs.BLOCK_SIZE
	endndOffset := fileOffset + int64(length)
	endAlignedOffset := ((endndOffset + fs.BLOCK_SIZE - 1) / fs.BLOCK_SIZE) * fs.BLOCK_SIZE
	alignedReadSize := endAlignedOffset - alignedStartOffset

	page := fc.readPageAllocator.Get(int(alignedReadSize))

	fc.file.Pread(alignedStartOffset, page.Buf)

	start := int(fileOffset - alignedStartOffset)
	n := copy(buf, page.Buf[start:start+int(length)])
	fc.readPageAllocator.Put(page)
	return n
}

func (fc *ShardCache) readFromDiskAsync(fileOffset int64, length uint16, buf []byte) int {
	alignedStartOffset := (fileOffset / fs.BLOCK_SIZE) * fs.BLOCK_SIZE
	endndOffset := fileOffset + int64(length)
	endAlignedOffset := ((endndOffset + fs.BLOCK_SIZE - 1) / fs.BLOCK_SIZE) * fs.BLOCK_SIZE
	alignedReadSize := int(endAlignedOffset - alignedStartOffset)
	page := fc.readPageAllocator.Get(alignedReadSize)

	// Use exactly alignedReadSize bytes, not the full page.Buf which may be
	// larger due to slab allocator rounding to the next size class.
	readBuf := page.Buf[:alignedReadSize]

	var n int
	var err error
	// Batched path: validate offset locally, then submit to the global
	// io_uring batch reader which accumulates requests across all shards.
	var validOffset int64
	validOffset, err = fc.file.ValidateReadOffset(alignedStartOffset, alignedReadSize)
	if err == nil {
		n, err = fc.batchReader.Submit(fc.file.ReadFd, readBuf, uint64(validOffset))
	}

	if err != nil || n != alignedReadSize {
		// ErrFileOffsetOutOfRange is expected for stale index entries.
		if err != nil && err != fs.ErrFileOffsetOutOfRange {
			log.Warn().Err(err).
				Int64("offset", alignedStartOffset).
				Int("alignedReadSize", alignedReadSize).
				Int("n", n).
				Msg("io_uring pread failed")
		}
		fc.readPageAllocator.Put(page)
		return 0
	}

	start := int(fileOffset - alignedStartOffset)
	copied := copy(buf, page.Buf[start:start+int(length)])
	fc.readPageAllocator.Put(page)
	return copied
}

func (fc *ShardCache) GetRingBufferActiveEntries() int {
	return fc.keyIndex.GetRB().ActiveEntries()
}
