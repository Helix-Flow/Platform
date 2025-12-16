#!/usr/bin/env python3
"""
Direct test of auth service login endpoint
"""
import requests
import json

def test_auth_direct():
    """Test auth service directly"""
    print("Testing auth service directly...")
    
    # Test health endpoint
    try:
        response = requests.get("http://localhost:8082/health", timeout=5)
        print(f"Health check: {response.status_code} - {response.text}")
    except Exception as e:
        print(f"Health check failed: {e}")
        return False
    
    # Test login with various credentials
    login_attempts = [
        {"email": "admin", "password": "password"},
        {"email": "admin", "password": "admin"},
        {"email": "admin", "password": "admin123"},
        {"email": "admin@helixflow.com", "password": "password"},
        {"email": "demo", "password": "password"},
        {"email": "demo", "password": "demo"},
        {"email": "demo@helixflow.com", "password": "password"},
        {"email": "testuser", "password": "password"},
        {"email": "testuser", "password": "test"},
        {"email": "test@example.com", "password": "password"},
    ]
    
    for attempt in login_attempts:
        print(f"\nTrying {attempt['email']} with password '{attempt['password']}'")
        try:
            response = requests.post(
                "http://localhost:8082/login", 
                json=attempt, 
                timeout=5
            )
            print(f"  Status: {response.status_code}")
            if response.status_code == 200:
                print(f"  ✅ SUCCESS: {response.text}")
                return True
            elif response.status_code == 401:
                print(f"  ❌ Authentication failed")
            else:
                print(f"  ⚠️  Unexpected response: {response.text}")
        except Exception as e:
            print(f"  ❌ Error: {e}")
    
    return False

if __name__ == "__main__":
    if test_auth_direct():
        print("\n🎉 Authentication working!")
    else:
        print("\n❌ Authentication failed for all attempts")