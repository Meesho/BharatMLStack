package qdrant

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
)

// LoadEmbeddingsFromFile loads embeddings from a JSON file
// Supports both wrapped format {"embeddings": [...]} and direct array format
func LoadEmbeddingsFromFile(filepath string) ([][]float64, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read embeddings file: %w", err)
	}

	// Try to unmarshal as wrapped format first
	var wrappedEmbeddings struct {
		Embeddings [][]float64 `json:"embeddings"`
	}
	if err := json.Unmarshal(data, &wrappedEmbeddings); err == nil && len(wrappedEmbeddings.Embeddings) > 0 {
		return wrappedEmbeddings.Embeddings, nil
	}

	// Fallback to direct array format
	var embeddings [][]float64
	if err := json.Unmarshal(data, &embeddings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embeddings: %w", err)
	}

	return embeddings, nil
}

// CalculateStatistics computes mean, std dev, min, max, and median of a float64 slice
func CalculateStatistics(values []float64) (mean, std, min, max, median float64) {
	if len(values) == 0 {
		return 0, 0, 0, 0, 0
	}

	// Mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))

	// Std dev
	sumSq := 0.0
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	std = math.Sqrt(sumSq / float64(len(values)))

	// Min/Max
	min = values[0]
	max = values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Median
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	midIdx := len(sorted) / 2
	if len(sorted)%2 == 0 {
		median = (sorted[midIdx-1] + sorted[midIdx]) / 2
	} else {
		median = sorted[midIdx]
	}

	return
}

// CollectionNameFromCentroid generates a collection name from a centroid ID
func CollectionNameFromCentroid(prefix string, centroidID int) string {
	return fmt.Sprintf("%s_c%04d", prefix, centroidID+1)
}
