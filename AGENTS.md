# AGENTS.md

HelixFlow AI inference platform - enterprise-grade microservices providing OpenAI-compatible API access to 300+ AI models.

## Build/Test Commands

### Go Services
```bash
# Build services
cd api-gateway && go build -o bin/api-gateway src/main.go
cd auth-service && go build -o bin/auth-service src/main.go  
cd inference-pool && go build -o bin/inference-pool src/main.go

# Python SDK
cd sdks/python && python setup.py build
```

### Testing
```bash
# Run single test
python -m pytest tests/integration/test_auth.py::test_login -v

# Test categories
python -m pytest tests/integration/    # Integration tests
python -m pytest tests/contract/       # API contract tests
python -m pytest tests/security/       # Security tests
python -m pytest tests/performance/    # Load tests
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