# square-talk Cases menu: list + detail + Close/Assign (v1 scope)

- Ticket: NOJIRA
- Author: Hermes (CPO), for pchero (CEO/CTO)
- Status: DESIGN APPROVED — closed after 8 review rounds (16 independent
  reviews total, 2 consecutive fully clean rounds). Ready for
  implementation.
- Supersedes: `2026-07-21-service-agent-case-endpoints-design.md` (PR #1125,
  CLOSED without merge). That design covered case-scoped message history
  and Assign/Unassign as a broad Conversation-replacement effort. pchero
  redirected to a much narrower v1: a Cases menu in square-talk showing a
  list + read-only detail, with only Close and Assign as mutating actions.
  Message history, Unassign, and Continue are explicitly out of scope
  here (see §5).

## 1. Scope (confirmed with pchero, verbatim)

1. Backend exposes **one new list endpoint only** — full list, no
   server-side owner-scoping. Mine / Others / Unassigned split happens
   **client-side** in square-talk against the fetched list.
2. Case **detail** view (read-only fields) reachable by clicking a list
   row.
3. Actions: **Close** and **Assign** only. (Unassign, Continue, message
   send, notes, tags: not in this PR.)
4. **No authorization restriction** among agents: every authenticated
   agent of the customer can list/view/close/assign ANY case, regardless
   of current owner. No self-only-assign rule, no owner-or-admin gate on
   Close. This is a deliberate, explicit v1 simplification — see §6 for
   the tradeoff this accepts.

## 2. Current backend state (confirmed against real code)

`bin-api-manager/pkg/servicehandler/case.go`'s `CaseList`/`CaseGet`/
`CaseClose` all gate on `amagent.PermissionCustomerAdmin|
amagent.PermissionCustomerManager` (lines 78, 142, 172 respectively).
There is no
`/service_agents/contact_cases/*` surface at all — zero files matching
that name under `bin-api-manager/server` or
`bin-openapi-manager/openapi/paths/service_agents/`. square-talk is
Agent-facing and must call only `/service_agents/*` (per
`bin-openapi-manager/CLAUDE.md`'s "never widen the Admin/Manager
endpoint" rule) — so this requires new endpoints, not a permission
relaxation on the existing ones.

`bin-contact-manager`'s `casehandler.Close` (lifecycle.go) already
implements the idempotent guarded-close semantics this PR needs
unchanged — `bin-api-manager`'s new endpoint is a thin wrapper, nothing
new needed in `bin-contact-manager` or `bin-common-handler` for Close.

**Case.Owner is currently a fully dead-write field** (re-confirmed from
the prior PR #1125's review: `Create`/`insertWithRetry` never set it,
`Continue` only reads it, no `CaseAssign` RPC exists anywhere). This PR
is the first to actually make it writable.

## 3. Backend changes

### 3.1 New service-agent endpoints (bin-api-manager only — no
    bin-contact-manager/bin-common-handler dbhandler changes needed for
    List/Get/Close; Assign needs new plumbing, see §3.2)

Following the established `service_agents/<resource>` checklist
(`bin-openapi-manager/CLAUDE.md`'s "Adding a Service Agent resource"
section, worked example: `serviceagent_transcribe.go`):

