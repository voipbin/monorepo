# AIcall terminate-drop & no-final-event recovery — implementation plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Stop silently dropping LLM replies in conversation-type AIcalls when Pipecat hangs mid-stream or the pipecatcall is terminated; if everything else fails, send a user-visible fallback.

**Architecture:** Three coordinated changes across two services. (1) `bin-pipecat-manager` adds a deterministic flush-and-finalize on terminate, an idle watchdog inside the streaming flush goroutine, and a new `pipecatcall_terminated` event. (2) `bin-ai-manager` records `pipecatcall_id` + `delivery_status` on each assistant row, persists `pending` then updates to `delivered` after a successful send, and adds a `EventPMPipecatcallTerminated` backstop handler that fires a fallback reply only when no `delivered` row exists for the pipecatcall. (3) `bin-dbscheme-manager` ships an Alembic migration with the two new columns + composite index.

**Tech Stack:** Go 1.22+, MySQL via sqlx, RabbitMQ pub/sub, Alembic (Python), gomock for tests, Prometheus client.

**Design doc:** `docs/plans/2026-04-29-aicall-terminate-drop-design.md` (this directory)

**Branch:** `NOJIRA-aicall-terminate-drop-design` (the worktree this plan lives in)

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-aicall-terminate-drop-design`

---

## Phasing

The plan is split into three phases that ship independently:

- **Phase 0** — Alembic migration (1 task block, ~15 min). Lands and bakes 2 h before Phase 1.
- **Phase 1** — `bin-pipecat-manager` (Tasks 1.1–1.18). Independent of Phase 2; bakes 24 h.
- **Phase 2** — `bin-ai-manager` (Tasks 2.1–2.20). Depends on Phase 0 and Phase 1.

Each phase ends with the standard verification workflow (`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`).

---

## Conventions used in this plan

- File paths are relative to the worktree root unless prefixed with `~`.
- Each task has 5 steps: (1) write failing test, (2) run to confirm fail, (3) implement, (4) run to confirm pass, (5) commit.
- For changes where TDD doesn't fit (constants, schema migrations, generated mocks), the test step is replaced with a "smoke check" — a build/compile/lint command that fails before and passes after.
- Commit messages follow the monorepo convention: title `NOJIRA-aicall-terminate-drop-design`, body with `bin-<service>:` prefixes per change. **Do not include AI attribution.**
- Run all commands from the worktree root unless noted otherwise.
- `go generate` regenerates mocks via `mockgen`; run it after any interface change.

---

## PHASE 0 — Alembic migration

### Task 0.1: Generate the migration file

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<auto-rev>_add_pipecatcall_id_delivery_status_to_ai_messages.py`

**Step 1: Generate empty revision via Alembic** (the only correct way — never hand-pick revision IDs)

```bash
cd bin-dbscheme-manager
alembic -c bin-manager/main/alembic.ini revision -m "ai_messages add pipecatcall_id delivery_status"
```

Expected output: a new file under `bin-manager/main/versions/` with auto-generated revision ID. Note the path — you'll edit it next.

**Step 2: Verify the generated file exists and has correct skeleton**

```bash
ls -la bin-manager/main/versions/*pipecatcall_id*.py
```

**Step 3: Edit the file** — fill in `upgrade()` and `downgrade()`:

```python
def upgrade():
    op.add_column('ai_messages',
        sa.Column('pipecatcall_id', mysql.BINARY(16), nullable=True))
    op.add_column('ai_messages',
        sa.Column('delivery_status', sa.String(16), nullable=False,
                  server_default='delivered'))
    op.create_index('idx_ai_messages_pcc_delivery', 'ai_messages',
                    ['pipecatcall_id', 'delivery_status'])

def downgrade():
    op.drop_index('idx_ai_messages_pcc_delivery', 'ai_messages')
    op.drop_column('ai_messages', 'delivery_status')
    op.drop_column('ai_messages', 'pipecatcall_id')
```

Make sure imports include `from sqlalchemy.dialects import mysql`.

**Step 4: Smoke-check with `alembic check`**

```bash
cd bin-dbscheme-manager && alembic -c bin-manager/main/alembic.ini check
```

Expected: no errors. Do **not** run `alembic upgrade` — that's a human's job per `bin-dbscheme-manager/CLAUDE.md`.

**Step 5: Commit**

```bash
git add bin-dbscheme-manager/bin-manager/main/versions/*pipecatcall_id*.py
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-dbscheme-manager: Add migration for ai_messages.pipecatcall_id (BINARY(16)) and ai_messages.delivery_status (VARCHAR(16) NOT NULL DEFAULT 'delivered'), with composite index idx_ai_messages_pcc_delivery
EOF
)"
```

---

## PHASE 1 — bin-pipecat-manager

> **Pre-flight:** `cd bin-pipecat-manager && go test ./... 2>&1 | tail -20` should be green before starting. If it isn't, stop and ask.

### Task 1.1: Add `EventTypePipecatcallTerminated` constant

**Files:**
- Modify: `bin-pipecat-manager/models/pipecatcall/event.go`
- Test: `bin-pipecat-manager/models/pipecatcall/event_test.go` (create)

**Step 1: Write the failing test**

```go
// bin-pipecat-manager/models/pipecatcall/event_test.go
package pipecatcall

import "testing"

func TestEventTypePipecatcallTerminated_value(t *testing.T) {
    if EventTypePipecatcallTerminated != "pipecatcall_terminated" {
        t.Fatalf("expected pipecatcall_terminated, got %q", EventTypePipecatcallTerminated)
    }
}
```

**Step 2: Run** — `cd bin-pipecat-manager && go test ./models/pipecatcall/ -run TestEventTypePipecatcallTerminated_value -v` — expected: undefined symbol error.

**Step 3: Implement** — append to `bin-pipecat-manager/models/pipecatcall/event.go`:

```go
EventTypePipecatcallTerminated string = "pipecatcall_terminated"
```

**Step 4: Run** — same command, expected: PASS.

**Step 5: Commit**

```bash
git add bin-pipecat-manager/models/pipecatcall/event.go bin-pipecat-manager/models/pipecatcall/event_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: Add EventTypePipecatcallTerminated event constant for the new terminate-publish flow
EOF
)"
```

### Task 1.2: Add `LLMFlushOnce` and `LLMStopReason` fields to Session

**Files:**
- Modify: `bin-pipecat-manager/models/pipecatcall/session.go`

**Step 1: Write the failing test**

```go
// bin-pipecat-manager/models/pipecatcall/session_test.go (extend existing or create)
func TestSession_FlushPrimitives(t *testing.T) {
    s := &Session{}
    // Channels are zero-valued; Once and atomic.Int32 are usable in zero-value.
    s.LLMFlushOnce.Do(func() {})  // should not panic
    if got := s.LLMStopReason.Load(); got != 0 {
        t.Fatalf("expected initial reason 0, got %d", got)
    }
}
```

**Step 2: Run** — `go test ./models/pipecatcall/ -run TestSession_FlushPrimitives -v` — expected: undefined fields.

**Step 3: Implement** — in `models/pipecatcall/session.go`, in the LLM section (lines 36-43), add:

```go
LLMFlushOnce  sync.Once    `json:"-"` // ensures LLMStopChan is closed at most once
LLMStopReason atomic.Int32 `json:"-"` // set by closer via CAS; read by flush goroutine for metric attribution
```

Add `"sync/atomic"` import if not present. `sync` is already imported.

**Step 4: Run** — same command, expected: PASS.

**Step 5: Commit**

```bash
git add bin-pipecat-manager/models/pipecatcall/session.go bin-pipecat-manager/models/pipecatcall/session_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: Add Session.LLMFlushOnce (sync.Once) and Session.LLMStopReason (atomic.Int32) for safe close-attribution between flush goroutine, BotLLMStopped handler, watchdog, and terminate
EOF
)"
```

### Task 1.3: Migrate `LLMFlushing` from `bool` to `atomic.Bool`

> Per design §3.7.1, this is a compile-break that touches existing test files.

**Files:**
- Modify: `bin-pipecat-manager/models/pipecatcall/session.go`
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` (lines 521, 526, 548)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go` (lines 294, 318, 325, 475)

**Step 1: Audit all sites first**

```bash
cd bin-pipecat-manager && grep -nR 'LLMFlushing' .
```

Expected sites: `models/pipecatcall/session.go:42`, `pkg/pipecatcallhandler/runner.go:521,526,548,551`, `pkg/pipecatcallhandler/runner_flush_test.go:294,318,325,475`. Confirm no others before proceeding.

