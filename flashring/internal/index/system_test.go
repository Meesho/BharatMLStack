package index

import (
	"encoding/binary"
	"testing"
)

func TestLoadByteOrder_PopulatesGlobal(t *testing.T) {
	if ByteOrder == nil {
		t.Fatal("ByteOrder is nil after package init")
	}
	if ByteOrder.ByteOrder == nil {
		t.Fatal("ByteOrder.ByteOrder is nil after loadByteOrder")
	}
	// On every platform this suite runs on, the embedded order must be one
	// of the two stdlib singletons — never the catch-all panic branch.
	if ByteOrder.ByteOrder != binary.LittleEndian && ByteOrder.ByteOrder != binary.BigEndian {
		t.Fatalf("ByteOrder.ByteOrder = %v, want LittleEndian or BigEndian", ByteOrder.ByteOrder)
	}
}

func TestLoadByteOrder_Idempotent_RestoresGlobal(t *testing.T) {
	prev := ByteOrder
	t.Cleanup(func() { ByteOrder = prev })

	loadByteOrder()

	if ByteOrder == nil {
		t.Fatal("ByteOrder is nil after second loadByteOrder")
	}
	if ByteOrder.ByteOrder != prev.ByteOrder {
		t.Fatalf("ByteOrder changed across calls: was %v, now %v", prev.ByteOrder, ByteOrder.ByteOrder)
	}
}

func TestCustomByteOrder_PutInt64_RoundTrips(t *testing.T) {
	buf := make([]byte, 8)
	for _, v := range []int64{0, 1, -1, 1 << 32, -(1 << 32), 1<<63 - 1, -1 << 63} {
		ByteOrder.PutInt64(buf, v)
		if got := ByteOrder.Int64(buf); got != v {
			t.Fatalf("Int64 round-trip for %d = %d", v, got)
		}
	}
}

func TestCustomByteOrder_PutInt32_RoundTrips(t *testing.T) {
	buf := make([]byte, 4)
	for _, v := range []int32{0, 1, -1, 1 << 16, -(1 << 16), 1<<31 - 1, -1 << 31} {
		ByteOrder.PutInt32(buf, v)
		if got := ByteOrder.Int32(buf); got != v {
			t.Fatalf("Int32 round-trip for %d = %d", v, got)
		}
	}
}

func TestCustomByteOrder_PutUint32_RoundTrips(t *testing.T) {
	buf := make([]byte, 4)
	for _, v := range []uint32{0, 1, 1 << 16, 1<<32 - 1} {
		ByteOrder.PutUint32(buf, v)
		if got := ByteOrder.Uint32(buf); got != v {
			t.Fatalf("Uint32 round-trip for %d = %d", v, got)
		}
	}
}
