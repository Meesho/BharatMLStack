package maths

import (
	"testing"
	"time"
)

func defaultTuples() []WeightTuple {
	return []WeightTuple{
		{WFreq: 1.0, WLA: 1.0},
		{WFreq: 2.0, WLA: 2.0},
		{WFreq: 3.0, WLA: 3.0},
	}
}

func newGSE(t *testing.T, dur time.Duration) (*GridSearchEstimator, *Estimator) {
	t.Helper()
	tuples := defaultTuples()
	est := NewEstimator(tuples[0].WFreq, tuples[0].WLA)
	return NewGridSearchEstimator(dur, tuples, est, 0.0001), est
}

func TestNewGridSearchEstimator_Defaults(t *testing.T) {
	g, est := newGSE(t, time.Minute)

	if g.LiveEstimator != est {
		t.Fatal("LiveEstimator not stored")
	}
	if len(g.Tuples) != 3 {
		t.Fatalf("len(Tuples) = %d, want 3", len(g.Tuples))
	}
	if g.bestTuple != g.Tuples[0] {
		t.Fatalf("bestTuple = %+v, want first tuple", g.bestTuple)
	}
	if g.CurrIndex != 0 {
		t.Fatalf("CurrIndex = %d, want 0", g.CurrIndex)
	}
	if g.epsilon != 0.0001 {
		t.Fatalf("epsilon = %v, want 0.0001", g.epsilon)
	}
}

func TestCalculateRewriteScore_RisingFreqIncreasesScore(t *testing.T) {
	est := NewEstimator(1.0, 1.0)

	low := est.CalculateRewriteScore(1, 5, 0, 1, 4)
	high := est.CalculateRewriteScore(100, 5, 0, 1, 4)

	if !(high > low) {
		t.Fatalf("score did not rise with freq: low=%v high=%v", low, high)
	}
}

func TestCalculateRewriteScore_HigherLastAccessDecreasesScore(t *testing.T) {
	est := NewEstimator(1.0, 1.0)

	recent := est.CalculateRewriteScore(10, 1, 0, 1, 4)
	stale := est.CalculateRewriteScore(10, 100, 0, 1, 4)

	if !(recent > stale) {
		t.Fatalf("score did not drop with stale last-access: recent=%v stale=%v",
			recent, stale)
	}
}

func TestCalculateRewriteScore_RingWrap_ModuloHandled(t *testing.T) {
	est := NewEstimator(1.0, 1.0)

	// keyMemId=3, activeMemId=0 with max=4 → risk=(0-3+4)%4 = 1.
	got := est.CalculateRewriteScore(10, 5, 3, 0, 4)
	if got <= 0 {
		t.Fatalf("score for wrapped ring = %v, want > 0", got)
	}
}

func TestRecordHitRate_BelowDuration_AdvancesStats(t *testing.T) {
	g, _ := newGSE(t, time.Hour) // far longer than test duration

	g.RecordHitRate(0.5)
	g.RecordHitRate(0.7)

	tup := g.Tuples[0]
	stat := g.TupleStats[tup]
	if stat == nil {
		t.Fatal("TupleStats nil for current tuple")
	}
	if stat.Trials != 2 {
		t.Fatalf("Trials = %d, want 2", stat.Trials)
	}
	want := (0.5 + 0.7) / 2.0
	if stat.HitRate != want {
		t.Fatalf("HitRate = %v, want %v", stat.HitRate, want)
	}
	// Still at index 0 because duration hasn't elapsed.
	if g.CurrIndex != 0 {
		t.Fatalf("CurrIndex = %d, want 0", g.CurrIndex)
	}
}

func TestRecordHitRate_PastDuration_AdvancesIndex(t *testing.T) {
	g, _ := newGSE(t, time.Nanosecond) // immediately expired

	g.RecordHitRate(0.5)

	if g.CurrIndex != 1 {
		t.Fatalf("CurrIndex = %d, want 1 after duration elapsed", g.CurrIndex)
	}
	// LiveEstimator should now match Tuples[1].
	if g.LiveEstimator.Weights().WFreq != g.Tuples[1].WFreq {
		t.Fatalf("LiveEstimator.Weights().WFreq = %v, want %v", g.LiveEstimator.Weights().WFreq, g.Tuples[1].WFreq)
	}
}

