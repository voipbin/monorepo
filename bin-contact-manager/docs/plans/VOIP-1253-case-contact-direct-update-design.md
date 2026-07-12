# VOIP-1253 -- Case-Contact Direct Update (Design v0.1)

Repo: voipbin/monorepo
Ticket: VOIP-1253
Author: Hermes (CPO) on behalf of pchero (CEO/CTO)
Date: 2026-07-13
Status: Draft, awaiting independent review

## Changelog

- v0.1 (2026-07-13). Initial draft. Reverts VOIP-1252's Resolution-based
  case-level attribution mechanism (merged 2026-07-12, commit `40ce3eafc`)
  in favor of a direct `Case.contact_id` write.
- v0.2 (2026-07-13). Round-1 review findings addressed:
  - **BLOCKER fix**: `ReconcileContact` (and its `case-control
    reconcile-contact` CLI command) is now explicitly scoped for
    deletion, not "preserved standalone" -- it hard-depends on
    `deriveCaseContactIDTx`/`applyDerivedContactID` and its entire
    purpose (recomputing `Case.contact_id` from Resolution-row drift)
    is moot once Resolution is no longer the source of truth (§4, §6).
  - **BLOCKER fix**: §5.2's `h.db.CaseClearContactID` (non-Tx) does NOT
    already exist -- only `CaseClearContactIDTx` does. A new non-Tx
    wrapper must be added to `dbhandler` (§5.1.1, new).
  - **MAJOR fix**: added `bin-api-manager/pkg/servicehandler/case_resolution_test.go`
    to the §4 removal table (was missing, would fail to compile).
  - **MINOR fixes**: clarified the Conference PUT precedent is an
    API-shape convention only (Conference persists a literal zero-UUID,
    no SQL NULL; Case's clear path uses true NULL via the pre-existing
    `CaseClearContactIDTx`) -- not identical DB persistence semantics;
    added `case_tag.go`'s `verifyCaseOwnership` comment to the §7
    doc-comment cleanup list; added explicit reasoning in §5.2 for why
    dropping the transaction wrapper is safe.
- v0.3 (2026-07-13). Round-2 review finding addressed:
  - **MAJOR fix**: §4's removal table omitted
    `bin-api-manager/pkg/servicehandler/main.go` from the list of files
    needing `CaseResolutionCreate`/`CaseResolutionDelete` removed from
    the `ServiceHandler` interface declaration -- same defect class
    round 1 caught for `case_resolution_test.go` (leaving a dangling
    interface declaration after deleting its implementation breaks
    compilation). Added an explicit removal-table row for `main.go`
    (interface) and its mock regeneration.
  - Round 2 independently re-verified all 6 of round-1's fixes against
    actual main-branch source (not just doc text) and confirmed every
    one is genuinely correct, including that the new
    `CaseClearContactID` wrapper would actually compile.

## 1. Why this exists (session history)

VOIP-1252 shipped `POST/DELETE /v1/cases/{id}/resolutions`, reusing the
interaction-level Resolution mechanism (contact_resolutions table,
append-only, tm_delete-based retraction) for Case-level Contact
attribution. During square-admin UI wiring for this API, a real gap was
found: detaching a Contact from a Case requires the caller to supply a
`resolution_id`, but `GET /v1/cases/{id}` never exposes it, and no
`GET /v1/cases/{id}/resolutions` endpoint exists to look it up. A UI
"detach" button was therefore impossible to build without adding a new
read endpoint.

