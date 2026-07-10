# Contact Case Create Action (Flow Action + AI Tool) — Design

**Date:** 2026-07-10
**Status:** Design — pending review
**Ticket:** VOIP-1243
**Author:** brainstormed with Lux/Hermes (CPO); owner Sungtae Kim
**Related:** `docs/plans/2026-07-07-contact-case-management-design.md` (Case entity origin design — `GetOrCreate`, ownership, lifecycle), `docs/plans/2026-06-26-add-contact-crm-interaction-timeline-design.md` (Interaction/Resolution foundation)

---

## 1. Motivation

The original Case design (2026-07-07) created `contact_cases` with an implicit,
automatic `GetOrCreate` triggered by every `call_created` /
`conversation_message_created` webhook event
(`contacthandler.EventCallCreated` / `EventConversationMessageCreated` →
`caseHandler.GetOrCreate`). Every CRM-eligible call or message therefore
always produces (or reuses) exactly one open Case, with zero control over
whether a Case should exist at all.

Product decision (2026-07 CPO discussion, VOIP-1243): flip this to an
**explicit, opt-in** model. Case creation becomes something a Flow author or
an AI decides to do — via a new `case_create` Flow action and a new
`case_create` AI tool — not something that happens automatically on every
inbound touch. Tenants who don't want Cases simply never call the action/tool
and never get one. Tenants who do want them wire `case_create` into their
Flow (or let their AI decide to create one).

## 2. Scope

### In scope

- **Remove** the automatic Case creation calls inside
  `contacthandler.EventCallCreated` / `EventConversationMessageCreated`.
  Interaction projection (the `contact_interactions` row + `deriveEndpoints` /
  `isCRMEligiblePeer` filtering) is **unchanged** — only the
  `h.caseHandler.GetOrCreate(...)` call site inside each function, and the
  resulting `CaseID: &c.ID` field on the projected `Interaction`, are removed
  (see §7 for the exact diff and the resulting behavior of
  `Interaction.CaseID`).
- **New RPC**: `ContactV1CaseCreate` on `bin-contact-manager`, exposed via
  `bin-common-handler/pkg/requesthandler`. This is a **plain Create**, not a
  get-or-create — see §3 for exact semantics.
- **New route**: `POST /v1/cases` on `bin-contact-manager`'s listenhandler
  (did not exist before this ticket; existing routes are List/Get/Close/
  Continue/Notes/Tags only).
- **New Flow action**: `case_create` (`bin-flow-manager`). Usable in both
  call-activeflow and conversation-activeflow contexts, including **mid-call**
  (i.e. while a call is actively progressing through other actions, not only
  at flow start). **Scope limit (made explicit per round-2 review): this
  action supports ONLY `activeflow.ReferenceTypeCall` and
  `activeflow.ReferenceTypeConversation`.** `bin-flow-manager`'s
  `activeflow.ReferenceType` enum has 8 real, actively-used values (`None`,
  `AI`, `API`, `Call`, `Campaign`, `Conversation`, `Transcribe`,
  `Recording` — confirmed in `models/activeflow/activeflow.go:79-87`, all
  exercised in real test fixtures, not dead enum values). For the other 5
  non-empty reference types (`AI`, `API`, `Campaign`, `Transcribe`,
  `Recording`), `case_create` is a **silent no-op**: it logs a warning and
  returns without creating a Case, without erroring the activeflow. This is
  a deliberate scope decision (a Case's peer/reference_type derivation only
  has a defined meaning for a live call or a live conversation), not an
  oversight — see §5.2 for the exact mechanics and §10 for the required
  test coverage of this no-op path.
- **New AI tool**: `case_create` (`bin-ai-manager`). Usable in both AI calls
  and AI conversations (added to `ConversationSafeTools`).
- **New optional Case fields**: `name`, `detail` (both nullable, both
  settable only at creation time in this scope). Requires an Alembic
  migration on `contact_cases`.
- **Optional initial note**: `note` param on Create, reusing the existing
  `CaseNoteCreate` primitive.
- **Failure handling**: log-and-continue. `case_create` joins the flow's
  existing non-critical-action list (`email_send`, `webhook_send`,
  `conversation_send`) — a failed Case creation must never abort the
  activeflow or the AI call/conversation.

### Explicitly out of scope

- `case_id_hint` / `contact_id` override parameters on Create — dropped
  from scope per 2026-07 discussion. `Create` does not accept either.
- Any get-or-create semantics, reuse-if-open behavior, or timeout-based
  reopen logic for Create. That entire algorithm (peer-tuple lookup,
  `previous_case_id` auto-chaining, timeout-close-and-reopen) stays exactly
  as implemented for `Continue` only — it is **not** invoked by Create.
- A "make Cases mandatory" mode, a per-customer feature flag to force Case
  creation, or a flow-validation warning for "no case_create action found".
  Reference: raised and explicitly deferred in the 2026-07 CPO discussion —
  pchero accepted the risk of silent CRM data gaps in exchange for full
  flexibility.
- A default/template Flow showcasing `case_create`. This is a
  square-admin (frontend) concern; out of scope for this backend ticket.
- Any change to `Continue`, `Close`, `CaseNoteCreate`/`Delete`/`List`,
  `CaseTagAdd`/`Remove`/`List`, or the existing peer-lock / deadlock-retry
  machinery in `casehandler/getorcreate.go` and `casehandler/peerlock.go`.
  Those stay untouched.

## 3. `ContactV1CaseCreate` semantics

### 3.1 Why plain Create, not get-or-create