func TestRecordHitRate_FullSweep_RefinesGrid(t *testing.T) {
	g, _ := newGSE(t, time.Nanosecond)

	// Each Record advances index. After all 3 tuples are sampled enough
	// (Trials>=3 for BestTuple), the wrap to index 0 calls RefineGridAroundBest
	// which generates a refined grid (since delta is much larger than epsilon).
	originalTuplesAddr := &g.Tuples[0]

	for i := 0; i < 9; i++ {
		g.RecordHitRate(0.4 + 0.1*float64(i%3))
	}

	if &g.Tuples[0] == originalTuplesAddr {
		t.Fatal("Tuples slice not replaced after refine")
	}
	if g.stopGridSearch {
		t.Fatal("stopGridSearch flipped unexpectedly during refinement")
	}
}

func TestRecordHitRate_StopGridSearch_BranchTracksStat(t *testing.T) {
	g, _ := newGSE(t, time.Hour)
	g.stopGridSearch = true
	g.bestHitRate = 0.8

	g.RecordHitRate(0.81)

	stat := g.TupleStats[g.bestTuple]
	if stat == nil {
		t.Fatal("TupleStats not seeded for bestTuple")
	}
	if stat.Trials != 1 {
		t.Fatalf("Trials = %d, want 1", stat.Trials)
	}
	if stat.HitRate != 0.81 {
		t.Fatalf("HitRate = %v, want 0.81", stat.HitRate)
	}
}

func TestRecordHitRate_StopGridSearch_BelowThreshold_Restarts(t *testing.T) {
	g, _ := newGSE(t, time.Hour)
	g.stopGridSearch = true
	g.bestHitRate = 1.0
	g.bestTuple = g.Tuples[0]

	// stat.HitRate = 0.5 < 1.0 * 0.9 = 0.9 → triggers RestartGridSearch.
	g.RecordHitRate(0.5)

	if g.stopGridSearch {
		t.Fatal("stopGridSearch still true after restart")
	}
	if g.bestHitRate != 0 {
		t.Fatalf("bestHitRate = %v, want 0 after restart", g.bestHitRate)
	}
}

func TestRecordHitRate_StopGridSearch_StatExistsAlready(t *testing.T) {
	g, _ := newGSE(t, time.Hour)
	g.stopGridSearch = true
	g.bestTuple = g.Tuples[0]
	g.TupleStats[g.bestTuple] = &Stats{HitRate: 0.5, Trials: 1}

	// stat already exists → second Record updates running average.
	g.RecordHitRate(0.5)

	stat := g.TupleStats[g.bestTuple]
	if stat.Trials != 2 {
		t.Fatalf("Trials = %d, want 2", stat.Trials)
	}
}

func TestBestTuple_NoQualifyingTuple_ReturnsZeroValue(t *testing.T) {
	g, _ := newGSE(t, time.Hour)

	// No trials recorded → BestTuple skips them all.
	got := g.BestTuple()
	if got != (WeightTuple{}) {
		t.Fatalf("BestTuple = %+v, want zero", got)
	}
}

func TestBestTuple_ChoosesHighestHitRate(t *testing.T) {
	g, _ := newGSE(t, time.Hour)

	g.TupleStats[g.Tuples[0]] = &Stats{HitRate: 0.4, Trials: 5}
	g.TupleStats[g.Tuples[1]] = &Stats{HitRate: 0.9, Trials: 5}
	g.TupleStats[g.Tuples[2]] = &Stats{HitRate: 0.6, Trials: 5}

	got := g.BestTuple()
	if got != g.Tuples[1] {
		t.Fatalf("BestTuple = %+v, want %+v", got, g.Tuples[1])
	}
}

