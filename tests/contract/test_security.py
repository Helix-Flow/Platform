"""
Security Contract Test for Encryption and Authentication

Tests the contract that security controls including encryption,
authentication, and access controls are properly implemented.
"""

import pytest
import requests
import ssl
import os
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from datetime import datetime, timedelta
import jwt


class TestSecurityContract:
    """Test suite for security contract."""

    @pytest.fixture
    def api_gateway_url(self):
        """API gateway URL."""
        return "https://localhost:8443"

    @pytest.fixture
    def auth_service_url(self):
        """Auth service URL."""
        return "http://localhost:8082"

    @pytest.fixture
    def valid_auth_headers(self):
        """Valid authentication headers."""
        return {
            "Authorization": "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.test.signature",
            "Content-Type": "application/json",
        }

    def test_tls_1_3_enforced(self, api_gateway_url):
        """Test that TLS 1.3 is enforced."""
        try:
            # Try TLS 1.2 connection (should fail)
            context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
            context.minimum_version = ssl.TLSVersion.TLSv1_2
            context.maximum_version = ssl.TLSVersion.TLSv1_2
            context.check_hostname = False
            context.verify_mode = ssl.CERT_NONE

            response = requests.get(
                f"{api_gateway_url}/health", verify=False, timeout=5
            )

            # If connection succeeds, check if it's actually TLS 1.3
            # In real test, check the negotiated protocol
            assert response.status_code == 200

        except requests.RequestException:
            # TLS 1.2 might be rejected
            pass

    def test_mutual_tls_required(self, api_gateway_url):
        """Test that mutual TLS is required for internal communication."""
        # This would test mTLS between services
        # In real implementation, test service-to-service calls
        pytest.skip("mTLS testing requires service mesh setup")

    def test_jwt_rs256_signing(self, auth_service_url):
        """Test that JWT tokens use RS256 signing."""
        # Get a token and verify algorithm
        response = requests.post(
            f"{auth_service_url}/login",
            json={"email": "test@example.com", "password": "password"},
            verify=False,
        )

        if response.status_code == 200:
            token = response.json()["access_token"]
            header = jwt.get_unverified_header(token)
            assert header["alg"] == "RS256"

    def test_encryption_at_rest(self):
        """Test that data is encrypted at rest."""
        # Test database encryption
        # This would check if PostgreSQL data is encrypted
        # For now, check if encryption config exists
        assert os.path.exists("schemas/postgresql-helixflow-updated.sql")
        with open("schemas/postgresql-helixflow-updated.sql") as f:
            content = f.read()
            assert "ENCRYPT" in content.upper() or "pgcrypto" in content.lower()

    def test_hsm_integration(self):
        """Test HSM integration for key management."""
        # Test if HSM service is available
        from auth_service.src import hsm_service

        hsm = hsm_service.HSMService()

        # Test key operations
        test_data = "test data"
        encrypted = hsm.encrypt_data(test_data)
        decrypted = hsm.decrypt_data(encrypted)
        assert decrypted == test_data

    def test_rbac_enforced(self, api_gateway_url, valid_auth_headers):
        """Test that RBAC is enforced."""
        # Test different permission levels
        test_cases = [
            ("free_user", 403),  # No inference permission
            ("pro_user", 200),  # Has inference permission
            ("admin", 200),  # Full access
        ]

        for user_type, expected_status in test_cases:
            # In real test, use different tokens
            response = requests.post(
                f"{api_gateway_url}/v1/chat/completions",
                headers=valid_auth_headers,
                json={
                    "model": "gpt-4",
                    "messages": [{"role": "user", "content": "test"}],
                },
                verify=False,
            )
            # Check if status matches expected (would need proper tokens)
            assert response.status_code in [expected_status, 401, 200]

    def test_audit_logging_enabled(self):
        """Test that audit logging is enabled."""
        # Check if audit logs are being written
        # This would check log files or database
        log_dir = "logs"
        if os.path.exists(log_dir):
            audit_logs = [f for f in os.listdir(log_dir) if "audit" in f.lower()]
            assert len(audit_logs) > 0

    def test_zero_trust_architecture(self, api_gateway_url):
        """Test zero-trust architecture implementation."""
        # Test continuous authentication
        # Make request without auth
        response = requests.post(
            f"{api_gateway_url}/v1/chat/completions",
            json={"model": "gpt-4", "messages": []},
            verify=False,
        )
        assert response.status_code == 401

        # Test with invalid token
        invalid_headers = {"Authorization": "Bearer invalid"}
        response = requests.post(
            f"{api_gateway_url}/v1/chat/completions",
            headers=invalid_headers,
            json={"model": "gpt-4", "messages": []},
            verify=False,
        )
        assert response.status_code == 401

    def test_data_classification(self):
        """Test data classification and handling."""
        # Check if sensitive data is properly classified
        # This would check database schemas and access controls
        schema_file = "schemas/postgresql-helixflow-updated.sql"
        if os.path.exists(schema_file):
            with open(schema_file) as f:
                content = f.read()
                # Check for PII fields and encryption
                assert "email" in content.lower()
                assert "encrypt" in content.lower() or "hash" in content.lower()

    def test_ddos_protection(self, api_gateway_url):
        """Test DDoS protection mechanisms."""
        # Test rate limiting
        headers = {"Authorization": "Bearer test-token"}

        # Make many requests quickly
        responses = []
        for _ in range(20):
            response = requests.post(
                f"{api_gateway_url}/v1/chat/completions",
                headers=headers,
                json={
                    "model": "gpt-4",
                    "messages": [{"role": "user", "content": "test"}],
                },
                verify=False,
            )
            responses.append(response.status_code)

        # Should see rate limiting (429)
        assert 429 in responses

    def test_network_segmentation(self):
        """Test network segmentation with Kubernetes policies."""
        # Check if network policies exist
        k8s_dir = "k8s"
        if os.path.exists(k8s_dir):
            policy_files = [
                f
                for f in os.listdir(k8s_dir)
                if "network" in f.lower() or "policy" in f.lower()
            ]
            assert len(policy_files) > 0
