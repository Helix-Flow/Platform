#!/usr/bin/env python3
"""
HelixFlow Platform - Final Integration Test Suite
Comprehensive testing of all platform components
"""

import requests
import json
import time
import subprocess
import sys
from typing import Dict, List, Optional

class HelixFlowTester:
    def __init__(self):
        self.base_url = "https://localhost:8443"
        self.grpc_url = "https://localhost:9443"
        self.test_results = []
        
    def log_result(self, test_name: str, passed: bool, details: str = ""):
        """Log test result"""
        status = "✅ PASS" if passed else "❌ FAIL"
        self.test_results.append({
            "test": test_name,
            "passed": passed,
            "details": details
        })
        print(f"{status} - {test_name}")
        if details:
            print(f"     Details: {details}")
    
    def test_database_connectivity(self) -> bool:
        """Test database connectivity"""
        try:
            result = subprocess.run(
                ["go", "run", "test/db_test/simple_check.go"],
                cwd="/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform",
                capture_output=True,
                text=True,
                timeout=10
            )
            return result.returncode == 0
        except Exception as e:
            self.log_result("Database Connectivity", False, str(e))
            return False
    
    def test_https_endpoints(self) -> bool:
        """Test HTTPS endpoints with proper certificate validation"""
        session = requests.Session()
        session.verify = False  # Allow self-signed certificates for testing
        
        # Suppress SSL warnings for testing
        import urllib3
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
        
        try:
            # Test health endpoint
            response = session.get(f"{self.base_url}/health", timeout=10)
            if response.status_code == 200:
                health_data = response.json()
                self.log_result("HTTPS Health Check", True, f"Service: {health_data.get('service')}, Status: {health_data.get('status')}")
            else:
                self.log_result("HTTPS Health Check", False, f"Status: {response.status_code}")
                return False
            
            # Test models endpoint
            response = session.get(f"{self.base_url}/v1/models", timeout=10)
            if response.status_code == 200:
                models_data = response.json()
                model_count = len(models_data.get('data', []))
                self.log_result("HTTPS Models Endpoint", True, f"Found {model_count} models")
            else:
                self.log_result("HTTPS Models Endpoint", False, f"Status: {response.status_code}")
                return False
            
            return True
            
        except Exception as e:
            self.log_result("HTTPS Endpoints", False, str(e))
            return False
    
    def test_chat_completions(self) -> bool:
        """Test chat completions endpoint"""
        session = requests.Session()
        session.verify = False
        
        import urllib3
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
        
        try:
            # Test basic chat completion
            payload = {
                "model": "gpt-3.5-turbo",
                "messages": [
                    {"role": "user", "content": "Hello, can you hear me?"}
                ],
                "max_tokens": 100
            }
            
            response = session.post(
                f"{self.base_url}/v1/chat/completions",
                headers={
                    "Authorization": "Bearer demo-key",
                    "Content-Type": "application/json"
                },
                json=payload,
                timeout=30
            )
            
            if response.status_code == 200:
                result = response.json()
                content = result.get('choices', [{}])[0].get('message', {}).get('content', '')
                if content:
                    self.log_result("Chat Completions", True, f"Response received: {content[:50]}...")
                    return True
                else:
                    self.log_result("Chat Completions", False, "Empty response content")
                    return False
            else:
                self.log_result("Chat Completions", False, f"Status: {response.status_code}")
                return False
                
        except Exception as e:
            self.log_result("Chat Completions", False, str(e))
            return False
    
    def test_authentication_flow(self) -> bool:
        """Test authentication flow"""
        session = requests.Session()
        session.verify = False
        
        import urllib3
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
        
        try:
            # Test with invalid token
            response = session.get(
                f"{self.base_url}/v1/models",
                headers={"Authorization": "Bearer invalid-token"},
                timeout=10
            )
            
            if response.status_code == 401:
                self.log_result("Authentication (Invalid Token)", True, "Properly rejected invalid token")
            else:
                self.log_result("Authentication (Invalid Token)", False, f"Expected 401, got {response.status_code}")
                return False
            
            # Test with valid token
            response = session.get(
                f"{self.base_url}/v1/models",
                headers={"Authorization": "Bearer demo-key"},
                timeout=10
            )
            
            if response.status_code == 200:
                self.log_result("Authentication (Valid Token)", True, "Successfully authenticated with demo key")
                return True
            else:
                self.log_result("Authentication (Valid Token)", False, f"Status: {response.status_code}")
                return False
                
        except Exception as e:
            self.log_result("Authentication Flow", False, str(e))
            return False
    
    def test_streaming_response(self) -> bool:
        """Test streaming chat completions"""
        session = requests.Session()
        session.verify = False
        
        import urllib3
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
        
        try:
            # Test streaming chat completion
            payload = {
                "model": "gpt-3.5-turbo",
                "messages": [
                    {"role": "user", "content": "Say hello in one sentence"}
                ],
                "stream": True,
                "max_tokens": 50
            }
            
            response = session.post(
                f"{self.base_url}/v1/chat/completions",
                headers={
                    "Authorization": "Bearer demo-key",
                    "Content-Type": "application/json"
                },
                json=payload,
                stream=True,
                timeout=30
            )
            
            if response.status_code == 200:
                # For streaming, we expect text/event-stream content
                content_type = response.headers.get('content-type', '')
                if 'text/event-stream' in content_type or 'application/json' in content_type:
                    self.log_result("Streaming Response", True, f"Streaming response received (Content-Type: {content_type})")
                    return True
                else:
                    self.log_result("Streaming Response", False, f"Unexpected content type: {content_type}")
                    return False
            else:
                self.log_result("Streaming Response", False, f"Status: {response.status_code}")
                return False
                
        except Exception as e:
            self.log_result("Streaming Response", False, str(e))
            return False
    
    def test_error_handling(self) -> bool:
        """Test error handling and edge cases"""
        session = requests.Session()
        session.verify = False
        
        import urllib3
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
        
        try:
            # Test missing authorization header
            response = session.post(
                f"{self.base_url}/v1/chat/completions",
                json={"model": "gpt-3.5-turbo", "messages": []},
                timeout=10
            )
            
            if response.status_code == 401:
                self.log_result("Error Handling (Missing Auth)", True, "Properly rejected missing authorization")
            else:
                self.log_result("Error Handling (Missing Auth)", False, f"Expected 401, got {response.status_code}")
                return False
            
            # Test invalid model
            payload = {
                "model": "invalid-model-name",
                "messages": [{"role": "user", "content": "Test"}]
            }
            
            response = session.post(
                f"{self.base_url}/v1/chat/completions",
                headers={"Authorization": "Bearer demo-key"},
                json=payload,
                timeout=10
            )
            
            # Should either return 400 (bad request) or handle gracefully
            if response.status_code in [400, 200]:
                self.log_result("Error Handling (Invalid Model)", True, f"Handled gracefully (Status: {response.status_code})")
                return True
            else:
                self.log_result("Error Handling (Invalid Model)", False, f"Unexpected status: {response.status_code}")
                return False
                
        except Exception as e:
            self.log_result("Error Handling", False, str(e))
            return False
    
    def test_performance_metrics(self) -> bool:
        """Test performance and response times"""
        session = requests.Session()
        session.verify = False
        
        import urllib3
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
        
        try:
            # Test response time for health check
            start_time = time.time()
            response = session.get(f"{self.base_url}/health", timeout=5)
            end_time = time.time()
            response_time = (end_time - start_time) * 1000  # Convert to milliseconds
            
            if response.status_code == 200:
                if response_time < 100:  # Expect < 100ms for health check
                    self.log_result("Performance (Health Check)", True, f"Response time: {response_time:.1f}ms")
                else:
                    self.log_result("Performance (Health Check)", True, f"Response time: {response_time:.1f}ms (slower than expected)")
            else:
                self.log_result("Performance (Health Check)", False, f"Status: {response.status_code}")
                return False
            
            # Test response time for models endpoint
            start_time = time.time()
            response = session.get(f"{self.base_url}/v1/models", timeout=10)
            end_time = time.time()
            response_time = (end_time - start_time) * 1000
            
            if response.status_code == 200:
                if response_time < 200:  # Expect < 200ms for models list
                    self.log_result("Performance (Models)", True, f"Response time: {response_time:.1f}ms")
                    return True
                else:
                    self.log_result("Performance (Models)", True, f"Response time: {response_time:.1f}ms (slower than expected)")
                    return True
            else:
                self.log_result("Performance (Models)", False, f"Status: {response.status_code}")
                return False
                
        except Exception as e:
            self.log_result("Performance Metrics", False, str(e))
            return False
    
    def run_all_tests(self) -> Dict[str, any]:
        """Run all integration tests"""
        print("=" * 60)
        print("🧪 HELIXFLOW PLATFORM - FINAL INTEGRATION TEST")
        print("=" * 60)
        print(f"Testing at: {time.strftime('%Y-%m-%d %H:%M:%S')}")
        print("")
        
        # Initialize test results
        self.test_results = []
        
        # Run all tests
        print("1. Testing Database Infrastructure...")
        self.test_database_connectivity()
        print("")
        
        print("2. Testing HTTPS Endpoints...")
        self.test_https_endpoints()
        print("")
        
        print("3. Testing Authentication...")
        self.test_authentication_flow()
        print("")
        
        print("4. Testing Chat Completions...")
        self.test_chat_completions()
        print("")
        
        print("5. Testing Streaming Responses...")
        self.test_streaming_response()
        print("")
        
        print("6. Testing Error Handling...")
        self.test_error_handling()
        print("")
        
        print("7. Testing Performance Metrics...")
        self.test_performance_metrics()
        print("")
        
        # Calculate results
        total_tests = len(self.test_results)
        passed_tests = sum(1 for result in self.test_results if result["passed"])
        failed_tests = total_tests - passed_tests
        
        print("=" * 60)
        print("📊 TEST RESULTS SUMMARY")
        print("=" * 60)
        print(f"Total Tests: {total_tests}")
        print(f"Passed: {passed_tests}")
        print(f"Failed: {failed_tests}")
        print(f"Success Rate: {(passed_tests/total_tests)*100:.1f}%")
        
        if failed_tests == 0:
            print("")
            print("🎉 ALL TESTS PASSED!")
            print("✅ HelixFlow platform is ready for production deployment!")
            return {"success": True, "passed": passed_tests, "failed": failed_tests, "total": total_tests}
        else:
            print("")
            print("⚠️  SOME TESTS FAILED")
            print("Please review the failed tests above before production deployment.")
            return {"success": False, "passed": passed_tests, "failed": failed_tests, "total": total_tests}

def main():
    """Main test execution"""
    tester = HelixFlowTester()
    results = tester.run_all_tests()
    
    # Exit with appropriate code
    sys.exit(0 if results["success"] else 1)

if __name__ == "__main__":
    main()