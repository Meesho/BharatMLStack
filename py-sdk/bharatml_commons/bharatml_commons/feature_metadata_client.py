"""
Feature Metadata Client - REST API based metadata operations

This module provides HTTP-based feature metadata operations that can be used by any SDK
or as a standalone client for REST API interactions with BharatML Stack services.
"""

from typing import Dict, List, Any, Optional
from .http_client import BharatMLHTTPClient
from .sdk_template import BaseSDKClient
from .column_utils import clean_column_name
from .feature_utils import extract_entity_info


class FeatureMetadataClient(BaseSDKClient):
    """
    HTTP-based client for feature metadata operations via REST API
    
    This client provides a REST API interface for feature metadata operations,
    serving as a shared utility for other SDKs or as a standalone client.
    
    Example Usage:
        client = FeatureMetadataClient("https://api.example.com", "job123", "token456")
        metadata = client.get_feature_metadata(["user_features"])
        features = client.get_features({"user_id": "123"}, ["user_features"])
    """
    
    def __init__(self, base_url: str, job_id: str, job_token: str):
        """
        Initialize Feature Metadata Client
        
        Args:
            base_url: Base URL for the feature service API
            job_id: Job identifier for authentication
            job_token: Job authentication token
        """
        super().__init__(job_id, job_token, base_url)
        self.base_url = base_url.rstrip('/')  # Remove trailing slash
    
    def get_feature_metadata(self, feature_group_ids: Optional[List[str]] = None) -> Dict[str, Any]:
        """
        Retrieve feature metadata via HTTP API
        
        Args:
            feature_group_ids: Optional list of feature group IDs to filter
            
        Returns:
            Feature metadata response
            
        Raises:
            Exception: If HTTP request fails or returns error status
        """
        endpoint = f"{self.base_url}/api/v1/features/metadata"
        
        additional_params = {}
        if feature_group_ids:
            additional_params["feature_groups"] = ",".join(feature_group_ids)
        
        return self.http_client.post(
            url=endpoint,
            job_id=self.job_id,
            job_token=self.job_token,
            additional_params=additional_params
        )
    
    def get_features(self, entity_keys: Dict[str, Any], 
                    feature_groups: Optional[List[str]] = None) -> Dict[str, Any]:
        """
        Get feature values for given entity keys
        
        Args:
            entity_keys: Dictionary of entity key-value pairs
            feature_groups: Optional list of feature groups to retrieve
            
        Returns:
            Feature values response
            
        Example:
            features = client.get_features(
                entity_keys={"user_id": "123", "item_id": "456"},
                feature_groups=["user_features", "item_features"]
            )
        """
        endpoint = f"{self.base_url}/api/v1/features/get"
        
        request_payload = {
            "entity_keys": entity_keys
        }
        
        if feature_groups:
            request_payload["feature_groups"] = feature_groups
        
        return self.http_client.post(
            url=endpoint,
            job_id=self.job_id,
            job_token=self.job_token,
            additional_params=request_payload
        )
    
    def persist_features(self, entity_label: str, entity_keys: Dict[str, Any], 
                        features: Dict[str, Any]) -> Dict[str, Any]:
        """
        Persist feature values via HTTP API
        
        Args:
            entity_label: Label for the entity type
            entity_keys: Dictionary of entity key-value pairs
            features: Dictionary of feature name-value pairs
            
        Returns:
            Persist operation response
        """
        endpoint = f"{self.base_url}/api/v1/features/persist"
        
        request_payload = {
            "entity_label": entity_label,
            "entity_keys": entity_keys,
            "features": features
        }
        
        return self.http_client.post(
            url=endpoint,
            job_id=self.job_id,
            job_token=self.job_token,
            additional_params=request_payload
        )
    
    def validate_feature_names(self, feature_names: List[str]) -> List[str]:
        """
        Validate and clean feature names using shared utilities
        
        Args:
            feature_names: List of feature names to validate
            
        Returns:
            List of cleaned feature names
        """
        return [clean_column_name(name) for name in feature_names]
    
    def process_metadata_response(self, metadata_response: Dict[str, Any]) -> Dict[str, Any]:
        """
        Process metadata response using shared utilities
        
        Args:
            metadata_response: Raw metadata response from API
            
        Returns:
            Processed metadata with extracted entity information
        """
        if "keys" in metadata_response:
            entity_label, entity_columns = extract_entity_info(metadata_response)
            
            return {
                "entity_label": entity_label,
                "entity_columns": entity_columns,
                "raw_response": metadata_response
            }
        
        return metadata_response
    
    def health_check(self) -> Dict[str, Any]:
        """
        Check if the feature service is healthy
        
        Returns:
            Health status response with 'status' field ('healthy' or 'unhealthy')
        """
        endpoint = f"{self.base_url}/health"
        
        try:
            response = self.http_client.post(
                url=endpoint,
                job_id=self.job_id,
                job_token=self.job_token
            )
            return {"status": "healthy", "response": response}
        except Exception as e:
            return {"status": "unhealthy", "error": str(e)}
    
    def list_feature_groups(self) -> Dict[str, Any]:
        """
        List all available feature groups
        
        Returns:
            Response containing list of feature groups
        """
        endpoint = f"{self.base_url}/api/v1/features/groups"
        
        return self.http_client.post(
            url=endpoint,
            job_id=self.job_id,
            job_token=self.job_token
        )
    
    def get_feature_group_schema(self, feature_group_id: str) -> Dict[str, Any]:
        """
        Get schema for a specific feature group
        
        Args:
            feature_group_id: ID of the feature group
            
        Returns:
            Feature group schema response
        """
        endpoint = f"{self.base_url}/api/v1/features/groups/{feature_group_id}/schema"
        
        return self.http_client.post(
            url=endpoint,
            job_id=self.job_id,
            job_token=self.job_token
        )


# Convenience function for quick metadata client creation
def create_feature_metadata_client(base_url: str, job_id: str, job_token: str) -> FeatureMetadataClient:
    """
    Factory function to create a Feature Metadata Client
    
    Args:
        base_url: Base URL for the feature service API
        job_id: Job identifier for authentication
        job_token: Job authentication token
        
    Returns:
        Configured FeatureMetadataClient instance
    """
    return FeatureMetadataClient(base_url, job_id, job_token) 