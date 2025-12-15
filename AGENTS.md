# AGENTS.md

HelixFlow AI inference platform - enterprise-grade microservices providing OpenAI-compatible API access to 300+ AI models.

## Build/Test Commands

### Go Services
```bash
# Build services (go.mod inside src directory)
cd api-gateway/src && go build -o ../bin/api-gateway .
cd auth-service/src && go build -o ../bin/auth-service .
cd inference-pool/src && go build -o ../bin/inference-pool .
cd monitoring/src && go build -o ../bin/monitoring .

# Python SDK
cd sdks/python && python setup.py build
```

### Testing
```bash
# Run integration test script (comprehensive)
./test_integration.sh

# Run single test
python -m pytest tests/integration/test_auth.py::test_login -v

# Test token revocation functionality
python3 test_revocation_now.py

# Test categories
python -m pytest tests/integration/    # Integration tests (note: rate limiting test may fail)
python -m pytest tests/contract/       # API contract tests
python -m pytest tests/security/       # Security tests
python -m pytest tests/performance/    # Load tests

# Suppress TLS verification warnings in tests
export PYTHONWARNINGS="ignore:Unverified HTTPS request"
```

**Note**: The `test_rate_limiting_integration` test may fail due to Redis dependency. All other auth tests pass with token revocation fully functional.

### Local Development
```bash
# Start all services locally
./start_all_services.sh

# Stop all services
kill $(cat logs/service_pids.txt) 2>/dev/null

# Test API Gateway health (with TLS verification disabled for self-signed certs)
curl -k https://localhost:8443/health

# Test token revocation functionality
python3 test_revocation_now.py

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
```

## Code Style Guidelines

### Go (1.21)
- Standard project layout with `src/` directory
- Structured logging with contextual information
- Clean architecture: handlers, services, repositories layers
- Error wrapping: `fmt.Errorf("operation failed: %w", err)`
- Environment variables via `getEnv(key, default)` helper
- TLS 1.3 only, mTLS between services

### Python (3.11)
- PEP 8 with type hints required
- Flask 2.3.3 with blueprint organization
- PyJWT 2.8.0 RS256 for authentication
- Requests for HTTP clients with session reuse
- Custom exceptions in `exceptions.py`

### General
- Feature branches: `001-feature-name` format
- No automated CI/CD - manual quality gates only
- Never commit secrets/API keys
- Use absolute paths from repo root
- Error messages to stderr (`>&2`)

## Troubleshooting & Debugging

### Token Revocation Issues
**Problem**: Revoked tokens still accepted by API gateway
**Solution**: 
1. Ensure API gateway is using gRPC authentication (enhanced version)
2. Check auth service is running on port 8081 (gRPC) and 8082 (HTTP)
3. Verify environment variables: `AUTH_SERVICE_GRPC=localhost:8081`
4. Confirm API gateway logs show "Auth service gRPC connection established"

### gRPC Connectivity Patterns
- **Localhost connections**: Use insecure transport (`grpc.WithTransportCredentials(insecure.NewCredentials())`)
- **Production connections**: Use TLS with certificates
- **Fallback design**: Services should handle missing gRPC connections gracefully
- **Port configuration**: Auth service gRPC on 8081, HTTP on 8082

### Certificate Management
- **Self-signed certs**: Use `verify=False` in Python tests
- **Certificate paths**: Relative paths from service directories (e.g., `../certs/`)
- **Local testing**: Insecure transport acceptable for localhost connections
- **Production**: Always use TLS 1.3 with valid certificates

### Service Ports (Current Configuration)
- **Auth Service**: gRPC 8081, HTTP 8082
- **Inference Pool**: gRPC 50051
- **API Gateway**: HTTPS 8443 (TLS), HTTP fallback if no certs
- **Monitoring**: gRPC 8083

### Key Environment Variables
```
TLS_CERT="../certs/api-gateway.crt"
TLS_KEY="../certs/api-gateway-key.pem"
INFERENCE_POOL_URL="localhost:50051"
AUTH_SERVICE_GRPC="localhost:8081"
AUTH_SERVICE_URL="localhost:8081"
PORT="8443"
HTTP_PORT="8082" (auth service HTTP port)
```