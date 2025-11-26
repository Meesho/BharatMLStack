package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

const (
	BufferSize = 128 * 1024 * 1024 // 128 MB
	ChunkSize  = 8 * 1024 * 1024   // 8 MB

)

func main() {
	ctx := context.Background()

	// -----------------------------
	// 1. Client with gRPC pool
	// -----------------------------
	client, err := storage.NewGRPCClient(
		ctx,
		option.WithGRPCConnectionPool(32), // <— IMPORTANT
	)
	if err != nil {
		log.Fatalf("client init: %v", err)
	}

	bucketName := "gcs-dsci-srch-search-prd"
	objectName := fmt.Sprintf("/Image_search/gcs-flush/poc-upload-%d.bin", time.Now().UnixNano())

	// -----------------------------
	// 2. Prepare 128MB buffer
	// -----------------------------
	buf := make([]byte, BufferSize)
	_, _ = rand.Read(buf) // Fill with random bytes (optional)

	// -----------------------------
	// 3. Upload & Measure
	// -----------------------------
	fmt.Printf("Uploading %d MB buffer...\n", BufferSize/(1024*1024))

	latTotalStart := time.Now()

	w := client.Bucket(bucketName).Object(objectName).NewWriter(ctx)

	w.ChunkSize = ChunkSize

	var (
		chunkCount int
		totalBytes int64
	)

	for offset := 0; offset < len(buf); offset += ChunkSize {
		end := offset + ChunkSize
		if end > len(buf) {
			end = len(buf)
		}

		chunk := buf[offset:end]

		// Measure per-chunk latency
		chunkStart := time.Now()
		n, err := w.Write(chunk)
		chunkLatency := time.Since(chunkStart)

		if err != nil {
			log.Fatalf("write chunk: %v", err)
		}
		totalBytes += int64(n)
		chunkCount++

		throughput := float64(n) / chunkLatency.Seconds() / (1024 * 1024)

		fmt.Printf("Chunk %-3d | %-4d KB | %4d ms | %7.2f MB/s\n",
			chunkCount,
			n/1024,
			chunkLatency.Milliseconds(),
			throughput,
		)
	}

	if err := w.Close(); err != nil {
		log.Fatalf("close writer: %v", err)
	}

	latTotal := time.Since(latTotalStart)
	totalMB := float64(totalBytes) / (1024 * 1024)
	throughputTotal := totalMB / latTotal.Seconds()

	// -----------------------------
	// 4. Summary
	// -----------------------------
	fmt.Println("\n================ Summary ================")
	fmt.Printf("Object:          %s/%s\n", bucketName, objectName)
	fmt.Printf("Total Size:      %.2f MB\n", totalMB)
	fmt.Printf("Total Time:      %v\n", latTotal)
	fmt.Printf("Overall Speed:   %.2f MB/s\n", throughputTotal)
	fmt.Printf("Chunk Count:     %d\n", chunkCount)
	fmt.Println("=========================================")
}
