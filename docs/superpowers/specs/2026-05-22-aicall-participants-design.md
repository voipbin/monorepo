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
- Silently deduplicate: A→B→A writes AI A once (second write is ignored).
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
    tm_create  DATETIME(6),
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
| No `customer_id` | Evaluation queries will join to `ai_aicalls` or `ai_ais` for tenant scoping |
| `tm_create` | Cheap to store; useful for debugging and auditing |
| `INSERT IGNORE` | Silently drops duplicate `(ai_id, aicall_id)` pairs at the MySQL level |

---

## 5. New Package: `pkg/participanthandler`

A dedicated package owns the participant concept, keeping `aicallhandler` and
`messagehandler` thin.

### File layout

```
pkg/participanthandler/
├── main.go          # interface + constructor
├── db.go            # Create implementation (INSERT IGNORE)
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
func New(db dbhandler.DBHandler, utilHandler utilhandler.UtilHandler) ParticipantHandler
```

Follows the existing constructor pattern used by `aicallhandler`, `messagehandler`, etc.

### `db.go` — core write

```go
func (h *participantHandler) Create(ctx context.Context, aicallID, aiID uuid.UUID) error {
    // INSERT IGNORE INTO ai_aicall_participants (ai_id, aicall_id, tm_create)
    // VALUES (?, ?, ?)
}
```

`INSERT IGNORE` means a duplicate `(ai_id, aicall_id)` pair returns no error and
affects 0 rows — the caller never needs to distinguish "inserted" from "already existed".

---

## 6. dbhandler Addition

New file `pkg/dbhandler/participant.go`:

```go
const participantTable = "ai_aicall_participants"

func (h *handler) ParticipantCreate(ctx context.Context, aicallID, aiID uuid.UUID) error
```

The `dbhandler` interface and mock are updated via `go generate ./...`.

---

## 7. Write Sites

Participant rows are written at two trigger points:

### 7.1 AIcall creation (initial AI)

Both `startAIcallByRealtime` and `startAIcallByMessaging` in `pkg/aicallhandler/start.go`
call `participantHandler.Create(ctx, res.ID, a.ID)` immediately after the AIcall record
is persisted, where `a` is the resolved `*ai.AI`.

This covers both `AssistanceTypeAI` (the AI config IS the ai_id) and `AssistanceTypeTeam`
(the resolved start member's `AIID` is the ai_id).

### 7.2 Team member switch

`messagehandler.EventPMTeamMemberSwitched` in `pkg/messagehandler/event.go` calls
`participantHandler.Create(ctx, evt.PipecatcallReferenceID, toMemberAIID)` after
recording the member-switched notification message, where `toMemberAIID` is resolved
from `evt.ToMember.AIID` (already available in the event payload).

### Error handling

Both write sites treat participant writes as **best-effort**:

```go
if err := h.participantHandler.Create(ctx, aicallID, aiID); err != nil {
    log.Warnf("Could not record aicall participant. aicall_id: %s, ai_id: %s, err: %v",
        aicallID, aiID, err)
}
```

A missed participant write is non-critical — it would produce an incomplete evaluation
aggregate but must not fail or roll back the parent operation (AIcall creation or
message delivery).

---

## 8. Dependency Injection Changes

`participantHandler ParticipantHandler` is added to the `aicallHandler` and
`messageHandler` structs and their constructors. Callers (service wiring in `cmd/`) pass
in a concrete `participanthandler.New(db, util)` instance.

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
| `dbhandler/participant_test.go` | `ParticipantCreate`: normal insert, duplicate insert (no error), DB error |
| `participanthandler/main_test.go` | `Create`: normal path, error propagation from dbhandler |
| `aicallhandler` existing tests | `MockParticipantHandler` injected; assert `Create` called once with correct `aiID` on AIcall creation |
| `messagehandler` existing tests | `MockParticipantHandler` injected; assert `Create` called with `toMember.AIID` in `TestEventPMTeamMemberSwitched` |

All mocks generated via `go.uber.org/mock/mockgen`, consistent with existing mocks.

---

## 11. Acceptance Criteria

- [ ] `ai_aicall_participants` table created via Alembic migration with correct PK and index.
- [ ] `INSERT IGNORE` silently deduplicates `(ai_id, aicall_id)` pairs.
- [ ] Participant row written on AIcall creation for both `AssistanceTypeAI` and `AssistanceTypeTeam`.
- [ ] Participant row written on each team member switch (using the incoming member's AI ID).
- [ ] A→B→A pattern produces exactly two rows: one for A, one for B.
- [ ] Failed participant write logs a warning but does not fail the parent operation.
- [ ] All new and modified tests pass (`go test ./...`).
- [ ] Linter passes (`golangci-lint run -v --timeout 5m`).
