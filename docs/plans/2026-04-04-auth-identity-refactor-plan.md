# AuthIdentity Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace `*amagent.Agent` with a unified `*auth.AuthIdentity` across all servicehandler methods in bin-api-manager.

**Architecture:** Create a new `AuthIdentity` type in `models/auth/` that wraps agent, accesskey, and direct-scope auth into a single struct. Update all ~320 interface methods, ~60 implementation files, 62 server files, middleware, and ~62 test files. The `CustomerID` direct field design minimizes body changes.

**Tech Stack:** Go, gomock, gin, JWT (HS256)

**Design Doc:** `docs/plans/2026-04-04-auth-identity-refactor-design.md`

---

## Compilation Strategy

Nothing compiles until the interface, implementations, and callers all agree. The plan follows this order:
1. Create new type (compiles independently)
2. Update interface + all implementations together (must be atomic for compilation)
3. Regenerate mocks
4. Update all tests
5. Update middleware + server layer (callers)
6. Verify

Within phase 2, the interface change and ALL implementation files must be done before anything compiles. Use parallel agents per batch.

---

### Task 1: Create AuthIdentity Type

**Files:**
- Create: `bin-api-manager/models/auth/auth.go`
- Create: `bin-api-manager/models/auth/auth_test.go`

**Step 1: Create `models/auth/auth.go`**

```go
package auth

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	"github.com/gofrs/uuid"
)

// Type represents the authentication method used.
type Type string

const (
	TypeAgent     Type = "agent"
	TypeAccesskey Type = "accesskey"
	TypeDirect    Type = "direct"
)

// AuthIdentity is the unified authentication identity for all request types.
type AuthIdentity struct {
	Type        Type
	CustomerID  uuid.UUID // always set — populated from agent, accesskey, or direct scope

	Agent       *amagent.Agent         // non-nil for TypeAgent
	Accesskey   *csaccesskey.Accesskey // non-nil for TypeAccesskey
	DirectScope *DirectScope           // non-nil for TypeDirect
}

// DirectScope represents a resource-scoped JWT claim set.
type DirectScope struct {
	CustomerID           uuid.UUID `json:"customer_id"`
	ResourceType         string    `json:"resource_type"`
	ResourceID           uuid.UUID `json:"resource_id"`
	AllowedResourceTypes []string  `json:"allowed_resource_types"`
}

// IsAgent returns true if the identity was created from an agent JWT.
func (a *AuthIdentity) IsAgent() bool {
	return a.Type == TypeAgent
}

// IsAccesskey returns true if the identity was created from an accesskey.
func (a *AuthIdentity) IsAccesskey() bool {
	return a.Type == TypeAccesskey
}

// IsDirect returns true if the identity was created from a direct-scope JWT.
func (a *AuthIdentity) IsDirect() bool {
	return a.Type == TypeDirect
}

// HasPermission checks agent permissions.
// Agent: delegates to Agent.HasPermission().
// Accesskey: hardcoded CustomerAdmin (matches current behavior).
// Direct: returns false (use HasAllowedResourceType instead).
func (a *AuthIdentity) HasPermission(p amagent.Permission) bool {
	switch a.Type {
	case TypeAgent:
		if a.Agent == nil {
			return false
		}
		return a.Agent.HasPermission(p)
	case TypeAccesskey:
		return (amagent.PermissionCustomerAdmin & p) != 0
	default:
		return false
	}
}

// HasAllowedResourceType checks if the direct scope permits the given resource type.
func (a *AuthIdentity) HasAllowedResourceType(rt string) bool {
	if a.DirectScope == nil {
		return false
	}
	for _, t := range a.DirectScope.AllowedResourceTypes {
		if t == rt {
			return true
		}
	}
	return false
}

// AgentID returns the agent UUID. Returns uuid.Nil for non-agent auth.
func (a *AuthIdentity) AgentID() uuid.UUID {
	if a.Agent == nil {
		return uuid.Nil
	}
	return a.Agent.ID
}

// AccesskeyID returns the accesskey UUID. Returns uuid.Nil for non-accesskey auth.
func (a *AuthIdentity) AccesskeyID() uuid.UUID {
	if a.Accesskey == nil {
		return uuid.Nil
	}
	return a.Accesskey.ID
}

// AgentUsername returns the agent username. Returns "" for non-agent auth.
func (a *AuthIdentity) AgentUsername() string {
	if a.Agent == nil {
		return ""
	}
	return a.Agent.Username
}

// DisplayName returns a human-readable name for logging.
func (a *AuthIdentity) DisplayName() string {
	switch a.Type {
	case TypeAgent:
		if a.Agent != nil {
			return a.Agent.Username
		}
		return "agent"
	case TypeAccesskey:
		if a.Accesskey != nil {
			return "accesskey:" + a.Accesskey.Name
		}
		return "accesskey"
	case TypeDirect:
		if a.DirectScope != nil {
			return "direct:" + a.DirectScope.ResourceType
		}
		return "direct"
	default:
		return "unknown"
	}
}

// NewAgentIdentity constructs an AuthIdentity from an agent.
func NewAgentIdentity(agent *amagent.Agent) *AuthIdentity {
	return &AuthIdentity{
		Type:       TypeAgent,
		CustomerID: agent.CustomerID,
		Agent:      agent,
	}
}

// NewAccesskeyIdentity constructs an AuthIdentity from an accesskey.
func NewAccesskeyIdentity(ak *csaccesskey.Accesskey) *AuthIdentity {
	return &AuthIdentity{
		Type:       TypeAccesskey,
		CustomerID: ak.CustomerID,
		Accesskey:  ak,
	}
}

// NewDirectIdentity constructs an AuthIdentity from a direct scope.
func NewDirectIdentity(scope *DirectScope) *AuthIdentity {
	return &AuthIdentity{
		Type:        TypeDirect,
		CustomerID:  scope.CustomerID,
		DirectScope: scope,
	}
}
```

