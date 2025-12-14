#!/bin/bash

set -e

echo "🚀 HelixFlow Integration Test Suite"
echo "===================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test results
PASSED=0
FAILED=0

# Function to run test
run_test() {
    local test_name=$1
    local test_command=$2
    local expected_result=$3
    
    echo -n "Testing: $test_name... "
    
    if eval "$test_command"; then
        if [ "$expected_result" = "pass" ]; then
            echo -e "${GREEN}✓ PASS${NC}"
            PASSED=$((PASSED + 1))
        else
            echo -e "${RED}✗ FAIL (expected to fail)${NC}"
            FAILED=$((FAILED + 1))
        fi
    else
        if [ "$expected_result" = "fail" ]; then
            echo -e "${GREEN}✓ PASS (expected to fail)${NC}"
            PASSED=$((PASSED + 1))
        else
            echo -e "${RED}✗ FAIL${NC}"
            FAILED=$((FAILED + 1))
        fi
    fi
}

echo "1. Service Binary Tests"
echo "-----------------------"

# Check if binaries exist
run_test "API Gateway binary exists" "[ -f ./api-gateway/bin/api-gateway ]" "pass"
run_test "Auth Service binary exists" "[ -f ./auth-service/bin/auth-service ]" "pass"
run_test "Inference Pool binary exists" "[ -f ./inference-pool/bin/inference-pool ]" "pass"
run_test "Monitoring Service binary exists" "[ -f ./monitoring/bin/monitoring ]" "pass"

echo ""
echo "2. Service Health Tests"
echo "-----------------------"

# Start services in background for testing
echo "Starting services for testing..."

# Start monitoring service
./monitoring/bin/monitoring &
MONITORING_PID=$!
sleep 2

# Start inference pool service
./inference-pool/bin/inference-pool &
INFERENCE_PID=$!
sleep 2

# Start auth service
./auth-service/bin/auth-service &
AUTH_PID=$!
sleep 2

# Test service health endpoints
run_test "Monitoring service health" "curl -s -f http://localhost:8083/health > /dev/null" "pass"
run_test "Inference service health" "curl -s -f http://localhost:50051/health > /dev/null" "pass"
run_test "Auth service health" "curl -s -f http://localhost:8081/health > /dev/null" "pass"

echo ""
echo "3. API Gateway Tests"
echo "--------------------"

# Start API Gateway
./api-gateway/bin/api-gateway &
GATEWAY_PID=$!
sleep 3

# Test API Gateway health
run_test "API Gateway health" "curl -s -f http://localhost:8080/health > /dev/null" "pass"

# Test models endpoint
run_test "Models endpoint" "curl -s -f http://localhost:8080/v1/models > /dev/null" "pass"

# Test chat completion endpoint
run_test "Chat completion endpoint" "curl -s -f -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer demo-key' \
  -d '{\"model\": \"gpt-3.5-turbo\", \"messages\": [{\"role\": \"user\", \"content\": \"Hello\"}]}' > /dev/null" "pass"

echo ""
echo "4. Response Validation Tests"
echo "----------------------------"

# Test response format
run_test "Response format validation" "curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer demo-key' \
  -d '{\"model\": \"gpt-3.5-turbo\", \"messages\": [{\"role\": \"user\", \"content\": \"Hello\"}]}' | \
  jq -e '.choices[0].message.content != null' > /dev/null" "pass"

# Test usage information
run_test "Usage information present" "curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer demo-key' \
  -d '{\"model\": \"gpt-3.5-turbo\", \"messages\": [{\"role\": \"user\", \"content\": \"Hello\"}]}' | \
  jq -e '.usage.total_tokens > 0' > /dev/null" "pass"

echo ""
echo "5. Security Tests"
echo "-----------------"

# Test authentication without token
run_test "Authentication required" "curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/v1/chat/completions | grep -q '401'" "pass"

# Test with invalid token
run_test "Invalid token rejected" "curl -s -o /dev/null -w '%{http_code}' -H 'Authorization: Bearer invalid-token' http://localhost:8080/v1/chat/completions | grep -q '401'" "pass"

echo ""
echo "6. Performance Tests"
echo "--------------------"

# Test response time
start_time=$(date +%s%3N)
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer demo-key' \
  -d '{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}' > /dev/null
end_time=$(date +%s%3N)
response_time=$((end_time - start_time))

if [ $response_time -lt 1000 ]; then
    echo -e "Response time test... ${GREEN}✓ PASS${NC} (${response_time}ms)"
    PASSED=$((PASSED + 1))
else
    echo -e "Response time test... ${RED}✗ FAIL${NC} (${response_time}ms > 1000ms)"
    FAILED=$((FAILED + 1))
fi

echo ""
echo "7. Model Support Tests"
echo "----------------------"

# Test different models
models=("gpt-3.5-turbo" "gpt-4" "claude-3-sonnet" "llama-2-70b")
for model in "${models[@]}"; do
    run_test "Model support: $model" "curl -s -f -X POST http://localhost:8080/v1/chat/completions \
      -H 'Content-Type: application/json' \
      -H 'Authorization: Bearer demo-key' \
      -d '{\"model\": \"$model\", \"messages\": [{\"role\": \"user\", \"content\": \"Test\"}]}' > /dev/null" "pass"
done

echo ""
echo "8. Error Handling Tests"
echo "-----------------------"

# Test invalid JSON
run_test "Invalid JSON handling" "curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer demo-key' \
  -d 'invalid json' | grep -q '400'" "pass"

# Test missing required fields
run_test "Missing model field" "curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer demo-key' \
  -d '{\"messages\": [{\"role\": \"user\", \"content\": \"Hello\"}]}' | grep -q '400'" "pass"

# Test missing messages
run_test "Missing messages field" "curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer demo-key' \
  -d '{\"model\": \"gpt-3.5-turbo\"}' | grep -q '400'" "pass"

echo ""
echo "Cleaning up..."
echo "=============="

# Kill background processes
kill $GATEWAY_PID $MONITORING_PID $INFERENCE_PID $AUTH_PID 2>/dev/null || true
wait $GATEWAY_PID $MONITORING_PID $INFERENCE_PID $AUTH_PID 2>/dev/null || true

echo ""
echo "===================================="
echo "Integration Test Results"
echo "===================================="
echo -e "Total Tests: $((PASSED + FAILED))"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo -e "Success Rate: ${YELLOW}$(( PASSED * 100 / (PASSED + FAILED) ))%${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All integration tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some integration tests failed.${NC}"
    exit 1
fi