# Add Team Parameter Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `Parameter` field (`map[string]any`) to the Team model, mirroring the AI model's `EngineData` pattern, with flow variable substitution at runtime.

**Architecture:** The `Parameter` field threads through the entire stack: Team model → DB → create/update handlers → RPC → API server → OpenAPI. At aicall start, the team's `Parameter` is copied into the aicall record as `TeamParameter` and processed with flow variable substitution as a separate system message.

**Tech Stack:** Go, MySQL (JSON column), Alembic migrations, OpenAPI 3.0, squirrel query builder, commondatabasehandler.

**Worktree:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter/`

---

### Task 1: Add Parameter to Team Model

**Files:**
- Modify: `bin-ai-manager/models/team/main.go:14-25`
- Modify: `bin-ai-manager/models/team/field.go:6-17`
- Modify: `bin-ai-manager/models/team/webhook.go:13-40`

**Step 1: Add Parameter field to Team struct**

In `bin-ai-manager/models/team/main.go`, add the `Parameter` field after `Members`:

```go
type Team struct {
	identity.Identity

	Name          string         `json:"name,omitempty" db:"name"`
	Detail        string         `json:"detail,omitempty" db:"detail"`
	StartMemberID uuid.UUID      `json:"start_member_id,omitempty" db:"start_member_id,uuid"`
	Members       []Member       `json:"members,omitempty" db:"members,json"`
	Parameter     map[string]any `json:"parameter,omitempty" db:"parameter,json"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
```

**Step 2: Add FieldParameter constant**

In `bin-ai-manager/models/team/field.go`, add after `FieldMembers`:

```go
FieldParameter     Field = "parameter"
```

**Step 3: Add Parameter to WebhookMessage**

In `bin-ai-manager/models/team/webhook.go`, add `Parameter` to the struct and `ConvertWebhookMessage()`:

```go
type WebhookMessage struct {
	commonidentity.Identity

	Name          string         `json:"name,omitempty"`
	Detail        string         `json:"detail,omitempty"`
	StartMemberID uuid.UUID      `json:"start_member_id,omitempty"`
	Members       []Member       `json:"members,omitempty"`
	Parameter     map[string]any `json:"parameter,omitempty"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

func (h *Team) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Name:          h.Name,
		Detail:        h.Detail,
		StartMemberID: h.StartMemberID,
		Members:       h.Members,
		Parameter:     h.Parameter,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}
```

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter
git add bin-ai-manager/models/team/main.go bin-ai-manager/models/team/field.go bin-ai-manager/models/team/webhook.go
git commit -m "NOJIRA-add-team-parameter

- bin-ai-manager: Add Parameter field to Team model, field constants, and WebhookMessage"
```

---

### Task 2: Add TeamParameter to AIcall Model

**Files:**
- Modify: `bin-ai-manager/models/aicall/main.go:13-41`
- Modify: `bin-ai-manager/models/aicall/field.go:7-38`
- Modify: `bin-ai-manager/models/aicall/webhook.go:14-73`

**Step 1: Add TeamParameter field to AIcall struct**

In `bin-ai-manager/models/aicall/main.go`, add after `AISTTType`:

```go
AISTTType     ai.STTType     `json:"ai_stt_type,omitempty" db:"ai_stt_type"`

TeamParameter map[string]any `json:"team_parameter,omitempty" db:"team_parameter,json"`
```

**Step 2: Add FieldTeamParameter constant**

In `bin-ai-manager/models/aicall/field.go`, add after `FieldAISTTType`:

```go
FieldTeamParameter Field = "team_parameter"
```

**Step 3: Add TeamParameter to WebhookMessage**

In `bin-ai-manager/models/aicall/webhook.go`, add to the struct after `AISTTType`:

```go
AISTTType     ai.STTType     `json:"ai_stt_type,omitempty"`

TeamParameter map[string]any `json:"team_parameter,omitempty"`
```

And in `ConvertWebhookMessage()`, add:

