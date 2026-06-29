# VOIP-1209: CRM Read API + Resolution

- Issue: VOIP-1209
- Parent design: docs/plans/2026-06-26-add-contact-crm-interaction-timeline-design.md (VOIP-1204)
- Class: new read API + new DB table (contact_resolutions) + RPC client methods
- Date: 2026-06-28

## 1. Scope

This document covers implementation step 4 of the VOIP-1204 CRM v1 plan:

- Read API for the interaction timeline (`GET /v1/interactions`)
- Resolution CRUD (`POST/DELETE /v1/interactions/{id}/resolutions`)
- Unresolved queue (`GET /v1/interactions/unresolved`)
- External RPC client additions in `bin-common-handler`

Out of scope: sessionize, backfill (M2), address management, webhook events for
resolutions.

## 2. Prerequisites (all Done)

- VOIP-1206: 3 new tables (contact_addresses, contact_interactions,
  contact_resolutions) - schema landed, Alembic head ac5d4e18060c.
- VOIP-1207: M1 address migration (phone/email -> contact_addresses), hot path
  code rewire.
- VOIP-1208: Projection handlers (EventCallCreated,
  EventConversationMessageCreated).
- VOIP-1215: Source/destination added to conversation_message webhook event
  (unblocked VOIP-1208).

## 3. New table: contact_resolutions

Already in the Alembic migration (ac5d4e18060c). Test SQL fixture is NOT yet
present; must be added.

Schema (from VOIP-1204 design doc §3.3):

```sql
create table contact_resolutions (
  id                binary(16)    not null,
  customer_id       binary(16)    not null,
  contact_id        binary(16)    not null,
  interaction_id    binary(16)    not null,
  resolution_type   varchar(255)  not null default '',
  resolved_by_type  varchar(255)  not null default '',
  resolved_by_id    binary(16)    not null,
  tm_create         datetime(6),
  tm_update         datetime(6),
  tm_delete         datetime(6),
  primary key(id)
);

create index idx_contact_resolutions_contact_interaction
  on contact_resolutions(customer_id, contact_id, tm_delete);
create index idx_contact_resolutions_interaction
  on contact_resolutions(customer_id, interaction_id, tm_delete);
```

## 4. New model: models/resolution/

New package `bin-contact-manager/models/resolution/` (parallel to
`models/interaction/`).

### resolution.go

```go
package resolution

type Resolution struct {
    ID             uuid.UUID  `json:"id"              db:"id,uuid"`
    CustomerID     uuid.UUID  `json:"customer_id"     db:"customer_id,uuid"`
    ContactID      uuid.UUID  `json:"contact_id"      db:"contact_id,uuid"`
    InteractionID  uuid.UUID  `json:"interaction_id"  db:"interaction_id,uuid"`
    ResolutionType string     `json:"resolution_type" db:"resolution_type"`
    ResolvedByType string     `json:"resolved_by_type" db:"resolved_by_type"`
    ResolvedByID   uuid.UUID  `json:"resolved_by_id"  db:"resolved_by_id,uuid"`
    TMCreate       *time.Time `json:"tm_create"       db:"tm_create"`
    TMUpdate       *time.Time `json:"tm_update"       db:"tm_update"`
    TMDelete       *time.Time `json:"tm_delete"       db:"tm_delete"`
}
```

Constants:

```go
const (
    ResolutionTypePositive = "positive"
    ResolutionTypeNegative = "negative"

    ResolvedByTypeAgent  = "agent"
    ResolvedByTypeSystem = "system"
    ResolvedByTypeRule   = "rule"
)
```

## 5. DB layer additions (pkg/dbhandler/)

### 5.1 interaction.go additions

Three new methods added to the existing file:

```go
// InteractionGet fetches a single interaction by ID.
// Returns ErrNotFound if absent.
InteractionGet(ctx context.Context, id uuid.UUID) (*interaction.Interaction, error)

// InteractionList returns a page of interactions filtered by customerID
// and optionally by peer (peer_type + peer_target) or by a set of
// (peer_type, peer_target) pairs (for contact_id expansion).
// Pagination: cursor = tm_create DESC + id DESC.
//
// EMPTY addressSet GUARD: if peerType, peerTarget are both empty AND
// addressSet is empty, return (nil, nil) immediately without issuing a
// SQL query. An empty IN () is invalid SQL in both MySQL and SQLite.
// The contactID read path calls this with an empty addressSet when the
// contact has no registered addresses, and correctly falls back to
// manual-resolution-only attribution.
InteractionList(
    ctx context.Context,
    customerID uuid.UUID,
    size uint64,
    token string,
    peerType, peerTarget string,        // direct peer filter (optional)
    addressSet []AddressPair,           // contact_id expansion (optional)
) ([]*interaction.Interaction, error)

// InteractionListByIDs fetches interactions for a given set of IDs,
// scoped to customerID (tenant guard — never returns another tenant's rows).
// Used during resolution union/minus in the contact_id read path to
// load manually attributed interactions not in the peer set.
InteractionListByIDs(ctx context.Context, customerID uuid.UUID, ids []uuid.UUID) ([]*interaction.Interaction, error)
```

`AddressPair` is an exported struct (capitalised so contacthandler can construct
it without a type-assertion on the interface):

```go
// AddressPair is a (type, target) pair used for multi-column IN expansion.
// Exported so contacthandler and tests can construct slices without
// importing dbhandler internals.
type AddressPair struct { Type, Target string }
```

**Pagination token encoding.** Token = `base64(json({"tm_create":"<RFC3339Nano>","id":"<uuid>"}))`.
Page boundary: `WHERE (tm_create, id) < (cursor.tm_create, cursor.id)` (tuple
comparison valid for DESC order: each next page returns rows strictly older than
the cursor row). `ORDER BY tm_create DESC, id DESC LIMIT size+1` (extra row
signals hasMore). An empty token string means "start from the newest row."

**Shared encode/decode helpers.** Locate in `pkg/dbhandler/interaction.go`:
```go
func encodePageToken(tm *time.Time, id uuid.UUID) string  // base64+json
func decodePageToken(token string) (time.Time, uuid.UUID, error) // inverse
```
These are package-internal helpers. contacthandler does not need to call them;
it passes token strings through to dbhandler unmodified.

**SQLite note.** Tests use SQLite (in-memory). Tuple comparison `(a, b) < (x, y)`
is valid SQL and SQLite supports it since 3.15. The IN multi-column form
`(peer_type, peer_target) IN (...)` is also supported in SQLite 3.15+; use it
for addressSet expansion.

**Squirrel limitation.** `squirrel` does not natively support multi-column IN.
Use raw `sq.Expr("(peer_type, peer_target) IN (%s)", placeholders...)` with
manually built `(?,?),(?,?)...` placeholder list and flattened arg slice. The
empty-addressSet guard above ensures this branch is never reached with an empty
list.

### 5.2 resolution.go (new file)

```go
// ResolutionCreate inserts a resolution row.
// No idempotency key: the grain is (contact_id, interaction_id, resolution_type);
// duplicates are caller-prevented (listenhandler checks existing before insert).
ResolutionCreate(ctx context.Context, r *resolution.Resolution) error

// ResolutionDelete soft-deletes a resolution (sets tm_delete = now).
ResolutionDelete(ctx context.Context, id uuid.UUID) error

// ResolutionListByInteraction returns all ACTIVE resolutions for an interaction
// (tm_delete IS NULL), scoped to customer_id.
ResolutionListByInteraction(
    ctx context.Context,
    customerID uuid.UUID,
    interactionID uuid.UUID,
) ([]*resolution.Resolution, error)

// ResolutionListByContact returns all ACTIVE resolutions for a contact,
// used during contact_id read-path set-MINUS combination.
ResolutionListByContact(
    ctx context.Context,
    customerID uuid.UUID,
    contactID uuid.UUID,
) ([]*resolution.Resolution, error)
```

## 6. DBHandler interface additions

Add to `pkg/dbhandler/main.go` DBHandler interface:

```go
// Interaction read operations
InteractionGet(ctx context.Context, id uuid.UUID) (*interaction.Interaction, error)
InteractionList(ctx context.Context, customerID uuid.UUID, size uint64, token string,
    peerType, peerTarget string, addressSet []AddressPair) ([]*interaction.Interaction, error)
InteractionListByIDs(ctx context.Context, customerID uuid.UUID, ids []uuid.UUID) ([]*interaction.Interaction, error)

// Address read operations (used by contact_id and address_id read paths)
AddressListByContact(ctx context.Context, customerID, contactID uuid.UUID) ([]AddressPair, error)
AddressGetByID(ctx context.Context, id uuid.UUID) (AddressPair, error)

// Resolution operations
ResolutionCreate(ctx context.Context, r *resolution.Resolution) error
ResolutionDelete(ctx context.Context, id uuid.UUID) error
ResolutionListByInteraction(ctx context.Context, customerID uuid.UUID, interactionID uuid.UUID) ([]*resolution.Resolution, error)
ResolutionListByContact(ctx context.Context, customerID uuid.UUID, contactID uuid.UUID) ([]*resolution.Resolution, error)
```