Matches the platform's existing API convention (see e.g. `Contact`/`Address`
creation): a get-or-create action endpoint is avoided in favor of the client
(here: the Flow action / AI tool) explicitly deciding to create, with the DB
unique index as the final concurrency backstop. This also keeps Create's
contract simple and predictable for a Flow author: "this action creates a
Case row" — not "this action might create, might silently return an
existing unrelated Case depending on timing."

### 3.2 Signature

```go
// bin-contact-manager/pkg/casehandler
func (h *caseHandler) Create(
    ctx context.Context,
    customerID uuid.UUID,
    self commonaddress.Address,
    peerType commonaddress.Type,
    peerTarget, referenceType string,
    name, detail string, // both optional; empty string persisted as NULL
) (*kase.Case, error)
```

- `self`/`peerType`/`peerTarget`/`referenceType` are derived by the caller
  (Flow action / AI tool) from the call/conversation context — see §5.1/§5.2
  — never supplied directly by an end user or LLM free-text argument. This
  matches `GetOrCreate`'s existing shape and avoids letting an LLM tool-call
  argument set an arbitrary peer_target.
- No `caseIDHint`, no `previousCaseID` parameter.

### 3.3 Behavior

1. Insert a new `Case` row via the existing `dbhandler.CaseInsert` primitive
   (no transaction, no retry loop — this is deliberately simpler than
   `GetOrCreate`'s `insertWithRetry`).
2. If the insert violates `uq_case_open_peer` (an open Case already exists
   for this `(customer_id, peer_type, peer_target, reference_type)` tuple),
   `CaseInsert` already returns `dbhandler.ErrDuplicate` — confirmed present
   in current code (`pkg/dbhandler/kase.go:82-107`, `caseInsertExec`).
   `Create` translates this to a typed `cerrors.AlreadyExists(...)`
   (following the codebase convention of translating raw dbhandler
   sentinels to typed errors at the handler layer — see
   `contacthandler.ResolutionCreate` for the naming/translation
   convention *only*; that precedent does not itself demonstrate
   concurrency-race handling, see the deadlock point below). It does
   **not** retry, does **not** fall back to fetching the existing open
   Case, and does **not** auto-chain via `previous_case_id`.

   **`ErrDeadlock` handling (added per round-1 review finding):**
   `Create` deliberately has none of `GetOrCreate`'s concurrency apparatus
   — no `BeginTx`, no `acquirePeerLock`, no `maxDeadlockRetries` outer loop
   (see `getorcreate.go:70-116` for what `GetOrCreate` has that `Create`
   does not). Under real concurrent load (e.g. two near-simultaneous
   `case_create` tool calls, or a Flow `goto`/loop re-executing the action
   for the same peer), MySQL can resolve the race two ways:
   - Most commonly, a clean `uq_case_open_peer` violation → `ErrDuplicate`
     → `cerrors.AlreadyExists` (handled as above).
   - Under InnoDB gap-lock contention on the same not-yet-existing
     generated-column value, the loser can instead see `errno 1213`
     (`dbhandler.ErrDeadlock`, same sentinel `GetOrCreate` retries on).
     `Create` does **not** retry this. It translates `ErrDeadlock` to
     `cerrors.Unavailable(...)` (a transient-failure typed error, matching
     the existing `cerrors.Unavailable` constructor in
     `bin-common-handler/models/errors/constructors.go`) and returns it —
     never a raw/untyped error. Both the Flow action (§5.2) and the AI
     tool (§6.3) treat `AlreadyExists` and `Unavailable` identically
     (log-and-continue / `fillFailed`, never retried, never escalated to
     `GetOrCreate`'s heavier machinery). A single deadlock-loser Case
     creation attempt is simply lost — the caller (Flow author or AI) may
     re-issue `case_create` later if they want a Case for that peer, but
     `Create` itself never auto-retries.
   - **A Flow `goto`/loop re-executing `case_create` for the same peer**,
     or two rapid AI tool calls for the same peer, both resolve safely:
     the second call receives `AlreadyExists` (or, rarely, `Unavailable`
     under deadlock contention), logs it, and does not create a second
     Case. This is the intended, safe outcome and is now stated explicitly
     rather than left to inference.
3. `Case.Status` is always `StatusOpen` at creation (no other status is
   possible via Create).
4. `Case.PreviousCaseID` is always `nil` for a Create-produced Case. Chaining
   to a prior closed Case for the same peer remains exclusively a `Continue`
   behavior.
5. `Case.OwnerType`/`OwnerID` are left unset (zero value) at creation. No
   auto-assignment to the calling agent. Rationale: a Flow-triggered or
   AI-triggered Case creation has no natural "acting agent" in most cases
   (e.g. an unattended IVR flow, or an AI conversation with no live agent
   at all) — auto-assigning ownership would be a fabricated fact. Assignment
   remains a separate, existing action (not touched by this ticket).
6. If `note` is non-empty (only reachable via the Flow-action/AI-tool
   wrappers in §5, not a `Create` RPC parameter itself — see §3.2), the
   caller issues a **separate** `CaseNoteCreate` call immediately after a
   successful `Create`, using the existing standalone primitive
   (`pkg/dbhandler/casenote.go`). This is NOT wrapped in the same
   transaction as the Case insert: a Case-created-but-note-failed outcome is
   an acceptable, log-and-continue partial success (the Case itself is the
   load-bearing artifact; the note is a convenience). `author_type` is
   `"system"` for the Flow action, `"ai"` for the AI tool call site.

### 3.4 `Name`/`Detail` schema addition

`kase.Case` currently has no `Name`/`Detail` fields (confirmed absent in
`models/kase/kase.go`). Add:

```go
type Case struct {
    // ... existing fields unchanged ...
    Name   string `json:"name,omitempty"   db:"name"`
    Detail string `json:"detail,omitempty" db:"detail"`
}
```

Alembic migration on `contact_cases` (via `bin-dbscheme-manager`, generated —
never hand-written per repo convention). Column-order placement verified
against the actual `CREATE TABLE contact_cases`
(`f718e26f2c44_contact_cases_create_table.py:36-49`): real column order is
`id, customer_id, peer_type, peer_target, reference_type, contact_id,
owner_type, ...`, so `name after reference_type` places it correctly
immediately before `contact_id`, and `detail after name` chains correctly.
No prior migration alters `contact_cases`' columns (the only post-creation
migration touching a `contact_*` table is
`8d5a344905e7_contact_interactions_add_column_case_id.py`, which alters
`contact_interactions`, not `contact_cases`):

```sql
alter table contact_cases
  add column name   varchar(255) not null default '' after reference_type,
  add column detail text after name;
```

`name` follows the existing `varchar(255) not null default ''` convention
used elsewhere in this table — the real width precedent is `peer_type`/
`peer_target` (both `VARCHAR(255)`, `f718e26f2c44...py:40-41`), NOT
`closed_reason`/`closed_by_type` (those are `VARCHAR(32)` — corrected
citation, round-2 review caught this). `detail` uses bare `text` (no
explicit `null` keyword — corrected to match the two closest existing
precedents for adding a nullable text/varchar `name`+`detail` pair,
`af6321e8bdef_add_users_name_detail.py:21` and
`dbbf8225587a_routes_add_column_name_detail.py:21`, both of which use
`add column detail text after name;` verbatim, bare `text` relying on
MySQL's implicit nullable default rather than a redundant explicit `null`).
`detail`'s type matches `contact_case_notes.text`'s column type (`TEXT`) —
note this is a match on TYPE only, not nullability: `contact_case_notes.text`
is itself `NOT NULL` (`437ca5f2e4ee_contact_case_notes_create_table.py:37`),
whereas this new `Case.Detail` column is nullable by design (optional at
creation) — corrected citation, round-2 review caught the original draft
conflating "same type" with "same nullability."

Both fields are simple additive columns — no index, no uniqueness
constraint, no interaction with `open_peer_uk`.

## 4. RPC / route wiring (cross-layer)

Per the design-workflow's cross-layer parity checklist, tracing all layers
explicitly:

### Layer: `bin-contact-manager/pkg/casehandler`

New file `create.go`:
```go
func (h *caseHandler) Create(ctx context.Context, customerID uuid.UUID, self commonaddress.Address, peerType commonaddress.Type, peerTarget, referenceType, name, detail string) (*kase.Case, error)
```
Add to the `CaseHandler` interface in `pkg/casehandler/main.go`. Regenerate
`mock_main.go` (`go generate ./pkg/casehandler/...`).

### Layer: `bin-contact-manager/pkg/dbhandler`

No new dbhandler method needed — `Create` reuses the existing
`CaseInsert(ctx, c *kase.Case) error` primitive as-is (it already returns
`ErrDuplicate` on conflict).

### Layer: `bin-contact-manager/pkg/listenhandler`

New route:
```go
regV1Cases = regexp.MustCompile(`/v1/cases$`) // NEW — distinct from existing regV1CasesGet (`/v1/cases\?(.*)$`)
```
New handler `processV1CasesPost` in `v1_cases.go`, dispatched on
`case regV1Cases.MatchString(m.URI) && m.Method == sock.RequestMethodPost`.
Request body: `V1DataCasesPost{CustomerID uuid.UUID; Self commonaddress.Address; PeerType commonaddress.Type; PeerTarget, ReferenceType, Name, Detail string}`.
Note: registration order matters here — `regV1CasesGet` only matches URIs
with a literal `?` present; a bare `/v1/cases` POST does not collide with
it, but this must be verified with a route-dispatch test (existing pattern:
`v1_cases_test.go`).

### Layer: `bin-common-handler/pkg/requesthandler`

New file addition to `contact_cases.go`:
```go
func (r *requestHandler) ContactV1CaseCreate(ctx context.Context, customerID uuid.UUID, self commonaddress.Address, peerType commonaddress.Type, peerTarget, referenceType, name, detail string) (*cmkase.Case, error)
```
Add to the `RequestHandler` interface in `main.go` (alongside the existing
`ContactV1Case*` entries at line ~928). Regenerate `mock_main.go`.

### Layer: consumers (new, this ticket)

- `bin-flow-manager/pkg/activeflowhandler` — new `actionHandleCaseCreate`,
  registered in `pkg/activeflowhandler/execute.go`'s dispatch switch (the
  ACTUAL registration point — see §5.3 correction; `bin-flow-manager` has
  no `pkg/actionhandler` package, an earlier draft of this design
  incorrectly referenced one).
- `bin-ai-manager/pkg/aicallhandler` — new `toolHandleCaseCreate`,
  registered in whatever switch/map in `aicallhandler` currently dispatches
  `tool.Function.Name` → handler function (verify the exact dispatch site
  during implementation — same class of registration as the Flow action;
  do not skip it, an unregistered tool name causes the LLM's tool call to
  fail to resolve to any handler at runtime).

## 5. Flow action: `case_create`

### 5.1 Action type + option

`bin-flow-manager/models/action/action.go`:
```go
TypeCaseCreate Type = "case_create"
```

`bin-flow-manager/models/action/option.go`:
```go
// OptionCaseCreate defines action case_create's option.
type OptionCaseCreate struct {
    Name   string `json:"name,omitempty"`
    Detail string `json:"detail,omitempty"`
    Note   string `json:"note,omitempty"`
    Sync   bool   `json:"sync,omitempty"` // matches conversation_send/email_send's sync/async toggle
}
```
No `peer`/`reference_type`/`self` fields — always derived from the
activeflow's own reference context (§5.2), matching the "no explicit peer
override" decision from §2.

**Required registration points (added per round-1 review finding — a new
action `Type` touches THREE registries, not just `action.go`'s constant and
`option.go`'s struct):**

1. `action.TypeListAll` (`action.go`) — the enumeration list every existing
   action type is registered in.
2. `action.MapRequiredMediasByType` (`action.go`) — every existing action
   type has an entry; per this map's own role (media-type requirement
   lookup per action), `case_create` needs an entry with no required media
   (it is not a media-producing/consuming action, same category as
   `webhook_send`/`conversation_send`).
3. `action.OptionStructByType` (`option_registry.go`) — explicitly
   documented in that file as required for every new action type, checked
   by `TestActionCatalogFieldsMatchOptionStructs`. Omitting this entry
   fails that test, not just a runtime dispatch failure.
4. `bin-ai-manager/pkg/actioncatalog/main.go`'s `actionCatalog` slice — a
   **cross-repo** companion entry required whenever a new Flow action type
   is added (per `option_registry.go`'s own sync-note), checked by
   `TestActionCatalogMatchesTypeListAll` AND
   `TestActionCatalogFieldsMatchOptionStructs` (both in
   `bin-ai-manager/pkg/actioncatalog/main_test.go` — the latter also added
   per round-2 review; it enforces catalog-entry field-name parity against
   `OptionCaseCreate`'s json tags and would fail independently of the
   former test if the catalog entry's `Options` field names don't match).
   This is a `bin-ai-manager` change coupled to the `bin-flow-manager`
   change and **must land in the same PR** (not "same PR or immediately
   after" — tightened per round-2 review: these are two separate repos
   with independent CI, so any window where `bin-flow-manager`'s
   `action.TypeCaseCreate` merges to main before `bin-ai-manager`'s catalog
   entry follows is a real, if narrow, cross-repo CI-breakage risk given
   this is a hard test dependency) — see §11's revised implementation
   order.

### 5.2 Peer/reference_type derivation (mirrors `contacthandler.deriveEndpoints`)

`actionHandleCaseCreate` in `bin-flow-manager/pkg/activeflowhandler/actionhandle.go`,
registered in `pkg/activeflowhandler/execute.go`'s dispatch switch (the
ACTUAL registration point — confirmed via `TypeConversationSend`/
`TypeEmailSend`'s existing case arms in that file; `bin-flow-manager` has
no `pkg/actionhandler` package, corrected from an earlier draft of this
design that referenced one incorrectly):

```go
func (h *activeflowHandler) actionHandleCaseCreate(ctx context.Context, af *activeflow.Activeflow) error {
    // non-critical action: every error path below logs and returns nil,
    // never propagates up to abort the activeflow (see §7/§8).
    log := logrus.WithFields(logrus.Fields{"func": "actionHandleCaseCreate", "activeflow_id": af.ID})

    tmpOption, err := json.Marshal(af.CurrentAction.Option)
    if err != nil {
        log.Errorf("could not marshal the option. err: %v", err)
        return nil
    }
    var opt action.OptionCaseCreate
    if err := json.Unmarshal(tmpOption, &opt); err != nil {
        log.Errorf("could not unmarshal the case_create option. err: %v", err)
        return nil
    }

    var self, peer commonaddress.Address
    var referenceType string

    switch af.ReferenceType {
    case activeflow.ReferenceTypeCall:
        c, err := h.reqHandler.CallV1CallGet(ctx, af.ReferenceID) // same call site pattern as actionHandleEmailSend (actionhandle.go:210, :667)
        if err != nil {
            log.Errorf("could not get call. err: %v", err)
            return nil
        }
        peer, self = deriveEndpointsForCase(string(c.Direction), c.Source, c.Destination) // NEW helper in bin-flow-manager, mirrors contacthandler.deriveEndpoints's logic but is its own separate implementation -- contacthandler.deriveEndpoints is unexported in a different service and cannot be imported. Needs its own unit test (see §10).
        referenceType = "call"

    case activeflow.ReferenceTypeConversation:
        cv, err := h.reqHandler.ConversationV1ConversationGet(ctx, af.ReferenceID)
        if err != nil {
            log.Errorf("could not get conversation. err: %v", err)
            return nil
        }
        peer, self = cv.Peer, cv.Self // confirmed: conversation.Conversation has Self/Peer commonaddress.Address fields (conversation.go:29-30) -- verified against source, not assumed
        referenceType = "conversation_message"

    default:
        // Deliberate scope limit (documented in §2, confirmed per round-2
        // review): af.ReferenceType has 8 real, actively-used values in
        // this codebase (None, AI, API, Call, Campaign, Conversation,
        // Transcribe, Recording -- models/activeflow/activeflow.go:79-87),
        // not just Call/Conversation. A Case's peer/reference_type only
        // has a defined derivation for a live call or live conversation, so
        // every other reference type is a SILENT NO-OP here: log a warning,
        // create no Case, do not error the activeflow. A Flow author
        // wiring case_create into e.g. a campaign- or API-triggered
        // activeflow gets no Case and only this log line -- no
        // activeflow-level error signal. See §10 for the required explicit
        // test of this no-op branch (not just implicit exercise via the
        // Call/Conversation happy-path tests).
        log.Warnf("case_create is not supported for reference_type: %s", af.ReferenceType)
        return nil
    }

    // CRM-eligibility check: a legitimate SKIP, not a failure -- logged at
    // Debug level, never treated as an error. This is symmetric with
    // §6.3's tool-side handling after the round-1 fix (previously the two
    // surfaces disagreed on whether this is a skip or an error).
    if !isCRMEligiblePeer(peer.Type) { // duplicated from contacthandler.isCRMEligiblePeer -- see §9 Open Question #1 on whether/when to promote to bin-common-handler. Needed in BOTH bin-flow-manager (here) AND bin-ai-manager (§6.3) -- two duplications, not one; corrected after round-1 review flagged the original draft's undercount.
        log.Debugf("peer type is not CRM-eligible; skipping case_create. peer_type: %s", peer.Type)
        return nil
    }

    peerTarget, err := commonaddress.NormalizeTarget(peer.Type, peer.Target)
    if err != nil {
        log.WithError(err).Warnf("could not normalize peer target; using raw value. peer_type: %s", peer.Type)
        peerTarget = peer.Target
    }

    res, err := h.reqHandler.ContactV1CaseCreate(ctx, af.CustomerID, self, peer.Type, peerTarget, referenceType, opt.Name, opt.Detail)
    if err != nil {
        // Covers BOTH cerrors.AlreadyExists (an open case for this peer
        // already exists -- expected under the "goto/loop re-executes
        // case_create for the same peer" scenario, see §3.3) and
        // cerrors.Unavailable (rare ErrDeadlock race loser, see §3.3).
        // Neither is escalated, retried, or distinguished here -- both are
        // simply logged and the activeflow continues without a new Case.
        log.Errorf("could not create case. err: %v", err)
        return nil
    }
    if opt.Note != "" {
        if _, errNote := h.reqHandler.ContactV1CaseNoteCreate(ctx, af.CustomerID, res.ID, "system", nil, opt.Note); errNote != nil {
            log.Errorf("could not create initial case note. err: %v", errNote)
        }
    }
    return nil
}
```

**`isCRMEligiblePeer` reuse note**: this filter currently lives as an
unexported function in `bin-contact-manager/pkg/contacthandler/interaction.go`.
Since `case_create` needs the identical check from BOTH `bin-flow-manager`
(§5.2 above) AND `bin-ai-manager` (§6.3 below) — two separate services,
neither of which can import the contact-manager's unexported function —
this requires **two duplications** (corrected count; an earlier draft of
this design under-counted this as a single flow-manager-only duplication).
Given `bin-common-handler`'s admission rule (3+ consumers), and this
feature immediately creating exactly 3 total implementations (the
contact-manager original + 2 new duplicates), promoting the small
map+function to `bin-common-handler/models/address` now, at the same time
this ticket lands, is arguably the cleaner call rather than duplicating
twice and promoting later — flagged as a design decision to confirm during
review (§9 Open Questions).

### 5.3 Registration in the action dispatch switch

Per `bin-flow-manager/CLAUDE.md`'s "Action Dispatch Safety" rule: register
`action.TypeCaseCreate` in `pkg/activeflowhandler/execute.go`'s dispatch
switch (the actual, verified registration point — `bin-flow-manager` has no
`pkg/actionhandler` package; an earlier draft of this design incorrectly
referenced one, corrected per round-1 review) alongside the other
non-critical action types (`TypeConversationSend`, `TypeEmailSend`).
Missing this registration means an activeflow errors at runtime on
encountering the action — must be caught by a dispatch-switch test, not just
a unit test of `actionHandleCaseCreate` in isolation.

### 5.4 Metrics

Per `main.go`'s existing convention (`promActionErrorTotal` "does NOT count
non-critical action errors"): `case_create` is added to that documented
non-critical list. `promActionExecutedTotal{type="case_create"}` still
increments on every attempt (success or swallowed failure) — only the
FATAL-error counter excludes it.

## 6. AI tool: `case_create`

### 6.1 Tool name + registration

`bin-ai-manager/models/tool/main.go`:
```go
ToolNameCaseCreate ToolName = "case_create"
```
Add to `AllToolNames`. Add to `bin-ai-manager/pkg/toolhandler/whitelist.go`'s
`ConversationSafeTools` map (per 2026-07 decision: usable in AI
conversations, not just AI calls — this is the whitelist that gates
conversation-typed AIcalls; voice-only tools like `connect_call` are
excluded from it today, but `case_create` has no call-specific
prerequisite, so it belongs on the safe list).

### 6.2 Tool definition (`pkg/toolhandler/definitions.go`)

```go
{
    Name: tool.ToolNameCaseCreate,
    Description: `Creates a new CRM case for the current contact/interaction.

WHEN TO USE:
- The caller's issue is substantive and should be tracked as a case (e.g. a complaint, a multi-step request, something requiring follow-up).
- An agent or the AI itself judges this interaction needs a trackable record beyond the raw interaction log.

WHEN NOT TO USE:
- Casual/short interactions with no follow-up need.
- A case may already be open for this contact/channel — creating another will fail silently (existing open case is not returned; this call will simply not create a duplicate). Do not retry on failure.

Optional name/detail/note describe the case for a human agent reviewing it later.`,
    Parameters: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "run_llm": map[string]any{
                "type":        "boolean",
                "description": "Set true to have the assistant mention the case was created (e.g. tell the caller 'I've opened a case for this'). Set false to create silently.",
                "default":     true,
            },
            "name":   map[string]any{"type": "string", "description": "Short case title (optional)."},
            "detail": map[string]any{"type": "string", "description": "Longer free-text description of the issue (optional)."},
            "note":   map[string]any{"type": "string", "description": "An initial internal note for the agent (optional, not shown to the customer)."},
        },
    },
    RunLLM: true, // static default; matches send_email/send_message's convention. Per tool/main.go:65-67's documented convention, EVERY existing tool exposes an LLM-overridable `run_llm` property in Parameters (confirmed at definitions.go:101-108, :182-189, etc.) -- an earlier draft of this design omitted this property entirely, deviating from the established convention with no justification. Fixed here to match.
},
```
No `peer`/`reference_type`/`self` parameters exposed to the LLM — same
rationale as §5.1 (never let free-text tool arguments set the case's
identity key).

### 6.3 Handler (`pkg/aicallhandler/tool.go`)

Dispatched from whichever switch/map in `aicallhandler` currently routes
`tool.Function.Name` → handler function (the same class of registration
requirement as the Flow action's dispatch switch in §5.3 — verify the exact
site during implementation; an unregistered tool name means the LLM's
tool-call simply has nothing to resolve to at runtime, the AI-tool
equivalent of the Flow action's "activeflow errors at runtime" failure
mode).

```go
func (h *aicallHandler) toolHandleCaseCreate(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
    // mirrors toolHandleEmailSend's shape exactly
    res := newToolResult(tool.ID)
    var tmpOpt struct{ Name, Detail, Note string }
    if err := json.Unmarshal([]byte(tool.Function.Arguments), &tmpOpt); err != nil {
        fillFailed(res, err)
        return res
    }

    self, peer, referenceType, err := h.deriveCaseEndpointsForAIcall(ctx, c) // new small helper; switches on c.ReferenceType (Call vs Conversation), same pattern as §5.2. Needs its own unit test (see §10) -- this is new code, not a reuse of anything existing.
    if err != nil {
        fillFailed(res, err)
        return res
    }

    // CRM-eligibility check: a legitimate SKIP (not a genuine tool
    // failure). Corrected per round-1 review: an earlier draft of this
    // design reported this as fillFailed (a reported LLM-visible error),
    // which was INCONSISTENT with the Flow action's Debug-level silent
    // skip for the identical business condition (§5.2). Both surfaces now
    // agree: an ineligible peer is not something to create a case for, and
    // is reported to the LLM as a benign non-outcome, not an error -- using
    // fillSuccess with an explanatory message, not fillFailed, so the LLM
    // does not treat this as a retry-worthy failure.
    if !isCRMEligiblePeer(peer.Type) {
        fillSuccess(res, "case", "", fmt.Sprintf("No case was created: peer type %s is not eligible for CRM case tracking.", peer.Type))
        return res
    }

    peerTarget, err := commonaddress.NormalizeTarget(peer.Type, peer.Target)
    if err != nil {
        peerTarget = peer.Target // best-effort fallback, matches §5.2's Flow-action handling
    }

    created, err := h.reqHandler.ContactV1CaseCreate(ctx, c.CustomerID, self, peer.Type, peerTarget, referenceType, tmpOpt.Name, tmpOpt.Detail)
    if err != nil {
        // Covers cerrors.AlreadyExists (an open case for this peer already
        // exists -- see §3.3) and cerrors.Unavailable (rare ErrDeadlock
        // race loser -- see §3.3). Both are reported to the LLM via
        // fillFailed (per this tool's convention, matching every other
        // toolHandle* function) -- the tool description's own "Do not
        // retry on failure" text (§6.2) is the mechanism that keeps this
        // from causing a retry loop; the LLM is expected to move on.
        fillFailed(res, err)
        return res
    }
    if tmpOpt.Note != "" {
        _, _ = h.reqHandler.ContactV1CaseNoteCreate(ctx, c.CustomerID, created.ID, "ai", nil, tmpOpt.Note) // best-effort, error swallowed same as §3.3 point 6
    }

    fillSuccess(res, "case", created.ID.String(), "Case created successfully.")
    return res
}
```

**Failure handling difference from the Flow action**: `toolHandleCaseCreate`
returns a `fillFailed` result (visible to the LLM as a tool-call failure
signal) rather than silently swallowing the error — this is consistent with
every existing `toolHandle*` function (`toolHandleEmailSend`,
`toolHandleMessageSend`, etc. all call `fillFailed` on error). "Log and
continue" at the AI-call level means: the AIcall/conversation itself does
not terminate on this tool's failure (matches §2's decided policy) — but
the LLM DOES get told the tool call failed, exactly as it does for
`send_email` today. This is the correct reading of "로그만 남기고 진행" for a
tool (vs. a Flow action, which has no LLM to report failure to, so it truly
only logs).

