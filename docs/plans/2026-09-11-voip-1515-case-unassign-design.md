# VOIP-1515: Case Owner Unassign (top-level + service_agents)

- JIRA: https://voipbin.atlassian.net/browse/VOIP-1515
- Status: Design (no implementation in this change)
- Worktree: `~/gitvoipbin/monorepo/.worktrees/VOIP-1515-case-unassign-endpoints`
- Branch: `VOIP-1515-case-unassign-endpoints`

## 1. Background

VOIP-1514 added `POST /contact_cases/{id}/assign` (top-level, Admin/Manager)
and `POST /service_agents/contact_cases/{id}/assign` (agent self-service),
both of which set a case's `owner_type`/`owner_id`. Case has no symmetric
"unassign" operation today, unlike Conversation, which already has both
`ConversationUnassign` (top-level) and `ServiceAgentConversationUnassign`
(service_agents), added ahead of Case.

This design adds the two missing Case endpoints, mirroring Conversation's
existing unassign pattern as closely as the two resources' actual code
allows, and reusing VOIP-1514's Case-specific conventions (closed-case
handling, cross-tenant anti-enumeration, ServiceHandler layering) where
Conversation's pattern and Case's pattern diverge.

### 1.1 Reference code read for this design (file:line as of this branch)

- `bin-api-manager/pkg/servicehandler/conversation.go` `ConversationUnassign`
  (top-level unassign; permission model to mirror)
- `bin-api-manager/pkg/servicehandler/serviceagent_conversation.go`
  `ServiceAgentConversationUnassign` (service_agents unassign; permission
  model to mirror)
- `bin-api-manager/pkg/servicehandler/case.go` `CaseAssign`, `caseGet`
  (Case-specific conventions: closed-case guard, `caseGet` tenant-scoped
  fetch, `cmkase.Case` returned directly with no WebhookMessage wrapper)
- `bin-api-manager/pkg/servicehandler/serviceagent_case.go`
  `ServiceAgentCaseAssign`, `ServiceAgentCaseClose` (service_agents
  conventions: `PermissionAll` gate, `caseGet(ctx, a.CustomerID, id)`)
- `bin-contact-manager/pkg/casehandler/assign.go` `Assign` (generic
  owner-type/owner-id writer; already unassign-capable by contract)
- `bin-contact-manager/pkg/dbhandler/kase.go` `CaseUpdateOwner` (plain
  UPDATE, no schema change needed)
- `bin-contact-manager/pkg/listenhandler/v1_cases.go`
  `processV1CasesIDAssignPost` (existing RPC endpoint, reused as-is)
- `bin-common-handler/pkg/requesthandler/contact_cases.go`
  `ContactV1CaseAssign` (RPC client; hardcodes `owner_type=agent`, which is
  wrong for unassign and must not be reused as-is -- see §3)
- `bin-common-handler/models/identity/owner.go` (`OwnerTypeNone = ""`,
  `OwnerTypeAgent = "agent"`)
- `bin-api-manager/server/contact_cases.go` `PostContactCasesIdAssign`
  (server handler pattern to mirror, no-body variant per
  `PostContactCasesIdClose`)
- `bin-api-manager/pkg/serviceerrors/sentinels.go` (`ErrPermissionDenied`,
  `ErrNotFound`, `ErrCaseClosed`, `ErrDirectAccessNotSupported`,
  `ErrAuthenticationRequired`)
- `bin-openapi-manager/openapi/paths/conversations/id_unassign.yaml` +
  `service_agents/conversations_id_unassign.yaml` (no-body POST shape to
  mirror)
- `bin-openapi-manager/openapi/paths/contact_cases/id_assign.yaml` +
  `service_agents/contact_cases_id_assign.yaml` (Case-specific yaml
  conventions: schema ref `ContactManagerCase`, description style)
- `bin-openapi-manager/openapi/openapi.yaml:8670-8889` (path registration
  block for `/contact_cases/*` and `/service_agents/contact_cases/*`)
- `bin-api-manager/pkg/servicehandler/main.go:531-568,1030-1039`
  (`ServiceHandler` interface, Case + ServiceAgentCase method blocks)
- `bin-api-manager/docsdev/source/contact_case_overview.rst`,
  `service_agent_case_overview.rst` (RST prose to extend)
