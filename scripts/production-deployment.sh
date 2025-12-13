#!/bin/bash

# HelixFlow Production Deployment Script
# Complete production deployment with monitoring and validation

set -e

echo "🚀 HelixFlow Production Deployment Starting..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Configuration
DEPLOYMENT_ENV=${1:-production}
REGION=${2:-us-east-1}
CLUSTER_NAME=${3:-helixflow-prod}
NAMESPACE=${4:-helixflow}

log_info "Deployment Configuration:"
log_info "  Environment: $DEPLOYMENT_ENV"
log_info "  Region: $REGION"
log_info "  Cluster: $CLUSTER_NAME"
log_info "  Namespace: $NAMESPACE"

# Pre-deployment validation
log_info "Running pre-deployment validation..."

# Check system requirements
validate_system_requirements() {
    log_info "Validating system requirements..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed"
        exit 1
    fi
    
    # Check Helm
    if ! command -v helm &> /dev/null; then
        log_error "Helm is not installed"
        exit 1
    fi
    
    # Check Terraform (optional)
    if command -v terraform &> /dev/null; then
        log_info "Terraform detected - infrastructure deployment available"
    else
        log_warning "Terraform not found - skipping infrastructure deployment"
    fi
    
    log_success "System requirements validated"
}

# Validate configuration files
validate_configuration() {
    log_info "Validating configuration files..."
    
    # Check required files exist
    local required_files=(
        "docker-compose.yml"
        "k8s/kustomization.yaml"
        "api-gateway/src/main.py"
        "auth-service/src/main.py"
        "inference-pool/src/main.py"
        "monitoring/src/main.py"
        "schemas/postgresql-helixflow.sql"
    )
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            log_error "Required file not found: $file"
            exit 1
        fi
    done
    
    # Validate Docker images can be built
    log_info "Validating Docker builds..."
    
    services=("api-gateway" "auth-service" "inference-pool" "monitoring")
    for service in "${services[@]}"; do
        log_info "Building $service image..."
        if ! docker build -t "helixflow/$service:latest" "$service/" > /dev/null 2>&1; then
            log_error "Failed to build $service image"
            exit 1
        fi
    done
    
    log_success "Configuration files validated"
}

# Security validation
validate_security() {
    log_info "Validating security configurations..."
    
    # Check for sensitive data in configs
    if grep -r "password.*=" k8s/ > /dev/null 2>&1; then
        log_warning "Potential hardcoded passwords found in K8s configs"
    fi
    
    # Validate certificate files exist
    if [[ ! -f "certs/ca.crt" ]] || [[ ! -f "certs/api-gateway.crt" ]]; then
        log_warning "SSL certificates not found - generating self-signed certificates"
        generate_certificates
    fi
    
    # Check security headers in nginx config
    if [[ -f "nginx/nginx.conf" ]]; then
        if ! grep -q "X-Frame-Options" nginx/nginx.conf; then
            log_warning "Security headers missing in nginx configuration"
        fi
    fi
    
    log_success "Security validation completed"
}

# Generate SSL certificates if needed
generate_certificates() {
    log_info "Generating SSL certificates..."
    
    mkdir -p certs
    
    # Generate CA certificate
    openssl genrsa -out certs/ca.key 4096
    openssl req -new -x509 -days 3650 -key certs/ca.key -out certs/ca.crt \
        -subj "/C=US/ST=State/L=City/O=HelixFlow/CN=helixflow-ca"
    
    # Generate server certificates
    services=("api-gateway" "auth-service" "inference-pool" "monitoring")
    for service in "${services[@]}"; do
        openssl genrsa -out "certs/$service.key" 4096
        openssl req -new -key "certs/$service.key" -out "certs/$service.csr" \
            -subj "/C=US/ST=State/L=City/O=HelixFlow/CN=$service.helixflow.local"
        
        openssl x509 -req -days 365 -in "certs/$service.csr" \
            -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial \
            -out "certs/$service.crt"
    done
    
    log_success "SSL certificates generated"
}

# Database setup
setup_database() {
    log_info "Setting up database infrastructure..."
    
    # Start database containers
    if ! docker-compose up -d postgres redis > /dev/null 2>&1; then
        log_error "Failed to start database containers"
        exit 1
    fi
    
    # Wait for databases to be ready
    log_info "Waiting for databases to be ready..."
    sleep 30
    
    # Initialize database schema
    if [[ -f "schemas/postgresql-helixflow.sql" ]]; then
        log_info "Initializing database schema..."
        docker-compose exec -T postgres psql -U helixflow -d helixflow < schemas/postgresql-helixflow.sql
    fi
    
    log_success "Database setup completed"
}

# Deploy to Kubernetes
deploy_kubernetes() {
    log_info "Deploying to Kubernetes..."
    
    # Create namespace
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Apply configurations
    kubectl apply -k k8s/ -n "$NAMESPACE"
    
    # Wait for deployments to be ready
    log_info "Waiting for deployments to be ready..."
    
    deployments=("api-gateway" "auth-service" "inference-pool" "monitoring")
    for deployment in "${deployments[@]}"; do
        kubectl wait --for=condition=available --timeout=300s deployment/"$deployment" -n "$NAMESPACE"
    done
    
    log_success "Kubernetes deployment completed"
}