**Step 2: Create `models/auth/auth_test.go`**

Write unit tests for:
- `NewAgentIdentity` — verifies Type, CustomerID, Agent fields
- `NewAccesskeyIdentity` — verifies Type, CustomerID, Accesskey fields
- `NewDirectIdentity` — verifies Type, CustomerID, DirectScope fields
- `HasPermission` — agent delegates, accesskey returns CustomerAdmin, direct returns false
- `HasAllowedResourceType` — only works for direct
- `AgentID`, `AccesskeyID`, `AgentUsername`, `DisplayName` — returns correct values per type
- Nil-safety — all methods safe to call when inner pointer is nil

**Step 3: Verify type compiles**

Run: `cd bin-api-manager && go build ./models/auth/`

**Step 4: Run tests**

Run: `cd bin-api-manager && go test ./models/auth/ -v`

**Step 5: Commit**

```
Add AuthIdentity type for unified authentication

- bin-api-manager: Create models/auth/auth.go with AuthIdentity struct
- bin-api-manager: Support agent, accesskey, and direct auth types
- bin-api-manager: Add constructors, type checks, permission delegation
- bin-api-manager: Add unit tests for all methods and nil-safety
```

---

### Task 2: Update ServiceHandler Interface and hasPermission

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (lines 100-933)
- Modify: `bin-api-manager/pkg/servicehandler/etc.go`

**Step 1: Update imports in main.go**

Add import:
```go
"monorepo/bin-api-manager/models/auth"
```

Keep `amagent` import (needed for Permission constants and Agent type in other params).

**Step 2: Replace all `a *amagent.Agent` with `a *auth.AuthIdentity` in the interface**

Mechanical replacement across all ~320 method signatures. The `amagent` import stays for types like `amagent.Permission`, `amagent.RingMethod`, `amagent.Status` used in non-auth parameters.

Remove `AuthAccesskeyParse` from the interface (middleware will call `AccesskeyRawGetByToken` directly).

**Step 3: Update `etc.go`**

