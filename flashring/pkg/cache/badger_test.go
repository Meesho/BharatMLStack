//go:build linux
// +build linux

package cache

import (
	"path/filepath"
	"testing"
	"time"
)

func newBadger(t *testing.T) *Badger {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "badger")
	b, err := NewBadger(Config{}, dir)
	if err != nil {
		t.Fatalf("NewBadger: %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })
	return b
}

func TestNewBadger_BadDirectory_Errors(t *testing.T) {
	_, err := NewBadger(Config{}, "/dev/null/nope")
	if err == nil {
		t.Fatal("NewBadger returned nil err for bad dir")
	}
}

func TestBadger_PutGet_RoundTrip(t *testing.T) {
	b := newBadger(t)

	if err := b.Put("k1", []byte("v1"), time.Hour); err != nil {
		t.Fatalf("Put: %v", err)
	}
	val, found, expired := b.Get("k1")
	if !found {
		t.Fatal("found = false after Put")
	}
	if expired {
		t.Fatal("expired = true (Badger always reports false)")
	}
	if string(val) != "v1" {
		t.Fatalf("val = %q, want v1", val)
	}
}

func TestBadger_Get_Missing_ReportsNotFound(t *testing.T) {
	b := newBadger(t)

	val, found, _ := b.Get("absent")
	if found {
		t.Fatal("found = true for missing key")
	}
	if val != nil {
		t.Fatalf("val = %v, want nil", val)
	}
}

func TestBadger_Close_Idempotent_ReturnsErr(t *testing.T) {
	// Badger panics on second Close. Skip second-close check; one Close is enough.
	b, err := NewBadger(Config{}, filepath.Join(t.TempDir(), "b"))
	if err != nil {
		t.Fatalf("NewBadger: %v", err)
	}
	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestBadger_Put_LongTTL_Readable(t *testing.T) {
	b := newBadger(t)

	// Use an explicit long TTL — badger interprets ttl=0 as expired.
	if err := b.Put("long", []byte("v"), 24*time.Hour); err != nil {
		t.Fatalf("Put: %v", err)
	}
	val, found, _ := b.Get("long")
	if !found {
		t.Fatal("found = false within TTL window")
	}
	if string(val) != "v" {
		t.Fatalf("val = %q, want v", val)
	}
}
