package etcd

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const idempotencyBasePath = "/idempotency"

type EtcdIdempotencyKeyStore struct {
	client *clientv3.Client
}

func NewEtcdIdempotencyKeyStore(client *clientv3.Client) *EtcdIdempotencyKeyStore {
	return &EtcdIdempotencyKeyStore{client: client}
}

func (s *EtcdIdempotencyKeyStore) Get(ctx context.Context, scope, key string) (*models.IdempotencyRecord, error) {
	resp, err := s.client.Get(ctx, idempotencyKey(scope, key))
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var record models.IdempotencyRecord
	if err := json.Unmarshal(resp.Kvs[0].Value, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *EtcdIdempotencyKeyStore) Put(ctx context.Context, scope, key string, record models.IdempotencyRecord) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = s.client.Put(ctx, idempotencyKey(scope, key), string(raw))
	return err
}

func idempotencyKey(scope, key string) string {
	return idempotencyBasePath + "/" + url.PathEscape(scope) + "/" + url.PathEscape(key)
}
