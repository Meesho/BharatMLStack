// Package estimator implements online adaptive grid search for tuning
// weights (wFreq, wLA) to optimize cache rewrite decisions based on hit ratio.
package maths

import (
	"math"
	"time"
)

const (
	missBaseline = float64(1e-9)
)

type WeightTuple struct {
	WFreq float64
	WLA   float64
}

type Stats struct {
	HitRate float64 // averaged hit rate over time window
	Trials  int
}

type GridSearchEstimator struct {
	Tuples         []WeightTuple
	InitialTuples  []WeightTuple
	bestTuple      WeightTuple
	TupleStats     map[WeightTuple]*Stats
	CurrIndex      int
	StartTime      time.Time
	Duration       time.Duration
	LiveEstimator  *Estimator
	stopGridSearch bool
	bestHitRate    float64
	epsilon        float64
}

type Estimator struct {
	WFreq float64
	WLA   float64
}

func NewGridSearchEstimator(duration time.Duration, initialTuples []WeightTuple, estimator *Estimator, epsilon float64) *GridSearchEstimator {
	return &GridSearchEstimator{
		Tuples:         initialTuples,
		InitialTuples:  initialTuples,
		bestTuple:      initialTuples[0],
		TupleStats:     make(map[WeightTuple]*Stats),
		CurrIndex:      0,
		StartTime:      time.Now(),
		Duration:       duration,
		LiveEstimator:  estimator,
		bestHitRate:    0,
		stopGridSearch: false,
		epsilon:        epsilon,
	}
}

func (e *Estimator) CalculateRewriteScore(freq uint64, lastAccess uint64, keyMemId, activeMemId, maxMemTableCount uint32) float32 {
	overWriteRisk := (activeMemId - keyMemId + maxMemTableCount) % maxMemTableCount
	overWriteRiskScore := float32(overWriteRisk) / float32(maxMemTableCount)

	fScore := 1 - math.Exp(-e.WFreq*float64(freq))
	laScore := math.Exp(-e.WLA * float64(lastAccess))
	return float32(fScore+laScore) * overWriteRiskScore
}

func (g *GridSearchEstimator) RecordHitRate(hitRate float64) {
	if g.stopGridSearch {
		tuple := g.bestTuple
		if _, ok := g.TupleStats[tuple]; !ok {
			g.TupleStats[tuple] = &Stats{}
		}
		stat := g.TupleStats[tuple]
		stat.HitRate = (stat.HitRate*float64(stat.Trials) + hitRate) / float64(stat.Trials+1)
		stat.Trials++
		if stat.HitRate < g.bestHitRate*0.9 {
			g.RestartGridSearch()
		}
		return
	}
	tuple := g.Tuples[g.CurrIndex]
	if _, ok := g.TupleStats[tuple]; !ok {
		g.TupleStats[tuple] = &Stats{}
	}
	stat := g.TupleStats[tuple]
	stat.HitRate = (stat.HitRate*float64(stat.Trials) + hitRate) / float64(stat.Trials+1)
	stat.Trials++

	if time.Since(g.StartTime) < g.Duration {
		return
	}
	// Advance to next tuple
	g.CurrIndex = (g.CurrIndex + 1) % len(g.Tuples)
	if g.CurrIndex == 0 {
		ok := g.RefineGridAroundBest(2, 0.001)
		if !ok {
			g.stopGridSearch = true
			return
		}
	}
	g.StartTime = time.Now()

	// Update live estimator
	next := g.Tuples[g.CurrIndex]
	g.LiveEstimator.WFreq = next.WFreq
	g.LiveEstimator.WLA = next.WLA
}

func (g *GridSearchEstimator) BestTuple() WeightTuple {

	best := WeightTuple{}
	bestScore := -1.0

	for _, tup := range g.Tuples {
		stat := g.TupleStats[tup]
		if stat == nil || stat.Trials < 3 {
			continue
		}
		if stat.HitRate > bestScore {
			bestScore = stat.HitRate
			best = tup
		}
	}

	return best
}

func (g *GridSearchEstimator) GenerateRefinedGrid(base WeightTuple, steps int, delta float64) ([]WeightTuple, bool) {
	refined := make([]WeightTuple, 0, (2*steps+1)*(2*steps+1))
	for i := -steps; i <= steps; i++ {
		for j := -steps; j <= steps; j++ {
			wf := base.WFreq + float64(i)*delta
			la := base.WLA + float64(j)*delta
			if math.Abs(wf-base.WFreq) < g.epsilon && math.Abs(la-base.WLA) < g.epsilon {
				return refined, false
			}
			if wf > 0 && la > 0 {
				refined = append(refined, WeightTuple{wf, la})
			}
		}
	}
	return refined, true
}

func (g *GridSearchEstimator) RefineGridAroundBest(steps int, delta float64) bool {
	best := g.BestTuple()
	refined, ok := g.GenerateRefinedGrid(best, steps, delta)
	if !ok {
		g.LiveEstimator.WFreq = best.WFreq
		g.LiveEstimator.WLA = best.WLA
		g.bestHitRate = g.TupleStats[best].HitRate
		g.bestTuple = best
		return false
	}
	g.Tuples = refined
	g.CurrIndex = 0
	g.TupleStats = make(map[WeightTuple]*Stats)
	g.LiveEstimator.WFreq = g.Tuples[0].WFreq
	g.LiveEstimator.WLA = g.Tuples[0].WLA
	g.StartTime = time.Now()
	return true
}

func (g *GridSearchEstimator) RestartGridSearch() {
	g.stopGridSearch = false
	g.Tuples = g.InitialTuples
	g.CurrIndex = 0
	g.TupleStats = make(map[WeightTuple]*Stats)
	g.LiveEstimator.WFreq = g.Tuples[0].WFreq
	g.LiveEstimator.WLA = g.Tuples[0].WLA
	g.StartTime = time.Now()
	g.bestHitRate = 0
}
