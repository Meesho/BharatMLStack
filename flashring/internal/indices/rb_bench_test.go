package indices

import (
	"testing"
)

// BenchmarkRingBufferPush50M benchmarks pushing 50 million elements to the ring buffer
func BenchmarkRingBufferPush50M(b *testing.B) {
	rb := NewRingBuffer(1000, 50_000_000)

	b.ResetTimer()
	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rb.Add(&Entry{})
		}
	})
	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rb.Get(i)
		}
	})
}
