package etcd

import (
	"context"
	"sort"
	"strings"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdConsumerMembershipStore struct {
	client     *clientv3.Client
	ttlSeconds int64
}

func NewEtcdConsumerMembershipStore(client *clientv3.Client, ttlSeconds int64) *EtcdConsumerMembershipStore {
	if ttlSeconds <= 0 {
		ttlSeconds = 15
	}
	return &EtcdConsumerMembershipStore{
		client:     client,
		ttlSeconds: ttlSeconds,
	}
}

func (s *EtcdConsumerMembershipStore) Register(ctx context.Context, groupID, consumerID string) (models.LeaseHandle, error) {
	key := membershipKey(groupID, consumerID)
	lease, err := s.client.Grant(ctx, s.ttlSeconds)
	if err != nil {
		return models.LeaseHandle{}, err
	}
	_, err = s.client.Put(ctx, key, consumerID, clientv3.WithLease(lease.ID))
	if err != nil {
		return models.LeaseHandle{}, err
	}
	return models.LeaseHandle{
		Key:     key,
		LeaseID: int64(lease.ID),
		Owner:   consumerID,
	}, nil
}

func (s *EtcdConsumerMembershipStore) KeepAlive(ctx context.Context, handle models.LeaseHandle) error {
	if handle.LeaseID == 0 {
		return nil
	}
	_, err := s.client.KeepAliveOnce(ctx, clientv3.LeaseID(handle.LeaseID))
	return err
}

func (s *EtcdConsumerMembershipStore) ListMembers(ctx context.Context, groupID string) ([]string, error) {
	resp, err := s.client.Get(ctx, membershipPrefix(groupID), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	members := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		val := strings.TrimSpace(string(kv.Value))
		if val == "" {
			continue
		}
		members = append(members, val)
	}
	sort.Strings(members)
	return members, nil
}

func (s *EtcdConsumerMembershipStore) Revoke(ctx context.Context, handle models.LeaseHandle) error {
	if handle.LeaseID != 0 {
		_, err := s.client.Revoke(ctx, clientv3.LeaseID(handle.LeaseID))
		return err
	}
	if handle.Key == "" {
		return nil
	}
	_, err := s.client.Delete(ctx, handle.Key)
	return err
}

func membershipPrefix(groupID string) string {
	return appBasePath() + "/consumer-groups/" + strings.TrimSpace(groupID) + "/members"
}

func membershipKey(groupID, consumerID string) string {
	return membershipPrefix(groupID) + "/" + strings.TrimSpace(consumerID)
}
