-- MySQL / MariaDB (UUID stored as CHAR(36))
-- Assumptions:
-- - You already have tables: events(uuid), event_categories(uuid)
-- - If you also have teams(uuid) and archers(uuid), those are referenced too.
-- - "Reset bracket" can be implemented by deleting the bracket row
--   (FK ON DELETE CASCADE will delete matches/ends/entries automatically).

CREATE TABLE elimination_brackets (
  uuid CHAR(36) PRIMARY KEY,
  event_uuid CHAR(36) NOT NULL,
  category_uuid CHAR(36) NOT NULL,

  -- individual / team / mixed
  bracket_type ENUM('individual','team3','mixed2') NOT NULL,

  -- only 2 formats you support now
  format ENUM('recurve_set','compound_total') NOT NULL,

  -- bracket size: 8/16/32/64 etc
  bracket_size INT UNSIGNED NOT NULL,

  -- how many qualify from qualification (usually equals bracket_size, but can differ if you allow BYE fill)
  qualified_count INT UNSIGNED NOT NULL,

  status ENUM('draft','generated','running','closed') NOT NULL DEFAULT 'draft',

  generated_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  KEY idx_eb_event_category (event_uuid, category_uuid),
  KEY idx_eb_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE elimination_entries (
  uuid CHAR(36) PRIMARY KEY,
  bracket_uuid CHAR(36) NOT NULL,

  -- one bracket can contain either archers or teams; keep it generic
  participant_type ENUM('archer','team') NOT NULL,
  participant_uuid CHAR(36) NOT NULL,

  seed INT UNSIGNED NOT NULL,

  -- snapshot from qualification (optional but useful for audit)
  qual_total_score INT UNSIGNED NULL,
  qual_total_x INT UNSIGNED NULL,
  qual_total_10 INT UNSIGNED NULL,

  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

  UNIQUE KEY uq_ee_bracket_seed (bracket_uuid, seed),
  UNIQUE KEY uq_ee_bracket_participant (bracket_uuid, participant_type, participant_uuid),

  KEY idx_ee_bracket (bracket_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE elimination_matches (
  uuid CHAR(36) PRIMARY KEY,
  bracket_uuid CHAR(36) NOT NULL,

  round_no INT UNSIGNED NOT NULL,     -- 1=R32, 2=R16, 3=QF, 4=SF, 5=Final (depending on size)
  match_no INT UNSIGNED NOT NULL,     -- sequence within the round

  entry_a_uuid CHAR(36) NULL,
  entry_b_uuid CHAR(36) NULL,

  winner_entry_uuid CHAR(36) NULL,

  is_bye TINYINT(1) NOT NULL DEFAULT 0,
  scheduled_at DATETIME NULL,

  status ENUM('scheduled','running','finished') NOT NULL DEFAULT 'scheduled',

  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  UNIQUE KEY uq_em_round_match (bracket_uuid, round_no, match_no),

  KEY idx_em_bracket_round (bracket_uuid, round_no),
  KEY idx_em_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- This table stores scoring per "end" for each side (A/B) in a match.
-- Works for both recurve_set and compound_total; you can compute:
-- - recurve_set: compare end_total A vs B to assign set points
-- - compound_total: sum end_total across ends for final total
CREATE TABLE elimination_match_ends (
  uuid CHAR(36) PRIMARY KEY,
  match_uuid CHAR(36) NOT NULL,

  end_no INT UNSIGNED NOT NULL,                 -- 1..N
  side ENUM('A','B') NOT NULL,                  -- score belongs to slot A or slot B

  end_total INT UNSIGNED NOT NULL DEFAULT 0,    -- sum of arrows for this end
  x_count INT UNSIGNED NOT NULL DEFAULT 0,
  ten_count INT UNSIGNED NOT NULL DEFAULT 0,

  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  UNIQUE KEY uq_eme_match_end_side (match_uuid, end_no, side),
  KEY idx_eme_match (match_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE elimination_match_arrow_scores (
  uuid CHAR(36) PRIMARY KEY,
  match_end_uuid CHAR(36) NOT NULL,
  arrow_no INT UNSIGNED NOT NULL,
  score TINYINT UNSIGNED NOT NULL,     -- 0..10
  is_x TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_emas_end_arrow (match_end_uuid, arrow_no),
  KEY idx_emas_end (match_end_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
