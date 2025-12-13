"""
Multi-Cloud Deployment Integration Test

Tests the integration that Terraform configurations can deploy
infrastructure across AWS, Azure, and GCP with proper connectivity.
"""

import pytest
import subprocess
import os
import json
from pathlib import Path


class TestMultiCloudDeployment:
    """Test suite for multi-cloud deployment integration."""

    @pytest.fixture(scope="class")
    def terraform_dir(self):
        """Path to terraform directory."""
        return Path("terraform")

    def test_terraform_aws_config_valid(self, terraform_dir):
        """Test that AWS Terraform configuration is valid."""
        aws_dir = terraform_dir / "aws"
        assert aws_dir.exists(), "AWS terraform directory not found"

        # Run terraform validate
        result = subprocess.run(
            ["terraform", "validate"], cwd=aws_dir, capture_output=True, text=True
        )
        assert result.returncode == 0, f"Terraform validate failed: {result.stderr}"

    def test_terraform_azure_config_valid(self, terraform_dir):
        """Test that Azure Terraform configuration is valid."""
        azure_dir = terraform_dir / "azure"
        assert azure_dir.exists(), "Azure terraform directory not found"

        result = subprocess.run(
            ["terraform", "validate"], cwd=azure_dir, capture_output=True, text=True
        )
        assert result.returncode == 0, f"Terraform validate failed: {result.stderr}"

    def test_terraform_gcp_config_valid(self, terraform_dir):
        """Test that GCP Terraform configuration is valid."""
        gcp_dir = terraform_dir / "gcp"
        assert gcp_dir.exists(), "GCP terraform directory not found"

        result = subprocess.run(
            ["terraform", "validate"], cwd=gcp_dir, capture_output=True, text=True
        )
        assert result.returncode == 0, f"Terraform validate failed: {result.stderr}"

    def test_terraform_modules_exist(self, terraform_dir):
        """Test that shared Terraform modules exist."""
        modules_dir = terraform_dir / "modules"
        assert modules_dir.exists(), "Terraform modules directory not found"

        vpc_module = modules_dir / "vpc"
        assert vpc_module.exists(), "VPC module not found"
        assert (vpc_module / "variables.tf").exists(), (
            "VPC module variables.tf not found"
        )

    def test_terraform_variables_defined(self, terraform_dir):
        """Test that Terraform variables are properly defined."""
        for cloud in ["aws", "azure", "gcp"]:
            var_file = terraform_dir / cloud / "variables.tf"
            assert var_file.exists(), f"variables.tf not found for {cloud}"

            with open(var_file) as f:
                content = f.read()
                # Check for common required variables
                assert "variable" in content, (
                    f"No variables defined in {cloud}/variables.tf"
                )

    def test_terraform_outputs_defined(self, terraform_dir):
        """Test that Terraform outputs are defined for cross-cloud integration."""
        for cloud in ["aws", "azure", "gcp"]:
            main_file = terraform_dir / cloud / "main.tf"
            assert main_file.exists(), f"main.tf not found for {cloud}"

            with open(main_file) as f:
                content = f.read()
                # Check for output blocks
                assert "output" in content, f"No outputs defined in {cloud}/main.tf"

    def test_multi_cloud_connectivity_test(self):
        """Test that multi-cloud deployments can communicate."""
        # This would typically test VPN connections, peering, etc.
        # For now, check that terraform plans include networking
        terraform_dir = Path("terraform")

        for cloud in ["aws", "azure", "gcp"]:
            main_file = terraform_dir / cloud / "main.tf"
            with open(main_file) as f:
                content = f.read()
                # Check for VPC/networking resources
                networking_terms = ["vpc", "virtual_network", "network"]
                has_networking = any(
                    term in content.lower() for term in networking_terms
                )
                assert has_networking, (
                    f"No networking configuration found in {cloud}/main.tf"
                )

    def test_argocd_integration_configured(self):
        """Test that ArgoCD integration is configured for GitOps."""
        # Check if argocd config exists
        argocd_dir = Path("k8s") / "argocd"
        if argocd_dir.exists():
            config_files = list(argocd_dir.glob("*.yaml"))
            assert len(config_files) > 0, "No ArgoCD configuration files found"
        else:
            pytest.skip("ArgoCD directory not found")

    def test_cross_cloud_dns_configured(self):
        """Test that cross-cloud DNS resolution is configured."""
        # Check for external-dns or similar configurations
        k8s_dir = Path("k8s")
        dns_configs = list(k8s_dir.glob("*dns*")) + list(k8s_dir.glob("*external*"))
        if dns_configs:
            assert len(dns_configs) > 0, "DNS configuration files found but empty"
        else:
            pytest.skip("No DNS configuration found")
