# Data Model: HelixFlow Complete Nano-Level Specification

**Date**: 2025-12-13
**Feature**: specs/001-helixflow-complete-spec/spec.md

## Entity Definitions

### AI Models
**Purpose**: Representing 300+ AI models with capabilities, pricing, and configuration details

**Fields**:
- `model_id` (UUID, Primary Key): Unique identifier for the model
- `provider` (String, Required): Provider name (OpenAI, Anthropic, Google, Meta, etc.)
- `model_name` (String, Required): Model identifier (gpt-4, claude-3, gemini-pro, etc.)
- `model_type` (Enum, Required): TEXT, IMAGE, AUDIO, MULTIMODAL, CODE
- `context_window` (Integer, Required): Maximum token context length
- `pricing_per_token_input` (Decimal, Required): Cost per input token
- `pricing_per_token_output` (Decimal, Required): Cost per output token
- `capabilities` (JSON Array, Required): List of supported capabilities
- `aliases` (JSON Array, Optional): Alternative names for the model
- `status` (Enum, Required): ACTIVE, DEPRECATED, BETA
- `created_at` (Timestamp, Required): Model registration timestamp
- `updated_at` (Timestamp, Required): Last update timestamp

**Validation Rules**:
- `context_window` must be positive integer
- `pricing_per_token_*` must be non-negative decimals
- `capabilities` must contain at least one capability
- `model_name` must be unique per provider

**Relationships**:
- One-to-many with API Endpoints (models can be accessed via multiple endpoints)

### Users
**Purpose**: Representing platform users including developers, enterprises, and researchers

**Fields**:
- `user_id` (UUID, Primary Key): Unique user identifier
- `email` (String, Required, Unique): User email address
- `subscription_tier` (Enum, Required): FREE, PRO, ENTERPRISE, RESEARCH
- `api_key_hash` (String, Required): Hashed API key for authentication
- `usage_limits` (JSON, Required): Monthly token limits and rate limits
- `billing_info` (JSON, Optional): Payment method and billing address
- `organization_id` (UUID, Optional): For enterprise users
- `created_at` (Timestamp, Required): Account creation timestamp
- `last_active` (Timestamp, Required): Last API usage timestamp
- `status` (Enum, Required): ACTIVE, SUSPENDED, DELETED

**Validation Rules**:
- `email` must be valid email format
- `usage_limits` must contain token_limit and rate_limit fields
- Enterprise users must have `organization_id`

**Relationships**:
- One-to-many with API Endpoints (users can have multiple active sessions)
- Many-to-one with Organizations (for enterprise accounts)

### Services
**Purpose**: Representing microservices with health checks and scaling configurations

**Fields**:
- `service_id` (UUID, Primary Key): Unique service identifier
- `service_name` (String, Required, Unique): Service name (api-gateway, inference-pool, etc.)
- `service_type` (Enum, Required): GATEWAY, INFERENCE, AUTH, MONITORING, STORAGE
- `health_check_url` (String, Required): Health check endpoint
- `scaling_policy` (JSON, Required): HPA configuration and scaling rules
- `resource_limits` (JSON, Required): CPU, memory, and GPU limits
- `dependencies` (JSON Array, Required): List of dependent services
- `version` (String, Required): Current deployed version
- `status` (Enum, Required): HEALTHY, DEGRADED, UNHEALTHY
- `last_health_check` (Timestamp, Required): Last health check timestamp

**Validation Rules**:
- `health_check_url` must be valid HTTP/HTTPS URL
- `scaling_policy` must contain min_replicas, max_replicas, and target_cpu fields
- `resource_limits` must specify CPU and memory limits

**Relationships**:
- Many-to-many with Infrastructure Components (services run on infrastructure)
- One-to-many with Monitoring Metrics (services generate metrics)

### Infrastructure Components
**Purpose**: Representing physical and virtual infrastructure including GPUs, databases, and load balancers

**Fields**:
- `component_id` (UUID, Primary Key): Unique component identifier
- `component_type` (Enum, Required): GPU, CPU, DATABASE, LOAD_BALANCER, CACHE
- `provider` (String, Required): Cloud provider or hardware vendor
- `instance_type` (String, Required): Specific instance/model (A100, H100, m5.large, etc.)
- `region` (String, Required): Deployment region
- `capacity` (JSON, Required): Resource specifications (CPU cores, RAM, GPU memory, etc.)
- `utilization` (JSON, Required): Current usage metrics
- `status` (Enum, Required): AVAILABLE, IN_USE, MAINTENANCE, FAILED
- `cost_per_hour` (Decimal, Required): Hourly operational cost
- `created_at` (Timestamp, Required): Component provisioning timestamp

**Validation Rules**:
- `capacity` must include relevant metrics for component_type
- `utilization` must be percentage values (0-100)
- `cost_per_hour` must be non-negative

**Relationships**:
- Many-to-many with Services (infrastructure hosts services)
- One-to-many with Monitoring Metrics (components generate metrics)

### Security Policies
**Purpose**: Representing security configurations including encryption keys and access controls

**Fields**:
- `policy_id` (UUID, Primary Key): Unique policy identifier
- `policy_type` (Enum, Required): ENCRYPTION, ACCESS_CONTROL, NETWORK, COMPLIANCE
- `scope` (Enum, Required): GLOBAL, SERVICE, USER, DATA
- `rules` (JSON, Required): Policy rules and configurations
- `encryption_key_id` (UUID, Optional): Reference to encryption key
- `compliance_framework` (String, Optional): SOC2, GDPR, HIPAA, etc.
- `status` (Enum, Required): ACTIVE, INACTIVE, EXPIRED
- `created_at` (Timestamp, Required): Policy creation timestamp
- `expires_at` (Timestamp, Optional): Policy expiration date

