package circuitbreaker

import (
	"context"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

var (
	_ CircuitBreaker[any, any] = (*MockCircuitBreaker[any, any])(nil)
	_ ManualCircuitBreaker     = (*MockManualCircuitBreaker)(nil)
)

// MockCircuitBreaker is a mock implementation of the CircuitBreaker interface.
type MockCircuitBreaker[Request any, Response any] struct {
	mock.Mock
}

func (m *MockCircuitBreaker[Request, Response]) Execute(request Request, task func(Request) (Response, error)) (Response, error) {
	args := m.Called(request, task)
	var res Response
	if args.Get(0) == nil {
		return res, args.Error(1)
	} else if val, ok := args.Get(0).(Response); ok {
		res = val
	}
	return res, args.Error(1)
}

func (m *MockCircuitBreaker[Request, Response]) ExecuteForGrpc(ctx context.Context, request Request, task func(context.Context, Request, ...grpc.CallOption) (Response, error)) (Response, error) {
	args := m.Called(ctx, request, task)
	var res Response
	if args.Get(0) == nil {
		return res, args.Error(1)
	} else if val, ok := args.Get(0).(Response); ok {
		res = val
	}
	return res, args.Error(1)
}

// MockManualCircuitBreaker is a mock implementation of the ManualCircuitBreaker interface.
type MockManualCircuitBreaker struct {
	mock.Mock
}

func (m *MockManualCircuitBreaker) IsAllowed() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockManualCircuitBreaker) RecordSuccess() {
	m.Called()
}

func (m *MockManualCircuitBreaker) RecordFailure() {
	m.Called()
}
