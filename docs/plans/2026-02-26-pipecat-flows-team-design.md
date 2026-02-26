# Design: Pipecat Flows — Team Resource in ai-manager

**Date:** 2026-02-26
**Branch:** NOJIRA-add-pipecat-flows-team-support
**Status:** Draft

## Problem Statement

VoIPbin's ai-manager currently uses a monolithic approach for AI conversations — each session gets a single system prompt (`InitPrompt`) and one tool set (`ToolNames`). The LLM sees everything at once for the entire conversation.

Pipecat Flows demonstrates that breaking conversations into focused steps (nodes), each with specific instructions and limited tools, dramatically improves LLM accuracy and reduces hallucinations. We want to bring this capability to VoIPbin.

## Approach

Introduce a **Team** resource in ai-manager that composes existing AI entities into a directed graph. Each node in the graph is a **Member** backed by a reusable AI config, with **Transitions** defining how the LLM moves between members via function calling.

The flow execution logic lives on the Python side (pipecat-manager), which translates the Team config into Pipecat Flows' `NodeConfig` format at runtime. The Go side (ai-manager) stores, validates, and serves the configuration.

### Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Flow execution location | Pipecat (Python) | Pipecat Flows is a Python library; avoids reimplementation in Go |
| Flow config storage | Separate Team resource | Flows are complex enough for own lifecycle; keeps AI model clean |
| Resource naming | Team / Member | Each node is a "team member" with a specialized role |
| Member AI reference | Reference by ID (not snapshot) | Design-time reference; snapshot happens at runtime (existing AIcall pattern) |
| Members storage | JSON column in Team table | Team is managed as a single document; matches Pipecat Flows model |
| Entry point | Explicit `StartMemberID` on Team | Unambiguous, no magic ordering |
| pre_actions / post_actions | Deferred | Existing tools + flow-manager cover current needs |
| Context strategy | Default APPEND, no config | Simplest; full context is safest default; add later if needed |
| Transition parameters | Deferred | Simple routing doesn't need parameters; add later if needed |

## Data Model

### Team

```go
// models/team/main.go
type Team struct {
    identity.Identity                                            // ID, CustomerID (uuid)

    Name          string    `json:"name,omitempty" db:"name"`
    Detail        string    `json:"detail,omitempty" db:"detail"`
    StartMemberID uuid.UUID `json:"start_member_id,omitempty" db:"start_member_id,uuid"`
    Members       []Member  `json:"members,omitempty" db:"members,json"`

    TMCreate *time.Time `json:"tm_create" db:"tm_create"`
    TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
    TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
```

### Member

```go
// models/team/member.go
type Member struct {
    ID          uuid.UUID    `json:"id,omitempty"`
    Name        string       `json:"name,omitempty"`
    AIID        uuid.UUID    `json:"ai_id,omitempty"`
    Transitions []Transition `json:"transitions,omitempty"`
}
```

### Transition

```go
// models/team/transition.go
type Transition struct {
    FunctionName string    `json:"function_name,omitempty"`
    Description  string    `json:"description,omitempty"`
    NextMemberID uuid.UUID `json:"next_member_id,omitempty"`
}
```

### Field Constants

```go
// models/team/field.go
type Field string

const (
    FieldID            Field = "id"
    FieldCustomerID    Field = "customer_id"
    FieldName          Field = "name"
    FieldDetail        Field = "detail"
    FieldStartMemberID Field = "start_member_id"
    FieldMembers       Field = "members"
    FieldTMCreate      Field = "tm_create"
    FieldTMUpdate      Field = "tm_update"
    FieldTMDelete      Field = "tm_delete"
    FieldDeleted       Field = "deleted"
)
```

### Filter Struct

```go
// models/team/filters.go
type FieldStruct struct {
    CustomerID uuid.UUID `filter:"customer_id"`
    Name       string    `filter:"name"`
    Deleted    bool      `filter:"deleted"`
}
```

### WebhookMessage

