# Xuroi Master Wish List

**Purpose:** Everything we will eventually need — mined from XenForo, phpBB, SMF archaeology + Xuroi-only ambitions. Work **backwards** from this list: PutterTalk launch pecks off P0, then P1, etc.

**Product:** Xuroi · **Flagship:** PutterTalk · **Distribution:** `Powered by Xuroi` on every public page (locked)

**Last updated:** July 2, 2026 — P0 code sweep: search, mod queue, thread delete

---

## How to read this doc

| Priority | Meaning |
|---|---|
| **P0** | PutterTalk launch — cannot go live without |
| **P1** | First 90 days post-launch — operators feel the pain without these |
| **P2** | Second forum / scale — Doug runs another site; exposure play |
| **P3** | Xuroi as product — other communities adopt |
| **P4** | Someday / maybe — don't build until demanded |

**Status column (left):**

| Symbol | Meaning |
|---|---|
| **✓** | Shipped in dev |
| **partial** | Started; known gaps remain |
| **scaffolded** | Foundation only — not production-ready |
| **parked** | Built; blocked on external factor |
| **—** | Not started |

**Tags:** `[XF]` XenForo has it · `[PB]` phpBB · `[SMF]` SMF · `[XU]` Xuroi-only differentiator

---

## Audit snapshot (July 2, 2026)

Full pass against `PROJECT-STATE.md`, changelog, and codebase. **Status is always the leftmost column** so you can scan done vs remaining.

| Status | ~Count | Highlights |
|---|---|---|
| **✓** | ~73 | Core forum (B1–B6, B7–B9, B13–B14, B24, B28), auth (C1–C5, C7, C9, C35–C36), media (G1, G3–G4, G10), SEO (H10–H11, H13, H15), admin (K1, K3–K5, K9, K19), platform (M1–M2, M6–M7), notifications (I1–I2, I4, I8–I9), avatars (C10), legal (N1–N2), moderation (E3, E4), PutterTalk seed (P1, P5) |
| **partial** | ~25 | A1/A8/A10/A11, B10, B35/B38, C3 (passkeys parked), E1/E5/E7, F1/F2, G2, H16, I7, J1/J3/J7/J10, K2, L1, M3 |
| **—** | ~176 | CDN/ops cutover (G5, M5, M8–M9, P7), group PM (D2) |
| **P0 remaining** | ~12 | Mostly ops: H7 async search, A12 CDN split, G5 CDN, M5/M8/M9, P7 DNS, J2 Mustache renderer (deferred — Astro is launch UI), A10 validator CLI |

**Fixes this audit:** B1–B6 marked shipped · I1/I8/I9 marked shipped · F3/F8/F9 staff rooms · E20 IP audit · J9 dark mode · C18 display names · D1 bumped to P1 · duplicate K19 → K25 · N/O/P status typos corrected

---

## A. Xuroi Differentiators (the room we make)

These are why we exist. Incumbents don't have them natively.

| Status | # | Item | Pri | Notes |
|---|---|---|---|---|
| partial | A1 | Thread intelligence — auto summary | P0 | Heuristic v1 on thread + `/meta.json`; LLM optional via env |
| ✓ | A2 | `/meta.json` per thread | P0 | Summary, participants, model_version live |
| ✓ | A3 | `/llm.txt` per thread | P1 | Plain-text digest for crawlers |
| ✓ | A4 | JSON-LD beyond snippet — FAQPage when Q&A | P0 | Heuristic question titles → FAQPage schema |
| — | A5 | Semantic search (pgvector) | P1 | "face balanced fast greens" finds right threads |
| — | A6 | Agent actor type + disclosure badge | P1 | Not human accounts pretending |
| — | A7 | Agent API — scoped tokens | P1 | read, draft, post, summarize, flag |
| partial | A8 | Media pipeline — WebP/AVIF + CDN | P0 | WebP upload + thumbs live; CDN at cutover |
| — | A9 | Image auto alt-text + semantic tags | P1 | SEO + accessibility |
| partial | A10 | Theme contract — schema + fixtures + validator | P0 | `theme-contract.json` + fixtures + THEMING.md; CLI validator pending |
| partial | A11 | Event-sourced posts — full edit history | P0 | Revisions stored + UI overlay on thread pages |
| — | A12 | Read/write path split — CDN SSR public | P0 | SEO + performance moat |
| — | A13 | Crawl-budget-aware sitemaps | P1 | Priority by intelligence + activity |
| — | A14 | Entity extraction in threads (gear, brands) | P2 | "Odyssey", "Scotty Cameron" as first-class |
| — | A15 | Consensus / accepted answer signal | P1 | Manual pin v1; auto v2 |
| ✓ | A16 | `Powered by Xuroi` footer + link | P0 | **LOCKED** — every rendered public page |
| — | A17 | MCP / tool surface for agents | P2 | `get_thread_context`, `search_semantic` |
| — | A18 | Per-thread machine export (cite-friendly) | P2 | For LLMs quoting community knowledge |

