# Design: Conversation-Manager Account Handler Fixes

**Date:** 2026-03-01
**Service:** bin-conversation-manager
**Scope:** Account handler — security, consistency, and cleanup fixes

## Problem Statement

The conversation-manager's account handler has 8 issues identified during a code review:

1. **Credentials exposed in API responses** — `Secret` and `Token` fields are included in `WebhookMessage` and raw `Account` responses, violating the OpenAPI spec's "Write-only" designation.
2. **Inconsistent event publishing** — Create and Delete use `PublishEvent` (internal only), while Update uses `PublishWebhookEvent` (internal + customer webhooks).
3. **Error masking** — GET/PUT/DELETE return `simpleResponse(500)` for all errors including not-found. Other services return 404.
4. **No LINE webhook teardown on delete** — Deleting a LINE account leaves a dangling webhook URL registered with LINE's API.
5. **ExecContext missing in AccountUpdate** — Uses `h.db.Exec` instead of `h.db.ExecContext`, breaking context cancellation.
6. **Missing debug logs** — No debug logging after successful data retrieval (required by CLAUDE.md conventions).
7. **Dead interface surface** — `DBHandler.AccountSet` is exposed but never called by business logic.
8. **Raw Account struct returned instead of WebhookMessage** — Listenhandler serializes the internal struct directly, bypassing the WebhookMessage pattern.

## Approach

Single PR with all 9 fixes (8 issues + 1 OpenAPI annotation). All changes are within `bin-conversation-manager` except the OpenAPI annotation. No cross-service dependencies. Each fix is isolated to specific files.

## Detailed Design

### Fix 1: Strip credentials from WebhookMessage

**File:** `models/account/webhook.go`

Remove `Secret` and `Token` fields from `WebhookMessage` struct and from `ConvertWebhookMessage()` method. This corrects a spec violation — the OpenAPI schema already marks these fields as "Write-only."

Pattern reference: `bin-agent-manager` excludes `PasswordHash` from its `WebhookMessage`.

### Fix 2: Consistent event publishing

**File:** `pkg/accounthandler/db.go`

Change Create (line 61) and Delete (line 142) from `PublishEvent` to `PublishWebhookEvent`.

How it works:
- `PublishWebhookEvent(ctx, customerID, eventType, data)` internally calls both `PublishEvent` (internal queue, `json.Marshal(data)` with full credentials) and `PublishWebhook` (customer webhooks via `CreateWebhookEvent()` → `ConvertWebhookMessage()` → stripped credentials)
- Internal subscribers still get the full struct
- External webhooks get sanitized data
- `*account.Account` already satisfies the `notifyhandler.WebhookMessage` interface

Impact: No external service subscribes to conversation-manager account events (confirmed by grep). This is additive — customers with webhooks configured will now receive create/delete events.

### Fix 3: Return 404 instead of 500

**File:** `pkg/listenhandler/v1_accounts.go`

Change `simpleResponse(500)` to `simpleResponse(404)` in:
- `processV1AccountsIDGet` (line 125)
- `processV1AccountsIDPut` (line 185)
- `processV1AccountsIDDelete` (line 222)

Matches the convention used by call-manager, agent-manager, and flow-manager. The list endpoint (`processV1AccountsGet`) keeps 500 since a list error is a server error, not a not-found.

Scoped to accounts only — conversation handlers have the same issue but are out of scope.

### Fix 4: LINE webhook teardown on delete

**Files:**
- `pkg/linehandler/main.go` — Add `Teardown(ctx context.Context, ac *account.Account) error` to interface
- `pkg/linehandler/teardown.go` — New file: calls `c.SetWebhookEndpointURL("").WithContext(ctx).Do()`
- `pkg/accounthandler/setup.go` — Add `teardown()` private method mirroring `setup()` dispatch pattern (LINE → teardown, SMS → no-op, unknown → nil)
- `pkg/accounthandler/db.go` — Restructure Delete flow

Delete flow changes from:
```
DB delete → Get deleted record → Publish event
```
To:
```
Get account → Teardown (best-effort) → DB delete → Get deleted record → Publish event
```

Edge cases:
- Teardown failure (LINE API down, invalid credentials): log warning, proceed with deletion
- SMS type: teardown is a no-op
- Unknown type: teardown returns nil

### Fix 5: ExecContext in AccountUpdate

**File:** `pkg/dbhandler/account.go` line 238

Change `h.db.Exec(sqlStr, args...)` to `h.db.ExecContext(ctx, sqlStr, args...)`.

### Fix 6: Debug logs after data retrieval

**File:** `pkg/accounthandler/db.go`

Add debug logs in `Get` and `List`:
```go
// Get:
log.WithField("account", res).Debugf("Retrieved account info. account_id: %s", id)

// List:
log.WithField("accounts", res).Debugf("Retrieved account list. count: %d", len(res))
```

### Fix 7: Remove unused DBHandler.AccountSet

Note: `CacheHandler.AccountSet` (cache setter) is a different method and must stay.

**Files:**
- `pkg/dbhandler/main.go` — Remove `AccountSet` from DBHandler interface (line 28)
- `pkg/dbhandler/account.go` — Remove `AccountSet` function (lines 203-213)
- `pkg/dbhandler/account_test.go` — Remove `Test_AccountSet` (lines 122-208)
- Regenerate mocks: `go generate ./...`

Confirmed: No callers in `cmd/` or `pkg/accounthandler/`.

### Fix 8: Return WebhookMessage from listenhandler

**File:** `pkg/listenhandler/v1_accounts.go`

Convert to WebhookMessage before marshaling in all 5 handlers:
- Single responses: `json.Marshal(tmp.ConvertWebhookMessage())`
- List response: convert each account in slice, then marshal `[]*account.WebhookMessage`

The `accountHandler` interface still returns `*account.Account` — internal callers (conversationhandler, messagehandler, smshandler, CLI tool) need full credentials.

### Fix 9: OpenAPI writeOnly annotation

**File:** `bin-openapi-manager/openapi/openapi.yaml`

Add `writeOnly: true` to `secret` and `token` properties in `ConversationManagerAccount` schema. This formalizes the existing "Write-only" description. oapi-codegen ignores `writeOnly` for type generation — no impact on generated Go types.

## Test Updates

| Test file | Changes |
|-----------|---------|
| `pkg/accounthandler/db_test.go` | Mock `lineHandler.Teardown` in Delete test; update Create/Delete to expect `PublishWebhookEvent` instead of `PublishEvent` |
| `pkg/linehandler/teardown_test.go` | New: test Teardown calls `SetWebhookEndpointURL("")` |
| `pkg/accounthandler/setup_test.go` | Add teardown dispatch tests (LINE, SMS, unknown type) |
| `pkg/listenhandler/v1_accounts_test.go` | Update: expect no secret/token in responses; 404 instead of 500 for error cases |
| `pkg/dbhandler/account_test.go` | Remove `Test_AccountSet` |
| Mock regeneration | `go generate ./...` after linehandler and dbhandler interface changes |

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Credentials in internal event queue | Acceptable — internal queue, no external subscribers |
| LINE teardown API failure | Best-effort, logged as warning, non-blocking |
| Breaking API change (no secret/token in responses) | Corrects spec violation — OpenAPI already says "Write-only" |
| 404 masking real server errors | Matches monorepo convention; error details are logged |
| CLI tool credentials in output | Admin tool, full data expected — no change needed |

## Verification

- `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` in `bin-conversation-manager`
- `go generate ./...` in `bin-openapi-manager` (for Fix 9)
- `go generate ./...` in `bin-api-manager` (if openapi types are consumed)
