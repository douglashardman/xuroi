-- Forums can require any one of several access groups (OR logic).

ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS access_levels TEXT[] NOT NULL DEFAULT ARRAY['public']::TEXT[];

UPDATE categories
SET access_levels = ARRAY[access_level]::TEXT[]
WHERE access_levels = ARRAY['public']::TEXT[]
  AND access_level IS NOT NULL
  AND access_level <> 'public';

CREATE INDEX IF NOT EXISTS idx_categories_access_levels
    ON categories USING GIN (access_levels);