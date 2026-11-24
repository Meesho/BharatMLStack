package maths

import (
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
)

type Params struct {
	Freq        uint64
	LastAccess  uint64
	KeyMemId    uint32
	ActiveMemId uint32
}

// Predictor decides whether a key read should trigger an asynchronous
// rewrite that copies the value back to the active memtable, moving hot
// data away from the punch-hole horizon before its memtable is recycled.
// The score combines frequency, recency, and over-write risk (how close
// the key's memtable is to being overwritten by the active writer). The
// (wFreq, wLA) weights driving the score are tuned online by an embedded
// GridSearchEstimator that consumes hit-rate samples via Observe.
type Predictor struct {
	Estimator             *Estimator
	GridSearchEstimator   *GridSearchEstimator
	ReWriteScoreThreshold float32
	MaxMemTableCount      uint32
	freqBands             FreqBands
	recencyBands          RecencyBands
	hitRateCh             chan float64
	closeOnce             sync.Once
}

// FreqBands defines the upper bounds for frequency band labels.
// Keys with freq <= Cold are "cold", <= Warm are "warm", <= Hot are "hot",
// and anything above Hot is "very_hot".
type FreqBands struct {
	Cold uint64
	Warm uint64
	Hot  uint64
}

// RecencyBands defines upper-bound thresholds for recency band labels.
// lastAccess represents how long ago a key was accessed (higher = older).
// Keys with lastAccess <= Hot are "very_hot", <= Warm are "hot",
// <= Cold are "warm", and anything above Cold is "cold".
type RecencyBands struct {
	Hot  uint64
	Warm uint64
	Cold uint64
}

type PredictorConfig struct {
	ReWriteScoreThreshold float32
	Weights               []WeightTuple
	SampleDuration        time.Duration
	MaxMemTableCount      uint32
	GridSearchEpsilon     float64
	FreqBands             FreqBands
	RecencyBands          RecencyBands
}

// NewPredictor constructs a Predictor and starts a background goroutine
// that drains hitRateCh into the GridSearchEstimator. Call Close to stop
// the goroutine and release the channel; long-lived callers (WrapCache)
// close as part of their own teardown. Default FreqBands ({1,5,20}) and
// RecencyBands ({5,50,500}) are applied if config leaves them all zero.
func NewPredictor(config PredictorConfig) *Predictor {
	estimator := &Estimator{
		WFreq: config.Weights[0].WFreq,
		WLA:   config.Weights[0].WLA,
	}
	gridSearchEstimator := NewGridSearchEstimator(config.SampleDuration, config.Weights, estimator, config.GridSearchEpsilon)
	fb := config.FreqBands
	if fb.Cold == 0 && fb.Warm == 0 && fb.Hot == 0 {
		fb = FreqBands{Cold: 1, Warm: 5, Hot: 20}
	}
	rb := config.RecencyBands
	if rb.Hot == 0 && rb.Warm == 0 && rb.Cold == 0 {
		rb = RecencyBands{Hot: 5, Warm: 50, Cold: 500}
	}
	p := &Predictor{
		Estimator:             estimator,
		GridSearchEstimator:   gridSearchEstimator,
		ReWriteScoreThreshold: config.ReWriteScoreThreshold,
		MaxMemTableCount:      config.MaxMemTableCount,
		freqBands:             fb,
		recencyBands:          rb,
		hitRateCh:             make(chan float64, 1024),
	}
	go func() {
		for hitRate := range p.hitRateCh {
			p.GridSearchEstimator.RecordHitRate(hitRate)
		}
	}()
	return p
}

func scoreBucket(score float32) string {
	switch {
	case score < 0.1:
		return "0.0-0.1"
	case score < 0.3:
		return "0.1-0.3"
	case score < 0.5:
		return "0.3-0.5"
	case score < 0.7:
		return "0.5-0.7"
	case score < 1.0:
		return "0.7-1.0"
	default:
		return "1.0+"
	}
}

func ringZone(keyMemId, activeMemId, maxMemTableCount uint32) string {
	// Same guard as Estimator.CalculateRewriteScore — modulo-by-zero
	// panics on int, and the metrics path shouldn't take down a Get.
	if maxMemTableCount == 0 {
		return "unknown"
	}
	risk := (activeMemId - keyMemId + maxMemTableCount) % maxMemTableCount
	pct := float64(risk) / float64(maxMemTableCount)
	switch {
	case pct < 0.25:
		return "0-25%"
	case pct < 0.50:
		return "25-50%"
	case pct < 0.75:
		return "50-75%"
	default:
		return "75-100%"
	}
}

