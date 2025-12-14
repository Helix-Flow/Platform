#!/bin/bash

set -e

echo "🗄️ Setting up SQLite Database for HelixFlow Development"
echo "======================================================"

# Database configuration
DB_FILE="${DB_FILE:-./data/helixflow.db}"
DB_DIR=$(dirname "$DB_FILE")

# Create data directory if it doesn't exist
echo "Creating data directory..."
mkdir -p "$DB_DIR"

# Check if SQLite is available
if ! command -v sqlite3 &> /dev/null; then
    echo "❌ SQLite3 is not installed"
    echo "Please install SQLite3:"
    echo "  Ubuntu/Debian: sudo apt-get install sqlite3"
    echo "  macOS: brew install sqlite3"
    echo "  CentOS/RHEL: sudo yum install sqlite"
    exit 1
fi

echo "✅ SQLite3 is available"

# Create SQLite database with schema
echo "Creating SQLite database with schema..."
sqlite3 "$DB_FILE" << 'EOF'
-- Enable foreign keys
PRAGMA foreign_keys = ON;

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    first_name TEXT,
    last_name TEXT,
    organization TEXT,
    role TEXT DEFAULT 'user',
    active INTEGER DEFAULT 1,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
    last_login_at TEXT
);

-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT,
    key_hash TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    permissions TEXT, -- JSON array stored as text
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    expires_at TEXT,
    last_used_at TEXT,
    usage_count INTEGER DEFAULT 0,
    active INTEGER DEFAULT 1
);

-- Inference Logs table
CREATE TABLE IF NOT EXISTS inference_logs (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    model_id TEXT,
    request_size INTEGER,
    response_size INTEGER,
    latency_ms INTEGER,
    status_code INTEGER,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    ip_address TEXT,
    user_agent TEXT
);

-- API Usage Logs table
CREATE TABLE IF NOT EXISTS api_usage_logs (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    api_key_id TEXT REFERENCES api_keys(id),
    method TEXT,
    path TEXT,
    status_code INTEGER,
    latency_ms INTEGER,
    request_size INTEGER,
    response_size INTEGER,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    ip_address TEXT,
    user_agent TEXT
);

-- System Metrics table
CREATE TABLE IF NOT EXISTS system_metrics (
    id TEXT PRIMARY KEY,
    service_name TEXT,
    metric_type TEXT,
    metric_value REAL,
    unit TEXT,
    labels TEXT, -- JSON object stored as text
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

-- Alerts table
CREATE TABLE IF NOT EXISTS alerts (
    id TEXT PRIMARY KEY,
    name TEXT,
    description TEXT,
    severity TEXT,
    condition TEXT,
    threshold REAL,
    current_value REAL,
    status TEXT DEFAULT 'active',
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    resolved_at TEXT,
    notification_channels TEXT, -- JSON array stored as text
    labels TEXT -- JSON object stored as text
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(active);
CREATE INDEX IF NOT EXISTS idx_inference_logs_user_id ON inference_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_inference_logs_created_at ON inference_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_api_usage_logs_user_id ON api_usage_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_api_usage_logs_created_at ON api_usage_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_system_metrics_service_name ON system_metrics(service_name);
CREATE INDEX IF NOT EXISTS idx_system_metrics_created_at ON system_metrics(created_at);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at);
EOF

echo "✅ SQLite database schema created successfully"

# Create sample data
echo "Creating sample data..."
sqlite3 "$DB_FILE" << 'EOF'
-- Insert sample users
INSERT INTO users (id, username, email, password_hash, first_name, last_name, organization, role) VALUES
('550e8400-e29b-41d4-a716-446655440001', 'admin', 'admin@helixflow.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'Admin', 'User', 'HelixFlow', 'admin'),
('550e8400-e29b-41d4-a716-446655440002', 'demo', 'demo@helixflow.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'Demo', 'User', 'Demo Corp', 'user'),
('550e8400-e29b-41d4-a716-446655440003', 'testuser', 'test@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'Test', 'User', 'Test Organization', 'user');

-- Insert sample API keys for demo user
INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, permissions, active) 
VALUES 
('550e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440002', 'Demo API Key', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'hf_demo', '["read", "write"]', 1),
('550e8400-e29b-41d4-a716-446655440005', '550e8400-e29b-41d4-a716-446655440003', 'Test API Key', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'hf_test', '["read"]', 1);

-- Insert sample system metrics
INSERT INTO system_metrics (id, service_name, metric_type, metric_value, unit, labels) VALUES
('550e8400-e29b-41d4-a716-446655440006', 'api-gateway', 'cpu_usage', 45.2, 'percent', '{"instance": "api-gateway-1"}'),
('550e8400-e29b-41d4-a716-446655440007', 'api-gateway', 'memory_usage', 67.8, 'percent', '{"instance": "api-gateway-1"}'),
('550e8400-e29b-41d4-a716-446655440008', 'inference-pool', 'gpu_utilization', 23.1, 'percent', '{"gpu_id": "gpu-0"}'),
('550e8400-e29b-41d4-a716-446655440009', 'inference-pool', 'memory_usage', 78.9, 'percent', '{"gpu_id": "gpu-0"}');

-- Insert sample alerts
INSERT INTO alerts (id, name, description, severity, condition, threshold, current_value, status, notification_channels, labels) VALUES
('550e8400-e29b-41d4-a716-446655440010', 'High CPU Usage', 'CPU usage is above 80%', 'high', 'cpu_usage > 80', 80.0, 85.2, 'active', '["email", "slack"]', '{"service": "api-gateway"}'),
('550e8400-e29b-41d4-a716-446655440011', 'Low Disk Space', 'Available disk space is below 10%', 'critical', 'disk_free < 10', 10.0, 8.5, 'active', '["email", "pagerduty"]', '{"service": "api-gateway"}');
EOF

echo "✅ Sample data created successfully"

# Verify database creation
echo ""
echo "Verifying database creation..."
echo "Users table: $(sqlite3 "$DB_FILE" "SELECT COUNT(*) FROM users;") records"
echo "API Keys table: $(sqlite3 "$DB_FILE" "SELECT COUNT(*) FROM api_keys;") records"
echo "System Metrics table: $(sqlite3 "$DB_FILE" "SELECT COUNT(*) FROM system_metrics;") records"
echo "Alerts table: $(sqlite3 "$DB_FILE" "SELECT COUNT(*) FROM alerts;") records"

echo ""
echo "🎉 SQLite database setup completed successfully!"
echo ""
echo "Database file: $DB_FILE"
echo "Connection string: sqlite://$DB_FILE"
echo ""
echo "Sample users created:"
echo "  admin@helixflow.com (admin role)"
echo "  demo@helixflow.com (user role)"
echo "  testuser@example.com (user role)"
echo ""
echo "API Keys created:"
echo "  Demo Key: hf_demo... (for demo@helixflow.com)"
echo "  Test Key: hf_test... (for testuser@example.com)"
echo ""
echo "To connect to the database:"
echo "  sqlite3 $DB_FILE"
echo "  .tables  # List all tables"
echo "  .schema users  # Show users table schema"
echo "  SELECT * FROM users;  # Query users"