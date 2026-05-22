# Design: ai_aicall_participants Table

**Date:** 2026-05-22
**Branch:** NOJIRA-Add-aicall-participants-table
**Status:** Draft

---

## 1. Background & Motivation

`bin-ai-manager` tracks AI agent conversations via AIcall sessions (`ai_aicalls` table).
A Team AIcall can involve multiple AI agents (members) that switch during a single
conversation (A→B→A→C→A patterns are possible). The `ai_aicalls.current_member_id`
column only tracks the *current* active member — there is no historical record of which
AI agents participated in a given AIcall.

We need a participants table to efficiently answer: **"which AIcalls did a specific AI
agent participate in?"** This is required for per-agent evaluation aggregation in the
upcoming AI evaluation feature.

---

## 2. Goals

- Efficiently look up all AIcalls a given `ai_id` participated in.
- Efficiently look up all `ai_id`s that participated in a given AIcall.
- Silently deduplicate: A→B→A writes AI A once (second write is silently ignored).
- Keep the table minimal — no surrogate ID, no `customer_id`.

## 3. Non-Goals

- Tracking participation duration or per-span timestamps.
- Tracking by `member_id` (team-member UUID); only `ai_id` (AI config UUID) matters.
- Replacing or augmenting `ai_aicalls.current_member_id`.

---

## 4. Schema

```sql
CREATE TABLE ai_aicall_participants (
    ai_id      BINARY(16) NOT NULL,
    aicall_id  BINARY(16) NOT NULL,
    tm_create  DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (ai_id, aicall_id),
    INDEX idx_ai_aicall_participants_aicall_id (aicall_id)
);
```

### Design decisions

| Decision | Rationale |
|---|---|
| `PRIMARY KEY (ai_id, aicall_id)` | Covers uniqueness constraint + efficient `WHERE ai_id = ?` prefix scan |
| `INDEX (aicall_id)` | Enables reverse lookup: "which AIs participated in this AIcall?" |
| No surrogate `id` | Pure join/lookup table; no need for a single-column PK |
| No `customer_id` | Evaluation queries join to `ai_aicalls` or `ai_ais` for tenant scoping |
| `tm_create NOT NULL DEFAULT CURRENT_TIMESTAMP(6)` | Consistent with other tables; DB-provided value avoids nullable ordering issues |
| `INSERT INTO … ON DUPLICATE KEY UPDATE tm_create = tm_create` | Deduplicates on the PK only using the non-PK column as the no-op target; other errors (type violations, etc.) propagate normally. Preferred over `INSERT IGNORE` which silently swallows all errors |

---

## 5. New Package: `pkg/participanthandler`

A dedicated package owns the participant concept, keeping `aicallhandler` and
`messagehandler` thin.

### File layout

```
pkg/participanthandler/
├── main.go          # interface + constructor
├── db.go            # Create implementation
├── mock_main.go     # gomock-generated mock (go generate)
└── main_test.go     # table-driven unit tests
```

### Interface

```go
// ParticipantHandler manages aicall participant records.
type ParticipantHandler interface {
    Create(ctx context.Context, aicallID uuid.UUID, aiID uuid.UUID) error
}
```

### Constructor (dependency injection)

```go
func New(db dbhandler.DBHandler) ParticipantHandler
```

`utilHandler` is constructed internally via `utilhandler.NewUtilHandler()`, consistent
with the pattern used by `aicallhandler`, `messagehandler`, and other handlers in this
service.

### `db.go` — core write

```go
func (h *participantHandler) Create(ctx context.Context, aicallID, aiID uuid.UUID) error {
    // INSERT INTO ai_aicall_participants (ai_id, aicall_id)
    // VALUES (?, ?)
    // ON DUPLICATE KEY UPDATE tm_create = tm_create
}
```

A duplicate `(ai_id, aicall_id)` pair is silently ignored by the `ON DUPLICATE KEY`
clause using `tm_create` as the no-op target (consistent with §4); all other errors
propagate normally.

