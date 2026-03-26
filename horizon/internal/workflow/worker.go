package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Worker manages async workflow execution
type Worker struct {
	orchestrator *Orchestrator
	workers      int
	jobQueue     chan string // workflow IDs
	stopChan     chan struct{}
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewWorker creates a new workflow worker
func NewWorker(orchestrator *Orchestrator, numWorkers int) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		orchestrator: orchestrator,
		workers:      numWorkers,
		jobQueue:     make(chan string, 100), // Buffered channel
		stopChan:     make(chan struct{}),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start starts the worker pool
func (w *Worker) Start() {
	log.Info().Int("workers", w.workers).Msg("Starting workflow worker pool")

	for i := 0; i < w.workers; i++ {
		w.wg.Add(1)
		go w.workerLoop(i)
	}
}

// Stop stops the worker pool gracefully
func (w *Worker) Stop() {
	log.Info().Msg("Stopping workflow worker pool")
	close(w.stopChan)
	w.cancel()
	w.wg.Wait()
	log.Info().Msg("Workflow worker pool stopped")
}

// Enqueue adds a workflow ID to the job queue
func (w *Worker) Enqueue(workflowID string) error {
	log.Info().
		Str("workflowID", workflowID).
		Int("queueCapacity", cap(w.jobQueue)).
		Int("queueLength", len(w.jobQueue)).
		Msg("Worker: Attempting to enqueue workflow")

	select {
	case w.jobQueue <- workflowID:
		log.Info().
			Str("workflowID", workflowID).
			Int("queueLength", len(w.jobQueue)).
			Int("queueCapacity", cap(w.jobQueue)).
			Msg("Worker: Workflow enqueued successfully for processing")
		return nil
	case <-time.After(5 * time.Second):
		err := fmt.Errorf("failed to enqueue workflow: queue full")
		log.Error().
			Err(err).
			Str("workflowID", workflowID).
			Int("queueCapacity", cap(w.jobQueue)).
			Int("queueLength", len(w.jobQueue)).
			Dur("timeout", 5*time.Second).
			Msg("Worker: Failed to enqueue workflow - job queue is full, workers may be overloaded or stuck")
		return err
	}
}

// workerLoop processes workflows from the queue
func (w *Worker) workerLoop(workerID int) {
	defer w.wg.Done()

	log.Info().Int("workerID", workerID).Msg("Worker started")

	for {
		select {
		case <-w.stopChan:
			log.Info().Int("workerID", workerID).Msg("Worker stopping")
			return
		case workflowID := <-w.jobQueue:
			startTime := time.Now()
			log.Info().
				Int("workerID", workerID).
				Str("workflowID", workflowID).
				Int("remainingQueueLength", len(w.jobQueue)).
				Msg("Worker: Processing workflow from queue - starting execution")

			// Execute workflow
			if err := w.orchestrator.ExecuteWorkflow(w.ctx, workflowID); err != nil {
				duration := time.Since(startTime)
				log.Error().
					Err(err).
					Int("workerID", workerID).
					Str("workflowID", workflowID).
					Dur("executionDuration", duration).
					Msg("Worker: Workflow execution failed - check orchestrator logs for detailed error information including activity failures, etcd errors, and GitHub operations")
			} else {
				duration := time.Since(startTime)
				log.Info().
					Int("workerID", workerID).
					Str("workflowID", workflowID).
					Dur("executionDuration", duration).
					Msg("Worker: Workflow execution completed successfully - all activities finished")
			}
		case <-w.ctx.Done():
			log.Info().Int("workerID", workerID).Msg("Worker context cancelled")
			return
		}
	}
}