```go
AISTTType:     h.AISTTType,

TeamParameter: h.TeamParameter,
```

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter
git add bin-ai-manager/models/aicall/main.go bin-ai-manager/models/aicall/field.go bin-ai-manager/models/aicall/webhook.go
git commit -m "NOJIRA-add-team-parameter

- bin-ai-manager: Add TeamParameter field to AIcall model, field constants, and WebhookMessage"
```

---

### Task 3: Database Migration and Test Scripts

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<new_id>_ai_teams_add_parameter_aicalls_add_team_parameter.py`
- Modify: `bin-ai-manager/scripts/database_scripts_test/table_ai_teams.sql:1-22`
- Modify: `bin-ai-manager/scripts/database_scripts_test/table_ai_aicalls.sql:1-42`

**Step 1: Create Alembic migration**

Create `bin-dbscheme-manager/bin-manager/main/versions/f1a2b3c4d5e6_ai_teams_add_parameter_aicalls_add_team_parameter.py`:

```python
"""ai_teams add parameter column and ai_aicalls add team_parameter column

Revision ID: f1a2b3c4d5e6
Revises: ed4cff99a82e
Create Date: 2026-02-28 01:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'f1a2b3c4d5e6'
down_revision = 'ed4cff99a82e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE ai_teams ADD parameter JSON AFTER members;""")
    op.execute("""ALTER TABLE ai_aicalls ADD team_parameter JSON AFTER ai_stt_type;""")


def downgrade():
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN team_parameter;""")
    op.execute("""ALTER TABLE ai_teams DROP COLUMN parameter;""")
```

**Step 2: Update test SQL scripts**

In `bin-ai-manager/scripts/database_scripts_test/table_ai_teams.sql`, add after `members`:

```sql
  members         json,           -- members as json array
  parameter       json,           -- team-level parameter data
```

In `bin-ai-manager/scripts/database_scripts_test/table_ai_aicalls.sql`, add after `ai_stt_type`:

```sql
  ai_stt_type      varchar(255),
  team_parameter   json,
```

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter
git add bin-dbscheme-manager/bin-manager/main/versions/f1a2b3c4d5e6_ai_teams_add_parameter_aicalls_add_team_parameter.py
git add bin-ai-manager/scripts/database_scripts_test/table_ai_teams.sql bin-ai-manager/scripts/database_scripts_test/table_ai_aicalls.sql
git commit -m "NOJIRA-add-team-parameter

- bin-dbscheme-manager: Add Alembic migration for parameter and team_parameter columns
- bin-ai-manager: Update test SQL scripts with new columns"
```

---

### Task 4: Team Create/Update Handler Chain (ai-manager internal)

**Files:**
- Modify: `bin-ai-manager/pkg/teamhandler/main.go:19-25` (interface)
- Modify: `bin-ai-manager/pkg/teamhandler/handler.go:16,112` (Create and Update)
- Modify: `bin-ai-manager/pkg/listenhandler/models/request/teams.go:12-28`
- Modify: `bin-ai-manager/pkg/listenhandler/v1_teams.go:92-98,175-182`

**Step 1: Add `parameter` to request structs**

In `bin-ai-manager/pkg/listenhandler/models/request/teams.go`:

```go
type V1DataTeamsPost struct {
	CustomerID    uuid.UUID      `json:"customer_id,omitempty"`
	Name          string         `json:"name,omitempty"`
	Detail        string         `json:"detail,omitempty"`
	StartMemberID uuid.UUID      `json:"start_member_id,omitempty"`
	Members       []team.Member  `json:"members,omitempty"`
	Parameter     map[string]any `json:"parameter,omitempty"`
}

