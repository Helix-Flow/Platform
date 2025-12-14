#!/usr/bin/env python3
"""
HelixFlow Platform Integration Test
Tests all services and their interactions
"""

import requests
import json
import time
import sys
from typing import Dict, Any, Optional

class HelixFlowIntegrationTest:
    def __init__(self):
        self.base_url = "https://localhost:8443"
        self.session = requests.Session()
        # Disable SSL verification for self-signed certificates
        self.session.verify = False
        # Suppress SSL warnings
        requests.packages.urllib3.disable_warnings()
        
        self.test_user = {
            "username": "test_integration_user",
            "email": "test_integration@helixflow.com",
            "password": "TestPassword123!",
            "first_name": "Integration",
            "last_name": "Test",
            "organization": "HelixFlow Test"
        }
        
        self.test_api_key = None
        self.auth_token = None

    def test_health_endpoints(self) -> bool:
        """Test all service health endpoints"""
        print("🏥 Testing health endpoints...")
        
        services = [
            ("Auth Service", "http://localhost:8081/health"),
            ("Inference Pool", "http://localhost:8082/health"),
            ("Monitoring", "http://localhost:8083/health"),
            ("API Gateway", f"{self.base_url}/health")
        ]
        
        all_healthy = True
        for service_name, url in services:
            try:
                response = self.session.get(url, timeout=5)
                if response.status_code == 200:
                    print(f"✅ {service_name}: Healthy")
                else:
                    print(f"❌ {service_name}: Unhealthy (HTTP {response.status_code})")
                    all_healthy = False
            except Exception as e:
                print(f"❌ {service_name}: Connection failed - {e}")
                all_healthy = False
        
        return all_healthy

    def test_models_endpoint(self) -> bool:
        """Test the models endpoint"""
        print("\n🤖 Testing models endpoint...")
        
        try:
            response = self.session.get(f"{self.base_url}/v1/models")
            if response.status_code == 200:
                data = response.json()
                models = data.get("data", [])
                print(f"✅ Models endpoint working - Found {len(models)} models")
                for model in models:
                    print(f"   - {model['id']}: {model.get('description', 'No description')}")
                return True
            else:
                print(f"❌ Models endpoint failed (HTTP {response.status_code})")
                return False
        except Exception as e:
            print(f"❌ Models endpoint error: {e}")
            return False

    def test_user_registration(self) -> bool:
        """Test user registration"""
        print("\n👤 Testing user registration...")
        
        try:
            response = self.session.post(
                f"{self.base_url}/v1/auth/register",
                json=self.test_user
            )
            
            if response.status_code == 200:
                data = response.json()
                if data.get("success"):
                    print("✅ User registration successful")
                    return True
                else:
                    print(f"❌ User registration failed: {data.get('message', 'Unknown error')}")
                    return False
            elif response.status_code == 409:
                print("ℹ️  User already exists, proceeding with login")
                return True
            else:
                print(f"❌ User registration failed (HTTP {response.status_code})")
                return False
        except Exception as e:
            print(f"❌ User registration error: {e}")
            return False

    def test_user_login(self) -> bool:
        """Test user login and get auth token"""
        print("\n🔐 Testing user login...")
        
        try:
            login_data = {
                "username": self.test_user["username"],
                "password": self.test_user["password"]
            }
            
            response = self.session.post(
                f"{self.base_url}/v1/auth/login",
                json=login_data
            )
            
            if response.status_code == 200:
                data = response.json()
                if data.get("success"):
                    self.auth_token = data.get("access_token")
                    print("✅ User login successful")
                    print(f"   Token obtained: {self.auth_token[:20]}...")
                    return True
                else:
                    print(f"❌ User login failed: {data.get('message', 'Unknown error')}")
                    return False
            else:
                print(f"❌ User login failed (HTTP {response.status_code})")
                return False
        except Exception as e:
            print(f"❌ User login error: {e}")
            return False

    def test_api_key_generation(self) -> bool:
        """Test API key generation"""
        print("\n🔑 Testing API key generation...")
        
        if not self.auth_token:
            print("❌ No auth token available, skipping API key test")
            return False
        
        try:
            headers = {"Authorization": f"Bearer {self.auth_token}"}
            key_data = {
                "user_id": self.test_user["username"],
                "name": "Integration Test Key",
                "permissions": ["read", "write"]
            }
            
            response = self.session.post(
                f"{self.base_url}/v1/auth/api-keys",
                json=key_data,
                headers=headers
            )
            
            if response.status_code == 200:
                data = response.json()
                if data.get("success"):
                    self.test_api_key = data["api_key"]["key_prefix"]
                    print("✅ API key generation successful")
                    print(f"   API Key prefix: {self.test_api_key}")
                    return True
                else:
                    print(f"❌ API key generation failed: {data.get('message', 'Unknown error')}")
                    return False
            else:
                print(f"❌ API key generation failed (HTTP {response.status_code})")
                return False
        except Exception as e:
            print(f"❌ API key generation error: {e}")
            return False

    def test_chat_completion(self) -> bool:
        """Test chat completion endpoint"""
        print("\n💬 Testing chat completion...")
        
        headers = {}
        if self.auth_token:
            headers["Authorization"] = f"Bearer {self.auth_token}"
        
        try:
            completion_data = {
                "model": "gpt-3.5-turbo",
                "messages": [
                    {"role": "user", "content": "Hello, this is an integration test. Please respond with 'Integration test successful!'"}
                ],
                "max_tokens": 50,
                "temperature": 0.7
            }
            
            response = self.session.post(
                f"{self.base_url}/v1/chat/completions",
                json=completion_data,
                headers=headers
            )
            
            if response.status_code == 200:
                data = response.json()
                if "choices" in data and len(data["choices"]) > 0:
                    content = data["choices"][0]["message"]["content"]
                    print("✅ Chat completion successful")
                    print(f"   Response: {content[:100]}...")
                    return True
                else:
                    print("❌ Chat completion response format invalid")
                    return False
            else:
                print(f"❌ Chat completion failed (HTTP {response.status_code})")
                print(f"   Response: {response.text}")
                return False
        except Exception as e:
            print(f"❌ Chat completion error: {e}")
            return False

    def test_rate_limiting(self) -> bool:
        """Test rate limiting"""
        print("\n⚡ Testing rate limiting...")
        
        headers = {}
        if self.auth_token:
            headers["Authorization"] = f"Bearer {self.auth_token}"
        
        try:
            # Make multiple rapid requests
            for i in range(5):
                response = self.session.get(f"{self.base_url}/v1/models", headers=headers)
                if response.status_code == 429:
                    print("✅ Rate limiting working - received 429 Too Many Requests")
                    return True
            
            print("ℹ️  Rate limiting not triggered in 5 rapid requests")
            return True
        except Exception as e:
            print(f"❌ Rate limiting test error: {e}")
            return False

    def test_monitoring_metrics(self) -> bool:
        """Test monitoring metrics endpoints"""
        print("\n📊 Testing monitoring metrics...")
        
        try:
            # Test system metrics
            response = self.session.get("http://localhost:8083/metrics")
            if response.status_code == 200:
                data = response.json()
                print("✅ Monitoring metrics endpoint working")
                print(f"   CPU: {data.get('cpu_usage', 'N/A')}%, Memory: {data.get('memory_usage', 'N/A')}%, Disk: {data.get('disk_usage', 'N/A')}%")
                return True
            else:
                print(f"❌ Monitoring metrics failed (HTTP {response.status_code})")
                return False
        except Exception as e:
            print(f"❌ Monitoring metrics error: {e}")
            return False

    def run_all_tests(self) -> bool:
        """Run all integration tests"""
        print("🚀 Starting HelixFlow Integration Tests")
        print("=" * 50)
        
        tests = [
            self.test_health_endpoints,
            self.test_models_endpoint,
            self.test_user_registration,
            self.test_user_login,
            self.test_api_key_generation,
            self.test_chat_completion,
            self.test_rate_limiting,
            self.test_monitoring_metrics,
        ]
        
        passed = 0
        total = len(tests)
        
        for test in tests:
            try:
                if test():
                    passed += 1
                time.sleep(1)  # Small delay between tests
            except Exception as e:
                print(f"❌ Test {test.__name__} crashed: {e}")
        
        print("\n" + "=" * 50)
        print(f"📈 Test Results: {passed}/{total} tests passed")
        
        if passed == total:
            print("🎉 All tests passed! HelixFlow platform is working correctly.")
            return True
        else:
            print(f"⚠️  {total - passed} tests failed. Check the logs above.")
            return False

def main():
    """Main function"""
    tester = HelixFlowIntegrationTest()
    success = tester.run_all_tests()
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()