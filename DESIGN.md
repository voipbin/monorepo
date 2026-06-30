# DESIGN: Unify Contact Addresses API to Match DB Schema

**Issue:** #1031
**Branch:** NOJIRA-Unify-Contact-addresses-API-to-match-DB-schema
**Author:** Hermes (CPO)
**Status:** DRAFT v7 — Round-6 final defect fixed — APPROVED FOR IMPLEMENTATION

---

## 1. Background

VOIP-1207 migrated `contact_phone_numbers` + `contact_emails` to a single
`contact_addresses` table (columns: `id`, `customer_id`, `contact_id`, `type`,
`target`, `target_name`, `is_primary`, `tm_create`).

The DB layer already uses `contact_addresses` exclusively. The API surface,
Go model structs, OpenAPI spec, and frontend still expose the old split shape.

This PR completes the unification by removing `phone_numbers[]`/`emails[]`
from every layer and replacing them with a single `addresses[]` field.

---

## 2. Decision: Hard Cut to Addresses (no backward compat shim)

`PhoneNumbers []PhoneNumber` and `Emails []Email` are removed from:
- `contact.Contact` model struct
- `contact.WebhookMessage` struct
- `ContactCreate` / `ContactUpdate` request models
- `ContactManagerContact` OpenAPI response schema
- All API endpoints that return a Contact

The `/contacts/{id}/phone-numbers` and `/contacts/{id}/emails` endpoint
families are **removed** (not deprecated). New unified endpoints
`/contacts/{id}/addresses` replace them.

This is a coordinated change across `bin-contact-manager`, `bin-common-handler`,
`bin-api-manager`, `bin-openapi-manager` (single PR in the Go monorepo), and
`square-admin` (**separate PR in `monorepo-javascript`** — see §5.5).

---

## 3. Target State

### 3.1 `Address` model (`bin-contact-manager/models/contact/address.go`)

```go
// Address represents a single row in contact_addresses.
// type = "tel"   -> target holds an E.164 phone number
// type = "email" -> target holds a lowercase email address
type Address struct {
    ID         uuid.UUID  `json:"id"`
    CustomerID uuid.UUID  `json:"customer_id"`
    ContactID  uuid.UUID  `json:"contact_id"`
    Type       string     `json:"type"`       // "tel" | "email"
    Target     string     `json:"target"`     // E.164 or email
    IsPrimary  bool       `json:"is_primary"`
    TMCreate   *time.Time `json:"tm_create"`
}

const (
    AddressTypeTel   = "tel"
    AddressTypeEmail = "email"
)
```

No sub-type field (mobile/work/home etc.). Those sub-types were never persisted
to `contact_addresses` — the VOIP-1207 migration comment in `email.go` and
`phone_number.go` explicitly notes "sub-type is dropped (§3.2)". The DB column
`target_name` is always written as `""`. This design does not reintroduce them.

### 3.2 Updated `Contact` model

```go
// Before (removed):
PhoneNumbers []PhoneNumber `json:"phone_numbers,omitempty" db:"-"`
Emails       []Email       `json:"emails,omitempty"       db:"-"`

// After:
Addresses []Address `json:"addresses,omitempty" db:"-"`
```

### 3.3 Updated `WebhookMessage`

Same replacement: `PhoneNumbers`/`Emails` removed, `Addresses []Address` added.
`ConvertWebhookMessage()` maps `c.Addresses`.

### 3.4 Request models (listenhandler)

Remove `PhoneNumberCreate`, `PhoneNumberUpdate`, `EmailCreate`, `EmailUpdate`.
Add:

```go
// AddressCreate is the body for POST /v1/contacts/{id}/addresses
type AddressCreate struct {
    Type      string `json:"type"`       // "tel" | "email" — required
    Target    string `json:"target"`     // E.164 or email   — required
    IsPrimary bool   `json:"is_primary"`
}

// AddressUpdate is the body for PUT /v1/contacts/{id}/addresses/{address_id}
type AddressUpdate struct {
    Target    *string `json:"target,omitempty"`
    IsPrimary *bool   `json:"is_primary,omitempty"`
}
```

`ContactCreate` request: remove `PhoneNumbers []PhoneNumberCreate` and
`Emails []EmailCreate`, add `Addresses []AddressCreate`.

---

## 4. API Changes

### 4.1 New endpoints

| Method | Path | Description |
|---|---|---|
| GET    | `/contacts/{id}/addresses`              | List all addresses for a contact |
| POST   | `/contacts/{id}/addresses`              | Add an address (tel or email) |
| PUT    | `/contacts/{id}/addresses/{address_id}` | Update target or is_primary |
| DELETE | `/contacts/{id}/addresses/{address_id}` | Remove an address |

**GET `/contacts/{id}/addresses`**

