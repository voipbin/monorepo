# ai_aicall_participants Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the `ai_aicall_participants` table and all write/query logic so any AIcall's full set of participating AI agents can be efficiently looked up by `ai_id` or `aicall_id`.

**Architecture:** A new `participanthandler` package owns the participant concept (one `Create` method using `ON DUPLICATE KEY UPDATE` for silent dedup). Two write sites trigger it: AIcall creation in `aicallhandler/start.go`, and team-member switches in `messagehandler/event.go`. Both treat the write as best-effort (log warning, never block the parent operation). DI is injected top-down through constructor parameters.

**Tech Stack:** Go, gomock (go.uber.org/mock), sqlite3 in-memory tests, MySQL (BINARY(16) UUIDs), Alembic (Python) for schema migrations.

---

## File Map

| Action | Path |
|---|---|
| Create | `bin-dbscheme-manager/bin-manager/main/versions/<rev>_ai_aicall_participants_create_table.py` |
| Create | `bin-ai-manager/scripts/database_scripts_test/table_ai_aicall_participants.sql` |
| Create | `bin-ai-manager/pkg/dbhandler/participant.go` |
| Create | `bin-ai-manager/pkg/dbhandler/participant_test.go` |
| Modify | `bin-ai-manager/pkg/dbhandler/main.go` — add `ParticipantCreate` to interface |
| Regen  | `bin-ai-manager/pkg/dbhandler/mock_main.go` — via `go generate ./...` |
| Create | `bin-ai-manager/pkg/participanthandler/main.go` |
| Create | `bin-ai-manager/pkg/participanthandler/db.go` |
| Create | `bin-ai-manager/pkg/participanthandler/main_test.go` |
| Regen  | `bin-ai-manager/pkg/participanthandler/mock_main.go` — via `go generate ./...` |
| Modify | `bin-ai-manager/pkg/aicallhandler/main.go` — add `participantHandler` field + param |
| Modify | `bin-ai-manager/pkg/aicallhandler/start.go` — two write sites |
| Modify | `bin-ai-manager/pkg/messagehandler/main.go` — add `participantHandler` field + param |
| Modify | `bin-ai-manager/pkg/messagehandler/event.go` — one write site |
| Modify | `bin-ai-manager/cmd/ai-manager/main.go` — wire `participanthandler.New(db)` |
| Modify | `bin-ai-manager/cmd/ai-control/main.go` — pass `nil` for participantHandler |

---

### Task 1: Alembic migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<rev>_ai_aicall_participants_create_table.py`

- [ ] **Step 1: Generate the migration file**

```bash
cd bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "ai_aicall_participants_create_table"
```

Expected: a new file `main/versions/<timestamp>_ai_aicall_participants_create_table.py` with stub `upgrade()` / `downgrade()` functions.

- [ ] **Step 2: Fill in upgrade() and downgrade()**

Open the generated file and replace the stub bodies:

```python
def upgrade():
    op.execute("""
        CREATE TABLE ai_aicall_participants (
            ai_id      BINARY(16) NOT NULL,
            aicall_id  BINARY(16) NOT NULL,
            tm_create  DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
            PRIMARY KEY (ai_id, aicall_id),
            INDEX idx_ai_aicall_participants_aicall_id (aicall_id)
        )
    """)


def downgrade():
    op.execute("DROP TABLE ai_aicall_participants")
```

- [ ] **Step 3: Commit the migration file**

```bash
git add bin-dbscheme-manager/bin-manager/main/versions/
git commit -m "NOJIRA-Add-aicall-participants-table

- bin-dbscheme-manager: Add Alembic migration for ai_aicall_participants table"
```

---

### Task 2: SQLite test SQL and dbhandler ParticipantCreate

**Files:**
- Create: `bin-ai-manager/scripts/database_scripts_test/table_ai_aicall_participants.sql`
- Create: `bin-ai-manager/pkg/dbhandler/participant_test.go`
- Create: `bin-ai-manager/pkg/dbhandler/participant.go`
- Modify: `bin-ai-manager/pkg/dbhandler/main.go`
- Regen:  `bin-ai-manager/pkg/dbhandler/mock_main.go`

- [ ] **Step 1: Write the failing test**

Create `bin-ai-manager/pkg/dbhandler/participant_test.go`:

```go
package dbhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/pkg/cachehandler"
)

func Test_ParticipantCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type input struct {
		aicallID uuid.UUID
		aiID     uuid.UUID
	}

	tests := []struct {
		name      string
		input     input
		expectErr bool
	}{
		{
			name: "creates participant successfully",
			input: input{
				aicallID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				aiID:     uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			},
			expectErr: false,
		},
		{
			name: "duplicate insert is silently ignored",
			input: input{
				aicallID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				aiID:     uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			},
			expectErr: false,
		},
	}

	h := handler{
		db: dbTest,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = mockCache
			err := h.ParticipantCreate(context.Background(), tt.input.aicallID, tt.input.aiID)
			if tt.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
cd bin-ai-manager
go test ./pkg/dbhandler/ -run Test_ParticipantCreate -v
```

Expected: FAIL — `h.ParticipantCreate undefined` (method not yet implemented).

- [ ] **Step 3: Create the SQLite test schema file**

Create `bin-ai-manager/scripts/database_scripts_test/table_ai_aicall_participants.sql`:

```sql
create table ai_aicall_participants (
  ai_id      binary(16) not null,
  aicall_id  binary(16) not null,
  tm_create  datetime(6),
  primary key (ai_id, aicall_id)
);

create index idx_ai_aicall_participants_aicall_id on ai_aicall_participants(aicall_id);
```

Note: SQLite doesn't support `DEFAULT CURRENT_TIMESTAMP(6)` with fractional seconds the same way as MySQL, so we omit the `NOT NULL DEFAULT` clause in the test file. The migration file uses the real MySQL syntax.

- [ ] **Step 4: Add `ParticipantCreate` to the DBHandler interface**

In `bin-ai-manager/pkg/dbhandler/main.go`, add to the `DBHandler` interface (at the end, before the closing brace):

```go
	// Participant
	ParticipantCreate(ctx context.Context, aicallID uuid.UUID, aiID uuid.UUID) error
```

- [ ] **Step 5: Create `pkg/dbhandler/participant.go`**

```go
package dbhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
)

const participantTable = "ai_aicall_participants"

func (h *handler) ParticipantCreate(ctx context.Context, aicallID uuid.UUID, aiID uuid.UUID) error {
	q := fmt.Sprintf(`INSERT INTO %s (ai_id, aicall_id, tm_create) VALUES (?, ?, NOW(6)) ON DUPLICATE KEY UPDATE tm_create = tm_create`, participantTable)

	_, err := h.db.ExecContext(ctx, q, aiID.Bytes(), aicallID.Bytes())
	return err
}
```

- [ ] **Step 6: Regenerate the dbhandler mock**

```bash
cd bin-ai-manager
go generate ./pkg/dbhandler/...
```

Expected: `pkg/dbhandler/mock_main.go` is updated with `MockParticipantCreate` method.

- [ ] **Step 7: Run the test to confirm it passes**

```bash
cd bin-ai-manager
go test ./pkg/dbhandler/ -run Test_ParticipantCreate -v
```

Expected: PASS — both cases pass including the silent duplicate.

- [ ] **Step 8: Commit**

```bash
cd bin-ai-manager
git add scripts/database_scripts_test/table_ai_aicall_participants.sql \
        pkg/dbhandler/participant.go \
        pkg/dbhandler/participant_test.go \
        pkg/dbhandler/main.go \
        pkg/dbhandler/mock_main.go
git commit -m "NOJIRA-Add-aicall-participants-table

- bin-ai-manager: Add ParticipantCreate to DBHandler interface and implementation"
```

---

### Task 3: participanthandler package

**Files:**
- Create: `bin-ai-manager/pkg/participanthandler/main.go`
- Create: `bin-ai-manager/pkg/participanthandler/db.go`
- Create: `bin-ai-manager/pkg/participanthandler/main_test.go`
- Regen:  `bin-ai-manager/pkg/participanthandler/mock_main.go`

- [ ] **Step 1: Write the failing test**

Create `bin-ai-manager/pkg/participanthandler/main_test.go`:

