# Move default outgoing source number ownership: Customer → OutboundConfig

**Date:** 2026-05-08
**Status:** Approved (4-round review-and-fix loop with architect + critic agents)
**Branch:** `NOJIRA-Move-default-outgoing-source-to-outboundconfig`

---

## Goal

Move ownership of the per-customer default outgoing source number from `Customer` (in `bin-customer-manager`) to `OutboundConfig` (in `bin-call-manager`). After this change, the source-number fallback at call time reads from the customer's OutboundConfig only; if the customer has no OutboundConfig, the call is rejected.

## Non-goals

- Changing OutboundConfig cardinality from 1:1 (one OutboundConfig per customer) to N:1.
- Adding back-compat shims or a deprecation period for the old Customer endpoint. This is a hard cut.
- Restructuring the `OutboundConfig` Update API surface beyond extending the existing `UpdateRequest`.

## Decisions made (with the user)

| Decision | Choice |
|----------|--------|
| Cardinality | 1:1 — one OutboundConfig per customer (matches current `GetByCustomerID` shape) |
| Missing OutboundConfig at call time | Reject the call (no fallback to legacy field) |
| Migration strategy | Hard cut, no dual-read, no deprecation period |
| Existing Customer values | Copied into the new OutboundConfig column during Mig 1 (data preservation; not "backfill" of new rows) |

---

## Data model changes

### `bin-call-manager/models/outboundconfig/outboundconfig.go`

```go
type OutboundConfig struct {
    // ... existing fields ...
    DefaultOutgoingSourceNumberID uuid.UUID `json:"default_outgoing_source_number_id" db:"default_outgoing_source_number_id,uuid"`
}

type UpdateRequest struct {
    // ... existing pointer fields ...
    DefaultOutgoingSourceNumberID *uuid.UUID `json:"default_outgoing_source_number_id,omitempty"`
}
```

**Pointer semantics for `UpdateRequest.DefaultOutgoingSourceNumberID`:**
- `nil` pointer → no change.
- pointer → `uuid.Nil` → clear (set to `uuid.Nil`).
- pointer → valid UUID → set (after validation, see below).

This matches the existing pattern documented at `bin-call-manager/models/outboundconfig/outboundconfig.go:23-29`.

### `bin-call-manager/models/outboundconfig/webhook.go`

Add `DefaultOutgoingSourceNumberID` to `WebhookMessage` and copy it in `ConvertWebhookMessage`. RST struct docs (`*_struct_*.rst`) must match `WebhookMessage`, not the internal struct.

### `bin-customer-manager/models/customer/`

Remove from `customer.go`, `webhook.go`, `field.go`:
- `Customer.DefaultOutgoingSourceNumberID` field and `db:` tag.
- `WebhookMessage.DefaultOutgoingSourceNumberID` and the corresponding line in `ConvertWebhookMessage`.
- `FieldDefaultOutgoingSourceNumberID` constant.

---

## Schema changes (Alembic, two migrations)

Migration files MUST be generated via `alembic -c alembic.ini revision -m "..."` — never hand-pick revision IDs. Table name is `call_outbound_configs` (plural; verified at `bin-dbscheme-manager/bin-manager/main/versions/60e68bfd6442_outbound_configs_create_table.py`).

### Mig 1 — `add_default_outgoing_source_number_id_to_call_outbound_configs.py`

Three-step pattern to safely add a `BINARY(16) NOT NULL` column with data preservation:

```sql
-- Step 1: ADD as NULL so existing rows accept it.
ALTER TABLE call_outbound_configs
  ADD COLUMN default_outgoing_source_number_id BINARY(16) NULL;

-- Step 2: Copy existing customer values onto outbound_config rows.
-- No tm_delete filter — soft-deleted customers also have call_outbound_configs rows
-- (UNIQUE KEY uq_customer_id was bootstrapped per-customer at migration dd500d4e8cb7).
UPDATE call_outbound_configs o
  JOIN customer_customers c ON o.customer_id = c.id
  SET o.default_outgoing_source_number_id = c.default_outgoing_source_number_id;

-- Step 2b: Defensive zero-fill for any orphan rows where the JOIN missed
-- (e.g., outbound_config rows whose customer was hard-deleted somehow).
UPDATE call_outbound_configs
  SET default_outgoing_source_number_id = UNHEX('00000000000000000000000000000000')
  WHERE default_outgoing_source_number_id IS NULL;

-- Step 3: Tighten to NOT NULL.
ALTER TABLE call_outbound_configs
  MODIFY default_outgoing_source_number_id BINARY(16) NOT NULL;
```

