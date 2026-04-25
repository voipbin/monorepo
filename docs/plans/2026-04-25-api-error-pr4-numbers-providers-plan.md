# PR 4 — Numbers & providers handler migration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the numbers, available-numbers, providers, providercalls, trunks, and routes handlers to the canonical error envelope. First PR to wire the §6.1 **402 modifier** for billing-sensitive write endpoints.

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 3 (`NOJIRA-api-error-pr3-flows-activeflows`, merged `5abcb9e8b`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr4-numbers-providers`
**Branch:** `NOJIRA-api-error-pr4-numbers-providers` (branched from `origin/main` at `5abcb9e8b`)

## Scope

| File | LoC | Sites | Handlers |
|---|---|---|---|
| `server/numbers.go` | 317 | 26 | 8 |
| `server/available_numbers.go` | 68 | 3 | 1 |
| `server/providers.go` | 280 | 21 | 6 |
| `server/providercalls.go` | 231 | 13 | 4 |
| `server/trunks.go` | 208 | 15 | 5 |
| `server/routes.go` | 220 | 15 | 5 |

**Total: 93 sites across 29 handlers in 6 files.**

### Handler classification (per §6.1)

**Read (no path param) → 401, 500:**
- `GetNumbers`, `GetAvailableNumbers`, `GetProviders`, `GetProvidercalls`, `GetTrunks`, `GetRoutes`

**Read (with resource ID) → 400, 401, 403, 404, 500:**
- `GetNumbersId`, `GetProvidersId`, `GetProvidercallsId`, `GetTrunksId`, `GetRoutesId`

**Write (no resource ID) → 400, 401, 500:**
- `PostProviders`, `PostProvidersSetup` — provider catalog setup, admin-gated
- `PostProvidercalls` — provider-call admin op, gated by `PermissionProjectSuperAdmin`
- `PostTrunks` — trunk creation, admin
- `PostRoutes` — route creation, admin

**Write (no resource ID, BILLING-SENSITIVE) → 400, 401, 402, 500:**
- `PostNumbers` — number purchase. Deducts balance via `bin-number-manager` → `bin-billing-manager`. Identity verification gate.
- `PostNumbersRenew` — renewal cycle, recurrent charge.

**Write (with resource ID) → 400, 401, 403, 404, 500:**
- `PutNumbersId`, `PutNumbersIdFlowIds`, `PutNumbersIdMetadata`, `DeleteNumbersId`
- `PutProvidersId`, `DeleteProvidersId`
- `DeleteProvidercallsId`
- `PutTrunksId`, `DeleteTrunksId`
- `PutRoutesId`, `DeleteRoutesId`

**State-transition (+409):** none — these resources don't have status-driven state machines that conflict with normal CRUD.

**RPC-heavy (+503):** none — number/provider operations route to a single manager (number-manager or call-manager). Single-hop.

## §6.1 402 modifier — first wiring

This is the first PR to apply 402 PAYMENT_REQUIRED in the OpenAPI envelope. The translator already maps `"insufficient"` substring → 402 / `INSUFFICIENT_BALANCE` (added in PR 2 anticipating this). What this PR adds:

1. **OpenAPI declarations** — `POST /numbers` and `POST /numbers/renew` paths declare the `'402'` response with `$ref: '#/components/responses/PaymentRequired'` (already defined in `bin-openapi-manager` from PR 1b).
2. **No new translator pattern needed** — billing failures from `bin-billing-manager` propagate as `"insufficient balance"` style messages and route correctly today.
3. **RST enrichment** — populate the `billing-manager` catalog section (left as a placeholder in PR 1b/2/3) with the `INSUFFICIENT_BALANCE` reason and document which endpoints can return 402.

## Identity verification gate (`PostNumbers`)

`servicehandler.NumberCreate` enforces `customer.IdentityVerificationStatus == Verified` for non-virtual purchases. Today the error is `fmt.Errorf("customer identity verification required for number purchase")` — falls through translator to **500 INTERNAL**. That's wrong: this is a user-actionable precondition, not a server error.

**Fix:** Add a translator substring pattern `"identity verification required"` → **403 FORBIDDEN** with reason `IDENTITY_VERIFICATION_REQUIRED`. Treat this as access policy (the customer must complete an out-of-band verification flow), not 402 (which is balance-specific).

**Why 403 not 412/422:** the convention deliberately omits 412/422. `IDENTITY_VERIFICATION_REQUIRED` is access-shaped (a permissions-like gate, not request-shape validation), so 403 with a distinct reason is the cleanest fit.

**OpenAPI impact:** `POST /numbers` already has 403 in its baseline; no new status code needed.

## Forward-dependency notes

Captured in PR 3 for downstream PRs — restated here for PR 4:

- **PR 5 (Billing & customers):** populate empty `billing-manager` catalog section. PR 4 starts that work for the `INSUFFICIENT_BALANCE` row.
- **PR 6 (Messages & emails):** `POST /messages` and `POST /emails` are also billing-sensitive — wire 402 same as PR 4.
- **Per-resource `"deleted <X>"` patterns:** translator's `"deleted"` set is currently `call/groupcall/recording`. Numbers don't currently use `"deleted number"` strings, but if any new sentinel emerges in this group, add it here.

## Tasks

### Task 1: Plan doc + commit (this file)

### Task 2: Migrate `server/numbers.go`

8 handlers, 26 sites. Standard mappings:
- `commonmiddleware.AuthIdentity` missing → `abortWithError(c, cerrors.Unauthenticated(..., "AUTHENTICATION_REQUIRED", ...))`
- `c.BindJSON` failures → `abortWithError(c, cerrors.InvalidArgument(..., "INVALID_JSON_BODY", ...))`
- `uuid.Parse` failures → `abortWithError(c, cerrors.InvalidArgument(..., "INVALID_ID", ...))`
- servicehandler errors → `abortWithServiceError(c, err)` (delegates to translator)

Path-UUID validation hardening for `GetNumbersId`, `DeleteNumbersId`, `PutNumbersId`, `PutNumbersIdFlowIds`, `PutNumbersIdMetadata`: malformed IDs return 400 INVALID_ID, not forwarding `uuid.Nil` downstream.

Sample tests:
- `Test_numbersPost_MissingAuthIdentity` (UNAUTHENTICATED / AUTHENTICATION_REQUIRED)
- `Test_numbersPost_InvalidJSONBody` (INVALID_ARGUMENT / INVALID_JSON_BODY)
- `Test_numbersIDPut_InvalidID` (INVALID_ARGUMENT / INVALID_ID)
- `Test_numbersPost_InsufficientBalance` (PAYMENT_REQUIRED / INSUFFICIENT_BALANCE) — mock servicehandler to return `fmt.Errorf("insufficient balance")`
- `Test_numbersPost_IdentityVerificationRequired` (PERMISSION_DENIED / IDENTITY_VERIFICATION_REQUIRED) — mock to return `fmt.Errorf("customer identity verification required for number purchase")`

### Task 3: Migrate `server/available_numbers.go`

1 handler, 3 sites. Trivial — auth → 401, servicehandler → translated.

### Task 4: Migrate `server/providers.go`

6 handlers, 21 sites. Same pattern as numbers.go. All mutating operations are admin-gated; permission failures route via `"no permission"` translator pattern → 403.

Sample test: `Test_providersIDPut_InvalidID`.

### Task 5: Migrate `server/providercalls.go`

4 handlers, 13 sites. Per file comment: `Gated by PermissionProjectSuperAdmin`. Permission failures route via `"no permission"` → 403.

### Task 6: Migrate `server/trunks.go`

5 handlers, 15 sites. Standard. Path-UUID hardening on Get/Put/Delete by ID.

### Task 7: Migrate `server/routes.go`

5 handlers, 15 sites. Standard. Path-UUID hardening on Get/Put/Delete by ID.

### Task 8: Translator — add `"identity verification required"` pattern

Edit `bin-api-manager/server/error_translate.go`. Add to the `"no permission"` case-block:

```go
case strings.Contains(lowered, "no permission"),
    strings.Contains(lowered, "permission denied"),
    strings.Contains(lowered, "forbidden"),
    strings.Contains(lowered, "direct access"),
    strings.Contains(lowered, "does not belong"):
    return cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "PERMISSION_DENIED", "...").Wrap(err)
