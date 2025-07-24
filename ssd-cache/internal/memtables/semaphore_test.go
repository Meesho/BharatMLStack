package memtables

import (
	"sync"
	"testing"
	"time"
)

func TestNewSemaphore(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
	}{
		{"zero capacity", 0},
		{"single capacity", 1},
		{"multiple capacity", 5},
		{"large capacity", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sem := NewSemaphore(tt.capacity)
			if sem == nil {
				t.Error("NewSemaphore returned nil")
			}
			if sem.ch == nil {
				t.Error("semaphore channel is nil")
			}
			if cap(sem.ch) != tt.capacity {
				t.Errorf("expected capacity %d, got %d", tt.capacity, cap(sem.ch))
			}
		})
	}
}

func TestSemaphore_Acquire(t *testing.T) {
	t.Run("single acquire", func(t *testing.T) {
		sem := NewSemaphore(1)
		result := sem.Acquire()
		if !result {
			t.Error("Acquire should return true")
		}
	})

	t.Run("multiple acquire within capacity", func(t *testing.T) {
		capacity := 3
		sem := NewSemaphore(capacity)

		for i := 0; i < capacity; i++ {
			result := sem.Acquire()
			if !result {
				t.Errorf("Acquire %d should return true", i+1)
			}
		}
	})

	t.Run("acquire blocks when capacity exceeded", func(t *testing.T) {
		sem := NewSemaphore(1)
		sem.Acquire() // Fill the semaphore

		done := make(chan bool)
		go func() {
			sem.Acquire() // This should block
			done <- true
		}()

		// Give the goroutine time to attempt acquisition
		select {
		case <-done:
			t.Error("Acquire should have blocked")
		case <-time.After(50 * time.Millisecond):
			// Good, it blocked as expected
		}

		// Release and verify the blocked goroutine proceeds
		sem.Release()
		select {
		case <-done:
			// Good, unblocked
		case <-time.After(100 * time.Millisecond):
			t.Error("Acquire should have unblocked after Release")
		}
	})
}

func TestSemaphore_Release(t *testing.T) {
	t.Run("release after acquire", func(t *testing.T) {
		sem := NewSemaphore(1)
		sem.Acquire()
		sem.Release() // Should not panic or block

		// Verify we can acquire again
		result := sem.Acquire()
		if !result {
			t.Error("Should be able to acquire after release")
		}
	})

	t.Run("multiple release", func(t *testing.T) {
		capacity := 3
		sem := NewSemaphore(capacity)

		// Acquire all permits
		for i := 0; i < capacity; i++ {
			sem.Acquire()
		}

		// Release all permits
		for i := 0; i < capacity; i++ {
			sem.Release()
		}

		// Verify we can acquire all again
		for i := 0; i < capacity; i++ {
			result := sem.Acquire()
			if !result {
				t.Errorf("Should be able to acquire permit %d after releases", i+1)
			}
		}
	})
}

func TestSemaphore_TryAcquire(t *testing.T) {
	t.Run("try acquire when permits available", func(t *testing.T) {
		sem := NewSemaphore(2)

		result1 := sem.TryAcquire()
		if !result1 {
			t.Error("TryAcquire should succeed when permits available")
		}

		result2 := sem.TryAcquire()
		if !result2 {
			t.Error("TryAcquire should succeed when permits still available")
		}
	})

	t.Run("try acquire when no permits available", func(t *testing.T) {
		sem := NewSemaphore(1)
		sem.Acquire() // Fill the semaphore

		result := sem.TryAcquire()
		if result {
			t.Error("TryAcquire should fail when no permits available")
		}
	})

	t.Run("try acquire after release", func(t *testing.T) {
		sem := NewSemaphore(1)
		sem.Acquire()

		// Should fail when full
		if sem.TryAcquire() {
			t.Error("TryAcquire should fail when semaphore is full")
		}

		sem.Release()

		// Should succeed after release
		if !sem.TryAcquire() {
			t.Error("TryAcquire should succeed after release")
		}
	})

	t.Run("try acquire zero capacity", func(t *testing.T) {
		sem := NewSemaphore(0)
		result := sem.TryAcquire()
		if result {
			t.Error("TryAcquire should fail on zero capacity semaphore")
		}
	})
}

