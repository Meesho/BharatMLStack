package shard

import (
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkShardHandler_Write_concurrent(b *testing.B) {
	const chunkSize = 1024 * 256 // 256 KB payload

	data := make([]byte, chunkSize)

	// Allocate enough capacity for ALL writes across ALL goroutines.
	// RunParallel executes exactly b.N iterations total.
	shardHandler := NewShardHandler(4, chunkSize*1024)

	b.ResetTimer()
	// for i := 0; i < b.N; i++ {
	// 	if !shardHandler.Write(data) {
	// 		panic("shard reported full unexpectedly")
	// 	}
	// }

	total := atomic.Int64{}
	success := atomic.Int64{}
	failure := atomic.Int64{}
	duration := atomic.Int64{}
	b.RunParallel(func(pb *testing.PB) {
		successCount := 0
		failureCount := 0
		totalCount := 0
		startTime := time.Now()
		for pb.Next() {
			// Each Next() == one Write()
			if !shardHandler.Write(data) {
				failureCount++
			} else {
				successCount++
			}
			totalCount++
		}
		duration.Add(int64(time.Since(startTime).Nanoseconds()))
		total.Add(int64(totalCount))
		success.Add(int64(successCount))
		failure.Add(int64(failureCount))
		b.Logf("successCount: %d", successCount)
		b.Logf("failureCount: %d", failureCount)
		b.Logf("totalCount: %d", totalCount)
	})

	b.Logf("total: %d", total.Load())
	b.Logf("success: %d", success.Load())
	b.Logf("failure: %d", failure.Load())
	b.Logf("throughput: %d", total.Load()*chunkSize*1000000000/duration.Load())
}
