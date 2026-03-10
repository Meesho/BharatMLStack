package redisq

import (
	"testing"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
)

func TestHashQueuePartitionerDeterministic(t *testing.T) {
	partitioner := NewHashQueuePartitioner(8)
	intent := models.WatchIntent{
		Operation: "CREATE_DEPLOYABLE",
		Resource: models.WatchResource{
			Namespace:     "int",
			LabelSelector: "name=test",
			Name:          "test",
		},
	}

	a := partitioner.Partition(intent)
	b := partitioner.Partition(intent)
	if a != b {
		t.Fatalf("expected deterministic partition, got %d and %d", a, b)
	}
	if a < 1 || a > 8 {
		t.Fatalf("partition out of range: %d", a)
	}
}
