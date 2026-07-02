-- Online presence privacy (C21 extension).

ALTER TABLE actors
    ADD COLUMN IF NOT EXISTS hide_online_status BOOLEAN NOT NULL DEFAULT FALSE;