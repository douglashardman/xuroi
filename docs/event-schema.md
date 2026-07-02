# Xuroi Event Schema

**Status:** Phase 0 spec · v1  
**Last updated:** July 1, 2026

This document defines the append-only event log, stream conventions, event types, and projection rules for Xuroi's event-sourced core.

---

## 1. Design principles

1. **Append-only.** Events are never updated or deleted. Corrections append new events.
2. **Stream-scoped ordering.** Events within a stream are totally ordered by `sequence`. Cross-stream order is not guaranteed.
3. **Projections are disposable.** Current state lives in projection tables rebuilt from the log.
4. **Payloads are versioned.** Each event type has a `schema_version` in its payload envelope.
5. **Actor attribution.** Every mutating event records `actor_id` (who caused it).
6. **Idempotency.** Commands carry `idempotency_key`; duplicate keys within a stream are rejected.

---

## 2. Identifiers

| Entity | Format | Example |
|---|---|---|
| Actor | `act_{ulid}` | `act_01JXYZ...` |
| Category | `cat_{ulid}` | `cat_01JXYZ...` |
| Thread | `thr_{ulid}` | `thr_01JXYZ...` |
| Post | `pst_{ulid}` | `pst_01JXYZ...` |
| Media | `med_{ulid}` | `med_01JXYZ...` |
| Event | `evt_{ulid}` | `evt_01JXYZ...` |

**Stream ID** = namespace + entity ID:

```
site                          — global site config (single stream)
category:{category_id}        — category lifecycle
thread:{thread_id}            — thread + posts in that thread
actor:{actor_id}              — actor profile changes
media:{media_id}              — upload pipeline
```

Posts are events on the **thread stream**, not a separate post stream. A post's identity is `post_id` in the payload; the stream is always `thread:{thread_id}`.

---

## 3. Event envelope (storage)

Stored in `events` table:

```json
{
  "id": "evt_01JXYZ...",
  "stream_id": "thread:thr_01JXYZ...",
  "sequence": 3,
  "type": "post.created",
  "actor_id": "act_01JXYZ...",
  "payload": { },
  "schema_version": 1,
  "idempotency_key": "optional-client-key",
  "created_at": "2026-07-01T12:00:00Z"
}
```

| Field | Type | Notes |
|---|---|---|
| `id` | string | ULID, globally unique |
| `stream_id` | string | Partition key for ordering |
| `sequence` | int64 | Monotonic per stream, assigned by store |
| `type` | string | Dot-notation event name |
| `actor_id` | string | Nullable for system/service actors |
| `payload` | jsonb | Type-specific body |
| `schema_version` | int | Payload schema version |
| `idempotency_key` | string | Optional; unique per stream when set |
| `created_at` | timestamptz | Server-assigned |

**Invariant:** `(stream_id, sequence)` is unique. `(stream_id, idempotency_key)` is unique when key is non-null.

---

## 4. Actor types

| Type | Description |
|---|---|
| `human` | Registered user |
| `agent` | Automated participant; `disclosure_required: true` |
| `service` | Internal worker (summarizer, indexer) |

Actor registration is outside the event log (direct insert to `actors` table). Profile changes emit `actor.updated` on `actor:{id}` stream.

---

## 5. Event catalog

### Phase 0 (implement now)

#### `category.created`

Stream: `site`  
Actor: admin or service (seed)

```json
{
  "category_id": "cat_01JXYZ...",
  "slug": "equipment",
  "name": "Equipment & Reviews",
  "description": "Putters, grips, shafts...",
  "sort_order": 3,
  "parent_id": null
}
```

**Projection:** Insert row in `categories`.

---

#### `thread.created`

Stream: `thread:{thread_id}`  
Sequence: 1 (first event on stream)

```json
{
  "thread_id": "thr_01JXYZ...",
  "post_id": "pst_01JXYZ...",
  "category_id": "cat_01JXYZ...",
  "title": "Odyssey White Hot vs Scotty Newport",
  "slug": "odyssey-white-hot-vs-scotty-newport",
  "author_id": "act_01JXYZ...",
  "body_markdown": "Trying to decide between these two...",
  "body_html": "<p>Trying to decide...</p>"
}
```

