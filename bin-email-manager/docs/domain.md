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
| `provider_type` | string | `sendgrid` or `mailgun` |
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
   - SendGrid attempted first
   - Mailgun attempted if SendGrid fails
5. Provider reference ID stored on Email record.
6. Status updated to `processed` after successful provider acceptance.

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
1. SendGrid (primary)
2. Mailgun (fallback)

If SendGrid returns an error, Mailgun is attempted immediately. The first successful provider's reference ID is stored. If both fail, the email remains at `initiated` status and the error is returned to the caller.

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
