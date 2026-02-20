package etcd

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const operationIntentBasePath = "/operation-intents"

type EtcdOperationStore struct {
	client *clientv3.Client
}

func NewEtcdOperationStore(client *clientv3.Client) *EtcdOperationStore {
	return &EtcdOperationStore{client: client}
}

func (s *EtcdOperationStore) SaveWatchIntent(ctx context.Context, intent models.WatchIntent) error {
	raw, err := json.Marshal(intent)
	if err != nil {
		return err
	}
	_, err = s.client.Put(ctx, operationIntentKey(intent.Operation, intent.RequestID), string(raw))
	return err
}

func operationIntentKey(operation, requestID string) string {
	return operationIntentBasePath + "/" + url.PathEscape(operation) + "/" + url.PathEscape(requestID)
}
