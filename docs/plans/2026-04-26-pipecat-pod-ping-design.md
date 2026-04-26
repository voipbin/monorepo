# Per-pod liveness ping for `bin-pipecat-manager`

**Date:** 2026-04-26
**Status:** Design (pre-implementation)
**Owner:** ai-platform
**Related:** Conversation-AI text chatbot (`bin-conversation-manager` ↔ `bin-ai-manager` ↔ `bin-pipecat-manager`); independent of and non-blocking for that v1 work.

---

## 1. Problem

`bin-pipecat-manager` and `bin-ai-manager` run on Kubernetes with multiple replicas. Each pipecat session is pinned to a specific pipecat-manager pod via `pipecatcall.HostID`, which is set to that pod's `POD_IP` at session creation (`bin-pipecat-manager/cmd/pipecat-manager/main.go:116-120`). Per-pod RPCs (`PipecatV1MessageSend`, `PipecatV1PipecatcallTerminate`) target a per-pod RabbitMQ queue:

```
bin-manager.pipecat-manager.request.<host_id>
```

When a pipecat-manager pod restarts (rolling deploy, OOM kill, eviction, etc.), the new pod gets a new `POD_IP`. Any `pipecatcall` row whose `HostID` still points to the old IP is now permanently unreachable: the volatile per-pod queue is auto-deleted by RabbitMQ once the consumer disconnects (`pkg/listenhandler/main.go:125`), and the in-memory session state in the original pod is gone.

Today, `bin-ai-manager` cannot tell whether a stored `HostID` is alive without firing a real RPC and waiting for it to time out. The default per-call RPC timeout is 3 seconds (`bin-common-handler/pkg/requesthandler/main.go:130`). On the conversation chat hot path (`pkg/aicallhandler/send.go:46`), this means a single user message stalls **3 seconds** before failing. Once the existing per-target circuit breaker in `bin-common-handler/pkg/circuitbreakerhandler/` records 5 consecutive failures (5 separate user messages in a 30s window — not 5 within one message), it opens for 30 seconds and subsequent calls fast-fail in microseconds. So the customer pain is roughly:
- First user message after a pod restart: **3 s** stall, then error.
- Next ≤4 user messages while CB is closed: **3 s** each, then error.
- After CB opens: 30 seconds of µs-fast-fail; next attempt at T+30s probes again.

This design adds a sub-second decisive liveness preflight on the chat hot path. The goal is to cut the per-call stall from **3 s → 1 s**, and to start that 30-second negative-cache window after far less wasted latency.

## 2. Goals & non-goals

### Goals
- Give `bin-ai-manager` a sub-second decisive liveness signal for any `HostID` before issuing the per-pod chat-message RPC.
- Cut wasted latency on the dead-pod chat path from 3 s → 1 s per call.
- Reuse the existing `circuitbreakerhandler` so dead pods are fast-failed (~µs) for 30 seconds after the breaker trips, with no new CB code.
- Be safe across rolling deploys (mixed old/new pipecat pods).

