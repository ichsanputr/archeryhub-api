-- Migration: Create event_reset_codes table
-- Date: 2026-05-19
-- Description: Creates table to store OTP codes for event reset validation

CREATE TABLE IF NOT EXISTS event_reset_codes (
    uuid VARCHAR(36) PRIMARY KEY,
    event_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    code VARCHAR(6) NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_event_user_code (event_id, user_id, code),
    INDEX idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
