# AGENTS.md

HelixFlow AI inference platform configuration repository with Specify development workflow. This is an enterprise-grade AI inference platform providing universal access to 300+ AI models through a single OpenAI-compatible API, emphasizing developer experience, performance, security, and multi-cloud deployment capabilities.

## Project Overview

HelixFlow is a comprehensive microservices-based AI inference platform designed for enterprise-scale deployment. The platform provides a unified API gateway that routes requests to specialized inference pools, with full OpenAI API compatibility for seamless integration.

**Key Capabilities:**
- Universal AI model access (300+ models) via OpenAI-compatible API
- Multi-cloud deployment (AWS, Azure, GCP) with Terraform IaC
- Enterprise-grade security with zero-trust architecture
- Sub-100ms latency for popular models with 99.9% uptime SLA
- Comprehensive SDK support across 7+ programming languages

## Technology Stack

### Core Services (Go 1.21)
- **API Gateway**: Gorilla Mux router with Redis caching
- **Authentication Service**: JWT RS256 with bcrypt password hashing
- **Inference Pool**: gRPC-based model orchestration with GPU support

### Backend Infrastructure
- **Container Orchestration**: Kubernetes with Istio service mesh
- **Databases**: PostgreSQL (SQLCipher AES-256), Redis Cluster, Neo4j, Qdrant
- **Monitoring**: Prometheus, Grafana, Sentry, ELK Stack
- **Security**: TLS 1.3, mTLS, JWT RS256, SOC 2/GDPR compliance

### SDKs & Client Libraries
- **Python**: Flask-based services, setuptools packaging
- **JavaScript/TypeScript**: Node.js SDK with npm publishing
- **Java 17, C# .NET 8, Rust 1.75**: Enterprise integration libraries
- **PHP**: Web application integrations

## Project Structure

```
/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/
├── api-gateway/                # API Gateway service (Go)
│   ├── src/main.go            # Main gateway implementation
│   └── requirements.txt       # Flask 2.3.3, PyJWT 2.8.0, cryptography 41.0.4
├── auth-service/              # Authentication service (Go)
│   ├── src/main.go            # JWT, API key management
│   └── requirements.txt       # Flask, bcrypt 4.0.1, Redis 4.6.0
├── inference-pool/            # AI model inference service (Go)
│   ├── src/main.go            # Model orchestration
│   └── src/grpc/              # gRPC protocols
├── k8s/                       # Kubernetes manifests
│   ├── api-gateway.yaml       # Gateway deployment (3 replicas)
│   ├── istio-config.yaml      # Service mesh configuration
│   ├── prometheus-config.yaml # Monitoring setup
│   └── gpu-operator.yaml      # GPU scheduling
├── terraform/                 # Infrastructure as Code
│   ├── aws/                   # AWS-specific modules
│   ├── azure/                 # Azure-specific modules
│   ├── gcp/                   # GCP-specific modules
│   └── modules/vpc/           # Reusable VPC module
├── sdks/                      # Language-specific SDKs
│   └── python/                # Python SDK (v1.0.0)
├── tests/                     # Comprehensive test suite
│   ├── integration/           # Integration tests
│   ├── contract/              # API contract tests
│   ├── security/              # Security tests
│   └── performance/           # Load testing
├── .specify/                  # Development workflow automation
├── .opencode/                 # AI agent command definitions
└── specs/                     # Feature specifications
    └── 001-helixflow-complete-spec/  # Current feature branch
```

## Build and Test Commands

### Service Development
```bash
# API Gateway (Go)
cd api-gateway && go build -o bin/api-gateway src/main.go

# Authentication Service (Go)  
cd auth-service && go build -o bin/auth-service src/main.go

# Inference Pool (Go)
cd inference-pool && go build -o bin/inference-pool src/main.go

# Python SDK
cd sdks/python && python setup.py build
```

### Testing Strategy
```bash
# Integration tests (Python)
python -m pytest tests/integration/

# Contract tests
python -m pytest tests/contract/

# Security tests
python -m pytest tests/security/

# Performance tests
python -m pytest tests/performance/
```

### Deployment
```bash
# Kubernetes deployment
kubectl apply -f k8s/api-gateway.yaml
kubectl apply -f k8s/istio-config.yaml

# Terraform infrastructure
terraform init
terraform plan
terraform apply
```

## Development Commands

