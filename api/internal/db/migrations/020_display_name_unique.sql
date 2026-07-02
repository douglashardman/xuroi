-- Case-insensitive display names for human members (Doug == doug).
-- Rename seed persona rows that collide with real accounts before adding the index.

UPDATE actors a
SET display_name = 'PutterTalk'
WHERE a.type = 'human'
  AND LOWER(TRIM(a.display_name)) = 'doug'
  AND NOT EXISTS (SELECT 1 FROM actor_emails e WHERE e.actor_id = a.id);

CREATE UNIQUE INDEX IF NOT EXISTS actors_human_display_name_lower_uniq
  ON actors (LOWER(TRIM(display_name)))
  WHERE type = 'human';