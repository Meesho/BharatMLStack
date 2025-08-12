package simdmap

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"testing"
)

func TestPutGet(t *testing.T) {
	m := New[int](1 << 10)

	// Insert 10 000 random keys.
	kvs := make([]uint64, 10_000)
	for i := range kvs {
		_ = binary.Read(crand.Reader, binary.LittleEndian, &kvs[i])
		m.Put(kvs[i], int(i))
	}

	// Verify all keys are present.
	for i, k := range kvs {
		v, ok := m.Get(k)
		if !ok || v != i {
			t.Fatalf("key %d lost: got (%d,%v)", k, v, ok)
		}
	}

	// Delete half, ensure they’re gone.
	for i := 0; i < len(kvs); i += 2 {
		m.Delete(kvs[i])
		if _, ok := m.Get(kvs[i]); ok {
			t.Fatalf("key %d should have been deleted", kvs[i])
		}
	}
}

func BenchmarkMixed_SIMDMap(b *testing.B) {

	//m := map[uint64]struct{}{}
	sm := New[struct{}](1_000_000)
	b.Run("simdmap-put", func(b *testing.B) {

		var h uint64
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = binary.Read(crand.Reader, binary.LittleEndian, &h)
			sm.Put(h, struct{}{})
		}
		b.StopTimer()
		b.ReportAllocs()
	})

	b.Run("simdmap-get", func(b *testing.B) {
		var h uint64
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = binary.Read(crand.Reader, binary.LittleEndian, &h)
			_, _ = sm.Get(h)
		}
		b.StopTimer()
		b.ReportAllocs()
	})

}

func BenchmarkMixed_GOMap(b *testing.B) {
	m := make(map[uint64]struct{}, 1_000_000)
	b.Run("map-put", func(b *testing.B) {
		var h uint64
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = binary.Read(crand.Reader, binary.LittleEndian, &h)
			m[h] = struct{}{}
		}
		b.StopTimer()
		b.ReportAllocs()
	})

	b.Run("map-get", func(b *testing.B) {

		var h uint64
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = binary.Read(crand.Reader, binary.LittleEndian, &h)
			_, _ = m[h]
		}
		b.StopTimer()
		b.ReportAllocs()
	})
}

func BenchmarkGet_Hit(b *testing.B) {
	m := New[struct{}](1 << 20)

	// Fill the map with 1 M random keys
	keys := make([]uint64, 1<<20)
	for i := range keys {
		_ = binary.Read(crand.Reader, binary.LittleEndian, &keys[i])
		m.Put(keys[i], struct{}{})
	}

	// Deterministic PRNG for benchmark loop
	rng := rand.New(rand.NewSource(42))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k := keys[rng.Intn(len(keys))]
		_, _ = m.Get(k)
	}
}

// -------- ultra‑cheap 64‑bit key generator (SplitMix64) -------------
var x uint64 = 0x9e3779b97f4a7c15

func next() uint64 {
	z := x + 0x9e3779b97f4a7c15
	x = z
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// ------------ reusable key slice: *zero* cost in hot loop -----------
const N = 1 << 20 // 1 048 576 keys
var keys [N]uint64

func init() {
	for i := range keys {
		keys[i] = next()
	}
}

// ----------------------- benchmarks ---------------------------------
func BenchmarkPutGet_SIMD(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := New[struct{}](N) // capacity == live set
		for _, k := range keys {
			m.Put(k, struct{}{})
		}
		for _, k := range keys {
			_, _ = m.Get(k)
		}
	}
}

func BenchmarkPutGet_Go(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := make(map[uint64]struct{}, N) // same load‑factor
		for _, k := range keys {
			m[k] = struct{}{}
		}
		for _, k := range keys {
			_, _ = m[k]
		}
	}
}