type V1DataTeamsIDPut struct {
	Name          string         `json:"name,omitempty"`
	Detail        string         `json:"detail,omitempty"`
	StartMemberID uuid.UUID      `json:"start_member_id,omitempty"`
	Members       []team.Member  `json:"members,omitempty"`
	Parameter     map[string]any `json:"parameter,omitempty"`
}
```

**Step 2: Update TeamHandler interface**

In `bin-ai-manager/pkg/teamhandler/main.go`, add `parameter map[string]any` param to Create and Update:

```go
type TeamHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []team.Member, parameter map[string]any) (*team.Team, error)
	Get(ctx context.Context, id uuid.UUID) (*team.Team, error)
	List(ctx context.Context, size uint64, token string, filters map[team.Field]any) ([]*team.Team, error)
	Delete(ctx context.Context, id uuid.UUID) (*team.Team, error)
	Update(ctx context.Context, id uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []team.Member, parameter map[string]any) (*team.Team, error)
}
```

**Step 3: Update handler implementations**

In `bin-ai-manager/pkg/teamhandler/handler.go`:

Update `Create` signature and set Parameter on the struct:

```go
func (h *teamHandler) Create(ctx context.Context, customerID uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []team.Member, parameter map[string]any) (*team.Team, error) {
```

In the struct construction, add:

```go
	t := &team.Team{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Name:          name,
		Detail:        detail,
		StartMemberID: startMemberID,
		Members:       members,
		Parameter:     parameter,
	}
```

Update `Update` signature:

```go
func (h *teamHandler) Update(ctx context.Context, id uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []team.Member, parameter map[string]any) (*team.Team, error) {
```

Add `FieldParameter` to the fields map:

```go
	fields := map[team.Field]any{
		team.FieldName:          name,
		team.FieldDetail:        detail,
		team.FieldStartMemberID: startMemberID,
		team.FieldMembers:       members,
		team.FieldParameter:     parameter,
	}
```

**Step 4: Update listenhandler callers**

In `bin-ai-manager/pkg/listenhandler/v1_teams.go`:

Update `processV1TeamsPost` to pass `req.Parameter`:

```go
	tmp, err := h.teamHandler.Create(
		ctx,
		req.CustomerID,
		req.Name,
		req.Detail,
		req.StartMemberID,
		req.Members,
		req.Parameter,
	)
```

Update `processV1TeamsIDPut` to pass `req.Parameter`:

```go
	tmp, err := h.teamHandler.Update(
		ctx,
		id,
		req.Name,
		req.Detail,
		req.StartMemberID,
		req.Members,
		req.Parameter,
	)
```

**Step 5: Regenerate mocks**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter/bin-ai-manager
go generate ./pkg/teamhandler/...
```

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter
git add bin-ai-manager/pkg/teamhandler/ bin-ai-manager/pkg/listenhandler/
git commit -m "NOJIRA-add-team-parameter

- bin-ai-manager: Add parameter to TeamHandler Create/Update interface and implementations
- bin-ai-manager: Add parameter to RPC request structs and listenhandler callers"
```

---

### Task 5: RPC and API Layer (bin-common-handler + bin-api-manager)

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go:217-233` (interface)
- Modify: `bin-common-handler/pkg/requesthandler/ai_teams.go:61-95,119-152`
- Modify: `bin-api-manager/pkg/servicehandler/main.go:266,270` (interface)
- Modify: `bin-api-manager/pkg/servicehandler/team.go:33-70,194-239`
- Modify: `bin-api-manager/server/teams.go:13-55,176-227`

**Step 1: Update requesthandler interface**

In `bin-common-handler/pkg/requesthandler/main.go`, add `parameter map[string]any` to both:

```go
	AIV1TeamCreate(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		startMemberID uuid.UUID,
		members []amteam.Member,
		parameter map[string]any,
	) (*amteam.Team, error)
	AIV1TeamDelete(ctx context.Context, teamID uuid.UUID) (*amteam.Team, error)
	AIV1TeamUpdate(
		ctx context.Context,
		teamID uuid.UUID,
		name string,
		detail string,
		startMemberID uuid.UUID,
		members []amteam.Member,
		parameter map[string]any,
	) (*amteam.Team, error)
```

**Step 2: Update requesthandler implementations**

In `bin-common-handler/pkg/requesthandler/ai_teams.go`:

Update `AIV1TeamCreate`:

```go
func (r *requestHandler) AIV1TeamCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	startMemberID uuid.UUID,
	members []amteam.Member,
	parameter map[string]any,
) (*amteam.Team, error) {
	uri := "/v1/teams"

	data := &amrequest.V1DataTeamsPost{
		CustomerID:    customerID,
		Name:          name,
		Detail:        detail,
		StartMemberID: startMemberID,
		Members:       members,
		Parameter:     parameter,
	}
```

Update `AIV1TeamUpdate`:

```go
func (r *requestHandler) AIV1TeamUpdate(
	ctx context.Context,
	teamID uuid.UUID,
	name string,
	detail string,
	startMemberID uuid.UUID,
	members []amteam.Member,
	parameter map[string]any,
) (*amteam.Team, error) {
	uri := fmt.Sprintf("/v1/teams/%s", teamID)

	data := &amrequest.V1DataTeamsIDPut{
		Name:          name,
		Detail:        detail,
		StartMemberID: startMemberID,
		Members:       members,
		Parameter:     parameter,
	}
```

**Step 3: Update servicehandler interface**

In `bin-api-manager/pkg/servicehandler/main.go`:

```go
	TeamCreate(ctx context.Context, a *amagent.Agent, name string, detail string, startMemberID uuid.UUID, members []amteam.Member, parameter map[string]any) (*amteam.WebhookMessage, error)
	// ...
	TeamUpdate(ctx context.Context, a *amagent.Agent, id uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []amteam.Member, parameter map[string]any) (*amteam.WebhookMessage, error)
```

**Step 4: Update servicehandler implementations**

In `bin-api-manager/pkg/servicehandler/team.go`:

Update `TeamCreate`:

```go
func (h *serviceHandler) TeamCreate(
	ctx context.Context,
	a *amagent.Agent,
	name string,
	detail string,
	startMemberID uuid.UUID,
	members []amteam.Member,
	parameter map[string]any,
) (*amteam.WebhookMessage, error) {
```

Pass `parameter` to the RPC call:

```go
	tmp, err := h.reqHandler.AIV1TeamCreate(
		ctx,
		a.CustomerID,
		name,
		detail,
		startMemberID,
		members,
		parameter,
	)
```

Update `TeamUpdate`:

```go
func (h *serviceHandler) TeamUpdate(
	ctx context.Context,
	a *amagent.Agent,
	id uuid.UUID,
	name string,
	detail string,
	startMemberID uuid.UUID,
	members []amteam.Member,
	parameter map[string]any,
) (*amteam.WebhookMessage, error) {
```

Pass `parameter` to the RPC call:

```go
	tmp, err := h.reqHandler.AIV1TeamUpdate(
		ctx,
		id,
		name,
		detail,
		startMemberID,
		members,
		parameter,
	)
```

**Step 5: Update server/teams.go**

In `bin-api-manager/server/teams.go`, update `PostTeams`:

After `members := convertOpenAPIMembers(req.Members)`, extract parameter:

```go
	members := convertOpenAPIMembers(req.Members)

	var parameter map[string]any
	if req.Parameter != nil {
		parameter = *req.Parameter
	}

	res, err := h.serviceHandler.TeamCreate(
		c.Request.Context(),
		&a,
		req.Name,
		req.Detail,
		startMemberID,
		members,
		parameter,
	)
```

Update `PutTeamsId` similarly:

```go
	members := convertOpenAPIMembers(req.Members)

	var parameter map[string]any
	if req.Parameter != nil {
		parameter = *req.Parameter
	}

	res, err := h.serviceHandler.TeamUpdate(
		c.Request.Context(),
		&a,
		target,
		req.Name,
		req.Detail,
		startMemberID,
		members,
		parameter,
	)
```

Note: `req.Parameter` will be `*map[string]interface{}` from the generated OpenAPI types (optional field with `omitempty`). We dereference it if non-nil.

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter
git add bin-common-handler/pkg/requesthandler/main.go bin-common-handler/pkg/requesthandler/ai_teams.go
git add bin-api-manager/pkg/servicehandler/main.go bin-api-manager/pkg/servicehandler/team.go bin-api-manager/server/teams.go
git commit -m "NOJIRA-add-team-parameter

- bin-common-handler: Add parameter to AIV1TeamCreate and AIV1TeamUpdate RPC signatures
- bin-api-manager: Thread parameter through servicehandler and server layers"
```

---

### Task 6: AIcall Runtime Integration

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:21-51,110,159,209,440,477`
- Modify: `bin-ai-manager/pkg/aicallhandler/db.go:18-30,37-63`
- Modify: `bin-ai-manager/pkg/aicallhandler/chat.go:61-109,383-417`

**Step 1: Update `resolveAI` to return team Parameter**

In `bin-ai-manager/pkg/aicallhandler/start.go`, change `resolveAI` to return the team Parameter as a second value:

```go
func (h *aicallHandler) resolveAI(ctx context.Context, assistanceType aicall.AssistanceType, assistanceID uuid.UUID) (*ai.AI, map[string]any, error) {
	switch assistanceType {
	case aicall.AssistanceTypeAI:
		c, err := h.aiHandler.Get(ctx, assistanceID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "could not get ai info. ai_id: %s", assistanceID)
		}
		return c, nil, nil

	case aicall.AssistanceTypeTeam:
		t, err := h.teamHandler.Get(ctx, assistanceID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "could not get team info. team_id: %s", assistanceID)
		}

		for _, m := range t.Members {
			if m.ID == t.StartMemberID {
				c, errAI := h.aiHandler.Get(ctx, m.AIID)
				if errAI != nil {
					return nil, nil, errors.Wrapf(errAI, "could not get ai info for team start member. ai_id: %s", m.AIID)
				}
				return c, t.Parameter, nil
			}
		}
		return nil, nil, fmt.Errorf("could not find start member in team. team_id: %s, start_member_id: %s", assistanceID, t.StartMemberID)

	default:
		return nil, nil, fmt.Errorf("unsupported assistance type: %s", assistanceType)
	}
}
```

**Step 2: Update all `resolveAI` callers**

In `Start()`:

```go
	c, teamParameter, err := h.resolveAI(ctx, assistanceType, assistanceID)
	if err != nil {
		return nil, errors.Wrap(err, "could not resolve ai config")
	}
```

Then pass `teamParameter` to `startReferenceTypeCall`, `startReferenceTypeConversation`, `startReferenceTypeNone`.

In `StartTask()`:

```go
	c, teamParameter, err := h.resolveAI(ctx, assistanceType, assistanceID)
```

Then pass `teamParameter` to `startAIcall`.

**Step 3: Thread `teamParameter` through start functions**

Add `teamParameter map[string]any` parameter to:
- `startReferenceTypeCall`
- `startReferenceTypeConversation`
- `startReferenceTypeNone`
- `startAIcall`

Each function passes it downstream to `startAIcall`, and `startAIcall` passes it to `Create` and `startInitMessages`.

Example for `startAIcall`:

```go
func (h *aicallHandler) startAIcall(
	ctx context.Context,
	a *ai.AI,
	assistanceType aicall.AssistanceType,
	assistanceID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
	confbridgeID uuid.UUID,
	gender aicall.Gender,
	language string,
	isTask bool,
	teamParameter map[string]any,
) (*aicall.AIcall, error) {
```

Pass to `Create`:

```go
	res, err := h.Create(ctx, a, assistanceType, assistanceID, activeflowID, referenceType, referenceID, confbridgeID, pipecatcallID, gender, language, teamParameter)
```

Pass to `startInitMessages`:

```go
	if errInitMessages := h.startInitMessages(ctx, a, res, isTask, teamParameter); errInitMessages != nil {
```

**Step 4: Update `Create` to store TeamParameter**

In `bin-ai-manager/pkg/aicallhandler/db.go`, add `teamParameter map[string]any` to the `Create` signature:

```go
func (h *aicallHandler) Create(
	ctx context.Context,
	c *ai.AI,
	assistanceType aicall.AssistanceType,
	assistanceID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
	confbridgeID uuid.UUID,
	pipecatcallID uuid.UUID,
	gender aicall.Gender,
	language string,
	teamParameter map[string]any,
) (*aicall.AIcall, error) {
```

Add `TeamParameter` to the struct:

```go
	tmp := &aicall.AIcall{
		// ... existing fields ...
		AISTTType:     c.STTType,

		TeamParameter: teamParameter,

		ActiveflowID:  activeflowID,
		// ...
	}
```

**Step 5: Refactor `getEngineData` to accept `map[string]any`**

In `bin-ai-manager/pkg/aicallhandler/chat.go`, rename `getEngineData` to `getDataAsJSON` and change its signature to accept `map[string]any` directly:

```go
func (h *aicallHandler) getDataAsJSON(ctx context.Context, data map[string]any, activeflowID uuid.UUID) string {
	if data == nil {
		return "{}"
	}

	wg := sync.WaitGroup{}
	tmpRes := sync.Map{}
	for k, v := range data {
		wg.Add(1)

		go func(key string, value any) {
			defer wg.Done()
			data := h.getEngineDataValue(ctx, value, activeflowID)
			tmpRes.Store(key, data)
		}(k, v)
	}
	wg.Wait()

	tmpMap := map[string]any{}
	tmpRes.Range(func(key, value any) bool {
		k, ok := key.(string)
		if !ok {
			logrus.WithFields(logrus.Fields{
				"func": "getDataAsJSON",
				"key":  key,
			}).Warn("Non-string key encountered in tmpRes; skipping entry")
			return true
		}
		tmpMap[k] = value
		return true
	})

	dataBytes, err := json.Marshal(tmpMap)
	if err != nil {
		logrus.Errorf("Could not marshal data back to string. err: %v", err)
		return "{}"
	}

	return string(dataBytes)
}
```

**Step 6: Update `startInitMessages` to process both EngineData and Parameter**

Add `teamParameter map[string]any` to the signature:

```go
func (h *aicallHandler) startInitMessages(ctx context.Context, a *ai.AI, c *aicall.AIcall, isTask bool, teamParameter map[string]any) error {
```

Replace the `getEngineData` call and add the team parameter processing:

```go
	// parse engine data
	if msg := h.getDataAsJSON(ctx, a.EngineData, c.ActiveflowID); msg != "{}" {
		messages = append(messages, msg)
	}
	log.Debugf("Parsed engine data. aicall_id: %s", c.ID)

	// parse team parameter
	if msg := h.getDataAsJSON(ctx, teamParameter, c.ActiveflowID); msg != "{}" {
		messages = append(messages, msg)
	}
	log.Debugf("Parsed team parameter. aicall_id: %s", c.ID)
```

**Step 7: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter
git add bin-ai-manager/pkg/aicallhandler/start.go bin-ai-manager/pkg/aicallhandler/db.go bin-ai-manager/pkg/aicallhandler/chat.go
git commit -m "NOJIRA-add-team-parameter

- bin-ai-manager: Thread team Parameter through resolveAI, startAIcall, and Create
- bin-ai-manager: Refactor getEngineData to reusable getDataAsJSON accepting map[string]any
- bin-ai-manager: Process team Parameter as separate system message in startInitMessages"
```

---

### Task 7: OpenAPI Spec and Code Generation

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml:1827-1856` (AIManagerTeam schema)
- Modify: `bin-openapi-manager/openapi/openapi.yaml:2068-2074` (AIManagerAIcall schema)
- Modify: `bin-openapi-manager/openapi/paths/teams/main.yaml:35-60` (POST request body)
- Modify: `bin-openapi-manager/openapi/paths/teams/id.yaml:59-84` (PUT request body)

**Step 1: Add `parameter` to AIManagerTeam schema**

In `bin-openapi-manager/openapi/openapi.yaml`, in the `AIManagerTeam` properties section, add after `members`:

```yaml
        members:
          type: array
          items:
            $ref: '#/components/schemas/AIManagerTeamMember'
          description: List of team members forming the graph nodes.
        parameter:
          type: object
          additionalProperties: true
          description: Custom key-value parameter data for the team. Supports flow variable substitution at runtime.
```

Note: Do NOT add `parameter` to the `required` list (it's optional).

**Step 2: Add `team_parameter` to AIManagerAIcall schema**

In the `AIManagerAIcall` properties section, add after `ai_stt_type`:

```yaml
        ai_stt_type:
          $ref: '#/components/schemas/AIManagerAISTTType'
          description: Speech-to-text provider type used for this call.
          example: "deepgram"
        team_parameter:
          type: object
          additionalProperties: true
          description: Custom key-value parameter data from the team configuration. Present when assistance_type is team.
```

**Step 3: Add `parameter` to POST /teams request body**

In `bin-openapi-manager/openapi/paths/teams/main.yaml`, add `parameter` property to the POST schema (after `members`):

```yaml
            parameter:
              type: object
              additionalProperties: true
              description: Custom key-value parameter data for the team. Supports flow variable substitution at runtime.
```

Do NOT add `parameter` to the `required` list.

**Step 4: Add `parameter` to PUT /teams request body**

In `bin-openapi-manager/openapi/paths/teams/id.yaml`, add `parameter` property to the PUT schema (after `members`):

```yaml
            parameter:
              type: object
              additionalProperties: true
              description: Custom key-value parameter data for the team. Supports flow variable substitution at runtime.
```

Do NOT add `parameter` to the `required` list.

**Step 5: Regenerate OpenAPI types and API server code**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter/bin-openapi-manager
go generate ./...

cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter/bin-api-manager
go generate ./...
```

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter
git add bin-openapi-manager/ bin-api-manager/gens/
git commit -m "NOJIRA-add-team-parameter

- bin-openapi-manager: Add parameter to AIManagerTeam schema and team endpoints
- bin-openapi-manager: Add team_parameter to AIManagerAIcall schema
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

### Task 8: Run Verification and Fix Issues

**Step 1: Run full verification for bin-ai-manager**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter/bin-ai-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Fix any compilation errors from:
- Test files referencing old `Create`, `resolveAI`, `startAIcall`, etc. signatures
- Mock files that need regeneration
- Any callers of the changed functions that were missed

**Step 2: Run full verification for bin-common-handler**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter/bin-common-handler
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 3: Run full verification for bin-api-manager**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter/bin-api-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 4: Run full verification for bin-openapi-manager**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter/bin-openapi-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 5: Check all other services that import bin-common-handler for breakage**

The `RequestHandler` interface changed (new `parameter` param on `AIV1TeamCreate` and `AIV1TeamUpdate`). Any service that has its own mock of `RequestHandler` needs regeneration.

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter
grep -rl "AIV1TeamCreate\|AIV1TeamUpdate" --include="*.go" | grep -v vendor | grep -v bin-common-handler | grep -v bin-ai-manager | grep -v bin-api-manager | sort -u
```

For each affected service, run `go generate ./...` and then the full verification workflow.

**Step 6: Commit fixes**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter
git add -A
git commit -m "NOJIRA-add-team-parameter

- Fix compilation errors and regenerate mocks across affected services"
```

---

### Task 9: Final Verification and Push

**Step 1: Run verification for all changed services**

Run the full verification workflow for each changed service:
- `bin-ai-manager`
- `bin-common-handler`
- `bin-api-manager`
- `bin-openapi-manager`
- Any other service found in Task 8 Step 5

**Step 2: Check for conflicts with main**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

**Step 3: Push and create PR**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-team-parameter
git push -u origin NOJIRA-add-team-parameter
```

Create PR with title `NOJIRA-add-team-parameter`.
