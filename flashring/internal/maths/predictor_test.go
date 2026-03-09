package maths

import (
	"testing"
	"time"
)

func TestNewPredictor(t *testing.T) {
	config := PredictorConfig{
		ReWriteScoreThreshold: 0.5,
		Weights: []WeightTuple{
			{WFreq: 0.1, WLA: 0.2},
			{WFreq: 0.2, WLA: 0.3},
		},
		SampleDuration:    100 * time.Millisecond,
		MaxMemTableCount:  10,
		GridSearchEpsilon: 0.001,
	}

	predictor := NewPredictor(config)

	// Verify predictor initialization
	if predictor == nil {
		t.Fatal("NewPredictor returned nil")
	}
	if predictor.ReWriteScoreThreshold != 0.5 {
		t.Errorf("Expected ReWriteScoreThreshold 0.5, got %f", predictor.ReWriteScoreThreshold)
	}
	if predictor.MaxMemTableCount != 10 {
		t.Errorf("Expected MaxMemTableCount 10, got %d", predictor.MaxMemTableCount)
	}

	// Verify estimator initialization
	if predictor.Estimator == nil {
		t.Fatal("Estimator not initialized")
	}
	if predictor.Estimator.WFreq != 0.1 {
		t.Errorf("Expected WFreq 0.1, got %f", predictor.Estimator.WFreq)
	}
	if predictor.Estimator.WLA != 0.2 {
		t.Errorf("Expected WLA 0.2, got %f", predictor.Estimator.WLA)
	}

	// Verify grid search estimator initialization
	if predictor.GridSearchEstimator == nil {
		t.Fatal("GridSearchEstimator not initialized")
	}

	// Verify channel initialization
	if predictor.hitRateCh == nil {
		t.Fatal("hitRateCh not initialized")
	}
}

func TestPredictorPredict(t *testing.T) {
	config := PredictorConfig{
		ReWriteScoreThreshold: 0.5,
		Weights: []WeightTuple{
			{WFreq: 0.1, WLA: 0.2},
		},
		SampleDuration:    100 * time.Millisecond,
		MaxMemTableCount:  10,
		GridSearchEpsilon: 0.001,
	}

	predictor := NewPredictor(config)

	tests := []struct {
		name          string
		freq          uint64
		lastAccess    uint64
		keyMemId      uint32
		activeMemId   uint32
		expectRewrite bool
	}{
		{
			name:          "high frequency, recent access, high overwrite risk",
			freq:          100,
			lastAccess:    1,
			keyMemId:      0,
			activeMemId:   8,
			expectRewrite: true,
		},
		{
			name:          "low frequency, old access, low overwrite risk",
			freq:          1,
			lastAccess:    1000,
			keyMemId:      5,
			activeMemId:   6,
			expectRewrite: false,
		},
		{
			name:          "medium frequency, medium access, medium overwrite risk",
			freq:          10,
			lastAccess:    50,
			keyMemId:      3,
			activeMemId:   7,
			expectRewrite: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := predictor.Predict(tt.freq, tt.lastAccess, tt.keyMemId, tt.activeMemId)
			if result != tt.expectRewrite {
				score := predictor.Estimator.CalculateRewriteScore(
					tt.freq, tt.lastAccess, tt.keyMemId, tt.activeMemId, predictor.MaxMemTableCount)
				t.Errorf("Expected %v, got %v (score: %f, threshold: %f)",
					tt.expectRewrite, result, score, predictor.ReWriteScoreThreshold)
			}
		})
	}
}

func TestPredictorObserve(t *testing.T) {
	config := PredictorConfig{
		ReWriteScoreThreshold: 0.5,
		Weights: []WeightTuple{
			{WFreq: 0.1, WLA: 0.2},
		},
		SampleDuration:    10 * time.Millisecond,
		MaxMemTableCount:  10,
		GridSearchEpsilon: 0.001,
	}

	predictor := NewPredictor(config)

	// Test observing hit rates
	hitRates := []float64{0.8, 0.7, 0.9, 0.6}

	for _, hitRate := range hitRates {
		predictor.Observe(hitRate)
	}

	// Give some time for the goroutine to process
	time.Sleep(50 * time.Millisecond)

	// Channel should not block on additional observations
	for i := 0; i < 10; i++ {
		predictor.Observe(0.5)
	}
}