func freqBand(freq uint64, fb FreqBands) string {
	switch {
	case freq <= fb.Cold:
		return "cold"
	case freq <= fb.Warm:
		return "warm"
	case freq <= fb.Hot:
		return "hot"
	default:
		return "very_hot"
	}
}

func recencyBand(lastAccess uint64, rb RecencyBands) string {
	switch {
	case lastAccess <= rb.Hot:
		return "very_hot"
	case lastAccess <= rb.Warm:
		return "hot"
	case lastAccess <= rb.Cold:
		return "warm"
	default:
		return "cold"
	}
}

func (p *Predictor) Predict(freq uint64, lastAccess uint64, keyMemId uint32, activeMemId uint32) bool {
	score := p.Estimator.CalculateRewriteScore(freq, lastAccess, keyMemId, activeMemId, p.MaxMemTableCount)
	rewrite := score > p.ReWriteScoreThreshold

	// Checked here, not only inside the metric calls: computeMetrics builds
	// tag slices before any of them can no-op.
	if metrics.Enabled() && metrics.PredictorMetricsEnabled() {
		computeMetrics(keyMemId, activeMemId, p, freq, lastAccess, rewrite, score)
	}

	return rewrite
}

// computeMetrics emits the rewrite-predictor telemetry group. KEY_REWRITE_DECISION
// carries a 4-tag cross-product (decision × ring_zone × freq_band × recency_band),
// giving up to 2×4×4×4 = 128 series per host and firing on every Predict call —
// the highest-cardinality hot-path emission in flashring. Consumers on tight
// cardinality budgets can disable the whole group via
// FLASHRING_METRICS_PREDICTOR_ENABLED=false; see metrics.PredictorMetricsEnabled.
func computeMetrics(keyMemId uint32, activeMemId uint32, p *Predictor, freq uint64, lastAccess uint64, rewrite bool, score float32) {
	zone := ringZone(keyMemId, activeMemId, p.MaxMemTableCount)
	fBand := freqBand(freq, p.freqBands)
	rBand := recencyBand(lastAccess, p.recencyBands)
	decision := "skip"
	if rewrite {
		decision = "rewrite"
	}

	metrics.Timing(metrics.KEY_ACCESS_FREQ, time.Duration(freq)*time.Millisecond, nil)
	metrics.Timing(metrics.KEY_LAST_ACCESS, time.Duration(lastAccess)*time.Millisecond, nil)
	metrics.Incr(metrics.KEY_REWRITE_SCORE, metrics.BuildTag(metrics.NewTag(metrics.TAG_SCORE_BUCKET, scoreBucket(score))))
	metrics.Incr(metrics.KEY_REWRITE_DECISION, metrics.BuildTag(
		metrics.NewTag(metrics.TAG_DECISION, decision),
		metrics.NewTag(metrics.TAG_RING_ZONE, zone),
		metrics.NewTag(metrics.TAG_FREQ_BAND, fBand),
		metrics.NewTag(metrics.TAG_RECENCY_BAND, rBand),
	))
}

// Observe delivers a hit-rate sample to the grid-search loop. The send
// is non-blocking — if the channel buffer is full (1024 samples), the
// sample is silently dropped rather than back-pressuring the caller. In
// steady state this is acceptable because the grid search converges on
// time-windowed averages, but a stuck consumer goroutine would cause all
// samples to be dropped.
func (p *Predictor) Observe(hitRate float64) {
	select {
	case p.hitRateCh <- hitRate:
	default:
	}
}

// Close stops the grid-search background goroutine by closing hitRateCh.
// The goroutine exits when its `for range p.hitRateCh` loop drains the
// closed channel. Idempotent — safe to call multiple times via the
// closeOnce guard. Callers must not Observe after Close (send-on-closed
// panics); WrapCache.Close serialises this correctly by tearing down
// the predictor after all shard writers/readers have stopped.
func (p *Predictor) Close() {
	p.closeOnce.Do(func() {
		close(p.hitRateCh)
	})
}
