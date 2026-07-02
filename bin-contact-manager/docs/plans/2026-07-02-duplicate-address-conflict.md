# Design: distinguishable conflict status on duplicate contact address (#1044)

## Problem

`AddAddress` (pkg/contacthandler/contact.go) does not classify the DB error
it receives from `db.AddressCreate`. `contact_addresses` enforces a unique
index on `(customer_id, type, target)`, so adding an address that already
exists for the customer always fails at the DB layer with a duplicate-key
error. Both listenhandler call sites that reach `AddAddress` collapse every
error into a bare 500:

- `processV1ContactsAddressesPost` (pkg/listenhandler/v1_contacts_addresses.go,
  the endpoint named in the issue: `POST /v1/contacts/{id}/addresses`)
- `processV1ContactAddressesPost` (pkg/listenhandler/v1_contact_addresses.go,
  the sibling independent-resource endpoint: `POST /v1/contact_addresses`
  with `contact_id` in the body). This one calls the identical
  `contactHandler.AddAddress` for the resolved-contact branch and has the
  exact same bug, just not named in the issue text.

Both must be fixed together or the platform ships an inconsistency where one
endpoint returns 409 and the sibling still returns bare 500 for the same
underlying DB conflict.

## Reference pattern already in the codebase

`ClaimAddress` / `dbhandler.AddressClaim` already established the intended
shape for this class of bug:

- dbhandler owns a sentinel error (`ErrConflict`) and returns it on the
  specific DB condition it detects.
- contacthandler classifies with `stderrors.Is(err, dbhandler.ErrConflict)`
  and maps it to a typed `cerrors.AlreadyExists(...)`.
- listenhandler routes the typed error through `errorResponse(err)`, which
  already knows how to turn a `*cerrors.VoipbinError` into the correct
  `sock.Response` (see `errorResponse` in pkg/listenhandler/main.go, and
  `cerrors.AlreadyExists` doc comment confirming it maps to 409).

`InteractionCreate` (pkg/dbhandler/interaction.go) already has the exact
MySQL-errno-1062 / SQLite-"UNIQUE constraint failed" detection idiom this
fix needs, just used for a different purpose (idempotent silent-ignore
instead of surfacing a conflict).

## Plan

### 1. dbhandler: detect the unique-constraint violation on `AddressCreate`

