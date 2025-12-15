#!/usr/bin/env python3
import requests
import json
import sys

def test_token_revocation():
    auth_url = "http://localhost:8082"
    gateway_url = "https://localhost:8443"
    
    print("1. Logging in with test credentials...")
    login_data = {"email": "test@example.com", "password": "password"}
    try:
        resp = requests.post(f"{auth_url}/login", json=login_data, timeout=5)
        resp.raise_for_status()
        login_result = resp.json()
        token = login_result.get("access_token")
        if not token:
            print("ERROR: No access_token in response", login_result)
            return False
        print(f"   Got token: {token[:20]}...")
    except Exception as e:
        print(f"ERROR: Login failed: {e}")
        # Try to register first
        print("   Attempting registration...")
        reg_data = {"username": "testuser", "email": "test@example.com", "password": "password"}
        try:
            resp = requests.post(f"{auth_url}/register", json=reg_data, timeout=5)
            resp.raise_for_status()
            print("   Registered test user")
            # Now login
            resp = requests.post(f"{auth_url}/login", json=login_data, timeout=5)
            resp.raise_for_status()
            login_result = resp.json()
            token = login_result.get("access_token")
            if not token:
                print("ERROR: No access_token after registration")
                return False
            print(f"   Got token: {token[:20]}...")
        except Exception as e2:
            print(f"ERROR: Registration also failed: {e2}")
            return False
    
    print("2. Testing valid token with gateway /v1/models...")
    headers = {"Authorization": f"Bearer {token}"}
    try:
        resp = requests.get(f"{gateway_url}/v1/models", headers=headers, verify=False, timeout=5)
        print(f"   Response status: {resp.status_code}")
        if resp.status_code == 200:
            print("   ✅ Valid token accepted")
        else:
            print(f"   ❌ Unexpected status: {resp.text}")
            return False
    except Exception as e:
        print(f"ERROR: Gateway request failed: {e}")
        return False
    
    print("3. Revoking token...")
    revoke_data = {"token": token}
    try:
        resp = requests.post(f"{auth_url}/revoke", json=revoke_data, timeout=5)
        resp.raise_for_status()
        print("   ✅ Token revoked")
    except Exception as e:
        print(f"ERROR: Revoke failed: {e}")
        return False
    
    print("4. Testing revoked token with gateway /v1/models...")
    try:
        resp = requests.get(f"{gateway_url}/v1/models", headers=headers, verify=False, timeout=5)
        print(f"   Response status: {resp.status_code}")
        if resp.status_code == 401:
            print("   ✅ Revoked token correctly rejected with 401")
            return True
        else:
            print(f"   ❌ Expected 401, got {resp.status_code}: {resp.text}")
            return False
    except Exception as e:
        print(f"ERROR: Gateway request failed: {e}")
        return False

if __name__ == "__main__":
    success = test_token_revocation()
    sys.exit(0 if success else 1)