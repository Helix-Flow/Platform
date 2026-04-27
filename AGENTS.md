# AGENTS.md

HelixFlow AI inference platform - enterprise-grade microservices providing OpenAI-compatible API access to AI models.

## Project Overview

HelixFlow is a comprehensive AI inference platform built as a set of independent Go microservices with Python-based testing and SDKs. It exposes an OpenAI-compatible REST API for chat completions and model management, with gRPC-based internal service communication.

**Core Services:**
- **API Gateway** (`api-gateway/`) - HTTPS/WebSocket entry point, OpenAI-compatible REST API, rate limiting, request routing
- **Auth Service** (`auth-service/`) - JWT authentication, user management, API keys, token revocation
- **Inference Pool** (`inference-pool/`) - GPU resource management, model caching, inference execution (mock engine)
- **Monitoring Service** (`monitoring/`) - System metrics, alerting, predictive scaling recommendations

**Technology Stack:**
- **Go 1.22.2** - All microservices
- **Python 3.11+** - Testing framework, client SDK
- **gRPC + Protocol Buffers** - Internal service communication
- **SQLite** (development) / **PostgreSQL** (production) - Primary database
- **Redis** - Caching and rate limiting
- **Prometheus + Grafana** - Metrics and dashboards
- **Kubernetes + Helm + Terraform** - Deployment infrastructure

**Protocols:**
- HTTP/HTTPS (external), WebSocket (streaming)
- gRPC (internal service-to-service)
- TLS 1.3 for external, mTLS between internal services

---

## Directory Structure

```
HelixFlow/
├── api-gateway/              # API Gateway service (Go)
│   ├── src/                  # Go source files (package main)
│   ├── bin/                  # Compiled binary output
│   └── requirements.txt      # Python deps (FastAPI placeholder)
├── auth-service/             # Authentication service (Go)
│   ├── src/                  # Go source files
│   ├── bin/                  # Compiled binary output
│   └── requirements.txt      # Python deps (FastAPI placeholder)
├── inference-pool/           # AI inference service (Go)
│   ├── src/                  # Go source files
│   ├── bin/                  # Compiled binary output
│   └── requirements.txt      # Python deps (FastAI/ML placeholder)
├── monitoring/               # Monitoring service (Go)
│   ├── src/                  # Go source files
│   ├── bin/                  # Compiled binary output
│   └── requirements.txt      # Python deps (FastAPI placeholder)
├── internal/                 # Shared internal libraries
│   └── database/             # Database abstraction (SQLite + PostgreSQL)
├── helixflow/                # Generated gRPC protobuf Go modules
│   ├── auth/                 # Auth service protobuf stubs
│   ├── inference/            # Inference service protobuf stubs
│   └── monitoring/           # Monitoring service protobuf stubs
├── proto/                    # Protocol buffer definitions (.proto files)
├── tests/                    # Comprehensive test suite
│   ├── conftest.py           # Pytest configuration and fixtures
│   ├── complete_testing_framework.py
│   ├── final_validation_suite.py
│   ├── integration/          # End-to-end integration tests
│   ├── contract/             # API contract and compliance tests
│   ├── security/             # Penetration testing
│   ├── performance/          # Load testing
│   └── unit/                 # Unit tests
├── scripts/                  # Deployment and management shell scripts
├── sdks/                     # Client SDKs
│   └── python/               # Python SDK (HTTP client)
├── k8s/                      # Kubernetes manifests
├── helm/                     # Helm charts
├── terraform/                # Infrastructure as code (AWS, Azure, GCP)
├── schemas/                  # Database schemas and configs
├── certs/                    # TLS certificates and JWT keys
├── data/                     # SQLite database files
├── logs/                     # Service logs
├── nginx/                    # Nginx reverse proxy configuration
├── docs/                     # Project documentation
├── docker-compose.yml        # Local development stack
├── requirements-master.txt   # Python testing dependencies
└── .github/workflows/        # GitHub Actions CI/CD
```

---

## Build System

### Go Services

Each service is an independent Go module with local `replace` directives for shared libraries.

