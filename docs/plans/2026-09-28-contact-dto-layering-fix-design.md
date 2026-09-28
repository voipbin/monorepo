# Design: Fix contact wire-DTO layering violation in bin-api-manager

Date: 2026-09-28
Author: CPO (Claude, on behalf of 대표님)
Status: draft

## Problem

CLAUDE.md's "Layering — transport DTO ownership" rule states that a service's
`pkg/listenhandler/models/{request,response}` package may only be referenced
from that service's own `listenhandler`, with one sanctioned exception:
`bin-common-handler/pkg/requesthandler` (the RPC client), which may reference
a producing service's transport DTOs to marshal/unmarshal wire calls.

Today `bin-api-manager` violates this rule outside that exception. It imports
`bin-contact-manager/pkg/listenhandler/models/request` (aliased `cmrequest`)
directly in its own business/service layer and its OpenAPI server layer:

- `bin-api-manager/pkg/servicehandler/contact.go`
- `bin-api-manager/pkg/servicehandler/serviceagent_contact.go`
- `bin-api-manager/pkg/servicehandler/main.go` (interface signatures)
- `bin-api-manager/server/contacts.go`
- `bin-api-manager/server/service_agents_contacts.go`
- plus three matching `_test.go` files (`pkg/servicehandler/contact_test.go`,
  `pkg/servicehandler/serviceagent_contact_test.go`,
  `server/service_agents_contacts_test.go`) and one generated
  `pkg/servicehandler/mock_main.go` (9 files total)

All nine usages are the same symbol: `cmrequest.AddressCreate`, used as the
element type of an `addresses []cmrequest.AddressCreate` parameter threaded
from the OpenAPI handler down through `servicehandler.ContactCreate` /
`serviceHandler.ServiceAgentContactCreate` into
`requestHandler.ContactV1ContactCreate`.

Introduced 2026-07-02 (contact/Case feature work). Root cause: no domain-level
"address creation input" shape existed in `bin-contact-manager/models/contact`
at the time, only the domain `Address` (a full DB row: ID, CustomerID,
ContactID, TMCreate) and the wire `request.AddressCreate` (flat create input).
The wire DTO was reached for because it already had the right shape.

## Fix

Add a small domain-level input type, `contact.AddressInput`, to
`bin-contact-manager/models/contact`, next to the existing `Address` domain
type. Mirror the field set of `request.AddressCreate` (`Type`, `Target`,
`Name`, `Detail`, `IsPrimary`) by embedding `commonaddress.Address` plus
`IsPrimary`, matching the existing convention in `contact.Address` (which
already embeds `commonaddress.Address` rather than hand-copying its fields —
see the comment on `contact.Address`, and `kase.Case` embedding
`commonidentity.Owner` elsewhere in the repo for the same pattern).

Named `AddressInput`, not `AddressCreate`: the obvious first choice
(`AddressCreate`, matching the wire DTO it replaces) reads as if it *were*
the wire `request.AddressCreate` DTO — defeating the point of this PR, which
is to make the domain/wire boundary visible at the type name, not just in
its package path. `bin-talk-manager/models/participant.ParticipantInput`
(input shape for adding participants during chat creation) is the existing
repo precedent for this naming: `*Input` for a domain-level write-input
shape, distinct from both the full read-model type and any wire DTO.

`Address` is then changed to embed `AddressInput` instead of hand-copying
its four shared fields (`Type`/`Target`/`Name`/`Detail`/`IsPrimary` via
`commonaddress.Address` + `IsPrimary`) alongside its own persistence fields
(`ID`/`CustomerID`/`ContactID`/`TMCreate`). This removes the `IsPrimary`
duplication that existed between the two structs when `AddressInput` had
its own separate `commonaddress.Address` embed, and matches
`AddressInput`'s json tags directly to `Address`'s public fields via
promotion — `Address`'s public JSON shape is unchanged (field order in the
JSON output shifts, since embedded-struct fields serialize in struct
declaration order, but the field set and each field's json tag are
identical; this only affects any test asserting on an exact ordered JSON
string, which were updated to match in the same PR).

```go
// AddressInput is the domain input shape for creating a contact address.
// Embeds commonaddress.Address for the same reason and with the same
// caveats as Address below -- see Address's comment.
type AddressInput struct {
    commonaddress.Address
    IsPrimary bool `json:"is_primary"`
}

// Address represents a single row in contact_addresses.
// Embeds AddressInput rather than hand-copying its fields, consistent with
// the monorepo's "embed a shared struct" convention.
type Address struct {
    AddressInput
    ID         uuid.UUID  `json:"id"`
    CustomerID uuid.UUID  `json:"customer_id"`
    ContactID  uuid.UUID  `json:"contact_id"`
    TMCreate   *time.Time `json:"tm_create"`
}
```

`commonaddress.Address`'s `Type` accepts a wider set of values than the
contact-address create path allows, and `TargetName` is unused here — both
true of `Address` too, both before and after this restructuring. Neither is
a reason to break the embedding convention: they are narrowed the same way
in either type — by validation at the call site
(`pkg/contacthandler.isValidContactAddressType`), not by the type shape. An
earlier revision of this design hand-copied `AddressInput`'s four fields
instead of embedding `commonaddress.Address`, reasoning that the wider
`Type` and unused `TargetName` were `AddressInput`-specific problems; that
duplicated `Address`'s field list inside the same file for no real gain and
was reverted back to embedding before this PR merged — flagged in review as
introducing exactly the kind of struct duplication the "embed, don't
hand-copy" convention exists to avoid. A follow-up round then collapsed
`Address`'s own hand-copied `commonaddress.Address`+`IsPrimary` pair into an
`AddressInput` embed for the same reason, once `AddressInput` existed as a
ready-made "the fields `Address` needs, minus persistence metadata" shape.

