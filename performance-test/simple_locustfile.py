#!/usr/bin/env python3
"""
Simple Locust file for testing ONE service at a time
Usage: 
  locust -f simple_locustfile.py --host http://localhost:8082  # Test Java
  locust -f simple_locustfile.py --host http://localhost:8080  # Test Rust  
  locust -f simple_locustfile.py --host http://localhost:8081  # Test Go
"""

import time
import json
from locust import HttpUser, task, between

class SimpleFeatureStoreUser(HttpUser):
    wait_time = between(0.1, 0.5)  # Wait 0.1-0.5 seconds between requests
    
    @task
    def test_retrieve_features(self):
        """Test the /retrieve-features endpoint"""
        with self.client.post(
            "/retrieve-features",
            headers={"Content-Type": "application/json"},
            catch_response=True,
            name="retrieve_features"
        ) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    if data.get("success"):
                        response.success()
                    else:
                        response.failure(f"API returned success=false: {data.get('message', 'Unknown error')}")
                except json.JSONDecodeError:
                    response.failure("Invalid JSON response")
            else:
                response.failure(f"HTTP {response.status_code}: {response.text}")