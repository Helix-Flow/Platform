#!/usr/bin/env python3

import os
import sys
import json
import time
import logging
from typing import Dict, Any, Optional

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class APIGateway:
    """Simple API Gateway implementation for testing purposes."""
    
    def __init__(self):
        self.port = int(os.getenv('API_GATEWAY_PORT', 8080))
        self.health_status = "healthy"
        
    def health_check(self) -> Dict[str, Any]:
        """Health check endpoint."""
        return {
            "status": self.health_status,
            "timestamp": int(time.time()),
            "service": "api-gateway",
            "version": "1.0.0"
        }
    
    def chat_completions(self, request_data: Dict[str, Any]) -> Dict[str, Any]:
        """Mock chat completions endpoint."""
        try:
            # Validate request
            if not request_data.get("model") or not request_data.get("messages"):
                return {"error": "Missing required fields: model, messages"}
            
            # Mock response
            return {
                "id": f"chatcmpl-{int(time.time())}",
                "object": "chat.completion",
                "created": int(time.time()),
                "model": request_data["model"],
                "choices": [
                    {
                        "index": 0,
                        "message": {
                            "role": "assistant",
                            "content": "This is a mock response from HelixFlow API Gateway."
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
        except Exception as e:
            logger.error(f"Error in chat_completions: {e}")
            return {"error": str(e)}
    
    def list_models(self) -> Dict[str, Any]:
        """List available models."""
        return {
            "object": "list",
            "data": [
                {
                    "id": "gpt-3.5-turbo",
                    "object": "model",
                    "created": 1677610602,
                    "owned_by": "openai"
                },
                {
                    "id": "gpt-4",
                    "object": "model",
                    "created": 1687882411,
                    "owned_by": "openai"
                }
            ]
        }

def main():
    """Main function for testing."""
    gateway = APIGateway()
    
    # Test health check
    health = gateway.health_check()
    print(f"Health Check: {json.dumps(health, indent=2)}")
    
    # Test chat completions
    test_request = {
        "model": "gpt-3.5-turbo",
        "messages": [{"role": "user", "content": "Hello"}]
    }
    response = gateway.chat_completions(test_request)
    print(f"Chat Completion: {json.dumps(response, indent=2)}")
    
    # Test list models
    models = gateway.list_models()
    print(f"Available Models: {json.dumps(models, indent=2)}")

if __name__ == "__main__":
    main()