---

## B. Core Content (forums 101)

| Status | # | Item | Pri | Incumbent | Notes |
|---|---|---|---|---|---|
| ✓ | B1 | Categories / nodes | P0 | All | Nested groups (7 sections · 22 forums); admin CRUD + drag reorder |
| ✓ | B2 | Threads (topics) | P0 | All | |
| ✓ | B3 | Posts (replies) | P0 | All | |
| ✓ | B4 | Thread pagination | P0 | All | |
| ✓ | B5 | Create thread | P0 | All | Markdown composer + image upload |
| ✓ | B6 | Reply to thread | P0 | All | Quote + reactions |
| ✓ | B7 | Quote post | P0 | All | Reference by post ID |
| ✓ | B8 | Edit own post | P0 | All | `site.json` `posts.edit_*`; PutterTalk 30 min |
| ✓ | B9 | Edit history / revisions | P0 | [XF] | Clickable Edited stamp → revision overlay |
| ✓ | B10 | Soft delete post/thread | P0 | All | Post + staff thread delete (`DELETE /v1/threads/{id}`) |
| — | B11 | Hard delete (mod) | P1 | All | |
| — | B12 | Restore deleted (mod) | P1 | [XF] | |
| ✓ | B13 | Sticky / pin thread | P0 | All | Admin bar on thread page |
| ✓ | B14 | Lock / unlock thread | P0 | All | Blocks replies when locked |
| ✓ | B15 | Move thread (category) | P1 | All | Mod gear picker · API + UI |
| — | B16 | Merge threads | P2 | All | |
| — | B17 | Split thread | P2 | [XF] | |
| — | B18 | Thread prefixes / labels | P2 | [XF] | "FS", "WTT", "[SOLD]" for BST |
| — | B19 | Polls | P3 | All | |
| — | B20 | Thread types (question, article, suggestion) | P2 | [XF] | Question → solution flow |
| — | B21 | Drafts (save post before submit) | P1 | All | |
| ✓ | B22 | Post preview | P1 | All | Reply composer Preview toggle |
| ✓ | B23 | @mentions | P1 | [XF][SMF] | `@slug`, `@"Name"`, `@[Name]` → profile links + notify |
| ✓ | B24 | Markdown authoring → sanitized HTML | P0 | | goldmark + bluemonday UGC |
| — | B25 | Link unfurl / embed preview | P2 | [XF] | |
| — | B26 | Spoilers / collapsible sections | P2 | All | |
| — | B27 | Code blocks with syntax highlight | P2 | All | Low priority for PutterTalk |
| ✓ | B28 | Post reactions (like) | P0 | All | Single type v1 + karma |
| — | B29 | Multi-reaction types | P3 | [XF] | |
| — | B30 | Content voting (up/down) | P3 | [XF] | |
| — | B31 | Thread tags (freeform) | P2 | [XF] | |
| — | B32 | Read time / word count on threads | P3 | — | Nice for SEO pages |
| — | B33 | Print-friendly thread view | P4 | [SMF] | |
| — | B34 | RSS/Atom feeds per category | P2 | All | |
| ✓ | B35 | Activity feed / recent posts | P1 | All | `/whats-new` · home recent · per-forum latest |
| ✓ | B36 | "What's new" / unread tracking | P1 | All | Unread filter on `/whats-new` · community unread counts |
| ✓ | B37 | Mark forum read | P1 | All | `POST /v1/categories/{slug}/read` · button on forum page |
| ✓ | B38 | Mark thread read | P1 | All | Badges on forum + whats-new · nav unread count |
| ✓ | B39 | Thread watch / subscribe | P1 | All | Watch/mute toggle on thread · `email_thread_mutes` |
| — | B40 | Category watch | P2 | All | |
| ✓ | B41 | Thread view counter | P1 | All | Member dedup 6h · `view_count` on thread |
| — | B42 | Similar / related threads | P2 | [XU] | Semantic |
| — | B43 | Featured / spotlight threads | P2 | [XF] | Homepage curation |
| — | B44 | Scheduled publish (thread) | P4 | [XF] | |
| — | B45 | Redirect thread (URL move) | P2 | [SMF] | |
| — | B46 | Anonymous posting | P4 | [SMF] | Probably never |
| — | B47 | Post icons / emoticon per thread | P4 | [SMF] | |

