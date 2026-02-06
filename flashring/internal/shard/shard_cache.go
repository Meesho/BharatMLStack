package filecache

import (
	"fmt"
	"hash/crc32"
	"strconv"
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
	mm                *memtables.MemtableManager
	readPageAllocator *allocators.SlabAlignedPageAllocator
	dm                *indices.DeleteManager
	predictor         *maths.Predictor
	startAt           int64
	shardId           int
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

	//batching reads
	EnableBatching bool
	BatchWindow    time.Duration
	MaxBatchSize   int
}

func NewShardCache(config ShardCacheConfig, sl *sync.RWMutex, shardId int) *ShardCache {
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
	ki := indices.NewIndex(0, config.RbInitial, config.RbMax, config.DeleteAmortizedStep)
	sizeClasses := make([]allocators.SizeClass, 0)
	i := fs.BLOCK_SIZE
	iMax := (1 << 16)
	for i < iMax {
		sizeClasses = append(sizeClasses, allocators.SizeClass{Size: i, MinCount: 1000})
		i *= 2
	}
	readPageAllocator, err := allocators.NewSlabAlignedPageAllocator(allocators.SlabAlignedPageAllocatorConfig{SizeClasses: sizeClasses})
	if err != nil {
		log.Panic().Err(err).Msg("Failed to create read page allocator")
	}
	dm := indices.NewDeleteManager(ki, file, config.DeleteAmortizedStep)
	sc := &ShardCache{
		keyIndex:          ki,
		mm:                memtableManager,
		file:              file,
		readPageAllocator: readPageAllocator,
		dm:                dm,
		predictor:         config.Predictor,
		startAt:           time.Now().Unix(),
		shardId:           shardId,
	}
	return sc
}

