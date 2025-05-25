package ds

import "container/list"

// Set is the interface for a generic set.
type Set[T comparable] interface {
	Add(e T) Set[T]
	Remove(e T) Set[T]
	AddBatch(elements []T) Set[T]
	RemoveBatch(elements []T) Set[T]
	Has(e T) bool
	Intersection(s2 Set[T]) Set[T]
	Union(s2 Set[T]) Set[T]
	Difference(s2 Set[T]) Set[T]
	From(s2 Set[T]) Set[T]
	IsEmpty() bool
	AddAll(s2 Set[T]) Set[T]
	AddWithMeta(e T, meta interface{}) Set[T]
	GetMeta(e T) interface{}
	Iterator(f func(T, interface{}) bool)
	KeyIterator(f func(T) bool)
	Size() int
	FastIterator(f func(T, interface{}))
	FastKeyIterator(f func(T))
}

func NewOrderedSet[T comparable]() Set[T] {
	return &OrderedSet[T]{
		data:  make(map[T]*list.Element),
		order: list.New(),
	}
}

func NewOrderedSetWithCapacity[T comparable](capacity int) Set[T] {
	return &OrderedSet[T]{
		data:  make(map[T]*list.Element, capacity),
		order: list.New(),
	}
}

// OrderedSet is the concrete implementation of Set
type OrderedSet[T comparable] struct {
	data  map[T]*list.Element
	order *list.List
}

// entry stores key-value pair in the linked list
type entry[T comparable] struct {
	key   T
	value interface{}
}

// Add returns the current OrderedSet by adding a new element.
func (s *OrderedSet[T]) Add(e T) Set[T] {
	if _, exists := s.data[e]; !exists {
		// Create entry only once
		entry := &entry[T]{key: e}
		elem := s.order.PushBack(entry)
		s.data[e] = elem
	}
	return s
}

// AddWithMeta returns the current OrderedSet by adding a new element.
func (s *OrderedSet[T]) AddWithMeta(e T, meta interface{}) Set[T] {
	if elem, exists := s.data[e]; !exists {
		entry := &entry[T]{key: e, value: meta}
		elem := s.order.PushBack(entry)
		s.data[e] = elem
	} else {
		// Just update the value instead of creating new entry
		elem.Value.(*entry[T]).value = meta
	}
	return s
}

// Add batch operation support
func (s *OrderedSet[T]) AddBatch(elements []T) Set[T] {
	// Pre-allocate capacity if needed
	if len(s.data) == 0 {
		s.data = make(map[T]*list.Element, len(elements))
	}

	for _, e := range elements {
		if _, exists := s.data[e]; !exists {
			entry := &entry[T]{key: e}
			elem := s.order.PushBack(entry)
			s.data[e] = elem
		}
	}
	return s
}

func (s *OrderedSet[T]) RemoveBatch(elements []T) Set[T] {
	for _, e := range elements {
		if elem, exists := s.data[e]; exists {
			s.order.Remove(elem)
			delete(s.data, e)
		}
	}
	return s
}

// GetMeta returns the metadata for the element
func (s *OrderedSet[T]) GetMeta(e T) interface{} {
	if elem, exists := s.data[e]; exists {
		return elem.Value.(*entry[T]).value
	}
	return nil
}

// Remove returns the current OrderedSet by removing an element.
func (s *OrderedSet[T]) Remove(e T) Set[T] {
	if elem, exists := s.data[e]; exists {
		s.order.Remove(elem)
		delete(s.data, e)
	}
	return s
}

// Has returns true when the element is in the OrderedSet. Otherwise, returns false.
func (s *OrderedSet[T]) Has(e T) bool {
	_, exists := s.data[e]
	return exists
}

// Intersection optimization
func (s *OrderedSet[T]) Intersection(s2 Set[T]) Set[T] {
	result := NewOrderedSet[T]()
	s22 := s2.(*OrderedSet[T])

	// Early exit for empty sets
	if s.IsEmpty() || s22.IsEmpty() {
		return result
	}

	// Pre-allocate with smaller capacity
	smallerSize := len(s.data)
	if len(s22.data) < smallerSize {
		smallerSize = len(s22.data)
	}
	result = &OrderedSet[T]{
		data:  make(map[T]*list.Element, smallerSize),
		order: list.New(),
	}

	// Always iterate over the smaller set
	if len(s.data) > len(s22.data) {
		s, s22 = s22, s
	}

	// Use direct map access instead of Iterator for better performance
	for elem := s.order.Front(); elem != nil; elem = elem.Next() {
		ent := elem.Value.(*entry[T])
		if elem2, exists := s22.data[ent.key]; exists {
			// Copy metadata from the larger set if available
			meta := elem2.Value.(*entry[T]).value
			if meta == nil {
				meta = ent.value
			}
			result.AddWithMeta(ent.key, meta)
		}
	}
	return result
}

