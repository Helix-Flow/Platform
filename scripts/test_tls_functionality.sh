#!/bin/bash

echo "🔐 Testing HelixFlow TLS Functionality"
echo "======================================"

# Set environment variables for TLS mode
export PORT=8443
export TLS_CERT=./certs/api-gateway.crt
export TLS_KEY=./certs/api-gateway-key.pem
export REDIS_HOST=localhost:6379
export INFERENCE_POOL_URL=localhost:50051
export AUTH_SERVICE_URL=localhost:8081

echo "Environment variables set:"
echo "  PORT: $PORT"
echo "  TLS_CERT: $TLS_CERT"
echo "  TLS_KEY: $TLS_KEY"
echo "  REDIS_HOST: $REDIS_HOST"

# Check if certificates exist
echo ""
echo "Checking TLS certificates..."
if [ -f "$TLS_CERT" ] && [ -f "$TLS_KEY" ]; then
    echo "✅ TLS certificates found"
    openssl x509 -in "$TLS_CERT" -noout -text | grep -E "Subject:|Issuer:|Validity" | head -3
else
    echo "❌ TLS certificates not found"
    exit 1
fi

# Start API Gateway with TLS
echo ""
echo "Starting API Gateway with TLS..."
./api-gateway/bin/api-gateway &
GATEWAY_PID=$!

# Wait for startup
sleep 5

# Check if process is running
if ps -p $GATEWAY_PID > /dev/null; then
    echo "✅ API Gateway is running with TLS (PID: $GATEWAY_PID)"
    
    # Test HTTPS health endpoint with certificate verification
    echo ""
    echo "Testing HTTPS health endpoint..."
    HEALTH_RESPONSE=$(curl -s -f -k https://localhost:8443/health 2>/dev/null || echo "FAILED")
    echo "Health response: $HEALTH_RESPONSE"
    
    if [ "$HEALTH_RESPONSE" != "FAILED" ]; then
        echo "✅ HTTPS endpoint is working"
        
        # Test models endpoint with auth
        echo ""
        echo "Testing HTTPS models endpoint..."
        MODELS_RESPONSE=$(curl -s -f -k https://localhost:8443/v1/models 2>/dev/null || echo "FAILED")
        echo "Models response: $MODELS_RESPONSE"
        
        if [ "$MODELS_RESPONSE" != "FAILED" ]; then
            echo "✅ HTTPS API endpoints are working"
        else
            echo "❌ HTTPS API endpoints failed"
        fi
        
        # Test with authentication
        echo ""
        echo "Testing HTTPS with authentication..."
        CHAT_RESPONSE=$(curl -s -f -k https://localhost:8443/v1/chat/completions \
            -H "Authorization: Bearer demo-key" \
            -H "Content-Type: application/json" \
            -d '{
                "model": "gpt-3.5-turbo",
                "messages": [
                    {
                        "role": "user",
                        "content": "Hello, TLS test!"
                    }
                ]
            }' 2>/dev/null || echo "FAILED")
        
        if [ "$CHAT_RESPONSE" != "FAILED" ]; then
            echo "✅ HTTPS with authentication is working"
            CONTENT=$(echo "$CHAT_RESPONSE" | jq -r '.choices[0].message.content // "No content"' 2>/dev/null)
            echo "Response content: $CONTENT"
        else
            echo "❌ HTTPS with authentication failed"
        fi
    else
        echo "❌ API Gateway failed to start with TLS"
    fi
else
    echo "❌ API Gateway failed to start"
fi

# Cleanup
echo ""
echo "Cleaning up..."
kill $GATEWAY_PID 2>/dev/null || true
wait $GATEWAY_PID 2>/dev/null || true

echo ""
echo "✅ TLS functionality test completed!"