**Step 2: Write the new failing test**

```go
// bin-pipecat-manager/models/pipecatcall/session_test.go (append)
func TestSession_LLMFlushing_isAtomicBool(t *testing.T) {
    s := &Session{}
    s.LLMFlushing.Store(true)
    if !s.LLMFlushing.Load() {
        t.Fatalf("expected LLMFlushing true after Store(true)")
    }
    s.LLMFlushing.Store(false)
    if s.LLMFlushing.Load() {
        t.Fatalf("expected LLMFlushing false after Store(false)")
    }
}
```

**Step 3: Run** — `go test ./models/pipecatcall/ -run TestSession_LLMFlushing_isAtomicBool -v` — expected: cannot call `.Store` on `bool`.

**Step 4: Migrate the field and all call sites**

In `session.go` line 42: `LLMFlushing atomic.Bool` (was `bool`). The `atomic.Bool` zero value is false; no init needed.

In `runner.go`:
- Line 521: `if !se.LLMFlushing { ...` → `if !se.LLMFlushing.Load() { ...`
- Line 526: `se.LLMFlushing = true` → `se.LLMFlushing.Store(true)`
- Line 548: `if se.LLMFlushing {` → `if se.LLMFlushing.Load() {`
- Line 551: **delete the `se.LLMFlushing = false` line entirely** — Task 1.7 will replace it with a deferred reset inside the goroutine.

In `runner_flush_test.go`:
- Line 294 — composite literal `LLMFlushing: true,` → remove from literal; insert `s.LLMFlushing.Store(true)` after the struct construction.
- Line 318: `se.LLMFlushing = false` → `se.LLMFlushing.Store(false)`.
- Line 325: `se.LLMFlushing = true` → `se.LLMFlushing.Store(true)`.
- Line 475: `se.LLMFlushing = true` → `se.LLMFlushing.Store(true)`.

**Step 5: Run the full pipecatcallhandler suite**

```bash
go test ./models/pipecatcall/... ./pkg/pipecatcallhandler/... -race -v 2>&1 | tail -30
```

Expected: PASS. (The intermediate state at this point may have a slightly stale handler at runner.go:551 — that's fine because the deletion is a no-op against the new test expectations; the goroutine-level reset comes in Task 1.7.)

**Step 6: Commit**

```bash
git add bin-pipecat-manager/models/pipecatcall/session.go bin-pipecat-manager/models/pipecatcall/session_test.go bin-pipecat-manager/pkg/pipecatcallhandler/runner.go bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: Migrate Session.LLMFlushing to atomic.Bool to remove the data race between the WebSocket read loop and the flush goroutine; update all call sites in runner.go and runner_flush_test.go
EOF
)"
```

### Task 1.4: Add `StopReason` enum and `reasonLabel` helper

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go`
- Test: `bin-pipecat-manager/pkg/pipecatcallhandler/runner_test.go` (extend or create)

**Step 1: Write the failing test**

```go
func TestReasonLabel(t *testing.T) {
    cases := []struct {
        in   StopReason
        want string
    }{
        {StopReasonNormal, "stopped_normal"},
        {StopReasonIdleWatchdog, "idle_watchdog"},
        {StopReasonTerminateForce, "terminate_force"},
        {StopReasonContextCancel, "context_cancelled"},
        {StopReasonUnset, "unknown"},
    }
    for _, c := range cases {
        if got := reasonLabel(c.in); got != c.want {
            t.Errorf("reasonLabel(%d) = %q, want %q", c.in, got, c.want)
        }
    }
}
```

**Step 2: Run** — undefined symbols.

**Step 3: Implement** — in `runner.go` (top-level, near other constants):

```go
type StopReason int32

const (
    StopReasonUnset StopReason = iota
    StopReasonNormal
    StopReasonIdleWatchdog
    StopReasonTerminateForce
    StopReasonContextCancel
)

func reasonLabel(r StopReason) string {
    switch r {
    case StopReasonNormal:
        return "stopped_normal"
    case StopReasonIdleWatchdog:
        return "idle_watchdog"
    case StopReasonTerminateForce:
        return "terminate_force"
    case StopReasonContextCancel:
        return "context_cancelled"
    default:
        return "unknown"
    }
}
```

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/runner.go bin-pipecat-manager/pkg/pipecatcallhandler/runner_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: Add StopReason enum and reasonLabel helper for goroutine exit attribution metrics
EOF
)"
```

### Task 1.5: Add the three new Prometheus metrics

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/metrics.go`
- Test: `bin-pipecat-manager/pkg/pipecatcallhandler/metrics_test.go` (extend or create)

**Step 1: Write the failing test**

```go
func TestNewMetrics_registered(t *testing.T) {
    // metrics package is registered at init; just assert the names exist.
    for _, name := range []string{
        "pipecat_manager_llm_flush_exit_total",
        "pipecat_manager_llm_idle_watchdog_fired_total",
        "pipecat_manager_llm_flush_finalize_outcome_total",
    } {
        if !metricRegistered(name) { // helper using prometheus.DefaultGatherer
            t.Errorf("metric %s not registered", name)
        }
    }
}
```

(If a `metricRegistered` helper doesn't exist, write a small one using `prometheus.DefaultGatherer.Gather()`.)

**Step 2: Run** — FAIL (metrics not registered).

**Step 3: Implement** — in `metrics.go`:

```go
var (
    metricsLLMFlushExit = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "pipecat_manager_llm_flush_exit_total",
        Help: "Counter of runLLMIntermediateFlush goroutine exits, by reason.",
    }, []string{"reason"})

    metricsIdleWatchdogFired = promauto.NewCounter(prometheus.CounterOpts{
        Name: "pipecat_manager_llm_idle_watchdog_fired_total",
        Help: "Counter of idle-watchdog firings (no tokens for idleWatchdogTimeout while flushing).",
    })

    metricsFlushFinalizeOutcome = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "pipecat_manager_llm_flush_finalize_outcome_total",
        Help: "Counter of flushAndFinalize outcomes from the terminate caller's perspective.",
    }, []string{"outcome"})
)
```

(Use `promauto` if the file already does; otherwise `prometheus.MustRegister` in an `init()`.)

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/metrics.go bin-pipecat-manager/pkg/pipecatcallhandler/metrics_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: Add Prometheus metrics for flush goroutine exits, idle-watchdog firings, and flushAndFinalize outcomes
EOF
)"
```

### Task 1.6: Switch `BotLLMStopped` handler to atomic CAS + Once

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` (lines 542-554)
- Test: `bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go` (extend)

**Step 1: Write the failing test** — verifies that calling `BotLLMStopped` twice in a row does not panic and only attributes the close once.

```go
func TestBotLLMStopped_doubleStop_noPanic(t *testing.T) {
    se := newTestSessionWithFlushArmed(t) // helper creates Session with channels + flush goroutine
    // Simulate first BotLLMStopped
    sendBotLLMStopped(t, se)
    // Simulate second BotLLMStopped (should be no-op due to Once + LLMFlushing=false)
    sendBotLLMStopped(t, se)
    // Reason should remain StopReasonNormal (set by first call), not double-toggled.
    if got := StopReason(se.LLMStopReason.Load()); got != StopReasonNormal {
        t.Fatalf("expected StopReasonNormal, got %d", got)
    }
}
```

**Step 2: Run** — FAIL (test won't compile until handler is updated).

**Step 3: Implement** — replace `runner.go:542-554`:

```go
case pipecatframe.RTVIFrameTypeBotLLMStopped:
    msg := pipecatframe.RTVIBotLLMStoppedMessage{}
    if err := json.Unmarshal(m, &msg); err != nil {
        return errors.Wrapf(err, "could not unmarshal bot-llm-stopped message")
    }
    if !se.LLMFlushing.Load() {
        log.Debugf("BotLLMStopped received but no tokens were received for this generation.")
        break
    }
    atomic.CompareAndSwapInt32((*int32)(unsafe.Pointer(&se.LLMStopReason)),
        int32(StopReasonUnset), int32(StopReasonNormal))
    // Cleaner alternative if Session.LLMStopReason is atomic.Int32:
    se.LLMStopReason.CompareAndSwap(int32(StopReasonUnset), int32(StopReasonNormal))
    se.LLMFlushOnce.Do(func() { close(se.LLMStopChan) })
    <-se.LLMDoneChan
