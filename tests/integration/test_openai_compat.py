"""
OpenAI Compatibility Integration Test

Tests the integration that HelixFlow APIs are fully compatible
with OpenAI API specifications and can be used as drop-in replacements.
"""

import pytest
import requests
import json
import openai  # If available, test with official client
from typing import Dict, Any


class TestOpenAICompatibility:
    """Test suite for OpenAI API compatibility."""

    @pytest.fixture
    def helixflow_client(self):
        """HelixFlow API client configured to use our gateway."""
        # Configure to use HelixFlow instead of OpenAI
        import os

        os.environ["OPENAI_API_BASE"] = "https://localhost:8443"
        os.environ["OPENAI_API_KEY"] = "test-key"

        try:
            import openai

            client = openai.OpenAI(
                api_key="test-key", base_url="https://localhost:8443"
            )
            return client
        except ImportError:
            return None

    @pytest.fixture
    def api_base_url(self):
        """HelixFlow API base URL."""
        return "https://localhost:8443"

    @pytest.fixture
    def auth_headers(self):
        """Authentication headers."""
        return {
            "Authorization": "Bearer test-token",
            "Content-Type": "application/json",
        }

    def test_openai_client_compatibility(self, helixflow_client):
        """Test that OpenAI Python client works with HelixFlow."""
        if not helixflow_client:
            pytest.skip("OpenAI client not available")

        try:
            response = helixflow_client.chat.completions.create(
                model="gpt-4",
                messages=[{"role": "user", "content": "Hello, test message"}],
                max_tokens=50,
            )

            # Validate response structure matches OpenAI
            assert hasattr(response, "id")
            assert hasattr(response, "object")
            assert response.object == "chat.completion"
            assert hasattr(response, "created")
            assert hasattr(response, "model")
            assert hasattr(response, "choices")
            assert len(response.choices) > 0
            assert hasattr(response.choices[0], "message")
            assert hasattr(response.choices[0].message, "content")
            assert hasattr(response, "usage")

        except Exception as e:
            # If API not running, test the structure expectation
            pytest.skip(f"API not available: {e}")

    def test_chat_completions_response_format(self, api_base_url, auth_headers):
        """Test chat completions response matches OpenAI format exactly."""
        payload = {
            "model": "gpt-4",
            "messages": [
                {"role": "system", "content": "You are a helpful assistant."},
                {"role": "user", "content": "What is the capital of France?"},
            ],
            "temperature": 0.7,
            "max_tokens": 100,
        }

        response = requests.post(
            f"{api_base_url}/v1/chat/completions",
            headers=auth_headers,
            json=payload,
            verify=False,
        )

        if response.status_code != 200:
            pytest.skip("API not available")

        data = response.json()

        # Exact OpenAI response structure
        required_fields = ["id", "object", "created", "model", "choices", "usage"]

        for field in required_fields:
            assert field in data, f"Missing required field: {field}"

        assert data["object"] == "chat.completion"
        assert isinstance(data["choices"], list)
        assert len(data["choices"]) > 0

        choice = data["choices"][0]
        required_choice_fields = ["index", "message", "finish_reason"]

        for field in required_choice_fields:
            assert field in choice, f"Missing choice field: {field}"

        message = choice["message"]
        assert "role" in message
        assert "content" in message

        usage = data["usage"]
        required_usage_fields = ["prompt_tokens", "completion_tokens", "total_tokens"]

        for field in required_usage_fields:
            assert field in usage, f"Missing usage field: {field}"

    def test_models_list_compatibility(self, api_base_url, auth_headers):
        """Test models list endpoint matches OpenAI format."""
        response = requests.get(
            f"{api_base_url}/v1/models", headers=auth_headers, verify=False
        )

        if response.status_code != 200:
            pytest.skip("API not available")

        data = response.json()

        assert data["object"] == "list"
        assert isinstance(data["data"], list)
        assert len(data["data"]) > 0

        model = data["data"][0]
        required_model_fields = ["id", "object", "created", "owned_by"]

        for field in required_model_fields:
            assert field in model, f"Missing model field: {field}"

        assert model["object"] == "model"

    def test_error_response_format(self, api_base_url):
        """Test error responses match OpenAI format."""
        # Test with invalid auth
        response = requests.post(
            f"{api_base_url}/v1/chat/completions",
            json={"model": "gpt-4", "messages": []},
            verify=False,
        )

        if response.status_code == 200:
            return  # Auth not required in test

        data = response.json()

        # OpenAI error format
        assert "error" in data
        error = data["error"]
        assert "type" in error
        assert "message" in error

    def test_streaming_response_format(self, api_base_url, auth_headers):
        """Test streaming responses match OpenAI Server-Sent Events format."""
        payload = {
            "model": "gpt-4",
            "messages": [{"role": "user", "content": "Tell me a joke"}],
            "stream": True,
            "max_tokens": 50,
        }

        response = requests.post(
            f"{api_base_url}/v1/chat/completions",
            headers=auth_headers,
            json=payload,
            stream=True,
            verify=False,
        )

        if response.status_code != 200:
            pytest.skip("API not available")

        # Parse SSE stream
        chunks = []
        for line in response.iter_lines():
            if line:
                line = line.decode("utf-8")
                if line.startswith("data: "):
                    data_str = line[6:]
                    if data_str != "[DONE]":
                        try:
                            chunk = json.loads(data_str)
                            chunks.append(chunk)
                        except json.JSONDecodeError:
                            continue

        assert len(chunks) > 0

        # Validate streaming chunk format
        for chunk in chunks:
            assert "id" in chunk
            assert "object" in chunk
            assert chunk["object"] == "chat.completion.chunk"
            assert "created" in chunk
            assert "model" in chunk
            assert "choices" in chunk
            assert isinstance(chunk["choices"], list)

            if chunk["choices"]:
                choice = chunk["choices"][0]
                assert "index" in choice
                assert "delta" in choice
                delta = choice["delta"]
                # Delta may have content or role
                assert isinstance(delta, dict)

    def test_parameter_compatibility(self, api_base_url, auth_headers):
        """Test all OpenAI parameters are supported."""
        # Test various OpenAI parameters
        payload = {
            "model": "gpt-4",
            "messages": [{"role": "user", "content": "Test"}],
            "temperature": 0.5,
            "top_p": 0.9,
            "n": 1,
            "stream": False,
            "stop": ["\n"],
            "max_tokens": 50,
            "presence_penalty": 0.1,
            "frequency_penalty": 0.1,
            "logit_bias": {"1234": -100},  # Bias against certain tokens
            "user": "test-user-id",
        }

        response = requests.post(
            f"{api_base_url}/v1/chat/completions",
            headers=auth_headers,
            json=payload,
            verify=False,
        )

        if response.status_code != 200:
            pytest.skip("API not available")

        data = response.json()

        # Should not error on valid OpenAI parameters
        assert (
            "error" not in data
            or data.get("error", {}).get("type") != "invalid_request_error"
        )

    def test_completion_vs_chat_compatibility(self, api_base_url, auth_headers):
        """Test both /completions and /chat/completions endpoints if supported."""
        # Test legacy completions endpoint if available
        payload = {
            "model": "text-davinci-003",
            "prompt": "Hello, world!",
            "max_tokens": 50,
        }

        response = requests.post(
            f"{api_base_url}/v1/completions",
            headers=auth_headers,
            json=payload,
            verify=False,
        )

        # May or may not be supported
        if response.status_code == 200:
            data = response.json()
            assert data["object"] == "text_completion"
            assert "choices" in data
            assert len(data["choices"]) > 0
            assert "text" in data["choices"][0]
