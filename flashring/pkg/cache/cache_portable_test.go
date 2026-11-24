// Portable Cache-API tests that exercise the public Put / Get / MGet /
// Delete shape against either the real Linux io_uring + O_DIRECT path or
// the non-Linux fakes in internal/fs / internal/iouring. The remaining
// O_DIRECT / io_uring-specific assertions stay in cache_test.go behind
// //go:build linux.

package cache

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/testutil/mountfs"
)

// portableConfig returns a Config that passes validate() and works against
// both the real and fake disk paths. Renamed from validConfig to avoid
// clashing with the helper of the same name in cache_test.go on Linux.
func portableConfig() Config {
	return Config{
		NumShards:             1,
		KeysPerShard:          1024,
		FileSize:              4 * 1024 * 1024, // 4 MiB
		MemtableSize:          64 * 1024,       // 64 KiB
		ReWriteScoreThreshold: 0.5,
		GridSearchEpsilon:     0.01,
		SampleDuration:        time.Minute,
		FreqBands:             []int{10, 100, 1000},
		RecencyBands:          []int{1000, 100, 10},
	}
}

// newPortableCache wires a WrapCache against either mountfs.New (which
// resolves to the loopback ext4 mount on Linux or t.TempDir() on darwin)
// and registers Close as a cleanup. The portable suite uses a long TTL so
// values stay live across the test — see cache_test.go testTTL for the
// rationale.
func newPortableCache(t *testing.T) *WrapCache {
	t.Helper()
	mount := mountfs.New(t)
	wc, err := NewWrapCache(portableConfig(), mount)
	if err != nil {
		t.Fatalf("NewWrapCache: %v", err)
	}
	t.Cleanup(func() { _ = wc.Close() })
	return wc
}

const portableTTL = 24 * time.Hour

// TestPortable_PutGet_RoundTrip exercises the basic Put/Get path through
// the public Cache interface.
func TestPortable_PutGet_RoundTrip(t *testing.T) {
	wc := newPortableCache(t)
	key, value := "portable-key", []byte("hello-portable")
	if err := wc.Put(key, value, portableTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, found, expired := wc.Get(key)
	if !found {
		t.Fatalf("Get found=false after Put")
	}
	if expired {
		t.Fatalf("Get expired=true with %s TTL", portableTTL)
	}
	if !bytes.Equal(got, value) {
		t.Fatalf("Get value = %q, want %q", got, value)
	}
}

// TestPortable_Get_UnknownKey asserts the cache returns a clean
// not-found / not-expired tuple for keys that were never inserted.
func TestPortable_Get_UnknownKey(t *testing.T) {
	wc := newPortableCache(t)
	got, found, expired := wc.Get("missing-key")
	if found {
		t.Fatalf("Get found=true for unknown key, got value %q", got)
	}
	if expired {
		t.Fatalf("Get expired=true for unknown key")
	}
}

// TestPortable_Delete_MakesKeyUnreachable inserts then deletes a key and
// asserts the public Get path no longer reports it as found.
func TestPortable_Delete_MakesKeyUnreachable(t *testing.T) {
	wc := newPortableCache(t)
	key, value := "to-delete", []byte("doomed")
	if err := wc.Put(key, value, portableTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if !wc.Delete(key) {
		t.Fatalf("Delete returned false for existing key")
	}
	if _, found, _ := wc.Get(key); found {
		t.Fatalf("Get still reports key as found after Delete")
	}
}

// TestPortable_MGet_AllKeysPresent inserts a small batch of keys and reads
// them back through the public MGet path, asserting every key reports as
// found with the expected value.
func TestPortable_MGet_AllKeysPresent(t *testing.T) {
	wc := newPortableCache(t)
	const N = 4
	keys := make([]string, N)
	values := make([][]byte, N)
	for i := 0; i < N; i++ {
		keys[i] = "mget-" + strconv.Itoa(i)
		values[i] = []byte("v-" + strconv.Itoa(i))
		if err := wc.Put(keys[i], values[i], portableTTL); err != nil {
			t.Fatalf("Put #%d: %v", i, err)
		}
	}
	results := wc.MGet(keys)
	if len(results) != N {
		t.Fatalf("MGet len=%d, want %d", len(results), N)
	}
	for i, res := range results {
		if !res.Found {
			t.Fatalf("MGet[%d] found=false", i)
		}
		if res.Expired {
			t.Fatalf("MGet[%d] expired=true with %s TTL", i, portableTTL)
		}
		if !bytes.Equal(res.Value, values[i]) {
			t.Fatalf("MGet[%d] value=%q, want %q", i, res.Value, values[i])
		}
	}
}

// TestPortable_MGet_EmptyKeys asserts MGet on an empty slice returns an
// empty result without panicking.
func TestPortable_MGet_EmptyKeys(t *testing.T) {
	wc := newPortableCache(t)
	results := wc.MGet(nil)
	if len(results) != 0 {
		t.Fatalf("MGet(nil) len=%d, want 0", len(results))
	}
}