func TestEstimatorCalculateRewriteScore(t *testing.T) {
	estimator := &Estimator{
		WFreq: 0.1,
		WLA:   0.2,
	}

	tests := []struct {
		name             string
		freq             uint64
		lastAccess       uint64
		keyMemId         uint32
		activeMemId      uint32
		maxMemTableCount uint32
		expectHighScore  bool
	}{
		{
			name:             "high frequency, recent access, high overwrite risk",
			freq:             100,
			lastAccess:       1,
			keyMemId:         0,
			activeMemId:      9,
			maxMemTableCount: 10,
			expectHighScore:  true,
		},
		{
			name:             "low frequency, old access, low overwrite risk",
			freq:             1,
			lastAccess:       1000,
			keyMemId:         5,
			activeMemId:      6,
			maxMemTableCount: 10,
			expectHighScore:  false,
		},
		{
			name:             "zero frequency should give low score",
			freq:             0,
			lastAccess:       0,
			keyMemId:         0,
			activeMemId:      0,
			maxMemTableCount: 10,
			expectHighScore:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := estimator.CalculateRewriteScore(
				tt.freq, tt.lastAccess, tt.keyMemId, tt.activeMemId, tt.maxMemTableCount)

			if tt.expectHighScore && score < 0.1 {
				t.Errorf("Expected high score, got %f", score)
			}
			if !tt.expectHighScore && score > 0.5 {
				t.Errorf("Expected low score, got %f", score)
			}

			// Score should always be non-negative
			if score < 0 {
				t.Errorf("Score should be non-negative, got %f", score)
			}
		})
	}
}

func TestEstimatorScoreComponents(t *testing.T) {
	estimator := &Estimator{
		WFreq: 0.1,
		WLA:   0.2,
	}

	// Test that frequency score increases with frequency
	score1 := estimator.CalculateRewriteScore(1, 100, 0, 5, 10)
	score2 := estimator.CalculateRewriteScore(10, 100, 0, 5, 10)
	score3 := estimator.CalculateRewriteScore(100, 100, 0, 5, 10)

	if !(score1 < score2 && score2 < score3) {
		t.Errorf("Score should increase with frequency: %f, %f, %f", score1, score2, score3)
	}

	// Test that last access score decreases with time
	score1 = estimator.CalculateRewriteScore(10, 1, 0, 5, 10)
	score2 = estimator.CalculateRewriteScore(10, 10, 0, 5, 10)
	score3 = estimator.CalculateRewriteScore(10, 100, 0, 5, 10)

	if !(score1 > score2 && score2 > score3) {
		t.Errorf("Score should decrease with last access time: %f, %f, %f", score1, score2, score3)
	}

	// Test overwrite risk calculation
	score1 = estimator.CalculateRewriteScore(10, 10, 0, 1, 10) // low risk
	score2 = estimator.CalculateRewriteScore(10, 10, 0, 5, 10) // medium risk
	score3 = estimator.CalculateRewriteScore(10, 10, 0, 9, 10) // high risk

	if !(score1 < score2 && score2 < score3) {
		t.Errorf("Score should increase with overwrite risk: %f, %f, %f", score1, score2, score3)
	}
}