Response uses standard list envelope consistent with other sub-resource lists:
```json
{
  "result": [
    {
      "id": "...",
      "customer_id": "...",
      "contact_id": "...",
      "type": "tel",
      "target": "+15551234567",
      "is_primary": true,
      "tm_create": "2026-01-01T00:00:00Z"
    }
  ]
}
```

**POST `/contacts/{id}/addresses`**

Request:
```json
{ "type": "tel", "target": "+15556543210", "is_primary": false }
```

Response: updated `Contact` object (same as existing phone-numbers/emails POST pattern).

Validation:
- `type` must be `"tel"` or `"email"`. Any other value returns 400 `INVALID_ARGUMENT`.
- `target` must be non-empty.
- `"tel"`: E.164 normalization applied via `commonaddress.NormalizeTarget`.
- `"email"`: lowercase normalization via `commonaddress.NormalizeTarget`.

**PUT `/contacts/{id}/addresses/{address_id}`**

Request: `{ "target": "+15559999999" }` or `{ "is_primary": true }` or both.

Response: updated `Contact` object.

**DELETE `/contacts/{id}/addresses/{address_id}`**

Response: updated `Contact` object.

### 4.2 Removed endpoints

All of the following are removed (no deprecated shim):
- `POST   /contacts/{id}/phone-numbers`
- `PUT    /contacts/{id}/phone-numbers/{phone_number_id}`
- `DELETE /contacts/{id}/phone-numbers/{phone_number_id}`
- `POST   /contacts/{id}/emails`
- `PUT    /contacts/{id}/emails/{email_id}`
- `DELETE /contacts/{id}/emails/{email_id}`

`square-admin` is updated in the same PR, so there are no in-flight consumers
of those endpoints after merge.

---

## 5. Implementation Plan

### 5.1 `bin-contact-manager`

#### New file: `models/contact/address.go`
- `Address` struct
- `AddressTypeTel`, `AddressTypeEmail` constants
- `AddressField` constants with explicit DB column mapping:

```go
const (
    AddressFieldTarget    AddressField = "target"     // maps to DB column target
    AddressFieldIsPrimary AddressField = "is_primary" // maps to DB column is_primary
)
```

Callers of `AddressUpdate` MUST use `AddressFieldTarget` (not `"number"` or
`"address"`, which were the old type-specific keys).

#### Modified: `models/contact/contact.go`
- Remove `PhoneNumbers []PhoneNumber` and `Emails []Email`
- Add `Addresses []Address`

#### Modified: `models/contact/webhook.go`
- `WebhookMessage`: remove `PhoneNumbers`/`Emails`, add `Addresses []Address`
- `ConvertWebhookMessage()`: map `c.Addresses`

#### Delete: `models/contact/phonenumber.go` and `models/contact/email.go`
- These structs and their constants have no remaining callers after the migration.

#### Modified: `models/contact/field.go`
- Remove `PhoneNumberField` and `EmailField` constant blocks.
- Add `AddressField` constants.

#### Modified: `pkg/dbhandler/main.go` (interface)
- Remove: `PhoneNumber*`, `Email*` method signatures
- Add: `AddressCreate`, `AddressGet`, `AddressListByContactID`, `AddressUpdate`,
  `AddressDelete`, `AddressResetPrimary`

**Reconciliation with existing partial Address primitives:**

The current interface already has two address-related methods with different names
and return types. These are explicitly replaced as follows:

| Existing method | Action | New method |
|---|---|---|
| `AddressGetByID(ctx, customerID, id) (AddressPair, error)` | RENAME + return type change | `AddressGet(ctx, customerID, id) (*contact.Address, error)` |
| `AddressListByContact(ctx, customerID, contactID) ([]AddressPair, error)` | REPLACE | `AddressListByContactID(ctx, contactID) ([]contact.Address, error)` |

`AddressPair` (currently defined in `interaction.go`) is kept for `InteractionList`
which uses `(type, target)` pairs for SQL IN-list expansion — this is a separate
concern from the public Address model. `AddressPair` is NOT deleted.

**`AddressGet` signature retains `customerID`:**
```go
AddressGet(ctx context.Context, customerID, id uuid.UUID) (*contact.Address, error)
```
This preserves the cross-tenant guard that exists in the current `AddressGetByID`
implementation (verified by `Test_AddressGetByID` cross-tenant test in
`address_test.go`). The `contacthandler` layer passes `c.CustomerID` — obtained
from the prior `ContactGet` — to `AddressGet` in all `UpdateAddress`/`RemoveAddress`
calls.

`AddressListByContactID` omits `customerID` (consistent with existing
`PhoneNumberListByContactID` / `EmailListByContactID` signatures). The
`ContactGet` that precedes it guarantees contact ownership; no additional
DB-layer tenant filtering is needed for the list path.

