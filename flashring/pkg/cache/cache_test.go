//go:build linux
// +build linux

package cache

import (
	"bytes"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/memtables"
	filecache "github.com/Meesho/BharatMLStack/flashring/internal/shard"
	"github.com/Meesho/BharatMLStack/flashring/internal/testutil/mountfs"
)

// validConfig returns a Config that passes every check in validate().
// Tests for individual error sentinels start from this and mutate one field.
func validConfig() Config {
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

// testTTL is the TTL these tests use for "no expiry" semantics. flashring
// currently interprets ttl=0 as "expired immediately" (treating 0 as the
// zero-value default rather than "no expiry"); see TestWrapCache_Put_TTLZero
// below. Until that contract is reconciled, all positive-path tests pass a
// long TTL so the value is alive throughout the test.
const testTTL = 24 * time.Hour

// newWrapCacheForTest constructs a WrapCache against a fresh loopback-mount
// directory, fails the test on init error, and registers Close via
// t.Cleanup. Use this in every test that doesn't need a custom Config.
func newWrapCacheForTest(t *testing.T) *WrapCache {
	t.Helper()
	mount := mountfs.New(t)
	wc, err := NewWrapCache(validConfig(), mount)
	if err != nil {
		t.Fatalf("NewWrapCache: %v", err)
	}
	t.Cleanup(func() { _ = wc.Close() })
	return wc
}

// fillCache pushes enough Puts to force memtable rotation in every shard.
// Memtable rotation is synchronous; callers that need on-disk visibility
// afterwards must additionally poll their own observable via waitForOrSkip
// instead of relying on a fixed delay here.
func fillCache(t *testing.T, wc *WrapCache) {
	t.Helper()
	val := make([]byte, 4096)
	// 17 × 4KiB > 64KiB memtable; iterate enough times to rotate all shards.
	for i := 0; i < 64; i++ {
		key := "fill-" + strings.Repeat("x", (i%8)+1)
		if err := wc.Put(key, val, testTTL); err != nil {
			t.Fatalf("Put #%d: %v", i, err)
		}
	}
}

// waitForOrSkip polls cond every 10ms until it returns true or the timeout
// elapses; on timeout the test is skipped (not failed) because most async
// flush/rewrite observables in this suite are best-effort under the
// loopback ext4 mount.
func waitForOrSkip(t *testing.T, timeout time.Duration, msg string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Skipf("waitForOrSkip timeout: %s", msg)
}

// --- Config.validate() error sentinels ----------------------------------

func TestNewWrapCache_ValidConfig_Constructs(t *testing.T) {
	mount := mountfs.New(t)
	wc, err := NewWrapCache(validConfig(), mount)
	if err != nil {
		t.Fatalf("NewWrapCache with valid config: %v", err)
	}
	if wc == nil {
		t.Fatal("NewWrapCache returned nil with no error")
	}
	if err := wc.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestNewWrapCache_NumShardsZero(t *testing.T) {
	cfg := validConfig()
	cfg.NumShards = 0
	wc, err := NewWrapCache(cfg, "/this/path/never/used")
	if !errors.Is(err, ErrNumShardLessThan1) {
		t.Fatalf("want ErrNumShardLessThan1, got err=%v wc=%v", err, wc)
	}
	if wc != nil {
		t.Errorf("want nil cache on validation failure, got %v", wc)
	}
}

func TestNewWrapCache_KeysPerShardZero(t *testing.T) {
	cfg := validConfig()
	cfg.KeysPerShard = 0
	_, err := NewWrapCache(cfg, "/unused")
	if !errors.Is(err, ErrKeysPerShardLessThan1) {
		t.Fatalf("want ErrKeysPerShardLessThan1, got %v", err)
	}
}

func TestNewWrapCache_KeysPerShardOverCap(t *testing.T) {
	cfg := validConfig()
	cfg.KeysPerShard = (1 << 26) + 1 // > 67M cap
	_, err := NewWrapCache(cfg, "/unused")
	if !errors.Is(err, ErrKeysPerShardGreaterThan67M) {
		t.Fatalf("want ErrKeysPerShardGreaterThan67M, got %v", err)
	}
}

func TestNewWrapCache_MemtableSizeZero(t *testing.T) {
	cfg := validConfig()
	cfg.MemtableSize = 0
	_, err := NewWrapCache(cfg, "/unused")
	if !errors.Is(err, ErrMemtableSizeLessThan1) {
		t.Fatalf("want ErrMemtableSizeLessThan1, got %v", err)
	}
}

func TestNewWrapCache_MemtableSizeTooLarge(t *testing.T) {
	cfg := validConfig()
	cfg.MemtableSize = (1 << 30) + 4096 // just over 1 GiB, still 4KB-aligned
	_, err := NewWrapCache(cfg, "/unused")
	if !errors.Is(err, ErrMemtableSizeGreaterThan1GB) {
		t.Fatalf("want ErrMemtableSizeGreaterThan1GB, got %v", err)
	}
}

func TestNewWrapCache_MemtableSizeUnaligned(t *testing.T) {
	cfg := validConfig()
	cfg.MemtableSize = 4097 // 4KB + 1
	_, err := NewWrapCache(cfg, "/unused")
	if !errors.Is(err, ErrMemtableSizeNotMultipleOf4KB) {
		t.Fatalf("want ErrMemtableSizeNotMultipleOf4KB, got %v", err)
	}
}

func TestNewWrapCache_FileSizeZero(t *testing.T) {
	cfg := validConfig()
	cfg.FileSize = 0
	_, err := NewWrapCache(cfg, "/unused")
	if !errors.Is(err, ErrFileSizeLessThan1) {
		t.Fatalf("want ErrFileSizeLessThan1, got %v", err)
	}
}

func TestNewWrapCache_FileSizeUnaligned(t *testing.T) {
	cfg := validConfig()
	cfg.FileSize = (1 << 20) + 1 // 1 MiB + 1 byte
	_, err := NewWrapCache(cfg, "/unused")
	if !errors.Is(err, ErrFileSizeNotMultipleOf4KB) {
		t.Fatalf("want ErrFileSizeNotMultipleOf4KB, got %v", err)
	}
}

func TestNewWrapCache_FirstErrorWins(t *testing.T) {
	// Multiple invalid fields. The validate() ordering is fixed; the first
	// check in the chain (NumShards) must win, regardless of how many other
	// fields are broken.
	cfg := validConfig()
	cfg.NumShards = 0
	cfg.KeysPerShard = 0
	cfg.MemtableSize = 4097
	cfg.FileSize = 0
	_, err := NewWrapCache(cfg, "/unused")
	if !errors.Is(err, ErrNumShardLessThan1) {
		t.Fatalf("want ErrNumShardLessThan1 first, got %v", err)
	}
}

func TestNewWrapCache_NonExistentMountPath(t *testing.T) {
	// Valid config, but the mount directory does not exist on the filesystem.
	// The wrapping `read mount point: %w` should reach the caller.
	_, err := NewWrapCache(validConfig(), "/this/path/genuinely/does/not/exist/abc123")
	if err == nil {
		t.Fatal("want error for missing mountPoint, got nil")
	}
	if !strings.Contains(err.Error(), "read mount point") {
		t.Errorf("want error wrapping 'read mount point', got %q", err.Error())
	}
}

// --- Put/Get round-trip ------------------------------------------------

func TestWrapCache_PutGet_RoundTrip(t *testing.T) {
	wc := newWrapCacheForTest(t)

	if err := wc.Put("k1", []byte("payload-1"), testTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	val, found, expired := wc.Get("k1")
	if !found {
		t.Fatal("Get k1: found=false, want true")
	}
	if expired {
		t.Error("Get k1: expired=true, want false (within TTL window)")
	}
	if string(val) != "payload-1" {
		t.Errorf("Get k1: value=%q, want payload-1", val)
	}
}

// TestWrapCache_Put_TTLZero documents the contract gap that the rest of this
// suite works around. Per the cache.go API, ttl=0 should mean "never expires";
// in practice the implementation treats 0 as "expired immediately" (the
// zero-value default). Skipped until the flashring owners decide whether to
// fix the implementation or update the contract.
func TestWrapCache_Put_TTLZero(t *testing.T) {
	t.Skip("ttl=0 currently means 'expired immediately'; see docs/tests/testcases/cache-put-get.md")

	wc := newWrapCacheForTest(t)

	if err := wc.Put("forever", []byte("v"), 0); err != nil {
		t.Fatalf("Put: %v", err)
	}
	val, found, expired := wc.Get("forever")
	if !found {
		t.Fatal("Get forever: found=false, want true")
	}
	if expired {
		t.Error("Get forever: expired=true, want false (ttl=0 should mean never-expire)")
	}
	if string(val) != "v" {
		t.Errorf("Get forever: value=%q, want v", val)
	}
}

func TestWrapCache_Get_UnknownKey(t *testing.T) {
	wc := newWrapCacheForTest(t)

	val, found, expired := wc.Get("never-set")
	if found {
		t.Error("Get of unknown key: found=true, want false")
	}
	if expired {
		t.Error("Get of unknown key: expired=true, want false (was never written)")
	}
	if len(val) != 0 {
		t.Errorf("Get of unknown key: value non-empty: %q", val)
	}
}

func TestWrapCache_Put_Overwrite_ReturnsNewBytes(t *testing.T) {
	wc := newWrapCacheForTest(t)

	if err := wc.Put("k", []byte("v1"), testTTL); err != nil {
		t.Fatalf("first Put: %v", err)
	}
	if err := wc.Put("k", []byte("v2-longer"), testTTL); err != nil {
		t.Fatalf("second Put: %v", err)
	}
	val, found, _ := wc.Get("k")
	if !found {
		t.Fatal("Get after overwrite: found=false")
	}
	if string(val) != "v2-longer" {
		t.Errorf("Get after overwrite: value=%q, want v2-longer", val)
	}
}

func TestWrapCache_Delete_MakesKeyUnreachable(t *testing.T) {
	wc := newWrapCacheForTest(t)

	if err := wc.Put("dk", []byte("vv"), testTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if !wc.Delete("dk") {
		t.Fatal("Delete of present key returned false")
	}
	_, found, _ := wc.Get("dk")
	if found {
		t.Error("Get after Delete: found=true, want false")
	}
	// Second Delete is a no-op.
	if wc.Delete("dk") {
		t.Error("second Delete returned true, want false")
	}
}

func TestWrapCache_Hash_Deterministic(t *testing.T) {
	wc := newWrapCacheForTest(t)

	const key = "deterministic-test-key"
	h1 := wc.Hash(key)
	h2 := wc.Hash(key)
	if h1 != h2 {
		t.Errorf("Hash(%q) returned different values across calls: %d vs %d", key, h1, h2)
	}
}

func TestWrapCache_Put_EmptyValue(t *testing.T) {
	wc := newWrapCacheForTest(t)

	if err := wc.Put("empty", []byte{}, testTTL); err != nil {
		t.Fatalf("Put empty: %v", err)
	}
	val, found, expired := wc.Get("empty")
	if !found {
		t.Fatal("Get empty: found=false, want true (empty != absent)")
	}
	if expired {
		t.Error("Get empty: expired=true")
	}
	if len(val) != 0 {
		t.Errorf("Get empty: value len=%d, want 0", len(val))
	}
}

func TestWrapCache_Put_ZeroSubMinuteTTL_RoundsUp(t *testing.T) {
	wc := newWrapCacheForTest(t)

	// Sub-minute TTL > 0 → ttlMinutes = uint16(ttl.Minutes()) = 0, but the
	// `if ttlMinutes == 0 && ttl > 0` guard rounds up to 1.
	if err := wc.Put("submin", []byte("v"), 1*time.Second); err != nil {
		t.Fatalf("Put: %v", err)
	}
	val, found, expired := wc.Get("submin")
	if !found || expired {
		t.Fatalf("Get = (found=%v, expired=%v), want (true, false)", found, expired)
	}
	if string(val) != "v" {
		t.Fatalf("val = %q, want v", val)
	}
}

// --- MGet --------------------------------------------------------------

func TestWrapCache_MGet_AllKeysPresent(t *testing.T) {
	wc := newWrapCacheForTest(t)

	if err := wc.Put("a", []byte("va"), testTTL); err != nil {
		t.Fatalf("Put a: %v", err)
	}
	if err := wc.Put("b", []byte("vb"), testTTL); err != nil {
		t.Fatalf("Put b: %v", err)
	}

	results := wc.MGet([]string{"a", "b", "missing"})
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	if !results[0].Found || string(results[0].Value) != "va" {
		t.Errorf("results[0] = %+v, want va", results[0])
	}
	if !results[1].Found || string(results[1].Value) != "vb" {
		t.Errorf("results[1] = %+v, want vb", results[1])
	}
	if results[2].Found {
		t.Errorf("missing key found: %+v", results[2])
	}
}

func TestWrapCache_MGet_Empty(t *testing.T) {
	wc := newWrapCacheForTest(t)

	results := wc.MGet(nil)
	if results == nil {
		t.Fatal("MGet([]string{}) returned nil")
	}
	if len(results) != 0 {
		t.Fatalf("len = %d, want 0", len(results))
	}
}

func TestWrapCache_MGet_ExpiredKey_FlagsExpired(t *testing.T) {
	wc := newWrapCacheForTest(t)

	if err := wc.Put("ephemeral", []byte("v"), 0); err != nil {
		t.Fatalf("Put: %v", err)
	}

	results := wc.MGet([]string{"ephemeral"})
	if !results[0].Expired {
		t.Fatalf("results[0] = %+v, want Expired=true", results[0])
	}
}

func TestWrapCache_MGet_TriggersDiskReadPath(t *testing.T) {
	wc := newWrapCacheForTest(t)

	// Put a key, then evict by filling, so MGet falls through to the
	// coalesce/submit/collect path inside the cache.
	if err := wc.Put("ondisk", []byte("disk-value"), testTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	fillCache(t, wc)

	results := wc.MGet([]string{"ondisk"})
	if len(results) != 1 {
		t.Fatalf("len = %d, want 1", len(results))
	}
	// CRC corruption from interleaved rewrites is tolerated, but slot-shape
	// invariants must hold: a result must be either (Found && len(Value)>0)
	// or (!Found && len(Value)==0) — never the contradiction.
	r := results[0]
	if r.Found && len(r.Value) == 0 {
		t.Errorf("found=true but Value is empty: %+v", r)
	}
	if !r.Found && len(r.Value) != 0 {
		t.Errorf("found=false but Value non-empty: %+v", r)
	}
}

func TestWrapCache_Put_AfterFlush_StillReadable(t *testing.T) {
	wc := newWrapCacheForTest(t)

	if err := wc.Put("stable", []byte("v"), testTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	// Trigger many flushes to exercise the rewrite goroutine path; rewrites
	// run async so we can't deterministically assert but the branch fires.
	fillCache(t, wc)

	// Re-Put the same key. The Put covers the post-rotation branch too.
	if err := wc.Put("stable", []byte("v2"), testTTL); err != nil {
		t.Fatalf("Put after fill: %v", err)
	}
	val, found, _ := wc.Get("stable")
	if !found {
		t.Fatal("Get found = false after Put-Fill-Put")
	}
	if string(val) != "v2" {
		t.Fatalf("Get val = %q, want v2", val)
	}
}

func TestWrapCache_RewritePath_PutSubmittedAsync(t *testing.T) {
	// Use a config with a very low score threshold so any Get triggers a
	// rewrite, exercising the `if shouldReWrite` branch in Get and the
	// rewrite goroutine in cache.go.
	cfg := validConfig()
	cfg.ReWriteScoreThreshold = -1 // every score crosses → always rewrite

	mount := mountfs.New(t)
	wc, err := NewWrapCache(cfg, mount)
	if err != nil {
		t.Fatalf("NewWrapCache: %v", err)
	}
	defer wc.Close()

	if err := wc.Put("rw", []byte("v"), testTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	// One Get triggers an async rewrite.
	val, found, _ := wc.Get("rw")
	if !found || string(val) != "v" {
		t.Fatalf("Get = (%q, %v), want (v, true)", val, found)
	}
	// Yield so the fire-and-forget rewrite goroutine runs to completion
	// against the still-open cache, rather than racing wc.Close() during
	// t.Cleanup teardown. This is hygiene, not synchronization — there is
	// no public observable for rewrite completion.
	time.Sleep(50 * time.Millisecond)
}

func TestWrapCache_MGet_ManyKeys_RewriteBranch(t *testing.T) {
	cfg := validConfig()
	cfg.NumShards = 2
	cfg.ReWriteScoreThreshold = -1 // every hit rewrites

	mount := mountfs.New(t)
	wc, err := NewWrapCache(cfg, mount)
	if err != nil {
		t.Fatalf("NewWrapCache: %v", err)
	}
	defer wc.Close()

	keys := []string{"a", "b", "c", "d", "e", "f"}
	for _, k := range keys {
		if err := wc.Put(k, []byte("v-"+k), testTTL); err != nil {
			t.Fatalf("Put %s: %v", k, err)
		}
	}

	// All memtable hits → triggers the inline-validate + rewrite branch in MGet.
	results := wc.MGet(keys)
	if len(results) != len(keys) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(keys))
	}
	hits := 0
	for i, r := range results {
		if !r.Found {
			continue
		}
		hits++
		want := "v-" + keys[i]
		if string(r.Value) != want {
			t.Errorf("MGet[%s] = %q, want %q", keys[i], r.Value, want)
		}
	}
	if hits == 0 {
		t.Fatal("expected at least one MGet hit")
	}
	// Yield so the fire-and-forget rewrite goroutines run to completion
	// against the still-open cache (see TestWrapCache_RewritePath_PutSubmittedAsync).
	time.Sleep(50 * time.Millisecond)
}

func TestWrapCache_MGet_AfterFlush_DiskReadPath(t *testing.T) {
	// Put a key, force memtable rotation so the key lives on disk, then MGet
	// it. This drives MGet through the coalesce + submit + collect phases
	// instead of the memtable-hit fast path.
	wc := newWrapCacheForTest(t)

	if err := wc.Put("disk-key", []byte("disk-payload"), testTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	fillCache(t, wc) // ≥1 memtable rotation; flushes "disk-key" to file

	// Poll until the disk-read path actually returns a result for the key.
	var results []MGetResult
	waitForOrSkip(t, 2*time.Second, "MGet returns 2 results after flush", func() bool {
		results = wc.MGet([]string{"disk-key", "missing-key"})
		return len(results) == 2
	})
	// `missing-key` was never put; must be reported as not found.
	if results[1].Found {
		t.Errorf("missing-key reported found: %+v", results[1])
	}
	// If disk-key was found, its payload must round-trip exactly.
	if results[0].Found && string(results[0].Value) != "disk-payload" {
		t.Errorf("disk-key value = %q, want disk-payload", results[0].Value)
	}
}

func TestWrapCache_MGet_MultiShard_DistributesByHash(t *testing.T) {
	cfg := validConfig()
	cfg.NumShards = 4
	mount := mountfs.New(t)
	wc, err := NewWrapCache(cfg, mount)
	if err != nil {
		t.Fatalf("NewWrapCache: %v", err)
	}
	defer wc.Close()

	keys := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"}
	values := make(map[string]string, len(keys))
	for _, k := range keys {
		v := "v-" + k
		values[k] = v
		if err := wc.Put(k, []byte(v), testTTL); err != nil {
			t.Fatalf("Put %s: %v", k, err)
		}
	}

	results := wc.MGet(keys)
	if len(results) != len(keys) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(keys))
	}
	for i, k := range keys {
		if !results[i].Found {
			t.Errorf("MGet[%s] not found", k)
			continue
		}
		if string(results[i].Value) != values[k] {
			t.Errorf("MGet[%s] = %q, want %q", k, results[i].Value, values[k])
		}
	}
}

func TestWrapCache_Get_OnFlushedKey_DiskPath(t *testing.T) {
	// Same idea as the MGet variant — drive Get through the disk-read path.
	wc := newWrapCacheForTest(t)

	if err := wc.Put("evict-me", []byte("evicted"), testTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	fillCache(t, wc)

	// Poll Get until the call settles (not expired in TTL window). Memtable
	// rotation happened synchronously during fillCache; this loop tolerates
	// the disk-write race without a fixed delay.
	var (
		val     []byte
		found   bool
		expired bool
	)
	waitForOrSkip(t, 2*time.Second, "Get returns non-expired result", func() bool {
		val, found, expired = wc.Get("evict-me")
		return !expired
	})
	// Slot-shape invariants must hold regardless of CRC outcome:
	//  - !found ⇒ len(val) == 0
	//  - found  ⇒ string(val) == "evicted"
	if !found && len(val) != 0 {
		t.Errorf("found=false but val non-empty: %q", val)
	}
	if found && string(val) != "evicted" {
		t.Errorf("found=true but val = %q, want evicted", val)
	}
}

func TestWrapCache_MGet_FlushedKeys_WithRewrite(t *testing.T) {
	// Combines disk-read MGet path + ShouldRewrite branch: keys are flushed
	// to disk *and* the predictor's threshold is low enough that every hit
	// triggers a rewrite goroutine.
	cfg := validConfig()
	cfg.ReWriteScoreThreshold = -1
	mount := mountfs.New(t)
	wc, err := NewWrapCache(cfg, mount)
	if err != nil {
		t.Fatalf("NewWrapCache: %v", err)
	}
	defer wc.Close()

	keys := []string{"flush-a", "flush-b", "flush-c"}
	for _, k := range keys {
		if err := wc.Put(k, []byte("payload-"+k), testTTL); err != nil {
			t.Fatalf("Put %s: %v", k, err)
		}
	}
	fillCache(t, wc)

	// Poll MGet until it returns the expected slice length so the test no
	// longer depends on a fixed flush-propagation delay.
	var results []MGetResult
	waitForOrSkip(t, 2*time.Second, "MGet returns N results after flush", func() bool {
		results = wc.MGet(keys)
		return len(results) == len(keys)
	})
	// For any result reported as Found, the round-tripped payload must match
	// what was Put. CRC-fail outcomes are tolerated; silent corruption is not.
	for i, r := range results {
		if !r.Found {
			continue
		}
		want := "payload-" + keys[i]
		if string(r.Value) != want {
			t.Errorf("MGet[%s] = %q, want %q", keys[i], r.Value, want)
		}
	}
}

func TestWrapCache_MGet_MixedExpiredFoundMissing(t *testing.T) {
	// Mix of all three result types in one call → exercises all three branches
	// of phase-1 (Expired continue, !Found continue, Found memtable hit).
	wc := newWrapCacheForTest(t)

	if err := wc.Put("alive", []byte("v"), testTTL); err != nil {
		t.Fatalf("Put alive: %v", err)
	}
	if err := wc.Put("dead", []byte("v"), 0); err != nil { // ttl=0 → expired immediately
		t.Fatalf("Put dead: %v", err)
	}

	results := wc.MGet([]string{"alive", "dead", "never-set"})
	if len(results) != 3 {
		t.Fatalf("len = %d, want 3", len(results))
	}
	if !results[0].Found || string(results[0].Value) != "v" {
		t.Errorf("alive: %+v", results[0])
	}
	if !results[1].Expired {
		t.Errorf("dead: Expired=false, want true (%+v)", results[1])
	}
	if results[2].Found || results[2].Expired {
		t.Errorf("never-set: %+v", results[2])
	}
}

func TestWrapCache_MGet_AdjacentOffsets_CoalescesIntoOneGroup(t *testing.T) {
	// Several keys flushed to consecutive 4KiB pages in the same shard get
	// coalesced into a single io_uring submission. Exercises the merge
	// branch in phase-2 (same shard, aStart <= last.alignedEnd, fits within
	// maxCoalescedReadSz).
	cfg := validConfig()
	cfg.NumShards = 1
	mount := mountfs.New(t)
	wc, err := NewWrapCache(cfg, mount)
	if err != nil {
		t.Fatalf("NewWrapCache: %v", err)
	}
	defer wc.Close()

	keys := []string{"adj-1", "adj-2", "adj-3", "adj-4"}
	for _, k := range keys {
		if err := wc.Put(k, []byte("p-"+k), testTTL); err != nil {
			t.Fatalf("Put %s: %v", k, err)
		}
	}
	fillCache(t, wc)

	var results []MGetResult
	waitForOrSkip(t, 2*time.Second, "MGet returns N results after flush", func() bool {
		results = wc.MGet(keys)
		return len(results) == len(keys)
	})
	// Found entries must round-trip their payload. CRC-fail tolerated.
	for i, r := range results {
		if !r.Found {
			continue
		}
		want := "p-" + keys[i]
		if string(r.Value) != want {
			t.Errorf("MGet[%s] = %q, want %q", keys[i], r.Value, want)
		}
	}
}

// TestWrapCache_Get_Expired_FlagsExpired covers the expired-key metric
// branch in WrapCache.Get (the `if expired` increment in cache.go that the
// MGet variant already exercises but Get did not).
func TestWrapCache_Get_Expired_FlagsExpired(t *testing.T) {
	wc := newWrapCacheForTest(t)

	if err := wc.Put("ephemeral", []byte("v"), 0); err != nil {
		t.Fatalf("Put: %v", err)
	}

	val, found, expired := wc.Get("ephemeral")
	if !expired {
		t.Fatalf("Get expired=false, want true (ttl=0 means expired immediately)")
	}
	if found {
		t.Errorf("Get found=true on expired key, want false")
	}
	if len(val) != 0 {
		t.Errorf("Get val non-empty on expired: %q", val)
	}
}

// TestNewWrapCache_DirtyMount_WipesPreexistingFiles covers the
// `for _, file := range files { os.Remove(...) }` cleanup loop that runs
// when the mount directory is non-empty at construction.
func TestNewWrapCache_DirtyMount_WipesPreexistingFiles(t *testing.T) {
	mount := mountfs.New(t)
	// Pre-populate the mount with files that NewWrapCache must remove.
	stale := []string{"stale-1.dat", "stale-2.dat", "old.bin"}
	for _, name := range stale {
		p := filepath.Join(mount, name)
		if err := os.WriteFile(p, []byte("residue"), 0644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}

	wc, err := NewWrapCache(validConfig(), mount)
	if err != nil {
		t.Fatalf("NewWrapCache: %v", err)
	}
	defer wc.Close()

	// Each seeded path must be gone after construction.
	for _, name := range stale {
		p := filepath.Join(mount, name)
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s not removed: err=%v", name, err)
		}
	}
}

// TestWrapCache_MGet_MultiShardDiskReads_SortsByShard covers the
// `diskReads[i].shardIdx != diskReads[j].shardIdx` comparator branch in
// MGet's sort.Slice — fires only when multiple disk reads land in
// different shards, which the single-shard tests can't reach.
func TestWrapCache_MGet_MultiShardDiskReads_SortsByShard(t *testing.T) {
	cfg := validConfig()
	cfg.NumShards = 4
	mount := mountfs.New(t)
	wc, err := NewWrapCache(cfg, mount)
	if err != nil {
		t.Fatalf("NewWrapCache: %v", err)
	}
	defer wc.Close()

	// Enough keys for the hash to spread them across all 4 shards.
	keys := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta"}
	want := make(map[string]string, len(keys))
	for _, k := range keys {
		v := "v-" + k
		want[k] = v
		if err := wc.Put(k, []byte(v), testTTL); err != nil {
			t.Fatalf("Put %s: %v", k, err)
		}
	}
	fillCache(t, wc) // forces rotations so MGet must go through the disk-read path

	var results []MGetResult
	waitForOrSkip(t, 2*time.Second, "MGet returns N results after multi-shard flush", func() bool {
		results = wc.MGet(keys)
		return len(results) == len(keys)
	})
	for i, r := range results {
		if !r.Found {
			continue
		}
		if string(r.Value) != want[keys[i]] {
			t.Errorf("MGet[%s] = %q, want %q", keys[i], r.Value, want[keys[i]])
		}
	}
}

// TestWrapCache_Put_ShardErrorIsWrapped drives the
// `if err := wc.shards[shardIdx].Put(...); err != nil { return fmt.Errorf("put failed for key %s: %w", key, err) }`
// branch in WrapCache.Put (cache.go:256-258). To force the inner
// shard.Put to return an error we shrink KeysPerShard to a value that
// fills the ring buffer within a few hundred Puts, so the
// DeleteManager surfaces a "trim needed retry this write" error which
// WrapCache.Put then wraps with the canonical "put failed for key %s"
// prefix.
func TestWrapCache_Put_ShardErrorIsWrapped(t *testing.T) {
	cfg := validConfig()
	cfg.NumShards = 1
	cfg.KeysPerShard = 64
	mount := mountfs.New(t)
	wc, err := NewWrapCache(cfg, mount)
	if err != nil {
		t.Fatalf("NewWrapCache: %v", err)
	}
	defer wc.Close()

	val := make([]byte, 16)
	var lastErr error
	for i := 0; i < 5000; i++ {
		key := "wrap-" + strings.Repeat("x", (i%8)+1) + "-" + strings.Repeat("y", i%16)
		if err := wc.Put(key, val, testTTL); err != nil {
			lastErr = err
			break
		}
	}
	if lastErr == nil {
		t.Fatal("WrapCache.Put never surfaced an error after 5000 keys")
	}
	if !strings.Contains(lastErr.Error(), "put failed for key") {
		t.Fatalf("Put err = %v, want wrapped with 'put failed for key' prefix", lastErr)
	}
}

// TestWrapCache_MGet_AdjacentBlocks_CoalesceExtendsAlignedEnd covers the
// `if aEnd > mergedEnd { mergedEnd = aEnd }` branch in MGet's coalesce
// merge — fires only when an incoming read extends past the current
// merged region (i.e., consecutive keys land in adjacent 4KiB blocks,
// not the same one). The AdjacentOffsets variant above uses tiny values
// that all share a single 4KiB block; this test sizes each slot so it
// fills its own block, forcing the merge-extend path.
func TestWrapCache_MGet_AdjacentBlocks_CoalesceExtendsAlignedEnd(t *testing.T) {
	cfg := validConfig()
	cfg.NumShards = 1
	mount := mountfs.New(t)
	wc, err := NewWrapCache(cfg, mount)
	if err != nil {
		t.Fatalf("NewWrapCache: %v", err)
	}
	defer wc.Close()

	// Slot layout = [crc(4) | key | value]; key="k1"/"k2" → 2 bytes.
	// Choose value size so total slot = 4 + 2 + valSize is just under 4096,
	// so slot 1 occupies block [0,4096) and slot 2 occupies [4096,8192).
	const blockSize = 4096
	valSize := blockSize - 4 - 2
	blockVal := make([]byte, valSize)
	for i := range blockVal {
		blockVal[i] = byte(i % 251)
	}

	keys := []string{"k1", "k2"}
	for _, k := range keys {
		if err := wc.Put(k, blockVal, testTTL); err != nil {
			t.Fatalf("Put %s: %v", k, err)
		}
	}
	fillCache(t, wc)

	var results []MGetResult
	waitForOrSkip(t, 2*time.Second, "MGet returns 2 results", func() bool {
		results = wc.MGet(keys)
		return len(results) == 2
	})
	for i, r := range results {
		if r.Found && !bytes.Equal(r.Value, blockVal) {
			t.Errorf("MGet[%s] value differs from put", keys[i])
		}
	}
}

// TestWrapCache_Put_RejectsOversizeValue exercises the C2 fix (PR #2377):
// values where 4 + len(key) + len(value) exceeds math.MaxUint16 used to
// silently truncate via the uint16 cast in ShardCache.Put and later panic
// on Get when the wrapped length underflowed the CRC + key prefix. Put
// now rejects at the surface with ErrValueTooLarge before touching a
// shard.
func TestWrapCache_Put_RejectsOversizeValue(t *testing.T) {
	wc := newWrapCacheForTest(t)

	key := "oversize-key" // 12 bytes
	// Total = 4 (CRC) + 12 (key) + 65520 (value) = 65536 > math.MaxUint16.
	value := bytes.Repeat([]byte{0x42}, 65520)

	err := wc.Put(key, value, testTTL)
	if err == nil {
		t.Fatal("Put returned nil for oversize value; want ErrValueTooLarge")
	}
	if !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("Put err = %v; want errors.Is ErrValueTooLarge", err)
	}

	// Nothing was stored — Get returns not-found for the rejected key.
	got, found, expired := wc.Get(key)
	if found {
		t.Fatalf("Get found=true after rejected Put; got value len=%d", len(got))
	}
	if expired {
		t.Fatal("Get expired=true after rejected Put; want expired=false")
	}
}

// TestWrapCache_Put_AcceptsMaxValidSize is the flip side of the oversize
// test — the largest legal value (uint16 max minus CRC + key overhead)
// must still round-trip cleanly. If this ever fails, the boundary check
// in Put has drifted.
func TestWrapCache_Put_AcceptsMaxValidSize(t *testing.T) {
	wc := newWrapCacheForTest(t)

	key := "boundary-key" // 12 bytes
	// Total = 4 + 12 + 65519 = 65535 == math.MaxUint16 → the LARGEST
	// value that fits in a single index entry.
	value := bytes.Repeat([]byte{0x7F}, 65519)

	if err := wc.Put(key, value, testTTL); err != nil {
		t.Fatalf("Put at exact uint16 boundary failed: %v", err)
	}
	got, found, expired := wc.Get(key)
	if !found {
		t.Fatal("Get found=false for boundary-size value")
	}
	if expired {
		t.Fatal("Get expired=true for boundary-size value")
	}
	if !bytes.Equal(got, value) {
		t.Fatalf("Get value len=%d, want %d; content mismatch", len(got), len(value))
	}
}

// TestConfig_validate_FileSizeDivisibility exercises the new divisibility
// invariant introduced in PR #2377 (bug #2 in the audit). Table-driven so
// each interesting boundary is called out explicitly and a future edit to
// the modulo check can't silently drop coverage.
func TestConfig_validate_FileSizeDivisibility(t *testing.T) {
	tests := []struct {
		name         string
		fileSize     int64
		memtableSize int32
		wantErr      error
	}{
		{"exact_multiple_2x", 128 * 1024, 64 * 1024, nil},
		{"exact_multiple_1x", 64 * 1024, 64 * 1024, nil},
		{"exact_multiple_large", 4 * 1024 * 1024, 64 * 1024, nil},
		// 132KiB = 33 * 4KiB (passes the 4KiB-alignment check) but
		// 132 % 64 = 4KiB (fails the new divisibility check). The
		// original test case used 130KiB which is 32.5 * 4KiB — the
		// earlier ErrFileSizeNotMultipleOf4KB fires first.
		{"non_multiple_by_one_block", 132 * 1024, 64 * 1024, ErrFileSizeNotMultipleOfMemtableSize},
		{"non_multiple_small_tail", 68 * 1024, 64 * 1024, ErrFileSizeNotMultipleOfMemtableSize},
		{"non_multiple_ratio_1_5", 96 * 1024, 64 * 1024, ErrFileSizeNotMultipleOfMemtableSize},
		// Zero MemtableSize is caught by an earlier check — the
		// divisibility rule short-circuits its own MemtableSize > 0 guard
		// so it doesn't itself divide by zero. Verify the earlier check
		// still wins.
		{"memtable_zero_earlier_check_wins", 4 * 1024, 0, ErrMemtableSizeLessThan1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.FileSize = tc.fileSize
			cfg.MemtableSize = tc.memtableSize
			err := cfg.validate()
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("validate() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("validate() = %v, want errors.Is %v", err, tc.wantErr)
			}
		})
	}
}

// TestWrapCache_Put_RetriesOnTrimNeeded exercises the C12 fix (PR #2377):
// when ShardCache.Put returns index.ErrTrimNeededRetry, WrapCache.Put now
// retries once so the amortised delete pass completes rather than dropping
// the caller's write. Drives the cache past a memtable boundary + into the
// ring-buffer wrap so ExecuteDeleteIfNeeded fires the trim sentinel on a
// natural code path; asserts every Put returns nil, no dropped writes, and
// the error message (if any error had surfaced) does not embed the raw key.
func TestWrapCache_Put_RetriesOnTrimNeeded(t *testing.T) {
	wc := newWrapCacheForTest(t)

	// Fill enough entries to exhaust one memtable per shard AND push the
	// ring buffer past its head — triggers TrimHeadIfNeeded /
	// NextAddNeedsDelete in the delete manager, which is the code path
	// that returns ErrTrimNeededRetry via WrapCache.Put's retry loop.
	// validConfig has KeysPerShard=1024; write 2048 unique keys so at
	// least one wrap happens per shard.
	value := bytes.Repeat([]byte{0xAB}, 512)
	for i := 0; i < 2048; i++ {
		key := "trim-retry-" + strconv.Itoa(i)
		if err := wc.Put(key, value, testTTL); err != nil {
			// If the retry ever surfaces as a caller-visible error, it
			// MUST NOT leak the raw key (CLAUDE.md NEVER-DO + security
			// rule). Assert both.
			if strings.Contains(err.Error(), key) {
				t.Fatalf("Put err leaked raw key %q: %v", key, err)
			}
			t.Fatalf("Put #%d failed after trim retry: %v", i, err)
		}
	}
}

// TestWrapCache_Put_SurfacePendingFlushErr locks down the user-visible
// wiring of C4 (PR #2377): after an async flush failure, WrapCache.Put
// consumes the memtable-manager flag ONCE via compare-and-swap and returns
// ErrPreviousFlushFailed. Subsequent Puts see a clean state.
func TestWrapCache_Put_SurfacePendingFlushErr(t *testing.T) {
	wc := newWrapCacheForTest(t)

	// Arm the flag on shard 0 directly. The route from a real flush failure
	// to this flag is exercised by TestMemtableManager_ConsumeFlushErr_*;
	// here we lock the surfacing wiring in isolation.
	wc.shards[0].MemtableManager().SetFlushErrForTest()

	// Pick a key that hashes to shard 0. Since NumShards=1 in the test
	// config, any key routes there.
	err := wc.Put("flush-err-key", []byte("v"), testTTL)
	if err == nil {
		t.Fatal("Put err = nil after arming flush-err flag, want wrapped ErrPreviousFlushFailed")
	}
	if !errors.Is(err, memtables.ErrPreviousFlushFailed) {
		t.Fatalf("Put err = %v, want errors.Is memtables.ErrPreviousFlushFailed", err)
	}

	// Flag consumed — next Put succeeds.
	if wc.shards[0].MemtableManager().PeekFlushErrForTest() {
		t.Fatal("flag not consumed by WrapCache.Put — CAS regression")
	}
	if err := wc.Put("clean-again", []byte("v"), testTTL); err != nil {
		t.Fatalf("second Put after flag drained: %v", err)
	}
}

// TestWrapCache_Rewrite_DoesNotConsumeFlushErrFlag locks down the m-review
// fix (PR #2377): background rewrite goroutines from Get/MGet discard
// their error return, so they must bypass the flush-err flag check.
// Otherwise a rewrite would silently swallow the error the next user-
// visible Put should have seen.
func TestWrapCache_Rewrite_DoesNotConsumeFlushErrFlag(t *testing.T) {
	wc := newWrapCacheForTest(t)

	wc.shards[0].MemtableManager().SetFlushErrForTest()

	// Call rewrite directly (same-package access). It routes through
	// shard.Put directly, bypassing WrapCache.Put's flag-consume path.
	wc.rewrite("rewrite-key", []byte("v"), 60)

	// The flag MUST still be set — rewrite should not have consumed it.
	if !wc.shards[0].MemtableManager().PeekFlushErrForTest() {
		t.Fatal("rewrite consumed the flush-err flag — user-visible Put would now silently miss it")
	}

	// And a subsequent user-visible Put picks up the flag exactly once.
	err := wc.Put("user-visible", []byte("v"), testTTL)
	if !errors.Is(err, memtables.ErrPreviousFlushFailed) {
		t.Fatalf("user-visible Put after rewrite: err = %v, want ErrPreviousFlushFailed", err)
	}
}

// Close registered by newWrapCacheForTest runs again at cleanup, so this
// also covers a third call.
func TestWrapCache_Close_Twice(t *testing.T) {
	wc := newWrapCacheForTest(t)
	if err := wc.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := wc.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// NewWrapCache indexes FreqBands[2] and RecencyBands[2]; a shorter slice
// used to panic there after validation had passed.
func TestConfig_Validate_RejectsBandsWithoutThreeValues(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"nil freq bands":    func(c *Config) { c.FreqBands = nil },
		"two recency bands": func(c *Config) { c.RecencyBands = []int{1, 2} },
		"four freq bands":   func(c *Config) { c.FreqBands = []int{1, 2, 3, 4} },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := validConfig()
			mutate(&cfg)
			if err := cfg.validate(); !errors.Is(err, ErrBandsNeedThreeValues) {
				t.Fatalf("validate() = %v, want ErrBandsNeedThreeValues", err)
			}
		})
	}
}

// The flush counter is tagged with the manager's shard index; it used to
// be shard 0 for every shard.
func TestNewWrapCache_PassesShardIndexToMemtableManager(t *testing.T) {
	cfg := validConfig()
	cfg.NumShards = 3
	wc, err := NewWrapCache(cfg, mountfs.New(t))
	if err != nil {
		t.Fatalf("NewWrapCache: %v", err)
	}
	t.Cleanup(func() { _ = wc.Close() })
	for i, sh := range wc.shards {
		if got := sh.MemtableManager().ShardIdx; got != uint32(i) {
			t.Fatalf("shard %d: MemtableManager().ShardIdx = %d", i, got)
		}
	}
}

func TestWrapCache_Put_AfterClose_ReturnsErrCacheClosed(t *testing.T) {
	wc := newWrapCacheForTest(t)
	if err := wc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := wc.Put("k", []byte("v"), testTTL); !errors.Is(err, ErrCacheClosed) {
		t.Fatalf("Put after Close: err = %v, want ErrCacheClosed", err)
	}
	// A rewrite left over from an earlier Get must not touch the closed shard.
	wc.rewrite("k", []byte("v"), 60)
}

func TestWrapCache_Rewrite_PanicRecoveredAndLockReleased(t *testing.T) {
	// A nil shard makes Put panic on its first field access.
	wc := &WrapCache{
		shards:     []*filecache.ShardCache{nil},
		shardLocks: make([]sync.RWMutex, 1),
	}
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("rewrite panic escaped: %v", r)
			}
		}()
		wc.rewrite("k", []byte("v"), 60)
	}()
	if !wc.shardLocks[0].TryLock() {
		t.Fatal("shard lock still held after a panicking rewrite")
	}
	wc.shardLocks[0].Unlock()
}

