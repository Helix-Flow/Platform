#!/bin/bash

echo "🧪 Testing API Gateway HTTP Mode"
echo "==============================="

# Set environment variables to force HTTP mode
export PORT=8080
export TLS_CERT=""
export TLS_KEY=""
export REDIS_HOST="localhost:6379"
export INFERENCE_POOL_URL="localhost:50051"
export AUTH_SERVICE_URL="localhost:8081"

echo "Environment variables:"
echo "  PORT: $PORT"
echo "  TLS_CERT: '$TLS_CERT'"
echo "  TLS_KEY: '$TLS_KEY'"
echo "  REDIS_HOST: $REDIS_HOST"
echo "  INFERENCE_POOL_URL: $INFERENCE_POOL_URL"
echo "  AUTH_SERVICE_URL: $AUTH_SERVICE_URL"

# Start API Gateway
echo ""
echo "Starting API Gateway..."
./api-gateway/bin/api-gateway &
GATEWAY_PID=$!

# Wait for startup
sleep 5

# Check if process is running
if ps -p $GATEWAY_PID > /dev/null; then
    echo "✅ API Gateway is running (PID: $GATEWAY_PID)"
    
    # Test health endpoint
    echo ""
    echo "Testing health endpoint..."
    HEALTH_RESPONSE=$(curl -s http://localhost:8080/health 2>/dev/null || echo "FAILED")
    echo "Health response: $HEALTH_RESPONSE"
    
    # Test models endpoint
    echo ""
    echo "Testing models endpoint..."
    MODELS_RESPONSE=$(curl -s http://localhost:8080/v1/models 2>/dev/null || echo "FAILED")
    echo "Models response: $MODELS_RESPONSE"
    
    # Test chat completion
    echo ""
    echo "Testing chat completion..."
    CHAT_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/chat/completions \
        -H "Content-Type: application/json" \
        -d '{
            "model": "gpt-3.5-turbo",
            "messages": [
                {
                    "role": "user",
                    "content": "Hello, test!"
                }
            ]
        }' 2>/dev/null || echo "FAILED")
    
    echo "Chat response: $CHAT_RESPONSE"
    
    if [ "$CHAT_RESPONSE" != "FAILED" ]; then
        CONTENT=$(echo "$CHAT_RESPONSE" | jq -r '.choices[0].message.content // "No content"' 2>/dev/null)
        echo "Response content: $CONTENT"
    fi
    
else
    echo "❌ API Gateway failed to start"
fi

# Cleanup
echo ""
echo "Cleaning up..."
kill $GATEWAY_PID 2>/dev/null || true
wait $GATEWAY_PID 2>/dev/null || true

echo "Test completed."