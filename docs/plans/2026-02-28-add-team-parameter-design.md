# Add Team Parameter Field

## Problem

The Team model in ai-manager has no freeform data field. Customers need a way to attach arbitrary key-value configuration to teams, similar to how `EngineData` works on the AI model. The data should support flow variable substitution at runtime (e.g., `{{voipbin.flow.id}}`).

## Approach

Mirror the AI model's `EngineData` pattern exactly: add a `Parameter` field (`map[string]any`, stored as JSON) to the Team model and thread it through the full create/update/aicall chain.

- `Parameter` is optional in both POST and PUT requests (nil means no team-level data).
- At aicall start, `Parameter` is copied into the aicall record as `TeamParameter`.
- At runtime, `Parameter` undergoes flow variable substitution and is injected as a separate system message (independent from AI's EngineData).

## Changes

### 1. Team Model

**`bin-ai-manager/models/team/main.go`**
- Add `Parameter map[string]any` with tags `json:"parameter,omitempty" db:"parameter,json"`

**`bin-ai-manager/models/team/field.go`**
- Add `FieldParameter Field = "parameter"`

**`bin-ai-manager/models/team/webhook.go`**
- Add `Parameter map[string]any` to `WebhookMessage`
- Wire in `ConvertWebhookMessage()`

### 2. AIcall Model

**`bin-ai-manager/models/aicall/main.go`**
- Add `TeamParameter map[string]any` with tags `json:"team_parameter,omitempty" db:"team_parameter,json"`

**`bin-ai-manager/models/aicall/field.go`**
- Add `FieldTeamParameter Field = "team_parameter"`

**`bin-ai-manager/models/aicall/webhook.go`**
- Add `TeamParameter map[string]any` to `WebhookMessage`
- Wire in `ConvertWebhookMessage()`

### 3. Database Migration

**Alembic migration in `bin-dbscheme-manager`**
- Add `parameter json` column to `ai_teams` table (nullable, default NULL)
- Add `team_parameter json` column to `ai_aicalls` table (nullable, default NULL)

**`bin-ai-manager/scripts/database_scripts_test/`**
- Update `table_ai_teams.sql` to include `parameter json`
- Update `table_ai_aicalls.sql` to include `team_parameter json`

### 4. Team Create/Update Flow

**`bin-ai-manager/pkg/teamhandler/main.go`**
- Add `parameter map[string]any` param to `Create()` and `Update()` interface signatures

**`bin-ai-manager/pkg/teamhandler/handler.go`**
- Set `Parameter` on Team struct in `Create()`
- Add `FieldParameter` to the update fields map in `Update()`

**`bin-ai-manager/pkg/listenhandler/models/request/teams.go`**
- Add `Parameter map[string]any` to `V1DataTeamsPost` and `V1DataTeamsIDPut`

**`bin-ai-manager/pkg/listenhandler/v1_teams.go`**
- Pass `req.Parameter` to `teamHandler.Create()` and `teamHandler.Update()`

**`bin-common-handler/pkg/requesthandler/ai_teams.go`**
- Add `parameter map[string]any` to `AIV1TeamCreate()` and `AIV1TeamUpdate()` function signatures
- Include in RPC request data structs

**`bin-api-manager/pkg/servicehandler/team.go`**
- Add `parameter` param to `TeamCreate()` and `TeamUpdate()`, pass through to requesthandler

**`bin-api-manager/server/teams.go`**
- Extract `Parameter` from OpenAPI request body in `PostTeams` and `PutTeamsId`
- Pass downstream to serviceHandler

### 5. AIcall Runtime Integration

**`bin-ai-manager/pkg/aicallhandler/start.go`**
- Modify `resolveAI()` to also return `map[string]any` (team Parameter) alongside `*ai.AI`
- For `AssistanceTypeTeam`: return the team's `Parameter` from the fetched team
- For `AssistanceTypeAI`: return nil
- Thread the parameter through `startReferenceTypeCall` → `startAIcall` → `Create`

**`bin-ai-manager/pkg/aicallhandler/db.go`**
- Add `teamParameter map[string]any` param to `Create()`
- Set `TeamParameter` on the aicall struct

**`bin-ai-manager/pkg/aicallhandler/chat.go`**
- Refactor `getEngineData()` to accept `map[string]any` directly instead of `*ai.AI` (reusable for both EngineData and Parameter)
- In `startInitMessages()`: call the refactored function for both `a.EngineData` and team Parameter, injecting each as a separate system message
- Skip substitution when Parameter is nil (returns `"{}"`)

### 6. OpenAPI Spec

**`bin-openapi-manager/openapi/openapi.yaml`**
- `AIManagerTeam` schema: Add `parameter` field (`type: object, additionalProperties: true`, optional)
- `AIManagerAIcall` schema: Add `team_parameter` field (`type: object, additionalProperties: true`)

**`bin-openapi-manager/openapi/paths/teams/main.yaml`**
- POST request body: Add `parameter` (optional, not in `required` list)

**`bin-openapi-manager/openapi/paths/teams/id.yaml`**
- PUT request body: Add `parameter` (optional, not in `required` list)

**Code generation**
- `cd bin-openapi-manager && go generate ./...`
- `cd bin-api-manager && go generate ./...`

## Runtime Data Flow

```
Team.Parameter (map[string]any)
  → stored in ai_teams.parameter (json column)

On aicall start (assistanceType == "team"):
  → resolveAI() fetches team, returns Parameter alongside *ai.AI
  → Create() stores Parameter as aicall.TeamParameter in ai_aicalls.team_parameter
  → startInitMessages() processes Parameter with flow variable substitution
  → injected as separate system message (independent from EngineData)

On aicall start (assistanceType == "ai"):
  → TeamParameter is nil, no system message injected
```

## Trade-offs

- **Optional vs required**: Parameter is optional to avoid breaking existing team workflows. Users who don't need it simply omit it.
- **Separate system messages**: EngineData and Parameter produce independent messages rather than merging. This keeps them semantically distinct and avoids key collision issues.
- **Refactoring getEngineData**: Changing the signature from `*ai.AI` to `map[string]any` is a small breaking change to the internal API but enables clean reuse without duplication.
