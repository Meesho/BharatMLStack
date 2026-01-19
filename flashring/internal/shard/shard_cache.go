package filecache

import (
	"fmt"
	"hash/crc32"
	"sync"
	"sync/atomic"
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

	//batching reads
	BatchReader *BatchReaderV2

	//Lockless read and write
	ReadCh  chan *ReadRequestV2
	WriteCh chan *WriteRequestV2
}

type Stats struct {
	KeyNotFoundCount atomic.Int64
	KeyExpiredCount  atomic.Int64
	BadDataCount     atomic.Int64
	BadLengthCount   atomic.Int64
	BadCR32Count     atomic.Int64
	BadKeyCount      atomic.Int64
	MemIdCount       sync.Map // key: uint32, value: *atomic.Int64
	LastDeletedMemId atomic.Uint32
	DeletedKeyCount  atomic.Int64
	BadCRCMemIds     sync.Map // key: uint32, value: *atomic.Int64
	BadKeyMemIds     sync.Map // key: uint32, value: *atomic.Int64
	BatchTracker     *BatchTracker
}

// Helper method to increment a counter in a sync.Map
func (s *Stats) incMapCounter(m *sync.Map, key uint32) {
	val, _ := m.LoadOrStore(key, &atomic.Int64{})
	val.(*atomic.Int64).Add(1)
}

// IncMemIdCount atomically increments the counter for the given memId
func (s *Stats) IncMemIdCount(memId uint32) {
	s.incMapCounter(&s.MemIdCount, memId)
}

// IncBadCRCMemIds atomically increments the bad CRC counter for the given memId
func (s *Stats) IncBadCRCMemIds(memId uint32) {
	s.incMapCounter(&s.BadCRCMemIds, memId)
}

// IncBadKeyMemIds atomically increments the bad key counter for the given memId
func (s *Stats) IncBadKeyMemIds(memId uint32) {
	s.incMapCounter(&s.BadKeyMemIds, memId)
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

	//lockless
	EnableLockless bool
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
		Stats: &Stats{
			// sync.Map fields have zero values that are ready to use
			BatchTracker: NewBatchTracker(),
		},
	}

	// Initialize batch reader if enabled
	if config.EnableBatching {
		sc.BatchReader = NewBatchReaderV2(BatchReaderV2Config{
			BatchWindow:  config.BatchWindow,
			MaxBatchSize: config.MaxBatchSize,
		}, sc, sl)
	}

	if config.EnableLockless {

		sc.ReadCh = make(chan *ReadRequestV2, 500)
		sc.WriteCh = make(chan *WriteRequestV2, 500)

		go sc.startReadWriteRoutines()
	}

	return sc
}

// function that starts go routine to process the read and write requests
func (fc *ShardCache) startReadWriteRoutines() {
	go func() {
		for {
			select {
			case writeReq := <-fc.WriteCh: // Writes get priority
				err := fc.Put(writeReq.Key, writeReq.Value, writeReq.ExptimeInMinutes)
				writeReq.Result <- err
			case readReq := <-fc.ReadCh:
				found, data, ttl, expired, shouldRewrite := fc.GetSlowPath(readReq.Key)
				readReq.Result <- ReadResultV2{Found: found, Data: data, TTL: ttl, Expired: expired, ShouldRewrite: shouldRewrite, Error: nil}
			}
		}
	}()
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
	fc.Stats.IncMemIdCount(mtId)
	return nil
}

func (fc *ShardCache) Get(key string) (bool, []byte, uint16, bool, bool) {
	length, lastAccess, remainingTTL, freq, memId, offset, status := fc.keyIndex.Get(key)
	if status == indices.StatusNotFound {
		fc.Stats.KeyNotFoundCount.Add(1)
		return false, nil, 0, false, false
	}

	if status == indices.StatusExpired {
		fc.Stats.KeyExpiredCount.Add(1)
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
		// Allocate buffer of exact size needed - no pool since readFromDisk already copies once
		buf = make([]byte, length)
		fileOffset := uint64(memId)*uint64(fc.mm.Capacity) + uint64(offset)
		n := fc.readFromDisk(int64(fileOffset), length, buf)
		if n != int(length) {
			fc.Stats.BadLengthCount.Add(1)
			return false, nil, 0, false, shouldReWrite
		}
	} else {
		buf, exists = mt.GetBufForRead(int(offset), length)
		if !exists {
			panic("memtable exists but buf not found")
		}
	}
	gotCR32 := indices.ByteOrder.Uint32(buf[0:4])
	computedCR32 := crc32.ChecksumIEEE(buf[4:length])
	gotKey := string(buf[4 : 4+len(key)])
	if gotCR32 != computedCR32 {
		fc.Stats.BadCR32Count.Add(1)
		fc.Stats.IncBadCRCMemIds(memId)
		return false, nil, 0, false, shouldReWrite
	}
	if gotKey != key {
		fc.Stats.BadKeyCount.Add(1)
		fc.Stats.IncBadKeyMemIds(memId)
		return false, nil, 0, false, shouldReWrite
	}
	valLen := int(length) - 4 - len(key)
	return true, buf[4+len(key) : 4+len(key)+valLen], remainingTTL, false, shouldReWrite
}