func TestSemaphore_Concurrent(t *testing.T) {
	t.Run("concurrent acquire and release", func(t *testing.T) {
		capacity := 10
		sem := NewSemaphore(capacity)
		iterations := 100
		var wg sync.WaitGroup

		// Start multiple goroutines that acquire and release
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					sem.Acquire()
					time.Sleep(time.Microsecond) // Simulate some work
					sem.Release()
				}
			}()
		}

		wg.Wait()
		// Test should complete without deadlock
	})

	t.Run("concurrent try acquire", func(t *testing.T) {
		capacity := 5
		sem := NewSemaphore(capacity)
		iterations := 50
		successCount := int64(0)
		var mu sync.Mutex
		var wg sync.WaitGroup

		// Start multiple goroutines trying to acquire
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					if sem.TryAcquire() {
						mu.Lock()
						successCount++
						mu.Unlock()
						time.Sleep(time.Microsecond)
						sem.Release()
					}
				}
			}()
		}

		wg.Wait()

		if successCount == 0 {
			t.Error("Expected some successful acquisitions")
		}
	})

	t.Run("mixed acquire and try acquire", func(t *testing.T) {
		capacity := 3
		sem := NewSemaphore(capacity)
		var wg sync.WaitGroup

		// Goroutines using blocking acquire
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					sem.Acquire()
					time.Sleep(time.Millisecond)
					sem.Release()
				}
			}()
		}

		// Goroutines using non-blocking try acquire
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 20; j++ {
					if sem.TryAcquire() {
						time.Sleep(time.Millisecond)
						sem.Release()
					}
					time.Sleep(time.Millisecond / 2)
				}
			}()
		}

		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// Test completed successfully
		case <-time.After(5 * time.Second):
			t.Error("Test timed out, possible deadlock")
		}
	})
}

func TestSemaphore_EdgeCases(t *testing.T) {
	t.Run("release without prior acquire should be avoided", func(t *testing.T) {
		// This test documents that Release() without Acquire() will block
		// In practice, this should be avoided as it's not a valid use pattern
		sem := NewSemaphore(1)

		// First acquire, then we can safely release
		sem.Acquire()
		sem.Release() // Now this is safe

		// Verify normal operation
		result := sem.Acquire()
		if !result {
			t.Error("Should be able to acquire after proper acquire/release cycle")
		}
	})

	t.Run("multiple acquire and release cycles", func(t *testing.T) {
		sem := NewSemaphore(2)

		// Multiple cycles of acquire/release
		for i := 0; i < 5; i++ {
			sem.Acquire()
			sem.Acquire() // Fill to capacity
			sem.Release()
			sem.Release() // Empty completely
		}

		// Should still work normally
		if !sem.TryAcquire() {
			t.Error("Semaphore should be functional after multiple cycles")
		}
		sem.Release()
	})

	t.Run("stress test with high concurrency", func(t *testing.T) {
		capacity := 20
		sem := NewSemaphore(capacity)
		numGoroutines := 100
		iterationsPerGoroutine := 10
		var wg sync.WaitGroup

		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterationsPerGoroutine; j++ {
					if id%2 == 0 {
						// Half use blocking acquire
						sem.Acquire()
						sem.Release()
					} else {
						// Half use try acquire
						if sem.TryAcquire() {
							sem.Release()
						}
					}
				}
			}(i)
		}

		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			duration := time.Since(start)
			t.Logf("Stress test completed in %v", duration)
		case <-time.After(10 * time.Second):
			t.Error("Stress test timed out")
		}
	})
}

// Benchmark tests
func BenchmarkSemaphore_Acquire(b *testing.B) {
	sem := NewSemaphore(1000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sem.Acquire()
			sem.Release()
		}
	})
}

func BenchmarkSemaphore_TryAcquire(b *testing.B) {
	sem := NewSemaphore(1000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if sem.TryAcquire() {
				sem.Release()
			}
		}
	})
}
