# voip-kamailio-proxy — Architecture

Service class: **A+sub** (Go RPC manager + embedded Kamailio SIP proxy daemon).

## Component Overview

`voip-kamailio-proxy` is a lightweight RabbitMQ-to-Kamailio bridge. It exposes a single RPC endpoint that accepts SIP provider health check requests from the platform and dispatches SIP OPTIONS packets directly over UDP — without going through Kamailio. The Go service runs co-located with a Kamailio daemon (same pod) and registers its identity on the message bus using the MAC address of its primary network interface.

```
                    ┌──────────────────────────────────────┐
                    │         voip-kamailio-proxy           │
                    │                                       │
  RabbitMQ ────────►│  ListenHandler                        │
  (requests)        │  • permanent queue (voip.kamailio.request)│
                    │  • volatile queue (voip.kamailio.<mac>.request)│
                    │         │                             │
                    │         ▼                             │
                    │    processRequest()                   │
                    │    POST /v1/providers/health          │──► UDP SIP OPTIONS
                    │         │                             │    (port 5060)
                    │         ▼                             │
                    │    SIPChecker (siphandler pkg)        │
                    └──────────────────────────────────────┘
                              │
                              ▼
                         Kamailio SIP Proxy
                         (co-located daemon)
```

The SIP OPTIONS health check is performed by the Go service itself (not forwarded to Kamailio). Kamailio handles all actual SIP routing; the Go proxy only handles health checks and RabbitMQ routing.

## Layer Responsibilities

| Layer | Package / file | Responsibility |
|-------|---------------|----------------|
| Entry point | `cmd/kamailio-proxy/main.go` | Wire handlers, connect RabbitMQ, derive instance ID from MAC address |
| Configuration | `cmd/kamailio-proxy/init.go` | Bind all Viper flags + env vars; initialize Prometheus |
| Listen handler | `pkg/listenhandler/` | Consume RabbitMQ queues; route incoming RPC to health check handler |
| SIP handler | `pkg/siphandler/` | Send SIP OPTIONS over raw UDP; parse SIP response line |

## Request Routing

The listen handler matches incoming RabbitMQ RPC requests by URI pattern:

| Pattern (regex) | Method | Handler | Action |
|-----------------|--------|---------|--------|
| `^/v1/providers/health$` | POST | `processV1ProvidersHealthPost` | Dial UDP to `hostname:5060`, send SIP OPTIONS, return `healthy`/`unhealthy` + SIP response code |

Unmatched URIs return status 404. Pattern matching runs in `processRequest()` in `pkg/listenhandler/main.go`.

### Queue naming convention

| Queue type | Name format | Purpose |
|------------|------------|---------|
| Permanent | `voip.kamailio.request` | Shared; any proxy instance can handle the request |
| Volatile | `voip.kamailio.<mac-address>.request` | Instance-specific; routes requests to one specific Kamailio pod |

Both queues are consumed concurrently. The volatile queue format uses the MAC address of `--interface_name` (default `eth0`).