## 7. Removal of automatic Case creation

### 7.1 Exact diff

`bin-contact-manager/pkg/contacthandler/interaction.go`:

- `EventCallCreated`: remove the `h.caseHandler.GetOrCreate(...)` call
  (lines ~100-103) and the `CaseID: &c.ID` field from the constructed
  `Interaction` (line ~115). The function still computes `peer`/`local`,
  still checks `isCRMEligiblePeer`, still normalizes targets, still calls
  `h.db.InteractionCreate` — only the Case side-effect and the resulting
  `CaseID` field are removed. `Interaction.CaseID` becomes permanently `nil`
  for every interaction projected from this path going forward (until/unless
  a Case happens to exist and something explicitly links them — out of
  scope here, see §9).
- `EventConversationMessageCreated`: identical removal (lines ~154-157,
  `CaseID: &c.ID` at ~170).
- `caseHandler` dependency: if `contactHandler` has no other remaining
  caller of `caseHandler.GetOrCreate`/`Close`/etc. through this exact
  struct field, do NOT remove the field/dependency wiring wholesale —
  `casehandler` is still a live, actively-used package (Close/Continue/List/
  Notes/Tags all still work via listenhandler directly, not through
  `contactHandler`). Confirm via `search_files` that `contactHandler` struct
  itself doesn't otherwise reference `caseHandler` before deciding whether to
  drop an unused field.

