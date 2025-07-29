-- Drop triggers
DROP TRIGGER IF EXISTS update_api_keys_updated_at ON api_keys;
DROP TRIGGER IF EXISTS update_alerts_updated_at ON alerts;
DROP TRIGGER IF EXISTS update_billing_records_updated_at ON billing_records;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop foreign key constraints
ALTER TABLE usage_logs DROP CONSTRAINT IF EXISTS fk_usage_logs_api_key_id;
ALTER TABLE alerts DROP CONSTRAINT IF EXISTS fk_alerts_api_key_id;
ALTER TABLE rate_limit_violations DROP CONSTRAINT IF EXISTS fk_rate_violations_api_key_id;
ALTER TABLE billing_records DROP CONSTRAINT IF EXISTS fk_billing_records_api_key_id;

-- Drop indexes
DROP INDEX IF EXISTS idx_api_keys_key_hash;
DROP INDEX IF EXISTS idx_api_keys_status;
DROP INDEX IF EXISTS idx_api_keys_tier;
DROP INDEX IF EXISTS idx_api_keys_user_id;
DROP INDEX IF EXISTS idx_api_keys_team_id;
DROP INDEX IF EXISTS idx_api_keys_created_at;

DROP INDEX IF EXISTS idx_usage_logs_api_key_id;
DROP INDEX IF EXISTS idx_usage_logs_timestamp;
DROP INDEX IF EXISTS idx_usage_logs_endpoint;
DROP INDEX IF EXISTS idx_usage_logs_status_code;
DROP INDEX IF EXISTS idx_usage_logs_ip_address;

DROP INDEX IF EXISTS idx_alerts_api_key_id;
DROP INDEX IF EXISTS idx_alerts_type;
DROP INDEX IF EXISTS idx_alerts_severity;
DROP INDEX IF EXISTS idx_alerts_status;
DROP INDEX IF EXISTS idx_alerts_created_at;

DROP INDEX IF EXISTS idx_rate_violations_api_key_id;
DROP INDEX IF EXISTS idx_rate_violations_timestamp;
DROP INDEX IF EXISTS idx_rate_violations_endpoint;
DROP INDEX IF EXISTS idx_rate_violations_client_ip;
DROP INDEX IF EXISTS idx_rate_violations_processed;
DROP INDEX IF EXISTS idx_rate_violations_is_repeated;

DROP INDEX IF EXISTS idx_billing_records_api_key_id;
DROP INDEX IF EXISTS idx_billing_records_period;
DROP INDEX IF EXISTS idx_billing_records_status;
DROP INDEX IF EXISTS idx_billing_records_created_at;

-- Drop partitions for rate_limit_violations
DROP TABLE IF EXISTS rate_limit_violations_y2024m01;
DROP TABLE IF EXISTS rate_limit_violations_y2024m02;
DROP TABLE IF EXISTS rate_limit_violations_y2024m03;
DROP TABLE IF EXISTS rate_limit_violations_y2024m04;
DROP TABLE IF EXISTS rate_limit_violations_y2024m05;
DROP TABLE IF EXISTS rate_limit_violations_y2024m06;
DROP TABLE IF EXISTS rate_limit_violations_y2024m07;
DROP TABLE IF EXISTS rate_limit_violations_y2024m08;
DROP TABLE IF EXISTS rate_limit_violations_y2024m09;
DROP TABLE IF EXISTS rate_limit_violations_y2024m10;
DROP TABLE IF EXISTS rate_limit_violations_y2024m11;
DROP TABLE IF EXISTS rate_limit_violations_y2024m12;

-- Drop partitions for usage_logs
DROP TABLE IF EXISTS usage_logs_y2024m01;
DROP TABLE IF EXISTS usage_logs_y2024m02;
DROP TABLE IF EXISTS usage_logs_y2024m03;
DROP TABLE IF EXISTS usage_logs_y2024m04;
DROP TABLE IF EXISTS usage_logs_y2024m05;
DROP TABLE IF EXISTS usage_logs_y2024m06;
DROP TABLE IF EXISTS usage_logs_y2024m07;
DROP TABLE IF EXISTS usage_logs_y2024m08;
DROP TABLE IF EXISTS usage_logs_y2024m09;
DROP TABLE IF EXISTS usage_logs_y2024m10;
DROP TABLE IF EXISTS usage_logs_y2024m11;
DROP TABLE IF EXISTS usage_logs_y2024m12;

-- Drop tables
DROP TABLE IF EXISTS billing_records;
DROP TABLE IF EXISTS rate_limit_violations;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS usage_logs;
DROP TABLE IF EXISTS api_keys;

-- Drop enum types
DROP TYPE IF EXISTS billing_period_status;
DROP TYPE IF EXISTS alert_status;
DROP TYPE IF EXISTS alert_severity;
DROP TYPE IF EXISTS alert_type;
DROP TYPE IF EXISTS api_key_tier;
DROP TYPE IF EXISTS api_key_status;