**Projection:**
- Insert `threads` row (OP post is post #1).
- Insert `posts` row with `position: 1`, `is_op: true`.

---

#### `post.created`

Stream: `thread:{thread_id}`

```json
{
  "post_id": "pst_01JXYZ...",
  "thread_id": "thr_01JXYZ...",
  "author_id": "act_01JXYZ...",
  "body_markdown": "White Hot on fast greens...",
  "body_html": "<p>White Hot on fast greens...</p>",
  "quoted_post_id": null
}
```

**Projection:**
- Insert `posts` row with next `position`.
- Increment `threads.reply_count`.
- Update `threads.last_activity_at`.

---

#### `post.edited`

Stream: `thread:{thread_id}`

```json
{
  "post_id": "pst_01JXYZ...",
  "thread_id": "thr_01JXYZ...",
  "body_markdown": "Updated text...",
  "body_html": "<p>Updated text...</p>",
  "edit_reason": null
}
```

**Projection:**
- Update `posts.body_markdown`, `body_html`, `edited_at`.
- Append row to `post_revisions` (full snapshot).

---

#### `post.deleted`

Stream: `thread:{thread_id}`

```json
{
  "post_id": "pst_01JXYZ...",
  "thread_id": "thr_01JXYZ...",
  "reason": "spam",
  "hard": false
}
```

**Projection:**
- Set `posts.deleted_at`, `posts.deleted_by`.
- If OP and `hard: false`, thread remains; body shows "[deleted]".
- Decrement `threads.reply_count` if not OP.

---

#### `thread.locked` / `thread.unlocked`

Stream: `thread:{thread_id}`

```json
{
  "thread_id": "thr_01JXYZ..."
}
```

**Projection:** Set `threads.is_locked`.

---

#### `thread.pinned` / `thread.unpinned`

Stream: `thread:{thread_id}`

```json
{
  "thread_id": "thr_01JXYZ..."
}
```

**Projection:** Set `threads.is_pinned`.

---

#### `reaction.added`

Stream: `thread:{thread_id}`

```json
{
  "post_id": "pst_01JXYZ...",
  "reactor_id": "act_01JXYZ...",
  "reaction_type": "like"
}
```

**Projection:** Upsert `reactions`; increment `posts.reaction_count`.

---

#### `reaction.removed`

Stream: `thread:{thread_id}`

```json
{
  "post_id": "pst_01JXYZ...",
  "reactor_id": "act_01JXYZ...",
  "reaction_type": "like"
}
```

**Projection:** Delete `reactions` row; decrement count.

---

### Phase 1+ (defined, not implemented in Phase 0)

| Type | Stream | Notes |
|---|---|---|
| `thread.moved` | `thread:{id}` | Change category |
| `thread.title_changed` | `thread:{id}` | Slug may change; old slug 301s |
| `media.uploaded` | `media:{id}` | Object storage reference |
| `media.attached` | `thread:{id}` | Link media to post |
| `actor.role_changed` | `actor:{id}` | member → moderator |
| `thread.summarized` | `thread:{id}` | Intelligence layer output |

---

## 6. Projection tables

### `categories`

| Column | Source |
|---|---|
| `id`, `slug`, `name`, `description` | `category.created` |
| `sort_order`, `parent_id` | `category.created` |
| `thread_count`, `post_count` | Denormalized counters (updated on thread/post events) |

### `threads`

| Column | Source |
|---|---|
| `id`, `category_id`, `title`, `slug` | `thread.created`, `thread.title_changed` |
| `author_id` | `thread.created` |
| `reply_count` | Incremented on `post.created`, decremented on `post.deleted` |
| `is_locked`, `is_pinned` | Lock/pin events |
| `created_at` | `thread.created` timestamp |
| `last_activity_at` | Latest post or edit |
| `deleted_at` | `thread.deleted` (Phase 1) |

### `posts`

| Column | Source |
|---|---|
| `id`, `thread_id`, `author_id` | `post.created` |
| `position` | Sequence within thread (1 = OP) |
| `body_markdown`, `body_html` | Latest from create/edit events |
| `quoted_post_id` | `post.created` |
| `is_op` | `position == 1` |
| `reaction_count` | `reaction.*` events |
| `created_at`, `edited_at`, `deleted_at` | Event timestamps |

### `post_revisions`

Full snapshot on each `post.edited`: `post_id`, `revision`, `body_markdown`, `body_html`, `edited_at`, `editor_id`.

### `reactions`

`(post_id, reactor_id, reaction_type)` unique.

---

## 7. Projection rules

### Apply (online)

On each appended event:
1. Load projector for `event.type`.
2. Apply mutation to projection tables in same DB transaction as append.
3. Commit or rollback atomically.

### Rebuild (offline)

```sql
TRUNCATE categories, threads, posts, post_revisions, reactions;
-- Replay all events ORDER BY stream_id, sequence
```

Used for: schema migration, bug fix, disaster recovery.

**Phase 0 target:** Rebuild 1M events in < 5 minutes on dev hardware.

---

## 8. Command → event mapping (API)

| HTTP command | Events emitted |
|---|---|
| `POST /v1/categories` | `category.created` on `site` |
| `POST /v1/threads` | `thread.created` on new `thread:{id}` stream |
| `POST /v1/threads/{id}/posts` | `post.created` on `thread:{id}` |
| `PATCH /v1/posts/{id}` | `post.edited` on `thread:{id}` |
| `DELETE /v1/posts/{id}` | `post.deleted` on `thread:{id}` |
| `POST /v1/posts/{id}/reactions` | `reaction.added` |

Commands validate permissions before append. Read path queries projections only — never the event log.

---

## 9. Optimistic concurrency

Clients may send `expected_sequence` (last known sequence on stream). If actual sequence differs, return `409 Conflict` with current head sequence. Prevents lost updates on rapid edits.

---

## 10. Schema evolution

1. Add new event types freely — old projectors ignore unknown types.
2. Bump `schema_version` on payload shape changes.
3. Projectors handle all known versions or fail rebuild with clear error.
4. Never remove event types from the log.

---

## 11. Phase 0 exit checklist

- [ ] `events` table with stream ordering
- [ ] Append with idempotency
- [ ] Projectors: `category.created`, `thread.created`, `post.created`
- [ ] Rebuild from log
- [ ] API: health, create thread, create post
- [ ] Seed PutterTalk categories via `category.created` events

---

## References

- `NEXT-GEN-FORUM-BATTLE-PLAN.md` §5.2 (core data model sketch)
- `xuroi/WISH-LIST.md` A11 (event-sourced posts)
- `xuroi/sites/puttertalk/site.json` (category seed data)