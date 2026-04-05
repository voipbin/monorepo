# AuthIdentity Refactor Design

**Date:** 2026-04-04
**Status:** Approved
**Prerequisite:** Resource-Scoped JWT Design (2026-04-04-resource-scoped-jwt-design.md)

## Problem

All servicehandler public methods accept `*amagent.Agent` as the auth parameter. This worked when agent login was the only auth path, but now we have three:

1. **Agent JWT** — `POST /auth/login`, full agent claims
2. **Accesskey** — raw token, resolves to customer-scoped access with hardcoded `CustomerAdmin`
3. **Direct JWT** — `POST /boot`, resource-scoped, lean claims

Accesskeys currently create a **fake `amagent.Agent`** (stuffing the accesskey ID into the agent ID field, hardcoding permission). This is a semantic lie that obscures the actual auth type in handler logic and logging.

## Goal

Replace `*amagent.Agent` with a unified `*auth.AuthIdentity` type across all servicehandler methods so there is a single, typed auth interface. Each handler can then dispatch on the auth type explicitly.

## AuthIdentity Type

Location: `bin-api-manager/models/auth/auth.go`

```go
type Type string

const (
    TypeAgent     Type = "agent"
    TypeAccesskey Type = "accesskey"
    TypeDirect    Type = "direct"
)

type AuthIdentity struct {
    Type        Type
    CustomerID  uuid.UUID              // always set — from agent, accesskey, or direct scope

    Agent       *amagent.Agent         // non-nil for TypeAgent
    Accesskey   *csaccesskey.Accesskey // non-nil for TypeAccesskey
    DirectScope *DirectScope           // non-nil for TypeDirect
}

type DirectScope struct {
    CustomerID           uuid.UUID
    ResourceType         string
    ResourceID           uuid.UUID
    AllowedResourceTypes []string
}
```

### Design Decision: CustomerID as Direct Field

`CustomerID` is a direct field on AuthIdentity (not a method). This means the ~434 occurrences of `a.CustomerID` throughout servicehandler bodies require **no change**. The constructor populates it from whichever auth source applies.

### Methods

| Method | Agent | Accesskey | Direct |
|---|---|---|---|
| `IsAgent()` | true | false | false |
| `IsAccesskey()` | false | true | false |
| `IsDirect()` | false | false | true |
| `HasPermission(p)` | delegates to `Agent.HasPermission(p)` | hardcoded `CustomerAdmin & p` | false |
| `HasAllowedResourceType(rt)` | false | false | checks `DirectScope.AllowedResourceTypes` |
| `AgentID()` | `Agent.ID` | `uuid.Nil` | `uuid.Nil` |
| `AccesskeyID()` | `uuid.Nil` | `Accesskey.ID` | `uuid.Nil` |
| `AgentUsername()` | `Agent.Username` | `""` | `""` |
| `DisplayName()` | `Agent.Username` | `"accesskey:<name>"` | `"direct:<resource_type>"` |

### Constructors

```go
func NewAgentIdentity(agent *amagent.Agent) *AuthIdentity
func NewAccesskeyIdentity(ak *csaccesskey.Accesskey) *AuthIdentity
func NewDirectIdentity(scope *DirectScope) *AuthIdentity
```

## Middleware Changes

File: `bin-api-manager/lib/middleware/authenticate.go`

### Current Flow

```
getAuthString() → token or accesskey
  → token:     AuthJWTParse() → map with "agent" key → unmarshal Agent → c.Set("agent", a)
  → accesskey: AuthAccesskeyParse() → map with FAKE "agent" → unmarshal Agent → c.Set("agent", a)
```

### New Flow

```
getAuthString() → token or accesskey
  → token:     AuthJWTParse() → check "type" field:
                  "agent"/missing → unmarshal Agent → NewAgentIdentity()
                  "direct"        → unmarshal DirectScope → NewDirectIdentity()
  → accesskey: AccesskeyRawGetByToken() → validate expiry → NewAccesskeyIdentity()
→ c.Set("auth_identity", identity)
```

Key changes:
- `AuthAccesskeyParse` is removed from the ServiceHandler interface. Accesskey resolution uses `AccesskeyRawGetByToken` directly, with expiry/deletion validation in the middleware.
- No more fake agent construction.
- New `buildJWTIdentity(authData)` helper dispatches on `"type"` field.

### Frozen Account Check

- **Agent:** checks frozen status (unchanged)
- **Accesskey:** checks frozen status (unchanged — `HasPermission(CustomerAdmin)` returns true, `CustomerGet` succeeds)
- **Direct:** skips entirely (short-lived, resource-scoped)

## ServiceHandler Interface

File: `bin-api-manager/pkg/servicehandler/main.go`

All ~320 method signatures change:
```go
// Before
BillingGet(ctx context.Context, a *amagent.Agent, billingID uuid.UUID) (*bmbilling.WebhookMessage, error)

// After
BillingGet(ctx context.Context, a *auth.AuthIdentity, billingID uuid.UUID) (*bmbilling.WebhookMessage, error)
```

`AuthAccesskeyParse` is removed from the interface. `amagent` import remains for permission constants.

## hasPermission Helper

File: `bin-api-manager/pkg/servicehandler/etc.go`

