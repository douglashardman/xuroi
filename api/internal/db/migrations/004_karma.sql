-- Karma from likes received (self-likes excluded at projection time)

ALTER TABLE actors ADD COLUMN karma INT NOT NULL DEFAULT 0;

UPDATE actors a
SET karma = COALESCE(sub.cnt, 0)
FROM (
    SELECT p.author_id, count(*)::int AS cnt
    FROM reactions r
    JOIN posts p ON p.id = r.post_id AND p.deleted_at IS NULL
    WHERE r.reactor_id <> p.author_id
    GROUP BY p.author_id
) sub
WHERE a.id = sub.author_id;