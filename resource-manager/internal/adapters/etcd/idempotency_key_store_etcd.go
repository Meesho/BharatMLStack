package etcd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	"github.com/rs/zerolog/log"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const defaultIdempotencyTTLSeconds int64 = 24 * 60 * 60

type EtcdIdempotencyKeyStore struct {
	client     *clientv3.Client
	ttlSeconds int64
}

func NewEtcdIdempotencyKeyStore(client *clientv3.Client, ttlSeconds ...int64) *EtcdIdempotencyKeyStore {
	ttl := defaultIdempotencyTTLSeconds
	if len(ttlSeconds) > 0 && ttlSeconds[0] > 0 {
		ttl = ttlSeconds[0]
	}
	return &EtcdIdempotencyKeyStore{
		client:     client,
		ttlSeconds: ttl,
	}
}

func (s *EtcdIdempotencyKeyStore) Get(ctx context.Context, scope, key string) (*models.IdempotencyRecord, error) {
	etcdKey := idempotencyKey(scope, key)
	resp, err := s.client.Get(ctx, etcdKey)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		log.Debug().Str("scope", scope).Str("idempotency_key", key).Str("etcd_key", etcdKey).Msg("idempotency miss in etcd")
		return nil, nil
	}

	var record models.IdempotencyRecord
	if err := json.Unmarshal(resp.Kvs[0].Value, &record); err != nil {
		return nil, err
	}
	log.Info().Str("scope", scope).Str("idempotency_key", key).Str("etcd_key", etcdKey).Msg("idempotency hit in etcd")
	return &record, nil
}

func (s *EtcdIdempotencyKeyStore) Put(ctx context.Context, scope, key string, record models.IdempotencyRecord) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	lease, err := s.client.Grant(ctx, s.ttlSeconds)
	if err != nil {
		return fmt.Errorf("failed to create idempotency ttl lease: %w", err)
	}
	etcdKey := idempotencyKey(scope, key)
	_, err = s.client.Put(ctx, etcdKey, string(raw), clientv3.WithLease(lease.ID))
	if err == nil {
		log.Info().
			Str("scope", scope).
			Str("idempotency_key", key).
			Str("etcd_key", etcdKey).
			Int64("ttl_seconds", s.ttlSeconds).
			Msg("stored idempotency record in etcd with ttl lease")
	}
	return err
}

func idempotencyKey(scope, key string) string {
	return idempotencyBasePath() + "/" + scopeFingerprint(scope) + "/" + strings.TrimSpace(key)
}

func idempotencyBasePath() string {
	base := strings.TrimSpace(os.Getenv("ETCD_APP_NAME"))
	if base == "" {
		base = strings.TrimSpace(os.Getenv("APP_NAME"))
	}
	if base == "" {
		base = "resource-manager"
	}
	return "/config/" + base + "/idempotency"
}

func scopeFingerprint(scope string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(scope))))
	return hex.EncodeToString(sum[:])
}
