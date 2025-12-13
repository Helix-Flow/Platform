-- HelixFlow Data Models Implementation
-- SQL DDL for all 9 entities from data-model.md

-- AI Models table
CREATE TABLE IF NOT EXISTS ai_models (
    model_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    model_type VARCHAR(20) NOT NULL CHECK (model_type IN ('TEXT', 'IMAGE', 'AUDIO', 'MULTIMODAL', 'CODE')),
    context_window INTEGER NOT NULL CHECK (context_window > 0),
    pricing_per_token_input DECIMAL(10,6) NOT NULL CHECK (pricing_per_token_input >= 0),
    pricing_per_token_output DECIMAL(10,6) NOT NULL CHECK (pricing_per_token_output >= 0),
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    aliases JSONB DEFAULT '[]'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DEPRECATED', 'BETA')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(provider, model_name)
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    subscription_tier VARCHAR(20) NOT NULL CHECK (subscription_tier IN ('FREE', 'PRO', 'ENTERPRISE', 'RESEARCH')),
    api_key_hash VARCHAR(255) NOT NULL,
    usage_limits JSONB NOT NULL DEFAULT '{"token_limit": 1000, "rate_limit": 60}'::jsonb,
    billing_info JSONB,
    organization_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_active TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED', 'DELETED'))
);

