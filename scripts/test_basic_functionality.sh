#!/bin/bash

set -e

echo "🚀 HelixFlow Basic Functionality Test"
echo "======================================"

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
echo "2. Monitoring Service Tests (No Dependencies)"
echo "----------------------------------------------"

echo "Starting monitoring service..."
./monitoring/bin/monitoring &
MONITORING_PID=$!
sleep 2

# Test monitoring service health
run_test "Monitoring service health" "curl -s -f http://localhost:8083/health > /dev/null" "pass"
run_test "Monitoring service metrics" "curl -s -f http://localhost:8083/metrics > /dev/null" "pass"

echo ""
echo "3. API Gateway Tests (HTTP Mode)"
echo "--------------------------------"

echo "Starting API Gateway in HTTP mode..."
# Start API Gateway without TLS
PORT=8080 ./api-gateway/bin/api-gateway &
GATEWAY_PID=$!
sleep 3

# Test API Gateway health
run_test "API Gateway health (HTTP)" "curl -s -f http://localhost:8080/health > /dev/null" "pass"

# Test models endpoint without auth
run_test "Models endpoint (no auth)" "curl -s -f http://localhost:8080/v1/models > /dev/null" "pass"

echo ""
echo "4. Basic API Functionality Tests"
echo "--------------------------------"

# Test basic API structure
run_test "Models response format" "curl -s http://localhost:8080/v1/models | jq -e '.data != null' > /dev/null" "pass"

# Test model list
run_test "Model list contains GPT models" "curl -s http://localhost:8080/v1/models | jq -e '.data[] | select(.id == \"gpt-3.5-turbo\")' > /dev/null" "pass"

# Test basic chat completion without auth requirement
run_test "Chat completion mock response" "curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{\"model\": \"gpt-3.5-turbo\", \"messages\": [{\"role\": \"user\", \"content\": \"Hello\"}]}' | jq -e '.choices[0].message.content != null' > /dev/null" "pass"

echo ""
echo "5. Response Quality Tests"
echo "-------------------------"

# Test response content quality
run_test "Response content is meaningful" "curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{\"model\": \"gpt-3.5-turbo\", \"messages\": [{\"role\": \"user\", \"content\": \"Hello\"}]}' | jq -e '.choices[0].message.content | length > 10' > /dev/null" "pass"

# Test usage information
run_test "Usage information present" "curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{\"model\": \"gpt-3.5-turbo\", \"messages\": [{\"role\": \"user\", \"content\": \"Hello\"}]}' | jq -e '.usage.total_tokens > 0' > /dev/null" "pass"

echo ""
echo "6. Performance Tests"
echo "--------------------"

# Test response time
start_time=$(date +%s%3N)
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}' > /dev/null
end_time=$(date +%s%3N)
response_time=$((end_time - start_time))

if [ $response_time -lt 2000 ]; then
    echo -e "Response time test... ${GREEN}✓ PASS${NC} (${response_time}ms)"
    PASSED=$((PASSED + 1))
else
    echo -e "Response time test... ${RED}✗ FAIL${NC} (${response_time}ms > 2000ms)"
    FAILED=$((FAILED + 1))
fi

echo ""
echo "7. Error Handling Tests"
echo "-----------------------"

# Test invalid JSON
run_test "Invalid JSON handling" "curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d 'invalid json' | grep -q '400'" "pass"

# Test missing required fields
run_test "Missing model field" "curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{\"messages\": [{\"role\": \"user\", \"content\": \"Hello\"}]}' | grep -q '400'" "pass"

echo ""
echo "Cleaning up..."
echo "=============="

# Kill background processes
kill $GATEWAY_PID $MONITORING_PID 2>/dev/null || true
wait $GATEWAY_PID $MONITORING_PID 2>/dev/null || true

echo ""
echo "======================================"
echo "Basic Functionality Test Results"
echo "======================================"
echo -e "Total Tests: $((PASSED + FAILED))"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo -e "Success Rate: ${YELLOW}$(( PASSED * 100 / (PASSED + FAILED) ))%${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All basic functionality tests passed!${NC}"
    echo ""
    echo "✅ Core services are working correctly"
    echo "✅ API Gateway provides realistic mock responses"
    echo "✅ Response format matches OpenAI API specification"
    echo "✅ Performance is acceptable for demo purposes"
    echo ""
    echo "Next steps:"
    echo "1. Set up PostgreSQL database for auth service"
    echo "2. Generate TLS certificates for secure communication"
    echo "3. Implement real gRPC integration between services"
    echo "4. Add comprehensive test coverage"
    exit 0
else
    echo -e "${RED}❌ Some basic functionality tests failed.${NC}"
    exit 1
fi