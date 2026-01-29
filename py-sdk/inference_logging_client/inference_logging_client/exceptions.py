"""Custom exceptions for inference-logging-client."""


class InferenceLoggingError(Exception):
    """Base exception for inference-logging-client errors."""
    pass


class SchemaFetchError(InferenceLoggingError):
    """Raised when fetching schema from inference service fails."""
    pass


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
