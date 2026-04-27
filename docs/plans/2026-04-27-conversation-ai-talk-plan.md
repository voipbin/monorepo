# Conversation ↔ AI Talk Bridge Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Wire `bin-ai-manager` to deliver LLM replies back to `bin-conversation-manager` so AI participates as a text chatbot in SMS/LINE conversations, with concurrency safety, idle-timeout lifecycle, ping-gated session interrupts, and a text-safe tool whitelist.

**Architecture:** All Go logic lands in `bin-ai-manager`. The `bin-pipecat-manager` per-pod ping endpoint and circuit breaker plumbing in `bin-common-handler` are already on `main` and reused as-is. ai-manager owns delivery (`ConversationV1MessageSend`); the activeflow advances past `ai_talk` immediately. Multi-pod safety is provided by a decisive PipecatcallID response guard at delivery time, with best-effort interrupt as an optimization.

**Tech Stack:** Go 1.x, gomock (`go.uber.org/mock`), Squirrel SQL, RabbitMQ RPC via `bin-common-handler/pkg/requesthandler`, Prometheus client_golang, Sphinx RST docs.

**Design source:** [docs/plans/2026-04-27-conversation-ai-talk-design.md](2026-04-27-conversation-ai-talk-design.md)

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-conversation-ai-talk-bridge` (already created and on branch `NOJIRA-conversation-ai-talk-bridge`)

---

## Sequencing

The plan is organized into thin vertical slices. Each slice is independently committable and testable. After each slice, run the **per-slice verification** before continuing. After the final slice, run the **service-wide verification** (the 5-step workflow from root `CLAUDE.md`).

Slice ordering rationale: lowest blast radius first (helpers, config, db method) → behavior changes (start, send, event) → cross-cutting (metrics, dashboard, RST) → final verification.

```
Slice 0 — Tool whitelist decision spike (NO CODE; just a written decision)
Slice 1 — Helpers package + UpdateActiveflowID method + idle config
Slice 2 — Tool whitelist (path chosen in Slice 0)
Slice 3 — start.go: reusability check + interrupt + status update + tool filter site
Slice 4 — send.go: same interrupt + ActiveflowID update
Slice 5 — event.go: response guard + delivery
Slice 6 — Metrics
Slice 7 — Tests (slice-by-slice tests are TDD-first; this slice is the integration sweep)
Slice 8 — Grafana dashboard panels + RST docs
Slice 9 — Final service-wide verification + commit cleanup
```

---

## Verification items from design §13 (carried into the plan)

Track these as you go; they are tested or asserted in specific slices.

| Verification | Tested in slice | How |
|---|---|---|
| `startPipecatcall` payload assembly is the right site for tool filter | Slice 0 (decision) + Slice 3 | Slice 0 documents the chosen site and its trade-offs; Slice 3 implements at that site |
| Voice paths byte-for-byte unchanged | Slice 7 (regression sweep) + every slice's `go test ./...` | Run existing voice tests before adding new ones; all must pass |
| `evt.PipecatcallID` populated by pipecat-manager | Slice 5 | Verified in event handler tests by asserting the guard reads a non-nil `evt.PipecatcallID`; visually confirmed in pipecat-manager runner.go:454,670,691 |
| `UpdateActiveflowID` method needed in dbhandler | Slice 1 | Implementation slice itself adds the method |

---

## Slice 0 — Tool whitelist decision spike (NO CODE)

**Why:** The design assumed the tool whitelist could be applied "before pipecat payload assembly." Code exploration shows pipecat-manager fetches the AI by ID at session start (`bin-pipecat-manager/pkg/pipecatcallhandler/run.go:163` and `runner.go:82`) and reads `ai.ToolNames` directly. There is no per-AIcall tool list passed through `PipecatV1PipecatcallStart`. So filtering at the ai-manager call site does not propagate to pipecat.

Two mutually exclusive paths exist; choose one before writing any code in slices 2 / 3.

### Path A — Defer whitelist to v2 (recommended for v1)

- **Pros:** zero schema change, zero pipecat-manager change, matches design's "no schema, no pipecat changes" goal, minimum blast radius.
- **Cons:** LLM may invoke `connect_call` / `stop_media` / `stop_flow` in a conversation context. Each fails at execute time (no live call → tool returns failure result). LLM observes failure and adapts in next turn or re-tries. User-visible impact: occasional confusing AI responses if the LLM tries to "connect a call" inside a chat thread, but no system harm. Tracked as a known v2 gap.
- **Implementation effect on this plan:** Slice 2 ships only `ConversationSafeTools` constants and `FilterToolsForConversation` as documented utilities (no call sites). Slice 3 does not invoke them. RST docs note the limitation.

### Path B — AIcall.ToolNames field + 1-line pipecat read change

- **Schema:** add `tool_names JSON NULL` column to `ai_aicalls` table (Alembic migration).
- **Model:** add `ToolNames []tool.ToolName` with db tag `db:"tool_names,json"` to `bin-ai-manager/models/aicall/main.go`.
- **dbhandler:** include in `GetDBFields` / `PrepareFields` paths.
- **ai-manager populate site:** in `startAIcallByMessaging` (start.go:530+), set `ToolNames = FilterToolsForConversation(a.ToolNames)` when `referenceType == ReferenceTypeConversation`; pass-through otherwise.
- **pipecat-manager change:** `pkg/pipecatcallhandler/run.go:163` and `runner.go:82` change from `ai.ToolNames` to `coalesce(aicall.ToolNames, ai.ToolNames)` — i.e., prefer per-call when present. ~5 lines.
- **Pros:** correctly enforces whitelist at runtime; no false tool calls.
- **Cons:** schema change, cross-service change, longer review.

### Decision recording

Write the decision into the **plan** (this file) by editing this section before starting Slice 2. Default if no decision recorded: **Path A**.

**Step 1: Make the decision and record it here.**

Edit this file at the line below and replace `[CHOSEN: A or B]` with the decision and a 1–2 sentence rationale.

```
SLICE-0 DECISION: [CHOSEN: A or B]
Rationale: ...
```

**Step 2: Commit the decision (no code change).**

```bash
git add docs/plans/2026-04-27-conversation-ai-talk-plan.md
git commit -m "NOJIRA-conversation-ai-talk-bridge

- docs/plans: record Slice 0 decision on tool whitelist site"
```

---

## Slice 1 — Helpers package, idle config, UpdateActiveflowID

Three small, independent additions that the later slices depend on. Each is its own test-and-commit cycle.

### Task 1.1: `UpdateActiveflowID` method on `aicallhandler`

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/main.go` (add method to `AIcallHandler` interface)
- Modify: `bin-ai-manager/pkg/aicallhandler/db.go` (add implementation alongside `UpdatePipecatcallID`)
- Test: `bin-ai-manager/pkg/aicallhandler/db_test.go` (add table-driven test alongside existing update tests)

**Step 1: Write failing test.**

Add to `bin-ai-manager/pkg/aicallhandler/db_test.go`. Mirror the structure of the existing `Test_UpdatePipecatcallID` (or equivalent). Asserts `db.AIcallUpdate` is called with `aicall.FieldActiveflowID` and the updated AIcall is returned.

