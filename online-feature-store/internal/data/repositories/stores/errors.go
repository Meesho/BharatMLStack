package stores

import "errors"

var (
	// ErrNotImplemented is returned when a store method is not implemented
	ErrNotImplemented = errors.New("method not implemented")

	// ErrInvalidInput is returned when input parameters are invalid
	ErrInvalidInput = errors.New("invalid input parameters")

	// ErrStoreOperation is returned when a store operation fails
	ErrStoreOperation = errors.New("store operation failed")
)
