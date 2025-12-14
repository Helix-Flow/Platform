#!/bin/bash

# HelixFlow Service Startup Script
# This script starts all HelixFlow services with proper configuration

echo "🚀 Starting HelixFlow Platform Services..."

# Set environment variables
export DATABASE_TYPE=sqlite
export DATABASE_PATH=/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/data/helixflow.db
export REDIS_HOST=localhost
export REDIS_PORT=6379
export HTTP_PORT=8082

# Create logs directory
mkdir -p /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs

# Function to start a service
start_service() {
    local service_name=$1
    local service_binary=$2
    local service_port=$3
    
    echo "Starting $service_name on port $service_port..."
    
    # Start the service in background
    $service_binary > /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/$service_name.log 2>&1 &
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

# Kill any existing services first
echo "Stopping any existing HelixFlow services..."
pkill -f "auth-service" 2>/dev/null
pkill -f "inference-pool" 2>/dev/null
pkill -f "monitoring" 2>/dev/null
pkill -f "api-gateway" 2>/dev/null
sleep 2

# Start services using pre-built binaries
start_service "auth-service" "/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/auth-service/bin/auth-service" "8081"
start_service "inference-pool" "/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/inference-pool/bin/inference-pool" "50051"
start_service "monitoring" "/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/monitoring/bin/monitoring" "8083"
start_service "api-gateway" "/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/api-gateway/bin/api-gateway" "8443"

echo ""
echo "🎯 All services started!"
echo ""
echo "Service Status:"
echo "- Auth Service: gRPC on localhost:8081, HTTP on localhost:8082"
echo "- Inference Pool: gRPC on localhost:50051" 
echo "- Monitoring: gRPC on localhost:8083"
echo "- API Gateway: HTTPS on https://localhost:8443/health"
echo ""
echo "📊 Check logs:"
echo "- tail -f /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/auth-service.log"
echo "- tail -f /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/inference-pool.log"
echo "- tail -f /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/monitoring.log"
echo "- tail -f /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/api-gateway.log"
echo ""
echo "🛑 To stop all services: kill \$(cat /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/logs/service_pids.txt) 2>/dev/null"
echo ""
echo "🔧 Testing endpoints:"
echo "- curl -k https://localhost:8443/v1/models"
echo "- curl -k https://localhost:8443/v1/chat/completions -H 'Content-Type: application/json' -d '{\"model\":\"gpt-3.5-turbo\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello!\"}]}'"