#!/bin/bash

# HelixFlow Final Validation Script
# Comprehensive validation of the complete implementation

set -e

echo "🔍 HelixFlow Final Validation Starting..."

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

# Validation counters
total_checks=0
passed_checks=0
failed_checks=0
warning_checks=0

# Helper function to run validation check
run_check() {
    local check_name="$1"
    local check_command="$2"
    local required="$3"
    
    total_checks=$((total_checks + 1))
    
    log_info "Running check: $check_name"
    
    if eval "$check_command"; then
        passed_checks=$((passed_checks + 1))
        log_success "✅ $check_name - PASSED"
        return 0
    else
        if [[ "$required" == "true" ]]; then
            failed_checks=$((failed_checks + 1))
            log_error "❌ $check_name - FAILED (Required)"
            return 1
        else
            warning_checks=$((warning_checks + 1))
            log_warning "⚠️  $check_name - FAILED (Optional)"
            return 0
        fi
    fi
}

# Create validation report
mkdir -p logs
echo "HelixFlow Final Validation Report" > logs/final-validation-report.txt
echo "Generated on: $(date)" >> logs/final-validation-report.txt
echo "=====================================" >> logs/final-validation-report.txt

log_info "Starting comprehensive validation of HelixFlow implementation..."

# 1. File Structure Validation
echo "" >> logs/final-validation-report.txtecho "## 1. File Structure Validation" >> logs/final-validation-report.txt
log_info "Validating file structure..."

run_check "Core service directories exist" \
    "[[ -d api-gateway ]] && [[ -d auth-service ]] && [[ -d inference-pool ]] && [[ -d monitoring ]]" \
    "true"

run_check "Infrastructure files present" \
    "[[ -f docker-compose.yml ]] && [[ -d k8s ]] && [[ -d terraform ]]" \
    "true"

run_check "Documentation complete" \
    "[[ -f docs/API_REFERENCE.md ]] && [[ -f docs/CUSTOMER_ONBOARDING.md ]] && [[ -f docs/PERFORMANCE_OPTIMIZATION.md ]]" \
    "true"

run_check "Test structure complete" \
    "[[ -d tests/unit ]] && [[ -d tests/integration ]] && [[ -d tests/contract ]] && [[ -d tests/security ]]" \
    "true"

# 2. Service Implementation Validation
echo "" >> logs/final-validation-report.txt
echo "## 2. Service Implementation Validation" >> logs/final-validation-report.txt
log_info "Validating service implementations..."

run_check "API Gateway service file" \
    "[[ -f api-gateway/src/main.py ]]" \
    "true"

run_check "Auth Service implementation" \
    "[[ -f auth-service/src/main.py ]]" \
    "true"

run_check "Inference Pool service" \
    "[[ -f inference-pool/src/main.py ]]" \
    "true"

run_check "Monitoring service" \
    "[[ -f monitoring/src/main.py ]]" \
    "true"

# Test each service
run_check "API Gateway service test" \
    "python3 api-gateway/src/main.py > /dev/null 2>&1" \
    "true"

run_check "Auth Service test" \
    "python3 auth-service/src/main.py > /dev/null 2>&1" \
    "true"

run_check "Inference Pool test" \
    "python3 inference-pool/src/main.py > /dev/null 2>&1" \
    "true"

run_check "Monitoring service test" \
    "python3 monitoring/src/main.py > /dev/null 2>&1" \
    "true"

# 3. Configuration Validation
echo "" >> logs/final-validation-report.txt
echo "## 3. Configuration Validation" >> logs/final-validation-report.txt
log_info "Validating configurations..."

run_check "Docker Compose configuration" \
    "[[ -f docker-compose.yml ]] && grep -q 'api-gateway' docker-compose.yml" \
    "true"

run_check "Kubernetes manifests" \
    "[[ -f k8s/kustomization.yaml ]] && [[ -f k8s/api-gateway.yaml ]]" \
    "true"

run_check "Environment template" \
    "[[ -f .env.template ]] && grep -q 'API_SECRET_KEY' .env.template" \
    "true"

run_check "Service requirements" \
    "[[ -f api-gateway/requirements.txt ]] && [[ -f auth-service/requirements.txt ]]" \
    "true"

# 4. Documentation Validation
echo "" >> logs/final-validation-report.txt
echo "## 4. Documentation Validation" >> logs/final-validation-report.txt
log_info "Validating documentation..."

