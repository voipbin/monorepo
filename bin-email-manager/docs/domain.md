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
| `tm_delete` | timestamp | Soft-delete timestamp (`NULL` = active) |

### Attachment

A file attachment linked to an email via a reference to `bin-storage-manager`.

| Field | Type | Description |
|-------|------|-------------|
| `reference_type` | string | `recording` (a call recording) or `file` (a storage file of type `talk`, `media` or `recording`). Any other value, including empty, is refused. |
| `reference_id` | UUID | Recording ID for `recording`, storage file ID for `file` |

Rules checked by `Create` before anything is stored (`pkg/emailhandler/attachment.go`), with the limits as constants in that file:

- At most 5 attachments, no duplicate (type, id), and the sum of the original sizes (`Filesize` of the files; for a recording the sum of its files) at most 18 MiB (18 MiB itself passes).
- The attachment must belong to the customer of the email. A `file` must exist, not be deleted and have the customer id of the email; a `recording` is checked by listing its files with the customer id (empty list: not found; a list of 100 files is refused because it may be cut). A missing, deleted and foreign resource give the same answer (404 `ATTACHMENT_NOT_FOUND`).
- A `file` of type `rag` or of an empty type is refused (400 `ATTACHMENT_FILE_TYPE_INVALID`). Other refusals are 400 (`ATTACHMENT_REFERENCE_INVALID`, `ATTACHMENT_DUPLICATE`, `ATTACHMENT_TOO_MANY`, `ATTACHMENT_TOO_LARGE`, `ATTACHMENT_SIZE_INVALID`).
- All storage requests of the check share one deadline of 2 seconds. A storage error or the deadline gives 503 `ATTACHMENT_CHECK_UNAVAILABLE` and no email is created.
- The `bin-storage-manager` download request of a recording does not take a customer, so the check here is the only ownership check. The engines check the attachment again when they send (a file deleted after `Create` makes the send fail and the email stays `initiated`).

## Key Business Rules

### Send Flow

1. Email record created in database with status `initiated`.
2. `email.EventTypeCreated` published.
3. Attachments resolved by each provider engine when it sends: the attachment is checked again, then `emailhandler` calls `bin-storage-manager` to get a download URL (`StorageV1FileDownloadURIRefresh` for a `file`, `StorageV1RecordingGet` for a `recording`) and downloads the content. The attachment file name is the file name of the storage file without path parts (`<recording id>.zip` for a recording); a second attachment with the same name gets a `-1`, `-2` suffix.
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
