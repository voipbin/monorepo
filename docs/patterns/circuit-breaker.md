# Circuit Breaker

VoIPbin has a per-target (per-RabbitMQ-queue) circuit breaker built into every `r.sendRequest()` call in `bin-common-handler/pkg/requesthandler/`. You almost never need to add your own retry / timeout / fast-fail logic — the shared CB covers all RPC paths.

## Where it lives

- Implementation: `bin-common-handler/pkg/circuitbreakerhandler/`
- Integration point: `bin-common-handler/pkg/requesthandler/send_request.go:50-67`
- Defaults: `bin-common-handler/pkg/circuitbreakerhandler/option.go`
  - `defaultFailureThreshold = 5` (consecutive failures before opening)
  - `defaultOpenDuration = 30 * time.Second` (window during which requests fast-fail)

## State machine

- **Closed** — `cb.Allow()` returns nil; the underlying RPC runs.
- **Open** — `cb.Allow()` returns `ErrCircuitOpen` immediately; no RPC is issued. Triggered by 5 consecutive failures recorded against the same target.
- **Half-open** — entered automatically 30s after Open. One probe is allowed; success → Closed, failure → Open for another 30s.

State is per-target (per RabbitMQ queue name). One dead pipecat-manager pod's per-pod queue trips its own breaker without affecting the shared `pipecat-manager.request` queue or any other service.

## Free Prometheus metrics

The CB auto-exports per-target metrics on the namespace of the consuming service (e.g., `ai_manager_*`):

- `<namespace>_circuitbreaker_state{target="<queue>"}` — gauge: 0 closed, 1 open, 2 half-open
- `<namespace>_circuitbreaker_state_transitions_total{target,from,to}` — counter
- `<namespace>_circuitbreaker_rejected_total{target}` — counter incremented on each Open-state rejection

Add these to your service Grafana dashboard if you handle critical RPC paths.

## When NOT to add a new circuit breaker

Don't, unless your RPC path bypasses `r.sendRequest()`. The vast majority of services route through it; just call your `r.<Service>V1<Method>(...)` and you get the breaker for free.

## When to tune the threshold

The defaults (5 failures / 30s) suit ~90% of cases. If telemetry shows you need a more aggressive threshold for a specific target (e.g., per-pod queues that should fast-fail after one timeout), file an issue first; tuning is currently per-target and requires a change inside `circuitbreakerhandler/`. Don't roll a parallel breaker — extend the shared one.

## Reference: how PR #832 leveraged this

The per-pod liveness preflight in PR #832 (see `docs/patterns/per-pod-liveness-preflight.md`) routes its 1-second `PipecatV1Ping` through `r.sendRequest()` so 5 consecutive ping failures against the same dead `HostID` open the breaker, giving 30 seconds of microsecond fast-fail with zero new state. That design choice — "ping is just another sendRequest" — let v1 ship without a new CB or cache.
