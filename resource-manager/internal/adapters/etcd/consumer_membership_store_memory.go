package etcd

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
)

type MemoryConsumerMembershipStore struct {
	mu      sync.Mutex
	members map[string]map[string]models.LeaseHandle
}

func NewMemoryConsumerMembershipStore() *MemoryConsumerMembershipStore {
	return &MemoryConsumerMembershipStore{
		members: make(map[string]map[string]models.LeaseHandle),
	}
}

func (s *MemoryConsumerMembershipStore) Register(_ context.Context, groupID, consumerID string) (models.LeaseHandle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.members[groupID]; !ok {
		s.members[groupID] = make(map[string]models.LeaseHandle)
	}
	handle := models.LeaseHandle{
		Key:     fmt.Sprintf("memory-membership/%s/%s", groupID, consumerID),
		LeaseID: time.Now().UnixNano(),
		Owner:   consumerID,
	}
	s.members[groupID][consumerID] = handle
	return handle, nil
}

func (s *MemoryConsumerMembershipStore) KeepAlive(_ context.Context, _ models.LeaseHandle) error {
	return nil
}

func (s *MemoryConsumerMembershipStore) ListMembers(_ context.Context, groupID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	group := s.members[groupID]
	out := make([]string, 0, len(group))
	for member := range group {
		out = append(out, member)
	}
	sort.Strings(out)
	return out, nil
}

func (s *MemoryConsumerMembershipStore) Revoke(_ context.Context, handle models.LeaseHandle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for groupID, members := range s.members {
		for memberID, lease := range members {
			if lease.LeaseID == handle.LeaseID {
				delete(s.members[groupID], memberID)
			}
		}
	}
	return nil
}