```

(Use the `atomic.Int32.CompareAndSwap` form, not `unsafe.Pointer`.)

Note: the line `se.LLMFlushing = false` was already deleted in Task 1.3; the goroutine's `defer` (added in Task 1.7) owns the reset.

**Step 4: Run** — `go test ./pkg/pipecatcallhandler/... -race -v 2>&1 | tail -10` — expected: PASS.

**Step 5: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/runner.go bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: BotLLMStopped handler uses atomic CAS for StopReason and sync.Once for closing LLMStopChan, removing the unguarded close racing with watchdog and flushAndFinalize
EOF
)"
```

### Task 1.7: Add `defer LLMFlushing.Store(false)` inside `runLLMIntermediateFlush`

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` (top of `runLLMIntermediateFlush`)
- Test: `bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go` (extend)

**Step 1: Write the failing test** — verifies `LLMFlushing.Load()` returns false after the goroutine returns, regardless of which exit path was taken.

```go
func TestRunLLMIntermediateFlush_resetsLLMFlushing_onAllExits(t *testing.T) {
    cases := []struct {
        name   string
        causeExit func(*pipecatcall.Session)
    }{
        {"stopChan", func(s *pipecatcall.Session) { close(s.LLMStopChan) }},
        {"ctxDone",  func(s *pipecatcall.Session) { s.Cancel() }},
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            se := newTestSessionWithFlushArmed(t)
            c.causeExit(se)
            <-se.LLMDoneChan // wait for goroutine to exit
            if se.LLMFlushing.Load() {
                t.Fatalf("LLMFlushing not reset on %s exit", c.name)
            }
        })
    }
}
```

**Step 2: Run** — FAIL (LLMFlushing remains true after Ctx.Done path).

**Step 3: Implement** — at the top of `runLLMIntermediateFlush`, add:

```go
defer close(se.LLMDoneChan)        // already present
defer se.LLMFlushing.Store(false)  // NEW — runs before close so observers see flushing=false
```

The order of `defer` is LIFO: `LLMFlushing.Store(false)` runs first, then `close(LLMDoneChan)`. That's correct — readers waking on the close should see `LLMFlushing=false`.

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/runner.go bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: Reset Session.LLMFlushing inside runLLMIntermediateFlush via defer so it always clears regardless of which exit path triggered the goroutine to return
EOF
)"
```

### Task 1.8: Switch `publishFinalBotLLMEvent` and `publishIntermediateEvent` to `context.Background()`

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` (around lines 609 and 634; verify exact line numbers post-Task-1.7)
- Test: `bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go`

**Step 1: Write the failing test** — verifies that publishing succeeds even after `se.Ctx` is cancelled.

```go
func TestRunLLMIntermediateFlush_publishesAfterCtxCancel(t *testing.T) {
    se := newTestSessionWithFlushArmed(t)
    // Send one token, then cancel context, then close StopChan.
    se.LLMTokenChan <- "hello "
    se.Cancel()                  // cancels se.Ctx
    close(se.LLMStopChan)        // triggers final publish via the LLMStopChan branch
    // Wait for goroutine to exit, then verify a final event was published.
    <-se.LLMDoneChan
    assertFinalEventPublished(t, se, "hello ")
}
```

**Step 2: Run** — depending on which branch runs first, may fail because publish currently uses `se.Ctx`.

**Step 3: Implement** — in the `case <-se.LLMStopChan:` branch (around runner.go:634), replace:

```go
h.publishFinalBotLLMEvent(se.Ctx, se, messageID, fullText)
```

with:

```go
h.publishFinalBotLLMEvent(context.Background(), se, messageID, fullText)
```

In the `case <-ticker.C:` branch (around line 609), replace `h.publishIntermediateEvent(se, messageID, deltaBuffer, sequence)` (which internally uses `se.Ctx`) with a new helper or pass context explicitly:

```go
h.publishIntermediateEventBg(se, messageID, deltaBuffer, sequence)
```

Add the helper:

```go
func (h *pipecatcallHandler) publishIntermediateEventBg(se *pipecatcall.Session, messageID uuid.UUID, delta string, sequence int) {
    evt := message.Message{ /* same fields as publishIntermediateEvent */ }
    h.notifyHandler.PublishEvent(context.Background(), message.EventTypeBotLLMIntermediate, evt)
}
```

(Or simply change the existing `publishIntermediateEvent` to accept `ctx context.Context` and pass `context.Background()` from this call site.)

**Step 4: Run** — `go test ./pkg/pipecatcallhandler/... -race -v 2>&1 | tail -10` — PASS.

**Step 5: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/runner.go bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: Use context.Background() for both intermediate and final message_bot_llm publishes so a cancelled session context (from terminate path) doesn't drop the event
EOF
)"
```

### Task 1.9: Add idle watchdog to `runLLMIntermediateFlush`

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` (the select loop)
- Test: `bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go`

**Step 1: Write the failing test** — fires watchdog after silence, verifies metric and exit reason.

```go
func TestRunLLMIntermediateFlush_idleWatchdog_fires(t *testing.T) {
    // Use a test-injectable clock OR shorter idleWatchdogTimeout via build tags.
    se := newTestSessionWithFlushArmed(t)
    se.LLMTokenChan <- "first token "
    // Wait > idleWatchdogTimeout (use real time in test only if timeout is short enough)
    time.Sleep(idleWatchdogTimeout + 500*time.Millisecond)
    <-se.LLMDoneChan
    if got := StopReason(se.LLMStopReason.Load()); got != StopReasonIdleWatchdog {
        t.Fatalf("expected StopReasonIdleWatchdog, got %d", got)
    }
    assertMetricIncremented(t, "pipecat_manager_llm_idle_watchdog_fired_total", 1)
}

func TestRunLLMIntermediateFlush_idleWatchdog_doesNotFireBeforeFirstToken(t *testing.T) {
    se := newTestSessionWithFlushArmed(t)
    // No tokens. Wait > idleWatchdogTimeout. Watchdog should NOT fire.
    time.Sleep(idleWatchdogTimeout + 500*time.Millisecond)
    if se.LLMStopReason.Load() != int32(StopReasonUnset) {
        t.Fatalf("watchdog fired before any token arrived")
    }
    // Cleanup
    se.Cancel()
    <-se.LLMDoneChan
}
```

For local dev set `idleWatchdogTimeout = 200*time.Millisecond` via a test-only build tag or override variable. (Or use a global `var` instead of `const` so tests can patch.)

**Step 2: Run** — FAIL (watchdog logic not present).

**Step 3: Implement** — in `runner.go`, near other constants:

```go
var (
    idleWatchdogTimeout  = 8 * time.Second   // var (not const) so tests can patch
    idleWatchdogTickRate = 1 * time.Second
)
```

In `runLLMIntermediateFlush`, before the for-loop:

```go
watchdog := time.NewTicker(idleWatchdogTickRate)
defer watchdog.Stop()
var lastToken time.Time   // zero until first token
```

In the `case token := <-se.LLMTokenChan:` branch, add `lastToken = time.Now()`.

Add the new case in the select:

```go
case now := <-watchdog.C:
    if !lastToken.IsZero() && now.Sub(lastToken) >= idleWatchdogTimeout {
        if se.LLMStopReason.CompareAndSwap(int32(StopReasonUnset), int32(StopReasonIdleWatchdog)) {
            metricsIdleWatchdogFired.Inc()
        }
        se.LLMFlushOnce.Do(func() { close(se.LLMStopChan) })
    }
```

In the `case <-se.LLMStopChan:` branch, after publishing the final event, increment the exit metric:

```go
metricsLLMFlushExit.WithLabelValues(reasonLabel(StopReason(se.LLMStopReason.Load()))).Inc()
```

In the `case <-se.Ctx.Done():` branch, do the CAS first then increment:

```go
se.LLMStopReason.CompareAndSwap(int32(StopReasonUnset), int32(StopReasonContextCancel))
// existing publish-partial logic (already uses context.Background() per Task 1.8 if you extended that to here)
metricsLLMFlushExit.WithLabelValues(reasonLabel(StopReason(se.LLMStopReason.Load()))).Inc()
```

**Step 4: Run** — `go test ./pkg/pipecatcallhandler/... -race -v 2>&1 | tail -20` — PASS (note: tests use a patched short `idleWatchdogTimeout`).

**Step 5: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/runner.go bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: Add idle watchdog to runLLMIntermediateFlush — fires after idleWatchdogTimeout (8s) of no tokens, but only after the first token has arrived; exit reason and metric label reflect the watchdog-triggered exit
EOF
)"
```

