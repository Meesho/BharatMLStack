package failsafecb

import (
	"context"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

var (
	// CbMap to be used for force closing and opening of CB upon any event listener
	CbMap = &sync.Map{}
)

type FailSafeCB[R, T any] struct {
	Cb circuitbreaker.CircuitBreaker[any]
}

func NewFailSafe[R, T any](config *CBConfig) *FailSafeCB[R, T] {
	cb := circuitbreaker.Builder[any]().
		WithFailureRateThreshold(uint(config.FailureRateThreshold), uint(config.FailureExecutionThreshold), time.Duration(config.FailureThresholdingPeriodInMS)*time.Millisecond).
		WithSuccessThresholdRatio(uint(config.SuccessRatioThreshold), uint(config.SuccessThresholdingCapacity)).
		WithDelay(time.Duration(config.WithDelayInMS) * time.Millisecond).
		OnStateChanged(func(event circuitbreaker.StateChangedEvent) {
			log.Debug().Msgf("Circuit Breaker '%s' changed state from %s to %s\n", config.CBName, event.OldState, event.NewState)
			metric.Incr("CB_STATE_CHANGED", []string{"name", config.CBName, "from", event.OldState.String(), "to", event.NewState.String()})
		}).
		Build()
	f := &FailSafeCB[R, T]{
		Cb: cb,
	}
	CbMap.Store(config.CBName, f.Cb)
	return f
}

func (f *FailSafeCB[R, T]) Execute(request R, task func(R) (T, error)) (T, error) {
	var result T
	var taskErr error
	err := failsafe.Run(func() error {
		// Inside Run, execute the task using the circuit breaker
		result, taskErr = task(request)
		if taskErr != nil {
			return taskErr // Return the error if the task failed
		}
		return nil // No error, task executed successfully
	}, f.Cb)

	// Handle errors from failsafe.Run
	if err != nil {
		return result, err
	}
	return result, nil
}

func (f *FailSafeCB[R, T]) ExecuteForGrpc(ctx context.Context, request R, task func(context.Context, R, ...grpc.CallOption) (T, error)) (T, error) {
	var result T
	var taskErr error
	err := failsafe.Run(func() error {
		// Inside Run, execute the task using the circuit breaker
		result, taskErr = task(ctx, request)
		if taskErr != nil {
			return taskErr // Return the error if the task failed
		}
		return nil // No error, task executed successfully
	}, f.Cb)

	// Handle errors from failsafe.Run
	if err != nil {
		return result, err
	}
	return result, nil
}