### 7.2 What does NOT change

- `casehandler.GetOrCreate` itself (the whole `getorcreate.go` file, its
  peer-lock/deadlock-retry machinery) stays exactly as-is. It remains used
  by `Continue`'s `insertWithRetry` reuse and remains fully tested. Nothing
  in this ticket deletes it — only its TWO webhook call sites are removed.
- `linkSiblingConversation`'s proactive-link write inside `GetOrCreate` is
  therefore also unaffected (it only ever ran as part of `GetOrCreate`,
  which Create does not call).

## 8. Failure-handling summary (cross-reference)

| Call site | On `ContactV1CaseCreate` error (AlreadyExists or Unavailable) | On CRM-ineligible peer | On `ContactV1CaseNoteCreate` error |
|---|---|---|---|
| Flow action (`case_create`) | `logrus.Errorf`, return `nil` (activeflow continues) | `logrus.Debugf` (benign skip, not an error) | `logrus.Errorf`, no further action |
| AI tool (`case_create`) | `fillFailed(res, err)` — LLM sees tool failure; AIcall continues | `fillSuccess(res, ...)` with an explanatory "no case created" message — NOT reported as a tool failure | error swallowed, tool call still reports success |

Both paths satisfy "log and continue, never abort" (§2) for genuine
errors — they differ only in whether an LLM-facing signal exists to report
the failure, which is inherent to the two surfaces (a Flow action has no
LLM to tell; a tool call does). The CRM-ineligible-peer condition is
explicitly NOT treated as a failure on either surface (corrected per
round-1 review — an earlier draft had the AI tool path incorrectly
reporting this benign skip as a tool failure, inconsistent with the Flow
action's silent-skip treatment of the identical condition).

## 9. Open questions for review

1. **`isCRMEligiblePeer` duplication/promotion** (§5.2/§5.3): this ticket's
   own work creates a 3rd implementation of this filter (contact-manager
   original + flow-manager + ai-manager duplicates) — confirm whether to
   promote it to `bin-common-handler/models/address` immediately (satisfies
   the 3+-consumer admission rule the moment this ticket lands) or duplicate
   twice now and promote in a later cleanup PR.
2. **`ConversationV1ConversationGet` peer/self field names**: CONFIRMED
   during round-1 review — `conversation.Conversation` has
   `Self commonaddress.Address` / `Peer commonaddress.Address` fields
   (`conversation.go:29-30`). No longer an open question; retained here only
   as a record that this was explicitly verified against source, not
   assumed.
3. **`Interaction.CaseID` going permanently nil**: is this acceptable
   long-term, or does a future ticket need a "link this interaction to that
   open case after the fact" mechanism now that Case creation is decoupled
   from interaction projection? Flagging per pchero's "인터렉션은 일단은 그냥
   두자" — explicitly parked, not silently dropped. Note (round-1 review
   finding): no such linking mechanism (`InteractionUpdateCaseID` or
   similar) exists anywhere in the current codebase today — this is a
   purely hypothetical future capability, not a live code path being
   deferred.
