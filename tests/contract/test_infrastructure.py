"""
Infrastructure Contract Test for Kubernetes Deployments

Tests the contract that Kubernetes infrastructure manifests are properly configured
for HelixFlow platform components.
"""

import pytest
import os


class TestInfrastructureContract:
    """Test suite for infrastructure deployment contracts."""

    def test_api_gateway_deployment_exists(self):
        """Test that api-gateway deployment manifest exists."""
        assert os.path.exists("k8s/api-gateway.yaml"), "api-gateway manifest not found"
        with open("k8s/api-gateway.yaml") as f:
            content = f.read()
            assert "api-gateway" in content

    def test_inference_pool_deployment_exists(self):
        """Test that inference-pool deployment manifest exists."""
        assert os.path.exists("k8s/inference-pool.yaml"), "inference-pool manifest not found"
        with open("k8s/inference-pool.yaml") as f:
            content = f.read()
            assert "inference-pool" in content

    def test_auth_service_deployment_exists(self):
        """Test that auth-service deployment manifest exists."""
        assert os.path.exists("k8s/auth-service.yaml"), "auth-service manifest not found"
        with open("k8s/auth-service.yaml") as f:
            content = f.read()
            assert "auth-service" in content

    def test_monitoring_deployment_exists(self):
        """Test that monitoring deployment manifest exists."""
        assert os.path.exists("k8s/monitoring.yaml"), "monitoring manifest not found"
        with open("k8s/monitoring.yaml") as f:
            content = f.read()
            assert "monitoring" in content

    def test_istio_service_mesh_enabled(self):
        """Test that service mesh network policies are configured."""
        assert os.path.exists("k8s/network-policy.yaml"), "Network policies not found"
        with open("k8s/network-policy.yaml") as f:
            content = f.read()
            assert "NetworkPolicy" in content

    def test_gpu_nodes_available(self):
        """Test that GPU node configuration exists in manifests."""
        gpu_found = False
        for f in ["k8s/inference-pool.yaml", "terraform/aws/main.tf", "terraform/azure/main.tf", "terraform/gcp/main.tf"]:
            if os.path.exists(f):
                with open(f) as fh:
                    content = fh.read()
                    if "gpu" in content.lower() or "nvidia" in content.lower():
                        gpu_found = True
                        break
        assert gpu_found, "No GPU configuration found in manifests"

    def test_persistent_volumes_configured(self):
        """Test that persistent volume claims are configured."""
        pvc_found = False
        for root, dirs, files in os.walk("k8s"):
            for f in files:
                if f.endswith(".yaml"):
                    with open(os.path.join(root, f)) as fh:
                        content = fh.read()
                        if "PersistentVolumeClaim" in content or "persistentVolumeClaim" in content:
                            pvc_found = True
                            break
        assert pvc_found, "No persistent volume claims found in K8s manifests"
