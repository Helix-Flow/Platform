"""
Service Mesh Integration Test for Istio Communication

Tests the integration that services can communicate through Istio service mesh
with proper mTLS, traffic policies, and observability.
"""

import pytest
import requests
from kubernetes import client, config
from kubernetes.client.rest import ApiException


class TestServiceMeshIntegration:
    """Test suite for Istio service mesh integration."""

    @pytest.fixture(scope="class")
    def k8s_client(self):
        """Initialize Kubernetes client."""
        try:
            config.load_kube_config()
            return client.CoreV1Api()
        except Exception as e:
            pytest.skip(f"Kubernetes config not available: {e}")

    def test_istio_sidecar_injected(self, k8s_client):
        """Test that Istio sidecars are injected into pods."""
        pods = k8s_client.list_namespaced_pod(namespace="helixflow")
        for pod in pods.items:
            containers = pod.spec.containers
            container_names = [c.name for c in containers]
            assert "istio-proxy" in container_names, (
                f"Pod {pod.metadata.name} missing istio-proxy sidecar"
            )

    def test_mutual_tls_enabled(self, k8s_client):
        """Test that mTLS is enabled for service communication."""
        # Check PeerAuthentication policy
        try:
            custom_api = client.CustomObjectsApi()
            peer_auth = custom_api.get_namespaced_custom_object(
                group="security.istio.io",
                version="v1beta1",
                namespace="helixflow",
                plural="peerauthentications",
                name="default",
            )
            assert peer_auth["spec"]["mtls"]["mode"] == "STRICT"
        except ApiException as e:
            if e.status == 404:
                pytest.fail("PeerAuthentication policy not found")
            raise

    def test_service_to_service_communication(self):
        """Test that services can communicate with each other through service mesh."""
        # Test api-gateway to auth-service communication
        try:
            response = requests.get(
                "https://localhost:8443/health", timeout=10, verify=False
            )
            assert response.status_code == 200
        except requests.RequestException:
            pytest.fail("Cannot reach api-gateway health endpoint")

        # Test auth-service to inference-pool communication
        try:
            response = requests.get(
                "https://localhost:8081/health", timeout=10, verify=False
            )
            assert response.status_code == 200
        except requests.RequestException:
            pytest.fail("Cannot reach auth-service health endpoint")

    def test_traffic_policies_applied(self):
        """Test that Istio traffic policies are applied."""
        # Check VirtualService and DestinationRule
        custom_api = client.CustomObjectsApi()

        # Check VirtualService for api-gateway
        try:
            vs = custom_api.get_namespaced_custom_object(
                group="networking.istio.io",
                version="v1beta1",
                namespace="helixflow",
                plural="virtualservices",
                name="api-gateway",
            )
            assert "http" in vs["spec"]
        except ApiException as e:
            if e.status == 404:
                pytest.fail("VirtualService for api-gateway not found")
            raise

        # Check DestinationRule for mTLS
        try:
            dr = custom_api.get_namespaced_custom_object(
                group="networking.istio.io",
                version="v1beta1",
                namespace="helixflow",
                plural="destinationrules",
                name="api-gateway",
            )
            traffic_policy = dr["spec"]["trafficPolicy"]
            assert "tls" in traffic_policy
            assert traffic_policy["tls"]["mode"] == "ISTIO_MUTUAL"
        except ApiException as e:
            if e.status == 404:
                pytest.fail("DestinationRule for api-gateway not found")
            raise

    def test_observability_enabled(self):
        """Test that Istio observability features are enabled."""
        # Check if Prometheus can scrape Istio metrics
        try:
            response = requests.get(
                "http://localhost:8083/api/v1/query?query=istio_requests_total",
                timeout=10,
            )
            assert response.status_code == 200
            data = response.json()
            assert "data" in data
        except requests.RequestException:
            pytest.fail("Cannot query Istio metrics from Prometheus")

    def test_circuit_breaker_configured(self):
        """Test that circuit breaker is configured for resilience."""
        custom_api = client.CustomObjectsApi()

        try:
            dr = custom_api.get_namespaced_custom_object(
                group="networking.istio.io",
                version="v1beta1",
                namespace="helixflow",
                plural="destinationrules",
                name="inference-pool",
            )
            traffic_policy = dr["spec"]["trafficPolicy"]
            assert "connectionPool" in traffic_policy
            pool = traffic_policy["connectionPool"]
            assert "tcp" in pool
            assert "maxConnections" in pool["tcp"]
        except ApiException as e:
            if e.status == 404:
                pytest.fail("DestinationRule with circuit breaker not found")
            raise
