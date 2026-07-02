# Xuroi Theme Contract — Builder & AI Handoff

**Tagline:** Themes are folders. Here's the spec. Make it yours.

## For PutterTalk designers

PutterTalk is a golf putter community relaunching on Xuroi. The theme should feel **modern, premium, and unexpected** — not a 2010 phpBB clone. Think editorial clarity for long gear threads, excellent typography, fast image loading.

## What you receive

| File | Purpose |
|---|---|
| `theme-contract.json` | Page list, required regions, constraints |
| `fixtures/*.json` | Realistic sample data — design without running the engine |
| `schemas/*.json` | JSON Schema per page (coming Phase 0) |
| `reference-theme/bare/` | Minimal working theme to fork |

## What you deliver

```
themes/your-theme/
  theme.json
  tokens.css
  pages/*.html
  partials/*.html
  assets/
```

## Template syntax

Mustache. Examples:

```html
<h1>{{thread.title}}</h1>
{{#thread.summary}}<p class="summary">{{thread.summary}}</p>{{/thread.summary}}
{{#posts}}{{> post-card}}{{/posts}}
{{> pagination}}
```

## Rules

### Do

- Use `tokens.css` for all colors, fonts, radii, spacing
- Use semantic HTML (`article`, `nav`, `time`)
- Design for long reading sessions (gear comparison threads)
- Show agent posts distinctly when `author.is_agent` is true

### Don't

- Add `<script>` tags
- Remove `{{> pagination}}` or required post regions
- Put permission logic in templates — use `ui.show_mod_bar` flags only
- Duplicate SEO `<head>` content — engine injects it

## Engine injects (don't duplicate)

- `<title>`, meta description, canonical
- JSON-LD (`DiscussionForumPosting`, `FAQPage`)
- Open Graph / Twitter cards
- Analytics (if configured)

## AI design prompt (copy/paste)

```
You are designing a theme for Xuroi, a modern forum engine.

Attach:
- theme-contract/theme-contract.json
- theme-contract/fixtures/thread.json
- theme-contract/fixtures/category.json

Task: Create a complete theme folder for PutterTalk.com — a premium golf
putter community. Dark-friendly, clean typography, excellent readability
for long comparison threads. Output pages/ and partials/ as Mustache HTML
and tokens.css with CSS custom properties.

Constraints: No JavaScript. Mustache syntax only. Include all required
regions from theme-contract.json.
```

## Validate before deploy

```bash
forum theme validate ./themes/your-theme
forum theme preview ./themes/your-theme --page thread --fixture fixtures/thread.json
```