### Task 1.10: Implement `flushAndFinalize` helper

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` (or wherever helpers live)
- Test: `bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go`

**Step 1: Write the failing test** — table-driven for the four outcomes:

```go
func TestFlushAndFinalize_outcomes(t *testing.T) {
    tt := []struct {
        name    string
        setup   func(*pipecatcall.Session)
        outcome string
    }{
        {"never_started", func(s *pipecatcall.Session) { /* no flush armed */ }, "noop_never_started"},
        {"already_done",  func(s *pipecatcall.Session) { armAndExitFlush(s) },   "noop_already_done"},
        {"done",          func(s *pipecatcall.Session) { armFlush(s) },          "done"},
        {"timeout",       func(s *pipecatcall.Session) { armFlushBlocking(s) },  "timeout"},
    }
    for _, c := range tt {
        t.Run(c.name, func(t *testing.T) {
            h := newTestHandler(t)
            se := newSession()
            c.setup(se)
            h.flushAndFinalize(se)
            assertOutcomeMetric(t, c.outcome, 1)
        })
    }
}
```

For the `timeout` case use a patched short `flushFinalizeTimeout` (e.g., 50 ms) and a flush goroutine that blocks longer.

**Step 2: Run** — FAIL.

**Step 3: Implement** — in `runner.go`:

```go
var flushFinalizeTimeout = 3 * time.Second // var so tests can patch

func (h *pipecatcallHandler) flushAndFinalize(se *pipecatcall.Session) {
    if !se.LLMFlushing.Load() {
        if se.LLMMessageID == uuid.Nil {
            metricsFlushFinalizeOutcome.WithLabelValues("noop_never_started").Inc()
        } else {
            metricsFlushFinalizeOutcome.WithLabelValues("noop_already_done").Inc()
        }
        return
    }

    se.LLMStopReason.CompareAndSwap(int32(StopReasonUnset), int32(StopReasonTerminateForce))
    se.LLMFlushOnce.Do(func() { close(se.LLMStopChan) })

    timer := time.NewTimer(flushFinalizeTimeout)
    defer timer.Stop()

    select {
    case <-se.LLMDoneChan:
        metricsFlushFinalizeOutcome.WithLabelValues("done").Inc()
    case <-timer.C:
        metricsFlushFinalizeOutcome.WithLabelValues("timeout").Inc()
    }
}
```

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/runner.go bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: Add flushAndFinalize — synchronously closes LLMStopChan and waits for the flush goroutine to publish its final event, with bounded timeout. Records four outcome labels for observability.
EOF
)"
```

### Task 1.11: Add `markTerminatedOnce` dedupe map

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/main.go` (handler struct + constructor)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/start.go` (helper)
- Test: `bin-pipecat-manager/pkg/pipecatcallhandler/start_test.go` (extend)

**Step 1: Write the failing test**

```go
func TestMarkTerminatedOnce_idempotent(t *testing.T) {
    h := newTestHandler(t)
    id := uuid.Must(uuid.NewV4())
    if !h.markTerminatedOnce(id) {
        t.Fatalf("first call should claim")
    }
    if h.markTerminatedOnce(id) {
        t.Fatalf("second call should not claim")
    }
}
```

**Step 2: Run** — FAIL.

**Step 3: Implement** — in `main.go` handler struct, add:

```go
muTerminated        sync.Mutex
terminatedPublished map[uuid.UUID]struct{}
```

In the constructor, initialize: `terminatedPublished: make(map[uuid.UUID]struct{})`.

In `start.go` (or `terminate.go` if a new file fits better):

```go
func (h *pipecatcallHandler) markTerminatedOnce(id uuid.UUID) bool {
    h.muTerminated.Lock()
    defer h.muTerminated.Unlock()
    if _, ok := h.terminatedPublished[id]; ok {
        return false
    }
    h.terminatedPublished[id] = struct{}{}
    return true
}

func (h *pipecatcallHandler) terminatedDeleteEntry(id uuid.UUID) {
    h.muTerminated.Lock()
    defer h.muTerminated.Unlock()
    delete(h.terminatedPublished, id)
}
```

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/main.go bin-pipecat-manager/pkg/pipecatcallhandler/start.go bin-pipecat-manager/pkg/pipecatcallhandler/start_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: Add in-memory terminatedPublished map + mutex with markTerminatedOnce/terminatedDeleteEntry helpers to dedupe pipecatcall_terminated event publication on repeated terminate() calls
EOF
)"
```

### Task 1.12: Update `SessionStop` to call `Cancel()` and prune dedupe entry

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/session.go` (around line 82)
- Test: `bin-pipecat-manager/pkg/pipecatcallhandler/session_test.go`

**Step 1: Write the failing test**

```go
func TestSessionStop_cancelsCtx(t *testing.T) {
    h, se := newTestHandlerWithSession(t)
    done := make(chan struct{})
    go func() {
        <-se.Ctx.Done()
        close(done)
    }()
    h.SessionStop(se.ID)
    select {
    case <-done:
    case <-time.After(100 * time.Millisecond):
        t.Fatalf("Ctx.Done() did not fire after SessionStop")
    }
}

func TestSessionStop_prunesTerminatedMap(t *testing.T) {
    h, se := newTestHandlerWithSession(t)
    h.markTerminatedOnce(se.ID) // simulate a prior terminate() publish
    h.SessionStop(se.ID)
    if h.markTerminatedOnce(se.ID) == false {
        t.Fatalf("expected map entry pruned; second markTerminatedOnce should claim again")
    }
}
```

**Step 2: Run** — FAIL.

**Step 3: Implement** — in `session.go`, in `SessionStop`, after the existing `wait on ConnAstReady or Ctx.Done` block (around line 105) and **before** `close(ConnAst)`:

```go
pc.Cancel()  // Session.Cancel — same func used by deferred cleanup in start.go
```

At the very end of `SessionStop` (after all teardown):

```go
h.terminatedDeleteEntry(pc.ID)
```

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/session.go bin-pipecat-manager/pkg/pipecatcallhandler/session_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: SessionStop now cancels pc.Ctx (so the flush goroutine's Ctx.Done branch fires) and prunes the terminatedPublished map entry to bound memory growth
EOF
)"
```

### Task 1.13: Update `terminate()` to call `flushAndFinalize` + publish + dedupe

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/start.go` (around line 300)
- Test: `bin-pipecat-manager/pkg/pipecatcallhandler/start_test.go`

**Step 1: Write the failing test**

```go
func TestTerminate_publishesTerminatedEventOnce(t *testing.T) {
    h, se := newTestHandlerWithSession(t)
    pc := pcFromSession(se)
    h.terminate(context.Background(), pc)
    h.terminate(context.Background(), pc) // second call
    publishCalls := h.notifyHandler.PublishEventCalls()
    terminatedCalls := 0
    for _, c := range publishCalls {
        if c.Type == pipecatcall.EventTypePipecatcallTerminated {
            terminatedCalls++
        }
    }
    if terminatedCalls != 1 {
        t.Fatalf("expected 1 terminated publish, got %d", terminatedCalls)
    }
}

func TestTerminate_callsFlushBeforeStop(t *testing.T) {
    h, se := newTestHandlerWithSession(t)
    pc := pcFromSession(se)
    armAndStallFlush(se) // flush goroutine is running
    h.terminate(context.Background(), pc)
    // After terminate, flush goroutine must have exited via terminate_force.
    if got := StopReason(se.LLMStopReason.Load()); got != StopReasonTerminateForce {
        t.Fatalf("expected StopReasonTerminateForce, got %d", got)
    }
}
```

**Step 2: Run** — FAIL.

**Step 3: Implement** — modify `terminate()`:

```go
func (h *pipecatcallHandler) terminate(ctx context.Context, pc *pipecatcall.Pipecatcall) {
    log := logrus.WithFields(...)
    log.Infof("Terminating pipecatcall...")

    if se, _ := h.SessionGet(pc.ID); se != nil {
        h.flushAndFinalize(se)  // C2
    }

    if h.markTerminatedOnce(pc.ID) {
        h.notifyHandler.PublishEvent(
            context.Background(),
            pipecatcall.EventTypePipecatcallTerminated,
            pc,
        )
    }

    // existing reference-type-specific cleanup
    switch pc.ReferenceType {
    case pipecatcall.ReferenceTypeCall:
        if errTerminate := h.terminateReferenceTypeCall(ctx, pc); errTerminate != nil { ... }
    case pipecatcall.ReferenceTypeAICall:
        if errTerminate := h.terminateReferenceTypeAICall(ctx, pc); errTerminate != nil { ... }
    default:
        log.Debugf("No action needed to stop for reference type: %v", pc.ReferenceType)
    }

    h.SessionStop(pc.ID)
    log.Infof("Pipecatcall terminated. pipecatcall_id: %s", pc.ID)
}
```

