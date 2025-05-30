package utils

// Set is the set type
type Set[T comparable] map[T]struct{}

// Add returns the current Set by adding a new element.
func (s Set[T]) Add(e T) Set[T] {
	s[e] = struct{}{}
	return s
}

// Remove returns the current Set by removing an element.
func (s Set[T]) Remove(e T) Set[T] {
	delete(s, e)
	return s
}

// Has returns true when the element is in the Set. Otherwise, returns false.
func (s Set[T]) Has(e T) bool {
	_, ok := s[e]
	return ok
}

// Intersection returns a new Set containing elements that are present in both sets.
func (s Set[T]) Intersection(s2 Set[T]) Set[T] {
	result := make(Set[T]) // Use the generic type instead of hardcoding int
	for elem := range s {
		if s2.Has(elem) {
			result[elem] = struct{}{}
		}
	}
	return result
}

func (s Set[T]) Union(s2 Set[T]) Set[T] {
	result := make(Set[T])
	for elem := range s {
		result[elem] = struct{}{}
	}
	for elem := range s2 {
		result[elem] = struct{}{}
	}
	return result
}

// Difference returns a new Set containing elements that are present in the first set but not in the second set.
func (s Set[T]) Difference(s2 Set[T]) Set[T] {
	result := make(Set[T]) // Use the generic type
	for elem := range s {
		if !s2.Has(elem) {
			result[elem] = struct{}{}
		}
	}
	return result
}

// From returns a new Set with the elements provided.
func (s Set[T]) From(s2 Set[T]) Set[T] {
	for elem := range s2 {
		s[elem] = struct{}{}
	}
	return s
}
