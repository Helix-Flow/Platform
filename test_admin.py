#!/usr/bin/env python3
"""
Test with existing admin credentials
"""
import requests
import json
import sys
from urllib3.exceptions import InsecureRequestWarning

# Suppress TLS warnings
requests.packages.urllib3.disable_warnings(category=InsecureRequestWarning)

def test_existing_credentials():
    """Test login with existing admin user"""
    print("Testing with existing admin credentials...")
    
    # Try with admin user
    login_data = {"username": "admin", "password": "admin123"}
    
    response = requests.post(
        "http://localhost:8082/login", 
        json=login_data, 
        timeout=5
    )
    
    print(f"Login response status: {response.status_code}")
    print(f"Login response: {response.text}")
    
    if response.status_code == 200:
        token_data = response.json()
        token = token_data.get("token")
        print("✅ Successfully authenticated with admin credentials")
        
        # Test API with token
        headers = {"Authorization": f"Bearer {token}"}
        chat_data = {
            "model": "gpt-3.5-turbo",
            "messages": [{"role": "user", "content": "Hello from authenticated test!"}],
            "max_tokens": 50
        }
        
        response = requests.post(
            "https://localhost:8443/v1/chat/completions", 
            json=chat_data, 
            headers=headers,
            verify=False, 
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            message = result.get('choices', [{}])[0].get('message', {}).get('content', 'No content')
            print(f"✅ Authenticated API working: {message}")
            return True
        else:
            print(f"❌ API call failed: {response.status_code} - {response.text}")
            return False
    else:
        print(f"❌ Authentication failed")
        return False

if __name__ == "__main__":
    if test_existing_credentials():
        print("\n🎉 Platform is fully operational!")
    else:
        print("\n❌ Authentication issues detected")