Walking through *why* the Resolution mechanism existed at all (see
VOIP-1204's original CRM design doc, 2026-06-26) surfaced that its
justification -- automatic peer-address matching cannot cover a
borrowed-phone / late-identified-anonymous-session / wrong-auto-match
case, so a manual judgment needs to be recorded as an immutable fact --
is a real requirement for **interaction-level** attribution, but does
not transfer cleanly to Case-level attribution:

- A Case is a single per-channel session header, not a stream of
  auto-matched events. There is no "automatic match" for
  `Case.contact_id` to override -- it starts NULL and an agent either
  sets it or doesn't.
- The audit-trail need ("who attributed this Case to this Contact and
  when") is real, but it does not require a queryable, retractable,
  append-only table with its own derivation function
  (`deriveCaseContactID`) and its own transaction discipline. It only
  requires that the state change be *recorded somewhere queryable* --
  which the platform's existing event-publishing infrastructure already
  provides for every other Case field (see `casenote.go`'s
  `case_note_created`/`case_note_deleted` precedent).

Cross-checking the OpenAPI surface (`bin-openapi-manager/openapi/paths/`)
confirmed the platform's actual majority convention for
single/few-field record updates is a generic `PUT /{resource}/{id}` with
an explicit field list (Contact, Agent, Queue, Conference, Flow,
Customer, Tag -- 7+ resources). `Call` is the sole outlier with no PUT,
but for an unrelated reason: Call fields (hold/mute/hangup) map to live
media-session commands, not record field values -- not applicable here.
`bin-conference-manager`'s `PreFlowID`/`PostFlowID` establish the exact
precedent needed: a nullable-FK-shaped field updated via `PUT`, where an
empty UUID in the request body clears the link and a non-empty UUID sets
it (see `pkg/conferencehandler/conference.go:109-110`,
`models/conference/webhook.go:25-26`, OpenAPI `conferences/id.yaml`
`put:` with `pre_flow_id`/`post_flow_id` as plain string fields).
**Precision note (round-1 correction):** Conference establishes the
API-shape convention only (empty value in the PUT body -> field
cleared), not identical DB persistence semantics -- `PreFlowID`/
`PostFlowID` are non-pointer required fields, so Conference persists a
literal zero-UUID (`00000000-...`) into the column, not SQL `NULL`.
Case's clear path uses true `NULL` via the pre-existing (Tx-scoped)
`CaseClearContactIDTx`/(new, non-Tx) `CaseClearContactID` -- this
design follows Conference's API shape, not its storage mechanics.

**Decision (pchero, 2026-07-13):** revert VOIP-1252's Resolution-based
case-level attribution. Replace with `PUT /v1/cases/{id}
{ contact_id: "<uuid or empty>" }`, following the Conference
PreFlowID/PostFlowID convention exactly. Audit trail is preserved via
`notifyHandler.PublishEvent` (no new infrastructure --
`bin-timeline-manager` already subscribes to
`bin-manager.contact-manager.event` and ingests it into ClickHouse with
zero additional wiring required).

**Interaction-level Resolution is UNCHANGED and out of scope.** The
`POST/DELETE /v1/interactions/{id}/resolutions` mechanism
(`contacthandler.ResolutionCreate`/`ResolutionDelete`, keyed on
`interaction_id`) remains exactly as-is. Its justification (correcting
automatic peer-match errors across a continuous event stream) is real
and does not apply to this revert -- only the Case-level branch
(`resolution.CaseID`-keyed rows, `casehandler.ResolutionCreateCaseLevel`/
`ResolutionDeleteCaseLevel`) is removed.

## 2. Goal

- An agent (or admin/manager) can attach an open, unresolved Case to a
  specific, existing Contact via a single `PUT` call, and detach it by
  sending an empty `contact_id` in the same call -- no separate DELETE,
  no resolution_id to track.
- `Case.contact_id` in the `GET /v1/cases/{id}` response is the single,
  immediately-consistent source of truth for "who is this Case
  attributed to right now" -- exactly as it already is today (this
  property does not change).
- Every attribution/de-attribution is recorded as a queryable audit
  event (`case_contact_attributed`/`case_contact_detached`) via the
  existing event-publishing pipeline, picked up automatically by
  bin-timeline-manager.
- Cross-tenant Case ids and cross-tenant Contact ids are both rejected,
  preserving VOIP-1252's two security fixes (case ownership check,
  contact ownership check) -- these are NOT being reverted, only the
  Resolution-table mechanism they were layered on top of.

## 3. Out of scope

- square-admin UI changes (follow-up, built on top of this API once
  merged).
- Any change to interaction-level Resolution
  (`POST/DELETE /v1/interactions/{id}/resolutions`).