- `bin-api-manager/pkg/servicehandler/conversation_test.go`
  `Test_ConversationUnassign`, `bin-api-manager/pkg/servicehandler/case_test.go`
  `Test_CaseAssign`, `bin-api-manager/server/contact_cases_test.go`
  `Test_PostContactCasesIdAssign` (test patterns to mirror)

## 2. Scope

Two new endpoints, both operating on the existing `owner_type`/`owner_id`
columns on `kase` (no DB migration):

| Endpoint | Caller | Behavior |
|---|---|---|
| `POST /contact_cases/{id}/unassign` | Admin/Manager, OR the agent who currently owns the case | Clears `owner_type`/`owner_id` on the given case |
| `POST /service_agents/contact_cases/{id}/unassign` | The agent who currently owns the case only | Same, agent-only surface |

Out of scope: any change to `Assign`'s owner-validation behavior, any DB
schema change, any change to Conversation's existing unassign endpoints.

## 3. bin-contact-manager layer: reuse existing `Assign` RPC, no new RPC

**Decision: reuse the existing `POST /v1/cases/{id}/assign` RPC endpoint
and `casehandler.Assign` function as-is. No new RPC/listenhandler route is
added in bin-contact-manager.**

Evidence this works cleanly for unassign:

- `casehandler.Assign(ctx, customerID, id, ownerType, ownerID)` is generic
  over `commonidentity.OwnerType` and `uuid.UUID`. It performs a
  tenant-scoped `CaseGetByID`, then unconditionally calls
  `h.db.CaseUpdateOwner(ctx, customerID, id, ownerType, ownerID)` and
  returns the refreshed case. There is **no validation inside `Assign`**
  that `ownerType`/`ownerID` are non-empty/non-nil, and no lookup against
  agent-manager at this layer (that check lives one layer up, in
  `bin-api-manager/pkg/servicehandler`, only for the *assign* direction --
  see `CaseAssign`'s `AgentV1AgentGet` call). Calling
  `Assign(ctx, customerID, id, commonidentity.OwnerTypeNone, uuid.Nil)`
  is therefore already a fully correct unassign: it writes
  `owner_type=""`, `owner_id=00000000-0000-0000-0000-000000000000` via
  `CaseUpdateOwner`, exactly mirroring what `ConversationV1ConversationUpdate`
  does for Conversation with `FieldOwnerID: uuid.Nil`.
- `CaseUpdateOwner` is a plain `sq.Update(caseTable).Set("owner_type",
  ...).Set("owner_id", ...)`. It has no special-case handling to add or
  remove; writing `OwnerTypeNone`/`uuid.Nil` is already representable in
  the schema (`OwnerTypeNone = ""` is a defined enum value in
  `bin-common-handler/models/identity/owner.go`, used for the unowned
  state -- it is not a placeholder that was only ever meant for
  `assign`'s pre-fill).
- `processV1CasesIDAssignPost` in `bin-contact-manager/pkg/listenhandler/v1_cases.go`
  unmarshals `request.V1DataCasesIDAssign{CustomerID, OwnerType, OwnerID}`
  and calls `caseHandler.Assign(ctx, body.CustomerID, id,
  commonidentity.OwnerType(body.OwnerType), body.OwnerID)` verbatim --
  no assign-specific business logic lives in the listenhandler. Posting
  `{"customer_id": ..., "owner_type": "", "owner_id":
  "00000000-0000-0000-0000-000000000000"}` to the same route is a
  complete, correct unassign at this layer.

**What must change: the `bin-common-handler` RPC client.**
`ContactV1CaseAssign(ctx, customerID, id, ownerID)` currently hardcodes
`OwnerType: string(commonidentity.OwnerTypeAgent)` on the wire and does not
accept an owner type parameter at all (documented in its own doc comment
as deliberate, since assign only ever needs `agent`). It cannot be
reused unmodified for unassign without smuggling in a wrong,
non-empty `owner_type: "agent"` alongside `owner_id: uuid.Nil` -- that
would leave a dangling `owner_type=agent` with no real owner id, an
inconsistent state Conversation's `FieldOwnerID: uuid.Nil` update does
not produce (Conversation's owner_type/owner_id apparently move together
via `cvconversation.FieldOwnerID`, in effect a single logical field on
that model; Case's schema exposes them as two independent columns, so
they must be zeroed together explicitly).

**Decision: add one new, dedicated RPC client function**,
`ContactV1CaseUnassign(ctx context.Context, customerID, id uuid.UUID)
(*cmkase.Case, error)`, in
`bin-common-handler/pkg/requesthandler/contact_cases.go`, alongside
`ContactV1CaseAssign`. It POSTs to the *same* URI
(`/v1/cases/{id}/assign`) and reuses the *same* wire DTO
(`cmrequest.V1DataCasesIDAssign`), but populates
`OwnerType: string(commonidentity.OwnerTypeNone)` and `OwnerID: uuid.Nil`
instead of the agent-owner values. This is a client-side-only addition:
no new bin-contact-manager listenhandler route, no new
`request.V1DataCasesIDUnassign` DTO, no new `casehandler` method. It
exists purely so servicehandler callers get a clearly-named,
intention-revealing function (`ContactV1CaseUnassign`) instead of having
to poke `OwnerTypeNone`/`uuid.Nil` into `ContactV1CaseAssign`'s signature
(which is intentionally agent-only per its own doc comment) or expose the
raw generic three-argument `Assign` shape at the requesthandler layer.

This new function must also be added to the `RequestHandler` interface
declaration in `bin-common-handler/pkg/requesthandler/main.go` (next to
the existing `ContactV1CaseAssign` line, ~952-953) -- `bin-api-manager`'s
`reqHandler` field is typed as this interface, not the concrete struct,
so a servicehandler call to `h.reqHandler.ContactV1CaseUnassign(...)`
will not compile until the interface itself declares the method. See §7
for the full file list.

Rejected alternative: reuse `ContactV1CaseAssign(ctx, customerID, id,
uuid.Nil)` directly for unassign. Rejected because that function's
implementation and doc comment permanently fix `owner_type` to
`"agent"`, which is exactly the value unassign must **not** write; a
caller passing `uuid.Nil` through this path would corrupt the row into
`owner_type=agent, owner_id=00000000-...` (a "ghost owner"), not the
correct `owner_type="", owner_id=00000000-...` (no owner). This is a
real, not merely cosmetic, divergence and is the reason a second RPC
client function is needed even though the RPC route and casehandler
function are unchanged.

## 4. bin-api-manager layer

### 4.1 New: `bin-api-manager/pkg/servicehandler/case.go` -- `CaseUnassign`

Signature: `CaseUnassign(ctx context.Context, a *auth.AuthIdentity, id
uuid.UUID) (*cmkase.Case, error)`

Mirrors `ConversationUnassign`'s permission logic exactly, adapted to
Case's existing helpers/conventions (`caseGet`, `a.AgentID()`,
`commonidentity.OwnerTypeAgent`):

```go
func (h *serviceHandler) CaseUnassign(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmkase.Case, error) {
    log := logrus.WithFields(logrus.Fields{
        "func":        "CaseUnassign",
        "customer_id": a.CustomerID,
        "case_id":     id,
    })

    if a.IsDirect() {
        return nil, serviceerrors.ErrDirectAccessNotSupported
    }

    c, err := h.caseGet(ctx, a.CustomerID, id)
    if err != nil {
        log.Errorf("Could not get the case info. err: %v", err)
        return nil, err
    }

    isAdminOrManager := h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager)
    isOwningAgent := a.IsAgent() && a.Agent != nil &&
        c.OwnerType == commonidentity.OwnerTypeAgent &&
        c.OwnerID == a.Agent.ID

    if !isAdminOrManager && !isOwningAgent {
        log.Info("Caller has no permission to unassign the case.")
        return nil, serviceerrors.ErrPermissionDenied
    }

    if c.Status == cmkase.StatusClosed {
        log.Infof("Case is closed, status: %s", c.Status)
        return nil, serviceerrors.ErrCaseClosed
    }

    res, err := h.reqHandler.ContactV1CaseUnassign(ctx, a.CustomerID, id)
    if err != nil {
        log.Errorf("Could not unassign the case. err: %v", err)
        return nil, err
    }

    return res, nil
}
```

Notes / deliberate decisions:

- Uses `caseGet(ctx, a.CustomerID, id)`, not a bare `ContactV1CaseGet`, so
  the tenant scoping and not-found semantics exactly match
  `CaseAssign`/`CaseGet`'s existing anti-enumeration behavior (cross-tenant
  and nonexistent IDs are indistinguishable).
