# AIcall terminate-drop & no-final-event recovery — design

**Date:** 2026-04-29
**Status:** Draft
**Owner:** pchero
**Related incident:** aicall `93d1eeac-ae0c-42ba-93e5-0d977d6f532f` stopped responding to a LINE conversation on 2026-04-29 ~01:52 UTC

## 1. Problem, Goals, Scope

### 1.1 Problem

A conversation-type AIcall can lose a user-bound LLM reply when:

1. Pipecat does not emit `BotLLMStopped` for a generation (observed: `gemini-2.5-flash` stream stalled mid-reply after a tool-result re-injection), AND
2. ai-manager's 30 s `TerminateWithDelay` (`bin-ai-manager/pkg/aicallhandler/send.go:106`) fires while the flush goroutine is still accumulating tokens.

`SessionStop` (`bin-pipecat-manager/pkg/pipecatcallhandler/session.go:82`) does **not** cancel `pc.Ctx`, so the only fallback that publishes a partial `message_bot_llm` (`runLLMIntermediateFlush` → `case <-se.Ctx.Done()` at `runner.go:638-657`) never runs. The user gets total silence.

A second, related defect surfaced during review: `EventPMMessageBotLLM` (`bin-ai-manager/pkg/messagehandler/event.go:35-131`) persists the assistant row **between** guard #1 (pre-persist) and guard #2 (post-persist re-check). When guard #2 fails (a newer pipecatcall has registered), the row stays in `ai_messages` but no conversation message is sent. A naive `AssistantReplyExists` check that only inspects row existence would falsely conclude "delivered" and suppress the backstop. Fix: track a per-row `delivery_status`.

### 1.2 Reproducer (live evidence from the incident)

| Time (UTC) | Event |
|------------|-------|
| `01:52:11.6` | New pipecatcall `9d757f46…` created. ai-manager schedules `TerminateWithDelay(30000ms)`. |
| `01:52:14.2` | First `bot-llm-started` (pre-tool). |
| `01:52:15.27` | Gemini emits a function call → `BotLLMStopped` with **0 tokens**. |
| `01:52:15.95` | RAG (`search_knowledge`) returns 5 sources. |
| `01:52:16.16` | Second `bot-llm-started` (post-tool). Flush goroutine starts. |
| `01:52:16.60` | First sentence aggregated for TTS. |
| `01:52:16.76` | Intermediate flush sequence:1, delta_len:100. |
| `01:52:16.76 → 01:52:42.19` | **24 s of silence — no further `bot-output`, no `BotLLMStopped`.** |
| `01:52:42.19` | 30 s `TerminateWithDelay` fires → `terminate()` → `SessionStop()`. |
| `01:53:04.16` | Python side notices WebSocket close. |

No final `message_bot_llm` event ever published → ai-manager sent nothing to the LINE conversation.

### 1.3 Goals

1. **No silent drops.** Every terminated pipecatcall that had any LLM tokens must publish a final `message_bot_llm` event with the accumulated text.
2. **No silent stalls.** A stuck stream (no tokens for N s) must be forcibly finalized inside pipecat-manager rather than relying solely on the upstream terminate timer.
3. **No silent users.** If both 1 and 2 still fail to deliver a reply, ai-manager must send a fallback message to the conversation. The check must distinguish *persisted* from *delivered* so guard-#2 race losers don't suppress the backstop.
4. **Observability.** Every exit path of `runLLMIntermediateFlush` and every backstop firing must be counted in Prometheus and logged at INFO+.

### 1.4 Non-goals

- Fixing the upstream Gemini stream hang itself (out of our control).
- Changing the 30 s `TerminateWithDelay` value or making it adaptive — separate concern.
- Voice (Asterisk-bound) aicalls — pipecat-manager fixes (C0–C4) apply uniformly, but the user-facing fallback (Goal 3, C5–C7) only fires for conversation/text aicalls. Voice path correctness is verified in §3.8.

### 1.5 Affected services

- `bin-pipecat-manager` — Goals 1, 2, 4. Adds a new event type.
- `bin-ai-manager` — Goal 3, plus Goal 4 metrics. Subscribes to the new event; persists `pipecatcall_id` and `delivery_status` on assistant rows.
- `bin-dbscheme-manager` — Alembic migration to add `pipecatcall_id` and `delivery_status` columns to `ai_messages`.

## 2. Architecture & changes per service

### 2.1 Change map

```
bin-dbscheme-manager
└── alembic/versions/<rev>_add_pipecatcall_id_delivery_status_to_ai_messages.py    [ADD]
    + ai_messages.pipecatcall_id    BINARY(16)         NULL,        indexed
    + ai_messages.delivery_status   VARCHAR(16)        NOT NULL DEFAULT 'delivered'
    + composite index (pipecatcall_id, delivery_status)

bin-pipecat-manager
├── models/pipecatcall/event.go               [MODIFY]  + EventTypePipecatcallTerminated (string)
├── models/pipecatcall/session.go             [MODIFY]  + LLMFlushOnce sync.Once;
│                                                       + LLMStopReason atomic.Int32;
│                                                       LLMFlushing → atomic.Bool
├── pkg/pipecatcallhandler/session.go         [MODIFY]  SessionStop calls se.Cancel()
├── pkg/pipecatcallhandler/start.go           [MODIFY]  terminate() → flushAndFinalize, publish terminated, SessionStop;
│                                                       per-pipecatcall publish-once dedupe (in-memory map)
├── pkg/pipecatcallhandler/runner.go          [MODIFY]  C2 helper; C3 watchdog; H-4/L-1 ctx fixes;
│                                                       defer-reset LLMFlushing on goroutine exit
├── pkg/pipecatcallhandler/metrics.go         [MODIFY]  + 3 new prom metrics
└── (tests for each)                          [ADD]

bin-ai-manager
├── models/message/main.go                    [MODIFY]  + Message.PipecatcallID, Message.DeliveryStatus
│                                                       (both omitted from WebhookMessage — internal only)
├── pkg/dbhandler/message.go                  [MODIFY]  + AssistantReplyExists; UpdateDeliveryStatus
├── pkg/messagehandler/main.go                [MODIFY]  + CreateOptions (functional options) for new fields
├── pkg/messagehandler/event.go               [MODIFY]  EventPMMessageBotLLM persists 'pending' →
│                                                       on successful send updates to 'delivered'
├── pkg/messagehandler/event_pm.go            [MODIFY]  + EventPMPipecatcallTerminated handler
├── pkg/subscribehandler/pipecat_pipecatcall.go [MODIFY] route new event to handler
├── pkg/aicallhandler/metrics.go              [MODIFY]  + ai_manager_aicall_backstop_reply_total{result}
└── (tests for each)                          [ADD]
```