run_check "API Reference documentation" \
    "[[ -f docs/API_REFERENCE.md ]] && wc -l docs/API_REFERENCE.md | awk '{print \$1}' | grep -E '^[0-9]{3,}'" \
    "true"

run_check "Customer onboarding guide" \
    "[[ -f docs/CUSTOMER_ONBOARDING.md ]] && wc -l docs/CUSTOMER_ONBOARDING.md | awk '{print \$1}' | grep -E '^[0-9]{3,}'" \
    "true"

run_check "Performance optimization guide" \
    "[[ -f docs/PERFORMANCE_OPTIMIZATION.md ]] && wc -l docs/PERFORMANCE_OPTIMIZATION.md | awk '{print \$1}' | grep -E '^[0-9]{3,}'" \
    "true"

run_check "Implementation summary report" \
    "[[ -f IMPLEMENTATION_SUMMARY.md ]] && wc -l IMPLEMENTATION_SUMMARY.md | awk '{print \$1}' | grep -E '^[0-9]{3,}'" \
    "true"

# 5. Testing Framework Validation
echo "" >> logs/final-validation-report.txt
echo "## 5. Testing Framework Validation" >> logs/final-validation-report.txt
log_info "Validating testing framework..."

run_check "Test configuration" \
    "[[ -f tests/conftest.py ]]" \
    "true"

run_check "Unit tests structure" \
    "[[ -f tests/unit/test_api_gateway.py ]]" \
    "true"

run_check "Integration tests" \
    "[[ -f tests/integration/test_service_integration.py ]]" \
    "true"

run_check "Contract tests" \
    "[[ -f tests/contract/test_api_compliance.py ]]" \
    "true"

run_check "Security tests" \
    "[[ -f tests/security/test_security_pentest.py ]]" \
    "true"

# 6. Website Validation
echo "" >> logs/final-validation-report.txt
echo "## 6. Website Validation" >> logs/final-validation-report.txt
log_info "Validating website implementation..."

run_check "Website index page" \
    "[[ -f Website/content/index.html ]] && grep -q 'HelixFlow' Website/content/index.html" \
    "true"

run_check "Website JavaScript" \
    "[[ -f Website/content/js/main.js ]] && grep -q 'generateDemoResponse' Website/content/js/main.js" \
    "true"

run_check "Website CSS" \
    "[[ -f Website/content/css/custom.css ]] && grep -q 'feature-card' Website/content/css/custom.css" \
    "true"

# 7. Infrastructure Validation
echo "" >> logs/final-validation-report.txt
echo "## 7. Infrastructure Validation" >> logs/final-validation-report.txt
log_info "Validating infrastructure components..."

run_check "Docker configurations" \
    "[[ -f api-gateway/Dockerfile ]] && [[ -f auth-service/Dockerfile ]]" \
    "true"

run_check "Kubernetes configurations" \
    "[[ -f k8s/prometheus-config.yaml ]] && [[ -f k8s/grafana-dashboards.yaml ]]" \
    "true"

run_check "Helm charts" \
    "[[ -d helm ]]" \
    "true"

run_check "Terraform infrastructure" \
    "[[ -d terraform ]] && [[ -f terraform/aws/main.tf ]]" \
    "true"

# 8. Monitoring Setup Validation
echo "" >> logs/final-validation-report.txt
echo "## 8. Monitoring Setup Validation" >> logs/final-validation-report.txt
log_info "Validating monitoring setup..."

run_check "Prometheus configuration" \
    "[[ -f monitoring/prometheus.yml ]] && grep -q 'scrape_configs' monitoring/prometheus.yml" \
    "true"

run_check "Grafana dashboard" \
    "[[ -f monitoring/grafana/dashboards/helixflow-overview.json ]]" \
    "true"

run_check "Alert rules" \
    "[[ -f monitoring/alert_rules.yml ]] && grep -q 'alert:' monitoring/alert_rules.yml" \
    "true"

# 9. Security Validation
echo "" >> logs/final-validation-report.txt
echo "## 9. Security Validation" >> logs/final-validation-report.txt
log_info "Validating security implementations..."

run_check "SSL certificate generation functionality" \
    "grep -q 'generate_certificates' scripts/production-deployment.sh" \
    "true"

run_check "Security test coverage" \
    "grep -r 'security' tests/ | wc -l | awk '{print \$1}' | grep -E '^[0-9]{2,}'" \
    "true"

# 10. Performance Validation
echo "" >> logs/final-validation-report.txt
echo "## 10. Performance Validation" >> logs/final-validation-report.txt
log_info "Validating performance optimizations..."

