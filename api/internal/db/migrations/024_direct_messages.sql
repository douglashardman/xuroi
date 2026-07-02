-- D1 private messaging + D7 DM privacy

ALTER TABLE actors
    ADD COLUMN dm_privacy TEXT NOT NULL DEFAULT 'everyone'
    CHECK (dm_privacy IN ('everyone', 'friends_only', 'off'));

CREATE TABLE dm_conversations (
    id               TEXT PRIMARY KEY,
    participant_a    TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    participant_b    TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_message_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (participant_a < participant_b)
);

CREATE UNIQUE INDEX dm_conversations_pair_idx ON dm_conversations (participant_a, participant_b);
CREATE INDEX dm_conversations_last_message_idx ON dm_conversations (last_message_at DESC);

CREATE TABLE dm_messages (
    id              TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES dm_conversations(id) ON DELETE CASCADE,
    sender_id       TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    body_markdown   TEXT NOT NULL,
    body_html       TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX dm_messages_conversation_idx ON dm_messages (conversation_id, created_at ASC);

CREATE TABLE dm_reads (
    actor_id         TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    conversation_id  TEXT NOT NULL REFERENCES dm_conversations(id) ON DELETE CASCADE,
    last_read_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (actor_id, conversation_id)
);