Downgrade:
```sql
ALTER TABLE call_outbound_configs DROP COLUMN default_outgoing_source_number_id;
```

### Mig 2 — `drop_default_outgoing_source_number_id_from_customer_customers.py`

```sql
ALTER TABLE customer_customers DROP COLUMN IF EXISTS default_outgoing_source_number_id;
```

`IF EXISTS` is a defensive idempotency guard against partial-state replays. Downgrade re-adds the column as `BINARY(16) NOT NULL DEFAULT (UNHEX('00000000000000000000000000000000'))` (data is not restored — Mig 2 is the one-way door).

---

## Call-time flow changes

### `bin-call-manager/pkg/callhandler/outgoing_call.go`

**`getValidatedSourceForOutgoingCall` signature (line 747)** — add `outboundCfg *outboundconfig.OutboundConfig`:

```go
getValidatedSourceForOutgoingCall(
    ctx context.Context,
    source commonaddress.Address,
    destination commonaddress.Address,
    cu *cucustomer.Customer,
    outboundCfg *outboundconfig.OutboundConfig,
    metadata map[string]interface{},
) *commonaddress.Address
```

`cu` stays — still used for caller-supplied source ownership validation at lines 786–800.

**Replace the fallback block at lines 805–818.** Use `NumberV1NumberList` with the same filters as the caller-supplied path (lines 786–797), not bare `NumberV1NumberGet`. This protects against soft-deleted, released, ownership-changed, virtual, or inactive numbers — `NumberV1NumberGet` does not filter `tm_delete` (verified at `bin-number-manager/pkg/dbhandler/number.go:155`).

```go
// outboundCfg may be nil (non-tel destination, internal-system caller, or
// transient fetch failure handled at the call site).
if outboundCfg == nil || outboundCfg.DefaultOutgoingSourceNumberID == uuid.Nil {
    log.Infof("No valid source number available. Rejecting call.")
    return nil
}

filters := map[nmnumber.Field]any{
    nmnumber.FieldCustomerID: cu.ID,
    nmnumber.FieldID:         outboundCfg.DefaultOutgoingSourceNumberID,
    nmnumber.FieldType:       nmnumber.TypeNormal,
    nmnumber.FieldStatus:     nmnumber.StatusActive,
    nmnumber.FieldDeleted:    false,
}
nums, err := h.reqHandler.NumberV1NumberList(ctx, "", 1, filters)
if err != nil || len(nums) == 0 {
    log.Errorf("Default outgoing source number is not valid. number_id: %s, err: %v",
        outboundCfg.DefaultOutgoingSourceNumberID, err)
    return nil
}

defaultNum := nums[0]
return &commonaddress.Address{
    Type:       commonaddress.TypeTel,
    Target:     defaultNum.Number,
    TargetName: defaultNum.Number,
}
```

**Caller wiring (around line 181):** the OutboundConfig is already fetched once for codec embed and whitelist enforcement. Pass it through to `getValidatedSourceForOutgoingCall`. For the non-tel and internal-system branches that don't fetch the config, pass `nil`; those branches return early before the fallback runs.

**Fail-closed on cfg fetch error (line 181):** today the code says `outboundCfg = nil` and silently continues. Change to **reject the call** when `cfgErr != nil`. The silent-continue masks transient failures and leaves the customer with whatever fallback exists. With hard-cut, fail fast.

### `bin-call-manager/pkg/outboundconfighandler/`

**`validate.go`** — change signature to `validateUpdateRequest(ctx context.Context, customerID uuid.UUID, req *UpdateRequest) error`. Both `Create` and `Update` pass it.

