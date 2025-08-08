package filecache

import (
	"fmt"
	"hash/crc32"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/allocators"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/fs"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/indices"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/maths"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/memtables"
	"github.com/rs/zerolog/log"
)

type FileCache struct {
	keyIndex          *indices.KeyIndex
	file              *fs.WrapAppendFile
	mm                *memtables.MemtableManager
	readPageAllocator *allocators.SlabAlignedPageAllocator
	dm                *indices.DeleteManager
	predictor         *maths.Predictor
	Stats             Stats
}

type Stats struct {
	KeyNotFoundCount int
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

type FileCacheConfig struct {
	Rounds              int
	RbInitial           int
	RbMax               int
	DeleteAmortizedStep int
	MemtableSize        int32
	MaxFileSize         int64
	BlockSize           int
	Directory           string
	Predictor           *maths.Predictor
}

func NewFileCache(config FileCacheConfig) *FileCache {
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
	ki := indices.NewKeyIndex(config.Rounds, config.RbInitial, config.RbMax, config.DeleteAmortizedStep)
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
	return &FileCache{
		keyIndex:          ki,
		mm:                memtableManager,
		file:              file,
		readPageAllocator: readPageAllocator,
		dm:                dm,
		predictor:         config.Predictor,
		Stats: Stats{
			MemIdCount:   make(map[uint32]int),
			BadCRCMemIds: make(map[uint32]int),
			BadKeyMemIds: make(map[uint32]int),
		},
	}
}

func (fc *FileCache) Put(key string, value []byte, exptime uint64) error {
	size := 4 + len(key) + len(value)
	mt, mtId, _ := fc.mm.GetMemtable()
	buf, offset, length, readyForFlush := mt.GetBufForAppend(uint16(size))
	if readyForFlush {
		trimmedHead := fc.file.TrimHeadIfNeeded()
		if trimmedHead {
			fc.Stats.DeletedKeyCount += fc.Stats.MemIdCount[fc.Stats.LastDeletedMemId]
			fc.Stats.LastDeletedMemId++
			log.Info().Msg("trimmed head")
			fc.keyIndex.StartTrim()
		}
		fc.mm.Flush()
		mt, mtId, _ = fc.mm.GetMemtable()
		buf, offset, length, _ = mt.GetBufForAppend(uint16(size))
	}
	copy(buf[4:], key)
	copy(buf[4+len(key):], value)
	crc := crc32.ChecksumIEEE(buf[4:])
	indices.ByteOrder.PutUint32(buf[0:4], crc)
	fc.keyIndex.Put(key, length, mtId, uint32(offset), exptime)
	fc.Stats.MemIdCount[mtId]++
	return nil
}

func (fc *FileCache) PutV2(key string, value []byte, exptime uint64) error {
	size := 4 + len(key) + len(value)
	mt, mtId, _ := fc.mm.GetMemtable()
	err := fc.dm.ExecuteDeleteIfNeeded()
	if err != nil {
		return err
	}
	buf, offset, length, readyForFlush := mt.GetBufForAppend(uint16(size))
	if readyForFlush {
		// trimmedHead := fc.file.TrimHeadIfNeeded()
		// if trimmedHead {
		// 	fc.Stats.DeletedKeyCount += fc.Stats.MemIdCount[fc.Stats.LastDeletedMemId]
		// 	fc.Stats.LastDeletedMemId++
		// 	log.Info().Msg("trimmed head")
		// 	fc.keyIndex.StartTrim()
		// }
		fc.mm.Flush()
		mt, mtId, _ = fc.mm.GetMemtable()
		buf, offset, length, _ = mt.GetBufForAppend(uint16(size))
	}
	copy(buf[4:], key)
	copy(buf[4+len(key):], value)
	crc := crc32.ChecksumIEEE(buf[4:])
	indices.ByteOrder.PutUint32(buf[0:4], crc)
	fc.keyIndex.PutV2(key, length, mtId, uint32(offset), exptime)
	fc.dm.IncMemtableKeyCount(mtId)
	fc.Stats.MemIdCount[mtId]++
	return nil
}

func (fc *FileCache) Get(key string) ([]byte, uint64, bool, bool, bool) {
	memId, length, offset, lastAccess, freq, exptime, idx, found := fc.keyIndex.GetMetaV2(key)
	_, mtId, _ := fc.mm.GetMemtable()
	shouldReWrite := fc.predictor.Predict(freq, uint64(lastAccess), memId, mtId)

	if !found {
		fc.Stats.KeyNotFoundCount++
		return nil, 0, false, false, shouldReWrite
	}
	idxs := fmt.Sprintf("%d", idx)
	if !strings.Contains(key, idxs) {
		fc.Stats.BadDataCount++
	}
	if exptime < uint64(time.Now().Unix()) {
		return nil, 0, false, true, shouldReWrite
	}
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
			return nil, 0, false, false, shouldReWrite
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
		return nil, 0, false, false, shouldReWrite
	}
	if gotKey != key {
		fc.Stats.BadKeyCount++
		fc.Stats.BadKeyMemIds[memId]++
		return nil, 0, false, false, shouldReWrite
	}
	valLen := int(length) - 4 - len(key)
	valBuf := make([]byte, valLen)
	copy(valBuf, buf[4+len(key):])
	return valBuf, exptime, true, false, shouldReWrite
}

func (fc *FileCache) readFromDisk(fileOffset int64, length uint16, buf []byte) int {
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

// Debug methods to expose ring buffer state
func (fc *FileCache) GetRingBufferNextIndex() int {
	return fc.keyIndex.GetRingBufferNextIndex()
}

func (fc *FileCache) GetRingBufferSize() int {
	return fc.keyIndex.GetRingBufferSize()
}

func (fc *FileCache) GetRingBufferCapacity() int {
	return fc.keyIndex.GetRingBufferCapacity()
}
