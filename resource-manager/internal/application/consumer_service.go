package application

import (
	"context"
	"time"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/ports"
	"github.com/rs/zerolog/log"
)

type ConsumerService struct {
	consumerID      string
	groupID         string
	partitionCount  int
	queueConsumer   ports.QueueConsumer
	queueLease      ports.QueueLeaseStore
	membership      ports.ConsumerMembershipStore
	watchManager    ports.WatchManager
	callbacks       ports.CallbackDispatcher
	pollInterval    time.Duration
	renewInterval   time.Duration
	rebalanceTicker time.Duration
	drainTimeout    time.Duration

	workers map[int]*partitionWorker
}

type partitionWorker struct {
	cancel context.CancelFunc
	done   chan struct{}
	lease  models.LeaseHandle
}

func NewConsumerService(
	consumerID string,
	groupID string,
	partitionCount int,
	queueConsumer ports.QueueConsumer,
	queueLease ports.QueueLeaseStore,
	membership ports.ConsumerMembershipStore,
	watchManager ports.WatchManager,
	callbacks ports.CallbackDispatcher,
	pollInterval time.Duration,
	renewInterval time.Duration,
	drainTimeout time.Duration,
) *ConsumerService {
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	if renewInterval <= 0 {
		renewInterval = 5 * time.Second
	}
	if drainTimeout <= 0 {
		drainTimeout = 5 * time.Second
	}
	return &ConsumerService{
		consumerID:      consumerID,
		groupID:         groupID,
		partitionCount:  partitionCount,
		queueConsumer:   queueConsumer,
		queueLease:      queueLease,
		membership:      membership,
		watchManager:    watchManager,
		callbacks:       callbacks,
		pollInterval:    pollInterval,
		renewInterval:   renewInterval,
		rebalanceTicker: 2 * time.Second,
		drainTimeout:    drainTimeout,
		workers:         make(map[int]*partitionWorker),
	}
}

func (s *ConsumerService) Run(ctx context.Context) error {
	memberHandle, err := s.membership.Register(ctx, s.groupID, s.consumerID)
	if err != nil {
		return err
	}
	defer func() {
		_ = s.membership.Revoke(context.Background(), memberHandle)
		s.stopAllWorkers()
	}()

	go s.keepMembershipAlive(ctx, memberHandle)

	ticker := time.NewTicker(s.rebalanceTicker)
	defer ticker.Stop()

	for {
		if err := s.reconcile(ctx); err != nil {
			log.Error().Err(err).Msg("consumer reconcile failed")
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (s *ConsumerService) keepMembershipAlive(ctx context.Context, handle models.LeaseHandle) {
	ticker := time.NewTicker(s.renewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.membership.KeepAlive(ctx, handle); err != nil {
				log.Error().Err(err).Str("consumer_id", s.consumerID).Msg("consumer membership keepalive failed")
			}
		}
	}
}

func (s *ConsumerService) reconcile(ctx context.Context) error {
	members, err := s.membership.ListMembers(ctx, s.groupID)
	if err != nil {
		return err
	}
	assigned := computeRangeAssignment(s.partitionCount, members, s.consumerID)
	target := make(map[int]struct{}, len(assigned))
	for _, queueID := range assigned {
		target[queueID] = struct{}{}
	}

	for queueID, worker := range s.workers {
		if _, ok := target[queueID]; ok {
			continue
		}
		worker.cancel()
		select {
		case <-worker.done:
		case <-time.After(s.drainTimeout):
		}
		_ = s.queueLease.Release(context.Background(), worker.lease)
		delete(s.workers, queueID)
	}

	for _, queueID := range assigned {
		if _, ok := s.workers[queueID]; ok {
			continue
		}
		lease, ok, err := s.queueLease.Acquire(ctx, queueID, s.consumerID)
		if err != nil {
			log.Error().Err(err).Int("queue_id", queueID).Msg("failed to acquire queue lease")
			continue
		}
		if !ok {
			continue
		}
		workerCtx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		s.workers[queueID] = &partitionWorker{cancel: cancel, done: done, lease: lease}
		go s.runPartitionWorker(workerCtx, queueID, lease, done)
		log.Info().Int("queue_id", queueID).Str("consumer_id", s.consumerID).Msg("started partition worker")
	}
	return nil
}

func (s *ConsumerService) runPartitionWorker(ctx context.Context, queueID int, lease models.LeaseHandle, done chan struct{}) {
	defer close(done)
	renewTicker := time.NewTicker(s.renewInterval)
	defer renewTicker.Stop()
	pollTicker := time.NewTicker(s.pollInterval)
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-renewTicker.C:
			if err := s.queueLease.KeepAlive(ctx, lease); err != nil {
				log.Error().Err(err).Int("queue_id", queueID).Msg("queue lease keepalive failed, stopping worker")
				return
			}
		case <-pollTicker.C:
			intent, msgID, err := s.queueConsumer.ConsumeWatchIntent(ctx, queueID)
			if err != nil {
				log.Error().Err(err).Int("queue_id", queueID).Msg("failed consuming queue message")
				continue
			}
			if intent == nil {
				continue
			}
			watchErr := s.watchManager.Watch(ctx, *intent)
			callbackErr := s.callbacks.Dispatch(ctx, *intent, watchErr)
			if watchErr == nil && callbackErr == nil {
				_ = s.queueConsumer.Ack(ctx, queueID, msgID)
			} else {
				_ = s.queueConsumer.Nack(ctx, queueID, msgID)
			}
		}
	}
}

func (s *ConsumerService) stopAllWorkers() {
	for queueID, worker := range s.workers {
		worker.cancel()
		select {
		case <-worker.done:
		case <-time.After(s.drainTimeout):
		}
		_ = s.queueLease.Release(context.Background(), worker.lease)
		delete(s.workers, queueID)
	}
}
