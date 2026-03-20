# Expose Paddle Fields on BillingManagerAccount Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Expose `paddle_customer_id` and `paddle_subscription_id` on the `BillingManagerAccount` API response so the frontend can integrate with Paddle.js.

**Architecture:** Add two read-only string fields to the existing WebhookMessage pattern. Fields already exist in DB and internal model — we're removing the filter. No new endpoints, no write-path changes, no DB migration.

**Tech Stack:** Go, OpenAPI/YAML, Sphinx RST

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-paddle-fields-on-billing-account`

---

## Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| `omitempty` on JSON tags | **Yes** | Existing fields like `name` and `payment_method` do NOT use `omitempty`, so this is intentionally inconsistent. Paddle IDs are meaningless when empty (unlike `name` which is always relevant). Most customers won't have Paddle subscriptions, so hiding empty fields reduces noise for Paddle.js integration. |
| bin-api-manager test changes | **None needed** | Tests construct `WebhookMessage{}` with zero-value paddle fields (empty strings). With `omitempty`, empty strings don't appear in JSON, so expected JSON strings remain unchanged. Struct comparison tests also pass because zero values match on both sides. |
| Webhook event payloads | **Backward-compatible** | `CreateWebhookEvent()` calls `ConvertWebhookMessage()`, so webhook payloads will now include paddle fields. With `omitempty`, accounts WITHOUT Paddle subscriptions produce identical payloads (empty strings omitted). Accounts WITH subscriptions get two new fields — this is additive and backward-compatible. No schema version bump needed. |

## Known Pre-Existing Issue (Out of Scope)

The `balance_add_force` and `balance_subtract_force` endpoints return raw `Account` structs instead of `WebhookMessage`. This is a pre-existing WebhookMessage pattern violation — not introduced by this change. Consequence: those endpoints always serialize `"paddle_subscription_id":""` (no `omitempty`), while GET/PUT endpoints using `WebhookMessage` will omit empty paddle fields. Fixing this inconsistency (converting balance_force endpoints to return `WebhookMessage`) should be a separate follow-up PR.

## Test Impact Verification

All 7 test functions in `bin-api-manager` verified — none require changes:

| Test (server/) | Returns | Paddle in expected JSON? | After change |
|---|---|---|---|
| `Test_GetBillingAccountsId` (line 51) | `WebhookMessage` | No | `omitempty` omits empty → unchanged |
| `Test_PutBillingAccountsId` (line 127) | `WebhookMessage` | No | Same |
| `Test_PutBillingAccountsIdPaymentInfo` (line 206) | `WebhookMessage` | No | Same |
| `Test_PostBillingAccountsIdBalanceAddForce` (line 283) | `Account` | Yes (already) | Account untouched |
| `Test_PostBillingAccountsIdBalanceSubtractForce` (line 357) | `Account` | Yes (already) | Account untouched |

| Test (servicehandler/) | Comparison | After change |
|---|---|---|
| `Test_BillingAccountUpdateBasicInfo` (line 112) | Struct pointer | Zero-value fields match both sides |
| `Test_BillingAccountUpdatePaymentInfo` (line 185) | Struct pointer | Zero-value fields match both sides |

---

### Task 1: Update WebhookMessage Test (Red)

**Files:**
- Modify: `bin-billing-manager/models/account/webhook_test.go`

**Step 1: Add paddle fields to TestAccount_ConvertWebhookMessage test data**

In the "full account data" test case (lines 22-38), add after `PaymentMethod` (line 34):

```go
PaddleSubscriptionID: "sub_01h8bxq9f3e4t5a6g7h8j9k0",
PaddleCustomerID:     "ctm_01h8bxq9f3e4t5a6g7h8j9k0",
```

**Step 2: Add paddle field assertions to TestAccount_ConvertWebhookMessage**

After the PaymentMethod assertion block (after line 93), add:

```go
if result.PaddleSubscriptionID != tt.account.PaddleSubscriptionID {
    t.Errorf("PaddleSubscriptionID = %s, expected %s", result.PaddleSubscriptionID, tt.account.PaddleSubscriptionID)
}