```bash
# Build all services
cd api-gateway/src && go build -o ../bin/api-gateway .
cd auth-service/src && go build -o ../bin/auth-service .
cd inference-pool/src && go build -o ../bin/inference-pool .
cd monitoring/src && go build -o ../bin/monitoring .

# Build shared libraries
cd internal/database && go build .
cd helixflow/auth && go build .
cd helixflow/inference && go build .
cd helixflow/monitoring && go build .

# Clean builds
find . -name "bin" -type d -exec rm -rf {} + 2>/dev/null
find . -name "*.out" -delete
```

**Go Module Structure:**
| Module | Path | Key Dependencies |
|--------|------|------------------|
| `helixflow/api-gateway` | `api-gateway/src/go.mod` | gorilla/mux, gorilla/websocket, go-redis/v9, grpc |
| `helixflow/auth-service` | `auth-service/src/go.mod` | golang-jwt/jwt/v5, google/uuid, golang.org/x/crypto, grpc |
| `helixflow/inference-pool` | `inference-pool/src/go.mod` | grpc only |
| `helixflow/monitoring-service` | `monitoring/src/go.mod` | go-redis/v9, grpc |
| `helixflow/database` | `internal/database/go.mod` | lib/pq, go-sqlite3, go-redis/v9 |
| `helixflow/auth` | `helixflow/auth/go.mod` | grpc, protobuf |
| `helixflow/inference` | `helixflow/inference/go.mod` | minimal |
| `helixflow/monitoring` | `helixflow/monitoring/go.mod` | minimal |

All services use `replace` directives to reference local `helixflow/` and `internal/database` modules.

### Python Environment

```bash
# Install all test dependencies
pip install -r requirements-master.txt

# Build Python SDK
cd sdks/python && python setup.py build

# Create virtual environment
python -m venv venv
source venv/bin/activate
pip install -r requirements-master.txt
```

### Protocol Buffer Generation

```bash
# Generate Go code from .proto files
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/*.proto
```

Generated code is committed to `helixflow/{auth,inference,monitoring}/`.

---

## Testing Strategy

### Test Categories

| Category | Directory | Description |
|----------|-----------|-------------|
| Unit | `tests/unit/` | API Gateway mock-based unit tests |
| Integration | `tests/integration/` | End-to-end service integration, auth, OpenAI compat, compliance, scalability, service mesh, multicloud |
| Contract | `tests/contract/` | API compliance, performance SLAs, security contracts, infrastructure validation |
| Security | `tests/security/` | Penetration testing (SQLi, XSS, CSRF, SSRF, brute force, headers) |
| Performance | `tests/performance/` | Async load testing with aiohttp |

### Running Tests

```bash
# Run all integration tests via bash script (starts services automatically)
./scripts/test_integration.sh

# Run pytest test categories
python -m pytest tests/unit/ -v
python -m pytest tests/integration/ -v
python -m pytest tests/contract/ -v
python -m pytest tests/security/ -v
python -m pytest tests/performance/ -v

# Run specific test files
python -m pytest tests/integration/test_auth.py -v
python -m pytest tests/security/test_security_pentest.py -v

# Root-level Python test scripts
python3 test_auth_api.py
python3 test_revocation_now.py
python3 final_integration_test.py

# Quick smoke test (starts monitoring + gateway only)
./scripts/quick_test.sh

# Suppress TLS verification warnings in tests
export PYTHONWARNINGS="ignore:Unverified HTTPS request"
```

**Important Notes:**
- Integration tests target `localhost` (not Kubernetes service URLs)
- All HTTPS tests use `verify=False` due to self-signed certificates
- The `test_rate_limiting_integration` test may fail without Redis
- Services must be pre-built in `./{service}/bin/` directories for bash-based test scripts

### Test Configuration

`tests/conftest.py` defines custom markers (`integration`, `contract`, `security`, `performance`) and fixtures:
- `test_config`: Service URLs (`https://localhost:8443`, `http://localhost:8082`, etc.)
- `sample_chat_request`: GPT-3.5-turbo request template
- `sample_auth_credentials`: testuser/testpass123

---

## Development Workflow

### Service Dependencies

```
Clients → API Gateway (HTTPS 8443)
              │
              ├──gRPC──→ Auth Service (gRPC 8081 / HTTP 8082)
              │              │
              │              └──→ Database (SQLite/PostgreSQL)
              │
              ├──gRPC──→ Inference Pool (gRPC 50051)
              │
              └──gRPC──→ Monitoring (gRPC / HTTP 8083)
```

