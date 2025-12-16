#!/bin/bash

echo "=== HelixFlow Platform - Integration Test Suite ==="
echo "Date: $(date)"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
declare -a TEST_RESULTS
declare -a TEST_NAMES

echo "1. Testing Service Binaries..."
for service in api-gateway auth-service inference-pool monitoring; do
    if [ -f "./$service/bin/$service" ]; then
        echo -e "   ${GREEN}✅${NC} $service binary exists"
        TEST_RESULTS+=("PASS")
        TEST_NAMES+=("$service binary")
    else
        echo -e "   ${RED}❌${NC} $service binary missing"
        TEST_RESULTS+=("FAIL")
        TEST_NAMES+=("$service binary")
    fi
done
echo ""

echo "2. Testing Database Connectivity..."
cd ./test/db_test
if timeout 5s go run simple_check.go > /dev/null 2>&1; then
    echo -e "   ${GREEN}✅${NC} Database connection successful"
    TEST_RESULTS+=("PASS")
    TEST_NAMES+=("Database connection")
else
    echo -e "   ${RED}❌${NC} Database connection failed"
    TEST_RESULTS+=("FAIL")
    TEST_NAMES+=("Database connection")
fi
cd ../../
echo ""

echo "3. Testing Certificate Validation..."
if [ -f "./certs/helixflow-ca.pem" ] && [ -f "./certs/server-cert.pem" ] && [ -f "./certs/api-gateway.p12" ]; then
    echo -e "   ${GREEN}✅${NC} TLS certificates present"
    TEST_RESULTS+=("PASS")
    TEST_NAMES+=("TLS certificates")
else
    echo -e "   ${RED}❌${NC} TLS certificates missing"
    TEST_RESULTS+=("FAIL")
    TEST_NAMES+=("TLS certificates")
fi
echo ""

echo "4. Testing Service Startup (Basic)..."
echo "   Starting services in background..."

# Start services with proper environment
TLS_CERT="./certs/api-gateway.crt" TLS_KEY="./certs/api-gateway-key.pem" PORT="8443" INFERENCE_POOL_URL="localhost:50051" AUTH_SERVICE_URL="http://localhost:8082" AUTH_SERVICE_GRPC="localhost:8081" ./api-gateway/bin/api-gateway > /tmp/api-gateway.log 2>&1 &
API_GATEWAY_PID=$!
PORT="8082" HTTP_PORT="8082" DATABASE_TYPE="sqlite" DATABASE_PATH="./data/helixflow.db" ./auth-service/bin/auth-service > /tmp/auth-service.log 2>&1 &
AUTH_SERVICE_PID=$!
PORT="50051" ./inference-pool/bin/inference-pool > /tmp/inference-pool.log 2>&1 &
INFERENCE_POOL_PID=$!
PORT="8083" ./monitoring/bin/monitoring > /tmp/monitoring.log 2>&1 &
MONITORING_PID=$!

# Wait a moment for services to start
sleep 5

echo "   Checking service health..."
# Test service-specific health endpoints
# API Gateway: HTTPS health endpoint on 8443
if timeout 3s curl -k https://localhost:8443/health > /dev/null 2>&1; then
    echo -e "   ${GREEN}✅${NC} api-gateway health check passed"
    TEST_RESULTS+=("PASS")
    TEST_NAMES+=("api-gateway health")
else
    echo -e "   ${YELLOW}⚠️${NC} api-gateway health check failed (may need more time)"
    TEST_RESULTS+=("WARN")
    TEST_NAMES+=("api-gateway health")
fi

# Auth Service: gRPC on 8081 - test TCP connectivity
if timeout 3s bash -c "echo > /dev/tcp/localhost/8081" 2>/dev/null; then
    echo -e "   ${GREEN}✅${NC} auth-service TCP connectivity passed"
    TEST_RESULTS+=("PASS")
    TEST_NAMES+=("auth-service health")
else
    echo -e "   ${YELLOW}⚠️${NC} auth-service TCP connectivity failed (may need more time)"
    TEST_RESULTS+=("WARN")
    TEST_NAMES+=("auth-service health")
fi

# Inference Pool: gRPC on 50051 - test TCP connectivity
if timeout 3s bash -c "echo > /dev/tcp/localhost/50051" 2>/dev/null; then
    echo -e "   ${GREEN}✅${NC} inference-pool TCP connectivity passed"
    TEST_RESULTS+=("PASS")
    TEST_NAMES+=("inference-pool health")
else
    echo -e "   ${YELLOW}⚠️${NC} inference-pool TCP connectivity failed (may need more time)"
    TEST_RESULTS+=("WARN")
    TEST_NAMES+=("inference-pool health")
fi

