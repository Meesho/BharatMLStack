package errors

import "errors"

var (
	ErrNotFound            = errors.New("resource not found")
	ErrAlreadyProcured     = errors.New("resource already procured")
	ErrCASConflict         = errors.New("compare-and-swap conflict")
	ErrInvalidOwner        = errors.New("invalid owner")
	ErrInvalidAction       = errors.New("invalid action")
	ErrInvalidRequest      = errors.New("invalid request")
	ErrUnsupportedEnv      = errors.New("unsupported environment")
	ErrIdempotencyMismatch = errors.New("idempotency key reused with different request")
)
