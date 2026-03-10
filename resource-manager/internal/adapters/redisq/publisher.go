package redisq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
)

type queueMessage struct {
	id     string
	intent models.WatchIntent
}

type InMemoryQueueAdapter struct {
	mu       sync.Mutex
	queues   map[int][]queueMessage
	sequence uint64
}

func NewInMemoryQueueAdapter() *InMemoryQueueAdapter {
	return &InMemoryQueueAdapter{
		queues: make(map[int][]queueMessage),
	}
}

func (p *InMemoryQueueAdapter) PublishWatchIntent(_ context.Context, queueID int, intent models.WatchIntent) (models.PublishResult, error) {
	_, err := json.Marshal(intent)
	if err != nil {
		return models.PublishResult{}, err
	}
	if queueID <= 0 {
		queueID = 1
	}
	id := atomic.AddUint64(&p.sequence, 1)
	msgID := fmt.Sprintf("msg-%d", id)
	p.mu.Lock()
	p.queues[queueID] = append(p.queues[queueID], queueMessage{
		id:     msgID,
		intent: intent,
	})
	p.mu.Unlock()
	return models.PublishResult{MessageID: msgID}, nil
}