```go
func Test_UpdateActiveflowID(t *testing.T) {
    tests := []struct {
        name           string
        id             uuid.UUID
        activeflowID   uuid.UUID
        responseAIcall *aicall.AIcall
    }{
        {
            name:         "normal",
            id:           uuid.FromStringOrNil("aaaa0000-0000-0000-0000-000000000001"),
            activeflowID: uuid.FromStringOrNil("bbbb0000-0000-0000-0000-000000000001"),
            responseAIcall: &aicall.AIcall{
                Identity: identity.Identity{ID: uuid.FromStringOrNil("aaaa0000-0000-0000-0000-000000000001")},
                ActiveflowID: uuid.FromStringOrNil("bbbb0000-0000-0000-0000-000000000001"),
            },
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()
            mockDB := dbhandler.NewMockDBHandler(mc)
            h := &aicallHandler{db: mockDB}
            ctx := context.Background()

            mockDB.EXPECT().AIcallUpdate(ctx, tt.id, map[aicall.Field]any{
                aicall.FieldActiveflowID: tt.activeflowID,
            }).Return(nil)
            mockDB.EXPECT().AIcallGet(ctx, tt.id).Return(tt.responseAIcall, nil)

            res, err := h.UpdateActiveflowID(ctx, tt.id, tt.activeflowID)
            if err != nil {
                t.Fatalf("Wrong match. expected: ok, got: %v", err)
            }
            if !reflect.DeepEqual(res, tt.responseAIcall) {
                t.Errorf("Wrong match.\nexpected: %v\nreceived: %v", tt.responseAIcall, res)
            }
        })
    }
}
```

**Step 2: Run test — expect FAIL with "undefined: UpdateActiveflowID".**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-conversation-ai-talk-bridge/bin-ai-manager
go test -run Test_UpdateActiveflowID ./pkg/aicallhandler/
```
Expected: compile error or test failure.

**Step 3: Implement.**

In `pkg/aicallhandler/main.go`, add to the `AIcallHandler` interface (alphabetically near other `Update*` methods):

```go
UpdateActiveflowID(ctx context.Context, id uuid.UUID, activeflowID uuid.UUID) (*aicall.AIcall, error)
```

In `pkg/aicallhandler/db.go`, append after `UpdatePipecatcallID`:

```go
// UpdateActiveflowID updates the activeflow_id for the aicall. Used when a
// long-lived AIcall is reused across multiple per-message activeflows
// (conversation chat) so tools that read flow variables target the current flow.
func (h *aicallHandler) UpdateActiveflowID(ctx context.Context, id uuid.UUID, activeflowID uuid.UUID) (*aicall.AIcall, error) {
    fields := map[aicall.Field]any{
        aicall.FieldActiveflowID: activeflowID,
    }
    if errUpdate := h.db.AIcallUpdate(ctx, id, fields); errUpdate != nil {
        return nil, errors.Wrapf(errUpdate, "could not update the activeflow id for aicall. aicall_id: %s", id)
    }

    res, err := h.db.AIcallGet(ctx, id)
    if err != nil {
        return nil, errors.Wrapf(err, "could not get updated aicall info. aicall_id: %s", id)
    }

    return res, nil
}
```

**Step 4: Regenerate mocks.**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-conversation-ai-talk-bridge/bin-ai-manager
go generate ./pkg/aicallhandler/...
```

**Step 5: Run test — expect PASS.**

```bash
go test -run Test_UpdateActiveflowID -v ./pkg/aicallhandler/
```

**Step 6: Commit.**

```bash
git add bin-ai-manager/pkg/aicallhandler/main.go \
        bin-ai-manager/pkg/aicallhandler/db.go \
        bin-ai-manager/pkg/aicallhandler/db_test.go \
        bin-ai-manager/pkg/aicallhandler/mock_main.go
git commit -m "NOJIRA-conversation-ai-talk-bridge

- bin-ai-manager: add aicallhandler.UpdateActiveflowID for per-call activeflow rebinding"
```

### Task 1.2: Idle-timeout configuration

**Files:**
- Modify: `bin-ai-manager/internal/config/main.go`
- Test: `bin-ai-manager/internal/config/main_test.go`

**Step 1: Write failing test.**

Add a test asserting `Get().AIcallConversationIdleTimeoutHours` is populated from viper. Pattern after existing config tests in the same file.

```go
func Test_AIcallConversationIdleTimeoutHours_default(t *testing.T) {
    // setup viper with the default
    viper.Reset()
    viper.SetDefault("aicall_conversation_idle_timeout_hours", 24)

    // simulate Bootstrap → LoadGlobalConfig (or call them directly if exposed in tests)
    once = sync.Once{} // reset singleton
    LoadGlobalConfig()

    if got := Get().AIcallConversationIdleTimeoutHours; got != 24 {
        t.Errorf("expected 24, got %d", got)
    }
}
```

**Step 2: Run — expect FAIL (field undefined).**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-conversation-ai-talk-bridge/bin-ai-manager
go test -run Test_AIcallConversationIdleTimeoutHours_default ./internal/config/
```

**Step 3: Implement.**

In `internal/config/main.go`:

1. Add field to `Config` struct:
```go
AIcallConversationIdleTimeoutHours int // Idle timeout in hours after which a conversation-typed AIcall is treated as expired and a new one is created on the next inbound message.
```

2. Add flag in `bindConfig`:
```go
f.Int("aicall_conversation_idle_timeout_hours", 24, "Idle timeout (hours) for conversation-typed AIcalls before they expire")
```

3. Add to `bindings` map:
```go
"aicall_conversation_idle_timeout_hours": "AICALL_CONVERSATION_IDLE_TIMEOUT_HOURS",
```

4. Add to `LoadGlobalConfig`:
```go
AIcallConversationIdleTimeoutHours: viper.GetInt("aicall_conversation_idle_timeout_hours"),
```

**Step 4: Run — expect PASS.**

```bash
go test -run Test_AIcallConversationIdleTimeoutHours_default -v ./internal/config/
```

**Step 5: Commit.**

```bash
git add bin-ai-manager/internal/config/main.go bin-ai-manager/internal/config/main_test.go
git commit -m "NOJIRA-conversation-ai-talk-bridge

- bin-ai-manager: add AICALL_CONVERSATION_IDLE_TIMEOUT_HOURS config (default 24)"
```

### Task 1.3: Helpers — `isAIcallReusable`, `isAIcallIdleExpired`, `interruptPreviousPipecatcall`

**Files:**
- Create: `bin-ai-manager/pkg/aicallhandler/helpers.go`
- Test: `bin-ai-manager/pkg/aicallhandler/helpers_test.go`

`pingPipecatHost` already exists at `bin-ai-manager/pkg/aicallhandler/ping.go` — we just call it from the new helpers.

**Step 1: Write failing tests for the three helpers.**

```go
// helpers_test.go
package aicallhandler

