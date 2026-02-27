# Pipecat Flows — Team Resource Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a Team resource to bin-ai-manager that composes existing AI entities into a directed graph for Pipecat Flows-based multi-agent conversations.

**Architecture:** Team is a new CRUD resource in ai-manager with Members stored as a JSON column. Members reference existing AIs by ID. Transitions between members are driven by LLM function calling at runtime via pipecat-manager. The Go side stores/validates/serves the config; Python side executes it.

**Tech Stack:** Go, MySQL, Redis, RabbitMQ RPC, squirrel query builder, gomock

**Design doc:** `docs/plans/2026-02-26-pipecat-flows-team-design.md`

**Worktree:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-flows-team-support/`

**All file paths below are relative to:** `bin-ai-manager/`

---

### Task 1: Team Model Files

Create all model structs for Team, Member, Transition, ResolvedAI, and supporting types.

**Files:**
- Create: `models/team/main.go`
- Create: `models/team/member.go`
- Create: `models/team/resolved.go`
- Create: `models/team/field.go`
- Create: `models/team/filters.go`
- Create: `models/team/webhook.go`
- Create: `models/team/event.go`
- Reference: `models/ai/main.go` (pattern to follow)
- Reference: `models/ai/field.go`, `filters.go`, `webhook.go`, `event.go`

**Step 1: Create models/team/main.go**

```go
package team

import (
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/identity"
)

