#!/bin/bash

# HelixFlow Development Environment Startup Script
# Starts all services with proper configuration for local development

set -e

echo "🔄 Starting HelixFlow Development Environment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Kill any existing services
echo -e "${YELLOW}Stopping any existing services...${NC}"
pkill -f "api-gateway" || true
pkill -f "auth-service" || true  
pkill -f "inference-pool" || true
pkill -f "monitoring" || true

sleep 2

# Create logs directory
mkdir -p logs

# Start services with proper environment variables
echo -e "${GREEN}Starting services...${NC}"

# 1. Start Auth Service (gRPC: 8081, HTTP: 8082)
echo -e "${YELLOW}Starting Auth Service...${NC}"
PORT="8081" HTTP_PORT="8082" DATABASE_TYPE="sqlite" DATABASE_PATH="../data/helixflow.db" ./auth-service/bin/auth-service > logs/auth-service.log 2>&1 &
AUTH_PID=$!
echo "Auth Service PID: $AUTH_PID"

# 2. Start Inference Pool (gRPC: 50051)
echo -e "${YELLOW}Starting Inference Pool...${NC}"
PORT="50051" ./inference-pool/bin/inference-pool > logs/inference-pool.log 2>&1 &
INFERENCE_PID=$!
echo "Inference Pool PID: $INFERENCE_PID"

# 3. Start Monitoring (gRPC: 8083)
echo -e "${YELLOW}Starting Monitoring Service...${NC}"
PORT="8083" ./monitoring/bin/monitoring > logs/monitoring.log 2>&1 &
MONITORING_PID=$!
echo "Monitoring Service PID: $MONITORING_PID"

# 4. Start API Gateway (HTTPS: 8443)
sleep 3

echo -e "${YELLOW}Starting API Gateway...${NC}"
TLS_CERT="./certs/api-gateway.crt" TLS_KEY="./certs/api-gateway-key.pem" PORT="8443" INFERENCE_POOL_URL="localhost:50051" AUTH_SERVICE_URL="http://localhost:8082" AUTH_SERVICE_GRPC="localhost:8081" ./api-gateway/bin/api-gateway > logs/api-gateway.log 2>&1 &
API_PID=$!
echo "API Gateway PID: $API_PID"

# Save PIDs to file
echo $AUTH_PID > logs/service_pids.txt
echo $INFERENCE_PID >> logs/service_pids.txt
echo $MONITORING_PID >> logs/service_pids.txt
echo $API_PID >> logs/service_pids.txt

# Wait for services to start
echo -e "${YELLOW}Waiting for services to initialize...${NC}"
sleep 5

# Test service health
echo -e "${GREEN}Testing service health...${NC}"

# Test Auth Service HTTP endpoint
if curl -s http://localhost:8082/health > /dev/null; then
    echo -e "${GREEN}✅ Auth Service (HTTP) is healthy${NC}"
else
    echo -e "${RED}❌ Auth Service (HTTP) health check failed${NC}"
fi

# Test API Gateway HTTPS endpoint
if curl -s -k https://localhost:8443/health > /dev/null; then
    echo -e "${GREEN}✅ API Gateway (HTTPS) is healthy${NC}"
else
    echo -e "${RED}❌ API Gateway (HTTPS) health check failed${NC}"
fi

# Test endpoints
echo -e "${GREEN}Testing API endpoints...${NC}"

# Test models endpoint
if curl -s -k https://localhost:8443/v1/models > /dev/null; then
    echo -e "${GREEN}✅ Models endpoint is accessible${NC}"
else
    echo -e "${RED}❌ Models endpoint failed${NC}"
fi

# Test auth endpoint
if curl -s http://localhost:8082/health > /dev/null; then
    echo -e "${GREEN}✅ Auth endpoints are accessible${NC}"
else
    echo -e "${RED}❌ Auth endpoints failed${NC}"
fi

echo ""
echo -e "${GREEN}🎉 HelixFlow Development Environment Started!${NC}"
echo ""
echo "Service URLs:"
echo "  🔐 API Gateway: https://localhost:8443"
echo "  🔑 Auth Service: http://localhost:8082"
echo "  🤖 Inference Pool: localhost:50051"
echo "  📊 Monitoring: localhost:8083"
echo ""
echo "API Endpoints:"
echo "  GET  /health                 - Health check"
echo "  GET  /v1/models            - List available models"
echo "  POST /v1/chat/completions  - Chat completions"
echo ""
echo "Log files available in: logs/"
echo "To stop services: pkill -f 'api-gateway|auth-service|inference-pool|monitoring'"
echo ""

# Keep script running
echo -e "${YELLOW}Services are running in background. Press Ctrl+C to stop.${NC}"

# Function to cleanup on exit
cleanup() {
    echo -e "${YELLOW}Stopping services...${NC}"
    kill $AUTH_PID 2>/dev/null || true
    kill $INFERENCE_PID 2>/dev/null || true
    kill $MONITORING_PID 2>/dev/null || true
    kill $API_PID 2>/dev/null || true
    echo -e "${GREEN}Services stopped.${NC}"
    exit 0
}

# Trap SIGINT (Ctrl+C) and cleanup
trap cleanup SIGINT

# Wait indefinitely
while true; do
    sleep 60
    # Optional: Add periodic health checks here
    if ! curl -s -k https://localhost:8443/health > /dev/null; then
        echo -e "${RED}⚠️  API Gateway health check failed, might need restart${NC}"
    fi
done