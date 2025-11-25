package filecache

import (
	"fmt"
	"hash/crc32"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/allocators"
	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	indices "github.com/Meesho/BharatMLStack/flashring/internal/indicesV3"
	"github.com/Meesho/BharatMLStack/flashring/internal/maths"
	"github.com/Meesho/BharatMLStack/flashring/internal/memtables"
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
	Stats             *Stats
}

type Stats struct {
	KeyNotFoundCount int
	KeyExpiredCount  int
	BadDataCount     int
	BadLengthCount   int
	BadCR32Count     int
	BadKeyCount      int
	MemIdCount       map[uint32]int
	LastDeletedMemId uint32
	DeletedKeyCount  int
	BadCRCMemIds     map[uint32]int
	BadKeyMemIds     map[uint32]int
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
}

func NewShardCache(config ShardCacheConfig) *ShardCache {
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
	return &ShardCache{
		keyIndex:          ki,
		mm:                memtableManager,
		file:              file,
		readPageAllocator: readPageAllocator,
		dm:                dm,
		predictor:         config.Predictor,
		startAt:           time.Now().Unix(),
		Stats: &Stats{
			MemIdCount:   make(map[uint32]int),
			BadCRCMemIds: make(map[uint32]int),
			BadKeyMemIds: make(map[uint32]int),
		},
	}
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
	fc.Stats.MemIdCount[mtId]++
	return nil
}

func (fc *ShardCache) Get(key string) (bool, []byte, uint16, bool, bool) {
	length, lastAccess, remainingTTL, freq, memId, offset, status := fc.keyIndex.Get(key)

	if status == indices.StatusNotFound {
		fc.Stats.KeyNotFoundCount++
		return false, nil, 0, false, false
	}

	if status == indices.StatusExpired {
		fc.Stats.KeyExpiredCount++
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
		buf = make([]byte, length)
		fileOffset := uint64(memId)*uint64(fc.mm.Capacity) + uint64(offset)
		n := fc.readFromDisk(int64(fileOffset), length, buf)
		if n != int(length) {
			fc.Stats.BadLengthCount++
			return false, nil, 0, false, shouldReWrite
		}
	} else {
		buf, exists = mt.GetBufForRead(int(offset), length)
		if !exists {
			panic("memtable exists but buf not found")
		}
	}
	gotCR32 := indices.ByteOrder.Uint32(buf[0:4])
	computedCR32 := crc32.ChecksumIEEE(buf[4:])
	gotKey := string(buf[4 : 4+len(key)])
	if gotCR32 != computedCR32 {
		fc.Stats.BadCR32Count++
		fc.Stats.BadCRCMemIds[memId]++
		return false, nil, 0, false, shouldReWrite
	}
	if gotKey != key {
		fc.Stats.BadKeyCount++
		fc.Stats.BadKeyMemIds[memId]++
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

func (fc *ShardCache) GetRingBufferActiveEntries() int {
	return fc.keyIndex.GetRB().ActiveEntries()
}