### 2.2 Component responsibilities

| # | Component | Service | Goal |
|---|-----------|---------|------|
| **C0** | Publish `pipecatcall_terminated` event from `terminate()` | pipecat-manager | 3 (prereq) |
| C1 | `SessionStop` cancels `pc.Ctx` | pipecat-manager | 1 |
| C2 | `flushAndFinalize` synchronous flush before SessionStop | pipecat-manager | 1 |
| C3 | Idle watchdog inside `runLLMIntermediateFlush` | pipecat-manager | 2 |
| C4 | Structured exit-reason logs + Prom counters | pipecat-manager | 4 |
| **C4a** | `pipecatcall_id` + `delivery_status` columns; persist+update flow | dbscheme + ai-manager | 3 (prereq, scoping, delivery truth) |
| C5 | `AssistantReplyExists` query (filters `delivery_status='delivered'`) | ai-manager | 3 |
| C6 | `EventPMPipecatcallTerminated` backstop handler | ai-manager | 3 |
| C7 | Backstop counter metric | ai-manager | 4 |

### 2.3 Event flow (happy → degraded → worst-case)

```
Happy path
──────────
Pipecat → BotLLMText (N tokens) → BotLLMStopped
   pipecat-manager: flush goroutine → publishFinalBotLLMEvent (msg_bot_llm)
   ai-manager:     EventPMMessageBotLLM
                     guard#1 ✓ → persist row(pipecatcall_id, delivery_status='pending')
                     guard#2 ✓ → ConversationV1MessageSend ✓
                                → UPDATE delivery_status='delivered'
   pipecat-manager: terminate() → C2 noop_already_done → publishTerminated → SessionStop
   ai-manager:     EventPMPipecatcallTerminated → AssistantReplyExists(delivered)=true → skipped_seen


Degraded path (LLM stalls; current bug)
───────────────────────────────────────
Pipecat → BotLLMText (some tokens) → ⊘ no BotLLMStopped
   C3 watchdog: idleWatchdogTimeout reached after first token
                → CAS StopReason=idle_watchdog → close LLMStopChan
                → flush goroutine drains, publishFinalBotLLMEvent (background ctx, partial)
   ai-manager:    EventPMMessageBotLLM → persist 'pending' → guard#2 ✓ → send ✓ → 'delivered'
   pipecat-manager: TerminateWithDelay → terminate() → C2 noop_already_done → publishTerminated
   ai-manager:    EventPMPipecatcallTerminated → delivered=true → skipped_seen


Worst-case (watchdog also missed; terminate fires)
──────────────────────────────────────────────────
ai-manager TerminateWithDelay → pipecat-manager terminate()
   C2 flushAndFinalize: CAS StopReason=terminate_force → close LLMStopChan once
                        wait LLMDoneChan up to flushFinalizeTimeout
                        → flush goroutine drains, publishFinalBotLLMEvent (background ctx, partial)
   C0 publishTerminated (background ctx)
   C1 SessionStop → se.Cancel()
   ai-manager:  EventPMMessageBotLLM → persist 'pending' → guard#2 ✓ → send ✓ → 'delivered'
   ai-manager:  EventPMPipecatcallTerminated → delivered=true → skipped_seen


Guard-#2 race (newer turn arrived during this turn's send)
──────────────────────────────────────────────────────────
   ai-manager EventPMMessageBotLLM (this turn):
                     guard#1 ✓ → persist 'pending'
                     guard#2 ✗ → return WITHOUT updating delivery_status
                                 row stays 'pending' (semantic: not delivered)
   ai-manager EventPMPipecatcallTerminated (this turn):
                     AssistantReplyExists(delivered)=false → C6 backstop fires
   (the user gets the backstop for this turn; the newer turn delivers separately)


Total-failure path (nothing ever streamed; flush goroutine never started)
──────────────────────────────────────────────────────────────────────────
   pipecat-manager: terminate() → C2 noop_never_started → publishTerminated → SessionStop
   ai-manager:    EventPMPipecatcallTerminated → AssistantReplyExists=false
                  → C6 backstop: persist 'delivered' row, send fallback reply
                  → C7 metric: …backstop_reply_total{result="sent"}++
```

### 2.4 Why both C1 and C2

C1 alone would work via the `Ctx.Done` branch in the flush goroutine, but propagation is asynchronous: `SessionStop` returns and starts tearing down the Python runner before the flush goroutine has actually published. C2 makes it deterministic by closing `LLMStopChan` and waiting on `LLMDoneChan` synchronously, with a bounded timeout. C1 is kept as a safety net for any future code path that skips `flushAndFinalize`.

The `terminate()` ordering is **fixed** so C2 always runs before C1's cancel propagates:

```
terminate()
  ├── flushAndFinalize(se)        ← C2 (closes Stop chan, waits Done chan)
  ├── publishTerminated(pc)       ← C0 (after flush so any final msg_bot_llm precedes the terminated event)
  └── SessionStop(pc.ID)          ← C1 (cancels pc.Ctx, cleanup)
```

## 3. Pipecat-manager components (C0–C4)

### 3.1 C0 — Publish `pipecatcall_terminated` event

Code search confirmed only `EventTypeInitialized` exists today (`bin-pipecat-manager/models/pipecatcall/event.go`). Constants are declared as `string` (matching `EventTypeCreated`/`EventTypeDeleted`); we follow the same convention:

```go
// bin-pipecat-manager/models/pipecatcall/event.go
const (
    EventTypeCreated     string = "pipecatcall_created"
    EventTypeDeleted     string = "pipecatcall_deleted"
    EventTypeInitialized string = "pipecatcall_initialized"
    EventTypePipecatcallTerminated string = "pipecatcall_terminated"   // NEW
)
```

