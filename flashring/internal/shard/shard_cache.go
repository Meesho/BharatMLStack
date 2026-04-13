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

	// FlushStaggerOffset pre-advances the first memtable so shards flush at
	// staggered times instead of all at once.
	FlushStaggerOffset int
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
	memtableManager, err := memtables.NewMemtableManager(file, config.MemtableSize, config.FlushStaggerOffset)
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

// DeleteKey removes the key from the index only. Debug use only.
func (fc *ShardCache) DeleteKey(key string) bool {
	return fc.keyIndex.DeleteKey(key)
}

func (fc *ShardCache) GetRingBufferActiveEntries() int {
	return fc.keyIndex.GetRB().ActiveEntries()
}

// ---------------------------------------------------------------------------
// MGet support — separate functions that duplicate parts of Get/readFromDiskAsync
// to allow the caller to split index lookups from disk I/O.
// ---------------------------------------------------------------------------

// MGetMeta holds the result of an index lookup for batch gets.
type MGetMeta struct {
	Found         bool
	Expired       bool
	ShouldReWrite bool
	RemainingTTL  uint16
	// Value is non-nil when the data was found in a memtable (no disk read needed).
	Value         []byte
	NeedsDiskRead bool
	Length        uint16
	FileOffset    int64
}

// PendingRead represents an in-flight async io_uring disk read.
type PendingRead struct {
	done        <-chan iouring.ReadResult
	page        *fs.AlignedPage
	alignedSize int
	pageOffset  int
	length      uint16
}

// GetMetaForMGet performs an index lookup and memtable check for a single key
// without issuing any disk I/O. This is the first phase of an MGet operation.
func (fc *ShardCache) GetMetaForMGet(key string) MGetMeta {
	length, lastAccess, remainingTTL, freq, memId, offset, status := fc.keyIndex.Get(key)

	if status == index.StatusNotFound {
		metrics.Incr(metrics.KEY_KEY_NOT_FOUND_COUNT, []string{})
		return MGetMeta{}
	}

	metrics.Timing(metrics.KEY_DATA_LENGTH, time.Duration(length), []string{})

	if status == index.StatusExpired {
		metrics.Incr(metrics.KEY_KEY_EXPIRED_COUNT, []string{})
		return MGetMeta{Expired: true}
	}

	_, currMemId, _ := fc.mm.GetMemtable()
	shouldReWrite := fc.predictor.Predict(uint64(freq), uint64(lastAccess), memId, currMemId)

	mt := fc.mm.GetMemtableById(memId)
	if mt != nil {
		metrics.Incr(metrics.KEY_MEMTABLE_HIT, []string{})
		buf, exists := mt.GetBufForRead(int(offset), length)
		if !exists {
			return MGetMeta{ShouldReWrite: shouldReWrite}
		}
		return MGetMeta{
			Found:         true,
			Value:         buf,
			Length:        length,
			RemainingTTL:  remainingTTL,
			ShouldReWrite: shouldReWrite,
		}
	}

	metrics.Incr(metrics.KEY_MEMTABLE_MISS, []string{})
	fileOffset := int64(uint64(memId)*uint64(fc.mm.Capacity) + uint64(offset))

	return MGetMeta{
		Found:         true,
		NeedsDiskRead: true,
		Length:        length,
		FileOffset:    fileOffset,
		RemainingTTL:  remainingTTL,
		ShouldReWrite: shouldReWrite,
	}
}

// SubmitDiskReadAsync enqueues an aligned disk read via io_uring without
// blocking for completion. Returns a PendingRead handle for CollectDiskRead.
func (fc *ShardCache) SubmitDiskReadAsync(fileOffset int64, length uint16) (*PendingRead, error) {
	alignedStart, alignedSize := fs.AlignRange(fileOffset, int(length), fs.BLOCK_SIZE)
	page := fc.readPageAllocator.Get(int(alignedSize))
	readBuf := page.Buf[:alignedSize]

	validOffset, err := fc.file.ValidateReadOffset(alignedStart, int(alignedSize))
	if err != nil {
		fc.readPageAllocator.Put(page)
		return nil, err
	}

	done := fc.iouringReader.SubmitAsync(fc.file.ReadFd, readBuf, uint64(validOffset))

	return &PendingRead{
		done:        done,
		page:        page,
		alignedSize: int(alignedSize),
		pageOffset:  int(fileOffset - alignedStart),
		length:      length,
	}, nil
}

