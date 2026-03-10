package etcd

import (
	"context"
	"testing"
)

func TestMemoryQueueLeaseStoreAcquireAndRelease(t *testing.T) {
	store := NewMemoryQueueLeaseStore()
	handle, ok, err := store.Acquire(context.Background(), 1, "consumer-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected acquire success")
	}

	_, ok, err = store.Acquire(context.Background(), 1, "consumer-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected acquire conflict for same queue")
	}

	if err := store.Release(context.Background(), handle); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}

	_, ok, err = store.Acquire(context.Background(), 1, "consumer-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected acquire success after release")
	}
}