Publish from `terminate()` after `flushAndFinalize` and before `SessionStop`. Payload = the `pipecatcall.Pipecatcall` struct snapshot (same shape as `EventPMPipecatcallInitialized`). Verified the struct already carries the fields the ai-manager handler needs: `ID`, `ReferenceType`, `ReferenceID`, `ActiveflowID`.

```go
func (h *pipecatcallHandler) terminate(ctx context.Context, pc *pipecatcall.Pipecatcall) {
    se, _ := h.SessionGet(pc.ID)
    if se != nil {
        h.flushAndFinalize(se)            // C2
    }

    if h.markTerminatedOnce(pc.ID) {      // dedupe (see 3.1.1)
        h.notifyHandler.PublishEvent(
            context.Background(),
            pipecatcall.EventTypePipecatcallTerminated,
            pc,
        )
    }

    switch pc.ReferenceType { /* existing */ }

    h.SessionStop(pc.ID)                  // C1 cancel happens inside SessionStop
}
```

#### 3.1.1 Idempotent terminated-event publish (in-memory dedupe)

`terminate()` may be invoked twice if both the per-pod stop RPC and a subsequent reaper fire. The DB-mapped `Pipecatcall` struct does **not** get a new flag (per review H2-1: adding fields with `db:` tags requires migrations and is risky). Instead, dedupe lives in the handler's in-memory map:

```go
// pipecatcallHandler holds:
//   muTerminated         sync.Mutex
//   terminatedPublished  map[uuid.UUID]struct{}

func (h *pipecatcallHandler) markTerminatedOnce(id uuid.UUID) bool {
    h.muTerminated.Lock()
    defer h.muTerminated.Unlock()
    if _, ok := h.terminatedPublished[id]; ok {
        return false
    }
    h.terminatedPublished[id] = struct{}{}
    return true
}
```

Memory is bounded by pod lifetime + concurrent active calls (~hundreds). On pod restart the map is empty, so a duplicate event after restart is possible — but the ai-manager handler is idempotent (C5 short-circuits via `delivery_status='delivered'` rows), so duplicate publishes are non-harmful.

**Map cleanup (iter-3 M3-4):** to avoid long-tail bloat over multi-week pod lifetimes, `SessionStop` removes the entry **after** `terminate()`'s publish has fired:

```go
// In SessionStop, after the existing teardown:
h.muTerminated.Lock()
delete(h.terminatedPublished, pc.ID)
h.muTerminated.Unlock()
```