#### Modified: `pkg/dbhandler/address.go`
- Extend `addressRow` to include `type` column (currently missing from scan target).
- Implement `AddressCreate`, `AddressGet`, `AddressListByContactID`,
  `AddressUpdate`, `AddressDelete`, `AddressResetPrimary` as first-class methods.
- `AddressResetPrimary` replaces `addressResetPrimaryForContact` (make exported).
- **Each mutating method (`AddressCreate`, `AddressUpdate`, `AddressDelete`,
  `AddressResetPrimary`) MUST call `h.contactUpdateToCache(ctx, contactID)` after
  the DB write**, matching the pattern in the deleted `phone_number.go` and
  `email.go`. For `AddressUpdate`/`AddressDelete`, resolve `contactID` first via
  the existing private helper `h.addressContactID(id uuid.UUID)` before the
  mutation.

#### Delete: `pkg/dbhandler/phone_number.go` and `pkg/dbhandler/email.go`
- All logic migrated to `address.go`.

#### Modified: `pkg/listenhandler/models/request/contacts.go`
- Remove `PhoneNumberCreate`, `PhoneNumberUpdate`, `EmailCreate`, `EmailUpdate`
- Add `AddressCreate`, `AddressUpdate`
- `ContactCreate`: replace `PhoneNumbers []PhoneNumberCreate` and
  `Emails []EmailCreate` with `Addresses []AddressCreate`

#### Modified: `pkg/listenhandler/main.go`
- Remove regex and routing for `/phone-numbers` and `/emails`
- Add `regV1ContactsAddresses` and `regV1ContactsAddressesID`
- Add routing cases: GET + POST `/addresses`, PUT + DELETE `/addresses/{id}`

#### Modified: `pkg/listenhandler/v1_contacts.go`
- `processV1ContactsPost`: replace `c.PhoneNumbers`/`c.Emails` construction
  (iterating over `reqData.PhoneNumbers` / `reqData.Emails`) with
  `c.Addresses` construction (iterating over `reqData.Addresses`, mapping each
  to `contact.Address{Type: a.Type, Target: a.Target, IsPrimary: a.IsPrimary}`).

#### New file: `pkg/listenhandler/v1_contacts_addresses.go`
- `processV1ContactsAddressesGet` (GET list — returns `[]Address`, list envelope)
- `processV1ContactsAddressesPost` (POST create — returns updated `Contact`)
- `processV1ContactsAddressesIDPut` (PUT update — returns updated `Contact`)
- `processV1ContactsAddressesIDDelete` (DELETE — returns updated `Contact`)

#### Delete: `pkg/listenhandler/v1_contacts_phonenumbers.go`
#### Delete: `pkg/listenhandler/v1_contacts_emails.go`
(if these files exist; otherwise remove the functions from `v1_contacts.go`)

#### Modified: `pkg/contacthandler/contact.go`
- `Create()`: migrate inline address creation loop. Full pseudo-code:
  ```
  addresses := c.Addresses
  c.PhoneNumbers = nil  // removed field — no-op in new struct, kept for clarity
  c.Emails = nil        // removed field — no-op in new struct, kept for clarity
  c.Addresses = nil     // MUST zero before db.ContactCreate to avoid DB write
  c.TagIDs = nil

  db.ContactCreate(ctx, c)  // inserts contact row only

  for _, a := range addresses {
      a.ID = UUIDCreate(); a.CustomerID = c.CustomerID; a.ContactID = c.ID
      // normalize target
      if a.Type == AddressTypeTel   { a.Target = normalizeE164("", a.Target) }
      if a.Type == AddressTypeEmail { a.Target, _ = NormalizeTarget(TypeEmail, a.Target) }
      if a.IsPrimary { db.AddressResetPrimary(ctx, c.ID) }  // cross-type reset
      db.AddressCreate(ctx, &a)
  }

  // Tag handling: UNCHANGED — loop over tagIDs calling db.TagAssignmentCreate
  for _, tagID := range tagIDs { db.TagAssignmentCreate(ctx, c.ID, tagID) }
  ```
  Note: `AddressResetPrimary` is now **cross-type** — it resets all address
  primaries for the contact (both tel and email), replacing the previous per-type
  behavior of `PhoneNumberResetPrimary`/`EmailResetPrimary`.

- Remove `AddPhoneNumber`, `UpdatePhoneNumber`, `RemovePhoneNumber`, `AddEmail`, `UpdateEmail`, `RemoveEmail`
- Add: `AddAddress`, `UpdateAddress`, `RemoveAddress`

**`AddAddress` logic:**
```
1. ContactGet to verify existence and get CustomerID
2. Assign ID, CustomerID, ContactID
3. Normalize target: tel -> E.164, email -> lowercase
4. If IsPrimary: call AddressResetPrimary(contactID)
5. AddressCreate(ctx, addr)
6. ContactGet to return updated Contact
7. publishEvent(ContactUpdated)
```

