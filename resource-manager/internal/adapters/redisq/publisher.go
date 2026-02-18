package redisq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
)

type InMemoryPublisher struct {
	sequence uint64
}

func NewInMemoryPublisher() *InMemoryPublisher {
	return &InMemoryPublisher{}
}

func (p *InMemoryPublisher) PublishWatchIntent(_ context.Context, intent models.WatchIntent) (models.PublishResult, error) {
	_, err := json.Marshal(intent)
	if err != nil {
		return models.PublishResult{}, err
	}
	id := atomic.AddUint64(&p.sequence, 1)
	return models.PublishResult{MessageID: fmt.Sprintf("msg-%d", id)}, nil
}