By the time `SessionStop` runs, the dedupe was already used (or `terminate()` was never called and there's nothing to delete). This caps map size at the count of currently-active pipecatcalls.

#### 3.1.2 WebhookMessage / RST docs

The `pipecatcall_terminated` event is internal — consumed only by ai-manager. It is NOT documented in `bin-api-manager/docsdev/source/` and the payload is NOT exposed via any `WebhookMessage`. This follows the root CLAUDE.md "RST struct docs must match `WebhookMessage`, not internal model structs" rule. RST update not required.

### 3.2 C1 — `SessionStop` cancels `pc.Ctx`

**File:** `bin-pipecat-manager/pkg/pipecatcallhandler/session.go`

`Session.Cancel` is verified to be the same `cancel` func captured at session creation (`session.go:39`). The call site is the one used by `defer se.Cancel()` in `start.go:140,158,167,246,261,283`.

**Change:** in `SessionStop`, call `se.Cancel()` after the existing `wait on ConnAstReady or Ctx.Done` block but **before** `close(ConnAst)`.

Sequencing inside `SessionStop`:

```
wait on ConnAstReady or Ctx.Done       (already there — lets in-flight Asterisk connect race finish)
[NEW] cancel pc.Ctx                    ← C1
close ConnAst (if any)                 (already there)
delete from session map
stop python runner
```

Placing `cancel()` *after* the `ConnAstReady` wait avoids racing with the parallel Asterisk-connect goroutine. If `SessionStop` is invoked while the Asterisk connect is still pending, we wait at most until the connect goroutine signals one channel or the other, then cancel deterministically.

### 3.3 C2 — `flushAndFinalize`

#### 3.3.1 Concurrency contract (revised after iteration-2 review)

The Session struct gains three fields:

- `LLMFlushing  atomic.Bool` (was plain `bool`; reads/writes from both WebSocket and flush goroutines now race-free)
- `LLMFlushOnce sync.Once` (idempotent close of `LLMStopChan`)
- `LLMStopReason atomic.Int32` (attribution: which closer won)

```go
type StopReason int32
const (
    StopReasonUnset          StopReason = iota
    StopReasonNormal                     // BotLLMStopped
    StopReasonIdleWatchdog
    StopReasonTerminateForce
    StopReasonContextCancel
)
```

Closers MUST CAS the reason before invoking `LLMFlushOnce.Do(close)`:

```go
atomic.CompareAndSwapInt32(&se.LLMStopReason,
    int32(StopReasonUnset), int32(StopReasonTerminateForce))
se.LLMFlushOnce.Do(func() { close(se.LLMStopChan) })
```

The flush goroutine reads the reason after `<-LLMStopChan` returns:

```go
case <-se.LLMStopChan:
    reason := StopReason(atomic.LoadInt32(&se.LLMStopReason))
    // drain remaining tokens (existing loop)
    h.publishFinalBotLLMEvent(context.Background(), se, messageID, fullText) // L-1: was se.Ctx
    metricsLLMFlushExit.WithLabelValues(reasonLabel(reason)).Inc()
    return
```

`LLMFlushing` reset (review H-4): the WebSocket goroutine cannot reset `LLMFlushing` reliably because `BotLLMStopped` may never arrive. Move the reset to a `defer` inside the flush goroutine itself:

```go
func (h *pipecatcallHandler) runLLMIntermediateFlush(se *pipecatcall.Session, messageID uuid.UUID) {
    defer close(se.LLMDoneChan)
    defer se.LLMFlushing.Store(false)   // ← reset on every exit, regardless of which path
    ...
}
```

This guarantees `LLMFlushing` reflects "flush goroutine running" as the single source of truth, regardless of who tripped the close. The `BotLLMStopped` handler on the WebSocket goroutine no longer touches `LLMFlushing`; it just CASes the reason and closes the chan once via `Once`.

#### 3.3.2 Helper

```go
// flushAndFinalize is invoked from terminate(). It forces the flush goroutine
// (if any) to publish a final message_bot_llm event with whatever tokens have
// accumulated, then waits for the goroutine to exit (bounded). Idempotent
// w.r.t. concurrent BotLLMStopped or watchdog firing.
func (h *pipecatcallHandler) flushAndFinalize(se *pipecatcall.Session) {
    if !se.LLMFlushing.Load() {
        // Distinguish two cases for observability:
        //   never_started: flush goroutine never armed for this pipecatcall
        //   already_done : flush goroutine ran and exited cleanly
        if se.LLMMessageID == uuid.Nil {
            h.metricsFlushFinalizeOutcome.WithLabelValues("noop_never_started").Inc()
        } else {
            h.metricsFlushFinalizeOutcome.WithLabelValues("noop_already_done").Inc()
        }
        return
    }

    atomic.CompareAndSwapInt32(&se.LLMStopReason,
        int32(StopReasonUnset), int32(StopReasonTerminateForce))
    se.LLMFlushOnce.Do(func() { close(se.LLMStopChan) })

    timer := time.NewTimer(flushFinalizeTimeout)
    defer timer.Stop()

    select {
    case <-se.LLMDoneChan:
        h.metricsFlushFinalizeOutcome.WithLabelValues("done").Inc()
    case <-timer.C:
        // Flush goroutine didn't return in time. We proceed to teardown anyway.
        // C1 (Ctx.Done in SessionStop) will eventually unblock the goroutine.
        h.metricsFlushFinalizeOutcome.WithLabelValues("timeout").Inc()
    }
}
```

`flushFinalizeTimeout = 3 s` (raised from 1.5 s after iteration-1 review).

`metricsFlushFinalizeOutcome` is a separate counter (`pipecat_manager_llm_flush_finalize_outcome_total{outcome=noop_never_started|noop_already_done|done|timeout}`); `noop_*` split per iteration-2 M2-2.

#### 3.3.3 Existing `BotLLMStopped` handler — required edits

`runner.go:548-554` becomes:

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
    atomic.CompareAndSwapInt32(&se.LLMStopReason,
        int32(StopReasonUnset), int32(StopReasonNormal))
    se.LLMFlushOnce.Do(func() { close(se.LLMStopChan) })
    <-se.LLMDoneChan   // wait for goroutine to publish + reset LLMFlushing
```

The `LLMFlushing = false` line is **removed** — the goroutine's `defer` now owns this reset.

#### 3.3.4 Sequence: C2 timeout → C1 cancel → goroutine exits

```
t=0      terminate() called
t=0      C2 flushAndFinalize: CAS reason=terminate_force, close LLMStopChan via Once
t=0      flush goroutine wakes on <-LLMStopChan, drains, calls publishFinalBotLLMEvent
         publishFinalBotLLMEvent uses context.Background() (L-1 fix), so even if Ctx is cancelled later
         the publish proceeds
t=3s     C2 flushFinalizeTimeout fires (publishEvent took >3s — degraded broker)
         metric: flush_finalize_outcome_total{outcome="timeout"}++
         C2 returns; terminate() proceeds.
t=3s+ε   C0 publishTerminated()
t=3s+ε   C1 SessionStop → wait ConnAstReady (no-op for conv aicall) → se.Cancel()
t=3s+ε   flush goroutine still blocked inside notifyHandler.PublishEvent (RabbitMQ slow path)
t=3s+ε   PublishEvent eventually returns; goroutine sets metric exit_total{reason="terminate_force"}
         defer LLMFlushing.Store(false); defer close(LLMDoneChan) fire.
         Goroutine exits cleanly. No leak.
```

If `PublishEvent` hangs forever (broker-down), the goroutine leaks until the publisher's own internal timeout (set in `bin-common-handler/notifyhandler`) fires — which is bounded. We do not add a second timeout; double-bounding makes diagnostics confusing.

### 3.4 C3 — Idle watchdog inside `runLLMIntermediateFlush`

A new ticker tracks "time since last token." Watchdog fires only if `lastToken` is non-zero (first token has arrived) AND no token has arrived for `idleWatchdogTimeout` (review H-1, M-2).

```go
const (
    idleWatchdogTimeout  = 8 * time.Second
    idleWatchdogTickRate = 1 * time.Second
)
```

Sketch (additions in **bold**):

```go
ticker := time.NewTicker(200 * time.Millisecond)
**watchdog := time.NewTicker(idleWatchdogTickRate)**
**defer watchdog.Stop()**
**var lastToken time.Time   // zero until first token**
for {
    select {
    case token := <-se.LLMTokenChan:
        fullText += token
        deltaBuffer += token
        **lastToken = time.Now()**
    case <-ticker.C:
        if deltaBuffer != "" {
            sequence++
            // L-1: use context.Background() for intermediate too,
            //      mirroring the final publish, so terminate-cancel doesn't drop intermediates.
            h.publishIntermediateEventBg(se, messageID, deltaBuffer, sequence)
            deltaBuffer = ""
        }
    **case now := <-watchdog.C:
        if !lastToken.IsZero() && now.Sub(lastToken) >= idleWatchdogTimeout {
            atomic.CompareAndSwapInt32(&se.LLMStopReason,
                int32(StopReasonUnset), int32(StopReasonIdleWatchdog))
            se.LLMFlushOnce.Do(func() { close(se.LLMStopChan) })
        }**
    case <-se.LLMStopChan:
        // existing finalize, exit reason from atomic
    case <-se.Ctx.Done():
        // existing fallback; reason = StopReasonContextCancel set by a CAS at the head of this branch
    }
}
```

**Tool-call boundary:** `LLMFlushing` is reset to `false` by the goroutine's `defer` after each generation completes. The post-tool `BotLLMText` re-enters `runner.go:521`'s init branch (now `if !se.LLMFlushing.Load()`), creates new `LLMStopChan`/`LLMDoneChan`/`LLMStopReason` and launches a new flush goroutine. New goroutine starts with `lastToken == 0`, so its watchdog is correctly armed only after its own first token. No false fire across the tool boundary.

### 3.5 Bug fixes lifted from review

- **H-4** (iter 1): `publishFinalBotLLMEvent` in the `LLMStopChan` branch unconditionally uses `context.Background()` (was `se.Ctx`).
- **L-1** (iter 1, made HIGH in iter 2): `publishIntermediateEvent` in the ticker branch also uses `context.Background()` so the goroutine doesn't lose intermediates if `se.Ctx` is cancelled mid-tick. Implemented via a new `publishIntermediateEventBg` wrapper that calls into the existing helper with a background context.
- The `<-se.Ctx.Done()` branch CASes `StopReason=StopReasonContextCancel` before publishing the partial final, so the metric label correctly attributes the exit.

### 3.6 C4 — Observability (logs + counters)

```go
metricsLLMFlushExit = prometheus.NewCounterVec(prometheus.CounterOpts{
    Name: "pipecat_manager_llm_flush_exit_total",
    Help: "Counter of runLLMIntermediateFlush goroutine exits, by reason.",
}, []string{"reason"})
// labels: stopped_normal | idle_watchdog | terminate_force | context_cancelled

metricsIdleWatchdogFired = prometheus.NewCounter(prometheus.CounterOpts{
    Name: "pipecat_manager_llm_idle_watchdog_fired_total",
    Help: "Counter of idle-watchdog firings (no tokens for idleWatchdogTimeout while flushing).",
})

metricsFlushFinalizeOutcome = prometheus.NewCounterVec(prometheus.CounterOpts{
    Name: "pipecat_manager_llm_flush_finalize_outcome_total",
    Help: "Counter of flushAndFinalize outcomes from the terminate caller's perspective.",
}, []string{"outcome"})
// labels: noop_never_started | noop_already_done | done | timeout
```

Each goroutine exit logs at INFO with `pipecatcall_id`, `message_id`, `full_text_len`, `sequence_count`, `reason`.

### 3.7 Touch points (pipecat-manager)

| File | Change |
|------|--------|
| `models/pipecatcall/event.go` | + `EventTypePipecatcallTerminated` (string) |
| `models/pipecatcall/session.go` | `LLMFlushing` → `atomic.Bool`; + `LLMFlushOnce`, `LLMStopReason` |
| `pkg/pipecatcallhandler/main.go` | + `terminatedPublished` map + mutex |
| `pkg/pipecatcallhandler/session.go` | C1: `Cancel()` in `SessionStop` after ConnAstReady wait |
| `pkg/pipecatcallhandler/start.go` | C0 + C2 wiring in `terminate()`; `markTerminatedOnce` helper |
| `pkg/pipecatcallhandler/runner.go` | C2 helper; C3 watchdog (first-token guard); H-4/L-1 ctx fixes; defer-reset `LLMFlushing`; switch all closers to atomic CAS + Once |
| `pkg/pipecatcallhandler/metrics.go` | 3 new metrics |
| Tests | exit-reason matrix, watchdog before/after first token, tool-call boundary, ConnAstReady race, terminated-event publish dedupe, C2 timeout-then-Ctx.Done sequence |

#### 3.7.1 Test migrations (compile-break list)

Switching `LLMFlushing` from plain `bool` to `atomic.Bool` is a compile break for any test that builds a `Session` with a struct literal `LLMFlushing: true` or assigns `se.LLMFlushing = false`. Each site below MUST be migrated to `Load()`/`Store(true|false)`:

- `bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go` — sites at lines ~294, ~318, ~325, ~475 (per code search). Replace literal initializers with post-construction `se.LLMFlushing.Store(true)` and assertions with `se.LLMFlushing.Load()`.
- Any helper / fixture in `pkg/pipecatcallhandler/*_test.go` that constructs a `Session` should be audited via `grep -nR 'LLMFlushing' bin-pipecat-manager/`.

This is a mechanical change but easy to miss; the `go test ./...` step in the verification workflow will catch it, and the audit is listed here as a deliverable.

### 3.8 Voice-path correctness (iter-2 H2-5)

C0–C4 ship to voice (`ReferenceTypeCall`) pipecatcalls too. Correctness review per component:

- **C0** publishes `pipecatcall_terminated` for voice as well. ai-manager's C6 handler returns `skipped_voice` for non-conversation aicalls — no user-visible side effect.
- **C1** `Cancel()` in `SessionStop` for voice respects the `ConnAstReady` wait, so an in-flight Asterisk connect isn't aborted before completion.
- **C2** `flushAndFinalize` runs on voice too. `flushFinalizeTimeout = 3 s` is well within voice termination latency budgets (Asterisk hangup finalization is typically <1 s for the WS close, plus DB cleanup). Even a 3 s tail-end delay during terminate is invisible to the voice user, who has already hung up.
- **C3** watchdog for voice: same logic. If a voice LLM stalls, we get a partial final and the conversation history is preserved (visible in summaries / transcripts), even though the user heard nothing. This is strictly better than today's silent drop.
- **C4** metrics span both reference types; existing dashboards inherit voice traffic without changes.

No voice-specific test is added in v1, but the existing pipecatcallhandler tests cover both reference types.

## 4. AI-manager components (C4a–C7)

### 4.1 C4a — DB schema and persist+update flow

#### 4.1.1 Migration (in `bin-dbscheme-manager`)

```python
# alembic/versions/<rev>_add_pipecatcall_id_delivery_status_to_ai_messages.py
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

Per `bin-dbscheme-manager/CLAUDE.md`, this migration is **created by AI** (`alembic revision`) but **never run by AI** — a human applies it.

**Default-value rationale:** existing rows are reasonable to call `delivered` because they were created by code paths that always followed persist-then-send (and we have no row-level evidence to the contrary). The default lets pre-existing inserts continue to work with `NULL pipecatcall_id` and `delivered` status — both are correctly ignored by `AssistantReplyExists` (which requires `pipecatcall_id = ?`).

#### 4.1.2 Phase-coordination safety (iter-2 M2-3)

Phase 0 (this migration) deploys before any Go code references the new columns. Until Phase 2 ships:

- Inserts from old code omit both columns. MySQL fills `pipecatcall_id NULL` and `delivery_status='delivered'` (default).
- Reads from old code do not select the new columns. Schema is forward-compatible.
- `AssistantReplyExists` (Phase 2 only) requires `pipecatcall_id = ?` — old NULL rows never match, so no false positives.

#### 4.1.3 Model & WebhookMessage (iter-2 H2-2 — explicit decision)

`bin-ai-manager/models/message/main.go`:

- `Message.PipecatcallID  uuid.UUID` — **internal correlation only**. Omit from `WebhookMessage.ConvertWebhookMessage()`.
- `Message.DeliveryStatus DeliveryStatus` (enum: `pending|delivered`) — **internal**. Omit from `WebhookMessage`.

This follows the root CLAUDE.md "RST struct docs must match `WebhookMessage`, not internal model structs" rule. The RST `*_struct_*.rst` files do **not** require updates.

#### 4.1.4 `messageHandler.Create` — functional options (iter-2 H2-3)

Adding a 10th positional parameter to `Create()` would break 8+ callers (`engine_openai_handler`, `engine_dialogflow_handler`, `pkg/aicallhandler/*`, `tool.go`, etc.). Switch to functional options to keep the existing shape and make new fields additive:

```go
// Create signature (unchanged for existing callers)
Create(
    ctx context.Context,
    id, customerID, aicallID, activeflowID uuid.UUID,
    direction message.Direction,
    role message.Role,
    content string,
    toolCalls []message.ToolCall,
    toolCallID string,
    opts ...CreateOption,    // NEW
) (*message.Message, error)

// Options
type CreateOption func(*createParams)
type createParams struct {
    pipecatcallID  uuid.UUID
    deliveryStatus message.DeliveryStatus
}
func WithPipecatcallID(id uuid.UUID) CreateOption  { ... }
func WithDeliveryStatus(s message.DeliveryStatus) CreateOption { ... }
```

Existing callers compile unchanged. New callers (`EventPMMessageBotLLM` and `EventPMPipecatcallTerminated`) opt in:

```go
h.Create(ctx, evt.ID, ..., toolCalls, toolCallID,
    messagehandler.WithPipecatcallID(evt.PipecatcallID),
    messagehandler.WithDeliveryStatus(message.DeliveryStatusPending))
```

This pattern matches `~/.claude/rules/golang/patterns.md`'s recommended functional-options idiom.

#### 4.1.5 `EventPMMessageBotLLM` — persist+update flow (iter-2 C2-1)

The handler now persists with `delivery_status='pending'` and updates to `'delivered'` only after `ConversationV1MessageSend` succeeds. The two-guard pattern is preserved exactly as today, but the **completion semantic** is encoded in `delivery_status` rather than mere row existence.

```go
// Guard #1 (unchanged) — drop stale BEFORE any DB write
if ac.PipecatcallID != evt.PipecatcallID { ... return }

// Persist with delivery_status='pending' (NEW)
tmp, err := h.Create(ctx, evt.ID, ..., evt.Text, nil, "",
    messagehandler.WithPipecatcallID(evt.PipecatcallID),
    messagehandler.WithDeliveryStatus(message.DeliveryStatusPending))
if err != nil { ... return }

// Guard #2 (unchanged) — re-check; if fails, row stays 'pending' → backstop will fire
if acFinal.PipecatcallID != evt.PipecatcallID { ... return }

// Send to conversation
sent, errSend := h.reqHandler.ConversationV1MessageSend(...)
if errSend != nil { ... return }   // row stays 'pending' → backstop will fire

// Mark delivered (NEW)
// Iter-3 L3-1: retry once with short backoff before logging.
// The conversation send already succeeded; we want the row to reflect that
// to prevent the backstop from firing a duplicate user-visible reply.
errUpd := h.dbHandler.MessageUpdateDeliveryStatus(ctx, tmp.ID, message.DeliveryStatusDelivered)
if errUpd != nil {
    time.Sleep(deliveryStatusUpdateRetryDelay) // 100 ms
    errUpd = h.dbHandler.MessageUpdateDeliveryStatus(ctx, tmp.ID, message.DeliveryStatusDelivered)
}
if errUpd != nil {
    log.Errorf("Could not mark message delivered after retry. msg_id: %s err: %v", tmp.ID, errUpd)
    promConversationDeliveryStatusUpdateFailedTotal.Inc()
    // Worst case: backstop additionally fires → user sees the real reply + fallback.
    // Acceptable rare degradation; the metric pages on alarm.
}
```

The two existing metrics (`promConversationStaleResponseDroppedTotal`, `promConversationReplySendTotal`) are preserved unchanged. One new metric `promConversationDeliveryStatusUpdateFailedTotal` is added to flag the rare case where the send succeeded but the status update failed (the user got their reply but the row says 'pending'). This would manifest as a backstop firing in addition to the real reply — a known small risk we monitor rather than complicate the code further to prevent.

### 4.2 C5 — `AssistantReplyExists`

```sql
SELECT 1 FROM ai_messages
 WHERE pipecatcall_id  = ?
   AND delivery_status = 'delivered'
   AND direction       = 'incoming'
   AND role            = 'assistant'
   AND tm_delete IS NULL
 LIMIT 1
```

The query uses the composite index `idx_ai_messages_pcc_delivery`. Sub-millisecond expected.

`delivery_status='delivered'` is the canonical signal. A row that is `'pending'` (guard-#2 failure or send failure) does NOT count as delivered, so the backstop correctly fires.

**Note (iter-3 M3-1):** `EventPMMessageBotLLMIntermediate` does NOT persist rows in `ai_messages` — it only publishes a webhook event. So intermediate streaming activity cannot accidentally satisfy `AssistantReplyExists`. If a future refactor moves intermediates to persistent storage, the `delivery_status='delivered'` filter still excludes them as long as intermediates are persisted with `delivery_status='pending'` (or a new `'intermediate'` status).

### 4.3 C6 — `EventPMPipecatcallTerminated` backstop handler

```go
func (h *messageHandler) EventPMPipecatcallTerminated(ctx context.Context, ev *pmpipecatcall.Pipecatcall) error {
    log := logrus.WithFields(...)

    if ev.ReferenceType != pmpipecatcall.ReferenceTypeAICall {
        promBackstopReplyTotal.WithLabelValues("skipped_not_aicall").Inc()
        return nil
    }

    aicall, err := h.reqHandler.AIV1AIcallGet(ctx, ev.ReferenceID)
    if err != nil {
        // unknown aicall — log+swallow; not a backstop case
        log.Errorf("Could not get aicall for terminated pipecatcall. err: %v", err)
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

    // Race grace: let any in-flight EventPMMessageBotLLM land on the other pod.
    timer := time.NewTimer(backstopGraceDelay)
    select {
    case <-timer.C:
    case <-ctx.Done():
        timer.Stop()
        return ctx.Err()
    }

    seen, err := h.dbHandler.MessageAssistantReplyExists(ctx, ev.ID)
    if err != nil {
        return err
    }
    if seen {
        promBackstopReplyTotal.WithLabelValues("skipped_seen").Inc()
        return nil
    }

    // Backstop send: persist 'delivered' immediately (we ARE the delivery), then send.
    msg, err := h.Create(ctx, uuid.Nil, aicall.CustomerID, aicall.ID, aicall.ActiveflowID,
        message.DirectionIncoming, message.RoleAssistant, backstopReplyText, nil, "",
        messagehandler.WithPipecatcallID(ev.ID),
        messagehandler.WithDeliveryStatus(message.DeliveryStatusDelivered))
    if err != nil {
        promBackstopReplyTotal.WithLabelValues("failed").Inc()
        return errors.Wrap(err, "could not persist backstop message")
    }

    if _, errSend := h.reqHandler.ConversationV1MessageSend(ctx, aicall.ReferenceID,
        backstopReplyText, []cvmedia.Media{}); errSend != nil {
        // Row exists with 'delivered' (a slight lie under failure, but it makes replays idempotent —
        // we never want a duplicate fallback message). Log+metric.
        promBackstopReplyTotal.WithLabelValues("send_failed").Inc()
        return errors.Wrap(errSend, "could not send backstop conversation reply")
    }

    promBackstopReplyTotal.WithLabelValues("sent").Inc()
    log.WithField("message_id", msg.ID).Infof(
        "Backstop reply sent. aicall_id: %s, pipecatcall_id: %s",
        aicall.ID, ev.ID)
    return nil
}
```

**Constants:**

```go
backstopGraceDelay = 3 * time.Second
backstopReplyText  = "Sorry, I'm having trouble responding right now. Please try again."
```

**Idempotency under at-least-once delivery:** the backstop's `Create` call uses `delivery_status='delivered'`. A duplicate event 30 s later finds the row → `skipped_seen`. Even if the send failed, the row is marked delivered to prevent duplicate user-visible fallbacks (review tradeoff — better to silently swallow than to spam).

### 4.4 C7 — Backstop counter metric

```go
promBackstopReplyTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
    Name: "ai_manager_aicall_backstop_reply_total",
    Help: "Counter of pipecatcall_terminated backstop attempts in messagehandler.EventPMPipecatcallTerminated.",
}, []string{"result"})
```

**Label values (closed set):**

- `sent` — backstop reply persisted and dispatched
- `failed` — DB insert error
- `send_failed` — DB OK, conversation send error (row marked 'delivered' to suppress retries)
- `skipped_seen` — final reply already 'delivered' in DB
- `skipped_voice` — non-conversation aicall
- `skipped_terminated` — aicall already terminated
- `skipped_not_aicall` — pipecatcall reference_type != ai_call

### 4.5 Dashboard / alert hooks

- **Headline ratio:** `rate(ai_manager_aicall_backstop_reply_total{result="sent"}[5m]) / rate(ai_manager_aicall_backstop_reply_total[5m])`. Sustained >5% means upstream isn't recovering reliably.
- **Pair with C4 metric:** `pipecat_manager_llm_idle_watchdog_fired_total` should rise *before* `…backstop_reply_total{result="sent"}` does in any incident.
- **New companion:** `ai_manager_message_delivery_status_update_failed_total` — flags the rare "sent but row stayed pending" case.

### 4.6 Touch points (ai-manager + dbscheme)

| File | Change |
|------|--------|
| `bin-dbscheme-manager/alembic/versions/<rev>_*.py` | + `pipecatcall_id` and `delivery_status` columns, composite index |
| `bin-ai-manager/models/message/main.go` | + `PipecatcallID`, `DeliveryStatus`; both omitted from `WebhookMessage` |
| `bin-ai-manager/pkg/dbhandler/message.go` | populate + read new columns; `MessageAssistantReplyExists`, `MessageUpdateDeliveryStatus` |
| `bin-ai-manager/pkg/messagehandler/main.go` | + `CreateOption` functional options; **add `EventPMPipecatcallTerminated(ctx, *pmpipecatcall.Pipecatcall) error` to `MessageHandler` interface** so mockgen regenerates (iter-3 M3-3); doc the persist-then-update invariant |
| `bin-ai-manager/pkg/messagehandler/event.go` | persist 'pending' → conditional update to 'delivered'; new failure metric |
| `bin-ai-manager/pkg/messagehandler/event_pm.go` | + `EventPMPipecatcallTerminated` |
| `bin-ai-manager/pkg/subscribehandler/pipecat_pipecatcall.go` | + `processEventPMPipecatcallTerminated(ctx, m *sock.Event) error` adapter that unmarshals the event payload and calls `MessageHandler.EventPMPipecatcallTerminated` |
| `bin-ai-manager/pkg/subscribehandler/main.go` | one-line dispatch case in the `switch (publisher, type)` block at the existing site (~lines 167-186) |
| `bin-ai-manager/pkg/aicallhandler/metrics.go` | + `promBackstopReplyTotal` |
| Tests | per-result-label, grace-delay, persist+update interplay, guard-#2-failure path |

## 5. Testing, rollout, risks

### 5.1 Unit tests

| # | What to verify | Test technique |
|---|---|---|
| C0 | `terminate()` publishes once; duplicate `terminate()` does NOT republish | Mock notify; assert publish count |
| C1 | `SessionStop` causes `<-pc.Ctx.Done()` to fire within 50 ms; does not race with pending Asterisk-connect | Two tests: post-connect cancel; connect-pending cancel asserting no leak |
| C2 | Outcomes: `noop_never_started`, `noop_already_done`, `done`, `timeout` | Table-driven with controlled flush goroutine timing |
| C2 | Idempotent under concurrent `flushAndFinalize` | Run two concurrent calls; assert single attribution wins |
| C3 | Watchdog fires only after first token; fires once; correct atomic reason | Inject clock; assert no fire pre-token |
| C3 | Tool-call boundary: gen 1 ends, gen 2 re-arms watchdog correctly | Sequence test |
| C4 | Each goroutine exit emits exactly one counter increment with the correct label | Counter assertions |
| C4 | Multiple closers race → loser's reason CAS is dropped; metric reflects winner | Two goroutines + atomic check |
| C4a | `pipecatcall_id` populated; `delivery_status` defaults to `pending` via opt; `delivered` via opt | dbhandler test |
| C4a | Migration upgrade/downgrade is idempotent on re-run | Alembic test |
| C5 | `AssistantReplyExists` matches only `delivery_status='delivered'`, scoped by `pipecatcall_id` | Synthetic rows: pending+delivered+other-pcc; assert correct |
| EventBotLLM | Persist-then-update: happy path → `delivered`; guard-#2 fail → stays `pending`; send fail → stays `pending`; status-update fail → metric ticks but conv send still succeeded | Mock branches |
| C6 | All 7 result branches | gomock per branch |
| C6 | Grace delay honored; `NewTimer.Stop()` called on ctx cancel (no leak) | Inject clock; race ctx cancel |
| C6 | Idempotency: replay → `skipped_seen` | Two sequential calls; second sees the persisted row |
| C7 | Counter exists; increments per branch | Asserted in C6 |

**Race detector:** all touched packages must pass `go test -race ./...`.

### 5.2 Negative cases (explicit)

- `BotLLMStopped` arrives between `flushAndFinalize`'s close and the wait → flush goroutine drains and exits via `LLMDoneChan` close; no double-close panic.
- Watchdog and `BotLLMStopped` race on `LLMFlushOnce` → only one close; atomic CAS attributes to whoever won.
- `EventPMPipecatcallTerminated` arrives before `EventPMMessageBotLLM` (cross-pod race) → grace covers it; `skipped_seen` after grace.
- `terminate()` called twice → second call's `flushAndFinalize` returns `noop_already_done`; second `pipecatcall_terminated` event suppressed by `markTerminatedOnce`.
- Status-update RPC fails after a successful send → metric `delivery_status_update_failed_total` ticks; row stays `pending` → backstop will additionally fire (rare; user receives reply + fallback). Prefer this over a duplicate-suppress complication.

### 5.3 Manual / integration

- Reproduce the watchdog: feed tokens then drop the WebSocket without `BotLLMStopped`. Confirm watchdog fires within ~8 s and conversation gets the partial reply with `delivery_status='delivered'`.
- Reproduce the guard-#2 race: in pre-prod, force two rapid user messages so the second registers a new pipecatcall before the first's `EventPMMessageBotLLM` fires guard #2. Confirm the first turn's row stays `pending` and the backstop fires for the first turn.
- Reproduce the backstop **end-to-end**: kill the Python runner during a real-traffic pipecatcall before any token. Confirm phase-2's wiring receives `pipecatcall_terminated` and fires the backstop exactly once.

### 5.4 Rollout

**Order matters.**

1. **Phase 0 — Migration.** Apply the C4a Alembic migration in `bin-dbscheme-manager`. Reversible. Bake **2 h**.
2. **Phase 1 — pipecat-manager (C0–C4).** Backwards-compatible. Publishes new event with no consumers yet. Bake **24 h**.
3. **Phase 1 success gates:**
   - `pipecat_manager_llm_flush_exit_total{reason="stopped_normal"}` >95% of total.
   - `pipecat_manager_llm_idle_watchdog_fired_total` non-zero and stable.
   - `pipecat_manager_llm_flush_finalize_outcome_total{outcome="timeout"}` ≈ 0.
   - The new event visible at the broker.
4. **Phase 2 — ai-manager (C4a model + persist+update + C5/C6/C7).** Pre-deploy: synthetic terminated-event test. Bake **24 h**.
5. **Phase 2 success gates:**
   - `ai_manager_aicall_backstop_reply_total{result="sent"}` is single-digit per day.
   - `{result="skipped_seen"}` dominates total.
   - `{result="failed"}`, `{result="send_failed"}` ≈ 0.
   - `ai_manager_message_delivery_status_update_failed_total` ≈ 0.

**Tunable defaults (all package-level `const`):**

- `idleWatchdogTimeout = 8 s`, `idleWatchdogTickRate = 1 s`
- `flushFinalizeTimeout = 3 s`
- `backstopGraceDelay = 3 s`
- `deliveryStatusUpdateRetryDelay = 100 ms` (one retry on the post-send DB update; see §4.1.5)

**Rollback:** each phase reverts independently. Phase 0's column remains harmless after a code revert (nullable + default).

### 5.5 Risks & mitigations

| Risk | Mitigation |
|------|------------|
| C1 cancels `pc.Ctx` too early | Fixed terminate ordering: flush → publish → SessionStop. Publishes use `context.Background()`. |
| C3 cuts off slow stream | First-token guard. Watchdog tracks inter-token gap, not total duration. |
| Backstop spam under event redelivery | First backstop persists `delivered` row; replays take `skipped_seen`. |
| Cross-pod race grace insufficient | Persist-then-update narrows the window. Observable via `{result="sent"}`; bump to 5 s if needed. |
| `terminate()` twice | `flushAndFinalize` returns `noop_already_done`; `markTerminatedOnce` dedupes the publish. |
| Backstop text non-localized | English in v1; locale-keyed text table is a follow-up. |
| DB query latency | Single composite-indexed SELECT; sub-millisecond expected. Monitor `subscribe_event_process_time{type="pipecatcall_terminated"}`. |
| Migration deployed without code | Phased rollout enforces order; default `'delivered'` keeps old rows correct. |
| Goroutine leak on `flushAndFinalize` timeout | C1 cancel + publisher's own timeout bound the goroutine. Counted as `outcome="timeout"`. |
| Status-update succeeds-but-fails-mid-write | One-retry mitigates transient failures; logged; `delivery_status_update_failed_total` metric. Worst case: extra backstop reply. |
| Voice path collateral damage | C0–C4 verified safe for voice in §3.8. C5–C7 hit `skipped_voice`. |

### 5.6 Verification before merge

Per monorepo `CLAUDE.md`:

```
go mod tidy && go mod vendor && go generate ./... && \
  go test ./... && golangci-lint run -v --timeout 5m
```

in **both** `bin-pipecat-manager/` and `bin-ai-manager/`. Plus `go test -race ./pkg/pipecatcallhandler/...` and `go test -race ./pkg/messagehandler/... ./pkg/subscribehandler/...`.

Migration verification: `cd bin-dbscheme-manager && alembic check`. Application is a manual step.
