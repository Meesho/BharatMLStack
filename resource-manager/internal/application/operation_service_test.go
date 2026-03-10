package application

import (
	"context"
	"testing"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
)

type testPublisher struct {
	queueID int
	intent  models.WatchIntent
}

func (p *testPublisher) PublishWatchIntent(_ context.Context, queueID int, intent models.WatchIntent) (models.PublishResult, error) {
	p.queueID = queueID
	p.intent = intent
	return models.PublishResult{MessageID: "msg-1"}, nil
}

type testPartitioner struct {
	queueID int
}

func (p *testPartitioner) Partition(_ models.WatchIntent) int {
	return p.queueID
}

func TestSubmitAsyncOperationAssignsQueueIDAndPublishesToSameQueue(t *testing.T) {
	publisher := &testPublisher{}
	partitioner := &testPartitioner{queueID: 3}
	svc := NewOperationService(publisher, partitioner, nil, nil)

	_, err := svc.SubmitAsyncOperation(
		context.Background(),
		"req-1",
		"CREATE_DEPLOYABLE",
		models.WatchResource{Kind: "Deployment", Namespace: "int", LabelSelector: "name=a"},
		"Available",
		models.Callback{URL: "https://example.com"},
		models.WorkflowContext{RunID: "run-1"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if publisher.queueID != 3 {
		t.Fatalf("expected queue id 3, got %d", publisher.queueID)
	}
	if publisher.intent.QueueID != 3 {
		t.Fatalf("expected intent queue id 3, got %d", publisher.intent.QueueID)
	}
}
