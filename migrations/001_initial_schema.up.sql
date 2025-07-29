-- Initial schema for Viva Rate Limiter
-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Create enum types
CREATE TYPE api_key_status AS ENUM ('active', 'suspended', 'revoked');
CREATE TYPE api_key_tier AS ENUM ('free', 'pro', 'enterprise');
CREATE TYPE alert_type AS ENUM ('usage_threshold', 'rate_limit_exceeded', 'billing_overage', 'security_alert', 'system_health');
CREATE TYPE alert_severity AS ENUM ('low', 'medium', 'high', 'critical');
CREATE TYPE alert_status AS ENUM ('active', 'resolved', 'suppressed');
CREATE TYPE billing_period_status AS ENUM ('active', 'completed', 'processing');

-- API Keys table
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(500),
    tier api_key_tier NOT NULL DEFAULT 'free',
    rate_limit INTEGER NOT NULL DEFAULT 1000,
    rate_window INTEGER NOT NULL DEFAULT 3600,
    status api_key_status NOT NULL DEFAULT 'active',
    metadata JSONB,
    tags TEXT[],
    user_id UUID,
    team_id UUID,
    last_used_at TIMESTAMPTZ,
    total_usage BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Usage logs table (partitioned by month)
CREATE TABLE usage_logs (
    id BIGSERIAL PRIMARY KEY,
    api_key_id UUID NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER NOT NULL,
    response_time INTEGER,
    user_agent VARCHAR(500),
    ip_address VARCHAR(45),
    country VARCHAR(2),
    region VARCHAR(100),
    request_size BIGINT,
    response_size BIGINT,
    metadata JSONB,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (timestamp);

-- Create initial partitions for usage logs (current month and next month)
CREATE TABLE usage_logs_y2024m01 PARTITION OF usage_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE usage_logs_y2024m02 PARTITION OF usage_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE usage_logs_y2024m03 PARTITION OF usage_logs
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');
CREATE TABLE usage_logs_y2024m04 PARTITION OF usage_logs
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');
CREATE TABLE usage_logs_y2024m05 PARTITION OF usage_logs
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');
CREATE TABLE usage_logs_y2024m06 PARTITION OF usage_logs
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');
CREATE TABLE usage_logs_y2024m07 PARTITION OF usage_logs
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
CREATE TABLE usage_logs_y2024m08 PARTITION OF usage_logs
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
CREATE TABLE usage_logs_y2024m09 PARTITION OF usage_logs
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');
CREATE TABLE usage_logs_y2024m10 PARTITION OF usage_logs
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');
CREATE TABLE usage_logs_y2024m11 PARTITION OF usage_logs
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
CREATE TABLE usage_logs_y2024m12 PARTITION OF usage_logs
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- Alerts table
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID,
    type alert_type NOT NULL,
    severity alert_severity NOT NULL,
    status alert_status NOT NULL DEFAULT 'active',
    title VARCHAR(255) NOT NULL,
    message VARCHAR(1000) NOT NULL,
    description VARCHAR(2000),
    threshold DECIMAL(15,4),
    current_value DECIMAL(15,4),
    unit VARCHAR(50),
    metadata JSONB,
    tags TEXT[],
    triggered_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Rate limit violations table (partitioned by month)
CREATE TABLE rate_limit_violations (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    api_key_id UUID NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    client_ip VARCHAR(45),
    user_agent VARCHAR(500),
    limit_value INTEGER NOT NULL,
    window_seconds INTEGER NOT NULL,
    current_count INTEGER NOT NULL,
    tier_type VARCHAR(20),
    is_repeated BOOLEAN DEFAULT FALSE,
    violation_count INTEGER DEFAULT 1,
    country VARCHAR(2),
    region VARCHAR(100),
    processed_at TIMESTAMPTZ,
    metadata JSONB,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (timestamp);

-- Create initial partitions for rate limit violations
CREATE TABLE rate_limit_violations_y2024m01 PARTITION OF rate_limit_violations
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE rate_limit_violations_y2024m02 PARTITION OF rate_limit_violations
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE rate_limit_violations_y2024m03 PARTITION OF rate_limit_violations
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');
CREATE TABLE rate_limit_violations_y2024m04 PARTITION OF rate_limit_violations
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');
CREATE TABLE rate_limit_violations_y2024m05 PARTITION OF rate_limit_violations
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');
CREATE TABLE rate_limit_violations_y2024m06 PARTITION OF rate_limit_violations
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');
CREATE TABLE rate_limit_violations_y2024m07 PARTITION OF rate_limit_violations
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
CREATE TABLE rate_limit_violations_y2024m08 PARTITION OF rate_limit_violations
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
CREATE TABLE rate_limit_violations_y2024m09 PARTITION OF rate_limit_violations
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');
CREATE TABLE rate_limit_violations_y2024m10 PARTITION OF rate_limit_violations
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');
CREATE TABLE rate_limit_violations_y2024m11 PARTITION OF rate_limit_violations
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
CREATE TABLE rate_limit_violations_y2024m12 PARTITION OF rate_limit_violations
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- Billing records table
CREATE TABLE billing_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    status billing_period_status NOT NULL DEFAULT 'active',
    total_requests BIGINT NOT NULL DEFAULT 0,
    success_requests BIGINT NOT NULL DEFAULT 0,
    error_requests BIGINT NOT NULL DEFAULT 0,
    overage_requests BIGINT NOT NULL DEFAULT 0,
    rate_limit_hits BIGINT NOT NULL DEFAULT 0,
    total_bandwidth BIGINT NOT NULL DEFAULT 0,
    base_amount DECIMAL(10,4) DEFAULT 0,
    overage_amount DECIMAL(10,4) DEFAULT 0,
    total_amount DECIMAL(10,4) DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    tier_at_start VARCHAR(20),
    tier_at_end VARCHAR(20),
    metadata JSONB,
    calculated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_api_keys_key_hash ON api_keys (key_hash);
CREATE INDEX idx_api_keys_status ON api_keys (status) WHERE status = 'active';
CREATE INDEX idx_api_keys_tier ON api_keys (tier);
CREATE INDEX idx_api_keys_user_id ON api_keys (user_id);
CREATE INDEX idx_api_keys_team_id ON api_keys (team_id);
CREATE INDEX idx_api_keys_created_at ON api_keys (created_at);

CREATE INDEX idx_usage_logs_api_key_id ON usage_logs (api_key_id);
CREATE INDEX idx_usage_logs_timestamp ON usage_logs (timestamp DESC);
CREATE INDEX idx_usage_logs_endpoint ON usage_logs (endpoint);
CREATE INDEX idx_usage_logs_status_code ON usage_logs (status_code);
CREATE INDEX idx_usage_logs_ip_address ON usage_logs (ip_address);

CREATE INDEX idx_alerts_api_key_id ON alerts (api_key_id);
CREATE INDEX idx_alerts_type ON alerts (type);
CREATE INDEX idx_alerts_severity ON alerts (severity);
CREATE INDEX idx_alerts_status ON alerts (status);
CREATE INDEX idx_alerts_created_at ON alerts (created_at DESC);

CREATE INDEX idx_rate_violations_api_key_id ON rate_limit_violations (api_key_id);
CREATE INDEX idx_rate_violations_timestamp ON rate_limit_violations (timestamp DESC);
CREATE INDEX idx_rate_violations_endpoint ON rate_limit_violations (endpoint);
CREATE INDEX idx_rate_violations_client_ip ON rate_limit_violations (client_ip);
CREATE INDEX idx_rate_violations_processed ON rate_limit_violations (processed_at) WHERE processed_at IS NULL;
CREATE INDEX idx_rate_violations_is_repeated ON rate_limit_violations (is_repeated);

CREATE INDEX idx_billing_records_api_key_id ON billing_records (api_key_id);
CREATE INDEX idx_billing_records_period ON billing_records (period_start, period_end);
CREATE INDEX idx_billing_records_status ON billing_records (status);
CREATE INDEX idx_billing_records_created_at ON billing_records (created_at DESC);

-- Create foreign key constraints
ALTER TABLE usage_logs ADD CONSTRAINT fk_usage_logs_api_key_id 
    FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE;
ALTER TABLE alerts ADD CONSTRAINT fk_alerts_api_key_id 
    FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE;
ALTER TABLE rate_limit_violations ADD CONSTRAINT fk_rate_violations_api_key_id 
    FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE;
ALTER TABLE billing_records ADD CONSTRAINT fk_billing_records_api_key_id 
    FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE;

-- Create trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_alerts_updated_at BEFORE UPDATE ON alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_billing_records_updated_at BEFORE UPDATE ON billing_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();