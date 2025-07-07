package pool

import (
	"testing"
)

// Simple object for benchmarking
type benchObject struct {
	data [64]byte // 64 bytes of data
}

func createBenchObject() interface{} {
	return &benchObject{}
}

// BenchmarkLeakyPool_Get_NewObjects benchmarks getting objects when pool is empty
func BenchmarkLeakyPool_Get_NewObjects(b *testing.B) {
	pool := NewLeakyPool(100, createBenchObject)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pool.Get()
	}
}

// BenchmarkLeakyPool_Get_FromPool benchmarks getting objects from pool availability list
func BenchmarkLeakyPool_Get_FromPool(b *testing.B) {
	pool := NewLeakyPool(100, createBenchObject)

	// Pre-populate the pool
	objects := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		objects[i], _ = pool.Get()
	}
	for i := 0; i < 100; i++ {
		pool.Put(objects[i])
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		obj, _ := pool.Get()
		pool.Put(obj) // Put it back for next iteration
	}
}

// BenchmarkLeakyPool_Put benchmarks putting objects back to pool
func BenchmarkLeakyPool_Put(b *testing.B) {
	pool := NewLeakyPool(b.N, createBenchObject)

	// Pre-create objects
	objects := make([]interface{}, b.N)
	for i := 0; i < b.N; i++ {
		objects[i], _ = pool.Get()
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pool.Put(objects[i])
	}
}

// BenchmarkLeakyPool_GetPut_Cycle benchmarks realistic Get/Put cycles
func BenchmarkLeakyPool_GetPut_Cycle(b *testing.B) {
	pool := NewLeakyPool(100, createBenchObject)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		obj, _ := pool.Get()
		pool.Put(obj)
	}
}

// BenchmarkLeakyPool_Burst_Operations benchmarks burst of operations
func BenchmarkLeakyPool_Burst_Operations(b *testing.B) {
	pool := NewLeakyPool(100, createBenchObject)

	b.ResetTimer()
	b.ReportAllocs()

	// Simulate burst workload - each operation is either get or put
	objects := make([]interface{}, 50)
	objCount := 0

	for i := 0; i < b.N; i++ {
		if i%2 == 0 { // Get operation
			if objCount < len(objects) {
				objects[objCount], _ = pool.Get()
				objCount++
			} else {
				pool.Get() // Will create new object
			}
		} else { // Put operation
			if objCount > 0 {
				objCount--
				pool.Put(objects[objCount])
			}
		}
	}
}

// BenchmarkLeakyPool_Get_ExceedsCapacity benchmarks behavior when usage exceeds capacity
func BenchmarkLeakyPool_Get_ExceedsCapacity(b *testing.B) {
	pool := NewLeakyPool(10, createBenchObject) // Small capacity

	// Fill beyond capacity
	for i := 0; i < 20; i++ {
		pool.Get()
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pool.Get() // These should all be "leaked" objects
	}
}

