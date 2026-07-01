# Unresolved contact_addresses: create, list, claim (backend)

Status: Draft

## 1. Problem statement

GitHub issue #1040 (`NOJIRA-Support-unresolved-contact-address-reuse`)
requests that Square-admin's contact address UI let an agent pick from a
pool of addresses "not yet attached to any contact in the customer account"
when adding an address to a contact, with free-text custom input as a
fallback. If the picked address is already attached to a *different*
contact, the UI must show an error, not silently move or copy it.

The `contact_addresses` table already has the schema support for this: the
`contact_id` column is `BINARY(16) DEFAULT NULL -- NULL = unresolved` (see
`bin-dbscheme-manager/bin-manager/main/versions/
ac5d4e18060c_contact_crm_create_tables.py` line 58, and the design rationale
in `docs/plans/2026-06-26-add-contact-crm-interaction-timeline-design.md`
§3.1: "contact_id binary(16) nullable (NULL = unresolved)"). This slot was
seeded at CRM-timeline design time (VOIP-1206) for a different original
purpose (attributing interactions to contacts later), but no producer or
consumer code path exists today that actually creates or lists a
`contact_id IS NULL` row.

Current state, verified against source (2026-07-02):

- `POST /v1/contact_addresses` (`bin-contact-manager/pkg/listenhandler/
  v1_contact_addresses.go` lines 74-115) requires `contact_id` and returns
  400 if absent (line 87-90: `if reqData.ContactID == uuid.Nil { ... return
  simpleResponse(400) }`).
- `GET /v1/contact_addresses` (same file, lines 18-72) supports an optional
  `contact_id` filter that, when present, scopes results to a single
  contact. There is no way to request "rows where contact_id IS NULL".
- `PUT /v1/contact_addresses/{id}` (`request.AddressUpdate`, lines 55-60 of
  `pkg/listenhandler/models/request/contacts.go`) only accepts
  `target`/`name`/`detail`/`is_primary`. There is no way to set `contact_id`
  on an existing row — i.e. no way to "claim" an unresolved address into a
  contact.
- **Critical write-path gap** (found during this design's investigation,
  not previously documented): `dbhandler.AddressCreate`
  (`pkg/dbhandler/address.go` lines 107-136) unconditionally does
  `"contact_id": a.ContactID.Bytes()` in the `SetMap`. `uuid.Nil.Bytes()` is
  16 zero bytes, **not** SQL `NULL`. If `POST /v1/contact_addresses` were
  simply allowed to skip the `contact_id` presence check without also
  fixing this insert, every "unresolved" row would silently write a
  fake all-zero `contact_id`, not a real `NULL` — breaking the entire
  "unresolved pool" concept at the storage layer. This must be fixed as
  part of this change, not assumed to already work.
- **Read-path is already correct and requires no change**: `scanFullAddressRow`
  (`pkg/dbhandler/address.go` lines 48-64) reads through
  `commondatabasehandler.ScanRow`, whose `copyUUID()` helper
  (`bin-common-handler/pkg/databasehandler/mapping.go` lines 380-392) already
  converts a SQL `NULL` column to `uuid.Nil` in Go
  (`t.fieldVal.Set(reflect.ValueOf(uuid.Nil))` on the `!nullStr.Valid`
  branch). So on the read side, "unresolved" is already transparently
  `uuid.Nil` in the Go `contact.Address.ContactID` field — no dbhandler
  read-path change needed, only the write path (`AddressCreate`) needs a
  fix.

## 2. Goals

1. `POST /v1/contact_addresses` accepts a request with `contact_id` omitted
   (or explicitly `uuid.Nil`), creating an "unresolved" address row with a
   real SQL `NULL` `contact_id` for the given `customer_id`.
2. `GET /v1/contact_addresses` supports listing only unresolved addresses
   for a customer (the "pool" the Square-admin picker will search),
   independent of the existing `contact_id`-scoped filter.