// GetFastPath attempts to read from memtable only (no disk I/O).
// Returns: (found, data, ttl, expired, needsSlowPath)
// If needsSlowPath is true, caller should use GetSlowPath for disk read.
func (fc *ShardCache) GetFastPath(key string) (bool, []byte, uint16, bool, bool) {
	length, lastAccess, remainingTTL, freq, memId, offset, status := fc.keyIndex.Get(key)
	if status == indices.StatusNotFound {
		fc.Stats.KeyNotFoundCount.Add(1)
		return false, nil, 0, false, false // needsSlowPath = false (not found)
	}

	if status == indices.StatusExpired {
		fc.Stats.KeyExpiredCount.Add(1)
		return false, nil, 0, true, false // needsSlowPath = false (expired)
	}

	// Check if data is in memtable
	mt := fc.mm.GetMemtableById(memId)
	if mt == nil {
		// Data not in memtable, needs disk read - signal slow path needed
		return false, nil, remainingTTL, false, true // needsSlowPath = true
	}

	// Fast path: read from memtable
	buf, exists := mt.GetBufForRead(int(offset), length)
	if !exists {
		panic("memtable exists but buf not found")
	}

	// Validate CRC and key
	gotCR32 := indices.ByteOrder.Uint32(buf[0:4])
	computedCR32 := crc32.ChecksumIEEE(buf[4:])
	if gotCR32 != computedCR32 {
		fc.Stats.BadCR32Count.Add(1)
		fc.Stats.IncBadCRCMemIds(memId)
		_, currMemId, _ := fc.mm.GetMemtable()
		shouldReWrite := fc.predictor.Predict(uint64(freq), uint64(lastAccess), memId, currMemId)
		_ = shouldReWrite // Not returning shouldReWrite in fast path for simplicity
		return false, nil, 0, false, false
	}

	gotKey := string(buf[4 : 4+len(key)])
	if gotKey != key {
		fc.Stats.BadKeyCount.Add(1)
		fc.Stats.IncBadKeyMemIds(memId)
		return false, nil, 0, false, false
	}

	valLen := int(length) - 4 - len(key)
	return true, buf[4+len(key) : 4+len(key)+valLen], remainingTTL, false, false // needsSlowPath = false
}

// GetSlowPath reads data from disk. Used when GetFastPath indicates needsSlowPath.
// Returns: (found, data, ttl, expired, shouldRewrite)
func (fc *ShardCache) GetSlowPath(key string) (bool, []byte, uint16, bool, bool) {
	length, lastAccess, remainingTTL, freq, memId, offset, status := fc.keyIndex.Get(key)
	if status == indices.StatusNotFound {
		fc.Stats.KeyNotFoundCount.Add(1)
		return false, nil, 0, false, false
	}

	if status == indices.StatusExpired {
		fc.Stats.KeyExpiredCount.Add(1)
		return false, nil, 0, true, false
	}

	_, currMemId, _ := fc.mm.GetMemtable()
	shouldReWrite := fc.predictor.Predict(uint64(freq), uint64(lastAccess), memId, currMemId)

	// Check memtable again (might have changed since fast path check)
	mt := fc.mm.GetMemtableById(memId)
	if mt != nil {
		// Data is now in memtable, use fast path logic
		buf, exists := mt.GetBufForRead(int(offset), length)
		if !exists {
			panic("memtable exists but buf not found")
		}
		return fc.validateAndReturnBuffer(key, buf, length, memId, remainingTTL, shouldReWrite)
	}

	// Read from disk - allocate buffer of exact size needed (no pool since readFromDisk already copies once)
	buf := make([]byte, length)
	fileOffset := uint64(memId)*uint64(fc.mm.Capacity) + uint64(offset)
	n := fc.readFromDisk(int64(fileOffset), length, buf)
	if n != int(length) {
		fc.Stats.BadLengthCount.Add(1)
		return false, nil, 0, false, shouldReWrite
	}

	return fc.validateAndReturnBuffer(key, buf, length, memId, remainingTTL, shouldReWrite)
}

// validateAndReturnBuffer validates CRC and key, then returns the value
func (fc *ShardCache) validateAndReturnBuffer(key string, buf []byte, length uint16, memId uint32, remainingTTL uint16, shouldReWrite bool) (bool, []byte, uint16, bool, bool) {
	gotCR32 := indices.ByteOrder.Uint32(buf[0:4])
	computedCR32 := crc32.ChecksumIEEE(buf[4:length])
	if gotCR32 != computedCR32 {
		fc.Stats.BadCR32Count.Add(1)
		fc.Stats.IncBadCRCMemIds(memId)
		return false, nil, 0, false, shouldReWrite
	}

	gotKey := string(buf[4 : 4+len(key)])
	if gotKey != key {
		fc.Stats.BadKeyCount.Add(1)
		fc.Stats.IncBadKeyMemIds(memId)
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

// batching reads
func (fc *ShardCache) processBuffer(key string, buf []byte, length uint16) ReadResult {
	gotCR32 := indices.ByteOrder.Uint32(buf[0:4])
	computedCR32 := crc32.ChecksumIEEE(buf[4:])
	gotKey := string(buf[4 : 4+len(key)])

	if gotCR32 != computedCR32 {
		fc.Stats.BadCR32Count.Add(1)
		return ReadResult{Found: false, Error: fmt.Errorf("crc mismatch")}
	}
	if gotKey != key {
		fc.Stats.BadKeyCount.Add(1)
		return ReadResult{Found: false, Error: fmt.Errorf("key mismatch")}
	}

	valLen := int(length) - 4 - len(key)
	value := make([]byte, valLen)
	copy(value, buf[4+len(key):4+len(key)+valLen])

	return ReadResult{
		Found: true,
		Data:  value,
	}
}