func TestGridSearchEstimator(t *testing.T) {
	initialTuples := []WeightTuple{
		{WFreq: 0.1, WLA: 0.1},
		{WFreq: 0.2, WLA: 0.2},
		{WFreq: 0.3, WLA: 0.3},
	}

	estimator := &Estimator{WFreq: 0.1, WLA: 0.1}
	gridSearch := NewGridSearchEstimator(
		50*time.Millisecond,
		initialTuples,
		estimator,
		0.001,
	)

	// Test initialization
	if len(gridSearch.Tuples) != 3 {
		t.Errorf("Expected 3 tuples, got %d", len(gridSearch.Tuples))
	}
	if gridSearch.CurrIndex != 0 {
		t.Errorf("Expected CurrIndex 0, got %d", gridSearch.CurrIndex)
	}

	// Test recording hit rates
	hitRates := []float64{0.8, 0.7, 0.9}
	for i, hitRate := range hitRates {
		gridSearch.RecordHitRate(hitRate)
		if i < len(hitRates)-1 {
			time.Sleep(60 * time.Millisecond) // Wait for duration to pass
		}
	}

	// Verify stats are recorded
	for _, tuple := range initialTuples {
		if stat, ok := gridSearch.TupleStats[tuple]; ok && stat.Trials > 0 {
			if stat.HitRate < 0 || stat.HitRate > 1 {
				t.Errorf("Invalid hit rate %f for tuple %+v", stat.HitRate, tuple)
			}
		}
	}
}

func TestGridSearchBestTuple(t *testing.T) {
	initialTuples := []WeightTuple{
		{WFreq: 0.1, WLA: 0.1},
		{WFreq: 0.2, WLA: 0.2},
		{WFreq: 0.3, WLA: 0.3},
	}

	estimator := &Estimator{WFreq: 0.1, WLA: 0.1}
	gridSearch := NewGridSearchEstimator(
		10*time.Millisecond,
		initialTuples,
		estimator,
		0.001,
	)

	// Manually add stats
	gridSearch.TupleStats[initialTuples[0]] = &Stats{HitRate: 0.7, Trials: 5}
	gridSearch.TupleStats[initialTuples[1]] = &Stats{HitRate: 0.9, Trials: 5}
	gridSearch.TupleStats[initialTuples[2]] = &Stats{HitRate: 0.6, Trials: 5}

	best := gridSearch.BestTuple()
	expected := initialTuples[1] // Should be the one with 0.9 hit rate

	if best.WFreq != expected.WFreq || best.WLA != expected.WLA {
		t.Errorf("Expected best tuple %+v, got %+v", expected, best)
	}
}

func TestGridSearchRefinement(t *testing.T) {
	initialTuples := []WeightTuple{
		{WFreq: 0.2, WLA: 0.2},
	}

	estimator := &Estimator{WFreq: 0.2, WLA: 0.2}

	t.Run("delta > epsilon: refinement possible", func(t *testing.T) {
		gs := NewGridSearchEstimator(10*time.Millisecond, initialTuples, estimator, 0.01)
		base := WeightTuple{WFreq: 0.2, WLA: 0.2}
		refined, ok := gs.GenerateRefinedGrid(base, 1, 0.1)
		if !ok {
			t.Error("Expected ok=true when delta > epsilon (refinement still useful)")
		}
		// 3x3 grid minus the center = 8 points; all have positive wf and la
		// (base 0.2 ± 0.1 yields 0.1..0.3)
		if len(refined) != 8 {
			t.Errorf("Expected 8 refined tuples, got %d", len(refined))
		}
	})

	t.Run("delta < epsilon: convergence detected", func(t *testing.T) {
		gs := NewGridSearchEstimator(10*time.Millisecond, initialTuples, estimator, 0.1)
		base := WeightTuple{WFreq: 0.2, WLA: 0.2}
		_, ok := gs.GenerateRefinedGrid(base, 1, 0.01)
		if ok {
			t.Error("Expected ok=false when delta < epsilon (converged)")
		}
	})

	t.Run("larger grid with delta > epsilon", func(t *testing.T) {
		gs := NewGridSearchEstimator(10*time.Millisecond, initialTuples, estimator, 0.001)
		base := WeightTuple{WFreq: 0.5, WLA: 0.5}
		refined, ok := gs.GenerateRefinedGrid(base, 2, 0.1)
		if !ok {
			t.Error("Expected ok=true when delta >> epsilon")
		}
		// 5x5 grid minus center = 24 points; all positive (0.3..0.7)
		if len(refined) != 24 {
			t.Errorf("Expected 24 refined tuples, got %d", len(refined))
		}
	})
}