if result.PaddleCustomerID != tt.account.PaddleCustomerID {
    t.Errorf("PaddleCustomerID = %s, expected %s", result.PaddleCustomerID, tt.account.PaddleCustomerID)
}
```

**Step 3: Add paddle fields to TestAccount_CreateWebhookEvent test data**

In the "full account data" test case (lines 108-124), add after `PaymentMethod` (line 120):

```go
PaddleSubscriptionID: "sub_01h8bxq9f3e4t5a6g7h8j9k0",
PaddleCustomerID:     "ctm_01h8bxq9f3e4t5a6g7h8j9k0",
```

No new assertions needed for CreateWebhookEvent — it validates JSON marshaling and only checks ID/Name (lines 160-166).

**Step 4: Run test to verify it fails**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-paddle-fields-on-billing-account/bin-billing-manager && go test -v ./models/account/ -run TestAccount_ConvertWebhookMessage`

Expected: FAIL — `WebhookMessage` struct does not have `PaddleSubscriptionID` or `PaddleCustomerID` fields (compile error).

---

### Task 2: Update WebhookMessage Struct and ConvertWebhookMessage (Green)

**Files:**
- Modify: `bin-billing-manager/models/account/webhook.go`

**Step 1: Add paddle fields to WebhookMessage struct**

In `webhook.go`, add after `PaymentMethod` (line 23) and before `TmLastTopUp` (line 25):

```go
PaddleSubscriptionID string `json:"paddle_subscription_id,omitempty"`
PaddleCustomerID     string `json:"paddle_customer_id,omitempty"`
```

**Step 2: Update ConvertWebhookMessage to copy paddle fields**

In `ConvertWebhookMessage()`, add after `PaymentMethod` assignment (line 48) and before `TmLastTopUp` (line 50):

```go
PaddleSubscriptionID: h.PaddleSubscriptionID,
PaddleCustomerID:     h.PaddleCustomerID,
```

**Step 3: Run test to verify it passes**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-paddle-fields-on-billing-account/bin-billing-manager && go test -v ./models/account/ -run TestAccount`

Expected: PASS — both `TestAccount_ConvertWebhookMessage` and `TestAccount_CreateWebhookEvent`.

---

### Task 3: Update OpenAPI Schema

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (insert after line 443, before `tm_last_topup` at line 444)

**Step 1: Add paddle fields to BillingManagerAccount schema**

Insert after the `payment_method` `$ref` line (line 443), before `tm_last_topup` (line 444). Use 8-space indent for property names, 10-space for sub-properties (matches existing pattern):

```yaml
        paddle_subscription_id:
          type: string
          description: "The Paddle subscription identifier for this billing account. Populated automatically when a Paddle subscription is created via Paddle webhook processing. Read-only — not settable via API. Present only when the account has an active Paddle subscription."
          example: "sub_01h8bxq9f3e4t5a6g7h8j9k0"
        paddle_customer_id:
          type: string
          description: "The Paddle customer identifier for this billing account. Populated automatically when a Paddle customer record is created via Paddle webhook processing. Read-only — not settable via API. Present only when the account has a linked Paddle customer."
          example: "ctm_01h8bxq9f3e4t5a6g7h8j9k0"
```

**AI-Native compliance notes:**
- Provenance: These are system-managed fields, not obtained from a user-facing API endpoint. The descriptions state they are read-only and populated by Paddle webhook processing.
- Strict typing: `type: string` is correct — Paddle IDs are opaque strings with vendor-defined format.
- Realistic examples: Paddle ID format uses `sub_`/`ctm_` prefixes per Paddle's actual ID scheme.

**Step 2: Run OpenAPI verification**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-paddle-fields-on-billing-account/bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`

Expected: All pass.

---

### Task 4: Regenerate API Manager Server Code

**Files:**
- Regenerated: `bin-api-manager/gens/openapi_server/gen.go` (auto-generated from openapi.yaml via oapi-codegen)

**Note:** `bin-api-manager` does NOT have a Go module dependency on `bin-openapi-manager`. Code generation reads the YAML file directly via `oapi-codegen`. No `replace` directive needed.

