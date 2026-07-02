-- One strike per incident; many posts can attach to the same strike within the window.
-- Each post may only be warned once (ever).

CREATE TABLE warning_posts (
    post_id     TEXT PRIMARY KEY REFERENCES posts(id) ON DELETE CASCADE,
    warning_id  TEXT NOT NULL REFERENCES actor_warnings(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX warning_posts_warning_idx ON warning_posts (warning_id);