**Step 4: Run** — `go test ./pkg/pipecatcallhandler/... -race -v 2>&1 | tail -20` — PASS.

**Step 5: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/start.go bin-pipecat-manager/pkg/pipecatcallhandler/start_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: terminate() now (1) calls flushAndFinalize before any teardown so partial LLM responses are published, (2) emits pipecatcall_terminated event exactly once per pipecatcall via markTerminatedOnce, (3) preserves existing reference-type cleanup before SessionStop
EOF
)"
```

### Task 1.14: Verify Phase 1 fully

**Step 1: Run the full verification workflow**

```bash
cd bin-pipecat-manager && \
  go mod tidy && \
  go mod vendor && \
  go generate ./... && \
  go test ./... && \
  golangci-lint run -v --timeout 5m
```

Expected: all green. Fix any issues before proceeding.

**Step 2: Run with race detector**

```bash
cd bin-pipecat-manager && go test -race ./pkg/pipecatcallhandler/...
```

Expected: PASS, no data races reported.

**Step 3: Commit any housekeeping changes**

```bash
git add -u
git status  # confirm only go.mod/go.sum/vendor adjustments
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-pipecat-manager: Run go mod tidy + go mod vendor + go generate after Phase 1 changes
EOF
)" 2>/dev/null || echo "no changes to commit"
```

---

## PHASE 2 — bin-ai-manager

> **Pre-flight:** Phase 0 migration must be applied to staging/prod. Phase 1 must have baked 24 h with green metrics.

### Task 2.1: Add `DeliveryStatus` type to message model

**Files:**
- Modify: `bin-ai-manager/models/message/main.go`
- Test: `bin-ai-manager/models/message/main_test.go` (extend or create)

**Step 1: Write the failing test**

```go
func TestDeliveryStatus_values(t *testing.T) {
    if string(DeliveryStatusPending) != "pending" {
        t.Fatalf("expected pending")
    }
    if string(DeliveryStatusDelivered) != "delivered" {
        t.Fatalf("expected delivered")
    }
}
```

**Step 2: Run** — FAIL.

**Step 3: Implement** — in `bin-ai-manager/models/message/main.go`:

```go
type DeliveryStatus string

const (
    DeliveryStatusPending   DeliveryStatus = "pending"
    DeliveryStatusDelivered DeliveryStatus = "delivered"
)
```

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-ai-manager/models/message/main.go bin-ai-manager/models/message/main_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: Add DeliveryStatus enum (pending|delivered) for the new message-delivery-truth tracking
EOF
)"
```

### Task 2.2: Add `Message.PipecatcallID` and `Message.DeliveryStatus` fields

**Files:**
- Modify: `bin-ai-manager/models/message/main.go` (Message struct)
- Test: `bin-ai-manager/models/message/main_test.go`

**Step 1: Write the failing test** — verifies the new fields exist on Message and are NOT in WebhookMessage.

```go
func TestMessage_hasPipecatcallIDAndDeliveryStatus(t *testing.T) {
    m := Message{
        PipecatcallID:  uuid.Must(uuid.NewV4()),
        DeliveryStatus: DeliveryStatusPending,
    }
    if m.PipecatcallID == uuid.Nil { t.Fatal("PipecatcallID not set") }
    if m.DeliveryStatus != DeliveryStatusPending { t.Fatal("DeliveryStatus not set") }
}

func TestWebhookMessage_omitsInternalFields(t *testing.T) {
    m := Message{
        PipecatcallID:  uuid.Must(uuid.NewV4()),
        DeliveryStatus: DeliveryStatusDelivered,
    }
    wm := m.ConvertWebhookMessage()
    raw, _ := json.Marshal(wm)
    if strings.Contains(string(raw), "pipecatcall_id") {
        t.Fatalf("webhook leaks pipecatcall_id: %s", raw)
    }
    if strings.Contains(string(raw), "delivery_status") {
        t.Fatalf("webhook leaks delivery_status: %s", raw)
    }
}
```

**Step 2: Run** — FAIL.

**Step 3: Implement** — in `Message` struct, append:

```go
PipecatcallID  uuid.UUID      `json:"-" db:"pipecatcall_id"`
DeliveryStatus DeliveryStatus `json:"-" db:"delivery_status"`
```

(`json:"-"` ensures these never appear in JSON serialization, including the WebhookMessage path.)

Confirm `ConvertWebhookMessage()` doesn't explicitly copy these — it shouldn't, since it constructs `WebhookMessage` from named fields.

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-ai-manager/models/message/main.go bin-ai-manager/models/message/main_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: Add Message.PipecatcallID and Message.DeliveryStatus fields. Both are internal-only (json:"-"); ConvertWebhookMessage does not expose them per the WebhookMessage-vs-internal-model rule.
EOF
)"
```

### Task 2.3: Add the new columns to dbhandler INSERT/SELECT

**Files:**
- Modify: `bin-ai-manager/pkg/dbhandler/message.go`
- Test: `bin-ai-manager/pkg/dbhandler/message_test.go`

**Step 1: Write the failing test** — round-trips a Message with the new fields through the DB layer.

```go
func TestMessageCreate_persistsPipecatcallIDAndDeliveryStatus(t *testing.T) {
    h, mockDB := newTestDBHandler(t)
    pcID := uuid.Must(uuid.NewV4())
    msg := &message.Message{
        Identity:       commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
        AicallID:       uuid.Must(uuid.NewV4()),
        PipecatcallID:  pcID,
        DeliveryStatus: message.DeliveryStatusPending,
        // ... other required fields
    }
    mockDB.ExpectExec("INSERT INTO ai_messages.*pipecatcall_id.*delivery_status.*")...
    err := h.MessageCreate(ctx, msg)
    require.NoError(t, err)
}
```

(Use the existing test infra; if mocks are sqlmock, patterns above apply. If gomock against an interface, mirror that.)

**Step 2: Run** — FAIL (column not in INSERT).

**Step 3: Implement** — add `pipecatcall_id` and `delivery_status` to the INSERT column list and `:pipecatcall_id`, `:delivery_status` to the values clause. Add the same columns to the SELECT used by `MessageGet`.

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/dbhandler/message.go bin-ai-manager/pkg/dbhandler/message_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: dbhandler/message persists and reads the new pipecatcall_id and delivery_status columns
EOF
)"
```

### Task 2.4: Add `MessageAssistantReplyExists`

**Files:**
- Modify: `bin-ai-manager/pkg/dbhandler/main.go` (interface)
- Modify: `bin-ai-manager/pkg/dbhandler/message.go` (impl)
- Test: `bin-ai-manager/pkg/dbhandler/message_test.go`

**Step 1: Write the failing test**

```go
func TestMessageAssistantReplyExists(t *testing.T) {
    cases := []struct{
        name     string
        rows     []*message.Message  // pre-seeded
        query    uuid.UUID            // pipecatcall_id
        expected bool
    }{
        {"delivered_match", []*message.Message{{PipecatcallID: pcA, DeliveryStatus: DeliveryStatusDelivered, Direction: DirectionIncoming, Role: RoleAssistant}}, pcA, true},
        {"pending_only",    []*message.Message{{PipecatcallID: pcA, DeliveryStatus: DeliveryStatusPending,   Direction: DirectionIncoming, Role: RoleAssistant}}, pcA, false},
        {"different_pcc",   []*message.Message{{PipecatcallID: pcB, DeliveryStatus: DeliveryStatusDelivered, Direction: DirectionIncoming, Role: RoleAssistant}}, pcA, false},
        {"wrong_role",      []*message.Message{{PipecatcallID: pcA, DeliveryStatus: DeliveryStatusDelivered, Direction: DirectionIncoming, Role: RoleUser}}, pcA, false},
        {"deleted_row",     []*message.Message{{PipecatcallID: pcA, DeliveryStatus: DeliveryStatusDelivered, Direction: DirectionIncoming, Role: RoleAssistant, TMDelete: someTimestamp}}, pcA, false},
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) { /* seed + assert */ })
    }
}
```

**Step 2: Run** — FAIL.

**Step 3: Implement** — in `dbhandler/main.go` interface, add: `MessageAssistantReplyExists(ctx context.Context, pipecatcallID uuid.UUID) (bool, error)`. In `dbhandler/message.go`:

```go
func (h *handler) MessageAssistantReplyExists(ctx context.Context, pipecatcallID uuid.UUID) (bool, error) {
    const q = `SELECT 1 FROM ai_messages
               WHERE pipecatcall_id = ?
                 AND delivery_status = 'delivered'
                 AND direction = 'incoming'
                 AND role = 'assistant'
                 AND tm_delete IS NULL
               LIMIT 1`
    var x int
    err := h.db.QueryRowContext(ctx, q, pipecatcallID).Scan(&x)
    if err == sql.ErrNoRows {
        return false, nil
    }
    if err != nil {
        return false, fmt.Errorf("MessageAssistantReplyExists: %w", err)
    }
    return true, nil
}
```

(Adjust to `sqlx` style if the codebase uses it.)

Run `go generate ./pkg/dbhandler/...` to regenerate the mock.

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/dbhandler/main.go bin-ai-manager/pkg/dbhandler/message.go bin-ai-manager/pkg/dbhandler/mock_main.go bin-ai-manager/pkg/dbhandler/message_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: Add dbhandler.MessageAssistantReplyExists query — single-row indexed SELECT scoped by pipecatcall_id with delivery_status='delivered' filter; regenerate mock
EOF
)"
```

### Task 2.5: Add `MessageUpdateDeliveryStatus`

**Files:**
- Modify: `bin-ai-manager/pkg/dbhandler/main.go` (interface)
- Modify: `bin-ai-manager/pkg/dbhandler/message.go` (impl)
- Test: `bin-ai-manager/pkg/dbhandler/message_test.go`

**Step 1: Write the failing test**

```go
func TestMessageUpdateDeliveryStatus(t *testing.T) {
    h, mockDB := newTestDBHandler(t)
    id := uuid.Must(uuid.NewV4())
    mockDB.ExpectExec("UPDATE ai_messages SET delivery_status = ?.*WHERE id = ?").
        WithArgs("delivered", id[:]).
        WillReturnResult(sqlmock.NewResult(0, 1))
    err := h.MessageUpdateDeliveryStatus(ctx, id, message.DeliveryStatusDelivered)
    require.NoError(t, err)
}
```

**Step 2: Run** — FAIL.

**Step 3: Implement**

```go
func (h *handler) MessageUpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status message.DeliveryStatus) error {
    const q = `UPDATE ai_messages SET delivery_status = ?, tm_update = ? WHERE id = ?`
    _, err := h.db.ExecContext(ctx, q, string(status), h.utilHandler.TimeGetCurTime(), id)
    if err != nil {
        return fmt.Errorf("MessageUpdateDeliveryStatus: %w", err)
    }
    return nil
}
```

Add to interface, regenerate mock.

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/dbhandler/main.go bin-ai-manager/pkg/dbhandler/message.go bin-ai-manager/pkg/dbhandler/mock_main.go bin-ai-manager/pkg/dbhandler/message_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: Add dbhandler.MessageUpdateDeliveryStatus for the persist-then-mark-delivered flow
EOF
)"
```

### Task 2.6: Add `CreateOption` functional options

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/main.go`
- Test: `bin-ai-manager/pkg/messagehandler/main_test.go`

**Step 1: Write the failing test**

```go
func TestCreateOptions_apply(t *testing.T) {
    pcID := uuid.Must(uuid.NewV4())
    var p createParams
    WithPipecatcallID(pcID)(&p)
    WithDeliveryStatus(message.DeliveryStatusPending)(&p)
    if p.pipecatcallID != pcID || p.deliveryStatus != message.DeliveryStatusPending {
        t.Fatalf("options not applied: %+v", p)
    }
}
```

**Step 2: Run** — FAIL.

**Step 3: Implement** — in `messagehandler/main.go`:

```go
type CreateOption func(*createParams)
type createParams struct {
    pipecatcallID  uuid.UUID
    deliveryStatus message.DeliveryStatus
}
func WithPipecatcallID(id uuid.UUID) CreateOption {
    return func(p *createParams) { p.pipecatcallID = id }
}
func WithDeliveryStatus(s message.DeliveryStatus) CreateOption {
    return func(p *createParams) { p.deliveryStatus = s }
}
```

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/messagehandler/main.go bin-ai-manager/pkg/messagehandler/main_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: Add functional CreateOption / WithPipecatcallID / WithDeliveryStatus so messagehandler.Create can accept new fields without breaking the 8+ existing callers
EOF
)"
```

### Task 2.7: Extend `MessageHandler.Create` signature with `opts ...CreateOption`

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/main.go` (interface + impl)
- Modify: `bin-ai-manager/pkg/messagehandler/main_test.go`
- Regenerate mocks

**Step 1: Write the failing test**

```go
func TestCreate_withPipecatcallID_andDeliveryStatus(t *testing.T) {
    h, mockDB := newTestMessageHandler(t)
    pcID := uuid.Must(uuid.NewV4())
    mockDB.EXPECT().MessageCreate(gomock.Any(), messageWithPCC(pcID, message.DeliveryStatusPending)).Return(nil)
    msg, err := h.Create(ctx, uuid.Nil, customerID, aicallID, activeflowID,
        message.DirectionIncoming, message.RoleAssistant, "hi", nil, "",
        WithPipecatcallID(pcID), WithDeliveryStatus(message.DeliveryStatusPending))
    require.NoError(t, err)
    require.Equal(t, pcID, msg.PipecatcallID)
    require.Equal(t, message.DeliveryStatusPending, msg.DeliveryStatus)
}

func TestCreate_withoutOpts_defaultsDelivered(t *testing.T) {
    // Existing callers don't supply opts → DeliveryStatus defaults to "delivered"
    // (matching the DB column default + existing call-site semantics).
    h, mockDB := newTestMessageHandler(t)
    mockDB.EXPECT().MessageCreate(gomock.Any(), messageWithDelivery(message.DeliveryStatusDelivered)).Return(nil)
    _, err := h.Create(ctx, uuid.Nil, customerID, aicallID, activeflowID,
        message.DirectionIncoming, message.RoleAssistant, "hi", nil, "")
    require.NoError(t, err)
}
```

**Step 2: Run** — FAIL.

**Step 3: Implement** — change interface and impl:

```go
Create(
    ctx context.Context,
    id, customerID, aicallID, activeflowID uuid.UUID,
    direction message.Direction,
    role message.Role,
    content string,
    toolCalls []message.ToolCall,
    toolCallID string,
    opts ...CreateOption,
) (*message.Message, error)
```

Inside the impl, build `createParams` with sensible defaults:

```go
p := createParams{
    pipecatcallID:  uuid.Nil,
    deliveryStatus: message.DeliveryStatusDelivered, // safe default for legacy callers
}
for _, opt := range opts { opt(&p) }
```

Then populate `Message.PipecatcallID` and `Message.DeliveryStatus` from `p` before calling `dbHandler.MessageCreate`.

Regenerate mocks: `cd bin-ai-manager && go generate ./pkg/messagehandler/...`.

**Step 4: Run** — `go test ./pkg/messagehandler/... -v` — PASS.

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/messagehandler/main.go bin-ai-manager/pkg/messagehandler/mock_main.go bin-ai-manager/pkg/messagehandler/main_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: Extend MessageHandler.Create with variadic opts ...CreateOption; default delivery_status='delivered' for legacy callers; regenerate mocks. No callers broken.
EOF
)"
```

### Task 2.8: Update `EventPMMessageBotLLM` — persist 'pending', update to 'delivered' after send

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/event.go` (EventPMMessageBotLLM, AICall conversation branch)
- Test: `bin-ai-manager/pkg/messagehandler/event_test.go`

**Step 1: Write the failing tests** — table-driven for the four outcomes:

```go
func TestEventPMMessageBotLLM_conversation(t *testing.T) {
    cases := []struct{
        name              string
        guard1Pass        bool
        guard2Pass        bool
        sendOK            bool
        updateOK          bool
        updateRetryOK     bool
        wantPersistStatus message.DeliveryStatus
        wantUpdateCalls   int
        wantConvSendCalls int
    }{
        {"happy",                true, true,  true,  true,  true, "pending", 1, 1},
        {"guard2_fail",          true, false, false, false, false, "pending", 0, 0},
        {"send_fail",            true, true,  false, false, false, "pending", 0, 1},
        {"update_fail_but_retry_ok", true, true, true, false, true, "pending", 2, 1},
        {"update_fail_both",     true, true,  true,  false, false, "pending", 2, 1},
    }
    // For each case: setup mocks, call handler, assert MessageCreate gets persistStatus=pending,
    // assert MessageUpdateDeliveryStatus is called wantUpdateCalls times,
    // assert ConversationV1MessageSend is called wantConvSendCalls times.
}
```

**Step 2: Run** — FAIL.

**Step 3: Implement** — modify `event.go` AICall conversation branch (lines ~78-130). Key changes:

```go
// Guard #1 (unchanged)
if ac.PipecatcallID != evt.PipecatcallID { ... return }

