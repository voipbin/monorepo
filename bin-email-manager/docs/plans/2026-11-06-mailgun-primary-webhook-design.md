# Mailgun primary provider and webhook status handling (VOIP-1680)

## 1. Issue analysis

Checked against origin/main at the time of writing.

- `pkg/emailhandler/hook.go` `hookMailgun` only logs the payload (TODO). Mailgun delivery events never update the email status.
- `pkg/emailhandler/send.go` tries SendGrid first, then Mailgun.
- `pkg/emailhandler/email.go` `Create` stores `ProviderTypeSendgrid` regardless of the provider that sent the email. `models/email/main.go` has no Mailgun provider type.
- `bin-hook-manager/pkg/servicehandler/email.go` reads `r.Body` with `io.ReadAll` and forwards the bytes unchanged as `Hook.ReceivedData`. No hook-manager change is needed. 
- `provider_type` is not exposed: `WebhookMessage` has no provider fields, no OpenAPI path or RST doc describes it, and no code filters on `ProviderTypeSendgrid`. The column is `varchar(255) DEFAULT NULL`, so a new value or an empty string needs no schema change. Old replicas read the value as a plain string, so rolling deploy is safe.
- Consumers of the status: `bin-conversation-manager` subscribes to `email_updated` (`conversationhandler.EmailEventUpdated`) and maps delivered/open/click/unsubscribe/spamreport to message `done` and bounce/dropped to `failed`; other statuses are no-ops. Because Mailgun webhooks were not processed until now, messages of Mailgun-sent emails stayed `progressing`. After this change they reach a terminal state; this is the intended outcome and a visible behavior change. conversation-manager keeps the first terminal state, so a bounce reported after a delivered event updates the email but not the conversation message (known limitation, conversation-manager is not changed). Outside bin-email-manager, no code reads `provider_type` (grep over the monorepo Go sources and the OpenAPI/RST sources).
- User docs: the OpenAPI/RST email status values are unchanged and provider neutral, so no RST/OpenAPI change is needed. Service docs (`CLAUDE.md`, `docs/domain.md`, `architecture.md`, `dependencies.md`, `operations.md`, `README.md`) are updated for the order, status mapping and the stale 'processed after send' sentence.
- Not in scope (suggest only): processRequest ERROR log, attachments, retry queue, failed status, SendGrid removal, inbound.

## 2. Design (what and why)

### 2.1 Order and provider record
- `send.go`: providers = Mailgun, then SendGrid.
- Provider is unknown at create time, so `create` stores an empty `ProviderType` (new `ProviderTypeNone = ""`). After a successful send, one UPDATE sets `provider_type` and `provider_reference_id` together (`UpdateProvider`, replaces `UpdateProviderReferenceID`, which has no other caller). If all providers fail, or the provider update itself fails, the record keeps an empty provider (the failure is logged, as is done today for the reference id). The DB column is `DEFAULT NULL`; NULL and empty string are treated alike. The only reader is the `provider_type` list filter, which no caller uses with `sendgrid`.
- New constant `ProviderTypeMailgun = "mailgun"`.

### 2.2 Matching an event to an email (verified in Mailgun docs)
- Webhook payload `message.headers` carries only message-id (plus from/to/subject on some events). Custom MIME headers are not documented as returned, so `X-Voipbin-Message-Id` cannot be relied on.
- Mailgun docs: variables sent as `v:<name>` appear in events and webhooks under `user-variables`. `engine_mailgun.go` therefore also sets the variable `voipbin_message_id` (the header stays unchanged). This also removes the race between the webhook and the `provider_reference_id` update.
- Emails sent before this change have no variable. Their events are acknowledged and ignored (a message-id lookup is not added: events for them arrive before the webhook is registered, only late opens or clicks are lost).

### 2.3 Payload shape
Docs show the event object at top level, and the signing page describes `signature` posted alongside `event-data`. The parser accepts both: if `event-data` exists use it, otherwise treat the body as the event. `user-variables` is `[]` when empty and an object otherwise, so it is parsed as raw JSON.

