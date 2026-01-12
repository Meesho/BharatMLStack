package metrics

import (
	"sync"
	"time"
)

// MetricAverager maintains running averages for a metric
type MetricAverager struct {
	mu        sync.RWMutex
	sum       float64
	count     int64
	lastValue float64
}

func (ma *MetricAverager) Add(value float64) {
	if value == 0 {
		return // Ignore zero values
	}
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.sum += value
	ma.count++
	ma.lastValue = value
}

func (ma *MetricAverager) AddDuration(value time.Duration) {
	if value == 0 {
		return // Ignore zero values
	}
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.sum += float64(value)
	ma.count++
}

func (ma *MetricAverager) Average() float64 {
	ma.mu.RLock()
	defer ma.mu.RUnlock()
	if ma.count == 0 {
		return 0
	}
	return ma.sum / float64(ma.count)
}

func (ma *MetricAverager) Latest() float64 {
	ma.mu.RLock()
	defer ma.mu.RUnlock()
	return ma.lastValue
}

func (ma *MetricAverager) Reset() {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.sum = 0
	ma.count = 0
}