- **Check ordering: `caseGet` -> `hasPermission`/owning-agent decision ->
  closed-case guard -- this exact order, confirmed against
  `CaseAssign`'s real source (`case.go:169-182`), not assumed.** An
  earlier draft of this section had the closed-case check before the
  permission check, which a design-review round caught as producing a
  different observable response than `CaseAssign` for a caller who is
  neither admin/manager nor the owner attempting to act on a closed case
  (`ErrCaseClosed` instead of `ErrPermissionDenied`). `CaseAssign` checks
  permission first (`case.go:175-177`) and closed-status second
  (`case.go:179-182`); this ordering is now byte-for-byte matched here so
  a non-privileged, non-owning caller gets `ErrPermissionDenied` (403)
  regardless of the case's status, exactly as `CaseAssign` would.
  `ConversationUnassign`'s ordering is fetch-then-decide-permission with
  no closed-case guard at all (Conversation has no closed-case concept),
  so it offers no ordering precedent for the closed-case check itself --
  `CaseAssign` is the sole precedent for that ordering, and it is now
  followed exactly.
- **Closed-case guard: present, mirroring `CaseAssign` exactly.**
  `bin-contact-manager/models/kase/kase.go:73-76` documents Owner as a
  **load-bearing invariant**: "NEVER cleared by closing a Case (design
  §7) -- this is a load-bearing invariant for `/continue`'s authorization
  (design §5.3)." `casehandler.Continue`
  (`bin-contact-manager/pkg/casehandler/lifecycle.go:138-142`) reads
  `source.OwnerType == callerType && source.OwnerID == callerID` directly
  off the closed case's Owner fields to decide whether a non-admin caller
  may re-open it. Zeroing Owner on a closed case via unassign -- even
  though "unassign" is a different action than "close" -- defeats this
  invariant's entire purpose exactly as effectively as closing itself
  clearing it would: the original owning agent permanently loses
  `/continue` eligibility on their own case (silently demoted to an
  admin-only recovery path), and the "who owned this when it closed"
  audit fact is lost (distinct from `closed_by`/`closed_reason`, which
  record who performed the close action, not who owned the case).
  Reopening this invariant would require a coordinated redesign of
  `/continue`'s authorization (e.g. a separate `closed_owner_id`
  snapshot column) that is out of scope for this ticket. **Revised
  decision (superseding the earlier recommendation in this section):
  `CaseUnassign`/`ServiceAgentCaseUnassign` copy `CaseAssign`'s
  `ErrCaseClosed` guard verbatim** (`if c.Status == cmkase.StatusClosed {
  return nil, serviceerrors.ErrCaseClosed }`, placed identically after
  the permission/owning-agent check and before the RPC call, matching
  `CaseAssign`'s exact ordering -- `caseGet` -> permission check ->
  closed-case guard -> RPC call, per `case.go:169-190`). This makes
  Unassign's closed-case behavior no longer a synthesis point at all --
  it is now a byte-for-byte copy of `CaseAssign`'s existing pattern
  (including check order), with no open question remaining.
