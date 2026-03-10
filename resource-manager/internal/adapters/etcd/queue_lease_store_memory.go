package etcd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
)

type MemoryQueueLeaseStore struct {
	mu     sync.Mutex
	leases map[int]models.LeaseHandle
}

func NewMemoryQueueLeaseStore() *MemoryQueueLeaseStore {
	return &MemoryQueueLeaseStore{
		leases: make(map[int]models.LeaseHandle),
	}
}

func (s *MemoryQueueLeaseStore) Acquire(_ context.Context, queueID int, owner string) (models.LeaseHandle, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.leases[queueID]; ok {
		return models.LeaseHandle{}, false, nil
	}
	handle := models.LeaseHandle{
		Key:     fmt.Sprintf("memory-queue-lease/%d", queueID),
		LeaseID: time.Now().UnixNano(),
		Owner:   owner,
	}
	s.leases[queueID] = handle
	return handle, true, nil
}

func (s *MemoryQueueLeaseStore) KeepAlive(_ context.Context, _ models.LeaseHandle) error {
	return nil
}

func (s *MemoryQueueLeaseStore) Release(_ context.Context, handle models.LeaseHandle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for queueID, lease := range s.leases {
		if lease.LeaseID == handle.LeaseID {
			delete(s.leases, queueID)
			break
		}
	}
	return nil
}