```go
package participanthandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	aicallID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	aiID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	dbErr := errors.New("db error")

	tests := []struct {
		name      string
		aicallID  uuid.UUID
		aiID      uuid.UUID
		mockSetup func(mockDB *dbhandler.MockDBHandler)
		expectErr bool
	}{
		{
			name:     "creates participant successfully",
			aicallID: aicallID,
			aiID:     aiID,
			mockSetup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().ParticipantCreate(gomock.Any(), aicallID, aiID).Return(nil).Times(1)
			},
			expectErr: false,
		},
		{
			name:     "db error propagates",
			aicallID: aicallID,
			aiID:     aiID,
			mockSetup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().ParticipantCreate(gomock.Any(), aicallID, aiID).Return(dbErr).Times(1)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := dbhandler.NewMockDBHandler(mc)
			tt.mockSetup(mockDB)

			h := New(mockDB)
			err := h.Create(context.Background(), tt.aicallID, tt.aiID)
			if tt.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
cd bin-ai-manager
go test ./pkg/participanthandler/ -run Test_Create -v
```

Expected: FAIL — package `participanthandler` does not exist yet.

- [ ] **Step 3: Create `pkg/participanthandler/main.go`**

```go
package participanthandler

//go:generate mockgen -package participanthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// ParticipantHandler manages aicall participant records.
type ParticipantHandler interface {
	Create(ctx context.Context, aicallID uuid.UUID, aiID uuid.UUID) error
}

type participantHandler struct {
	db dbhandler.DBHandler
}

// New returns a new ParticipantHandler backed by db.
func New(db dbhandler.DBHandler) ParticipantHandler {
	return &participantHandler{db: db}
}
```

- [ ] **Step 4: Create `pkg/participanthandler/db.go`**

```go
package participanthandler

import (
	"context"

	"github.com/gofrs/uuid"
)

func (h *participantHandler) Create(ctx context.Context, aicallID uuid.UUID, aiID uuid.UUID) error {
	return h.db.ParticipantCreate(ctx, aicallID, aiID)
}
```

- [ ] **Step 5: Generate the mock**

```bash
cd bin-ai-manager
go generate ./pkg/participanthandler/...
```

Expected: `pkg/participanthandler/mock_main.go` is created with `MockParticipantHandler`.

- [ ] **Step 6: Run the test to confirm it passes**

```bash
cd bin-ai-manager
go test ./pkg/participanthandler/ -run Test_Create -v
```

Expected: PASS — both cases pass.

- [ ] **Step 7: Commit**

```bash
git add bin-ai-manager/pkg/participanthandler/
git commit -m "NOJIRA-Add-aicall-participants-table

- bin-ai-manager: Add participanthandler package with Create method and mock"
```

---

### Task 4: aicallhandler — inject participantHandler and write at creation

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/main.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go`

The tests in `pkg/aicallhandler/start_test.go` construct `&aicallHandler{...}` struct literals directly — adding a new field will not break them (Go allows partial struct initialization when using field names). However, you must add `MockParticipantHandler` expectations in any test that hits the write sites.

- [ ] **Step 1: Add `participantHandler` field to `aicallHandler` struct and constructor**

In `bin-ai-manager/pkg/aicallhandler/main.go`, change:

```go
// old imports — add this:
import (
    ...
    "monorepo/bin-ai-manager/pkg/participanthandler"
    ...
)

// old struct:
type aicallHandler struct {
    utilHandler   utilhandler.UtilHandler
    reqHandler    requesthandler.RequestHandler
    notifyHandler notifyhandler.NotifyHandler
    db            dbhandler.DBHandler
    aiHandler      aihandler.AIHandler
    teamHandler    teamhandler.TeamHandler
    messageHandler messagehandler.MessageHandler
}

// new struct (add field at the end):
type aicallHandler struct {
    utilHandler        utilhandler.UtilHandler
    reqHandler         requesthandler.RequestHandler
    notifyHandler      notifyhandler.NotifyHandler
    db                 dbhandler.DBHandler
    aiHandler          aihandler.AIHandler
    teamHandler        teamhandler.TeamHandler
    messageHandler     messagehandler.MessageHandler
    participantHandler participanthandler.ParticipantHandler
}
```

Change the constructor:

```go
// old signature:
func NewAIcallHandler(
    req            requesthandler.RequestHandler,
    notify         notifyhandler.NotifyHandler,
    db             dbhandler.DBHandler,
    aiHandler      aihandler.AIHandler,
    teamHandler    teamhandler.TeamHandler,
    messageHandler messagehandler.MessageHandler,
) AIcallHandler {
    return &aicallHandler{
        utilHandler:    utilhandler.NewUtilHandler(),
        reqHandler:     req,
        notifyHandler:  notify,
        db:             db,
        aiHandler:      aiHandler,
        teamHandler:    teamHandler,
        messageHandler: messageHandler,
    }
}

