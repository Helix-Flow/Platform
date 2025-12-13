"""
HelixFlow Python Client
"""

import requests
import json
import time
from typing import Dict, List, Optional, Union, Iterator
from .exceptions import HelixFlowError, AuthenticationError, RateLimitError, APIError


class HelixFlow:
    """Main HelixFlow client class."""
    
    def __init__(self, api_key: str, base_url: str = "https://api.helixflow.ai"):
        self.api_key = api_key
        self.base_url = base_url.rstrip("/")
        self.session = requests.Session()
        self.session.headers.update({
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        })
    
    def chat_completion(self, 
                       model: str, 
                       messages: List[Dict[str, str]], 
                       **kwargs) -> Dict:
        """Create a chat completion."""
        data = {
            "model": model,
            "messages": messages,
            **kwargs
        }
        
        response = self._post("/v1/chat/completions", data)
        return response
    
    def chat_completion_stream(self, 
                              model: str, 
                              messages: List[Dict[str, str]], 
                              **kwargs) -> Iterator[Dict]:
        """Create a streaming chat completion."""
        data = {
            "model": model,
            "messages": messages,
            "stream": True,
            **kwargs
        }
        
        response = self._post_stream("/v1/chat/completions", data)
        
        for line in response.iter_lines():
            if line:
                line = line.decode('utf-8')
                if line.startswith('data: '):
                    data_str = line[6:]
                    if data_str == '[DONE]':
                        break
                    try:
                        chunk = json.loads(data_str)
                        yield chunk
                    except json.JSONDecodeError:
                        continue
    
    def list_models(self) -> Dict:
        """List available models."""
        return self._get("/v1/models")
    
    def get_model(self, model_id: str) -> Dict:
        """Get information about a specific model."""
        return self._get(f"/v1/models/{model_id}")
    
    def _get(self, endpoint: str) -> Dict:
        """Make a GET request."""
        url = f"{self.base_url}{endpoint}"
        response = self.session.get(url)
        return self._handle_response(response)
    
    def _post(self, endpoint: str, data: Dict) -> Dict:
        """Make a POST request."""
        url = f"{self.base_url}{endpoint}"
        response = self.session.post(url, json=data)
        return self._handle_response(response)
    
    def _post_stream(self, endpoint: str, data: Dict) -> requests.Response:
        """Make a streaming POST request."""
        url = f"{self.base_url}{endpoint}"
        response = self.session.post(url, json=data, stream=True)
        self._handle_response(response)  # Check for errors
        return response
    
    def _handle_response(self, response: requests.Response) -> Dict:
        """Handle API response."""
        if response.status_code == 401:
            raise AuthenticationError("Invalid API key")
        elif response.status_code == 429:
            raise RateLimitError("Rate limit exceeded")
        elif not response.ok:
            try:
                error_data = response.json()
                raise APIError(f"API error: {error_data.get('error', 'Unknown error')}")
            except json.JSONDecodeError:
                raise APIError(f"HTTP {response.status_code}: {response.text}")
        
        return response.json()


class CogneeMemoryEngine:
    """Cognee memory enhancement for HelixFlow."""
    
    def __init__(self, api_key: str, graph_db_url: str = None, vector_db_url: str = None):
        self.api_key = api_key
        self.graph_db_url = graph_db_url or "bolt://localhost:7687"
        self.vector_db_url = vector_db_url or "http://localhost:6333"
        self.client = HelixFlow(api_key)
    
    def enhance_chat(self, model: str, messages: List[Dict], **kwargs) -> Dict:
        """Enhanced chat completion with memory."""
        # Add memory context to messages
        enhanced_messages = self._add_memory_context(messages)
        
        return self.client.chat_completion(model, enhanced_messages, **kwargs)
    
    def _add_memory_context(self, messages: List[Dict]) -> List[Dict]:
        """Add relevant memory context to messages."""
        # This would query the knowledge graph and vector database
        # For now, return messages unchanged
        return messages
