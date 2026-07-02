# Email deliverability — PutterTalk / Xuroi

Operational checklist for reliable transactional email (auth, notifications, digests).

## Provider

- **Amazon SES** in `us-east-1` (or your chosen region)
- Dev: log mailer when `EMAIL_PROVIDER=log`
- Production: set `EMAIL_PROVIDER=ses` with `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`

## DNS (at cutover)

Configure on the **sending domain** (e.g. `mail.puttertalk.com` or root domain per your SES setup):

| Record | Purpose |
|--------|---------|
| **SPF** | `v=spf1 include:amazonses.com ~all` on the envelope-from domain |
| **DKIM** | Three CNAMEs from SES console → `_domainkey` tokens |
| **DMARC** | `v=DMARC1; p=none; rua=mailto:postmaster@puttertalk.com` — tighten to `quarantine`/`reject` after monitoring |
| **MAIL FROM** | SES custom MAIL FROM subdomain (e.g. `bounce.mail.puttertalk.com`) with MX + SPF |

## SES account

1. Verify domain and from-address in SES
2. Request **production access** (out of sandbox)
3. Set bounce/complaint notifications → SNS or mailbox
4. Keep complaint rate under 0.1%

## Application settings

- `site.json` → `email.from_address`, `email.from_name`, `email.reply_to`
- Admin → **Email** section mirrors sender identity
- `List-Unsubscribe` on notification mail (implemented)
- `/settings/email` — member opt-out for thread reply and @mention mail

## Retry / queue

- `email_queue` table with exponential backoff (`cmd/notify`)
- Category-watch and thread-reply digests share the same retry path

## Monitoring

- Watch SES reputation dashboard
- Log mailer output in dev; CloudWatch or log drain in prod
- Bounce handling: disable sends to hard-bounced addresses (future enhancement)

## Pre-launch test

```bash
cd xuroi/api
go run ./cmd/emailtest --to you@example.com
```

Confirm: arrives in inbox (not spam), links work, From/Reply-To correct, unsubscribe footer present on notifications.