Note: `AddressGet` returns enough information for the `address_id` read path
to extract `(type, target)`. Expose a small public struct
`AddressRecord { Type, Target string }` or reuse `AddressPair`. Keep internal
`addressRow` unexported; `AddressGet` returns `AddressPair` directly.

## 7. Business logic layer (pkg/contacthandler/)

### 7.1 interaction_read.go (new file)

```go
// InteractionGet returns a single interaction, scoped to customerID.
InteractionGet(ctx context.Context, customerID, id uuid.UUID) (*interaction.Interaction, error)

// InteractionList is the main timeline read path.
// Exactly one of (peerType+peerTarget), contactID, or addressID must be non-zero.
// Returns a response envelope with items and next_page_token.
InteractionList(
    ctx context.Context,
    customerID uuid.UUID,
    size uint64,
    token string,
    peerType, peerTarget string,  // ?peer_type=&peer_target= path
    contactID uuid.UUID,          // ?contact_id= path
    addressID uuid.UUID,          // ?address_id= path
) (*InteractionListResponse, error)

// InteractionListUnresolved returns interactions with zero-contact attribution
// (no peer match to any active address, no active positive resolution) AND
// peer_type != web_session. Supports pagination (size + token).
InteractionListUnresolved(ctx context.Context, customerID uuid.UUID, size uint64, token string, since time.Time) (*InteractionListResponse, error)
```

Response envelope:

```go
// InteractionListResponse is the response shape for all interaction list endpoints.
// Defined in models/interaction/ (not in pkg/contacthandler) to avoid circular imports:
// bin-common-handler/pkg/requesthandler imports bin-contact-manager/models/interaction,
// and bin-contact-manager/pkg/contacthandler imports bin-common-handler/pkg/requesthandler.
// Placing this type in pkg/contacthandler would create a confirmed cycle.
//
// next_page_token is empty if no more pages exist.
type InteractionListResponse struct {
    Items         []*Interaction `json:"items"`
    NextPageToken string         `json:"next_page_token"`
}
```

File location: `bin-contact-manager/models/interaction/list_response.go`

This envelope is returned by the listen handler serialised as JSON. All three
consumer services (ai-manager, flow-manager, api-manager) receive an opaque
`next_page_token` string and forward it back verbatim on subsequent requests.
No client-side token encoding is needed.

**contact_id resolution algorithm** (VOIP-1204 §5.1):

```
STEP 0: Existence + tenant check.
   c, err := h.db.ContactGet(ctx, contactID)
   if err != nil (ErrNotFound) OR c.CustomerID != customerID:
     return nil, ErrNotFound   // cross-tenant guard: never leak row existence
   (contactHandler.Get also checks soft-delete; the above uses dbhandler.ContactGet
    directly and must also verify c.TMDelete == nil for a soft-deleted contact.)

STEP 1: Expand address set.
   h.db.AddressListByContact(ctx, customerID, contactID)
   -> addressPairs []AddressPair{(type, target), ...}  // may be empty

STEP 2: Fetch ALL automatic peer matches (internal cap, NOT caller page size).
   If addressPairs is non-empty:
     h.db.InteractionList(ctx, customerID, internalCap, "" /*token*/, "", "", addressPairs)
     -> automatic []Interaction  // up to internalCap rows; cap = max(5000, size*100)
   Else:
     automatic = []  // short-circuit; do NOT call with empty addressPairs

   Note: token is "" here (unconditional scan from newest). The caller-supplied
   token is applied in STEP 6 AFTER set-MINUS, so no candidate is missed by
   premature cursor truncation. The internal cap (5000) is the known v1 limitation:
   contacts with >5000 automatic interactions may see incomplete results on this
   path; a cursor-walk implementation is the M2 solution.

STEP 3: Fetch active resolutions.
   h.db.ResolutionListByContact(ctx, customerID, contactID)
   -> []Resolution (tm_delete IS NULL)

STEP 4: Set-MINUS combination.
   positiveIDs  := {r.InteractionID | r.ResolutionType == "positive"}
   negativeIDs  := {r.InteractionID | r.ResolutionType == "negative"}
   automaticIDs := {i.ID | i in automatic}

   include = (automaticIDs | positiveIDs) - negativeIDs
   de-dup by ID

   PRECEDENCE: negative always wins, regardless of tm_create ordering.
   Do NOT use latest-polarity (LWW). This is a set-MINUS, not a timestamp race.

STEP 5: Load positive-only interactions not in automatic set.
   missing = positiveIDs - automaticIDs
   If missing is non-empty:
     h.db.InteractionListByIDs(ctx, customerID, missing)  // customerID-scoped!
     -> extra []Interaction
     Merge extra into result, de-dup by ID.

STEP 6: Apply include filter, sort, then apply caller cursor + size slice.
   1. Filter merged set to include IDs only.
   2. Sort by (tm_create DESC, id DESC).
   3. Apply caller token cursor: skip rows until (tm_create, id) < cursor.
   4. Take first `size` rows.
   5. Build InteractionListResponse{Items: taken, NextPageToken: encodePageToken(taken[size].TMCreate, taken[size].ID) if hasMore else ""}.
```

