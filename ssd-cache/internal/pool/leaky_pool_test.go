package pool

import (
	"testing"
)

func TestNewLeakyPool(t *testing.T) {
	capacity := 5
	createFunc := func() interface{} { return "new_object" }

	pool := NewLeakyPool(capacity, createFunc)

	if pool.capacity != capacity {
		t.Errorf("Expected capacity %d, got %d", capacity, pool.capacity)
	}

	if pool.usage != 0 {
		t.Errorf("Expected initial usage 0, got %d", pool.usage)
	}

	if pool.idx != -1 {
		t.Errorf("Expected initial idx -1, got %d", pool.idx)
	}

	if len(pool.availabilityList) != capacity {
		t.Errorf("Expected availabilityList length %d, got %d", capacity, len(pool.availabilityList))
	}

	if pool.createFunc == nil {
		t.Error("Expected createFunc to be set")
	}

	if pool.preDrefHook != nil {
		t.Error("Expected preDrefHook to be nil initially")
	}
}

func TestLeakyPool_RegisterPreDrefHook(t *testing.T) {
	pool := NewLeakyPool(2, func() interface{} { return "object" })

	called := false
	hook := func(obj interface{}) {
		called = true
	}

	pool.RegisterPreDrefHook(hook)

	if pool.preDrefHook == nil {
		t.Error("Expected preDrefHook to be set")
	}

	// Test that hook is actually called when needed
	// Fill pool completely: get 3 objects, put back 2 to fill the pool
	obj1, _ := pool.Get()
	obj2, _ := pool.Get()
	obj3, _ := pool.Get()
	pool.Put(obj1) // idx becomes 0
	pool.Put(obj2) // idx becomes 1, pool is now full

	// Now pool is full (idx=1, capacity=2), putting another should trigger hook
	pool.Put(obj3) // This should trigger the hook because idx will become 2 == capacity

	if !called {
		t.Error("Expected preDrefHook to be called when pool is full")
	}
}

func TestLeakyPool_Get_EmptyPool_WithinCapacity(t *testing.T) {
	capacity := 3
	createFunc := func() interface{} { return "new_object" }
	pool := NewLeakyPool(capacity, createFunc)

	// First get when pool is empty and usage is within capacity
	obj, leaked := pool.Get()

	if obj != "new_object" {
		t.Errorf("Expected 'new_object', got %v", obj)
	}

	if leaked {
		t.Error("Expected not leaked when usage is within capacity")
	}

	if pool.usage != 1 {
		t.Errorf("Expected usage 1, got %d", pool.usage)
	}

	if pool.idx != -1 {
		t.Errorf("Expected idx still -1, got %d", pool.idx)
	}
}

func TestLeakyPool_Get_EmptyPool_ExceedsCapacity(t *testing.T) {
	capacity := 2
	createFunc := func() interface{} { return "leaked_object" }
	pool := NewLeakyPool(capacity, createFunc)

	// Get objects to exceed capacity
	pool.Get()                // usage = 1
	pool.Get()                // usage = 2
	obj, leaked := pool.Get() // usage = 3, exceeds capacity

	if obj != "leaked_object" {
		t.Errorf("Expected 'leaked_object', got %v", obj)
	}

	if !leaked {
		t.Error("Expected leaked when usage exceeds capacity")
	}

	if pool.usage != 3 {
		t.Errorf("Expected usage 3, got %d", pool.usage)
	}
}

func TestLeakyPool_Get_FromAvailabilityList(t *testing.T) {
	capacity := 3
	createFunc := func() interface{} { return "new_object" }
	pool := NewLeakyPool(capacity, createFunc)

	// First, get and put back objects to populate the availability list
	obj1, _ := pool.Get()
	obj2, _ := pool.Get()

	pool.Put(obj1)
	pool.Put(obj2)

	// Now pool.idx should be 1 (0-indexed, 2 objects in pool)
	if pool.idx != 1 {
		t.Errorf("Expected idx 1, got %d", pool.idx)
	}

	// Get object from availability list
	retrievedObj, leaked := pool.Get()

	if retrievedObj != obj2 { // Should get the last put object
		t.Errorf("Expected obj2, got %v", retrievedObj)
	}

	if leaked {
		t.Error("Expected not leaked when getting from availability list")
	}

	if pool.idx != 0 {
		t.Errorf("Expected idx decremented to 0, got %d", pool.idx)
	}
}

func TestLeakyPool_Put_NormalCase(t *testing.T) {
	capacity := 3
	createFunc := func() interface{} { return "object" }
	pool := NewLeakyPool(capacity, createFunc)

	// Get an object first to increase usage
	obj, _ := pool.Get()
	initialUsage := pool.usage

	// Put it back
	pool.Put(obj)

	if pool.usage != initialUsage-1 {
		t.Errorf("Expected usage decremented to %d, got %d", initialUsage-1, pool.usage)
	}

	if pool.idx != 0 {
		t.Errorf("Expected idx incremented to 0, got %d", pool.idx)
	}

	if pool.availabilityList[0] != obj {
		t.Errorf("Expected object stored at index 0, got %v", pool.availabilityList[0])
	}
}

