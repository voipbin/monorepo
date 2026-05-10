# voip-asterisk-proxy — Architecture

Service class: **A+sub** (Go RPC manager + embedded Asterisk PBX daemon).

## Component Overview

`voip-asterisk-proxy` is a bidirectional proxy that bridges the Asterisk PBX subsystem (via ARI WebSocket and AMI TCP) with the VoIPbin RabbitMQ message bus. It runs co-located with an Asterisk daemon (same pod/host) and acts as its sole interface to the rest of the platform.

```
                    ┌──────────────────────────────────────┐
                    │         voip-asterisk-proxy           │
                    │                                       │
  RabbitMQ ────────►│  ListenHandler                        │
  (requests)        │  • permanent queue (asterisk.*.request)│
                    │  • volatile queue (asterisk.<id>.request)│
                    │         │                             │
                    │         ▼                             │
                    │    processRequest()                   │──► Asterisk ARI  (HTTP)
                    │    /ari/* → ARI HTTP proxy            │──► Asterisk AMI  (TCP)
                    │    /ami/* → AMI action                │──► GCS upload
                    │    /proxy/recording_file_move → GCS   │
                    │                                       │
  RabbitMQ ◄────────│  EventHandler                         │◄── Asterisk ARI  (WebSocket)
  (events)          │  • ARI WebSocket reader               │◄── Asterisk AMI  (TCP events)
                    │  • AMI event reader                   │
                    │                                       │
                    │  ServiceHandler (GCS recordings)      │
                    │  Annotation updater (Kubernetes)      │
                    │  Redis address registration           │
                    └──────────────────────────────────────┘
                              │
                              ▼
                         Asterisk PBX
                        (co-located)
```

## Layer Responsibilities

| Layer | Package / file | Responsibility |
|-------|---------------|----------------|
| Entry point | `cmd/asterisk-proxy/main.go` | Wire all handlers, connect to RabbitMQ + AMI, identify instance via MAC address, start goroutines |
| Configuration | `cmd/asterisk-proxy/init.go` | Bind all Viper flags + env vars; initialize Prometheus |
| Kubernetes annotation | `cmd/asterisk-proxy/annotation.go` | Patch pod annotation `asterisk-id` with MAC; update Redis key `asterisk.<id>.address-internal` every 5 min |
| Event handler | `pkg/eventhandler/` | Maintain ARI WebSocket + AMI connections; forward events to RabbitMQ; auto-reconnect |
| Listen handler | `pkg/listenhandler/` | Consume RabbitMQ request queues; route to ARI/AMI/proxy; return RPC replies |
| Service handler | `pkg/servicehandler/` | Upload Asterisk recording files to Google Cloud Storage (GCS) |

## Request Routing

The listen handler matches incoming RabbitMQ RPC requests by URI pattern:

| Pattern (regex) | Handler | Action |
|-----------------|---------|--------|
| `^/ari/` | `ariSendRequestToAsterisk` | HTTP-proxy the request to Asterisk ARI at `http://<ari_address>/<path>` |
| `^/ami/` | AMI action sender | Serialize the request body as an AMI action and send over the TCP AMI socket |
| `^/proxy/recording_file_move$` | `serviceHandler.MoveRecordingFile` | Read recording from Asterisk directory, upload to GCS bucket |

Unmatched URIs return HTTP 400. The pattern match runs in `processRequest()` in `pkg/listenhandler/main.go`.

### Queue naming convention

| Queue type | Name format | Purpose |
|------------|------------|---------|
| Permanent | `asterisk.<type>.request` (e.g., `asterisk.call.request`) | Shared across all proxy instances of this Asterisk type |
| Volatile | `asterisk.<mac-address>.request` | Instance-specific; routes requests to one specific Asterisk pod |

Both queues are consumed concurrently. The volatile queue enables targeted control (e.g., hang up a specific call on a specific pod).
