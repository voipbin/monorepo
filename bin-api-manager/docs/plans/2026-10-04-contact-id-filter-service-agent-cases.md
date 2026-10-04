# VOIP-1563: Add contact_id filter to service_agents contact_cases

Status: Draft
Ticket: VOIP-1563 (sibling: SQUARE-87, which depends on this)
Source: square-talk in-call analysis decision (9) — CEO confirmed 9-a (2026-10-04): add the backend filter rather than client-side filtering (9-b), as a separate small ticket.

## 1. Problem statement

`GET /service_agents/contact_cases` (the agent-facing case list) has no `contact_id` filter. `ServiceAgentCaseList` (`bin-api-manager/pkg/servicehandler/serviceagent_case.go:21-34`) always calls `ContactV1CaseList` with `contactID=uuid.Nil`, by design comment: *"contact_id filters are deliberately left empty/nil -- this returns the full list for the customer with no server-side owner filtering; square-talk filters client-side."*

The admin-facing equivalent (`GET /contact_cases`, `CaseList` in `bin-api-manager/pkg/servicehandler/case.go:53-83`) and the underlying RPC (`ContactV1CaseList`, `bin-common-handler/pkg/requesthandler/contact_cases.go:83-91`, `contactID uuid.UUID` parameter at position 5) already fully support this filter. The gap is purely in the service-agent HTTP surface: the OpenAPI param is missing (`bin-openapi-manager/openapi/paths/service_agents/contact_cases.yaml`) and the handler/servicehandler hard-code `uuid.Nil`.

Consequence: square-talk (SQUARE-87) cannot reliably show "this contact's cases" — it must load the tenant's first 100 cases client-side and filter in JS, silently dropping any case beyond the 100th for tenants with more cases. An agent could conclude "no cases for this contact" when cases exist but are paginated out.

## 2. Goals

1. `GET /service_agents/contact_cases?contact_id=<uuid>` returns only cases whose `contact_id` matches, with correct server-side pagination (no 100-row silent-drop risk).
2. Omitting `contact_id` preserves exactly today's behavior (full tenant list) — backward compatible, no breaking change for existing square-talk callers.
3. No change to the admin surface (`/contact_cases`), `bin-contact-manager`, or the `ContactV1CaseList` RPC signature — they already support this; this ticket only wires the existing capability into the service-agent HTTP layer.

## 3. Non-goals

- Any other `contact_cases` filter (status, owner_type, owner_id, reference_id) for the service-agent surface — out of scope, not requested, would need separate CEO sign-off on whether agents should see those filters.
- `square-talk` frontend changes to actually call this new parameter — tracked in SQUARE-87, which depends on this ticket merging first.
- Any permission model change — `ServiceAgentCaseList`'s existing `hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll)` check and tenant-scoping (`a.CustomerID`) are unchanged; `contact_id` is an additional narrowing filter within the same tenant boundary, never a cross-tenant query (enforced server-side identically to the admin surface, which resolves the same `ContactV1CaseList` RPC and has no reported cross-tenant leak).

## 4. Affected files