func TestBestTuple_SkipsLowTrialsAndNilStats(t *testing.T) {
	g, _ := newGSE(t, time.Hour)

	g.TupleStats[g.Tuples[0]] = &Stats{HitRate: 0.95, Trials: 1} // skipped (low)
	g.TupleStats[g.Tuples[1]] = &Stats{HitRate: 0.50, Trials: 5} // kept
	// Tuples[2] has no stat (nil) — skipped via the nil branch.

	got := g.BestTuple()
	if got != g.Tuples[1] {
		t.Fatalf("BestTuple = %+v, want %+v", got, g.Tuples[1])
	}
}

func TestGenerateRefinedGrid_RespectsSteps(t *testing.T) {
	g, _ := newGSE(t, time.Hour)

	refined, ok := g.GenerateRefinedGrid(WeightTuple{WFreq: 5, WLA: 5}, 1, 0.5)
	if !ok {
		t.Fatal("GenerateRefinedGrid returned ok=false for delta >> epsilon")
	}
	// (2*1+1)^2 - 1 = 8 candidates. All have wf,la > 0 since base 5,5 and
	// delta 0.5 keeps everything positive.
	if len(refined) != 8 {
		t.Fatalf("refined len = %d, want 8", len(refined))
	}
}

func TestGenerateRefinedGrid_EpsilonAbort(t *testing.T) {
	g, _ := newGSE(t, time.Hour)

	// delta=0.00001 < g.epsilon=0.0001 → first non-skip iteration hits the
	// epsilon abort branch.
	refined, ok := g.GenerateRefinedGrid(WeightTuple{WFreq: 1, WLA: 1}, 1, 0.00001)
	if ok {
		t.Fatal("ok = true when delta < epsilon")
	}
	if len(refined) != 0 {
		t.Fatalf("refined non-empty on abort: %v", refined)
	}
}

func TestGenerateRefinedGrid_NegativeCandidatesSkipped(t *testing.T) {
	g, _ := newGSE(t, time.Hour)

	// base (1, 1) with delta=2 and steps=1 → some candidates go negative or
	// zero (e.g. 1 + (-1)*2 = -1) and are filtered. With i,j in [-1,0,1]^2
	// minus origin, 8 candidates total; (-1,-1),(-1,0),(-1,1),(0,-1),(0,1),
	// (1,-1),(1,0),(1,1) → after >0 filter, only (1,0)→(3,1) and (0,1)→(1,3)
	// and (1,1)→(3,3) survive.
	refined, ok := g.GenerateRefinedGrid(WeightTuple{WFreq: 1, WLA: 1}, 1, 2.0)
	if !ok {
		t.Fatal("ok = false for valid grid")
	}
	for _, r := range refined {
		if r.WFreq <= 0 || r.WLA <= 0 {
			t.Fatalf("non-positive candidate slipped through: %+v", r)
		}
	}
	if len(refined) >= 8 {
		t.Fatalf("expected some candidates filtered, got %d", len(refined))
	}
}

func TestRefineGridAroundBest_RefineSucceeds(t *testing.T) {
	g, _ := newGSE(t, time.Hour)

	g.TupleStats[g.Tuples[1]] = &Stats{HitRate: 0.9, Trials: 5}

	ok := g.RefineGridAroundBest(1, 0.5)
	if !ok {
		t.Fatal("RefineGridAroundBest returned false for valid refinement")
	}
	if g.CurrIndex != 0 {
		t.Fatalf("CurrIndex = %d, want 0 after refine", g.CurrIndex)
	}
	if len(g.TupleStats) != 0 {
		t.Fatalf("TupleStats not cleared after refine: %d entries", len(g.TupleStats))
	}
}

