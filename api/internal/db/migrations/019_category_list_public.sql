-- Per-forum visibility on community index: listed (locked) vs hidden.

ALTER TABLE categories
    ADD COLUMN list_public BOOLEAN NOT NULL DEFAULT TRUE;

-- Staff/admin rooms hidden by default; supporter areas listed (locked) by default.
UPDATE categories SET list_public = FALSE WHERE access_level IN ('staff', 'admin');