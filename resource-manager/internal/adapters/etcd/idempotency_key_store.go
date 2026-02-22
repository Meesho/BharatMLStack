package etcd

import (
	"context"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
)

const cleanupEveryPuts = 100

type memoryIdempotencyRecord struct {
	record    models.IdempotencyRecord
	expiresAt time.Time
}

type MemoryIdempotencyKeyStore struct {
	mu         sync.RWMutex
	records    map[string]memoryIdempotencyRecord
	ttl        time.Duration
	putCounter uint64
}

func NewMemoryIdempotencyKeyStore(ttlSeconds ...int64) *MemoryIdempotencyKeyStore {
	ttl := 24 * time.Hour
	if len(ttlSeconds) > 0 && ttlSeconds[0] > 0 {
		ttl = time.Duration(ttlSeconds[0]) * time.Second
	}
	return &MemoryIdempotencyKeyStore{
		records: make(map[string]memoryIdempotencyRecord),
		ttl:     ttl,
	}
}

func (s *MemoryIdempotencyKeyStore) Get(_ context.Context, scope, key string) (*models.IdempotencyRecord, error) {
	composite := scope + "::" + key

	s.mu.RLock()
	record, ok := s.records[composite]
	if !ok {
		s.mu.RUnlock()
		return nil, nil
	}
	if time.Now().After(record.expiresAt) {
		s.mu.RUnlock()
		s.mu.Lock()
		delete(s.records, composite)
		s.mu.Unlock()
		return nil, nil
	}
	copied := record.record
	s.mu.RUnlock()
	return &copied, nil
}

func (s *MemoryIdempotencyKeyStore) Put(_ context.Context, scope, key string, record models.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[scope+"::"+key] = memoryIdempotencyRecord{
		record:    record,
		expiresAt: time.Now().Add(s.ttl),
	}
	s.putCounter++
	if s.putCounter%cleanupEveryPuts == 0 {
		s.cleanupExpiredLocked()
	}
	return nil
}

func (s *MemoryIdempotencyKeyStore) cleanupExpiredLocked() {
	now := time.Now()
	for key, record := range s.records {
		if now.After(record.expiresAt) {
			delete(s.records, key)
		}
	}
}