```go
// models/team/webhook.go
type WebhookMessage struct {
    commonidentity.Identity

    Name          string    `json:"name,omitempty"`
    Detail        string    `json:"detail,omitempty"`
    StartMemberID uuid.UUID `json:"start_member_id,omitempty"`
    Members       []Member  `json:"members,omitempty"`

    TMCreate *time.Time `json:"tm_create"`
    TMUpdate *time.Time `json:"tm_update"`
    TMDelete *time.Time `json:"tm_delete"`
}

func (h *Team) ConvertWebhookMessage() *WebhookMessage {
    return &WebhookMessage{
        Identity:      h.Identity,
        Name:          h.Name,
        Detail:        h.Detail,
        StartMemberID: h.StartMemberID,
        Members:       h.Members,
        TMCreate:      h.TMCreate,
        TMUpdate:      h.TMUpdate,
        TMDelete:      h.TMDelete,
    }
}
```

### Event Types

```go
// models/team/event.go
const (
    EventTypeCreated string = "team_created"
    EventTypeUpdated string = "team_updated"
    EventTypeDeleted string = "team_deleted"
)
```

## Database

### Table: ai_teams

| Column | Type | Notes |
|--------|------|-------|
| id | binary(16) | UUID, PK |
| customer_id | binary(16) | UUID |
| name | varchar(255) | |
| detail | text | |
| start_member_id | binary(16) | UUID, references member in JSON |
| members | json | Array of Member structs |
| tm_create | datetime(6) | |
| tm_update | datetime(6) | |
| tm_delete | datetime(6) | Soft delete |

### Column addition: ai_aicalls

| Column | Type | Notes |
|--------|------|-------|
| team_id | binary(16) | UUID, nullable, references ai_teams.id |

Both schema changes created as a single Alembic migration in `bin-dbscheme-manager` (NOT executed by AI).

## API Endpoints

| Method | URI | Purpose |
|--------|-----|---------|
| GET | /v1/teams? | List teams (paginated, filtered) |
| POST | /v1/teams | Create team |
| GET | /v1/teams/{id} | Get team |
| PUT | /v1/teams/{id} | Update team |
| DELETE | /v1/teams/{id} | Soft delete team |

Standard CRUD following existing ai-manager patterns.

## Validation Rules

Enforced on Create and Update:

1. **StartMemberID must not be uuid.Nil** — reject zero-value UUID
2. **StartMemberID must exist in Members** — reject if the start member ID doesn't match any member in the array
3. **Member IDs must be unique** — reject duplicate UUIDs in the members array
4. **Member.ID must not be uuid.Nil** — reject zero-value member IDs
5. **Member.AIID must not be uuid.Nil** — reject zero-value AI references
6. **Member.AIID must reference an existing AI** — verify each AI exists via dbhandler
7. **Transition.NextMemberID must reference an existing member** — reject dangling references
8. **Transition.FunctionName must not collide with reserved tool names** — check programmatically against `tool.AllToolNames` (not a hardcoded list) to stay in sync as new tools are added
9. **Transition.FunctionName must be unique within a member** — reject duplicate function names on the same member
10. **Members list must not be empty** — reject teams with no members
11. **Member.Name must not be empty** — each member needs a name for debugging/logging

Graph reachability is NOT validated — circular references are valid, unreachable members are harmless.

**Note on cross-member function names:** The same `FunctionName` MAY appear on different members (e.g., both billing and support members can have `transfer_to_greeter`). This is intentionally allowed — Pipecat Flows registers functions per-node, so each member has its own isolated function set. Only within a single member must function names be unique.

## Runtime Flow

### AIcall Start Strategy

When a team-based session starts, `aicallHandler` gains a new method:

```go
func (h *aicallHandler) StartWithTeam(
    ctx context.Context,
    teamID uuid.UUID,
    activeflowID uuid.UUID,
    referenceType aicall.ReferenceType,
    referenceID uuid.UUID,
    gender aicall.Gender,
    language string,
) (*aicall.AIcall, error)
```

**Dependencies:** `aicallHandler` gains two new dependencies (added to the struct and `NewAIcallHandler` constructor):
- `teamHandler teamhandler.TeamHandler` — used to fetch the Team by ID
- `toolHandler toolhandler.ToolHandler` — used to resolve tool definitions for each member via `GetByNames()`

The existing `aiHandler` dependency handles AI config resolution.