// Persist with delivery_status='pending' (NEW)
tmp, err := h.Create(ctx, evt.ID, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
    message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "",
    WithPipecatcallID(evt.PipecatcallID),
    WithDeliveryStatus(message.DeliveryStatusPending))
if err != nil { ... return }

// Guard #2 (unchanged) — if fail, row stays 'pending' → backstop will fire later.
acFinal, errFinal := h.reqHandler.AIV1AIcallGet(ctx, evt.PipecatcallReferenceID)
if errFinal != nil { ... return }
if acFinal.PipecatcallID != evt.PipecatcallID { ... return }

// Send (unchanged)
sent, errSend := h.reqHandler.ConversationV1MessageSend(ctx, acFinal.ReferenceID, evt.Text, []cvmedia.Media{})
if errSend != nil { ... return } // row stays 'pending'

// Mark delivered with one retry (NEW)
errUpd := h.dbHandler.MessageUpdateDeliveryStatus(ctx, tmp.ID, message.DeliveryStatusDelivered)
if errUpd != nil {
    time.Sleep(deliveryStatusUpdateRetryDelay)
    errUpd = h.dbHandler.MessageUpdateDeliveryStatus(ctx, tmp.ID, message.DeliveryStatusDelivered)
}
if errUpd != nil {
    log.Errorf("Could not mark message delivered after retry. msg_id: %s err: %v", tmp.ID, errUpd)
    promConversationDeliveryStatusUpdateFailedTotal.Inc()
}

promConversationReplySendTotal.WithLabelValues("success").Inc()
log.Debugf("Sent conversation reply.")
```

Add the constant `deliveryStatusUpdateRetryDelay = 100 * time.Millisecond` near other timeouts.

**Step 4: Run** — `go test ./pkg/messagehandler/... -v` — PASS.

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/messagehandler/event.go bin-ai-manager/pkg/messagehandler/event_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: EventPMMessageBotLLM AICall-conversation branch persists with delivery_status='pending' and only updates to 'delivered' after a successful conversation send (with one 100ms retry on the update). A guard-#2 failure or send failure leaves the row 'pending' so the backstop will fire.
EOF
)"
```

### Task 2.9: Add `promConversationDeliveryStatusUpdateFailedTotal` metric

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/metrics.go` (or wherever `promConversationReplySendTotal` lives)

**Step 1: Write the failing test**

```go
func TestPromConversationDeliveryStatusUpdateFailedTotal_registered(t *testing.T) {
    if !metricRegistered("ai_manager_message_delivery_status_update_failed_total") {
        t.Fatal("metric not registered")
    }
}
```

**Step 2: Run** — FAIL.

**Step 3: Implement**

```go
promConversationDeliveryStatusUpdateFailedTotal = promauto.NewCounter(prometheus.CounterOpts{
    Name: "ai_manager_message_delivery_status_update_failed_total",
    Help: "Counter of post-send delivery_status updates that failed even after one retry.",
})
```

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/messagehandler/metrics.go bin-ai-manager/pkg/messagehandler/metrics_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: Add Prometheus counter ai_manager_message_delivery_status_update_failed_total
EOF
)"
```

### Task 2.10: Add `EventPMPipecatcallTerminated` to MessageHandler interface

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/main.go` (interface)
- Regenerate mocks (no impl yet)

**Step 1: Write the failing test**

```go
func TestMessageHandler_hasEventPMPipecatcallTerminated(t *testing.T) {
    var _ interface {
        EventPMPipecatcallTerminated(context.Context, *pmpipecatcall.Pipecatcall) error
    } = (MessageHandler)(nil)
}
```

**Step 2: Run** — FAIL (interface doesn't have the method).

**Step 3: Implement** — add to the interface:

```go
EventPMPipecatcallTerminated(ctx context.Context, evt *pmpipecatcall.Pipecatcall) error
```

Add a stub to the impl:

```go
func (h *messageHandler) EventPMPipecatcallTerminated(ctx context.Context, evt *pmpipecatcall.Pipecatcall) error {
    return errors.New("not implemented") // implementation in Task 2.11
}
```

Regenerate mocks: `go generate ./pkg/messagehandler/...`.

**Step 4: Run** — PASS (interface check), other tests still green (the stub returns an error but no caller invokes it yet).

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/messagehandler/main.go bin-ai-manager/pkg/messagehandler/mock_main.go bin-ai-manager/pkg/messagehandler/main_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: Declare MessageHandler.EventPMPipecatcallTerminated interface method (stub impl) so mockgen regenerates; real impl lands in the next commit
EOF
)"
```

### Task 2.11: Add `promBackstopReplyTotal` metric

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/metrics.go`

**Step 1: Write the failing test**

```go
func TestPromBackstopReplyTotal_registered(t *testing.T) {
    if !metricRegistered("ai_manager_aicall_backstop_reply_total") {
        t.Fatal("metric not registered")
    }
}
```

**Step 2: Run** — FAIL.

**Step 3: Implement**

```go
promBackstopReplyTotal = promauto.NewCounterVec(prometheus.CounterOpts{
    Name: "ai_manager_aicall_backstop_reply_total",
    Help: "Counter of pipecatcall_terminated backstop attempts in messagehandler.EventPMPipecatcallTerminated.",
}, []string{"result"})
```

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/metrics.go bin-ai-manager/pkg/aicallhandler/metrics_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: Add Prometheus counter ai_manager_aicall_backstop_reply_total{result}
EOF
)"
```

### Task 2.12: Implement `EventPMPipecatcallTerminated` handler

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/event_pm.go` (or `event.go`, wherever `EventPMMessageBotLLM` lives)
- Test: `bin-ai-manager/pkg/messagehandler/event_pm_test.go`

**Step 1: Write the failing tests** — one per result label:

```go
func TestEventPMPipecatcallTerminated(t *testing.T) {
    cases := []struct{
        name           string
        evRefType      pmpipecatcall.ReferenceType
        aicallRefType  amaicall.ReferenceType
        aicallStatus   amaicall.Status
        replyExists    bool
        sendOK         bool
        wantResultLabel string
        wantSendCalls  int
    }{
        {"skipped_not_aicall",  pmpipecatcall.ReferenceTypeCall, "", "", false, false, "skipped_not_aicall", 0},
        {"skipped_voice",       pmpipecatcall.ReferenceTypeAICall, amaicall.ReferenceTypeCall, amaicall.StatusProgressing, false, false, "skipped_voice", 0},
        {"skipped_terminated",  pmpipecatcall.ReferenceTypeAICall, amaicall.ReferenceTypeConversation, amaicall.StatusTerminated, false, false, "skipped_terminated", 0},
        {"skipped_seen",        pmpipecatcall.ReferenceTypeAICall, amaicall.ReferenceTypeConversation, amaicall.StatusProgressing, true,  false, "skipped_seen", 0},
        {"sent",                pmpipecatcall.ReferenceTypeAICall, amaicall.ReferenceTypeConversation, amaicall.StatusProgressing, false, true,  "sent", 1},
        {"send_failed",         pmpipecatcall.ReferenceTypeAICall, amaicall.ReferenceTypeConversation, amaicall.StatusProgressing, false, false, "send_failed", 1},
        {"failed",              /* DB Create returns error */ ...},
    }
    // Setup gomock per branch; assert metric label + ConversationV1MessageSend call count.
}
```

For grace delay testing, inject a fake sleep — e.g., a package-level `var backstopGraceSleep = time.Sleep` that tests override.

**Step 2: Run** — FAIL.

**Step 3: Implement** — in `event_pm.go` (replacing the stub):

```go
const (
    backstopGraceDelay = 3 * time.Second
    backstopReplyText  = "Sorry, I'm having trouble responding right now. Please try again."
)

var backstopGraceSleep = time.Sleep // var so tests can patch

