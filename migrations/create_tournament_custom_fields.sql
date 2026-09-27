-- Migration: create tournament_custom_fields and participant_custom_field_values
-- Description: Custom Registration Fields & Ticketing Engine for tournaments

CREATE TABLE IF NOT EXISTS tournament_custom_fields (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    uuid VARCHAR(64) UNIQUE NOT NULL,
    tournament_uuid VARCHAR(64) NOT NULL,
    field_key VARCHAR(64) NOT NULL,
    label_id VARCHAR(255) NOT NULL,
    label_en VARCHAR(255) NULL,
    placeholder_id VARCHAR(255) NULL,
    placeholder_en VARCHAR(255) NULL,
    description_id TEXT NULL,
    description_en TEXT NULL,
    field_type ENUM(
        'text',
        'textarea',
        'number',
        'select',
        'radio',
        'checkbox',
        'date',
        'file'
    ) NOT NULL DEFAULT 'text',
    options_json JSON NULL,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    applies_to_category_ids JSON NULL,
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_tournament_uuid (tournament_uuid),
    INDEX idx_field_order (tournament_uuid, display_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS participant_custom_field_values (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    uuid VARCHAR(64) UNIQUE NOT NULL,
    participant_uuid VARCHAR(64) NOT NULL,
    field_uuid VARCHAR(64) NOT NULL,
    field_value LONGTEXT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_participant_uuid (participant_uuid),
    INDEX idx_field_uuid (field_uuid),
    UNIQUE KEY uq_part_field (participant_uuid, field_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