// CollectDiskRead blocks until the pending io_uring read completes, copies
// the result into a new buffer, and frees the aligned page. Returns nil on failure.
func (fc *ShardCache) CollectDiskRead(pr *PendingRead) []byte {
	result := <-pr.done
	defer fc.readPageAllocator.Put(pr.page)

	if result.Err != nil || result.N != pr.alignedSize {
		if result.Err != nil {
			log.Warn().Err(result.Err).Msg("io_uring pread failed in MGet")
		}
		return nil
	}

	buf := make([]byte, pr.length)
	copy(buf, pr.page.Buf[pr.pageOffset:pr.pageOffset+int(pr.length)])
	return buf
}

// CoalescedPendingRead represents an in-flight async io_uring disk read that
// covers a merged aligned region shared by multiple keys.
type CoalescedPendingRead struct {
	done        <-chan iouring.ReadResult
	page        *fs.AlignedPage
	alignedSize int
}

// SubmitCoalescedReadAsync enqueues a single aligned disk read that covers
// multiple keys whose file offsets fall within [alignedStart, alignedStart+alignedSize).
func (fc *ShardCache) SubmitCoalescedReadAsync(alignedStart int64, alignedSize int) (*CoalescedPendingRead, error) {
	page := fc.readPageAllocator.Get(alignedSize)
	readBuf := page.Buf[:alignedSize]

	validOffset, err := fc.file.ValidateReadOffset(alignedStart, alignedSize)
	if err != nil {
		fc.readPageAllocator.Put(page)
		return nil, err
	}

	done := fc.iouringReader.SubmitAsync(fc.file.ReadFd, readBuf, uint64(validOffset))

	return &CoalescedPendingRead{
		done:        done,
		page:        page,
		alignedSize: alignedSize,
	}, nil
}

// CollectCoalescedRead blocks until the coalesced io_uring read completes and
// returns the full aligned buffer. The caller extracts individual key regions
// using each key's offset relative to the aligned start.
func (fc *ShardCache) CollectCoalescedRead(pr *CoalescedPendingRead) []byte {
	result := <-pr.done
	defer fc.readPageAllocator.Put(pr.page)

	if result.Err != nil || result.N != pr.alignedSize {
		if result.Err != nil {
			log.Warn().Err(result.Err).Msg("io_uring coalesced pread failed in MGet")
		}
		return nil
	}

	buf := make([]byte, pr.alignedSize)
	copy(buf, pr.page.Buf[:pr.alignedSize])
	return buf
}

// ValidateAndExtract checks the CRC32 and key, then extracts the value from
// a raw data buffer. Used by MGet for both memtable and disk-read results.
func (fc *ShardCache) ValidateAndExtract(buf []byte, key string, length uint16) ([]byte, bool) {
	if int(length) > len(buf) || length < 4 {
		metrics.Incr(metrics.KEY_BAD_LENGTH_COUNT, []string{})
		return nil, false
	}
	gotCRC := index.ByteOrder.Uint32(buf[0:4])
	computedCRC := crc32.ChecksumIEEE(buf[4:length])
	if gotCRC != computedCRC {
		metrics.Incr(metrics.KEY_BAD_CR32_COUNT, []string{})
		return nil, false
	}
	gotKey := string(buf[4 : 4+len(key)])
	if gotKey != key {
		metrics.Incr(metrics.KEY_BAD_KEY_COUNT, []string{})
		return nil, false
	}
	valLen := int(length) - 4 - len(key)
	return buf[4+len(key) : 4+len(key)+valLen], true
}
