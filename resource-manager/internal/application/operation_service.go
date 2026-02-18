package application

import (
	"context"
	"time"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/ports"
)

type OperationService struct {
	publisher ports.QueuePublisher
	kube      ports.KubernetesExecutor
}

func NewOperationService(publisher ports.QueuePublisher, kube ports.KubernetesExecutor) *OperationService {
	return &OperationService{publisher: publisher, kube: kube}
}

func (s *OperationService) CreateDeployable(ctx context.Context, env string, spec models.DeployableSpec) error {
	return s.kube.CreateDeployable(ctx, env, spec)
}

func (s *OperationService) LoadModel(ctx context.Context, env string, spec models.ModelLoadSpec) error {
	return s.kube.LoadModel(ctx, env, spec)
}

func (s *OperationService) TriggerJob(ctx context.Context, env string, spec models.JobSpec) error {
	return s.kube.TriggerJob(ctx, env, spec)
}

func (s *OperationService) RestartDeployable(ctx context.Context, env string, spec models.RestartSpec) error {
	return s.kube.RestartDeployable(ctx, env, spec)
}

func (s *OperationService) SubmitAsyncOperation(
	ctx context.Context,
	requestID string,
	operation string,
	resource models.WatchResource,
	desiredCondition string,
	callback models.Callback,
	workflow models.WorkflowContext,
) (models.PublishResult, error) {
	intent := models.WatchIntent{
		SchemaVersion:    "v1",
		RequestID:        requestID,
		Operation:        operation,
		Resource:         resource,
		DesiredCondition: desiredCondition,
		Callback:         callback,
		Workflow:         workflow,
		CreatedAt:        time.Now().UTC(),
	}
	return s.publisher.PublishWatchIntent(ctx, intent)
}