### Feature Management
- Create new feature: `.specify/scripts/bash/create-new-feature.sh <feature-name>`
- Check prerequisites: `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- Setup implementation plan: `.specify/scripts/bash/setup-plan.sh`
- Update agent context: `.specify/scripts/bash/update-agent-context.sh`

### Git Operations
- **Upstream**: git@github.com:Helix-Flow/Platform.git
- **Feature branches**: Must use 3-digit prefix (e.g., 001-feature-name)
- **Current branch detection**: Uses SPECIFY_FEATURE env var or git branch

### CI/CD and Quality Assurance
- **Manual Execution**: All CI/CD processes must be executed manually
- **No GitHub Actions**: No automated workflows configured
- **No Git Hooks**: No pre-commit, pre-push, or other automated Git hooks
- **Manual Quality Gates**: All testing, linting, and reviews performed manually
- **Explicit Commands**: Developers must run all checks before commits and deployments

## Code Style Guidelines

### Go Services
- Use Go 1.21 with standard project layout
- Implement structured logging with contextual information
- Use dependency injection for testability
- Follow clean architecture with separate layers
- Implement comprehensive error handling with wrapped errors

### Python Services
- **Flask 2.3.3** for web services with proper blueprints
- Use **PyJWT 2.8.0** for JWT handling with RS256 algorithm
- Implement **bcrypt 4.0.1** for password hashing
- Use **cryptography 41.0.4** for encryption operations
- Follow PEP 8 style guidelines with type hints

### Bash/Shell Scripts (.sh)
- Use `#!/usr/bin/env bash` shebang
- Source `common.sh` for shared functions (get_repo_root, get_current_branch, etc.)
- Use absolute paths from `get_repo_root()`
- Handle both git and non-git repositories gracefully
- Follow existing function patterns: `check_*`, `get_*`, `find_*`

### File Organization
- **Scripts**: `.specify/scripts/bash/` with executable permissions
- **Templates**: `.specify/templates/` with `.md` extension
- **Feature specs**: `specs/XXX-feature-name/` directories
- **Opencode commands**: `.opencode/command/` with `.md` extension

### Error Handling
- Explicit error checking with meaningful messages
- Use `>&2` for error output
- Return appropriate exit codes
- Validate prerequisites before execution
- Never commit secrets or API keys

### Naming Conventions
- **Feature branches**: 3-digit prefix with hyphen (001-feature-name)
- **Functions**: snake_case with descriptive names
- **Variables**: UPPER_SNAKE_CASE for exports, lower_snake_case for locals
- **Files**: kebab-case for directories, descriptive names for scripts

## Security Considerations

### Encryption & TLS
- **TLS 1.3** minimum for all communications
- **SQLCipher AES-256** encryption for PostgreSQL
- **mTLS** between all microservices
- **JWT RS256** for API authentication

### Authentication & Authorization
- Multi-factor authentication support
- API key management with rotation
- Role-based access control (RBAC)
- Zero-trust architecture implementation

### Compliance
- SOC 2 Type II compliance
- GDPR data protection
- CCPA privacy regulations
- Enterprise security standards

## Performance Requirements

### Latency Targets
- **Sub-100ms** for popular AI models
- **Global distribution** with multi-region deployment
- **Caching strategy** using Redis Cluster
- **GPU optimization** with model quantization

### Scalability
- **Horizontal scaling** with Kubernetes HPA
- **99.9% uptime** SLA guarantee
- **Load balancing** with Istio service mesh
- **Auto-scaling** based on demand

## Current Implementation Status

- **Branch**: `001-helixflow-complete-spec`
- **Phase**: Complete implementation (Phases 6-9)
- **Auto-generated**: Model quantization code (`quantization.go`)
- **Testing**: Integration, security, and performance tests implemented
- **Documentation**: Comprehensive technical specification available

## Active Technologies

- **Languages**: Python 3.11, Go 1.21, JavaScript/TypeScript, Java 17, C# .NET 8, Rust 1.75, PHP
- **Infrastructure**: Kubernetes, Istio, PostgreSQL, Redis, Prometheus, Grafana, Sentry, Terraform, ArgoCD, Docker, Consul
- **Storage**: PostgreSQL with SQLCipher AES-256 encryption, Redis Cluster, Neo4j (for Cognee), Qdrant/Pinecone (vector databases)
- **Security**: TLS 1.3, JWT RS256, mTLS, SOC 2/GDPR compliance

## Recent Changes

- 001-helixflow-complete-spec: Added comprehensive enterprise technology stack with multi-language SDK support, Kubernetes orchestration, and enterprise-grade security implementation