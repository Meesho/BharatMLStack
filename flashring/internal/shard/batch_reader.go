package filecache

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// ===========batching reads ==========
// ReadRequest represents a single read request
type ReadRequest struct {
	Key    string
	Length uint16
	MemId  uint32
	Offset uint32
	Result chan ReadResult
}

// ReadResult contains the response for a read request
type ReadResult struct {
	Found         bool
	Data          []byte
	TTL           uint16
	Expired       bool
	ShouldRewrite bool
	Error         error
}

// BatchReader handles batching of disk reads
type BatchReader struct {
	requests     chan *ReadRequest
	batchWindow  time.Duration
	maxBatchSize int
	shardCache   *ShardCache
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

// Config for BatchReader
type BatchReaderConfig struct {
	BatchWindow  time.Duration // e.g., 5-10μs
	MaxBatchSize int           // e.g., 32-64 requests
}

func NewBatchReader(config BatchReaderConfig, sc *ShardCache) *BatchReader {
	br := &BatchReader{
		requests:     make(chan *ReadRequest, config.MaxBatchSize*2),
		batchWindow:  config.BatchWindow,
		maxBatchSize: config.MaxBatchSize,
		shardCache:   sc,
		stopCh:       make(chan struct{}),
	}

	// Start batch processor goroutine
	br.wg.Add(1)
	go br.processBatches()

	return br
}

func (br *BatchReader) processBatches() {
	defer br.wg.Done()

	for {
		select {
		case <-br.stopCh:
			return
		case firstReq := <-br.requests:
			batch := br.collectBatch(firstReq)
			br.executeBatch(batch)
		}
	}
}

func (br *BatchReader) collectBatch(firstReq *ReadRequest) []*ReadRequest {
	batch := make([]*ReadRequest, 0, br.maxBatchSize)
	batch = append(batch, firstReq)

	deadline := time.Now().Add(br.batchWindow)

	for len(batch) < br.maxBatchSize && time.Now().Before(deadline) {
		select {
		case req := <-br.requests:
			batch = append(batch, req)
		default:
			// No more requests immediately available
			if len(batch) >= 8 { // Have enough, execute now
				return batch
			}
			// Small wait for more requests (100ns)
			time.Sleep(50 * time.Nanosecond)
		}
	}

	return batch
}

func (br *BatchReader) executeBatch(batch []*ReadRequest) {
	// Separate memtable hits from disk reads
	diskReads := make([]*ReadRequest, 0, len(batch))

	for _, req := range batch {
		mt := br.shardCache.mm.GetMemtableById(req.MemId)
		if mt != nil {
			// Fast path: memtable hit
			buf, exists := mt.GetBufForRead(int(req.Offset), req.Length)
			if exists {
				result := br.shardCache.processBuffer(req.Key, buf, req.Length)
				req.Result <- result
				continue
			}
		}
		// Needs disk read
		diskReads = append(diskReads, req)
	}

	if len(diskReads) == 0 {
		return
	}

	// Sort disk reads by file offset
	sort.Slice(diskReads, func(i, j int) bool {
		offsetI := uint64(diskReads[i].MemId)*uint64(br.shardCache.mm.Capacity) +
			uint64(diskReads[i].Offset)
		offsetJ := uint64(diskReads[j].MemId)*uint64(br.shardCache.mm.Capacity) +
			uint64(diskReads[j].Offset)
		return offsetI < offsetJ
	})

	// Execute disk reads (could be parallelized or merged here)
	for _, req := range diskReads {
		result := br.executeReadFromDisk(req)
		req.Result <- result
	}
}

func (br *BatchReader) executeReadFromDisk(req *ReadRequest) ReadResult {
	buf := make([]byte, req.Length)
	fileOffset := uint64(req.MemId)*uint64(br.shardCache.mm.Capacity) +
		uint64(req.Offset)

	n := br.shardCache.readFromDisk(int64(fileOffset), req.Length, buf)
	if n != int(req.Length) {
		return ReadResult{Error: fmt.Errorf("bad read length")}
	}

	return br.shardCache.processBuffer(req.Key, buf, req.Length)
}

func (br *BatchReader) Close() {
	close(br.stopCh)
	br.wg.Wait()
}
