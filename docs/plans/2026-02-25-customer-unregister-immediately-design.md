# Customer Unregister Immediately — Design

## Problem Statement

The current self-service unregistration flow (`POST /auth/unregister`) freezes the customer account and schedules deletion after a 30-day grace period. There is no way for a customer to skip the grace period and delete their account immediately via self-service.

The admin hard-delete (`DELETE /v1.0/customers/{id}`) already supports immediate deletion but does not anonymize PII.

## Goal

Add an `immediate` boolean field to the existing `POST /auth/unregister` request body. When `immediate: true`, the account is frozen and then immediately deleted (PII anonymized, all resources cascade-deleted) in a single request.

## Current State

### Two existing deletion paths

1. **Self-service freeze** (`POST /auth/unregister`): Sets `status=frozen`, `tm_deletion_scheduled=now`. A daily background goroutine permanently deletes frozen customers after 30 days.
2. **Admin hard-delete** (`DELETE /v1.0/customers/{id}`): Sets `status=deleted`, `tm_delete=now` immediately. No PII anonymization.

### Event sequence on deletion

- `customer_frozen` event: Only bin-call-manager reacts (hangs up active calls).
- `customer_deleted` event: All services cascade-delete resources (agents, numbers, flows, queues, trunks, extensions, files, billing, tags, transcriptions, contacts).

### Recovery

`DELETE /auth/unregister` cancels the freeze (sets status back to `active`, clears `tm_deletion_scheduled`).

## Approach

**Freeze first, then immediately delete** — reuse existing `Freeze()` logic, then immediately run the same anonymization + deletion logic from `cleanupFrozenExpired`. This keeps event ordering consistent (`customer_frozen` → `customer_deleted`).

### Why not other approaches

- **Two sequential RPC calls from api-manager** (freeze + existing delete): The existing `Delete` method does not anonymize PII. Also not atomic — if the second call fails, the customer is left frozen.
- **Flag on existing Freeze RPC**: Overloads the freeze method with dual behavior, changes RPC signature affecting mocks across 30+ services.

## Design

### Section 1: API Layer (bin-api-manager)

**Request body** — Add `immediate` boolean to `RequestBodyUnregisterPOST`:

```go
type RequestBodyUnregisterPOST struct {
    Password           string `json:"password"`
    ConfirmationPhrase string `json:"confirmation_phrase"`
    Immediate          bool   `json:"immediate"`
}
```

**Handler** — In `PostAuthUnregister` (`lib/service/unregister.go`), after credential validation:
- If `req.Immediate` is false (or omitted): call `CustomerSelfFreeze` (existing behavior)
- If `req.Immediate` is true: call new `CustomerSelfFreezeAndDelete`

**New servicehandler method** — `CustomerSelfFreezeAndDelete` in `pkg/servicehandler/customer.go`:
- Same permission check as `CustomerSelfFreeze` (`PermissionCustomerAdmin`)
- Calls new RPC `CustomerV1CustomerFreezeAndDelete`

**Files:**
- `bin-api-manager/lib/service/unregister.go`
- `bin-api-manager/pkg/servicehandler/customer.go`
- `bin-api-manager/pkg/servicehandler/main.go` (interface)

### Section 2: RPC Layer (bin-common-handler)

**New method** — `CustomerV1CustomerFreezeAndDelete` in `requesthandler/customer_customer.go`:
- URI: `/v1/customers/<customer-id>/freeze_and_delete`
- Method: POST
- Same pattern as `CustomerV1CustomerFreeze`

**Files:**
- `bin-common-handler/pkg/requesthandler/customer_customer.go`
- `bin-common-handler/pkg/requesthandler/main.go` (interface)

### Section 3: Business Logic (bin-customer-manager)

**New method** — `FreezeAndDelete` in `customerhandler/freeze.go`:

```
1. Call Freeze(ctx, id) — idempotent, handles already-frozen
2. Check if returned customer is already StatusDeleted → return early (idempotent)
3. Generate anonymized identifiers from UUID (same as cleanupFrozenExpired)
4. Call db.CustomerAnonymizePII(ctx, id, anonName, anonEmail)
5. Fetch updated customer via db.CustomerGet(ctx, id)
6. Publish customer_deleted event
7. Return deleted customer
```

**Edge cases:**
- Already frozen: `Freeze()` returns current state (no-op), then anonymization proceeds. Valid use case — customer can freeze first, then later choose immediate delete.
- Already deleted: Guard after `Freeze()` return, return early.
- Concurrent requests: `Freeze()` handles its own race conditions. If two requests both pass freeze and try to anonymize, the second should handle "already deleted" gracefully.

**New listenhandler route:**
- Regex: `regV1CustomersIDFreezeAndDelete = regexp.MustCompile("/v1/customers/" + regUUID + "/freeze_and_delete$")`
- Handler: `processV1CustomersIDFreezeAndDeletePost` — extracts UUID, calls `h.customerHandler.FreezeAndDelete(ctx, id)`
- Note: Existing `/freeze$` regex uses `$` anchor, so no ordering conflict.

