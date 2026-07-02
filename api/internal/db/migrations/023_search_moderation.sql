-- H7 async search indexing + E2 post moderation queue

ALTER TABLE categories
    ADD COLUMN post_moderation BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE posts
    ADD COLUMN moderation_status TEXT NOT NULL DEFAULT 'approved'
    CHECK (moderation_status IN ('approved', 'pending', 'rejected'));

CREATE INDEX posts_moderation_pending_idx ON posts (created_at DESC)
    WHERE moderation_status = 'pending' AND deleted_at IS NULL;

CREATE TABLE search_documents (
    entity_id     TEXT PRIMARY KEY,
    doc_type      TEXT NOT NULL CHECK (doc_type IN ('thread', 'post')),
    thread_id     TEXT NOT NULL,
    category_id   TEXT NOT NULL,
    title         TEXT NOT NULL DEFAULT '',
    body          TEXT NOT NULL DEFAULT '',
    author_name   TEXT NOT NULL DEFAULT '',
    thread_slug   TEXT NOT NULL DEFAULT '',
    thread_title  TEXT NOT NULL DEFAULT '',
    access_level  TEXT NOT NULL DEFAULT 'public',
    search_vector tsvector,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX search_documents_vector_idx ON search_documents USING GIN (search_vector);
CREATE INDEX search_documents_thread_idx ON search_documents (thread_id);

CREATE TABLE search_index_queue (
    entity_id   TEXT PRIMARY KEY,
    doc_type    TEXT NOT NULL CHECK (doc_type IN ('thread', 'post')),
    enqueued_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX search_index_queue_enqueued_idx ON search_index_queue (enqueued_at);

-- Classifieds forums require mod approval (E2 / BST)
UPDATE categories SET post_moderation = TRUE
WHERE slug IN ('free-classifieds', 'wanted-trade', 'ebay-items');