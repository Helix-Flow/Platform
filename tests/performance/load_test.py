#!/usr/bin/env python3

"""
HelixFlow Performance Load Testing Script
"""

import asyncio
import aiohttp
import time
import json
import statistics
from concurrent.futures import ThreadPoolExecutor
import argparse

class LoadTester:
    def __init__(self, base_url="http://localhost:8080", api_key="test-key"):
        self.base_url = base_url
        self.api_key = api_key
        self.headers = {
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json"
        }
    
    async def test_health_endpoint(self, num_requests=100):
        """Test health endpoint performance"""
        print(f"Testing health endpoint with {num_requests} requests...")
        
        latencies = []
        errors = 0
        
        async with aiohttp.ClientSession() as session:
            tasks = []
            
            for i in range(num_requests):
                task = self.make_health_request(session, i)
                tasks.append(task)
            
            results = await asyncio.gather(*tasks, return_exceptions=True)
            
            for result in results:
                if isinstance(result, Exception):
                    errors += 1
                else:
                    latencies.append(result)
        
        return self.analyze_results(latencies, errors, num_requests)
    
    async def test_chat_endpoint(self, num_requests=50):
        """Test chat completion endpoint performance"""
        print(f"Testing chat endpoint with {num_requests} requests...")
        
        latencies = []
        errors = 0
        
        async with aiohttp.ClientSession() as session:
            tasks = []
            
            for i in range(num_requests):
                task = self.make_chat_request(session, i)
                tasks.append(task)
            
            results = await asyncio.gather(*tasks, return_exceptions=True)
            
            for result in results:
                if isinstance(result, Exception):
                    errors += 1
                else:
                    latencies.append(result)
        
        return self.analyze_results(latencies, errors, num_requests)
    
    async def make_health_request(self, session, request_id):
        """Make a single health request"""
        try:
            start_time = time.time()
            
            async with session.get(
                f"{self.base_url}/health",
                headers=self.headers
            ) as response:
                await response.json()
                
                end_time = time.time()
                latency = (end_time - start_time) * 1000  # Convert to ms
                
                if response.status != 200:
                    raise Exception(f"Health check failed: {response.status}")
                
                return latency
                
        except Exception as e:
            print(f"Request {request_id} failed: {e}")
            return e
    
    async def make_chat_request(self, session, request_id):
        """Make a single chat completion request"""
        try:
            start_time = time.time()
            
            data = {
                "model": "gpt-3.5-turbo",
                "messages": [
                    {"role": "user", "content": f"Test request {request_id}"}
                ],
                "max_tokens": 50
            }
            
            async with session.post(
                f"{self.base_url}/api/v1/chat/completions",
                headers=self.headers,
                json=data
            ) as response:
                await response.json()
                
                end_time = time.time()
                latency = (end_time - start_time) * 1000  # Convert to ms
                
                if response.status != 200:
                    raise Exception(f"Chat request failed: {response.status}")
                
                return latency
                
        except Exception as e:
            print(f"Request {request_id} failed: {e}")
            return e
    
    def analyze_results(self, latencies, errors, total_requests):
        """Analyze performance results"""
        if not latencies:
            return {
                "success_rate": 0,
                "error_rate": 100,
                "avg_latency": 0,
                "p95_latency": 0,
                "p99_latency": 0,
                "min_latency": 0,
                "max_latency": 0,
                "total_requests": total_requests,
                "successful_requests": 0,
                "failed_requests": errors
            }
        
        latencies.sort()
        
        # Calculate percentiles
        p95_index = int(len(latencies) * 0.95)
        p99_index = int(len(latencies) * 0.99)
        
        p95_latency = latencies[p95_index] if p95_index < len(latencies) else latencies[-1]
        p99_latency = latencies[p99_index] if p99_index < len(latencies) else latencies[-1]
        
        return {
            "success_rate": (len(latencies) / total_requests) * 100,
            "error_rate": (errors / total_requests) * 100,
            "avg_latency": statistics.mean(latencies),
            "p95_latency": p95_latency,
            "p99_latency": p99_latency,
            "min_latency": min(latencies),
            "max_latency": max(latencies),
            "total_requests": total_requests,
            "successful_requests": len(latencies),
            "failed_requests": errors
        }

def print_results(test_name, results):
    """Print formatted results"""
    print(f"\n{'='*50}")
    print(f"{test_name} Results")
    print(f"{'='*50}")
    print(f"Success Rate: {results['success_rate']:.2f}%")
    print(f"Error Rate: {results['error_rate']:.2f}%")
    print(f"Average Latency: {results['avg_latency']:.2f}ms")
    print(f"95th Percentile: {results['p95_latency']:.2f}ms")
    print(f"99th Percentile: {results['p99_latency']:.2f}ms")
    print(f"Min Latency: {results['min_latency']:.2f}ms")
    print(f"Max Latency: {results['max_latency']:.2f}ms")
    print(f"Total Requests: {results['total_requests']}")
    print(f"Successful Requests: {results['successful_requests']}")
    print(f"Failed Requests: {results['failed_requests']}")
    
    # Performance assessment
    if results['success_rate'] >= 99.9 and results['avg_latency'] < 100:
        print("\n🎉 EXCELLENT PERFORMANCE!")
    elif results['success_rate'] >= 99.0 and results['avg_latency'] < 200:
        print("\n✅ GOOD PERFORMANCE!")
    else:
        print("\n⚠️  NEEDS OPTIMIZATION!")

async def main():
    """Main function"""
    parser = argparse.ArgumentParser(description="HelixFlow Performance Load Testing")
    parser.add_argument("--base-url", default="http://localhost:8080", help="Base URL for API")
    parser.add_argument("--api-key", default="test-key", help="API key for authentication")
    parser.add_argument("--health-requests", type=int, default=100, help="Number of health requests")
    parser.add_argument("--chat-requests", type=int, default=50, help="Number of chat requests")
    parser.add_argument("--output", default="performance-results.json", help="Output file for results")
    
    args = parser.parse_args()
    
    print("🚀 HelixFlow Performance Load Testing")
    print(f"Base URL: {args.base_url}")
    print(f"API Key: {args.api_key}")
    print(f"Health Requests: {args.health_requests}")
    print(f"Chat Requests: {args.chat_requests}")
    print()
    
    tester = LoadTester(args.base_url, args.api_key)
    
    # Run health endpoint test
    health_results = await tester.test_health_endpoint(args.health_requests)
    print_results("Health Endpoint", health_results)
    
    # Run chat endpoint test
    chat_results = await tester.test_chat_endpoint(args.chat_requests)
    print_results("Chat Endpoint", chat_results)
    
    # Combined results
    combined_results = {
        "health_endpoint": health_results,
        "chat_endpoint": chat_results,
        "timestamp": time.time(),
        "configuration": {
            "base_url": args.base_url,
            "health_requests": args.health_requests,
            "chat_requests": args.chat_requests
        }
    }
    
    # Save results to file
    with open(args.output, 'w') as f:
        json.dump(combined_results, f, indent=2)
    
    print(f"\n📊 Results saved to: {args.output}")
    
    # Performance summary
    print("\n📈 Performance Summary:")
    print(f"Health Endpoint - Avg: {health_results['avg_latency']:.1f}ms, p95: {health_results['p95_latency']:.1f}ms, Success: {health_results['success_rate']:.1f}%")
    print(f"Chat Endpoint - Avg: {chat_results['avg_latency']:.1f}ms, p95: {chat_results['p95_latency']:.1f}ms, Success: {chat_results['success_rate']:.1f}%")

if __name__ == "__main__":
    asyncio.run(main())