-- Remove max_participants from events; max participants is per event category (event_categories), not per event
ALTER TABLE events DROP COLUMN max_participants;