3. A new claim operation lets a caller attach a previously-unresolved
   address to a specific contact by ID, subject to:
   - the address must currently be unresolved (`contact_id IS NULL`);
     claiming an address that is already attached to a *different* contact
     returns a conflict, never a silent move/copy (issue #1040 constraint).
   - the target contact must belong to the same `customer_id` as the
     address (tenant isolation).
   - the existing `is_primary` uniqueness/CHECK constraints
     (`chk_contact_addresses_primary_resolved`,
     `idx_contact_addresses_primary`) continue to hold; claiming does not
     implicitly set `is_primary`.
3. `bin-openapi-manager` spec reflects all three changes so
   `bin-api-manager` and generated client types stay in sync.

## 3. Non-goals (explicit scope cuts)

- **No Square-admin UI work in this PR.** The searchable combobox +
  custom-input UI described in issue #1040 is a separate, subsequent PR in
  `monorepo-javascript` once this backend surface exists. This PR is
  backend-only (`bin-contact-manager` + `bin-openapi-manager`).
- **No move/copy semantics.** Per issue #1040's explicit constraint,
  claiming an address already attached to a *different* contact is
  rejected with an error. There is no "reassign" operation in this PR.
- **No cross-customer search.** The unresolved pool is always scoped to
  the caller's `customer_id`; there is no cross-tenant address lookup.
- **No new DB migration.** The `contact_id NULL` schema support already
  exists (VOIP-1206/ac5d4e18060c). This PR is pure application-layer work.
- **No change to `AddAddress`/`UpdateAddress`/`RemoveAddress` on
  `contactHandler`** (the contact-nested address paths at
  `/v1/contacts/{id}/addresses`). Those remain contact-scoped only; the new
  unresolved-pool operations live entirely under the independent
  `/v1/contact_addresses` resource, consistent with the existing split
  between `v1_contacts_addresses.go` (nested, contact-scoped) and
  `v1_contact_addresses.go` (independent resource) documented in the
  contact-address API design from 2026-06.
- **No webhook/event schema change.** `contact_updated` continues to fire
  on claim (since claiming changes contact address membership), reusing
  the existing `contactUpdateToCache` + `publishEvent` pattern already used
  by `AddAddress`/`RemoveAddress`. No new event type.

## 4. Affected files

**Architecture correction from Draft v1 (found during Phase 1.6 point-check,
before this doc's first design review):** `customer_id` on an independent
`/contact_addresses` write is **not** a client-supplied body field. Per
`bin-api-manager`'s CLAUDE.md: "Auth and ownership checks belong ONLY in
bin-api-manager. Backend services never check JWT or customer ownership."
Verified against `bin-api-manager/pkg/servicehandler/contact_address.go`:
every existing `ContactAddress*` servicehandler method receives
`a *auth.AuthIdentity` (already carrying the authenticated `a.CustomerID`)
and either (a) uses `a.CustomerID` directly (`ContactAddressList`,
`ContactAddressGet`), or (b) for `ContactAddressCreateIndependent`,
resolves ownership by fetching the **contact** first (`h.contactGet(ctx,
contactID)`) and checking `ct.CustomerID` — a pattern that has no analog
when there is no contact to fetch (the unresolved-creation case). This
means unresolved creation needs a **new servicehandler code path** that
uses `a.CustomerID` directly (like `ContactAddressList` does), not the
contact-ownership-check pattern `ContactAddressCreateIndependent` uses
today. This is a 4-layer change, not a 1-layer change as Draft v1 assumed:

| File | Why |
|---|---|
| `bin-contact-manager/pkg/dbhandler/address.go` | Fix `AddressCreate` to write real SQL `NULL` for `contact_id` when unresolved, and skip the doomed `contactUpdateToCache(ctx, uuid.Nil)` tail call in that case (§5.1). Add `AddressClaim` (§5.3). Add unresolved-pool filter to `AddressList` (§5.2). |
| `bin-contact-manager/pkg/dbhandler/main.go` | `DBHandler` interface: add `AddressClaim` signature (manual edit required before mock regeneration succeeds — `go generate` reads this file to produce `mock_main.go`). **Also add a new `ErrConflict = fmt.Errorf("address already claimed")` sentinel to the existing `var (...)` block that currently declares only `ErrNotFound` (main.go:68-71) — design review iter-4 finding: this sentinel is referenced by both the `AddressClaim` dbhandler pseudocode and `ClaimAddress` handler pseudocode in §5.3 but was never previously listed as a concrete file edit here, so it did not actually exist anywhere in the codebase.** |
| `bin-contact-manager/pkg/listenhandler/v1_contact_addresses.go` | Relax the `POST` contact_id-required check (§5.1, now driven by a `customer_id` field the listenhandler already trusts from the internal RPC caller — see §5.1 corrected flow); add `unresolved=true` query param handling to `GET` (§5.2); add the claim route (§5.3). |
| `bin-contact-manager/pkg/listenhandler/models/request/contacts.go` | `ContactAddressCreate.ContactID` stays a plain `uuid.UUID` (no Go type change — `uuid.Nil` already means "absent" at this layer, consistent with `contactID == uuid.Nil` checks used throughout the file). Add `CustomerID uuid.UUID` to `ContactAddressCreate` (populated by the internal RPC caller, i.e. `bin-api-manager`'s servicehandler — NOT the external REST client, see §5.1). Add a new `ContactAddressClaim` request struct (§5.3). |
| `bin-contact-manager/pkg/contacthandler/contact.go` | **Corrected placement (design review iter-1 finding #2)**: add `CreateUnresolvedAddress` and `ClaimAddress` to `contactHandler`, NOT `addresshandler`. `addresshandler`'s own doc comment (`pkg/addresshandler/main.go` lines 14-17) explicitly states write operations that publish contact events must live in `ContactHandler`, and `publishEvent` (`contacthandler/event.go` line 14) is a private method on `contactHandler` only, taking `*contact.Contact` — `addressHandler` has no access to it and no `*contact.Contact` return path. This mirrors the existing `AddAddress`/`UpdateAddress`/`RemoveAddress` placement exactly. |
| `bin-contact-manager/pkg/contacthandler/main.go` | `ContactHandler` interface (lines 36-39 region): add `CreateUnresolvedAddress(ctx, customerID uuid.UUID, a *contact.Address) (*contact.Address, error)` and `ClaimAddress(ctx, customerID, addressID, contactID uuid.UUID) (*contact.Address, error)` signatures (manual edit required before mock regeneration). |
| `bin-contact-manager/pkg/listenhandler/main.go` (routing table) | Register the new claim route. **Concrete edit (design review iter-6 finding, same recurring gap class resurfacing at the routing layer)**: add a new `regV1ContactAddressesIDClaim = regexp.MustCompile("/v1/contact_addresses/" + regUUID + "/claim$")` constant alongside the existing `regV1ContactAddressesID` (main.go:57), following the `regV1ContactsTags`/`regV1ContactsTagsID` precedent (main.go:64-65) for nested sub-resource actions — the existing `regV1ContactAddressesID` is `$`-anchored to bare `{id}` and will NOT match the `/claim` suffix. Add the corresponding `case regV1ContactAddressesIDClaim.MatchString(m.URI) && m.Method == sock.RequestMethodPost: response, err = h.processV1ContactAddressesIDClaim(ctx, m)` switch arm (placed before the more general `regV1ContactAddressesID` POST-adjacent cases, matching how `regV1ContactsTagsID` is ordered relative to `regV1ContactsTags`). |
| `bin-openapi-manager/openapi/paths/contact_addresses/main.yaml` | `POST`: make `contact_id` non-required in the **external-facing** schema (the `customer_id` injection is internal-only, see §5.1 — external clients never supply it); `GET`: add `unresolved` query param. |
| `bin-openapi-manager/openapi/paths/contact_addresses/id_claim.yaml` (new file) | New `POST /contact_addresses/{id}/claim` path (§5.3), following this spec's existing `{resource}/id_{action}.yaml` convention for nested-action-on-a-resource-id (verified precedent: `contacts/id_tags.yaml` for `POST /contacts/{id}/tags`, `contacts/id_addresses.yaml` for `POST /contacts/{id}/addresses`). **Full YAML content given in §5.3 (design review iter-5 finding #4: previously prose-only).** |
| `bin-openapi-manager/openapi/openapi.yaml` | Add the new `contact_addresses/{id}/claim` path `$ref` entry (main spec file wires all `paths:` entries to their per-resource YAML files). |
| `bin-openapi-manager/openapi/paths/service_agents/contact_addresses.yaml` / `contact_addresses_id.yaml` | Mirror the same `POST`/`GET`/claim changes for the ServiceAgent-facing surface (this spec has a parallel `service_agents/` tree for agent-scoped access, already handling `name`/`detail` identically to the customer-facing tree — see PR #1039). A ServiceAgent claim path file (`service_agents/contact_addresses_id_claim.yaml`, matching this tree's flat naming convention rather than the customer tree's `id_claim.yaml` subfolder style — verify exact naming against how `service_agents/contact_addresses_id.yaml` names itself, which is flat, not nested) also needs a `$ref` entry in `openapi.yaml`. |
| `bin-api-manager/server/contact_addresses.go` | `PostContactAddresses`: relax the `contactID == uuid.Nil` 400 check (line 79-83) to allow unresolved creation; route to a new/adjusted servicehandler call. Add a new claim HTTP handler (`PostContactAddressesIdClaim` or similar, generated name depends on oapi-codegen once the spec path is added). |
| `bin-api-manager/server/service_agents_contact_addresses.go` | Mirror the same relaxation + claim handler for the ServiceAgent surface. |
| `bin-api-manager/pkg/servicehandler/contact_address.go` | Add `ContactAddressCreateIndependent`'s unresolved branch (when `contactID == uuid.Nil`, skip the contact-ownership-check path and use `a.CustomerID` directly instead, mirroring `ContactAddressList`'s pattern — see §5.1). Add `ContactAddressClaim`/`ServiceAgentContactAddressClaim` (§5.3), which must verify **both** the address and the target contact belong to `a.CustomerID` before delegating to the backend (tenant isolation is bin-api-manager's job, not bin-contact-manager's, per the CLAUDE.md rule quoted above). |
| `bin-api-manager/pkg/servicehandler/main.go` | `ServiceHandler` interface (entries near the existing `ContactAddress*`/`ServiceAgentContactAddress*` group — verify exact line, cited as ~463/~926 in design review iter-1): add `ContactAddressClaim`/`ServiceAgentContactAddressClaim` signatures (manual edit required before mock regeneration). |
| `bin-common-handler/pkg/requesthandler/contact_contact_addresses.go` | `ContactV1ContactAddressCreate`: add a `customerID uuid.UUID` parameter and thread it into `cmrequest.ContactAddressCreate.CustomerID` (§5.1 — see explicit decision below on how the 2 existing call sites are updated). Add `ContactV1ContactAddressClaim` (§5.3). |
| `bin-common-handler/pkg/requesthandler/main.go` | `RequestHandler` interface (line ~905 per design review iter-1): update `ContactV1ContactAddressCreate`'s signature to add the `customerID` parameter, add `ContactV1ContactAddressClaim` (manual edit required before mock regeneration). |
| `bin-contact-manager/pkg/dbhandler/address_test.go` | Add tests: unresolved create writes real `NULL`; `AddressList` unresolved filter; claim guard rejects already-resolved rows. |
| `bin-contact-manager/pkg/listenhandler/*_test.go` | Add handler-level tests for the three surface changes. |
| `bin-contact-manager/pkg/contacthandler/contact_test.go` | Add tests for `CreateUnresolvedAddress`/`ClaimAddress` at the handler layer (mocking `dbhandler`), including the event-publish call. |
| `bin-api-manager/pkg/servicehandler/contact_address_test.go` | Add tests for the new unresolved-creation branch and the claim method's tenant-isolation checks (address AND contact both scoped to `a.CustomerID`). |

## 5. Exact string replacements / API changes

### 5.1 `POST /v1/contact_addresses`: allow unresolved creation

**Wire-field checklist** (source: `request.ContactAddressCreate`,
`v1_contact_addresses.go` lines 81-98):

| Field | Required before | Required after (external REST) | Notes |
|---|---|---|---|
| `contact_id` | yes (400 if `uuid.Nil`) | **no** — omitted or `uuid.Nil` means "create unresolved" | |
| `type` | yes | yes (unchanged) | |
| `target` | yes | yes (unchanged) | |
| `name` | no | no (unchanged) | |
| `detail` | no | no (unchanged) | |
| `is_primary` | no | no, but **rejected with 400 if `true` AND contact_id is absent** | An unresolved address cannot be primary (DB `CHECK
  (is_primary = 0 OR contact_id IS NOT NULL)` already enforces this at the
  SQL layer, but the handler should reject with a clear 400 before hitting
  a raw SQL constraint violation) |

**Corrected flow (fixes Draft v1's mistaken assumption that an external
client supplies `customer_id` in the POST body):** tenant scoping for an
unresolved address must follow the same pattern every other independent
`/contact_addresses` operation already uses — the authenticated
`a.CustomerID` from `bin-api-manager`'s `auth.AuthIdentity`, never a
client-supplied value. Concretely:

1. **`bin-api-manager/server/contact_addresses.go` `PostContactAddresses`**:
   remove the `contactID == uuid.Nil` 400 check (lines 79-83); allow
   `contactID` to be `uuid.Nil` and pass it through unchanged.
2. **`bin-api-manager/pkg/servicehandler/contact_address.go`
   `ContactAddressCreateIndependent`**: branch on `contactID`. If
   `uuid.Nil`, skip the `h.contactGet(ctx, contactID)` ownership-lookup path
   entirely (there is no contact to look up) and instead permission-check
   directly against `a.CustomerID` (mirroring `ContactAddressList`'s
   existing pattern at line 32: `h.hasPermission(ctx, a, a.CustomerID,
   ...)`), then call a new/adjusted `reqHandler.ContactV1ContactAddressCreate`
   overload that also passes `a.CustomerID` down.
3. **`bin-common-handler/pkg/requesthandler/contact_contact_addresses.go`
   `ContactV1ContactAddressCreate`**: add a `customerID uuid.UUID` parameter,
   set it on `cmrequest.ContactAddressCreate.CustomerID` before marshaling.
   This is the ONLY place `CustomerID` is populated — it flows from the
   authenticated identity down through the RPC call, never from an external
   request body field. **Explicit decision (design review iter-1 finding
   #4)**: modify the existing function's signature in place (add the
   parameter as the 2nd argument, after `ctx`), rather than adding a
   separate overload. The 2 existing call sites
   (`bin-api-manager/pkg/servicehandler/contact_address.go` lines 114 and
   290, both currently resolved-address-creation paths where `contactID !=
   uuid.Nil`) are updated to pass `a.CustomerID` /  `agent.CustomerID`
   respectively (both already have that value in scope) — this is a
   compile-time-enforced change (adding a required parameter breaks the
   build until every call site is updated), which is the safest way to
   guarantee no call site is silently missed.
4. **`bin-contact-manager/pkg/listenhandler/models/request/contacts.go`**:
   add `CustomerID uuid.UUID \`json:"customer_id"\`` to `ContactAddressCreate`.
   This field IS present in the RabbitMQ RPC JSON body (internal
   service-to-service, not the external REST body) — `bin-api-manager` is
   the only caller of this internal queue message and it always sets a
   real `customer_id` before this point, so `bin-contact-manager`'s
   listenhandler can trust it without re-deriving it (consistent with the
   CLAUDE.md rule: auth/ownership checks live only in `bin-api-manager`;
   downstream services trust what they receive on the internal queue).

**`v1_contact_addresses.go` `processV1ContactAddressesPost`** (currently
lines 87-90):

```go
if reqData.ContactID == uuid.Nil {
    log.Error("Missing contact_id.")
    return simpleResponse(400), nil
}
```

becomes:

```go
if reqData.ContactID == uuid.Nil {
    if reqData.CustomerID == uuid.Nil {
        log.Error("customer_id is required when creating an unresolved address (no contact_id).")
        return simpleResponse(400), nil
    }
    if reqData.IsPrimary {
        log.Error("An unresolved address (no contact_id) cannot be primary.")
        return simpleResponse(400), nil
    }
}
```

**Handler-layer routing (corrected placement, design review iter-1 finding
#2)**: `AddAddress` on `contactHandler` (contact-scoped,
`contacthandler/contact.go` lines 258-297) always resolves a real
`contactID` first via `h.db.ContactGet`, so it can never construct an
unresolved address — this path is unaffected. The listenhandler's POST
handler must **bypass** `contactHandler.AddAddress` when `ContactID ==
uuid.Nil` and call a new `contactHandler.CreateUnresolvedAddress(ctx,
customerID uuid.UUID, a *contact.Address) (*contact.Address, error)`
method instead. This new method lives on `contactHandler` (not
`addressHandler`) because it must call `publishEvent` on completion — see
§4's corrected-placement note. Since there is no contact to fetch when
unresolved, `CreateUnresolvedAddress` calls `h.db.AddressCreate` directly
(no `h.db.ContactGet` precondition, unlike `AddAddress`), then publishes a
different signal than `EventTypeContactUpdated` (there is no contact yet to
attach the update to) — **decision: no event is published on unresolved
creation**. The address isn't associated with any contact yet, so there is
no contact-scoped event to emit; the first event fires later when
`ClaimAddress` succeeds (§5.3).

**Pseudocode (design review iter-2 finding #4: this method was previously
prose-only, unlike `ClaimAddress`'s full pseudocode; added here for
implementation-risk parity, since `AddAddress` explicitly sets both `a.ID`
and `a.CustomerID` before its dbhandler call, `contact.go:266-268`, and this
method must do the same)**:

```go
// CreateUnresolvedAddress creates an address row with no contact_id yet.
// No event is published — the address is not attached to any contact.
func (h *contactHandler) CreateUnresolvedAddress(ctx context.Context, customerID uuid.UUID, a *contact.Address) (*contact.Address, error) {
    a.ID = h.utilHandler.UUIDCreate()
    a.CustomerID = customerID
    a.ContactID = uuid.Nil // explicit: unresolved

    if err := h.db.AddressCreate(ctx, a); err != nil {
        return nil, fmt.Errorf("could not create unresolved address: %w", err)
    }

    res, err := h.db.AddressGet(ctx, customerID, a.ID)
    if err != nil {
        return nil, fmt.Errorf("could not get created address: %w", err)
    }
    return res, nil
}
```

**`dbhandler.AddressCreate`** (the critical fix, `pkg/dbhandler/address.go`
lines 107-136): change

```go
query, args, err := sq.Insert(addressTable).
    SetMap(map[string]any{
        "id":          a.ID.Bytes(),
        "customer_id": a.CustomerID.Bytes(),
        "contact_id":  a.ContactID.Bytes(),
        ...
```

to conditionally pass Go `nil` (which the MySQL driver serializes as SQL
`NULL`) instead of `uuid.Nil.Bytes()` when unresolved:

```go
var contactIDValue any
if a.ContactID != uuid.Nil {
    contactIDValue = a.ContactID.Bytes()
}
// ... SetMap(map[string]any{ ..., "contact_id": contactIDValue, ... })
```

**Guard the existing cache-refresh tail call (design review iter-1 finding
#5)**: `AddressCreate`'s existing final line, `_ =
h.contactUpdateToCache(ctx, a.ContactID)`, must be skipped when
`a.ContactID == uuid.Nil` — otherwise every unresolved creation triggers a
doomed `contactGetFromDB(ctx, uuid.Nil)` lookup (the error is silently
swallowed today, so this is not a correctness bug, but it's wasted work on
every single unresolved-address creation and should be guarded):

```go
if a.ContactID != uuid.Nil {
    _ = h.contactUpdateToCache(ctx, a.ContactID)
}
```

### 5.2 `GET /v1/contact_addresses`: unresolved pool filter

**New query param**: `unresolved=true` (boolean). When present and `true`,
list only rows where `contact_id IS NULL`, scoped to `customer_id`. Mutually
exclusive with `contact_id` (if both given, `unresolved=true` wins and
`contact_id` is ignored — document this in the OpenAPI spec).

**`v1_contact_addresses.go` `processV1ContactAddressesGet`** (currently
lines 21-72): add after the existing `contact_id` filter parsing (lines
38-43):

```go
if v := u.Query().Get("unresolved"); v == "true" {
    filters["unresolved"] = true
}
```

**`dbhandler.AddressList`** (`pkg/dbhandler/address.go` lines 171-218): add
a new filter branch:

```go
if v, ok := filters["unresolved"]; ok {
    if unresolved, ok2 := v.(bool); ok2 && unresolved {
        q = q.Where(sq.Eq{"contact_id": nil})  // squirrel renders IS NULL for nil
    }
}
```

(squirrel's `sq.Eq{"col": nil}` renders `col IS NULL`, not `col = NULL` —
verify this exact behavior against the installed squirrel version during
implementation; if it does not, use `sq.Expr("contact_id IS NULL")`
instead.)

### 5.3 Claim: attach an unresolved address to a contact

**Open question flagged for design review**: issue #1040 suggested
"extend PUT /contact_addresses/{id}" OR a dedicated endpoint. This design
proposes a **dedicated endpoint**, `POST /v1/contact_addresses/{id}/claim`,
rather than extending the generic `PUT`, for these reasons:

1. `PUT`'s `AddressUpdate` today is a generic partial-update surface
   (target/name/detail/is_primary) with no business-rule guards beyond
   is_primary reset. Claiming has a strict, claim-specific guard ("must
   currently be `NULL`, else 409") that doesn't fit the generic
   partial-update model — mixing it in would make `AddressUpdate` handle
   two very different semantics (cosmetic field edit vs. identity-changing
   claim) behind one endpoint.
2. A claim is conceptually a different HTTP action (idempotent "assign
   ownership") from a `PUT` (idempotent "replace fields"), and deserves its
   own 409 Conflict response code path, which `PUT`'s current handler
   doesn't have a slot for.

**Request struct** (`bin-contact-manager/pkg/listenhandler/models/request/contacts.go`,
design review iter-5 finding #2: previously only a JSON example was shown,
not the actual Go struct; add alongside the existing `AddressUpdate`/
`TagAssignment` structs in the same file):

```go
// ContactAddressClaim is the body for POST /v1/contact_addresses/{id}/claim
type ContactAddressClaim struct {
    ContactID uuid.UUID `json:"contact_id"` // required
}
```

**Listenhandler pseudocode** (design review iter-5 finding #1: previously
missing entirely, unlike every other endpoint in this doc, which all get
full Go pseudocode; add to
`bin-contact-manager/pkg/listenhandler/v1_contact_addresses.go` alongside
`processV1ContactAddressesIDGet`/`processV1ContactAddressesIDPut`, and
register the route in `pkg/listenhandler/main.go`'s routing table):

```go
// processV1ContactAddressesIDClaim handles POST /v1/contact_addresses/{id}/claim
// Body: {contact_id}
func (h *listenHandler) processV1ContactAddressesIDClaim(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    uriItems := strings.Split(m.URI, "/")
    if len(uriItems) < 5 { // /v1/contact_addresses/{id}/claim
        return simpleResponse(400), nil
    }

    id := uuid.FromStringOrNil(uriItems[3])
    log := logrus.WithFields(logrus.Fields{
        "func":       "processV1ContactAddressesIDClaim",
        "address_id": id,
    })
    log.WithField("request", m).Debug("Received request.")

    // customer_id arrives via query param, same pattern as every other
    // /v1/contact_addresses/{id} sibling endpoint (GET/PUT/DELETE above) —
    // it is populated by the internal RPC caller (bin-api-manager), never
    // by an external client directly.
    u, err := url.Parse(m.URI)
    if err != nil {
        log.Errorf("Could not parse URI. err: %v", err)
        return simpleResponse(400), nil
    }
    customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))
    if customerID == uuid.Nil {
        log.Error("Missing or invalid customer_id.")
        return simpleResponse(400), nil
    }

    var reqData request.ContactAddressClaim
    if err := json.Unmarshal(m.Data, &reqData); err != nil {
        log.Errorf("Could not unmarshal the request. err: %v", err)
        return simpleResponse(400), nil
    }
    if reqData.ContactID == uuid.Nil {
        log.Error("Missing contact_id.")
        return simpleResponse(400), nil
    }

    tmp, err := h.contactHandler.ClaimAddress(ctx, customerID, id, reqData.ContactID)
    if err != nil {
        log.Errorf("Could not claim address. err: %v", err)
        return errorResponse(err), nil // routes cerrors.NotFound/AlreadyExists to 404/409
    }

    data, err := json.Marshal(tmp)
    if err != nil {
        log.Debugf("Could not marshal the response message. err: %v", err)
        return simpleResponse(500), nil
    }

    return &sock.Response{
        StatusCode: 200,
        DataType:   "application/json",
        Data:       data,
    }, nil
}
```

**`bin-common-handler` RPC client** (design review iter-5 finding #3:
previously referenced only as a one-line prose call; add to
`bin-common-handler/pkg/requesthandler/contact_contact_addresses.go`,
mirroring `ContactV1ContactAddressGet`/`ContactV1ContactAddressUpdate`'s
explicit `fmt.Sprintf` URI-construction pattern exactly):

```go
// ContactV1ContactAddressClaim sends a request to contact-manager to claim
// an unresolved address onto a contact.
func (r *requestHandler) ContactV1ContactAddressClaim(
    ctx context.Context,
    customerID uuid.UUID,
    addressID uuid.UUID,
    contactID uuid.UUID,
) (*cmcontact.Address, error) {
    uri := fmt.Sprintf("/v1/contact_addresses/%s/claim?customer_id=%s", addressID, customerID)

    data := &cmrequest.ContactAddressClaim{ContactID: contactID}
    m, err := json.Marshal(data)
    if err != nil {
        return nil, err
    }

    tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/contact_addresses/<id>/claim", requestTimeoutDefault, 0, ContentTypeJSON, m)
    if err != nil {
        return nil, err
    }

    var res cmcontact.Address
    if errParse := parseResponse(tmp, &res); errParse != nil {
        return nil, errParse
    }

    return &res, nil
}
```

Add the matching `ContactV1ContactAddressClaim` method signature to the
`RequestHandler` interface in `bin-common-handler/pkg/requesthandler/main.go`
(already listed in §4's affected-files table).

**Response codes**:
- `200` — claimed successfully, returns the updated `ContactManagerAddress`.
- `400` — missing/invalid `contact_id` in body.
- `404` — address `{id}` not found (or not in caller's `customer_id` scope).
- `404` — target contact not found (or not in caller's `customer_id` scope).
- `409` — address is already resolved to a *different* contact_id (the
  "no move/copy" guard from issue #1040). If the address is already
  resolved to the *same* contact_id being claimed, treat as idempotent
  success (200), not a conflict — re-claiming your own already-claimed
  address should not error.

**OpenAPI YAML** (design review iter-5 finding #4: previously only
prose-described; add as
`bin-openapi-manager/openapi/paths/contact_addresses/id_claim.yaml`,
following `contacts/id_tags.yaml`'s structure):

```yaml
post:
  summary: Claim an unresolved contact address
  description: Attaches a currently-unresolved contact address (contact_id
    is NULL) to the given contact. Idempotent if already claimed by the
    same contact; returns 409 if claimed by a different contact.
  tags:
    - Contact
  parameters:
    - name: id
      in: path
      description: The ID of the contact address to claim.
      required: true
      schema:
        type: string
        format: uuid
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          required:
            - contact_id
          properties:
            contact_id:
              type: string
              format: uuid
              example: 5e4a0680-eba3-4001-a000-000000000001
  responses:
    '200':
      description: Address claimed successfully.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ContactManagerAddress'
    '400':
      $ref: '#/components/responses/BadRequest'
    '401':
      $ref: '#/components/responses/Unauthenticated'
    '403':
      $ref: '#/components/responses/PermissionDenied'
    '404':
      $ref: '#/components/responses/NotFound'
    '409':
      $ref: '#/components/responses/Conflict'
    '500':
      $ref: '#/components/responses/InternalError'
```

Mirror the identical path body into the ServiceAgent-facing
`bin-openapi-manager/openapi/paths/service_agents/contact_addresses_id_claim.yaml`
(flat naming, matching `service_agents/contact_addresses_id.yaml`'s
existing convention). **Verified against the real spec (design review
iter-5 fix)**: the correct shared response component name for 409 is
`#/components/responses/Conflict` (`openapi.yaml:7691-7696`, described as
"State conflict (ALREADY_EXISTS or FAILED_PRECONDITION)"), not
`AlreadyExists` — there is no `AlreadyExists` response component in this
spec; `Conflict` is the only 409-mapped one and is reused here. Add both
files' `$ref` entries to `openapi/openapi.yaml`'s `paths:` section.

**`dbhandler.AddressClaim`** (new function, `pkg/dbhandler/address.go`;
pure SQL-layer primitive. **Correction (design review iter-2 finding #1)**:
it DOES call `h.contactUpdateToCache`, matching `AddressUpdate`/
`AddressDelete`'s existing shape exactly — those two functions already call
`contactUpdateToCache` from the dbhandler layer (see `address.go:289` and
`:315`); only `publishEvent` is exclusively `contactHandler`'s job, because
`contactUpdateToCache` is an unexported method on dbhandler's `*handler`
(`contact.go:59`, not part of the `DBHandler` interface) and
`contactHandler` has no way to call it. Draft v2 incorrectly generalized
"`publishEvent` is contactHandler-only" into "all cache/event side effects
are contactHandler-only," which would have left the contact-body Redis
cache stale after every successful claim until an unrelated write touched
that contact. This draft fixes that regression):

```go
// AddressClaim attaches contact_id to a currently-unresolved address.
// Returns ErrConflict if the address is already resolved to a DIFFERENT
// contact_id. No-ops (success) if already resolved to the SAME contact_id.
func (h *handler) AddressClaim(ctx context.Context, customerID, addressID, contactID uuid.UUID) error {
    existing, err := h.AddressGet(ctx, customerID, addressID) // tenant-scoped fetch
    if err != nil {
        return err // ErrNotFound propagates as-is
    }
    if existing.ContactID == contactID {
        return nil // already claimed by this contact — idempotent success
    }
    if existing.ContactID != uuid.Nil {
        return ErrConflict // resolved to a DIFFERENT contact — reject, no move
    }

    query, args, err := sq.Update(addressTable).
        Set("contact_id", contactID.Bytes()).
        Set("tm_update", h.utilHandler.TimeNow()).
        Where(sq.Eq{"id": addressID.Bytes()}).
        Where(sq.Eq{"contact_id": nil}). // race guard: only claim if STILL unresolved
        ToSql()
    if err != nil {
        return fmt.Errorf("could not build query. AddressClaim. err: %v", err)
    }
    result, err := h.db.Exec(query, args...)
    if err != nil {
        return fmt.Errorf("could not execute. AddressClaim. err: %v", err)
    }
    rows, _ := result.RowsAffected()
    if rows == 0 {
        return ErrConflict // lost the race — someone else claimed it between Get and Update
    }

    // Refresh the contact-body cache, mirroring AddressUpdate/AddressDelete.
    // This is the dbhandler layer's own responsibility (unexported method,
    // not reachable from contactHandler) and is independent of publishEvent.
    _ = h.contactUpdateToCache(ctx, contactID)

    return nil
}
```

The `Where(sq.Eq{"contact_id": nil})` on the `UPDATE` (not just the
pre-check `Get`) closes a TOCTOU race: two concurrent claims of the same
unresolved address must not both succeed. **This requires adding a new
`ErrConflict` sentinel to `dbhandler/main.go`'s existing `var (...)` block,
alongside `ErrNotFound` (main.go:68-71) — design review iter-4 finding:
this was previously only asserted in prose and not carried into §4's
affected-files table, so it was never actually specified as a concrete
edit and would not exist anywhere in the codebase as written:**

```go
// dbhandler/main.go
var (
    ErrNotFound = fmt.Errorf("record not found")
    ErrConflict = fmt.Errorf("address already claimed")
)
```

and a corresponding 409 mapping in the listenhandler's
`errorResponse()` helper (verify `errorResponse` already has a slot for
non-404 domain errors, or extend it).

**Handler layer (corrected placement, design review iter-1 findings #1 and
#2)**: add `ClaimAddress(ctx context.Context, customerID, addressID,
contactID uuid.UUID) (*contact.Address, error)` to `contactHandler` (NOT
`addresshandler` — see §4's placement note), in
`bin-contact-manager/pkg/contacthandler/contact.go`, mirroring
`AddAddress`'s structure:

```go
// ClaimAddress attaches a currently-unresolved address to contactID.
// Publishes EventTypeContactUpdated on success (the address becomes part
// of the contact's address set). Returns ErrConflict (mapped to 409 by the
// listenhandler) if the address is already resolved to a DIFFERENT contact.
func (h *contactHandler) ClaimAddress(ctx context.Context, customerID, addressID, contactID uuid.UUID) (*contact.Address, error) {
    // Verify the target contact exists and belongs to this customer
    // (defense-in-depth re-check; bin-api-manager already verified this,
    // see the tenant-isolation section below).
    c, err := h.db.ContactGet(ctx, contactID)
    if err != nil {
        return nil, err
    }
    if c.CustomerID != customerID {
        // Correction (design review iter-3 finding #3): the previous draft's
        // literal "..." placeholder was not real code. Use the full
        // 3-argument constructor, matching ResolutionCreate's identical
        // cross-tenant pattern (resolution.go:60-64).
        return nil, cerrors.NotFound(
            commonoutline.ServiceNameContactManager,
            "CONTACT_NOT_FOUND",
            "The contact was not found.",
        ) // treat cross-tenant contact as not-found, not permission-denied, to avoid leaking existence
    }

    if err := h.db.AddressClaim(ctx, customerID, addressID, contactID); err != nil {
        // Correction (design review iter-3 findings #1 and #2): the previous
        // draft's `cerrors.AlreadyExists(...)`/`cerrors.NotFound(...)` calls
        // used the wrong arity (these constructors take 3 args:
        // domain, reason, message — see constructors.go:35-48) and referenced
        // an undeclared bare `errors`/`ErrConflict`/`ErrNotFound`. This file
        // aliases the standard library as `stderrors` (contact.go:5) because
        // `cerrors` already occupies the conceptual "errors" name, and the
        // dbhandler sentinels must be qualified as `dbhandler.ErrConflict`/
        // `dbhandler.ErrNotFound`. Corrected to match ResolutionCreate's
        // exact translation pattern (resolution.go:50-56), including
        // `.Wrap(err)` to preserve the cause for server logs.
        if stderrors.Is(err, dbhandler.ErrConflict) {
            return nil, cerrors.AlreadyExists(
                commonoutline.ServiceNameContactManager,
                "ADDRESS_ALREADY_CLAIMED",
                "The address is already claimed by another contact.",
            ).Wrap(err)
        }
        if stderrors.Is(err, dbhandler.ErrNotFound) {
            return nil, cerrors.NotFound(
                commonoutline.ServiceNameContactManager,
                "ADDRESS_NOT_FOUND",
                "The address was not found.",
            ).Wrap(err)
        }
        return nil, err
    }

    addr, err := h.db.AddressGet(ctx, customerID, addressID)
    if err != nil {
        return nil, fmt.Errorf("could not get claimed address: %w", err)
    }

    // Publish the contact_updated event — this is the ONLY place that
    // triggers the event for a successful claim (dbhandler.AddressClaim
    // itself does NOT call publishEvent; that remains a contactHandler-only
    // concern, consistent with AddAddress/RemoveAddress). Reuse `c`, fetched
    // above for the tenant check, instead of re-fetching the contact
    // (design review iter-2 finding #3: avoid the redundant second
    // h.db.ContactGet call).
    h.publishEvent(ctx, contact.EventTypeContactUpdated, c)

    return addr, nil
}
```

**Explicit correction (design review iter-1 finding #1, cache/event
contradiction)**: `dbhandler.AddressClaim` (§5.3's `AddressClaim` function
above) does **NOT** call `contactUpdateToCache` or `publishEvent` itself —
those remain `contactHandler`-only responsibilities, called exclusively
from the new `ClaimAddress` method shown here. The dbhandler function is a
pure SQL-layer primitive (matching `AddressUpdate`/`AddressDelete`'s
existing dbhandler-layer shape, which also do NOT call `publishEvent`
directly — only `contactHandler.UpdateAddress`/`RemoveAddress` do, after
calling the dbhandler primitive). §8's earlier claim that
`contactUpdateToCache` happens "already inside `dbhandler.AddressClaim`"
was incorrect in Draft v1 and is corrected here.

**`bin-api-manager` tenant-isolation layer (the claim's actual security
boundary, per the "auth/ownership only in bin-api-manager" rule)**: a new
`ContactAddressClaim`/`ServiceAgentContactAddressClaim` servicehandler
method must, before calling the backend:

1. Fetch the address via `reqHandler.ContactV1ContactAddressGet(ctx,
   a.CustomerID, addressID)` — this already tenant-scopes by
   `customer_id` (existing pattern, see `ContactAddressGet` lines 68 and
   the `res.CustomerID != a.CustomerID` check at line 75).
2. Fetch the target contact via `h.contactGet(ctx, contactID)` and verify
   `ct.CustomerID == a.CustomerID` (existing pattern, see
   `ContactAddressCreateIndependent` lines 104-112).
3. Only if both checks pass, call the new
   `reqHandler.ContactV1ContactAddressClaim(ctx, a.CustomerID, addressID,
   contactID)`.

This means `bin-contact-manager`'s `AddressClaim` (§5.3 dbhandler function
above) does NOT need to re-verify tenant ownership of the *contact* side —
`bin-api-manager` has already done so before the RPC call reaches it,
consistent with every other independent-resource operation in this file.
`bin-contact-manager`'s own `AddressGet(ctx, customerID, addressID)` call
inside `AddressClaim` still tenant-scopes the *address* side (defense in
depth, matching the existing pattern in `UpdateAddress`/`RemoveAddress`
which also re-verify via `h.db.AddressGet(ctx, c.CustomerID, addressID)`
even though `bin-api-manager` already checked).

## 6. Copy/decision rationale

- Dedicated `POST .../claim` endpoint (not PUT extension): see §5.3
  reasoning — keeps a strict identity-changing operation with its own
  conflict semantics separate from the generic cosmetic-field PUT.
- `unresolved=true` boolean query param (not a magic `contact_id=unresolved`
  sentinel string): keeps `contact_id`'s type contract as "valid UUID or
  absent" rather than overloading it with a non-UUID sentinel value that
  would need special parsing and could collide with a real (malformed)
  UUID string in the future.
- `customer_id` becomes a new required-conditionally field on
  `ContactAddressCreate`: flagged explicitly as the largest wire-contract
  change in this PR precisely because every existing POST caller derives
  `customer_id` from the contact and has never had to supply it directly.

## 7. Verification plan

1. `cd bin-contact-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`.
2. `cd bin-openapi-manager && oapi-codegen -config configs/config_model/config.generate.yaml openapi/openapi.yaml > /dev/null` (spec validates) then `go generate ./...`.
3. `cd bin-api-manager && go build ./...` (confirm the openapi-manager schema change doesn't break the consumer).
4. New dbhandler test: `AddressCreate` with `ContactID: uuid.Nil` — assert
   the inserted row's `contact_id` column is SQL `NULL` (query
   `information_schema` or re-`SELECT ... WHERE contact_id IS NULL` against
   a real/throwaway DB, not just a mock — mocks won't catch the
   `uuid.Nil.Bytes()` vs `nil` bug since gomock doesn't validate actual SQL
   semantics).
5. New dbhandler test: `AddressList` with `filters["unresolved"] = true`
   only returns `contact_id IS NULL` rows.
6. New dbhandler test: `AddressClaim` — (a) claiming an unresolved address
   succeeds and sets `contact_id`; (b) claiming an address already resolved
   to a *different* contact returns `ErrConflict`; (c) re-claiming an
   address already resolved to the *same* contact returns success
   (idempotent), not a conflict; (d) concurrent claim race — two goroutines
   claiming the same unresolved address, exactly one succeeds (this
   exercises the `WHERE contact_id IS NULL` race guard on the `UPDATE`, not
   just the pre-check).
7. Handler-level test: `POST /v1/contact_addresses` without `contact_id`
   creates successfully when `customer_id` is present, 400s when
   `customer_id` is also absent, 400s when `is_primary=true` is combined
   with no `contact_id`.
8. Handler-level test: `POST /v1/contact_addresses/{id}/claim` covers the
   200/400/404/404/409 response matrix from §5.3.

## 8. Rollout / risk

Medium risk (higher than the prior name/detail PR): this is the first time
`contact_addresses` will ever hold a real `contact_id IS NULL` row in
production, exercising a schema path that has existed since VOIP-1206 but
was never actually written to. The critical `AddressCreate` NULL-write fix
(§5.1) is the single highest-risk line in this PR — if missed, unresolved
addresses would silently write fake all-zero `contact_id` values instead of
NULL, which would not error but would corrupt the "unresolved pool"
semantics invisibly (no test failure until someone actually queries for
`contact_id IS NULL` and finds nothing, or worse, finds phantom
all-zero-UUID matches if any other code ever joins on `contact_id` without
an explicit NULL check). Mitigated by the explicit dbhandler-level test in
§7 item 4 that checks the raw SQL semantics, not just a mock.

The TOCTOU race in claim (§5.3) is a second material risk if the
`WHERE contact_id IS NULL` guard on the `UPDATE` is dropped during
implementation in favor of only checking via the pre-`Get`. Mitigated by
§7 item 6(d)'s concurrent-claim test.

## 9. Open questions

1. **Dedicated claim endpoint vs. PUT extension** (§5.3) — this design
   picks dedicated endpoint; reviewer should confirm or push back with a
   concrete PUT-extension counter-proposal.
2. ~~`customer_id` in `ContactAddressCreate`~~ — **RESOLVED** during this
   draft's own Phase 1.6 point-check (see §4's "Architecture correction"
   note and §5.1's "Corrected flow"): `customer_id` is never client-supplied
   on the external REST body. It flows from `bin-api-manager`'s
   authenticated `a.CustomerID` through the internal RabbitMQ RPC call to
   `bin-contact-manager`, consistent with the `CLAUDE.md` rule that
   auth/ownership checks live only in `bin-api-manager`. The `CustomerID`
   field added to `request.ContactAddressCreate` is populated by
   `bin-api-manager`'s `requesthandler` layer only, never by an external
   caller — this is now reflected as a corrected 4-layer change in §4/§5.1,
   not a 1-layer change.

## 9.1 Known gap: no rate limiting/pool size cap on unresolved addresses

Nothing in this design caps how many unresolved addresses a customer can
accumulate. If Square-admin's future picker UI never gets built or agents
never claim rows, the unresolved pool grows unbounded. Accepted as a known
gap for this PR (mirrors the `9.1` treatment from the name/detail PR) —
not blocking, but flagged so it isn't silently forgotten. Follow-up (if it
ever matters): a periodic cleanup job or a customer-level pool size limit
returning 429 on `POST` past the cap.

## Approval status

Draft — pending Design Review→Fix loop.