### Service Port Configuration

| Service | Protocol | Port | Notes |
|---------|----------|------|-------|
| API Gateway | HTTPS | 8443 | TLS 1.3; falls back to HTTP if no certs |
| API Gateway | HTTP | 8080 | HTTP fallback / Docker Compose mode |
| Auth Service | gRPC | 8081 | Internal auth validation |
| Auth Service | HTTP | 8082 | REST endpoints for login/register |
| Inference Pool | gRPC | 50051 | Model inference and streaming |
| Monitoring | HTTP/gRPC | 8083 | Metrics and health |
| PostgreSQL | SQL | 5432 | Production database |
| Redis | Redis | 6379 | Caching and rate limiting |
| Prometheus | HTTP | 9091 | Metrics collection |
| Grafana | HTTP | 3000 | Dashboards |

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

# Ports
PORT="8443"           # API Gateway HTTPS
HTTP_PORT="8082"      # Auth Service HTTP

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
```

### Local Development Startup

```bash
# Start all services with proper configuration
./start_all_services.sh

# Start service groups
./start_phase1_services.sh  # Core services
./start_phase2_services.sh  # Advanced services

# Development environment startup (kills existing, starts all, saves PIDs)
./scripts/start_development.sh

# Stop all services
kill $(cat logs/service_pids.txt) 2>/dev/null
```

### Manual Service Startup (Debugging)

```bash
# Auth service (gRPC 8081, HTTP 8082)
cd auth-service && HTTP_PORT=8082 PORT=8081 DATABASE_TYPE=sqlite DATABASE_PATH=../data/helixflow.db ./bin/auth-service

# Inference pool (gRPC 50051)
cd inference-pool && PORT=50051 ./bin/inference-pool

# API gateway (TLS 8443)
cd api-gateway && TLS_CERT="../certs/api-gateway.crt" TLS_KEY="../certs/api-gateway-key.pem" INFERENCE_POOL_URL=localhost:50051 AUTH_SERVICE_GRPC=localhost:8081 PORT=8443 ./bin/api-gateway

# Monitoring service (gRPC + HTTP 8083)
cd monitoring && PORT=8083 ./bin/monitoring
```

### Docker Compose Development

```bash
# Start database and infrastructure services only
docker-compose up -d postgres redis prometheus grafana nginx

# Generate TLS certificates
./scripts/generate_certificates.sh