**Step 1: Regenerate and verify api-manager**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-paddle-fields-on-billing-account/bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`

Expected: All pass. Existing tests in `server/billing_accounts_test.go` and `pkg/servicehandler/billingaccount_test.go` pass without modification because:
- Struct literal comparisons: unset paddle fields default to empty strings (zero value) on both sides
- JSON string comparisons: `omitempty` means empty paddle fields don't appear in serialized JSON

---

### Task 5: Update RST Documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/billing_account_struct.rst`

**Step 1: Update JSON example block**

In the JSON example (lines 12-28), insert after `"payment_method": "",` (line 22) and before `"tm_last_topup"` (line 23):

```
        "paddle_subscription_id": "sub_01h8bxq9f3e4t5a6g7h8j9k0",
        "paddle_customer_id": "ctm_01h8bxq9f3e4t5a6g7h8j9k0",
```

**Step 2: Add field descriptions**

After the `payment_method` description (line 38) and before `tm_last_topup` (line 39), add descriptions matching existing convention (no "Optional" marker — existing fields don't use it):

```rst
* ``paddle_subscription_id`` (String): The Paddle subscription identifier. Populated automatically when a Paddle subscription is created via Paddle webhook processing. Read-only. Omitted from the response when not set (no Paddle subscription).
* ``paddle_customer_id`` (String): The Paddle customer identifier. Populated automatically when a Paddle customer record is created via Paddle webhook processing. Read-only. Omitted from the response when not set (no Paddle customer).
```

**Step 3: Rebuild HTML documentation**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-paddle-fields-on-billing-account/bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`

Expected: Build succeeds with no warnings on the modified file.

---

### Task 6: Run Full Verification on All Changed Services

Run verification for each changed service:

**Step 1: bin-billing-manager**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-paddle-fields-on-billing-account/bin-billing-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`

**Step 2: bin-openapi-manager**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-paddle-fields-on-billing-account/bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`

**Step 3: bin-api-manager**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-paddle-fields-on-billing-account/bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`

Expected: All pass for all three services.

---

### Task 7: Commit and Push

**Step 1: Stage all changes**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-paddle-fields-on-billing-account
git add docs/plans/2026-03-20-expose-paddle-fields-billing-account.md
git add bin-billing-manager/models/account/webhook.go
git add bin-billing-manager/models/account/webhook_test.go
git add bin-billing-manager/go.mod bin-billing-manager/go.sum
git add bin-openapi-manager/openapi/openapi.yaml
git add bin-openapi-manager/gens/
git add bin-openapi-manager/go.mod bin-openapi-manager/go.sum
git add bin-api-manager/gens/
git add bin-api-manager/go.mod bin-api-manager/go.sum
git add bin-api-manager/docsdev/source/billing_account_struct.rst
git add -f bin-api-manager/docsdev/build/
```

Only stage `go.mod`/`go.sum` if actually changed by `go mod tidy`. Do NOT stage `vendor/` directories.

**Step 2: Commit**

```bash
git commit -m "NOJIRA-Expose-paddle-fields-on-billing-account

Expose paddle_customer_id and paddle_subscription_id on the BillingManagerAccount
API response for Paddle.js frontend integration.

- bin-billing-manager: Add paddle fields to WebhookMessage with omitempty
- bin-billing-manager: Update ConvertWebhookMessage to copy paddle fields
- bin-billing-manager: Add test coverage for paddle field conversion
- bin-openapi-manager: Add paddle_subscription_id and paddle_customer_id to BillingManagerAccount schema
- bin-api-manager: Regenerate server code from updated OpenAPI spec
- bin-api-manager: Update billing_account_struct.rst with paddle field documentation
- docs: Add design document for paddle fields exposure"
```

**Step 3: Check for conflicts with main**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

**Step 4: Push and create PR**

```bash
git push -u origin NOJIRA-Expose-paddle-fields-on-billing-account
```

Then create PR with title: `NOJIRA-Expose-paddle-fields-on-billing-account`