case strings.Contains(lowered, "identity verification required"):
    return cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "IDENTITY_VERIFICATION_REQUIRED", "Customer identity verification is required for this operation.").Wrap(err)
```

Order matters: `"identity verification required"` must come **before** any future fallback that would catch it generically. Place it as its own case after the `"no permission"` block to keep the `IDENTITY_VERIFICATION_REQUIRED` reason distinguishable from generic `PERMISSION_DENIED`.

Add a translator unit test confirming the routing.

### Task 9: OpenAPI path wiring

Wire all `/numbers*`, `/available-numbers`, `/providers*`, `/providercalls*`, `/trunks*`, `/routes*` paths per §6.1 baseline:

- `GET /numbers`, `GET /available-numbers`, `GET /providers`, `GET /providercalls`, `GET /trunks`, `GET /routes` → 401, 500
- `GET /numbers/{id}`, `GET /providers/{id}`, `GET /providercalls/{id}`, `GET /trunks/{id}`, `GET /routes/{id}` → 400, 401, 403, 404, 500
- `POST /numbers`, `POST /numbers/renew` → 400, 401, **402**, 500
- `POST /providers`, `POST /providers/setup`, `POST /providercalls`, `POST /trunks`, `POST /routes` → 400, 401, 500 (admin gates produce 403, but the 403 modifier is for ID-bound endpoints; admin gates here are auth-class)

  Actually re-think: even admin-only POST endpoints should declare 403 since `hasPermission` failure produces it. Per §6.1 strict reading, write-no-id is `400, 401, 500` and 403 is only declared on ID-bound endpoints. To match the convention exactly, omit 403 from these POST endpoints and rely on 401 covering "you're not an admin." If reviewer challenges this, reconsider.

- `PUT /numbers/{id}`, `PUT /numbers/{id}/flow-ids`, `PUT /numbers/{id}/metadata`, `DELETE /numbers/{id}`, `PUT /providers/{id}`, `DELETE /providers/{id}`, `DELETE /providercalls/{id}`, `PUT /trunks/{id}`, `DELETE /trunks/{id}`, `PUT /routes/{id}`, `DELETE /routes/{id}` → 400, 401, 403, 404, 500

Regenerate `gens/openapi_server/gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 10: RST catalog — populate billing-manager + add number-manager + provider/trunk/route