Replace imports and signature:
```go
import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"

	"github.com/gofrs/uuid"
)

func (h *serviceHandler) hasPermission(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID, permission amagent.Permission) bool {
	// Body unchanged — a.HasPermission() and a.CustomerID work on AuthIdentity
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

**Do NOT run build yet** — implementations don't match the new interface.

---

### Task 3: Update All ServiceHandler Implementations (Batch 1 — Simple CRUD)

**Files (15 files):** Each file gets the same mechanical changes:
- `pkg/servicehandler/billings.go`
- `pkg/servicehandler/billingaccount.go`
- `pkg/servicehandler/available_numbers.go`
- `pkg/servicehandler/call.go`
- `pkg/servicehandler/campaigns.go`
- `pkg/servicehandler/campaigncalls.go`
- `pkg/servicehandler/conference.go`
- `pkg/servicehandler/conferencecall.go`
- `pkg/servicehandler/contact.go`
- `pkg/servicehandler/conversation.go`
- `pkg/servicehandler/conversation_account.go`
- `pkg/servicehandler/conversation_message.go`
- `pkg/servicehandler/customer.go`
- `pkg/servicehandler/email.go`
- `pkg/servicehandler/extension.go`

**Per-file changes:**

1. **Add import:** `"monorepo/bin-api-manager/models/auth"`
2. **Replace all function signatures:** `a *amagent.Agent` → `a *auth.AuthIdentity`
3. **Add guard** at the top of each public method: `if a.IsDirect() { return ..., fmt.Errorf("direct access not supported") }`
4. **Replace `a.Username`** in log fields with `a.DisplayName()`
5. **`a.CustomerID`** — no change (direct field on AuthIdentity)
6. **`hasPermission(ctx, a, ...)`** — no change (signature already updated)

Example transformation for `billings.go`:
```go
// Before
func (h *serviceHandler) BillingGet(ctx context.Context, a *amagent.Agent, billingID uuid.UUID) (*bmbilling.WebhookMessage, error) {
    log := logrus.WithFields(logrus.Fields{
        "func":     "BillingGet",
        "username": a.Username,
    })
    // ...
}

