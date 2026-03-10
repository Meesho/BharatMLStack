package ports

import (
	"context"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	rmtypes "github.com/Meesho/BharatMLStack/resource-manager/internal/types"
)

type KubernetesExecutor interface {
	CreateDeployable(ctx context.Context, env string, spec models.DeployableSpec) error
	LoadModel(ctx context.Context, env string, spec models.ModelLoadSpec) error
	TriggerJob(ctx context.Context, env string, spec models.JobSpec) error
	RestartDeployable(ctx context.Context, env string, spec models.RestartSpec) error
}

type ShadowStateStore interface {
	List(ctx context.Context, env string, filter models.ShadowFilter) ([]models.ShadowDeployable, error)
	Procure(ctx context.Context, env, name, runID, plan string) (models.ShadowDeployable, bool, error)
	Release(ctx context.Context, env, name, runID string) (models.ShadowDeployable, bool, error)
	ChangeMinPodCount(ctx context.Context, env, name string, action rmtypes.Action, count int) (models.ShadowDeployable, error)
}

type ShadowCache interface {
	ListShadowDeployables(filter models.ShadowFilter) []models.ShadowDeployable
}

type QueuePublisher interface {
	PublishWatchIntent(ctx context.Context, queueID int, intent models.WatchIntent) (models.PublishResult, error)
}

type QueueConsumer interface {
	ConsumeWatchIntent(ctx context.Context, queueID int) (*models.WatchIntent, string, error)
	Ack(ctx context.Context, queueID int, messageID string) error
	Nack(ctx context.Context, queueID int, messageID string) error
}

type QueuePartitioner interface {
	Partition(intent models.WatchIntent) int
}

type OperationStore interface {
	SaveWatchIntent(ctx context.Context, intent models.WatchIntent) error
}

type IdempotencyKeyStore interface {
	Get(ctx context.Context, scope, key string) (*models.IdempotencyRecord, error)
	Put(ctx context.Context, scope, key string, record models.IdempotencyRecord) error
}

type QueueLeaseStore interface {
	Acquire(ctx context.Context, queueID int, owner string) (models.LeaseHandle, bool, error)
	KeepAlive(ctx context.Context, handle models.LeaseHandle) error
	Release(ctx context.Context, handle models.LeaseHandle) error
}

type ConsumerMembershipStore interface {
	Register(ctx context.Context, groupID, consumerID string) (models.LeaseHandle, error)
	KeepAlive(ctx context.Context, handle models.LeaseHandle) error
	ListMembers(ctx context.Context, groupID string) ([]string, error)
	Revoke(ctx context.Context, handle models.LeaseHandle) error
}

type WatchManager interface {
	Watch(ctx context.Context, intent models.WatchIntent) error
}

type CallbackDispatcher interface {
	Dispatch(ctx context.Context, intent models.WatchIntent, watchErr error) error
}
