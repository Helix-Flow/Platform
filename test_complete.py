#!/usr/bin/env python3
"""
Complete API test with working authentication
"""
import requests
import json
import sys
from urllib3.exceptions import InsecureRequestWarning

# Suppress TLS warnings
requests.packages.urllib3.disable_warnings(category=InsecureRequestWarning)

def get_auth_token():
    """Get authentication token with correct credentials"""
    print("Getting authentication token...")
    
    login_data = {"email": "admin", "password": "password"}
    
    response = requests.post(
        "http://localhost:8082/login", 
        json=login_data, 
        timeout=5
    )
    
    if response.status_code == 200:
        token_data = response.json()
        token = token_data.get("access_token")
        print("✅ Authentication token obtained successfully")
        return token
    else:
        print(f"❌ Authentication failed: {response.status_code} - {response.text}")
        return None

def test_api_endpoints(token):
    """Test API Gateway endpoints with authentication"""
    print("\nTesting API Gateway endpoints...")
    
    headers = {"Authorization": f"Bearer {token}"}
    base_url = "https://localhost:8443"
    
    # Test models endpoint
    try:
        response = requests.get(f"{base_url}/v1/models", headers=headers, verify=False, timeout=5)
        if response.status_code == 200:
            models_data = response.json()
            print(f"✅ Models endpoint: Found {len(models_data.get('data', []))} models")
        else:
            print(f"❌ Models endpoint: {response.status_code}")
    except Exception as e:
        print(f"❌ Models endpoint error: {e}")
        return False
    
    # Test chat completion
    try:
        chat_data = {
            "model": "gpt-3.5-turbo",
            "messages": [{"role": "user", "content": "Hello! What is HelixFlow?"}],
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
            print(f"✅ Chat completion working: {message[:100]}...")
            return True
        else:
            print(f"❌ Chat completion: {response.status_code} - {response.text}")
            return False
            
    except Exception as e:
        print(f"❌ Chat completion error: {e}")
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
            print("✅ Token revoked successfully")
            
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
                print("✅ Revoked token correctly rejected")
                return True
            else:
                print(f"❌ Revoked token was accepted: {response.status_code}")
                return False
        else:
            print(f"❌ Token revocation failed: {response.status_code}")
            return False
            
    except Exception as e:
        print(f"❌ Token revocation error: {e}")
        return False

if __name__ == "__main__":
    print("🚀 Testing Complete HelixFlow Platform")
    print("=" * 50)
    
    # Get authentication token
    token = get_auth_token()
    
    if not token:
        print("\n❌ Cannot proceed without authentication token")
        sys.exit(1)
    
    # Test API endpoints
    api_success = test_api_endpoints(token)
    
    # Test token revocation
    revoke_success = test_token_revocation(token)
    
    print("\n" + "=" * 50)
    if api_success and revoke_success:
        print("🎉 All tests passed! Platform is fully operational!")
        print("✅ Authentication service working")
        print("✅ API Gateway working with TLS")
        print("✅ Inference service working")
        print("✅ Token revocation working")
        sys.exit(0)
    else:
        print("❌ Some tests failed")
        sys.exit(1)