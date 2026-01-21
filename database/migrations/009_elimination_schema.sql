-- Migration 009: Elimination Phase Schema and Assignment History

CREATE TABLE IF NOT EXISTS `matches` (
    `uuid` CHAR(36) PRIMARY KEY,
    `event_category_uuid` CHAR(36) NOT NULL,
    `round_name` VARCHAR(50) NOT NULL COMMENT 'e.g., 1/32, 1/16, Quarter-final, Semi-final, Final',
    `match_order` INT NOT NULL DEFAULT 1,
    `status` VARCHAR(20) NOT NULL DEFAULT 'scheduled' COMMENT 'scheduled, live, completed, bye',
    `winner_uuid` CHAR(36) DEFAULT NULL,
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_matches_event_category` (`event_category_uuid`),
    INDEX `idx_matches_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `match_participants` (
    `uuid` CHAR(36) PRIMARY KEY,
    `match_uuid` CHAR(36) NOT NULL,
    `archer_uuid` CHAR(36) DEFAULT NULL COMMENT 'NULL for bye or TBD',
    `seed` INT DEFAULT NULL,
    `score` INT DEFAULT 0,
    `result` VARCHAR(10) DEFAULT NULL COMMENT 'win, loss',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_mp_match` (`match_uuid`),
    INDEX `idx_mp_archer` (`archer_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `match_target_assignments` (
    `uuid` CHAR(36) PRIMARY KEY,
    `match_uuid` CHAR(36) NOT NULL,
    `target_number` INT NOT NULL,
    `target_position` VARCHAR(5) DEFAULT NULL COMMENT 'A, B, or empty for full match on target',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_mta_match` (`match_uuid`),
    UNIQUE KEY `uk_mta_target` (`match_uuid`, `target_number`, `target_position`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `match_end_scores` (
    `uuid` CHAR(36) PRIMARY KEY,
    `match_participant_uuid` CHAR(36) NOT NULL,
    `end_number` INT NOT NULL,
    `arrow_scores` VARCHAR(255) DEFAULT NULL COMMENT 'Comma separated values: 10,9,X,...',
    `end_total` INT NOT NULL DEFAULT 0,
    `end_10_count` INT NOT NULL DEFAULT 0,
    `end_x_count` INT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_mes_participant` (`match_participant_uuid`),
    UNIQUE KEY `uk_mes_end` (`match_participant_uuid`, `end_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `assignment_history` (
    `uuid` CHAR(36) PRIMARY KEY,
    `event_uuid` CHAR(36) NOT NULL,
    `entity_type` VARCHAR(50) NOT NULL COMMENT 'archer, match, target',
    `entity_uuid` CHAR(36) NOT NULL,
    `previous_assignment` TEXT DEFAULT NULL,
    `new_assignment` TEXT DEFAULT NULL,
    `changed_by` CHAR(36) DEFAULT NULL COMMENT 'user_uuid who made the change',
    `reason` VARCHAR(255) DEFAULT NULL,
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX `idx_ah_event` (`event_uuid`),
    INDEX `idx_ah_entity` (`entity_type`, `entity_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
