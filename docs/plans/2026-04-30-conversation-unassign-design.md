# Conversation Unassign — Design

**Date:** 2026-04-30
**Status:** Approved

---

## 1. Motivation

The current `PUT /v1.0/conversations/<id>` endpoint carries a complex per-field permission gate
(`payloadIsExactlySelfUnassign`) to allow owning agents to self-unassign while blocking all other
agent-initiated updates. This design:

- Removes that complexity by making `PUT /conversations/<id>` strictly admin/manager-only.
- Introduces two explicit, intent-clear unassign endpoints — one on each API surface.
- Adds `PUT /service_agents/conversations/<id>` so admin/manager users can update conversations
  through the agent-facing surface.

---

## 2. Scope

### In scope

- `PUT /conversations/<id>` — tighten to admin/manager only (remove owning-agent carve-out).
- `POST /conversations/<id>/unassign` — new endpoint, admin/manager + owning agent.
- `PUT /service_agents/conversations/<id>` — new endpoint, admin/manager only.
- `POST /service_agents/conversations/<id>/unassign` — new endpoint, admin/manager + owning agent.

### Out of scope

- conversation-manager changes (no new RPC, no new DB columns).
- Any multi-owner or team-assignment features.
- Peer-to-peer hand-off between agents.

---

## 3. Breaking Change

**`PUT /conversations/<id>` no longer accepts requests from owning agents.**

Previously, an owning agent could call:
```json
PUT /v1.0/conversations/<id>
{"owner_id": "00000000-0000-0000-0000-000000000000"}
```
and receive a `200`. After this change, the same call returns `403 ErrPermissionDenied`.

Agents must migrate to `POST /v1.0/conversations/<id>/unassign` or
`POST /v1.0/service_agents/conversations/<id>/unassign`.

---

## 4. Permission Matrix

| Endpoint | Admin/Manager | Owning Agent | Other Agent | Non-agent / direct |
|---|---|---|---|---|
| `PUT /conversations/<id>` | ✓ | ✗ | ✗ | ✗ |
| `POST /conversations/<id>/unassign` | ✓ | ✓ | ✗ | ✗ |
| `PUT /service_agents/conversations/<id>` | ✓ | ✗ | ✗ | ErrAuthenticationRequired |
| `POST /service_agents/conversations/<id>/unassign` | ✓ | ✓ | ✗ | ErrAuthenticationRequired |

**Owning agent** = authenticated agent whose `agent.ID == conversation.owner_id`
AND `conversation.owner_type == "agent"`.

An **already-unassigned** conversation (`owner_id == uuid.Nil`):
- Agent calling unassign → `403` (nil ≠ agent ID, owning-agent check fails).
- Admin/manager calling unassign → `200` idempotent (nil→nil update is a no-op in effect, though
  conversation-manager will still fire a `conversation_updated` event).

---

## 5. API Surface

### 5.1 `POST /v1.0/conversations/{id}/unassign`

**New file:** `bin-openapi-manager/openapi/paths/conversations/id_unassign.yaml`

```yaml
post:
  summary: Unassign the conversation
  description: |
    Removes the current owner from the conversation. Admin and manager callers may unassign any
    conversation. The owning agent may unassign themselves. Returns 403 if the caller is neither
    an admin/manager nor the current owner.
  tags:
    - Conversation
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
        format: uuid
      description: The unique identifier of the conversation.
  responses:
    '200':
      description: Conversation after unassignment.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ConversationManagerConversation'
    '400': { $ref: '#/components/responses/BadRequest' }
    '401': { $ref: '#/components/responses/Unauthenticated' }
    '403': { $ref: '#/components/responses/PermissionDenied' }
    '404': { $ref: '#/components/responses/NotFound' }
    '500': { $ref: '#/components/responses/InternalError' }
```

Registered in `openapi.yaml` under:
```yaml
/v1.0/conversations/{id}/unassign:
  $ref: './paths/conversations/id_unassign.yaml'
```

### 5.2 `PUT /service_agents/conversations/{id}` (modify existing file)

**Modify:** `bin-openapi-manager/openapi/paths/service_agents/conversations_id.yaml`

