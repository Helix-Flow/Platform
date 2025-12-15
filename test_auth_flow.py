#!/usr/bin/env python3
import json
import requests
import sys

# Disable SSL warnings for self-signed certs
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

def login():
    url = "http://localhost:8082/login"
    data = {
        "email": "admin@helixflow.com",
        "password": "password"
    }
    resp = requests.post(url, json=data, timeout=5)
    if resp.status_code != 200:
        print(f"Login failed: {resp.status_code} {resp.text}")
        sys.exit(1)
    tokens = resp.json()
    access_token = tokens['access_token']
    refresh_token = tokens['refresh_token']
    print(f"Login successful. Access token: {access_token[:50]}...")
    return access_token, refresh_token

def test_protected_endpoint(token):
    url = "https://localhost:8443/v1/chat/completions"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    # Minimal payload
    payload = {
        "model": "gpt-3.5-turbo",
        "messages": [{"role": "user", "content": "Hello"}],
        "max_tokens": 10
    }
    resp = requests.post(url, json=payload, headers=headers, verify=False, timeout=10)
    print(f"Protected endpoint status: {resp.status_code}")
    if resp.status_code != 200:
        print(f"Response: {resp.text}")
    else:
        print(f"Response: {resp.json()}")
    return resp.status_code

def test_invalid_token():
    url = "https://localhost:8443/v1/chat/completions"
    headers = {
        "Authorization": "Bearer invalid_token",
        "Content-Type": "application/json"
    }
    payload = {
        "model": "gpt-3.5-turbo",
        "messages": [{"role": "user", "content": "Hello"}],
        "max_tokens": 10
    }
    resp = requests.post(url, json=payload, headers=headers, verify=False, timeout=10)
    print(f"Invalid token test status: {resp.status_code}")
    if resp.status_code == 401:
        print("✓ Expected 401 Unauthorized")
    else:
        print(f"Unexpected response: {resp.text}")
    return resp.status_code

def main():
    print("=== Testing HelixFlow Authentication Flow ===")
    print("1. Logging in...")
    access_token, refresh_token = login()
    
    print("\n2. Testing protected endpoint with valid token...")
    status = test_protected_endpoint(access_token)
    if status == 200:
        print("✓ Authentication successful! API Gateway validated token via gRPC.")
    else:
        print("✗ Authentication failed.")
    
    print("\n3. Testing protected endpoint with invalid token...")
    test_invalid_token()
    
    print("\n=== Test complete ===")

if __name__ == "__main__":
    main()