"""
Configuration classes for gRPC Feature Client
"""

from dataclasses import dataclass
from typing import Optional, Dict, Any


@dataclass
class GRPCClientConfig:
    """
    Configuration for gRPC Feature Client
    
    Args:
        server_address: gRPC server address (e.g., 'localhost:50051')
        job_id: Job identifier for authentication
        job_token: Job authentication token
        use_tls: Whether to use TLS/SSL for secure connection
        timeout_seconds: Request timeout in seconds
        max_receive_message_length: Maximum message size for receiving
        max_send_message_length: Maximum message size for sending
        compression: Compression algorithm ('gzip' or None)
        metadata: Additional metadata to send with requests
    """
    server_address: str
    job_id: str
    job_token: str
    use_tls: bool = False
    timeout_seconds: float = 30.0
    max_receive_message_length: int = 4 * 1024 * 1024  # 4MB
    max_send_message_length: int = 4 * 1024 * 1024     # 4MB
    compression: Optional[str] = None
    metadata: Optional[Dict[str, str]] = None
    
    def __post_init__(self):
        """Validate configuration after initialization"""
        if not self.server_address:
            raise ValueError("server_address cannot be empty")
        if not self.job_id:
            raise ValueError("job_id cannot be empty")
        if not self.job_token:
            raise ValueError("job_token cannot be empty")
        if self.timeout_seconds <= 0:
            raise ValueError("timeout_seconds must be positive")
        
        # Initialize metadata if not provided
        if self.metadata is None:
            self.metadata = {}
        
        # Add authentication metadata
        self.metadata.update({
            'job-id': self.job_id,
            'job-token': self.job_token
        })
    
    def get_channel_options(self) -> list:
        """
        Get gRPC channel options based on configuration
        
        Returns:
            List of gRPC channel options
        """
        options = [
            ('grpc.max_receive_message_length', self.max_receive_message_length),
            ('grpc.max_send_message_length', self.max_send_message_length),
        ]
        
        return options 