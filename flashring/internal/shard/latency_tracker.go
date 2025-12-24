package filecache

import (
	"sort"
	"sync"
	"time"
)

type LatencyTracker struct {
	mu           sync.RWMutex
	getLatencies []time.Duration
	putLatencies []time.Duration
	maxSamples   int
	getIndex     int
	putIndex     int
	getCount     int64
	putCount     int64
}

const defaultMaxSamples = 100000

func NewLatencyTracker() *LatencyTracker {
	return &LatencyTracker{
		getLatencies: make([]time.Duration, defaultMaxSamples),
		putLatencies: make([]time.Duration, defaultMaxSamples),
		maxSamples:   defaultMaxSamples,
	}
}

func (lt *LatencyTracker) RecordGet(duration time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	lt.getLatencies[lt.getIndex] = duration
	lt.getIndex = (lt.getIndex + 1) % lt.maxSamples
	lt.getCount++
}

func (lt *LatencyTracker) RecordPut(duration time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	lt.putLatencies[lt.putIndex] = duration
	lt.putIndex = (lt.putIndex + 1) % lt.maxSamples
	lt.putCount++
}

func (lt *LatencyTracker) GetLatencyPercentiles() (p25, p50, p99 time.Duration) {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	samples := lt.getCount
	if samples > int64(lt.maxSamples) {
		samples = int64(lt.maxSamples)
	}

	if samples == 0 {
		return 0, 0, 0
	}

	latenciesCopy := make([]time.Duration, samples)
	copy(latenciesCopy, lt.getLatencies[:samples])
	sort.Slice(latenciesCopy, func(i, j int) bool {
		return latenciesCopy[i] < latenciesCopy[j]
	})

	p25 = latenciesCopy[int(float64(samples)*0.25)]
	p50 = latenciesCopy[int(float64(samples)*0.50)]
	p99 = latenciesCopy[int(float64(samples)*0.99)]

	return p25, p50, p99
}

func (lt *LatencyTracker) PutLatencyPercentiles() (p25, p50, p99 time.Duration) {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	samples := lt.putCount
	if samples > int64(lt.maxSamples) {
		samples = int64(lt.maxSamples)
	}

	if samples == 0 {
		return 0, 0, 0
	}

	latenciesCopy := make([]time.Duration, samples)
	copy(latenciesCopy, lt.putLatencies[:samples])
	sort.Slice(latenciesCopy, func(i, j int) bool {
		return latenciesCopy[i] < latenciesCopy[j]
	})

	p25 = latenciesCopy[int(float64(samples)*0.25)]
	p50 = latenciesCopy[int(float64(samples)*0.50)]
	p99 = latenciesCopy[int(float64(samples)*0.99)]

	return p25, p50, p99
}
