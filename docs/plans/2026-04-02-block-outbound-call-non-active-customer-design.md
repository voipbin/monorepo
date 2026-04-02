# Block Outbound Calls for Non-Active Customers

## Problem

Currently, only customers with `frozen` status are blocked from making calls (via `ValidateCustomerNotFrozen()` in `bin-call-manager`). Customers with `initial`, `expired`, or `deleted` status can still initiate outbound calls. This is undesirable — only fully activated customers should be able to make outbound calls.

## Approach

Replace `ValidateCustomerNotFrozen()` with two direction-specific validation functions in `bin-call-manager/pkg/callhandler/validate.go`:

- **`ValidateCustomerStatusOutgoing()`** — requires `status == active`. Blocks `initial`, `frozen`, `expired`, `deleted`.
- **`ValidateCustomerStatusIncoming()`** — requires `status == active` or `status == initial`. Initial customers can still receive calls (they're onboarding but should be reachable).

Both functions preserve the existing fail-open behavior: if customer-manager is unavailable, allow the call (billing-manager provides a second enforcement layer).

## Call Sites

There are exactly 2 call sites for `ValidateCustomerNotFrozen()`:

| Call site | File:Line | Direction | Notes |
|-----------|-----------|-----------|-------|
| `CreateCallOutgoing` | `outgoing_call.go:133` | Outgoing | Returns `cu` for subsequent `ValidateCustomerIdentityVerified` |
| `startCallTypeFlow` | `start.go:572` | Incoming | Discards customer (`_`), single funnel for all incoming types |

`startCallTypeFlow` is called from trunk, SIP, registrar, direct queue, and direct flow paths — so changing it covers all incoming call types.

## Changes

### `bin-call-manager/pkg/callhandler/validate.go`

- Remove `ValidateCustomerNotFrozen()`
- Add `ValidateCustomerStatusOutgoing(ctx, customerID) (*Customer, bool)`:
  - Fetch customer via `reqHandler.CustomerV1CustomerGet`
  - Return `(cu, true)` if `status == active`
  - Return `(cu, false)` otherwise
  - Return `(nil, true)` if customer-manager unavailable (fail-open)
- Add `ValidateCustomerStatusIncoming(ctx, customerID) (*Customer, bool)`:
  - Same fetch and fail-open behavior
  - Return `(cu, true)` if `status == active` or `status == initial`
  - Return `(cu, false)` otherwise

### `bin-call-manager/pkg/callhandler/outgoing_call.go`

- Line 133: Replace `ValidateCustomerNotFrozen` with `ValidateCustomerStatusOutgoing`
- Update variable name and error message to `"customer account is not active"`

### `bin-call-manager/pkg/callhandler/start.go`

- Line 572: Replace `ValidateCustomerNotFrozen` with `ValidateCustomerStatusIncoming`
- Update error log message

### Tests — New `validate_test.go`

Add unit tests for both new functions covering:

| Scenario | Outgoing | Incoming |
|----------|----------|----------|
| `active` | allowed | allowed |
| `initial` | rejected | allowed |
| `frozen` | rejected | rejected |
| `expired` | rejected | rejected |
| `deleted` | rejected | rejected |
| customer-manager unavailable | fail-open (allowed) | fail-open (allowed) |

### Tests — Existing (comment updates only)

- `start_test.go` (lines 255, 452, 646): Update comments from `ValidateCustomerNotFrozen` to `ValidateCustomerStatusIncoming`
- `outgoing_call_test.go`, `start_incoming_domain_type_*_test.go`: No changes needed (already use `StatusActive`)

## What does NOT change

- `ValidateCustomerIdentityVerified()` — still runs separately for PSTN calls
- `CallHandler` interface — `ValidateCustomerNotFrozen` is a private method, not in the interface
- API middleware frozen check in `bin-api-manager/lib/middleware/authenticate.go` — still blocks frozen accounts at the API level for all requests
- No other services affected
- No mock regeneration needed
