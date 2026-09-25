-- Migration: Normalize tournament page_settings into structured relational columns & tables

-- 1. Add fee column to tournament_categories
ALTER TABLE archeris.tournament_categories 
ADD COLUMN IF NOT EXISTS fee DECIMAL(12,2) NOT NULL DEFAULT 0.00 AFTER category_name_custom;

-- 2. Add relational fields to tournaments
ALTER TABLE archeris.tournaments
ADD COLUMN IF NOT EXISTS registration_start DATETIME NULL AFTER registration_deadline,
ADD COLUMN IF NOT EXISTS fee_mode ENUM('per_type', 'per_category') NOT NULL DEFAULT 'per_type' AFTER entry_fee,
ADD COLUMN IF NOT EXISTS fee_individual DECIMAL(12,2) NOT NULL DEFAULT 0.00 AFTER fee_mode,
ADD COLUMN IF NOT EXISTS fee_team DECIMAL(12,2) NOT NULL DEFAULT 0.00 AFTER fee_individual,
ADD COLUMN IF NOT EXISTS fee_mixed_team DECIMAL(12,2) NOT NULL DEFAULT 0.00 AFTER fee_team,
ADD COLUMN IF NOT EXISTS currency VARCHAR(10) NOT NULL DEFAULT 'IDR' AFTER fee_mixed_team,
ADD COLUMN IF NOT EXISTS country_code VARCHAR(5) NOT NULL DEFAULT 'ID' AFTER currency,
ADD COLUMN IF NOT EXISTS enable_manual_payment TINYINT(1) NOT NULL DEFAULT 1 AFTER country_code,
ADD COLUMN IF NOT EXISTS results_type ENUM('system', 'manual', 'external') NOT NULL DEFAULT 'system' AFTER enable_manual_payment;

-- 3. Create tournament_page_sections table (1-to-1)
CREATE TABLE IF NOT EXISTS archeris.tournament_page_sections (
    tournament_id VARCHAR(36) NOT NULL PRIMARY KEY,
    show_about TINYINT(1) NOT NULL DEFAULT 1,
    show_divisions TINYINT(1) NOT NULL DEFAULT 1,
    show_fees TINYINT(1) NOT NULL DEFAULT 1,
    show_payment_methods TINYINT(1) NOT NULL DEFAULT 1,
    show_prizes TINYINT(1) NOT NULL DEFAULT 1,
    show_schedule TINYINT(1) NOT NULL DEFAULT 1,
    show_location TINYINT(1) NOT NULL DEFAULT 1,
    show_faq TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_tps_tournament FOREIGN KEY (tournament_id) REFERENCES archeris.tournaments(uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4. Create tournament_prizes table (1-to-Many)
CREATE TABLE IF NOT EXISTS archeris.tournament_prizes (
    uuid VARCHAR(36) NOT NULL PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    rank_position INT NOT NULL DEFAULT 1,
    prize_title VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_tp_tournament (tournament_id),
    CONSTRAINT fk_tp_tournament FOREIGN KEY (tournament_id) REFERENCES archeris.tournaments(uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 5. Create tournament_documents table (1-to-Many)
CREATE TABLE IF NOT EXISTS archeris.tournament_documents (
    uuid VARCHAR(36) NOT NULL PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    doc_type ENUM('guidebook', 'result', 'regulation', 'other') NOT NULL DEFAULT 'guidebook',
    title VARCHAR(255) NOT NULL DEFAULT '',
    file_url VARCHAR(500) NOT NULL,
    description TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_td_tournament (tournament_id),
    CONSTRAINT fk_td_tournament FOREIGN KEY (tournament_id) REFERENCES archeris.tournaments(uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6. Create tournament_facilities table (1-to-Many)
CREATE TABLE IF NOT EXISTS archeris.tournament_facilities (
    uuid VARCHAR(36) NOT NULL PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    facility_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_tf_tournament (tournament_id),
    CONSTRAINT fk_tf_tournament FOREIGN KEY (tournament_id) REFERENCES archeris.tournaments(uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7. Create tournament_selected_payment_methods table (1-to-Many)
CREATE TABLE IF NOT EXISTS archeris.tournament_selected_payment_methods (
    uuid VARCHAR(36) NOT NULL PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    payment_method_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_tspm_tournament (tournament_id),
    CONSTRAINT fk_tspm_tournament FOREIGN KEY (tournament_id) REFERENCES archeris.tournaments(uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
