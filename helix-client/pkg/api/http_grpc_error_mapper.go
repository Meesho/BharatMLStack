package api

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

// ConvertHttpErrorToGrpc maps an HTTP error to a corresponding gRPC error.
func ConvertHttpErrorToGrpc(httpError *Error) error {
	switch httpError.StatusCode {
	case http.StatusBadRequest:
		return status.New(codes.InvalidArgument, httpError.Message).Err()
	case http.StatusUnauthorized:
		return status.New(codes.Unauthenticated, httpError.Message).Err()
	case http.StatusForbidden:
		return status.New(codes.PermissionDenied, httpError.Message).Err()
	case http.StatusNotFound:
		return status.New(codes.NotFound, httpError.Message).Err()
	case http.StatusConflict:
		return status.New(codes.AlreadyExists, httpError.Message).Err()
	case http.StatusTooManyRequests:
		return status.New(codes.ResourceExhausted, httpError.Message).Err()
	case http.StatusInternalServerError:
		return status.New(codes.Internal, httpError.Message).Err()
	case http.StatusNotImplemented:
		return status.New(codes.Unimplemented, httpError.Message).Err()
	case http.StatusServiceUnavailable:
		return status.New(codes.Unavailable, httpError.Message).Err()
	case http.StatusGatewayTimeout:
		return status.New(codes.DeadlineExceeded, httpError.Message).Err()
	default:
		return status.New(codes.Unknown, "Unknown error").Err()
	}
}