When `req.DefaultOutgoingSourceNumberID != nil` and `*req.DefaultOutgoingSourceNumberID != uuid.Nil`:
- Call `NumberV1NumberList` with filters `(customer_id=customerID, id=*req.DefaultOutgoingSourceNumberID, type=normal, status=active, deleted=false)`.
- If list returns zero rows, return a typed error mapped to HTTP 400 by api-manager.

This catches typos/mistakes at config time and gives 400s instead of call-time 5xx. It is **not** the only line of defense — call-time re-validation above is the security gate, since cached OutboundConfig values may be stale.

**`outbound_config.go`** — `Update` path must fetch `cfg.CustomerID` via `db.OutboundConfigGetByID(ctx, id)` before calling the validator. `applyUpdateRequest` gains a branch that copies `*req.DefaultOutgoingSourceNumberID` to the struct when non-nil.

### `bin-call-manager/models/call/metadata.go:25`

Stale comment references `Customer.DefaultOutgoingSourceNumberID`. Update or remove.

---

## Cascading removals

### `bin-customer-manager`

- `models/customer/customer.go`, `webhook.go`, `field.go` — remove field, db tag, webhook field, Field constant, `ConvertWebhookMessage` line.
- `pkg/customerhandler/main.go`, `db.go`, `mock_main.go` — remove `UpdateDefaultOutgoingSourceNumberID` interface method, implementation, and mock.
- `pkg/listenhandler/main.go` — remove `regV1CustomersIDIsDefaultOutgoingSourceNumberID` and the case branch dispatching to `processV1CustomersIDDefaultOutgoingSourceNumberIDPut`.
- Remove `processV1CustomersIDDefaultOutgoingSourceNumberIDPut` handler.
- Update fixtures in `pkg/listenhandler/v1_customers_test.go:56,84,171,242,304`.

### `bin-common-handler`

- `pkg/requesthandler/main.go`, `mock_main.go`, `customer_customer.go` — remove `CustomerV1CustomerUpdateDefaultOutgoingSourceNumberID` interface method, implementation, mock.
- `pkg/sock/sockrequest` — remove `V1DataCustomersIDDefaultOutgoingSourceNumberIDPut` request struct.

### `bin-api-manager`

- Remove the public-facing route `PUT /customers/<id>/default_outgoing_source_number_id`.
- Remove `servicehandler.CustomerUpdateDefaultOutgoingSourceNumberID` (admin) and `CustomerSelfUpdateDefaultOutgoingSourceNumberID` (self) plus mocks.
- **Customer-create hardening at `pkg/servicehandler/customer.go:76,735`**: change OutboundConfig auto-create from fire-and-forget (current: `if cfgErr != nil { log.Warnf(...) }`) to blocking with retry. On permanent failure, return error and fail customer creation. Prevents creating customers who can't dial out.

### `bin-openapi-manager`

- Delete `openapi/paths/customer/default_outgoing_source_number_id.yaml`.
- Remove `default_outgoing_source_number_id` from the Customer schema and any response models.
- Add `default_outgoing_source_number_id` to the OutboundConfig schema and update request body. Use **non-nullable** `format: uuid` (matches the existing convention; no nullable-uuid prior art in the codebase).
- Field `description:` must explicitly state: *"`00000000-0000-0000-0000-000000000000` clears the default. Field omitted from request body = no change."*
- Regenerate `gens/models/gen.go`.

### RST docs (`bin-api-manager/docsdev/source/`)