**`UpdateAddress` logic:**
```
1. AddressGet(ctx, customerID, addressID) — returns ErrNotFound if absent or wrong tenant
   → if ErrNotFound: return 404 CONTACT_ADDRESS_NOT_FOUND
2. Normalize target if being updated (dispatch on type from step 1)
3. If IsPrimary=true: AddressResetPrimary(contactID)
4. AddressUpdate(addressID, fields)
5. ContactGet, publishEvent
```

**`RemoveAddress` logic:**
```
1. AddressGet(ctx, customerID, addressID) — 404 if absent or wrong tenant
2. AddressDelete(addressID)
3. ContactGet, publishEvent
```

#### Modified: `pkg/dbhandler/contact.go` — `ContactGet`
```go
// Before:
res.PhoneNumbers, _ = h.PhoneNumberListByContactID(ctx, id)
res.Emails, _ = h.EmailListByContactID(ctx, id)

// After:
res.Addresses, _ = h.AddressListByContactID(ctx, id)
```

#### Modified: `pkg/contacthandler/main.go` (interface)
- Remove: `AddPhoneNumber`, `UpdatePhoneNumber`, `RemovePhoneNumber`, `AddEmail`, `UpdateEmail`, `RemoveEmail`
- Add: `AddAddress`, `UpdateAddress`, `RemoveAddress`

### 5.2 `bin-common-handler`

#### Delete: `pkg/requesthandler/contact_phonenumbers.go`
#### Delete: `pkg/requesthandler/contact_emails.go`

#### New file: `pkg/requesthandler/contact_addresses.go`
- `ContactV1AddressCreate(ctx, contactID, addrType, target, isPrimary) (*cmcontact.Contact, error)`
- `ContactV1AddressUpdate(ctx, contactID, addressID, fields) (*cmcontact.Contact, error)`
- `ContactV1AddressDelete(ctx, contactID, addressID) (*cmcontact.Contact, error)`

**No standalone `ContactV1AddressList` RPC.** This follows the established codebase
pattern: sub-resource lists (tags, phone numbers, emails) are returned as part of
`ContactGet`, not via a separate list RPC. `bin-api-manager`'s
`GetContactsIdAddresses` handler calls `ContactV1ContactGet` and returns
`contact.Addresses` from the response. This reuses the Redis contact body cache
and avoids introducing a new RabbitMQ message type.

#### Modified: `pkg/requesthandler/contact_contacts.go`
- `ContactV1ContactCreate` signature: replace `phoneNumbers []cmrequest.PhoneNumberCreate,
  emails []cmrequest.EmailCreate` with `addresses []cmrequest.AddressCreate`
- Update the marshalled `request.ContactCreate` body accordingly

#### Modified: `pkg/requesthandler/main.go` (interface)
- Remove: `ContactV1PhoneNumber*`, `ContactV1Email*` method signatures
- Update: `ContactV1ContactCreate` signature (addresses param)
- Add: `ContactV1AddressCreate`, `ContactV1AddressUpdate`, `ContactV1AddressDelete`

### 5.3 `bin-openapi-manager`

#### New schema: `ContactManagerAddress`
```yaml
ContactManagerAddress:
  type: object
  properties:
    id:
      type: string
      format: uuid
    customer_id:
      type: string
      format: uuid
    contact_id:
      type: string
      format: uuid
    type:
      $ref: '#/components/schemas/ContactManagerAddressType'
    target:
      type: string
      description: E.164 phone number (type=tel) or email address (type=email).
    is_primary:
      type: boolean
    tm_create:
      type: string
      format: date-time
```

#### New schema: `ContactManagerAddressType`
```yaml
ContactManagerAddressType:
  type: string
  enum:
    - tel
    - email
```

Note: enum values are wire values (`"tel"`, `"email"`), not Go constant identifiers.
Go code generation will produce `ContactManagerAddressTypeTel` / `ContactManagerAddressTypeEmail`
constants mapped to these wire strings.

#### Modified: `ContactManagerContact` schema
- Remove `phone_numbers` and `emails` fields
- Add `addresses` field: `type: array`, items `$ref: ContactManagerAddress`

#### Remove schemas: `ContactManagerPhoneNumber`, `ContactManagerPhoneNumberType`, `ContactManagerEmail`, `ContactManagerEmailType`

#### New path files:
- `openapi/paths/contacts/id_addresses.yaml` (GET + POST)
- `openapi/paths/contacts/id_addresses_id.yaml` (PUT + DELETE)

#### Remove path files:
- `openapi/paths/contacts/id_phonenumbers.yaml`
- `openapi/paths/contacts/id_phonenumbers_id.yaml`
- `openapi/paths/contacts/id_emails.yaml`
- `openapi/paths/contacts/id_emails_id.yaml`

