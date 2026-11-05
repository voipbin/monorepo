# bin-email-manager — Domain

## Domain Entities

### Email

An outgoing email with status tracking, provider reference, and optional attachments.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning customer |
| `source` | string | Sender email address |
| `destinations` | []string | Recipient email addresses |
| `subject` | string | Email subject line |
| `content` | string | Email body (HTML or plain text) |
| `status` | string | Delivery status (see lifecycle below) |
| `provider_type` | string | `mailgun` or `sendgrid`; empty until a provider has accepted the email |
| `provider_reference_id` | string | Provider's message reference for tracking |
| `attachments` | []Attachment | File attachments |
| `tm_delete` | timestamp | Soft-delete sentinel (`9999-01-01` = active) |

### Attachment

A file attachment linked to an email via a reference to `bin-storage-manager`.

| Field | Type | Description |
|-------|------|-------------|
| `reference_type` | string | `recording` — links to a call recording |
| `reference_id` | UUID | ID of the referenced resource (e.g. recording ID) |

## Key Business Rules

### Send Flow

1. Email record created in database with status `initiated`.
2. `email.EventTypeCreated` published.
3. Attachments resolved: `emailhandler` calls `bin-storage-manager` to fetch file URLs.
4. Provider failover loop attempts delivery:
   - Mailgun attempted first
   - SendGrid attempted if Mailgun fails
5. `provider_type` and `provider_reference_id` are stored together on the Email record after the first successful send. They stay empty if all providers fail.
6. The status moves on from `initiated` when a provider webhook reports it (Mailgun `accepted` event gives `processed`).

### Email Status Lifecycle

```
initiated
    └→ processed   (provider accepted the email)
           └→ delivered  (recipient's server confirmed receipt)
                  └→ open     (recipient opened the email)
                  └→ click    (recipient clicked a link)
           └→ bounce         (undeliverable)
           └→ dropped        (provider filtered)
           └→ deferred       (temporary delivery failure, will retry)
           └→ unsubscribe    (recipient unsubscribed)
           └→ spamreport     (recipient marked as spam)
```

Status transitions are driven by provider webhooks processed at `POST /v1/hooks`.

### Provider Failover

Provider selection iterates in order:
1. Mailgun (primary)
2. SendGrid (fallback)

If Mailgun returns an error, SendGrid is attempted immediately. The first successful provider and its reference ID are stored. If both fail, the email remains at `initiated` status.

### Mailgun Webhook Events

The body is either the event itself or an object with the event under `event-data`. The email is identified by the `voipbin_message_id` user variable (sent as `v:voipbin_message_id` when the email is sent), which Mailgun returns in `user-variables`.

| Mailgun event | Status |
|---------------|--------|
| `accepted` | `processed` |
| `delivered` | `delivered` |
| `failed`, severity `temporary` | `deferred` |
| `failed`, severity `permanent` | `bounce` |
| `opened` | `open` |
| `clicked` | `click` |
| `unsubscribed` | `unsubscribe` |
| `complained` | `spamreport` |

Other events, events without a valid email id, and events for unknown emails are ignored. `dropped` is produced only by the SendGrid webhook.

Events can arrive out of order and can be repeated, so a Mailgun event is applied only when its status is later than the current one in this order: `initiated`, `processed`, `deferred`, `delivered`, `bounce` (same level as `dropped`), `open`, `click`, `unsubscribe`, `spamreport`. The check and the update are one conditional statement, so concurrent updates cannot move the status back. A stale or repeated event changes nothing and publishes no `email_updated` event. Deleted emails are not updated.

### Webhook Processing

Provider webhooks arrive at `POST /v1/hooks` with query parameters identifying the source:
- `?uri=/v1/hooks/sendgrid` → SendGrid webhook handler
- `?uri=/v1/hooks/mailgun` → Mailgun webhook handler

Provider webhooks are forwarded by `bin-hook-manager` from external inbound endpoints:
- SendGrid: `https://hook.voipbin.net/v1.0/emails/sendgrid`
- Mailgun: `https://hook.voipbin.net/v1.0/emails/mailgun`

### Event Publishing

| Event | Trigger |
|-------|---------|
| `email.EventTypeCreated` | Email record created |
| `email.EventTypeDeleted` | Email soft-deleted |
| `email.EventTypeUpdated` | Email status updated (via webhook) |

Events are consumed by `bin-billing-manager` to charge accounts for email usage.

### Soft Deletes

Emails use `tm_delete` for soft deletes. Default active sentinel: `9999-01-01 00:00:00.000000`.

### Database Queries

Uses Squirrel SQL builder (not raw SQL):
```go
sq.Select("*").From("email_emails").Where(sq.Eq{"id": id}).Where(sq.Eq{"tm_delete": DefaultTimeStamp})
```