import (
    "context"
    "testing"
    "time"

    "github.com/gofrs/uuid"
    gomock "go.uber.org/mock/gomock"
    "github.com/pkg/errors"

    "monorepo/bin-ai-manager/internal/config"
    "monorepo/bin-ai-manager/models/aicall"
    "monorepo/bin-common-handler/models/identity"
    "monorepo/bin-common-handler/pkg/requesthandler"
    pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

func Test_isAIcallReusable(t *testing.T) {
    now := time.Now()
    fresh := now.Add(-1 * time.Hour)
    expired := now.Add(-25 * time.Hour)

    tests := []struct {
        name string
        ac   *aicall.AIcall
        want bool
    }{
        {"nil", nil, false},
        {"progressing fresh", &aicall.AIcall{Status: aicall.StatusProgressing, TMUpdate: &fresh}, true},
        {"initiating fresh", &aicall.AIcall{Status: aicall.StatusInitiating, TMUpdate: &fresh}, true},
        {"terminated", &aicall.AIcall{Status: aicall.StatusTerminated, TMUpdate: &fresh}, false},
        {"terminating", &aicall.AIcall{Status: aicall.StatusTerminating, TMUpdate: &fresh}, false},
        {"idle expired", &aicall.AIcall{Status: aicall.StatusProgressing, TMUpdate: &expired}, false},
        {"nil tm_update", &aicall.AIcall{Status: aicall.StatusProgressing}, true}, // no timestamp ⇒ treat as fresh
    }
    // ensure config has default
    config.SetForTest(24) // helper to set AIcallConversationIdleTimeoutHours; add if not present
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            h := &aicallHandler{}
            if got := h.isAIcallReusable(tt.ac); got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}

func Test_interruptPreviousPipecatcall(t *testing.T) {
    pcID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
    hostID := "10.0.0.1"

    tests := []struct {
        name      string
        pcID      uuid.UUID
        mockSetup func(rh *requesthandler.MockRequestHandler)
    }{
        {
            name: "nil pcID — no calls",
            pcID: uuid.Nil,
            mockSetup: func(rh *requesthandler.MockRequestHandler) {
                // no expectations
            },
        },
        {
            name: "Get fails — return without further calls",
            pcID: pcID,
            mockSetup: func(rh *requesthandler.MockRequestHandler) {
                rh.EXPECT().PipecatV1PipecatcallGet(gomock.Any(), pcID).Return(nil, errors.New("not found"))
            },
        },
        {
            name: "ping returns dead — terminate skipped",
            pcID: pcID,
            mockSetup: func(rh *requesthandler.MockRequestHandler) {
                rh.EXPECT().PipecatV1PipecatcallGet(gomock.Any(), pcID).Return(&pmpipecatcall.Pipecatcall{
                    Identity: identity.Identity{ID: pcID}, HostID: hostID,
                }, nil)
                rh.EXPECT().PipecatV1Ping(gomock.Any(), hostID).Return(errors.New("dead"))
            },
        },
        {
            name: "ping ok — terminate called",
            pcID: pcID,
            mockSetup: func(rh *requesthandler.MockRequestHandler) {
                rh.EXPECT().PipecatV1PipecatcallGet(gomock.Any(), pcID).Return(&pmpipecatcall.Pipecatcall{
                    Identity: identity.Identity{ID: pcID}, HostID: hostID,
                }, nil)
                rh.EXPECT().PipecatV1Ping(gomock.Any(), hostID).Return(nil)
                rh.EXPECT().PipecatV1PipecatcallTerminate(gomock.Any(), hostID, pcID).Return(nil, nil)
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()
            rh := requesthandler.NewMockRequestHandler(mc)
            tt.mockSetup(rh)
            h := &aicallHandler{reqHandler: rh}
            h.interruptPreviousPipecatcall(context.Background(), tt.pcID)
        })
    }
}
```

**Step 2: Run — expect FAIL.**

```bash
go test -run "Test_isAIcallReusable|Test_interruptPreviousPipecatcall" ./pkg/aicallhandler/
```

**Step 3: Implement helpers.**

Create `bin-ai-manager/pkg/aicallhandler/helpers.go`:

```go
package aicallhandler

import (
    "context"
    "time"

    "github.com/gofrs/uuid"
    "github.com/sirupsen/logrus"

    "monorepo/bin-ai-manager/internal/config"
    "monorepo/bin-ai-manager/models/aicall"
)

// isAIcallIdleExpired returns true if the AIcall has been idle longer than
// the configured conversation idle timeout. Returns false when TMUpdate is nil
// (treated as freshly created).
func (h *aicallHandler) isAIcallIdleExpired(c *aicall.AIcall) bool {
    if c == nil || c.TMUpdate == nil {
        return false
    }
    threshold := time.Duration(config.Get().AIcallConversationIdleTimeoutHours) * time.Hour
    return time.Since(*c.TMUpdate) > threshold
}

// isAIcallReusable returns true if the AIcall is suitable to be reused for
// the next inbound message in the same conversation: it must exist, be in a
// non-terminal status, and not be idle-expired.
func (h *aicallHandler) isAIcallReusable(c *aicall.AIcall) bool {
    if c == nil {
        return false
    }
    if c.Status == aicall.StatusTerminated || c.Status == aicall.StatusTerminating {
        return false
    }
    if h.isAIcallIdleExpired(c) {
        return false
    }
    return true
}

// interruptPreviousPipecatcall attempts a synchronous, ping-gated termination
// of the previous pipecat session. Best-effort: errors are logged at DEBUG and
// swallowed. Correctness is provided by the response guard at delivery time.
//
// The Get call is bounded by a 1.5s context to avoid blocking the user-facing
// path on a degraded shared queue. The ping is bounded by 1.1s (inside
// pingPipecatHost). Total worst case: ~2.6s before this returns.
func (h *aicallHandler) interruptPreviousPipecatcall(ctx context.Context, pcID uuid.UUID) {
    if pcID == uuid.Nil {
        return
    }
    log := logrus.WithFields(logrus.Fields{
        "func":           "interruptPreviousPipecatcall",
        "pipecatcall_id": pcID,
    })

    gctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
    defer cancel()

    pc, errGet := h.reqHandler.PipecatV1PipecatcallGet(gctx, pcID)
    if errGet != nil {
        log.Debugf("Could not get previous pipecatcall — assuming gone. err: %v", errGet)
        return
    }
    if !h.pingPipecatHost(ctx, pc.HostID) {
        log.Debugf("Previous pipecatcall pod unreachable — skipping terminate. host_id: %s", pc.HostID)
        return
    }
    if _, errTerm := h.reqHandler.PipecatV1PipecatcallTerminate(ctx, pc.HostID, pc.ID); errTerm != nil {
        log.Debugf("Previous pipecatcall terminate failed — response guard will handle. err: %v", errTerm)
    }
}
```

If `config.SetForTest` does not exist, add a small helper to `internal/config/main.go`:

```go
// SetForTest sets the AIcallConversationIdleTimeoutHours field for tests.
// Production code MUST go through Bootstrap + LoadGlobalConfig.
func SetForTest(idleHours int) {
    globalConfig.AIcallConversationIdleTimeoutHours = idleHours
}
```

**Step 4: Run — expect PASS.**

```bash
go test -run "Test_isAIcallReusable|Test_interruptPreviousPipecatcall" -v ./pkg/aicallhandler/
```

**Step 5: Commit.**

```bash
git add bin-ai-manager/pkg/aicallhandler/helpers.go \
        bin-ai-manager/pkg/aicallhandler/helpers_test.go \
        bin-ai-manager/internal/config/main.go
git commit -m "NOJIRA-conversation-ai-talk-bridge

- bin-ai-manager: add aicallhandler helpers (isAIcallReusable, isAIcallIdleExpired, interruptPreviousPipecatcall) using existing pingPipecatHost"
```

### Slice 1 verification

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-conversation-ai-talk-bridge/bin-ai-manager
go test ./pkg/aicallhandler/... ./internal/config/...
```
Expected: PASS. (Full 5-step verification deferred to Slice 9.)

---

## Slice 2 — Tool whitelist

Implement `ConversationSafeTools` and `FilterToolsForConversation`. The exact wiring depends on the Slice 0 decision.

### Task 2.1: Whitelist constants and filter function (both paths)

**Files:**
- Create: `bin-ai-manager/pkg/toolhandler/whitelist.go`
- Test: `bin-ai-manager/pkg/toolhandler/whitelist_test.go`

**Step 1: Write failing test.**

```go
package toolhandler

import (
    "reflect"
    "sort"
    "testing"

    "monorepo/bin-ai-manager/models/tool"
)

func sorted(in []tool.ToolName) []string {
    out := make([]string, 0, len(in))
    for _, n := range in {
        out = append(out, string(n))
    }
    sort.Strings(out)
    return out
}

func Test_FilterToolsForConversation(t *testing.T) {
    tests := []struct {
        name   string
        in     []tool.ToolName
        wantIn []tool.ToolName // order-insensitive
    }{
        {"empty in", nil, []tool.ToolName{}},
        {"strips voice-only", []tool.ToolName{tool.ToolNameConnectCall, tool.ToolNameSendEmail, tool.ToolNameStopMedia}, []tool.ToolName{tool.ToolNameSendEmail}},
        {"keeps text-safe", []tool.ToolName{tool.ToolNameSendMessage, tool.ToolNameSetVariables, tool.ToolNameStopService}, []tool.ToolName{tool.ToolNameSendMessage, tool.ToolNameSetVariables, tool.ToolNameStopService}},
        {"all expands to whitelist", []tool.ToolName{tool.ToolNameAll}, []tool.ToolName{
            tool.ToolNameSendEmail, tool.ToolNameSendMessage, tool.ToolNameSetVariables,
            tool.ToolNameGetVariables, tool.ToolNameStopService, tool.ToolNameGetAIcallMessages, tool.ToolNameSearchKnowledge,
        }},
        {"strips stop_flow", []tool.ToolName{tool.ToolNameStopFlow, tool.ToolNameSendEmail}, []tool.ToolName{tool.ToolNameSendEmail}},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := FilterToolsForConversation(tt.in)
            if !reflect.DeepEqual(sorted(got), sorted(tt.wantIn)) {
                t.Errorf("got %v, want %v", got, tt.wantIn)
            }
        })
    }
}
```

**Step 2: Run — expect FAIL.**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-conversation-ai-talk-bridge/bin-ai-manager
go test -run Test_FilterToolsForConversation ./pkg/toolhandler/
```

**Step 3: Implement.**

Create `bin-ai-manager/pkg/toolhandler/whitelist.go`:

```go
package toolhandler

import "monorepo/bin-ai-manager/models/tool"

// ConversationSafeTools is the set of tool names enabled for AIcalls with
// ReferenceTypeConversation. Voice-specific tools (connect_call, stop_media,
// stop_flow) are excluded — they assume a live phone call.
//
// See docs/plans/2026-04-27-conversation-ai-talk-design.md §9 for rationale.
var ConversationSafeTools = map[tool.ToolName]bool{
    tool.ToolNameSendEmail:         true,
    tool.ToolNameSendMessage:       true,
    tool.ToolNameSetVariables:      true,
    tool.ToolNameGetVariables:      true,
    tool.ToolNameStopService:       true,
    tool.ToolNameGetAIcallMessages: true,
    tool.ToolNameSearchKnowledge:   true,
}

// FilterToolsForConversation returns the subset of names that are safe for a
// conversation-typed AIcall. ToolNameAll expands to the whitelist (not the
// full registry). Output preserves stable iteration order for reproducible
// tests by deriving from the whitelist's known keys when ToolNameAll is given;
// otherwise it preserves input order.
func FilterToolsForConversation(names []tool.ToolName) []tool.ToolName {
    out := make([]tool.ToolName, 0, len(names))
    for _, n := range names {
        if n == tool.ToolNameAll {
            for k := range ConversationSafeTools {
                out = append(out, k)
            }
            continue
        }
        if ConversationSafeTools[n] {
            out = append(out, n)
        }
    }
    return out
}
```

**Step 4: Run — expect PASS.**

```bash
go test -run Test_FilterToolsForConversation -v ./pkg/toolhandler/
```

**Step 5: Commit.**

```bash
git add bin-ai-manager/pkg/toolhandler/whitelist.go \
        bin-ai-manager/pkg/toolhandler/whitelist_test.go
git commit -m "NOJIRA-conversation-ai-talk-bridge

- bin-ai-manager: add ConversationSafeTools whitelist and FilterToolsForConversation"
```

### Task 2.2 (only if Slice 0 = Path B): wire AIcall.ToolNames + pipecat read

If Path A was chosen, **skip this task entirely** and move to Slice 3.

If Path B was chosen, the work is large enough to deserve its own dedicated plan section — split into a separate plan `2026-04-27-conversation-ai-talk-toolnames-plan.md` and execute before resuming this plan at Slice 3. The split work would include:
- Alembic migration for `ai_aicalls.tool_names` JSON column (in `bin-dbscheme-manager`)
- `aicall.AIcall.ToolNames []tool.ToolName` field with `db:"tool_names,json"` tag
- `aicall.FieldToolNames`
- `dbhandler` updates (tests + scan + insert + update)
- Populate filtered `ToolNames` in `startAIcallByMessaging` when `referenceType == ReferenceTypeConversation`
- `bin-pipecat-manager/pkg/pipecatcallhandler/run.go:163` and `runner.go:82`: prefer `aicall.ToolNames` over `ai.ToolNames` when non-nil
- Cross-service verification per §1 of root CLAUDE.md (4 services)

### Slice 2 verification

```bash
go test ./pkg/toolhandler/...
```
Expected: PASS.

---

## Slice 3 — `start.go`: reusability + interrupt + status update + tool filter site

The core behavior change. Replace the existing reuse branch (`bin-ai-manager/pkg/aicallhandler/start.go:178–194`) with the new logic.

### Task 3.1: Tests for `startReferenceTypeConversation`

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/start_test.go` (add 6 new sub-tests)

**Step 1: Add table-driven tests** for the cases in design §12.1:

- `fresh AIcall path` — no existing AIcall, calls `startAIcallByMessaging`, then `UpdateStatus(Progressing)`, then proceeds
- `reuse + alive previous pipecat` — existing AIcall with `Status=Progressing`, ping returns alive, terminate is called, then `UpdatePipecatcallID` + `UpdateActiveflowID`
- `reuse + dead previous pipecat` — existing AIcall, ping returns dead, terminate is NOT called, but `UpdatePipecatcallID` + `UpdateActiveflowID` still happen
- `terminated AIcall → fresh` — existing AIcall with `Status=Terminated`, treated as not reusable, fresh AIcall created
- `idle-expired AIcall → fresh` — existing AIcall with old `TMUpdate`, treated as not reusable, marked `Terminated` for hygiene then fresh AIcall created
- `AssistanceTypeTeam smoke` — `assistanceType=Team`, the team resolution branch runs and reaches `startAIcallByMessaging`

Pattern: extend the existing `Test_startReferenceTypeConversation` (or `Test_Start_ReferenceTypeConversation`) table with new cases. Each case sets up `MockRequestHandler`, `MockDBHandler`, `MockMessageHandler`, `MockAIHandler`, `MockTeamHandler` per the existing mock fixtures.

**Step 2: Run — expect FAIL.**

```bash
go test -run Test_startReferenceTypeConversation -v ./pkg/aicallhandler/
```

**Step 3: Implement the change in `pkg/aicallhandler/start.go`.**

Replace lines 178–194 with:

```go
// get existing aicall info — decide reuse or create fresh
res, err := h.GetByReferenceID(ctx, referenceID)
reusable := err == nil && h.isAIcallReusable(res)

if !reusable {
    // mark idle-expired AIcalls as Terminated for hygiene before recreating
    if err == nil && res.Status != aicall.StatusTerminated && res.Status != aicall.StatusTerminating && h.isAIcallIdleExpired(res) {
        log.Infof("Existing AIcall idle-expired — terminating and starting fresh. aicall_id: %s", res.ID)
        if _, errEnd := h.UpdateStatus(ctx, res.ID, aicall.StatusTerminated); errEnd != nil {
            log.Warnf("Could not terminate idle AIcall: %v", errEnd)
        }
    }
    res, err = h.startAIcallByMessaging(ctx, a, assistanceType, assistanceID, activeflowID, aicall.ReferenceTypeConversation, referenceID, false, teamParameter, currentMemberID)
    if err != nil {
        return nil, errors.Wrapf(err, "could not create aicall. activeflow_id: %s", activeflowID)
    }
    res, err = h.UpdateStatus(ctx, res.ID, aicall.StatusProgressing)
    if err != nil {
        return nil, errors.Wrapf(err, "could not update status to Progressing. aicall_id: %s", res.ID)
    }
} else {
    // reuse: interrupt previous pipecat session (best-effort), then update IDs
    h.interruptPreviousPipecatcall(ctx, res.PipecatcallID)
    newPipecatcallID := h.utilHandler.UUIDCreate()
    tmp, errUpdate := h.UpdatePipecatcallID(ctx, res.ID, newPipecatcallID)
    if errUpdate != nil {
        return nil, errors.Wrapf(errUpdate, "could not update the pipecatcall id for existing aicall. aicall_id: %s", res.ID)
    }
    res = tmp
    tmp2, errAF := h.UpdateActiveflowID(ctx, res.ID, activeflowID)
    if errAF != nil {
        return nil, errors.Wrapf(errAF, "could not update the activeflow id for existing aicall. aicall_id: %s", res.ID)
    }
    res = tmp2
}
log.WithField("aicall", res).Debugf("AIcall ready. aicall_id: %s", res.ID)
```

**Step 4: Run — expect PASS.**

```bash
go test -run Test_startReferenceTypeConversation -v ./pkg/aicallhandler/
```

**Step 5: Commit.**

```bash
git add bin-ai-manager/pkg/aicallhandler/start.go \
        bin-ai-manager/pkg/aicallhandler/start_test.go
git commit -m "NOJIRA-conversation-ai-talk-bridge

- bin-ai-manager: rewrite startReferenceTypeConversation reuse branch with status+idle reusability check, ping-gated interrupt, ActiveflowID rebinding, and Progressing status update on fresh path"
```

### Task 3.2 (only if Slice 0 = Path A — recommended): document the deferred whitelist

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go`

Add a comment block where `FilterToolsForConversation` would be invoked, so a future implementer (Path B follow-up) has a clear anchor:

```go
// NOTE: Tool whitelist for conversation-typed AIcalls is deferred to v2 — see
// docs/plans/2026-04-27-conversation-ai-talk-design.md §13 and the Slice 0
// decision in 2026-04-27-conversation-ai-talk-plan.md. The LLM may invoke
// connect_call / stop_media / stop_flow in a chat context; each fails at
// execute time and is observable via ai_manager_aicall_tool_execute_total.
```

Place immediately after `tmp` is created from `messageHandler.Create` (start.go:198–202), before `startPipecatcall`.

**Step 1 (no test needed for a comment):** edit the file.

**Step 2: Verify build still compiles.**

```bash
go build ./...
```

**Step 3: Commit.**

```bash
git add bin-ai-manager/pkg/aicallhandler/start.go
git commit -m "NOJIRA-conversation-ai-talk-bridge

- bin-ai-manager: document deferred conversation-tool whitelist site for future Path B wiring"
```

### Task 3.3 (only if Slice 0 = Path B): apply `FilterToolsForConversation`

In `startAIcallByMessaging` (start.go:509), where the AIcall is constructed, set the filtered ToolNames when `referenceType == ReferenceTypeConversation`. Detail belongs to the separate Path B plan referenced in Slice 2 Task 2.2.

### Slice 3 verification

```bash
go test ./pkg/aicallhandler/... ./internal/config/...
```
Expected: PASS.

---

## Slice 4 — `send.go`: same interrupt + ActiveflowID update

Apply the interrupt + activeflow rebinding pattern in `SendReferenceTypeOthers` (send.go:55–93).

### Task 4.1: Tests for `SendReferenceTypeOthers`

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/send_test.go`

**Step 1: Add sub-tests** mirroring those in Slice 3 (alive previous pipecat, dead previous pipecat). The existing test fixture is `Test_SendReferenceTypeOthers` or similar — extend its table.

**Step 2: Run — expect FAIL** for the new cases that assert the new mock expectations on `interruptPreviousPipecatcall` chain (`PipecatV1PipecatcallGet`, `PipecatV1Ping`, optional `PipecatV1PipecatcallTerminate`) and the new `UpdateActiveflowID` call.

**Step 3: Implement.**

In `bin-ai-manager/pkg/aicallhandler/send.go`, replace lines 67–72 (the segment that allocates `newPipecatcallID` and calls `UpdatePipecatcallID`):

```go
// interrupt the previous pipecat session (best-effort) before allocating a new one
h.interruptPreviousPipecatcall(ctx, c.PipecatcallID)

newPipecatcallID := h.utilHandler.UUIDCreate()
c, errTerminate = h.UpdatePipecatcallID(ctx, aicallID, newPipecatcallID)
if errTerminate != nil {
    return nil, errors.Wrapf(errTerminate, "could not update the pipecatcall id for existing aicall. aicall_id: %s", aicallID)
}

// rebind ActiveflowID so tools that read flow variables target the current flow.
// The activeflow ID on the AIcall record may be stale (from a previous turn).
// Use c.ActiveflowID — Send is invoked in the same logical turn, the caller
// has already set the right ActiveflowID at the API boundary.
// (No update needed if Send is API-driven and uses the current AIcall.ActiveflowID.
//  Reviewer: confirm this assumption holds before merging.)
```

> **Reviewer note:** confirm whether `Send` updates `ActiveflowID` upstream. If not, add `c, _ = h.UpdateActiveflowID(ctx, aicallID, c.ActiveflowID)` here. Track as an open item if ambiguous.

**Step 4: Run — expect PASS.**

```bash
go test -run Test_SendReferenceType -v ./pkg/aicallhandler/
```

**Step 5: Commit.**

```bash
git add bin-ai-manager/pkg/aicallhandler/send.go \
        bin-ai-manager/pkg/aicallhandler/send_test.go
git commit -m "NOJIRA-conversation-ai-talk-bridge

- bin-ai-manager: ping-gated interrupt before allocating new PipecatcallID in SendReferenceTypeOthers"
```

### Slice 4 verification

```bash
go test ./pkg/aicallhandler/...
```
Expected: PASS.

---

## Slice 5 — `event.go`: response guard + delivery

The decisive correctness slice. `EventPMMessageBotLLM` learns to deliver to conversation-manager when the AIcall's reference type is conversation, and drops stale responses via the PipecatcallID guard.

### Task 5.1: Tests for `EventPMMessageBotLLM`

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/event_test.go`

**Step 1: Add sub-tests** for design §12.1 cases:

- `voice path unchanged` — `evt.PipecatcallReferenceType = ReferenceTypeAICall`, AIV1AIcallGet returns AIcall with `ReferenceType = Call`. Behavior: persists message, no `ConversationV1MessageSend` call. Equivalent to current behavior.
- `non-aicall ref type` — `evt.PipecatcallReferenceType ≠ ReferenceTypeAICall`. Behavior: persists message, returns. (Existing behavior preserved — keep early-return at top.)
- `conversation guard #1 miss` — `ac.PipecatcallID ≠ evt.PipecatcallID`. Behavior: NO persistence, NO `ConversationV1MessageSend`, increments `stale_response_dropped{guard=primary}` (Slice 6 hooks; for now assert no DB write and no send).
- `conversation guard #1 pass, guard #2 miss` — first AIV1AIcallGet matches, after persistence the second AIV1AIcallGet shows a different PipecatcallID (race during persistence). Behavior: persistence happens, `ConversationV1MessageSend` skipped, `stale_response_dropped{guard=secondary}` increments.
- `conversation guards both pass` — `ConversationV1MessageSend` is invoked with `ac.ReferenceID, evt.Text, []`.
- `conversation send fails` — `ConversationV1MessageSend` returns error. Behavior: silent, no panic, error logged, `conversation_reply_send{result=failure}` increments.

For the metrics counters, use the actual counter package; `Slice 6` creates them. To make Slice 5 self-contained, you can either:

(a) introduce the counters as no-op stubs first (defined in Slice 6), or
(b) skip the counter-increment assertions in Slice 5 and tighten them in Slice 7.

Default: (a) — define the counters in this slice (just declarations), and have Slice 6 add registrations + dashboard panels.

**Step 2: Run — expect FAIL.**

```bash
go test -run Test_EventPMMessageBotLLM -v ./pkg/messagehandler/
```

**Step 3: Implement.**

Replace `pkg/messagehandler/event.go` `EventPMMessageBotLLM` with:

```go
func (h *messageHandler) EventPMMessageBotLLM(ctx context.Context, evt *pmmessage.Message) {
    log := logrus.WithFields(logrus.Fields{
        "func":  "EventPMMessageBotLLM",
        "event": evt,
    })

    if evt.Text == "" {
        return
    }

    // Only AIcall-typed pipecatcalls flow through this path
    if evt.PipecatcallReferenceType != pmpipecatcall.ReferenceTypeAICall {
        // legacy non-AIcall path: persist and return (existing behavior)
        if _, err := h.Create(ctx, evt.ID, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
            message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, ""); err != nil {
            log.Errorf("Could not create message: %v", err)
        }
        return
    }

    ac, err := h.reqHandler.AIV1AIcallGet(ctx, evt.PipecatcallReferenceID)
    if err != nil {
        log.Errorf("Could not get aicall — skipping. err: %v", err)
        return
    }

    // Voice / task: keep existing behavior (persist, no delivery)
    if ac.ReferenceType != aicall.ReferenceTypeConversation {
        if _, err := h.Create(ctx, evt.ID, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
            message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, ""); err != nil {
            log.Errorf("Could not create message: %v", err)
        }
        return
    }

    // Guard #1 — drop stale responses BEFORE any DB write (per pattern doc rule 3)
    if ac.PipecatcallID != evt.PipecatcallID {
        log.Infof("Dropping stale response (guard primary). aicall_id: %s, current_pcc: %s, event_pcc: %s",
            ac.ID, ac.PipecatcallID, evt.PipecatcallID)
        promConversationStaleResponseDroppedTotal.WithLabelValues("primary").Inc()
        return
    }

    // Persist the assistant message
    if _, err := h.Create(ctx, evt.ID, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
        message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, ""); err != nil {
        log.Errorf("Could not create message: %v", err)
        return
    }

    // Guard #2 — re-check after persistence to narrow the dual-delivery race window
    acFinal, errFinal := h.reqHandler.AIV1AIcallGet(ctx, evt.PipecatcallReferenceID)
    if errFinal != nil {
        log.Warnf("Re-check AIcall fetch failed; skipping conversation delivery. err: %v", errFinal)
        return
    }
    if acFinal.PipecatcallID != evt.PipecatcallID {
        log.Infof("Race detected at delivery time (guard secondary). aicall_id: %s, event_pcc: %s",
            ac.ID, evt.PipecatcallID)
        promConversationStaleResponseDroppedTotal.WithLabelValues("secondary").Inc()
        return
    }

    // Deliver — silent failure on error
    sent, errSend := h.reqHandler.ConversationV1MessageSend(ctx, acFinal.ReferenceID, evt.Text, []conversationmedia.Media{})
    if errSend != nil {
        log.Errorf("Could not send conversation message (silent failure): %v", errSend)
        promConversationReplySendTotal.WithLabelValues("failure").Inc()
        return
    }
    promConversationReplySendTotal.WithLabelValues("success").Inc()
    log.WithField("conversation_message", sent).Debugf("Sent conversation reply.")
}
```

Required new imports:
```go
import (
    "monorepo/bin-ai-manager/models/aicall"
    conversationmedia "monorepo/bin-conversation-manager/models/media"
)
```

**Step 4: Add metric stubs** (declarations only; full registration in Slice 6).

In `pkg/messagehandler/main.go` (or a new `pkg/messagehandler/metrics.go`), add:

```go
var (
    promConversationReplySendTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: metricsNamespace,
            Name:      "conversation_reply_send_total",
            Help:      "Total ConversationV1MessageSend attempts from AI delivery, by result.",
        },
        []string{"result"},
    )
    promConversationStaleResponseDroppedTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: metricsNamespace,
            Name:      "aicall_stale_response_dropped_total",
            Help:      "Stale BotLLM responses dropped by PipecatcallID guard.",
        },
        []string{"guard"},
    )
)

func init() {
    prometheus.MustRegister(promConversationReplySendTotal, promConversationStaleResponseDroppedTotal)
}
```

> **Important:** put these in a NEW init() in a NEW file (e.g., `pkg/messagehandler/metrics_conversation.go`) to avoid touching the existing init at line ~67. Two `init()` calls in the same package are fine — Go invokes them in lex order; both register their metrics.

**Step 5: Run — expect PASS.**

```bash
go test -run Test_EventPMMessageBotLLM -v ./pkg/messagehandler/
```

**Step 6: Commit.**

```bash
git add bin-ai-manager/pkg/messagehandler/event.go \
        bin-ai-manager/pkg/messagehandler/event_test.go \
        bin-ai-manager/pkg/messagehandler/metrics_conversation.go
git commit -m "NOJIRA-conversation-ai-talk-bridge

- bin-ai-manager: EventPMMessageBotLLM delivers conversation replies via ConversationV1MessageSend with primary+secondary PipecatcallID guards; voice path unchanged"
```

### Slice 5 verification

```bash
go test ./pkg/messagehandler/... ./pkg/aicallhandler/...
```
Expected: PASS.

---

## Slice 6 — Remaining metrics

Slice 5 already declared the conversation-side counters. This slice adds the AIcallhandler-side counters (`idle_expired`, `interrupt_attempted`).

### Task 6.1: Idle-expiry and interrupt-attempt counters

**Files:**
- Create: `bin-ai-manager/pkg/aicallhandler/metrics_conversation.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go` (one increment)
- Modify: `bin-ai-manager/pkg/aicallhandler/helpers.go` (interrupt-result increments)

**Step 1: Add tests asserting the counters increment** in `helpers_test.go` and `start_test.go`. Use `testutil.ToFloat64(promCounter.WithLabelValues(...))` from `prometheus/client_golang/prometheus/testutil`.

**Step 2: Run — expect FAIL.**

**Step 3: Implement.**

`pkg/aicallhandler/metrics_conversation.go`:

```go
package aicallhandler

import "github.com/prometheus/client_golang/prometheus"

var (
    promAIcallIdleExpiredTotal = prometheus.NewCounter(
        prometheus.CounterOpts{
            Namespace: metricsNamespace,
            Name:      "aicall_idle_expired_total",
            Help:      "Total AIcalls terminated due to conversation idle-timeout on reuse path.",
        },
    )
    promAIcallInterruptAttemptedTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: metricsNamespace,
            Name:      "aicall_interrupt_attempted_total",
            Help:      "Pipecat interrupt attempts on AIcall reuse, by outcome.",
        },
        []string{"result"},
    )
)

func init() {
    prometheus.MustRegister(promAIcallIdleExpiredTotal, promAIcallInterruptAttemptedTotal)
}
```

Update `start.go` (idle-expired branch):
```go
log.Infof("Existing AIcall idle-expired ...")
promAIcallIdleExpiredTotal.Inc()
```

Update `helpers.go` (`interruptPreviousPipecatcall`):
```go
// after PipecatV1PipecatcallGet error
promAIcallInterruptAttemptedTotal.WithLabelValues("gone").Inc()
return

// after ping returns false
promAIcallInterruptAttemptedTotal.WithLabelValues("dead").Inc()
return

// after successful terminate
promAIcallInterruptAttemptedTotal.WithLabelValues("alive").Inc()

// after terminate error
promAIcallInterruptAttemptedTotal.WithLabelValues("error").Inc()
```

**Step 4: Run — expect PASS.**

**Step 5: Commit.**

```bash
git add bin-ai-manager/pkg/aicallhandler/metrics_conversation.go \
        bin-ai-manager/pkg/aicallhandler/start.go \
        bin-ai-manager/pkg/aicallhandler/helpers.go \
        bin-ai-manager/pkg/aicallhandler/start_test.go \
        bin-ai-manager/pkg/aicallhandler/helpers_test.go
git commit -m "NOJIRA-conversation-ai-talk-bridge

- bin-ai-manager: add aicall_idle_expired_total and aicall_interrupt_attempted_total counters"
```

### Slice 6 verification

```bash
go test ./pkg/aicallhandler/... ./pkg/messagehandler/...
```
Expected: PASS.

---

## Slice 7 — Voice-path regression sweep + integration sweep

### Task 7.1: Run full ai-manager test suite — voice tests must pass unchanged

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-conversation-ai-talk-bridge/bin-ai-manager
go test ./...
```

Expected: PASS. If any voice test fails, stop and investigate — voice paths must be byte-for-byte unchanged. The most likely culprit is the conversation-only branch in `EventPMMessageBotLLM` accidentally triggering for voice events; re-read the early returns.

### Task 7.2: Customer-isolation regression test

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/event_test.go`

Add a sub-test where the mock `AIV1AIcallGet` returns an AIcall whose `CustomerID` differs from `evt.CustomerID`. Expected behavior: still proceed (we don't cross-check at runtime per design §11), but the test serves as a safeguard against accidental future changes that would silently leak customer data. Leave a comment explaining intent.

```go
{
    name: "different customer_id between aicall and event — current design accepts (no cross-check)",
    // event.CustomerID = X, aicall.CustomerID = Y
    // expect: ConversationV1MessageSend invoked with aicall.ReferenceID
    // SAFETY: this test is here to detect accidental tightening that breaks unit tests;
    //         actual customer isolation is enforced upstream (flow-manager + conversation-manager).
}
```

### Task 7.3: api-validator integration test

**Files:**
- Create: `~/gitvoipbin/monorepo-monitoring/api-validator/scenarios/test_conversation_ai_talk.py` (or matching path; verify the project's test discovery convention)

This is an end-to-end scenario:

1. Create test customer + AI config + flow with `ai_talk` action targeting the AI.
2. Set up an SMS test number and bind the flow.
3. POST a synthetic incoming SMS via the test API.
4. Wait for the conversation to receive an outbound AI reply (poll `GET /v1/conversations/<id>/messages`).
5. Assert: a message with `direction=outgoing` and `role=assistant` appears within 30s.

If the api-validator project doesn't have AI-mocking infra, skip this task and document as a follow-up.

**Step 1: Write test using existing api-validator scaffolding.**

Reference existing scenarios in the directory for patterns. Most use a fixture customer.

**Step 2: Run.**

```bash
cd ~/gitvoipbin/monorepo-monitoring/api-validator
pytest scenarios/test_conversation_ai_talk.py -v
```

If LLM mocking is not available, mark with `pytest.mark.skip(reason="LLM mock pending")` and note in the plan.

**Step 3: Commit.**

```bash
cd ~/gitvoipbin/monorepo-monitoring
git add api-validator/scenarios/test_conversation_ai_talk.py
git commit -m "Add conversation+ai_talk end-to-end scenario test"
```

### Slice 7 verification

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-conversation-ai-talk-bridge/bin-ai-manager
go test ./...
```
Expected: PASS.

---

## Slice 8 — Grafana dashboard panels + RST docs

### Task 8.1: Grafana dashboard panels

**Files:**
- Modify: `monitoring/grafana/dashboards/ai-manager.json`

Add a "Conversation AI" panel row containing four panels using the new counters:

- `rate(ai_manager_conversation_reply_send_total{result="success"}[5m])` — successful deliveries (graph)
- `rate(ai_manager_conversation_reply_send_total{result="failure"}[5m])` — failed deliveries (graph, alert candidate)
- `rate(ai_manager_aicall_stale_response_dropped_total[5m])` by `guard` — stale drops (stacked area)
- `rate(ai_manager_aicall_idle_expired_total[5m])` — idle expirations (graph)
- `rate(ai_manager_aicall_interrupt_attempted_total[5m])` by `result` — interrupt outcomes (stacked area)
- `ai_manager_circuitbreaker_state{target=~".*conversation.*|.*pipecat.*"}` — CB state for relevant queues (stat panel)

If the dashboard file does not exist, create it as a minimal file with just these panels. Otherwise insert the new row.

**Step 1: Edit JSON.** Use a Grafana dashboard you trust (e.g., flow-manager.json or call-manager.json) as a template if needed.

**Step 2: Validate.** No automated test; manually inspect the JSON for syntax (`python3 -c 'import json; json.load(open("monitoring/grafana/dashboards/ai-manager.json"))'`).

**Step 3: Commit.**

```bash
git add monitoring/grafana/dashboards/ai-manager.json
git commit -m "NOJIRA-conversation-ai-talk-bridge

- monitoring: add Conversation AI panels to ai-manager dashboard"
```

### Task 8.2: RST docs

**Files:**
- Modify: `bin-api-manager/docsdev/source/ai_overview.rst` (or wherever ai_talk is documented; locate via `grep -rln "ai_talk" bin-api-manager/docsdev/source/`)
- Modify: `bin-api-manager/docsdev/source/conversation_overview.rst` (analogous)
- Modify: tool reference (locate via `grep -rln "connect_call" bin-api-manager/docsdev/source/`)

**Step 1: Locate target files.**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-conversation-ai-talk-bridge
grep -rln "ai_talk" bin-api-manager/docsdev/source/
grep -rln "connect_call" bin-api-manager/docsdev/source/
```

**Step 2: Add a "Using AI in conversations" section** to the AI overview (or to flow_action docs) that explains:

- A flow attached to a number's `MessageFlowID` runs once per inbound message
- An `ai_talk` action with the AI's `id` starts (or reuses) an AIcall keyed by conversation_id
- AI replies are delivered to the conversation via the same channel the inbound message used
- AIcall reused across turns; reset by `stop_service`, idle-timeout (24h default), or manual termination
- Tool availability: voice-only tools (`connect_call`, `stop_media`, `stop_flow`) may fail when invoked in chat context (Path A) OR are filtered out (Path B); LLM should rely on `send_email`, `send_message`, `set_variables`, `get_variables`, `stop_service`, `get_aicall_messages`, `search_knowledge`

**Step 3: Update tool availability matrix.**

Add a column "Available in conversation" with ✓/✗ per the table in design §9.

**Step 4: Clean rebuild Sphinx HTML.**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-conversation-ai-talk-bridge/bin-api-manager/docsdev
rm -rf build && python3 -m sphinx -M html source build
```

**Step 5: Force-add the build output.**

```bash
git add -f bin-api-manager/docsdev/build/
git add bin-api-manager/docsdev/source/
```

**Step 6: Commit.**

```bash
git commit -m "NOJIRA-conversation-ai-talk-bridge

- bin-api-manager: document AI in conversations (per-message flow trigger, AIcall reuse, idle timeout, tool availability matrix)"
```

### Slice 8 verification

Manual: open `bin-api-manager/docsdev/build/html/index.html` in a browser; confirm the new sections render correctly and cross-links resolve.

---

## Slice 9 — Final service-wide verification + PR readiness

### Task 9.1: Run the full 5-step verification workflow per root CLAUDE.md

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-conversation-ai-talk-bridge/bin-ai-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: all 5 steps pass. If `go mod tidy` or `vendor` produces changes, commit them as a final hygiene commit:

```bash
git add bin-ai-manager/go.mod bin-ai-manager/go.sum
git commit -m "NOJIRA-conversation-ai-talk-bridge

- bin-ai-manager: go mod tidy/vendor after design implementation"
```

### Task 9.2: Pull latest main, check conflicts

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-conversation-ai-talk-bridge
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" || echo "no conflicts"
git log --oneline HEAD..origin/main
```

If conflicts: rebase or merge main, resolve, re-run Slice 9 Task 9.1.

### Task 9.3: PR creation (on user authorization)

PR title MUST match branch name: `NOJIRA-conversation-ai-talk-bridge`.

PR body draft:

```
Bridge bin-conversation-manager and bin-ai-manager so AI participates as a text
chatbot in SMS/LINE conversations. Closes the loop where the LLM reply
currently never reaches the user.

- bin-ai-manager: ai-manager owns delivery via ConversationV1MessageSend at EventPMMessageBotLLM
- bin-ai-manager: PipecatcallID response guard (primary + secondary) drops stale/raced LLM responses
- bin-ai-manager: ping-gated synchronous interrupt of previous pipecat session before new turn
- bin-ai-manager: AIcall reuse across turns with status+idle reusability check
- bin-ai-manager: 24h idle-timeout (AICALL_CONVERSATION_IDLE_TIMEOUT_HOURS) for conversation AIcalls
- bin-ai-manager: ActiveflowID rebinding on AIcall reuse so tools target the current flow
- bin-ai-manager: ConversationSafeTools whitelist + FilterToolsForConversation utility
- bin-ai-manager: aicall_idle_expired_total, aicall_interrupt_attempted_total, conversation_reply_send_total, aicall_stale_response_dropped_total metrics
- bin-ai-manager: AssistanceTypeAI and AssistanceTypeTeam share one path
- bin-pipecat-manager: no code changes
- bin-common-handler: no code changes
- bin-api-manager: docs/RST conversation+AI flow example and tool availability matrix
- monitoring: ai-manager dashboard panels for new counters
- docs: design at docs/plans/2026-04-27-conversation-ai-talk-design.md, plan at docs/plans/2026-04-27-conversation-ai-talk-plan.md
```

DO NOT use `gh pr create` until the user explicitly approves.

### Slice 9 verification

User approval required before PR creation. Until then, branch is ready for review locally.

---

## Quality gates summary

| Gate | When | Pass criteria |
|---|---|---|
| Per-slice unit tests | After each slice | `go test ./<changed pkg>/...` green |
| Voice-path regression | Slice 7 | All existing tests pass without modification |
| End-to-end integration | Slice 7 | api-validator scenario passes (or marked skip with reason) |
| Service-wide verification | Slice 9 | All 5 steps green: tidy, vendor, generate, test, lint |
| Conflict check vs main | Slice 9 | No conflicts (or resolved + re-verified) |
| Voice paths byte-for-byte unchanged | Continuous | No diff to voice-only test files; all voice tests pass |

## Execution Handoff

Plan complete and saved to `docs/plans/2026-04-27-conversation-ai-talk-plan.md`. Two execution options:

**1. Subagent-Driven (this session)** — dispatch fresh subagent per task, review between tasks, fast iteration.

**2. Parallel Session (separate)** — open new session with executing-plans, batch execution with checkpoints.

Which approach?