// Team represents a group of AI members organized as a directed graph.
// Each member is backed by an existing AI config, and transitions between
// members are driven by LLM function calling at runtime.
type Team struct {
	identity.Identity

	Name          string    `json:"name,omitempty" db:"name"`
	Detail        string    `json:"detail,omitempty" db:"detail"`
	StartMemberID uuid.UUID `json:"start_member_id,omitempty" db:"start_member_id,uuid"`
	Members       []Member  `json:"members,omitempty" db:"members,json"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
```

Check the exact import path for `identity` by reading `models/ai/main.go` — use the same import path.

**Step 2: Create models/team/member.go**

```go
package team

import "github.com/gofrs/uuid"

// Member represents a node in the team graph, backed by an existing AI config.
type Member struct {
	ID          uuid.UUID    `json:"id,omitempty"`
	Name        string       `json:"name,omitempty"`
	AIID        uuid.UUID    `json:"ai_id,omitempty"`
	Transitions []Transition `json:"transitions,omitempty"`
}

// Transition defines an LLM function that triggers a switch to another member.
type Transition struct {
	FunctionName string    `json:"function_name,omitempty"`
	Description  string    `json:"description,omitempty"`
	NextMemberID uuid.UUID `json:"next_member_id,omitempty"`
}
```

**Step 3: Create models/team/resolved.go**

```go
package team

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/identity"
	"monorepo/bin-ai-manager/models/tool"
)

// ResolvedAI is a stripped-down version of ai.AI for passing to pipecat-manager.
// It omits EngineKey (sensitive) but includes Identity for logging/tracing.
type ResolvedAI struct {
	identity.Identity

	EngineModel ai.EngineModel  `json:"engine_model,omitempty"`
	EngineData  map[string]any  `json:"engine_data,omitempty"`
	InitPrompt  string          `json:"init_prompt,omitempty"`
	TTSType     ai.TTSType      `json:"tts_type,omitempty"`
	TTSVoiceID  string          `json:"tts_voice_id,omitempty"`
	STTType     ai.STTType      `json:"stt_type,omitempty"`
	ToolNames   []tool.ToolName `json:"tool_names,omitempty"`
}

// ResolvedMember is a member with its AI config and tools fully resolved.
type ResolvedMember struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	AI          ResolvedAI   `json:"ai"`
	Tools       []tool.Tool  `json:"tools"`
	Transitions []Transition `json:"transitions"`
}

// ResolvedTeam carries the fully resolved team config across the RPC boundary.
type ResolvedTeam struct {
	ID            uuid.UUID        `json:"id"`
	StartMemberID uuid.UUID        `json:"start_member_id"`
	Members       []ResolvedMember `json:"members"`
}

// ConvertResolvedAI builds a ResolvedAI from an ai.AI, stripping EngineKey.
func ConvertResolvedAI(a *ai.AI) ResolvedAI {
	return ResolvedAI{
		Identity:    a.Identity,
		EngineModel: a.EngineModel,
		EngineData:  a.EngineData,
		InitPrompt:  a.InitPrompt,
		TTSType:     a.TTSType,
		TTSVoiceID:  a.TTSVoiceID,
		STTType:     a.STTType,
		ToolNames:   a.ToolNames,
	}
}
```

Check the exact import path for `identity` — it may be `commonidentity` depending on the package. Match what `models/ai/main.go` uses.

**Step 4: Create models/team/field.go**

```go
package team

// Field represents a database field name for type-safe updates.
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

**Step 5: Create models/team/filters.go**

```go
package team

import "github.com/gofrs/uuid"

// FieldStruct defines filterable fields for Team list queries.
type FieldStruct struct {
	CustomerID uuid.UUID `filter:"customer_id"`
	Name       string    `filter:"name"`
	Deleted    bool      `filter:"deleted"`
}
```

**Step 6: Create models/team/webhook.go**

Follow the exact pattern from `models/ai/webhook.go` — use `commonidentity.Identity` (check the actual import alias used in the AI webhook).

```go
package team

import (
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// WebhookMessage is the external-facing representation of a Team.
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

// ConvertWebhookMessage converts the internal Team to an external WebhookMessage.
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

**Step 7: Create models/team/event.go**

```go
package team

const (
	EventTypeCreated string = "team_created"
	EventTypeUpdated string = "team_updated"
	EventTypeDeleted string = "team_deleted"
)
```

**Step 8: Verify models compile**

Run: `cd bin-ai-manager && go build ./models/team/...`
Expected: Clean build, no errors.

**Step 9: Commit**

```
git add bin-ai-manager/models/team/
git commit -m "NOJIRA-add-pipecat-flows-team-support

- bin-ai-manager: Add Team, Member, Transition model structs
- bin-ai-manager: Add ResolvedAI, ResolvedMember, ResolvedTeam for RPC
- bin-ai-manager: Add field constants, filters, webhook, event types"
```

---

### Task 2: DB Handler — Interface + Implementation

Add Team CRUD methods to the DBHandler interface and implement them.

**Files:**
- Modify: `pkg/dbhandler/main.go` (add interface methods)
- Create: `pkg/dbhandler/team.go` (implementation)
- Reference: `pkg/dbhandler/ai.go` (pattern to follow)

**Step 1: Add Team methods to DBHandler interface in pkg/dbhandler/main.go**

Add these methods to the existing `DBHandler` interface:

```go
// Team
TeamCreate(ctx context.Context, t *team.Team) error
TeamDelete(ctx context.Context, id uuid.UUID) error
TeamGet(ctx context.Context, id uuid.UUID) (*team.Team, error)
TeamList(ctx context.Context, size uint64, token string, filters map[team.Field]any) ([]*team.Team, error)
TeamUpdate(ctx context.Context, id uuid.UUID, fields map[team.Field]any) error
```

Add the import for `"monorepo/bin-ai-manager/models/team"`.

**Step 2: Create pkg/dbhandler/team.go**

Follow the exact pattern from `pkg/dbhandler/ai.go`. Key patterns:
- Table name constant: `teamTable = "ai_teams"`
- `TeamCreate`: `PrepareFields()` → `sq.Insert().SetMap()` → `h.db.Exec()` → update cache
- `TeamGet`: Try cache → fallback DB → set cache
- `TeamList`: `GetDBFields()` → `sq.Select()` → `ApplyFields()` → scan loop → **init slice as `[]*team.Team{}`** (NOT nil)
- `TeamUpdate`: Convert `team.Field` map → `PrepareFields()` → `sq.Update().SetMap()` → update cache
- `TeamDelete`: Soft delete via `sq.Update()` setting `tm_delete` → update cache

```go
package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-ai-manager/models/team"
)

const teamTable = "ai_teams"

func (h *handler) TeamCreate(ctx context.Context, t *team.Team) error {
	t.TMCreate = h.utilHandler.TimeNow()
	t.TMUpdate = nil
	t.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(t)
	if err != nil {
		return fmt.Errorf("could not prepare the fields. err: %v", err)
	}

	query, args, err := sq.Insert(teamTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not create query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. err: %v", err)
	}

	_ = h.teamUpdateToCache(ctx, t.ID)

	return nil
}

func (h *handler) TeamDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	query, args, err := sq.Update(teamTable).SetMap(map[string]any{
		"tm_update": ts,
		"tm_delete": ts,
	}).Where(sq.Eq{"id": id.Bytes()}).ToSql()
	if err != nil {
		return fmt.Errorf("could not create query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. err: %v", err)
	}

	_ = h.teamUpdateToCache(ctx, id)

	return nil
}

func (h *handler) TeamGet(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	res, err := h.teamGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.teamGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.teamSetToCache(ctx, res)

	return res, nil
}

func (h *handler) teamGetFromDB(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	cols := commondatabasehandler.GetDBFields(team.Team{})

	query, args, err := sq.Select(cols...).
		From(teamTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not create query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &team.Team{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan. err: %v", err)
	}

	return res, nil
}

func (h *handler) TeamList(ctx context.Context, size uint64, token string, filters map[team.Field]any) ([]*team.Team, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(team.Team{})

	builder := sq.Select(cols...).
		From(teamTable).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size)

	// Apply filters
	tmpFilters := make(map[string]any)
	for k, v := range filters {
		tmpFilters[string(k)] = v
	}

	builder, err := commondatabasehandler.ApplyFields(builder, tmpFilters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not create query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*team.Team{}
	for rows.Next() {
		t := &team.Team{}
		if err := commondatabasehandler.ScanRow(rows, t); err != nil {
			return nil, fmt.Errorf("could not scan. err: %v", err)
		}
		res = append(res, t)
	}

	return res, nil
}

func (h *handler) TeamUpdate(ctx context.Context, id uuid.UUID, fields map[team.Field]any) error {
	updateFields := make(map[string]any)
	for k, v := range fields {
		updateFields[string(k)] = v
	}
	updateFields[string(team.FieldTMUpdate)] = h.utilHandler.TimeNow()

	preparedFields, err := commondatabasehandler.PrepareFields(updateFields)
	if err != nil {
		return fmt.Errorf("could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Update(teamTable).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not create query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. err: %v", err)
	}

	_ = h.teamUpdateToCache(ctx, id)

	return nil
}

// Cache helpers
func (h *handler) teamGetFromCache(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	return h.cache.TeamGet(ctx, id)
}

func (h *handler) teamSetToCache(ctx context.Context, data *team.Team) error {
	return h.cache.TeamSet(ctx, data)
}

func (h *handler) teamUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.teamGetFromDB(ctx, id)
	if err != nil {
		return err
	}
	return h.teamSetToCache(ctx, res)
}
```

Verify the exact patterns by reading `pkg/dbhandler/ai.go` — match import paths, error message format, and cache helper naming.

**Step 3: Verify compilation**

Run: `cd bin-ai-manager && go build ./pkg/dbhandler/...`
Expected: May fail on missing cache methods — that's expected, we add those next.

**Step 4: Commit**

```
git add bin-ai-manager/pkg/dbhandler/team.go bin-ai-manager/pkg/dbhandler/main.go
git commit -m "NOJIRA-add-pipecat-flows-team-support

- bin-ai-manager: Add Team CRUD methods to DBHandler interface
- bin-ai-manager: Implement Team DB operations with cache integration"
```

---

### Task 3: Cache Handler

Add Team cache methods.

**Files:**
- Modify: `pkg/cachehandler/main.go` (add interface methods)
- Create: `pkg/cachehandler/team.go` (implementation)
- Reference: `pkg/cachehandler/ai.go` (pattern to follow)

**Step 1: Add Team methods to CacheHandler interface in pkg/cachehandler/main.go**

```go
// Team
TeamGet(ctx context.Context, id uuid.UUID) (*team.Team, error)
TeamSet(ctx context.Context, data *team.Team) error
```

Add the import for the team model package.

**Step 2: Create pkg/cachehandler/team.go**

Follow the exact pattern from `pkg/cachehandler/ai.go`:

```go
package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/team"
)

const teamPrefix = "team:"

func (h *handler) TeamGet(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	key := fmt.Sprintf("%s%s", teamPrefix, id.String())

	val, err := h.Cache.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var res team.Team
	if err := json.Unmarshal([]byte(val), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (h *handler) TeamSet(ctx context.Context, data *team.Team) error {
	key := fmt.Sprintf("%s%s", teamPrefix, data.ID.String())

	val, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := h.Cache.Set(ctx, key, val, cacheTimeout).Err(); err != nil {
		return err
	}

	return nil
}
```

Check `pkg/cachehandler/ai.go` for the exact `cacheTimeout` constant and import patterns.

**Step 3: Regenerate mocks**

Run: `cd bin-ai-manager && go generate ./pkg/dbhandler/... ./pkg/cachehandler/...`
Expected: Mocks regenerated for both interfaces.

**Step 4: Verify compilation**

Run: `cd bin-ai-manager && go build ./...`
Expected: Build should pass (or fail only on listenhandler/aicallhandler which we haven't modified yet).

**Step 5: Commit**

```
git add bin-ai-manager/pkg/cachehandler/
git commit -m "NOJIRA-add-pipecat-flows-team-support

- bin-ai-manager: Add Team cache methods (TeamGet, TeamSet)"
```

---

### Task 4: Team Handler — Interface, Validation, Business Logic

Create the teamhandler package with validation and CRUD operations.

**Files:**
- Create: `pkg/teamhandler/main.go` (interface + constructor)
- Create: `pkg/teamhandler/validation.go` (validation rules 1-11)
- Create: `pkg/teamhandler/handler.go` (business logic)
- Reference: `pkg/aihandler/main.go`, `pkg/aihandler/db.go` (patterns)

**Step 1: Create pkg/teamhandler/main.go**

```go
//go:generate mockgen -package teamhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

package teamhandler

import (
	"context"

	"github.com/gofrs/uuid"

	commonnotify "monorepo/bin-common-handler/pkg/notifyhandler"
	commonrequest "monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/dbhandler"

	utilhandler "monorepo/bin-common-handler/pkg/utilhandler"
)

// TeamHandler provides CRUD operations for Team resources.
type TeamHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []team.Member) (*team.Team, error)
	Get(ctx context.Context, id uuid.UUID) (*team.Team, error)
	List(ctx context.Context, size uint64, token string, filters map[team.Field]any) ([]*team.Team, error)
	Delete(ctx context.Context, id uuid.UUID) (*team.Team, error)
	Update(ctx context.Context, id uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []team.Member) (*team.Team, error)
}

type teamHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    commonrequest.RequestHandler
	notifyHandler commonnotify.NotifyHandler
	db            dbhandler.DBHandler
}

func NewTeamHandler(
	reqHandler commonrequest.RequestHandler,
	notifyHandler commonnotify.NotifyHandler,
	db dbhandler.DBHandler,
) TeamHandler {
	return &teamHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
		db:            db,
	}
}
```

Check the exact import aliases used in `pkg/aihandler/main.go` for `requesthandler`, `notifyhandler`, and `utilhandler`.

**Step 2: Create pkg/teamhandler/validation.go**

```go
package teamhandler

import (
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/models/tool"
)

// validateTeam checks all 11 validation rules.
func validateTeam(startMemberID uuid.UUID, members []team.Member) error {
	// Rule 10: Members list must not be empty
	if len(members) == 0 {
		return fmt.Errorf("members list must not be empty")
	}

	// Rule 1: StartMemberID must not be uuid.Nil
	if startMemberID == uuid.Nil {
		return fmt.Errorf("start_member_id must not be empty")
	}

	// Build member ID set for lookups
	memberIDs := make(map[uuid.UUID]bool)
	for _, m := range members {
		// Rule 4: Member.ID must not be uuid.Nil
		if m.ID == uuid.Nil {
			return fmt.Errorf("member id must not be empty")
		}

		// Rule 3: Member IDs must be unique
		if memberIDs[m.ID] {
			return fmt.Errorf("duplicate member id: %s", m.ID)
		}
		memberIDs[m.ID] = true
	}

	// Rule 2: StartMemberID must exist in Members
	if !memberIDs[startMemberID] {
		return fmt.Errorf("start_member_id %s not found in members", startMemberID)
	}

	// Build reserved tool names set
	reservedNames := make(map[string]bool)
	for _, tn := range tool.AllToolNames {
		reservedNames[string(tn)] = true
	}

	for _, m := range members {
		// Rule 11: Member.Name must not be empty
		if m.Name == "" {
			return fmt.Errorf("member name must not be empty for member %s", m.ID)
		}

		// Rule 5: Member.AIID must not be uuid.Nil
		if m.AIID == uuid.Nil {
			return fmt.Errorf("member ai_id must not be empty for member %s", m.ID)
		}

		// Check transitions
		fnNames := make(map[string]bool)
		for _, t := range m.Transitions {
			// Rule 8: FunctionName must not collide with reserved tool names
			if reservedNames[t.FunctionName] {
				return fmt.Errorf("transition function_name %q collides with reserved tool name for member %s", t.FunctionName, m.ID)
			}

			// Rule 9: FunctionName must be unique within a member
			if fnNames[t.FunctionName] {
				return fmt.Errorf("duplicate transition function_name %q for member %s", t.FunctionName, m.ID)
			}
			fnNames[t.FunctionName] = true

			// Rule 7: NextMemberID must reference an existing member
			if !memberIDs[t.NextMemberID] {
				return fmt.Errorf("transition next_member_id %s not found in members for member %s", t.NextMemberID, m.ID)
			}
		}
	}

	return nil
}
```

Note: Rule 6 (AIID references existing AI) is checked in the handler's Create/Update methods since it requires DB access.

**Step 3: Write validation tests — pkg/teamhandler/validation_test.go**

```go
package teamhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"monorepo/bin-ai-manager/models/team"
)

func Test_validateTeam(t *testing.T) {
	memberA := uuid.Must(uuid.NewV4())
	memberB := uuid.Must(uuid.NewV4())
	aiA := uuid.Must(uuid.NewV4())
	aiB := uuid.Must(uuid.NewV4())

	validMembers := []team.Member{
		{
			ID:   memberA,
			Name: "Greeter",
			AIID: aiA,
			Transitions: []team.Transition{
				{FunctionName: "transfer_to_b", Description: "Go to B", NextMemberID: memberB},
			},
		},
		{
			ID:   memberB,
			Name: "Specialist",
			AIID: aiB,
			Transitions: []team.Transition{
				{FunctionName: "transfer_to_a", Description: "Go to A", NextMemberID: memberA},
			},
		},
	}

	tests := []struct {
		name          string
		startMemberID uuid.UUID
		members       []team.Member
		wantErr       bool
		errContains   string
	}{
		{
			name:          "valid team",
			startMemberID: memberA,
			members:       validMembers,
		},
		{
			name:          "empty members",
			startMemberID: memberA,
			members:       []team.Member{},
			wantErr:       true,
			errContains:   "members list must not be empty",
		},
		{
			name:          "nil start member id",
			startMemberID: uuid.Nil,
			members:       validMembers,
			wantErr:       true,
			errContains:   "start_member_id must not be empty",
		},
		{
			name:          "start member id not in members",
			startMemberID: uuid.Must(uuid.NewV4()),
			members:       validMembers,
			wantErr:       true,
			errContains:   "not found in members",
		},
		{
			name:          "duplicate member ids",
			startMemberID: memberA,
			members: []team.Member{
				{ID: memberA, Name: "A", AIID: aiA},
				{ID: memberA, Name: "B", AIID: aiB},
			},
			wantErr:     true,
			errContains: "duplicate member id",
		},
		{
			name:          "nil member id",
			startMemberID: memberA,
			members: []team.Member{
				{ID: uuid.Nil, Name: "A", AIID: aiA},
			},
			wantErr:     true,
			errContains: "member id must not be empty",
		},
		{
			name:          "nil member ai_id",
			startMemberID: memberA,
			members: []team.Member{
				{ID: memberA, Name: "A", AIID: uuid.Nil},
			},
			wantErr:     true,
			errContains: "member ai_id must not be empty",
		},
		{
			name:          "empty member name",
			startMemberID: memberA,
			members: []team.Member{
				{ID: memberA, Name: "", AIID: aiA},
			},
			wantErr:     true,
			errContains: "member name must not be empty",
		},
		{
			name:          "transition function name collides with reserved tool",
			startMemberID: memberA,
			members: []team.Member{
				{
					ID: memberA, Name: "A", AIID: aiA,
					Transitions: []team.Transition{
						{FunctionName: "connect_call", Description: "bad", NextMemberID: memberA},
					},
				},
			},
			wantErr:     true,
			errContains: "collides with reserved tool name",
		},
		{
			name:          "duplicate function name within member",
			startMemberID: memberA,
			members: []team.Member{
				{
					ID: memberA, Name: "A", AIID: aiA,
					Transitions: []team.Transition{
						{FunctionName: "do_thing", Description: "first", NextMemberID: memberA},
						{FunctionName: "do_thing", Description: "second", NextMemberID: memberA},
					},
				},
			},
			wantErr:     true,
			errContains: "duplicate transition function_name",
		},
		{
			name:          "transition next member id not found",
			startMemberID: memberA,
			members: []team.Member{
				{
					ID: memberA, Name: "A", AIID: aiA,
					Transitions: []team.Transition{
						{FunctionName: "go_nowhere", Description: "bad", NextMemberID: uuid.Must(uuid.NewV4())},
					},
				},
			},
			wantErr:     true,
			errContains: "next_member_id",
		},
		{
			name:          "same function name on different members is allowed",
			startMemberID: memberA,
			members: []team.Member{
				{
					ID: memberA, Name: "A", AIID: aiA,
					Transitions: []team.Transition{
						{FunctionName: "transfer_back", Description: "go to B", NextMemberID: memberB},
					},
				},
				{
					ID: memberB, Name: "B", AIID: aiB,
					Transitions: []team.Transition{
						{FunctionName: "transfer_back", Description: "go to A", NextMemberID: memberA},
					},
				},
			},
		},
		{
			name:          "single member no transitions",
			startMemberID: memberA,
			members: []team.Member{
				{ID: memberA, Name: "Solo", AIID: aiA},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTeam(tt.startMemberID, tt.members)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

**Step 4: Run validation tests**

Run: `cd bin-ai-manager && go test ./pkg/teamhandler/... -v -run Test_validateTeam`
Expected: All tests pass.

**Step 5: Create pkg/teamhandler/handler.go**

Follow the exact pattern from `pkg/aihandler/db.go`:

```go
package teamhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/team"
)

func (h *teamHandler) Create(ctx context.Context, customerID uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []team.Member) (*team.Team, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
	})

	// Validate team structure (rules 1-5, 7-11)
	if err := validateTeam(startMemberID, members); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Rule 6: Verify each member's AIID references an existing AI
	for _, m := range members {
		if _, err := h.db.AIGet(ctx, m.AIID); err != nil {
			return nil, fmt.Errorf("member %s references non-existent ai %s: %w", m.ID, m.AIID, err)
		}
	}

	id := h.utilHandler.UUIDCreate()
	t := &team.Team{
		Identity: team.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Name:          name,
		Detail:        detail,
		StartMemberID: startMemberID,
		Members:       members,
	}

	if err := h.db.TeamCreate(ctx, t); err != nil {
		log.Errorf("Could not create team. err: %v", err)
		return nil, errors.Wrapf(err, "could not create team")
	}

	res, err := h.db.TeamGet(ctx, t.ID)
	if err != nil {
		log.Errorf("Could not get created team. err: %v", err)
		return nil, errors.Wrapf(err, "could not get created team")
	}
	log.WithField("team", res).Debugf("Created team. team_id: %s", res.ID)

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, team.EventTypeCreated, res)
	return res, nil
}

