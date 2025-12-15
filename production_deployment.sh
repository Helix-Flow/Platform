#!/bin/bash

echo "=== HelixFlow Platform - Production Deployment ==="
echo "Date: $(date)"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVICES_DIR="/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform"
LOGS_DIR="/var/log/helixflow"
PID_DIR="/var/run/helixflow"
CERTS_DIR="/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/certs"

# Create necessary directories
echo "1. Creating deployment directories..."
sudo mkdir -p $LOGS_DIR $PID_DIR
sudo chmod 755 $LOGS_DIR $PID_DIR
sudo chown $USER:$USER $LOGS_DIR $PID_DIR 2>/dev/null || true
echo -e "   ${GREEN}✅${NC} Directories created"
echo ""

# Function to start a service
start_service() {
    local service_name=$1
    local service_path=$2
    local service_port=$3
    local log_file="$LOGS_DIR/$service_name.log"
    local pid_file="$PID_DIR/$service_name.pid"
    
    echo "   Starting $service_name on port $service_port..."
    
    # Check if already running
    if [ -f "$pid_file" ] && kill -0 $(cat "$pid_file") 2>/dev/null; then
        echo -e "   ${YELLOW}⚠️${NC} $service_name already running (PID: $(cat $pid_file))"
        return 0
    fi
    
    # Start service
    cd "$SERVICES_DIR/$service_path"
    nohup ./bin/$service_name > "$log_file" 2>&1 &
    local pid=$!
    echo $pid > "$pid_file"
    
    # Wait for service to start
    sleep 2
    
    # Check if service started successfully
    if kill -0 $pid 2>/dev/null; then
        echo -e "   ${GREEN}✅${NC} $service_name started (PID: $pid)"
        return 0
    else
        echo -e "   ${RED}❌${NC} $service_name failed to start"
        echo "   Check logs: tail -20 $log_file"
        return 1
    fi
}

# Function to stop a service
stop_service() {
    local service_name=$1
    local pid_file="$PID_DIR/$service_name.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if kill -0 $pid 2>/dev/null; then
            echo "   Stopping $service_name (PID: $pid)..."
            kill $pid
            rm -f "$pid_file"
            echo -e "   ${GREEN}✅${NC} $service_name stopped"
        else
            echo -e "   ${YELLOW}⚠️${NC} $service_name not running"
        fi
    else
        echo -e "   ${YELLOW}⚠️${NC} $service_name not running"
    fi
}

# Function to check service health
check_health() {
    local service_name=$1
    local health_url=$2
    local timeout=5
    
    echo "   Checking $service_name health..."
    
    if timeout $timeout curl -k -s -f "$health_url" > /dev/null 2>&1; then
        echo -e "   ${GREEN}✅${NC} $service_name healthy"
        return 0
    else
        echo -e "   ${RED}❌${NC} $service_name unhealthy"
        return 1
    fi
}

# Function to show service status
show_status() {
    echo ""
    echo -e "${BLUE}=== Service Status ===${NC}"
    echo ""
    
    local services=("api-gateway" "auth-service" "inference-pool" "monitoring")
    local ports=("8443" "8081" "50051" "8083")
    
    for i in "${!services[@]}"; do
        local service=${services[$i]}
        local port=${ports[$i]}
        local pid_file="$PID_DIR/$service.pid"
        local status="${RED}STOPPED${NC}"
        
        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if kill -0 $pid 2>/dev/null; then
                status="${GREEN}RUNNING${NC} (PID: $pid, Port: $port)"
            else
                status="${RED}CRASHED${NC} (PID: $pid)"
            fi
        fi
        
        printf "%-20s %s\n" "$service:" "$status"
    done
    
}

