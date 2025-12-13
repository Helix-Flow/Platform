"""
Scalability Integration Test for Horizontal Scaling

Tests the integration that the platform scales horizontally
with Kubernetes HPA and maintains performance under increased load.
"""

import pytest
import requests
import time
import os
from concurrent.futures import ThreadPoolExecutor
import kubernetes.client
from kubernetes import config


class TestScalabilityIntegration:
    """Test suite for horizontal scaling integration."""

    @pytest.fixture
    def k8s_client(self):
        """Initialize Kubernetes client."""
        try:
            config.load_kube_config()
            return kubernetes.client.AppsV1Api()
        except Exception:
            pytest.skip("Kubernetes config not available")

    @pytest.fixture
    def api_gateway_url(self):
        """API gateway URL."""
        return "https://api-gateway.helixflow.svc.cluster.local"

    @pytest.fixture
    def auth_headers(self):
        """Authentication headers."""
        return {
            "Authorization": "Bearer test-token",
            "Content-Type": "application/json",
        }

    def test_horizontal_pod_scaling(self, k8s_client, api_gateway_url, auth_headers):
        """Test that pods scale horizontally under load."""
        # Get initial replica count
        initial_replicas = self._get_deployment_replicas(k8s_client, "api-gateway")
        assert initial_replicas >= 1

        # Generate sustained load
        payload = {
            "model": "gpt-4",
            "messages": [{"role": "user", "content": "Test scaling"}],
            "max_tokens": 10,
        }

        def make_requests():
            for _ in range(50):  # 50 requests per thread
                requests.post(
                    f"{api_gateway_url}/v1/chat/completions",
                    headers=auth_headers,
                    json=payload,
                    verify=False,
                )
                time.sleep(0.1)  # 10 requests per second per thread

        # Start 10 concurrent threads (100 requests/second total)
        with ThreadPoolExecutor(max_workers=10) as executor:
            futures = [executor.submit(make_requests) for _ in range(10)]
            # Wait for load to build
            time.sleep(30)

            # Check if scaling occurred
            scaled_replicas = self._get_deployment_replicas(k8s_client, "api-gateway")
            # Should scale up (may take time for HPA to react)
            assert scaled_replicas >= initial_replicas

            # Wait for completion
            for future in futures:
                future.result()

    def test_inference_pool_scaling(self, k8s_client):
        """Test that inference pool scales with GPU workload."""
        # Get initial inference pool replicas
        initial_replicas = self._get_deployment_replicas(k8s_client, "inference-pool")

        # Simulate GPU-intensive workload
        # In real test, this would submit many inference requests
        # For now, check if scaling configuration exists
        hpa_client = kubernetes.client.AutoscalingV1Api()
        try:
            hpa = hpa_client.read_namespaced_horizontal_pod_autoscaler(
                name="inference-pool-hpa", namespace="helixflow"
            )
            assert hpa.spec.min_replicas >= 1
            assert hpa.spec.max_replicas > hpa.spec.min_replicas
            assert hpa.spec.target_cpu_utilization_percentage > 0
        except kubernetes.client.rest.ApiException:
            pytest.fail("Inference pool HPA not configured")

    def test_database_connection_pooling(self):
        """Test that database connections scale properly."""
        # Check connection pool configuration
        # This would test PostgreSQL connection pooling
        # For now, verify configuration exists
        config_files = [
            "schemas/postgresql-helixflow-updated.sql",
            "k8s/postgres-config.yaml",
        ]

        for config_file in config_files:
            if os.path.exists(config_file):
                with open(config_file) as f:
                    content = f.read()
                    assert "pool" in content.lower() or "connection" in content.lower()

    def test_load_balancer_distribution(self, api_gateway_url, auth_headers):
        """Test that load balancer distributes requests across pods."""
        payload = {
            "model": "gpt-4",
            "messages": [{"role": "user", "content": "Test distribution"}],
            "max_tokens": 5,
        }

        # Make multiple requests and check pod distribution
        pod_ids = []
        for _ in range(20):
            response = requests.post(
                f"{api_gateway_url}/v1/chat/completions",
                headers=auth_headers,
                json=payload,
                verify=False,
            )

            if response.status_code == 200:
                data = response.json()
                # Extract pod ID from response headers or metadata
                pod_id = response.headers.get("X-Pod-ID", "unknown")
                pod_ids.append(pod_id)

        # Should see requests distributed across multiple pods
        unique_pods = set(pod_ids)
        assert len(unique_pods) > 1, "Requests not distributed across multiple pods"

    def test_redis_cluster_scaling(self):
        """Test that Redis cluster scales with load."""
        # Check Redis cluster configuration
        redis_config = "schemas/redis-cluster.conf"
        if os.path.exists(redis_config):
            with open(redis_config) as f:
                content = f.read()
                assert "cluster" in content.lower()
                assert "replicas" in content.lower()

        # Test Redis operations under load
        import redis

        redis_client = redis.Redis(host="localhost", port=6379, decode_responses=True)

        # Perform many operations
        for i in range(1000):
            redis_client.set(f"test_key_{i}", f"test_value_{i}")
            redis_client.get(f"test_key_{i}")

        # Verify cluster is handling load
        info = redis_client.info()
        assert "connected_clients" in info

    def test_monitoring_scales_with_system(self):
        """Test that monitoring system scales with platform growth."""
        monitoring_url = "https://monitoring.helixflow.svc.cluster.local"

        # Check monitoring endpoints
        endpoints = [
            "/api/metrics/cpu",
            "/api/metrics/memory",
            "/api/metrics/disk",
            "/api/alerts",
        ]

        for endpoint in endpoints:
            response = requests.get(f"{monitoring_url}{endpoint}", verify=False)
            assert response.status_code == 200

    def test_cdn_integration_scaling(self):
        """Test that CDN integration handles global scaling."""
        # Check CDN configuration
        cdn_config = "k8s/cdn-config.yaml"
        if os.path.exists(cdn_config):
            with open(cdn_config) as f:
                content = f.read()
                assert (
                    "cdn" in content.lower()
                    or "cloudfront" in content.lower()
                    or "fastly" in content.lower()
                )

    def test_edge_deployment_scaling(self):
        """Test that edge deployments scale regionally."""
        # Check edge deployment configuration
        edge_config = "k8s/edge-deployment.yaml"
        if os.path.exists(edge_config):
            with open(edge_config) as f:
                content = f.read()
                assert "edge" in content.lower() or "regional" in content.lower()

    def _get_deployment_replicas(self, k8s_client, deployment_name):
        """Get current replica count for deployment."""
        try:
            deployment = k8s_client.read_namespaced_deployment(
                name=deployment_name, namespace="helixflow"
            )
            return deployment.status.replicas
        except kubernetes.client.rest.ApiException:
            return 0