Edit `bin-api-manager/docsdev/source/restful_api_errors.rst`:

**`billing-manager` (currently empty placeholder):**
- `INSUFFICIENT_BALANCE` (402) — populated from translator pattern. Reachable today via "insufficient" substring fallback.

**`number-manager` (new section):**
- `NUMBER_NOT_FOUND` (404) — typed-error future; reachable today via "not found" fallback.
- `IDENTITY_VERIFICATION_REQUIRED` (403) — newly wired in this PR. Reachable via translator's "identity verification required" pattern.

**`provider-manager` / `trunk-manager` / `route-manager` (new combined "provisioning admin" section):**
- `PROVIDER_NOT_FOUND`, `TRUNK_NOT_FOUND`, `ROUTE_NOT_FOUND` (404) — reachable today via "not found".
- Generic note: these are admin-only resources; non-admins receive 403 via the "no permission" translator pattern.

**`available-numbers` (no new entries needed):** standard 401/500 from §6.1.

Match the disclaimer style introduced in PR 2/3: enumerate which reasons are reachable today via translator fallback vs which require typed-error migration.

Rebuild Sphinx HTML.

### Task 11: Full verification

Standard 5-step workflow for both `bin-openapi-manager` and `bin-api-manager`.

### Task 12: Push + open PR

Conflict check, push, open PR.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr4-numbers-providers` on every commit.
- No AI attribution.
- `commonoutline.ServiceName*` constants.
- 3-group import layout in test files.
- Reuse `assertMissingAuthIdentity` and `assertErrorResponse` helpers.
- Preserve existing `log.Errorf/Infof` site-specific lines.

## Success criteria

- All 12 tasks committed
- `go test -race ./...` green for both `bin-api-manager` and `bin-openapi-manager`
- `golangci-lint` 0 issues
- Loose `ServerInterface` signatures unchanged
- Billing-manager catalog section populated with `INSUFFICIENT_BALANCE`
- New number-manager and provisioning-admin catalog sections
- All 93 sites converted
- 402 declared on `POST /numbers` and `POST /numbers/renew`
- New `IDENTITY_VERIFICATION_REQUIRED` reason wired and tested
