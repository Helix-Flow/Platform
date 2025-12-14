#!/bin/bash

set -e

echo "🏗️ HelixFlow Production Infrastructure Setup"
echo "=============================================="

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-helixflow}"
DB_USER="${DB_USER:-helixflow}"
DB_PASSWORD="${DB_PASSWORD:-helixflow123}"
CERT_DIR="${CERT_DIR:-./certs}"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Function to print status
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if PostgreSQL is running
    if ! command -v pg_isready &> /dev/null; then
        print_error "PostgreSQL client tools not found. Please install postgresql-client"
        exit 1
    fi
    
    if ! pg_isready -h $DB_HOST -p $DB_PORT -U postgres &> /dev/null; then
        print_error "PostgreSQL is not running on $DB_HOST:$DB_PORT"
        print_warning "Please start PostgreSQL or run it via Docker"
        exit 1
    fi
    
    # Check if OpenSSL is available
    if ! command -v openssl &> /dev/null; then
        print_error "OpenSSL is not installed"
        exit 1
    fi
    
    # Check if Redis is running
    if ! command -v redis-cli &> /dev/null; then
        print_warning "Redis client not found. Redis will be optional for now."
    fi
    
    print_success "Prerequisites check passed"
}

# Setup PostgreSQL
setup_postgresql() {
    print_status "Setting up PostgreSQL database..."
    
    if [ -f "./scripts/setup_postgresql.sh" ]; then
        ./scripts/setup_postgresql.sh
        print_success "PostgreSQL setup completed"
    else
        print_error "PostgreSQL setup script not found"
        exit 1
    fi
}

# Generate TLS certificates
setup_tls_certificates() {
    print_status "Generating TLS certificates..."
    
    if [ -f "./scripts/generate_certificates.sh" ]; then
        ./scripts/generate_certificates.sh
        print_success "TLS certificates generated"
    else
        print_error "Certificate generation script not found"
        exit 1
    fi
}

# Update service configurations
update_service_configs() {
    print_status "Updating service configurations..."
    
    # API Gateway configuration
    cat > api-gateway/config.env << EOF
# Database Configuration
DB_HOST=$DB_HOST
DB_PORT=$DB_PORT
DB_NAME=$DB_NAME
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD

# Redis Configuration
REDIS_HOST=$REDIS_HOST
REDIS_PORT=$REDIS_PORT

# TLS Configuration
TLS_CERT=$CERT_DIR/api-gateway.crt
TLS_KEY=$CERT_DIR/api-gateway-key.pem
CA_CERT=$CERT_DIR/helixflow-ca.pem

# Service URLs
INFERENCE_POOL_URL=localhost:50051
AUTH_SERVICE_URL=localhost:8081
MONITORING_SERVICE_URL=localhost:8083

# JWT Configuration
JWT_PRIVATE_KEY=$CERT_DIR/jwt-private.pem
JWT_PUBLIC_KEY=$CERT_DIR/jwt-public.pem

# API Configuration
PORT=8443
LOG_LEVEL=info
MAX_REQUEST_SIZE=10485760
RATE_LIMIT_PER_MINUTE=1000
EOF

    # Auth Service configuration
    cat > auth-service/config.env << EOF
# Database Configuration
DB_HOST=$DB_HOST
DB_PORT=$DB_PORT
DB_NAME=$DB_NAME
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD

# Redis Configuration
REDIS_HOST=$REDIS_HOST
REDIS_PORT=$REDIS_PORT

# TLS Configuration
TLS_CERT=$CERT_DIR/auth-service.crt
TLS_KEY=$CERT_DIR/auth-service-key.pem
CA_CERT=$CERT_DIR/helixflow-ca.pem

# JWT Configuration
JWT_PRIVATE_KEY=$CERT_DIR/jwt-private.pem
JWT_PUBLIC_KEY=$CERT_DIR/jwt-public.pem

# Service Configuration
PORT=8081
LOG_LEVEL=info
JWT_EXPIRY_HOURS=24
REFRESH_TOKEN_EXPIRY_DAYS=7
MAX_LOGIN_ATTEMPTS=5
EOF

    # Inference Pool configuration
    cat > inference-pool/config.env << EOF
# Database Configuration
DB_HOST=$DB_HOST
DB_PORT=$DB_PORT
DB_NAME=$DB_NAME
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD

# Redis Configuration
REDIS_HOST=$REDIS_HOST
REDIS_PORT=$REDIS_PORT

# TLS Configuration
TLS_CERT=$CERT_DIR/inference-pool.crt
TLS_KEY=$CERT_DIR/inference-pool-key.pem
CA_CERT=$CERT_DIR/helixflow-ca.pem

# Service Configuration
PORT=50051
LOG_LEVEL=info
MAX_WORKERS=10
JOB_QUEUE_SIZE=1000
GPU_MEMORY_THRESHOLD=0.9
MODEL_CACHE_SIZE=5
EOF

    # Monitoring Service configuration
    cat > monitoring/config.env << EOF
# Database Configuration
DB_HOST=$DB_HOST
DB_PORT=$DB_PORT
DB_NAME=$DB_NAME
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD

# Redis Configuration
REDIS_HOST=$REDIS_HOST
REDIS_PORT=$REDIS_PORT

# TLS Configuration
TLS_CERT=$CERT_DIR/monitoring.crt
TLS_KEY=$CERT_DIR/monitoring-key.pem
CA_CERT=$CERT_DIR/helixflow-ca.pem

# Service Configuration
PORT=8083
LOG_LEVEL=info
METRICS_RETENTION_DAYS=30
ALERT_CHECK_INTERVAL=60
PROMETHEUS_ENDPOINT=http://localhost:9090
GRAFANA_ENDPOINT=http://localhost:3000
EOF

    print_success "Service configurations updated"
}

