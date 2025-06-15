package blocks

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/persist"
	"google.golang.org/protobuf/proto"

	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
	"github.com/apache/arrow/go/v17/arrow/ipc"
	"github.com/apache/arrow/go/v17/arrow/memory"
)

func init() {
	system.Init()
}

// Feature group sizes to test
var featureSizes = []int{100, 1000, 10000, 100000}

// Get the global PSDBPool instance
var psdbPool = GetPSDBPool()

// Data generation
func generateInt32Data(size int) []int32 {
	data := make([]int32, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Int31()
	}
	return data
}

// PSDB serialization without compression
func serializeInt32PSDB(data []int32) ([]byte, error) {
	psdb := psdbPool.Get()
	defer psdbPool.Put(psdb)

	// Set up PSDB
	psdb.layoutVersion = 1
	psdb.dataType = types.DataTypeInt32
	psdb.compressionType = compression.TypeNone
	psdb.featureSchemaVersion = 1
	psdb.expiryAt = uint64(time.Now().Add(time.Hour).Unix())
	psdb.noOfFeatures = len(data)
	psdb.Data = data

	// Allocate buffers
	headerSize := PSDBLayout1LengthBytes
	dataSize := len(data) * 4 // 4 bytes per int32

	if cap(psdb.buf) < headerSize {
		psdb.buf = make([]byte, headerSize)
	} else {
		psdb.buf = psdb.buf[:headerSize]
	}

	if cap(psdb.originalData) < dataSize {
		psdb.originalData = make([]byte, dataSize)
	} else {
		psdb.originalData = psdb.originalData[:dataSize]
	}

	return psdb.Serialize()
}

// Protocol Buffers serialization
func serializeInt32Proto(data []int32) ([]byte, error) {
	// Use the Values message type with Int32Values field
	values := &persist.Values{
		Int32Values: data,
	}
	return proto.Marshal(values)
}

