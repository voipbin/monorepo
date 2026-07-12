# VOIP-1252 -- Case-Level Manual Contact Attribution API (Design v0.1)

Repo: voipbin/monorepo
Ticket: VOIP-1252
Author: Hermes (CPO) on behalf of pchero (CEO/CTO)
Date: 2026-07-12
Status: Draft, awaiting independent review

## Changelog

- v0.1 (2026-07-12). Initial draft.

## 1. Problem recap

A `contact_case` (kase.Case) can be created for a peer address (phone/email/SIP)
that has no matching `Contact` yet -- e.g. an inbound call from a number nobody
has entered into the CRM. Today the platform has NO reachable way for an agent
or admin to say "this Case belongs to this specific existing Contact." The
domain logic and DB write path (`casehandler.ResolutionCreateCaseLevel`) is
fully implemented, transactional, and unit-tested, but it has zero callers
outside tests/mocks -- there is no listenhandler route, no requesthandler RPC
client method, and no bin-api-manager servicehandler/route exposing it. The
only reachable recovery tool is the case-control CLI's `reconcile-contact`,
which only RE-DERIVES `contact_id` from Resolution rows that already exist --
it cannot create the first attribution.

Severity: feature gap (not a regression). No workaround exists today except a
raw SQL insert against `contact_resolutions`, which bypasses tenant validation
and the atomic contact_id-derivation write.

## 2. Goal

- An agent (or admin/manager) can attach an open, unresolved Case to a
  specific, existing Contact via a single API call.
- The same caller can undo that attribution (soft-delete) if it was wrong,
  and `Case.contact_id` automatically reflects the corrected state -- no
  separate "recompute" step required.
- Every attribution/de-attribution is attributed (who did it, when) via the
  existing `contact_resolutions` audit trail -- nothing new needed here, the
  underlying `ResolutionCreateCaseLevel`/`ResolutionDeleteCaseLevel` already
  do this.
- Cross-tenant Case ids are rejected the same way every other Case mutation
  endpoint rejects them (via `verifyCaseOwnership`, already implemented).

## 3. Out of scope

- square-admin UI changes to surface this action (follow-up ticket once the
  API exists).
- Bulk/batch attribution tooling.
- Changing `deriveCaseContactID`'s derivation algorithm itself.
- Interaction-level (not Case-level) resolutions -- `POST
  /v1/interactions/{id}/resolutions` already exists and is unaffected.
- Listing a Case's resolutions (not strictly needed for the attach/detach
  flow; `ResolutionListByCase` already exists at the dbhandler layer if a
  future GET is wanted -- flagged as an open question, Section 8).

## 4. API design

### 4.1 New resource shape

Follows the existing `/v1/cases/{id}/notes` and `/v1/cases/{id}/tags`
sub-resource convention exactly (see `v1_cases.go`, `v1_notes.go` pattern):

```
POST   /v1/cases/{id}/resolutions          -- attach Case to a Contact
DELETE /v1/cases/{id}/resolutions/{res_id} -- undo an attribution
```

No GET is added in v1 (see Section 8, open question 1).

### 4.2 bin-contact-manager: listenhandler

New file `bin-contact-manager/pkg/listenhandler/v1_case_resolutions.go`,
mirroring `v1_cases.go`'s note/tag handlers:

```go
// processV1CasesIDResolutionsPost handles POST /v1/cases/{id}/resolutions.
func (h *listenHandler) processV1CasesIDResolutionsPost(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDResolutionsPost
	if err := json.Unmarshal(req.Data, &body); err != nil {
		return simpleResponse(400), nil
	}
	if body.CustomerID == uuid.Nil || body.ContactID == uuid.Nil {
		return simpleResponse(400), nil
	}

	res, err := h.caseHandler.ResolutionCreateCaseLevel(
		ctx, body.CustomerID, id, body.ContactID,
		body.ResolutionType, body.ResolvedByType, body.ResolvedByID,
	)
	if err != nil {
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}
	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1CasesIDResolutionsIDDelete handles
// DELETE /v1/cases/{id}/resolutions/{resolution_id}.
func (h *listenHandler) processV1CasesIDResolutionsIDDelete(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}
	resolutionID := caseSubIDFromURI(req.URI)
	if resolutionID == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDResolutionsIDDelete
	if err := json.Unmarshal(req.Data, &body); err != nil {
		return simpleResponse(400), nil
	}
	if body.CustomerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	if err := h.caseHandler.ResolutionDeleteCaseLevel(ctx, body.CustomerID, id, resolutionID); err != nil {
		return errorResponse(err), nil
	}
	return simpleResponse(200), nil
}
```

Reuses `caseIDFromURI`/`caseSubIDFromURI` already defined in `v1_cases.go` --
no new URI-parsing helpers needed since the shape (`/v1/cases/<uuid>/<sub>/<uuid>`)
is identical to the existing notes/tags sub-resources.

Register both routes in the listenhandler's route table (wherever
`processV1CasesIDNotesPost`/`processV1CasesIDNotesIDDelete` are registered --
same file, same pattern).

