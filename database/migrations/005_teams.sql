-- Teams table for team events
CREATE TABLE IF NOT EXISTS teams (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    event_id VARCHAR(36) NOT NULL,
    team_name VARCHAR(100) NOT NULL,
    country_code VARCHAR(10) NOT NULL,
    country_name VARCHAR(100),
    team_rank INT,
    total_score INT DEFAULT 0,
    total_x_count INT DEFAULT 0,
    status ENUM('active', 'eliminated', 'qualified') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES tournament_events(id) ON DELETE CASCADE,
    INDEX idx_teams_tournament (tournament_id),
    INDEX idx_teams_event (event_id)
);

-- Team members table
CREATE TABLE IF NOT EXISTS team_members (
    id VARCHAR(36) PRIMARY KEY,
    team_id VARCHAR(36) NOT NULL,
    participant_id VARCHAR(36) NOT NULL,
    member_order INT NOT NULL,
    is_substitute BOOLEAN DEFAULT FALSE,
    total_score INT DEFAULT 0,
    total_x_count INT DEFAULT 0,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (participant_id) REFERENCES tournament_participants(id) ON DELETE CASCADE,
    INDEX idx_team_members_team (team_id)
);

-- Team scores table
CREATE TABLE IF NOT EXISTS team_scores (
    id VARCHAR(36) PRIMARY KEY,
    team_id VARCHAR(36) NOT NULL,
    tournament_id VARCHAR(36) NOT NULL,
    session INT NOT NULL,
    distance_order INT NOT NULL,
    end_number INT NOT NULL,
    member_scores JSON,
    end_total INT DEFAULT 0,
    x_count INT DEFAULT 0,
    running_total INT DEFAULT 0,
    verified BOOLEAN DEFAULT FALSE,
    entered_by VARCHAR(36),
    entered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    UNIQUE KEY unique_team_score (team_id, session, distance_order, end_number),
    INDEX idx_team_scores_team (team_id)
);

-- Team elimination matches table
CREATE TABLE IF NOT EXISTS team_elimination_matches (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    event_id VARCHAR(36) NOT NULL,
    round VARCHAR(10) NOT NULL,
    match_number INT NOT NULL,
    team1_id VARCHAR(36),
    team2_id VARCHAR(36),
    score1 INT DEFAULT 0,
    score2 INT DEFAULT 0,
    set_score1 INT DEFAULT 0,
    set_score2 INT DEFAULT 0,
    winner_id VARCHAR(36),
    status ENUM('pending', 'ongoing', 'completed', 'bye') DEFAULT 'pending',
    scheduled_time DATETIME,
    actual_start_time DATETIME,
    actual_end_time DATETIME,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES tournament_events(id) ON DELETE CASCADE,
    FOREIGN KEY (team1_id) REFERENCES teams(id) ON DELETE SET NULL,
    FOREIGN KEY (team2_id) REFERENCES teams(id) ON DELETE SET NULL,
    FOREIGN KEY (winner_id) REFERENCES teams(id) ON DELETE SET NULL,
    INDEX idx_team_matches_event (event_id),
    INDEX idx_team_matches_round (round)
);

-- Activity logs table (if not exists)
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
    INDEX idx_activity_user (user_id),
    INDEX idx_activity_tournament (tournament_id),
    INDEX idx_activity_action (action)
);

-- Awards/Medals table
CREATE TABLE IF NOT EXISTS awards (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    event_id VARCHAR(36) NOT NULL,
    recipient_id VARCHAR(36) NOT NULL,
    recipient_type ENUM('individual', 'team') NOT NULL,
    award_type ENUM('gold', 'silver', 'bronze', 'participation') NOT NULL,
    rank INT NOT NULL,
    awarded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    awarded_by VARCHAR(36),
    notes TEXT,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES tournament_events(id) ON DELETE CASCADE,
    INDEX idx_awards_tournament (tournament_id),
    INDEX idx_awards_event (event_id),
    INDEX idx_awards_type (award_type)
);

-- Accreditations table
CREATE TABLE IF NOT EXISTS accreditations (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL,
    participant_id VARCHAR(36) NOT NULL,
    card_number VARCHAR(50) NOT NULL UNIQUE,
    card_type ENUM('athlete', 'coach', 'official', 'media', 'vip') NOT NULL,
    status ENUM('pending', 'printed', 'issued', 'revoked') DEFAULT 'pending',
    printed_at DATETIME,
    issued_at DATETIME,
    access_areas TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    FOREIGN KEY (participant_id) REFERENCES tournament_participants(id) ON DELETE CASCADE,
    INDEX idx_accred_tournament (tournament_id),
    INDEX idx_accred_card (card_number),
    INDEX idx_accred_status (status)
);

-- Gate check logs table
CREATE TABLE IF NOT EXISTS gate_check_logs (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36),
    accreditation_id VARCHAR(36),
    gate_name VARCHAR(50) NOT NULL,
    direction ENUM('in', 'out') NOT NULL,
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    checked_by VARCHAR(36),
    access_granted BOOLEAN DEFAULT FALSE,
    reason VARCHAR(255),
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    FOREIGN KEY (accreditation_id) REFERENCES accreditations(id) ON DELETE SET NULL,
    INDEX idx_gate_tournament (tournament_id),
    INDEX idx_gate_accred (accreditation_id),
    INDEX idx_gate_time (checked_at)
);