This method:
1. Fetches the Team by ID via `teamHandler.Get()`
2. Finds the start member via `Team.StartMemberID`
3. Uses the start member's `AIID` to fetch the AI config via `aiHandler.Get()`
4. Creates the AIcall record via the existing `Create()` method (which gains an optional `teamID uuid.UUID` parameter — existing callers pass `uuid.Nil`):
   - `AIID` = start member's AI ID
   - `TeamID` = the team ID
   - `AIEngineModel`, `AIEngineData`, `AITTSType`, etc. = snapshotted from start member's AI (existing pattern)
5. Resolves the full team config into `ResolvedTeam` (see below) — for each member, fetches AI via `aiHandler.Get()`, builds `ResolvedAI` (stripping `EngineKey`), resolves tools via `toolHandler.GetByNames()`
6. Passes the resolved config to pipecat-manager via `PipecatV1PipecatcallStart`

**Error handling at resolution time:** If any member's AI has been deleted between team creation and session start, `StartWithTeam` returns an error. The caller (flow-manager) handles this as a failed service start.

**Reference type support:** `StartWithTeam` supports all existing reference types (`ReferenceTypeCall`, `ReferenceTypeConversation`, `ReferenceTypeNone`) — it dispatches to the same `startReferenceType*` helpers as `Start()`, with the addition of passing the `ResolvedTeam` to the pipecat start call.

The existing `Start()` method remains unchanged for single-AI sessions.

### Init Messages Behavior for Team Sessions

When `TeamID` is set on an AIcall, `startInitMessages` behavior changes:
- The default system prompt (`defaultCommonAIcallSystemPrompt` or `defaultCommonAItaskSystemPrompt`) is still injected — it provides conversation behavior guidelines
- The AI's `InitPrompt` is still injected for the **starting member only** — Pipecat Flows handles per-node prompt switching after that
- Pipecat-manager is responsible for swapping prompts/tools on each member transition

### ResolvedTeam: RPC Data Structure

A new struct carries the fully resolved team config across the RPC boundary from ai-manager to pipecat-manager.