// After
func (h *serviceHandler) BillingGet(ctx context.Context, a *auth.AuthIdentity, billingID uuid.UUID) (*bmbilling.WebhookMessage, error) {
    log := logrus.WithFields(logrus.Fields{
        "func":     "BillingGet",
        "username": a.DisplayName(),
    })

    if a.IsDirect() {
        return nil, fmt.Errorf("direct access not supported")
    }
    // ... rest unchanged ...
}
```

---

### Task 4: Update ServiceHandler Implementations (Batch 2 — More CRUD)

**Files (15 files):**
- `pkg/servicehandler/flow.go`
- `pkg/servicehandler/groupcall.go`
- `pkg/servicehandler/message.go`
- `pkg/servicehandler/numbers.go`
- `pkg/servicehandler/outdials.go`
- `pkg/servicehandler/outdialtargets.go`
- `pkg/servicehandler/outplans.go`
- `pkg/servicehandler/provider.go`
- `pkg/servicehandler/queue.go`
- `pkg/servicehandler/queuecall.go`
- `pkg/servicehandler/recording.go`
- `pkg/servicehandler/recordingfile.go`
- `pkg/servicehandler/route.go`
- `pkg/servicehandler/speaking.go`
- `pkg/servicehandler/storage_accounts.go`

**Same mechanical changes as Task 3.** Same per-file pattern.

---

### Task 5: Update ServiceHandler Implementations (Batch 3 — More CRUD + Special)

**Files (13 files):**
- `pkg/servicehandler/storage_file.go` — **special: `a.ID` in `storageFileCreate`** → use `a.AgentID()` for agent, `a.AccesskeyID()` for accesskey
- `pkg/servicehandler/tag.go`
- `pkg/servicehandler/team.go`
- `pkg/servicehandler/timeline.go`
- `pkg/servicehandler/timeline_sip.go`
- `pkg/servicehandler/transcribe.go`
- `pkg/servicehandler/transcript.go`
- `pkg/servicehandler/transfer.go`
- `pkg/servicehandler/trunk.go`
- `pkg/servicehandler/websock.go`
- `pkg/servicehandler/aggregated_events.go`
- `pkg/servicehandler/ai.go`
- `pkg/servicehandler/aisummary.go`
- `pkg/servicehandler/aimessage.go`
- `pkg/servicehandler/rag.go`

**Same mechanical changes as Task 3**, except:
- `storage_file.go`: For the `storageFileCreate` call, replace `a.ID` with a conditional:
  ```go
  ownerID := a.AgentID()
  if a.IsAccesskey() {
      ownerID = a.AccesskeyID()
  }
  ```

---

### Task 6: Update ServiceHandler Implementations (Batch 4 — a.ID Heavy Files)

**Files (6 files) — these use `a.ID` and need careful treatment:**
- `pkg/servicehandler/accesskeys.go` — `a.ID` used as `"agent_id"` filter in create
- `pkg/servicehandler/activeflows.go` — `a.ID` used as `"agent_id"` filter (4 places)
- `pkg/servicehandler/agent.go` — `a.ID` used in self-access check (3 places)
- `pkg/servicehandler/auth.go` — `a.ID` in debug log (1 place)

**Per-file specifics:**

**`accesskeys.go`:** In `AccesskeyCreate`, change `"agent_id": a.ID` to:
```go
// For accesskey-created accesskeys, store the accesskey ID as audit trail
ownerID := a.AgentID()
if a.IsAccesskey() {
    ownerID = a.AccesskeyID()
}
// ... use ownerID in the fields map
```

**`activeflows.go`:** In `ActiveflowGet`, `ActiveflowList`, `ActiveflowStop`, `ActiveflowDelete`, the `"agent_id"` filter:
```go
// For agent auth, filter by agent_id. For accesskey/direct, use customer_id only.
if a.IsAgent() {
    filters["agent_id"] = a.AgentID()
}
```

**`agent.go`:** Self-access checks like `a.ID != agentID`:
```go
// AgentID() returns uuid.Nil for non-agent, which never matches — falls through to hasPermission. Correct.
if a.AgentID() != agentID && !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
```

**`auth.go`:** Debug log — change `a.ID` to `a.AgentID()`. Remove `AuthAccesskeyParse` method (moved to middleware).

---

### Task 7: Update ServiceAgent Handlers (Agent-Only, a.ID Heavy)

**Files (11 files):**
- `pkg/servicehandler/serviceagent_agent.go`
- `pkg/servicehandler/serviceagent_call.go`
- `pkg/servicehandler/serviceagent_contact.go`
- `pkg/servicehandler/serviceagent_conversation.go`
- `pkg/servicehandler/serviceagent_conversationmessage.go`
- `pkg/servicehandler/serviceagent_customer.go`
- `pkg/servicehandler/serviceagent_extension.go`
- `pkg/servicehandler/serviceagent_file.go`
- `pkg/servicehandler/serviceagent_me.go`
- `pkg/servicehandler/serviceagent_tag.go`
- `pkg/servicehandler/serviceagent_talk.go`

**Per-file changes:**

1. Add import `"monorepo/bin-api-manager/models/auth"`
2. Replace all signatures: `a *amagent.Agent` → `a *auth.AuthIdentity`
3. Add guard: `if !a.IsAgent() { return ..., fmt.Errorf("agent authentication required") }`
4. Replace ALL `a.ID` → `a.AgentID()` (safe because IsAgent guard ensures non-nil)
5. Replace `a.Username` → `a.DisplayName()` in logs
6. `a.CustomerID` — unchanged

**Special: `serviceagent_talk.go`** also has `canAddParticipant` private helper:
```go
// Change signature
func (h *serviceHandler) canAddParticipant(ctx context.Context, a *auth.AuthIdentity, chatID uuid.UUID, ownerType string, ownerID uuid.UUID) bool {
    // Replace a.ID → a.AgentID(), a.CustomerID unchanged
}
```

---

### Task 8: Update Aicall Handler for Direct Access Support

**File:** `pkg/servicehandler/aicall.go`

This is the only handler that supports all three auth types.

```go
func (h *serviceHandler) AIcallCreate(
    ctx context.Context,
    a *auth.AuthIdentity,
    assistanceType amaicall.AssistanceType,
    assistanceID uuid.UUID,
    referenceType amaicall.ReferenceType,
    referenceID uuid.UUID,
) (*amaicall.WebhookMessage, error) {
    // ... resolve customerID from assistance (unchanged) ...

    // Authorization: support all three auth types
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

    // ... create aicall (unchanged) ...
}
```

Apply similar dual-path logic for `AIcallGet`, `AIcallGetsByCustomerID`, `AIcallDelete`.

---

### Task 9: Regenerate Mocks

**Step 1: Verify all implementations compile**

Run: `cd bin-api-manager && go build ./pkg/servicehandler/`

This MUST pass before proceeding. If it fails, fix any remaining signature mismatches.

**Step 2: Regenerate mock**

Run: `cd bin-api-manager && go generate ./pkg/servicehandler/`

This regenerates `pkg/servicehandler/mock_main.go` from `main.go`.

**Step 3: Verify mock compiles**

Run: `cd bin-api-manager && go build ./pkg/servicehandler/`

**Step 4: Commit all implementation changes**

```
Replace *amagent.Agent with *auth.AuthIdentity in servicehandler