- Any change to `deriveCaseContactID`'s *interaction-level* callers --
  there are none; `deriveCaseContactID` and
  `ResolutionListByCase`/`ResolutionListByCaseTx` were added
  exclusively for the case-level path being removed here, so they are
  deleted, not modified for reuse.
- Bulk/batch attribution tooling.
- Case.owner_type/owner_id assignment (no setter exists today; separate
  future ticket if needed, unrelated to this one).

## 4. What gets removed (VOIP-1252 revert scope)

| File | Change |
|---|---|
| `bin-contact-manager/pkg/casehandler/contact_attribution.go` | Delete entirely: `deriveCaseContactID`, `deriveCaseContactIDTx`, `firstCaseLevelPositiveContactID`, `applyDerivedContactID`, `ResolutionCreateCaseLevel`, `ResolutionDeleteCaseLevel`, **`ReconcileContact`** (round-1 correction: `ReconcileContact` hard-depends on `deriveCaseContactIDTx`/`applyDerivedContactID` and its entire purpose -- recomputing `Case.contact_id` from Resolution-row drift -- is moot once Resolution is no longer the source of truth; it is NOT preserved). `CaseListUnresolved`/`CaseListAll` (also in this file) do NOT depend on Resolution and move to a new file (see §6). |
| `bin-contact-manager/pkg/casehandler/contact_attribution_test.go`, `contact_attribution_write_test.go`, `reconcile_test.go` | Delete entirely (test the removed functions). |
| `bin-contact-manager/pkg/casehandler/main.go` | Remove `ResolutionCreateCaseLevel`/`ResolutionDeleteCaseLevel`/`ReconcileContact` from the `CaseHandler` interface; add `UpdateContact` (see §5). |
| `bin-contact-manager/cmd/case-control/main.go` | Remove the `reconcile-contact` subcommand (`cmdReconcileContact`, `runReconcileContact`) entirely -- its sole purpose (recovering from Resolution-row/Case.contact_id drift) no longer applies once Case.contact_id is the single directly-written source of truth with no derivation step to drift from. |
| `bin-contact-manager/pkg/listenhandler/v1_case_resolutions.go`, `v1_case_resolutions_test.go`, `models/request/v1_case_resolutions.go` | Delete entirely. |
| `bin-contact-manager/pkg/listenhandler/main.go` | Remove `regV1CasesIDResolutions`/`regV1CasesIDResolutionsID` routes; add `PUT /v1/cases/{id}` route. |
| `bin-contact-manager/pkg/dbhandler/resolution.go` | Remove `ResolutionListByCase`/`ResolutionListByCaseTx` (case-level only; `ResolutionListByInteraction` and all interaction-level resolution CRUD stay). |
| `bin-contact-manager/pkg/dbhandler/kase.go` | No removals. **Addition required** (round-1 correction, see §5.1.1): a non-Tx `CaseClearContactID` wrapper does not currently exist -- only `CaseClearContactIDTx` does. Must be added, mirroring the existing `CaseUpdateContactID`/`CaseUpdateContactIDTx` non-Tx/Tx pairing. |
| `bin-common-handler/pkg/requesthandler/contact_case_resolutions.go`, `contact_case_resolutions_test.go` | Delete entirely. |
| `bin-common-handler/pkg/requesthandler/main.go` | Remove `ContactV1CaseResolutionCreate`/`ContactV1CaseResolutionDelete` from the interface; add `ContactV1CaseUpdateContact`. |
| `bin-openapi-manager/openapi/paths/contact_cases/id_resolutions.yaml`, `id_resolutions_id.yaml` | Delete entirely. |
| `bin-openapi-manager/openapi/openapi.yaml` | Remove the two path registrations; remove `case_id` field from `ContactManagerResolution` (added by VOIP-1252, no longer has a producer); add `put:` block to `contact_cases/id.yaml` (new file content, not a new path entry -- path already exists for `get:`). |
| `bin-api-manager/pkg/servicehandler/case.go` | Remove `CaseResolutionCreate`/`CaseResolutionDelete`; add `CaseUpdateContact`. |
| `bin-api-manager/pkg/servicehandler/main.go` | Remove `CaseResolutionCreate`/`CaseResolutionDelete` from the `ServiceHandler` interface declaration (round-2 correction: originally omitted from this table -- leaving these declarations in place after deleting their `case.go` implementations breaks interface satisfaction and fails to compile, the same defect class round 1 caught for `case_resolution_test.go`); add `CaseUpdateContact`. Regenerate `mock_main.go` (confirmed both old methods are also declared there and must be removed by the regen). |
| `bin-api-manager/pkg/servicehandler/case_resolution_test.go` | Delete entirely (round-1 correction: was missing from the original removal list; tests `CaseResolutionCreate`/`Delete`, which no longer exist post-revert, so this file would fail to compile if left in place). |
| `bin-api-manager/server/contact_case_resolutions.go` | Delete entirely; add `PutContactCasesId` handler in a new/existing `server/contact_cases.go`. |

