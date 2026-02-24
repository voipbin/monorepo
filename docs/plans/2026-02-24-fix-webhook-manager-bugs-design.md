# Fix webhook-manager bugs

## Problem

The webhook-manager service has 4 bugs affecting webhook delivery reliability and cache correctness:

1. **HTTP request body consumed on retry** — `sendMessage` creates the `http.Request` once before the retry loop. After the first `client.Do(req)`, the body buffer is consumed. Retries 2 and 3 send empty bodies.

2. **Response body closed before caller reads it** — `defer resp.Body.Close()` runs on function return, so callers get a response with a closed body. Currently dormant (callers only read `StatusCode`), but a correctness issue.

3. **Redis cache key uses `%d` on UUID** — `fmt.Sprintf("webhook.account:%d", id)` uses integer format on `uuid.UUID` (`[16]byte`). Works consistently (set and get use the same mangled key) but produces unreadable Redis keys.

4. **subscribehandler is never started** — Fully implemented and tested but never wired in `main.go`. Customer webhook URI updates stay stale in Redis for up to 24h.

## Approach

### Fix 1: Retry body reuse + response body handling (message.go)

Move `http.NewRequest` inside the retry loop. Drain and close the body inside `sendMessage`. Change return type to `error` only (callers only used `StatusCode` for debug logging, which is now done inside `sendMessage`).

### Fix 2: Redis cache key format (cachehandler/handler.go)

Change `%d` to `%s` on lines 42 and 53. Existing `%d`-format keys become orphans, causing temporary cache misses that re-fetch from customer-manager and re-cache correctly. Old keys expire after 24h.

### Fix 3: Wire subscribehandler (main.go)

Add `runSubscribe()` following the same pattern as contact-manager and other services. Subscribe to `QueueNameCustomerEvent` to keep the webhook config cache warm when customers are created/updated.

### Fix 4: Update sendMessage callers (webhook.go)

Update goroutines in `SendWebhookToCustomer` and `SendWebhookToURI` to match new `sendMessage` signature (returns `error` only).

## Files changed

| File | Change |
|------|--------|
| `pkg/webhookhandler/message.go` | Recreate request per retry, drain/close body, return `error` only |
| `pkg/webhookhandler/webhook.go` | Update `sendMessage` callers |
| `pkg/cachehandler/handler.go` | `%d` → `%s` (lines 42, 53) |
| `cmd/webhook-manager/main.go` | Add `runSubscribe()`, wire from `run()` |
| Tests | Update affected test expectations |

## What is NOT changing

- MySQL connection removal (separate concern)
- `message_test.go` (commented-out integration tests)
