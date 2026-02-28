# Circuit Breaker for Inter-Service RPC

## Problem Statement

The VoIPbin monorepo has 30+ microservices communicating via RabbitMQ RPC through `requesthandler.sendRequest()`. Currently, if a downstream service becomes unavailable or slow:

- Every caller waits the full 3-second default timeout before getting an error
- There is no fast-fail mechanism — callers keep sending requests to a dead service
- No state tracking of downstream health exists
- Cascading failures can propagate across the service mesh

## Approach

Add a custom circuit breaker package (`circuitbreakerhandler`) to `bin-common-handler` and integrate it into `requestHandler.sendRequest()`. This gives all 30+ services circuit breaking with zero per-service code changes.

**Key decisions:**
- Scope: Inter-service RPC only (not external API calls)
- Granularity: Per target service queue (e.g., `bin-manager.call-manager.request`)
- State: In-memory per pod (no Redis dependency)
- Behavior: Fast-fail with `ErrCircuitOpen` error when circuit is open
- No new dependencies — custom implementation (~150 lines)

## State Machine

```
        +----------+    5 failures    +------+    30s timeout    +-----------+
        |  CLOSED  | --------------> |  OPEN | ----------------> | HALF-OPEN |
        | (normal) |                 |(fail) |                   |  (probe)  |
        +----------+                 +------+                   +-----------+
             ^                          ^                            |
             |   probe succeeds         |      probe fails           |
             +--------------------------+----------------------------+
```

**States:**
- Closed (default): Requests flow normally. Consecutive failures are counted.
- Open: Requests immediately rejected with `ErrCircuitOpen`. Entered when 5 consecutive failures occur.
- Half-Open: After 30s cooldown, 1 probe request is allowed. Success -> Closed, Failure -> Open.

**Default parameters:**
- Failure threshold: 5 consecutive failures
- Open duration: 30 seconds
- Half-open max probes: 1

## Package Structure

```
bin-common-handler/pkg/circuitbreakerhandler/
  main.go           # CircuitBreakerHandler interface + implementation (registry of per-target breakers)
  breaker.go         # Individual breaker state machine
  option.go          # Configuration constants
  main_test.go       # Tests for the registry
  breaker_test.go    # Tests for state transitions
  mock_main.go       # Generated mock
```

## Integration Point

The circuit breaker is injected into `requestHandler.sendRequest()` in `bin-common-handler/pkg/requesthandler/send_request.go`:

```go
type requestHandler struct {
    sock        sockhandler.SockHandler
    publisher   commonoutline.QueueName
    serviceName service.Name
    cb          circuitbreakerhandler.CircuitBreakerHandler  // NEW
}
```

In `sendRequest()`:

```go
func (r *requestHandler) sendRequest(ctx context.Context, queue commonoutline.QueueName, ...) (*sock.Response, error) {
    // ... existing code ...

    switch {
    case delay > 0:
        // delayed requests are fire-and-forget -- no circuit breaking
        return r.sendDelayedRequest(...)

    default:
        // Check circuit breaker BEFORE sending
        if err := r.cb.Allow(string(queue)); err != nil {
            return nil, err  // fast-fail: circuit is open
        }

        res, err := r.sendDirectRequest(cctx, string(queue), resource, req)

        // Record result for circuit breaker
        if err != nil {
            r.cb.RecordFailure(string(queue))
        } else {
            r.cb.RecordSuccess(string(queue))
        }

        return res, err
    }
}
```

**Key behaviors:**
- Delayed requests (fire-and-forget) are NOT circuit-broken
- Breakers are created lazily on first request to each target
- `NewRequestHandler()` creates the circuit breaker handler internally (no signature change)
- Zero changes to any service's `main.go`

## Error Type

```go
var ErrCircuitOpen = errors.New("circuit breaker is open")
```

Callers can check with `errors.Is(err, circuitbreakerhandler.ErrCircuitOpen)` for special handling. The error is wrapped with target context: `"circuit breaker is open for target: bin-manager.call-manager.request"`.

## Observability

**Prometheus metrics (namespaced per service):**
- `circuitbreaker_state_transitions_total{target, from, to}` — counter of state transitions
- `circuitbreaker_state{target}` — gauge of current state (0=closed, 1=half-open, 2=open)
- `circuitbreaker_rejected_total{target}` — counter of rejected requests

**Logging:**
- `log.Warnf` on state transitions only (Closed->Open, Open->HalfOpen, HalfOpen->Closed/Open)
- No per-request logging (too noisy)

## Configuration

Hardcoded defaults for v1. No per-service config needed. If tuning is needed later, add env vars or functional options.

The circuit breaker is created internally in `NewRequestHandler()` — no constructor signature change.

**Rollout safety:** The circuit breaker only activates after 5 consecutive failures per target. For emergency disable, an env var `CIRCUIT_BREAKER_ENABLED=false` can be added later.

## Testing Strategy

1. `breaker_test.go` — state machine:
   - Starts Closed
   - N consecutive failures -> Open
   - Open rejects with ErrCircuitOpen
   - After timeout -> Half-Open
   - Half-Open: success -> Closed, failure -> Open
   - Success resets failure count
   - Thread safety under concurrent access

2. `main_test.go` — registry:
   - Lazy creation per target
   - Independent breakers per target
   - Allow/RecordSuccess/RecordFailure dispatch correctly

3. `send_request_test.go` — integration:
   - Circuit open -> sendRequest returns ErrCircuitOpen without calling sock
   - Circuit closed -> request goes through normally
   - After threshold failures -> subsequent calls rejected

## Files to Change

**New files:**
- `bin-common-handler/pkg/circuitbreakerhandler/main.go`
- `bin-common-handler/pkg/circuitbreakerhandler/breaker.go`
- `bin-common-handler/pkg/circuitbreakerhandler/option.go`
- `bin-common-handler/pkg/circuitbreakerhandler/main_test.go`
- `bin-common-handler/pkg/circuitbreakerhandler/breaker_test.go`

**Modified files:**
- `bin-common-handler/pkg/requesthandler/main.go` — add `cb` field to struct, init in constructor
- `bin-common-handler/pkg/requesthandler/send_request.go` — add Allow/Record calls
- `bin-common-handler/pkg/requesthandler/send_request_test.go` — add circuit breaker test cases
- `bin-common-handler/pkg/requesthandler/prometheus.go` — register circuit breaker metrics

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Circuit trips on transient errors | 5-failure threshold absorbs transient blips |
| Circuit never recovers | Half-open probe with 30s timeout ensures automatic recovery testing |
| Metric name collision | Use unique prefix `circuitbreaker_` (not in existing requesthandler metrics) |
| All services need vendor update | No code changes needed, just `go mod vendor` per service |
| Thread safety | Use `sync.Mutex` per breaker for state transitions |
