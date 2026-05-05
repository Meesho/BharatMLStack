package api

import "net/http"

// Error represents an error that occurred while handling a request.
type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	return e.Message
}

// Common errors
var (
	BadRequestError     = &Error{StatusCode: http.StatusBadRequest}
	NotFoundError       = &Error{StatusCode: http.StatusNotFound}
	InternalServerError = &Error{StatusCode: http.StatusInternalServerError}
	UnauthorizedError   = &Error{StatusCode: http.StatusUnauthorized}
	NoContentError      = &Error{StatusCode: http.StatusNoContent}
	ForbiddenError      = &Error{StatusCode: http.StatusForbidden}
	TooManyRequests     = &Error{StatusCode: http.StatusTooManyRequests}
	BadGatewayError     = &Error{StatusCode: http.StatusBadGateway}
	ServiceUnavailable  = &Error{StatusCode: http.StatusServiceUnavailable}
	GatewayTimeout      = &Error{StatusCode: http.StatusGatewayTimeout}
)

func NewBadRequestError(message string) *Error {
	return &Error{StatusCode: http.StatusBadRequest, Message: message}
}

func NewNotFoundError(message string) *Error {
	return &Error{StatusCode: http.StatusNotFound, Message: message}
}

func NewInternalServerError(message string) *Error {
	return &Error{StatusCode: http.StatusInternalServerError, Message: message}
}

func NewUnauthorizedError(message string) *Error {
	return &Error{StatusCode: http.StatusUnauthorized, Message: message}
}

func NewNoContentError(message string) *Error {
	return &Error{StatusCode: http.StatusNoContent, Message: message}
}

func NewForbiddenError(message string) *Error {
	return &Error{StatusCode: http.StatusForbidden, Message: message}
}

func NewTooManyRequests(message string) *Error {
	return &Error{StatusCode: http.StatusTooManyRequests, Message: message}
}

func NewBadGatewayError(message string) *Error {
	return &Error{StatusCode: http.StatusBadGateway, Message: message}
}

func NewServiceUnavailable(message string) *Error {
	return &Error{StatusCode: http.StatusServiceUnavailable, Message: message}
}

func NewGatewayTimeout(message string) *Error {
	return &Error{StatusCode: http.StatusGatewayTimeout, Message: message}
}
