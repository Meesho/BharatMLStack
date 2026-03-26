package indexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQdrantIndexer_getKey(t *testing.T) {
	q := &QdrantIndexer{}
	key := q.getKey("entity1", "model1", "variant1", 3)
	assert.Equal(t, "entity1|model1|variant1|3", key)
}

func TestQdrantIndexer_getKey_ZeroVersion(t *testing.T) {
	q := &QdrantIndexer{}
	key := q.getKey("e", "m", "v", 0)
	assert.Equal(t, "e|m|v|0", key)
}

func TestEventType_Constants(t *testing.T) {
	assert.Equal(t, EventType("UPSERT"), Upsert)
	assert.Equal(t, EventType("DELETE"), Delete)
	assert.Equal(t, EventType("UPSERT_PAYLOAD"), UpsertPayload)
}
