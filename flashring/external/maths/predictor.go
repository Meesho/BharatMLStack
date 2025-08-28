package maths

import "time"

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

func (p *Predictor) Predict(freq uint64, lastAccess uint64, keyMemId uint32, activeMemId uint32) bool {
	score := p.Estimator.CalculateRewriteScore(freq, lastAccess, keyMemId, activeMemId, p.MaxMemTableCount)
	return score > p.ReWriteScoreThreshold
}

func (p *Predictor) Observe(hitRate float64) {
	select {
	case p.hitRateCh <- hitRate:
	default:
	}
}