New sentinel error, distinct from `ErrConflict` (whose existing message
"address already claimed" is specific to the claim flow and would be
misleading in AddAddress logs/tests; `ErrConflict` is also load-bearing
elsewhere, e.g. `Test_ClaimAddress_Conflict`, so reusing it for a different
condition risks confusing that test's intent).

`pkg/dbhandler/address.go` currently imports neither `strings` nor
`mysql_driver "github.com/go-sql-driver/mysql"` — both must be added,
mirroring the exact imports already present in `pkg/dbhandler/interaction.go`.

```go
// pkg/dbhandler/main.go
ErrDuplicateTarget = fmt.Errorf("address already exists for this customer")
```

In `AddressCreate` (pkg/dbhandler/address.go), wrap the `h.db.Exec` error
check with the same MySQL 1062 / SQLite UNIQUE-constraint detection used in
`InteractionCreate`, returning `ErrDuplicateTarget` instead of the generic
wrapped error.

### 2. contacthandler: classify and map to a typed conflict

In `AddAddress` (pkg/contacthandler/contact.go), after
`h.db.AddressCreate(ctx, a)`:

```go
if err := h.db.AddressCreate(ctx, a); err != nil {
    if stderrors.Is(err, dbhandler.ErrDuplicateTarget) {
        return nil, cerrors.AlreadyExists(
            commonoutline.ServiceNameContactManager,
            "ADDRESS_ALREADY_EXISTS",
            "An address with this type and target already exists for this customer.",
        ).Wrap(err)
    }
    return nil, fmt.Errorf("could not create address: %w", err)
}
```

Reason code `ADDRESS_ALREADY_EXISTS` is new and distinct from the existing
`ADDRESS_ALREADY_CLAIMED` (claim flow) so callers can tell the two conflict
classes apart if they ever need to.

### 3. listenhandler: route through `errorResponse` instead of a bare 500

Both call sites change from:

```go
tmp, err := h.contactHandler.AddAddress(ctx, contactID, address)
if err != nil {
    log.Errorf("Could not add address. err: %v", err)
    return simpleResponse(500), nil
}
```

to:

```go
tmp, err := h.contactHandler.AddAddress(ctx, contactID, address)
if err != nil {
    log.Errorf("Could not add address. err: %v", err)
    return errorResponse(err), nil
}
```

Files: `pkg/listenhandler/v1_contacts_addresses.go` (`processV1ContactsAddressesPost`)
and `pkg/listenhandler/v1_contact_addresses.go` (`processV1ContactAddressesPost`,
resolved-contact branch only, i.e. the branch that calls
`contactHandler.AddAddress`).

**Decided scope (not deferred to implementation):** the unresolved-address
branch of `processV1ContactAddressesPost` (calls `CreateUnresolvedAddress`,
lines ~100-110 of v1_contact_addresses.go) is explicitly OUT OF SCOPE for
this PR and keeps its current `simpleResponse(500)`. `CreateUnresolvedAddress`
also calls `h.db.AddressCreate` and will transitively receive the same new
`dbhandler.ErrDuplicateTarget` sentinel, but since it does not classify it,
the error still falls through to `fmt.Errorf(...): %w` and 500 unchanged.
This is NOT a regression (behavior is identical to today), just a known,
explicitly-tracked follow-up: classifying `CreateUnresolvedAddress` the same
way is a candidate for a dedicated follow-up issue/PR, not bundled here,
since the original issue #1044 scopes only to the resolved `AddAddress`
path.

**Documented side effect (intentional, in-scope):** `AddAddress`
(contact.go:261-264) calls `h.db.ContactGet(ctx, contactID)` first and
returns that error unwrapped if the contact does not exist. Today this is
masked as bare 500 by both call sites. After switching to
`errorResponse(err)`, a nonexistent `contactID` will now correctly surface
as **404** (via `errorResponse`'s `dbhandler.ErrNotFound` fallback) instead
of 500. This is a second, small, correct behavior change riding along with
the primary fix — called out explicitly here so it is not mistaken for an
unreviewed regression. The new listenhandler-routing tests (section 4)
must include a case asserting this 404 path in addition to the new 409
path.

### 4. Regression tests

- `pkg/dbhandler/address_test.go`: `Test_AddressCreate_Duplicate` — create
  once, attempt a second create with same `(customer_id, type, target)`,
  assert `errors.Is(err, ErrDuplicateTarget)`.
- `pkg/contacthandler/contact_test.go`: extend/add a case asserting
  `AddAddress` maps a duplicate-target dbhandler error to a
  `cerrors.VoipbinError` with `Status == cerrors.StatusAlreadyExists` and
  `Reason == "ADDRESS_ALREADY_EXISTS"`.
- `pkg/listenhandler/v1_contacts_addresses_test.go` (new or existing file):
  routing-level tests for `processV1ContactsAddressesPost` asserting:
  (a) 409 (not 500) when `contactHandler.AddAddress` returns the typed
  conflict, following the pattern used for issue #1042's regression test;
  (b) 404 (not 500) when `contactHandler.AddAddress` returns
  `dbhandler.ErrNotFound` (documented side effect above).
- Same two routing-level assertions (409 and 404) for
  `processV1ContactAddressesPost`'s resolved-contact branch.
- Confirm the existing `IsPrimary`-reset-then-create path (contact.go:279-283)
  is unaffected: `AddAddress`'s test coverage should include a case where
  `IsPrimary=true` and `AddressCreate` fails with the duplicate-target
  conflict, asserting the typed error is still returned correctly (primaries
  having already been reset with no rollback is pre-existing behavior, not
  newly introduced or worsened by this fix — note this in the test comment
  so a future reviewer does not mistake it for a new bug).

## Not in scope

- `CreateUnresolvedAddress` duplicate handling — explicitly deferred to a
  follow-up PR/issue, per the decided-scope note in section 3 (firm
  decision, not an open question). Not a blocker for the primary fix.
- Square-admin frontend changes (separate PR, referenced in the issue as
  the downstream consumer of this fix).
