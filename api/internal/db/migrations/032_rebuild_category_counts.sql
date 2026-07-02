-- Rebuild denormalized category counters from live thread/post data.

UPDATE categories c
SET
    thread_count = COALESCE((
        SELECT count(*)::int
        FROM threads t
        WHERE t.category_id = c.id AND t.deleted_at IS NULL
    ), 0),
    post_count = COALESCE((
        SELECT count(*)::int
        FROM posts p
        JOIN threads t ON t.id = p.thread_id
        WHERE t.category_id = c.id
          AND t.deleted_at IS NULL
          AND p.deleted_at IS NULL
          AND p.moderation_status = 'approved'
    ), 0);