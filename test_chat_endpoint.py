#!/usr/bin/env python3
"""Test the chat completions endpoint directly"""

import requests
import json
import urllib3

# Disable SSL warnings for testing
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

def test_chat_endpoint():
    """Test the chat completions endpoint"""
    
    url = "http://localhost:8443/v1/chat/completions"
    headers = {
        "Authorization": "Bearer demo-key",
        "Content-Type": "application/json"
    }
    
    payload = {
        "model": "gpt-3.5-turbo",
        "messages": [
            {"role": "user", "content": "Hello, can you hear me?"}
        ],
        "max_tokens": 100
    }
    
    try:
        print("Testing chat completions endpoint...")
        print(f"URL: {url}")
        print(f"Headers: {headers}")
        print(f"Payload: {json.dumps(payload, indent=2)}")
        print()
        
        response = requests.post(url, headers=headers, json=payload, timeout=30)
        
        print(f"Response Status: {response.status_code}")
        print(f"Response Headers: {dict(response.headers)}")
        print()
        
        if response.status_code == 200:
            result = response.json()
            print("✅ SUCCESS - Response received:")
            print(json.dumps(result, indent=2))
            
            # Extract the response content
            content = result.get('choices', [{}])[0].get('message', {}).get('content', '')
            if content:
                print(f"\nResponse Content: {content}")
            return True
        else:
            print(f"❌ FAILED - Status: {response.status_code}")
            print(f"Response: {response.text}")
            return False
            
    except requests.exceptions.ConnectionError as e:
        print(f"❌ CONNECTION FAILED: {e}")
        return False
    except Exception as e:
        print(f"❌ ERROR: {e}")
        return False

if __name__ == "__main__":
    success = test_chat_endpoint()
    exit(0 if success else 1)