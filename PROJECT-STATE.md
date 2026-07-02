# Xuroi Project State

**Grok: read `.grok/session/CHANGELOG.md` first** (local `Forum-Idea/.grok/session/`) — then this file.

**Last updated:** July 2, 2026 — P1 Batch 5 shipped (11/12)  
**Repo:** [github.com/douglashardman/xuroi](https://github.com/douglashardman/xuroi) (public)  
**Session status:** **P1 Batch 5** done — see `.grok/session/notes/2026-07-02-p1-batch-5-done.md` · Deferred: C6 2FA

---

## One-liner

**Xuroi** is a community knowledge engine. **PutterTalk** (puttertalk.com) is Customer #1 — greenfield relaunch of Doug's forum (2006–2019, ~17K members, ~4M posts, data lost, domain recovered).

---

## Strategy (locked)

| Decision | Value |
|---|---|
| Product name | Xuroi (xuroi.com) |
| Flagship site | PutterTalk |
| Model | Keynote — dogfood first, productize if earned |
| Migration | None — fresh start |
| Multi-tenant v1 | No — single deployment |
| Stack | Go API · Astro web · Postgres · Redis · Mustache themes |
| Footer | **Powered by Xuroi** on every public page — LOCKED |

---

## Key docs

| Doc | Path |
|---|---|
| Changelog (agent) | `../.grok/session/CHANGELOG.md` (local workspace) |
| This file | `PROJECT-STATE.md` |
| Backlog | `WISH-LIST.md` |
| Battle plan | `../NEXT-GEN-FORUM-BATTLE-PLAN.md` (local) |
| Theme contract | `theme-contract/THEMING.md` |
| Site config | `sites/puttertalk/site.json` |
| Email ops | `docs/email-deliverability.md` |

---

## Repo layout (current)

```
xuroi/                    ← git root on GitHub
  api/                    Go API, auth, events, migrations (001–035)
  web/                    Astro SSR — production UI for launch
  worker/                 README only; jobs run via api/cmd/* for now
  theme-contract/         Schema, fixtures, THEMING.md
  themes/puttertalk/      Brand theme + HTML mockups
  sites/puttertalk/       Categories, admin emails, feature flags
  infra/                  docker-compose, backup.sh, uploads
  docs/event-schema.md
  docs/email-deliverability.md
```

Local workspace also has `Forum-Idea/phpBB3/`, `xenforo_*`, `smf_*` (reference only — not in git).

---

## Shipped in dev (P0)

### Platform core
- [x] Event log + projections + rebuild (`docs/event-schema.md`, migrations 001+)
- [x] Go API (`cmd/xuroi`) · Postgres on :5433 · seed (`cmd/seed`)
- [x] Intelligence worker stub (`cmd/intelligence` — heuristic summaries)
- [x] Notify worker (`cmd/notify` — thread-reply email digests)
- [x] **Search indexer** (`cmd/searchindex` — async FTS, migration 023)
- [x] Backup script + launchd schedule (`infra/backup.sh`, `install-backup-schedule.sh`)
- [x] **Structured JSON request logging** (M12)

### Forum content
- [x] Categories, threads, posts, pagination, new thread + reply composers
- [x] Nested category groups (7 sections, 22 forums incl. Supporter + Staff areas)
- [x] Community index with latest activity per forum
- [x] Quote post · reactions/likes · karma · edit own post (30 min) · revision overlay
- [x] Pin/lock thread · soft-delete post · **staff thread delete** · **merge threads** (B16)
- [x] **Author delete own thread** (zero replies · configurable window)
- [x] **Compact mod gear** on threads/posts (E5 partial)
- [x] Markdown → sanitized HTML · image upload (WebP, EXIF strip, thumbs) · lightbox gallery
- [x] **Full-text search** — `/search` · `GET /v1/search`
- [x] **What's New** — `/whats-new` · unread filter · nav badge (H8)

### Auth & members
- [x] Registration · password login · magic link · email verification
- [x] Passkeys (WebAuthn) — code complete; *parked on Doug's Chrome/GPM setup*
- [x] Password reset · session cookie · public profiles `/u/{name}`
- [x] User states (valid/discouraged/banned) · login throttling · flood limits
- [x] Display names case-insensitive · reserved names anti-impersonation (K19)
- [x] Warning system (3 strikes → 7-day ban) · mod/admin ban tiers · perm_ban permission
- [x] Avatar upload (C10) — square crop · WebP · profile hover
- [x] **Sign out all other devices** (C8)

### Access control
- [x] Per-forum `access_level` (public, members, staff, admin, supporters, sponsors)
- [x] `list_public` toggle — locked row vs completely hidden on `/community`
- [x] Manual supporter/sponsor entitlements · staff/admin stealth rooms
- [x] Session-aware API on community/category pages (staff rooms visible when signed in)

### Moderation & admin
- [x] Report post · **report thread (E3)** · mod queue `/mod/reports` · **configurable report reasons (E4)** in `site.json`
- [x] **Post approval queue** `/mod/queue` — classifieds forums moderated (E2) · **spam hold + score (E9)**
- [x] Admin panel `/admin` — overview, users, categories CRUD (section delete, multi-group access), ban/restore/warn, member groups editor
- [x] **Site settings UI (K2)** — `/admin/settings` — posts, guests, email, spam, new members, report reasons, reserved names, trust/DMCA
- [x] Post author IP audit · email verification resend

### SEO, legal, marketing
- [x] Sitemap · robots.txt · canonical · OG/Twitter · JSON-LD (DiscussionForumPosting + FAQPage)
- [x] `/meta.json` per thread · Terms · Privacy · home hero (“we’re back”)
- [x] **Powered by Xuroi** footer (engine-injected)
- [x] `/abuse` · DMCA designated agent (N5)

### Email
- [x] SES + log mailer · styled auth/notification templates · unsubscribe
- [x] @mention emails (I2) — queued via `cmd/notify`
- [x] **Notification preferences (I5)** — `/settings/email` · thread reply + @mention email toggles
- [x] **Deliverability ops doc (I11)** — `docs/email-deliverability.md`

### Notifications (P1)
- [x] @mentions in posts/threads (B23) — `@slug`, `@"Name"`, `@[Name]` → profile links
- [x] In-app notification feed (I4) — bell badge in nav · `/notifications` · mark read

### Private messaging (P1)
- [x] **1:1 DMs (D1)** — `/messages` inbox · `/messages/{id}` thread · profile Message button
- [x] **DM privacy (D7)** — everyone / friends_only / off · profile settings · `dm_privacy` on `/v1/auth/me`
- [x] Migration **024** · `GET/POST /v1/dm/conversations` · send/read · in-app `dm_message` notifications

### Theme / tooling
- [x] Theme validator CLI (`cmd/themevalidate`) — contract check (J4 partial)
- [x] Astro PutterTalk theme = production UI (J2 Mustache deferred)

---

## Partial / parked

| Item | Status |
|---|---|
| Mustache theme renderer (J2) | **Deferred** — Astro mockup theme is production UI for launch |
| LLM thread summaries (A1) | Heuristic v1 live; LLM via env API key optional |
| Passkeys (C3) | Built; blocked on Doug's local passkey provider |
| TOTP 2FA (C6) | **Deferred** — next batch stretch |
| Inline mod tools (E5) | Gear popover shipped; more inline actions TBD |
| Redis cache / job workers (M3–M4) | In-memory limiter v1; Redis wired in compose, not used yet |
| S3/CDN (G5, M5, M8) | Local uploads; Cloudflare at cutover |
| Stripe/Patreon entitlements (L8) | Manual grants only; webhook stub |
| A12 CDN read/write split | Architectural — at hosting cutover |

---

## Not started (P0 ops — blocking launch)

- [ ] **puttertalk.com DNS + Cloudflare** (P7)
- [ ] **SES production approval** (ticket submitted)
- [ ] SSL/TLS at cutover (M9)
- [ ] G5/M5/M8 CDN + object storage at cutover

---

## Shipped (P1 Batch 5 — 11 items)

- [x] **P3** — WITB photo-first UX
- [x] **C8** — Logout all devices
- [x] **E12** — Duplicate post detection
- [x] **B16** — Merge threads
- [x] **E9** — Spam scoring verify
- [x] **H8** — What's New / unread polish
- [x] **N5** — DMCA takedown note
- [x] **B10** — Author soft-delete thread
- [x] **G9** — Guest attachment enforcement
- [x] **I11** — Email deliverability doc
- [x] **M12** — Structured request logging

Migration **035**

---

## PutterTalk categories (seeded)

7 sections · 22 forums — General Putter Talk (6) · Popular Manufacturers (7) · Manufacturer Specific (2) · Classifieds (3) · Website Announcements (2) · Supporter Areas (2) · Staff (2)

**Moderated forums:** Free Classifieds · Wanted/Trade · eBay Items (`post_moderation: true`)

---

## Workers to run in dev

```bash
cd xuroi/api
go run ./cmd/searchindex          # FTS queue (or --rebuild once)
go run ./cmd/notify               # email digests
go run ./cmd/intelligence         # thread summaries
```

---

## For the next session

1. Read `../.grok/session/CHANGELOG.md` first
2. Read this file + **WISH-LIST.md** P1 section
3. Default next work: **C6 TOTP 2FA** · remaining P1 · P0 ops (P7 DNS)
4. **Admin rule:** every setting gets an Admin section at ship time
5. End of session: update CHANGELOG + this file
6. Execute yourself — run commands, don't just instruct Doug
7. Backups are critical (2019 data loss)

---

*Update this file at end of every work session.*