`resolution.Resolution.CaseID` (the model field itself, in
`models/resolution/resolution.go`) is left in place even though no
future writer sets it non-nil. Removing it would require a
migration decision (drop column vs. leave dead) that is unrelated to
this ticket's goal; flagged as an open question (§8) rather than
silently deciding either way. **The `contact_resolutions` DB table
itself is untouched** -- interaction-level rows continue to use it.

## 5. What gets added

### 5.1 New resource shape

```
PUT /v1/cases/{id}
{ "contact_id": "<uuid, or empty string to clear>" }
```

Reuses the existing GET path (`contact_cases/{id}.yaml` already exists
for `get:`) -- this adds a `put:` block to that same file, exactly as
`conferences/id.yaml` and `contacts/id.yaml` do (`get:`/`put:`/`delete:`
coexisting in one path file).

### 5.1.1 New dbhandler primitive (round-1 correction)

`dbhandler.CaseUpdateContactID` (non-Tx) already exists and is reused
as-is. `dbhandler.CaseClearContactIDTx` (Tx-scoped) also already
exists, but there is currently no non-Tx `CaseClearContactID` -- every
existing caller of the clear path ran inside VOIP-1252's transaction.
Since `UpdateContact` (below) is a single-statement, non-transactional
write (see §5.2's reasoning), a new non-Tx wrapper is needed, mirroring
`CaseUpdateContactID`'s existing non-Tx/Tx pairing exactly:

```go
// CaseClearContactID reverts a Case's contact_id to NULL, scoped
// outside any caller-managed transaction (VOIP-1253's direct-write
// path has no multi-statement derivation step requiring atomicity, so
// no transaction wrapper is needed here -- see design §5.2).
func (h *handler) CaseClearContactID(ctx context.Context, customerID, id uuid.UUID) error {
	return caseClearContactIDExec(h.db, customerID, id)
}
```

`caseClearContactIDExec` is the existing shared implementation
`CaseClearContactIDTx` already delegates to -- this is a thin
non-Tx wrapper around it, the same relationship `CaseUpdateContactID`/
`CaseUpdateContactIDTx` already have via `caseUpdateContactIDExec`. Add
to the `DBHandler` interface in `dbhandler/main.go`, regenerate mock.

### 5.2 bin-contact-manager: casehandler

New file `bin-contact-manager/pkg/casehandler/contact_update.go`
(replaces `contact_attribution.go`'s Resolution-based version):

```go
package casehandler

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/gofrs/uuid"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// UpdateContact implements design VOIP-1253: attaches or detaches a
// Case's Contact via a direct Case.contact_id write, replacing
// VOIP-1252's Resolution-based mechanism. contactID == uuid.Nil clears
// the attribution (mirrors bin-conference-manager's PreFlowID/
// PostFlowID PUT convention: empty UUID in the request clears the
// link). Verifies the Case belongs to customerID (mirrors
// verifyCaseOwnership, preserved from VOIP-1252) and, when attaching
// (contactID != uuid.Nil), verifies the target Contact belongs to
// customerID too (preserved from VOIP-1252 round-1 review finding --
// without this check an agent of one tenant could attach their Case to
// another tenant's Contact).
func (h *caseHandler) UpdateContact(ctx context.Context, customerID, caseID, contactID uuid.UUID) (*kase.Case, error) {
	if err := verifyCaseOwnership(ctx, h.db, customerID, caseID); err != nil {
		return nil, err
	}

	eventType := "case_contact_detached"
	if contactID != uuid.Nil {
		ct, err := h.db.ContactGet(ctx, contactID)
		if err != nil {
			if stderrors.Is(err, dbhandler.ErrNotFound) {
				return nil, cerrors.NotFound(
					commonoutline.ServiceNameContactManager,
					"CONTACT_NOT_FOUND",
					"The contact was not found.",
				).Wrap(err)
			}
			return nil, fmt.Errorf("could not get contact. UpdateContact. err: %v", err)
		}
		if ct.CustomerID != customerID {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameContactManager,
				"CONTACT_NOT_FOUND",
				"The contact was not found.",
			)
		}

		if err := h.db.CaseUpdateContactID(ctx, customerID, caseID, contactID); err != nil {
			return nil, fmt.Errorf("could not update case contact_id. UpdateContact. err: %v", err)
		}
		eventType = "case_contact_attributed"
	} else {
		if err := h.db.CaseClearContactID(ctx, customerID, caseID); err != nil {
			return nil, fmt.Errorf("could not clear case contact_id. UpdateContact. err: %v", err)
		}
	}

	c, err := h.db.CaseGetByID(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("could not get updated case. UpdateContact. err: %v", err)
	}

	// Audit trail (replaces VOIP-1252's Resolution row): who/when
	// changed this Case's Contact attribution, picked up automatically
	// by bin-timeline-manager (already subscribes to
	// bin-manager.contact-manager.event, zero new wiring needed).
	// Mirrors casenote.go's PublishEvent-only precedent -- this is an
	// internal state-change event, not a customer-facing webhook, so
	// PublishEvent (never PublishWebhookEvent) is correct here too.
	h.notifyHandler.PublishEvent(ctx, eventType, map[string]uuid.UUID{
		"case_id":    caseID,
		"contact_id": contactID, // uuid.Nil on detach -- consumer reads eventType to disambiguate
	})

	return c, nil
}
```

Note (round-1 correction): `dbhandler.CaseUpdateContactID` already
exists (added by VOIP-1252, unchanged by this revert) and is pure
reuse. `dbhandler.CaseClearContactID` (non-Tx) does NOT already exist
-- see §5.1.1 for the new wrapper this design adds around the existing
`CaseClearContactIDTx`. Both `CaseUpdateContactID`'s and
`CaseClearContactIDTx`'s doc-comments ("design §3.4; single source of
truth is Resolution") need updating to drop the now-false Resolution
reference (§7).

**Why no transaction wrapper is needed here** (unlike VOIP-1252's
`ResolutionCreateCaseLevel`, which wrapped its Resolution insert +
`deriveCaseContactIDTx` read + `Case.contact_id` write in a single
`BeginTx`/`Commit`): that transaction existed because VOIP-1252 had two
separate writes (Resolution row insert, then a derived read-then-write
of `Case.contact_id`) that needed to be atomic with each other, or a
crash between them would leave a Resolution row with no corresponding
`Case.contact_id` update. `UpdateContact` has exactly ONE write
statement (`CaseUpdateContactID` or `CaseClearContactID`) -- there is
no second write to keep atomic with it, so no multi-statement
derivation step exists that a crash could leave half-done. The
ownership checks (§5.2's `verifyCaseOwnership` and Contact-tenant
check) happen before the single write, the same TOCTOU-tolerant pattern
`ResolutionCreateCaseLevel` already used (a check-then-write race here
is bounded by the same customer_id being re-validated inside the
WHERE clause of `CaseUpdateContactID`/`CaseClearContactID`'s underlying
SQL, per `caseUpdateContactIDExec`'s existing `WHERE id=? AND
customer_id=?` shape) -- this is not a new risk introduced by dropping
the transaction, it is the same check-then-write shape every other
single-field Case write in this file already uses (e.g.
`CaseUpdateStatusClosed`).

Add to `CaseHandler` interface in `main.go`:
```go
UpdateContact(ctx context.Context, customerID, caseID, contactID uuid.UUID) (*kase.Case, error)
```

### 5.3 bin-contact-manager: listenhandler

New file `bin-contact-manager/pkg/listenhandler/v1_case_contact.go`:

```go
// processV1CasesIDPut handles PUT /v1/cases/{id} (VOIP-1253): attaches
// or detaches a Case's Contact via a direct contact_id write.
func (h *listenHandler) processV1CasesIDPut(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDPut
	if err := json.Unmarshal(req.Data, &body); err != nil {
		return simpleResponse(400), nil
	}
	if body.CustomerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	res, err := h.caseHandler.UpdateContact(ctx, body.CustomerID, id, body.ContactID)
	if err != nil {
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}
	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}
```

Request model (new file `models/request/v1_case_contact.go`):
```go
// V1DataCasesIDPut is the request body for PUT /v1/cases/{id}.
type V1DataCasesIDPut struct {
	CustomerID uuid.UUID `json:"customer_id"`
	ContactID  uuid.UUID `json:"contact_id"` // uuid.Nil clears the attribution
}
```

Route registration in `main.go`: reuse the existing `regV1CasesID`
pattern (already matches `/v1/cases/{uuid}$`, currently only wired for
GET) -- add a `case regV1CasesID.MatchString(m.URI) && m.Method ==
sock.RequestMethodPut:` branch alongside the existing GET branch.

### 5.4 bin-common-handler: requesthandler

New file `bin-common-handler/pkg/requesthandler/contact_case_update.go`,
following `ContactV1CaseClose`'s exact shape:

```go
// ContactV1CaseUpdateContact attaches or detaches a case's contact via
// a direct contact_id write (VOIP-1253). contactID == uuid.Nil clears
// the attribution.
func (r *requestHandler) ContactV1CaseUpdateContact(ctx context.Context, customerID, caseID, contactID uuid.UUID) (*cmkase.Case, error) {
	uri := fmt.Sprintf("/v1/cases/%s", caseID)

	data := &cmrequest.V1DataCasesIDPut{CustomerID: customerID, ContactID: contactID}
	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPut, "contact/cases/<id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmkase.Case
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}
	return &res, nil
}
```

Add to `RequestHandler` interface, regenerate mock.

### 5.5 bin-api-manager: servicehandler + auth

New method in `bin-api-manager/pkg/servicehandler/case.go`, mirroring
`CaseClose`'s permission pattern:

```go
// CaseUpdateContact attaches or detaches a case's contact (design
// VOIP-1253, replaces VOIP-1252's CaseResolutionCreate/Delete).
// contactID == uuid.Nil clears the attribution.
func (h *serviceHandler) CaseUpdateContact(ctx context.Context, a *auth.AuthIdentity, id, contactID uuid.UUID) (*cmkase.Case, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	c, err := h.caseGet(ctx, a.CustomerID, id)
	if err != nil {
		return nil, err
	}
	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	return h.reqHandler.ContactV1CaseUpdateContact(ctx, a.CustomerID, id, contactID)
}
```

Add to `ServiceHandler` interface + regenerate mock. Add
`PutContactCasesId` to `server/` (new file `server/contact_cases.go` if
one doesn't already exist for Case, else appended to the existing
`case.go`/`cases.go` server file).

### 5.6 OpenAPI spec

Modify (not create) `bin-openapi-manager/openapi/paths/contact_cases/id.yaml`
to add a `put:` block alongside the existing `get:`, following
`conferences/id.yaml`'s `put:` shape exactly:

```yaml
put:
  summary: Attach or detach a case's contact
  description: |
    Attaches the case to a specific existing Contact, or detaches it,
    via a direct contact_id write (VOIP-1253). Send a non-empty
    contact_id to attach; send an empty string to detach (mirrors
    bin-conference-manager's pre_flow_id/post_flow_id PUT convention:
    an empty value in the request body clears the link). The target
    contact_id must belong to the same customer as the case; a
    cross-tenant contact_id is rejected as not found. Every
    attach/detach is recorded as a case_contact_attributed/
    case_contact_detached event, queryable via bin-timeline-manager's
    audit log (no separate resolution history endpoint needed).
  tags:
    - Case
  parameters:
    - name: id
      in: path
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
              description: The contact to attach. Empty string detaches.
              example: "660e8400-e29b-41d4-a716-446655440001"
  responses:
    '200':
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ContactManagerCase'
    '400':
      $ref: '#/components/responses/BadRequest'
    '401':
      $ref: '#/components/responses/Unauthenticated'
    '403':
      $ref: '#/components/responses/PermissionDenied'
    '404':
      $ref: '#/components/responses/NotFound'
    '500':
      $ref: '#/components/responses/InternalError'
```

Remove `/contact_cases/{id}/resolutions` and
`/contact_cases/{id}/resolutions/{resolution_id}` path registrations
and their two files. Remove the `case_id` field added to
`ContactManagerResolution` by VOIP-1252 (no longer has a producer once
the case-level write path is deleted).

## 6. Housekeeping: contact_attribution.go split

VOIP-1252 originally added `CaseListUnresolved`, `ReconcileContact`, and
`CaseListAll` to `contact_attribution.go` alongside the Resolution
functions being removed here. Of these three, **only
`CaseListUnresolved` and `CaseListAll` are unrelated to Resolution**
(round-1 correction: they are thin dbhandler delegations that read/list
whatever is already in `Case.contact_id`, with no dependency on
Resolution derivation) and must be preserved. `ReconcileContact` is
**not** preserved -- see §4's correction: it hard-depends on
`deriveCaseContactIDTx`/`applyDerivedContactID` and its whole reason to
exist (recomputing `Case.contact_id` after Resolution-row drift) is
gone once Resolution is no longer the source of truth. Move
`CaseListUnresolved`/`CaseListAll` to a new file
`bin-contact-manager/pkg/casehandler/contact_unresolved.go` before
deleting `contact_attribution.go`, so no functionality is lost in the
revert.

## 7. Doc-comment cleanup

Several existing doc-comments reference the now-removed Resolution
mechanism as the source of truth for `Case.contact_id` and must be
corrected as part of this change (not left stale):

- `kase.Case.ContactID` field comment ("single source of truth is
  Resolution, single derivation function") -> "single source of truth
  is this column itself; every write goes through
  `casehandler.UpdateContact`."
- `dbhandler.CaseUpdateContactID`/`CaseClearContactIDTx` doc-comments
  ("design §3.4; single source of truth is Resolution") -> reference
  VOIP-1253 instead.
- `models/resolution/resolution.go`'s package doc ("OR a whole
  contact_case") -> note that the CaseID branch is legacy/unused as of
  VOIP-1253 (see §4's note on leaving the column in place).
- `bin-contact-manager/pkg/casehandler/case_tag.go`'s
  `verifyCaseOwnership` doc-comment (round-1 correction: currently
  name-drops `ResolutionCreateCaseLevel`/`ResolutionDeleteCaseLevel` as
  example callers of this shared choke point) -> update the caller list
  to reference `UpdateContact` instead, so it doesn't dangle-reference
  deleted functions.

## 8. Open questions

1. Should `resolution.Resolution.CaseID` (the struct field) and its DB
   column be dropped in a follow-up migration, or left as permanently
   dead/unused? Leaving it costs nothing at read time (always nil going
   forward) but is a minor footgun for a future engineer who might
   assume it's still written. Recommend a follow-up ticket, not blocking
   this one.
2. Production verification: this design assumes zero case-level
   Resolution rows exist in production (VOIP-1252 merged 2026-07-12 with
   no UI client). Since production DB is not directly reachable
   (documented network-isolation finding from VOIP-1246), this can't be
   directly confirmed before merge. If wrong, existing case-level
   Resolution rows would become orphaned (still readable via
   interaction-level `ResolutionListByInteraction`... no, actually NOT
   readable at all, since case-level rows have `interaction_id IS NULL`
   and no query targets them post-revert). Mitigate by having
   case-control (or a one-off script) check for
   `SELECT COUNT(*) FROM contact_resolutions WHERE case_id IS NOT NULL`
   during a maintenance window before merge, or accept the risk given
   the near-zero likelihood.

## 9. Next steps

Independent subagent review loop (minimum 3 rounds) on this design doc
before implementation starts, per `voipbin-backend-feature-design`
skill policy.
