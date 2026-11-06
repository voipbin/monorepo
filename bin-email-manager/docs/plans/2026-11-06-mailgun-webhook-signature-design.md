# Mailgun webhook signature verification (VOIP-1681)

## 1. Issue analysis

Checked against origin/main (8c3246918) at the time of writing.

- `pkg/emailhandler/hook.go` `hookMailgun` parses the body, maps the event to a status, updates the email and publishes `email_updated`. Webhook requests are verified with the Mailgun signature before any of this happens.
- The body is forwarded unchanged by `bin-hook-manager` (`pkg/servicehandler/email.go`). `pkg/listenhandler/v1_hooks.go` logs a Hook error and answers 200, as does hook-manager. Response codes are not changed by this work. The hook listener's error and parse-failure logs do not print the request (it carries recipient addresses and the signature material); this applies to all hooks. Other log statements of the SendGrid path are not changed.
- `models/mailgun/main.go` `Webhook` gains the `Signature` object. `NewEmailHandler` is called from `cmd/email-manager/main.go` and `cmd/email-control/main.go` (and one test).
- Status updates are already monotonic and idempotent (`statusesBefore`), so a replayed genuine event cannot move a status backwards.

## 2. Design

### 2.1 Algorithm (Mailgun "Securing Webhooks", read from the official page)
The request carries `signature: {timestamp, token, signature}` next to `event-data` (the flat body, where the signature sits next to the event fields, is the form the existing parser already accepts and is supported the same way). The signature is the hex of HMAC-SHA256 with the Webhook Signing Key as the key over `timestamp` concatenated with `token`, no separator. Optional checks named by the page: do not honor a repeated token, and check that the timestamp is not too far from now without being aggressive. A subaccount adds `parent-signature`; it is not used because the account is a single primary account.

### 2.2 Verification (`verifyMailgunSignature`)
Order: JSON parse, signature, then the event handling. Rejections return an error before any db call or webhook publish.
1. Empty key: reject (fail closed).
2. Missing `signature` object, empty token, signature or timestamp: reject.
3. Timestamp: JSON string (documented) or bare number, parsed as an integer number of seconds. The original text is what is signed.
4. Signature: hex decoded, compared with `hmac.Equal` (constant time). Non-hex input is rejected.
5. Timestamp range, checked after the signature matches: not older than 24 hours and not more than 5 minutes ahead.
Errors name only the reason category. The key, signature, token, timestamp and recipient address are never logged, including by the hook listener (section 1). A rejected request logs one Warn line.

### 2.3 Timestamp window
Mailgun answers: a 200 is final, a 406 is final, anything else is retried for about 8 hours at 5 min, 10 min, 15 min, 1 h, 2 h and 4 h (webhook retries page). Whether a retry is re-signed is not stated, so the window must cover the whole retry period: 24 hours in the past (three times the retry period, absorbing queue delays), and 5 minutes into the future for clock skew. Because the listener always answers 200, a rejection is final for Mailgun (it is not retried); the retry period matters only when the hook gateway itself is unavailable, and the window covers that case. This is a coarse replay bound, not the primary control.

### 2.4 Replay (token store)
No token store. Reasons: two replicas make an in-memory cache ineffective; a shared store (Redis) adds state and a new failure mode on the webhook path; and a replayed genuine event is harmless because a status is applied only when it is later than the current one, and a repeated event publishes nothing. The signature and the 24 hour window remain. Residual risk, accepted: the signature covers `timestamp` and `token` only, not the event body, so a party that obtained one valid signature could attach another body within the window. The signature material is seen only on the TLS hop and inside Mailgun, the target email id must also be known, a status can only move forward, and the effect is bounded to one `email_updated` per status step.

### 2.5 Configuration
`MAILGUN_WEBHOOK_SIGNING_KEY` through the existing flag plus `BindEnv` pattern (`mailgun_webhook_signing_key`), passed to `NewEmailHandler` (new last parameter) from both callers. `komodo/docker-compose.yml` references `[[BIN_MANAGER__EMAIL_MANAGER_MAILGUN_WEBHOOK_SIGNING_KEY]]`. `email-manager` logs one warning at start when the key is empty. The legacy `k8s/deployment.yml` is not changed because it is not a deployment path.

### 2.6 Key rotation
One key only (see operations.md for the rotation steps).

### 2.7 Out of scope
The SendGrid hook, response codes, late permanent failures, attachments and a retry queue.

## 3. Compatibility
The change is code and one environment variable; no schema change. The new variable is read at start, and the stack definition references it.

## 4. Verification plan
- Valid signature accepted for the wrapped and the flat body, timestamp as string and as number, upper case hex, both window edges.
- Rejected: wrong signature, missing signature object, empty or truncated or non-hex signature, empty or changed token, changed or missing or non-numeric or fractional or object timestamp, timestamp outside the window (past and future), non-object signature, empty key (with a signature made with the empty key and with a valid one, wrapped and flat). Each rejection runs with strict mocks, so any db call or publish fails the test.
- `Hook` with the mailgun uri returns the error. The real clock is used when no clock is injected. Config binding of the environment variable.
- `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`. RED was run first: the rejection tests failed before `verifyMailgunSignature` existed.