**InteractionListUnresolved algorithm:**

Supports pagination: `size uint64` + `token string` (same cursor format as
InteractionList). `since time.Time` is a lower bound (only interactions created
after `since` are considered). Hard cap: if `size == 0`, use default 100;
maximum allowed size is 500 (return 400 if exceeded). See §7.3 for the SQL shape.

### 7.2 resolution.go (new file in contacthandler/)

```go
// ResolutionCreate creates a new resolution.
// Validates: interaction must exist and belong to customerID.
// Validates: no existing active resolution of IDENTICAL type for (contact_id,
//   interaction_id) — prevents duplicate positive or duplicate negative rows.
//   A positive + negative coexisting is allowed (competing attributions;
//   negative wins by set-MINUS precedence). Two positives or two negatives for
//   the same pair are rejected (application-level check; see §3 note below).
ResolutionCreate(
    ctx context.Context,
    customerID, contactID, interactionID uuid.UUID,
    resolutionType, resolvedByType string,
    resolvedByID uuid.UUID,
) (*resolution.Resolution, error)

// ResolutionDelete soft-deletes a resolution (sets tm_delete = now).
// Validates: resolution must exist and belong to customerID.
// customerID is passed to ResolutionDelete in the request body (DELETE handlers
// in this service use a JSON body for customerID, consistent with all other
// contact-manager DELETE endpoints). The contacthandler checks ownership via
// the stored customer_id column before soft-deleting.
ResolutionDelete(ctx context.Context, customerID, id uuid.UUID) error
```

**§3 note — duplicate-resolution application check vs DB constraint.**
The production schema (§3) does NOT have a UNIQUE constraint on
`(customer_id, contact_id, interaction_id, resolution_type)` because MySQL
treats NULL as distinct in UNIQUE and tm_delete makes soft-deleted rows nullable;
a generated-column workaround (like the `primary_contact_uk` pattern) would be
needed to cover only active rows. The risk is a concurrent-create TOCTOU: two
simultaneous `POST /v1/interactions/{id}/resolutions` for the same grain both
pass the \"no active duplicate\" check and both insert. In a GKE multi-instance
deployment this window exists.

Decision for v1: accept application-level-only dedup. The consequence is at most
two active positive (or two active negative) rows for the same pair. Two positives
are harmless to the set-MINUS logic (the interaction is still included once, after
de-dup). Two negatives are also harmless (the interaction is still excluded). The
visible artifact is that `ResolutionListByInteraction` returns 2 rows instead of 1,
and deleting one still leaves the other active — operator would need to delete twice.
This is a UX nuisance, not a data-loss or mis-attribution bug. Track as a known
limitation; add a DB UNIQUE constraint in a follow-up if dogfood surfaces the
concurrent-create pattern.

### 7.3 Unresolved queue SQL

Efficient query shape for large interaction tables:

