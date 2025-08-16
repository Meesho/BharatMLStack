package filecache

import (
	"fmt"
	"hash/crc32"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/external/allocators"
	"github.com/Meesho/BharatMLStack/flashring/external/fs"
	"github.com/Meesho/BharatMLStack/flashring/external/indices"
	"github.com/Meesho/BharatMLStack/flashring/external/maths"
	"github.com/Meesho/BharatMLStack/flashring/external/memtables"
	"github.com/rs/zerolog/log"
)

type diskReq struct {
	fileOffset int64
	length     uint16
	// completion is sent to ShardCache.DiskDone for this shard
	cookie any // carries original request context back to the worker
}

type DiskResp struct {
	Cookie any
	N      int
	Buf    []byte // slice into the aligned page (start..start+length)
	Put    func() // MUST call when done to return the page to allocator
	Err    error
	dc     diskReq
}

type asyncReader struct {
	fc      *AsyncShardCache
	q       chan diskReq
	workers int
}

// Export a per-shard completion channel so the pinned worker can select on it.
func (fc *AsyncShardCache) DiskDone() <-chan DiskResp { return fc.diskDone }

func (fc *AsyncShardCache) initAsyncReader(workers, qDepth int) {
	if workers <= 0 {
		workers = 32
	}
	if qDepth <= 0 {
		qDepth = 4096
	}
	fc.ar = &asyncReader{
		fc:      fc,
		q:       make(chan diskReq, qDepth),
		workers: workers,
	}
	fc.diskDone = make(chan DiskResp, qDepth)
	for i := 0; i < workers; i++ {
		go fc.ar.worker()
	}
}

func (ar *asyncReader) worker() {
	fc := ar.fc
	for r := range ar.q {
		// Align just like readFromDisk()
		buf := make([]byte, r.length)
		alignedStartOffset := (r.fileOffset / fs.BLOCK_SIZE) * fs.BLOCK_SIZE
		endndOffset := r.fileOffset + int64(r.length)
		endAlignedOffset := ((endndOffset + fs.BLOCK_SIZE - 1) / fs.BLOCK_SIZE) * fs.BLOCK_SIZE
		alignedReadSize := endAlignedOffset - alignedStartOffset
		page := fc.readPageAllocator.Get(int(alignedReadSize))
		fc.file.Pread(alignedStartOffset, page.Buf)
		start := int(r.fileOffset - alignedStartOffset)
		n := copy(buf, page.Buf[start:start+int(r.length)])
		fc.readPageAllocator.Put(page)
		fc.diskDone <- DiskResp{
			Cookie: r.cookie,
			N:      n,
			Buf:    buf,
			Err:    nil,
			Put:    func() { fc.readPageAllocator.Put(page) },
			dc:     r,
		}
	}
}

// Submit a read (non-blocking). cookie comes back in DiskResp to identify the request.
func (fc *AsyncShardCache) ReadAtAsync(fileOffset int64, length uint16, cookie any) {
	fc.ar.q <- diskReq{fileOffset: fileOffset, length: length, cookie: cookie}
}

// --- add to ShardCache struct and constructor ---

type AsyncShardCache struct {
	keyIndex          *indices.KeyIndex
	file              *fs.WrapAppendFile
	mm                *memtables.MemtableManager
	readPageAllocator *allocators.SlabAlignedPageAllocator
	dm                *indices.DeleteManager
	predictor         *maths.Predictor
	startAt           int64
	ar                *asyncReader
	diskDone          chan DiskResp
	Stats             *Stats
}

func NewAsyncShardCache(config ShardCacheConfig) *AsyncShardCache {
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
	fc := &AsyncShardCache{
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
	fc.initAsyncReader(config.AsyncReadWorkers, config.AsyncQueueDepth)
	return fc
}

func (fc *AsyncShardCache) Put(key string, value []byte, exptime uint64) error {
	deltaExptime := exptime - uint64(fc.startAt)
	deltaExptimeInMin := deltaExptime / 60
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
	fc.keyIndex.Put(key, length, mtId, uint32(offset), deltaExptimeInMin)
	fc.dm.IncMemtableKeyCount(mtId)
	fc.Stats.MemIdCount[mtId]++
	return nil
}

func (fc *AsyncShardCache) Get(key string) ([]byte, uint64, bool, bool, bool) {
	memId, length, offset, lastAccess, freq, exptime, idx, found := fc.keyIndex.Get(key)
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
	deltaCurTimeFromStart := deltaCurrTimeFromStartInMin(fc.startAt)
	if exptime < deltaCurTimeFromStart {
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

func (fc *AsyncShardCache) readFromDisk(fileOffset int64, length uint16, buf []byte) int {
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
func (fc *AsyncShardCache) GetRingBufferNextIndex() int {
	return fc.keyIndex.GetRingBufferNextIndex()
}

func (fc *AsyncShardCache) GetRingBufferSize() int {
	return fc.keyIndex.GetRingBufferSize()
}

func (fc *AsyncShardCache) GetRingBufferCapacity() int {
	return fc.keyIndex.GetRingBufferCapacity()
}

func (fc *AsyncShardCache) GetRingBufferActiveEntries() int {
	return fc.keyIndex.GetRingBufferActiveEntries()
}

func (fc *AsyncShardCache) GetKeyIndex() *indices.KeyIndex                 { return fc.keyIndex }
func (fc *AsyncShardCache) GetMemtableManager() *memtables.MemtableManager { return fc.mm }
func (fc *AsyncShardCache) Predictor() *maths.Predictor                    { return fc.predictor }
func (fc *AsyncShardCache) GetStartAt() int64                              { return fc.startAt }
