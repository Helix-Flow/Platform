#!/bin/bash

set -e

echo "🗄️ Setting up PostgreSQL for HelixFlow"
echo "======================================"

# PostgreSQL connection details
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-helixflow}"
DB_USER="${DB_USER:-helixflow}"
DB_PASSWORD="${DB_PASSWORD:-helixflow123}"

# Check if PostgreSQL is running
if ! pg_isready -h $DB_HOST -p $DB_PORT -U postgres 2>/dev/null; then
    echo "❌ PostgreSQL is not running on $DB_HOST:$DB_PORT"
    echo "Please start PostgreSQL or run it via Docker:"
    echo "docker run -d --name helixflow-postgres \\"
    echo "  -e POSTGRES_PASSWORD=postgres123 \\"
    echo "  -e POSTGRES_USER=postgres \\"
    echo "  -e POSTGRES_DB=helixflow \\"
    echo "  -p 5432:5432 \\"
    echo "  postgres:15"
    exit 1
fi

echo "✅ PostgreSQL is running on $DB_HOST:$DB_PORT"

# Create database and user
echo "Creating database and user..."
psql -h $DB_HOST -p $DB_PORT -U postgres << EOF
-- Create database
CREATE DATABASE $DB_NAME;

-- Create user
CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;

-- Create schema extensions
\c $DB_NAME
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
EOF

echo "✅ Database and user created successfully"

# Create tables
echo "Creating database tables..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << 'EOF'
-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    organization VARCHAR(255),
    role VARCHAR(50) DEFAULT 'user',
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP
);

-- API Keys table
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255),
    key_hash VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(16) NOT NULL,
    permissions TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    usage_count INTEGER DEFAULT 0,
    active BOOLEAN DEFAULT true
);

-- Inference Logs table
CREATE TABLE inference_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    model_id VARCHAR(255),
    request_size INTEGER,
    response_size INTEGER,
    latency_ms INTEGER,
    status_code INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ip_address INET,
    user_agent TEXT
);

-- API Usage Logs table
CREATE TABLE api_usage_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    api_key_id UUID REFERENCES api_keys(id),
    method VARCHAR(10),
    path VARCHAR(512),
    status_code INTEGER,
    latency_ms INTEGER,
    request_size INTEGER,
    response_size INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ip_address INET,
    user_agent TEXT
);

-- System Metrics table
CREATE TABLE system_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    service_name VARCHAR(255),
    metric_type VARCHAR(100),
    metric_value DOUBLE PRECISION,
    unit VARCHAR(50),
    labels JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Alerts table
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255),
    description TEXT,
    severity VARCHAR(50),
    condition TEXT,
    threshold DOUBLE PRECISION,
    current_value DOUBLE PRECISION,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    notification_channels TEXT[],
    labels JSONB
);

-- Create indexes for performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX idx_api_keys_active ON api_keys(active);
CREATE INDEX idx_inference_logs_user_id ON inference_logs(user_id);
CREATE INDEX idx_inference_logs_created_at ON inference_logs(created_at);
CREATE INDEX idx_api_usage_logs_user_id ON api_usage_logs(user_id);
CREATE INDEX idx_api_usage_logs_created_at ON api_usage_logs(created_at);
CREATE INDEX idx_system_metrics_service_name ON system_metrics(service_name);
CREATE INDEX idx_system_metrics_created_at ON system_metrics(created_at);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_created_at ON alerts(created_at);
EOF

echo "✅ Database tables created successfully"

# Create sample data
echo "Creating sample data..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << 'EOF'
-- Insert sample users
INSERT INTO users (id, username, email, password_hash, first_name, last_name, organization, role) VALUES
(uuid_generate_v4(), 'admin', 'admin@helixflow.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'Admin', 'User', 'HelixFlow', 'admin'),
(uuid_generate_v4(), 'demo', 'demo@helixflow.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'Demo', 'User', 'Demo Corp', 'user');

-- Insert sample API keys for demo user
INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, permissions, active) 
SELECT 
    uuid_generate_v4(),
    id,
    'Demo API Key',
    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
    'hf_demo',
    ARRAY['read', 'write'],
    true
FROM users WHERE username = 'demo';

-- Insert sample system metrics
INSERT INTO system_metrics (id, service_name, metric_type, metric_value, unit, labels) VALUES
(uuid_generate_v4(), 'api-gateway', 'cpu_usage', 45.2, 'percent', '{"instance": "api-gateway-1"}'),
(uuid_generate_v4(), 'api-gateway', 'memory_usage', 67.8, 'percent', '{"instance": "api-gateway-1"}'),
(uuid_generate_v4(), 'inference-pool', 'gpu_utilization', 23.1, 'percent', '{"gpu_id": "gpu-0"}'),
(uuid_generate_v4(), 'inference-pool', 'memory_usage', 78.9, 'percent', '{"gpu_id": "gpu-0"}');
EOF

echo "✅ Sample data created successfully"

# Create database functions
echo "Creating database functions..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << 'EOF'
-- Function to validate API key
CREATE OR REPLACE FUNCTION validate_api_key(p_key_prefix VARCHAR, p_permissions TEXT[])
RETURNS TABLE(user_id UUID, permissions TEXT[]) AS $$
BEGIN
    RETURN QUERY
    SELECT ak.user_id, ak.permissions
    FROM api_keys ak
    WHERE ak.key_prefix = p_key_prefix 
      AND ak.active = true
      AND (ak.expires_at IS NULL OR ak.expires_at > NOW())
      AND ak.permissions @> p_permissions;
END;
$$ LANGUAGE plpgsql;

-- Function to update API key usage
CREATE OR REPLACE FUNCTION update_api_key_usage(p_key_prefix VARCHAR)
RETURNS VOID AS $$
BEGIN
    UPDATE api_keys 
    SET last_used_at = NOW(), 
        usage_count = usage_count + 1
    WHERE key_prefix = p_key_prefix;
END;
$$ LANGUAGE plpgsql;

-- Function to record API usage
CREATE OR REPLACE FUNCTION record_api_usage(
    p_user_id UUID,
    p_api_key_id UUID,
    p_method VARCHAR,
    p_path VARCHAR,
    p_status_code INTEGER,
    p_latency_ms INTEGER,
    p_request_size INTEGER,
    p_response_size INTEGER,
    p_ip_address INET,
    p_user_agent TEXT
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO api_usage_logs (
        user_id, api_key_id, method, path, status_code, 
        latency_ms, request_size, response_size, ip_address, user_agent
    ) VALUES (
        p_user_id, p_api_key_id, p_method, p_path, p_status_code,
        p_latency_ms, p_request_size, p_response_size, p_ip_address, p_user_agent
    );
END;
$$ LANGUAGE plpgsql;
EOF

echo "✅ Database functions created successfully"

echo ""
echo "🎉 PostgreSQL setup completed successfully!"
echo ""
echo "Database connection details:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo "  User: $DB_USER"
echo "  Password: $DB_PASSWORD"
echo ""
echo "Sample users created:"
echo "  admin@helixflow.com (admin role)"
echo "  demo@helixflow.com (user role)"
echo ""
echo "API Key for demo user: hf_demo... (check database for full key)"