- bin-api-manager: Update ServiceHandler interface (~320 methods)
- bin-api-manager: Update hasPermission helper signature
- bin-api-manager: Add IsDirect()/IsAgent() guards to all handlers
- bin-api-manager: Migrate a.ID to a.AgentID() in ~15 files
- bin-api-manager: Migrate a.Username to a.DisplayName() in ~41 files
- bin-api-manager: Add dual-path auth for aicall handlers (agent + direct)
- bin-api-manager: Remove AuthAccesskeyParse from interface
- bin-api-manager: Regenerate mock_main.go
```

---

### Task 10: Update All Test Files (Batch 1)

**Files (16 test files):**
- `pkg/servicehandler/billings_test.go`
- `pkg/servicehandler/billingaccount_test.go`
- `pkg/servicehandler/available_numbers_test.go`
- `pkg/servicehandler/call_test.go`
- `pkg/servicehandler/campaigns_test.go`
- `pkg/servicehandler/campaigncalls_test.go`
- `pkg/servicehandler/conference_test.go`
- `pkg/servicehandler/conferencecall_test.go`
- `pkg/servicehandler/contact_test.go`
- `pkg/servicehandler/conversation_test.go`
- `pkg/servicehandler/conversation_account_test.go`
- `pkg/servicehandler/conversation_message_test.go`
- `pkg/servicehandler/customer_test.go`
- `pkg/servicehandler/email_test.go`
- `pkg/servicehandler/extension_test.go`
- `pkg/servicehandler/flow_test.go`

**Per-file mechanical changes:**

1. Add import: `"monorepo/bin-api-manager/models/auth"`
2. In test table structs, change `agent *amagent.Agent` to `agent *auth.AuthIdentity`
3. Wrap agent construction:

```go
// Before
agent: &amagent.Agent{
    Identity: commonidentity.Identity{
        ID:         uuid.FromStringOrNil("..."),
        CustomerID: uuid.FromStringOrNil("..."),
    },
    Permission: amagent.PermissionCustomerAdmin,
},

// After
agent: auth.NewAgentIdentity(&amagent.Agent{
    Identity: commonidentity.Identity{
        ID:         uuid.FromStringOrNil("..."),
        CustomerID: uuid.FromStringOrNil("..."),
    },
    Permission: amagent.PermissionCustomerAdmin,
}),
```

4. Update mock EXPECT calls if they match on the agent parameter — the type changes from `*amagent.Agent` to `*auth.AuthIdentity`

---

### Task 11: Update All Test Files (Batch 2)

**Files (16 test files):**
- `pkg/servicehandler/groupcall_test.go`
- `pkg/servicehandler/message_test.go`
- `pkg/servicehandler/numbers_test.go`
- `pkg/servicehandler/outdials_test.go`
- `pkg/servicehandler/outdialtargets_test.go`
- `pkg/servicehandler/outplans_test.go`
- `pkg/servicehandler/provider_test.go`
- `pkg/servicehandler/queue_test.go`
- `pkg/servicehandler/queuecall_test.go`
- `pkg/servicehandler/recording_test.go`
- `pkg/servicehandler/recordingfile_test.go`
- `pkg/servicehandler/route_test.go`
- `pkg/servicehandler/speaking_test.go`
- `pkg/servicehandler/storage_accounts_test.go`
- `pkg/servicehandler/storage_account_test.go`
- `pkg/servicehandler/storage_file_test.go`

**Same mechanical changes as Task 10.**

---

### Task 12: Update All Test Files (Batch 3)

**Files (16 test files):**
- `pkg/servicehandler/tag_test.go`
- `pkg/servicehandler/timeline_test.go`
- `pkg/servicehandler/timeline_sip_test.go`
- `pkg/servicehandler/transcribe_test.go`
- `pkg/servicehandler/transcript_test.go`
- `pkg/servicehandler/transfer_test.go`
- `pkg/servicehandler/trunk_test.go`
- `pkg/servicehandler/websock_test.go`
- `pkg/servicehandler/aggregated_events_test.go`
- `pkg/servicehandler/ai_test.go`
- `pkg/servicehandler/aicall_test.go`
- `pkg/servicehandler/aimessage_test.go`
- `pkg/servicehandler/aisummary_test.go`
- `pkg/servicehandler/rag_test.go`
- `pkg/servicehandler/accesskeys_test.go`
- `pkg/servicehandler/activeflows_test.go`

**Same mechanical changes as Task 10.**

---

### Task 13: Update All Test Files (Batch 4 — ServiceAgent + Auth)

**Files (14 test files):**
- `pkg/servicehandler/agent_test.go`
- `pkg/servicehandler/auth_test.go`
- `pkg/servicehandler/auth_password_test.go`
- `pkg/servicehandler/serviceagent_agent_test.go`
- `pkg/servicehandler/serviceagent_call_test.go`
- `pkg/servicehandler/serviceagent_contact_test.go`
- `pkg/servicehandler/serviceagent_conversation_test.go`
- `pkg/servicehandler/serviceagent_conversationmessage_test.go`
- `pkg/servicehandler/serviceagent_customer_test.go`
- `pkg/servicehandler/serviceagent_extension_test.go`
- `pkg/servicehandler/serviceagent_file_test.go`
- `pkg/servicehandler/serviceagent_me_test.go`
- `pkg/servicehandler/serviceagent_tag_test.go`
- `pkg/servicehandler/serviceagent_talk_test.go`

**Same mechanical changes as Task 10.** Auth tests may additionally need `AuthAccesskeyParse` references removed.

**Step: Run all tests**

Run: `cd bin-api-manager && go test ./pkg/servicehandler/ -v -count=1`

Must all pass.

**Step: Commit**

```
Update all servicehandler tests for AuthIdentity