### 2.4 Event to status mapping (Mailgun webhook payload event)
| Mailgun event | Status |
|---|---|
| accepted | processed |
| delivered | delivered |
| failed, severity=temporary | deferred |
| failed, severity=permanent | bounce |
| opened | open |
| clicked | click |
| unsubscribed | unsubscribe |
| complained | spamreport |
| anything else (rejected, stored, list events, failed with unknown severity) | ignored, debug log |

The webhook configuration names the failure events `temporary_fail` and `permanent_fail`, while the delivered payload has `event=failed` plus `severity`; the mapping uses the payload fields.

`dropped` is not produced by the Mailgun path: suppression drops are delivered as permanent failures and the documented examples do not give a closed list of reason values, so they map to bounce and the reason is logged. The Status constant comments become provider-neutral with the Mailgun correspondence.

### 2.5 Ordering and idempotence
Mailgun delivery is not ordered and is retried. Rank: initiated 0, processed 1, deferred 2, delivered 3, bounce 4 (dropped 4, legacy SendGrid value), open 5, click 6, unsubscribe 7, spamreport 8. A late permanent failure after an open is discarded on purpose (engagement implies delivery). NULL status is treated as rank 0 (`status IS NULL OR status IN (...)`). An event is applied only if its rank is strictly greater than the current one. This is done atomically in SQL (`UPDATE ... WHERE id=? AND tm_delete IS NULL AND (status IS NULL OR status IN (lower-rank statuses and empty))`; a soft-deleted email is never updated, unlike the generic `EmailUpdate`) so that two replicas cannot regress a status (accepted racing delivered). The write sets `tm_update`, refreshes the cache, and only when `RowsAffected` is 1 (a matched row always changes because the new status is never in the allowed-from set, so the status value itself differs; sqlite counts matched rows, so both engines agree). No row changed means the event was stale, duplicate, or the email does not exist: debug log, no webhook event, success. Only a real change publishes `EventTypeUpdated`; the payload is read right after the write, so a concurrent higher update can make it newer than the applied status, which is accepted (the event then reports the latest state). Events apply regardless of the stored provider: after a Mailgun timeout that falls back to SendGrid, a Mailgun event for the same email id is still real delivery information. The SendGrid path keeps its current behavior.

### 2.6 Rolling deploy
During the mixed period old replicas store `sendgrid` at create time and new replicas store an empty value then `mailgun`/`sendgrid` after send; both are valid values for the same column. New code to old schema and old code to new data are both safe (no schema change, `provider_type` unread elsewhere). An old replica receiving a Mailgun webhook keeps the TODO behavior (returns nil). A new variable on the Mailgun request is ignored by old readers.

### 2.7 Errors and retries
The hook listener logs a Hook error and answers 200, so the sender never retries a failed update. `hookMailgun` returns an error for invalid JSON, a missing event, or a database failure (logged by the listener); an unknown event, a missing or invalid email id, a stale or duplicate event, and an unknown email are not errors (debug or warning log, return nil). A transient database failure therefore loses that single event, as it does for the SendGrid path today.

### 2.8 Rollout order
Deploy the code first, then register the per-event webhook URLs on the provider side. Until registered, Mailgun emails stay at `initiated`. Only the `accepted`, `delivered`, `failed`, `opened`, `clicked`, `unsubscribed` and `complained` events need a URL; open and click events additionally need tracking enabled on the provider side.

## 3. Verification plan
- Unit tests (gomock, fixed payloads, example addresses only): mapping table for every event, both payload shapes, `user-variables` as `[]`/object/missing, unknown event, invalid JSON, invalid UUID, stale and duplicate events (no webhook publish), DB error.
- Tests also cover duplicate delivery of the same event (second is a no-op without publish), legacy `dropped` and NULL status, a soft-deleted email, and cache refresh after the conditional update.
- dbhandler test for the conditional update (changed and unchanged rows) with the existing sqlite harness.
- hook tests: DB error returns an error; repeated open/click is a no-op without publish.
- Send tests: order, all providers failing leaves an empty provider type, provider recorded for each provider, both fail.
- Engine test: variable added.
- `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`.
- Real sends and external APIs are never called.