4. **`OptionCaseCreate.Sync`**: does a Flow author need `sync=true` (block
   subsequent actions until the Case is confirmed created) at all, or is
   `case_create` always fire-and-forget like `webhook_send`'s async mode?
   Leaning toward defaulting `sync=false` semantics identical to
   `conversation_send`'s pattern, exposing `sync` only for parity/consistency.
5. **Unowned Case + `/continue` authorization (new, round-1 review
   finding)**: §3.3 point 5 deliberately leaves `OwnerType`/`OwnerID` unset
   at Create time (no fabricated ownership). This has a real, confirmed
   downstream consequence: `Continue`'s authorization check
   (`lifecycle.go:137-142`, `isOwner := source.OwnerType == callerType &&
   source.OwnerID == callerID`) has no "unowned = anyone may continue"
   fallback — a zero-value Owner never equals a real caller's identity, so
   it always falls into the `!isOwner` branch. **Concrete consequence: a
   Create-produced Case that is later closed without ever being explicitly
   assigned to an agent can only be resumed via `/continue` by an admin
   (`callerIsAdmin=true`), never by a regular non-owning agent.** This is
   confirmed safe (no bug — `Close` itself doesn't consult Owner either,
   confirmed at `lifecycle.go:65-111`) but is a real product/UX implication
   that should be surfaced to whoever designs the square-admin Case UI: an
   agent-created-and-later-closed Case may need an explicit "assign to me"
   step before it becomes agent-resumable, or the UI needs to route
   unowned-closed-Case continuations through an admin/manager path.