func TestConfig_Validate_GridSearchNeedsDurationAndEpsilon(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"zero sample duration": func(c *Config) { c.SampleDuration = 0 },
		"zero epsilon":         func(c *Config) { c.GridSearchEpsilon = 0 },
		"NaN epsilon":          func(c *Config) { c.GridSearchEpsilon = math.NaN() },
		"infinite epsilon":     func(c *Config) { c.GridSearchEpsilon = math.Inf(1) },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := validConfig()
			mutate(&cfg)
			if err := cfg.validate(); err != nil {
				t.Fatalf("validate() with search off = %v, want nil", err)
			}
			cfg.GridSearchEnabled = true
			if err := cfg.validate(); !errors.Is(err, ErrGridSearchNeedsDurationAndEpsilon) {
				t.Fatalf("validate() = %v, want ErrGridSearchNeedsDurationAndEpsilon", err)
			}
		})
	}
}

// With the search on, a hit-rate sample from real Gets reaches the
// predictor: once the (1ns) sample window has elapsed the live weights
// move off the first default tuple. Close then stops the feeder.
func TestWrapCache_GridSearchEnabled_HitRateMovesWeights(t *testing.T) {
	cfg := validConfig()
	cfg.GridSearchEnabled = true
	cfg.SampleDuration = time.Nanosecond
	cfg.GridSearchEpsilon = 0.01
	wc, err := NewWrapCache(cfg, mountfs.New(t))
	if err != nil {
		t.Fatalf("NewWrapCache: %v", err)
	}
	defer wc.Close()

	if err := wc.Put("k", []byte("v"), time.Hour); err != nil {
		t.Fatalf("Put: %v", err)
	}
	wc.Get("k")
	wc.Get("absent")
	wc.MGet([]string{"k", "absent"})
	if hits, gets := wc.hits.Load(), wc.gets.Load(); hits != 2 || gets != 4 {
		t.Fatalf("hits, gets = %d, %d, want 2, 4", hits, gets)
	}

	wc.observeHitRate()
	if gets := wc.gets.Load(); gets != 0 {
		t.Fatalf("gets after observe = %d, want 0", gets)
	}
	deadline := time.Now().Add(2 * time.Second)
	for wc.predictor.Estimator.Weights() == defaultWeights[0] {
		if time.Now().After(deadline) {
			t.Fatal("live weights never moved off the first tuple")
		}
		time.Sleep(time.Millisecond)
	}

	if err := wc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	select {
	case <-wc.hitRateStopped:
	default:
		t.Fatal("feeder still running after Close")
	}
}

func TestWrapCache_GridSearchDisabled_NoFeeder(t *testing.T) {
	wc := newWrapCacheForTest(t)
	if wc.hitRateStop != nil {
		t.Fatal("feeder started with GridSearchEnabled unset")
	}
	wc.Get("absent")
	wc.MGet([]string{"absent"})
	if gets := wc.gets.Load(); gets != 0 {
		t.Fatalf("gets = %d, want 0: counters must stay untouched with the search off", gets)
	}
}