func TestLeakyPool_Put_PoolFull_WithoutHook(t *testing.T) {
	capacity := 2
	createFunc := func() interface{} { return "object" }
	pool := NewLeakyPool(capacity, createFunc)

	// Fill the pool completely
	obj1, _ := pool.Get()
	obj2, _ := pool.Get()
	obj3, _ := pool.Get()
	pool.Put(obj1) // idx becomes 0
	pool.Put(obj2) // idx becomes 1, pool is now full

	// Now pool is full (idx = 1, capacity = 2)
	initialUsage := pool.usage

	// Try to put another object when pool is full
	pool.Put(obj3)

	// Object should be discarded, usage decremented, idx should remain at capacity-1
	if pool.usage != initialUsage-1 {
		t.Errorf("Expected usage %d, got %d", initialUsage-1, pool.usage)
	}

	if pool.idx != capacity-1 {
		t.Errorf("Expected idx %d, got %d", capacity-1, pool.idx)
	}
}

func TestLeakyPool_Put_PoolFull_WithHook(t *testing.T) {
	capacity := 2
	createFunc := func() interface{} { return "object" }
	pool := NewLeakyPool(capacity, createFunc)

	var hookedObject interface{}
	hook := func(obj interface{}) {
		hookedObject = obj
	}
	pool.RegisterPreDrefHook(hook)

	// Fill the pool completely: get 3 objects, put back 2 to fill the pool
	obj1, _ := pool.Get()
	obj2, _ := pool.Get()
	obj3, _ := pool.Get()
	pool.Put(obj1) // idx becomes 0
	pool.Put(obj2) // idx becomes 1, pool is now full

	// Try to put another object when pool is full (should trigger hook)
	pool.Put(obj3) // This should trigger the hook because idx will become 2 == capacity

	if hookedObject != obj3 {
		t.Errorf("Expected hook to be called with obj3, got %v", hookedObject)
	}
}

func TestLeakyPool_UsageTracking(t *testing.T) {
	capacity := 3
	createFunc := func() interface{} { return "object" }
	pool := NewLeakyPool(capacity, createFunc)

	// Test usage increases with Get
	if pool.usage != 0 {
		t.Errorf("Expected initial usage 0, got %d", pool.usage)
	}

	obj1, _ := pool.Get()
	if pool.usage != 1 {
		t.Errorf("Expected usage 1 after first Get, got %d", pool.usage)
	}

	obj2, _ := pool.Get()
	if pool.usage != 2 {
		t.Errorf("Expected usage 2 after second Get, got %d", pool.usage)
	}

	// Test usage decreases with Put
	pool.Put(obj1)
	if pool.usage != 1 {
		t.Errorf("Expected usage 1 after first Put, got %d", pool.usage)
	}

	pool.Put(obj2)
	if pool.usage != 0 {
		t.Errorf("Expected usage 0 after second Put, got %d", pool.usage)
	}
}

func TestLeakyPool_IndexTracking(t *testing.T) {
	capacity := 3
	createFunc := func() interface{} { return "object" }
	pool := NewLeakyPool(capacity, createFunc)

	// Initial state
	if pool.idx != -1 {
		t.Errorf("Expected initial idx -1, got %d", pool.idx)
	}

	// Add objects to pool
	obj1, _ := pool.Get()
	obj2, _ := pool.Get()
	obj3, _ := pool.Get()

	pool.Put(obj1) // idx should become 0
	if pool.idx != 0 {
		t.Errorf("Expected idx 0 after first Put, got %d", pool.idx)
	}

	pool.Put(obj2) // idx should become 1
	if pool.idx != 1 {
		t.Errorf("Expected idx 1 after second Put, got %d", pool.idx)
	}

	pool.Put(obj3) // idx should become 2
	if pool.idx != 2 {
		t.Errorf("Expected idx 2 after third Put, got %d", pool.idx)
	}

	// Get objects back
	pool.Get() // idx should become 1
	if pool.idx != 1 {
		t.Errorf("Expected idx 1 after first Get, got %d", pool.idx)
	}

	pool.Get() // idx should become 0
	if pool.idx != 0 {
		t.Errorf("Expected idx 0 after second Get, got %d", pool.idx)
	}

	pool.Get() // idx should become -1
	if pool.idx != -1 {
		t.Errorf("Expected idx -1 after third Get, got %d", pool.idx)
	}
}

func TestLeakyPool_CreateFuncCalled(t *testing.T) {
	capacity := 2
	callCount := 0
	createFunc := func() interface{} {
		callCount++
		return callCount
	}
	pool := NewLeakyPool(capacity, createFunc)

	// Get first object
	obj1, _ := pool.Get()
	if callCount != 1 {
		t.Errorf("Expected createFunc called once, got %d times", callCount)
	}
	if obj1 != 1 {
		t.Errorf("Expected first object to be 1, got %v", obj1)
	}

	// Get second object
	obj2, _ := pool.Get()
	if callCount != 2 {
		t.Errorf("Expected createFunc called twice, got %d times", callCount)
	}
	if obj2 != 2 {
		t.Errorf("Expected second object to be 2, got %v", obj2)
	}

	// Put back and get again - should not call createFunc
	pool.Put(obj1)
	obj3, _ := pool.Get()
	if callCount != 2 {
		t.Errorf("Expected createFunc still called twice, got %d times", callCount)
	}
	if obj3 != 1 { // Should get back obj1
		t.Errorf("Expected to get back obj1 (value 1), got %v", obj3)
	}
}