# Setup databases
./scripts/setup_sqlite_database.sh
./scripts/setup_postgresql.sh
```

**Note:** `docker-compose.yml` references Dockerfiles in service directories (`api-gateway/Dockerfile`, etc.), but these Dockerfiles do not currently exist in the repository.

---

## Code Organization

### Go Service Architecture

Each Go service uses `package main` with all source files in a single `src/` directory. There is no strict subdirectory separation into `handlers/`, `services/`, `repositories/`.

**API Gateway** (`api-gateway/src/`):
- `main.go` - Primary HTTP gateway: gorilla/mux router, OpenAI-compatible REST API (`/v1/chat/completions`, `/v1/models`), health checks, auth via gRPC, rate limiting, TLS 1.3 with HTTP fallback, mock inference fallback
- `main_grpc.go` - Alternative enhanced gRPC gateway (`APIGatewayGRPC`): mTLS connections to all backend services, uses gRPC for inference calls
- `inference_handler.go` - `InferenceHandler`: wraps gRPC calls to inference pool, converts OpenAI format to/from protobuf
- `websocket_handler.go` - WebSocket manager with connection pooling, ping/pong, broadcast, simulated streaming
- `rate_limiter_advanced.go` - Token bucket, sliding window, and fixed window rate limiting with Redis

**Auth Service** (`auth-service/src/`):
- `main.go` - Dual-server entry: HTTP REST (port 8082) in goroutine, gRPC (port 8081) in main thread
- `auth_service.go` - `AuthServiceServer`: RSA key pair generation, bcrypt password hashing, JWT access/refresh tokens (RS256, UUID v4 JTI), token revocation blacklist with expiry cleanup, rate limiting, API key management
- `http_handler.go` - `AuthHTTPServer`: wraps gRPC methods into HTTP endpoints (`/login`, `/refresh`, `/revoke`, `/register`, `/health`)

**Inference Pool** (`inference-pool/src/`):
- `main.go` - gRPC server: `GPUManager` (4 mock GPUs), `ModelCache` (LRU), `InferenceEngine` (mock responses), job queue with 10 workers, streaming inference
- `inference_engine.go` - Mock inference with keyword-based contextual responses for `gpt-3.5-turbo`, `gpt-4`, `claude-v1`, `llama-2-70b`
- `gpu_optimizer.go` - GPU scheduling (best-fit, first-fit, round-robin, least-loaded), memory defragmentation, model eviction
- `quantization.go` - Model quantization simulator (4-bit, 8-bit, 16-bit)

**Monitoring Service** (`monitoring/src/`):
- `main.go` - Entry point: gRPC server + HTTP server (port 8083)
- `main_grpc.go` - `MonitoringServiceServer`: mock system metrics, GPU metrics, alert rule CRUD, alert acknowledgment, predictive scaling recommendations, gRPC health checking
- `predictive_scaling.go` - `PredictiveScaler`: linear regression on CPU history in Redis, recommends replica counts (min 3, max 20)

### Shared Libraries

**`internal/database/`** - Database abstraction layer:
- `interfaces.go` - `DatabaseManager` interface
- `sqlite_manager.go` - SQLite implementation (development default)
- `postgres_manager.go` / `postgres_advanced.go` - PostgreSQL implementation
- `config.go` - Database configuration

**`helixflow/{auth,inference,monitoring}/`** - Generated gRPC/protobuf Go code:
- `*.pb.go` - Message structs
- `*_grpc.pb.go` - gRPC client/server interfaces

### Python SDK

**`sdks/python/helixflow/`**:
- `client.py` - `HelixFlow` HTTP client class with Bearer auth: `chat_completion()`, `chat_completion_stream()`, `list_models()`, `get_model()`
- `exceptions.py` - `HelixFlowError`, `AuthenticationError`, `RateLimitError`, `APIError`
- `__init__.py` - Package exports

---

## Deployment & Operations

### GitHub Actions CI/CD

`.github/workflows/ci-cd.yml` defines a pipeline triggered on push to `main`/`develop` and PRs to `main`:

1. **test** - Go tests with coverage (uses Redis + PostgreSQL services), uploads to Codecov
2. **security-scan** - Trivy vulnerability scanner, uploads SARIF to GitHub CodeQL
3. **lint** - golangci-lint with 5m timeout
4. **build** - Builds all 4 Go binaries + Docker images (needs test+scan+lint)
5. **deploy-dev** - Triggered on `develop` branch (placeholder ArgoCD sync)
6. **deploy-staging** - Triggered on `main` branch (placeholder)
7. **deploy-prod** - Triggered on `main` push after staging (placeholder)

**Note:** CI uses Go 1.21 but the project requires Go 1.22.2. The pipeline is Go-centric and does not run Python pytest tests.

### Kubernetes

`k8s/` contains 18 manifests for production infrastructure:
- `api-gateway.yaml` - Deployment + Service for API Gateway
- `argocd-config.yaml` - ArgoCD GitOps setup with Application CRDs
- `cert-manager.yaml` - Let's Encrypt certificate provisioning
- `consul-config.yaml` - Service mesh configuration
- `elasticsearch-config.yaml` / `logstash-config.yaml` - Logging stack
- `external-secrets.yaml` - AWS Secrets Manager integration
- `gpu-operator.yaml` - NVIDIA GPU Operator + AMD ROCm
- `grafana-dashboards.yaml` - Dashboard ConfigMaps
- `istio-config.yaml` / `istio-policies.yaml` - Istio service mesh, STRICT mTLS
- `kustomization.yaml` - Kustomize manifest
- `mtls-config.yaml` - TLS secrets, PeerAuthentication, AuthorizationPolicy
- `prometheus-config.yaml` / `prometheus-alert-rules.yaml` - Monitoring and alerting
- `sentry-config.yaml` - Sentry error tracking

**Note:** Only `api-gateway.yaml` exists as a direct service manifest. K8s manifests for `auth-service`, `inference-pool`, and `monitoring` deployments are referenced by ArgoCD but do not exist as separate files.

### Helm

`helm/helixflow/` contains `Chart.yaml` (v1.0.0) and `values.yaml` but **no template files** in `templates/`.

### Terraform

Multi-cloud infrastructure in `terraform/{aws,azure,gcp}/`:
- **AWS**: EKS (v1.28), RDS PostgreSQL 15.4, ElastiCache Redis, GPU node groups (g4dn)
- **Azure**: AKS (v1.28), PostgreSQL Flexible Server, Azure Cache for Redis, GPU nodes (NC6s_v3)
- **GCP**: GKE, Cloud SQL PostgreSQL 15, Memorystore Redis, GPU nodes (n1 + K80)

**Note:** Terraform configs reference local modules (`../modules/vpc`, etc.) that do not exist in the repository.

### Production Scripts

| Script | Purpose |
|--------|---------|
| `scripts/production-deployment.sh` | Full production deployment orchestration |
| `scripts/production-validation.sh` | Pre-production validation |
| `scripts/final-validation.sh` | Comprehensive validation suite |
| `scripts/quality-gates.sh` | CI-like quality gate checks |
| `scripts/setup_production_infrastructure.sh` | Complete production infrastructure setup |
| `scripts/generate_certificates.sh` | Full PKI generation (CA, per-service certs, JWT RSA keys) |

---

## Security Considerations

- **TLS 1.3** for all external communications
- **mTLS** between internal services using certificates in `certs/`
- **JWT Authentication** with RS256 signing (2048-bit RSA keys generated at runtime)
- **Token Revocation** via in-memory blacklist with expiration cleanup
- **Rate Limiting** on auth validation (configurable windows)
- **UUID v4 validation** for JWT JTI claims
- **Password Hashing** with bcrypt
- **API Keys** with hashed storage and prefix identification
- **Istio STRICT mTLS** enforced in Kubernetes
- **Self-signed certificates** used for local development

### Certificate Management

Certificates are stored in `certs/`:
- CA certificates (`ca.pem`, `helixflow-ca.pem`)
- Per-service server certificates (`api-gateway.crt`, `api-gateway-key.pem`, etc.)
- Per-service client certificates for mTLS
- PKCS12 files (password: `helixflow123`)
- JWT signing keys (`jwt-private.pem`, `jwt-public.pem`)

Generation: `./scripts/generate_certificates.sh` or `./certs/generate-certificates.sh`

---

## Database Management

### SQLite (Development)

```bash
export DATABASE_TYPE=sqlite
export DATABASE_PATH=../data/helixflow.db
./scripts/setup_sqlite_database.sh
```

### PostgreSQL (Production)

```bash
export DATABASE_TYPE=postgres
export DATABASE_URL="postgres://user:pass@localhost:5432/helixflow"
./scripts/setup_postgresql.sh
```

### Schema Files

| File | Description |
|------|-------------|
| `schemas/postgresql-helixflow.sql` | Original schema with 9 core tables |
| `schemas/postgresql-helixflow-complete.sql` | Expanded schema with RBAC, API keys, inference logs, usage tracking |
| `schemas/postgresql-helixflow-updated.sql` | Clean DDL with sample data |
| `schemas/migrate.sh` | Migration script with generate/run/rollback/status commands |
| `schemas/redis-cluster.conf` | Redis cluster configuration |
| `schemas/neo4j-config.properties` | Neo4j knowledge graph config |
| `schemas/qdrant-config.yaml` | Qdrant vector DB config |

---

## Known Gaps and Limitations

1. **Missing Dockerfiles** - `docker-compose.yml` and CI/CD reference Dockerfiles in service directories, but none exist.
2. **Missing Helm Templates** - `helm/helixflow/` has `Chart.yaml` and `values.yaml` but no `templates/` directory.
3. **Missing K8s Service Manifests** - Only `k8s/api-gateway.yaml` exists; manifests for auth-service, inference-pool, and monitoring are missing.
4. **Missing Terraform Modules** - Referenced local modules (`../modules/vpc`, etc.) do not exist.
5. **CI/CD Go Version Mismatch** - `.github/workflows/ci-cd.yml` uses Go 1.21, but `go.mod` files require 1.22.2.
6. **Mock Inference Engine** - The inference pool generates keyword-based mock responses; it does not connect to real LLM backends.
7. **Database Abstraction Gap** - The auth service may default to SQLite regardless of `DATABASE_TYPE` in some code paths.
8. **No Node.js/Rust** - The project is purely Go + Python. The VS Code extension in `extensions/vscode/` may contain `package.json` but is not part of the core platform.

---

## Troubleshooting

### Token Revocation Problems
- Ensure API gateway uses gRPC authentication (enhanced `main_grpc.go` version)
- Verify auth service runs on port 8081 (gRPC) and 8082 (HTTP)
- Check `AUTH_SERVICE_GRPC=localhost:8081` is set
- Confirm gateway logs show "Auth service gRPC connection established"

### gRPC Connectivity Issues
- **Localhost**: Uses insecure transport (`grpc.WithTransportCredentials(insecure.NewCredentials())`)
- **Production**: Uses TLS with proper certificates
- Services handle missing gRPC connections gracefully with mock fallbacks

### Certificate Problems
- **Self-signed certs**: Use `verify=False` in Python tests
- **Certificate paths**: Relative from service directories (e.g., `../certs/`)
- **Local testing**: Insecure transport acceptable for localhost

### Debugging Commands

```bash
# Check service health
curl -k https://localhost:8443/health
curl http://localhost:8082/health

