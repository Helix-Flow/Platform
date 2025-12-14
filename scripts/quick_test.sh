#!/bin/bash

echo "🚀 Quick HelixFlow Test"
echo "======================="

# Test monitoring service
echo "1. Testing Monitoring Service..."
./monitoring/bin/monitoring &
MONITORING_PID=$!
sleep 2

# Test monitoring health
echo "   Monitoring health: $(curl -s http://localhost:8083/health)"
echo "   Monitoring metrics: $(curl -s http://localhost:8083/metrics)"

# Test API Gateway in HTTP mode
echo ""
echo "2. Testing API Gateway (HTTP mode)..."
# Clear TLS environment variables to force HTTP mode
unset TLS_CERT TLS_KEY
PORT=8080 ./api-gateway/bin/api-gateway &
GATEWAY_PID=$!
sleep 3

# Test API Gateway health
echo "   API Gateway health: $(curl -s http://localhost:8080/health)"

# Test models endpoint
echo "   Models endpoint: $(curl -s http://localhost:8080/v1/models | jq -r '.data[0].id // "No models found"')"

# Test chat completion
echo ""
echo "3. Testing Chat Completion..."
RESPONSE=$(curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "Hello, how are you?"
      }
    ]
  }')

echo "   Response: $(echo "$RESPONSE" | jq -r '.choices[0].message.content // "No response"')"
echo "   Model: $(echo "$RESPONSE" | jq -r '.model')"
echo "   Total tokens: $(echo "$RESPONSE" | jq -r '.usage.total_tokens')"

# Cleanup
echo ""
echo "4. Cleaning up..."
kill $GATEWAY_PID $MONITORING_PID 2>/dev/null || true

echo ""
echo "✅ Quick test completed!"