| Method | Path | servicehandler function | Notes |
|---|---|---|---|
| GET | `/service_agents/contact_cases` | `ServiceAgentCaseList` | `func (h *serviceHandler) ServiceAgentCaseList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cmkase.Case, string, error)`. Full customer-scoped list, no owner filter param at all (client filters). Reuses `h.reqHandler.ContactV1CaseList(ctx, a.CustomerID, "", "", uuid.Nil, uuid.Nil, size, token)` — status/owner_type/owner_id/contact_id filters all left empty/nil, matching §1.1's "full list" decision. |
| GET | `/service_agents/contact_cases/{id}` | `ServiceAgentCaseGet` | `func (h *serviceHandler) ServiceAgentCaseGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmkase.Case, error)`. Thin wrapper around the existing private `h.caseGet` helper (case.go:30), no ownership gate. |
| POST | `/service_agents/contact_cases/{id}/close` | `ServiceAgentCaseClose` | `func (h *serviceHandler) ServiceAgentCaseClose(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmkase.Case, error)`. No request body. Thin wrapper around `h.reqHandler.ContactV1CaseClose`, `closed_by_id` derived server-side from `a.AgentID()` (same pattern as the existing admin/manager `CaseClose`, case.go:176) — even though §1.4 permits any agent to close ANY case, the actor who performed the close is still recorded truthfully. |
| POST | `/service_agents/contact_cases/{id}/assign` | `ServiceAgentCaseAssign` | `func (h *serviceHandler) ServiceAgentCaseAssign(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, ownerID uuid.UUID) (*cmkase.Case, error)`. HTTP request body: `{owner_id: string(uuid)}` only — `owner_type` is NOT a client-supplied HTTP field (see §3.2 for how the internal RPC wire contract still carries a hardcoded `"agent"` value one layer down, which is a separate, internal-only detail from this HTTP-facing signature). Case ownership in this codebase is agent-only today (no other owner type exists anywhere in the platform's Case/Conversation code) — this removes the otherwise-unanswered "what other owner_type values are valid" question by not exposing the field to clients at all. New. See §3.2 for the `owner_id`-validity question this raises. |

All four gated `amagent.PermissionAll` only (the "any authenticated agent
of this customer" sentinel, per §1.4 — explicitly NOT
`PermissionCustomerAdmin|PermissionCustomerManager`, and explicitly no
owner-or-admin split for Close either, unlike the existing admin/manager
`CaseClose`/`CaseContinue` which don't need such a split since they're
already admin-gated).

New files
  `bin-api-manager/pkg/servicehandler/serviceagent_case.go` (mirrors
  `serviceagent_conversation.go`'s structure). New file
  `bin-api-manager/server/service_agents_contact_cases.go` (mirrors
  `server/service_agents_conversations.go`'s structure — note the naming
  follows `contact_cases` per the existing top-level route's own naming,
  not bare `cases`, to stay consistent with `server/contact_cases.go`).

### 3.2 Case Assign — new plumbing required (bin-contact-manager +
    bin-common-handler)

Confirmed (§2): no write path to `Case.Owner` exists anywhere today, at
any layer. This PR adds the FIRST one, scoped to exactly what's needed
for Assign (no Unassign, no admin/manager tier — §1.4 removes the need
for that split entirely, simplifying what PR #1125's design required):

1. **`bin-contact-manager/pkg/dbhandler/kase.go`**: new
   `CaseUpdateOwner(ctx, customerID, id uuid.UUID, ownerType
   commonidentity.OwnerType, ownerID uuid.UUID) error` — `UPDATE
   contact_cases SET owner_type=?, owner_id=? WHERE id=? AND
   customer_id=?`, mirroring `caseUpdateContactIDExec`'s exact shape
   (kase.go, customer_id-scoped defense-in-depth). Added to the
   `DBHandler` interface in `main.go`.
2. **`bin-contact-manager/pkg/casehandler/assign.go`** (new file): new
   `Assign(ctx, customerID, id uuid.UUID, ownerType
   commonidentity.OwnerType, ownerID uuid.UUID) (*kase.Case, error)` —
   tenant-check via `CaseGetByID` (mirroring `Continue`'s pattern in
   lifecycle.go: `c.CustomerID != customerID` → `dbhandler.ErrNotFound`),
   then `CaseUpdateOwner`, then re-fetch and return. Added to the
   `CaseHandler` interface in `main.go`. **No authorization decision
   inside this function** — per §1.4 there is none to make; any caller
   who reaches this function (already authenticated as an agent of the
   tenant, per §3.1's `PermissionAll` gate) may assign to any
   `(ownerType, ownerID)` including a different agent than themselves.
   **`owner_id` existence validation (load-bearing, decided here, not
   left TBD):** `bin-api-manager`'s `ServiceAgentCaseAssign` (§3.1)
   validates `ownerID` refers to a real, active agent of the caller's
   own customer BEFORE calling `ContactV1CaseAssign` — via
   `h.reqHandler.AgentV1AgentGet(ctx, ownerID)` (the same primitive
   `ServiceAgentConversationUpdate`'s admin/manager reassignment path
   implicitly trusts the caller to have picked from a real agent list
   for; here there is no admin trust boundary, so an explicit existence
   + `agent.CustomerID == a.CustomerID` check is required) and rejecting
   with `serviceerrors.ErrNotFound`/an equivalent 404 if the agent
   doesn't exist or belongs to a different tenant. **Both failure modes
   ("agent doesn't exist" and "agent belongs to a different tenant")
   are deliberately collapsed into the same `ErrNotFound`/404 response
   — a standard anti-enumeration practice (matching `caseGet`'s own
   documented rationale for cross-tenant case ids, case.go's comment on
   `caseGet`), not an oversight.** `casehandler.Assign`
   itself does NOT re-validate this (it trusts the caller, matching
   every other tenant-scoped handler function in this package that
   accepts an already-validated id) — the check belongs at the API
   boundary where the client-supplied `owner_id` first enters the
   system, not duplicated at every internal layer.

   **Case-not-found path (separately load-bearing, previously
   unaddressed):** if the `case_id` in the URL path itself doesn't
   exist (no row at all — distinct from the owner_id-not-found case
   above), `ServiceAgentCaseGet`/`ServiceAgentCaseClose`/
   `ServiceAgentCaseAssign` all return `serviceerrors.ErrNotFound`/404
   via the SAME `h.caseGet` pre-check helper already used by the
   existing admin/manager `CaseClose`/`CaseContinue` (case.go:166,
   :201) — no new error path needed, this PR's new functions call the
   same helper before doing anything else, exactly like their
   admin/manager siblings already do.
3. **`bin-contact-manager/pkg/listenhandler`**: new route
   `POST /v1/cases/{id}/assign` → `processV1CasesIDAssignPost`, new
   request struct `V1DataCasesIDAssign{CustomerID uuid.UUID, OwnerType
   string, OwnerID uuid.UUID}` in
   `pkg/listenhandler/models/request/v1_cases.go`, mirroring
   `processV1CasesIDClosePost`'s exact structure.
4. **`bin-common-handler/pkg/requesthandler`**: new
   `ContactV1CaseAssign(ctx context.Context, customerID, id, ownerID
   uuid.UUID) (*cmkase.Case, error)` in `contact_cases.go` — `ownerID`
   typed as `uuid.UUID`, matching sibling `ContactV1CaseClose`/
   `ContactV1CaseContinue`'s existing ID-parameter convention
   (`closedByID`, `callerID` are both `uuid.UUID`, not `string`) and
   matching this same section's own `V1DataCasesIDAssign.OwnerID
   uuid.UUID` struct field — no string→UUID parse step anywhere in this
   call path. **`ownerType` is not an exported function parameter (no
   caller ever needs to supply it), but `ContactV1CaseAssign`'s
   internal implementation still populates
   `V1DataCasesIDAssign.OwnerType` on the wire with the hardcoded
   literal `string(commonidentity.OwnerTypeAgent)`** — this field
   exists on the request struct precisely because
   `bin-contact-manager`'s `casehandler.Assign`/`CaseUpdateOwner` are
   generic over `commonidentity.OwnerType` (§3.2 point 1/2) and require
   a real, non-empty value to write to the `owner_type` column; simply
   omitting the field from the wire payload (leaving it as Go's
   zero-value empty string) would silently write an empty
   `owner_type`, not `"agent"` — a real correctness bug, not a harmless
   simplification. In short: the field is absent from every
   CLIENT-facing surface (HTTP request body, JS function signature,
   OpenAPI spec — §3.1, §3.3, §4.1) but still present and populated on
   the internal RPC wire contract between `bin-api-manager` and
   `bin-contact-manager`, where it always carries the same
   `"agent"` literal. Added to the `RequestHandler` interface in
   `main.go`.

**Explicitly NOT built in this PR** (deferred, not silently dropped —
see §5): `CaseUnassign`/`CaseClearOwner`, `V1DataCasesIDUnassign` route,
`ContactV1CaseUnassign` RPC client. If Unassign becomes needed later,
add it as a symmetric follow-up; nothing in this PR's shape blocks that.

### 3.3 OpenAPI

New files under `bin-openapi-manager/openapi/paths/service_agents/`:
- `contact_cases.yaml` — GET (list), reusing the existing
  `ContactManagerCase` schema (already defined,
  `bin-openapi-manager/openapi/openapi.yaml`'s `ContactManagerCase`) in
  a `CommonPagination`-wrapped array response, mirroring
  `conversations.yaml`'s exact shape (see file read above).
- `contact_cases_id.yaml` — GET (get single case), same schema.
- `contact_cases_id_close.yaml` — POST, no request body, returns
  `ContactManagerCase`.
- `contact_cases_id_assign.yaml` — POST, request body `{owner_id:
  string(uuid)}` only — `owner_type` is NOT a request field (per §3.1/
  §3.2's decision to hardcode `commonidentity.OwnerTypeAgent`
  server-side), returns `ContactManagerCase`.

Register all four under `paths:` in `openapi.yaml`, alongside the
existing `/service_agents/conversations*` block (§ line ~8696-8703),
using path names `/service_agents/contact_cases`,
`/service_agents/contact_cases/{id}`, `/service_agents/contact_cases/
{id}/close`, `/service_agents/contact_cases/{id}/assign` — matching the
top-level route's own `contact_cases` naming (not bare `cases`), so the
resource name stays consistent between the admin/manager and
service-agent surfaces.

## 4. Frontend (square-talk)

### 4.1 API service layer

New file `square-talk/src/api/services/cases.js`, mirroring
`conversations.js`'s exact conventions (axios instance import, JSDoc
comments, `/v1.0/service_agents/...` path prefix):

```js
export async function getCasesAPI(pageSize = 100) {
  const response = await axiosInstance.get('/v1.0/service_agents/contact_cases', {
    params: { page_size: pageSize },
  });
  return response.data?.result || response.data || [];
}

export async function getCaseAPI(caseId) {
  const response = await axiosInstance.get(`/v1.0/service_agents/contact_cases/${caseId}`);
  return response.data;
}

export async function closeCaseAPI(caseId) {
  const response = await axiosInstance.post(`/v1.0/service_agents/contact_cases/${caseId}/close`);
  return response.data;
}

export async function assignCaseAPI(caseId, ownerId) {
  const response = await axiosInstance.post(
    `/v1.0/service_agents/contact_cases/${caseId}/assign`,
    { owner_id: ownerId }
  );
  return response.data;
}
```

No pagination beyond `page_size` cap is handled in v1 — §1.1's "full
list" decision means square-talk fetches one page at `page_size=100`
(matching `conversations.js`'s existing default) and does not implement
infinite-scroll/load-more; if a customer has more than 100 cases this is
an accepted v1 gap, not silently unhandled — flagged explicitly here so
it's a known, disclosed limitation rather than an oversight.

### 4.2 Redux slice

New `square-talk/src/store/slices/casesSlice.js`, mirroring
`conversationsSlice.js`'s manual-thunk shape (`FetchCases()` thunk,
`casesFetched`/`isLoading`/`cases` state, no `createAsyncThunk`). Store
raw list from `getCasesAPI()`; owner-based filtering (Mine/Others/
Unassigned) is NOT stored in Redux — it's a pure derived/client-side
computation in the list component (§4.4), keeping the slice itself
simple and the filter reactive to the currently-logged-in agent without
needing to re-fetch.

**Two required wiring steps, easy to miss and both load-bearing —
without them `state.cases` is `undefined` and the page 404s/crashes at
runtime, not just "incomplete":**

1. **Register the new reducer in `square-talk/src/store/rootReducer.js`**
   — the real file is a `combineReducers({ app, auth, me, conversations,
   chatrooms, agents, extensions, calls, contacts, talk, tags })` call;
   add `cases: casesReducer` alongside the existing entries, AND add
   the slice to whatever logout-reset switch/case this file also
   contains (mirror however `conversations`/`contacts` are already
   handled there).
2. **Add `dispatch(FetchCases())` to `DashboardLayout.jsx`'s existing
   post-login `useEffect`** (the same block that already dispatches
   `FetchTalks()`/`FetchTags()`/`FetchAgents()`/`FetchContacts()`/
   `FetchConversations()`) — without this the Cases list is simply
   never populated until/unless `CasesPage` itself triggers a fetch on
   mount (which is also an acceptable alternative if the implementer
   prefers page-level fetching over app-level pre-fetching, but ONE of
   the two must actually be wired; do not assume adding the slice alone
   makes data appear).

### 4.3 Navigation

`DashboardLayout.jsx`: add one `navItems` entry, `{ path: '/cases', icon:
FolderOpen (lucide-react, matching square-admin's own Cases icon choice
in `contact_cases_list.js`), label: 'Cases' }`. Route registration in
`routes/index.jsx` mirrors the existing `contacts`/`calls` entries
(lazy-loaded `CasesPage`).

### 4.4 List view — `features/cases/CasesPage.jsx`

- Three filter tabs/segments: **Mine** / **Others** / **Unassigned**,
  computed client-side from the fetched list + the logged-in agent's own
  id (`me.id` from `state.me`). **Rules are evaluated in this exact
  order, first match wins, so the three buckets are mutually
  exclusive by construction** (a naive independent-boolean formulation
  double-buckets an `owner_type: 'agent', owner_id: NIL_UUID` row into
  BOTH "Others" and "Unassigned" — this ordered-priority formulation is
  the fix, not an alternative phrasing):
  1. **Unassigned** (checked first): `!owner_type || !owner_id ||
     owner_id === NIL_UUID` (reuse square-talk's existing `NIL_UUID`
     constant from `utils/conversationPermissions.js`, same convention
     `AssignAgentDropdown.jsx` already uses for Conversation's identical
     unassigned-detection).
  2. **Mine** (checked second, only reached if not Unassigned):
     `owner_id === me.id` (owner_type is always `'agent'` by this point
     since Unassigned already excluded the empty/nil case, and no other
     owner_type value exists in this codebase per §3.1's note — no need
     to also check `owner_type === 'agent'` explicitly, though doing so
     defensively is harmless).
  3. **Others** (default, everything else): any case that reached this
     branch has a real, non-nil `owner_id` that isn't `me.id`.
  - Default active tab: **Mine**. **This is a fresh product decision for
    Cases, NOT inherited from an existing precedent** — checked against
    the actual runtime default: `conversationsSlice.js`'s
    `readOwnerFilter()` defaults Conversation's own equivalent filter to
    **'all'**, not 'mine'. Cases intentionally default differently
    (Mine) because case work is explicitly ownership-driven per this
    PR's own framing (an agent's daily queue is "what's assigned to
    me"), not because of a matching UI convention elsewhere in the
    codebase — no further "verify at implementation time" hedge needed,
    this is decided here.
- Each row: peer_target, channel (reference_type badge), **status
  badge** (open = default/primary badge color, closed = secondary/muted
  — mirroring square-admin's `contact_cases_list.js` status cell exactly,
  `Badge variant={status === 'open' ? 'default' : 'secondary'}`), opened_at
  (compact timestamp).
- Row click → navigate to detail (`/cases/:id`).
- Loading/empty/error states per this codebase's existing list-view
  conventions (`ConversationList.jsx`/`ContactList.jsx` as precedent).

### 4.5 Detail view — `features/cases/CaseDetail.jsx`

Read-only field display (per §1.2 — no notes/tags/message panels, no
interaction history; those are explicitly out of scope, §5). Displays
every `kase.Case` field relevant to an agent's read of case state:
- name / detail (the case's freeform title/detail text fields, present
  on `kase.Case` and already exposed in the `ContactManagerCase`
  OpenAPI schema — shown here even though this v1 has no edit path for
  them; read-only display of an existing field is not the same as
  building an edit feature)
- peer_type / peer_target
- reference_type (channel)
- status badge (same styling as list)
- contact_id — shown as a plain id/label in v1 (no contact-detail
  navigation link; linking to a Contact record is out of scope, same
  reasoning as §5's `previous_case_id` exclusion — square-talk has no
  Contact-detail route in this PR)
- owner (owner_type/owner_id — resolve to agent display name via the
  already-fetched `agents` slice if `owner_type === 'agent'`, same
  resolution helper `AssignAgentDropdown`/`OwnershipChip` already use
  for Conversation's owner display)
- opened_at / closed_at / closed_reason / closed_by (when closed)
- previous_case_id — rendered as plain text in v1, NOT a clickable link
  (square-admin's design treated this as a full navigability feature;
  v1 here has no case-to-case navigation route built, so linking it
  would 404 — explicitly deferred, added as its own bullet in §5).
- **Close button**: visible only when `status === 'open'`,
  `ConfirmDialog` before calling `closeCaseAPI`, then refetch/update the
  case in place (or navigate back to the list — implementation detail,
  not load-bearing).
- **Assign control**: **does NOT reuse `AssignAgentDropdown` unmodified**
  — the real component (`AssignAgentDropdown.jsx` lines 176-187)
  unconditionally renders a built-in "Unassign" menu item whenever
  `isAssigned` is true, dispatching `onSelect(NIL_UUID)`. Reusing it
  as-is would silently reintroduce Unassign, contradicting §1.3/§5's
  explicit exclusion. Required: either (a) fork a
  `CaseAssignDropdown` variant that omits the Unassign
  `DropdownMenuItem` block entirely, or (b) add a new prop to
  `AssignAgentDropdown` (e.g. `allowUnassign = true` defaulting true for
  Conversation's existing call sites, passed `false` for Case) gating
  that block — implementer's choice between (a)/(b), but the doc is
  explicit that ONE of them must be done; shipping the component
  unmodified is not an option. Per §1.4, `canSelfAssign` should be
  `true` unconditionally (no admin-only gate) — every agent can assign
  to anyone including themselves.

## 5. Explicitly out of scope for this PR (disclosed, not silently
   dropped)

- **Case-scoped message history / scrollback.** PR #1125's entire Gap A
  (the `case_conversations` join table, composite pagination cursor,
  backfill) — not needed since the detail view has no message panel in
  v1.
- **Unassign.** Only Assign is built (§3.2). If a case needs to become
  unowned again, no UI path exists in this PR.
- **Continue** (reopening a closed case). Detail view shows closed cases
  read-only with no action.
- **Case-linked message send**, notes, tags — all remain
  Admin/Manager-only (square-admin), not ported to square-talk here.
- **Server-side owner filtering / pagination beyond page_size=100** — §1.1
  and §4.1's explicit "full list, client-filtered" decision; a customer
  with a very large case volume is an accepted v1 gap.
- **`previous_case_id` navigability** (clickable link to the prior
  case). Rendered as plain text only (§4.5) — no case-to-case navigation
  route exists in this PR.
- **Any authorization/ownership gating on Close or Assign** — §1.4's
  explicit "no restriction" decision. Documented here as a deliberate
  choice with a known tradeoff (§6), not an oversight.

## 6. Accepted tradeoff: no authorization gating (§1.4)

Any authenticated agent can close or reassign ANY case, including one
actively owned/worked by a different agent, with zero confirmation of
"are you sure, this isn't yours." This mirrors a permissive, small-team-
trust model rather than the ownership-gated model
`ServiceAgentConversationUnassign`/`CaseClose`(admin/manager) already
established elsewhere in this codebase. Explicitly pchero's call for
this v1 — if this becomes a real operational problem (accidental
reassignment/closure of a case another agent is actively working), the
fix is additive (add an ownership check later, same shape as
Conversation's existing owner-or-admin pattern) and does not require
reworking anything built in this PR.

## 7. Test plan

- `bin-contact-manager`: dbhandler test for `CaseUpdateOwner` (success —
  asserting the persisted `owner_type` column equals
  `commonidentity.OwnerTypeAgent`/`"agent"`, not merely that `owner_id`
  changed, plus cross-tenant customer_id mismatch affects 0 rows,
  permanent regression test per this codebase's tenant-isolation
  convention); casehandler test for `Assign` (success — same
  `owner_type == "agent"` assertion via a re-fetched `kase.Case`, cross-
  tenant rejection).
- `bin-common-handler`: requesthandler test for `ContactV1CaseAssign`
  asserting the constructed `V1DataCasesIDAssign` RPC request body has
  `OwnerType` set to the hardcoded `commonidentity.OwnerTypeAgent`
  literal (mirroring `contact_cases_test.go`'s existing
  `Test_ContactV1CaseClose`-style assertion on constructed request
  bodies) — this is the exact function §3.2's wire-hardcoding fix lives
  in, and is the regression this multi-round review loop specifically
  fixed; a generic `go test ./...` pass is not sufficient evidence this
  stays correct.
- `bin-api-manager`: servicehandler tests for all four new
  `ServiceAgentCase*` functions, each asserting a PLAIN agent identity
  (not just admin/manager) is accepted — the whole point of this being a
  service-agent surface. **`ServiceAgentCaseAssign` specifically also
  needs**: (a) a test asserting 404/`ErrNotFound` when `owner_id`
  references a nonexistent agent, (b) a test asserting 404 when
  `owner_id` references a real agent belonging to a DIFFERENT customer
  (the cross-tenant case) — these two are the actual safety-critical
  behavior §3.2's `AgentV1AgentGet`-style check exists to enforce, not
  optional extras.
- `square-talk`: component tests for `CasesPage` (three filter tabs
  render correct subsets given a fixed cases+me fixture that explicitly
  INCLUDES an `owner_type:'agent', owner_id: NIL_UUID` row — the edge
  case that motivated §4.4's ordered-priority rewrite, proving it lands
  in Unassigned and nowhere else — plus status badge styling,
  empty/loading states) and `CaseDetail` (field rendering, Close button
  visibility gated on status, Assign control wiring, **and a test
  asserting the rendered Case assign dropdown contains NO "Unassign"
  menu item/option** — the exact regression §4.5's fork-or-gate decision
  exists to prevent; a passing "Assign control wiring" test that never
  checks for Unassign's absence would not catch a regression back to
  unmodified `AssignAgentDropdown` reuse).
- Full verification workflow (`go mod tidy && go mod vendor && go
  generate ./... && go test ./... && golangci-lint run -v --timeout 5m`)
  in `bin-contact-manager`, `bin-api-manager`, `bin-openapi-manager`,
  `bin-common-handler`.
- One real browser click-through against the live sandbox API before
  declaring done (per `voipbin-frontend-visual-verification-gate` skill
  — layout/visual verification is a separate, mandatory gate from code
  review for any square-talk UI work).

## 8. Review history (chronological, Round 1 → Round 6)

**Round 6 (2 parallel independent reviews: test-plan re-verification,
adversarial fresh pass) — test-plan APPROVED, adversarial pass REQUEST
CHANGES.**

Test-plan re-verification confirmed Round 5's `owner_type=='agent'`
assertion requirements and the new `ContactV1CaseAssign` requesthandler
test bullet are both genuinely present and unambiguous, and confirmed
§8's review history is a single, non-duplicated, chronologically
ordered section with no leftover fragment/truncation artifacts — no
issues.

Adversarial pass found two genuine, previously-undetected gaps: (a) §2
still cited "lines 78, 108, 142, 172" for `CaseList`/`CaseGet`/
`CaseClose`'s permission gates even though §8's own Round 1 entry
already documented that line 108 belongs to the unrelated
`CaseListUnresolved` function — the history recorded the fix but the
live prose was never actually corrected; fixed by removing the stale
line-108 citation from §2's sentence. (b) §4.5's CaseDetail field list
omitted `kase.Case`'s real `Name`/`Detail`/`ContactID` fields with no
disclosure that they were intentionally left out (unlike TagIDs/notes,
which §5 explicitly excludes) — corrected by adding all three to the
display list with explicit rationale (name/detail: read-only display of
an existing field, not an edit feature; contact_id: shown as plain
text, no Contact-detail route exists in this PR, same reasoning as
previous_case_id's exclusion).

This is the SAME "history documents a fix that was never applied to
the live prose" class of defect as the owner_type wire-drift Round 4
found — reinforcing that a review round's own history entry describing
a fix is not proof the fix was actually applied to the document body;
each claimed fix needs to be independently re-verified against the live
text, not just cross-referenced against the round that claimed to make
it.

**Round 1 (2 parallel independent reviews: technical accuracy,
implementability) — both REQUEST CHANGES.**

Technical review found: (a) a minor line-citation imprecision (line 108
belongs to `CaseListUnresolved`, not one of the three named functions —
the underlying permission-bitmask claim was still accurate); (b) a real
scope-leak risk — reusing `AssignAgentDropdown` unmodified would
silently reintroduce Unassign via its built-in menu item, contradicting
§1.3/§5's explicit exclusion. All other technical claims (permission
gating, dead-write Owner field, frontend precedent shapes, OpenAPI
pagination pattern) confirmed accurate against real code.

Implementability review found: (a) §3.1's endpoint table lacked full Go
signatures for the four new servicehandler functions — corrected, now
inline; (b) `owner_id` existence/tenant validation for Assign was
entirely unaddressed — corrected in §3.2 with an explicit
`AgentV1AgentGet` + cross-tenant check at the API boundary, and by
narrowing the request body to `owner_id` only (no client-supplied
`owner_type`, removing that ambiguity entirely); (c) §4.4's three filter
buckets were not mutually exclusive as literally specified (an
`owner_id === NIL_UUID` row satisfied both "Others" and "Unassigned")
— corrected to an explicit ordered-priority (first-match-wins)
formulation; (d) §4.5's `AssignAgentDropdown` reuse claim needed to say
plainly the component requires forking/gating, not just "verify at
implementation time" — corrected with two concrete implementation
options; (e) §5's `previous_case_id` cross-reference pointed at nothing
— corrected with its own explicit bullet.

**Round 2 (2 parallel independent fresh reviews: technical
re-verification, whole-document fresh read) — technical APPROVED,
whole-document REQUEST CHANGES.**

Technical review independently re-confirmed all of Round 1's five
fixes are genuinely present and match real code (agent-only OwnerType
verified against `commonidentity.Owner`'s actual two-constant
definition, `ContactV1CaseClose`'s real signature confirmed as the
mirrored precedent). No new technical defects found.

Whole-document review found: (a) §3.2's proposed
`ContactV1CaseAssign(..., ownerID string)` signature was internally
inconsistent with sibling `ContactV1CaseClose`/`ContactV1CaseContinue`
(both type their ID params `uuid.UUID`) AND with the doc's own
`V1DataCasesIDAssign.OwnerID uuid.UUID` struct — corrected to
`uuid.UUID` throughout, `ownerType` param dropped entirely (already
implicit from §3.1's hardcoded-agent decision); (b) §4.4's "Default tab:
Mine, consistent with existing UX convention" cited a precedent
(`conversationsSlice.js`'s `readOwnerFilter()`) that actually defaults to
'all', not 'mine' — corrected to state this is a fresh, deliberate
product decision for Cases, not an inherited convention, removing the
false citation; (c)/(d) §7's test plan omitted the two most
safety-critical new checks from Round 1's fixes (owner_id existence/
cross-tenant rejection for Assign, and Unassign-menu-item-absence for
the Case dropdown) — both added as explicit required test bullets,
plus the filter-tab test fixture now explicitly required to include the
`owner_id === NIL_UUID` edge case.

**Round 3 (2 parallel independent reviews: backend-engineer
re-verification, whole-document adversarial pass) — backend APPROVED,
whole-document REQUEST CHANGES.**

Backend re-verification confirmed all three of Round 2's fixes
(`ContactV1CaseAssign` signature, §4.4's corrected default-tab framing,
§7's two new test bullets) are genuinely present, correct, and match
real code — no new issues.

Whole-document adversarial pass reconfirmed the `PermissionAll` gate
convention and OpenAPI naming pattern against real code (both accurate,
no issues), but found a genuine, previously-unflagged frontend wiring
gap: §4.2 never mentioned registering the new `casesSlice` in
`rootReducer.js` (real file, `combineReducers({app, auth, me,
conversations, chatrooms, agents, extensions, calls, contacts, talk,
tags})` — every slice must be added there or `state.cases` is
`undefined`), and never called out adding `dispatch(FetchCases())` to
`DashboardLayout.jsx`'s existing post-login fetch `useEffect` (parallel
to the five existing `Fetch*` dispatches) — both are necessary,
non-optional wiring that would cause a real runtime failure if skipped.
Corrected: §4.2 now states both steps explicitly as required (with an
acceptable page-level-fetch alternative to the second one).

**Round 4 (2 parallel independent reviews: implementer re-verification,
adversarial fresh pass) — implementer APPROVED, adversarial pass
REQUEST CHANGES.**

Implementer re-verification confirmed Round 3's frontend wiring fix
(rootReducer.js registration, DashboardLayout fetch-effect) is
concrete, accurate against real code, and unambiguous — a full fresh
top-to-bottom read found no other issues.

Adversarial pass found a genuine, previously-undetected inconsistency:
§3.1/§3.2's load-bearing decision that `owner_type` is NOT
client-supplied (hardcoded server-side to `OwnerTypeAgent`) had NOT
been propagated to §3.3's OpenAPI request-body spec (still listed
`owner_type` as a field) or §4.1's `assignCaseAPI` JS function (still
threaded an `ownerType` parameter and sent it in the POST body) — both
corrected to `owner_id`-only. **A further, more subtle version of the
same drift was caught while making this correction**: dropping
`ownerType` from `ContactV1CaseAssign`'s exported Go signature (Round
2's fix) does NOT mean the field disappears from the wire entirely —
`V1DataCasesIDAssign.OwnerType` (the listenhandler request struct) and
`casehandler.Assign`'s own `ownerType commonidentity.OwnerType`
parameter are both still real, required inputs one layer down; §3.2 now
explicitly states `ContactV1CaseAssign`'s internal implementation still
populates that wire field with the hardcoded `"agent"` literal, so the
"not client-supplied" property holds at the HTTP boundary without
silently writing an empty `owner_type` column value at the DB layer.
Also found and corrected: the case-not-found failure path for the
three new servicehandler functions was entirely unaddressed (now
resolved by explicitly reusing the existing `h.caseGet` pre-check
helper, matching the admin/manager siblings' own pattern), and the
owner-id-not-found vs. cross-tenant-owner conflation into one 404 is
now stated as a deliberate anti-enumeration choice rather than left
ambiguous.

This is the SAME class of defect Round 2 fixed on the Go
`ContactV1CaseAssign` signature (a signature/contract change made in
one place but not propagated to sibling references) — recurring across
review rounds on different fields, underscoring that every
signature/contract change during this review loop needs a
whole-document grep for the old shape before the round is considered
closed, not just a fix at the first place it's noticed.

**Round 5 (2 parallel independent reviews: end-to-end wire-trace
verification, whole-document skeptical re-read) — wire-trace APPROVED,
whole-document REQUEST CHANGES.**

Wire-trace review independently traced the full owner_type data flow
from HTTP request through `ContactV1CaseAssign` → `V1DataCasesIDAssign`
→ `casehandler.Assign` → `CaseUpdateOwner`'s SQL UPDATE, confirming the
hardcoded `"agent"` literal genuinely reaches the DB write with no gap
where it could resolve to an empty string, and re-grepped every
remaining `owner_type`/`OwnerType` mention in the document to confirm
each is either correctly wire-only-hardcoded or correctly
untouched-generic. No new defects found.

Whole-document review found one real remaining gap: §7's test plan,
despite listing generic "success" assertions for `CaseUpdateOwner`/
`Assign`, never explicitly required asserting the persisted
`owner_type` value actually equals `"agent"` (as opposed to merely
`owner_id` changing), and had no test bullet at all for
`ContactV1CaseAssign` itself — the exact function this whole review
sub-thread (Rounds 4-5) was about. Corrected: both dbhandler/casehandler
success assertions now explicitly require checking `owner_type ==
"agent"`, and a new requesthandler test bullet requires asserting
`ContactV1CaseAssign` constructs its RPC request with the hardcoded
`OwnerType` literal. (This review also flagged §8/9's review-history
section ordering as chronologically scrambled — cosmetic only, doesn't
affect implementability from §§1-7; not otherwise corrected since
renumbering mid-loop risks introducing new inconsistencies for no
implementability benefit — §8 is now the single chronological history
section, Round 1 through Round 5 in order, resolving this on its own.)
