"""
Template for creating new SDKs in BharatML Stack

This file serves as a template and example for creating new SDKs.
Copy this structure when creating new SDKs like:
- fastapi_feature_client
- grpc_feature_client  
- model_serving_client
- etc.

Example SDK Structure:
    src/new_sdk_name/
    ├── __init__.py           # Main exports
    ├── client.py             # Main client class
    ├── config.py             # Configuration classes
    ├── exceptions.py         # Custom exceptions
    ├── utils/                # SDK-specific utilities
    │   ├── __init__.py
    │   └── helpers.py
    └── proto/                # Protobuf files (if needed)
        └── ...
"""

from bharatml_commons.http_client import BharatMLHTTPClient
from bharatml_commons.column_utils import clean_column_name
from bharatml_commons.feature_utils import get_fgs_to_feature_mappings


class BaseSDKClient:
    """
    Base class for BharatML Stack SDK clients
    
    Provides common functionality that all SDKs can inherit from.
    """
    
    def __init__(self, job_id: str, job_token: str, base_url: str = None):
        self.job_id = job_id
        self.job_token = job_token
        self.http_client = BharatMLHTTPClient(base_url=base_url)
    
    def authenticate(self) -> bool:
        """Override in subclasses to implement authentication logic"""
        return True
    
    def get_version(self) -> str:
        """Get SDK version"""
        from bharatml_commons import __version__
        return __version__


# Example of how a new SDK might be structured:
class ExampleNewSDKClient(BaseSDKClient):
    """
    Example new SDK client showing how to use shared utilities
    
    This is just a template - replace with actual SDK implementation
    """
    
    def __init__(self, job_id: str, job_token: str, metadata_url: str):
        super().__init__(job_id, job_token)
        self.metadata_url = metadata_url
    
    def get_features_metadata(self):
        """Example method using shared HTTP client"""
        return self.http_client.post(
            url=self.metadata_url,
            job_id=self.job_id,
            job_token=self.job_token
        )
    
    def process_column_names(self, column_names: list) -> list:
        """Example method using shared column utilities"""
        return [clean_column_name(col) for col in column_names] 