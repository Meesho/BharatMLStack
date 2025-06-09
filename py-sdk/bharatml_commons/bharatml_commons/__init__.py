"""
BharatML Stack Common Utilities

Shared utilities and components for BharatML Stack SDKs.
"""

from .feature_metadata_client import FeatureMetadataClient, create_feature_metadata_client
from .http_client import BharatMLHTTPClient, get_metadata_host_response
from .column_utils import clean_column_name, generate_renamed_column
from .feature_utils import get_fgs_to_feature_mappings, extract_entity_info
from .sdk_template import BaseSDKClient, ExampleNewSDKClient

__version__ = "0.1.0"

__all__ = [
    # Feature Metadata Client
    "FeatureMetadataClient",
    "create_feature_metadata_client", 
    
    # HTTP Utilities
    "BharatMLHTTPClient",
    "get_metadata_host_response",
    
    # Data Processing Utilities
    "clean_column_name",
    "generate_renamed_column",
    "get_fgs_to_feature_mappings",
    "extract_entity_info",
    
    # Base Classes
    "BaseSDKClient",
    "ExampleNewSDKClient",
] 