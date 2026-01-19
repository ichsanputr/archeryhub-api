-- Migration 006: Refactor user types to Archer, Organization, and Club
-- This removes the 'users' table and moves auth fields into specific entity tables.

SET FOREIGN_KEY_CHECKS=0;

-- 1. Modify 'archers' table
-- Check if columns exist before adding (using subquery check or just run it if we're sure)
-- Since we saw the DESCRIBE earlier, we know they are missing.
ALTER TABLE archers 
ADD COLUMN username VARCHAR(50) AFTER id,
ADD COLUMN email VARCHAR(100) AFTER username,
ADD COLUMN password VARCHAR(255) AFTER email,
ADD COLUMN role ENUM('archer', 'admin') DEFAULT 'archer' AFTER password;

-- 2. Modify 'organizations' table
ALTER TABLE organizations
ADD COLUMN username VARCHAR(50) AFTER id,
ADD COLUMN password VARCHAR(255) AFTER email,
ADD COLUMN role ENUM('organization', 'admin') DEFAULT 'organization' AFTER password;

-- 3. Modify 'clubs' table
ALTER TABLE clubs
ADD COLUMN username VARCHAR(50) AFTER id,
ADD COLUMN password VARCHAR(255) AFTER email,
ADD COLUMN role ENUM('club', 'admin') DEFAULT 'club' AFTER password;

-- 4. Create unique indexes for login fields
CREATE UNIQUE INDEX idx_archers_email ON archers(email);
CREATE UNIQUE INDEX idx_archers_username ON archers(username);
CREATE UNIQUE INDEX idx_orgs_email ON organizations(email);
CREATE UNIQUE INDEX idx_orgs_username ON organizations(username);
CREATE UNIQUE INDEX idx_clubs_email ON clubs(email);
CREATE UNIQUE INDEX idx_clubs_username ON clubs(username);

-- 5. Migrate existing data
-- Map admin and organizers to organizations
INSERT INTO organizations (id, name, email, username, password, role, status, created_at)
SELECT id, full_name, email, username, password, 
       IF(role = 'admin', 'admin', 'organization'), 
       status, created_at
FROM users WHERE role IN ('admin', 'organizer');

-- Map athlete to archers
INSERT INTO archers (id, full_name, email, username, password, role, status, created_at)
SELECT id, full_name, email, username, password, 'archer', status, created_at
FROM users WHERE role = 'athlete';

-- Map judge/scorekeeper to archers (optional, but better than losing them)
INSERT INTO archers (id, full_name, email, username, password, role, status, created_at)
SELECT id, full_name, email, username, password, 'archer', status, created_at
FROM users WHERE role IN ('judge', 'scorekeeper')
AND id NOT IN (SELECT id FROM archers);

-- 6. Update foreign keys
-- tournaments.organizer_id
ALTER TABLE tournaments DROP FOREIGN KEY tournaments_ibfk_1;
-- No need to add new FK to a generic table if we have multiple, 
-- but we can keep the column and it will point to IDs that now exist in multiple tables. 
-- In practice, organizer_id will point to organizations.id or clubs.id now.

-- athletes.user_id (remove it as we use archers table as primary)
ALTER TABLE athletes DROP FOREIGN KEY athletes_ibfk_1;
ALTER TABLE athletes DROP COLUMN user_id;

-- qualification_scores.entered_by
ALTER TABLE qualification_scores DROP FOREIGN KEY qualification_scores_ibfk_3;

-- activity_logs.user_id
-- activity_logs has a foreign key to users.id
ALTER TABLE activity_logs DROP FOREIGN KEY activity_logs_ibfk_1;

-- 7. Drop users table
DROP TABLE users;

SET FOREIGN_KEY_CHECKS=1;
