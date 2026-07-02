# Xuroi Project State

**Grok: read `.grok/session/CHANGELOG.md` first** — then this file. Living handoff doc — update every meaningful work session.

**Last updated:** July 2, 2026 — session-aware community/category API (staff rooms visible to admins)
**Session status:** Phase 0/1 · P0 code complete — ready for P1 (@mentions, bell, messaging)

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

## Key docs (read order)

1. **../.grok/session/CHANGELOG.md** — mandatory first read every session
2. **PROJECT-STATE.md** (this file) — where we are now
3. **WISH-LIST.md** — master backlog ~280 items, P0–P4 priorities
4. **../NEXT-GEN-FORUM-BATTLE-PLAN.md** — vision, archaeology, phases, theme contract
5. **../.grok/session/README.md** — handoff protocol
6. **theme-contract/THEMING.md** — designer/AI handoff
7. **sites/puttertalk/site.json** — PutterTalk categories + config

---

## Repo structure

```
Forum-Idea/
  NEXT-GEN-FORUM-BATTLE-PLAN.md
  xuroi/                          ← active codebase
    api/          (Go — skeleton running)
    web/          (Astro — SSR home/category/thread)
    worker/       (not started)
    theme-contract/  (scaffolded)
    themes/puttertalk/  (stub)
    sites/puttertalk/   (site.json seeded)
    infra/docker-compose.yml  (postgres, redis, minio)
  xenforo_.../    ← reference only
  phpBB3/         ← reference only
  smf_2-1-7_install/  ← reference only
```

---

## What's done

- [x] Strategic vision and battle plan
- [x] Archaeological review of XenForo, phpBB, SMF
- [x] Theme contract concept locked (Section 6 of battle plan)
- [x] Xuroi + PutterTalk naming and strategy locked
- [x] Master wish list (~280 items, priority tiers)
- [x] `xuroi/` scaffold — docker-compose, site config, theme contract, fixtures
- [x] PutterTalk thread + category fixtures (realistic gear data)
- [x] Grok session handoff — `.grok/session/`, `AGENTS.md`, CHANGELOG

---

## What's NOT done yet

- [x] Event schema spec (`docs/event-schema.md`)
- [x] Go API skeleton (`api/cmd/xuroi`)
- [x] Postgres migrations (`api/internal/db/migrations/001_initial.sql`)
- [x] Event sourcing primitives (append, project, rebuild)
- [x] Seed PutterTalk categories (`cmd/seed`)
- [x] Integration test + smoke test (Docker on :5433)
- [x] Auth stub (register/login/session via email — dev)
- [x] Read API (list categories, get thread/posts)
- [x] Astro public site (mockup theme, home/category/thread)
- [x] Reply composer (signed-in users post replies)
- [x] JSON-LD DiscussionForumPosting on thread pages
- [x] New thread composer (`/c/{slug}/new`)
- [x] Nav account chip (signed-in) + profile stub (`/u/{name}`)
- [x] `/meta.json` per thread (participants, summary, model_version)
- [x] Marketing home (`/`) + community index (`/community`)
- [x] Intelligence worker stub (heuristic summaries, `cmd/intelligence`)
- [x] Backup script (`infra/backup.sh` — manual; schedule TBD)
- [x] Quote post (B7) — reference by post ID, quote block in UI
- [x] Post reactions / likes (B28) — single type, event-sourced
- [x] Karma from likes received (self-likes excluded) — profile + post sidebar
- [x] Edit own post (B8) — site.json policy (30 min for PutterTalk), revisions via events
- [x] Post author IP capture + admin audit popup
- [x] Mod tools — pin/lock thread, soft delete post, edit history overlay
- [x] Delete policy (`delete_enabled` in site.json) + editable partial quotes with jump links
- [x] Quote excerpt validation (trim only, no invented text) + markdown sanitizer (B24)
- [x] Login throttling + post/thread flood control (in-memory limiter v1)
- [x] Image upload — WebP conversion, local `infra/uploads/`, editor Image btn + paste
- [x] Image lightbox on thread pages — full-res viewer, prev/next, arrow keys
- [x] Multiple images per post — gallery grid, multi-select/paste/drop
- [x] Report post (E3) — member reports + admin queue at `/mod/reports`
- [x] Reply redirect — scroll to new post at bottom of thread
- [x] Inline edit + delete on thread page (no reload)
- [x] Passkeys (WebAuthn) — signup, login, add on profile *(parked — Chrome/GPM issue on Doug's machine)*
- [x] Password login — registration + sign-in; legacy email stub for unmigrated accounts
- [x] Password reset — forgot-password flow with styled email (in-forum)
- [x] Magic link login — emergency parachute on /join; single-use 15 min links
- [x] Rich-text inline post edit — same editor as compose (formatting, images, paste/drop)
- [x] Email verification — register sends 48h link; posting blocked until confirmed
- [x] User states + ban management — valid/discouraged/banned; admin UI at `/admin/users`
- [x] SEO pack — sitemap, robots.txt, canonical, OG/Twitter, JSON-LD on threads
- [x] Terms + Privacy pages — `/terms`, `/privacy`; footer links wired
- [x] Admin panel (minimal) — `/admin` overview, user search, ban/restore
- [x] Category groups + forums — nested tree on `/community`; admin CRUD at `/admin/categories`
- [x] Authorized access rooms — per-forum `access_level`; supporter/sponsor entitlements; staff/admin stealth rooms
- [x] Warning system — 8h red overlay; 3 strikes → 7-day auto-ban; one warn per post; 24h incident window
- [x] Mod ban tiers — mods 7d timeout; admin 30d/perm; per-actor `perm_ban` permission; purge content on ban
- [ ] Theme renderer (Mustache) — deferred; Astro mockup theme is production UI for launch
- [x] LLM summaries — optional via env API key; heuristic-v1 fallback; `summary_label` in site.json
- [x] Automated backup schedule (launchd every 6h locally; `install-backup-schedule.sh`)

---

## Current phase

**Phase 0/1: P0 closed (code)**

Forum skeleton, auth, moderation, email digests, SEO, legal pages, and admin tools are live in dev.

**Next (P1):** @mentions, notification bell, private messaging — per prior architecture plan.

**Ops still on Doug/Simmons:** SES production approval (ticket submitted), puttertalk.com DNS + Cloudflare (P7), CDN/SSL at cutover.

---

## Session log

→ See **`.grok/session/CHANGELOG.md`** for full history.  
→ See **`.grok/session/notes/`** for detailed write-ups.

---

## Open questions

| # | Question |
|---|---|
| 1 | Hosting provider (Hetzner, Fly, bare VPS) |
| 2 | Xuroi license model (post-PutterTalk) |
| 3 | PutterTalk visual direction |
| 4 | Closed beta before public launch? (recommended: yes) |

---

## PutterTalk launch categories (seeded)

5 sections · 20 forums — General Putter Talk (6) · Popular & Supporting Manufacturers (7) · Manufacturer Specific (2) · Members Classifieds (3) · Website Announcements (2)

---

## For the next agent/session

1. Read `../.grok/session/CHANGELOG.md` **before anything else**
2. Read this file + WISH-LIST.md P0 section
3. Continue with event schema spec unless Doug directs otherwise
4. End of session: update CHANGELOG, PROJECT-STATE, notes if needed
5. Doug prefers execution over instructions — run commands, don't just suggest
6. Backups are emotionally and operationally critical (2019 data loss)

---

*Update this file at end of every work session.*