### 4.3 bin-contact-manager: request models

New file `bin-contact-manager/pkg/listenhandler/models/request/v1_case_resolutions.go`:

```go
package request

import "github.com/gofrs/uuid"

// V1DataCasesIDResolutionsPost is the request body for
// POST /v1/cases/{id}/resolutions.
type V1DataCasesIDResolutionsPost struct {
	CustomerID     uuid.UUID `json:"customer_id"`
	ContactID      uuid.UUID `json:"contact_id"`
	ResolutionType string    `json:"resolution_type"`
	ResolvedByType string    `json:"resolved_by_type"`
	ResolvedByID   uuid.UUID `json:"resolved_by_id"`
}

// V1DataCasesIDResolutionsIDDelete is the request body for
// DELETE /v1/cases/{id}/resolutions/{resolution_id}.
type V1DataCasesIDResolutionsIDDelete struct {
	CustomerID uuid.UUID `json:"customer_id"`
}
```

Note: unlike `V1DataCasesIDNotesPost` (which allows an optional/system
`AuthorID *uuid.UUID`), `ResolvedByID` is REQUIRED here -- there is no
"system-initiated" attach path in this design, only agent/admin action. If a
future automated-attribution path is added, this can loosen the same way
Notes did.

### 4.4 bin-common-handler: requesthandler RPC client

New file `bin-common-handler/pkg/requesthandler/contact_case_resolutions.go`,
following `ContactV1CaseClose`'s exact shape (URI templating, JSON marshal,
`sendRequestContact`, `parseResponse`):

```go
// ContactV1CaseResolutionCreate attaches a case to a contact by creating a
// case-level Resolution in contact-manager.
func (r *requestHandler) ContactV1CaseResolutionCreate(
	ctx context.Context,
	customerID, caseID, contactID uuid.UUID,
	resolutionType, resolvedByType string,
	resolvedByID uuid.UUID,
) (*cmresolution.Resolution, error) {
	uri := fmt.Sprintf("/v1/cases/%s/resolutions", caseID)

	data := &cmrequest.V1DataCasesIDResolutionsPost{
		CustomerID:     customerID,
		ContactID:      contactID,
		ResolutionType: resolutionType,
		ResolvedByType: resolvedByType,
		ResolvedByID:   resolvedByID,
	}
	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/cases/<id>/resolutions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmresolution.Resolution
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}
	return &res, nil
}

// ContactV1CaseResolutionDelete undoes a case-level Contact attribution.
func (r *requestHandler) ContactV1CaseResolutionDelete(ctx context.Context, customerID, caseID, resolutionID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/cases/%s/resolutions/%s", caseID, resolutionID)

	data := &cmrequest.V1DataCasesIDResolutionsIDDelete{CustomerID: customerID}
	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/cases/<id>/resolutions/<resolution-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	return err
}
```

Add both method signatures to the `RequestHandler` interface in
`bin-common-handler/pkg/requesthandler/main.go` (alongside the existing
`ContactV1Case*` group), regenerate the mock.

### 4.5 bin-api-manager: servicehandler + auth

New methods in `bin-api-manager/pkg/servicehandler/case.go`, mirroring
`CaseClose`'s permission pattern exactly (tenant pre-check via `caseGet`,
then `hasPermission` with `PermissionCustomerAdmin|PermissionCustomerManager`,
then delegate):

```go
// CaseResolutionCreate attaches a case to a contact (design VOIP-1252).
// resolved_by_id is derived server-side from the authenticated caller's own
// agent identity (a.AgentID()), matching CaseClose/CaseContinue's pattern --
// never accepted from client input, so attribution cannot be forged.
func (h *serviceHandler) CaseResolutionCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	id, contactID uuid.UUID,
	resolutionType string,
) (*cmresolution.Resolution, error) {
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

	return h.reqHandler.ContactV1CaseResolutionCreate(
		ctx, a.CustomerID, id, contactID,
		resolutionType, string(commonidentity.OwnerTypeAgent), a.AgentID(),
	)
}

// CaseResolutionDelete undoes a case-level Contact attribution.
func (h *serviceHandler) CaseResolutionDelete(ctx context.Context, a *auth.AuthIdentity, id, resolutionID uuid.UUID) error {
	if a.IsDirect() {
		return serviceerrors.ErrDirectAccessNotSupported
	}

	c, err := h.caseGet(ctx, a.CustomerID, id)
	if err != nil {
		return err
	}
	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return serviceerrors.ErrPermissionDenied
	}

	return h.reqHandler.ContactV1CaseResolutionDelete(ctx, a.CustomerID, id, resolutionID)
}
```

`resolution_type` is accepted from the client (positive/negative -- see
`resolution.ResolutionTypePositive`/`Negative`) since both are legitimate
client-facing actions (attach vs. explicitly suppress a wrong auto-match at
the Case level), but `resolved_by_id`/`resolved_by_type` are always
server-derived, matching every other Case mutation's closed_by/caller_id
pattern in this file.

