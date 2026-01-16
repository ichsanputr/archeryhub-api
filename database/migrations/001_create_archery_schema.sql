-- Archery Hub Database Schema
-- Migration 001: Create core tables

SET FOREIGN_KEY_CHECKS=0;
SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
START TRANSACTION;
SET time_zone = "+00:00";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    full_name VARCHAR(100),
    role ENUM('admin', 'organizer', 'judge', 'scorekeeper', 'athlete') DEFAULT 'athlete',
    avatar_url VARCHAR(255),
    phone VARCHAR(20),
    status ENUM('active', 'inactive', 'suspended') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_email (email),
    INDEX idx_username (username),
    INDEX idx_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Tournaments table
CREATE TABLE IF NOT EXISTS tournaments (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(200) NOT NULL,
    short_name VARCHAR(100),
    venue VARCHAR(200),
    location VARCHAR(200),
    country VARCHAR(3),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    start_date DATETIME NOT NULL,
    end_date DATETIME NOT NULL,
    description TEXT,
    banner_url VARCHAR(255),
    logo_url VARCHAR(255),
    type VARCHAR(50) COMMENT 'Indoor, Outdoor, Field, 3D',
    num_distances TINYINT DEFAULT 1,
    num_sessions TINYINT DEFAULT 1,
    status ENUM('draft', 'published', 'ongoing', 'completed', 'archived') DEFAULT 'draft',
    organizer_id VARCHAR(36),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (organizer_id) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_code (code),
    INDEX idx_status (status),
    INDEX idx_start_date (start_date),
    INDEX idx_organizer (organizer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Divisions table (Recurve, Compound, Barebow, etc.)
CREATE TABLE IF NOT EXISTS divisions (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    code VARCHAR(10) NOT NULL UNIQUE,
    description TEXT,
    display_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert default divisions
INSERT INTO divisions (id, name, code, description, display_order) VALUES
('div-recurve', 'Recurve', 'R', 'Recurve bow division', 1),
('div-compound', 'Compound', 'C', 'Compound bow division', 2),
('div-barebow', 'Barebow', 'BB', 'Barebow division', 3),
('div-longbow', 'Longbow', 'LB', 'Longbow division', 4),
('div-traditional', 'Traditional', 'T', 'Traditional bow division', 5);

-- Categories table (Senior Men, Junior Women, etc.)
CREATE TABLE IF NOT EXISTS categories (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    code VARCHAR(10) NOT NULL,
    age_from INT,
    age_to INT,
    gender ENUM('M', 'F', 'X') COMMENT 'M=Male, F=Female, X=Open/Mixed',
    display_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY unique_code (code, gender),
    INDEX idx_gender (gender)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert default categories
INSERT INTO categories (id, name, code, age_from, age_to, gender, display_order) VALUES
-- Men's categories
('cat-senior-m', 'Senior Men', 'SM', 21, NULL, 'M', 1),
('cat-junior-m', 'Junior Men', 'JM', 18, 20, 'M', 2),
('cat-cadet-m', 'Cadet Men', 'CM', 15, 17, 'M', 3),
('cat-master-m', 'Master Men', 'MM', 50, NULL, 'M', 4),
-- Women's categories
('cat-senior-f', 'Senior Women', 'SW', 21, NULL, 'F', 5),
('cat-junior-f', 'Junior Women', 'JW', 18, 20, 'F', 6),
('cat-cadet-f', 'Cadet Women', 'CW', 15, 17, 'F', 7),
('cat-master-f', 'Master Women', 'MW', 50, NULL, 'F', 8),
-- Open categories
('cat-open', 'Open', 'O', NULL, NULL, 'X', 9);

-- Tournament Events (combination of tournament + division + category)
CREATE TABLE IF NOT EXISTS tournament_events (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    division_id VARCHAR(36) NOT NULL,
    category_id VARCHAR(36) NOT NULL,
    max_participants INT DEFAULT 0,
    qualification_arrows INT DEFAULT 72,
    elimination_format VARCHAR(20) DEFAULT 'single' COMMENT 'single, round_robin',
    team_event BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    FOREIGN KEY (division_id) REFERENCES divisions(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
    UNIQUE KEY unique_event (tournament_id, division_id, category_id),
    INDEX idx_tournament (tournament_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Athletes table
CREATE TABLE IF NOT EXISTS athletes (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36),
    athlete_code VARCHAR(20) UNIQUE,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    date_of_birth DATE,
    gender ENUM('M', 'F', 'X'),
    country VARCHAR(3),
    club VARCHAR(100),
    email VARCHAR(100),
    phone VARCHAR(20),
    photo_url VARCHAR(255),
    address TEXT,
    emergency_contact VARCHAR(100),
    emergency_phone VARCHAR(20),
    status ENUM('active', 'inactive', 'suspended', 'pending') DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_code (athlete_code),
    INDEX idx_name (last_name, first_name),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Tournament Participants (Registration)
CREATE TABLE IF NOT EXISTS tournament_participants (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    athlete_id VARCHAR(36) NOT NULL,
    event_id VARCHAR(36) NOT NULL,
    back_number VARCHAR(10),
    target_number VARCHAR(10),
    session TINYINT,
    registration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    payment_status ENUM('pending', 'paid', 'waived', 'refunded') DEFAULT 'pending',
    payment_amount DECIMAL(10, 2) DEFAULT 0.00,
    accreditation_status ENUM('pending', 'printed', 'collected') DEFAULT 'pending',
    notes TEXT,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    FOREIGN KEY (athlete_id) REFERENCES athletes(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES tournament_events(id) ON DELETE CASCADE,
    UNIQUE KEY unique_participant (tournament_id, athlete_id, event_id),
    INDEX idx_tournament (tournament_id),
    INDEX idx_athlete (athlete_id),
    INDEX idx_event (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Sessions
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    session_order TINYINT NOT NULL,
    name VARCHAR(100),
    session_date DATE,
    start_time TIME,
    end_time TIME,
    num_targets INT,
    athletes_per_target TINYINT DEFAULT 4,
    locked BOOLEAN DEFAULT FALSE,
    notes TEXT,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    INDEX idx_tournament (tournament_id),
    INDEX idx_order (tournament_id, session_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Distances
CREATE TABLE IF NOT EXISTS distances (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    event_id VARCHAR(36) NOT NULL,
    distance_order TINYINT NOT NULL,
    distance_value INT NOT NULL COMMENT 'Distance in meters',
    arrows_per_end TINYINT DEFAULT 6,
    num_ends TINYINT DEFAULT 12,
    target_face VARCHAR(50) COMMENT '122cm, 80cm, 60cm, 40cm, etc.',
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES tournament_events(id) ON DELETE CASCADE,
    INDEX idx_tournament (tournament_id),
    INDEX idx_event (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Qualification Scores
CREATE TABLE IF NOT EXISTS qualification_scores (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    participant_id VARCHAR(36) NOT NULL,
    session TINYINT NOT NULL,
    distance_order TINYINT NOT NULL,
    end_number TINYINT NOT NULL,
    arrow_1 TINYINT,
    arrow_2 TINYINT,
    arrow_3 TINYINT,
    arrow_4 TINYINT,
    arrow_5 TINYINT,
    arrow_6 TINYINT,
    end_total SMALLINT,
    running_total SMALLINT,
    x_count TINYINT DEFAULT 0,
    ten_count TINYINT DEFAULT 0,
    verified BOOLEAN DEFAULT FALSE,
    entered_by VARCHAR(36),
    entered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    FOREIGN KEY (participant_id) REFERENCES tournament_participants(id) ON DELETE CASCADE,
    FOREIGN KEY (entered_by) REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE KEY unique_score (tournament_id, participant_id, session, distance_order, end_number),
    INDEX idx_tournament (tournament_id),
    INDEX idx_participant (participant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Elimination Matches
CREATE TABLE IF NOT EXISTS elimination_matches (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    event_id VARCHAR(36) NOT NULL,
    round VARCHAR(20) COMMENT 'R32, R16, R8, QF, SF, BM (Bronze), GM (Gold)',
    match_number TINYINT NOT NULL,
    participant1_id VARCHAR(36),
    participant2_id VARCHAR(36),
    score1 TINYINT DEFAULT 0,
    score2 TINYINT DEFAULT 0,
    set_score1 TINYINT DEFAULT 0,
    set_score2 TINYINT DEFAULT 0,
    winner_id VARCHAR(36),
    status ENUM('pending', 'ongoing', 'completed', 'bye') DEFAULT 'pending',
    scheduled_time DATETIME,
    actual_start_time DATETIME,
    actual_end_time DATETIME,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES tournament_events(id) ON DELETE CASCADE,
    FOREIGN KEY (participant1_id) REFERENCES tournament_participants(id) ON DELETE SET NULL,
    FOREIGN KEY (participant2_id) REFERENCES tournament_participants(id) ON DELETE SET NULL,
    FOREIGN KEY (winner_id) REFERENCES tournament_participants(id) ON DELETE SET NULL,
    INDEX idx_tournament (tournament_id),
    INDEX idx_event (event_id),
    INDEX idx_round (tournament_id, event_id, round)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Devices (for mobile scoring)
CREATE TABLE IF NOT EXISTS devices (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    device_code VARCHAR(20) UNIQUE NOT NULL,
    device_name VARCHAR(100),
    device_type ENUM('tablet', 'phone', 'scorekeeper', 'display') DEFAULT 'tablet',
    pin VARCHAR(10),
    qr_payload TEXT,
    target_assignment VARCHAR(10),
    session TINYINT,
    last_sync TIMESTAMP,
    status ENUM('active', 'inactive', 'blocked') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    INDEX idx_tournament (tournament_id),
    INDEX idx_code (device_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Activity Log
CREATE TABLE IF NOT EXISTS activity_logs (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36),
    tournament_id VARCHAR(36),
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50),
    entity_id VARCHAR(36),
    description TEXT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    INDEX idx_user (user_id),
    INDEX idx_tournament (tournament_id),
    INDEX idx_action (action),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET FOREIGN_KEY_CHECKS=1;
COMMIT;
