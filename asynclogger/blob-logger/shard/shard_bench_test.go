package shard

import (
	"testing"
)

func BenchmarkShard_Write(b *testing.B) {
	const chunkSize = 1024 * 256 // 256 KB

	data := make([]byte, chunkSize)

	// Ensure the shard has enough capacity for all iterations so we stay on the fast path.
	shard := NewShard(chunkSize*b.N, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, full := shard.Write(data); full {
			b.Fatalf("shard reported full unexpectedly at iteration %d", i)
		}
	}
}

func BenchmarkShard_Write_concurrent(b *testing.B) {
	const chunkSize = 1024 * 256 // 256 KB payload

	data := make([]byte, chunkSize)

	// Allocate enough capacity for ALL writes across ALL goroutines.
	// RunParallel executes exactly b.N iterations total.
	shard := NewShard(chunkSize*b.N, 0)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Each Next() == one Write()
			if _, full := shard.Write(data); full {
				panic("shard reported full unexpectedly")
			}
		}
	})
}

func BenchmarkShardV2_Write(b *testing.B) {
	const chunkSize = 1024 * 256 // 256 KB

	data := make([]byte, chunkSize)

	// Ensure the shard has enough capacity for all iterations so we stay on the fast path.
	shard := NewShardV2(chunkSize*b.N, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, full, _ := shard.Write(data); full {
			b.Fatalf("shard reported full unexpectedly at iteration %d", i)
		}
	}
}

func BenchmarkShardV2_Write_concurrent(b *testing.B) {
	const chunkSize = 1024 * 256 // 256 KB payload

	data := make([]byte, chunkSize)

	// Allocate enough capacity for ALL writes across ALL goroutines.
	// RunParallel executes exactly b.N iterations total.
	shard := NewShardV2(chunkSize*b.N, 0)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Each Next() == one Write()
			if _, _, full, _ := shard.Write(data); full {
				panic("shard reported full unexpectedly")
			}
		}
	})
}

func BenchmarkShardV3_Write(b *testing.B) {
	const chunkSize = 1024 * 256 // 256 KB

	data := make([]byte, chunkSize)

	// Ensure the shard has enough capacity for all iterations so we stay on the fast path.
	shard := NewShardV3(chunkSize*b.N, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, full := shard.Write(data); full {
			b.Fatalf("shard reported full unexpectedly at iteration %d", i)
		}
	}
}

func BenchmarkShardV3_Write_concurrent(b *testing.B) {
	const chunkSize = 1024 * 256 // 256 KB payload

	data := make([]byte, chunkSize)

	// Allocate enough capacity for ALL writes across ALL goroutines.
	// RunParallel executes exactly b.N iterations total.
	shard := NewShardV3(chunkSize*b.N, 0)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Each Next() == one Write()
			if _, full := shard.Write(data); full {
				panic("shard reported full unexpectedly")
			}
		}
	})
}