func (h *teamHandler) Get(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	res, err := h.db.TeamGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get team")
	}
	return res, nil
}

func (h *teamHandler) List(ctx context.Context, size uint64, token string, filters map[team.Field]any) ([]*team.Team, error) {
	res, err := h.db.TeamList(ctx, size, token, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not list teams")
	}
	return res, nil
}

func (h *teamHandler) Delete(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Delete",
		"id":   id,
	})

	if err := h.db.TeamDelete(ctx, id); err != nil {
		log.Errorf("Could not delete team. err: %v", err)
		return nil, errors.Wrapf(err, "could not delete team")
	}

	res, err := h.db.TeamGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted team. err: %v", err)
		return nil, errors.Wrapf(err, "could not get deleted team")
	}
	log.WithField("team", res).Debugf("Deleted team. team_id: %s", res.ID)

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, team.EventTypeDeleted, res)
	return res, nil
}

func (h *teamHandler) Update(ctx context.Context, id uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []team.Member) (*team.Team, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Update",
		"id":   id,
	})

	// Validate team structure (rules 1-5, 7-11)
	if err := validateTeam(startMemberID, members); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Rule 6: Verify each member's AIID references an existing AI
	for _, m := range members {
		if _, err := h.db.AIGet(ctx, m.AIID); err != nil {
			return nil, fmt.Errorf("member %s references non-existent ai %s: %w", m.ID, m.AIID, err)
		}
	}

	fields := map[team.Field]any{
		team.FieldName:          name,
		team.FieldDetail:        detail,
		team.FieldStartMemberID: startMemberID,
		team.FieldMembers:       members,
	}

	if err := h.db.TeamUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update team. err: %v", err)
		return nil, errors.Wrapf(err, "could not update team")
	}

	res, err := h.db.TeamGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated team. err: %v", err)
		return nil, errors.Wrapf(err, "could not get updated team")
	}
	log.WithField("team", res).Debugf("Updated team. team_id: %s", res.ID)

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, team.EventTypeUpdated, res)
	return res, nil
}
```

Note: The `Identity` type in the Team create — check whether models use `identity.Identity` directly or another alias. Match the pattern from `pkg/aihandler/db.go`.

**Step 6: Generate mocks**

Run: `cd bin-ai-manager && go generate ./pkg/teamhandler/...`
Expected: `mock_main.go` generated.

**Step 7: Write handler tests — pkg/teamhandler/handler_test.go**

Write tests using gomock for Create, Get, List, Update, Delete following the exact pattern from existing handler tests. Key test cases:
- Create happy path: mock `AIGet` for each member's AIID, mock `TeamCreate`, mock `TeamGet`, mock `PublishWebhookEvent`
- Create with non-existent AI → `AIGet` returns error → Create fails
- Create with validation failure → fails before DB call
- Get happy path / not found
- List empty → returns `[]`
- Delete happy path with webhook event
- Update happy path with re-validation

Read existing test files in `pkg/aihandler/` for the exact gomock setup pattern.

**Step 8: Run handler tests**

Run: `cd bin-ai-manager && go test ./pkg/teamhandler/... -v`
Expected: All tests pass.

**Step 9: Commit**

```
git add bin-ai-manager/pkg/teamhandler/
git commit -m "NOJIRA-add-pipecat-flows-team-support