6. **No activeflow-facing outcome signal for the Flow action (new, round-2
   review finding)**: `actionHandleCaseCreate` (§5.2) never sets an
   activeflow variable (e.g. `case_id`) reporting whether it created a
   Case, deduped against an existing one, or hit an error — a Flow author
   has no way to branch subsequent flow logic on the outcome. This matches
   the existing convention for sibling non-critical actions
   (`actionHandleEmailSend`/`actionHandleWebhookSend` likewise set no
   output variable), so it is NOT a regression or an inconsistency with
   established patterns — but it is worth flagging explicitly as a known
   v1 limitation rather than leaving it undiscussed. If a future Flow
   author needs conditional logic based on "did case_create actually
   create a case," that would be a separate, later enhancement (e.g. a
   `case_id`/`case_created` output variable), not in scope here.

## 10. Test plan (high level, per layer)

- `bin-contact-manager`: `pkg/casehandler/create_test.go` (happy path,
  `ErrDuplicate` → `AlreadyExists` translation, `ErrDeadlock` →
  `Unavailable` translation, name/detail persistence, note creation
  best-effort failure does not fail Create's own result, unset
  Owner/PreviousCaseID/Status=open invariants).
  `pkg/listenhandler/v1_cases_create_test.go` (route dispatch, request body
  parsing, 400 on missing customer_id).
