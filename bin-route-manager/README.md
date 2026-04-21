# route-manager

Call routing service for the VoIPbin platform. Manages SIP providers and per-customer
routes, selects outbound providers for a dial target, and monitors provider health via
the `voip-kamailio-proxy` sidecar.

## Features

- Provider CRUD (SIP trunks/gateways)
- Route CRUD (per-customer target → provider priority mapping)
- Dialroute selection with customer override + system-default fallback
- Periodic provider health checks (SIP OPTIONS via `voip-kamailio-proxy`), circuit-breaker protected

## Route select

### Provider

SIP trunks/gateways used for outbound calls. See `models/provider/provider.go`.

### Route

Per-customer `target → provider` priority mapping.
Basic route customer id: `00000000-0000-0000-0000-000000000001` (system-wide defaults).

### Health check

A background loop in `pkg/healthcheckhandler/` probes each provider every 30s by
sending a SIP OPTIONS request through `voip-kamailio-proxy` via RabbitMQ RPC
(queue `voip.kamailio.request`, defined as `commonoutline.QueueNameKamailioRequest`
in `bin-common-handler`). The provider's `health_status` field is updated to
`healthy`, `unhealthy`, or `unknown` based on the SIP response. A per-target
circuit breaker protects against downstream failure.

<!-- Updated dependencies: 2026-02-20 -->
