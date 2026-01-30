-- Remove country column from archers table
-- Country is replaced with city field which already exists
ALTER TABLE archers DROP COLUMN country;
