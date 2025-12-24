package filecache

import (
	"sort"
	"sync"
)

type BatchTracker struct {
	mu         sync.RWMutex
	getBatch   []int
	maxSamples int
	getIndex   int
}

// const defaultMaxSamples = 100000

func NewBatchTracker() *BatchTracker {
	return &BatchTracker{
		getBatch:   make([]int, defaultMaxSamples),
		maxSamples: defaultMaxSamples,
	}
}

func (bt *BatchTracker) RecordBatchSize(batchSize int) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.getBatch[bt.getIndex] = batchSize
	bt.getIndex = (bt.getIndex + 1) % bt.maxSamples
}

func (bt *BatchTracker) GetBatchSizePercentiles() (p25, p50, p99 int) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	samples := bt.getIndex
	if samples > int(bt.maxSamples) {
		samples = int(bt.maxSamples)
	}

	if samples == 0 {
		return 0, 0, 0
	}

	batchSizesCopy := make([]int, samples)
	copy(batchSizesCopy, bt.getBatch[:samples])
	sort.Slice(batchSizesCopy, func(i, j int) bool {
		return batchSizesCopy[i] < batchSizesCopy[j]
	})

	p25 = batchSizesCopy[int(float64(samples)*0.25)]
	p50 = batchSizesCopy[int(float64(samples)*0.50)]
	p99 = batchSizesCopy[int(float64(samples)*0.99)]

	return p25, p50, p99
}
