package maths

import (
	"time"

	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
)

type Params struct {
	Freq        uint64
	LastAccess  uint64
	KeyMemId    uint32
	ActiveMemId uint32
}
type Predictor struct {
	Estimator             *Estimator
	GridSearchEstimator   *GridSearchEstimator
	ReWriteScoreThreshold float32
	MaxMemTableCount      uint32
	hitRateCh             chan float64
}

type PredictorConfig struct {
	ReWriteScoreThreshold float32
	Weights               []WeightTuple
	SampleDuration        time.Duration
	MaxMemTableCount      uint32
	GridSearchEpsilon     float64
}

func NewPredictor(config PredictorConfig) *Predictor {
	estimator := &Estimator{
		WFreq: config.Weights[0].WFreq,
		WLA:   config.Weights[0].WLA,
	}
	gridSearchEstimator := NewGridSearchEstimator(config.SampleDuration, config.Weights, estimator, config.GridSearchEpsilon)
	p := &Predictor{
		Estimator:             estimator,
		GridSearchEstimator:   gridSearchEstimator,
		ReWriteScoreThreshold: config.ReWriteScoreThreshold,
		MaxMemTableCount:      config.MaxMemTableCount,
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

func freqBand(freq uint64) string {
	switch {
	case freq <= 1:
		return "cold"
	case freq <= 5:
		return "warm"
	case freq <= 20:
		return "hot"
	default:
		return "very_hot"
	}
}

func (p *Predictor) Predict(freq uint64, lastAccess uint64, keyMemId uint32, activeMemId uint32) bool {
	score := p.Estimator.CalculateRewriteScore(freq, lastAccess, keyMemId, activeMemId, p.MaxMemTableCount)
	rewrite := score > p.ReWriteScoreThreshold

	zone := ringZone(keyMemId, activeMemId, p.MaxMemTableCount)
	band := freqBand(freq)
	decision := "skip"
	if rewrite {
		decision = "rewrite"
	}

	metrics.Incr(metrics.KEY_REWRITE_SCORE, metrics.BuildTag(metrics.NewTag(metrics.TAG_SCORE_BUCKET, scoreBucket(score))))
	metrics.Incr(metrics.KEY_REWRITE_DECISION, metrics.BuildTag(
		metrics.NewTag(metrics.TAG_DECISION, decision),
		metrics.NewTag(metrics.TAG_RING_ZONE, zone),
		metrics.NewTag(metrics.TAG_FREQ_BAND, band),
	))

	return rewrite
}

func (p *Predictor) Observe(hitRate float64) {
	select {
	case p.hitRateCh <- hitRate:
	default:
	}
}
