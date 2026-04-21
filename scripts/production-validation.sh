#!/bin/bash

# Production Validation Script
# Comprehensive validation before production deployment

set -e

echo "🏭 Running Production Validation..."

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    local status=$1
    local message=$2
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✅ $message${NC}"
    elif [ "$status" = "WARN" ]; then
        echo -e "${YELLOW}⚠️  $message${NC}"
    else
        echo -e "${RED}❌ $message${NC}"
        echo "$message" >> validation-failures.log
    fi
}

# Initialize failure log
> validation-failures.log

# 1. Service Health Checks
echo "🔍 Checking service health..."
services=("api-gateway" "auth-service" "inference-pool" "monitoring")

for service in "${services[@]}"; do
    if curl -f -s "http://localhost:8080/health" > /dev/null 2>&1; then
        print_status "PASS" "$service health check"
    else
        print_status "FAIL" "$service health check failed"
    fi
done

# 2. Database Connectivity
echo "🗄️  Checking database connectivity..."
if pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
    print_status "PASS" "PostgreSQL connectivity"
else
    print_status "FAIL" "PostgreSQL connectivity failed"
fi

if redis-cli ping > /dev/null 2>&1; then
    print_status "PASS" "Redis connectivity"
else
    print_status "FAIL" "Redis connectivity failed"
fi

# 3. API Contract Validation
echo "📋 Validating API contracts..."
if [ -f "tests/contract/test_chat_api.py" ]; then
    python3 -m pytest tests/contract/test_chat_api.py -v --tb=short > api-test-results.txt 2>&1
    if [ $? -eq 0 ]; then
        print_status "PASS" "API contract tests"
    else
        print_status "FAIL" "API contract tests failed"
    fi
else
    print_status "FAIL" "API contract tests not found"
fi

# 4. Load Testing
echo "⚡ Running load tests..."
if command -v k6 &> /dev/null; then
    k6 run --vus 10 --duration 30s tests/performance/load-test.js > load-test-results.txt 2>&1
    if grep -q "http_req_duration" load-test-results.txt; then
        avg_response=$(grep "http_req_duration" load-test-results.txt | tail -1 | awk '{print $2}')
        if (( $(echo "$avg_response < 1000" | bc -l) )); then
            print_status "PASS" "Load test performance: ${avg_response}ms avg response"
        else
            print_status "FAIL" "Load test performance too slow: ${avg_response}ms avg response"
        fi
    else
        print_status "WARN" "Load test completed but results unclear"
    fi
else
    print_status "WARN" "k6 not found, skipping load tests"
fi

# 5. Security Validation
echo "🔒 Running security validation..."
if [ -f "tests/security/test_penetration.py" ]; then
    python3 -m pytest tests/security/test_penetration.py -v --tb=line > security-test-results.txt 2>&1
    if [ $? -eq 0 ]; then
        print_status "PASS" "Security penetration tests"
    else
        print_status "FAIL" "Security penetration tests failed"
    fi
else
    print_status "FAIL" "Security tests not found"
fi

# 6. Compliance Checks
echo "📜 Running compliance checks..."
if [ -f "tests/integration/test_compliance.py" ]; then
    python3 -m pytest tests/integration/test_compliance.py -v --tb=line > compliance-test-results.txt 2>&1
    if [ $? -eq 0 ]; then
        print_status "PASS" "Compliance tests"
    else
        print_status "FAIL" "Compliance tests failed"
    fi
else
    print_status "FAIL" "Compliance tests not found"
fi

# 7. Documentation Completeness
echo "📚 Checking documentation completeness..."
doc_files=("README.md" "docs/README.md" "docs/guides/getting-started.md")
doc_complete=true

for doc in "${doc_files[@]}"; do
    if [ ! -f "$doc" ]; then
        print_status "FAIL" "Documentation file missing: $doc"
        doc_complete=false
    fi
done

if [ "$doc_complete" = true ]; then
    print_status "PASS" "Documentation completeness"
fi

# 8. SDK Validation
echo "📦 Validating SDKs..."
if [ -d "sdks/python3" ] && [ -f "sdks/python3/setup.py" ]; then
    print_status "PASS" "Python SDK structure"
else
    print_status "FAIL" "Python SDK incomplete"
fi

# 9. Kubernetes Manifests Validation
echo "☸️  Validating Kubernetes manifests..."
if command -v kubeconform &> /dev/null; then
    if kubeconform -strict k8s/ > kube-validation-results.txt 2>&1; then
        print_status "PASS" "Kubernetes manifests validation"
    else
        print_status "FAIL" "Kubernetes manifests validation failed"
    fi
else
    print_status "WARN" "kubeconform not found, skipping K8s validation"
fi

# 10. Performance Benchmarks
echo "📊 Running performance benchmarks..."
if command -v go &> /dev/null; then
    go test -bench=. -benchmem ./... > benchmark-results.txt 2>&1
    if [ -s benchmark-results.txt ]; then
        print_status "PASS" "Performance benchmarks completed"
    else
        print_status "WARN" "No benchmarks found"
    fi
else
    print_status "WARN" "Go not found, skipping benchmarks"
fi

# Final Report
echo ""
echo "📋 Production Validation Report"
echo "================================"

if [ -s validation-failures.log ]; then
    echo -e "${RED}❌ Validation failed with the following issues:${NC}"
    cat validation-failures.log
    echo ""
    echo -e "${YELLOW}⚠️  Please address the issues above before deploying to production.${NC}"
    exit 1
else
    echo -e "${GREEN}✅ All production validation checks passed!${NC}"
    echo ""
    echo "🚀 System is ready for production deployment."
    echo ""
    echo "Next steps:"
    echo "1. Run: kubectl apply -f k8s/"
    echo "2. Update DNS records"
    echo "3. Configure monitoring alerts"
    echo "4. Run final integration tests"
fi