// new signature:
func NewAIcallHandler(
    req                requesthandler.RequestHandler,
    notify             notifyhandler.NotifyHandler,
    db                 dbhandler.DBHandler,
    aiHandler          aihandler.AIHandler,
    teamHandler        teamhandler.TeamHandler,
    messageHandler     messagehandler.MessageHandler,
    participantHandler participanthandler.ParticipantHandler,
) AIcallHandler {
    return &aicallHandler{
        utilHandler:        utilhandler.NewUtilHandler(),
        reqHandler:         req,
        notifyHandler:      notify,
        db:                 db,
        aiHandler:          aiHandler,
        teamHandler:        teamHandler,
        messageHandler:     messageHandler,
        participantHandler: participantHandler,
    }
}
```

- [ ] **Step 2: Add participant write to `startAIcallByRealtime` in `start.go`**

In `bin-ai-manager/pkg/aicallhandler/start.go`, in `startAIcallByRealtime`, after the `h.Create` call succeeds (after `log.WithField("aicall", res).Debugf("Created aicall. aicall_id: %s", res.ID)`), add:

```go
	if h.participantHandler != nil {
		if err := h.participantHandler.Create(ctx, res.ID, a.ID); err != nil {
			log.Warnf("Could not record aicall participant. aicall_id: %s, ai_id: %s, err: %v", res.ID, a.ID, err)
		}
	}
```

- [ ] **Step 3: Add participant write to `startAIcallByMessaging` in `start.go`**

In `bin-ai-manager/pkg/aicallhandler/start.go`, in `startAIcallByMessaging`, after the `h.CreateByMessaging` call succeeds (after `log.WithField("aicall", res).Debugf("Created aicall. aicall_id: %s", res.ID)`), add:

```go
	if h.participantHandler != nil {
		if err := h.participantHandler.Create(ctx, res.ID, a.ID); err != nil {
			log.Warnf("Could not record aicall participant. aicall_id: %s, ai_id: %s, err: %v", res.ID, a.ID, err)
		}
	}
```

- [ ] **Step 4: Run the tests**

```bash
cd bin-ai-manager
go test ./pkg/aicallhandler/... -v 2>&1 | tail -30
```

Expected: PASS — existing tests are unaffected because `participantHandler` is nil in existing struct literals, and the nil guard prevents panics. Tests that exercise `startAIcallByRealtime` / `startAIcallByMessaging` should still pass because no mock expectation is required when `participantHandler == nil`.

- [ ] **Step 5: Add a test case verifying the participant write is called**

Open `bin-ai-manager/pkg/aicallhandler/start_test.go`. In the `Test_startReferenceTypeCall` function, find an existing passing test case that reaches the `h.Create` success path (e.g., the "normal" case). Add a parallel test case that injects a `MockParticipantHandler` and asserts `Create` is called once with the correct `aiID`.

Add to the test's `mocks` struct:

```go
participant *participanthandler.MockParticipantHandler
```

Add to the mock setup within the new test case:

```go
mc := gomock.NewController(t)
defer mc.Finish()

mockParticipant := participanthandler.NewMockParticipantHandler(mc)
mockParticipant.EXPECT().Create(gomock.Any(), /* aicall res.ID */ gomock.Any(), /* a.ID */ uuid.FromStringOrNil("...")).Return(nil).Times(1)

h := &aicallHandler{
    // ... all existing fields ...
    participantHandler: mockParticipant,
}
```

Run the new test case:

```bash
cd bin-ai-manager
go test ./pkg/aicallhandler/... -run Test_startReferenceTypeCall -v 2>&1 | tail -20
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/main.go \
        bin-ai-manager/pkg/aicallhandler/start.go \
        bin-ai-manager/pkg/aicallhandler/start_test.go
git commit -m "NOJIRA-Add-aicall-participants-table

- bin-ai-manager: Inject participantHandler into aicallHandler; write participant on AIcall creation"
```

---

### Task 5: messagehandler — inject participantHandler and write on member switch

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/main.go`
- Modify: `bin-ai-manager/pkg/messagehandler/event.go`

- [ ] **Step 1: Add `participantHandler` field to `messageHandler` struct and constructor**

In `bin-ai-manager/pkg/messagehandler/main.go`, change:

```go
// add to imports:
"monorepo/bin-ai-manager/pkg/participanthandler"

// old struct:
type messageHandler struct {
    utilHandler   utilhandler.UtilHandler
    notifyHandler notifyhandler.NotifyHandler
    db            dbhandler.DBHandler
    reqHandler    requesthandler.RequestHandler

    engineOpenaiHandler     engine_openai_handler.EngineOpenaiHandler
    engineDialogflowHandler engine_dialogflow_handler.EngineDialogflowHandler
}

// new struct:
type messageHandler struct {
    utilHandler   utilhandler.UtilHandler
    notifyHandler notifyhandler.NotifyHandler
    db            dbhandler.DBHandler
    reqHandler    requesthandler.RequestHandler

    engineOpenaiHandler     engine_openai_handler.EngineOpenaiHandler
    engineDialogflowHandler engine_dialogflow_handler.EngineDialogflowHandler
    participantHandler      participanthandler.ParticipantHandler
}
```

Change the constructor:

```go
// old:
func NewMessageHandler(
    reqHandler              requesthandler.RequestHandler,
    notifyHandler           notifyhandler.NotifyHandler,
    db                      dbhandler.DBHandler,
    engineOpenaiHandler     engine_openai_handler.EngineOpenaiHandler,
    engineDialogflowHandler engine_dialogflow_handler.EngineDialogflowHandler,
) MessageHandler {
    return &messageHandler{
        reqHandler:              reqHandler,
        utilHandler:             utilhandler.NewUtilHandler(),
        notifyHandler:           notifyHandler,
        db:                      db,
        engineOpenaiHandler:     engineOpenaiHandler,
        engineDialogflowHandler: engineDialogflowHandler,
    }
}

// new:
func NewMessageHandler(
    reqHandler              requesthandler.RequestHandler,
    notifyHandler           notifyhandler.NotifyHandler,
    db                      dbhandler.DBHandler,
    engineOpenaiHandler     engine_openai_handler.EngineOpenaiHandler,
    engineDialogflowHandler engine_dialogflow_handler.EngineDialogflowHandler,
    participantHandler      participanthandler.ParticipantHandler,
) MessageHandler {
    return &messageHandler{
        reqHandler:              reqHandler,
        utilHandler:             utilhandler.NewUtilHandler(),
        notifyHandler:           notifyHandler,
        db:                      db,
        engineOpenaiHandler:     engineOpenaiHandler,
        engineDialogflowHandler: engineDialogflowHandler,
        participantHandler:      participantHandler,
    }
}
```

- [ ] **Step 2: Add participant write to `EventPMTeamMemberSwitched` in `event.go`**

In `bin-ai-manager/pkg/messagehandler/event.go`, in `EventPMTeamMemberSwitched`, after the `h.Create` notification message call (after `log.WithField("message", tmp).Debugf("Created member-switched notification message.")`), add:

```go
	if h.participantHandler != nil {
		if activeAIID == uuid.Nil {
			log.Warnf("Could not resolve AI ID for new member — skipping participant write. aicall_id: %s, member_id: %s",
				evt.PipecatcallReferenceID, evt.ToMember.ID)
		} else if err := h.participantHandler.Create(ctx, evt.PipecatcallReferenceID, activeAIID); err != nil {
			log.Warnf("Could not record aicall participant. aicall_id: %s, ai_id: %s, err: %v",
				evt.PipecatcallReferenceID, activeAIID, err)
		}
	}
```

Note: `activeAIID` is already resolved at line 343 via `h.resolveTeamMemberAIID`. We reuse the same value here.

- [ ] **Step 3: Run the tests**

```bash
cd bin-ai-manager
go test ./pkg/messagehandler/... -v 2>&1 | tail -30
```

Expected: PASS — existing tests pass because `participantHandler` is nil in most existing struct literals, and the nil guard prevents panics.

- [ ] **Step 4: Add test cases for the member-switch participant write**

In `bin-ai-manager/pkg/messagehandler/event_test.go`, find `TestEventPMTeamMemberSwitched` (or the relevant test function for `EventPMTeamMemberSwitched`). Add two new sub-cases:

**Case A: participantHandler.Create called when activeAIID resolves successfully**