- bin-ai-manager: Add TeamHandler interface with CRUD operations
- bin-ai-manager: Add team validation logic (11 rules)
- bin-ai-manager: Add handler tests and validation tests"
```

---

### Task 5: Listen Handler — Routing + Endpoints

Wire the Team endpoints into the listen handler.

**Files:**
- Modify: `pkg/listenhandler/main.go` (add routes + teamHandler dependency)
- Create: `pkg/listenhandler/v1_teams.go` (endpoint handlers)
- Create: `pkg/listenhandler/models/request/teams.go` (request structs)
- Reference: `pkg/listenhandler/v1_ais.go` (pattern to follow)
- Reference: `pkg/listenhandler/models/request/ais.go`

**Step 1: Create pkg/listenhandler/models/request/teams.go**

```go
package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/team"
)

// V1DataTeamsPost is the request body for POST /v1/teams.
type V1DataTeamsPost struct {
	CustomerID    uuid.UUID     `json:"customer_id,omitempty"`
	Name          string        `json:"name,omitempty"`
	Detail        string        `json:"detail,omitempty"`
	StartMemberID uuid.UUID     `json:"start_member_id,omitempty"`
	Members       []team.Member `json:"members,omitempty"`
}

// V1DataTeamsIDPut is the request body for PUT /v1/teams/{id}.
type V1DataTeamsIDPut struct {
	Name          string        `json:"name,omitempty"`
	Detail        string        `json:"detail,omitempty"`
	StartMemberID uuid.UUID     `json:"start_member_id,omitempty"`
	Members       []team.Member `json:"members,omitempty"`
}
```

**Step 2: Add route patterns and teamHandler to pkg/listenhandler/main.go**

Add regex patterns (next to existing ones):
```go
regV1TeamsGet = regexp.MustCompile(`/v1/teams\?`)
regV1Teams    = regexp.MustCompile("/v1/teams$")
regV1TeamsID  = regexp.MustCompile("/v1/teams/" + regUUID + "$")
```

Add `teamHandler` to the `listenHandler` struct and `NewListenHandler` constructor. Match the exact pattern used for `aiHandler`.

Add route cases to `processRequest()` — follow the same ordering pattern as existing routes:
```go
case regV1TeamsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
    response, err = h.processV1TeamsGet(ctx, m)
    requestType = "/v1/teams"

