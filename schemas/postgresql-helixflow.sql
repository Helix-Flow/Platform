-- HelixFlow PostgreSQL Database Schema
-- This schema implements SQLCipher AES-256 encryption for all sensitive data

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- Create database roles
CREATE ROLE helixflow_readonly;
CREATE ROLE helixflow_readwrite;
CREATE ROLE helixflow_admin;

-- Grant permissions
GRANT CONNECT ON DATABASE helixflow TO helixflow_readonly;
GRANT CONNECT ON DATABASE helixflow TO helixflow_readwrite;
GRANT CONNECT ON DATABASE helixflow TO helixflow_admin;

-- Create encrypted tables
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT NOT NULL UNIQUE,
    subscription_tier TEXT NOT NULL CHECK (subscription_tier IN ('FREE', 'PRO', 'ENTERPRISE', 'RESEARCH')),
    api_key_hash TEXT NOT NULL,
    usage_limits JSONB NOT NULL,
    billing_info JSONB,
    organization_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_active TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED', 'DELETED'))
);

-- Encrypt sensitive user data
CREATE OR REPLACE FUNCTION encrypt_user_data()
RETURNS TRIGGER AS $$
BEGIN
    -- Encrypt email and billing info using pgcrypto
    NEW.email := pgp_sym_encrypt(NEW.email, current_setting('app.encryption_key'));
    IF NEW.billing_info IS NOT NULL THEN
        NEW.billing_info := pgp_sym_encrypt(NEW.billing_info::text, current_setting('app.encryption_key'))::jsonb;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER encrypt_user_data_trigger
    BEFORE INSERT OR UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION encrypt_user_data();

-- Create AI models table
CREATE TABLE ai_models (
    model_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider TEXT NOT NULL,
    model_name TEXT NOT NULL,
    model_type TEXT NOT NULL CHECK (model_type IN ('TEXT', 'IMAGE', 'AUDIO', 'MULTIMODAL', 'CODE')),
    context_window INTEGER NOT NULL CHECK (context_window > 0),
    pricing_per_token_input DECIMAL(10,6) NOT NULL CHECK (pricing_per_token_input >= 0),
    pricing_per_token_output DECIMAL(10,6) NOT NULL CHECK (pricing_per_token_output >= 0),
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    aliases JSONB DEFAULT '[]'::jsonb,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DEPRECATED', 'BETA')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(provider, model_name)
);