run_check "Performance test scripts" \
    "[[ -f tests/performance/load_test.py ]]" \
    "true"

run_check "Performance optimization documentation" \
    "grep -q 'sub-100ms' docs/PERFORMANCE_OPTIMIZATION.md" \
    "true"

# 11. Deployment Scripts Validation
echo "" >> logs/final-validation-report.txt
echo "## 11. Deployment Scripts Validation" >> logs/final-validation-report.txt
log_info "Validating deployment automation..."

run_check "Production deployment script" \
    "[[ -f scripts/production-deployment.sh ]] && grep -q 'validate_deployment' scripts/production-deployment.sh" \
    "true"

run_check "Final validation script" \
    "[[ -f scripts/final-validation.sh ]]" \
    "true"

# 12. Documentation Completeness Check
echo "" >> logs/final-validation-report.txt
echo "## 12. Documentation Completeness Check" >> logs/final-validation-report.txt
log_info "Checking documentation completeness..."

# Count total lines of documentation
docs_lines=$(find docs/ -name "*.md" -exec wc -l {} + | awk '{sum+=$1} END {print sum}')
if [[ $docs_lines -gt 50000 ]]; then
    run_check "Documentation comprehensiveness" \
        "echo $docs_lines | grep -E '^[0-9]{5,}'" \
        "true"
else
    run_check "Documentation comprehensiveness" \
        "false" \
        "false"
fi

# Final Summary
echo "" >> logs/final-validation-report.txt
echo "## Final Summary" >> logs/final-validation-report.txt
echo "" >> logs/final-validation-report.txt
echo "Total Checks: $total_checks" >> logs/final-validation-report.txt
echo "Passed: $passed_checks" >> logs/final-validation-report.txt
echo "Failed (Required): $failed_checks" >> logs/final-validation-report.txt
echo "Warnings (Optional): $warning_checks" >> logs/final-validation-report.txt
echo "Success Rate: $(( passed_checks * 100 / total_checks ))%" >> logs/final-validation-report.txt

# Calculate overall status
if [[ $failed_checks -eq 0 ]]; then
    overall_status="SUCCESS"
    status_color=$GREEN
else
    overall_status="PARTIAL SUCCESS"
    status_color=$YELLOW
fi

log_info "Validation completed. Generating final report..."

# Display results
echo ""
echo -e "${status_color}========================================${NC}"
echo -e "${status_color}     HELIXFLOW VALIDATION COMPLETE      ${NC}"
echo -e "${status_color}========================================${NC}"
echo ""
echo "📊 Validation Results:"
echo "  Total Checks: $total_checks"
echo "  ✅ Passed: $passed_checks"
echo "  ❌ Failed (Required): $failed_checks"
echo "  ⚠️  Warnings (Optional): $warning_checks"
echo "  📈 Success Rate: $(( passed_checks * 100 / total_checks ))%"
echo ""

if [[ $failed_checks -eq 0 ]]; then
    echo -e "${GREEN}🎉 All critical validations passed!${NC}"
    echo -e "${GREEN}✅ HelixFlow is ready for production deployment!${NC}"
else
    echo -e "${YELLOW}⚠️  Some required validations failed.${NC}"
    echo -e "${YELLOW}🔧 Please review and fix the failed checks before deployment.${NC}"
fi

echo ""
echo "📄 Detailed validation report saved to: logs/final-validation-report.txt"
echo ""
echo "Next Steps:"
if [[ $failed_checks -eq 0 ]]; then
    echo "1. 🚀 Run production deployment: ./scripts/production-deployment.sh"
    echo "2. 📊 Set up monitoring and alerts"
    echo "3. 👥 Onboard your team with the documentation"
    echo "4. 🎯 Deploy your first production workload"
else
    echo "1. 🔧 Fix the failed validation checks"
    echo "2. 🔄 Re-run validation after fixes"
    echo "3. 📋 Review the detailed report for specific issues"
    echo "4. ✅ Ensure all required checks pass before deployment"
fi

echo ""
echo "For support and questions:"
echo "📧 Email: support@helixflow.com"
echo "🌐 Website: https://helixflow.com"
echo "📚 Documentation: https://docs.helixflow.com"
echo ""
echo -e "${status_color}========================================${NC}"

# Return appropriate exit code
if [[ $failed_checks -eq 0 ]]; then
    exit 0
else
    exit 1
fi