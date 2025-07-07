package pool

import (
	"testing"
)

// Test basic CustomPool functionality
func TestCustomPool_Basic(t *testing.T) {
	// Create a simple factory function
	newFunc := func() interface{} {
		return make([]byte, 1024)
	}

	pool := NewCustomPool(newFunc)

	// Test initial count (should be 0 - no objects in use)
	if count := pool.Count(); count != 0 {
		t.Errorf("Expected initial count 0, got %d", count)
	}

	// Test Get operation
	obj := pool.Get()
	if obj == nil {
		t.Error("Get() returned nil")
	}

	// Check that object is of expected type
	if buffer, ok := obj.([]byte); ok {
		if len(buffer) != 1024 {
			t.Errorf("Expected buffer length 1024, got %d", len(buffer))
		}
	} else {
		t.Error("Get() did not return expected []byte type")
	}

	// Test count after Get (should be 1 - one object in use)
	if count := pool.Count(); count != 1 {
		t.Errorf("Expected count 1 after Get(), got %d", count)
	}

	// Test Put operation
	pool.Put(obj)

	// Test count after Put (should be 0 - no objects in use)
	if count := pool.Count(); count != 0 {
		t.Errorf("Expected count 0 after Put(), got %d", count)
	}
}

// Test CustomPool with different factory functions
func TestCustomPool_DifferentFactories(t *testing.T) {
	// Test with string factory
	stringPool := NewCustomPool(func() interface{} {
		return "test string"
	})

	obj := stringPool.Get()
	if str, ok := obj.(string); ok {
		if str != "test string" {
			t.Errorf("Expected 'test string', got '%s'", str)
		}
	} else {
		t.Error("String factory did not return string")
	}
	stringPool.Put(obj)

	// Test with struct factory
	type TestStruct struct {
		Value int
	}

	structPool := NewCustomPool(func() interface{} {
		return &TestStruct{Value: 42}
	})

	obj2 := structPool.Get()
	if ts, ok := obj2.(*TestStruct); ok {
		if ts.Value != 42 {
			t.Errorf("Expected struct Value 42, got %d", ts.Value)
		}
	} else {
		t.Error("Struct factory did not return *TestStruct")
	}
	structPool.Put(obj2)
}

// Test multiple Get/Put operations
func TestCustomPool_MultipleOperations(t *testing.T) {
	pool := NewCustomPool(func() interface{} {
		return make([]int, 10)
	})

	const numOperations = 100
	objects := make([]interface{}, numOperations)

	// Get multiple objects
	for i := 0; i < numOperations; i++ {
		objects[i] = pool.Get()
		if objects[i] == nil {
			t.Errorf("Get() returned nil at iteration %d", i)
		}
	}

	// Check count (should equal numOperations - all objects are in use)
	if count := pool.Count(); count != numOperations {
		t.Errorf("Expected count %d after Gets, got %d", numOperations, count)
	}

	// Put all objects back
	for i := 0; i < numOperations; i++ {
		pool.Put(objects[i])
	}

	// Check final count (should be 0 - no objects in use)
	if count := pool.Count(); count != 0 {
		t.Errorf("Expected count 0 after Puts, got %d", count)
	}
}

// Test pool reuse behavior
func TestCustomPool_Reuse(t *testing.T) {
	callCount := 0
	pool := NewCustomPool(func() interface{} {
		callCount++
		return &struct{ ID int }{ID: callCount}
	})

	// Get an object
	obj1 := pool.Get()
	firstID := obj1.(*struct{ ID int }).ID

	if callCount != 1 {
		t.Errorf("Expected factory to be called once, called %d times", callCount)
	}

	// Put it back
	pool.Put(obj1)

	// Get another object - should reuse the same one
	obj2 := pool.Get()
	secondID := obj2.(*struct{ ID int }).ID

	if firstID != secondID {
		t.Errorf("Expected object reuse, got different IDs: %d vs %d", firstID, secondID)
	}

	if callCount != 1 {
		t.Errorf("Expected factory to still be called only once, called %d times", callCount)
	}

	pool.Put(obj2)
}

// Test that Count correctly tracks objects in use
func TestCustomPool_CountTracking(t *testing.T) {
	pool := NewCustomPool(func() interface{} {
		return make([]byte, 100)
	})

	// Initially should be 0
	if count := pool.Count(); count != 0 {
		t.Errorf("Expected initial count 0, got %d", count)
	}

	// Get 5 objects
	objects := make([]interface{}, 5)
	for i := 0; i < 5; i++ {
		objects[i] = pool.Get()
		expectedCount := int64(i + 1)
		if count := pool.Count(); count != expectedCount {
			t.Errorf("After Get %d: expected count %d, got %d", i+1, expectedCount, count)
		}
	}

	// Put back 3 objects
	for i := 0; i < 3; i++ {
		pool.Put(objects[i])
		expectedCount := int64(5 - i - 1)
		if count := pool.Count(); count != expectedCount {
			t.Errorf("After Put %d: expected count %d, got %d", i+1, expectedCount, count)
		}
	}

	// Should have 2 objects still in use
	if count := pool.Count(); count != 2 {
		t.Errorf("Expected final count 2, got %d", count)
	}

	// Put back remaining objects
	pool.Put(objects[3])
	pool.Put(objects[4])

	// Should be back to 0
	if count := pool.Count(); count != 0 {
		t.Errorf("Expected final count 0, got %d", count)
	}
}

// Test edge case: factory function that returns nil
func TestCustomPool_FactoryReturnsNil(t *testing.T) {
	pool := NewCustomPool(func() interface{} {
		return nil
	})

	obj := pool.Get()
	if obj != nil {
		t.Errorf("Expected nil from factory, got %v", obj)
	}

	// Should still track count correctly
	if count := pool.Count(); count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	pool.Put(obj)
	if count := pool.Count(); count != 0 {
		t.Errorf("Expected count 0 after Put, got %d", count)
	}
}

// Benchmark basic Get/Put operations
func BenchmarkCustomPool_GetPut(b *testing.B) {
	pool := NewCustomPool(func() interface{} {
		return make([]byte, 1024)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obj := pool.Get()
		pool.Put(obj)
	}
}

// Benchmark against direct allocation
func BenchmarkCustomPool_vs_DirectAllocation(b *testing.B) {
	pool := NewCustomPool(func() interface{} {
		return make([]byte, 1024)
	})
	b.Run("CustomPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			obj := pool.Get()
			pool.Put(obj)
		}
	})

	b.Run("DirectAllocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			x := make([]byte, 1024)
			_ = x
		}
	})
}

// Example usage demonstration
func ExampleCustomPool() {
	// Create a pool for byte slices
	pool := NewCustomPool(func() interface{} {
		return make([]byte, 1024) // 1KB buffers
	})

	// Get a buffer from the pool
	buffer := pool.Get().([]byte)

	// Use the buffer
	copy(buffer, []byte("Hello, CustomPool!"))

	// Check current usage count (should be 1)
	count := pool.Count()
	_ = count // count will be 1 while buffer is in use

	// Return it to the pool for reuse
	pool.Put(buffer)

	// Check count again (should be 0)
	count = pool.Count()
	_ = count // count will be 0 after Put
}
