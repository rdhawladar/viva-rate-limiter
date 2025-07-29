-- Initialize Viva Rate Limiter Database
-- This script runs when PostgreSQL container starts for the first time

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Create additional databases for testing
CREATE DATABASE viva_ratelimiter_test;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE viva_ratelimiter TO viva_user;
GRANT ALL PRIVILEGES ON DATABASE viva_ratelimiter_test TO viva_user;

-- Connect to main database
\c viva_ratelimiter;

-- Create enum types
CREATE TYPE api_key_status AS ENUM ('active', 'suspended', 'revoked');
CREATE TYPE api_key_tier AS ENUM ('free', 'pro', 'enterprise');
CREATE TYPE alert_type AS ENUM ('usage_threshold', 'rate_limit_exceeded', 'billing_overage', 'security_alert');

-- Create indexes for better performance
-- These will be created by migrations, but having them here for reference

-- Sample data for development (optional)
-- This will be handled by the application, not by init script