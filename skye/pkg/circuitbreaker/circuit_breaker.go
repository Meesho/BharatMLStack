package circuitbreaker

import (
	"context"
	"google.golang.org/grpc"
)

type CircuitBreaker[Request any, Response any] interface {
	Execute(request Request, task func(Request) (Response, error)) (Response, error)
	ExecuteForGrpc(ctx context.Context, request Request, task func(context.Context, Request, ...grpc.CallOption) (Response, error)) (Response, error)
}

type ManualCircuitBreaker interface {
	IsAllowed() bool
	RecordSuccess()
	RecordFailure()
}