---

## C. Users & Identity

| Status | # | Item | Pri | Incumbent | Notes |
|---|---|---|---|---|---|
| ✓ | C1 | Registration | P0 | All | Password + passkey |
| ✓ | C2 | Email verification | P0 | All | 48h link; blocks posting until confirmed |
| parked | C3 | Passkeys (WebAuthn) | P0 | [XF] | Code complete; blocked on Doug's Chrome/GPM setup |
| ✓ | C4 | Email magic link login | P0 | — | Emergency parachute on /join; 15 min single-use; log/SES |
| ✓ | C5 | Password login | P0 | All | bcrypt; password reset flow |
| — | C6 | TOTP 2FA | P1 | All | Mods + admins minimum |
| ✓ | C7 | Session management | P0 | All | 30-day cookie + X-Session-Token |
| — | C8 | Logout all devices | P2 | [XF] | |
| ✓ | C9 | User profiles (public) | P0 | All | `/u/{name}` karma + post count |
| ✓ | C10 | Avatar upload | P1 | All | Square crop · 256 + 64 WebP · Discord-style profile hover |
| — | C11 | Profile banner | P3 | [XF] | |
| — | C12 | Custom title | P2 | [XF] | |
| — | C13 | Signature | P2 | All | Markdown, length limit |
| ✓ | C14 | About / bio field | P1 | All | Profile bio · `PATCH /v1/me/profile` |
| — | C15 | Location, website fields | P3 | All | |
| — | C16 | Custom profile fields | P3 | All | |
| — | C17 | Username change | P2 | [XF] | With mod approval option |
| ✓ | C18 | Display name vs username | P3 | — | Display name is primary; slug from name; case-insensitive unique |
| — | C19 | User groups / roles beyond 5 | P2 | All | v1: Guest/Member/Mod/Admin/Agent |
| — | C20 | Ranks (post count titles) | P3 | [PB] | "200+ posts" |
| — | C21 | Online now / last seen | P1 | All | Privacy-aware |
| — | C22 | Member list | P2 | All | |
| — | C23 | User search | P2 | All | |
| — | C24 | Ignore / block user | P2 | [PB] | |
| — | C25 | Follow user | P3 | [XF] | |
| — | C26 | Privacy settings | P2 | [XF] | Profile visibility |
| — | C27 | GDPR data export | P1 | [XF] | |
| — | C28 | Account deletion | P1 | [XF] | |
| — | C29 | COPPA / age gate | P3 | [SMF] | If needed |
| — | C30 | Registration approval queue | P2 | [XF] | |
| — | C31 | Username denylist | P1 | [PB] | |
| — | C32 | Email denylist / disposable block | P1 | — | |
| — | C33 | Connected accounts (Google, Apple) | P2 | [XF] | |
| — | C34 | OAuth2 provider (login with Xuroi) | P3 | [XF] | For product phase |
| ✓ | C35 | User state: banned, discouraged, valid | P0 | All | `actors.state`; temp ban via `banned_until` |
| ✓ | C36 | Login attempt throttling | P0 | [PB] | IP + per-email failed-login limits |
| — | C37 | Remember me | P1 | All | 30-day session default; no explicit toggle |

---

## D. Private Messaging & Social