#### New path files (service agents):
- `openapi/paths/service_agents/contacts_id_addresses.yaml` (POST)
- `openapi/paths/service_agents/contacts_id_addresses_id.yaml` (PUT + DELETE)

Note: No separate GET `/service-agents/{id}/contacts/{contact_id}/addresses` path
is added. The existing GET `/service-agents/{id}/contacts/{contact_id}` in
`contacts_id.yaml` returns `ContactManagerContact` (via `$ref`), which already
embeds `addresses[]` after the schema update — consumers get the full address
list via the contact fetch automatically.

#### Remove path files (service agents):
- `openapi/paths/service_agents/contacts_id_phonenumbers.yaml`
- `openapi/paths/service_agents/contacts_id_phonenumbers_id.yaml`
- `openapi/paths/service_agents/contacts_id_emails.yaml`
- `openapi/paths/service_agents/contacts_id_emails_id.yaml`

#### Modified: `openapi/openapi.yaml`
- Remove old path registrations and schemas
- Add new path registrations and schemas
- Update `PostContacts` request body: replace `phone_numbers`/`emails` with `addresses`

### 5.4 `bin-api-manager`

#### Modified: `server/contacts.go`
- Remove: `PostContactsIdPhoneNumbers`, `PutContactsIdPhoneNumbersPhoneNumberId`,
  `DeleteContactsIdPhoneNumbersPhoneNumberId`, `PostContactsIdEmails`,
  `PutContactsIdEmailsEmailId`, `DeleteContactsIdEmailsEmailId`
- Add: `GetContactsIdAddresses`, `PostContactsIdAddresses`,
  `PutContactsIdAddressesAddressId`, `DeleteContactsIdAddressesAddressId`
- `PostContacts`: replace the two loops over `req.PhoneNumbers`/`req.Emails` with
  a single loop over `req.Addresses` calling `serviceHandler.ContactCreate(...)`.
  **This is a manual handler body edit** — `go generate` updates the
  `PostContactsJSONBody` type (from `phone_numbers`/`emails` to `addresses`) but
  the handler iteration logic must be updated by hand.

#### Modified: `server/service_agents_contacts.go`
This file has a complete parallel phone-number/email sub-resource handler set
for the service agents contacts resource. All must be replaced:
- Remove: `PostServiceAgentsContactsIdPhoneNumbers`,
  `PutServiceAgentsContactsIdPhoneNumbersPhoneNumberId`,
  `DeleteServiceAgentsContactsIdPhoneNumbersPhoneNumberId`,
  `PostServiceAgentsContactsIdEmails`,
  `PutServiceAgentsContactsIdEmailsEmailId`,
  `DeleteServiceAgentsContactsIdEmailsEmailId`
- Add: `PostServiceAgentsContactsIdAddresses`,
  `PutServiceAgentsContactsIdAddressesAddressId`,
  `DeleteServiceAgentsContactsIdAddressesAddressId`
- `PostServiceAgentsContacts` contact-create path: replace `req.PhoneNumbers` /
  `req.Emails` inline processing with `req.Addresses`

#### Modified: `pkg/servicehandler/contact.go`
This file contains `ContactCreate` and type-specific sub-resource methods that
are separate from the `ServiceAgentContact*` methods:
- `ContactCreate` signature: replace `phoneNumbers []cmrequest.PhoneNumberCreate,
  emails []cmrequest.EmailCreate` with `addresses []cmrequest.AddressCreate`
- Update the `h.reqHandler.ContactV1ContactCreate(ctx, …, phoneNumbers, emails, tagIDs)`
  **call site body** to pass `addresses` in place of `phoneNumbers, emails`
- Remove: `ContactPhoneNumberCreate`, `ContactPhoneNumberUpdate`,
  `ContactPhoneNumberDelete`, `ContactEmailCreate`, `ContactEmailUpdate`,
  `ContactEmailDelete`
- Add: `ContactAddressCreate`, `ContactAddressUpdate`, `ContactAddressDelete`
  (delegate to `ContactV1AddressCreate`, `ContactV1AddressUpdate`,
  `ContactV1AddressDelete` respectively)

#### Modified: `pkg/servicehandler/serviceagent_contact.go`
- Remove: `ServiceAgentContactPhoneNumberCreate`, `ServiceAgentContactPhoneNumberUpdate`,
  `ServiceAgentContactPhoneNumberDelete`, `ServiceAgentContactEmailCreate`,
  `ServiceAgentContactEmailUpdate`, `ServiceAgentContactEmailDelete`
- Add: `ServiceAgentContactAddressCreate`, `ServiceAgentContactAddressUpdate`,
  `ServiceAgentContactAddressDelete`

#### Modified: `pkg/servicehandler/main.go` (interface)
- Remove: `ContactPhoneNumberCreate`, `ContactPhoneNumberUpdate`,
  `ContactPhoneNumberDelete`, `ContactEmailCreate`, `ContactEmailUpdate`,
  `ContactEmailDelete`, `ServiceAgentContactPhoneNumber*`, `ServiceAgentContactEmail*`