- bin-api-manager: Wrap agent fixtures in auth.NewAgentIdentity()
- bin-api-manager: Update mock EXPECT calls for new type
- bin-api-manager: Remove AuthAccesskeyParse test references
```

---

### Task 14: Update Middleware

**File:** `bin-api-manager/lib/middleware/authenticate.go`

**Step 1: Rewrite Authenticate function**

```go
package middleware

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"

    amagent "monorepo/bin-agent-manager/models/agent"
    "monorepo/bin-api-manager/models/auth"
    modelscommon "monorepo/bin-api-manager/models/common"
    "monorepo/bin-api-manager/pkg/servicehandler"
    cscustomer "monorepo/bin-customer-manager/models/customer"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)

func Authenticate() gin.HandlerFunc {
    return func(c *gin.Context) {
        log := logrus.WithFields(logrus.Fields{
            "func":            "Authenticate",
            "request_address": c.ClientIP,
        })

        authType, authString, err := getAuthString(c)
        if err != nil {
            c.AbortWithStatus(401)
            return
        }

        serviceHandler := c.MustGet(modelscommon.OBJServiceHandler).(servicehandler.ServiceHandler)

        var identity *auth.AuthIdentity

        switch authType {
        case authTypeToken:
            authData, err := serviceHandler.AuthJWTParse(c.Request.Context(), authString)
            if err != nil {
                log.Errorf("Could not parse JWT. err: %v", err)
                c.AbortWithStatus(401)
                return
            }
            identity, err = buildJWTIdentity(authData)
            if err != nil {
                log.Errorf("Could not build auth identity from JWT. err: %v", err)
                c.AbortWithStatus(401)
                return
            }

        case authTypeAccesskey:
            ak, err := serviceHandler.AccesskeyRawGetByToken(c.Request.Context(), authString)
            if err != nil {
                log.Errorf("Could not get accesskey. err: %v", err)
                c.AbortWithStatus(401)
                return
            }
            curTime := time.Now().UTC()
            if ak.TMExpire != nil && ak.TMExpire.Before(curTime) {
                log.Errorf("Accesskey expired.")
                c.AbortWithStatus(401)
                return
            }
            if ak.TMDelete != nil {
                log.Errorf("Accesskey deleted.")
                c.AbortWithStatus(401)
                return
            }
            identity = auth.NewAccesskeyIdentity(ak)

        default:
            c.AbortWithStatus(401)
            return
        }

        c.Set("auth_identity", identity)

        if isFrozenAccountBlocked(c, identity) {
            return
        }

        c.Next()
    }
}