```sql
SELECT i.*
FROM contact_interactions i
WHERE i.customer_id = ?
  AND i.peer_type != 'web_session'
  AND NOT EXISTS (
      SELECT 1 FROM contact_addresses a
      WHERE a.customer_id = i.customer_id
        AND a.type = i.peer_type
        AND a.target = i.peer_target
  )
  AND NOT EXISTS (
      SELECT 1 FROM contact_resolutions r
      WHERE r.customer_id = i.customer_id
        AND r.interaction_id = i.id
        AND r.resolution_type = 'positive'
        AND r.tm_delete IS NULL
  )
  AND (i.tm_create, i.id) < (?, ?)   -- pagination cursor
ORDER BY i.tm_create DESC, i.id DESC
LIMIT ?
```

This query lands on indexes: `idx_contact_interactions_cursor` (customer_id,
tm_create) for the outer scan; `idx_contact_addresses_lookup` (customer_id,
type, target) for the NOT EXISTS address check;
`idx_contact_resolutions_interaction` (customer_id, interaction_id, tm_delete)
for the NOT EXISTS resolution check.

## 8. ContactHandler interface additions

Add to `pkg/contacthandler/main.go`:

```go
// Interaction read operations
InteractionGet(ctx context.Context, customerID, id uuid.UUID) (*interaction.Interaction, error)
InteractionList(ctx context.Context, customerID uuid.UUID, size uint64, token string,
    peerType, peerTarget string, contactID uuid.UUID, addressID uuid.UUID) (*interaction.InteractionListResponse, error)
InteractionListUnresolved(ctx context.Context, customerID uuid.UUID, size uint64, token string, since time.Time) (*interaction.InteractionListResponse, error)

// Resolution operations
ResolutionCreate(ctx context.Context, customerID, contactID, interactionID uuid.UUID,
    resolutionType, resolvedByType string, resolvedByID uuid.UUID) (*resolution.Resolution, error)
ResolutionDelete(ctx context.Context, customerID, id uuid.UUID) error
```

`InteractionListResponse` is defined in `models/interaction/list_response.go`, NOT in
this package, to avoid the confirmed circular import
(`pkg/contacthandler` imports `bin-common-handler/pkg/requesthandler`, and
`bin-common-handler/pkg/requesthandler` would need to import `InteractionListResponse`
back — confirmed cycle from existing imports in contacthandler/main.go).

Note: after this PR the ContactHandler interface has approximately 23 methods.
This is large but acceptable for v1; consider splitting into a sub-interface
(e.g. `InteractionHandler`) in a follow-up if the mock becomes unwieldy.

## 9. Listen handler additions (pkg/listenhandler/)

### 9.1 URL patterns (new in main.go)

```go
regV1Interactions          = /v1/interactions$
regV1InteractionsGet       = /v1/interactions\?(.*)$
regV1InteractionsUnresolved = /v1/interactions/unresolved(\?.*)?$
regV1InteractionsID        = /v1/interactions/<uuid>$
regV1InteractionsResolutions = /v1/interactions/<uuid>/resolutions$
regV1InteractionsResolutionsID = /v1/interactions/<uuid>/resolutions/<uuid>$
```

**Match order is critical.** `regV1InteractionsUnresolved` must be matched BEFORE
`regV1InteractionsID` in the switch, because `/v1/interactions/unresolved` would
otherwise match the UUID pattern (if "unresolved" were misread as a UUID, which it
won't be — regex requires `[0-9a-f]{8}-...` — but ordering still matters for
clarity and future safety).

### 9.2 v1_interactions.go (new file)

Handlers:

```
GET  /v1/interactions?...               -> processV1InteractionsGet
GET  /v1/interactions/unresolved        -> processV1InteractionsUnresolvedGet
GET  /v1/interactions/{id}              -> processV1InteractionsIDGet
POST /v1/interactions/{id}/resolutions  -> processV1InteractionsResolutionsPost
DELETE /v1/interactions/{id}/resolutions/{rid} -> processV1InteractionsResolutionsIDDelete
```

**customerID source for GET handlers.** All GET handlers extract `customer_id`
from the JSON request body, consistent with the existing `processV1ContactsGet`
pattern (`utilhandler.ParseFiltersFromRequestBody` + typed filter conversion).
The `customer_id` key is required in the request body; return 400 if absent.

Query parameter parsing for `GET /v1/interactions`:

```
?peer_type=<str>&peer_target=<str>   -> peerType + peerTarget path
?contact_id=<uuid>                   -> contactID path
?address_id=<uuid>                   -> addressID path
?page_size=<int>&page_token=<str>    -> pagination
```