- Add `default_outgoing_source_number_id` to outbound-config struct/overview/tutorial pages. Mirror the OpenAPI description sentence so external integrators see the same contract.
- Remove the field from customer struct doc and remove the dedicated endpoint section from the customer tutorial.
- Clean rebuild and force-add the build output: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`, then `git add -f bin-api-manager/docsdev/build/`.

---

## Observability

New Prometheus counter in `bin-call-manager` (registered alongside existing call-manager metrics):

```go
// outbound_config_fetch_error_total counts fetch failures from outboundConfigHandler.GetByCustomerID.
// Cardinality bound: ≤1 series per active customer × error_type. At current scale (~K customers, ~handful of error types)
// this is well within Prometheus best-practice limits. Revisit if customer count grows beyond ~100K.
outboundConfigFetchErrorTotal = prometheus.NewCounterVec(...)
```

**SLO trigger for adding a circuit-breaker fallback (per `docs/patterns/circuit-breaker.md`):** > 10 errors/min sustained over 5 min. Until then, fail-closed is the policy.

---

## Deployment ordering (strict)

| Step | Action | Verification |
|------|--------|--------------|
| 1 | Apply Mig 1 (3-step ADD/UPDATE/UPDATE/MODIFY) | Schema check; row count matches `customer_customers WHERE tm_delete IS NULL` |
| 2 | Deploy `bin-call-manager` | Pod readiness; `outbound_config_fetch_error_total` baseline |
| 3 | Deploy `bin-openapi-manager` | Generated types present |
| 4 | Deploy `bin-api-manager` | Customer-side route gone (404); OutboundConfig PUT accepts new field |
| 4.5 | Manually update + deploy api-validator with new tests | Positive (set valid number → 200) and negative (set invalid number → 400) tests pass |
| 5 | Deploy `bin-customer-manager` (also rebuilds `bin-common-handler` consumers) | (a) `kubectl rollout status deployment/customer-manager` complete; (b) RabbitMQ consumer-count for `bin-manager.customer-manager.request` shows only new pods; (c) metrics show 0 in-flight RPCs to deleted route |
| 6 | Apply Mig 2 — only after step 5 stable in production for **≥24 hours** with zero rollback signals | Column dropped; no `Unknown column` errors in customer-manager logs |

The 24h soak before Mig 2 is the explicit one-way door.

---

## Rollback

| Failure point | Rollback procedure |
|---------------|--------------------|
| After Mig 1 | `alembic downgrade -1` reverts Mig 1; no code changes deployed yet |
| After step 2 | Revert call-manager image; the new `call_outbound_configs` column remains harmless (extra column ignored by old code) |
| After steps 3–4.5 | Revert images; same as above |
| After step 5 | One-way door — code that previously read `Customer.DefaultOutgoingSourceNumberID` is gone. Reverting requires re-adding the field to customer-manager source AND re-running customer-manager deploy. The column is still present in `customer_customers` until step 6, so SELECT still works during this window. Rolling back here is painful but possible |
| After step 6 | Schema-level rollback: re-add column via forward migration; data is lost. Customer.DefaultOutgoingSourceNumberID values are unrecoverable from the customer table — but they were copied into `call_outbound_configs.default_outgoing_source_number_id` during Mig 1 step 2, so a recovery path exists if needed |

---

## Accepted risks (explicit)

1. **Brief 5xx window during step 4↔5** for clients that retry `PUT /customers/<id>/default_outgoing_source_number_id`. Per the user's "hard cut" decision. Mitigation: deploy step 5 promptly after step 4; the existing `bin-api-manager` retry/backoff will absorb the small number of 404s.
2. **Call-time fail-closed on transient OutboundConfig DB error**: a sustained 30s DB hiccup → all outgoing tel calls reject. Per the user's "hard cut" intent. SLO trigger above defines when to add a circuit-breaker fallback.
3. **No external clients consume `customer.WebhookMessage.default_outgoing_source_number_id`**: verified — the only subscriber, `bin-webhook-manager/pkg/subscribehandler/customermanager.go`, uses Go's default JSON decoder (tolerant of missing fields). Removal is safe.

---

## Test plan

Per-service unit tests run via the standard verification workflow (`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`).

### `bin-call-manager/pkg/callhandler/outgoing_call_test.go`

Table-driven cases for `getValidatedSourceForOutgoingCall`:
- valid caller-supplied source → returns source unchanged
- invalid source, OutboundConfig has valid `DefaultOutgoingSourceNumberID` → returns resolved address
- invalid source, OutboundConfig is `nil` → returns `nil` (reject)
- invalid source, OutboundConfig present but `DefaultOutgoingSourceNumberID == uuid.Nil` → reject
- invalid source, default points to a soft-deleted number → reject (regression guard for the call-time re-validation)
- invalid source, default points to a number owned by a different customer → reject
- invalid source, default points to a virtual number → reject
- invalid source, default points to an inactive number → reject
- **Number valid at config-set, then released, then call attempted → reject** (the dual-gate invariant — locks the design against future "optimization" that drops the call-time check)
- cfg fetch returns transient error → reject (fail-closed)

Delete the obsolete `cu.DefaultOutgoingSourceNumberID` test cases.

### `bin-call-manager/pkg/outboundconfighandler/outbound_config_test.go`

`Create` and `Update` matrix:
- `DefaultOutgoingSourceNumberID` omitted (nil pointer) → no change to existing struct
- Pointer to `uuid.Nil` → field cleared
- Pointer to valid number (own customer, active, normal, not deleted) → field set
- Pointer to non-existent number → reject
- Pointer to number owned by a different customer → reject
- Pointer to virtual number → reject
- Pointer to inactive number → reject
- Pointer to soft-deleted number → reject

### `bin-customer-manager/pkg/listenhandler/v1_customers_test.go`

Delete cases for `processV1CustomersIDDefaultOutgoingSourceNumberIDPut`. Update fixtures at lines 56, 84, 171, 242, 304 to drop the `default_outgoing_source_number_id` field.

### `bin-customer-manager/pkg/customerhandler/db_test.go`

Delete the `UpdateDefaultOutgoingSourceNumberID` test.

### `bin-common-handler/pkg/requesthandler/customer_customer_test.go`

Delete the `CustomerV1CustomerUpdateDefaultOutgoingSourceNumberID` test. Regenerate mocks via `go generate ./...`.

### `bin-api-manager/pkg/servicehandler/customer_test.go`

- Delete tests for `CustomerUpdateDefaultOutgoingSourceNumberID` and `CustomerSelfUpdateDefaultOutgoingSourceNumberID`.
- Add test for the customer-create hardening: OutboundConfig auto-create permanent failure → customer creation fails with appropriate error.

### api-validator (`monorepo-monitoring/api-validator/`)

Hand-written tests (api-validator is not generated from the OpenAPI spec):
- Positive: set `default_outgoing_source_number_id` on outbound-config to a valid customer-owned active number → 200.
- Negative: set to a non-existent number → 400.
- Negative: set to another customer's number → 400.
- Cleanup: previous customer-side endpoint tests removed.

---

## Audit findings (verified during design)

- **Only `bin-api-manager` calls `CustomerV1CustomerUpdateDefaultOutgoingSourceNumberID`** in production source (lines 560, 670 of `bin-api-manager/pkg/servicehandler/customer.go`).
- **No other monorepo service reads `Customer.DefaultOutgoingSourceNumberID`** — grep across all 34 services returned zero hits outside customer-manager / call-manager / common-handler / api-manager / openapi-manager.
- **Auto-create OutboundConfig already exists** for new customers at `bin-api-manager/pkg/servicehandler/customer.go:76,735` — the design hardens it from fire-and-forget to blocking-with-retry.
- **`customer-manager` SELECTs via `commondatabasehandler.GetDBFields(&customer.Customer{})`** (reflection over `db:` tags) at `bin-customer-manager/pkg/dbhandler/customer.go:116,192,391`. After Mig 2 drops the column, old customer-manager pods would fail SELECT — hence the hard kubectl-rollout-status + RabbitMQ-consumer-count gate before Mig 2.
- **Existing migration `dd500d4e8cb7` already bootstraps an empty `call_outbound_configs` row for every active customer** (`INSERT IGNORE ... FROM customer_customers WHERE c.tm_delete IS NULL`) with `UNIQUE KEY uq_customer_id`. No additional pre-flight bootstrap is needed.
- **`bin-webhook-manager/pkg/subscribehandler/customermanager.go`** is the only customer-event subscriber; uses Go's default JSON decoder (tolerant of missing fields). Field removal is safe for subscribers.
