"""Custom exceptions for inference-logging-client."""

from typing import Optional


class InferenceLoggingError(Exception):
    """Base exception for inference-logging-client errors."""

    pass


class SchemaFetchError(InferenceLoggingError):
    """Raised when fetching schema from inference service fails.

    The optional `status_code` attribute carries the HTTP status when the
    failure was an HTTP response (None for network/URL errors).
    """

    def __init__(self, message: str, status_code: Optional[int] = None):
        super().__init__(message)
        self.status_code = status_code


class SchemaNotFoundError(InferenceLoggingError):
    """Raised when no features are found in schema response."""

    pass


class DecodeError(InferenceLoggingError):
    """Raised when decoding feature data fails."""

    pass


class FormatError(InferenceLoggingError):
    """Raised when there's an issue with the data format."""

    pass


class ProtobufError(InferenceLoggingError):
    """Raised when parsing protobuf data fails."""

    pass
