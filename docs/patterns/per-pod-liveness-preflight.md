# Per-pod Liveness Preflight

When a per-pod RPC's stored `HostID` may go stale (e.g., the target pod restarts on K8s and gets a new IP), gate the call with a sub-second preflight `/v1/ping` against the per-pod queue. This pattern was introduced in PR #832 for `bin-pipecat-manager` and applies to any future per-pod service.

## When to use

- The consumer holds a `HostID` (typically `POD_IP`) that points to a specific pod owning in-memory state.
- The pod may die between the time the `HostID` was stored and the time it's used.
- The cost of a 3s RabbitMQ RPC timeout per failed call is unacceptable on a customer hot path.

## Server side

Add a lightweight `GET /v1/ping` route on the per-pod queue. Return process identity only — no DB I/O, no business logic.

```go
type PingResult struct {
    HostID    string    `json:"host_id"`
    Timestamp time.Time `json:"timestamp"`
}

func (h *handler) Ping(ctx context.Context) (*PingResult, error) {
    return &PingResult{HostID: h.hostID, Timestamp: time.Now().UTC()}, nil
}
```

Wire into the listenhandler's per-pod queue route table.

## Client side

Add a `XxxV1Ping(ctx, hostID) error` method to `bin-common-handler/pkg/requesthandler/` (alongside the existing per-service RPCs). 1-second timeout. Routes through `r.sendRequest()` so it shares the existing per-target circuit breaker (see [circuit-breaker.md](circuit-breaker.md)).

In the consumer service, add a small helper that distinguishes error classes via `errors.Is` (relies on `pkg/errors v0.9.0+` Unwrap):

```go
func (h *handler) pingHost(ctx context.Context, hostID string) bool {
    if hostID == "" {
        return false
    }
    cctx, cancel := context.WithTimeout(ctx, 1100*time.Millisecond)
    defer cancel()
    err := h.reqHandler.XxxV1Ping(cctx, hostID)
    if err == nil {
        return true
    }
    switch {
    case errors.Is(err, circuitbreakerhandler.ErrCircuitOpen):
        // CB is already open; fast-fail
    case errors.Is(err, context.DeadlineExceeded):
        // pod is dead
    default:
        // broker / transport error — distinguish from pod death
    }
    return false
}
```

## Rules

1. **Any response = alive.** Old pods returning 404 (because they predate the route) must be treated as alive — they responded. Do not add status-code checks. The only "dead" signal is `err != nil` (timeout or `ErrCircuitOpen`).
2. **Best-effort `host_id` echo.** Compare the response `HostID` to the requested one to catch routing bugs. This does NOT detect Calico POD_IP recycle (where a different pod takes the dead pod's IP).
3. **Run preflight before any DB write.** A dead-pod failure should not orphan rows. Lesson from PR #832 final review: persisting the message before the ping created orphan rows when the pod was dead.
4. **Outer 1.1s context wraps inner 1s RPC.** The inner sendRequest enforces the 1s hard timeout; the outer 100ms slack is a safety net for uncancellable upstream contexts.

## What this pattern does NOT solve

- **Calico POD_IP recycle.** Within minutes, a dead pod's IP may be reassigned to a new pod that responds to the ping. The `host_id` echo matches (both equal POD_IP) so detection fails. Fallback: the new pod's listenhandler returns 4xx because the session isn't in its session map; one wasted real RPC. Same as today's behavior. v2 candidate: store `POD_UID` for IP-recycle-safe identity.
- **Session-level liveness.** This is process-level only — the pod is up, but the specific session may have been cleaned up in-memory. Treat as the same edge case as Calico recycle.

## Reference implementation

- Design: `docs/plans/2026-04-26-pipecat-pod-ping-design.md`
- Code: `bin-pipecat-manager/pkg/listenhandler/v1_ping.go`, `bin-common-handler/pkg/requesthandler/pipecat_ping.go`, `bin-ai-manager/pkg/aicallhandler/ping.go`