**Files:**
- `bin-customer-manager/pkg/customerhandler/freeze.go`
- `bin-customer-manager/pkg/customerhandler/main.go` (interface)
- `bin-customer-manager/pkg/listenhandler/main.go` (regex + route case)
- `bin-customer-manager/pkg/listenhandler/v1_customers_freeze.go` (new handler function)

### Section 4: OpenAPI Spec (bin-openapi-manager)

Add `immediate` field to `RequestBodyAuthUnregisterPOST` schema:

```yaml
immediate:
  type: boolean
  description: "If true, skip the 30-day grace period and delete the account immediately. Default: false."
  example: false
```

Update `POST /auth/unregister` description to mention the immediate option.

**Files:**
- `bin-openapi-manager/openapi/openapi.yaml`
- `bin-openapi-manager/openapi/paths/auth/unregister.yaml`

### Section 5: RST Documentation (bin-api-manager)

The current customer RST docs are significantly out of sync with `WebhookMessage`. This feature requires updating them.

**customer_struct_customer.rst** — Rewrite to match `WebhookMessage` fields:
- Remove fields NOT in WebhookMessage: `username`, `line_secret`, `line_token`, `permission_ids`
- Add fields FROM WebhookMessage: `email`, `phone_number`, `address`, `billing_account_id`, `email_verified`, `status`, `tm_deletion_scheduled`

**customer_overview.rst** — Add:
- Account deletion lifecycle section (freeze → 30-day grace → delete, immediate delete option, recover)
- `status` enum values: `initial`, `active`, `frozen`, `deleted`, `expired`
- Update "Customer Properties" table to match WebhookMessage

**customer_tutorial.rst** — Add:
- "Unregister account" tutorial with curl example for `POST /auth/unregister`
- "Unregister account immediately" tutorial with curl example using `immediate: true`
- "Cancel unregistration" tutorial with curl example for `DELETE /auth/unregister`

**Build steps:**
```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
git add -f bin-api-manager/docsdev/build/
```

**Files:**
- `bin-api-manager/docsdev/source/customer_struct_customer.rst`
- `bin-api-manager/docsdev/source/customer_overview.rst`
- `bin-api-manager/docsdev/source/customer_tutorial.rst`
- `bin-api-manager/docsdev/build/` (rebuilt HTML)

### Section 6: Project Rule Updates (root CLAUDE.md)

**Strengthen "Feature Changes Require RST Documentation Updates" section:**

Add WebhookMessage rule:
> RST struct docs must only document fields from `WebhookMessage` (defined in `models/<entity>/webhook.go`), not the internal model struct. Fields stripped by `ConvertWebhookMessage()` must not appear in user-facing documentation.

Add RST check to verification workflow:
> After making user-facing changes, verify RST docs in `bin-api-manager/docsdev/source/` are in sync. Compare struct docs against the relevant `WebhookMessage` fields, not the internal model struct.

**Files:**
- `CLAUDE.md` (root)

### Section 7: Tests

**bin-customer-manager/pkg/customerhandler/freeze_test.go:**
- Test `FreezeAndDelete` normal case (active → frozen → deleted)
- Test `FreezeAndDelete` already frozen (frozen → deleted)
- Test `FreezeAndDelete` already deleted (return early, idempotent)
- Test `FreezeAndDelete` freeze failure (error propagation)

**bin-api-manager/pkg/servicehandler/customer_test.go:**
- Test `CustomerSelfFreezeAndDelete` with valid permission
- Test `CustomerSelfFreezeAndDelete` permission denied

## File Summary

| Service | File | Change |
|---|---|---|
| bin-api-manager | `lib/service/unregister.go` | Add `Immediate` field, branching |
| bin-api-manager | `pkg/servicehandler/customer.go` | Add `CustomerSelfFreezeAndDelete` |
| bin-api-manager | `pkg/servicehandler/main.go` | Interface addition |
| bin-api-manager | `pkg/servicehandler/customer_test.go` | New tests |
| bin-api-manager | `docsdev/source/customer_struct_customer.rst` | Rewrite to match WebhookMessage |
| bin-api-manager | `docsdev/source/customer_overview.rst` | Add deletion lifecycle, status enum |
| bin-api-manager | `docsdev/source/customer_tutorial.rst` | Add unregister tutorials |
| bin-common-handler | `pkg/requesthandler/customer_customer.go` | New RPC method |
| bin-common-handler | `pkg/requesthandler/main.go` | Interface addition |
| bin-customer-manager | `pkg/customerhandler/freeze.go` | Add `FreezeAndDelete` |
| bin-customer-manager | `pkg/customerhandler/main.go` | Interface addition |
| bin-customer-manager | `pkg/customerhandler/freeze_test.go` | New tests |
| bin-customer-manager | `pkg/listenhandler/main.go` | Regex + route case |
| bin-customer-manager | `pkg/listenhandler/v1_customers_freeze.go` | New handler function |
| bin-openapi-manager | `openapi/openapi.yaml` | Schema field addition |
| bin-openapi-manager | `openapi/paths/auth/unregister.yaml` | Description update |
| root | `CLAUDE.md` | WebhookMessage rule + RST check |
