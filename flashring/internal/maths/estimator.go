// Package estimator implements online adaptive grid search for tuning
// weights (wFreq, wLA) to optimize cache rewrite decisions based on hit ratio.
package maths

import (
	"math"
	"time"

	"github.com/rs/zerolog/log"
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
	// Defense in depth: pkg/cache.Config.validate now enforces FileSize
	// is a whole-number multiple of MemtableSize, so maxMemTableCount is
	// always >= 1 in valid configs. Guard anyway — this function is on
	// every Get's hot path and a stray zero here would panic on int mod.
	if maxMemTableCount == 0 {
		return 0
	}
	overWriteRisk := (activeMemId - keyMemId + maxMemTableCount) % maxMemTableCount
	overWriteRiskScore := float32(overWriteRisk) / float32(maxMemTableCount)

	fScore := 1 - math.Exp(-e.WFreq*float64(freq))
	laScore := math.Exp(-e.WLA * float64(lastAccess))
	return float32(fScore+laScore) * overWriteRiskScore
}

// RecordHitRate folds hitRate into the running average for the current
// tuple. While the search is active, advances to the next tuple after
// Duration has elapsed and refines around the best tuple at the end of
// each pass; the search stops when refinement falls below epsilon. After
// the search has stopped, a sustained drop below 90% of the previously
// observed best hit rate triggers a full restart from the initial grid.
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
			log.Info().Msgf("GridSearchRestarted: hitRate %v bestHitRate %v", stat.HitRate, g.bestHitRate)
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

			if i == 0 && j == 0 {
				continue
			}
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
	// BestTuple returns a zero WeightTuple when no candidate has been
	// observed >= 3 times (low-traffic startup). Detect the zero-tuple
	// case BEFORE calling GenerateRefinedGrid — otherwise if GRG
	// succeeds (delta > epsilon) we'd refine around (0,0), clear
	// existing stats, and switch live weights to near-zero values.
	// Return true (not false) so the caller keeps sampling the current
	// tuples rather than latching stopGridSearch — insufficient trials
	// is a "come back later" signal, not a "search is complete" one.
	if (best == WeightTuple{}) {
		return true
	}
	refined, ok := g.GenerateRefinedGrid(best, steps, delta)
	if !ok {
		// Non-zero best but convergence hit (all candidates within
		// epsilon of base). Apply best's weights and stop.
		if stats := g.TupleStats[best]; stats != nil {
			g.LiveEstimator.WFreq = best.WFreq
			g.LiveEstimator.WLA = best.WLA
			g.bestHitRate = stats.HitRate
			g.bestTuple = best
		}
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

func (g *GridSearchEstimator) IsGridSearchActive() bool {
	return !g.stopGridSearch
}