```go
// Signature changes, body unchanged
func (h *serviceHandler) hasPermission(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID, permission amagent.Permission) bool {
    if a.HasPermission(amagent.PermissionProjectSuperAdmin) {
        return true
    }
    if a.CustomerID != customerID {
        return false
    }
    if a.HasPermission(permission) {
        return true
    }
    return false
}
```

Works correctly for all three types:
- Agent: delegates to `Agent.HasPermission()`
- Accesskey: hardcoded `CustomerAdmin` check
- Direct: returns false — handlers must use separate direct-access logic

## Server Layer

62 files in `server/` + 2 in `lib/service/`.

### New Helper

File: `server/helper.go` (new)

```go
func getAuthIdentity(c *gin.Context) (*auth.AuthIdentity, bool) {
    tmp, exists := c.Get("auth_identity")
    if !exists {
        return nil, false
    }
    a, ok := tmp.(*auth.AuthIdentity)
    return a, ok
}
```

### Extraction Pattern Change

```go
// Before (repeated in 62 files)
tmp, exists := c.Get("agent")
if !exists { c.AbortWithStatus(400); return }
a := tmp.(amagent.Agent)

// After
a, ok := getAuthIdentity(c)
if !ok { c.AbortWithStatus(400); return }
```

`a.CustomerID` in filter building (e.g., contacts.go) — unchanged.
`a.Username` in logging — changes to `a.AgentUsername()` or `a.DisplayName()`.

## Handler Guard Patterns

### Most CRUD (~250 methods) — Agent + Accesskey, Block Direct

```go
func (h *serviceHandler) BillingGet(ctx context.Context, a *auth.AuthIdentity, billingID uuid.UUID) (...) {
    if a.IsDirect() {
        return nil, fmt.Errorf("direct access not supported")
    }
    // ... unchanged ...
}
```

### ServiceAgent* (~30 methods) — Agent Only

```go
func (h *serviceHandler) ServiceAgentTalkChatCreate(ctx context.Context, a *auth.AuthIdentity, ...) (...) {
    if !a.IsAgent() {
        return nil, fmt.Errorf("agent authentication required")
    }
    // a.AgentID() for owner operations, a.CustomerID unchanged
}
```

### Aicall (4 methods) — All Three Types

```go
func (h *serviceHandler) AIcallCreate(ctx context.Context, a *auth.AuthIdentity, ...) (...) {
    switch {
    case a.IsAgent() || a.IsAccesskey():
        if !h.hasPermission(ctx, a, customerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
            return nil, fmt.Errorf("user has no permission")
        }
    case a.IsDirect():
        if !a.HasAllowedResourceType("aicall") {
            return nil, fmt.Errorf("resource type not allowed")
        }
        if a.DirectScope.ResourceID != assistanceID {
            return nil, fmt.Errorf("resource not in scope")
        }
    }
}
```

## a.ID → a.AgentID() Migration (43 occurrences, ~15 files)

All `a.ID` references change to `a.AgentID()`. Affected files and decisions:

| File | Usage | Decision |
|---|---|---|
| `serviceagent_*.go` (~30 spots) | Owner checks, RPC calls | Safe — `IsAgent()` guard ensures non-nil |
| `activeflows.go` (4 spots) | `"agent_id": a.ID` filter | For non-agent auth, filter by `customer_id` only |
| `accesskeys.go` (1 spot) | `"agent_id": a.ID` in create | Store `AccesskeyID()` for accesskey auth |
| `agent.go` (3 spots) | Self-access check `a.ID != agentID` | `AgentID()` returns `uuid.Nil` for non-agent, falls through to `hasPermission` — correct |
| `storage_file.go` / `serviceagent_file.go` (2 spots) | File owner | Store `AccesskeyID()` for accesskey auth |
| `auth.go` (1 spot) | Debug logging | Use `AgentID()` — logs `uuid.Nil` for non-agent, acceptable |

## a.Username Migration (157 servicehandler + 15 server occurrences)

All logging-only. Change to `a.DisplayName()` for better observability across all auth types.

## Private Helpers That Also Change

| Helper | File | Change |
|---|---|---|
| `hasPermission` | `etc.go` | Signature only, body unchanged |
| `canAddParticipant` | `serviceagent_talk.go` | Signature + `a.ID` → `a.AgentID()` |

## Files Touched

| Layer | Count | Nature |
|---|---|---|
| `models/auth/auth.go` | 1 new | AuthIdentity type |
| `lib/middleware/authenticate.go` | 1 | Rewrite auth dispatch |
| `server/helper.go` | 1 new | `getAuthIdentity()` |
| `server/*.go` | 62 | Extraction pattern |
| `lib/service/unregister.go` | 1 | Extraction pattern |
| `pkg/servicehandler/main.go` | 1 | Interface signatures |
| `pkg/servicehandler/etc.go` | 1 | `hasPermission` signature |
| `pkg/servicehandler/*.go` | ~59 | Signatures + guards + field migration |
| `pkg/servicehandler/mock_main.go` | 1 | Regenerated |
| `pkg/servicehandler/*_test.go` | all | Wrap agents in `NewAgentIdentity()` |
| **Total** | **~130** | Mostly mechanical |

## Backward Compatibility

- Existing JWTs without `"type"` field are treated as `"agent"` (handled in `buildJWTIdentity`)
- Accesskey auth behavior preserved: same `CustomerAdmin` permission, same customer scope
- No API-facing changes — this is internal refactoring only
