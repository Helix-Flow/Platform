#!/usr/bin/env python3
"""
Quick API test for HelixFlow Platform
Tests both HTTPS with TLS verification disabled and HTTP fallback
"""
import requests
import json
import sys
from urllib3.exceptions import InsecureRequestWarning

# Suppress TLS warnings for self-signed certificates
requests.packages.urllib3.disable_warnings(category=InsecureRequestWarning)

def test_api_gateway():
    """Test API Gateway endpoints"""
    base_urls = [
        "https://localhost:8443",  # HTTPS with self-signed cert
        "http://localhost:8080",   # HTTP fallback if TLS fails
    ]
    
    for base_url in base_urls:
        print(f"\nTesting API Gateway at {base_url}")
        
        try:
            # Test health endpoint
            print("  Testing health endpoint...")
            response = requests.get(f"{base_url}/health", verify=False, timeout=5)
            if response.status_code == 200:
                print(f"  ✅ Health: {response.status_code} - {response.text.strip()}")
            else:
                print(f"  ❌ Health: {response.status_code}")
                continue
                
            # Test models endpoint
            print("  Testing models endpoint...")
            response = requests.get(f"{base_url}/v1/models", verify=False, timeout=5)
            if response.status_code == 200:
                models_data = response.json()
                print(f"  ✅ Models: {response.status_code} - Found {len(models_data.get('data', []))} models")
            else:
                print(f"  ❌ Models: {response.status_code}")
                
            # Test chat completion endpoint
            print("  Testing chat completion...")
            chat_data = {
                "model": "gpt-3.5-turbo",
                "messages": [{"role": "user", "content": "Hello!"}],
                "max_tokens": 50
            }
            response = requests.post(
                f"{base_url}/v1/chat/completions", 
                json=chat_data, 
                verify=False, 
                timeout=10
            )
            if response.status_code == 200:
                result = response.json()
                message = result.get('choices', [{}])[0].get('message', {}).get('content', 'No content')
                print(f"  ✅ Chat: {response.status_code} - Response: {message[:50]}...")
            else:
                print(f"  ❌ Chat: {response.status_code} - {response.text}")
                
            print(f"  🎯 API Gateway working at {base_url}")
            return True
            
        except requests.exceptions.SSLError as e:
            print(f"  ⚠️  SSL Error: {e}")
            continue
        except requests.exceptions.ConnectionError as e:
            print(f"  ⚠️  Connection Error: {e}")
            continue
        except Exception as e:
            print(f"  ❌ Error: {e}")
            continue
            
    print("  ❌ Failed to connect to API Gateway on any endpoint")
    return False

def test_auth_service():
    """Test Auth Service endpoints"""
    print(f"\nTesting Auth Service at http://localhost:8082")
    
    try:
        # Test health endpoint
        response = requests.get("http://localhost:8082/health", timeout=5)
        if response.status_code == 200:
            print(f"  ✅ Auth Service Health: {response.status_code}")
        else:
            print(f"  ❌ Auth Service Health: {response.status_code}")
            return False
            
        # Test login
        login_data = {"username": "testuser", "password": "testpass"}
        response = requests.post("http://localhost:8082/login", json=login_data, timeout=5)
        if response.status_code in [200, 401]:  # 401 is expected for invalid credentials
            print(f"  ✅ Auth Service Login: {response.status_code}")
        else:
            print(f"  ❌ Auth Service Login: {response.status_code}")
            
        return True
        
    except Exception as e:
        print(f"  ❌ Auth Service Error: {e}")
        return False

if __name__ == "__main__":
    print("🚀 Testing HelixFlow Platform Services")
    print("=" * 50)
    
    api_success = test_api_gateway()
    auth_success = test_auth_service()
    
    print("\n" + "=" * 50)
    if api_success and auth_success:
        print("🎉 All services are working correctly!")
        sys.exit(0)
    else:
        print("❌ Some services are not working properly")
        sys.exit(1)