// BenchmarkLeakyPool_Put_FullPool benchmarks putting objects when pool is full
func BenchmarkLeakyPool_Put_FullPool(b *testing.B) {
	capacity := 100
	pool := NewLeakyPool(capacity, createBenchObject)

	// Fill the pool completely
	objects := make([]interface{}, capacity*2)
	for i := 0; i < capacity*2; i++ {
		objects[i], _ = pool.Get()
	}

	// Fill the pool
	for i := 0; i < capacity; i++ {
		pool.Put(objects[i])
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Now benchmark putting objects when pool is full
	for i := 0; i < b.N; i++ {
		pool.Put(objects[capacity+i%capacity])
	}
}

// BenchmarkLeakyPool_Put_WithHook benchmarks putting objects with preDrefHook
func BenchmarkLeakyPool_Put_WithHook(b *testing.B) {
	capacity := 100
	pool := NewLeakyPool(capacity, createBenchObject)

	// Register a simple hook
	pool.RegisterPreDrefHook(func(obj interface{}) {
		// Simple hook that does minimal work
		_ = obj
	})

	// Fill the pool completely
	objects := make([]interface{}, capacity*2)
	for i := 0; i < capacity*2; i++ {
		objects[i], _ = pool.Get()
	}

	// Fill the pool
	for i := 0; i < capacity; i++ {
		pool.Put(objects[i])
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Now benchmark putting objects when pool is full (will trigger hook)
	for i := 0; i < b.N; i++ {
		pool.Put(objects[capacity+i%capacity])
	}
}

// BenchmarkLeakyPool_Mixed_Operations benchmarks mixed realistic workload
func BenchmarkLeakyPool_Mixed_Operations(b *testing.B) {
	pool := NewLeakyPool(50, createBenchObject)

	b.ResetTimer()
	b.ReportAllocs()

	objects := make([]interface{}, 10)
	objCount := 0

	for i := 0; i < b.N; i++ {
		if i%3 == 0 && objCount > 0 {
			// Put back an object
			objCount--
			pool.Put(objects[objCount])
		} else {
			// Get an object
			if objCount < len(objects) {
				objects[objCount], _ = pool.Get()
				objCount++
			} else {
				// Just get and discard
				pool.Get()
			}
		}
	}
}

// BenchmarkLeakyPool_CreateFunc_Expensive benchmarks with expensive object creation
func BenchmarkLeakyPool_CreateFunc_Expensive(b *testing.B) {
	expensiveCreateFunc := func() interface{} {
		// Simulate expensive object creation with allocation
		obj := &benchObject{}
		// Do some work to simulate expensive creation
		for i := 0; i < 100; i++ {
			obj.data[i%64] = byte(i)
		}
		return obj
	}

	pool := NewLeakyPool(100, expensiveCreateFunc)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		obj, _ := pool.Get()
		pool.Put(obj)
	}
}

// BenchmarkLeakyPool_Different_Capacities benchmarks different pool capacities
func BenchmarkLeakyPool_Capacity_10(b *testing.B) {
	benchmarkCapacity(b, 10)
}

func BenchmarkLeakyPool_Capacity_100(b *testing.B) {
	benchmarkCapacity(b, 100)
}

func BenchmarkLeakyPool_Capacity_1000(b *testing.B) {
	benchmarkCapacity(b, 1000)
}

func benchmarkCapacity(b *testing.B, capacity int) {
	pool := NewLeakyPool(capacity, createBenchObject)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		obj, _ := pool.Get()
		pool.Put(obj)
	}
}

// BenchmarkLeakyPool_MemoryUsage benchmarks memory usage patterns
func BenchmarkLeakyPool_MemoryUsage(b *testing.B) {
	pool := NewLeakyPool(1000, createBenchObject)

	// Pre-warm the pool
	objects := make([]interface{}, 500)
	for i := 0; i < 500; i++ {
		objects[i], _ = pool.Get()
	}
	for i := 0; i < 500; i++ {
		pool.Put(objects[i])
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(benchObject{}.data))) // Track bytes per operation

	for i := 0; i < b.N; i++ {
		obj, _ := pool.Get()
		pool.Put(obj)
	}
}

// BenchmarkLeakyPool_High_Usage simulates high usage scenario with small pool
func BenchmarkLeakyPool_High_Usage(b *testing.B) {
	pool := NewLeakyPool(10, createBenchObject) // Small pool, high usage

	b.ResetTimer()
	b.ReportAllocs()

	// Simulate high usage with random patterns
	objects := make([]interface{}, 20)
	objCount := 0

	for i := 0; i < b.N; i++ {
		if i%7 < 4 { // Get objects more frequently
			if objCount < len(objects) {
				objects[objCount], _ = pool.Get()
				objCount++
			} else {
				// Pool exhausted, get anyway (will leak)
				pool.Get()
			}
		} else { // Put objects back
			if objCount > 0 {
				objCount--
				pool.Put(objects[objCount])
			}
		}
	}
}