- Add: `ContactAddressCreate`, `ContactAddressUpdate`, `ContactAddressDelete`,
  `ServiceAgentContactAddressCreate`, `ServiceAgentContactAddressUpdate`,
  `ServiceAgentContactAddressDelete`
- `ContactCreate` signature updated (addresses param)

After `go generate ./...` on updated `bin-openapi-manager`, the generated server
interface adds the new method signatures and drops the old ones.

#### RST Docs (`bin-api-manager/docsdev/source/`)
Per `bin-api-manager/CLAUDE.md`, all user-visible API changes require RST doc updates:
- Remove `/phone-numbers` and `/emails` endpoint docs from the contacts section
- Add `/addresses` endpoint docs
- Update `ContactManagerContact` schema description (addresses[] field)
- Rebuild HTML per CLAUDE.md workflow

### 5.5 `square-admin` (`monorepo-javascript` — separate PR)

`square-admin` lives in the **`monorepo-javascript`** repository, which is
separate from the Go monorepo. This work requires a **separate PR** in
`monorepo-javascript`, coordinated with the Go monorepo PR (merge Go PR first,
then the JS PR).

#### Modified: `square-admin/src/views/contacts/contacts_detail.js`
- Replace `/phone-numbers` and `/emails` GET/POST/PUT/DELETE calls with `/addresses`
- `type="tel"` rows rendered in phone section, `type="email"` in email section
- Add/edit form adds a `type` selector (`tel` | `email`) or uses separate form paths

---

## 6. `addressRow` Fix (Critical)

`dbhandler/address.go` currently defines:
```go
type addressRow struct {
    ID         uuid.UUID
    CustomerID uuid.UUID
    ContactID  uuid.UUID
    Target     string
    IsPrimary  bool
    TMCreate   *time.Time
}
```

The `type` column is absent. This means `scanAddressRow` cannot populate
`Address.Type`. Must add:
```go
Type string `db:"type"`
```

and include `"type"` in `addressRowColumns()`. All existing callers
(`PhoneNumberGet`, `EmailGet`, etc.) scoped their queries with a WHERE on type,
so they do not rely on the scanned value — this addition is safe and
non-breaking for those paths.

---

## 7. Ripple: Other Services Consuming `contact.Contact`

Services that vendor `bin-contact-manager` and access `PhoneNumbers`/`Emails`
fields must be updated. Scan for usages:

```
grep -r "\.PhoneNumbers\|\.Emails\b" --include="*.go" -l
```

Known candidates from initial scan:
- `bin-contact-manager/pkg/contacthandler/contact.go` — covered above
- `bin-contact-manager/models/contact/webhook.go` — covered above
- `bin-contact-manager/pkg/listenhandler/v1_contacts.go` — covered above
- `bin-api-manager/server/contacts.go` — covered above
- `bin-openapi-manager/gens/models/gen.go` — regenerated by `go generate`
- `bin-api-manager/gens/openapi_server/gen.go` — regenerated by `go generate`

Any other service vendoring `bin-contact-manager` (e.g., `bin-tag-manager`,
`bin-email-manager`, `bin-sentinel-manager`) must run `go mod vendor` to pick
up the updated model. If those vendor copies reference `.PhoneNumbers` or
`.Emails` in their own source (not just vendored code), those must be patched.

Action: run the grep after implementing Step 5.1 and before committing.

---

## 8. Files Changed Summary

