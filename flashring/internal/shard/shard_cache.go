package filecache

import (
	"fmt"
	"hash/crc32"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/allocators"
	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	"github.com/Meesho/BharatMLStack/flashring/internal/index"
	"github.com/Meesho/BharatMLStack/flashring/internal/iouring"
	"github.com/Meesho/BharatMLStack/flashring/internal/maths"
	"github.com/Meesho/BharatMLStack/flashring/internal/memtables"
	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
	"github.com/rs/zerolog/log"
)

type ShardCache struct {
	keyIndex          *index.Index
	file              *fs.WrapAppendFile
	iouringReader     *iouring.ParallelBatchIoUringReader
	mm                *memtables.MemtableManager
	readPageAllocator *allocators.SlabAlignedPageAllocator
	dm                *index.DeleteManager
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
	Predictor           *maths.Predictor

	// Global batched io_uring reader (shared across all shards).
	IoUringReader *iouring.ParallelBatchIoUringReader

	// Dedicated io_uring writer for batched writes (shared across all shards).
	IoUringWriter *iouring.IoUringWriter
}

func NewShardCache(config ShardCacheConfig, sl *sync.RWMutex) (*ShardCache, error) {
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
		return nil, fmt.Errorf("create shard file: %w", err)
	}
	memtableManager, err := memtables.NewMemtableManager(file, config.MemtableSize)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("create memtable manager: %w", err)
	}
	ki := index.NewIndex(0, config.RbInitial, config.RbMax, config.DeleteAmortizedStep, sl)

	sizeClasses := make([]allocators.SizeClass, 0)
	i := fs.BLOCK_SIZE
	minCount := 24
	iMax := (1 << 16)
	for i < iMax {
		sizeClasses = append(sizeClasses, allocators.SizeClass{Size: i, MinCount: minCount})
		i *= 2
		minCount /= 2
	}
	readPageAllocator, err := allocators.NewSlabAlignedPageAllocator(allocators.SlabAlignedPageAllocatorConfig{SizeClasses: sizeClasses})
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("create read page allocator: %w", err)
	}
	dm := index.NewDeleteManager(ki, file, config.DeleteAmortizedStep)

	file.WriteRing = config.IoUringWriter

	sc := &ShardCache{
		keyIndex:          ki,
		mm:                memtableManager,
		file:              file,
		readPageAllocator: readPageAllocator,
		dm:                dm,
		predictor:         config.Predictor,
		startAt:           time.Now().Unix(),
	}

	if config.IoUringReader == nil {
		file.Close()
		return nil, fmt.Errorf("BatchIoUringReader is required")
	}
	sc.iouringReader = config.IoUringReader

	return sc, nil
}

func (fc *ShardCache) Put(key string, value []byte, ttlMinutes uint16) error {
	size := 4 + len(key) + len(value)
	mt, mtId, _ := fc.mm.GetMemtable()
	if err := fc.dm.ExecuteDeleteIfNeeded(); err != nil {
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
	index.ByteOrder.PutUint32(buf[0:4], crc)
	fc.keyIndex.Put(key, length, ttlMinutes, mtId, uint32(offset))
	fc.dm.IncMemtableKeyCount(mtId)
	return nil
}

func (fc *ShardCache) Get(key string) (bool, []byte, uint16, bool, bool) {
	length, lastAccess, remainingTTL, freq, memId, offset, status := fc.keyIndex.Get(key)
	if status == index.StatusNotFound {
		metrics.Incr(metrics.KEY_KEY_NOT_FOUND_COUNT, []string{})
		return false, nil, 0, false, false
	}

	metrics.Timing(metrics.KEY_DATA_LENGTH, time.Duration(length), []string{})

	if status == index.StatusExpired {
		metrics.Incr(metrics.KEY_KEY_EXPIRED_COUNT, []string{})
		return false, nil, 0, true, false
	}

	_, currMemId, _ := fc.mm.GetMemtable()
	shouldReWrite := fc.predictor.Predict(uint64(freq), uint64(lastAccess), memId, currMemId)

	var buf []byte
	mt := fc.mm.GetMemtableById(memId)
	if mt == nil {
		metrics.Incr(metrics.KEY_MEMTABLE_MISS, []string{})
		buf = make([]byte, length)
		fileOffset := uint64(memId)*uint64(fc.mm.Capacity) + uint64(offset)
		n := fc.readFromDiskAsync(int64(fileOffset), length, buf)
		if n != int(length) {
			metrics.Incr(metrics.KEY_BAD_LENGTH_COUNT, []string{})
			return false, nil, 0, false, shouldReWrite
		}
	} else {
		metrics.Incr(metrics.KEY_MEMTABLE_HIT, []string{})
		var exists bool
		buf, exists = mt.GetBufForRead(int(offset), length)
		if !exists {
			return false, nil, 0, false, shouldReWrite
		}
	}
	gotCR32 := index.ByteOrder.Uint32(buf[0:4])
	computedCR32 := crc32.ChecksumIEEE(buf[4:length])
	gotKey := string(buf[4 : 4+len(key)])
	if gotCR32 != computedCR32 {
		metrics.Incr(metrics.KEY_BAD_CR32_COUNT, []string{})
		return false, nil, 0, false, shouldReWrite
	}
	if gotKey != key {
		metrics.Incr(metrics.KEY_BAD_KEY_COUNT, []string{})
		return false, nil, 0, false, shouldReWrite
	}
	valLen := int(length) - 4 - len(key)
	return true, buf[4+len(key) : 4+len(key)+valLen], remainingTTL, false, shouldReWrite
}

func (fc *ShardCache) readFromDiskAsync(fileOffset int64, length uint16, buf []byte) int {
	alignedStart, alignedSize := fs.AlignRange(fileOffset, int(length), fs.BLOCK_SIZE)
	page := fc.readPageAllocator.Get(int(alignedSize))

	readBuf := page.Buf[:alignedSize]

	var n int
	var err error
	var validOffset int64
	validOffset, err = fc.file.ValidateReadOffset(alignedStart, int(alignedSize))
	if err == nil {
		n, err = fc.iouringReader.Submit(fc.file.ReadFd, readBuf, uint64(validOffset))
	}

	if err != nil || n != int(alignedSize) {
		if err != nil && err != fs.ErrFileOffsetOutOfRange {
			log.Warn().Err(err).
				Int64("offset", alignedStart).
				Int64("alignedReadSize", alignedSize).
				Int("n", n).
				Msg("io_uring pread failed")
		}
		fc.readPageAllocator.Put(page)
		return 0
	}

	start := int(fileOffset - alignedStart)
	copied := copy(buf, page.Buf[start:start+int(length)])
	fc.readPageAllocator.Put(page)
	return copied
}

func (fc *ShardCache) Flush() {
	fc.mm.Flush()
}

func (fc *ShardCache) Close() {
	fc.file.Close()
}

func (fc *ShardCache) GetRingBufferActiveEntries() int {
	return fc.keyIndex.GetRB().ActiveEntries()
}