---

## 6. dbhandler Addition

New file `pkg/dbhandler/participant.go`:

```go
const participantTable = "ai_aicall_participants"

func (h *handler) ParticipantCreate(ctx context.Context, aicallID, aiID uuid.UUID) error
```

The `dbhandler` interface (`pkg/dbhandler/main.go`) gains `ParticipantCreate` and
`mock_main.go` is regenerated via `go generate ./...` after the interface change.

---

## 7. Write Sites

Participant rows are written at two trigger points:

### 7.1 AIcall creation (initial AI)

Both `startAIcallByRealtime` and `startAIcallByMessaging` in `pkg/aicallhandler/start.go`
call `participantHandler.Create(ctx, res.ID, a.ID)` immediately after the AIcall record
is persisted, where `a` is the resolved `*ai.AI`.

This covers all creation paths:
- `AssistanceTypeAI`: `a.ID` is the AI config UUID directly.
- `AssistanceTypeTeam`: `resolveAI` calls `resolveTeamMemberAI` → `aiHandler.Get(m.AIID)`,
  returning the full `*ai.AI` struct for the start member. `a.ID` equals `m.AIID`. No
  separate `AIID` field exists on `*ai.AI`; `a.ID` is the AI config's own primary key.
- `StartTask` / `ReferenceTypeTask`: `StartTask` calls `startAIcallByMessaging` directly,
  so participant writes happen on the same path — task AIcalls are intentionally included.

**No `uuid.Nil` guard is needed at the creation site.** `aiHandler.Get` returns an error
if the AI record is not found, causing `resolveAI` (and thus `startAIcallByRealtime` /
`startAIcallByMessaging`) to return early before the participant write is reached. A
`uuid.Nil` `a.ID` is not a reachable runtime condition at this call site.

**Reused conversation AIcalls:** When `startReferenceTypeConversation` takes the reuse
branch, it does not call `startAIcallByMessaging`; no new AIcall row is created and
the participant row already exists from the original creation. The `ON DUPLICATE KEY`
clause would be a no-op anyway. This is correct and intentional.

### 7.2 Team member switch

`messagehandler.EventPMTeamMemberSwitched` in `pkg/messagehandler/event.go` calls
`resolveTeamMemberAIID(ctx, evt.PipecatcallReferenceID, evt.ToMember.ID)` to obtain the
incoming member's `ai_id`. Note: `MemberInfo` (the event payload type) carries only
`ID`, `Name`, `EngineModel`, `TTSType`, `TTSVoiceID`, and `STTType` — it does **not**
carry `AIID`. The resolution requires a DB round-trip (AIcall → team → member walk),
using the existing `resolveTeamMemberAIID` helper already present in the handler.

After resolution the write proceeds:

```go
toMemberAIID := h.resolveTeamMemberAIID(ctx, evt.PipecatcallReferenceID, evt.ToMember.ID)
if toMemberAIID == uuid.Nil {
    log.Warnf("Could not resolve AI ID for new member — skipping participant write. aicall_id: %s, member_id: %s",
        evt.PipecatcallReferenceID, evt.ToMember.ID)
} else if err := h.participantHandler.Create(ctx, evt.PipecatcallReferenceID, toMemberAIID); err != nil {
    log.Warnf("Could not record aicall participant. aicall_id: %s, ai_id: %s, err: %v",
        evt.PipecatcallReferenceID, toMemberAIID, err)
}
```

### Error handling

Both write sites treat participant writes as **best-effort**: log a warning on error but
do not fail or roll back the parent operation. A missed participant write produces an
incomplete evaluation aggregate but must never block AIcall creation or message delivery.

`a.ID` at the creation site comes from a successful `aiHandler.Get` DB read and is never
`uuid.Nil`, so no guard is needed there.

---

## 8. Dependency Injection Changes