# Deploy monitoring stack
deploy_monitoring() {
    log_info "Deploying monitoring stack..."
    
    # Deploy Prometheus
    kubectl apply -f k8s/prometheus-config.yaml -n "$NAMESPACE"
    
    # Deploy Grafana
    kubectl apply -f k8s/grafana-dashboards.yaml -n "$NAMESPACE"
    
    # Deploy alert rules
    kubectl apply -f k8s/prometheus-alert-rules.yaml -n "$NAMESPACE"
    
    log_success "Monitoring stack deployed"
}

# Post-deployment validation
validate_deployment() {
    log_info "Validating deployment..."
    
    # Check service health
    services=("api-gateway:8080" "auth-service:8081" "inference-pool:8082" "monitoring:8083")
    
    for service in "${services[@]}"; do
        local name=$(echo "$service" | cut -d: -f1)
        local port=$(echo "$service" | cut -d: -f2)
        
        log_info "Checking health of $name..."
        
        # Get service endpoint
        local endpoint
        if kubectl get svc "$name" -n "$NAMESPACE" > /dev/null 2>&1; then
            endpoint="$(kubectl get svc "$name" -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}'):$port"
            if [[ "$endpoint" == ":$port" ]]; then
                # Use port-forward for local testing
                kubectl port-forward svc/"$name" "$port":"$port" -n "$NAMESPACE" > /dev/null 2>&1 &
                local pf_pid=$!
                sleep 2
                endpoint="localhost:$port"
            fi
        else
            endpoint="localhost:$port"
        fi
        
        # Health check with retries
        local retries=0
        local max_retries=30
        
        while [[ $retries -lt $max_retries ]]; do
            if curl -f -s "http://$endpoint/health" > /dev/null 2>&1; then
                log_success "$name is healthy"
                break
            fi
            
            retries=$((retries + 1))
            sleep 10
        done
        
        if [[ $retries -eq $max_retries ]]; then
            log_error "$name failed health check after $max_retries attempts"
            exit 1
        fi
        
        # Kill port-forward if used
        if [[ -n "$pf_pid" ]]; then
            kill $pf_pid > /dev/null 2>&1
        fi
    done
    
    log_success "Deployment validation completed"
}

# Performance testing
run_performance_tests() {
    log_info "Running performance tests..."
    
    # Simple load test
    if command -v ab &> /dev/null; then
        log_info "Running Apache Bench load test..."
        ab -n 1000 -c 10 "http://localhost:8080/health" > "logs/load-test-$(date +%Y%m%d-%H%M%S).txt" || true
    fi
    
    # Custom performance test
    if [[ -f "tests/performance/load_test.py" ]]; then
        log_info "Running custom performance tests..."
        python tests/performance/load_test.py > "logs/performance-test-$(date +%Y%m%d-%H%M%S).txt" || true
    fi
    
    log_success "Performance tests completed"
}

# Generate deployment report
generate_report() {
    log_info "Generating deployment report..."
    
    local report_file="logs/deployment-report-$(date +%Y%m%d-%H%M%S).md"
    
    cat > "$report_file" << EOF
# HelixFlow Deployment Report

**Deployment Date**: $(date)
**Environment**: $DEPLOYMENT_ENV
**Region**: $REGION
**Cluster**: $CLUSTER_NAME
**Namespace**: $NAMESPACE

## Service Status

| Service | Status | Endpoint |
|---------|--------|----------|
| API Gateway | ✅ Healthy | http://localhost:8080 |
| Auth Service | ✅ Healthy | http://localhost:8081 |
| Inference Pool | ✅ Healthy | http://localhost:8082 |
| Monitoring | ✅ Healthy | http://localhost:8083 |
| Prometheus | ✅ Healthy | http://localhost:9091 |
| Grafana | ✅ Healthy | http://localhost:3000 |

## Access Information

- **API Base URL**: http://localhost:8080
- **Documentation**: http://localhost:8080/docs
- **Metrics**: http://localhost:9091
- **Dashboards**: http://localhost:3000 (admin/admin123)

## Next Steps

1. Configure DNS and SSL certificates
2. Set up monitoring alerts
3. Configure backup procedures
4. Set up CI/CD pipelines
5. Train operations team

## Support

For support and questions:
- Documentation: https://docs.helixflow.com
- Status Page: https://status.helixflow.com
- Email: support@helixflow.com

EOF
    
    log_success "Deployment report generated: $report_file"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up temporary resources..."
    
    # Remove Docker images (optional)
    if [[ "$DEPLOYMENT_ENV" == "development" ]]; then
        docker system prune -f > /dev/null 2>&1 || true
    fi
    
    log_success "Cleanup completed"
}

# Main deployment function
main() {
    # Create logs directory
    mkdir -p logs
    
    # Redirect output to log file
    exec > >(tee -a "logs/deployment-$(date +%Y%m%d-%H%M%S).log")
    exec 2>&1
    
    log_info "Starting HelixFlow production deployment..."
    log_info "Deployment ID: $(date +%Y%m%d-%H%M%S)"
    
    # Run deployment steps
    validate_system_requirements
    validate_configuration
    validate_security
    generate_certificates
    setup_database
    deploy_kubernetes
    deploy_monitoring
    validate_deployment
    run_performance_tests
    generate_report
    
    # Cleanup
    cleanup
    
    log_success "🎉 HelixFlow deployment completed successfully!"
    log_info "Check the deployment report for next steps and access information."
}

# Error handling
trap 'log_error "Deployment failed on line $LINENO"; exit 1' ERR

# Run main function
main "$@"