- `bin-common-handler`: `pkg/requesthandler/contact_cases_test.go` addition
  for `ContactV1CaseCreate`.
- `bin-flow-manager`:
  - `pkg/activeflowhandler/actionhandle_test.go` addition for
    `actionHandleCaseCreate` — call-context derivation, conversation-context
    derivation, CRM-ineligible-peer skip (Debug-level, non-error),
    `ContactV1CaseCreate` AlreadyExists/Unavailable both swallowed
    (activeflow continues), `peerTarget` normalization applied.
  - A NAMED unit test for `deriveEndpointsForCase` (the new helper itself,
    not just exercised indirectly through `actionHandleCaseCreate`) —
    incoming/outgoing/unknown direction cases, mirroring
    `contacthandler.deriveEndpoints`'s own existing test coverage shape.
  - Dispatch-switch registration test (`execute.go`) confirming
    `TypeCaseCreate` resolves to `actionHandleCaseCreate` and does not error
    at runtime.
  - **New per round-2 review**: an explicit test asserting the `default:`
    branch's no-op behavior — for at least one non-Call/Conversation
    `activeflow.ReferenceType` (e.g. `ReferenceTypeCampaign` or
    `ReferenceTypeAPI`), confirm `actionHandleCaseCreate` logs a warning,
    calls neither `ContactV1CaseCreate` nor `ContactV1CaseNoteCreate`, and
    returns `nil` (activeflow does not error). This scope limit is
    documented in §2/§5.2 and must have explicit coverage, not just be
    implied by the Call/Conversation happy-path tests.
  - `action.OptionStructByType`/`MapRequiredMediasByType`/`TypeListAll`
    registration — covered by the EXISTING `Test_OptionStructByType_CoversTypeListAll`
    (local to `bin-flow-manager`) once the new type is registered; no new
    test needed beyond ensuring this existing test still passes.