# Create systemd service files
create_systemd_services() {
    print_status "Creating systemd service files..."
    
    # API Gateway service
    cat > /etc/systemd/system/helixflow-api-gateway.service << EOF
[Unit]
Description=HelixFlow API Gateway
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=helixflow
Group=helixflow
WorkingDirectory=/opt/helixflow/api-gateway
ExecStart=/opt/helixflow/api-gateway/bin/api-gateway
Restart=always
RestartSec=10
EnvironmentFile=/opt/helixflow/api-gateway/config.env

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/helixflow/api-gateway/logs

# Resource limits
LimitNOFILE=65536
LimitNPROC=32768

[Install]
WantedBy=multi-user.target
EOF

    # Auth Service service
    cat > /etc/systemd/system/helixflow-auth-service.service << EOF
[Unit]
Description=HelixFlow Auth Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=helixflow
Group=helixflow
WorkingDirectory=/opt/helixflow/auth-service
ExecStart=/opt/helixflow/auth-service/bin/auth-service
Restart=always
RestartSec=10
EnvironmentFile=/opt/helixflow/auth-service/config.env

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/helixflow/auth-service/logs

# Resource limits
LimitNOFILE=65536
LimitNPROC=32768

[Install]
WantedBy=multi-user.target
EOF

    # Inference Pool service
    cat > /etc/systemd/system/helixflow-inference-pool.service << EOF
[Unit]
Description=HelixFlow Inference Pool
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=helixflow
Group=helixflow
WorkingDirectory=/opt/helixflow/inference-pool
ExecStart=/opt/helixflow/inference-pool/bin/inference-pool
Restart=always
RestartSec=10
EnvironmentFile=/opt/helixflow/inference-pool/config.env

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/helixflow/inference-pool/logs

# GPU access (if needed)
DeviceAllow=/dev/nvidia* rw
SupplementaryGroups=video

# Resource limits
LimitNOFILE=65536
LimitNPROC=32768

[Install]
WantedBy=multi-user.target
EOF

    # Monitoring Service service
    cat > /etc/systemd/system/helixflow-monitoring.service << EOF
[Unit]
Description=HelixFlow Monitoring Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=helixflow
Group=helixflow
WorkingDirectory=/opt/helixflow/monitoring
ExecStart=/opt/helixflow/monitoring/bin/monitoring
Restart=always
RestartSec=10
EnvironmentFile=/opt/helixflow/monitoring/config.env

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/helixflow/monitoring/logs

# Resource limits
LimitNOFILE=65536
LimitNPROC=32768

[Install]
WantedBy=multi-user.target
EOF

    # Reload systemd
    systemctl daemon-reload
    
    print_success "Systemd service files created"
}