| Status | # | Item | Pri | Incumbent | Notes |
|---|---|---|---|---|---|
| ✓ | D1 | Private messages (1:1) | P1 | All | `/messages` inbox + thread · API + migration 024 |
| — | D2 | Group conversations | P3 | [XF] | |
| — | D3 | PM attachments | P3 | All | |
| — | D4 | PM rules / filters | P4 | [PB] | |
| ✓ | D7 | PM privacy setting (everyone / friends-only / off) | P1 | — | Profile privacy form · `dm_privacy` on actors |
| — | D5 | Profile posts / wall | P4 | [XF] | Skip |
| — | D6 | Social feed (non-forum) | P4 | [XF] | Not a social network |

---

## E. Moderation & Safety

| Status | # | Item | Pri | Incumbent | Notes |
|---|---|---|---|---|---|
| partial | E1 | Moderator role | P0 | All | `moderator_emails` in site.json; admin superset |
| ✓ | E2 | Mod queue (approve posts) | P0 | All | `/mod/queue` · classifieds `post_moderation` · approve/reject API |
| ✓ | E3 | Report post / thread | P0 | All | Post + thread report · `thread_reports` (026) · flag in thread header · unified `/mod/reports` |
| ✓ | E4 | Report reasons (configurable) | P1 | [PB] | `site.json` `moderation.report_reasons` · picker UI on report · `GET /v1/moderation/report-reasons` |
| partial | E5 | Inline mod tools | P1 | [XF] | Compact gear popover on thread (pin/lock/reports/queue/delete) + post (audit/warn/remove) |
| ✓ | E6 | Ban user (temp / perm) | P0 | All | Admin UI; clears sessions |
| partial | E7 | IP ban | P1 | All | Post IPs banned with account; cleared on restore |
| — | E8 | Email ban | P1 | All | |
| — | E9 | Spam scoring (basic) | P1 | [XF] | Rate, links, new account |
| — | E10 | Spam scoring (ML / Akismet) | P2 | [XF] | |
| ✓ | E11 | Flood control / rate limits | P0 | All | Post + thread + login limits; in-memory v1 |
| — | E12 | Duplicate post detection | P2 | — | |
| — | E13 | Word censor / filter | P2 | All | |
| — | E14 | Link domain allow/deny | P2 | — | |
| ✓ | E15 | Force nofollow on user links | P1 | [XF] | `seo.nofollow_user_links` in Admin settings |
| ✓ | E16 | Warn user | P2 | [XF] | 8h red border overlay; 3 strikes → 7-day auto-ban |
| — | E17 | Warning points / expiry | P3 | [XF] | |
| — | E18 | Mod log (audit trail) | P1 | All | Event log helps |
| — | E19 | Admin log | P1 | [XF] | |
| ✓ | E20 | View IPs (mod) | P1 | All | Admin post audit (`author_ip`) |
| — | E21 | Thread reply ban | P2 | [XF] | |
| — | E22 | Discourage mode (shadow throttle) | P3 | [XF] | |
| — | E23 | Clean spammer (bulk delete) | P2 | [XF] | |
| — | E24 | DMCA / abuse contact flow | P1 | — | Legal |
| — | E25 | Cookie consent | P2 | [XF] | If EU traffic |
| — | E26 | Moderated categories (pre-approval) | P2 | All | BST candidate |
| — | E27 | Edit post by mod | P1 | All | With "edited by mod" |
| — | E28 | Lock reason visible | P1 | — | |
| — | E29 | Appeal flow | P4 | — | |

---

## F. Permissions & Access

| Status | # | Item | Pri | Incumbent | Notes |
|---|---|---|---|---|---|
| partial | F1 | 5-role model (Guest/Member/Mod/Admin/Agent) | P0 | — | Guest implicit; mod/admin via site.json |
| partial | F2 | Category-level permission overrides | P2 | All | `access_levels[]` multi-group OR logic · admin checkbox picker · member groups on Users |
| ✓ | F3 | Private categories | P2 | All | Locked rows + 403; staff rooms stealth; supporter entitlements |
| ✓ | F4 | Guest read-only mode | P0 | All | `guests.read_only` in site.json; sign-in to post |
| ✓ | F5 | Guest can't attach | P0 | All | Upload requires verified session |
| — | F6 | Permission to view attachments | P1 | [XF] | |
| — | F7 | New user restrictions (links, PM) | P1 | — | Anti-spam |
| ✓ | F8 | Staff-only areas | P2 | All | Staff + admin rooms in seed |
| ✓ | F9 | Node tree (nested categories) | P2 | [XF] | Section groups + forums; drag reorder admin |
| — | F10 | Per-user permission override | P3 | [XF] | |
| — | F11 | Permission audit / debug view | P3 | — | Admin |