- `bin-ai-manager`:
  - `pkg/aicallhandler/tool_test.go` addition for `toolHandleCaseCreate` —
    happy path, `fillFailed` on `ContactV1CaseCreate` error, `fillSuccess`
    (not `fillFailed`) on CRM-ineligible-peer skip, `ConversationSafeTools`
    whitelist inclusion test, tool-definitions schema validity test
    (existing pattern in `definitions_resource_test.go`), `run_llm`
    property present in the schema.
  - A NAMED unit test for `deriveCaseEndpointsForAIcall` (new helper).
  - Both `bin-ai-manager/pkg/actioncatalog/main.go`'s
    `TestActionCatalogMatchesTypeListAll` AND
    `TestActionCatalogFieldsMatchOptionStructs` (`main_test.go` — the
    latter added per round-2 review; verifies the new catalog entry's
    `Options` field names match `OptionCaseCreate`'s json tags exactly)
    must pass once the new `case_create` catalog entry is added (cross-repo
    coupling with the `bin-flow-manager` action-type addition, §5.1 point 4
    — must land in the SAME PR, not a follow-up).
- `bin-dbscheme-manager`: migration round-trip test (upgrade/downgrade,
  verify single Alembic head).

## 11. Implementation order

1. Alembic migration (`name`/`detail` columns) — independent, can run first.
2. `bin-contact-manager`: `kase.Case` field additions, `casehandler.Create`
   (including `ErrDuplicate`→`AlreadyExists` and `ErrDeadlock`→`Unavailable`
   translation), listenhandler route, tests.
3. `bin-common-handler`: `ContactV1CaseCreate` RPC client + interface entry
   + mock regen (depends on #2's Go types).
4. `bin-flow-manager` (action type + option + THREE registries per §5.1 +
   `deriveEndpointsForCase` + dispatch-switch registration + the
   `default:`-branch no-op test) and `bin-ai-manager` (tool name +
   definition with `run_llm` + handler + `deriveCaseEndpointsForAIcall` +
   `ConversationSafeTools` + dispatch registration) tracks can run in
   parallel once #3 lands (both only depend on the common-handler RPC
   client, not on each other) — **except** the
   `bin-ai-manager/pkg/actioncatalog` companion entry (§5.1 point 4).
   **Tightened per round-2 review**: this entry MUST land in the SAME PR
   as `bin-flow-manager`'s `action.TypeCaseCreate` addition, not a
   follow-up PR — the two repos have independent CI, and
   `TestActionCatalogMatchesTypeListAll` /
   `TestActionCatalogFieldsMatchOptionStructs` are hard test dependencies
   that would break `bin-ai-manager`'s CI in the window between two
   separate merges. In practice this means: land `bin-flow-manager`'s
   action-type addition and `bin-ai-manager`'s catalog-entry addition as
   a single combined PR (or two PRs merged back-to-back with the
   `bin-ai-manager` catalog PR gated on the `bin-flow-manager` PR's merge,
   never landing `bin-flow-manager`'s change alone first).
5. Removal PR (§7) — can be its own small PR, sequenced last, once the new
   creation paths are available (Cases can now be re-created explicitly).
