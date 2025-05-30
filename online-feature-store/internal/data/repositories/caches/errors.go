package caches

import "errors"

var (
	// ErrNotImplemented is returned when a cache method is not implemented
	ErrNotImplemented = errors.New("method not implemented")

	// ErrInvalidInput is returned when input parameters are invalid
	ErrInvalidInput = errors.New("invalid input parameters")

	// ErrCacheOperation is returned when a cache operation fails
	ErrCacheOperation = errors.New("cache operation failed")
)
