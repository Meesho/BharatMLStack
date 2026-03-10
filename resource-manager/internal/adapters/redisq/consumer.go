package redisq

import (
	"context"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
)

func (p *InMemoryQueueAdapter) ConsumeWatchIntent(_ context.Context, queueID int) (*models.WatchIntent, string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	msgs := p.queues[queueID]
	if len(msgs) == 0 {
		return nil, "", nil
	}
	msg := msgs[0]
	p.queues[queueID] = msgs[1:]
	intent := msg.intent
	return &intent, msg.id, nil
}

func (p *InMemoryQueueAdapter) Ack(_ context.Context, _ int, _ string) error {
	return nil
}

func (p *InMemoryQueueAdapter) Nack(_ context.Context, _ int, _ string) error {
	// No-op in mock adapter. Worker retries are handled by next poll cycle in this phase.
	return nil
}
