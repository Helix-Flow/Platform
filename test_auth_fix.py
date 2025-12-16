#!/usr/bin/env python3

import requests
import json
import sys

def test_auth_connectivity():
    """Test basic auth service connectivity"""
    
    print("Testing Auth Service Connectivity...")
    
    # Test health endpoint
    try:
        response = requests.get("http://localhost:8082/health", timeout=5)
        print(f"✅ Health endpoint: HTTP {response.status_code}")
        if response.status_code == 200:
            print(f"   Response: {response.text[:200]}")
    except Exception as e:
        print(f"❌ Health endpoint failed: {e}")
        return False
    
    # Test login endpoint
    try:
        login_data = {
            "email": "test@example.com",
            "password": "password"
        }
        response = requests.post(
            "http://localhost:8082/login",
            json=login_data,
            timeout=5
        )
        print(f"✅ Login endpoint: HTTP {response.status_code}")
        if response.status_code == 200:
            token_data = response.json()
            print(f"   Token received: {token_data.get('access_token', 'No token')[:50]}...")
        else:
            print(f"   Response: {response.text[:200]}")
    except Exception as e:
        print(f"❌ Login endpoint failed: {e}")
        return False
    
    return True

if __name__ == "__main__":
    if test_auth_connectivity():
        print("\n✅ Auth service connectivity test passed!")
        sys.exit(0)
    else:
        print("\n❌ Auth service connectivity test failed!")
        sys.exit(1)