# Main deployment logic
case "${1:-deploy}" in
    deploy)
        echo -e "${BLUE}🚀 Starting HelixFlow Platform Deployment${NC}"
        echo ""
        
        # Start services in order
        echo "2. Starting core services..."
        
        # Start monitoring first (for metrics collection)
        start_service "monitoring" "monitoring" "8083"
        
        # Start auth service
        start_service "auth-service" "auth-service" "8081"
        
        # Start inference pool
        start_service "inference-pool" "inference-pool" "50051"
        
        # Start API gateway (HTTP)
        start_service "api-gateway" "api-gateway" "8443"
        
        
        echo ""
        echo "3. Waiting for services to stabilize..."
        sleep 5
        
        echo ""
        echo "4. Performing health checks..."
        
        # Health checks
        check_health "monitoring" "http://localhost:8083/health"
        check_health "auth-service" "http://localhost:8081/health"
        check_health "inference-pool" "http://localhost:50051/health"
        check_health "api-gateway" "https://localhost:8443/health" || true
        
        echo ""
        show_status
        
        echo ""
        echo -e "${GREEN}🎉 Deployment completed!${NC}"
        echo ""
        echo "API Endpoints:"
        echo "  HTTP API Gateway:  https://localhost:8443"
        echo "  gRPC API Gateway:  https://localhost:9443"
        echo "  Auth Service:      http://localhost:8081"
        echo "  Inference Pool:    http://localhost:50051"
        echo "  Monitoring:        http://localhost:8083"
        echo ""
        echo "Key endpoints:"
        echo "  Health Check:      https://localhost:8443/health"
        echo "  Models List:       https://localhost:8443/v1/models"
        echo "  Chat Completions:  https://localhost:8443/v1/chat/completions"
        echo ""
        echo "Logs directory: $LOGS_DIR"
        echo "PID files: $PID_DIR"
        ;;
        
    stop)
        echo -e "${BLUE}🛑 Stopping HelixFlow Platform${NC}"
        echo ""
        
        # Stop services in reverse order
        stop_service "api-gateway"
        stop_service "inference-pool"
        stop_service "auth-service"
        stop_service "monitoring"
        
        echo ""
        echo -e "${GREEN}✅ All services stopped${NC}"
        ;;
        
    status)
        show_status
        ;;
        
    restart)
        $0 stop
        sleep 2
        $0 deploy
        ;;
        
    logs)
        local service=${2:-""}
        if [ -n "$service" ]; then
            tail -f "$LOGS_DIR/$service.log"
        else
            echo "Usage: $0 logs <service>"
            echo "Available services: api-gateway, auth-service, inference-pool, monitoring"
        fi
        ;;
        
    test)
        echo -e "${BLUE}🧪 Running integration tests${NC}"
        echo ""
        
        # Test basic connectivity
        echo "Testing service connectivity..."
        
        # Test database
        cd "$SERVICES_DIR/test/db_test"
        if go run simple_check.go > /dev/null 2>&1; then
            echo -e "   ${GREEN}✅${NC} Database connectivity"
        else
            echo -e "   ${RED}❌${NC} Database connectivity failed"
        fi
        cd "$SERVICES_DIR"
        
        # Test API Gateway
        if curl -k -s -f "https://localhost:8443/health" > /dev/null 2>&1; then
            echo -e "   ${GREEN}✅${NC} API Gateway (HTTPS)"
        else
            echo -e "   ${RED}❌${NC} API Gateway (HTTPS) failed"
        fi
        
        # Test models endpoint
        if curl -k -s -f "https://localhost:8443/v1/models" > /dev/null 2>&1; then
            echo -e "   ${GREEN}✅${NC} Models endpoint"
        else
            echo -e "   ${RED}❌${NC} Models endpoint failed"
        fi
        
        # Test chat completions
        echo "   Testing chat completions..."
        if curl -k -s -X POST "https://localhost:8443/v1/chat/completions" \
            -H "Authorization: Bearer demo-key" \
            -H "Content-Type: application/json" \
            -d '{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}' \
            > /tmp/test_response.json 2>/dev/null; then
            if [ -s /tmp/test_response.json ]; then
                echo -e "   ${GREEN}✅${NC} Chat completions (response received)"
            else
                echo -e "   ${YELLOW}⚠️${NC} Chat completions (empty response)"
            fi
        else
            echo -e "   ${RED}❌${NC} Chat completions failed"
        fi
        
        echo ""
        echo "Integration test completed. Check individual endpoints for detailed testing."
        ;;
        
    *)
        echo "Usage: $0 {deploy|stop|status|restart|logs|test}"
        echo ""
        echo "Commands:"
        echo "  deploy  - Start all services"
        echo "  stop    - Stop all services"
        echo "  status  - Show service status"
        echo "  restart - Restart all services"
        echo "  logs    - Show logs for a service"
        echo "  test    - Run integration tests"
        echo ""
        echo "Examples:"
        echo "  $0 deploy"
        echo "  $0 status"
        echo "  $0 logs api-gateway"
        echo "  $0 test"
        exit 1
        ;;
esac