func buildJWTIdentity(authData map[string]interface{}) (*auth.AuthIdentity, error) {
    tokenType, _ := authData["type"].(string)

    switch tokenType {
    case "direct":
        raw, err := json.Marshal(authData["direct"])
        if err != nil {
            return nil, fmt.Errorf("could not marshal direct scope: %w", err)
        }
        var scope auth.DirectScope
        if err := json.Unmarshal(raw, &scope); err != nil {
            return nil, fmt.Errorf("could not unmarshal direct scope: %w", err)
        }
        return auth.NewDirectIdentity(&scope), nil

    default: // "agent" or missing (backward compat)
        raw, err := json.Marshal(authData["agent"])
        if err != nil {
            return nil, fmt.Errorf("could not marshal agent: %w", err)
        }
        var a amagent.Agent
        if err := json.Unmarshal(raw, &a); err != nil {
            return nil, fmt.Errorf("could not unmarshal agent: %w", err)
        }
        return auth.NewAgentIdentity(&a), nil
    }
}

func isFrozenAccountBlocked(c *gin.Context, a *auth.AuthIdentity) bool {
    // Direct tokens skip frozen check — short-lived and resource-scoped
    if a.IsDirect() {
        return false
    }

    if a.HasPermission(amagent.PermissionProjectSuperAdmin) {
        return false
    }

    path := c.Request.URL.Path
    method := c.Request.Method
    if path == "/auth/unregister" && (method == http.MethodDelete || method == http.MethodPost) {
        return false
    }

    serviceHandler := c.MustGet(modelscommon.OBJServiceHandler).(servicehandler.ServiceHandler)
    cu, err := serviceHandler.CustomerGet(c.Request.Context(), a, a.CustomerID)
    if err != nil {
        return false
    }

    if cu.Status != cscustomer.StatusFrozen {
        return false
    }

    var deletionEffectiveAt *time.Time
    if cu.TMDeletionScheduled != nil {
        t := cu.TMDeletionScheduled.Add(30 * 24 * time.Hour)
        deletionEffectiveAt = &t
    }
    c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
        "error":                 "DELETION_SCHEDULED",
        "message":               "Account deletion scheduled",
        "deletion_scheduled_at": cu.TMDeletionScheduled,
        "deletion_effective_at": deletionEffectiveAt,
        "recovery_endpoint":     "DELETE /auth/unregister",
    })
    return true
}
```

**Step 2: Update `lib/middleware/authenticate_test.go`**

Update tests to verify:
- Agent JWT → `AuthIdentity{Type: TypeAgent}`
- Accesskey → `AuthIdentity{Type: TypeAccesskey}`
- Direct JWT → `AuthIdentity{Type: TypeDirect}`
- Backward compat (no type field) → treated as agent
- Frozen check skipped for direct tokens

**Step 3: Commit**

```
Rewrite auth middleware to produce AuthIdentity

- bin-api-manager: Dispatch on JWT type (agent/direct) and accesskey
- bin-api-manager: Remove fake-agent construction for accesskeys
- bin-api-manager: Set "auth_identity" in gin context
- bin-api-manager: Skip frozen check for direct tokens
```

---

### Task 15: Create Server Helper and Update Server Files (Batch 1)

**Files:**
- Create: `bin-api-manager/server/auth_helper.go`
- Modify: 20 files in `server/` (a-c alphabetically)

**Step 1: Create `server/auth_helper.go`**

```go
package server

import (
    "monorepo/bin-api-manager/models/auth"

    "github.com/gin-gonic/gin"
)

// getAuthIdentity extracts the AuthIdentity from the gin context.
func getAuthIdentity(c *gin.Context) (*auth.AuthIdentity, bool) {
    tmp, exists := c.Get("auth_identity")
    if !exists {
        return nil, false
    }
    a, ok := tmp.(*auth.AuthIdentity)
    return a, ok
}
```

**Step 2: Update 20 server files**

For each file, replace:
```go
// Before
import (
    amagent "monorepo/bin-agent-manager/models/agent"
    // ...
)

tmp, exists := c.Get("agent")
if !exists {
    log.Errorf("Could not find agent info.")
    c.AbortWithStatus(400)
    return
}
a := tmp.(amagent.Agent)
log = log.WithFields(logrus.Fields{
    "agent": a,
})
// ...
h.serviceHandler.Method(c.Request.Context(), &a, ...)
```

With:
```go
// After
import (
    // remove amagent import if no longer needed (some files use amagent types for params)
    // ...
)

