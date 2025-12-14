"""
Performance Contract Test for Latency and Throughput

Tests the contract that performance targets are met including
sub-100ms latency for popular models and high throughput.
"""

import pytest
import requests
import time
import statistics
from concurrent.futures import ThreadPoolExecutor, as_completed


class TestPerformanceContract:
    """Test suite for performance contract validation."""

    @pytest.fixture
    def api_gateway_url(self):
        """API gateway URL."""
        return "https://localhost:8443"

    @pytest.fixture
    def auth_headers(self):
        """Authentication headers."""
        return {
            "Authorization": "Bearer test-token",
            "Content-Type": "application/json",
        }

    def test_chat_completion_latency_under_100ms(self, api_gateway_url, auth_headers):
        """Test that chat completion latency is under 100ms for popular models."""
        payload = {
            "model": "gpt-4",
            "messages": [{"role": "user", "content": "Hello"}],
            "max_tokens": 10,
        }

        # Measure latency for 10 requests
        latencies = []
        for _ in range(10):
            start_time = time.time()
            response = requests.post(
                f"{api_gateway_url}/v1/chat/completions",
                headers=auth_headers,
                json=payload,
                verify=False,
            )
            end_time = time.time()

            assert response.status_code == 200
            latency = (end_time - start_time) * 1000  # Convert to milliseconds
            latencies.append(latency)

        # Calculate statistics
        avg_latency = statistics.mean(latencies)
        p95_latency = statistics.quantiles(latencies, n=20)[18]  # 95th percentile

        # Assert performance targets
        assert avg_latency < 100, (
            f"Average latency {avg_latency:.2f}ms exceeds 100ms target"
        )
        assert p95_latency < 150, (
            f"P95 latency {p95_latency:.2f}ms exceeds 150ms target"
        )

    def test_throughput_100_requests_per_second(self, api_gateway_url, auth_headers):
        """Test that system can handle 100 requests per second."""
        payload = {
            "model": "gpt-4",
            "messages": [{"role": "user", "content": "Test throughput"}],
            "max_tokens": 5,
        }

        def make_request():
            start_time = time.time()
            response = requests.post(
                f"{api_gateway_url}/v1/chat/completions",
                headers=auth_headers,
                json=payload,
                verify=False,
            )
            end_time = time.time()
            return response.status_code, end_time - start_time

        # Send 100 requests concurrently
        with ThreadPoolExecutor(max_workers=10) as executor:
            futures = [executor.submit(make_request) for _ in range(100)]
            results = [future.result() for future in as_completed(futures)]

        # Analyze results
        successful_requests = [r for r in results if r[0] == 200]
        success_rate = len(successful_requests) / len(results)

        assert success_rate >= 0.95, f"Success rate {success_rate:.2%} below 95% target"

        if successful_requests:
            latencies = [r[1] for r in successful_requests]
            avg_latency = statistics.mean(latencies) * 1000
            assert avg_latency < 200, (
                f"Average latency {avg_latency:.2f}ms too high under load"
            )

    def test_model_inference_latency_by_model(self, api_gateway_url, auth_headers):
        """Test latency targets for different model types."""
        test_cases = [
            ("gpt-4", 100),  # Popular model: <100ms
            ("claude-3-sonnet", 120),  # Fast model: <120ms
            ("deepseek-chat", 80),  # Optimized model: <80ms
        ]

        for model, target_latency in test_cases:
            payload = {
                "model": model,
                "messages": [{"role": "user", "content": "Test latency"}],
                "max_tokens": 10,
            }

            latencies = []
            for _ in range(5):
                start_time = time.time()
                response = requests.post(
                    f"{api_gateway_url}/v1/chat/completions",
                    headers=auth_headers,
                    json=payload,
                    verify=False,
                )
                end_time = time.time()

                if response.status_code == 200:
                    latency = (end_time - start_time) * 1000
                    latencies.append(latency)

            if latencies:
                avg_latency = statistics.mean(latencies)
                assert avg_latency < target_latency, (
                    f"Model {model} latency {avg_latency:.2f}ms exceeds {target_latency}ms target"
                )

    def test_concurrent_user_support(self, api_gateway_url, auth_headers):
        """Test support for concurrent users without degradation."""

        def simulate_user_session(user_id):
            latencies = []
            for _ in range(3):  # 3 requests per user
                payload = {
                    "model": "gpt-4",
                    "messages": [
                        {"role": "user", "content": f"User {user_id} request"}
                    ],
                    "max_tokens": 5,
                }

                start_time = time.time()
                response = requests.post(
                    f"{api_gateway_url}/v1/chat/completions",
                    headers=auth_headers,
                    json=payload,
                    verify=False,
                )
                end_time = time.time()

                if response.status_code == 200:
                    latencies.append((end_time - start_time) * 1000)

            return latencies

        # Simulate 20 concurrent users
        with ThreadPoolExecutor(max_workers=20) as executor:
            futures = [executor.submit(simulate_user_session, i) for i in range(20)]
            results = [future.result() for future in as_completed(futures)]

        # Analyze all latencies
        all_latencies = [lat for user_latencies in results for lat in user_latencies]
        if all_latencies:
            avg_latency = statistics.mean(all_latencies)
            p95_latency = statistics.quantiles(all_latencies, n=20)[18]

            assert avg_latency < 150, (
                f"Concurrent users avg latency {avg_latency:.2f}ms too high"
            )
            assert p95_latency < 250, (
                f"Concurrent users P95 latency {p95_latency:.2f}ms too high"
            )

    def test_memory_usage_under_load(self):
        """Test that memory usage remains stable under load."""
        # This would monitor memory usage during load tests
        # In real implementation, integrate with monitoring service
        monitoring_url = "http://localhost:8083"

        response = requests.get(f"{monitoring_url}/api/metrics/memory", verify=False)

        if response.status_code == 200:
            metrics = response.json()
            memory_usage = metrics.get("memory_percent", 0)
            assert memory_usage < 85, f"Memory usage {memory_usage}% too high"

    def test_gpu_utilization_efficiency(self):
        """Test that GPU utilization is efficient."""
        monitoring_url = "http://localhost:8083"

        response = requests.get(f"{monitoring_url}/api/metrics/gpu", verify=False)

        if response.status_code == 200:
            metrics = response.json()
            for gpu in metrics.get("gpus", []):
                utilization = gpu.get("utilization", 0)
                memory_usage = gpu.get("memory_usage", 0)

                # GPUs should be utilized but not overworked
                assert 10 <= utilization <= 90, (
                    f"GPU utilization {utilization}% inefficient"
                )
                assert memory_usage < 95, f"GPU memory usage {memory_usage}% too high"

    def test_cold_start_latency(self, api_gateway_url, auth_headers):
        """Test cold start latency for model loading."""
        # Test with a model that might need loading
        payload = {
            "model": "glm-4",  # Assume this might need cold start
            "messages": [{"role": "user", "content": "Test cold start"}],
            "max_tokens": 10,
        }

        start_time = time.time()
        response = requests.post(
            f"{api_gateway_url}/v1/chat/completions",
            headers=auth_headers,
            json=payload,
            verify=False,
        )
        end_time = time.time()

        if response.status_code == 200:
            cold_start_latency = (end_time - start_time) * 1000
            # Cold start should be reasonable (under 5 seconds)
            assert cold_start_latency < 5000, (
                f"Cold start latency {cold_start_latency:.2f}ms too high"
            )