**Important:** The `ResolvedAI` struct intentionally omits sensitive fields from `ai.AI` — specifically `EngineKey` (customer's LLM API key). It includes `Identity` (ID, CustomerID) for logging and tracing which member's AI is active during the conversation.

```go
// models/team/resolved.go (in ai-manager)
// Also mirrored in bin-pipecat-manager/models/ for deserialization

type ResolvedAI struct {
    identity.Identity                                        // ID, CustomerID — needed for logging/tracing
    EngineModel ai.EngineModel  `json:"engine_model,omitempty"`
    EngineData  map[string]any  `json:"engine_data,omitempty"`
    InitPrompt  string          `json:"init_prompt,omitempty"`
    TTSType     ai.TTSType      `json:"tts_type,omitempty"`
    TTSVoiceID  string          `json:"tts_voice_id,omitempty"`
    STTType     ai.STTType      `json:"stt_type,omitempty"`
    ToolNames   []tool.ToolName `json:"tool_names,omitempty"`
}

type ResolvedMember struct {
    ID          uuid.UUID        `json:"id"`
    Name        string           `json:"name"`
    AI          ResolvedAI       `json:"ai"`           // Stripped-down AI config (no EngineKey)
    Tools       []tool.Tool      `json:"tools"`         // Resolved tool definitions from AI.ToolNames
    Transitions []Transition     `json:"transitions"`   // Transition functions for this member
}

type ResolvedTeam struct {
    ID            uuid.UUID        `json:"id"`
    StartMemberID uuid.UUID        `json:"start_member_id"`
    Members       []ResolvedMember `json:"members"`
}
```

ai-manager resolves ALL member AIs and their tools upfront at session start, avoiding runtime RPC calls during live transitions. The `ResolvedAI` is built from `ai.AI` by copying only the non-sensitive fields needed by pipecat-manager.

### Pipecat RPC Interface Change

The `PipecatV1PipecatcallStart` function in `bin-common-handler/pkg/requesthandler/` gains an optional `resolvedTeam` parameter:

```go
func (r *requestHandler) PipecatV1PipecatcallStart(
    ctx context.Context,
    id uuid.UUID,
    customerID uuid.UUID,
    activeflowID uuid.UUID,
    referenceType pmpipecatcall.ReferenceType,
    referenceID uuid.UUID,
    llmType pmpipecatcall.LLMType,
    llmMessages []map[string]any,
    sttType pmpipecatcall.STTType,
    sttLanguage string,
    ttsType pmpipecatcall.TTSType,
    ttsLanguage string,
    ttsVoiceID string,
    resolvedTeam *team.ResolvedTeam,   // NEW — nil for single-AI sessions
) (*pmpipecatcall.Pipecatcall, error)
```

**Blast radius:** This changes the `RequestHandler` interface in `bin-common-handler`, which is vendored by all 30+ services. All callers of `PipecatV1PipecatcallStart` must be updated to pass `nil` for the new parameter. This is a single-line change per caller but affects multiple services. The verification workflow must be run on all affected services.

**Deployment ordering:** The change is backwards compatible at the wire level. If ai-manager deploys before pipecat-manager, the `resolved_team` JSON field is silently ignored by the old pipecat-manager (Go's JSON unmarshaler ignores unknown fields). If pipecat-manager deploys first, the missing field is zero-value (nil). Either order is safe.

### Pipecat-Manager Translation

When pipecat-manager receives a non-nil `resolvedTeam`, it translates to Pipecat Flows:

- Each `ResolvedMember` → Pipecat `NodeConfig`
  - `role_messages` from `AI.InitPrompt`
  - `functions` from `Tools` (resolved tool definitions) + `Transitions` (registered as edge functions)
- `StartMemberID` → initial node passed to `FlowManager.initialize()`
- `FlowManager` handles all transitions internally

When `resolvedTeam` is nil, pipecat-manager behaves as today (single-AI session).

### End-to-End Sequence

```
1. User creates AI configs (existing API)
   - AI-greeter, AI-billing, AI-support — each with own prompt + tools

2. User creates a Team (new API)
   POST /v1/teams
   {
     "name": "Customer Service Team",
     "start_member_id": "<greeter-member-id>",
     "members": [
       {
         "id": "<greeter-member-id>",
         "name": "Greeter",
         "ai_id": "<AI-greeter-id>",
         "transitions": [
           {
             "function_name": "transfer_to_billing",
             "description": "Customer has a billing question",
             "next_member_id": "<billing-member-id>"
           },
           {
             "function_name": "transfer_to_support",
             "description": "Customer has a technical issue",
             "next_member_id": "<support-member-id>"
           }
         ]
       },
       {
         "id": "<billing-member-id>",
         "name": "Billing Specialist",
         "ai_id": "<AI-billing-id>",
         "transitions": [
           {
             "function_name": "transfer_to_greeter",
             "description": "Customer needs something else",
             "next_member_id": "<greeter-member-id>"
           }
         ]
       },
       {
         "id": "<support-member-id>",
         "name": "Technical Support",
         "ai_id": "<AI-support-id>",
         "transitions": [
           {
             "function_name": "transfer_to_greeter",
             "description": "Customer needs something else",
             "next_member_id": "<greeter-member-id>"
           }
         ]
       }
     ]
   }

3. Flow-manager starts an AIcall with a Team reference
   - Calls StartWithTeam() instead of Start()

4. ai-manager resolves team config and sends to pipecat-manager
   - Fetches Team by ID
   - For each Member, fetches AI config + resolves tool definitions
   - Builds ResolvedTeam struct
   - Passes to PipecatV1PipecatcallStart with resolvedTeam parameter

5. pipecat-manager translates to Pipecat Flows
   - Each ResolvedMember → NodeConfig
     - role_messages from AI.InitPrompt
     - functions from resolved Tools + Transitions
   - StartMemberID → initial node
   - Builds FlowManager and runs the conversation

6. During the call, Pipecat handles node transitions
   - LLM calls "transfer_to_billing" → Pipecat switches to billing member
   - New prompt, new tools, conversation continues seamlessly
```

## Files Changed

### New files (bin-ai-manager)

| Path | Purpose |
|------|---------|
| models/team/main.go | Team struct |
| models/team/member.go | Member, Transition structs |
| models/team/resolved.go | ResolvedAI, ResolvedMember, ResolvedTeam structs |
| models/team/field.go | Field type constants + FieldDeleted |
| models/team/filters.go | FieldStruct for query filtering |
| models/team/webhook.go | WebhookMessage + ConvertWebhookMessage() |
| models/team/event.go | Event types (team_created, team_updated, team_deleted) |
| pkg/dbhandler/team.go | DB CRUD operations |
| pkg/cachehandler/team.go | Redis cache (TeamGet, TeamSet) |
| pkg/teamhandler/main.go | Handler interface + constructor + `//go:generate mockgen` directive |
| pkg/teamhandler/handler.go | Business logic (create, get, list, update, delete) |
| pkg/teamhandler/validation.go | Validation logic (rules 1-11) |
| pkg/listenhandler/v1_teams.go | Request handlers for /v1/teams endpoints |
| pkg/listenhandler/models/request/teams.go | Request structs (V1DataTeamsPost, V1DataTeamsIDPut) |
| scripts/database_scripts_test/table_ai_teams.sql | Test DB schema for ai_teams |

### New test files (bin-ai-manager)

| Path | Covers |
|------|--------|
| pkg/teamhandler/handler_test.go | Create, Get, List, Update, Delete — happy paths + error cases |
| pkg/teamhandler/validation_test.go | All 11 validation rules, edge cases, cross-member function name allowed |
| pkg/dbhandler/team_test.go | DB CRUD, JSON serialization, soft delete, FieldDeleted filter |
| pkg/listenhandler/v1_teams_test.go | Request routing, input validation, WebhookMessage conversion |
| pkg/aicallhandler/start_with_team_test.go | StartWithTeam happy path + error paths |

### Modified files (bin-ai-manager)

| Path | Change |
|------|--------|
| pkg/dbhandler/main.go | Add Team methods to DBHandler interface |
| pkg/cachehandler/main.go | Add Team methods (TeamGet, TeamSet) to CacheHandler interface |
| pkg/listenhandler/main.go | Add route regexes (regV1TeamsGet, regV1Teams, regV1TeamsID) + teamHandler dependency to struct and NewListenHandler constructor |
| models/aicall/main.go | Add `TeamID uuid.UUID` field with `json:"team_id,omitempty" db:"team_id,uuid"` tags |
| models/aicall/field.go | Add `FieldTeamID Field = "team_id"` constant |
| models/aicall/filters.go | Add `TeamID uuid.UUID \`filter:"team_id"\`` to FieldStruct |
| models/aicall/webhook.go | Add `TeamID uuid.UUID \`json:"team_id,omitempty"\`` to WebhookMessage + update ConvertWebhookMessage() |
| pkg/aicallhandler/main.go | Add StartWithTeam to AIcallHandler interface; add `teamHandler` and `toolHandler` dependencies to struct and NewAIcallHandler constructor |
| pkg/aicallhandler/db.go | Add `teamID uuid.UUID` parameter to Create() — existing callers pass `uuid.Nil` |
| pkg/aicallhandler/start.go | Add StartWithTeam method + team resolution logic; add `teamID` parameter to `startAIcall()` (intermediary between Start/StartWithTeam and Create); update existing callers of `startAIcall()` (`startReferenceTypeCall`, `startReferenceTypeConversation`, `startReferenceTypeNone`, `StartTask`) to pass `uuid.Nil` for teamID |
| cmd/ai-manager/main.go | Create teamHandler; pass teamHandler and toolHandler to NewAIcallHandler; pass teamHandler to NewListenHandler |

### Modified files (bin-common-handler)

| Path | Change |
|------|--------|
| pkg/requesthandler/main.go | Add `resolvedTeam *team.ResolvedTeam` parameter to PipecatV1PipecatcallStart interface method |
| pkg/requesthandler/pipecat_pipecatcall.go | Update PipecatV1PipecatcallStart implementation to include resolvedTeam in RPC payload |

**Note:** After bin-common-handler changes, ALL services that vendor it must run `go mod vendor` and pass verification. Existing callers pass `nil` for the new parameter.

### Other services (later phases)

| Service | Change |
|---------|--------|
| bin-openapi-manager | Add Team/Member/Transition/ResolvedTeam schemas + endpoints |
| bin-api-manager | Regenerate server code, add servicehandler for teams |
| bin-dbscheme-manager | Alembic migration: CREATE TABLE ai_teams + ALTER TABLE ai_aicalls ADD team_id |
| bin-pipecat-manager | Add ResolvedTeam field to V1DataPipecatcallsPost request struct; accept ResolvedTeam in start payload; translate to Pipecat Flows NodeConfig; build FlowManager |
| bin-flow-manager | Add optional team_id to AI action configuration for flow integration |

### Documentation (later phase)

| Path | Change |
|------|--------|
| bin-api-manager/docsdev/source/ | New RST files: team_overview.rst, team_tutorial.rst, team_struct.rst |

## Test Coverage

### teamhandler/handler_test.go

Uses gomock pattern:
```go
mc := gomock.NewController(t)
defer mc.Finish()
mockDB := dbhandler.NewMockDBHandler(mc)
mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// etc.
```

**Create — happy path:**
- Create team with valid members and transitions → success
- Verify returned team matches input
- Verify webhook event published

**Create — error cases:**
- StartMemberID is uuid.Nil → error
- StartMemberID not in members list → error
- Duplicate member IDs → error
- Member.ID is uuid.Nil → error
- Member.AIID is uuid.Nil → error
- Member.AIID references non-existent AI → error
- Transition.NextMemberID references non-existent member → error
- Transition.FunctionName collides with reserved tool name → error
- Duplicate FunctionName within same member → error
- Empty members list → error
- Empty member name → error
- DB create fails → error propagated

**Get:**
- Get existing team → success
- Get non-existent team → ErrNotFound

**List:**
- List with filters → correct results
- List with FieldDeleted=false → excludes soft-deleted teams
- List with FieldDeleted=true → includes soft-deleted teams
- List empty → empty slice (not nil)
- List with pagination token → correct ordering

**Update:**
- Update name/detail only → success, members unchanged
- Update members → all validations re-run
- Update with invalid transitions → error

**Delete:**
- Soft delete → tm_delete set, webhook event published
- Delete non-existent → error

### teamhandler/validation_test.go

Table-driven tests for each of the 11 validation rules:
- Valid team passes all checks
- Each rule tested independently with minimal failing input
- Boundary cases: single member team, member with no transitions
- uuid.Nil checks for StartMemberID, Member.ID, Member.AIID
- Same FunctionName on different members → allowed (intentional, per-node isolation)

### dbhandler/team_test.go

Requires `scripts/database_scripts_test/table_ai_teams.sql` for test DB setup.

- CRUD round-trip: create → get → matches
- JSON serialization: Members correctly stored and retrieved with nested Transitions
- Soft delete: tm_delete set, record still retrievable
- List with filters: customer_id filter works
- List with FieldDeleted filter: excludes/includes soft-deleted records
- List empty result: returns `[]` not nil

### listenhandler/v1_teams_test.go

- POST /v1/teams with valid body → 200 + team response
- POST /v1/teams with malformed JSON → 400
- POST /v1/teams with missing required fields → 400
- GET /v1/teams? with filters → correct delegation to teamhandler
- GET /v1/teams/{id} → 200 + WebhookMessage (not internal struct)
- PUT /v1/teams/{id} with valid body → 200
- DELETE /v1/teams/{id} → 200

### aicallhandler/start_with_team_test.go

**StartWithTeam — happy path:**
- Valid team with multiple members → AIcall created with TeamID set, ResolvedTeam passed to pipecat
- Verify AIcall snapshots start member's AI config (EngineModel, TTSType, etc.)
- Verify ResolvedTeam contains all members with stripped AI configs (has Identity, no EngineKey)

**StartWithTeam — error paths:**
- Non-existent team ID → error
- Team member's AI has been soft-deleted since team creation → error
- Tool resolution fails for a member → error
- AIcall DB create fails → error propagated
- Pipecat start call fails → error propagated, AIcall cleaned up

**StartWithTeam — reference types:**
- ReferenceTypeCall → dispatches correctly
- ReferenceTypeConversation → dispatches correctly
- ReferenceTypeNone (task) → dispatches correctly

## Deferred (Not In Scope)

- pre_actions / post_actions on Members
- Context strategy per transition (RESET, RESET_WITH_SUMMARY)
- Transition parameters (properties/required)
- Visual flow editor integration
- Team versioning
- Graph reachability validation
- Max Members/Transitions limits (add if needed based on usage)