# Create Docker Compose file for development
create_docker_compose() {
    print_status "Creating Docker Compose configuration..."
    
    cat > docker-compose.yml << EOF
version: '3.8'

services:
  postgres:
    image: postgres:15
    container_name: helixflow-postgres
    environment:
      POSTGRES_DB: $DB_NAME
      POSTGRES_USER: $DB_USER
      POSTGRES_PASSWORD: $DB_PASSWORD
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./schemas/postgresql-helixflow.sql:/docker-entrypoint-initdb.d/init.sql
    networks:
      - helixflow-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $DB_USER -d $DB_NAME"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: helixflow-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    networks:
      - helixflow-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  prometheus:
    image: prom/prometheus:latest
    container_name: helixflow-prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    networks:
      - helixflow-network
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'

  grafana:
    image: grafana/grafana:latest
    container_name: helixflow-grafana
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin123
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./monitoring/grafana/datasources:/etc/grafana/provisioning/datasources
    networks:
      - helixflow-network

networks:
  helixflow-network:
    driver: bridge

volumes:
  postgres_data:
  redis_data:
  prometheus_data:
  grafana_data:
EOF

    # Create Prometheus configuration
    mkdir -p monitoring
    cat > monitoring/prometheus.yml << EOF
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'api-gateway'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 5s

  - job_name: 'auth-service'
    static_configs:
      - targets: ['localhost:8081']
    metrics_path: '/metrics'
    scrape_interval: 5s

  - job_name: 'inference-pool'
    static_configs:
      - targets: ['localhost:50051']
    metrics_path: '/metrics'
    scrape_interval: 5s

  - job_name: 'monitoring-service'
    static_configs:
      - targets: ['localhost:8083']
    metrics_path: '/metrics'
    scrape_interval: 5s

  - job_name: 'postgres'
    static_configs:
      - targets: ['localhost:9187']
    metrics_path: '/metrics'
    scrape_interval: 15s

  - job_name: 'redis'
    static_configs:
      - targets: ['localhost:9121']
    metrics_path: '/metrics'
    scrape_interval: 15s
EOF

    print_success "Docker Compose configuration created"
}

# Create comprehensive test suite
create_test_suite() {
    print_status "Creating comprehensive test suite..."
    
    cat > scripts/integration_test_production.sh << 'EOF'
#!/bin/bash

set -e

echo "🧪 HelixFlow Production Integration Test"
echo "========================================="

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-helixflow}"
DB_USER="${DB_USER:-helixflow}"
DB_PASSWORD="${DB_PASSWORD:-helixflow123}"

# Test results
PASSED=0
FAILED=0

run_test() {
    local name=$1
    local command=$2
    local expected=$3
    
    echo -n "Testing: $name... "
    
    if eval "$command"; then
        if [ "$expected" = "pass" ]; then
            echo -e "${GREEN}✓ PASS${NC}"
            PASSED=$((PASSED + 1))
        else
            echo -e "${RED}✗ FAIL (expected to fail)${NC}"
            FAILED=$((FAILED + 1))
        fi
    else
        if [ "$expected" = "fail" ]; then
            echo -e "${GREEN}✓ PASS (expected to fail)${NC}"
            PASSED=$((PASSED + 1))
        else
            echo -e "${RED}✗ FAIL${NC}"
            FAILED=$((FAILED + 1))
        fi
    fi
}

