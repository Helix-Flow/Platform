"""
Infrastructure Contract Test for Kubernetes Deployments

Tests the contract that Kubernetes infrastructure is properly deployed
and configured for HelixFlow platform components.
"""

import pytest
from kubernetes import client, config
from kubernetes.client.rest import ApiException


class TestInfrastructureContract:
    """Test suite for infrastructure deployment contracts."""

    @pytest.fixture(scope="class")
    def k8s_client(self):
        """Initialize Kubernetes client."""
        try:
            config.load_kube_config()
            return client.CoreV1Api()
        except Exception as e:
            pytest.skip(f"Kubernetes config not available: {e}")

    def test_api_gateway_deployment_exists(self, k8s_client):
        """Test that api-gateway deployment exists and is healthy."""
        try:
            deployment = k8s_client.read_namespaced_deployment(
                name="api-gateway", namespace="helixflow"
            )
            assert deployment.status.replicas == deployment.status.ready_replicas
            assert deployment.status.ready_replicas > 0
        except ApiException as e:
            if e.status == 404:
                pytest.fail("api-gateway deployment not found")
            raise

    def test_inference_pool_deployment_exists(self, k8s_client):
        """Test that inference-pool deployment exists and is healthy."""
        try:
            deployment = k8s_client.read_namespaced_deployment(
                name="inference-pool", namespace="helixflow"
            )
            assert deployment.status.replicas == deployment.status.ready_replicas
            assert deployment.status.ready_replicas > 0
        except ApiException as e:
            if e.status == 404:
                pytest.fail("inference-pool deployment not found")
            raise

    def test_auth_service_deployment_exists(self, k8s_client):
        """Test that auth-service deployment exists and is healthy."""
        try:
            deployment = k8s_client.read_namespaced_deployment(
                name="auth-service", namespace="helixflow"
            )
            assert deployment.status.replicas == deployment.status.ready_replicas
            assert deployment.status.ready_replicas > 0
        except ApiException as e:
            if e.status == 404:
                pytest.fail("auth-service deployment not found")
            raise

    def test_monitoring_deployment_exists(self, k8s_client):
        """Test that monitoring deployment exists and is healthy."""
        try:
            deployment = k8s_client.read_namespaced_deployment(
                name="monitoring", namespace="helixflow"
            )
            assert deployment.status.replicas == deployment.status.ready_replicas
            assert deployment.status.ready_replicas > 0
        except ApiException as e:
            if e.status == 404:
                pytest.fail("monitoring deployment not found")
            raise

    def test_istio_service_mesh_enabled(self, k8s_client):
        """Test that Istio service mesh is enabled on namespace."""
        try:
            namespace = k8s_client.read_namespace(name="helixflow")
            labels = namespace.metadata.labels or {}
            assert "istio-injection" in labels
            assert labels["istio-injection"] == "enabled"
        except ApiException as e:
            if e.status == 404:
                pytest.fail("helixflow namespace not found")
            raise

    def test_gpu_nodes_available(self, k8s_client):
        """Test that GPU nodes are available in the cluster."""
        nodes = k8s_client.list_node()
        gpu_nodes = []
        for node in nodes.items:
            capacity = node.status.capacity
            if "nvidia.com/gpu" in capacity or "amd.com/gpu" in capacity:
                gpu_nodes.append(node)

        assert len(gpu_nodes) > 0, "No GPU nodes found in cluster"

    def test_persistent_volumes_configured(self, k8s_client):
        """Test that persistent volumes are configured for databases."""
        pv_list = k8s_client.list_persistent_volume()
        pv_names = [pv.metadata.name for pv in pv_list.items]

        required_pvs = ["postgresql-pv", "redis-pv", "neo4j-pv", "qdrant-pv"]
        for pv_name in required_pvs:
            assert pv_name in pv_names, f"Persistent volume {pv_name} not found"