func (fc *ShardCache) Put(key string, value []byte, ttlMinutes uint16) error {
	size := 4 + len(key) + len(value)
	mt, mtId, _ := fc.mm.GetMemtable()
	err := fc.dm.ExecuteDeleteIfNeeded()
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute delete if needed")
		return err
	}
	buf, offset, length, readyForFlush := mt.GetBufForAppend(uint16(size))
	if readyForFlush {
		start := time.Now()
		fc.mm.Flush()
		metrics.Timing("flashring.shard.put.memtable_flush.latency", time.Since(start), []string{"memtable_id", strconv.Itoa(int(mtId))})
		metrics.Count("flashring.shard.put.memtable_flush.count", 1, []string{"memtable_id", strconv.Itoa(int(mtId))})
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

// GetMetadata returns metadata for a key (used for lock-free SSD reads)
// Returns: found, isInMemtable, memtableBuf, length, memId, offset, remainingTTL, expired, shouldReWrite
func (fc *ShardCache) GetMetadata(key string) (found bool, isInMemtable bool, memtableBuf []byte, length uint16, memId uint32, offset uint32, remainingTTL uint16, expired bool, shouldReWrite bool) {
	keyLength, lastAccess, ttl, freq, mId, off, status := fc.keyIndex.Get(key)

	if status == indices.StatusNotFound {
		return false, false, nil, 0, 0, 0, 0, false, false
	}

	if status == indices.StatusExpired {
		return true, false, nil, 0, 0, 0, 0, true, false
	}

	_, currMemId, _ := fc.mm.GetMemtable()
	rewrite := fc.predictor.Predict(uint64(freq), uint64(lastAccess), mId, currMemId)

	// Check if data is in memtable
	mt := fc.mm.GetMemtableById(mId)
	if mt != nil {
		// Data is in memtable - read it now (fast, keep under lock)
		buf, exists := mt.GetBufForRead(int(off), keyLength)
		if !exists {
			return false, false, nil, 0, 0, 0, 0, false, false
		}
		return true, true, buf, keyLength, mId, off, ttl, false, rewrite
	}

	// Data is on SSD - return metadata for lock-free read
	return true, false, nil, keyLength, mId, off, ttl, false, rewrite
}

// ReadFromSSD reads data from SSD given metadata (can be called without lock)
func (fc *ShardCache) ReadFromSSD(memId uint32, offset uint32, length uint16) []byte {
	buf := make([]byte, length)
	fileOffset := uint64(memId)*uint64(fc.mm.Capacity) + uint64(offset)
	n := fc.readFromDisk(int64(fileOffset), length, buf)
	if n != int(length) {
		return nil
	}
	return buf
}

// ValidateBuffer validates CRC and key, returns value if valid
func (fc *ShardCache) ValidateBuffer(key string, buf []byte, length uint16) (bool, []byte) {
	if buf == nil || len(buf) < 4+len(key) {
		return false, nil
	}
	gotCRC := indices.ByteOrder.Uint32(buf[0:4])
	computedCRC := crc32.ChecksumIEEE(buf[4:length])
	if gotCRC != computedCRC {
		return false, nil
	}
	gotKey := string(buf[4 : 4+len(key)])
	if gotKey != key {
		return false, nil
	}
	valLen := int(length) - 4 - len(key)
	return true, buf[4+len(key) : 4+len(key)+valLen]
}

// Get is the original method (kept for compatibility, used for memtable reads)
func (fc *ShardCache) Get(key string) (bool, []byte, uint16, bool, bool) {
	length, lastAccess, remainingTTL, freq, memId, offset, status := fc.keyIndex.Get(key)

	if status == indices.StatusNotFound {
		return false, nil, 0, false, false
	}

	if status == indices.StatusExpired {
		return true, nil, 0, true, false
	}

	_, currMemId, _ := fc.mm.GetMemtable()
	shouldReWrite := fc.predictor.Predict(uint64(freq), uint64(lastAccess), memId, currMemId)

	var buf []byte
	mt := fc.mm.GetMemtableById(memId)
	if mt == nil {
		buf = make([]byte, length)
		fileOffset := uint64(memId)*uint64(fc.mm.Capacity) + uint64(offset)
		n := fc.readFromDisk(int64(fileOffset), length, buf)
		if n != int(length) {
			return false, nil, 0, false, shouldReWrite
		}
	} else {
		var exists bool
		buf, exists = mt.GetBufForRead(int(offset), length)
		if !exists {
			return false, nil, 0, false, false
		}
	}

	gotCRC := indices.ByteOrder.Uint32(buf[0:4])
	computedCRC := crc32.ChecksumIEEE(buf[4:length])
	gotKey := string(buf[4 : 4+len(key)])

	if gotCRC != computedCRC {
		return false, nil, 0, false, shouldReWrite
	}
	if gotKey != key {
		return false, nil, 0, false, shouldReWrite
	}
	valLen := int(length) - 4 - len(key)
	return true, buf[4+len(key) : 4+len(key)+valLen], remainingTTL, false, shouldReWrite
}

// validateAndReturnBuffer validates CRC and key, then returns the value
func (fc *ShardCache) validateAndReturnBuffer(key string, buf []byte, length uint16, memId uint32, remainingTTL uint16, shouldReWrite bool) (bool, []byte, uint16, bool, bool) {
	gotCR32 := indices.ByteOrder.Uint32(buf[0:4])
	computedCR32 := crc32.ChecksumIEEE(buf[4:length])
	if gotCR32 != computedCR32 {
		metrics.Count("flashring.shard.get.bad_crc.count", 1, []string{"memtable_id", strconv.Itoa(int(memId))})
		return false, nil, 0, false, shouldReWrite
	}

	gotKey := string(buf[4 : 4+len(key)])
	if gotKey != key {
		metrics.Count("flashring.shard.get.bad_key.count", 1, []string{"memtable_id", strconv.Itoa(int(memId))})
		return false, nil, 0, false, shouldReWrite
	}

	valLen := int(length) - 4 - len(key)
	return true, buf[4+len(key) : 4+len(key)+valLen], remainingTTL, false, shouldReWrite
}

func (fc *ShardCache) readFromDisk(fileOffset int64, length uint16, buf []byte) int {
	// shardTag := []string{"shard_id", strconv.Itoa(fc.shardId)}

	alignedStartOffset := (fileOffset / fs.BLOCK_SIZE) * fs.BLOCK_SIZE
	endndOffset := fileOffset + int64(length)
	endAlignedOffset := ((endndOffset + fs.BLOCK_SIZE - 1) / fs.BLOCK_SIZE) * fs.BLOCK_SIZE
	alignedReadSize := endAlignedOffset - alignedStartOffset

	// Measure allocator get time
	// allocGetStart := time.Now()
	page := fc.readPageAllocator.Get(int(alignedReadSize))
	// metrics.Timing("flashring.disk.alloc_get.latency", time.Since(allocGetStart), shardTag)

	// Measure pure Pread syscall time
	// preadStart := time.Now()
	fc.file.Pread(alignedStartOffset, page.Buf)
	// metrics.Timing("flashring.disk.pread.latency", time.Since(preadStart), shardTag)

	start := int(fileOffset - alignedStartOffset)
	n := copy(buf, page.Buf[start:start+int(length)])
	fc.readPageAllocator.Put(page)
	return n
}

func (fc *ShardCache) GetRingBufferActiveEntries() int {
	return fc.keyIndex.GetRB().ActiveEntries()
}