---

## G. Media & Attachments

| Status | # | Item | Pri | Incumbent | Notes |
|---|---|---|---|---|---|
| ✓ | G1 | Image upload on post | P0 | All | WebP in `infra/uploads/`, editor Image btn |
| partial | G2 | WebP/AVIF conversion | P0 | [XU] | WebP on upload; AVIF later |
| ✓ | G3 | Thumbnails / responsive sizes | P0 | [XU] | `_thumb.webp` on upload + lightbox |
| ✓ | G4 | EXIF strip | P0 | [XU] | Decode → WebP re-encode drops metadata |
| — | G5 | CDN delivery | P0 | [XU] | Local uploads; Cloudflare at cutover |
| — | G6 | Alt-text (manual + auto) | P1 | [XU] | |
| — | G7 | File attachments (PDF, etc.) | P2 | All | |
| — | G8 | Attachment quota per user | P2 | All | |
| — | G9 | Attachment permissions | P1 | [XF] | |
| ✓ | G10 | Inline image display | P0 | All | Inline in posts + lightbox viewer |
| ✓ | G11 | Image lightbox | P1 | — | Click post image; prev/next + arrow keys |
| — | G12 | Video embed (YouTube, etc.) | P2 | [XF] | oEmbed |
| — | G13 | Video upload | P4 | [XF] | Expensive |
| — | G14 | Image moderation (NSFW flag) | P2 | — | |
| — | G15 | Hotlink protection | P2 | All | |
| — | G16 | Storage quota / cleanup | P2 | — | |

---

## H. Search & Discovery

| Status | # | Item | Pri | Incumbent | Notes |
|---|---|---|---|---|---|
| ✓ | H1 | Full-text search (posts + threads) | P1 | All | Postgres FTS · `/v1/search` · `/search` |
| — | H2 | Semantic / vector search | P1 | [XU] | pgvector |
| — | H3 | Search by author | P1 | All | |
| — | H4 | Search in category | P1 | All | |
| — | H5 | Search filters (date, has image) | P2 | [XF] | |
| — | H6 | Search result highlighting | P1 | All | |
| ✓ | H7 | Async search indexing | P0 | [XF] jobs | `search_index_queue` + `cmd/searchindex` |
| — | H8 | Find new / unread | P1 | All | |
| — | H9 | Trending threads | P2 | — | |
| ✓ | H10 | Sitemap XML | P0 | [XF] | `/sitemap.xml` from API thread index |
| ✓ | H11 | robots.txt | P0 | All | `/robots.txt`; disallows /admin, /mod |
| — | H12 | Meta robots per content | P1 | [XF] | noindex moderated |
| ✓ | H13 | Canonical URLs | P0 | [XF] | `<link rel="canonical">` in Layout |
| — | H14 | Crawler / bot detection | P2 | [SMF] | Not SMF's log_spider_hits |
| ✓ | H15 | Open Graph + Twitter cards | P0 | [XF] | Layout meta; thread `og:type=article` |
| ✓ | H16 | Structured data (JSON-LD) | P0 | [XF] | DiscussionForumPosting + FAQPage for question threads |

---

## I. Notifications & Email

| Status | # | Item | Pri | Incumbent | Notes |
|---|---|---|---|---|---|
| ✓ | I1 | Email on reply (watched thread) | P1 | All | Debounced `email_notification_queue` + `cmd/notify`; participants auto-included |
| ✓ | I2 | Email on @mention | P1 | All | `email_mention_queue` + notify worker |
| — | I3 | Email digest (daily/weekly) | P2 | [SMF] | |
| ✓ | I4 | In-app notification feed | P1 | [XF] | Bell · `/notifications` · mentions + thread_reply; auto-clear on view |
| ✓ | I5 | Notification preferences | P1 | [XF] | `/settings/email` · thread replies + @mentions toggles · `GET/PATCH /v1/me/email-preferences` |
| — | I6 | Push notifications (web push) | P3 | [XF] | |
| partial | I7 | Email queue / retry | P1 | All | Queue tables + worker; retry/backoff basic |
| ✓ | I8 | Transactional email templates | P1 | — | thread_reply, magic_link, password_reset, mention |
| ✓ | I9 | Unsubscribe links | P1 | — | `/email/unsubscribe` + per-thread mutes |
| — | I10 | Admin notices / announcements | P2 | [XF] | |
| — | I11 | Email deliverability (SPF/DKIM docs) | P1 | — | Ops doc |