- Returns `*cmkase.Case` directly (not a `WebhookMessage`), consistent
  with every other Case servicehandler method's documented rationale
  (`kase.Case` carries no internal-only fields to strip).

### 4.2 New: `bin-api-manager/pkg/servicehandler/serviceagent_case.go` -- `ServiceAgentCaseUnassign`

Signature: `ServiceAgentCaseUnassign(ctx context.Context, a
*auth.AuthIdentity, id uuid.UUID) (*cmkase.Case, error)`

Mirrors `ServiceAgentConversationUnassign`, adapted to
`ServiceAgentCaseAssign`'s existing conventions (`PermissionAll` gate,
`caseGet(ctx, a.CustomerID, id)`):

```go
func (h *serviceHandler) ServiceAgentCaseUnassign(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmkase.Case, error) {
    log := logrus.WithFields(logrus.Fields{
        "func":        "ServiceAgentCaseUnassign",
        "customer_id": a.CustomerID,
        "case_id":     id,
    })

    if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
        log.Info("The agent has no permission.")
        return nil, serviceerrors.ErrPermissionDenied
    }

    c, err := h.caseGet(ctx, a.CustomerID, id)
    if err != nil {
        log.Errorf("Could not get the case info. err: %v", err)
        return nil, err
    }

    if c.Status == cmkase.StatusClosed {
        log.Infof("Case is closed, status: %s", c.Status)
        return nil, serviceerrors.ErrCaseClosed
    }

    isOwningAgent := c.OwnerType == commonidentity.OwnerTypeAgent && c.OwnerID == a.AgentID()
    if !isOwningAgent {
        log.Info("Agent is not the case owner.")
        return nil, serviceerrors.ErrPermissionDenied
    }

    res, err := h.reqHandler.ContactV1CaseUnassign(ctx, a.CustomerID, id)
    if err != nil {
        log.Errorf("Could not unassign the case. err: %v", err)
        return nil, err
    }

    return res, nil
}
```