Add both to the `ServiceHandler` interface + regenerate mock. Add
`server/case_resolutions.go` HTTP handlers (`PostCasesIdResolutions`,
`DeleteCasesIdResolutionsResolutionId`) mirroring `server/case.go`'s existing
close/continue handlers' request parsing.

### 4.6 OpenAPI spec

New files under `bin-openapi-manager/openapi/paths/contact_cases/`:

- `id_resolutions.yaml` (POST)
- `id_resolutions_id.yaml` (DELETE)

Registered in `openapi.yaml` under:
```yaml
/contact_cases/{id}/resolutions:
  $ref: './paths/contact_cases/id_resolutions.yaml'
/contact_cases/{id}/resolutions/{resolution_id}:
  $ref: './paths/contact_cases/id_resolutions_id.yaml'
```

Request schema `ContactManagerCaseResolutionCreate`: `contact_id` (uuid,
required), `resolution_type` (enum: positive, negative, required). Response
schema reuses/extends the existing Resolution webhook schema (verify whether
`ContactManagerResolution` already exists as a schema from the
interaction-level `POST /v1/interactions/{id}/resolutions` endpoint -- if so,
reuse it; `resolution.Resolution` carries no internal-only fields requiring
a separate webhook type, same rationale as `case.go`'s existing
`ConvertWebhookMessage` note for Case).

## 5. Integration plan

1. `bin-contact-manager`: add request models, listenhandler handlers + route
   registration, mock regeneration (`go generate ./pkg/listenhandler/...`).
2. `bin-common-handler`: add requesthandler methods + interface entries,
   mock regeneration.
3. `bin-openapi-manager`: add path files + schema, `go generate ./...`.
4. `bin-api-manager`: add servicehandler methods + interface + mock, add
   `server/` HTTP handlers wired to the new OpenAPI-generated route
   signatures, `go generate ./...` then `go build ./...`.
5. Run full verification workflow (`go mod tidy && go mod vendor && go
   generate ./... && go test ./... && golangci-lint run -v --timeout 5m`) in
   all four touched services.
6. RST docs: add a short "Attaching a Case to a Contact" subsection to
   `bin-api-manager/docsdev/source/` case docs (Case struct/tutorial rst),
   per bin-api-manager's CLAUDE.md mandate for user-visible API additions.

## 6. Test plan

| Layer | New test file | Cases covered |
|---|---|---|
| bin-contact-manager listenhandler | `v1_case_resolutions_test.go` | 200 success (create/delete), 400 missing customer_id/contact_id, cross-tenant case_id -> propagated NotFound/error, malformed JSON body |
| bin-contact-manager casehandler | already covered by existing `contact_attribution_write_test.go` -- no new domain-layer tests needed, this ticket only adds a transport wrapper |
| bin-common-handler requesthandler | `contact_case_resolutions_test.go` | URI construction, success parse, error propagation -- mirrors `Test_ContactV1CaseClose` shape |
| bin-api-manager servicehandler | `case_test.go` additions | permission denied (non-admin/manager), direct-access rejected, cross-tenant case_id rejected before contact_id is even used, success path asserts `resolved_by_id` is taken from `a.AgentID()` not client input |
| bin-api-manager server | `case_resolutions_test.go` | HTTP-level request parsing, status code mapping |

## 7. Risks & tradeoffs

- **Blast radius: small.** No changes to `ResolutionCreateCaseLevel`,
  `ResolutionDeleteCaseLevel`, or `deriveCaseContactID` -- this is pure
  transport wiring on top of an already-reviewed, already-tested domain
  function (contact_attribution_write_test.go). Risk is concentrated in the
  new listenhandler/servicehandler auth wiring, not in write correctness.
- **resolution_type=negative use case.** A case-level negative resolution
  suppresses a positive one for the same (case_id, contact_id) per set-MINUS
  semantics (see `resolution.go` doc comment) -- this endpoint technically
  allows an agent to submit a negative resolution for a Case that was never
  positively attached in the first place, which is a harmless no-op (derives
  to the same nil contact_id) but worth a unit test to confirm it doesn't
  error.
- **No GET added.** An agent cannot currently list a Case's resolution
  history via this API surface. Deferred to Section 8 as an open question
  rather than silently included, to keep this ticket's scope matching what
  pchero actually asked for (attach a case to a contact).

## 8. Open questions

1. Should a `GET /v1/cases/{id}/resolutions` (list) endpoint be added in the
   same PR, or deferred to a follow-up ticket? `ResolutionListByCase` already
   exists at the dbhandler layer, so the marginal cost is low, but it's not
   part of the originally-scoped ask.
2. Should the square-admin UI work (a button/flow to search-and-attach a
   Contact from an unresolved Case) be filed as a linked follow-up ticket now,
   or wait until this API ships?
3. Confirm whether `ContactManagerResolution` already exists as an OpenAPI
   schema (from the interaction-level resolutions endpoint) to reuse, or
   whether a new schema needs defining -- flagged for verification during
   implementation, not blocking design lock-in.

## 9. Next steps

Independent subagent review loop (minimum 3 rounds) on this design doc before
implementation starts, per `voipbin-backend-feature-design` skill policy.