echo "1. Database Connectivity Tests"
echo "------------------------------"

run_test "PostgreSQL connection" "pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME" "pass"
run_test "Users table exists" "psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c 'SELECT COUNT(*) FROM users;' | grep -q '[0-9]'" "pass"
run_test "API keys table exists" "psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c 'SELECT COUNT(*) FROM api_keys;' | grep -q '[0-9]'" "pass"

echo ""
echo "2. TLS Certificate Tests"
echo "------------------------"

CERT_DIR="${CERT_DIR:-./certs}"
SERVICES=("api-gateway" "auth-service" "inference-pool" "monitoring")

for service in "${SERVICES[@]}"; do
    run_test "$service certificate exists" "[ -f $CERT_DIR/${service}.crt ]" "pass"
    run_test "$service private key exists" "[ -f $CERT_DIR/${service}-key.pem ]" "pass"
    run_test "$service certificate valid" "openssl x509 -in $CERT_DIR/${service}.crt -noout -text | grep -q 'Validity'" "pass"
done

echo ""
echo "3. Service Health Tests"
echo "-----------------------"

# Start services
echo "Starting services..."

# Start with proper environment
export TLS_CERT=$CERT_DIR/api-gateway.crt
export TLS_KEY=$CERT_DIR/api-gateway-key.pem
export CA_CERT=$CERT_DIR/helixflow-ca.pem
export DB_HOST=$DB_HOST
export DB_PORT=$DB_PORT
export DB_NAME=$DB_NAME
export DB_USER=$DB_USER
export DB_PASSWORD=$DB_PASSWORD
export REDIS_HOST=localhost
export REDIS_PORT=6379

# Start services in background
./api-gateway/bin/api-gateway &
API_PID=$!
./auth-service/bin/auth-service &
AUTH_PID=$!
./inference-pool/bin/inference-pool &
INFERENCE_PID=$!
./monitoring/bin/monitoring &
MONITORING_PID=$!

# Wait for services to start
sleep 5

# Test health endpoints
run_test "API Gateway health (HTTPS)" "curl -s -f -k https://localhost:8443/health > /dev/null" "pass"
run_test "Auth Service health (gRPC)" "grpc_health_probe -addr localhost:8081 -tls -tls-ca-cert $CERT_DIR/helixflow-ca.pem -tls-client-cert $CERT_DIR/auth-service-client.crt -tls-client-key $CERT_DIR/auth-service-client-key.pem > /dev/null" "pass"

echo ""
echo "4. Authentication Tests"
echo "-----------------------"

# Test user registration
run_test "User registration" "grpcurl -plaintext -d '{\"username\": \"testuser\", \"email\": \"test@example.com\", \"password\": \"testpass123\", \"first_name\": \"Test\", \"last_name\": \"User\"}' localhost:8081 helixflow.auth.AuthService/Register | grep -q 'success'" "pass"

# Test user login
run_test "User login" "grpcurl -plaintext -d '{\"username\": \"demo@helixflow.com\", \"password\": \"demo123\"}' localhost:8081 helixflow.auth.AuthService/Login | grep -q 'access_token'" "pass"

echo ""
echo "5. AI Inference Tests"
echo "---------------------"

# Test model listing
run_test "Models endpoint with auth" "curl -s -f -k https://localhost:8443/v1/models -H 'Authorization: Bearer demo-key' | jq -e '.data | length > 0' > /dev/null" "pass"

# Test inference with proper auth
TOKEN=$(grpcurl -plaintext -d '{\"username\": \"demo@helixflow.com\", \"password\": \"demo123\"}' localhost:8081 helixflow.auth.AuthService/Login | jq -r '.access_token')

