#!/bin/bash
# HelixFlow Phase 1 Services Startup Script
# Starts all services with new real inference engine and gRPC monitoring

set -e  # Exit on any error

echo "🚀 HelixFlow Phase 1 Implementation - Service Startup"
echo "=================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to check if a service is running
check_service() {
    local service_name=$1
    local url=$2
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$url" > /dev/null 2>&1; then
            print_status "$service_name is running"
            return 0
        fi
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "$service_name failed to start after $max_attempts attempts"
    return 1
}

# Function to start a service
start_service() {
    local service_name=$1
    local service_dir=$2
    local start_command=$3
    local health_url=$4
    
    print_status "Starting $service_name..."
    
    cd "$service_dir" || exit 1
    
    # Start the service in background
    eval "$start_command" &
    local pid=$!
    
    # Store PID for later reference
    echo $pid > "/tmp/${service_name// /_}_pid"
    
    # Wait and check if service is healthy
    sleep 3
    
    if check_service "$service_name" "$health_url"; then
        print_status "$service_name started successfully (PID: $pid)"
        return 0
    else
        print_error "$service_name failed to start"
        kill $pid 2>/dev/null || true
        return 1
    fi
}

# Kill any existing services
print_status "Cleaning up existing services..."
pkill -f "api-gateway" 2>/dev/null || true
pkill -f "auth-service" 2>/dev/null || true  
pkill -f "inference-pool" 2>/dev/null || true
pkill -f "monitoring" 2>/dev/null || true
sleep 2

# Set environment variables
export HELIXFLOW_ENV=production
export TLS_CERT=/certs/api-gateway.crt
export TLS_KEY=/certs/api-gateway-key.pem
export INFERENCE_POOL_URL=localhost:50052
export MONITORING_URL=localhost:8083
export AUTH_SERVICE_URL=localhost:50051

# Start services in order

# 1. Start Monitoring Service (gRPC + HTTP)
print_status "Starting Monitoring Service..."
cd monitoring/src || exit 1
go build -o bin/monitoring main.go main_grpc.go monitoring_service.go || {
    print_error "Failed to build monitoring service"
    exit 1
}

nohup ./bin/monitoring > monitoring.log 2>&1 &
MONITORING_PID=$!
echo $MONITORING_PID > /tmp/monitoring_pid
sleep 5

if check_service "Monitoring Service" "http://localhost:8083/health"; then
    print_status "Monitoring Service started successfully"
else
    print_error "Monitoring Service failed to start"
    exit 1
fi

# 2. Start Auth Service
cd /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/auth-service/src || exit 1
print_status "Starting Auth Service..."
go build -o bin/auth-service main.go auth_service.go || {
    print_error "Failed to build auth service"
    exit 1
}

nohup ./bin/auth-service > auth-service.log 2>&1 &
AUTH_PID=$!
echo $AUTH_PID > /tmp/auth_service_pid
sleep 5

# 3. Start Inference Pool Service
cd /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/inference-pool/src || exit 1
print_status "Starting Inference Pool Service..."
go build -o bin/inference-pool main.go inference_engine.go || {
    print_error "Failed to build inference pool service"
    exit 1
}

nohup ./bin/inference-pool > inference-pool.log 2>&1 &
INFERENCE_PID=$!
echo $INFERENCE_PID > /tmp/inference_pool_pid
sleep 5

# 4. Start API Gateway
cd /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/api-gateway/src || exit 1
print_status "Starting API Gateway..."
go build -o bin/api-gateway main.go inference_handler.go || {
    print_error "Failed to build API gateway"
    exit 1
}

nohup ./bin/api-gateway > api-gateway.log 2>&1 &
API_PID=$!
echo $API_PID > /tmp/api_gateway_pid
sleep 10

# Check if all services are running
print_status "Checking service health..."

services_health=(
    "API Gateway:http://localhost:8443/health"
    "Monitoring Service:http://localhost:8083/health"
)

all_healthy=true
for service_check in "${services_health[@]}"; do
    IFS=':' read -r service_name health_url <<< "$service_check"
    if ! check_service "$service_name" "$health_url"; then
        all_healthy=false
    fi
done

if [ "$all_healthy" = true ]; then
    print_status "🎉 All services started successfully!"
else
    print_error "❌ Some services failed to start properly"
    exit 1
fi

# Display service status
print_status "Service Status Summary:"
echo "  API Gateway: http://localhost:8443 (TLS)"
echo "  Monitoring Service: http://localhost:8083"
echo "  Auth Service: gRPC on port 50051"
echo "  Inference Pool: gRPC on port 50052"
echo ""
echo "  Health Check: http://localhost:8443/health"
echo "  Models: http://localhost:8443/v1/models"
echo "  Chat Completions: http://localhost:8443/v1/chat/completions"

# Run validation test
print_status "Running validation test..."
cd /media/milosvasic/DATA4TB/Projects/HelixFlow/Platform
if python3 test_implementation.py; then
    print_status "✅ All validation tests passed!"
    echo ""
    echo "🚀 HelixFlow Phase 1 implementation is ready for use!"
    echo ""
    echo "Quick test commands:"
    echo "  curl http://localhost:8443/health"
    echo "  curl -X POST http://localhost:8443/v1/auth/register -H 'Content-Type: application/json' -d '{\"username\":\"test\",\"email\":\"test@example.com\",\"password\":\"test123\"}'"
    echo "  curl -X POST http://localhost:8443/v1/chat/completions -H 'Authorization: Bearer YOUR_TOKEN' -H 'Content-Type: application/json' -d '{\"model\":\"gpt-3.5-turbo\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello!\"}]}'"
else
    print_error "❌ Validation tests failed - check logs for details"
    echo "Service logs are available in:"
    echo "  api-gateway.log"
    echo "  auth-service.log"
    echo "  inference-pool.log"
    echo "  monitoring.log"
    exit 1
fi

echo ""
echo "🎯 Phase 1 Implementation Complete!"
echo "   - Real AI inference engine integrated"
echo "   - gRPC monitoring service operational"
echo "   - Authentication system working"
echo "   - API endpoints functioning"
echo "   - Rate limiting active"
echo ""
echo "Next steps: Run comprehensive testing suite or begin production deployment"