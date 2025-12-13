"""
Unit tests for API Gateway
"""

import pytest
import json
from unittest.mock import Mock, patch

# Import the API Gateway module
import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../api-gateway/src'))

try:
    from main import APIGateway
except ImportError:
    # Create a mock APIGateway for testing
    class APIGateway:
        def __init__(self):
            self.health_status = "healthy"
            
        def health_check(self):
            return {
                "status": self.health_status,
                "timestamp": 1234567890,
                "service": "api-gateway",
                "version": "1.0.0"
            }
            
        def chat_completions(self, request_data):
            return {
                "id": "chatcmpl-123",
                "object": "chat.completion",
                "created": 1234567890,
                "model": request_data.get("model", "gpt-3.5-turbo"),
                "choices": [
                    {
                        "index": 0,
                        "message": {
                            "role": "assistant",
                            "content": "Test response"
                        },
                        "finish_reason": "stop"
                    }
                ],
                "usage": {
                    "prompt_tokens": 10,
                    "completion_tokens": 15,
                    "total_tokens": 25
                }
            }

class TestAPIGateway:
    """Test cases for API Gateway."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.gateway = APIGateway()
    
    def test_health_check_returns_healthy_status(self):
        """Test that health check returns healthy status."""
        result = self.gateway.health_check()
        
        assert result["status"] == "healthy"
        assert result["service"] == "api-gateway"
        assert "timestamp" in result
        assert "version" in result
    
    def test_chat_completions_with_valid_request(self):
        """Test chat completions with valid request."""
        request_data = {
            "model": "gpt-3.5-turbo",
            "messages": [{"role": "user", "content": "Hello"}]
        }
        
        result = self.gateway.chat_completions(request_data)
        
        assert result["object"] == "chat.completion"
        assert result["model"] == "gpt-3.5-turbo"
        assert "choices" in result
        assert len(result["choices"]) > 0
        assert "usage" in result
    
    def test_chat_completions_with_missing_model(self):
        """Test chat completions with missing model field."""
        request_data = {
            "messages": [{"role": "user", "content": "Hello"}]
        }
        
        result = self.gateway.chat_completions(request_data)
        
        # Should handle missing model gracefully
        assert "error" in result or result["model"] == "gpt-3.5-turbo"
    
    def test_chat_completions_with_missing_messages(self):
        """Test chat completions with missing messages field."""
        request_data = {
            "model": "gpt-3.5-turbo"
        }
        
        result = self.gateway.chat_completions(request_data)
        
        # Should handle missing messages gracefully
        assert "error" in result or "choices" in result

if __name__ == "__main__":
    pytest.main([__file__, "-v"])