func (h *messageHandler) EventPMPipecatcallTerminated(ctx context.Context, ev *pmpipecatcall.Pipecatcall) error {
    log := logrus.WithFields(logrus.Fields{
        "func":           "EventPMPipecatcallTerminated",
        "pipecatcall_id": ev.ID,
    })

    if ev.ReferenceType != pmpipecatcall.ReferenceTypeAICall {
        promBackstopReplyTotal.WithLabelValues("skipped_not_aicall").Inc()
        return nil
    }

    aicall, err := h.reqHandler.AIV1AIcallGet(ctx, ev.ReferenceID)
    if err != nil {
        log.Errorf("Could not get aicall. err: %v", err)
        return nil
    }
    if aicall.ReferenceType != amaicall.ReferenceTypeConversation {
        promBackstopReplyTotal.WithLabelValues("skipped_voice").Inc()
        return nil
    }
    if aicall.Status == amaicall.StatusTerminated {
        promBackstopReplyTotal.WithLabelValues("skipped_terminated").Inc()
        return nil
    }

    backstopGraceSleep(backstopGraceDelay)

    seen, err := h.dbHandler.MessageAssistantReplyExists(ctx, ev.ID)
    if err != nil {
        return err
    }
    if seen {
        promBackstopReplyTotal.WithLabelValues("skipped_seen").Inc()
        return nil
    }

    msg, err := h.Create(ctx, uuid.Nil, aicall.CustomerID, aicall.ID, aicall.ActiveflowID,
        message.DirectionIncoming, message.RoleAssistant, backstopReplyText, nil, "",
        WithPipecatcallID(ev.ID),
        WithDeliveryStatus(message.DeliveryStatusDelivered))
    if err != nil {
        promBackstopReplyTotal.WithLabelValues("failed").Inc()
        return errors.Wrap(err, "could not persist backstop message")
    }

    if _, errSend := h.reqHandler.ConversationV1MessageSend(ctx, aicall.ReferenceID,
        backstopReplyText, []cvmedia.Media{}); errSend != nil {
        promBackstopReplyTotal.WithLabelValues("send_failed").Inc()
        return errors.Wrap(errSend, "could not send backstop conversation reply")
    }

    promBackstopReplyTotal.WithLabelValues("sent").Inc()
    log.WithField("message_id", msg.ID).Infof("Backstop reply sent. aicall_id: %s, pipecatcall_id: %s", aicall.ID, ev.ID)
    return nil
}
```

**Step 4: Run** — PASS.

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/messagehandler/event_pm.go bin-ai-manager/pkg/messagehandler/event_pm_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: Implement EventPMPipecatcallTerminated backstop handler with 7 result labels (sent | failed | send_failed | skipped_seen | skipped_voice | skipped_terminated | skipped_not_aicall). Persists the backstop reply with delivery_status='delivered' before sending so replays of the same event short-circuit on AssistantReplyExists.
EOF
)"
```

### Task 2.13: Wire the new event into `subscribehandler`

**Files:**
- Modify: `bin-ai-manager/pkg/subscribehandler/pipecat_pipecatcall.go` (add adapter)
- Modify: `bin-ai-manager/pkg/subscribehandler/main.go` (dispatch case)
- Test: `bin-ai-manager/pkg/subscribehandler/pipecat_pipecatcall_test.go`

**Step 1: Write the failing test**

```go
func TestProcessEventPMPipecatcallTerminated_dispatches(t *testing.T) {
    h, mockMH := newTestSubscribeHandler(t)
    pc := &pmpipecatcall.Pipecatcall{ID: uuid.Must(uuid.NewV4())}
    raw, _ := json.Marshal(pc)
    ev := &sock.Event{Type: pmpipecatcall.EventTypePipecatcallTerminated, Data: raw}
    mockMH.EXPECT().EventPMPipecatcallTerminated(gomock.Any(), pc).Return(nil)
    err := h.processEventPMPipecatcallTerminated(ctx, ev)
    require.NoError(t, err)
}
```

**Step 2: Run** — FAIL.

**Step 3: Implement** — in `subscribehandler/pipecat_pipecatcall.go`:

```go
func (h *subscribeHandler) processEventPMPipecatcallTerminated(ctx context.Context, m *sock.Event) error {
    var pc pmpipecatcall.Pipecatcall
    if err := json.Unmarshal(m.Data, &pc); err != nil {
        return errors.Wrap(err, "could not unmarshal pipecatcall_terminated payload")
    }
    return h.messageHandler.EventPMPipecatcallTerminated(ctx, &pc)
}
```

In `subscribehandler/main.go`, find the `switch (publisher, type)` block (~line 167-186) and add the case for `pmpipecatcall.EventTypePipecatcallTerminated`:

```go
case pmpipecatcall.EventTypePipecatcallTerminated:
    return h.processEventPMPipecatcallTerminated(ctx, m)
```

**Step 4: Run** — `go test ./pkg/subscribehandler/... -v` — PASS.

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/subscribehandler/pipecat_pipecatcall.go bin-ai-manager/pkg/subscribehandler/main.go bin-ai-manager/pkg/subscribehandler/pipecat_pipecatcall_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: Add subscribehandler.processEventPMPipecatcallTerminated adapter and dispatch case so pipecat-manager's pipecatcall_terminated event reaches the new MessageHandler.EventPMPipecatcallTerminated backstop
EOF
)"
```

### Task 2.14: Verify Phase 2 fully

**Step 1: Run the full verification workflow**

```bash
cd bin-ai-manager && \
  go mod tidy && \
  go mod vendor && \
  go generate ./... && \
  go test ./... && \
  golangci-lint run -v --timeout 5m
```

Expected: all green.

**Step 2: Race detector**

```bash
cd bin-ai-manager && go test -race ./pkg/messagehandler/... ./pkg/subscribehandler/... ./pkg/aicallhandler/... ./pkg/dbhandler/...
```

Expected: PASS.

**Step 3: Commit any housekeeping**

```bash
git add -u
git commit -m "$(cat <<'EOF'
NOJIRA-aicall-terminate-drop-design

- bin-ai-manager: Run go mod tidy + go mod vendor + go generate after Phase 2 changes
EOF
)" 2>/dev/null || echo "no changes to commit"
```

---

## Final integration check

### Task 3.1: Cross-service verification

**Step 1: Build both services from a clean checkout**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-aicall-terminate-drop-design
for svc in bin-pipecat-manager bin-ai-manager; do
  echo "=== $svc ==="
  cd "$svc" && go build ./... && cd -
done
```

Expected: clean compile for both.

**Step 2: Confirm no untracked files**

```bash
git status --short
```

Expected: empty (or only the design+plan docs).

**Step 3: Pull latest main and check for conflicts** (per root CLAUDE.md mandatory rule):

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" || echo "no conflicts"
git log --oneline HEAD..origin/main
```

If conflicts exist, rebase and re-run the verification workflow before opening a PR.

**Step 4: Push and open PR**

```bash
git push -u origin NOJIRA-aicall-terminate-drop-design
gh pr create --title "NOJIRA-aicall-terminate-drop-design" --body "$(cat <<'EOF'
Recover AIcall replies that today are silently dropped when the LLM stream stalls or the pipecatcall terminates mid-stream. Add a user-visible fallback for the worst-case path where no LLM event ever lands.

- bin-pipecat-manager: New EventTypePipecatcallTerminated; flushAndFinalize on terminate; idle watchdog with first-token guard; SessionStop cancels pc.Ctx; atomic StopReason + sync.Once for safe close attribution; three new Prometheus counters
- bin-ai-manager: New ai_messages.pipecatcall_id and ai_messages.delivery_status columns; persist-then-update flow in EventPMMessageBotLLM; CreateOption functional options; EventPMPipecatcallTerminated backstop handler with 7 result labels; new ai_manager_aicall_backstop_reply_total metric
- bin-dbscheme-manager: Alembic migration adding the two columns and composite index idx_ai_messages_pcc_delivery
EOF
)"
```

(Do **not** merge without explicit user authorization. Per root CLAUDE.md, all PRs MUST use squash merge.)

---

## Notes for the executor

- **Tests first, every time.** No "I'll write the test after." If you skip a test, you skip a row in the validation matrix.
- **One commit per task.** Each commit corresponds to a row in the design's touch-points table.
- **Run `go test -race` after every change** that touches concurrent code paths (Tasks 1.3, 1.6, 1.7, 1.9, 1.10, 1.13). Atomics and channels are easy to get subtly wrong.
- **Don't run Alembic upgrade.** Phase 0's migration file is created but not applied — a human deploys it.
- **Check the design doc** (`2026-04-29-aicall-terminate-drop-design.md`) when you're unsure about edge cases. The risk table in §5.5 is especially useful.
- If a task's test setup references a helper that doesn't exist (`newTestSessionWithFlushArmed`, `metricRegistered`, etc.), introduce it as a first-class testing helper in a `_test_helpers.go` file (build-tag `//go:build test_helpers`) or simply `*_test.go`. Don't pollute production code.