| File | Change |
|---|---|
| `bin-contact-manager/models/contact/address.go` | NEW |
| `bin-contact-manager/models/contact/contact.go` | REPLACE PhoneNumbers/Emails with Addresses |
| `bin-contact-manager/models/contact/webhook.go` | REPLACE PhoneNumbers/Emails with Addresses |
| `bin-contact-manager/models/contact/webhook_test.go` | UPDATE: replace PhoneNumbers/Emails assertions with Addresses |
| `bin-contact-manager/models/contact/phonenumber.go` | DELETE |
| `bin-contact-manager/models/contact/email.go` | DELETE |
| `bin-contact-manager/models/contact/field.go` | REMOVE PhoneNumberField/EmailField, ADD AddressField |
| `bin-contact-manager/pkg/dbhandler/address.go` | EXTEND: add type to addressRow, implement full Address* CRUD + contactUpdateToCache calls |
| `bin-contact-manager/pkg/dbhandler/contact.go` | MODIFY: `ContactGet` / `ContactList` — replace `PhoneNumberListByContactID`/`EmailListByContactID` calls (3 call sites) with `AddressListByContactID`; remove `res.PhoneNumbers` / `res.Emails` assignments, add `res.Addresses` |
| `bin-contact-manager/pkg/dbhandler/phone_number.go` | DELETE |
| `bin-contact-manager/pkg/dbhandler/email.go` | DELETE |
| `bin-contact-manager/pkg/dbhandler/main.go` | REPLACE PhoneNumber*/Email* interface with Address* |
| `bin-contact-manager/pkg/dbhandler/mock_main.go` | REGENERATED by `go generate ./...` |
| `bin-contact-manager/pkg/dbhandler/address_test.go` | UPDATE: rename `h.AddressListByContact(ctx, customerID, contactID)` → `h.AddressListByContactID(ctx, contactID)` (remove customerID arg); rename `h.AddressGetByID(ctx, customerID, id)` → `h.AddressGet(ctx, customerID, id)` (name only, signature unchanged); update return type assertions from `AddressPair` to `contact.Address`; add new tests for `AddressCreate`, `AddressUpdate`, `AddressDelete`, `AddressResetPrimary` |
| `bin-contact-manager/pkg/dbhandler/address_migration_test.go` | REPLACE: rewrite `Test_AddressMigration_CrossTypeSinglePrimary` using `AddressCreate`(type=tel) + `AddressResetPrimary` + `AddressCreate`(type=email), verifying the cross-type single-primary invariant |
| `bin-contact-manager/pkg/dbhandler/additional_test.go` | UPDATE: replace `PhoneNumberUpdate`/`PhoneNumberResetPrimary`/`EmailUpdate`/`EmailResetPrimary`/`PhoneNumberGet_NotFound`/`EmailGet_NotFound` test cases with `AddressUpdate`/`AddressResetPrimary`/`AddressGet_NotFound` equivalents |
| `bin-contact-manager/pkg/dbhandler/contact_test.go` | UPDATE: replace `.PhoneNumbers`/`.Emails` struct literals and assertions with `.Addresses`; replace `Test_Multiple_PhoneNumbers_ForSameContact` / `Test_Multiple_Emails_ForSameContact` with `Test_Multiple_Addresses_ForSameContact` |
| `bin-contact-manager/pkg/cachehandler/handler_test.go` | UPDATE: replace PhoneNumbers/Emails field assertions with Addresses |
| `bin-contact-manager/pkg/listenhandler/models/request/contacts.go` | REPLACE PhoneNumber*/Email* with Address* |
| `bin-contact-manager/pkg/listenhandler/main.go` | REPLACE routes |
| `bin-contact-manager/pkg/listenhandler/v1_contacts.go` | MODIFY: `processV1ContactsPost` — replace PhoneNumbers/Emails loop with Addresses loop; remove `processV1ContactsPhoneNumbers*` / `processV1ContactsEmails*` handler functions (they live inline in this file, not in separate files) |
| `bin-contact-manager/pkg/listenhandler/v1_contacts_test.go` | UPDATE: delete `TestProcessV1ContactsPhoneNumbers*` / `TestProcessV1ContactsEmails*` test functions; add `TestProcessV1ContactsAddresses*` covering GET/POST/DELETE; update `TestProcessV1ContactsPost` JSON body to use `addresses[]`; rename `TestProcessV1ContactsPost_WithPhoneAndEmail` → `TestProcessV1ContactsPost_WithAddresses` and rewrite its JSON payload from `phone_numbers[]/emails[]` to `addresses[]` using `type`/`target`/`is_primary` fields only (drop `"number"`, `"mobile"`, `"work"` sub-type keys which have no equivalent in `AddressCreate`) |
| `bin-contact-manager/pkg/listenhandler/v1_contacts_update_test.go` | UPDATE: delete `TestProcessV1ContactsPhoneNumbersIDPut*` / `TestProcessV1ContactsEmailsIDPut*`; add `TestProcessV1ContactsAddressesIDPut*` |
| `bin-contact-manager/pkg/listenhandler/v1_contacts_addresses.go` | NEW |
| `bin-contact-manager/pkg/listenhandler/v1_contacts_phonenumbers.go` | DELETE if file exists (functions may be inline in v1_contacts.go — confirm and remove accordingly) |
| `bin-contact-manager/pkg/listenhandler/v1_contacts_emails.go` | DELETE if file exists (same as above) |
| `bin-contact-manager/pkg/contacthandler/contact.go` | REPLACE AddPhoneNumber/Email with AddAddress etc. |
| `bin-contact-manager/pkg/contacthandler/contact_test.go` | UPDATE: replace `Test_AddPhoneNumber*`/`Test_AddEmail*`/`Test_RemovePhoneNumber*`/`Test_RemoveEmail*` with `Test_AddAddress*`/`Test_RemoveAddress*` |
| `bin-contact-manager/pkg/contacthandler/contact_additional_test.go` | UPDATE: replace `Test_UpdatePhoneNumber*`/`Test_UpdateEmail*`/`Test_AddPhoneNumber_ResetPrimaryError`/`Test_AddEmail_ResetPrimaryError` with `Test_UpdateAddress*`/`Test_AddAddress_ResetPrimaryError` |
| `bin-contact-manager/pkg/contacthandler/main.go` | REPLACE interface |
| `bin-contact-manager/pkg/contacthandler/mock_main.go` | REGENERATED by `go generate ./...` |
| `bin-common-handler/pkg/requesthandler/contact_phonenumbers.go` | DELETE |
| `bin-common-handler/pkg/requesthandler/contact_emails.go` | DELETE |
| `bin-common-handler/pkg/requesthandler/contact_contacts.go` | MODIFY: `ContactV1ContactCreate` signature (addresses param replaces phoneNumbers + emails) |
| `bin-common-handler/pkg/requesthandler/contact_addresses.go` | NEW |
| `bin-common-handler/pkg/requesthandler/main.go` | REPLACE interface + update `ContactV1ContactCreate` signature |
| `bin-common-handler/pkg/requesthandler/mock_main.go` | REGENERATED by `go generate ./...` |
| `bin-openapi-manager/openapi/openapi.yaml` | REPLACE schemas and paths |
| `bin-openapi-manager/openapi/paths/contacts/id_phonenumbers.yaml` | DELETE |
| `bin-openapi-manager/openapi/paths/contacts/id_phonenumbers_id.yaml` | DELETE |
| `bin-openapi-manager/openapi/paths/contacts/id_emails.yaml` | DELETE |
| `bin-openapi-manager/openapi/paths/contacts/id_emails_id.yaml` | DELETE |
| `bin-openapi-manager/openapi/paths/contacts/id_addresses.yaml` | NEW |
| `bin-openapi-manager/openapi/paths/contacts/id_addresses_id.yaml` | NEW |
| `bin-openapi-manager/openapi/paths/service_agents/contacts_id_phonenumbers.yaml` | DELETE |
| `bin-openapi-manager/openapi/paths/service_agents/contacts_id_phonenumbers_id.yaml` | DELETE |
| `bin-openapi-manager/openapi/paths/service_agents/contacts_id_emails.yaml` | DELETE |
| `bin-openapi-manager/openapi/paths/service_agents/contacts_id_emails_id.yaml` | DELETE |
| `bin-openapi-manager/openapi/paths/service_agents/contacts_id_addresses.yaml` | NEW |
| `bin-openapi-manager/openapi/paths/service_agents/contacts_id_addresses_id.yaml` | NEW |
| `bin-api-manager/server/contacts.go` | REPLACE handlers |
| `bin-api-manager/server/contacts_test.go` | UPDATE: replace `Test_PostContactsIdPhoneNumbers*`/`Test_PostContactsIdEmails*` with `Test_GetContactsIdAddresses`/`Test_PostContactsIdAddresses`/`Test_PutContactsIdAddressesAddressId`/`Test_DeleteContactsIdAddressesAddressId`; update expected JSON bodies to use `addresses[]` |
| `bin-api-manager/server/service_agents_contacts.go` | REPLACE PhoneNumber/Email handlers with Address handlers |
| `bin-api-manager/server/service_agents_contacts_test.go` | UPDATE: replace PhoneNumber/Email test cases with Address test cases |
| `bin-api-manager/pkg/servicehandler/contact.go` | REPLACE ContactPhoneNumber*/Email* with ContactAddress*, update ContactCreate signature |
| `bin-api-manager/pkg/servicehandler/contact_test.go` | UPDATE: replace Test_ContactCreate signature (addresses param), replace Test_ContactPhoneNumber*/Test_ContactEmail* with Test_ContactAddress* |
| `bin-api-manager/pkg/servicehandler/serviceagent_contact.go` | REPLACE PhoneNumber*/Email* with Address* |
| `bin-api-manager/pkg/servicehandler/serviceagent_contact_test.go` | UPDATE: replace test cases |
| `bin-api-manager/pkg/servicehandler/main.go` | REPLACE interface signatures |
| `bin-api-manager/pkg/servicehandler/mock_main.go` | REGENERATED by `go generate ./...` |
| `monorepo-javascript/square-admin/src/views/contacts/contacts_detail.js` | UPDATE to /addresses API (separate PR in monorepo-javascript) |

---

## 9. Open Questions (resolved)

| # | Question | Resolution |
|---|---|---|
| 1 | Backward compat shim? | No. Hard cut. Both sides updated atomically. |
| 2 | Sub-type (mobile/work/home)? | Not reintroduced. DB never stored them (target_name always ""). |
| 3 | GET list response envelope? | `{"result": [...]}` consistent with other list endpoints. |
| 4 | addresses[] in ContactCreate? | Yes — replaces phone_numbers[]/emails[] inline on create. |
| 5 | GET /addresses/{id} single-item? | Out of scope. PUT/DELETE identify the row by address_id in path; no pre-fetch needed. |