Decision on the "agent tries to unassign someone else's case" failure
mode -- task brief flags this as needing an explicit choice between
`ErrPermissionDenied` and `ErrNotFound`:

**Decision: `ErrPermissionDenied` (403), not `ErrNotFound` (404).**
Rationale: `ServiceAgentCaseGet` (same file) already establishes the
precedent that any authenticated agent of the tenant may `GET` any case
in that tenant ("No ownership check beyond tenant -- any authenticated
agent of the customer may view any case", per its own doc comment) --
i.e. case *existence and visibility* within a tenant is not something
this service_agents surface treats as sensitive/enumerable, unlike
cross-tenant case IDs (which do collapse into a single not-found/denied
response at the `caseGet` tenant-scope check). Since the agent can
already see the case (and its current owner) via `GET
/service_agents/contact_cases/{id}`, there is nothing to hide by
returning 403 instead of 404 when they attempt to unassign a
case they don't own -- 404 would be actively misleading (the case
plainly exists, the agent already fetched it) and provides no
anti-enumeration benefit here (contrast with `CaseAssign`'s owner-agent
lookup, which does collapse to `ErrNotFound` specifically because
*owner_id* validity/tenant membership is the sensitive fact being
protected against probing, not case existence). `ErrPermissionDenied` is
also what `ConversationUnassign`/`ServiceAgentConversationUnassign` both
return for the equivalent case.

### 4.3 `ServiceHandler` interface additions

`bin-api-manager/pkg/servicehandler/main.go`:

```go
// in the Case block, after CaseAssign (~line 551):
CaseUnassign(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmkase.Case, error)

// in the ServiceAgentCase block, after ServiceAgentCaseAssign (~line 1033):
ServiceAgentCaseUnassign(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmkase.Case, error)
```

`mock_main.go` is regenerated via `go generate ./...`
(`mockgen -package servicehandler -destination ./mock_main.go -source
main.go`) as part of implementation's verification workflow -- not hand
edited.

### 4.4 New server handlers: `bin-api-manager/server/contact_cases.go` and `service_agents_contact_cases.go`

No request body (mirrors `PostContactCasesIdClose`/conversation's
unassign, not `PostContactCasesIdAssign`, since there is no `owner_id` to
bind for unassign).

`bin-api-manager/server/contact_cases.go`, `PostContactCasesIdUnassign`:

```go
// PostContactCasesIdUnassign handles POST /contact_cases/{id}/unassign:
// Admin/Manager or the current owning agent unassigns the case's owner.
func (h *server) PostContactCasesIdUnassign(c *gin.Context, id openapi_types.UUID) {
    log := logrus.WithFields(logrus.Fields{
        "func":            "PostContactCasesIdUnassign",
        "request_address": c.ClientIP(),
        "id":              id,
    })

    a, ok := getAuthIdentity(c)
    if !ok {
        log.Errorf("Could not find auth identity.")
        abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
        return
    }
    log = log.WithField("customer_id", a.CustomerID)

    caseID := uuid.UUID(id)

    res, err := h.serviceHandler.CaseUnassign(c.Request.Context(), a, caseID)
    if err != nil {
        log.Errorf("Could not unassign case. err: %v", err)
        abortWithServiceError(c, err)
        return
    }

    c.JSON(200, res)
}
```

`bin-api-manager/server/service_agents_contact_cases.go` (confirmed: this
is where `ServiceAgentCaseAssign`/`ServiceAgentCaseClose`'s HTTP handlers
already live, per `grep -rl ServiceAgentCaseAssign bin-api-manager/server/`),
`PostServiceAgentsContactCasesIdUnassign`: identical shape, calling
`h.serviceHandler.ServiceAgentCaseUnassign(c.Request.Context(), a,
caseID)`.

Both handlers are generated-interface implementations
(`openapi_server.ServerInterface`); the exact method name
(`PostContactCasesIdUnassign` / `PostServiceAgentsContactCasesIdUnassign`)
is fixed by oapi-codegen from the `operationId`/path in the OpenAPI spec
once §5's yaml is added and `go generate` regenerates
`gens/openapi_server`.

