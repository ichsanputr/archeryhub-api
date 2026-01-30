-- Add location_type column to events table
-- This replaces the discipline reference (type column) with a direct string value
ALTER TABLE events ADD COLUMN location_type VARCHAR(100) NULL;

-- Migrate existing data: copy discipline name from ref_disciplines or use type value as fallback
UPDATE events e
LEFT JOIN ref_disciplines d ON e.type = d.uuid OR e.type = d.code
SET e.location_type = COALESCE(d.name, e.type, '')
WHERE e.location_type IS NULL;

-- After migration, we can optionally keep the type column for backward compatibility
-- or remove it later if not needed
