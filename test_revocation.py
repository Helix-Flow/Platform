#!/usr/bin/env python3
"""
Test token revocation specifically
"""
import requests
import json

def test_token_revocation():
    """Test token revocation with detailed output"""
    print("Testing token revocation...")
    
    # Get token
    login_data = {"email": "admin", "password": "password"}
    
    response = requests.post(
        "http://localhost:8082/login", 
        json=login_data, 
        timeout=5
    )
    
    if response.status_code != 200:
        print(f"❌ Login failed: {response.status_code} - {response.text}")
        return False
    
    token_data = response.json()
    token = token_data.get("access_token")
    print(f"✅ Got access token: {token[:50]}...")
    
    # Test revocation endpoint details
    # The revoke endpoint expects token in JSON body, not Authorization header
    revoke_data = {"token": token}
    
    print("\nTesting POST /revoke endpoint...")
    response = requests.post(
        "http://localhost:8082/revoke", 
        json=revoke_data,
        timeout=5
    )
    
    print(f"Revocation response status: {response.status_code}")
    print(f"Revocation response: {response.text}")
    print(f"Response headers: {dict(response.headers)}")
    
    if response.status_code == 200:
        print("✅ Token revoked successfully")
        
        # Try to use revoked token
        headers2 = {"Authorization": f"Bearer {token}"}
        response2 = requests.get(
            "http://localhost:8082/health", 
            headers=headers2,
            timeout=5
        )
        
        print(f"\nTest with revoked token: {response2.status_code}")
        if response2.status_code == 401:
            print("✅ Revoked token correctly rejected")
            return True
        else:
            print("❌ Revoked token was still accepted")
            return False
    else:
        print(f"❌ Token revocation failed: {response.status_code}")
        return False

if __name__ == "__main__":
    test_token_revocation()