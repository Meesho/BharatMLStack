package failsafecb

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"testing"
)

func TestNewFailSafeCB_Initialization(t *testing.T) {
	config := &CBConfig{
		CBName:                        "testCB",
		FailureRateThreshold:          50,
		FailureExecutionThreshold:     20,
		FailureThresholdingPeriodInMS: 10000,
		SuccessRatioThreshold:         75,
		SuccessThresholdingCapacity:   10,
		WithDelayInMS:                 1000,
	}

	cb := NewFailSafe[int, int](config)
	assert.NotNil(t, cb, "FailSafeCB should not be nil")
}

func TestFailSafeCB_Execute_Success(t *testing.T) {
	config := &CBConfig{
		CBName:                        "testCB",
		FailureRateThreshold:          50,
		FailureExecutionThreshold:     20,
		FailureThresholdingPeriodInMS: 10000,
		SuccessRatioThreshold:         75,
		SuccessThresholdingCapacity:   10,
		WithDelayInMS:                 1000,
	}

	cb := NewFailSafe[int, int](config)
	result, err := cb.Execute(5, func(i int) (int, error) {
		return 10, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 10, result)
}

func TestFailSafeCB_Execute_Failure(t *testing.T) {
	config := &CBConfig{
		CBName:                        "testCB",
		FailureRateThreshold:          50,
		FailureExecutionThreshold:     20,
		FailureThresholdingPeriodInMS: 10000,
		SuccessRatioThreshold:         75,
		SuccessThresholdingCapacity:   10,
		WithDelayInMS:                 1000,
	}

	cb := NewFailSafe[int, int](config)
	result, err := cb.Execute(5, func(i int) (int, error) {
		return 0, errors.New("task failed")
	})

	assert.Error(t, err)
	assert.Equal(t, 0, result)
}

func TestFailSafeCB_ExecuteForGrpc_Success(t *testing.T) {
	config := &CBConfig{
		CBName:                        "testCB",
		FailureRateThreshold:          50,
		FailureExecutionThreshold:     20,
		FailureThresholdingPeriodInMS: 10000,
		SuccessRatioThreshold:         75,
		SuccessThresholdingCapacity:   10,
		WithDelayInMS:                 1000,
	}

	cb := NewFailSafe[int, int](config)
	ctx := context.Background()
	result, err := cb.ExecuteForGrpc(ctx, 5, func(ctx context.Context, i int, opts ...grpc.CallOption) (int, error) {
		return 10, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 10, result)
}

func TestFailSafeCB_ExecuteForGrpc_Failure(t *testing.T) {
	config := &CBConfig{
		CBName:                        "testCB",
		FailureRateThreshold:          50,
		FailureExecutionThreshold:     20,
		FailureThresholdingPeriodInMS: 10000,
		SuccessRatioThreshold:         75,
		SuccessThresholdingCapacity:   10,
		WithDelayInMS:                 1000,
	}

	cb := NewFailSafe[int, int](config)
	ctx := context.Background()
	result, err := cb.ExecuteForGrpc(ctx, 5, func(ctx context.Context, i int, opts ...grpc.CallOption) (int, error) {
		return 0, errors.New("grpc task failed")
	})

	assert.Error(t, err)
	assert.Equal(t, 0, result)
}
