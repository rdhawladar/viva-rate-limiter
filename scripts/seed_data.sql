-- Seed Data for Viva Rate Limiter
-- This script creates sample API keys for testing

-- Insert sample API keys
INSERT INTO api_keys (
    id,
    key_hash,
    name,
    description,
    tier,
    rate_limit,
    rate_window,
    status,
    metadata,
    tags,
    total_usage,
    created_at,
    updated_at
) VALUES 
-- Test API Key 1 - Free Tier
(
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
    '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08', -- SHA256 of 'test-key-free-123'
    'Test Free Key',
    'Free tier API key for testing',
    'free',
    1000,
    3600,
    'active',
    '{"environment": "test", "created_by": "system"}',
    '{"testing", "free-tier"}',
    0,
    NOW(),
    NOW()
),
-- Test API Key 2 - Standard Tier
(
    'b2c3d4e5-f6g7-8901-bcde-f23456789012',
    '60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752e', -- SHA256 of 'test-key-standard-456'
    'Test Standard Key', 
    'Standard tier API key for testing',
    'standard',
    10000,
    3600,
    'active',
    '{"environment": "test", "created_by": "system"}',
    '{"testing", "standard-tier"}',
    0,
    NOW(),
    NOW()
),
-- Test API Key 3 - Pro Tier
(
    'c3d4e5f6-g7h8-9012-cdef-345678901234',
    'ef2d127de37b942baad06145e54b0c619a1f22327b2ebbcfbec78f5564afe39d', -- SHA256 of 'test-key-pro-789'
    'Test Pro Key',
    'Pro tier API key for testing',
    'pro', 
    100000,
    3600,
    'active',
    '{"environment": "test", "created_by": "system"}',
    '{"testing", "pro-tier"}',
    0,
    NOW(),
    NOW()
),
-- Test API Key 4 - Inactive Key
(
    'd4e5f6g7-h8i9-0123-defg-456789012345',
    'b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9', -- SHA256 of 'test-key-inactive-000'
    'Test Inactive Key', 
    'Inactive API key for testing',
    'basic',
    5000,
    3600,
    'inactive',
    '{"environment": "test", "created_by": "system"}',
    '{"testing", "inactive"}',
    0,
    NOW(),
    NOW()
);

-- Insert some sample usage logs
INSERT INTO usage_logs (
    id,
    api_key_id,
    endpoint,
    method,
    status_code,
    response_time_ms,
    request_size,
    response_size,
    user_agent,
    client_ip,
    country,
    request_count,
    timestamp,
    created_at
) VALUES
(
    gen_random_uuid(),
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
    '/api/v1/test',
    'GET',
    200,
    150,
    256,
    1024,
    'Test Client/1.0',
    '127.0.0.1',
    'US',
    1,
    NOW() - INTERVAL '1 hour',
    NOW() - INTERVAL '1 hour'
),
(
    gen_random_uuid(),
    'b2c3d4e5-f6g7-8901-bcde-f23456789012',
    '/api/v1/data',
    'POST',
    201,
    250,
    512,
    2048,
    'Standard Client/2.0',
    '192.168.1.1',
    'CA',
    1,
    NOW() - INTERVAL '30 minutes',
    NOW() - INTERVAL '30 minutes'
);

-- Insert sample rate limit violations
INSERT INTO rate_limit_violations (
    id,
    api_key_id,
    endpoint,
    method,
    client_ip,
    user_agent,
    country,
    violation_count,
    timestamp,
    created_at
) VALUES
(
    gen_random_uuid(),
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
    '/api/v1/test',
    'GET',
    '127.0.0.1',
    'Aggressive Client/1.0',
    'US',
    5,
    NOW() - INTERVAL '2 hours',
    NOW() - INTERVAL '2 hours'
);

-- Insert sample alerts
INSERT INTO alerts (
    id,
    api_key_id,
    type,
    severity,
    message,
    metadata,
    status,
    created_at,
    updated_at
) VALUES
(
    gen_random_uuid(),
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
    'rate_limit',
    'medium',
    'API key approaching rate limit threshold',
    '{"usage_percentage": 85, "threshold": 80}',
    'active',
    NOW() - INTERVAL '1 hour',
    NOW() - INTERVAL '1 hour'
);

-- Insert sample billing records
INSERT INTO billing_records (
    id,
    api_key_id,
    period_start,
    period_end,
    total_usage,
    total_cost,
    overage_usage,
    overage_cost,
    currency,
    status,
    created_at,
    updated_at
) VALUES
(
    gen_random_uuid(),
    'b2c3d4e5-f6g7-8901-bcde-f23456789012',
    DATE_TRUNC('month', NOW() - INTERVAL '1 month'),
    DATE_TRUNC('month', NOW()) - INTERVAL '1 second',
    8500,
    29.99,
    0,
    0.00,
    'USD',
    'paid',
    NOW() - INTERVAL '1 day',
    NOW() - INTERVAL '1 day'
);

-- Display created test data
SELECT 'API Keys Created:' as info;
SELECT name, tier, rate_limit, status FROM api_keys WHERE name LIKE 'Test%';

SELECT 'Usage Logs Created:' as info;
SELECT COUNT(*) as count FROM usage_logs;

SELECT 'Violations Created:' as info; 
SELECT COUNT(*) as count FROM rate_limit_violations;

SELECT 'Alerts Created:' as info;
SELECT COUNT(*) as count FROM alerts;

SELECT 'Billing Records Created:' as info;
SELECT COUNT(*) as count FROM billing_records;