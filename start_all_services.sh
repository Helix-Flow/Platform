#!/bin/bash

# HelixFlow Service Startup Script
# This script starts all HelixFlow services with proper configuration

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🚀 Starting HelixFlow Platform Services..."

# Set shared environment variables (do NOT export TLS_CERT/TLS_KEY globally)
export DATABASE_TYPE=sqlite
export DB_PATH=../data/helixflow.db
export REDIS_HOST=localhost:6379
export REDIS_PORT=6379
export HTTP_PORT=8082
export MONITORING_GRPC_PORT=50055
export MONITORING_PORT=8083
export INFERENCE_POOL_URL=localhost:50051
export AUTH_SERVICE_GRPC=localhost:8081
export AUTH_SERVICE_URL=localhost:8081

# Create logs directory
mkdir -p logs

# Clear old PID file
> logs/service_pids.txt

# Function to start a service
start_service() {
    local service_name=$1
    local service_dir=$2
    shift 2
    local service_cmd="$@"

    echo "Starting $service_name..."

    cd "$service_dir" && eval "$service_cmd >> ../logs/$service_name.log 2>&1 &"
    local pid=$!

    sleep 3

    if kill -0 $pid 2>/dev/null; then
        echo "✅ $service_name started successfully (PID: $pid)"
        echo $pid >> ../logs/service_pids.txt
    else
        echo "❌ $service_name failed to start"
        echo "Check logs at: ../logs/$service_name.log"
    fi
    
    cd "$SCRIPT_DIR"
}

# Kill any existing services first
echo "Stopping any existing HelixFlow services..."
pkill -f "auth-service" 2>/dev/null
pkill -f "inference-pool" 2>/dev/null
pkill -f "monitoring" 2>/dev/null
pkill -f "api-gateway" 2>/dev/null
sleep 2

# Start services - each with their own PORT and TLS settings
# Auth service: no TLS_CERT/TLS_KEY so gRPC uses plaintext (gateway connects insecure)
start_service "auth-service" "auth-service" "PORT=8081 ./bin/auth-service"
start_service "inference-pool" "inference-pool" "PORT=50051 ./bin/inference-pool"
start_service "monitoring" "monitoring" "MONITORING_TLS_CERT=../certs/monitoring.crt MONITORING_TLS_KEY=../certs/monitoring-key.pem PORT=8083 MONITORING_GRPC_PORT=50055 ./bin/monitoring"
start_service "api-gateway" "api-gateway" "TLS_CERT=../certs/api-gateway.crt TLS_KEY=../certs/api-gateway-key.pem INFERENCE_POOL_URL=localhost:50051 AUTH_SERVICE_GRPC=localhost:8081 PORT=8443 ./bin/api-gateway"

echo ""
echo "🎯 All services started!"
echo ""
echo "Service Status:"
echo "- Auth Service: gRPC on localhost:8081, HTTP on localhost:8082"
echo "- Inference Pool: gRPC on localhost:50051" 
echo "- Monitoring: HTTP on localhost:8083, gRPC on localhost:50055"
echo "- API Gateway: HTTPS on https://localhost:8443/health"
echo ""
echo "📊 Check logs:"
echo "- tail -f logs/auth-service.log"
echo "- tail -f logs/inference-pool.log"
echo "- tail -f logs/monitoring.log"
echo "- tail -f logs/api-gateway.log"
echo ""
echo "🛑 To stop all services: kill \$(cat logs/service_pids.txt) 2>/dev/null"
