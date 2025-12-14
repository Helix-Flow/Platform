#!/bin/bash

# HelixFlow Service Startup Script
# This script starts all HelixFlow services with proper configuration

echo "🚀 Starting HelixFlow Platform Services..."

# Set environment variables
export DATABASE_TYPE=sqlite
export DATABASE_PATH=/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/data/helixflow.db
export REDIS_HOST=localhost
export REDIS_PORT=6379
export TLS_CERT=/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/certs/api-gateway.crt
export TLS_KEY=/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/certs/api-gateway-key.pem

# Create logs directory
mkdir -p /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs

# Function to start a service
start_service() {
    local service_name=$1
    local service_path=$2
    local service_port=$3
    local service_cmd=$4
    
    echo "Starting $service_name on port $service_port..."
    cd $service_path
    
    # Start the service in background
    $service_cmd > /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/$service_name.log 2>&1 &
    local pid=$!
    
    # Wait a moment to check if it started successfully
    sleep 2
    
    if kill -0 $pid 2>/dev/null; then
        echo "✅ $service_name started successfully (PID: $pid)"
        echo $pid >> /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/service_pids.txt
    else
        echo "❌ $service_name failed to start"
        echo "Check logs at: /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/$service_name.log"
    fi
}

# Start services
start_service "auth-service" "/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/auth-service/src" "8081" "go run main.go auth_service.go"
start_service "inference-pool" "/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/inference-pool/src" "8082" "go run main.go inference_engine.go gpu_optimizer.go quantization.go"
start_service "monitoring" "/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/monitoring/src" "8083" "go run main.go monitoring_service.go main_grpc.go"
start_service "api-gateway" "/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/api-gateway/src" "8443" "go run main.go main_grpc.go inference_handler.go websocket_handler.go rate_limiter_advanced.go"

echo ""
echo "🎯 All services started!"
echo ""
echo "Service Status:"
echo "- Auth Service: http://localhost:8081/health"
echo "- Inference Pool: http://localhost:8082/health" 
echo "- Monitoring: http://localhost:8083/health"
echo "- API Gateway: https://localhost:8443/health"
echo ""
echo "📊 Check logs:"
echo "- tail -f /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/auth-service.log"
echo "- tail -f /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/inference-pool.log"
echo "- tail -f /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/monitoring.log"
echo "- tail -f /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/api-gateway.log"
echo ""
echo "🛑 To stop all services: kill $(cat /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/service_pids.txt)"
echo ""
echo "🔧 Testing endpoints:"
echo "- curl -k https://localhost:8443/v1/models"
echo "- curl -k https://localhost:8443/v1/chat/completions -H 'Content-Type: application/json' -d '{\"model\":\"gpt-3.5-turbo\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello!\"}]}'"