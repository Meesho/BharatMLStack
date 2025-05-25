package ds

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetOperations(t *testing.T) {
	s := NewOrderedSet[int]()

	// Test Add
	s = s.Add(1).Add(2).Add(3)
	assert.True(t, s.Has(1), "Set should contain 1")
	assert.True(t, s.Has(2), "Set should contain 2")
	assert.True(t, s.Has(3), "Set should contain 3")

	// Test Remove
	s = s.Remove(2)
	assert.False(t, s.Has(2), "Set should not contain 2")

	// Test Intersection
	s2 := NewOrderedSet[int]().Add(3).Add(4)
	intersect := s.Intersection(s2)
	assert.True(t, intersect.Has(3), "Intersection should contain 3")
	assert.False(t, intersect.Has(1), "Intersection should not contain 1")
	assert.False(t, intersect.Has(4), "Intersection should not contain 4")

	// Test Union
	union := s.Union(s2)
	assert.True(t, union.Has(1), "Union should contain 1")
	assert.True(t, union.Has(3), "Union should contain 3")
	assert.True(t, union.Has(4), "Union should contain 4")

	// Test Difference
	diff := s.Difference(s2)
	assert.True(t, diff.Has(1), "Difference should contain 1")
	assert.False(t, diff.Has(3), "Difference should not contain 3")

	// Test From
	fromSet := NewOrderedSet[int]().From(s2)
	assert.True(t, fromSet.Has(3), "From should contain 3")
	assert.True(t, fromSet.Has(4), "From should contain 4")

	// Test FastKeyIterator
	s = NewOrderedSet[int]().Add(1).Add(2).Add(3)
	count := 0
	expectedSum := 6 // 1 + 2 + 3
	actualSum := 0

	s.FastKeyIterator(func(key int) {
		count++
		actualSum += key
	})

	assert.Equal(t, 3, count, "FastKeyIterator should visit exactly 3 elements")
	assert.Equal(t, expectedSum, actualSum, "FastKeyIterator should visit all elements")

	// Test FastKeyIterator with empty set
	emptySet := NewOrderedSet[int]()
	emptyCount := 0
	emptySet.FastKeyIterator(func(key int) {
		emptyCount++
	})
	assert.Equal(t, 0, emptyCount, "FastKeyIterator should not visit any elements in empty set")

	// Compare FastKeyIterator with regular KeyIterator
	regularCount := 0
	regularSum := 0
	s.KeyIterator(func(key int) bool {
		regularCount++
		regularSum += key
		return true
	})

	assert.Equal(t, count, regularCount, "FastKeyIterator and KeyIterator should visit same number of elements")
	assert.Equal(t, actualSum, regularSum, "FastKeyIterator and KeyIterator should visit same elements")

	// Test order preservation
	orderedSet := NewOrderedSet[int]().Add(3).Add(1).Add(2)
	expected := []int{3, 1, 2}
	actual := make([]int, 0, 3)

	orderedSet.FastKeyIterator(func(key int) {
		actual = append(actual, key)
	})

	assert.Equal(t, expected, actual, "FastKeyIterator should preserve insertion order")
}