Add a `put:` block alongside the existing `get:`. Request body schema is identical to
`PUT /conversations/<id>` — fields: `owner_id`, `owner_type`, `name`, `detail`. Both PUT request
bodies should reference the same component schema (or be declared identically) to prevent drift.

### 5.3 `POST /service_agents/conversations/{id}/unassign`

**New file:** `bin-openapi-manager/openapi/paths/service_agents/conversations_id_unassign.yaml`

Identical shape to `§5.1`. Tag: `Service Agent`.

Registered in `openapi.yaml` under:
```yaml
/v1.0/service_agents/conversations/{id}/unassign:
  $ref: './paths/service_agents/conversations_id_unassign.yaml'
```

---

## 6. api-manager — servicehandler changes

### 6.1 `conversation.go` — simplify `ConversationUpdate`, add `ConversationUnassign`

**Remove:**
- `payloadIsExactlySelfUnassign()` helper function (entire function deleted).
- The owning-agent branch inside `ConversationUpdate` (lines ~176–188 in the current file).

`ConversationUpdate` permission check becomes a single call:
```go
if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
    return nil, serviceerrors.ErrPermissionDenied
}
```

**Add `ConversationUnassign`:**

```go
func (h *serviceHandler) ConversationUnassign(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error) {
    if a.IsDirect() {
        return nil, serviceerrors.ErrDirectAccessNotSupported
    }

    c, err := h.conversationGet(ctx, conversationID)
    if err != nil {
        return nil, err
    }

    isAdminOrManager := h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager)
    isOwningAgent := a.IsAgent() && a.Agent != nil && c.OwnerType == string(commonidentity.OwnerTypeAgent) && c.OwnerID == a.Agent.ID

    if !isAdminOrManager && !isOwningAgent {
        return nil, serviceerrors.ErrPermissionDenied
    }

    tmp, err := h.reqHandler.ConversationV1ConversationUpdate(ctx, conversationID, map[cvconversation.Field]any{
        cvconversation.FieldOwnerID: uuid.Nil,
    })
    if err != nil {
        return nil, errors.Wrap(err, "could not unassign the conversation")
    }
    return tmp.ConvertWebhookMessage(), nil
}
```

### 6.2 `serviceagent_conversation.go` — add two new methods

**Add `ServiceAgentConversationUpdate`:**

```go
func (h *serviceHandler) ServiceAgentConversationUpdate(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID, fields map[cvconversation.Field]any) (*cvconversation.WebhookMessage, error) {
    if !a.IsAgent() {
        return nil, serviceerrors.ErrAuthenticationRequired
    }

    c, err := h.conversationGet(ctx, conversationID)
    if err != nil {
        return nil, err
    }

    if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
        return nil, serviceerrors.ErrPermissionDenied
    }

    tmp, err := h.reqHandler.ConversationV1ConversationUpdate(ctx, conversationID, fields)
    if err != nil {
        return nil, errors.Wrap(err, "could not update the conversation")
    }
    return tmp.ConvertWebhookMessage(), nil
}
```

Uses the same `structToFilteredMap` decoding pipeline in the server layer as `ConversationUpdate`
to preserve the absent-vs-present-with-zero-value distinction.

**Add `ServiceAgentConversationUnassign`:**

```go
func (h *serviceHandler) ServiceAgentConversationUnassign(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error) {
    if !a.IsAgent() {
        return nil, serviceerrors.ErrAuthenticationRequired
    }

    c, err := h.conversationGet(ctx, conversationID)
    if err != nil {
        return nil, err
    }

    isAdminOrManager := h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager)
    isOwningAgent := a.Agent != nil && c.OwnerType == string(commonidentity.OwnerTypeAgent) && c.OwnerID == a.Agent.ID

    if !isAdminOrManager && !isOwningAgent {
        return nil, serviceerrors.ErrPermissionDenied
    }

    tmp, err := h.reqHandler.ConversationV1ConversationUpdate(ctx, conversationID, map[cvconversation.Field]any{
        cvconversation.FieldOwnerID: uuid.Nil,
    })
    if err != nil {
        return nil, errors.Wrap(err, "could not unassign the conversation")
    }
    return tmp.ConvertWebhookMessage(), nil
}
```