---

## J. Themes & Presentation

| Status | # | Item | Pri | Incumbent | Notes |
|---|---|---|---|---|---|
| scaffolded | J1 | Theme contract (schema + fixtures) | P0 | [XU] | JSON schema + fixtures; validator pending |
| — | J2 | Mustache template renderer | P0 | [XU] | **Deferred** — Astro is production UI for launch |
| partial | J3 | `tokens.css` design variables | P0 | [XU] | CSS vars in `pt.css`; separate tokens file TBD |
| partial | J4 | `forum theme validate` CLI | P0 | [XU] | `cmd/themevalidate` — validates contract; Mustache themes TBD |
| — | J5 | `forum theme preview` CLI | P1 | [XU] | |
| — | J6 | Reference themes (classic, bare) | P0 | [XU] | |
| partial | J7 | PutterTalk production theme | P0 | [XU] | Astro SSR theme live; HTML mockups in `themes/puttertalk/` |
| — | J8 | Theme hot-reload (dev) | P2 | — | |
| ✓ | J9 | Dark mode via tokens | P1 | — | `data-theme` light/dark + system preference |
| partial | J10 | Mobile-responsive (contract requirement) | P0 | — | Scrollable nav, compact header, footer staff links |
| ✓ | J11 | `Powered by Xuroi` footer partial | P0 | [XU] | **Engine-injected, not optional** |
| — | J12 | Theme import (zip / URL) | P2 | — | |
| — | J13 | Theme marketplace / directory | P3 | — | Optional curation |
| — | J14 | Per-site theme (multi-tenant) | P3 | — | |
| — | J15 | Email theme (separate contract) | P2 | — | |
| — | J16 | Smilies / emoji picker | P2 | All | Unicode-native OK |
| — | J17 | Custom CSS injection (admin) | P4 | — | Dangerous |

---

## K. Admin & Configuration

| Status | # | Item | Pri | Incumbent | Notes |
|---|---|---|---|---|---|
| ✓ | K1 | Admin panel | P0 | All | `/admin` overview + stats |
| partial | K2 | Site settings (title, description) | P0 | All | `site.json` source; read-only in admin |
| ✓ | K3 | Category management | P0 | All | Nested groups + forums; CRUD + drag reorder |
| ✓ | K4 | User management | P0 | All | `/admin/users` search + list |
| ✓ | K5 | Ban management | P0 | All | Ban / discourage / restore in admin UI |
| — | K6 | Agent / API key management | P1 | [XF] | |
| partial | K7 | Theme activation | P0 | [XU] | `default_theme` in site.json; Astro production UI |
| — | K8 | Backup / restore UI | P1 | — | **Critical for Doug** — script exists, no UI |
| ✓ | K9 | Automated backup schedule | P0 | | launchd 6h (Mac) + `backup.sh`; Linux cron in install script |
| — | K10 | Health dashboard | P2 | — | |
| — | K11 | Cron / job monitor | P2 | [XF] | |
| — | K12 | Error log viewer | P2 | All | |
| — | K13 | Statistics (posts/day, users) | P2 | All | |
| — | K14 | Help pages / FAQ CMS | P2 | [XF] | |
| — | K15 | Navigation editor | P3 | [XF] | |
| — | K16 | Notice system (banners) | P2 | [XF] | |
| — | K17 | Maintenance mode | P1 | All | |
| — | K18 | Force agreement (TOS) | P2 | [XF] | |
| ✓ | K19 | Reserved display names (anti-impersonation) | P0 | — | `reserved_display_names` in site.json |
| — | K25 | Contact form | P2 | [XF] | Was duplicate K19 |
| — | K20 | Multi-language / i18n | P4 | All | English v1 |
| — | K21 | Timezone handling | P1 | All | |
| — | K22 | Import tool (XenForo) | P3 | [XF] | Others may want |
| — | K23 | Import tool (phpBB / SMF) | P4 | — | |
| — | K24 | Data export (full site) | P2 | — | |