a, ok := getAuthIdentity(c)
if !ok {
    log.Errorf("Could not find auth identity.")
    c.AbortWithStatus(400)
    return
}
log = log.WithFields(logrus.Fields{
    "auth": a,
})
// ...
h.serviceHandler.Method(c.Request.Context(), a, ...)
//                                           ^ already pointer, no &
```

Also replace `a.Username` → `a.DisplayName()` and `a.CustomerID` → `a.CustomerID` (unchanged).

Files for this batch:
`accesskeys.go`, `activeflows.go`, `agents.go`, `aggregated_events.go`, `aicalls.go`, `aimessages.go`, `ais.go`, `aisummaries.go`, `available_numbers.go`, `billing_account.go`, `billing_accounts.go`, `billings.go`, `calls.go`, `campaigncalls.go`, `campaigns.go`, `conferencecalls.go`, `conferences.go`, `contacts.go`, `conversation_accounts.go`, `conversations.go`

---

### Task 16: Update Server Files (Batch 2)

**Files (20 files):**
`customer.go`, `customers.go`, `emails.go`, `extensions.go`, `flows.go`, `groupcalls.go`, `me.go`, `messages.go`, `numbers.go`, `outdials.go`, `outplans.go`, `providers.go`, `queuecalls.go`, `queues.go`, `rags.go`, `recordingfiles.go`, `recordings.go`, `routes.go`, `service_agents_agents.go`, `service_agents_calls.go`

**Same mechanical changes as Task 15 Step 2.**

---

### Task 17: Update Server Files (Batch 3)

**Files (remaining ~22 files):**
`service_agents_contacts.go`, `service_agents_conversations.go`, `service_agents_customer.go`, `service_agents_extensions.go`, `service_agents_files.go`, `service_agents_me.go`, `service_agents_tags.go`, `service_agents_talk.go`, `service_agents_ws.go`, `speakings.go`, `storage_account.go`, `storage_accounts.go`, `storage_files.go`, `tags.go`, `teams.go`, `timelines.go`, `timelines_sip.go`, `transcribes.go`, `transcripts.go`, `transfers.go`, `trunks.go`, `ws.go`

**Same mechanical changes as Task 15 Step 2.**

---

### Task 18: Update lib/service Files

**Files:**
- Modify: `bin-api-manager/lib/service/unregister.go`

**Changes:**
```go
// Before
tmpAgent, exists := c.Get("agent")
if !exists { ... }
a := tmpAgent.(amagent.Agent)

serviceHandler.CustomerSelfFreeze(c.Request.Context(), &a)

// After — import server package's helper or duplicate extraction
tmp, exists := c.Get("auth_identity")
if !exists { ... }
a, ok := tmp.(*auth.AuthIdentity)
if !ok { ... }

serviceHandler.CustomerSelfFreeze(c.Request.Context(), a)
```

Note: `lib/service/` cannot import `server/` package, so duplicate the extraction inline or move `getAuthIdentity` to a shared location (e.g., `models/auth/gin.go`).

---

### Task 19: Full Verification

**Step 1: Build**

Run: `cd bin-api-manager && go build ./...`

**Step 2: Tests**

Run: `cd bin-api-manager && go test ./... -count=1`

**Step 3: Lint**

Run: `cd bin-api-manager && golangci-lint run -v --timeout 5m`

**Step 4: Full verification workflow**

Run: `cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`

**Step 5: Final commit**

```
Update middleware, server layer, and lib/service for AuthIdentity

- bin-api-manager: Create server/auth_helper.go with getAuthIdentity()
- bin-api-manager: Update all 62 server files to extract AuthIdentity
- bin-api-manager: Update lib/service/unregister.go
- bin-api-manager: All tests passing, lint clean
```

---

## Task Dependency Graph

```
Task 1 (AuthIdentity type)
    ↓
Task 2 (Interface + hasPermission)
    ↓
Tasks 3-8 (All implementations — can be parallelized)
    ↓
Task 9 (Regenerate mocks — must wait for all impls)
    ↓
Tasks 10-13 (All tests — can be parallelized)
    ↓
Task 14 (Middleware)
Tasks 15-17 (Server files — can be parallelized)
Task 18 (lib/service)
    ↓
Task 19 (Full verification)
```

**Parallelizable batches:**
- Tasks 3, 4, 5, 6, 7, 8 — all implementation batches
- Tasks 10, 11, 12, 13 — all test batches
- Tasks 15, 16, 17 — all server file batches
