// Package util provides utility functions for handling slices and batching.

package utils

import (
	"errors"
)

// Batch is a generic function that batches elements of a list based on the specified batch size.
// It takes a pointer to a slice and a batchSize, and returns a 2D slice of batches.
func Batch[T any](in []T, batchSize int) ([][]T, error) {
	if batchSize <= 0 {
		return nil, errors.New("batch size must be greater than 0")
	}
	var batches [][]T
	for batchSize < len(in) {
		in, batches = in[batchSize:], append(batches, in[0:batchSize:batchSize])
	}
	if len(in) > 0 {
		batches = append(batches, in)
	}
	return batches, nil
}
