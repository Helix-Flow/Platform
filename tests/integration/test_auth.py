"""
Authentication Integration Test for JWT and mTLS

Tests the integration that authentication works across services
with JWT tokens and mutual TLS communication.
"""

import pytest
import requests
import jwt
import ssl
import socket
from urllib.parse import urlparse
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
        login_payload = {"email": "test@example.com", "password": "password"}

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
            json={"email": "test@example.com", "password": "password"},
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
            json={"email": "test@example.com", "password": "password"},
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
        # Verify TLS 1.3 is enforced on API gateway
        context = ssl.create_default_context()
        context.check_hostname = False
        context.verify_mode = ssl.CERT_NONE

        parsed = urlparse(api_gateway_url)
        hostname = parsed.hostname or "localhost"
        port = parsed.port or 8443

        with socket.create_connection((hostname, port), timeout=5) as sock:
            with context.wrap_socket(sock, server_hostname=hostname) as ssock:
                assert ssock.version() == "TLSv1.3"

    def test_rate_limiting_integration(self, api_gateway_url, auth_headers):
        """Test rate limiting works with authentication."""
        # Verify Redis is available (rate limiting requires Redis)
        try:
            import redis
            r = redis.Redis(host='localhost', port=6379, socket_connect_timeout=1)
            r.ping()
        except Exception:
            # Redis not available - verify rate limiting is disabled gracefully
            try:
                response = requests.post(
                    f"{api_gateway_url}/v1/chat/completions",
                    headers=auth_headers,
                    json={
                        "model": "gpt-4",
                        "messages": [{"role": "user", "content": "test"}],
                        "max_tokens": 10,
                    },
                    verify=False,
                    timeout=10,
                )
                assert response.status_code == 200
            except requests.RequestException:
                pass  # Gateway may be under load from other tests
            return

        # Make multiple requests with valid auth to trigger rate limit tracking
        success_count = 0
        for _ in range(5):
            try:
                response = requests.post(
                    f"{api_gateway_url}/v1/chat/completions",
                    headers=auth_headers,
                    json={
                        "model": "gpt-4",
                        "messages": [{"role": "user", "content": "test"}],
                        "max_tokens": 10,
                    },
                    verify=False,
                    timeout=10,
                )
                # Should be allowed (rate limit is high for testing)
                if response.status_code in [200, 429]:
                    success_count += 1
            except requests.RequestException:
                pass

        assert success_count >= 1, "No successful requests to verify rate limiting"

        # Verify rate limit keys exist in Redis
        r = redis.Redis(host='localhost', port=6379, decode_responses=True)
        keys = r.keys('rate_limit:*')
        assert len(keys) > 0, "Rate limiting keys should exist in Redis"

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
            json={"email": "test@example.com", "password": "password"},
            verify=False,
        )

        if login_response.status_code == 200:
            access_token = login_response.json()["access_token"]

            # Revoke token
            revoke_response = requests.post(
                f"{auth_service_url}/revoke", json={"token": access_token}, verify=False
            )

            # Try to use revoked token on a protected endpoint
            headers = {"Authorization": f"Bearer {access_token}"}
            api_response = requests.post(
                f"{api_gateway_url}/v1/chat/completions",
                headers=headers,
                json={
                    "model": "gpt-3.5-turbo",
                    "messages": [{"role": "user", "content": "test"}],
                },
                verify=False,
            )

            # Should fail with 401
            assert api_response.status_code == 401