func TestGridSearchConvergence(t *testing.T) {
	initialTuples := []WeightTuple{
		{WFreq: 0.1, WLA: 0.1},
	}

	estimator := &Estimator{WFreq: 0.1, WLA: 0.1}
	gridSearch := NewGridSearchEstimator(
		1*time.Millisecond,
		initialTuples,
		estimator,
		0.1, // Large epsilon for quick convergence
	)

	// Test convergence with very small delta
	base := WeightTuple{WFreq: 0.1, WLA: 0.1}
	_, ok := gridSearch.GenerateRefinedGrid(base, 1, 0.01) // Small delta

	if ok {
		t.Error("Grid refinement should fail when delta is smaller than epsilon")
	}
}

func BenchmarkEstimatorCalculateRewriteScore(b *testing.B) {
	estimator := &Estimator{
		WFreq: 0.1,
		WLA:   0.2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.CalculateRewriteScore(
			uint64(i%100+1),  // freq
			uint64(i%1000+1), // lastAccess
			uint32(i%10),     // keyMemId
			uint32((i+5)%10), // activeMemId
			10,               // maxMemTableCount
		)
	}
}

func BenchmarkPredictorPredict(b *testing.B) {
	config := PredictorConfig{
		ReWriteScoreThreshold: 0.5,
		Weights: []WeightTuple{
			{WFreq: 0.1, WLA: 0.2},
		},
		SampleDuration:    100 * time.Millisecond,
		MaxMemTableCount:  10,
		GridSearchEpsilon: 0.001,
	}

	predictor := NewPredictor(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		predictor.Predict(
			uint64(i%100+1),  // freq
			uint64(i%1000+1), // lastAccess
			uint32(i%10),     // keyMemId
			uint32((i+5)%10), // activeMemId
		)
	}
}

// Integration test that simulates a realistic cache scenario
func TestPredictorIntegration(t *testing.T) {
	config := PredictorConfig{
		ReWriteScoreThreshold: 0.3,
		Weights: []WeightTuple{
			{WFreq: 0.1, WLA: 0.1},
			{WFreq: 0.2, WLA: 0.2},
			{WFreq: 0.3, WLA: 0.3},
		},
		SampleDuration:    20 * time.Millisecond,
		MaxMemTableCount:  8,
		GridSearchEpsilon: 0.01,
	}

	predictor := NewPredictor(config)

	// Simulate cache operations
	type cacheOp struct {
		freq        uint64
		lastAccess  uint64
		keyMemId    uint32
		activeMemId uint32
	}

	operations := []cacheOp{
		{freq: 100, lastAccess: 1, keyMemId: 0, activeMemId: 7},  // Should rewrite
		{freq: 1, lastAccess: 1000, keyMemId: 6, activeMemId: 7}, // Should not rewrite
		{freq: 50, lastAccess: 10, keyMemId: 2, activeMemId: 6},  // Maybe rewrite
		{freq: 200, lastAccess: 5, keyMemId: 1, activeMemId: 7},  // Should rewrite
	}

	rewriteCount := 0
	for i, op := range operations {
		shouldRewrite := predictor.Predict(op.freq, op.lastAccess, op.keyMemId, op.activeMemId)
		if shouldRewrite {
			rewriteCount++
		}

		// Simulate hit rate feedback
		var hitRate float64
		if shouldRewrite {
			hitRate = 0.8 + 0.1*float64(i%3) // Simulated good hit rate for rewrites
		} else {
			hitRate = 0.6 + 0.1*float64(i%2) // Simulated moderate hit rate for no rewrites
		}

		predictor.Observe(hitRate)

		// Small delay to allow processing
		time.Sleep(5 * time.Millisecond)
	}

	// Should have made some rewrite decisions
	if rewriteCount == 0 {
		t.Error("Expected at least some rewrite decisions")
	}
	if rewriteCount == len(operations) {
		t.Error("Should not rewrite everything")
	}

	t.Logf("Made %d rewrites out of %d operations", rewriteCount, len(operations))
}
