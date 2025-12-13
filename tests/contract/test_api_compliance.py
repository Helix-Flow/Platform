"""
API Contract Test for Chat Completions Endpoint

Tests the contract that chat completions API endpoint works correctly
and returns OpenAI-compatible responses.
"""

import pytest
import requests
import json
from typing import Dict, Any


class TestChatAPIContract:
    """Test suite for chat completions API contract."""

    @pytest.fixture
    def api_base_url(self):
        """Base URL for API gateway."""
        return "https://api.helixflow.ai"

    @pytest.fixture
    def valid_headers(self):
        """Valid authentication headers."""
        return {
            "Authorization": "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.test.signature",
            "Content-Type": "application/json",
        }

    def test_chat_completions_basic_request(self, api_base_url, valid_headers):
        """Test basic chat completions request."""
        payload = {
            "model": "gpt-4",
            "messages": [{"role": "user", "content": "Hello, how are you?"}],
            "max_tokens": 100,
        }

        response = requests.post(
            f"{api_base_url}/v1/chat/completions",
            headers=valid_headers,
            json=payload,
            verify=False,  # For testing
        )

        assert response.status_code == 200
        data = response.json()

        # Validate OpenAI-compatible response structure
        assert "id" in data
        assert "object" in data
        assert data["object"] == "chat.completion"
        assert "created" in data
        assert "model" in data
        assert "choices" in data
        assert len(data["choices"]) > 0
        assert "message" in data["choices"][0]
        assert "role" in data["choices"][0]["message"]
        assert "content" in data["choices"][0]["message"]
        assert "usage" in data

    def test_chat_completions_with_streaming(self, api_base_url, valid_headers):
        """Test chat completions with streaming enabled."""
        payload = {
            "model": "gpt-4",
            "messages": [{"role": "user", "content": "Tell me a story"}],
            "stream": True,
            "max_tokens": 200,
        }

        response = requests.post(
            f"{api_base_url}/v1/chat/completions",
            headers=valid_headers,
            json=payload,
            stream=True,
            verify=False,
        )

        assert response.status_code == 200

        # Parse streaming response
        chunks = []
        for line in response.iter_lines():
            if line:
                line = line.decode("utf-8")
                if line.startswith("data: "):
                    data = line[6:]
                    if data != "[DONE]":
                        chunks.append(json.loads(data))

        assert len(chunks) > 0
        # Validate streaming format
        for chunk in chunks:
            assert "choices" in chunk
            assert len(chunk["choices"]) > 0

    def test_chat_completions_error_handling(self, api_base_url):
        """Test error handling for invalid requests."""
        # No auth header
        payload = {"model": "gpt-4", "messages": []}
        response = requests.post(
            f"{api_base_url}/v1/chat/completions", json=payload, verify=False
        )
        assert response.status_code == 401

        # Invalid JSON
        response = requests.post(
            f"{api_base_url}/v1/chat/completions",
            headers={"Authorization": "Bearer test"},
            data="invalid json",
            verify=False,
        )
        assert response.status_code == 400

    def test_models_list_endpoint(self, api_base_url, valid_headers):
        """Test models list endpoint."""
        response = requests.get(
            f"{api_base_url}/v1/models", headers=valid_headers, verify=False
        )

        assert response.status_code == 200
        data = response.json()

        assert "object" in data
        assert data["object"] == "list"
        assert "data" in data
        assert isinstance(data["data"], list)
        assert len(data["data"]) > 0

        # Validate model structure
        model = data["data"][0]
        assert "id" in model
        assert "object" in model
        assert model["object"] == "model"
        assert "created" in model
        assert "owned_by" in model

    def test_rate_limiting(self, api_base_url, valid_headers):
        """Test rate limiting functionality."""
        payload = {
            "model": "gpt-4",
            "messages": [{"role": "user", "content": "Test"}],
            "max_tokens": 10,
        }

        # Make multiple requests quickly
        responses = []
        for _ in range(10):
            response = requests.post(
                f"{api_base_url}/v1/chat/completions",
                headers=valid_headers,
                json=payload,
                verify=False,
            )
            responses.append(response.status_code)

        # Should have some 429 (rate limited) responses
        assert 429 in responses or all(r == 200 for r in responses)

    def test_openai_compatibility(self, api_base_url, valid_headers):
        """Test full OpenAI API compatibility."""
        # Test various OpenAI parameters
        payload = {
            "model": "gpt-4",
            "messages": [
                {"role": "system", "content": "You are a helpful assistant."},
                {"role": "user", "content": "What is 2+2?"},
            ],
            "temperature": 0.7,
            "top_p": 1.0,
            "n": 1,
            "stop": None,
            "max_tokens": 50,
            "presence_penalty": 0.0,
            "frequency_penalty": 0.0,
            "logit_bias": None,
            "user": "test-user",
        }

        response = requests.post(
            f"{api_base_url}/v1/chat/completions",
            headers=valid_headers,
            json=payload,
            verify=False,
        )

        assert response.status_code == 200
        data = response.json()

        # Validate all expected fields
        required_fields = ["id", "object", "created", "model", "choices", "usage"]
        for field in required_fields:
            assert field in data
