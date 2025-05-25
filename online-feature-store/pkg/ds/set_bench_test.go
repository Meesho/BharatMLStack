package ds

import (
	"testing"
)

// Helper function to create a set with n elements
func createSetWithSize[T comparable](n int) Set[int] {
	s := NewOrderedSetWithCapacity[int](n)
	for i := 0; i < n; i++ {
		s.Add(i)
	}
	return s
}

// O(n) Operations

func BenchmarkIterator10(b *testing.B) {
	s := createSetWithSize[int](10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Iterator(func(k int, v interface{}) bool { return true })
	}
}

func BenchmarkIterator100(b *testing.B) {
	s := createSetWithSize[int](100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Iterator(func(k int, v interface{}) bool { return true })
	}
}

func BenchmarkIterator1000(b *testing.B) {
	s := createSetWithSize[int](1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Iterator(func(k int, v interface{}) bool { return true })
	}
}

func BenchmarkFastIterator10(b *testing.B) {
	s := createSetWithSize[int](10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.FastIterator(func(k int, v interface{}) {})
	}
}

func BenchmarkFastIterator100(b *testing.B) {
	s := createSetWithSize[int](100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.FastIterator(func(k int, v interface{}) {})
	}
}

func BenchmarkFastIterator1000(b *testing.B) {
	s := createSetWithSize[int](1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.FastIterator(func(k int, v interface{}) {})
	}
}

// O(n x m) Operations

func BenchmarkIntersection10(b *testing.B) {
	s1 := createSetWithSize[int](10)
	s2 := createSetWithSize[int](10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s1.Intersection(s2)
	}
}

func BenchmarkIntersection100(b *testing.B) {
	s1 := createSetWithSize[int](100)
	s2 := createSetWithSize[int](100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s1.Intersection(s2)
	}
}

func BenchmarkIntersection1000(b *testing.B) {
	s1 := createSetWithSize[int](1000)
	s2 := createSetWithSize[int](1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s1.Intersection(s2)
	}
}

func BenchmarkUnion10(b *testing.B) {
	s1 := createSetWithSize[int](10)
	s2 := createSetWithSize[int](10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s1.Union(s2)
	}
}

func BenchmarkUnion100(b *testing.B) {
	s1 := createSetWithSize[int](100)
	s2 := createSetWithSize[int](100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s1.Union(s2)
	}
}

func BenchmarkUnion1000(b *testing.B) {
	s1 := createSetWithSize[int](1000)
	s2 := createSetWithSize[int](1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s1.Union(s2)
	}
}

// Batch Operations

func BenchmarkAddBatch10(b *testing.B) {
	elements := make([]int, 10)
	for i := 0; i < 10; i++ {
		elements[i] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewOrderedSetWithCapacity[int](10)
		s.AddBatch(elements)
	}
}

func BenchmarkAddBatch100(b *testing.B) {
	elements := make([]int, 100)
	for i := 0; i < 100; i++ {
		elements[i] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewOrderedSetWithCapacity[int](100)
		s.AddBatch(elements)
	}
}

func BenchmarkAddBatch1000(b *testing.B) {
	elements := make([]int, 1000)
	for i := 0; i < 1000; i++ {
		elements[i] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewOrderedSetWithCapacity[int](1000)
		s.AddBatch(elements)
	}
}
