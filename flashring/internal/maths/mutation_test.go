package maths

import (
	"math"
	"testing"
	"time"
)

// Boundary + wrap-around tests targeting mutants the broader behavioral
// suite leaves alive. Each test ties an assertion to one or two specific
// mutation positions; the existing tests already cover the happy path.

func TestNewPredictor_PartialZeroBands_NoDefaults(t *testing.T) {
	// `&&` between three == 0 checks: defaults apply only when ALL are zero.
	// Flipping `&&` to `||` would have any single zero trigger defaults.
	custom := FreqBands{Cold: 0, Warm: 7, Hot: 99}
	rcustom := RecencyBands{Hot: 99, Warm: 7, Cold: 0}
	p := NewPredictor(PredictorConfig{
		Weights:           []WeightTuple{{WFreq: 0.1, WLA: 0.2}},
		SampleDuration:    time.Hour,
		MaxMemTableCount:  10,
		GridSearchEpsilon: 0.001,
		FreqBands:         custom,
		RecencyBands:      rcustom,
	})
	if p.freqBands != custom {
		t.Errorf("freqBands = %+v, want %+v (any-zero must NOT trigger defaults)", p.freqBands, custom)
	}
	if p.recencyBands != rcustom {
		t.Errorf("recencyBands = %+v, want %+v", p.recencyBands, rcustom)
	}
}

func TestRingZone_WrapAround_NonZeroKey(t *testing.T) {
	// risk = (active - key + max) % max. Each case picks key so that
	// 2*key mod max != 0 AND the orig/mutated buckets differ — both
	// conditions are needed to actually kill the `-key`→`+key` mutation.
	cases := []struct {
		key, active, max uint32
		want             string
	}{
		{1, 0, 10, "75-100%"}, // orig risk=9 vs mutated risk=1
		{3, 1, 10, "75-100%"}, // orig risk=8 vs mutated risk=4
		{7, 1, 10, "25-50%"},  // orig risk=4 vs mutated risk=8
	}
	for _, tc := range cases {
		if got := ringZone(tc.key, tc.active, tc.max); got != tc.want {
			t.Errorf("ringZone(key=%d, active=%d, max=%d) = %q, want %q",
				tc.key, tc.active, tc.max, got, tc.want)
		}
	}
}

func TestCalculateRewriteScore_WrapAround_ExactValue(t *testing.T) {
	// Pin an exact score so arithmetic mutations on the overwrite-risk
	// expression (line 62) flip the result deterministically.
	est := &Estimator{WFreq: 0, WLA: 0}
	// With WFreq=0, WLA=0: fScore = 1 - exp(0) = 0; laScore = exp(0) = 1.
	// score = (0 + 1) * (risk / max) = risk / max.
	// risk = (0 - 3 + 10) % 10 = 7. max = 10. Expected score = 0.7.
	// Mutating `-key` to `+key` would give risk = 13 % 10 = 3 → score 0.3.
	got := est.CalculateRewriteScore(0, 0, 3, 0, 10)
	want := float32(0.7)
	if math.Abs(float64(got-want)) > 1e-6 {
		t.Errorf("CalculateRewriteScore wrap = %v, want %v", got, want)
	}
}

func TestPredict_AtExactThreshold_NoRewrite(t *testing.T) {
	// `rewrite := score > threshold` — at score == threshold the result
	// must be false. `>=` would flip this case to true.
	// With WFreq=0, WLA=0, key=0, active=1, max=2: risk=1, max=2 → score = 1*0.5 = 0.5.
	p := NewPredictor(PredictorConfig{
		ReWriteScoreThreshold: 0.5, // equal to the computed score
		Weights:               []WeightTuple{{WFreq: 0, WLA: 0}},
		SampleDuration:        time.Hour,
		MaxMemTableCount:      2,
		GridSearchEpsilon:     0.001,
	})
	if p.Predict(0, 0, 0, 1) {
		t.Error("Predict at exact threshold returned true; `>` boundary not respected")
	}
}

func TestRecordHitRate_RunningAverage_ExactValue(t *testing.T) {
	// Two hits at 0.4 and 0.8 must average to 0.6 exactly.
	// HitRate = (HitRate * Trials + new) / (Trials + 1).
	// Mutating the `+1` in the denominator breaks the average measurably.
	g, _ := newGSE(t, time.Hour)
	g.RecordHitRate(0.4)
	g.RecordHitRate(0.8)
	stat := g.TupleStats[g.Tuples[0]]
	if math.Abs(stat.HitRate-0.6) > 1e-9 {
		t.Errorf("running average = %v, want 0.6", stat.HitRate)
	}
}

func TestBestTuple_MinTrialsBoundary(t *testing.T) {
	// `stat.Trials < 3` — exactly 3 trials must qualify. `<=` would skip it.
	g, _ := newGSE(t, time.Hour)
	g.Tuples = []WeightTuple{{WFreq: 0.1, WLA: 0.1}}
	g.TupleStats[g.Tuples[0]] = &Stats{HitRate: 0.9, Trials: 3}
	if got := g.BestTuple(); got != g.Tuples[0] {
		t.Errorf("BestTuple at Trials==3 = %+v, want %+v", got, g.Tuples[0])
	}
}

