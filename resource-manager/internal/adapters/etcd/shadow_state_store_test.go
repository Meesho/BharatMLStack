package etcd

import (
	"context"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	rmtypes "github.com/Meesho/BharatMLStack/resource-manager/internal/types"
)

func TestProcureAndReleaseFlow(t *testing.T) {
	store := NewMemoryShadowStateStore(map[string][]models.ShadowDeployable{
		"int": {
			{
				Name:          "d1",
				NodePool:      "g2-std-8",
				DNS:           "d1.int",
				State:         rmtypes.ShadowStateFree,
				Version:       1,
				LastUpdatedAt: time.Now(),
			},
		},
	})

	_, conflict, err := store.Procure(context.Background(), "int", "d1", "run-1", "plan-1")
	if err != nil {
		t.Fatalf("unexpected procure error: %v", err)
	}
	if conflict {
		t.Fatalf("expected no conflict")
	}

	_, conflict, err = store.Procure(context.Background(), "int", "d1", "run-2", "plan-2")
	if err != nil {
		t.Fatalf("unexpected second procure error: %v", err)
	}
	if !conflict {
		t.Fatalf("expected conflict on second procure")
	}

	_, conflict, err = store.Release(context.Background(), "int", "d1", "run-2")
	if err == nil {
		t.Fatalf("expected invalid owner error")
	}
	if conflict {
		t.Fatalf("expected no CAS conflict when owner is invalid")
	}

	_, conflict, err = store.Release(context.Background(), "int", "d1", "run-1")
	if err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
	if conflict {
		t.Fatalf("expected no conflict on valid release")
	}
}
