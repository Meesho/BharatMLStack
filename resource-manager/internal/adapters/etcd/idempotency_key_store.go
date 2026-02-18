package etcd

import (
	"context"
	"sync"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
)

type MemoryIdempotencyKeyStore struct {
	mu      sync.RWMutex
	records map[string]models.IdempotencyRecord
}

func NewMemoryIdempotencyKeyStore() *MemoryIdempotencyKeyStore {
	return &MemoryIdempotencyKeyStore{
		records: make(map[string]models.IdempotencyRecord),
	}
}

func (s *MemoryIdempotencyKeyStore) Get(_ context.Context, scope, key string) (*models.IdempotencyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.records[scope+"::"+key]
	if !ok {
		return nil, nil
	}
	copied := record
	return &copied, nil
}

func (s *MemoryIdempotencyKeyStore) Put(_ context.Context, scope, key string, record models.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[scope+"::"+key] = record
	return nil
}