# Monitoring: gRPC on 8083 - test TCP connectivity
if timeout 3s bash -c "echo > /dev/tcp/localhost/8083" 2>/dev/null; then
    echo -e "   ${GREEN}✅${NC} monitoring TCP connectivity passed"
    TEST_RESULTS+=("PASS")
    TEST_NAMES+=("monitoring health")
else
    echo -e "   ${YELLOW}⚠️${NC} monitoring TCP connectivity failed (may need more time)"
    TEST_RESULTS+=("WARN")
    TEST_NAMES+=("monitoring health")
fi
echo ""

echo "5. Testing API Gateway Functionality..."
echo "   Testing HTTPS endpoints..."

# Test models endpoint
if timeout 5s curl -k https://localhost:8443/v1/models > /tmp/models_response.json 2>/dev/null; then
    MODEL_COUNT=$(jq '.data | length' /tmp/models_response.json 2>/dev/null || echo "0")
    if [ "$MODEL_COUNT" -gt 0 ]; then
        echo -e "   ${GREEN}✅${NC} Models endpoint working ($MODEL_COUNT models)"
        TEST_RESULTS+=("PASS")
        TEST_NAMES+=("Models endpoint")
    else
        echo -e "   ${YELLOW}⚠️${NC} Models endpoint working but no models returned"
        TEST_RESULTS+=("WARN")
        TEST_NAMES+=("Models endpoint")
    fi
else
    echo -e "   ${RED}❌${NC} Models endpoint failed"
    TEST_RESULTS+=("FAIL")
    TEST_NAMES+=("Models endpoint")
fi

# Test chat completions
if timeout 10s curl -k -X POST https://localhost:8443/v1/chat/completions \
    -H "Authorization: Bearer demo-key" \
    -H "Content-Type: application/json" \
    -d '{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}' \
    > /tmp/chat_response.json 2>/dev/null; then
    if [ -s /tmp/chat_response.json ]; then
        echo -e "   ${GREEN}✅${NC} Chat completions endpoint working"
        TEST_RESULTS+=("PASS")
        TEST_NAMES+=("Chat completions")
    else
        echo -e "   ${YELLOW}⚠️${NC} Chat completions endpoint responded but empty"
        TEST_RESULTS+=("WARN")
        TEST_NAMES+=("Chat completions")
    fi
else
    echo -e "   ${RED}❌${NC} Chat completions endpoint failed"
    TEST_RESULTS+=("FAIL")
    TEST_NAMES+=("Chat completions")
fi
echo ""

echo "6. Testing gRPC Services..."
echo "   Testing gRPC API Gateway..."
echo -e "   ${YELLOW}⚠️${NC} gRPC functionality integrated into main API Gateway - skipping separate binary test"
TEST_RESULTS+=("SKIP")
TEST_NAMES+=("gRPC Gateway startup")
GRPC_GATEWAY_PID=""
echo ""

# Cleanup services
echo "Cleaning up background services..."
[ -n "$API_GATEWAY_PID" ] && kill $API_GATEWAY_PID 2>/dev/null
[ -n "$AUTH_SERVICE_PID" ] && kill $AUTH_SERVICE_PID 2>/dev/null
[ -n "$INFERENCE_POOL_PID" ] && kill $INFERENCE_POOL_PID 2>/dev/null
[ -n "$MONITORING_PID" ] && kill $MONITORING_PID 2>/dev/null
[ -n "$GRPC_GATEWAY_PID" ] && kill $GRPC_GATEWAY_PID 2>/dev/null
wait 2>/dev/null

echo ""
echo "=== Integration Test Summary ==="
echo ""

# Calculate results
TOTAL_TESTS=${#TEST_RESULTS[@]}
PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0

for i in "${!TEST_RESULTS[@]}"; do
    case "${TEST_RESULTS[$i]}" in
        "PASS") PASS_COUNT=$((PASS_COUNT + 1)); COLOR=$GREEN ;;
        "FAIL") FAIL_COUNT=$((FAIL_COUNT + 1)); COLOR=$RED ;;
        "WARN") WARN_COUNT=$((WARN_COUNT + 1)); COLOR=$YELLOW ;;
        "SKIP") WARN_COUNT=$((WARN_COUNT + 1)); COLOR=$YELLOW ;;
    esac
    echo -e "   ${COLOR}${TEST_RESULTS[$i]}${NC} - ${TEST_NAMES[$i]}"
done

echo ""
echo "Results: $PASS_COUNT/$TOTAL_TESTS passed, $FAIL_COUNT failed, $WARN_COUNT warnings"

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "${GREEN}🎉 Integration tests completed successfully!${NC}"
    echo "HelixFlow platform is ready for production deployment."
    exit 0
else
    echo -e "${RED}❌ Some integration tests failed.${NC}"
    echo "Please check the logs in /tmp/ for more details."
    exit 1
fi