## 5. bin-openapi-manager: new path files

Both are simple parameterless POSTs, following
`conversations/id_unassign.yaml` and
`service_agents/conversations_id_unassign.yaml` exactly (no
`requestBody`), with the Case-specific schema ref
(`ContactManagerCase`) and tag conventions from `id_assign.yaml`.

### 5.1 `bin-openapi-manager/openapi/paths/contact_cases/id_unassign.yaml`

```yaml
post:
  summary: Unassign the case
  description: |
    Removes the current owner from the case. Admin and manager callers may unassign any
    case. The owning agent may unassign themselves. Returns 403 if the caller is neither
    an admin/manager nor the current owner.
  tags:
    - Case
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
        format: uuid
        example: "550e8400-e29b-41d4-a716-446655440000"
      description: "The unique identifier of the case. Returned from the `GET /contact_cases` response."
  responses:
    '200':
      description: The case after unassignment.
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

### 5.2 `bin-openapi-manager/openapi/paths/service_agents/contact_cases_id_unassign.yaml`

Same body, `tags: [Service Agent]`, description path reference changed to
`` `GET /service_agents/contact_cases` `` response, and the 403 description
narrowed to "the caller is not the current owner" (service_agents has no
admin/manager bypass concept for this action -- see §4.2).

```yaml
post:
  summary: Unassign the case
  description: |
    Removes the current owner from the case. The owning agent may unassign themselves.
    Returns 403 if the caller is not the current owner of the case.
  tags:
    - Service Agent
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
        format: uuid
        example: "550e8400-e29b-41d4-a716-446655440000"
      description: "The unique identifier of the case. Returned from the `GET /service_agents/contact_cases` response."
  responses:
    '200':
      description: The case after unassignment.
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

### 5.3 Path registration in `bin-openapi-manager/openapi/openapi.yaml`

Add alongside the existing `assign` entries (after line 8679 and 8882
respectively, in this pre-implementation snapshot):

```yaml
  /contact_cases/{id}/unassign:
    $ref: './paths/contact_cases/id_unassign.yaml'
```

```yaml
  /service_agents/contact_cases/{id}/unassign:
    $ref: './paths/service_agents/contact_cases_id_unassign.yaml'
```

## 6. Authorization boundary: bin-api-manager only