`participantHandler ParticipantHandler` is added to the `aicallHandler` and
`messageHandler` structs. The concrete `participanthandler.New(db)` instance is
passed in at the call sites in `cmd/` where both handlers are constructed. Both
`cmd/ai-manager/main.go` and `cmd/ai-control/main.go` construct `NewAIcallHandler`
and must be updated to pass the new `participantHandler` argument.

**`NewAIcallHandler` new signature (add `participantHandler` after existing args):**
```go
func NewAIcallHandler(
    req            requesthandler.RequestHandler,
    notify         notifyhandler.NotifyHandler,
    db             dbhandler.DBHandler,
    aiHandler      aihandler.AIHandler,
    teamHandler    teamhandler.TeamHandler,
    messageHandler messagehandler.MessageHandler,
    participantHandler participanthandler.ParticipantHandler,  // NEW
) AIcallHandler
```

Note: `utilHandler` is constructed internally via `utilhandler.NewUtilHandler()` and is
not an injected parameter.

**`NewMessageHandler` new signature (add `participantHandler` after existing args):**
```go
func NewMessageHandler(
    reqHandler              requesthandler.RequestHandler,
    notifyHandler           notifyhandler.NotifyHandler,
    db                      dbhandler.DBHandler,
    engineOpenaiHandler     engine_openai_handler.EngineOpenaiHandler,
    engineDialogflowHandler engine_dialogflow_handler.EngineDialogflowHandler,
    participantHandler      participanthandler.ParticipantHandler,  // NEW
) MessageHandler
```

Note: `utilHandler` and `aicallHandler` are not injected into `messageHandler`; `utilHandler`
is constructed internally.

After adding `ParticipantCreate` to the `dbhandler.DBHandler` interface, run
`go generate ./...` to regenerate `pkg/dbhandler/mock_main.go` and
`pkg/participanthandler/mock_main.go`.

---

## 9. Alembic Migration

New file in `bin-dbscheme-manager/bin-manager/main/versions/` generated via:

```bash
cd bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "ai_aicall_participants_create_table"
```

`upgrade()` creates the table; `downgrade()` drops it.

---

## 10. Testing Strategy

| Layer | What is tested |
|---|---|
| `dbhandler/participant_test.go` | `ParticipantCreate`: normal insert; duplicate insert (no error, 0 rows affected); DB error propagation |
| `participanthandler/main_test.go` | `Create`: normal path; error from dbhandler propagates. Note: `Create` itself has no `uuid.Nil` guard — callers are responsible. The `uuid.Nil` guard is tested at the `messagehandler` call site (see row below). |
| `aicallhandler` existing tests | `MockParticipantHandler` injected; assert `Create` called once with correct `aiID` on AIcall creation |
| `messagehandler` existing tests | `MockParticipantHandler` injected; assert `Create` called with resolved `toMemberAIID` in `TestEventPMTeamMemberSwitched`; assert write skipped when `resolveTeamMemberAIID` returns `uuid.Nil` |

All mocks generated via `go.uber.org/mock/mockgen`, consistent with existing mocks.

---

## 11. Acceptance Criteria

- [ ] `ai_aicall_participants` table created via Alembic migration with correct PK and index.
- [ ] `ON DUPLICATE KEY UPDATE` silently deduplicates `(ai_id, aicall_id)` pairs; other errors propagate.
- [ ] Participant row written on AIcall creation for both `AssistanceTypeAI` and `AssistanceTypeTeam` (including `StartTask`).
- [ ] Participant row written on each team member switch using `resolveTeamMemberAIID`; skipped (with warning) when resolution returns `uuid.Nil`.
- [ ] A→B→A pattern produces exactly two rows: one for A, one for B.
- [ ] Reused conversation AIcall does not write a duplicate participant row.
- [ ] Failed participant write logs a warning but does not fail the parent operation.
- [ ] All new and modified tests pass (`go test ./...`).
- [ ] Linter passes (`golangci-lint run -v --timeout 5m`).
