package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBatch(t *testing.T) {
	in := []int{1, 2, 3, 4, 5}
	batches, _ := Batch(in, 2)
	assert.Equal(t, 3, len(batches), "Expected length of 3, got %d", len(batches))
	assert.Equal(t, 2, len(batches[0]), "Expected length of 2, got %d", len(batches[0]))
	assert.Equal(t, 2, len(batches[1]), "Expected length of 2, got %d", len(batches[1]))
	assert.Equal(t, 1, len(batches[2]), "Expected length of 1, got %d", len(batches[2]))
}