func TestBestTuple_StrictGreaterBeatsEqual(t *testing.T) {
	// `stat.HitRate > bestScore` — equal hit rates must NOT replace the
	// running best. `>=` would let the last-seen tuple win, changing output.
	g, _ := newGSE(t, time.Hour)
	first := WeightTuple{WFreq: 0.1, WLA: 0.1}
	second := WeightTuple{WFreq: 0.2, WLA: 0.2}
	g.Tuples = []WeightTuple{first, second}
	g.TupleStats[first] = &Stats{HitRate: 0.7, Trials: 5}
	g.TupleStats[second] = &Stats{HitRate: 0.7, Trials: 5}
	if got := g.BestTuple(); got != first {
		t.Errorf("BestTuple with tied hit rates = %+v, want first %+v (strict >)", got, first)
	}
}

func TestGenerateRefinedGrid_DirectionAndValues(t *testing.T) {
	// `wf := base.WFreq + i*delta` — flipping `+` to `-` mirrors the grid.
	// `wf > 0 && la > 0` — strict positive excludes zero; `>=` would let
	// (0, x) and (x, 0) leak in.
	est := &Estimator{WFreq: 0.5, WLA: 0.5}
	g := NewGridSearchEstimator(time.Hour, []WeightTuple{{WFreq: 0.5, WLA: 0.5}}, est, 0.001)
	refined, ok := g.GenerateRefinedGrid(WeightTuple{WFreq: 0.5, WLA: 0.5}, 1, 0.5)
	if !ok {
		t.Fatal("GenerateRefinedGrid returned ok=false unexpectedly")
	}
	// Steps=1, delta=0.5 around (0.5, 0.5) produces a 3x3 grid centered there.
	// The four cells where either coord is zero must be excluded (wf > 0 && la > 0).
	// Surviving offsets: (-1,-1)=(0,0) excluded; (-1,0)=(0,0.5) excluded; (-1,1)=(0,1) excluded;
	// (0,-1)=(0.5,0) excluded; center (0,0) skipped; (0,1)=(0.5,1) kept;
	// (1,-1)=(1,0) excluded; (1,0)=(1,0.5) kept; (1,1)=(1,1) kept.
	want := map[WeightTuple]bool{
		{WFreq: 0.5, WLA: 1.0}: true,
		{WFreq: 1.0, WLA: 0.5}: true,
		{WFreq: 1.0, WLA: 1.0}: true,
	}
	if len(refined) != len(want) {
		t.Fatalf("refined size = %d, want %d (positive-only filter mis-fires)", len(refined), len(want))
	}
	for _, r := range refined {
		if !want[r] {
			t.Errorf("refined contains unexpected tuple %+v", r)
		}
	}
}

func TestRecordHitRate_DurationBoundary(t *testing.T) {
	// `if time.Since(g.StartTime) < g.Duration { return }` — at the boundary
	// (elapsed == duration), the function must NOT return early; advancement
	// is expected. `<=` would keep returning past the duration.
	g, _ := newGSE(t, time.Nanosecond)
	// Sleep guarantees elapsed > duration on any sane scheduler.
	time.Sleep(1 * time.Millisecond)
	g.RecordHitRate(0.5)
	if g.CurrIndex == 0 {
		t.Error("CurrIndex did not advance after duration elapsed; `<` boundary may be inverted")
	}
}

func TestRecordHitRate_RestartBoundary(t *testing.T) {
	// `if stat.HitRate < g.bestHitRate*0.9` — the restart fires strictly
	// below 90% of best. A `<=` mutation would also fire at exactly 90%.
	// Drive a sample exactly at the boundary and assert the search did NOT
	// restart (stopGridSearch stays true).
	g, _ := newGSE(t, time.Hour)
	g.stopGridSearch = true
	g.bestHitRate = 1.0
	// Seed bestTuple so the single Observe averages to 0.9.
	g.TupleStats[g.bestTuple] = &Stats{HitRate: 0.9, Trials: 0}
	g.RecordHitRate(0.9) // running avg becomes 0.9 / 1 = 0.9 exactly
	if !g.stopGridSearch {
		t.Error("Search restarted at exactly 90% threshold; boundary `<` must be strict")
	}
}

func TestMorrisLogCounter_ThresholdBoundary(t *testing.T) {
	// `if uint32(hlo) >= c.th[e] { return v, false }` — at the exact
	// threshold the increment must be REJECTED. `>` would accept it.
	c := New()
	// Use e=1 entry so th[1] is below maxUint32; assert the exact-threshold
	// hlo skips the increment.
	v := uint16(1) << eShift // mantissa=0, exponent=1
	hlo := uint64(c.th[1])   // exact threshold
	got, inc := c.Inc(v, hlo)
	if inc {
		t.Errorf("Inc at hlo==threshold returned inc=true; `>=` boundary not respected")
	}
	if got != v {
		t.Errorf("Inc at hlo==threshold returned counter %d, want unchanged %d", got, v)
	}
}
