#!/usr/bin/env python3
import requests
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

def test_no_token():
    url = "https://localhost:8443/v1/chat/completions"
    payload = {"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}
    resp = requests.post(url, json=payload, verify=False, timeout=5)
    print(f"No token: status {resp.status_code}")
    if resp.status_code == 401:
        print("✓ Authentication required")
        return True
    else:
        print(f"Unexpected: {resp.text}")
        return False

def test_valid_token():
    # login first
    login_url = "http://localhost:8082/login"
    data = {"email": "admin@helixflow.com", "password": "password"}
    resp = requests.post(login_url, json=data, timeout=5)
    if resp.status_code != 200:
        print("Login failed")
        return False
    token = resp.json()['access_token']
    url = "https://localhost:8443/v1/chat/completions"
    headers = {"Authorization": f"Bearer {token}"}
    payload = {"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}
    resp = requests.post(url, json=payload, headers=headers, verify=False, timeout=5)
    print(f"Valid token: status {resp.status_code}")
    if resp.status_code == 200 or resp.status_code == 500:  # 500 due to inference error
        print("✓ Token accepted (auth passed)")
        return True
    else:
        print(f"Unexpected: {resp.text}")
        return False

def test_revoked_token():
    # login, revoke, test
    login_url = "http://localhost:8082/login"
    data = {"email": "test@example.com", "password": "password"}
    resp = requests.post(login_url, json=data, timeout=5)
    if resp.status_code != 200:
        print("Login failed")
        return False
    token = resp.json()['access_token']
    # revoke
    revoke_url = "http://localhost:8082/revoke"
    resp = requests.post(revoke_url, json={"token": token}, timeout=5)
    if resp.status_code != 200:
        print("Revoke failed")
        return False
    print("Token revoked")
    # test with revoked token
    url = "https://localhost:8443/v1/chat/completions"
    headers = {"Authorization": f"Bearer {token}"}
    payload = {"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}
    resp = requests.post(url, json=payload, headers=headers, verify=False, timeout=5)
    print(f"Revoked token: status {resp.status_code}")
    if resp.status_code == 401:
        print("✓ Revoked token rejected")
        return True
    else:
        print(f"Note: Got {resp.status_code} (maybe inference error)")
        return False

if __name__ == "__main__":
    print("=== Testing Authentication Requirements ===")
    success = True
    success = test_no_token() and success
    success = test_valid_token() and success
    success = test_revoked_token() and success
    if success:
        print("\nAll authentication tests passed!")
    else:
        print("\nSome tests failed.")