-- Services table
CREATE TABLE IF NOT EXISTS services (
    service_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name VARCHAR(100) NOT NULL UNIQUE,
    service_type VARCHAR(20) NOT NULL CHECK (service_type IN ('GATEWAY', 'INFERENCE', 'AUTH', 'MONITORING', 'STORAGE')),
    health_check_url VARCHAR(500) NOT NULL,
    scaling_policy JSONB NOT NULL DEFAULT '{"min_replicas": 1, "max_replicas": 3}'::jsonb,
    resource_limits JSONB NOT NULL DEFAULT '{"cpu": "500m", "memory": "512Mi"}'::jsonb,
    dependencies JSONB DEFAULT '[]'::jsonb,
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    status VARCHAR(20) NOT NULL DEFAULT 'HEALTHY' CHECK (status IN ('HEALTHY', 'DEGRADED', 'UNHEALTHY')),
    last_health_check TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Infrastructure Components table
CREATE TABLE IF NOT EXISTS infrastructure_components (
    component_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    component_type VARCHAR(20) NOT NULL CHECK (component_type IN ('GPU', 'CPU', 'DATABASE', 'LOAD_BALANCER', 'CACHE')),
    provider VARCHAR(50) NOT NULL,
    instance_type VARCHAR(100) NOT NULL,
    region VARCHAR(50) NOT NULL,
    capacity JSONB NOT NULL,
    utilization JSONB NOT NULL DEFAULT '{"percentage": 0}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'AVAILABLE' CHECK (status IN ('AVAILABLE', 'IN_USE', 'MAINTENANCE', 'FAILED')),
    cost_per_hour DECIMAL(10,4) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Monitoring Metrics table
CREATE TABLE IF NOT EXISTS monitoring_metrics (
    metric_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_name VARCHAR(100) NOT NULL,
    metric_type VARCHAR(20) NOT NULL CHECK (metric_type IN ('GAUGE', 'COUNTER', 'HISTOGRAM', 'SUMMARY')),
    component_id UUID REFERENCES infrastructure_components(component_id),
    service_id UUID REFERENCES services(service_id),
    thresholds JSONB NOT NULL DEFAULT '{"warning": 80, "critical": 90}'::jsonb,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    retention_days INTEGER NOT NULL DEFAULT 30 CHECK (retention_days BETWEEN 7 AND 365),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Security Policies table
CREATE TABLE IF NOT EXISTS security_policies (
    policy_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_type VARCHAR(20) NOT NULL CHECK (policy_type IN ('ENCRYPTION', 'ACCESS_CONTROL', 'NETWORK', 'COMPLIANCE')),
    scope VARCHAR(20) NOT NULL CHECK (scope IN ('GLOBAL', 'SERVICE', 'USER', 'DATA')),
    rules JSONB NOT NULL,
    encryption_key_id UUID,
    compliance_framework VARCHAR(20) CHECK (compliance_framework IN ('SOC2', 'GDPR', 'CCPA', 'PIPL', 'LGPD', 'PDPB')),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'EXPIRED')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Deployment Environments table
CREATE TABLE IF NOT EXISTS deployment_environments (
    environment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(20) NOT NULL UNIQUE CHECK (name IN ('development', 'staging', 'production')),
    region VARCHAR(50) NOT NULL,
    replicas INTEGER NOT NULL DEFAULT 1 CHECK (replicas > 0),
    resource_allocation JSONB NOT NULL DEFAULT '{"cpu": "1000m", "memory": "1Gi"}'::jsonb,
    feature_flags JSONB NOT NULL DEFAULT '{}'::jsonb,
    config_overrides JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'MAINTENANCE', 'DEPRECATED')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Integration Points table
CREATE TABLE IF NOT EXISTS integration_points (
    integration_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    integration_type VARCHAR(20) NOT NULL CHECK (integration_type IN ('SDK', 'WEBHOOK', 'API', 'DATABASE', 'STORAGE')),
    name VARCHAR(100) NOT NULL,
    language VARCHAR(20),
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    configuration JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'BETA', 'DEPRECATED')),
    documentation_url VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- API Endpoints table (must be created after services and ai_models)
CREATE TABLE IF NOT EXISTS api_endpoints (
    endpoint_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    path VARCHAR(500) NOT NULL,
    method VARCHAR(10) NOT NULL CHECK (method IN ('GET', 'POST', 'PUT', 'DELETE', 'PATCH')),
    service_id UUID NOT NULL REFERENCES services(service_id),
    model_id UUID REFERENCES ai_models(model_id),
    authentication_required BOOLEAN NOT NULL DEFAULT true,
    rate_limit JSONB NOT NULL DEFAULT '{"requests_per_minute": 60}'::jsonb,
    request_schema JSONB,
    response_schema JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DEPRECATED', 'MAINTENANCE')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users USING gin (pgp_sym_decrypt(email, current_setting('app.encryption_key')));
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_subscription_tier ON users(subscription_tier);
CREATE INDEX IF NOT EXISTS idx_ai_models_provider_name ON ai_models(provider, model_name);
CREATE INDEX IF NOT EXISTS idx_ai_models_status ON ai_models(status);
CREATE INDEX IF NOT EXISTS idx_api_endpoints_path_method ON api_endpoints(path, method);
CREATE INDEX IF NOT EXISTS idx_api_endpoints_status ON api_endpoints(status);
CREATE INDEX IF NOT EXISTS idx_services_status ON services(status);
CREATE INDEX IF NOT EXISTS idx_infrastructure_components_status ON infrastructure_components(status);
CREATE INDEX IF NOT EXISTS idx_monitoring_metrics_name ON monitoring_metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_security_policies_type ON security_policies(policy_type);
CREATE INDEX IF NOT EXISTS idx_deployment_environments_name ON deployment_environments(name);

-- Insert sample data for testing
INSERT INTO ai_models (provider, model_name, model_type, context_window, pricing_per_token_input, pricing_per_token_output, capabilities)
VALUES
    ('openai', 'gpt-4', 'TEXT', 8192, 0.0015, 0.002, '["text-generation", "code", "reasoning"]'::jsonb),
    ('anthropic', 'claude-3-sonnet', 'TEXT', 200000, 0.0015, 0.002, '["text-generation", "analysis", "creative"]'::jsonb),
    ('google', 'gemini-pro', 'MULTIMODAL', 32768, 0.0010, 0.0015, '["text", "image", "multimodal"]'::jsonb)
ON CONFLICT (provider, model_name) DO NOTHING;

INSERT INTO services (service_name, service_type, health_check_url)
VALUES
    ('api-gateway', 'GATEWAY', 'http://localhost:8080/health'),
    ('inference-pool', 'INFERENCE', 'http://localhost:8080/health'),
    ('auth-service', 'AUTH', 'http://localhost:8080/health'),
    ('monitoring', 'MONITORING', 'http://localhost:8080/health')
ON CONFLICT (service_name) DO NOTHING;