| File | Why |
|---|---|
| `bin-openapi-manager/openapi/paths/service_agents/contact_cases.yaml` | Add `contact_id` query parameter spec, mirroring `bin-openapi-manager/openapi/paths/contact_cases/main.yaml`'s existing `contact_id` param definition exactly. |
| `bin-api-manager/gens/openapi_server/gen.go` | Regenerated (not hand-edited) via `go generate ./openapi/config_server/...` after the YAML change — adds `ContactId *openapi_types.UUID` to `GetServiceAgentsContactCasesParams`. |
| `bin-api-manager/server/service_agents_contact_cases.go` | Parse `params.ContactId` (mirroring `server/contact_cases.go:58-61`'s exact pattern) and pass it through to `ServiceAgentCaseList`. |
| `bin-api-manager/pkg/servicehandler/serviceagent_case.go` | `ServiceAgentCaseList` gains a `contactID uuid.UUID` parameter; passes it to `ContactV1CaseList` instead of the hard-coded `uuid.Nil`. Remove the now-stale design comment claiming "deliberately left empty." |
| `bin-api-manager/pkg/servicehandler/main.go` | `ServiceAgentCaseList` interface signature gains the `contactID uuid.UUID` parameter. |
| `bin-api-manager/pkg/servicehandler/mock_main.go` | Regenerated (not hand-edited) via `go generate ./pkg/servicehandler/...` (mockgen) to match the updated interface. |
| `bin-api-manager/pkg/servicehandler/serviceagent_case_test.go` | Update existing `Test_ServiceAgentCaseList` call sites for the new parameter; add new table-driven cases for the filter. |
| `bin-api-manager/server/service_agents_contact_cases_test.go` (if it exists; else create) | Add/update HTTP-layer test for `contact_id` query param parsing. |

No `bin-contact-manager` change: `ContactV1CaseList`'s RPC handler and the underlying `dbhandler` query already support `contactID` (proven by the admin surface using it today).

## 5. Exact change

### 5.1 OpenAPI spec (`service_agents/contact_cases.yaml`)

Add, immediately after the existing `PageSize`/`PageToken` refs are removed and re-ordered to put the new filter first (matching the admin yaml's order: filters before pagination):

```yaml
get:
  summary: Get list of contact cases
  description: Retrieves a paginated list of cases for the authenticated agent's customer, optionally filtered by contact_id.
  tags:
    - Service Agent
  parameters:
    - name: contact_id
      in: query
      description: Filter to cases attributed to this Contact.
      schema:
        type: string
        format: uuid
        example: "b2c3d4e5-f6a7-8901-bcde-f12345678901"
    - $ref: '#/components/parameters/PageSize'
    - $ref: '#/components/parameters/PageToken'
  responses:
    # unchanged
```

### 5.2 `serviceagent_case.go`

Current:
```go
// ServiceAgentCaseList sends a request to contact-manager to list cases for
// the service agent's own customer. Status/owner_type/owner_id/contact_id
// filters are deliberately left empty/nil -- this returns the full list for
// the customer with no server-side owner filtering; square-talk filters
// client-side (design §3.1/3.3).
func (h *serviceHandler) ServiceAgentCaseList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cmkase.Case, string, error) {
	...
	items, nextToken, err := h.reqHandler.ContactV1CaseList(ctx, a.CustomerID, "", "", uuid.Nil, uuid.Nil, size, token, "")
	...
}
```

New:
```go
// ServiceAgentCaseList sends a request to contact-manager to list cases for
// the service agent's own customer, optionally filtered by contactID.
// Status/owner_type/owner_id/reference_id filters remain unsupported on this
// surface (VOIP-1563 scope is contact_id only; square-talk still filters
// those client-side if ever needed -- see design §3).
func (h *serviceHandler) ServiceAgentCaseList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, contactID uuid.UUID) ([]*cmkase.Case, string, error) {
	...
	items, nextToken, err := h.reqHandler.ContactV1CaseList(ctx, a.CustomerID, "", "", uuid.Nil, contactID, size, token, "")
	...
}
```

### 5.3 `server/service_agents_contact_cases.go`

Insert before the `ServiceAgentCaseList` call, mirroring `server/contact_cases.go:58-61`:
```go
	contactID := uuid.Nil
	if params.ContactId != nil {
		contactID = uuid.UUID(*params.ContactId)
	}
```
and update the call site: `h.serviceHandler.ServiceAgentCaseList(c.Request.Context(), a, pageSize, pageToken, contactID)`.

### 5.4 Wire-field checklist (verified against this repo 2026-10-04)

| Field | Source | Verified shape |
|---|---|---|
| `ContactV1CaseList(ctx, customerID uuid.UUID, status, ownerType string, ownerID, contactID uuid.UUID, size uint64, token string, referenceID string)` | `bin-common-handler/pkg/requesthandler/contact_cases.go:83-91` | `contactID` is parameter 5, already present; admin's `CaseList` (`case.go:83`) already passes it through unmodified — this ticket does not touch this RPC. |
| `GetContactCasesParams.ContactId` (admin, reference pattern) | `bin-api-manager/gens/openapi_server/gen.go:8959` (generated from `openapi/paths/contact_cases/main.yaml`) | `*openapi_types.UUID` `form:"contact_id,omitempty" json:"contact_id,omitempty"` — this is the exact shape to replicate for the service-agent params struct. |
| `server/contact_cases.go:58-61` (admin handler, reference pattern) | existing code | `contactID := uuid.Nil; if params.ContactId != nil { contactID = uuid.UUID(*params.ContactId) }` — exact pattern to copy into `service_agents_contact_cases.go`. |
| gen.go regeneration command | `bin-api-manager/openapi/config_server/generate.go:3` | `go run -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config config.generate.yaml -o ../../gens/openapi_server/gen.go ../../../bin-openapi-manager/openapi/openapi.yaml` — run via `go generate ./openapi/config_server/...` from `bin-api-manager/`. |
| mock_main.go regeneration command | `bin-api-manager/pkg/servicehandler/main.go:3` | `//go:generate mockgen -package servicehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod` — run via `go generate ./pkg/servicehandler/...`. |

## 6. Verification plan

1. `go generate ./openapi/config_server/...` (regenerate `gen.go`) — diff should show only the new `ContactId` field added to `GetServiceAgentsContactCasesParams`, no other params/paths affected.
2. `go generate ./pkg/servicehandler/...` (regenerate `mock_main.go`) — diff should show only the `ServiceAgentCaseList` mock signature updated.
3. `go build ./...` — must pass.
4. `go test ./pkg/servicehandler/... -run ServiceAgentCaseList -v` — update existing test call sites (new parameter), add cases: (a) `contact_id` omitted → `uuid.Nil` passed through (regression, today's behavior), (b) `contact_id` provided → passed through unchanged to the RPC mock.
5. `go test ./server/... -run ServiceAgentContactCases -v` (or equivalent existing test file name) — HTTP-layer param parsing test for `contact_id` present/absent.
6. `go vet ./...`.
7. Full `go test ./...` for regression (no other `ServiceAgentCaseList` call sites expected — confirm via `grep -rn "ServiceAgentCaseList(" --include=*.go . | grep -v vendor`).

## 7. Rollout / risk

- **Risk: none for backward compatibility** — omitting `contact_id` is identical to today's call (`uuid.Nil`), verified by §5.2's diff.
- **Risk: none for cross-tenant exposure** — `contactID` is an additional AND-filter within `a.CustomerID`'s existing tenant scope; the RPC/dbhandler query ANDs both, never ORs or overrides the tenant filter (same guarantee the admin surface already relies on).
- **Rollback:** revert the diff; no data migration, no schema change.

## 8. Open questions

None outstanding. Scope was CEO-confirmed (decision 9, 9-a) before drafting.

## 9. Approval status

Draft — awaiting Design Review→Fix loop.
