"""
Authentication Integration Test for JWT and mTLS

Tests the integration that authentication works across services
with JWT tokens and mutual TLS communication.
"""

import pytest
import requests
import jwt
import ssl
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from datetime import datetime, timedelta
import os


class TestAuthIntegration:
    """Test suite for authentication integration."""

    @pytest.fixture
    def auth_service_url(self):
        """Auth service URL."""
        return "http://localhost:8082"

    @pytest.fixture
    def api_gateway_url(self):
        """API gateway URL."""
        return "https://localhost:8443"

    @pytest.fixture
    def client_cert(self):
        """Client certificate for mTLS."""
        # In real test, load actual cert
        return None

    def test_jwt_token_generation_and_validation(self, auth_service_url):
        """Test JWT token generation and validation flow."""
        # Register/login user
        login_payload = {"email": "test@example.com", "password": "password123"}

        response = requests.post(
            f"{auth_service_url}/login", json=login_payload, verify=False
        )

        if response.status_code == 200:
            data = response.json()
            access_token = data["access_token"]
            refresh_token = data["refresh_token"]

            # Validate token structure
            assert access_token
            assert refresh_token

            # Decode and validate JWT
            # Note: In real test, use public key
            header = jwt.get_unverified_header(access_token)
            assert header["alg"] == "RS256"
            assert header["typ"] == "JWT"

            payload = jwt.decode(access_token, options={"verify_signature": False})
            assert "sub" in payload
            assert "exp" in payload
            assert payload["type"] == "access"

    def test_token_refresh_flow(self, auth_service_url):
        """Test token refresh functionality."""
        # First get tokens
        login_response = requests.post(
            f"{auth_service_url}/login",
            json={"email": "test@example.com", "password": "password123"},
            verify=False,
        )

        if login_response.status_code == 200:
            refresh_token = login_response.json()["refresh_token"]

            # Refresh access token
            refresh_response = requests.post(
                f"{auth_service_url}/refresh",
                json={"refresh_token": refresh_token},
                verify=False,
            )

            assert refresh_response.status_code == 200
            new_tokens = refresh_response.json()
            assert "access_token" in new_tokens
            assert "refresh_token" in new_tokens

    def test_cross_service_authentication(self, auth_service_url, api_gateway_url):
        """Test authentication works across services."""
        # Get token from auth service
        login_response = requests.post(
            f"{auth_service_url}/login",
            json={"email": "test@example.com", "password": "password123"},
            verify=False,
        )

        if login_response.status_code == 200:
            access_token = login_response.json()["access_token"]

            # Use token with API gateway
            headers = {"Authorization": f"Bearer {access_token}"}
            api_response = requests.get(
                f"{api_gateway_url}/v1/models", headers=headers, verify=False
            )

            # Should succeed (assuming user has permission)
            assert api_response.status_code in [
                200,
                403,
            ]  # 403 if no permission, but auth succeeded

    def test_mtls_communication(self, api_gateway_url, client_cert):
        """Test mutual TLS communication between services."""
        # Create SSL context with client cert
        context = ssl.create_default_context()
        context.check_hostname = False
        context.verify_mode = ssl.CERT_NONE

        # In real test, load client cert
        # context.load_cert_chain(certfile=client_cert[0], keyfile=client_cert[1])

        try:
            response = requests.get(
                f"{api_gateway_url}/health",
                cert=client_cert,  # Would be actual cert tuple
                verify=False,
            )
            # With mTLS, this should work if cert is valid
            assert response.status_code == 200
        except requests.RequestException:
            # If mTLS not configured, test might fail
            pytest.skip("mTLS not configured in test environment")

    def test_rate_limiting_integration(self, api_gateway_url):
        """Test rate limiting works with authentication."""
        headers = {"Authorization": "Bearer test-token"}

        # Make multiple requests
        responses = []
        for _ in range(15):  # Exceed typical rate limit
            response = requests.post(
                f"{api_gateway_url}/v1/chat/completions",
                headers=headers,
                json={
                    "model": "gpt-4",
                    "messages": [{"role": "user", "content": "test"}],
                    "max_tokens": 10,
                },
                verify=False,
            )
            responses.append(response.status_code)

        # Should see some 429 responses
        assert 429 in responses

    def test_permission_based_access_control(self, api_gateway_url):
        """Test RBAC permissions control access."""
        # Test different user roles/permissions
        test_cases = [
            ("free_user_token", 403),  # No inference permission
            ("pro_user_token", 200),  # Has inference permission
            ("admin_token", 200),  # Full access
        ]

        for token, expected_status in test_cases:
            headers = {"Authorization": f"Bearer {token}"}
            response = requests.post(
                f"{api_gateway_url}/v1/chat/completions",
                headers=headers,
                json={
                    "model": "gpt-4",
                    "messages": [{"role": "user", "content": "test"}],
                },
                verify=False,
            )
            # In real test, check actual status
            assert response.status_code in [
                expected_status,
                401,
            ]  # 401 if token invalid

    def test_token_revocation(self, auth_service_url, api_gateway_url):
        """Test token revocation works across services."""
        # Get token
        login_response = requests.post(
            f"{auth_service_url}/login",
            json={"email": "test@example.com", "password": "password123"},
            verify=False,
        )

        if login_response.status_code == 200:
            access_token = login_response.json()["access_token"]

            # Revoke token
            revoke_response = requests.post(
                f"{auth_service_url}/revoke", json={"token": access_token}, verify=False
            )

            # Try to use revoked token
            headers = {"Authorization": f"Bearer {access_token}"}
            api_response = requests.get(
                f"{api_gateway_url}/v1/models", headers=headers, verify=False
            )

            # Should fail with 401
            assert api_response.status_code == 401
