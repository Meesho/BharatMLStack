package etcd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdQueueLeaseStore struct {
	client     *clientv3.Client
	ttlSeconds int64
}

func NewEtcdQueueLeaseStore(client *clientv3.Client, ttlSeconds int64) *EtcdQueueLeaseStore {
	if ttlSeconds <= 0 {
		ttlSeconds = 15
	}
	return &EtcdQueueLeaseStore{client: client, ttlSeconds: ttlSeconds}
}

func (s *EtcdQueueLeaseStore) Acquire(ctx context.Context, queueID int, owner string) (models.LeaseHandle, bool, error) {
	key := queueLeaseKey(queueID)
	lease, err := s.client.Grant(ctx, s.ttlSeconds)
	if err != nil {
		return models.LeaseHandle{}, false, err
	}
	resp, err := s.client.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, owner, clientv3.WithLease(lease.ID))).
		Commit()
	if err != nil {
		return models.LeaseHandle{}, false, err
	}
	if !resp.Succeeded {
		return models.LeaseHandle{}, false, nil
	}
	return models.LeaseHandle{
		Key:     key,
		LeaseID: int64(lease.ID),
		Owner:   owner,
	}, true, nil
}

func (s *EtcdQueueLeaseStore) KeepAlive(ctx context.Context, handle models.LeaseHandle) error {
	if handle.LeaseID == 0 {
		return fmt.Errorf("invalid lease handle: lease id is empty")
	}
	_, err := s.client.KeepAliveOnce(ctx, clientv3.LeaseID(handle.LeaseID))
	return err
}

func (s *EtcdQueueLeaseStore) Release(ctx context.Context, handle models.LeaseHandle) error {
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

func queueLeaseKey(queueID int) string {
	return appBasePath() + "/queue-leases/" + strconv.Itoa(queueID)
}

func appBasePath() string {
	base := strings.TrimSpace(os.Getenv("ETCD_APP_NAME"))
	if base == "" {
		base = strings.TrimSpace(os.Getenv("APP_NAME"))
	}
	if base == "" {
		base = "resource-manager"
	}
	return "/config/" + base
}