```go
func TestEventPMTeamMemberSwitched_participant_written(t *testing.T) {
    mc := gomock.NewController(t)
    defer mc.Finish()

    aicallID := uuid.Must(uuid.NewV4())
    memberID := uuid.Must(uuid.NewV4())
    aiID     := uuid.Must(uuid.NewV4())

    evt := &pmmessage.MemberSwitchedEvent{
        PipecatcallReferenceID: aicallID,
        CustomerID:             uuid.Must(uuid.NewV4()),
        ActiveflowID:           uuid.Must(uuid.NewV4()),
        ToMember: pmmessage.MemberInfo{ID: memberID},
    }

    mockDB      := dbhandler.NewMockDBHandler(mc)
    mockReq     := requesthandler.NewMockRequestHandler(mc)
    mockNotify  := notifyhandler.NewMockNotifyHandler(mc)
    mockPart    := participanthandler.NewMockParticipantHandler(mc)

    // resolveTeamMemberAIID resolves via AIV1AIcallGet then db.TeamGet
    mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(&aicall.AIcall{
        AssistanceType: aicall.AssistanceTypeTeam,
        AssistanceID:   uuid.Must(uuid.NewV4()),
    }, nil).Times(1)
    teamID := /* same as AssistanceID from above */ uuid.Must(uuid.NewV4())
    _ = teamID // substitute real teamID above
    // NOTE: for a real test you need to use the same uuid in both places;
    // see existing TestEventPMTeamMemberSwitched for the pattern of using
    // predefined variables for related UUIDs.

    // ... set up db.TeamGet to return a team with the member, create message mock ...

    mockPart.EXPECT().Create(gomock.Any(), aicallID, aiID).Return(nil).Times(1)

    h := &messageHandler{
        db:                 mockDB,
        reqHandler:         mockReq,
        notifyHandler:      mockNotify,
        utilHandler:        utilhandler.NewUtilHandler(),
        participantHandler: mockPart,
    }
    h.EventPMTeamMemberSwitched(context.Background(), evt)
}
```

**Case B: participant write skipped when activeAIID is uuid.Nil**

```go
func TestEventPMTeamMemberSwitched_participant_skipped_when_nil_ai(t *testing.T) {
    mc := gomock.NewController(t)
    defer mc.Finish()

    aicallID := uuid.Must(uuid.NewV4())
    memberID := uuid.Must(uuid.NewV4())

    evt := &pmmessage.MemberSwitchedEvent{
        PipecatcallReferenceID: aicallID,
        CustomerID:             uuid.Must(uuid.NewV4()),
        ActiveflowID:           uuid.Must(uuid.NewV4()),
        ToMember:               pmmessage.MemberInfo{ID: memberID},
    }

    mockDB     := dbhandler.NewMockDBHandler(mc)
    mockReq    := requesthandler.NewMockRequestHandler(mc)
    mockNotify := notifyhandler.NewMockNotifyHandler(mc)
    mockPart   := participanthandler.NewMockParticipantHandler(mc)
    // resolveTeamMemberAIID returns uuid.Nil (e.g. aicall get fails)
    mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(nil, errors.New("not found")).Times(1)

    // Create notification message must still succeed
    // ... set up db.MessageCreate / db.MessageGet / notify.PublishWebhookEvent as needed ...

    // participantHandler.Create must NOT be called
    mockPart.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

    h := &messageHandler{
        db:                 mockDB,
        reqHandler:         mockReq,
        notifyHandler:      mockNotify,
        utilHandler:        utilhandler.NewUtilHandler(),
        participantHandler: mockPart,
    }
    h.EventPMTeamMemberSwitched(context.Background(), evt)
}
```

Tip: model these closely on the existing `TestEventPMTeamMemberSwitched_*` tests already in the file for the exact mock chain needed for `Create` → `MessageCreate` → `MessageGet` → `PublishWebhookEvent`.

- [ ] **Step 5: Run the new tests**

```bash
cd bin-ai-manager
go test ./pkg/messagehandler/... -run TestEventPMTeamMemberSwitched -v 2>&1 | tail -30
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add bin-ai-manager/pkg/messagehandler/main.go \
        bin-ai-manager/pkg/messagehandler/event.go \
        bin-ai-manager/pkg/messagehandler/event_test.go
git commit -m "NOJIRA-Add-aicall-participants-table

- bin-ai-manager: Inject participantHandler into messageHandler; write participant on team member switch"
```

---

### Task 6: Wire participantHandler in cmd/

