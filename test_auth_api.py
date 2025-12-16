#!/usr/bin/env python3
"""
Complete API test for HelixFlow Platform
Tests authentication and inference endpoints
"""
import requests
import json
import sys
from urllib3.exceptions import InsecureRequestWarning

# Suppress TLS warnings for self-signed certificates
requests.packages.urllib3.disable_warnings(category=InsecureRequestWarning)

def get_auth_token():
    """Get authentication token from auth service"""
    print("Getting authentication token...")
    
    try:
        # First try to register a test user
        register_data = {
            "username": "testuser", 
            "email": "test@example.com",
            "password": "testpass123"
        }
        
        register_response = requests.post(
            "http://localhost:8082/register", 
            json=register_data, 
            timeout=5
        )
        
        if register_response.status_code == 201:
            print("  ✅ Test user registered successfully")
        elif register_response.status_code == 409:
            print("  ℹ️  Test user already exists")
        else:
            print(f"  ⚠️  Registration: {register_response.status_code}")
        
        # Login to get token
        login_data = {"username": "testuser", "password": "testpass123"}
        login_response = requests.post(
            "http://localhost:8082/login", 
            json=login_data, 
            timeout=5
        )
        
        if login_response.status_code == 200:
            token_data = login_response.json()
            token = token_data.get("token")
            print("  ✅ Authentication token obtained")
            return token
        else:
            print(f"  ❌ Login failed: {login_response.status_code} - {login_response.text}")
            return None
            
    except Exception as e:
        print(f"  ❌ Authentication error: {e}")
        return None

def test_authenticated_api(token):
    """Test API endpoints with authentication"""
    print("\nTesting authenticated API endpoints...")
    
    headers = {"Authorization": f"Bearer {token}"}
    base_url = "https://localhost:8443"
    
    try:
        # Test chat completion with authentication
        print("  Testing authenticated chat completion...")
        chat_data = {
            "model": "gpt-3.5-turbo",
            "messages": [{"role": "user", "content": "Hello! How are you?"}],
            "max_tokens": 100
        }
        
        response = requests.post(
            f"{base_url}/v1/chat/completions", 
            json=chat_data, 
            headers=headers,
            verify=False, 
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            message = result.get('choices', [{}])[0].get('message', {}).get('content', 'No content')
            print(f"  ✅ Authenticated Chat: {response.status_code}")
            print(f"  🤖 Response: {message}")
            return True
        else:
            print(f"  ❌ Authenticated Chat: {response.status_code} - {response.text}")
            return False
            
    except Exception as e:
        print(f"  ❌ Authenticated API error: {e}")
        return False

def test_token_revocation(token):
    """Test token revocation functionality"""
    print("\nTesting token revocation...")
    
    try:
        # Revoke the token
        headers = {"Authorization": f"Bearer {token}"}
        response = requests.post(
            "http://localhost:8082/revoke", 
            headers=headers,
            timeout=5
        )
        
        if response.status_code == 200:
            print("  ✅ Token revoked successfully")
            
            # Try to use the revoked token
            chat_data = {
                "model": "gpt-3.5-turbo",
                "messages": [{"role": "user", "content": "This should fail"}],
                "max_tokens": 10
            }
            
            response = requests.post(
                "https://localhost:8443/v1/chat/completions", 
                json=chat_data, 
                headers=headers,
                verify=False, 
                timeout=5
            )
            
            if response.status_code == 401:
                print("  ✅ Revoked token correctly rejected")
                return True
            else:
                print(f"  ❌ Revoked token was accepted: {response.status_code}")
                return False
        else:
            print(f"  ❌ Token revocation failed: {response.status_code}")
            return False
            
    except Exception as e:
        print(f"  ❌ Token revocation error: {e}")
        return False

if __name__ == "__main__":
    print("🔐 Testing HelixFlow Platform Authentication")
    print("=" * 50)
    
    # Get authentication token
    token = get_auth_token()
    
    if not token:
        print("\n❌ Cannot proceed without authentication token")
        sys.exit(1)
    
    # Test authenticated API
    auth_success = test_authenticated_api(token)
    
    # Test token revocation
    revoke_success = test_token_revocation(token)
    
    print("\n" + "=" * 50)
    if auth_success and revoke_success:
        print("🎉 All authentication tests passed!")
        sys.exit(0)
    else:
        print("❌ Some authentication tests failed")
        sys.exit(1)