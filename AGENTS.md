# AGENTS.md

HelixFlow AI inference platform - enterprise-grade microservices providing OpenAI-compatible API access to 300+ AI models.

## Project Overview

HelixFlow is a comprehensive AI inference platform with:
- **Microservices Architecture**: API Gateway, Auth Service, Inference Pool, Monitoring Service
- **Language Mix**: Go (services) + Python (SDK, testing)
- **Database Support**: SQLite (default), PostgreSQL (production)
- **Protocols**: gRPC (internal), HTTP/HTTPS (external), WebSocket
- **Security**: JWT authentication, TLS 1.3, mTLS between services

## Directory Structure

```
Platform/
├── api-gateway/          # Main API gateway service (Go)
├── auth-service/         # Authentication service (Go)
├── inference-pool/       # AI model inference service (Go)
├── monitoring/           # Monitoring service (Go)
├── internal/             # Shared internal libraries
│   └── database/        # Database abstraction layer
├── helixflow/           # gRPC protocol definitions
│   ├── auth/           # Auth service protobufs
│   ├── inference/      # Inference service protobufs
│   └── monitoring/     # Monitoring service protobufs
├── tests/              # Comprehensive test suite
│   ├── integration/    # Integration tests
│   ├── contract/       # API contract tests
│   ├── security/       # Security tests
│   ├── performance/    # Load tests
│   └── unit/          # Unit tests
├── scripts/            # Deployment and management scripts
├── certs/             # TLS certificates
├── data/              # Database files and data
├── logs/              # Service logs
├── sdks/              # Client SDKs (Python)
└── proto/             # Protocol buffer definitions
```

## Build/Test Commands

### Go Services (Go 1.22.2)
```bash
# Build individual services
cd api-gateway/src && go build -o ../bin/api-gateway .
cd auth-service/src && go build -o ../bin/auth-service .
cd inference-pool/src && go build -o ../bin/inference-pool .
cd monitoring/src && go build -o ../bin/monitoring .

# Build internal libraries
cd internal/database && go build .
cd helixflow/auth && go build .
cd helixflow/inference && go build .
cd helixflow/monitoring && go build .

# Clean builds
find . -name "bin" -type d -exec rm -rf {} + 2>/dev/null
find . -name "*.out" -delete
```

### Python Environment
```bash
# Install test dependencies
pip install -r requirements-master.txt

# Build Python SDK
cd sdks/python && python setup.py build

# Create virtual environment
python -m venv venv
source venv/bin/activate
pip install -r requirements-master.txt
```

### Testing Framework
```bash
# Run comprehensive integration test suite
./test_integration.sh

# Run specific test categories
python -m pytest tests/integration/     # Integration tests
python -m pytest tests/contract/        # API contract tests  
python -m pytest tests/security/        # Security tests
python -m pytest tests/performance/     # Load tests
python -m pytest tests/unit/            # Unit tests

# Run individual test files
python -m pytest tests/integration/test_auth.py -v
python -m pytest tests/security/test_security_pentest.py -v

# Run specific test functions
python -m pytest tests/integration/test_auth.py::test_login -v
python -m pytest tests/integration/test_auth.py::test_jwt_token_generation_and_validation -v

# Test token revocation functionality
python3 test_revocation_now.py

# Suppress TLS verification warnings in tests
export PYTHONWARNINGS="ignore:Unverified HTTPS request"

# Quick test script
./scripts/quick_test.sh
```

**Note**: The `test_rate_limiting_integration` test may fail due to Redis dependency. All other auth tests pass with token revocation fully functional.

### Local Development
```bash
# Start all services with proper configuration
./start_all_services.sh

# Start individual service groups
./start_phase1_services.sh  # Core services
./start_phase2_services.sh  # Advanced services

# Stop all services
kill $(cat logs/service_pids.txt) 2>/dev/null

# Test API Gateway health (with TLS verification disabled for self-signed certs)
curl -k https://localhost:8443/health

# Test with updated URLs (local Kubernetes service URLs replaced with localhost)
# All integration tests now target localhost instead of *.svc.cluster.local
```

