"""
Test configuration for HelixFlow platform
"""

import pytest
import json
import os
from unittest.mock import Mock, patch

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
        "auth_service_url": "https://localhost:8081",
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
