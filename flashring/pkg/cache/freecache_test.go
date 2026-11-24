//go:build linux
// +build linux

package cache

import (
	"testing"
	"time"
)

// newFreecacheForTest builds a Freecache, fails the test on init error,
// and registers Close via t.Cleanup so callers don't leak the underlying
// in-process LRU between tests.
func newFreecacheForTest(t *testing.T) *Freecache {
	t.Helper()
	c, err := NewFreecache(1 << 20)
	if err != nil {
		t.Fatalf("NewFreecache: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestNewFreecache_HappyPath(t *testing.T) {
	c, err := NewFreecache(1 << 20)
	if err != nil {
		t.Fatalf("NewFreecache: %v", err)
	}
	if c == nil {
		t.Fatal("NewFreecache returned nil cache")
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestFreecache_PutGet_RoundTrip(t *testing.T) {
	c := newFreecacheForTest(t)

	if err := c.Put("k", []byte("v"), time.Hour); err != nil {
		t.Fatalf("Put: %v", err)
	}
	val, found, expired := c.Get("k")
	if !found {
		t.Fatal("found = false after Put")
	}
	if expired {
		t.Fatal("expired = true (Freecache always reports false)")
	}
	if string(val) != "v" {
		t.Fatalf("val = %q, want v", val)
	}
}

func TestFreecache_Get_Missing_ReturnsFalse(t *testing.T) {
	c := newFreecacheForTest(t)

	val, found, _ := c.Get("absent")
	if found {
		t.Fatal("found = true for missing key")
	}
	if val != nil {
		t.Fatalf("val = %v, want nil", val)
	}
}

func TestFreecache_Close_NoError(t *testing.T) {
	c, err := NewFreecache(1 << 20)
	if err != nil {
		t.Fatalf("NewFreecache: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
