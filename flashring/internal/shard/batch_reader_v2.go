package filecache

import (
	"fmt"
	"sync"
	"time"
)

type ReadRequestV2 struct {
	Key    string
	Result chan ReadResultV2
}

type ReadResultV2 struct {
	Found         bool
	Data          []byte
	TTL           uint16
	Expired       bool
	ShouldRewrite bool
	Error         error
}

type WriteRequestV2 struct {
	Key              string
	Value            []byte
	ExptimeInMinutes uint16
	Result           chan error
}

type BatchReaderV2 struct {
	Requests     chan *ReadRequestV2
	batchWindow  time.Duration
	maxBatchSize int
	shardCache   *ShardCache
	stopCh       chan struct{}
	wg           sync.WaitGroup
	shardLock    *sync.RWMutex
}

type BatchReaderV2Config struct {
	BatchWindow  time.Duration
	MaxBatchSize int
}

var ReadRequestPool = sync.Pool{
	New: func() interface{} {
		return &ReadRequestV2{}
	},
}

var ReadResultPool = sync.Pool{
	New: func() interface{} {
		return make(chan ReadResultV2, 1)
	},
}

var ErrorPool = sync.Pool{
	New: func() interface{} {
		return make(chan error, 1)
	},
}

var BufPool = sync.Pool{
	New: func() interface{} {
		// Allocate max expected size - use pointer to avoid allocation on Put
		buf := make([]byte, 4096)
		return &buf
	},
}

func NewBatchReaderV2(config BatchReaderV2Config, sc *ShardCache, sl *sync.RWMutex) *BatchReaderV2 {
	br := &BatchReaderV2{
		Requests:     make(chan *ReadRequestV2, config.MaxBatchSize*2),
		batchWindow:  config.BatchWindow,
		maxBatchSize: config.MaxBatchSize,
		shardCache:   sc,
		stopCh:       make(chan struct{}),
		shardLock:    sl,
	}

	// Start batch processor goroutine
	br.wg.Add(1)
	go br.processBatchesV2()

	return br
}

func (br *BatchReaderV2) processBatchesV2() {
	defer br.wg.Done()

	for {
		select {
		case <-br.stopCh:
			return
		case firstReq := <-br.Requests:
			batch := br.collectBatchV2(firstReq)
			br.shardCache.Stats.BatchTracker.RecordBatchSize(len(batch))
			br.executeBatchV2(batch)
		}
	}
}

func (br *BatchReaderV2) collectBatchV2(firstReq *ReadRequestV2) []*ReadRequestV2 {
	batch := make([]*ReadRequestV2, 0, br.maxBatchSize)
	batch = append(batch, firstReq)

	timer := time.NewTimer(br.batchWindow)

	for len(batch) < br.maxBatchSize {
		select {
		case req := <-br.Requests:
			batch = append(batch, req)
		case <-timer.C:
			return batch
		}
	}

	return batch
}

func (br *BatchReaderV2) executeBatchV2(batch []*ReadRequestV2) {
	br.shardLock.RLock()
	defer br.shardLock.RUnlock()
	for _, req := range batch {
		found, data, ttl, expired, shouldRewrite := br.shardCache.Get(req.Key)
		if !found {
			req.Result <- ReadResultV2{Error: fmt.Errorf("key not found")}
		} else {
			req.Result <- ReadResultV2{Found: found, Data: data, TTL: ttl, Expired: expired, ShouldRewrite: shouldRewrite}
		}
	}
}