---

## L. API & Integrations

| Status | # | Item | Pri | Incumbent | Notes |
|---|---|---|---|---|---|
| partial | L1 | REST API v1 | P1 | [XF] | Extensive `/v1/*`; session auth; no public API keys |
| — | L2 | API key auth | P1 | [XF] | |
| — | L3 | Scoped permissions | P1 | [XF] | Extend for agents |
| — | L4 | OAuth2 clients | P3 | [XF] | |
| — | L5 | Webhooks (post.created, etc.) | P2 | — | |
| — | L6 | RSS import | P4 | [XF] | |
| — | L7 | Zapier / n8n friendly docs | P3 | — | |
| — | L8 | Stripe (paid memberships) | P4 | [XF] | Manual entitlements v1 |
| — | L9 | Analytics hook (Plausible, etc.) | P2 | — | |
| — | L10 | SSO (SAML/OIDC) | P4 | — | Enterprise |

---

## M. Platform & Infrastructure

| Status | # | Item | Pri | Incumbent | Notes |
|---|---|---|---|---|---|
| ✓ | M1 | Event log (append-only) | P0 | [XU] | Source of truth |
| ✓ | M2 | Projection rebuild from events | P0 | [XU] | `POST /v1/admin/rebuild-projections` |
| partial | M3 | Background job workers | P0 | [XF] | `cmd/notify`, `cmd/intelligence`; not full worker service |
| — | M4 | Redis cache | P0 | — | Wired in compose; in-memory limiter v1 |
| — | M5 | Object storage (S3/R2) | P0 | — | Minio in compose; local uploads in dev |
| ✓ | M6 | Docker Compose local dev | P0 | | Postgres + Redis + Minio |
| ✓ | M7 | PostgreSQL 16+ | P0 | — | :5433 in dev |
| — | M8 | CDN (Cloudflare) | P0 | — | |
| — | M9 | SSL / TLS | P0 | — | |
| — | M10 | Monitoring (uptime, metrics) | P1 | — | |
| — | M11 | Error tracking (Sentry) | P1 | — | |
| — | M12 | Structured logging | P1 | — | |
| — | M13 | Load test to 10k concurrent read | P2 | — | Doug's historical scale |
| — | M14 | Projection rebuild at 1M+ posts | P2 | — | Design now |
| — | M15 | Multi-site / multi-tenant | P3 | — | Single site v1 |
| — | M16 | Installer / setup wizard | P3 | All | Doug installs by hand v1 |
| — | M17 | Upgrade / migration runner | P3 | All | |
| — | M18 | Horizontal scaling playbook | P3 | — | |
| — | M19 | Read replicas | P3 | — | |
| — | M20 | Geo-redundant backups | P1 | — | Learned the hard way |

---

## N. Legal, Compliance, Trust

| Status | # | Item | Pri | Notes |
|---|---|---|---|---|
| ✓ | N1 | Terms of service page | P0 | `/terms` |
| ✓ | N2 | Privacy policy page | P0 | `/privacy` |
| — | N3 | Cookie policy | P2 | |
| — | N4 | GDPR export + delete | P1 | |
| — | N5 | DMCA agent / takedown | P1 | |
| — | N6 | Age verification (if needed) | P4 | |
| — | N7 | Content retention policy | P2 | |
| — | N8 | Public mod transparency report | P4 | |

---

## O. Product & Growth (Xuroi as platform)

| Status | # | Item | Pri | Notes |
|---|---|---|---|---|
| ✓ | O1 | `Powered by Xuroi` footer link → xuroi.com | P0 | **LOCKED** |
| — | O2 | xuroi.com marketing site | P2 | After PutterTalk live |
| — | O3 | Second Doug forum (exposure) | P2 | Dogfood × 2 |
| — | O4 | Install docs for self-hosters | P3 | |
| — | O5 | License decision (OSS / commercial) | P3 | |
| — | O6 | Theme gallery (community submitted) | P3 | |
| — | O7 | Hosted Xuroi (SaaS) | P4 | |
| — | O8 | Comparison page vs XenForo | P3 | SEO play |
| — | O9 | Public roadmap | P2 | |
| — | O10 | Changelog / release notes | P2 | |