**Validation Rules**:
- `rules` must be valid JSON schema for policy_type
- If `policy_type` is ENCRYPTION, `encryption_key_id` is required
- `expires_at` must be future date if specified

**Relationships**:
- One-to-many with Users (policies apply to users)
- One-to-many with Services (policies apply to services)

### API Endpoints
**Purpose**: Representing API endpoints with authentication and rate limiting configurations

**Fields**:
- `endpoint_id` (UUID, Primary Key): Unique endpoint identifier
- `path` (String, Required): API endpoint path (/v1/chat/completions)
- `method` (Enum, Required): GET, POST, PUT, DELETE, PATCH
- `service_id` (UUID, Required): Reference to implementing service
- `model_id` (UUID, Optional): Associated AI model for inference endpoints
- `authentication_required` (Boolean, Required): Whether endpoint requires auth
- `rate_limit` (JSON, Required): Rate limiting configuration
- `request_schema` (JSON, Required): OpenAPI request schema
- `response_schema` (JSON, Required): OpenAPI response schema
- `status` (Enum, Required): ACTIVE, DEPRECATED, MAINTENANCE

**Validation Rules**:
- `path` must be valid URL path starting with /
- `rate_limit` must contain requests_per_minute and burst_limit
- `request_schema` and `response_schema` must be valid JSON Schema

**Relationships**:
- Many-to-one with Services (endpoints belong to services)
- Many-to-one with AI Models (endpoints serve specific models)
- One-to-many with Users (users access endpoints)

### Deployment Environments
**Purpose**: Representing deployment environments with specific configurations

**Fields**:
- `environment_id` (UUID, Primary Key): Unique environment identifier
- `name` (String, Required, Unique): Environment name (development, staging, production)
- `region` (String, Required): Primary deployment region
- `replicas` (Integer, Required): Number of service replicas
- `resource_allocation` (JSON, Required): Environment-specific resource limits
- `feature_flags` (JSON, Required): Enabled/disabled features
- `config_overrides` (JSON, Required): Environment-specific configurations
- `status` (Enum, Required): ACTIVE, MAINTENANCE, DEPRECATED

**Validation Rules**:
- `replicas` must be positive integer
- `resource_allocation` must specify CPU and memory per replica
- `name` must be one of: development, staging, production, canary

**Relationships**:
- One-to-many with Services (environments contain service deployments)

### Monitoring Metrics
**Purpose**: Representing monitoring metrics and alerting configurations

**Fields**:
- `metric_id` (UUID, Primary Key): Unique metric identifier
- `metric_name` (String, Required): Metric name (cpu_usage, latency_p95, etc.)
- `metric_type` (Enum, Required): GAUGE, COUNTER, HISTOGRAM, SUMMARY
- `component_id` (UUID, Required): Source component
- `service_id` (UUID, Optional): Associated service
- `thresholds` (JSON, Required): Alert thresholds and conditions
- `labels` (JSON, Required): Metric labels for filtering
- `retention_days` (Integer, Required): Metric retention period
- `status` (Enum, Required): ACTIVE, INACTIVE

**Validation Rules**:
- `thresholds` must contain warning and critical thresholds
- `retention_days` must be between 7 and 365
- `labels` must include required labels (service, component, region)

**Relationships**:
- Many-to-one with Infrastructure Components (metrics from components)
- Many-to-one with Services (metrics from services)

### Integration Points
**Purpose**: Representing external integrations including SDKs and third-party services

**Fields**:
- `integration_id` (UUID, Primary Key): Unique integration identifier
- `integration_type` (Enum, Required): SDK, WEBHOOK, API, DATABASE, STORAGE
- `name` (String, Required): Integration name (python-sdk, stripe-billing, etc.)
- `language` (String, Optional): Programming language for SDKs
- `version` (String, Required): Current version
- `configuration` (JSON, Required): Integration-specific configuration
- `status` (Enum, Required): ACTIVE, BETA, DEPRECATED
- `documentation_url` (String, Optional): Documentation link

**Validation Rules**:
- If `integration_type` is SDK, `language` is required
- `version` must follow semantic versioning
- `configuration` must be valid for integration_type

**Relationships**:
- One-to-many with Users (users can configure integrations)

## State Transitions

### AI Model Lifecycle
1. **REGISTERED** → **VALIDATING** (automated compatibility checks)
2. **VALIDATING** → **ACTIVE** (passes all validation)
3. **ACTIVE** → **DEPRECATED** (newer version available)
4. **DEPRECATED** → **ARCHIVED** (no longer supported)

### User Account States
1. **PENDING** → **ACTIVE** (email verification completed)
2. **ACTIVE** → **SUSPENDED** (policy violation or payment failure)
3. **SUSPENDED** → **ACTIVE** (issue resolved)
4. **ACTIVE** → **DELETED** (user request or inactivity)

### Service Deployment States
1. **DEPLOYING** → **HEALTHY** (all health checks pass)
2. **HEALTHY** → **DEGRADED** (some health checks fail)
3. **DEGRADED** → **HEALTHY** (issues resolved)
4. **DEGRADED** → **FAILED** (critical failures)

## Data Volume and Scale Assumptions

- **AI Models**: 300+ active models, growing to 500+ within 12 months
- **Users**: 100,000+ active users, with 1M+ registered accounts
- **API Requests**: 10M+ requests per day, peaking at 100K requests per minute
- **Data Storage**: 100TB+ of model data, logs, and user content
- **Monitoring Data**: 1B+ metric data points per day
- **Audit Logs**: 10M+ security events per month