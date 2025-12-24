package middleware

import (
	"context"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"runtime/debug"
)

// GRPCRecovery handles context errors/panics and sets response code accordingly
func GRPCRecovery(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			// Recover from panic and create a gRPC error
			log.Error().Msgf("Panic occurred in method %s: %v\n%s", info.FullMethod, r, debug.Stack())
			err = status.Errorf(codes.Internal, "panic recovered: %v", r)
		}
	}()
	resp, err = handler(ctx, req)

	return resp, err
}