`AddressInput` carries json tags (`is_primary`), even though it is never
marshaled on its own — `requesthandler.ContactV1ContactCreate` maps it
field-by-field into the wire `request.AddressCreate` before marshal (see
Call-chain changes below). The tags exist so that `Address`'s embed of
`AddressInput` produces the correct wire shape when `Address` itself is
marshaled directly as part of `Contact`/`WebhookMessage` API responses.

This is a plain domain struct — it is NOT a `response.*`/`request.*` DTO
copy; it lives in `models/contact`, the domain package every consumer is
already allowed to import.

### Call-chain changes

1. **`bin-common-handler/pkg/requesthandler/contact_contacts.go`**
   `ContactV1ContactCreate` changes its `addresses` parameter type from
   `[]cmrequest.AddressCreate` to `[]cmcontact.AddressInput` (domain). Inside
   the function, map the domain slice to `[]cmrequest.AddressCreate` before
   constructing the `cmrequest.ContactCreate` wire payload. This keeps
   `requesthandler` as the single translation point between domain and wire,
   per the sanctioned-exception rule, and matches CLAUDE.md's stated
   preference ("cleaner long-term form... use the producing service's domain
   models/ types instead; prefer that for new client methods").

2. **`bin-api-manager/pkg/servicehandler/{main.go,contact.go,serviceagent_contact.go}`**
   Change the `addresses []cmrequest.AddressCreate` parameter (interface +
   both implementations) to `addresses []cmcontact.AddressInput`. Drop the
   `cmrequest` import entirely from `pkg/servicehandler`.

3. **`bin-api-manager/server/{contacts.go,service_agents_contacts.go}`**
   Change the local construction from `cmrequest.AddressCreate{...}` to
   `cmcontact.AddressInput{...}`. This is NOT a pure rename — the current
   code builds a plain `string` (`addrType := string(*v.Type)`) and assigns
   it straight to `cmrequest.AddressCreate.Type` (also plain `string`). The
   new `cmcontact.AddressInput` embeds `commonaddress.Address`, whose `Type`
   field is the named type `commonaddress.Type` (`type Type string`), so the
   assignment needs an explicit conversion. Concretely, in both files:

   ```go
   // before
   addr := cmrequest.AddressCreate{
       Type:   addrType,   // addrType is a plain string
       Target: *v.Target,
   }
   // ... addr.IsPrimary = *v.IsPrimary / addr.Name = *v.Name / addr.Detail = *v.Detail

   // after
   addr := cmcontact.AddressInput{
       Address: commonaddress.Address{
           Type:   commonaddress.Type(addrType), // explicit conversion, not a bare rename
           Target: *v.Target,
       },
   }
   // ... addr.IsPrimary = *v.IsPrimary / addr.Name = *v.Name / addr.Detail = *v.Detail
   ```

   (Field promotion means `addr.Name`/`addr.Detail` stay directly
   addressable after construction even though the composite literal sets
   `Type`/`Target` via the nested `Address:` key — Go promotes embedded
   fields for reads/writes, only the literal needs the nested key.)

   Add `commonaddress "monorepo/bin-common-handler/models/address"` and
   `cmcontact "monorepo/bin-contact-manager/models/contact"` imports; drop the
   `cmrequest` import from `server`.

4. **Test files** (`contact_test.go`, `serviceagent_contact_test.go`,
   `service_agents_contacts_test.go`) — same mechanical type swap plus field
   nesting update.

5. **Generated mocks regenerate as a byproduct of each service's own
   `go generate ./...`** — do not hand-edit them.
   - `bin-common-handler/pkg/requesthandler/mock_main.go` regenerates when
     running the verification workflow in `bin-common-handler` (step 1's
     `ContactV1ContactCreate` signature change).
   - `bin-api-manager/pkg/servicehandler/mock_main.go` ALSO imports
     `cmrequest` today (line 45, aliased `request`) and mocks
     `ContactCreate`/`ServiceAgentContactCreate` with the
     `[]request.AddressCreate` signature — this file is itself one of the 9
     violating files and regenerates when running the verification workflow
     in `bin-api-manager` (step 2's interface signature change). Both
     services' `go generate ./...` steps in the Verification plan below must
     run; skipping either leaves a stale mock that won't compile against the
     new interface.

### Non-goals

- Not touching any other DTO/domain boundary in this PR (e.g. other
  `request.*`/`response.*` usages elsewhere in the repo were not found by the
  grep sweep — this is the only violation of this rule in the tree today).
- Not changing the wire JSON shape of `POST /v1/contacts` or
  `POST /service_agents/contacts` — `request.AddressCreate`'s JSON tags are
  unchanged, so the OpenAPI contract and RST docs are unaffected.
- `Address` IS restructured (to embed `AddressInput`, see Fix above), but
  its public JSON field set, tags, and semantics are unchanged — only the
  internal Go struct composition and JSON key ordering shift.

## Verification plan

Per CLAUDE.md, run the full verification workflow in each touched service:

```bash
cd bin-contact-manager  && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-common-handler   && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-api-manager      && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Additionally, re-run the layering grep from the audit to confirm zero
remaining violations:

```bash
grep -rln --include='*.go' -E 'pkg/listenhandler/models/(request|response)"' . \
  | grep -v '/.worktrees/' | grep -v '/vendor/' | grep -v '/listenhandler/' | grep -v requesthandler
```

Expect empty output (down from the current 9 files).

## Risk

Low. Mechanical type substitution behind an unchanged JSON wire shape; no
behavior change. Three services touched but each change is localized and
covered by existing unit tests, which will be updated in the same PR.
