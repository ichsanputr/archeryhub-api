-- Migration: Remove nik, height_cm, and weight_kg from archers table
ALTER TABLE archers 
  DROP COLUMN IF EXISTS nik, 
  DROP COLUMN IF EXISTS height_cm, 
  DROP COLUMN IF EXISTS weight_kg;
