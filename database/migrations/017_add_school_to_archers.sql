-- Add school column to archers table
ALTER TABLE archers
  ADD COLUMN school VARCHAR(255) NULL AFTER city;

