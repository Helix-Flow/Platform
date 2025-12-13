#!/bin/bash

# Simple test runner that doesn't require pytest
echo "🧪 Running HelixFlow Simple Tests..."

# Test API Gateway
echo "Testing API Gateway..."
python3 api-gateway/src/main.py > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ API Gateway: PASSED"
else
    echo "❌ API Gateway: FAILED"
fi

# Test Auth Service
echo "Testing Auth Service..."
python3 auth-service/src/main.py > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Auth Service: PASSED"
else
    echo "❌ Auth Service: FAILED"
fi

# Test Inference Pool
echo "Testing Inference Pool..."
python3 inference-pool/src/main.py > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Inference Pool: PASSED"
else
    echo "❌ Inference Pool: FAILED"
fi

# Test Monitoring Service
echo "Testing Monitoring Service..."
python3 monitoring/src/main.py > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Monitoring Service: PASSED"
else
    echo "❌ Monitoring Service: FAILED"
fi

echo "🎉 Simple tests completed!"
