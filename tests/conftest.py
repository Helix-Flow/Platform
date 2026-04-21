"""
Test configuration for HelixFlow platform
"""

import pytest
import json
import os
import requests
from unittest.mock import Mock, patch

# Suppress TLS verification warnings for self-signed certs
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

def _get_auth_token():
    """Get a real auth token from the auth service."""
    import time
    for attempt in range(5):
        try:
            response = requests.post(
                "http://localhost:8082/login",
                json={"email": "testuser", "password": "password"},
                timeout=5,
            )
            if response.status_code == 200:
                return response.json().get("access_token", "")
        except Exception:
            pass
        time.sleep(1)
    return None

def pytest_configure(config):
    """Configure pytest with custom markers."""
    config.addinivalue_line(
        "markers", "integration: mark test as integration test"
    )
    config.addinivalue_line(
        "markers", "contract: mark test as contract test"
    )
    config.addinivalue_line(
        "markers", "security: mark test as security test"
    )
    config.addinivalue_line(
        "markers", "performance: mark test as performance test"
    )

@pytest.fixture
def test_config():
    """Test configuration."""
    return {
        "api_gateway_url": "https://localhost:8443",
        "auth_service_url": "http://localhost:8082",
        "inference_pool_url": "http://localhost:50051",
        "monitoring_url": "http://localhost:8083",
        "test_timeout": 30,
        "max_retries": 3
    }

@pytest.fixture
def sample_chat_request():
    """Sample chat completion request."""
    return {
        "model": "gpt-3.5-turbo",
        "messages": [
            {"role": "user", "content": "Hello, world!"}
        ],
        "max_tokens": 100,
        "temperature": 0.7
    }

@pytest.fixture
def sample_auth_credentials():
    """Sample authentication credentials."""
    return {
        "username": "testuser",
        "password": "testpass123",
        "email": "test@example.com"
    }

@pytest.fixture
def mock_response():
    """Mock API response."""
    return {
        "id": "test-123",
        "status": "success",
        "data": {"message": "Test response"}
    }


@pytest.fixture(scope="session")
def real_auth_token():
    """Get a real JWT access token from the auth service."""
    token = _get_auth_token()
    if not token:
        pytest.skip("Auth service not available - cannot obtain real token")
    return token


@pytest.fixture
def auth_headers(real_auth_token):
    """Authentication headers with a real token."""
    return {
        "Authorization": f"Bearer {real_auth_token}",
        "Content-Type": "application/json",
    }


@pytest.fixture
def api_base_url():
    """API gateway base URL."""
    return "https://localhost:8443"


@pytest.fixture
def auth_service_url():
    """Auth service URL."""
    return "http://localhost:8082"


@pytest.fixture
def monitoring_url():
    """Monitoring service URL."""
    return "http://localhost:8083"