### 6.3 `servicehandler/main.go` — interface additions

Add to the `ServiceHandler` interface:

```go
ConversationUnassign(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error)
ServiceAgentConversationUpdate(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID, fields map[cvconversation.Field]any) (*cvconversation.WebhookMessage, error)
ServiceAgentConversationUnassign(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error)
```

`mock_main.go` is regenerated automatically via `go generate ./...` in `bin-api-manager`.

---

## 7. api-manager — server layer changes

### 7.1 `server/conversations.go`

Add `PostConversationsIdUnassign(c *gin.Context, id string)`:
- Validate UUID from path param.
- Call `h.serviceHandler.ConversationUnassign(ctx, a, target)`.
- Return `200` with result.

### 7.2 `server/service_agents_conversations.go`

Add `PutServiceAgentsConversationsId(c *gin.Context, id string)`:
- Validate UUID.
- Decode JSON body via `PutServiceAgentsConversationsIdJSONBody` (generated type).
- Build `fields` map via `structToFilteredMap` (same pipeline as `PutConversationsId`).
- Call `h.serviceHandler.ServiceAgentConversationUpdate(ctx, a, target, fields)`.
- Return `200`.

Add `PostServiceAgentsConversationsIdUnassign(c *gin.Context, id string)`:
- Validate UUID.
- Call `h.serviceHandler.ServiceAgentConversationUnassign(ctx, a, target)`.
- Return `200`.

---

## 8. conversation-manager — No changes

All four new/modified endpoints call the existing `ConversationV1ConversationUpdate` RPC with
`{FieldOwnerID: uuid.Nil}`. No new RPC, no DB changes, no conversation-manager logic changes.

---

## 9. Testing

### 9.1 `servicehandler/conversation_test.go`

**Delete:** existing test cases for owning-agent self-unassign via `ConversationUpdate`.

**Add `ConversationUnassign` table-driven tests:**

| Caller | Conversation state | Expected |
|---|---|---|
| Admin | Assigned to agent A | 200, owner cleared |
| Manager | Assigned to agent A | 200, owner cleared |
| Agent A (owning) | Assigned to agent A | 200, owner cleared |
| Agent B (non-owning) | Assigned to agent A | ErrPermissionDenied |
| Agent A (owning) | Already unassigned (nil) | ErrPermissionDenied |
| Admin | Already unassigned (nil) | 200, idempotent |
| Direct caller | Any | ErrDirectAccessNotSupported |

### 9.2 `servicehandler/serviceagent_conversation_test.go`

**Add `ServiceAgentConversationUpdate` table-driven tests:**

| Caller | Expected |
|---|---|
| Admin | 200 |
| Manager | 200 |
| Agent (any) | ErrPermissionDenied |
| Non-agent | ErrAuthenticationRequired |

**Add `ServiceAgentConversationUnassign` table-driven tests:**

| Caller | Conversation state | Expected |
|---|---|---|
| Admin | Assigned | 200 |
| Manager | Assigned | 200 |
| Agent A (owning) | Assigned to A | 200 |
| Agent B (non-owning) | Assigned to A | ErrPermissionDenied |
| Agent A | Already unassigned | ErrPermissionDenied |
| Admin | Already unassigned | 200, idempotent |
| Non-agent | Any | ErrAuthenticationRequired |

---

## 10. Codegen ordering

```
1. Edit OpenAPI path files in bin-openapi-manager/
2. cd bin-openapi-manager && go generate ./...
3. cd bin-api-manager && go mod tidy && go mod vendor && go generate ./...
4. Implement server handlers and servicehandler methods
5. Run full verification: go test ./... && golangci-lint run -v --timeout 5m
```

---

## 11. RST docs

`bin-api-manager/docsdev/source/` — update conversation documentation to reflect:
- `PUT /conversations/<id>` is now admin/manager only.
- New `POST /conversations/<id>/unassign` endpoint (permission rules, response shape).
- New `POST /service_agents/conversations/<id>/unassign` endpoint.
- New `PUT /service_agents/conversations/<id>` endpoint.

After editing RST sources, clean-rebuild HTML:
```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
git add -f bin-api-manager/docsdev/build/
```
Commit RST sources and built HTML together.