run_test "Chat completion with auth" "curl -s -f -k https://localhost:8443/v1/chat/completions \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{\"model\": \"gpt-3.5-turbo\", \"messages\": [{\"role\": \"user\", \"content\": \"Hello\"}]}' | jq -e '.choices[0].message.content != null' > /dev/null" "pass"

echo ""
echo "6. Performance Tests"
echo "--------------------"

# Test response time
start_time=$(date +%s%3N)
curl -s -k https://localhost:8443/v1/chat/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Test"}]}' > /dev/null
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
echo "7. Security Tests"
echo "-----------------"

# Test invalid token
run_test "Invalid token rejected" "curl -s -o /dev/null -w '%{http_code}' -k https://localhost:8443/v1/models -H 'Authorization: Bearer invalid-token' | grep -q '401'" "pass"

# Test rate limiting
run_test "Rate limiting active" "for i in {1..1100}; do curl -s -o /dev/null -w '%{http_code}' -k https://localhost:8443/v1/models -H 'Authorization: Bearer $TOKEN'; done | grep -q '429'" "pass"

echo ""
echo "8. Monitoring Tests"
echo "-------------------"

# Test metrics collection
run_test "Metrics endpoint accessible" "curl -s -f http://localhost:8083/metrics > /dev/null" "pass"

# Test system metrics
run_test "System metrics valid" "curl -s http://localhost:8083/metrics | jq -e '.cpu_usage != null' > /dev/null" "pass"

echo ""
echo "Cleaning up..."
echo "=============="

# Kill background processes
kill $API_PID $AUTH_PID $INFERENCE_PID $MONITORING_PID 2>/dev/null || true
wait $API_PID $AUTH_PID $INFERENCE_PID $MONITORING_PID 2>/dev/null || true

echo ""
echo "====================================="
echo "Production Integration Test Results"
echo "====================================="
echo -e "Total Tests: $((PASSED + FAILED))"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo -e "Success Rate: ${YELLOW}$(( PASSED * 100 / (PASSED + FAILED) ))%${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All production integration tests passed!${NC}"
    echo ""
    echo "✅ Production infrastructure is ready"
    echo "✅ Database integration is working"
    echo "✅ TLS certificates are properly configured"
    echo "✅ All services are communicating via gRPC"
    echo "✅ Authentication system is functional"
    echo "✅ AI inference is working with real responses"
    echo "✅ Monitoring and metrics are collecting data"
    echo ""
    echo "🚀 HelixFlow is ready for production deployment!"
    exit 0
else
    echo -e "${RED}❌ Some production integration tests failed.${NC}"
    echo "Please review the failed tests and fix the issues before deployment."
    exit 1
fi
EOF

chmod +x scripts/integration_test_production.sh

    print_success "Comprehensive test suite created"
}

# Main execution
main() {
    echo "🚀 Starting HelixFlow Production Infrastructure Setup"
    echo "===================================================="
    echo ""
    echo "Configuration:"
    echo "  Database: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"
    echo "  Certificates: $CERT_DIR"
    echo "  Redis: $REDIS_HOST:$REDIS_PORT"
    echo ""
    
    check_prerequisites
    setup_postgresql
    setup_tls_certificates
    update_service_configs
    create_systemd_services
    create_docker_compose
    create_test_suite
    
    echo ""
    echo "🎉 Production infrastructure setup completed!"
    echo ""
    echo "Next steps:"
    echo "1. Review and customize the generated configuration files"
    echo "2. Start services using: systemctl start helixflow-*"
    echo "3. Run integration tests: ./scripts/integration_test_production.sh"
    echo "4. Set up monitoring with Prometheus and Grafana"
    echo "5. Configure backup and disaster recovery"
    echo ""
    echo "For Docker deployment:"
    echo "  docker-compose up -d"
    echo ""
    echo "For systemd deployment:"
    echo "  systemctl enable helixflow-{api-gateway,auth-service,inference-pool,monitoring}"
    echo "  systemctl start helixflow-{api-gateway,auth-service,inference-pool,monitoring}"
}

# Run main function
main "$@"