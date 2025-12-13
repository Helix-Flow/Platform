# HelixFlow API Reference

## Overview

The HelixFlow API provides enterprise-grade AI inference capabilities with OpenAI-compatible endpoints, advanced security features, and comprehensive monitoring.

## Base URL

```
https://api.helixflow.com/v1
```

## Authentication

All API requests require authentication using Bearer tokens in the Authorization header.

```http
Authorization: Bearer YOUR_API_KEY
```

### Getting an API Key

1. Sign up for a HelixFlow account at [https://helixflow.com/signup](https://helixflow.com/signup)
2. Navigate to the API Keys section in your dashboard
3. Generate a new API key
4. Store your API key securely - it cannot be retrieved once generated

### API Key Security

- Never expose your API key in client-side code
- Use environment variables to store API keys
- Rotate API keys regularly
- Use different API keys for different environments (development, staging, production)

## Rate Limiting

API requests are rate limited based on your plan:

| Plan | Requests per Minute | Requests per Month |
|------|-------------------|-------------------|
| Starter | 100 | 10,000 |
| Pro | 1,000 | 100,000 |
| Enterprise | Custom | Custom |

Rate limit information is included in response headers:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
```

## Response Format

All API responses follow a consistent JSON format:

### Success Response

```json
{
  "data": {},
  "meta": {
    "timestamp": "2024-01-01T00:00:00Z",
    "request_id": "req_1234567890"
  }
}
```

### Error Response

```json
{
  "error": {
    "code": "invalid_request",
    "message": "The request is missing required parameters",
    "details": {},
    "timestamp": "2024-01-01T00:00:00Z",
    "request_id": "req_1234567890"
  }
}
```

## Endpoints

### Chat Completions

Create a chat completion using the specified model.

**Endpoint:** `POST /chat/completions`

**Request Body:**

```json
{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "system",
      "content": "You are a helpful assistant."
    },
    {
      "role": "user",
      "content": "Hello, world!"
    }
  ],
  "temperature": 0.7,
  "max_tokens": 150,
  "top_p": 1.0,
  "frequency_penalty": 0.0,
  "presence_penalty": 0.0,
  "stop": ["\n", "User:"],
  "stream": false,
  "user": "user-123"
}
```

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| model | string | Yes | The model to use for completion |
| messages | array | Yes | Array of message objects |
| temperature | number | No | Sampling temperature (0-2, default: 1) |
| max_tokens | integer | No | Maximum tokens in response (default: 150) |
| top_p | number | No | Nucleus sampling parameter (0-1, default: 1) |
| frequency_penalty | number | No | Frequency penalty (-2 to 2, default: 0) |
| presence_penalty | number | No | Presence penalty (-2 to 2, default: 0) |
| stop | array | No | Stop sequences |
| stream | boolean | No | Enable streaming responses |
| user | string | No | Unique identifier for the end-user |

**Message Object:**

```json
{
  "role": "system" | "user" | "assistant",
  "content": "message content",
  "name": "optional name"
}
```

**Response:**

```json
{
  "id": "chatcmpl-1234567890",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "gpt-3.5-turbo",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 9,
    "completion_tokens": 12,
    "total_tokens": 21
  }
}
```

**Streaming Response:**

When `stream: true`, the API returns Server-Sent Events (SSE):

```
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}