// Apache Arrow serialization
func serializeInt32Arrow(data []int32) ([]byte, error) {
	pool := memory.NewGoAllocator()

	schema := arrow.NewSchema([]arrow.Field{
		{Name: "values", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
	}, nil)

	builder := array.NewInt32Builder(pool)
	defer builder.Release()

	builder.AppendValues(data, nil)
	arr := builder.NewInt32Array()
	defer arr.Release()

	record := array.NewRecord(schema, []arrow.Array{arr}, int64(len(data)))
	defer record.Release()

	var buf bytes.Buffer
	writer := ipc.NewWriter(&buf, ipc.WithSchema(schema))
	defer writer.Close()

	err := writer.Write(record)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Benchmark PSDB serialization
func BenchmarkInt32SerializationPSDB(b *testing.B) {
	for _, size := range featureSizes {
		data := generateInt32Data(size)

		b.Run(fmt.Sprintf("PSDB/Size-%d", size), func(b *testing.B) {
			b.ReportAllocs()
			var serializedSize int

			for i := 0; i < b.N; i++ {
				serialized, err := serializeInt32PSDB(data)
				if err != nil {
					b.Fatal(err)
				}
				serializedSize = len(serialized)
			}

			b.ReportMetric(float64(serializedSize), "bytes")
		})
	}
}

// Benchmark Protocol Buffers serialization
func BenchmarkInt32SerializationProto3(b *testing.B) {
	for _, size := range featureSizes {
		data := generateInt32Data(size)

		b.Run(fmt.Sprintf("Proto3/Size-%d", size), func(b *testing.B) {
			b.ReportAllocs()
			var serializedSize int

			for i := 0; i < b.N; i++ {
				serialized, err := serializeInt32Proto(data)
				if err != nil {
					b.Fatal(err)
				}
				serializedSize = len(serialized)
			}

			b.ReportMetric(float64(serializedSize), "bytes")
		})
	}
}

// Benchmark Apache Arrow serialization
func BenchmarkInt32SerializationArrow(b *testing.B) {
	for _, size := range featureSizes {
		data := generateInt32Data(size)

		b.Run(fmt.Sprintf("Arrow/Size-%d", size), func(b *testing.B) {
			b.ReportAllocs()
			var serializedSize int

			for i := 0; i < b.N; i++ {
				serialized, err := serializeInt32Arrow(data)
				if err != nil {
					b.Fatal(err)
				}
				serializedSize = len(serialized)
			}

			b.ReportMetric(float64(serializedSize), "bytes")
		})
	}
}

// Comprehensive comparison benchmark
func BenchmarkInt32SerializationComparison(b *testing.B) {
	for _, size := range featureSizes {
		data := generateInt32Data(size)

		b.Run(fmt.Sprintf("Comparison/Size-%d", size), func(b *testing.B) {
			// PSDB
			b.Run("PSDB", func(b *testing.B) {
				b.ReportAllocs()
				var serializedSize int
				for i := 0; i < b.N; i++ {
					serialized, err := serializeInt32PSDB(data)
					if err != nil {
						b.Fatal(err)
					}
					serializedSize = len(serialized)
				}
				b.ReportMetric(float64(serializedSize), "bytes")
			})

			// Proto3
			b.Run("Proto3", func(b *testing.B) {
				b.ReportAllocs()
				var serializedSize int
				for i := 0; i < b.N; i++ {
					serialized, err := serializeInt32Proto(data)
					if err != nil {
						b.Fatal(err)
					}
					serializedSize = len(serialized)
				}
				b.ReportMetric(float64(serializedSize), "bytes")
			})

			// Arrow
			b.Run("Arrow", func(b *testing.B) {
				b.ReportAllocs()
				var serializedSize int
				for i := 0; i < b.N; i++ {
					serialized, err := serializeInt32Arrow(data)
					if err != nil {
						b.Fatal(err)
					}
					serializedSize = len(serialized)
				}
				b.ReportMetric(float64(serializedSize), "bytes")
			})
		})
	}
}

// Size-only comparison (single run for accurate size measurement)
func BenchmarkInt32SizeComparison(b *testing.B) {
	for _, size := range featureSizes {
		data := generateInt32Data(size)

		b.Run(fmt.Sprintf("SizeOnly/Size-%d", size), func(b *testing.B) {
			// PSDB size
			psdbSerialized, err := serializeInt32PSDB(data)
			if err != nil {
				b.Fatal(err)
			}

			// Proto3 size
			proto3Serialized, err := serializeInt32Proto(data)
			if err != nil {
				b.Fatal(err)
			}

			// Arrow size
			arrowSerialized, err := serializeInt32Arrow(data)
			if err != nil {
				b.Fatal(err)
			}

			// Report all sizes
			b.ReportMetric(float64(len(psdbSerialized)), "psdb_bytes")
			b.ReportMetric(float64(len(proto3Serialized)), "proto3_bytes")
			b.ReportMetric(float64(len(arrowSerialized)), "arrow_bytes")

			// Calculate compression ratios vs raw data
			rawSize := size * 4 // 4 bytes per int32
			b.ReportMetric(float64(rawSize), "raw_bytes")
			b.ReportMetric(float64(len(psdbSerialized))/float64(rawSize)*100, "psdb_ratio_pct")
			b.ReportMetric(float64(len(proto3Serialized))/float64(rawSize)*100, "proto3_ratio_pct")
			b.ReportMetric(float64(len(arrowSerialized))/float64(rawSize)*100, "arrow_ratio_pct")
		})
	}
}

// Memory efficiency benchmark
func BenchmarkInt32MemoryEfficiency(b *testing.B) {
	for _, size := range featureSizes {
		data := generateInt32Data(size)

		b.Run(fmt.Sprintf("Memory/Size-%d", size), func(b *testing.B) {
			// Test memory allocations for each format
			b.Run("PSDB_Pooled", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_, err := serializeInt32PSDB(data)
					if err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("Proto3", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_, err := serializeInt32Proto(data)
					if err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("Arrow", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_, err := serializeInt32Arrow(data)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

// Throughput benchmark (operations per second)
func BenchmarkInt32Throughput(b *testing.B) {
	size := 1000 // Fixed size for throughput measurement
	data := generateInt32Data(size)

	b.Run("Throughput", func(b *testing.B) {
		b.Run("PSDB", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(size * 4)) // Report throughput in bytes/sec
			for i := 0; i < b.N; i++ {
				_, err := serializeInt32PSDB(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run("Proto3", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(size * 4))
			for i := 0; i < b.N; i++ {
				_, err := serializeInt32Proto(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run("Arrow", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(size * 4))
			for i := 0; i < b.N; i++ {
				_, err := serializeInt32Arrow(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}