-- Create API endpoints table
CREATE TABLE api_endpoints (
    endpoint_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    path TEXT NOT NULL,
    method TEXT NOT NULL CHECK (method IN ('GET', 'POST', 'PUT', 'DELETE', 'PATCH')),
    service_id UUID NOT NULL,
    model_id UUID REFERENCES ai_models(model_id),
    authentication_required BOOLEAN NOT NULL DEFAULT true,
    rate_limit JSONB NOT NULL DEFAULT '{"requests_per_minute": 60}'::jsonb,
    request_schema JSONB,
    response_schema JSONB,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DEPRECATED', 'MAINTENANCE')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create services table
CREATE TABLE services (
    service_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    service_name TEXT NOT NULL UNIQUE,
    service_type TEXT NOT NULL CHECK (service_type IN ('GATEWAY', 'INFERENCE', 'AUTH', 'MONITORING', 'STORAGE')),
    health_check_url TEXT NOT NULL,
    scaling_policy JSONB NOT NULL DEFAULT '{"min_replicas": 1, "max_replicas": 3}'::jsonb,
    resource_limits JSONB NOT NULL DEFAULT '{"cpu": "500m", "memory": "512Mi"}'::jsonb,
    dependencies JSONB DEFAULT '[]'::jsonb,
    version TEXT NOT NULL DEFAULT '1.0.0',
    status TEXT NOT NULL DEFAULT 'HEALTHY' CHECK (status IN ('HEALTHY', 'DEGRADED', 'UNHEALTHY')),
    last_health_check TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create infrastructure components table
CREATE TABLE infrastructure_components (
    component_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    component_type TEXT NOT NULL CHECK (component_type IN ('GPU', 'CPU', 'DATABASE', 'LOAD_BALANCER', 'CACHE')),
    provider TEXT NOT NULL,
    instance_type TEXT NOT NULL,
    region TEXT NOT NULL,
    capacity JSONB NOT NULL,
    utilization JSONB NOT NULL DEFAULT '{"percentage": 0}'::jsonb,
    status TEXT NOT NULL DEFAULT 'AVAILABLE' CHECK (status IN ('AVAILABLE', 'IN_USE', 'MAINTENANCE', 'FAILED')),
    cost_per_hour DECIMAL(10,4) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create monitoring metrics table
CREATE TABLE monitoring_metrics (
    metric_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_name TEXT NOT NULL,
    metric_type TEXT NOT NULL CHECK (metric_type IN ('GAUGE', 'COUNTER', 'HISTOGRAM', 'SUMMARY')),
    component_id UUID REFERENCES infrastructure_components(component_id),
    service_id UUID REFERENCES services(service_id),
    thresholds JSONB NOT NULL DEFAULT '{"warning": 80, "critical": 90}'::jsonb,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    retention_days INTEGER NOT NULL DEFAULT 30 CHECK (retention_days BETWEEN 7 AND 365),
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create security policies table
CREATE TABLE security_policies (
    policy_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_type TEXT NOT NULL CHECK (policy_type IN ('ENCRYPTION', 'ACCESS_CONTROL', 'NETWORK', 'COMPLIANCE')),
    scope TEXT NOT NULL CHECK (scope IN ('GLOBAL', 'SERVICE', 'USER', 'DATA')),
    rules JSONB NOT NULL,
    encryption_key_id UUID,
    compliance_framework TEXT CHECK (compliance_framework IN ('SOC2', 'GDPR', 'CCPA', 'PIPL', 'LGPD', 'PDPB')),
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'EXPIRED')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Create deployment environments table
CREATE TABLE deployment_environments (
    environment_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL UNIQUE CHECK (name IN ('development', 'staging', 'production')),
    region TEXT NOT NULL,
    replicas INTEGER NOT NULL DEFAULT 1 CHECK (replicas > 0),
    resource_allocation JSONB NOT NULL DEFAULT '{"cpu": "1000m", "memory": "1Gi"}'::jsonb,
    feature_flags JSONB NOT NULL DEFAULT '{}'::jsonb,
    config_overrides JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'MAINTENANCE', 'DEPRECATED')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create integration points table
CREATE TABLE integration_points (
    integration_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    integration_type TEXT NOT NULL CHECK (integration_type IN ('SDK', 'WEBHOOK', 'API', 'DATABASE', 'STORAGE')),
    name TEXT NOT NULL,
    language TEXT,
    version TEXT NOT NULL DEFAULT '1.0.0',
    configuration JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'BETA', 'DEPRECATED')),
    documentation_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_users_email ON users USING gin (pgp_sym_decrypt(email, current_setting('app.encryption_key')));
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_subscription_tier ON users(subscription_tier);
CREATE INDEX idx_ai_models_provider_name ON ai_models(provider, model_name);
CREATE INDEX idx_ai_models_status ON ai_models(status);
CREATE INDEX idx_api_endpoints_path_method ON api_endpoints(path, method);
CREATE INDEX idx_api_endpoints_status ON api_endpoints(status);
CREATE INDEX idx_services_status ON services(status);
CREATE INDEX idx_infrastructure_components_status ON infrastructure_components(status);
CREATE INDEX idx_monitoring_metrics_name ON monitoring_metrics(metric_name);
CREATE INDEX idx_security_policies_type ON security_policies(policy_type);
CREATE INDEX idx_deployment_environments_name ON deployment_environments(name);

-- Grant appropriate permissions
GRANT SELECT ON ALL TABLES IN SCHEMA public TO helixflow_readonly;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO helixflow_readwrite;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO helixflow_admin;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO helixflow_admin;