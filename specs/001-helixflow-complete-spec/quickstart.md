# HelixFlow Quick Start Guide

**Version**: 1.0.0
**Date**: 2025-12-13
**Feature**: specs/001-helixflow-complete-spec/spec.md

## Overview

HelixFlow is a universal AI inference platform providing OpenAI-compatible APIs for 300+ AI models from leading providers. This guide will get you up and running in minutes.

## Prerequisites

- **API Key**: Sign up at [helixflow.ai](https://helixflow.ai) to get your API key
- **Development Environment**: Any modern IDE (VS Code, Cursor, JetBrains, Vim/Neovim)
- **SDK**: Choose your preferred language SDK

## Installation

### Python SDK
```bash
pip install helixflow
```

### JavaScript/TypeScript SDK
```bash
npm install helixflow
```

### Go SDK
```bash
go get github.com/helixflow/helixflow-go
```

### Java SDK
```xml
<dependency>
    <groupId>ai.helixflow</groupId>
    <artifactId>helixflow-java</artifactId>
    <version>1.0.0</version>
</dependency>
```

### C# SDK
```bash
dotnet add package HelixFlow
```

### Rust SDK
```toml
[dependencies]
helixflow = "1.0"
```

### PHP SDK
```bash
composer require helixflow/helixflow-php
```

## Authentication

Set your API key as an environment variable:

```bash
export HELIXFLOW_API_KEY="your-api-key-here"
```

## Basic Usage

### Chat Completion (Python)
```python
import helixflow

client = helixflow.Client()

response = client.chat.completions.create(
    model="gpt-4",
    messages=[
        {"role": "user", "content": "Hello, how are you?"}
    ],
    max_tokens=100
)

print(response.choices[0].message.content)
```

### Chat Completion (JavaScript)
```javascript
import HelixFlow from 'helixflow';

const client = new HelixFlow.Client();

const response = await client.chat.completions.create({
    model: 'claude-3-sonnet',
    messages: [
        { role: 'user', content: 'Write a haiku about AI' }
    ],
    temperature: 0.8
});

console.log(response.choices[0].message.content);
```

### Chat Completion (Go)
```go
package main

import (
    "context"
    "fmt"
    "github.com/helixflow/helixflow-go"
)

func main() {
    client := helixflow.NewClient()

    response, err := client.Chat.Completions.Create(context.Background(), helixflow.ChatCompletionRequest{
        Model: "deepseek-chat",
        Messages: []helixflow.ChatMessage{
            {Role: "user", Content: "Explain quantum computing"},
        },
        MaxTokens: 200,
    })

    if err != nil {
        panic(err)
    }

    fmt.Println(response.Choices[0].Message.Content)
}
```

### Chat Completion (Java)
```java
import ai.helixflow.Client;
import ai.helixflow.models.ChatCompletionRequest;
import ai.helixflow.models.ChatMessage;

public class Example {
    public static void main(String[] args) {
        Client client = new Client();

        ChatCompletionRequest request = ChatCompletionRequest.builder()
            .model("glm-4")
            .addMessage(ChatMessage.builder()
                .role("user")
                .content("Create a simple recipe")
                .build())
            .maxTokens(150)
            .build();

        var response = client.chat().completions().create(request);
        System.out.println(response.getChoices().get(0).getMessage().getContent());
    }
}
```

### Chat Completion (C#)
```csharp
using HelixFlow;

var client = new Client();

var response = await client.Chat.Completions.CreateAsync(new ChatCompletionRequest
{
    Model = "qwen-max",
    Messages = new[]
    {
        new ChatMessage { Role = "user", Content = "Design a mobile app UI" }
    },
    MaxTokens = 300
});

Console.WriteLine(response.Choices[0].Message.Content);
```

### Chat Completion (Rust)
```rust
use helixflow::{Client, ChatCompletionRequest, ChatMessage};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = Client::new()?;

    let request = ChatCompletionRequest {
        model: "gpt-4".to_string(),
        messages: vec![
            ChatMessage {
                role: "user".to_string(),
                content: "Debug this code snippet".to_string(),
            }
        ],
        max_tokens: Some(200),
        ..Default::default()
    };

    let response = client.chat().completions().create(request).await?;
    println!("{}", response.choices[0].message.content);

    Ok(())
}
```

### Chat Completion (PHP)
```php
<?php

require 'vendor/autoload.php';

use HelixFlow\Client;

$client = new Client();

$response = $client->chat()->completions()->create([
    'model' => 'claude-3-haiku',
    'messages' => [
        ['role' => 'user', 'content' => 'Optimize this SQL query']
    ],
    'max_tokens' => 100
]);

echo $response->choices[0]->message->content;
```

## Streaming Responses

All SDKs support streaming for real-time responses:

```python
response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Tell me a story"}],
    stream=True
)

for chunk in response:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
```

## Available Models

Get a list of all available models:

```python
models = client.models.list()
for model in models.data:
    print(f"{model.id}: {model.owned_by}")
```

## Error Handling

```python
try:
    response = client.chat.completions.create(
        model="gpt-4",
        messages=[{"role": "user", "content": "Hello"}]
    )
except helixflow.RateLimitError:
    print("Rate limit exceeded. Please try again later.")
except helixflow.AuthenticationError:
    print("Invalid API key.")
except helixflow.APIError as e:
    print(f"API error: {e}")
```

## Rate Limits

- **Free Tier**: 100 requests/minute, 10,000 tokens/month
- **Pro Tier**: 1,000 requests/minute, 1M tokens/month
- **Enterprise Tier**: Custom limits based on contract

## Billing

Usage is billed per token:
- Input tokens: $0.0015 per 1K tokens (GPT-4)
- Output tokens: $0.002 per 1K tokens (GPT-4)

Monitor your usage in the dashboard at [helixflow.ai/dashboard](https://helixflow.ai/dashboard).

## Support

- **Documentation**: [docs.helixflow.ai](https://docs.helixflow.ai)
- **Community**: [community.helixflow.ai](https://community.helixflow.ai)
- **Enterprise Support**: enterprise@helixflow.ai
- **Status Page**: [status.helixflow.ai](https://status.helixflow.ai)

## Next Steps

1. **Explore Models**: Try different AI models for various use cases
2. **Monitor Usage**: Track your API usage and costs
3. **Join Community**: Connect with other developers on our forums
4. **Upgrade Plan**: Scale up as your needs grow

Welcome to HelixFlow! 🚀