data: [DONE]
```

### List Models

List available models.

**Endpoint:** `GET /models`

**Response:**

```json
{
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
```

### Model Information

Get information about a specific model.

**Endpoint:** `GET /models/{model}`

**Response:**

```json
{
  "id": "gpt-3.5-turbo",
  "object": "model",
  "created": 1677610602,
  "owned_by": "openai"
}
```

### Health Check

Check API health status.

**Endpoint:** `GET /health`

**Response:**

```json
{
  "status": "healthy",
  "timestamp": 1677652288,
  "version": "1.0.0",
  "services": {
    "database": "healthy",
    "redis": "healthy"
  }
}
```

### Authentication

**Login:** `POST /auth/login`

**Request:**

```json
{
  "username": "user@example.com",
  "password": "password123"
}
```

**Response:**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer",
  "expires_in": 3600,
  "user": {
    "id": "user-123",
    "username": "user@example.com",
    "email": "user@example.com"
  }
}
```

**Register:** `POST /auth/register`

**Request:**

```json
{
  "username": "user@example.com",
  "email": "user@example.com",
  "password": "password123"
}
```

**Response:**

```json
{
  "message": "User registered successfully",
  "user_id": "user-123",
  "username": "user@example.com"
}
```

## Error Codes

| Code | Description |
|------|-------------|
| invalid_request | The request is missing required parameters |
| invalid_api_key | The API key provided is invalid |
| rate_limit_exceeded | Rate limit has been exceeded |
| model_not_found | The specified model does not exist |
| insufficient_quota | Quota exceeded for the organization |
| server_error | Internal server error |
| service_unavailable | Service temporarily unavailable |
| timeout | Request timeout |
| authentication_failed | Authentication failed |
| authorization_failed | Authorization failed |

## Status Codes

| Status | Description |
|--------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 429 | Too Many Requests |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

## SDKs and Libraries

### Python

```bash
pip install helixflow
```

```python
import helixflow

client = helixflow.Client(api_key="your-api-key")

response = client.chat.completions.create(
    model="gpt-3.5-turbo",
    messages=[
        {"role": "user", "content": "Hello, world!"}
    ]
)

print(response.choices[0].message.content)
```

### JavaScript

```bash
npm install helixflow
```

```javascript
const HelixFlow = require('helixflow');

const client = new HelixFlow.Client({
    apiKey: 'your-api-key'
});

const response = await client.chat.completions.create({
    model: 'gpt-3.5-turbo',
    messages: [
        {role: 'user', content: 'Hello, world!'}
    ]
});

console.log(response.choices[0].message.content);
```

### Go

```bash
go get github.com/helixflow/helixflow-go
```

```go
package main

import (
    "context"
    "fmt"
    "github.com/helixflow/helixflow-go"
)

func main() {
    client := helixflow.NewClient("your-api-key")
    
    response, err := client.CreateChatCompletion(context.Background(), helixflow.ChatCompletionRequest{
        Model: "gpt-3.5-turbo",
        Messages: []helixflow.Message{
            {Role: "user", Content: "Hello, world!"},
        },
    })
    
    if err != nil {
        panic(err)
    }
    
    fmt.Println(response.Choices[0].Message.Content)
}
```

## Best Practices

### 1. Retry Logic

Implement exponential backoff for transient errors:

```python
import time
import random

def retry_with_backoff(func, max_retries=3, base_delay=1):
    for attempt in range(max_retries):
        try:
            return func()
        except Exception as e:
            if attempt == max_retries - 1:
                raise e
            delay = base_delay * (2 ** attempt) + random.uniform(0, 1)
            time.sleep(delay)
```

### 2. Request Batching

Batch multiple requests to reduce API calls:

```python
# Instead of multiple individual requests
responses = []
for message in messages:
    response = client.chat.completions.create(
        model="gpt-3.5-turbo",
        messages=[{"role": "user", "content": message}]
    )
    responses.append(response)

# Use batch processing
batch_response = client.batch.process([
    {"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": msg}]}
    for msg in messages
])
```

### 3. Caching

Cache responses when appropriate:

```python
import hashlib
import json
from functools import lru_cache

@lru_cache(maxsize=1000)
def cached_completion(model, messages_hash):
    messages = json.loads(messages_hash)
    return client.chat.completions.create(
        model=model,
        messages=messages
    )

def get_completion_with_cache(model, messages):
    messages_hash = json.dumps(messages, sort_keys=True)
    return cached_completion(model, messages_hash)
```

### 4. Error Handling

Implement comprehensive error handling:

```python
try:
    response = client.chat.completions.create(
        model="gpt-3.5-turbo",
        messages=messages
    )
except helixflow.RateLimitError as e:
    # Handle rate limit errors
    print(f"Rate limit exceeded: {e}")
    time.sleep(60)  # Wait before retrying
except helixflow.AuthenticationError as e:
    # Handle authentication errors
    print(f"Authentication failed: {e}")
    # Refresh API key or check credentials
except helixflow.APIError as e:
    # Handle other API errors
    print(f"API error: {e}")
    # Implement retry logic
```

### 5. Monitoring

Monitor API usage and performance:

```python
import time
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def monitored_completion(model, messages):
    start_time = time.time()
    
    try:
        response = client.chat.completions.create(
            model=model,
            messages=messages
        )
        
        duration = time.time() - start_time
        tokens = response.usage.total_tokens
        
        logger.info(f"API call completed in {duration:.2f}s with {tokens} tokens")
        
        return response
        
    except Exception as e:
        duration = time.time() - start_time
        logger.error(f"API call failed after {duration:.2f}s: {e}")
        raise
```

## Webhooks

Configure webhooks to receive real-time notifications:

### Webhook Events

- `completion.created` - New completion created
- `completion.failed` - Completion failed
- `rate_limit.exceeded` - Rate limit exceeded
- `quota.low` - Quota running low

### Webhook Payload

```json
{
  "event": "completion.created",
  "timestamp": "2024-01-01T00:00:00Z",
  "data": {
    "id": "chatcmpl-123",
    "model": "gpt-3.5-turbo",
    "user_id": "user-123"
  }
}
```

### Webhook Security

Verify webhook signatures:

```python
import hmac
import hashlib

def verify_webhook_signature(payload, signature, secret):
    expected_signature = hmac.new(
        secret.encode(),
        payload.encode(),
        hashlib.sha256
    ).hexdigest()
    
    return hmac.compare_digest(expected_signature, signature)
```

## Support

For API support:

- **Documentation**: [https://docs.helixflow.com](https://docs.helixflow.com)
- **Support Portal**: [https://support.helixflow.com](https://support.helixflow.com)
- **Email**: support@helixflow.com
- **Status Page**: [https://status.helixflow.com](https://status.helixflow.com)
- **Community**: [https://community.helixflow.com](https://community.helixflow.com)

## Changelog

### Version 1.0.0 (2024-01-01)

- Initial API release
- Chat completions endpoint
- Model listing endpoint
- Authentication system
- Rate limiting
- Multi-cloud support

### Version 1.1.0 (2024-02-15)

- Added streaming support
- Enhanced error handling
- Webhook support
- Batch processing
- Improved monitoring

---

For the most up-to-date API documentation, visit [https://docs.helixflow.com/api](https://docs.helixflow.com/api)