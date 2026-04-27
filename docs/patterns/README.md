# Patterns

Applied infrastructure patterns with reference implementations in the VoIPbin codebase. Distinguish from style conventions in [conventions/](../conventions/) — patterns here are about how to combine pieces (queues, breakers, RPCs); conventions are about how to format individual pieces.

| File | Description |
|---|---|
| [circuit-breaker.md](circuit-breaker.md) | Per-target circuit breaker auto-integrated with `r.sendRequest()` — defaults, state machine, free Prometheus metrics |
| [per-pod-liveness-preflight.md](per-pod-liveness-preflight.md) | Sub-second `/v1/ping` preflight for per-pod RPC, distinguishing dead pod from broker outage; rules from PR #832 |
| [per-pod-queues.md](per-pod-queues.md) | `<service>.request.<host_id>` volatile-queue convention for session-affinity routing |
| [webhook-message.md](webhook-message.md) | External API response pattern using `WebhookMessage` and `ConvertWebhookMessage()` to keep internal fields out of customer-facing responses |

## Admission criteria

A pattern belongs here when it has a reference implementation with code, ideally consumed by 2+ services or an obvious candidate for that. Single-service patterns belong inline in that service's `CLAUDE.md`. Style rules (naming, formatting, error wrapping) belong in [conventions/](../conventions/).
