#!/usr/bin/env python3
import json
import urllib.request
import urllib.error

url = "http://localhost:8082/login"
data = json.dumps({
    "email": "admin@helixflow.com",
    "password": "password"
}).encode('utf-8')

req = urllib.request.Request(url, data=data, headers={'Content-Type': 'application/json'})
try:
    response = urllib.request.urlopen(req, timeout=5)
    body = response.read()
    print(f"Status: {response.status}")
    print(f"Response: {body.decode('utf-8')}")
except urllib.error.HTTPError as e:
    print(f"HTTP Error {e.code}: {e.reason}")
    print(e.read().decode('utf-8'))
except Exception as e:
    print(f"Error: {e}")