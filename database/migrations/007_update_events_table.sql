-- Migration 007: Update events table with missing fields
ALTER TABLE events 
MODIFY COLUMN start_date DATETIME DEFAULT NULL,
MODIFY COLUMN end_date DATETIME DEFAULT NULL,
ADD COLUMN entry_fee DECIMAL(10, 2) DEFAULT 0.00 AFTER num_sessions,
ADD COLUMN max_participants INT DEFAULT NULL AFTER entry_fee,
ADD COLUMN registration_deadline DATETIME DEFAULT NULL AFTER end_date;