---

## P. PutterTalk-Specific

| Status | # | Item | Pri | Notes |
|---|---|---|---|---|
| ✓ | P1 | Category set (see site.json) | P0 | 7 sections · 22 forums seeded |
| — | P2 | BST category rules + pinned FAQ | P1 | Buy/sell/trade |
| — | P3 | WITB photo-first UX | P1 | Image pipeline priority |
| — | P4 | Gear entity tags (Odyssey, Scotty, etc.) | P2 | |
| ✓ | P5 | "We're back" landing narrative | P0 | Home hero — greenfield relaunch, no archive pretense |
| — | P6 | Founding member closed beta | P1 | 2 weeks pre-public |
| — | P7 | puttertalk.com DNS + Cloudflare | P0 | Ops blocker |
| — | P8 | Welcome-back email to old list (if exists) | P1 | |
| — | P9 | Agent: spec comparison bot (disclosed) | P2 | Putter use case |

---

## Working backwards — pecking order

### Phase 0–1: Skeleton that holds data (weeks 1–8)
Peck: M1, M2, M3, M4, M5, M6, M7, B1–B8, B13–B14, B24, B28, C1–C2, C3–C4, C7, C9, C35–C36, F1, F4–F5, E1, E6, E11, K1–K5, K9, N1–N2

### Phase 2: World can read it (weeks 9–12)
Peck: A1–A2, A4, A10–A12, A16, B4, G1, G10, H7, H10–H11, H13, H15–H16, J1–J7, J10–J11, P5, P7, O1

### Phase 3: World can find it (weeks 13–16)
Peck: A5, A8–A9, G2–G5, H1–H2, H6, B35, P3

### Phase 4: World can understand it (weeks 17–20)
Peck: A3, A13–A15, A14, B42, H9

### Phase 5: Community runs itself (weeks 21–24)
Peck: A6–A7, C10, **D1+D7**, **E3** ✓, E5 polish, E9, I1–I2, I4–I5, K6, L1–L3, P6

### Phase 6: PutterTalk live (weeks 25–28)
Peck: Everything remaining P0, M10–M11, M20, N4–N5, P1–P2, P8, launch checklist

### Post-launch P1 sweep (days 1–90)
All P1 items not yet done — especially **D1/D7**, I5, B21, E4, F7, G6, K8, M10

---

## What we deliberately skip (incumbent bloat)

Learned from 20 years of accretion — **do not build unless P3+ demand:**

- XenForo 118-permission matrix → 5 roles + simple overrides
- Addon hook / class extension system → API + theme contract
- Profile posts / social feed → not a social network
- Calendar, portal CMS, gallery, blog, classifieds engine → categories cover BST
- Payment / subscription system → Stripe later
- Multi-language i18n → English until someone pays
- SMF-style 24 log_* tables → structured logging + event log
- Video upload hosting → embed only
- Template modification XML → never

---

## Counts (sanity check)

| Priority | Items | ✓ shipped | partial+ |
|---|---|---|---|
| P0 | ~75 | ~52 | ~12 |
| P1 | ~65 | ~12 | ~6 |
| P2 | ~70 | ~2 | ~4 |
| P3 | ~45 | 0 | 0 |
| P4 | ~25 | 0 | 0 |
| **Total** | **~280** | **~72** | **~28** |

PutterTalk launch = **~75 P0 items** (~52 done, ~12 partial, ~11 not started — mostly ops/CDN/search).

---

## Cross-references

- [NEXT-GEN-FORUM-BATTLE-PLAN.md](../NEXT-GEN-FORUM-BATTLE-PLAN.md) — phases, architecture, archaeology
- [PROJECT-STATE.md](./PROJECT-STATE.md) — session continuity, current focus
- [sites/puttertalk/site.json](./sites/puttertalk/site.json) — launch config
- [theme-contract/](./theme-contract/) — designer handoff

---

*Update this file when items ship, priorities change, or new sessions add decisions.*