### Manual Service Startup (Debugging)
```bash
# Start auth service (gRPC on 8081, HTTP on 8082)
cd auth-service && HTTP_PORT=8082 PORT=8081 DATABASE_TYPE=sqlite DATABASE_PATH=../data/helixflow.db ./bin/auth-service

# Start inference pool (gRPC on 50051)
cd inference-pool && PORT=50051 ./bin/inference-pool

# Start API gateway (TLS on 8443)
cd api-gateway && TLS_CERT="../certs/api-gateway.crt" TLS_KEY="../certs/api-gateway-key.pem" INFERENCE_POOL_URL=localhost:50051 AUTH_SERVICE_GRPC=localhost:8081 PORT=8443 ./bin/api-gateway

# Start monitoring service (gRPC on 8083)
cd monitoring && PORT=8083 ./bin/monitoring
```

### Docker Development
```bash
# Start database services with Docker Compose
docker-compose up -d postgres redis

# Generate TLS certificates
./scripts/generate_certificates.sh

# Setup databases
./scripts/setup_sqlite_database.sh
./scripts/setup_postgresql.sh
```

## Code Style Guidelines

### Go (1.22.2)
- **Project Layout**: Standard structure with `src/` directory for each service
- **Logging**: Structured logging with contextual information
- **Architecture**: Clean architecture pattern (handlers, services, repositories layers)
- **Error Handling**: Error wrapping with `fmt.Errorf("operation failed: %w", err)`
- **Configuration**: Environment variables via `getEnv(key, default)` helper pattern
- **Security**: TLS 1.3 only, mTLS between services
- **gRPC**: Use protocol buffers for service communication
- **Dependencies**: Well-maintained go.mod files with local module replacements

### Python (3.11+)
- **Style**: PEP 8 compliance with mandatory type hints
- **Web Framework**: Flask 2.3.3 with blueprint organization
- **Authentication**: PyJWT 2.8.0 RS256 for JWT tokens
- **HTTP Clients**: Requests library with session reuse
- **Error Handling**: Custom exceptions in dedicated `exceptions.py` files
- **Testing**: Comprehensive pytest framework with fixtures

### General Conventions
- **Branch Naming**: `001-feature-name` format for feature branches
- **CI/CD**: No automated CI/CD - manual quality gates only
- **Secrets**: Never commit secrets/API keys to repository
- **Paths**: Use absolute paths from repository root
- **Error Output**: Error messages to stderr (`>&2`)
- **Documentation**: Comprehensive technical specification and implementation reports

## Development Workflow

### Service Dependencies
```
API Gateway → Auth Service (gRPC)
API Gateway → Inference Pool (gRPC)  
Auth Service → Database (SQLite/PostgreSQL)
All Services → Monitoring (gRPC)
```

### Protocol Buffer Generation
```bash
# Generate Go code from .proto files
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/*.proto
```

### Database Management
```bash
# SQLite (default development)
export DATABASE_TYPE=sqlite
export DATABASE_PATH=../data/helixflow.db

# PostgreSQL (production)
export DATABASE_TYPE=postgres
export DATABASE_URL="postgres://user:pass@localhost:5432/helixflow"

# Initialize databases
./scripts/setup_sqlite_database.sh
./scripts/setup_postgresql.sh
```

## Troubleshooting & Debugging

### Common Issues

**Token Revocation Problems**
- **Symptom**: Revoked tokens still accepted by API gateway
- **Solution**: 
  1. Ensure API gateway is using gRPC authentication (enhanced version)
  2. Check auth service is running on port 8081 (gRPC) and 8082 (HTTP)
  3. Verify environment variables: `AUTH_SERVICE_GRPC=localhost:8081`
  4. Confirm API gateway logs show "Auth service gRPC connection established"

**gRPC Connectivity Issues**
- **Localhost**: Use insecure transport (`grpc.WithTransportCredentials(insecure.NewCredentials())`)
- **Production**: Use TLS with proper certificates
- **Fallback**: Services handle missing gRPC connections gracefully
- **Ports**: Auth service gRPC on 8081, HTTP on 8082

