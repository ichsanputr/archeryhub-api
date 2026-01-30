-- Rename slug column to username in archers table
ALTER TABLE archers CHANGE COLUMN slug username VARCHAR(100) NULL;