func TestRefineGridAroundBest_EpsilonReached_LocksInBest(t *testing.T) {
	g, _ := newGSE(t, time.Hour)

	g.TupleStats[g.Tuples[1]] = &Stats{HitRate: 0.9, Trials: 5}

	// Delta below epsilon → GenerateRefinedGrid returns ok=false; method must
	// lock in best and return false.
	ok := g.RefineGridAroundBest(1, 0.00001)
	if ok {
		t.Fatal("RefineGridAroundBest returned true when no refinement possible")
	}
	if g.bestTuple != g.Tuples[1] {
		t.Fatalf("bestTuple = %+v, want %+v", g.bestTuple, g.Tuples[1])
	}
	if g.bestHitRate != 0.9 {
		t.Fatalf("bestHitRate = %v, want 0.9", g.bestHitRate)
	}
	if g.LiveEstimator.Weights().WFreq != g.Tuples[1].WFreq {
		t.Fatalf("LiveEstimator not updated to best")
	}
}

func TestRestartGridSearch_ResetsState(t *testing.T) {
	g, _ := newGSE(t, time.Hour)

	g.stopGridSearch = true
	g.bestHitRate = 0.7
	g.CurrIndex = 2
	g.TupleStats[g.Tuples[0]] = &Stats{Trials: 5}
	g.Tuples = []WeightTuple{{WFreq: 99, WLA: 99}}

	g.RestartGridSearch()

	if g.stopGridSearch {
		t.Fatal("stopGridSearch still true after restart")
	}
	if len(g.Tuples) != len(g.InitialTuples) {
		t.Fatalf("Tuples not restored to InitialTuples")
	}
	if g.CurrIndex != 0 {
		t.Fatalf("CurrIndex = %d, want 0", g.CurrIndex)
	}
	if len(g.TupleStats) != 0 {
		t.Fatalf("TupleStats not cleared: %d entries", len(g.TupleStats))
	}
	if g.bestHitRate != 0 {
		t.Fatalf("bestHitRate = %v, want 0", g.bestHitRate)
	}
	if g.LiveEstimator.Weights().WFreq != g.InitialTuples[0].WFreq {
		t.Fatal("LiveEstimator not reset to initial")
	}
}

func TestRecordHitRate_FullSweepConverges_StopsSearch(t *testing.T) {
	// Epsilon larger than the initial refinement step → the first
	// refinement converges and flips stopGridSearch.
	tuples := defaultTuples()
	est := NewEstimator(tuples[0].WFreq, tuples[0].WLA)
	g := NewGridSearchEstimator(time.Nanosecond, tuples, est, 0.5)

	// Each tuple needs >=3 trials so BestTuple picks a winner.
	for _, tup := range tuples {
		g.TupleStats[tup] = &Stats{HitRate: 0.5, Trials: 3}
	}
	g.CurrIndex = 2 // next advance wraps back to 0

	g.RecordHitRate(0.5)

	if !g.stopGridSearch {
		t.Fatal("stopGridSearch not flipped on convergence")
	}
}

func TestIsGridSearchActive_TracksFlag(t *testing.T) {
	g, _ := newGSE(t, time.Hour)

	if !g.IsGridSearchActive() {
		t.Fatal("IsGridSearchActive = false when search not stopped")
	}

	g.stopGridSearch = true
	if g.IsGridSearchActive() {
		t.Fatal("IsGridSearchActive = true after stop")
	}
}

// TestBestTuple_NoTrials_ReturnsZero verifies the pre-condition that
// RefineGridAroundBest depends on: when no tuple has been observed >= 3
// times, BestTuple returns the zero WeightTuple. Locks down the contract
// so the zero-tuple guard in RefineGridAroundBest (PR #2377) can trust
// its input.
func TestBestTuple_NoTrials_ReturnsZero(t *testing.T) {
	g, _ := newGSE(t, time.Hour)
	// No RecordHitRate calls; TupleStats is empty.
	best := g.BestTuple()
	if best != (WeightTuple{}) {
		t.Fatalf("BestTuple = %+v, want zero", best)
	}
}

