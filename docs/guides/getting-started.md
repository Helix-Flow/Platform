# Getting Started with HelixFlow

This guide will get you up and running with HelixFlow in minutes.

## Prerequisites

- API key from [helixflow.ai](https://helixflow.ai)
- Python 3.8+ or Node.js 16+

## Installation

### Python
```bash
pip install helixflow
```

### JavaScript/TypeScript
```bash
npm install helixflow
```

## Your First API Call

```python
import helixflow

client = helixflow.HelixFlow("your-api-key-here")

response = client.chat_completion(
    model="gpt-4",
    messages=[
        {"role": "user", "content": "Hello, how are you?"}
    ]
)

print(response["choices"][0]["message"]["content"])
```

## Next Steps

1. Explore available models
2. Learn about streaming responses
3. Implement error handling
4. Monitor your usage

## Support

Need help? Check our [troubleshooting guide](./troubleshooting.md) or visit our [community forum](https://community.helixflow.ai).
