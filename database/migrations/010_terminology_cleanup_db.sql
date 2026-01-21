-- Migration 010: Terminology Cleanup (Tournament -> Event)
-- This migration standardizes table columns and creates missing activity log table

SET FOREIGN_KEY_CHECKS=0;

-- Rename columns in teams table
-- tournament_id -> event_id (the parent event)
-- event_id -> category_id (the specific division/category)
ALTER TABLE teams CHANGE COLUMN tournament_id event_id VARCHAR(36) NOT NULL;
ALTER TABLE teams CHANGE COLUMN event_id category_id VARCHAR(36) NOT NULL;

-- Rename columns in event_participants table
-- event_category_id -> category_id
-- athlete_id -> archer_id (consistency with unification of 006)
ALTER TABLE event_participants CHANGE COLUMN athlete_id archer_id VARCHAR(36) NOT NULL;
ALTER TABLE event_participants CHANGE COLUMN event_category_id category_id VARCHAR(36) NOT NULL;

-- Ensure events table is correct
-- (It was already renamed from tournaments in a previous step, but let's be sure)

-- Create activity_logs table with modern terminology if it doesn't exist
CREATE TABLE IF NOT EXISTS activity_logs (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36),
    event_id VARCHAR(36),
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50),
    entity_id VARCHAR(36),
    description TEXT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
    INDEX idx_user (user_id),
    INDEX idx_event (event_id),
    INDEX idx_action (action),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET FOREIGN_KEY_CHECKS=1;
