# voip-rtpengine-proxy — Architecture

Service class: **A+sub** (Go RPC manager + embedded RTPEngine media proxy daemon).

## Component Overview

`voip-rtpengine-proxy` bridges the VoIPbin RabbitMQ message bus with an RTPEngine media proxy running on the same host. It forwards NG protocol commands to RTPEngine over UDP/bencode, manages tcpdump capture processes for call recording, and uploads captured pcap files to Google Cloud Storage.

```
                    ┌──────────────────────────────────────────┐
                    │          voip-rtpengine-proxy             │
                    │                                          │
  RabbitMQ ────────►│  ListenHandler                           │
  (requests)        │  • permanent queue (rtpengine.proxy.request)│
                    │  • volatile queue (rtpengine.<ip>.request)│
                    │         │                                │
                    │         ▼                                │
                    │    processCommandPost()                  │
                    │    POST /v1/commands                     │
                    │         │                                │
                    │    type: "ng"  ──────────────────────────┼──► RTPEngine NG
                    │    type: "exec" ─────────────────────────┼──► tcpdump (start)
                    │    type: "kill" ─────────────────────────┼──► tcpdump (stop)
                    │         │                                │     + GCS upload
                    │         ▼                                │
                    │    NGClient (bencode/UDP)                │
                    │    ProcessManager (tcpdump)              │
                    │    GCSUploader (pcap → GCS)              │
                    │    PcapWatcher (dir → GCS on file close) │
                    └──────────────────────────────────────────┘
                              │
                              ▼
                         RTPEngine daemon
                         (co-located, NG port 22222)
```

## Layer Responsibilities

| Layer | Package / file | Responsibility |
|-------|---------------|----------------|
| Entry point | `cmd/rtpengine-proxy/main.go` | Wire all components, connect RabbitMQ + Redis, derive instance ID from IP, start goroutines |
| Configuration | `cmd/rtpengine-proxy/init.go` | Bind all Viper flags + env vars; initialize Prometheus |
| Listen handler | `pkg/listenhandler/` | Consume RabbitMQ queues; route commands by type to NG client or process manager |
| NG client | `pkg/ngclient/` | Dial RTPEngine NG UDP port; encode/decode bencode; match cookie-based request/response |
| Process manager | `pkg/processmanager/` | Start/stop tcpdump processes by UUID; enforce safety timeout; upload pcap to GCS on kill |
| GCS uploader | `pkg/gcsuploader/` | Upload local files to a GCS bucket; used by process manager and pcap watcher |
| Pcap watcher | `pkg/pcapwatcher/` | Watch RTPEngine recording directory; upload completed pcap files to GCS |
| Command model | `models/command/` | Define `Command` struct and `Type` constants (`ng`, `exec`, `kill`) |

## Request Routing

The listen handler matches incoming RabbitMQ RPC requests by URI pattern:

| Pattern (regex) | Method | Handler | Action |
|-----------------|--------|---------|--------|
| `^/v1/commands$` | POST | `processCommandPost` | Dispatch by `type` field: `ng` → NGClient, `exec` → ProcessManager.Exec, `kill` → ProcessManager.Kill |

Unmatched URIs or unknown `type` values return status 404 or 400 respectively.

### Command type routing

| `type` value | Required fields | Action |
|-------------|----------------|--------|
| `ng` | `command` (string) | Forward raw bencode map to RTPEngine NG UDP endpoint; return RTPEngine response |
| `exec` | `id` (UUID), `command`, optional `parameters` | Start a tcpdump process; write pcap to `/tmp/<id>.pcap` |
| `kill` | `id` (UUID) | SIGTERM the tcpdump process; wait up to 5s; SIGKILL if needed; upload pcap to GCS |

### Queue naming convention

| Queue type | Name format | Purpose |
|------------|------------|---------|
| Permanent | `rtpengine.proxy.request` | Shared; any proxy instance handles the request |
| Volatile | `rtpengine.<ip-address>.request` | Instance-specific; routes to one specific RTPEngine pod |

Instance identity is the IPv4 address of `--interface_name` (default `eth0`), registered in Redis at startup and refreshed every 5 minutes.
