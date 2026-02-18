package api

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewGrpcInvalidArgumentError(message string) error {
	return status.New(codes.InvalidArgument, message).Err()
}

func NewGrpcNotFoundError(message string) error {
	return status.New(codes.NotFound, message).Err()
}

func NewGrpcInternalServerError(message string) error {
	return status.New(codes.Internal, message).Err()
}

func NewGrpcUnauthenticatedError(message string) error {
	return status.New(codes.Unauthenticated, message).Err()
}

func NewGrpcPermissionDeniedError(message string) error {
	return status.New(codes.PermissionDenied, message).Err()
}

func NewGrpcResourceExhaustedError(message string) error {
	return status.New(codes.ResourceExhausted, message).Err()
}

func NewGrpcAlreadyExistsError(message string) error {
	return status.New(codes.AlreadyExists, message).Err()
}

func NewGrpcUnavailableError(message string) error {
	return status.New(codes.Unavailable, message).Err()
}

func NewGrpcDeadlineExceededError(message string) error {
	return status.New(codes.DeadlineExceeded, message).Err()
}

func NewGrpcCancelledError(message string) error {
	return status.New(codes.Canceled, message).Err()
}

func NewGrpcUnimplementedError(message string) error {
	return status.New(codes.Unimplemented, message).Err()
}