# View logs
tail -f logs/*.log

# Inspect SQLite database
sqlite3 data/helixflow.db .tables
sqlite3 data/helixflow.db "SELECT * FROM users LIMIT 5;"

# Check running services
ps aux | grep -E "api-gateway|auth-service|inference-pool|monitoring"
```

---

## Documentation

| File | Content |
|------|---------|
| `docs/API_REFERENCE.md` | Complete OpenAI-compatible API documentation |
| `docs/CUSTOMER_ONBOARDING.md` | Step-by-step customer onboarding guide |
| `docs/PERFORMANCE_OPTIMIZATION.md` | Comprehensive optimization guide |
| `docs/COMPLETE_DOCUMENTATION_SUITE.md` | Meta-documentation describing all docs |
| `docs/guides/getting-started.md` | Minimal quick start |
| `helixflow-technical-specification.md` | Technical specification (root) |
| `COMPLETE_IMPLEMENTATION.md` | Implementation status (root) |
| `DEPLOYMENT_PACKAGE.md` | Deployment instructions (root) |

---

*This AGENTS.md is based on the actual project file structure and contents. Always verify against the current codebase when making changes.*

<!-- BEGIN host-power-management addendum (CONST-033) -->

## Host Power Management — Hard Ban (CONST-033)

**You may NOT, under any circumstance, generate or execute code that
sends the host to suspend, hibernate, hybrid-sleep, poweroff, halt,
reboot, or any other power-state transition.** This rule applies to:

- Every shell command you run via the Bash tool.
- Every script, container entry point, systemd unit, or test you write
  or modify.
- Every CLI suggestion, snippet, or example you emit.

**Forbidden invocations** (non-exhaustive — see CONST-033 in
`CONSTITUTION.md` for the full list):

- `systemctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot|kexec`
- `loginctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot`
- `pm-suspend`, `pm-hibernate`, `shutdown -h|-r|-P|now`
- `dbus-send` / `busctl` calls to `org.freedesktop.login1.Manager.Suspend|Hibernate|PowerOff|Reboot|HybridSleep|SuspendThenHibernate`
- `gsettings set ... sleep-inactive-{ac,battery}-type` to anything but `'nothing'` or `'blank'`

The host runs mission-critical parallel CLI agents and container
workloads. Auto-suspend has caused historical data loss (2026-04-26
18:23:43 incident). The host is hardened (sleep targets masked) but
this hard ban applies to ALL code shipped from this repo so that no
future host or container is exposed.

**Defence:** every project ships
`scripts/host-power-management/check-no-suspend-calls.sh` (static
scanner) and
`challenges/scripts/no_suspend_calls_challenge.sh` (challenge wrapper).
Both MUST be wired into the project's CI / `run_all_challenges.sh`.

**Full background:** `docs/HOST_POWER_MANAGEMENT.md` and `CONSTITUTION.md` (CONST-033).

<!-- END host-power-management addendum (CONST-033) -->