### Non-goals
- No DB schema change (no new column on `pipecatcall`).
- No background heartbeat goroutine.
- No pipecat-record reaper / cleanup process.
- No new circuit-breaker code or threshold tuning in v1.
- No changes to the Python pipecat side.
- No automatic session recovery (creating a fresh pipecatcall when the old one's pod is dead) — v2 concern.
- **No detection of POD_IP reuse by Calico CNI.** When K8s recycles an old IP for a new pod within minutes, the new pod will respond to the ping with a matching `host_id` echo but won't own the original session. In that edge case, behavior degrades to current behavior (real RPC sent and fails). Detecting this would require either a more durable pod identifier (e.g., `POD_UID`) stored in the `pipecatcall` row — which violates the no-DB-schema-change constraint — or a session-aware ping (out of v1 scope per §9 decisions).

## 3. Constraints (from problem statement)

- Lightweight ping/health RPC, routed via per-pod queue `bin-manager.pipecat-manager.request.<host_id>`.
- Short timeout (target: 1 s).
- Multi-pod safe; no leader election; no shared state beyond what already exists.
- Backwards compatible during rolling deploys.

## 4. Design

### 4.1 Architecture

```
ai-manager                         pipecat-manager (per pod)
─────────                          ────────────────────────
                                   listens on:
                                     bin-manager.pipecat-manager.request          (shared)
                                     bin-manager.pipecat-manager.request.<POD_IP> (volatile, per-pod)

PipecatV1Ping(hostID)
  → r.sendRequest(
       queue   = "...request.<hostID>",
       uri     = "/v1/ping",
       method  = GET,
       timeout = 1s)
  → cb.Allow(queue) ──[open? fast-fail with ErrCircuitOpen]
  → RabbitMQ RPC ─────────────────────►   listenhandler routes /v1/ping
                                            → pipecatcallHandler.Ping(ctx)
                                            → returns {host_id: <POD_IP>, timestamp: now}
  ◄────── response ────────────────────
  → cb.RecordSuccess(queue)
  → verify response.host_id == requested hostID; mismatch → return error
  → return nil

(timeout case)
  → cb.RecordFailure(queue) → return error   // pod dead
                                              // 5 consecutive failures trip CB → 30s fast-fail window
```

The ping uses the **same per-pod queue** as Send/Terminate, so they share the same circuit-breaker target. Failures from any source (ping or real call) move the same breaker.

### 4.2 Why this leverages the existing infrastructure

The `circuitbreakerhandler` in `bin-common-handler/pkg/circuitbreakerhandler/` is wired into every `r.sendRequest()` call (see `send_request.go:50-67`). It provides:

- Per-target (per queue name) breaker state.
- Default failure threshold: **5 consecutive failures** → `Closed → Open` (`option.go:9`).
- Default open duration: **30 s** — `cb.Allow()` rejects immediately during this window (`option.go:10`).
- Half-open probe after 30 s — one allowed request decides whether to close or re-open.
- Free Prometheus metrics: `*_circuitbreaker_state{target=...}`, `*_circuitbreaker_state_transitions_total`, `*_circuitbreaker_rejected_total`.

**Practical implication:** because the ping is just another `sendRequest` call against the per-pod queue, no new state, no new caches, and no new metrics need to be added in v1. The breaker provides the negative-cache effect (30 s of µs-fast-fail per dead `HostID`) for free, and exposes that state to Grafana.

### 4.3 Components

#### `bin-pipecat-manager` (server side)

1. `pkg/pipecatcallhandler/main.go` — extend the `PipecatcallHandler` interface with:
   ```go
   Ping(ctx context.Context) (*pipecatcall.PingResult, error)
   ```
   Returning `error` keeps the door open for forward-compatible states like "drain mode" (responding `503` while the pod is shutting down). v1 always returns `nil`.
2. `pkg/pipecatcallhandler/ping.go` (new) — implementation: `&PingResult{HostID: h.hostID, Timestamp: time.Now().UTC()}, nil`. No DB I/O, no mutex.
3. `models/pipecatcall/ping.go` (new) — `PingResult` struct.
4. `pkg/listenhandler/main.go` — register a new route:
   ```go
   regV1Ping = regexp.MustCompile(`/v1/ping$`)
   ```
   and a switch case for `GET /v1/ping`.
5. `pkg/listenhandler/v1_ping.go` (new) — `processV1PingGet`: calls `h.pipecatcallHandler.Ping(ctx)`, marshals the `PingResult`, returns `200 application/json`. On non-nil error, returns `500` (v1 won't hit this path).
6. Regenerate `pkg/pipecatcallhandler/mock_main.go` via the existing `//go:generate` directive.

#### `bin-common-handler` (client side)

1. `pkg/requesthandler/main.go` — add to the `RequestHandler` interface:
   ```go
   PipecatV1Ping(ctx context.Context, hostID string) error
   ```
2. `pkg/requesthandler/pipecat_ping.go` (new) — uses the same `outline` package alias as `pipecat_message.go`:
   ```go
   package requesthandler

   import (
       "context"
       "encoding/json"
       "fmt"

       outline "monorepo/bin-common-handler/models/outline"
       "monorepo/bin-common-handler/models/sock"
       pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
   )

   const requestTimeoutPipecatPing = 1000  // 1s

   // PipecatV1Ping issues a sub-second liveness probe against the per-pod queue
   // for hostID. Returns nil if the pod responded with a matching host_id (the
   // live case), or an error otherwise (timeout, circuit open, mismatched
   // host_id from a queue-name collision, etc.).
   //
   // IMPORTANT: do not add status-code checks here. A 404 from an old pipecat
   // pod that predates this route is a valid "alive" signal — the pod responded.
   // The only "dead" signal is err != nil, including ctx.DeadlineExceeded and
   // circuitbreakerhandler.ErrCircuitOpen.
   func (r *requestHandler) PipecatV1Ping(ctx context.Context, hostID string) error {
       queueName := fmt.Sprintf("%s.%s", outline.QueueNamePipecatRequest, hostID)
       res, err := r.sendRequest(
           ctx,
           outline.QueueName(queueName),
           "/v1/ping",
           sock.RequestMethodGet,
           "pipecat/ping",
           requestTimeoutPipecatPing,
           0,
           ContentTypeNone,
           nil,
       )
       if err != nil {
           return err
       }

       // Best-effort host_id echo verification: catches the case where the
       // queue is consumed by a pod with a different identity. Note: Calico
       // POD_IP recycle gives a matching IP, so this check does NOT cover
       // that case — see §4.6.
       // For old pods (404) the body is empty → skip the check (the 404
       // itself signals "alive but old code").
       if res != nil && res.StatusCode == 200 && len(res.Data) > 0 {
           var pr pmpipecatcall.PingResult
           if errParse := json.Unmarshal(res.Data, &pr); errParse == nil {
               if pr.HostID != "" && pr.HostID != hostID {
                   return fmt.Errorf("ping host_id mismatch: requested %s, got %s", hostID, pr.HostID)
               }
           }
       }
       return nil
   }
   ```
3. Regenerate `pkg/requesthandler/mock_main.go`.

**Admission-rule note:** Per root CLAUDE.md, packages in `bin-common-handler` should serve 3+ services. `bin-common-handler/pkg/requesthandler/pipecat_*.go` already contains pipecat-specific RPCs (`PipecatV1PipecatcallStart`, `PipecatV1PipecatcallGet`, `PipecatV1PipecatcallTerminate`, `PipecatV1PipecatcallTerminateWithDelay`, `PipecatV1MessageSend`) consumed by a single service (`bin-ai-manager`). Adding `PipecatV1Ping` follows that established pattern. The alternative (a `pipecatpinghandler/` package in `bin-ai-manager` calling `r.SendRequest(...)` directly) would split pipecat client logic across two locations — net negative for clarity. Decision: keep alongside the existing pipecat methods.

#### `bin-ai-manager` (preflight integration)

1. `pkg/aicallhandler/ping.go` (new) — small private helper that distinguishes broker errors from dead-pod errors:
   ```go
   package aicallhandler

   import (
       "context"
       "errors"
       "time"

       "github.com/sirupsen/logrus"

       "monorepo/bin-common-handler/pkg/circuitbreakerhandler"
   )

   // pingPipecatHost runs a 1s preflight against hostID. Returns true if the
   // pod is reachable and owns this host_id; false if the pod is unreachable
   // (timeout) or the breaker is open. Broker/transport errors return false
   // and are logged distinctly so an outage is not misclassified as pod death.
   //
   // Note: relies on errors.Is unwrapping through pkg/errors v0.9.0+ wrappers
   // applied by sendRequest. Verified pkg/errors v0.9.1 is pinned in
   // bin-common-handler/go.mod and bin-ai-manager/go.mod.
   func (h *aicallHandler) pingPipecatHost(ctx context.Context, hostID string) bool {
       log := logrus.WithFields(logrus.Fields{
           "func":    "pingPipecatHost",
           "host_id": hostID,
       })
       if hostID == "" {
           return false
       }
       cctx, cancel := context.WithTimeout(ctx, 1100*time.Millisecond) // small slack over the RPC timeout
       defer cancel()
       err := h.reqHandler.PipecatV1Ping(cctx, hostID)
       if err == nil {
           log.Debug("Pipecat host ping succeeded.")
           return true
       }
       switch {
       case errors.Is(err, circuitbreakerhandler.ErrCircuitOpen):
           log.Debug("Pipecat host ping skipped: circuit breaker open.")
       case errors.Is(err, context.DeadlineExceeded):
           log.Info("Pipecat host ping timed out; treating as dead.")
       default:
           log.Warnf("Pipecat ping failed with unexpected error; skipping per-pod RPC. err: %v", err)
       }
       return false
   }
   ```
2. Edit existing call site — gate the per-pod RPC with a preflight ping at exactly **one** location:
   - `pkg/aicallhandler/send.go:46` — `PipecatV1MessageSend` (the chat hot path).

   On preflight failure: return an error to the caller (`api-manager` / activeflow), do not auto-restart in v1.

3. Regenerate mocks (see §6 for full scope).

**Call sites NOT gated, with justification:**

| Call site | Why not gated |
|---|---|
| `pkg/aicallhandler/process.go:72` `PipecatV1PipecatcallTerminate` | Cleanup path. Code already log-and-continues on RPC error (lines 73-77). Gating saves 2 s on a non-customer-facing path; not worth the extra ping. |
| `pkg/aicallhandler/start.go:210` `PipecatV1PipecatcallTerminateWithDelay` | Delayed RPC. `sendRequest` returns immediately for `delay > 0` (`send_request.go:41-47`); there is no 3 s wait to save. The delayed message is dropped silently if the per-pod queue dies before delivery. |
| `pkg/aicallhandler/send.go:88` `PipecatV1PipecatcallTerminateWithDelay` | Same as above (delayed), and `pc` was just created via `startPipecatcall` on the same pod that just answered — `HostID` is freshly known to be alive. Pinging is pure waste. |

If operational data later shows pain at these sites, gating is a small follow-up — but YAGNI for v1.

### 4.4 Decision rule on the gated site

For `pkg/aicallhandler/send.go:46` (`PipecatV1MessageSend` — the chat hot path): on ping success, proceed with the `PipecatV1MessageSend` RPC. On ping failure, return an error to the caller; do **not** auto-restart in v1.

The message cannot be delivered to the bound session if the pod is dead, so the caller must know the operation failed. Auto-restart with a fresh pipecatcall is left as a v2 enhancement to keep v1 small (~10 lines of preflight, plus the new ping plumbing).

### 4.5 Data flow

#### Happy path: chat message to a live pod

```
api-manager → ai-manager.Send(aicall_id, "hello")
  → SendReferenceTypeCall(c)
    ├─ PipecatV1PipecatcallGet(c.PipecatcallID)        // shared queue, ~few ms
    │   → pc{HostID: "10.4.2.18", ID: <uuid>}
    ├─ messageHandler.Create(...)                       // DB write
    ├─ pingPipecatHost("10.4.2.18")                     // NEW
    │   → cb.Allow → ok
    │   → RPC GET /v1/ping → response in ~few ms
    │   → cb.RecordSuccess → host_id echo matches → returns true
    └─ PipecatV1MessageSend(pc.HostID, pc.ID, ...)      // proceeds
```

Net added latency on the live path: **one RabbitMQ round-trip (~few ms)**.

#### Dead-pod path: `HostID` points to a restarted pod

```
api-manager → ai-manager.Send(aicall_id, "hello")
  → SendReferenceTypeCall(c)
    ├─ PipecatV1PipecatcallGet(...) → pc{HostID: "10.4.2.18 (DEAD)"}
    ├─ messageHandler.Create(...)
    ├─ pingPipecatHost("10.4.2.18")
    │   → cb.Allow → ok (not yet tripped)
    │   → RPC GET /v1/ping → ctx deadline exceeded after 1 s
    │   → cb.RecordFailure → returns false
    └─ return errors.New("pipecat pod for this aicall is no longer reachable")
```

Per-call savings: **3 s → 1 s** on first failure. After **5 consecutive failed pings** (across separate user messages within 30 s) the breaker opens; subsequent calls within the next 30 s short-circuit at `cb.Allow()` in microseconds. At T+30 s the breaker enters half-open and admits one probe ping.

#### Degraded-pod path (slow but alive)

If the pod responds to ping in 950 ms but the real RPC then takes 3 s, the chat path eats **ping (≤1 s) + real RPC (≤3 s) = up to 4 s** vs 3 s today on the unhappy degraded path. This is the worst case the design accepts; degraded-but-not-dead pods are rare in practice (typically either healthy <50 ms or dead = full timeout).

### 4.6 Edge cases

| Case | Behavior | Justification |
|---|---|---|
| Old pipecat pod (no `/v1/ping` route) | `listenhandler` falls through to default 404. `sendRequest` returns `(*sock.Response{StatusCode: 404}, nil)`. Ping helper sees `err == nil` → returns `true` → preflight passes → real RPC proceeds normally. The host_id echo check skips because the response body is empty. | Rolling-deploy safety: old pods are alive, just unaware of the ping route. |
| `pc.HostID == ""` (race during start) | `pingPipecatHost("")` returns `false` immediately, no RPC issued. | Defensive guard; should not happen in practice. |
| RabbitMQ broker down | `sendRequest` errors out → CB records failure → preflight returns `false`. The error is logged distinctly (not as "pod dead"). | Broker-flap protection. |
| Network partition: ping succeeds, real RPC then times out | Real RPC's own `sendRequest` path goes through the same CB target; a real timeout still records `RecordFailure`. CB eventually trips. | Ping doesn't promise the next RPC succeeds — only that the pod *was* reachable µs ago. |
| Concurrent sends to the same dead `HostID` | Each independently calls ping; the first 5 each pay 1 s, then the CB opens for all subsequent calls in the 30-s window. Bounded blast radius: ≤5 × 1 s of ping waste per fresh dead host. | Acceptable. Tracked as test case in §6. |
| Calico POD_IP reuse: new pod takes the dead pod's IP | Ping reaches the new pod; new pod responds with `host_id == requested hostID` (both equal POD_IP). Helper returns `true`. Real `PipecatV1MessageSend` is sent; the new pod's listenhandler returns 4xx because the session isn't in `mapPipecatcallSession`. ai-manager surfaces the error after one wasted RPC (~3 s). | Same outcome as today. v2 candidate: store `POD_UID` (immutable across IP reuse) in `pipecatcall` and verify it in the ping echo. Out of scope for v1 per non-goals. |
| Pod alive, session lost (in-memory cleanup via `ConnAstDone` lifecycle monitor — see `bin-pipecat-manager/CLAUDE.md`) | Same as Calico IP reuse: ping passes, real RPC fails. Same v2 fix applies (session-aware ping). | Same outcome as today. |

### 4.7 Logging

Per the monorepo CLAUDE.md (`Debug Logging for Retrieved Data`, `External Event & Webhook Processing Logs`):

- Ping success in ai-manager helper: **Debug** — high-frequency, not interesting alone.
  ```go
  log.WithField("host_id", hostID).Debug("Pipecat host ping succeeded.")
  ```
- Ping fail with `ErrCircuitOpen`: **Debug** — already known steady-state; the CB itself logs the open transition at Warn (`circuitbreakerhandler/main.go:96,115,127`).
- Ping fail with `context.DeadlineExceeded`: **Info** — visible in production, useful for capacity planning and rolling-deploy diagnostics.
  ```go
  log.WithField("host_id", hostID).Infof("Pipecat host ping timed out; treating as dead.")
  ```
- Ping fail with other errors (broker/transport): **Warn**, distinguished from pod-death.
  ```go
  log.WithField("host_id", hostID).Warnf("Pipecat ping failed with unexpected error; skipping per-pod RPC. err: %v", err)
  ```
- The CB itself emits `log.Warnf` on state transitions (already present).

### 4.8 What does NOT change

- DB schema (no Alembic migration).
- `pipecatcall` model fields.
- Python pipecat side.
- Existing public-facing API surface (`bin-api-manager` REST routes).
- RST docs in `bin-api-manager/docsdev/source/` — `/v1/ping` is internal RPC, not public.
- Conversation-AI v1 chatbot work — independent.

## 5. Backwards compatibility, rollout, and rollback

### Rollout

The change is additive in `bin-common-handler` (new method, no signature changes) and `bin-pipecat-manager` (new route, new handler method). The behavioral change in `bin-ai-manager` only kicks in when `PipecatV1Ping` is wired into the call site.

Rolling deploy order:

1. **Deploy `bin-pipecat-manager` first** — wait for the new pods to be 100% rolled out. New pods serve `/v1/ping`; any still-running pre-rollout pod returns 404 (treated as alive — see edge cases).
2. **Bump `bin-common-handler` library version** consumed by `bin-ai-manager` (no service deploy).
3. **Deploy `bin-ai-manager`** — preflight pings activate. Mid-rollout, mixed old/new ai-manager replicas coexist; old replicas don't ping (current behavior), new replicas do (improved behavior). No coordination required.

If step 1's rollout is paused mid-way, ai-manager (still on old code) is unaffected. Only step 3 introduces the new behavior, and by then step 1 should be complete.

### Rollback

The pipecat `/v1/ping` route is harmless dead code if unused. Two rollback paths:

- **Forward rollback (preferred):** revert the `bin-ai-manager` PR only and redeploy. Reverting that commit removes the `PipecatV1Ping` call sites in ai-manager source; vendor regenerates from source on the next Docker build. (Monorepo uses local `replace` directives in `go.mod`, so there is no version pin to roll back separately.) ai-manager stops calling `PipecatV1Ping`; pipecat-manager's `/v1/ping` route remains but is unused. No coordinated rollback needed; no separate vendor surgery required.
- **Full rollback:** revert all three commits in reverse order (ai-manager → common-handler → pipecat-manager).

No feature flag needed in v1 — the gating is one helper call at one site, trivial to revert.

## 6. Testing strategy

### `bin-pipecat-manager`
- `pkg/pipecatcallhandler/ping_test.go` (new): construct handler with a known `hostID`, call `Ping(ctx)`, assert `PingResult.HostID` and recent `Timestamp`.
- `pkg/listenhandler/v1_ping_test.go` (new): mock `pipecatcallHandler.Ping`, assert `200 application/json` response and JSON body. Negative case: `POST /v1/ping` falls through to 404.

### `bin-common-handler`
- `pkg/requesthandler/pipecat_ping_test.go` (new), table-driven (mirrors `pipecat_message_test.go`):
  - **Success, host_id matches**: mock `RequestPublish` returns 200 with body `{host_id: "X", ...}` for requested hostID `X` → `nil` error.
  - **Success, host_id mismatch (Calico-reuse phantom case)**: mock returns 200 with body `{host_id: "Y", ...}` for requested hostID `X` → returns mismatch error.
  - **Old-pod 404**: mock returns 404 with `nil` error → ping returns `nil` (alive). Body parsing must skip cleanly.
  - **Empty body 200**: mock returns 200 with empty `Data` → `nil` error (no echo check possible; degrade gracefully).
  - **Timeout**: mock blocks until ctx cancels; assert `context.DeadlineExceeded` propagates.
  - **Circuit open**: mock the breaker into open state; assert `ErrCircuitOpen` propagates.
  - **Queue-name regression**: assert exact queue name `bin-manager.pipecat-manager.request.<hostID>`.
- Circuit-breaker integration is already covered by `send_request_test.go`; do not re-test.

### `bin-ai-manager`
- `pkg/aicallhandler/ping_test.go` (new):
  - Alive → returns `true`.
  - Empty `hostID` → returns `false` and no RPC issued.
  - `ErrCircuitOpen` → returns `false`, debug-level log.
  - `context.DeadlineExceeded` → returns `false`, info-level log.
  - Other error → returns `false`, warn-level log.
  - **Concurrency**: 10 parallel calls to `pingPipecatHost("dead-host")`; first 5 each pay ≤1 s, ≥6th onwards complete in <100 ms because CB opened. Use a fake clock if needed to keep the test under 6 s.
- Extend existing `pkg/aicallhandler/send_test.go`:
  - `Send_PipecatHostAlive_CallsMessageSend` — ping mock returns success → `PipecatV1MessageSend` mock asserted called once.
  - `Send_PipecatHostDead_ReturnsErrorWithoutCallingMessageSend` — ping mock returns error → `PipecatV1MessageSend` mock asserted **never called**, `Send` returns non-nil error.
- **No edits** needed to `process_test.go` or `start_test.go` — those call sites are not gated.

### Mock-regen scope (operational note)

Adding `PipecatV1Ping` to the `RequestHandler` interface means `bin-common-handler/pkg/requesthandler/mock_main.go` regenerates with a new mock method. Every consumer that already constructs a `MockRequestHandler` and uses strict expectations (rather than `EXPECT().AnyTimes()` defaults) will keep compiling unchanged because the new method is additive, but tests that explicitly enumerate "no other calls expected" patterns may need an update.

In practice, only `bin-ai-manager`'s own `aicallhandler` tests need attention because they set explicit expectations. To keep the blast radius bounded, the `bin-ai-manager` PR should:
1. Add `EXPECT().PipecatV1Ping(...).AnyTimes()` to existing per-pod test fixtures, or
2. Add explicit positive expectations only for the new gated path, leaving other tests unchanged.

This is mechanical, not architectural — call out in PR description so reviewers know to check.

### Out of scope
- End-to-end against a real RabbitMQ broker.
- Re-testing the `circuitbreakerhandler` state machine.

## 7. Verification workflow per service touched

Per monorepo CLAUDE.md, before commit:

```bash
# 1. bin-common-handler
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# 2. bin-pipecat-manager
cd bin-pipecat-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# 3. bin-ai-manager
cd bin-ai-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

No other services need verification — `bin-common-handler` adds one new method (additive only); no existing signatures change.

## 8. Open questions / future work

- **Lower CB threshold for per-pod queues.** Default `5` means up to 5 × 1 s = 5 s of pain (across separate user messages) per dead `HostID` before fast-fail. If metrics show this hurts, add an optional per-target threshold to `circuitbreakerhandler` (e.g., register per-pod queues with threshold `1`).
- **Auto-recover on dead-pod-detected in `Send`.** Today: surface error. v2 candidate: spin up a fresh pipecatcall on the shared queue and resend. Requires care around message ordering and the `pipecatcall_id` rotation that already happens in `SendReferenceTypeOthers`.
- **Session-aware ping.** Today's process-level ping doesn't catch Calico IP-reuse or in-memory session loss without IP change. v2 candidate: `Ping(ctx, pipecatcallID)` checks `mapPipecatcallSession` and returns 404 if the session isn't owned. Requires a small DB schema change (or POD_UID storage) only if we want to verify pod identity without trusting the responding pod's self-report.
- **DB reaper for orphaned `pipecatcall` rows.** Independent cleanup task; out of scope here.
- **Circuit-breaker map and Prometheus cardinality.** `breakers map[string]*breaker` in `circuitbreakerhandler` accumulates one entry per ever-seen per-pod queue name. Memory cost is small (~tens of bytes per entry), but Prometheus labels also accumulate one series per ever-seen `target` value. Over months of pod churn (POD_IP recycling) this is unbounded. Fix candidates: (a) TTL-evict breaker map entries that have been Closed and idle for >1 h; (b) strip the `<POD_IP>` from the metric label and aggregate at the queue-prefix level. Defer until profiling shows pressure.

## 9. Decisions log

- **Ping scope = process-level** (queue alive + pod responsive), not session-level. The failure mode in scope is "pod restarted with a new IP." Session-level would catch Calico IP reuse and in-memory session loss but adds DB or RPC complexity for v1.
- **No caching in ai-manager.** The existing CB already provides 30-s negative caching for free per dead `HostID`. Adding a positive cache would shave a few ms in chat bursts at the cost of staleness; not worth it for v1.
- **Single gated call site (`send.go:46`).** Other per-pod call sites either don't pay the 3-s timeout (delayed RPCs) or aren't on a customer hot path (cleanup terminate). Keep v1 minimal.
- **Decision rule: send returns error.** Auto-restart is a v2 enhancement.
- **Use existing CB unchanged.** Threshold tuning is a follow-up if data demands it.
- **Keep `PipecatV1Ping` in `bin-common-handler`** alongside existing `PipecatV1*` methods, despite the 3-service admission rule — established precedent, and splitting pipecat client logic across two locations has worse ergonomics.
- **`Ping(ctx) (*PingResult, error)` on the server interface** — error return enables future drain-mode / 503 semantics without an interface break.
- **`host_id` echo verification on the client** — best-effort detection of queue-name collisions (low confidence on Calico IP reuse, high confidence on routing bugs).
