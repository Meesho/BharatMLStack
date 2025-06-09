"""
HTTP Client utilities for BharatML Stack SDKs
"""

import requests
from typing import Dict, Any, Optional


class BharatMLHTTPClient:
    """Base HTTP client for BharatML Stack API interactions"""
    
    def __init__(self, base_url: str = None, default_headers: Dict[str, str] = None):
        self.base_url = base_url
        self.default_headers = default_headers or {"Content-Type": "application/json"}
    
    def post(self, url: str, job_id: str, job_token: str, 
             additional_params: Dict[str, Any] = None,
             additional_headers: Dict[str, str] = None) -> Dict[str, Any]:
        """
        Make a POST request with job authentication
        
        Args:
            url: The endpoint URL
            job_id: Job identifier for authentication
            job_token: Job token for authentication
            additional_params: Additional parameters to include
            additional_headers: Additional headers to include
            
        Returns:
            JSON response as dictionary
            
        Raises:
            AssertionError: If request fails or URL is invalid
        """
        assert url.startswith("http") or url.startswith("https"), \
            "URL must start with http or https"
        
        params = {"jobId": job_id, "jobToken": job_token}
        if additional_params:
            params.update(additional_params)
            
        headers = self.default_headers.copy()
        if additional_headers:
            headers.update(additional_headers)
        
        response = requests.post(url, headers=headers, params=params)
        
        assert response.status_code == 200, \
            f"Request failed with status code {response.status_code}: {response.json()} " \
            f"while calling {url} for job_id: {job_id} and job_token: {job_token}"
        
        return response.json()


def get_metadata_host_response(features_metadata_url: str, job_id: str, job_token: str) -> Dict[str, Any]:
    """
    Get metadata response from host - backward compatibility function
    
    Args:
        features_metadata_url: URL for features metadata endpoint
        job_id: Job identifier
        job_token: Job authentication token
        
    Returns:
        JSON response as dictionary
    """
    client = BharatMLHTTPClient()
    return client.post(features_metadata_url, job_id, job_token) 