case regV1Teams.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
    response, err = h.processV1TeamsPost(ctx, m)
    requestType = "/v1/teams"

case regV1TeamsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
    response, err = h.processV1TeamsIDGet(ctx, m)
    requestType = "/v1/teams/<team-id>"

case regV1TeamsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
    response, err = h.processV1TeamsIDPut(ctx, m)
    requestType = "/v1/teams/<team-id>"

case regV1TeamsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
    response, err = h.processV1TeamsIDDelete(ctx, m)
    requestType = "/v1/teams/<team-id>"
```

**Step 3: Create pkg/listenhandler/v1_teams.go**

Follow the exact pattern from `pkg/listenhandler/v1_ais.go`:

```go
package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/listenhandler/models/request"

	sock "monorepo/bin-common-handler/models/sock"
	utilhandler "monorepo/bin-common-handler/pkg/utilhandler"
)

func (h *listenHandler) processV1TeamsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1TeamsGet"})

	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse the request uri. err: %v", err)
		return simpleResponse(400), nil
	}

	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	typedFilters, err := utilhandler.ConvertFilters[team.FieldStruct, team.Field](team.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.teamHandler.List(ctx, pageSize, pageToken, typedFilters)
	if err != nil {
		log.Errorf("Could not get teams. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

func (h *listenHandler) processV1TeamsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1TeamsPost"})

	var req request.V1DataTeamsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.teamHandler.Create(ctx, req.CustomerID, req.Name, req.Detail, req.StartMemberID, req.Members)
	if err != nil {
		log.Errorf("Could not create team. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

func (h *listenHandler) processV1TeamsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1TeamsIDGet"})

	tmpID := extractUUID(m.URI)
	id, err := uuid.FromString(tmpID)
	if err != nil {
		log.Errorf("Could not parse the team id. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.teamHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get team. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

func (h *listenHandler) processV1TeamsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1TeamsIDPut"})

	tmpID := extractUUID(m.URI)
	id, err := uuid.FromString(tmpID)
	if err != nil {
		log.Errorf("Could not parse the team id. err: %v", err)
		return simpleResponse(400), nil
	}

	var req request.V1DataTeamsIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.teamHandler.Update(ctx, id, req.Name, req.Detail, req.StartMemberID, req.Members)
	if err != nil {
		log.Errorf("Could not update team. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

func (h *listenHandler) processV1TeamsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1TeamsIDDelete"})

	tmpID := extractUUID(m.URI)
	id, err := uuid.FromString(tmpID)
	if err != nil {
		log.Errorf("Could not parse the team id. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.teamHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete team. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}
```

Check how `extractUUID` works in `v1_ais.go` — match exactly. Also check if `PageSize`, `PageToken` are constants or need to be referenced.

**Step 4: Generate mocks**

Run: `cd bin-ai-manager && go generate ./pkg/listenhandler/... ./pkg/teamhandler/...`

**Step 5: Write listenhandler tests — pkg/listenhandler/v1_teams_test.go**

Follow the exact pattern from existing listenhandler tests. Test POST, GET list, GET by ID, PUT, DELETE with mocked teamHandler.

**Step 6: Run tests**

Run: `cd bin-ai-manager && go test ./pkg/listenhandler/... -v -run Teams`
Expected: All tests pass.

**Step 7: Commit**

```
git add bin-ai-manager/pkg/listenhandler/ bin-ai-manager/pkg/teamhandler/mock_main.go
git commit -m "NOJIRA-add-pipecat-flows-team-support

- bin-ai-manager: Add /v1/teams endpoints (GET, POST, PUT, DELETE)
- bin-ai-manager: Add request structs for teams API
- bin-ai-manager: Add listenhandler routing for teams"
```

---

### Task 6: AIcall Model Modifications

Add TeamID field to the AIcall model, field constants, filters, and webhook.

**Files:**
- Modify: `models/aicall/main.go` (add TeamID field)
- Modify: `models/aicall/field.go` (add FieldTeamID)
- Modify: `models/aicall/filters.go` (add TeamID filter)
- Modify: `models/aicall/webhook.go` (add TeamID + update ConvertWebhookMessage)

**Step 1: Add TeamID to models/aicall/main.go**

Add after the existing `AIID` field:
```go
TeamID uuid.UUID `json:"team_id,omitempty" db:"team_id,uuid"`
```

**Step 2: Add FieldTeamID to models/aicall/field.go**

```go
FieldTeamID Field = "team_id"
```

**Step 3: Add TeamID to models/aicall/filters.go**

```go
TeamID uuid.UUID `filter:"team_id"`
```

**Step 4: Add TeamID to models/aicall/webhook.go**

Add the field to `WebhookMessage`:
```go
TeamID uuid.UUID `json:"team_id,omitempty"`
```

Update `ConvertWebhookMessage()` to include:
```go
TeamID: h.TeamID,
```

**Step 5: Verify compilation**

Run: `cd bin-ai-manager && go build ./models/aicall/...`
Expected: Clean build.

**Step 6: Commit**

```
git add bin-ai-manager/models/aicall/
git commit -m "NOJIRA-add-pipecat-flows-team-support

- bin-ai-manager: Add TeamID field to AIcall model, field constants, filters, webhook"
```

---

### Task 7: Wiring in main.go

Wire teamHandler into the constructor chain.

**Files:**
- Modify: `cmd/ai-manager/main.go`

**Step 1: Create teamHandler and pass to constructors**

In `run()`:
```go
teamHandler := teamhandler.NewTeamHandler(requestHandler, notifyHandler, db)
```

Update `NewAIcallHandler` call — this will be done in a later phase when StartWithTeam is implemented. For now, just create teamHandler and pass it to `runListen`.

In `runListen()`:
- Add `teamHandler teamhandler.TeamHandler` parameter
- Pass it to `listenhandler.NewListenHandler()`

**Step 2: Verify full build**

Run: `cd bin-ai-manager && go build ./...`
Expected: Clean build.

**Step 3: Commit**

```
git add bin-ai-manager/cmd/
git commit -m "NOJIRA-add-pipecat-flows-team-support

- bin-ai-manager: Wire teamHandler into main.go and listenhandler"
```

---

### Task 8: Test DB Schema Script

Create the test SQL script for the ai_teams table.

**Files:**
- Create: `scripts/database_scripts_test/table_ai_teams.sql`
- Reference: Look at existing SQL scripts in the same directory for the pattern

**Step 1: Create the test SQL script**

```sql
CREATE TABLE IF NOT EXISTS ai_teams (
    id              binary(16)      NOT NULL,
    customer_id     binary(16)      NOT NULL,
    name            varchar(255)    NOT NULL DEFAULT '',
    detail          text,
    start_member_id binary(16)      NOT NULL,
    members         json,
    tm_create       datetime(6),
    tm_update       datetime(6),
    tm_delete       datetime(6),
    PRIMARY KEY (id)
);
```

Check the existing SQL scripts in the directory for exact formatting conventions (DEFAULT values, NOT NULL patterns, etc.).

**Step 2: Commit**

```
git add bin-ai-manager/scripts/
git commit -m "NOJIRA-add-pipecat-flows-team-support

- bin-ai-manager: Add test SQL schema for ai_teams table"
```

---

### Task 9: Full Verification

Run the complete verification workflow.

**Step 1: Run verification**

```bash
cd bin-ai-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass cleanly.

**Step 2: Fix any issues found by linter or tests**

Address golangci-lint findings. Common issues:
- Unused imports
- Error return values not checked
- Naming conventions

**Step 3: Final commit if any fixes were needed**

```
git add -A
git commit -m "NOJIRA-add-pipecat-flows-team-support

- bin-ai-manager: Fix lint issues from verification"
```

---

### Task 10: Alembic Migration (Create Only — Do NOT Execute)

Create the Alembic migration file for the new table and column.

**Files:**
- Create: Alembic migration in `bin-dbscheme-manager/`

**Step 1: Create migration**

```bash
cd bin-dbscheme-manager && alembic -c alembic.ini revision -m "add_ai_teams_table_and_aicalls_team_id"
```

**Step 2: Edit migration file**

Add to `upgrade()`:
```python
op.execute("""
    CREATE TABLE IF NOT EXISTS ai_teams (
        id              binary(16)      NOT NULL,
        customer_id     binary(16)      NOT NULL,
        name            varchar(255)    NOT NULL DEFAULT '',
        detail          text,
        start_member_id binary(16)      NOT NULL,
        members         json,
        tm_create       datetime(6),
        tm_update       datetime(6),
        tm_delete       datetime(6),
        PRIMARY KEY (id)
    )
""")

op.execute("""
    ALTER TABLE ai_aicalls ADD COLUMN team_id binary(16) DEFAULT NULL
""")
```

Add to `downgrade()`:
```python
op.execute("ALTER TABLE ai_aicalls DROP COLUMN team_id")
op.execute("DROP TABLE IF EXISTS ai_teams")
```

**CRITICAL: Do NOT run `alembic upgrade`. Only create the migration file.**

**Step 3: Commit**

```
git add bin-dbscheme-manager/
git commit -m "NOJIRA-add-pipecat-flows-team-support

- bin-dbscheme-manager: Add migration for ai_teams table and ai_aicalls.team_id column"
```

---

## Summary

| Task | Description | Depends On |
|------|-------------|------------|
| 1 | Team model files | — |
| 2 | DB handler interface + implementation | 1 |
| 3 | Cache handler | 1 |
| 4 | Team handler + validation + tests | 2, 3 |
| 5 | Listen handler + endpoints + tests | 4 |
| 6 | AIcall model modifications | 1 |
| 7 | Main.go wiring | 4, 5 |
| 8 | Test DB schema script | — |
| 9 | Full verification | All above |
| 10 | Alembic migration (create only) | — |

## Not In This Plan (Later Phases)

- `StartWithTeam` method in aicallHandler (requires bin-common-handler RPC change)
- `PipecatV1PipecatcallStart` signature change (bin-common-handler + 30+ services re-vendor)
- pipecat-manager Pipecat Flows integration
- OpenAPI schema updates (bin-openapi-manager)
- API manager service handler (bin-api-manager)
- flow-manager team_id support
- RST documentation