**Files:**
- Modify: `bin-ai-manager/cmd/ai-manager/main.go`
- Modify: `bin-ai-manager/cmd/ai-control/main.go`

- [ ] **Step 1: Update `cmd/ai-manager/main.go`**

In the `run` function, add import and instantiate `participantHandler` before `messageHandler` and `aicallHandler`:

Add to imports:
```go
"monorepo/bin-ai-manager/pkg/participanthandler"
```

Change the handler wiring section from:

```go
messageHandler := messagehandler.NewMessageHandler(requestHandler, notifyHandler, db, engineOpenaiHandler, engineDialogflowHandler)
aicallHandler := aicallhandler.NewAIcallHandler(requestHandler, notifyHandler, db, aiHandler, teamHandler, messageHandler)
```

To:

```go
participantHandler := participanthandler.New(db)
messageHandler := messagehandler.NewMessageHandler(requestHandler, notifyHandler, db, engineOpenaiHandler, engineDialogflowHandler, participantHandler)
aicallHandler := aicallhandler.NewAIcallHandler(requestHandler, notifyHandler, db, aiHandler, teamHandler, messageHandler, participantHandler)
```

- [ ] **Step 2: Update `cmd/ai-control/main.go`**

In `initAIcallHandler`, add `nil` for participantHandler. Change:

```go
return aicallhandler.NewAIcallHandler(reqHandler, notifyHandler, dbHandler, nil, nil, nil), nil
```

To:

```go
return aicallhandler.NewAIcallHandler(reqHandler, notifyHandler, dbHandler, nil, nil, nil, nil), nil
```

No import needed — the CLI only exercises read/delete paths that never trigger participant writes. Passing `nil` is consistent with the existing pattern of `nil` for aiHandler, teamHandler, messageHandler.

- [ ] **Step 3: Verify the build**

```bash
cd bin-ai-manager
go build ./cmd/ai-manager/ ./cmd/ai-control/
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add bin-ai-manager/cmd/ai-manager/main.go \
        bin-ai-manager/cmd/ai-control/main.go
git commit -m "NOJIRA-Add-aicall-participants-table

- bin-ai-manager: Wire participantHandler in ai-manager and ai-control cmd"
```

---

### Task 7: Full verification

- [ ] **Step 1: Run the full verification workflow**

```bash
cd bin-ai-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: all steps pass with zero errors.

- [ ] **Step 2: Confirm acceptance criteria**

Check each item from the design doc:
- [ ] `ai_aicall_participants` table created via Alembic migration with correct PK and index.
- [ ] `ON DUPLICATE KEY UPDATE` silently deduplicates `(ai_id, aicall_id)` pairs; other errors propagate.
- [ ] Participant row written on AIcall creation for both `AssistanceTypeAI` and `AssistanceTypeTeam` (including `StartTask`).
- [ ] Participant row written on each team member switch using `resolveTeamMemberAIID`; skipped (with warning) when resolution returns `uuid.Nil`.
- [ ] A→B→A pattern produces exactly two rows: one for A, one for B.
- [ ] Reused conversation AIcall does not write a duplicate participant row.
- [ ] Failed participant write logs a warning but does not fail the parent operation.
- [ ] All new and modified tests pass (`go test ./...`).
- [ ] Linter passes (`golangci-lint run -v --timeout 5m`).

- [ ] **Step 3: Create the PR**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-participants-table
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
# (resolve any conflicts if found)

gh pr create \
  --title "NOJIRA-Add-aicall-participants-table" \
  --body "Add ai_aicall_participants table and participant tracking for AIcall sessions.

- bin-dbscheme-manager: Add Alembic migration creating ai_aicall_participants table with composite PK (ai_id, aicall_id) and index on aicall_id
- bin-ai-manager: Add ParticipantCreate to DBHandler interface and SQLite test schema
- bin-ai-manager: Add participanthandler package (Create method, mock, tests)
- bin-ai-manager: Inject participantHandler into aicallHandler; write participant row on AIcall creation in startAIcallByRealtime and startAIcallByMessaging
- bin-ai-manager: Inject participantHandler into messageHandler; write participant row on EventPMTeamMemberSwitched using existing resolveTeamMemberAIID helper
- bin-ai-manager: Wire participanthandler.New(db) in ai-manager cmd; pass nil in ai-control cmd (read-only CLI paths)"
```