// TestRefineGridAroundBest_ZeroBest_KeepsSearchAlive locks down the m-review
// fix (PR #2377): when BestTuple returns the zero WeightTuple, refine must
// bail out BEFORE calling GenerateRefinedGrid (otherwise GRG succeeds and
// we'd refine around (0,0), clear stats, and set live weights near zero).
// The return value MUST be true so the caller keeps sampling initial tuples
// rather than latching stopGridSearch.
func TestRefineGridAroundBest_ZeroBest_KeepsSearchAlive(t *testing.T) {
	g, _ := newGSE(t, time.Hour)
	origTuplesLen := len(g.Tuples)
	origWFreq := g.LiveEstimator.Weights().WFreq
	origWLA := g.LiveEstimator.Weights().WLA

	// With no trials, BestTuple returns zero.
	if g.BestTuple() != (WeightTuple{}) {
		t.Fatal("test precondition failed: BestTuple non-zero on fresh estimator")
	}

	// GRG_delta (0.001) MUST exceed GridSearchEpsilon to prove the fix isn't
	// masked by the epsilon-early-return in GenerateRefinedGrid.
	if 0.001 <= g.epsilon {
		t.Skipf("test needs delta(0.001) > epsilon(%v)", g.epsilon)
	}

	ok := g.RefineGridAroundBest(2, 0.001)
	if !ok {
		t.Fatal("RefineGridAroundBest returned false — would latch stopGridSearch on empty stats")
	}
	if len(g.Tuples) != origTuplesLen {
		t.Fatalf("Tuples slice replaced despite zero best: len=%d, want %d", len(g.Tuples), origTuplesLen)
	}
	if g.LiveEstimator.Weights().WFreq != origWFreq || g.LiveEstimator.Weights().WLA != origWLA {
		t.Fatalf("LiveEstimator drifted: WFreq=%v (want %v), WLA=%v (want %v)",
			g.LiveEstimator.Weights().WFreq, origWFreq, g.LiveEstimator.Weights().WLA, origWLA)
	}
}

// TestCalculateRewriteScore_ZeroMaxMemTableCount_NoPanic locks down the
// defense-in-depth guard added in PR #2377. The Config divisibility check
// makes maxMemTableCount>=1 in valid configs, but the maths layer must
// not panic on modulo-by-zero regardless.
func TestCalculateRewriteScore_ZeroMaxMemTableCount_NoPanic(t *testing.T) {
	e := NewEstimator(0.5, 0.5)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("CalculateRewriteScore(...,0) panicked: %v", r)
		}
	}()
	score := e.CalculateRewriteScore(100, 50, 5, 7, 0)
	if score != 0 {
		t.Fatalf("score = %v, want 0 for maxMemTableCount=0", score)
	}
}

// The refinement step halves each pass, so a search fed steady samples
// stops once the step is below epsilon instead of refining forever.
func TestRecordHitRate_ShrinkingDeltaConverges(t *testing.T) {
	est := NewEstimator(0.5, 0.5)
	g := NewGridSearchEstimator(0, []WeightTuple{{WFreq: 0.5, WLA: 0.5}}, est, 0.01)

	for i := 0; i < 10000 && g.IsGridSearchActive(); i++ {
		g.RecordHitRate(0.8)
	}
	if g.IsGridSearchActive() {
		t.Fatal("search still active after 10000 samples")
	}
	// 0.1, 0.05, 0.025, 0.0125 refine; 0.00625 < epsilon stops.
	if want := initialRefineDelta / 16; g.delta != want {
		t.Fatalf("delta = %v, want %v", g.delta, want)
	}
	// A flat hit rate carries no signal, so the incumbent must survive
	// every pass rather than drift to a neighbour.
	if got, want := est.Weights(), (WeightTuple{WFreq: 0.5, WLA: 0.5}); got != want || g.bestTuple != want {
		t.Fatalf("live weights = %+v, best = %+v, want both %+v", got, g.bestTuple, want)
	}

	g.RestartGridSearch()
	if g.delta != initialRefineDelta {
		t.Fatalf("delta after restart = %v, want %v", g.delta, initialRefineDelta)
	}
}