`customer_id` is read from the JSON request body (not from query params).

Exactly one of the three filter modes must be present; return 400 if zero or
more than one is given.

**`since` parameter parsing** for `GET /v1/interactions/unresolved`:
```
?since=<Nd>   e.g. since=7d, since=30d (default: 30d if absent)
```
Parse `since` with a local helper `parseDaysDuration(s string) (time.Duration, error)`:
strip trailing `d`, parse integer with `strconv.Atoi`, multiply by `24*time.Hour`.
Return 400 for any other format (e.g. `7h`, `foo`, `-1d`). Cap: max `since=180d`;
return 400 if exceeded. `time.ParseDuration` is NOT used (it doesn't support `d`).

### 9.3 listenhandler/models/request/ additions

New file `v1_interactions.go`:

```go
type V1DataInteractionsResolutionsPost struct {
    CustomerID     uuid.UUID `json:"customer_id"`
    ContactID      uuid.UUID `json:"contact_id"`
    ResolutionType string    `json:"resolution_type"`
    ResolvedByType string    `json:"resolved_by_type"`
    ResolvedByID   uuid.UUID `json:"resolved_by_id"`
}
```

## 10. bin-common-handler RPC client additions

New file `bin-common-handler/pkg/requesthandler/contact_interactions.go`:

```go
// ContactV1InteractionGet fetches a single interaction.
ContactV1InteractionGet(ctx context.Context, id uuid.UUID) (*interaction.Interaction, error)

// ContactV1InteractionList lists interactions (one filter mode at a time).
// Returns *contacthandler.InteractionListResponse (imported from bin-contact-manager).
ContactV1InteractionList(
    ctx context.Context,
    size uint64,
    token string,
    customerID uuid.UUID,
    peerType, peerTarget string,
    contactID uuid.UUID,
    addressID uuid.UUID,
) (*contacthandler.InteractionListResponse, error)

// ContactV1InteractionListUnresolved lists unresolved interactions.
// since is encoded as "Nd" (e.g. "7d") in the query param.
ContactV1InteractionListUnresolved(
    ctx context.Context,
    customerID uuid.UUID,
    size uint64,
    token string,
    sinceDays int,  // e.g. 7 -> ?since=7d
) (*contacthandler.InteractionListResponse, error)

// ContactV1ResolutionCreate creates a resolution.
ContactV1ResolutionCreate(
    ctx context.Context,
    interactionID uuid.UUID,
    customerID, contactID uuid.UUID,
    resolutionType, resolvedByType string,
    resolvedByID uuid.UUID,
) (*resolution.Resolution, error)

// ContactV1ResolutionDelete soft-deletes a resolution.
// customerID is sent in the request body (DELETE body pattern, consistent
// with all other contact-manager DELETE methods).
ContactV1ResolutionDelete(ctx context.Context, customerID uuid.UUID, interactionID, resolutionID uuid.UUID) error
```

These are wrappers over `r.sendRequestContact(...)`, using queue constant from
`models/outline` (same contact-manager queue as all other `ContactV1*` methods).

Return type: `*interaction.InteractionListResponse` imported from
`monorepo/bin-contact-manager/models/interaction` — same established pattern as
`cmcontact "monorepo/bin-contact-manager/models/contact"` in `contact_contacts.go`.
No circular import: models packages are the cross-module boundary; neither
`models/interaction` nor `models/resolution` imports `bin-common-handler`.

## 11. DB helper: AddressListByContact and AddressGet

Both methods are required for the contactID and addressID read paths respectively.
Add to `pkg/dbhandler/address.go`:

```go
// AddressListByContact returns all (type, target) pairs for a contact.
// contact_addresses has no soft-delete, so no tm_delete filter needed.
AddressListByContact(ctx context.Context, customerID, contactID uuid.UUID) ([]AddressPair, error)

// AddressGetByID returns the (type, target) for a single address row by id,
// used by the address_id filter path in InteractionList.
// Returns ErrNotFound if absent.
AddressGetByID(ctx context.Context, id uuid.UUID) (AddressPair, error)
```

Both methods must be added to the DBHandler interface (§6).

## 12. Test fixtures

### 12.1 SQLite fixture addition (contacts.sql)

Append to `bin-contact-manager/scripts/database_scripts_test/contacts.sql`:

```sql
create table contact_resolutions (
  id                binary(16)    not null,
  customer_id       binary(16)    not null,
  contact_id        binary(16)    not null,
  interaction_id    binary(16)    not null,
  resolution_type   varchar(255)  not null default '',
  resolved_by_type  varchar(255)  not null default '',
  resolved_by_id    binary(16)    not null,
  tm_create         datetime(6),
  tm_update         datetime(6),
  tm_delete         datetime(6),
  primary key(id)
);

create index idx_contact_resolutions_contact_interaction
  on contact_resolutions(customer_id, contact_id, tm_delete);
create index idx_contact_resolutions_interaction
  on contact_resolutions(customer_id, interaction_id, tm_delete);
```

### 12.2 Test coverage targets

New tests required:

**dbhandler:**
- `Test_InteractionGet` (found, not found)
- `Test_InteractionList_byPeer` (direct peer path)
- `Test_InteractionList_byAddressSet` (contact_id expansion with multi-column IN)
- `Test_InteractionList_pagination` (cursor page boundary)
- `Test_InteractionListByIDs` (customerID-scoped)
- `Test_AddressListByContact`
- `Test_AddressGetByID` (found, not found)
- `Test_ResolutionCreate`
- `Test_ResolutionDelete` (soft-delete sets tm_delete)
- `Test_ResolutionListByInteraction` (active only, ignores tm_delete!=NULL rows)
- `Test_ResolutionListByContact` (active only)

**UUID namespace allocation for dbhandler tests** (all in shared SQLite DB).
Existing interaction tests use `a1b2c3d4-*`, `b1b2c3d4-*`, `c1b2c3d4-*`.
New resolution/interaction-read tests MUST use distinct namespaces to avoid
inter-test coupling on the shared in-memory DB:
- Resolution tests: `r1b2c3d4-*`
- InteractionList read tests: `l1b2c3d4-*`
- AddressListByContact tests: `al1b2c3d-*`
New tests insert all required rows (contacts, interactions, addresses) in the
test function body, never relying on side effects from other test functions.

**contacthandler (mock-based):**
- `Test_InteractionList_peerPath`
- `Test_InteractionList_contactIDPath` (verifies set-MINUS: negative wins)
- `Test_InteractionList_contactIDPath_noAddresses` (empty address set falls back to positive-only)
- `Test_InteractionList_contactIDPath_positiveWinsAutoMatch` (positive adds non-auto)
- `Test_InteractionList_contactIDPath_notFound` (step 0: contact not found -> error)
- `Test_InteractionList_unresolved` (web_session excluded, auto-matched excluded)
- `Test_ResolutionCreate_validate` (interaction not found -> error)
- `Test_ResolutionCreate_duplicateType` (dup positive -> error)

**listenhandler (mock-based):**
- `Test_V1InteractionsGet_peerPath`
- `Test_V1InteractionsGet_contactIDPath`
- `Test_V1InteractionsGet_badParams` (zero filter, multiple filters -> 400)
- `Test_V1InteractionsUnresolvedGet`
- `Test_V1InteractionsUnresolvedGet_badSince` (malformed / exceeds 180d -> 400)
- `Test_V1InteractionsIDGet`
- `Test_V1InteractionsResolutionsPost`
- `Test_V1InteractionsResolutionsIDDelete`

## 13. Mock regeneration

After modifying DBHandler or ContactHandler interfaces:

```bash
cd bin-contact-manager
go generate ./pkg/dbhandler/...
go generate ./pkg/contacthandler/...
go generate ./pkg/listenhandler/...
```

After modifying RequestHandler (in bin-common-handler):

```bash
cd bin-common-handler
go generate ./pkg/requesthandler/...
```

## 14. File change summary

### bin-contact-manager

| File | Change |
|------|--------|
| `models/resolution/resolution.go` | NEW |
| `models/interaction/list_response.go` | NEW: `InteractionListResponse{Items, NextPageToken}` |
| `scripts/database_scripts_test/contacts.sql` | APPEND contact_resolutions |
| `pkg/dbhandler/main.go` | ADD interface methods (InteractionGet/List/ListByIDs, AddressListByContact, AddressGetByID, ResolutionCreate/Delete/List*) |
| `pkg/dbhandler/interaction.go` | ADD InteractionGet, InteractionList, InteractionListByIDs + encodePageToken/decodePageToken helpers |
| `pkg/dbhandler/address.go` | ADD AddressListByContact, AddressGetByID |
| `pkg/dbhandler/resolution.go` | NEW |
| `pkg/dbhandler/interaction_test.go` | ADD read tests (namespace: l1b2c3d4-*) |
| `pkg/dbhandler/resolution_test.go` | NEW (namespace: r1b2c3d4-*) |
| `pkg/dbhandler/address_test.go` | ADD AddressListByContact, AddressGetByID tests (namespace: al1b2c3d-*) |
| `pkg/dbhandler/mock_main.go` | REGEN |
| `pkg/contacthandler/main.go` | ADD interface methods |
| `pkg/contacthandler/interaction_read.go` | NEW |
| `pkg/contacthandler/resolution.go` | NEW (contacthandler) |
| `pkg/contacthandler/interaction_read_test.go` | NEW |
| `pkg/contacthandler/resolution_test.go` | NEW |
| `pkg/contacthandler/mock_main.go` | REGEN |
| `pkg/listenhandler/main.go` | ADD URL patterns + switch cases |
| `pkg/listenhandler/v1_interactions.go` | NEW |
| `pkg/listenhandler/v1_interactions_test.go` | NEW |
| `pkg/listenhandler/models/request/v1_interactions.go` | NEW |
| `pkg/listenhandler/mock_main.go` | REGEN |

### bin-common-handler

| File | Change |
|------|--------|
| `pkg/requesthandler/contact_interactions.go` | NEW |
| `pkg/requesthandler/main.go` | ADD interface methods |
| `pkg/requesthandler/mock_main.go` | REGEN |

**bin-common-handler admission rule:** `contact_interactions` read API will be consumed by
`bin-ai-manager` (AIcall attribution), `bin-flow-manager` (Flow CRM actions),
and `bin-api-manager` (external REST proxy). That is 3+ consumers, satisfying
the admission rule (minimum 3 consumers for common-handler promotion).

## 15. Open items

1. **`since` parameter parsing.** RESOLVED: use `parseDaysDuration` helper (see §9.2).
   Default 30d if absent. Max 180d; return 400 if exceeded. `time.ParseDuration`
   is NOT used.

2. **`addressID` filter.** RESOLVED: `AddressGetByID` is added to dbhandler (§11)
   and DBHandler interface (§6). The listen handler reads `?address_id=`, calls
   `AddressGetByID` to get `(type, target)`, then calls `InteractionList` with a
   single-element `addressSet`.

3. **`since` default.** RESOLVED (in §9.2): if `since` is absent, use 30d default.

4. **RPC timeout.** Resolution Create/Delete use `requestTimeoutDefault` (same as
   all other contact-manager RPCs). No special timeout needed.

5. **bin-api-manager REST proxy.** Exposing the new endpoints through
   bin-api-manager is a separate ticket. The RPC client methods added here are
   sufficient for internal consumers (ai-manager, flow-manager).

6. **`InteractionListResponse` placement.** RESOLVED: placed in `models/interaction/list_response.go`
   (not in pkg/contacthandler). The circular import `pkg/contacthandler -> bin-common-handler/pkg/requesthandler -> pkg/contacthandler` is confirmed by existing imports; models packages are the established cross-module boundary.

7. **`resolved_by_id` system sentinel.** For system/rule-originated resolutions
   where no agent is involved, store `uuid.Nil` as `resolved_by_id`. Document this
   in the `models/resolution/resolution.go` constants block:
   `var ResolvedByIDSystem = uuid.Nil`.

8. **v1 limitation: contactID path with >5000 automatic interactions.** The internal
   cap of 5000 means contacts with more than 5000 peer-matching interactions may see
   incomplete results on the `?contact_id=` path. This is accepted for v1 expected
   volume. A cursor-walk implementation (multiple DB pages within the algorithm) is
   the M2 solution if dogfood surfaces the limit.

## 16. Invariants (do not regress)

- Negative resolution always wins over positive for the same (contact_id,
  interaction_id). LWW is forbidden.
- web_session peer_type is always excluded from the unresolved queue.
- contact_resolutions is soft-delete only (tm_delete); never physically deleted.
- contact_interactions is append-only; no UPDATE or DELETE on this table.
- Resolution does not affect interaction sort order (tm_create DESC is stable).
- Pagination cursor uses (tm_create, id), not tm_interaction.
