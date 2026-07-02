-- P1 Batch 5: thread merge redirect + optional spam score on pending posts

ALTER TABLE threads
    ADD COLUMN IF NOT EXISTS merged_into_thread_id TEXT REFERENCES threads(id);

CREATE INDEX IF NOT EXISTS threads_merged_into_idx
    ON threads (merged_into_thread_id)
    WHERE merged_into_thread_id IS NOT NULL;

ALTER TABLE posts
    ADD COLUMN IF NOT EXISTS spam_score INT NOT NULL DEFAULT 0;