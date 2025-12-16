#!/usr/bin/env python3
"""Test service health endpoints for security testing."""

import requests
import json

def test_services():
    urls = {
        "API Gateway": "https://localhost:8443/health",
        "Auth Service": "http://localhost:8082/health",
        "Monitoring Service": "http://localhost:8083/health"
    }
    
    for service, url in urls.items():
        try:
            response = requests.get(url, verify=False)
            print(f"✅ {service}: {response.status_code}")
            if response.status_code == 200:
                print(f"   Response: {response.text[:100]}")
        except Exception as e:
            print(f"❌ {service}: {e}")

if __name__ == "__main__":
    test_services()