// Union optimization
func (s *OrderedSet[T]) Union(s2 Set[T]) Set[T] {
	s22 := s2.(*OrderedSet[T])

	// Early exit for empty sets
	if s.IsEmpty() {
		result := NewOrderedSet[T]()
		return result.From(s22)
	}
	if s22.IsEmpty() {
		result := NewOrderedSet[T]()
		return result.From(s)
	}

	// Pre-allocate with combined capacity
	result := &OrderedSet[T]{
		data:  make(map[T]*list.Element, len(s.data)+len(s22.data)),
		order: list.New(),
	}

	// First, copy all elements from the first set
	for elem := s.order.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*entry[T])
		result.AddWithMeta(entry.key, entry.value)
	}

	// Then add elements from second set (only if they don't exist)
	for elem := s22.order.Front(); elem != nil; elem = elem.Next() {
		ent := elem.Value.(*entry[T])
		if existing, exists := result.data[ent.key]; exists {
			// Update metadata if the second set has non-nil metadata
			if ent.value != nil {
				existing.Value.(*entry[T]).value = ent.value
			}
		} else {
			result.AddWithMeta(ent.key, ent.value)
		}
	}

	return result
}

// Difference returns a new OrderedSet containing elements that are present in the first set but not in the second set.
func (s *OrderedSet[T]) Difference(s2 Set[T]) Set[T] {
	result := NewOrderedSet[T]()

	s.Iterator(func(key T, value interface{}) bool {
		if _, exists := s2.(*OrderedSet[T]).data[key]; !exists {
			result.AddWithMeta(key, value)
		}
		return true
	})

	return result
}

// From initializes the set with elements from another set.
func (s *OrderedSet[T]) From(s2 Set[T]) Set[T] {
	s22 := s2.(*OrderedSet[T])
	for _, elem := range s22.data {
		ent := elem.Value.(*entry[T])
		s.AddWithMeta(ent.key, ent.value)
	}
	return s
}

// IsEmpty returns true if the set is empty
func (s *OrderedSet[T]) IsEmpty() bool {
	return len(s.data) == 0
}

// AddAll adds all elements from the given set to the current set
func (s *OrderedSet[T]) AddAll(s2 Set[T]) Set[T] {
	s22 := s2.(*OrderedSet[T])
	for _, elem := range s22.data {
		ent := elem.Value.(*entry[T])
		s.AddWithMeta(ent.key, ent.value)
	}
	return s
}

func (s *OrderedSet[T]) Iterator(f func(T, interface{}) bool) {

	for e := s.order.Front(); e != nil; {
		next := e.Next() // Store next before potential deletion
		ent := e.Value.(*entry[T])

		if !f(ent.key, ent.value) {
			return
		}

		// If key is deleted inside the function, remove it from the map
		if _, exists := s.data[ent.key]; !exists {
			s.order.Remove(e)
		}

		e = next
	}
}

// Add a more efficient iterator that doesn't check for deletion
func (s *OrderedSet[T]) FastIterator(f func(T, interface{})) {
	for e := s.order.Front(); e != nil; e = e.Next() {
		ent := e.Value.(*entry[T])
		f(ent.key, ent.value)
	}
}

// KeyIterator returns an iterator for the keys
func (s *OrderedSet[T]) KeyIterator(f func(T) bool) {
	for e := s.order.Front(); e != nil; {
		next := e.Next() // Store next before potential deletion
		ent := e.Value.(*entry[T])

		if !f(ent.key) {
			return
		}

		// If key is deleted inside the function, remove it from the map
		if _, exists := s.data[ent.key]; !exists {
			s.order.Remove(e)
		}

		e = next
	}
}

// FastKeyIterator provides a more efficient key-only iteration without deletion checks
func (s *OrderedSet[T]) FastKeyIterator(f func(T)) {
	for e := s.order.Front(); e != nil; e = e.Next() {
		ent := e.Value.(*entry[T])
		f(ent.key)
	}
}

func (s *OrderedSet[T]) Size() int {
	return len(s.data)
}

// Helper method for bulk operations
func (s *OrderedSet[T]) bulkAdd(entries []*entry[T]) {
	for _, entry := range entries {
		if _, exists := s.data[entry.key]; !exists {
			elem := s.order.PushBack(entry)
			s.data[entry.key] = elem
		}
	}
}

// NewOrderedSetFromSlice creates a new OrderedSet from a slice
func NewOrderedSetFromSlice[T comparable](slice []T) Set[T] {
	set := &OrderedSet[T]{
		data:  make(map[T]*list.Element, len(slice)),
		order: list.New(),
	}

	for _, item := range slice {
		set.Add(item)
	}

	return set
}
