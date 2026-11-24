package pools

import (
	"sync"
	"sync/atomic"
	"testing"
)

func newIntPool(capacity int, seed *int) *LeakyPool[int] {
	return NewLeakyPool(LeakyPoolConfig[int]{
		Capacity: capacity,
		Meta:     "test-meta",
		CreateFunc: func() int {
			*seed++
			return *seed
		},
	})
}

func TestNewLeakyPool_InitialState(t *testing.T) {
	seed := 0
	p := newIntPool(4, &seed)

	if got, want := p.capacity, 4; got != want {
		t.Fatalf("capacity = %d, want %d", got, want)
	}
	if got, want := p.idx, -1; got != want {
		t.Fatalf("idx = %d, want %d", got, want)
	}
	if got, want := p.usage, 0; got != want {
		t.Fatalf("usage = %d, want %d", got, want)
	}
	if got, want := len(p.available), 4; got != want {
		t.Fatalf("len(available) = %d, want %d", got, want)
	}
	if p.Meta != "test-meta" {
		t.Fatalf("Meta = %v, want test-meta", p.Meta)
	}
}

func TestGet_EmptyPool_CallsCreateFunc(t *testing.T) {
	seed := 0
	p := newIntPool(2, &seed)

	v := p.Get()

	if v != 1 {
		t.Fatalf("first Get = %d, want 1", v)
	}
	if p.usage != 1 {
		t.Fatalf("usage = %d, want 1", p.usage)
	}
}

func TestGet_AfterPut_ReturnsFromPool(t *testing.T) {
	seed := 0
	p := newIntPool(2, &seed)

	a := p.Get() // 1 from createFunc
	p.Put(a)
	b := p.Get() // should come from available, not createFunc

	if b != 1 {
		t.Fatalf("Get after Put = %d, want 1", b)
	}
	if p.idx != -1 {
		t.Fatalf("idx after consuming buffered item = %d, want -1", p.idx)
	}
	if seed != 1 {
		t.Fatalf("createFunc called %d times, want 1", seed)
	}
}

func TestPut_WithinCapacity_KeepsObject(t *testing.T) {
	seed := 0
	p := newIntPool(3, &seed)

	a := p.Get()
	b := p.Get()
	p.Put(a)
	p.Put(b)

	if p.idx != 1 {
		t.Fatalf("idx = %d, want 1", p.idx)
	}
	if p.usage != 0 {
		t.Fatalf("usage = %d, want 0", p.usage)
	}
}

func TestPut_OverCapacity_DropsExcess(t *testing.T) {
	seed := 0
	p := newIntPool(2, &seed)

	// Manually fill the pool to capacity to simulate the overflow branch.
	p.Put(10)
	p.Put(20)
	if p.idx != 1 {
		t.Fatalf("idx = %d after filling, want 1", p.idx)
	}

	// The third Put should be dropped.
	p.Put(30)
	if p.idx != 1 {
		t.Fatalf("idx = %d after overflow Put, want 1 (drop)", p.idx)
	}
	if p.available[0] != 10 || p.available[1] != 20 {
		t.Fatalf("available corrupted: %v, want [10 20]", p.available)
	}
}

func TestPut_OverCapacity_InvokesPreDrefHook(t *testing.T) {
	seed := 0
	p := newIntPool(1, &seed)

	var hookCalls atomic.Int32
	var hookGot int
	p.RegisterPreDrefHook(func(obj int) {
		hookCalls.Add(1)
		hookGot = obj
	})

	p.Put(7)  // fits (idx 0)
	p.Put(99) // overflow → hook runs with 99

	if got, want := hookCalls.Load(), int32(1); got != want {
		t.Fatalf("hook calls = %d, want %d", got, want)
	}
	if hookGot != 99 {
		t.Fatalf("hook arg = %d, want 99", hookGot)
	}
	if p.idx != 0 {
		t.Fatalf("idx after overflow with hook = %d, want 0", p.idx)
	}
}

func TestPut_OverCapacity_WithoutHook_NoPanic(t *testing.T) {
	seed := 0
	p := newIntPool(1, &seed)

	p.Put(7)
	// No hook registered; overflow must not panic.
	p.Put(99)

	if p.idx != 0 {
		t.Fatalf("idx = %d, want 0", p.idx)
	}
}

func TestGetPut_Concurrent_NoRace(t *testing.T) {
	var seed int64
	p := NewLeakyPool(LeakyPoolConfig[int]{
		Capacity: 16,
		CreateFunc: func() int {
			return int(atomic.AddInt64(&seed, 1))
		},
	})

	var wg sync.WaitGroup
	const workers = 8
	const iters = 200

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				v := p.Get()
				p.Put(v)
			}
		}()
	}
	wg.Wait()

	if p.usage != 0 {
		t.Fatalf("usage = %d after balanced Get/Put, want 0", p.usage)
	}
}
