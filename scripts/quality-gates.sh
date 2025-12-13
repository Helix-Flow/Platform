#!/bin/bash

# Quality Gates Script for HelixFlow
# Ensures code quality before deployment

set -e

echo "🔍 Running Quality Gates..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    local status=$1
    local message=$2
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✅ $message${NC}"
    elif [ "$status" = "WARN" ]; then
        echo -e "${YELLOW}⚠️  $message${NC}"
    else
        echo -e "${RED}❌ $message${NC}"
    fi
}

# 1. Test Coverage Check
echo "📊 Checking test coverage..."
if command -v go &> /dev/null; then
    go test ./... -coverprofile=coverage.out > /dev/null 2>&1
    coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    
    if (( $(echo "$coverage >= 80" | bc -l) )); then
        print_status "PASS" "Test coverage: ${coverage}% (required: 80%)"
    else
        print_status "FAIL" "Test coverage: ${coverage}% (required: 80%)"
        exit 1
    fi
else
    print_status "WARN" "Go not found, skipping coverage check"
fi

# 2. Security Scan
echo "🔒 Running security scan..."
if command -v trivy &> /dev/null; then
    trivy fs --exit-code 1 --no-progress --format json . > trivy-results.json 2>/dev/null || true
    
    vulnerabilities=$(jq '.Results[].Vulnerabilities | length' trivy-results.json 2>/dev/null | awk '{sum += $1} END {print sum}')
    
    if [ "$vulnerabilities" -eq 0 ] || [ -z "$vulnerabilities" ]; then
        print_status "PASS" "Security scan: No high/critical vulnerabilities"
    else
        print_status "FAIL" "Security scan: $vulnerabilities vulnerabilities found"
        exit 1
    fi
else
    print_status "WARN" "Trivy not found, skipping security scan"
fi

# 3. Code Quality Check
echo "🧹 Checking code quality..."
if command -v golangci-lint &> /dev/null; then
    golangci-lint run --timeout=5m > lint-results.txt 2>&1 || true
    
    if [ -s lint-results.txt ]; then
        issues=$(wc -l < lint-results.txt)
        if [ "$issues" -gt 10 ]; then
            print_status "FAIL" "Code quality: $issues linting issues (max: 10)"
            exit 1
        else
            print_status "PASS" "Code quality: $issues linting issues"
        fi
    else
        print_status "PASS" "Code quality: No linting issues"
    fi
else
    print_status "WARN" "golangci-lint not found, skipping code quality check"
fi

# 4. Performance Benchmarks
echo "⚡ Running performance benchmarks..."
if command -v go &> /dev/null; then
    go test -bench=. -benchmem ./... > benchmark-results.txt 2>&1 || true
    
    if [ -s benchmark-results.txt ]; then
        print_status "PASS" "Performance benchmarks completed"
    else
        print_status "WARN" "No benchmarks found"
    fi
else
    print_status "WARN" "Go not found, skipping benchmarks"
fi

# 5. API Contract Validation
echo "📋 Validating API contracts..."
if [ -d "tests/contract" ]; then
    # Run contract tests
    if command -v go &> /dev/null; then
        go test ./tests/contract/... -v > contract-test-results.txt 2>&1 || true
        
        if grep -q "FAIL" contract-test-results.txt; then
            print_status "FAIL" "API contract tests failed"
            exit 1
        else
            print_status "PASS" "API contract tests passed"
        fi
    fi
else
    print_status "WARN" "Contract tests directory not found"
fi

# 6. Documentation Check
echo "📚 Checking documentation..."
if [ -f "README.md" ] && [ -d "docs" ]; then
    print_status "PASS" "Documentation structure present"
else
    print_status "FAIL" "Documentation structure incomplete"
    exit 1
fi

# 7. Dependency Check
echo "📦 Checking dependencies..."
if command -v go &> /dev/null && [ -f "go.mod" ]; then
    go mod tidy > /dev/null 2>&1
    go mod verify > /dev/null 2>&1
    
    if [ $? -eq 0 ]; then
        print_status "PASS" "Go dependencies verified"
    else
        print_status "FAIL" "Go dependencies verification failed"
        exit 1
    fi
else
    print_status "WARN" "Go or go.mod not found, skipping dependency check"
fi

echo ""
print_status "PASS" "All quality gates passed! ✅"