Confirmed unchanged for this design: **all auth/authz decisions
(`hasPermission`, `IsAgent`/owner-identity comparison,
`ErrPermissionDenied`) happen exclusively in
`bin-api-manager/pkg/servicehandler`**, matching `ConversationUnassign`/
`ServiceAgentConversationUnassign`/`CaseAssign`/`ServiceAgentCaseAssign`.
`bin-contact-manager`'s `casehandler.Assign` performs zero authorization
(per its own doc comment: "No authorization decision is made here -- ...
any caller who reaches this function ... may assign to any (ownerType,
ownerID)") and this design does not add any. The RPC boundary
(`ContactV1CaseUnassign`) carries no auth context beyond `customerID`,
consistent with every other `ContactV1Case*` RPC client function.

## 7. Affected files

| File | Change |
|---|---|
| `bin-common-handler/pkg/requesthandler/main.go` | `RequestHandler` interface: new `ContactV1CaseUnassign` method signature (added at line ~952-953, next to `ContactV1CaseAssign`). Required because `bin-api-manager`'s `reqHandler` field is typed as `requesthandler.RequestHandler` (an interface) -- without this addition, `h.reqHandler.ContactV1CaseUnassign(...)` in the new servicehandler code will not compile. |
| `bin-common-handler/pkg/requesthandler/contact_cases.go` | New `ContactV1CaseUnassign` implementation |
| `bin-common-handler/pkg/requesthandler/mock_main.go` | Regenerate (confirmed exact path: `mock_main.go`, not a separate `mock_requesthandler.go`) |
| `bin-api-manager/pkg/servicehandler/case.go` | New `CaseUnassign` |
| `bin-api-manager/pkg/servicehandler/serviceagent_case.go` | New `ServiceAgentCaseUnassign` |
| `bin-api-manager/pkg/servicehandler/main.go` | `ServiceHandler` interface: 2 new methods |
| `bin-api-manager/pkg/servicehandler/mock_main.go` | Regenerate (`go generate ./...`) |
| `bin-api-manager/server/contact_cases.go` | New `PostContactCasesIdUnassign` |
| `bin-api-manager/server/service_agents_contact_cases.go` | New `PostServiceAgentsContactCasesIdUnassign` (confirmed existing file) |
| `bin-api-manager/gens/openapi_server/*` | Regenerate from updated spec |
| `bin-openapi-manager/openapi/paths/contact_cases/id_unassign.yaml` | New |
| `bin-openapi-manager/openapi/paths/service_agents/contact_cases_id_unassign.yaml` | New |
| `bin-openapi-manager/openapi/openapi.yaml` | 2 new path registrations |
| `bin-api-manager/docsdev/source/contact_case_overview.rst` | New "Unassigning a Case" section (mirror "Assigning a Case") |
| `bin-api-manager/docsdev/source/service_agent_case_overview.rst` | New row in the endpoint table + "Unassign a case" curl example under "Assigning and Closing" |
| `bin-api-manager/pkg/servicehandler/case_test.go` | New `Test_CaseUnassign` |
| `bin-api-manager/pkg/servicehandler/serviceagent_case_test.go` | New `Test_ServiceAgentCaseUnassign` |
| `bin-api-manager/server/contact_cases_test.go` | New `Test_PostContactCasesIdUnassign` |
| `bin-api-manager/server/service_agents_contact_cases_test.go` | New `Test_PostServiceAgentsContactCasesIdUnassign` |
| `bin-common-handler/pkg/requesthandler/*_test.go` | New test for `ContactV1CaseUnassign` |

No changes needed in: `bin-contact-manager` (any file), DB schema
(`bin-dbscheme-manager`), Conversation code.

## 8. Test plan

Follow `Test_ConversationUnassign` (table-driven, `gomock`) and
`Test_CaseAssign`'s existing fixtures/style exactly.

### 8.1 `bin-api-manager/pkg/servicehandler` -- `Test_CaseUnassign`

Cases (table-driven, mirroring `Test_ConversationUnassign`'s table plus
Case-specific tenant/direct-access checks already established in
`case_test.go`):

1. Admin unassigns a case owned by someone else -> success, `reqHandler`
   `ContactV1CaseUnassign` called once, result reflects cleared owner.
2. Manager unassigns a case owned by someone else -> success.
3. Owning agent self-unassigns -> success.
4. Non-owning agent (no admin/manager permission) attempts unassign ->
   `ErrPermissionDenied`, `ContactV1CaseUnassign` never called.
5. Agent with no permission at all attempts unassign on an unowned case
   -> `ErrPermissionDenied`.
6. Direct-access caller (`a.IsDirect()`) -> `ErrDirectAccessNotSupported`,
   no downstream calls.
7. Cross-tenant / nonexistent case id -> `caseGet` returns `ErrNotFound`
   (propagated), permission check never reached.
8. Unassigning an already-unowned case (idempotency) -> decide and
   document: this design treats it as a no-op success (same call shape,
   `owner_type`/`owner_id` already empty) as long as the caller is
   admin/manager; an agent cannot pass the owning-agent check against an
   unowned case (empty owner never equals `a.Agent.ID`), so a non-admin
   agent calling unassign on an unowned case correctly gets
   `ErrPermissionDenied`, not a false-success no-op. Add an explicit test
   for this to lock the behavior in.
9. Unassigning a **closed** case as admin -> `ErrCaseClosed`, `caseGet`
   is called but `ContactV1CaseUnassign` is never called (confirms the
   revised §4.1 decision to copy `CaseAssign`'s closed-case guard
   verbatim, protecting the `/continue` authorization invariant
   documented in `kase.go:73-76`). Add an explicit regression test
   asserting this, since a future contributor might otherwise assume
   unassign should be exempt from the guard the way it initially seemed
   to make sense (see §4.1's full rationale for why it does not).
10. **Non-admin, non-owning agent attempts unassign on a closed case ->
    `ErrPermissionDenied`, not `ErrCaseClosed`.** This is the ordering
    regression test called out in §4.1: since the permission check runs
    before the closed-status check (matching `CaseAssign`'s exact
    ordering), a caller with no legitimate access to the case at all
    must be rejected on permission grounds first, regardless of case
    status. This test would have caught an earlier draft's inverted
    check order, where the same call incorrectly returned `ErrCaseClosed`
    instead.

### 8.2 `bin-api-manager/pkg/servicehandler` -- `Test_ServiceAgentCaseUnassign`

Mirroring `Test_ServiceAgentCaseAssign`'s fixtures and
`Test_ServiceAgentConversationUnassign`'s cases:

1. Owning agent unassigns own case -> success.
2. Agent attempts to unassign a case owned by a **different** agent ->
   `ErrPermissionDenied` (per §4.2's decision), `ContactV1CaseUnassign`
   never called.
3. Agent attempts to unassign an unowned case -> `ErrPermissionDenied`
   (owner comparison fails against empty owner).
4. Agent without `PermissionAll` on their own customer -> `ErrPermissionDenied`
   before `caseGet` is even reached (matches `ServiceAgentCaseAssign`'s
   ordering).
5. Cross-tenant case id -> `caseGet` returns `ErrNotFound`.
6. **Closed** case owned by the caller -> `ErrCaseClosed`,
   `ContactV1CaseUnassign` never called (same revised §4.1 decision
   applies here; add the explicit regression test on this surface too).

### 8.3 `bin-api-manager/server` -- HTTP-level tests

`Test_PostContactCasesIdUnassign` (mirror `Test_PostContactCasesIdAssign`'s
structure minus request-body binding, since unassign has none):

1. Authenticated request -> 200, mock `serviceHandler.CaseUnassign`
   called with `(ctx, a, caseID)`, body echoes the returned case JSON.
2. No auth identity in context -> 401 `AUTHENTICATION_REQUIRED`.
3. `serviceHandler.CaseUnassign` returns `ErrPermissionDenied` ->
   `abortWithServiceError` maps to 403 (verify against existing
   `abortWithServiceError` mapping table, same as other endpoints -- no
   new mapping needed).
4. `serviceHandler.CaseUnassign` returns `ErrNotFound` -> 404.

`Test_PostServiceAgentsContactCasesIdUnassign`: same shape, calling
`ServiceAgentCaseUnassign`.

### 8.4 `bin-common-handler/pkg/requesthandler` -- `ContactV1CaseUnassign`

New unit test asserting the outbound wire payload is exactly
`{"customer_id": ..., "owner_type": "", "owner_id":
"00000000-0000-0000-0000-000000000000"}` posted to
`/v1/cases/{id}/assign` -- i.e. explicitly pin down that `owner_type` is
empty-string, not `"agent"`, to guard against the exact bug this design
calls out in §3 (reusing `ContactV1CaseAssign`'s hardcoded `"agent"`
would silently corrupt data).

## 9. RST documentation updates

- `contact_case_overview.rst`: add an "Unassigning a Case" section
  immediately after "Assigning a Case" (mirrors that section's
  structure: one-paragraph description + curl example). Note the
  service_agents cross-reference sentence at the end of "Assigning a
  Case" (line ~70) needs a matching sentence for unassign, and the
  paragraph at line ~154 (which currently lists only `assign` as the
  service_agents-exposed capability) needs `unassign` added.
- `service_agent_case_overview.rst`: add a row to the capability table
  (~line 82-83) for `/service_agents/contact_cases/{id}/unassign`; add an
  "Unassign a case" curl example under "Assigning and Closing" (~line
  120-128); update the closing cross-reference paragraph (~line 202) that
  currently lists `assign`'s top-level equivalent to also mention
  `unassign`'s.
- Standard `docsdev` rebuild required per root CLAUDE.md: `cd
  bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html
  source build`, then `git add -f bin-api-manager/docsdev/build/`.

## 10. Open questions for review

1. **Closed-case guard (§4.1, §8.1 item 9, §8.2 item 6) -- RESOLVED, not
   open.** Initially recommended omitting the guard as a "cleanup action"
   synthesis; a design-review round found this would silently break the
   `/continue` authorization invariant documented in `kase.go:73-76`
   ("Owner NEVER cleared by closing a Case ... load-bearing invariant for
   `/continue`'s authorization"). Revised to copy `CaseAssign`'s
   `ErrCaseClosed` guard verbatim into both `CaseUnassign` and
   `ServiceAgentCaseUnassign`. No longer a decision point requiring
   reviewer sign-off -- both functions now match `CaseAssign`'s existing
   pattern exactly.
2. **service_agents error code for "not my case" (§4.2).** Recommends
   `ErrPermissionDenied` (403) over `ErrNotFound` (404), based on
   `ServiceAgentCaseGet`'s existing "any agent may view any tenant case"
   precedent. Flagging since the task brief explicitly asked this to
   be decided, not assumed.