**Certificate Problems**
- **Self-signed certs**: Use `verify=False` in Python tests
- **Certificate paths**: Relative paths from service directories (e.g., `../certs/`)
- **Local testing**: Insecure transport acceptable for localhost connections
- **Production**: Always use TLS 1.3 with valid certificates

### Service Port Configuration
- **Auth Service**: gRPC 8081, HTTP 8082
- **Inference Pool**: gRPC 50051  
- **API Gateway**: HTTPS 8443 (TLS), HTTP fallback if no certs
- **Monitoring**: gRPC 8083
- **PostgreSQL**: 5432
- **Redis**: 6379

### Key Environment Variables
```bash
# Database
DATABASE_TYPE=sqlite|postgres
DATABASE_PATH=../data/helixflow.db
DATABASE_URL=postgres://user:pass@host:5432/db

# TLS Configuration  
TLS_CERT="../certs/api-gateway.crt"
TLS_KEY="../certs/api-gateway-key.pem"

# Service URLs
INFERENCE_POOL_URL="localhost:50051"
AUTH_SERVICE_GRPC="localhost:8081"
AUTH_SERVICE_URL="localhost:8081"

# Port Configuration
PORT="8443"           # API Gateway HTTPS port
HTTP_PORT="8082"      # Auth service HTTP port

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
```

### Debugging Tools
```bash
# Check service status
curl -k https://localhost:8443/health
curl http://localhost:8082/health

# View service logs
tail -f logs/*.log

# Database inspection (SQLite)
sqlite3 data/helixflow.db .tables
sqlite3 data/helixflow.db "SELECT * FROM users LIMIT 5;"

# Generate debug information
./scripts/test_current_state.sh
```

## Deployment & Production

### Production Deployment
```bash
# Full production deployment
./scripts/production-deployment.sh

# Generate production certificates  
./scripts/generate_certificates.sh

# Setup production infrastructure
./scripts/setup_production_infrastructure.sh

# Run production validation
./scripts/production-validation.sh
./scripts/final-validation.sh
```

### Quality Gates
```bash
# Run quality gate checks
./scripts/quality-gates.sh

# Fix critical infrastructure issues
./scripts/fix-critical-infrastructure.sh

# Manual fixes and adjustments
./scripts/manual-fixes.sh
```

### Kubernetes & Helm
- **Helm Charts**: Located in `helm/` directory
- **Kubernetes Manifests**: Located in `k8s/` directory
- **Terraform**: Infrastructure as code in `terraform/` directory

## Testing Strategy

### Test Categories
1. **Integration Tests**: End-to-end service integration
2. **Contract Tests**: API compatibility and contract validation  
3. **Security Tests**: Penetration testing and security validation
4. **Performance Tests**: Load testing and performance benchmarking
5. **Unit Tests**: Individual component testing

### Test Execution Patterns
```python
# Example test fixture with service URLs
@pytest.fixture
def auth_service_url():
    return "http://localhost:8082"

@pytest.fixture  
def api_gateway_url():
    return "https://localhost:8443"

# TLS verification disabled for self-signed certs
response = requests.get(url, verify=False)
```

## Project Documentation

### Key Documentation Files
- `helixflow-technical-specification.md`: Comprehensive technical specification
- `COMPLETE_IMPLEMENTATION.md`: Implementation status and details
- `DEPLOYMENT_PACKAGE.md`: Deployment instructions and packages
- Various audit and status reports in root directory
- API documentation in `docs/` directory

### Codebase Organization
- **Services**: Independent Go modules in `api-gateway/`, `auth-service/`, etc.
- **Shared Code**: Internal libraries in `internal/` and `helixflow/`
- **Protocols**: gRPC protocol definitions with protobuf
- **Testing**: Comprehensive test suite organized by test type
- **Scripts**: Deployment, management, and utility scripts
- **Configuration**: TLS certs, database files, environment setup

This AGENTS.md provides comprehensive guidance for working with the HelixFlow codebase. Always refer to the specific service documentation and technical specifications for detailed implementation details.