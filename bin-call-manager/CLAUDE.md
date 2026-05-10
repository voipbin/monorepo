# bin-call-manager

## Overview

`bin-call-manager` is the core telephony service in VoIPbin. It manages call resources, handles Asterisk ARI events, executes atomic call actions, and orchestrates the full call lifecycle — including conferences, recordings, external media streams, and group calls. It is a Class A standard Go RPC manager.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-call-manager`.

## Key Concepts

- **Call**: Individual call session with status tracking (`dialing/ringing/progressing/terminating/hangup`), confbridge membership, recording state
- **Confbridge**: Conference bridge joining multiple calls; `connect` type auto-terminates, `conference` type persists
- **Channel**: Asterisk channel representing a single media stream (SIP leg or WebRTC)
- **Bridge**: Asterisk bridge connecting channels for media mixing
- **Recording**: Call or confbridge recording session with `wav` format output
- **ExternalMedia**: WebRTC or RTP stream spliced into a call or confbridge via Asterisk snoop
- **GroupCall**: Multi-destination outbound call coordinator (`ring_all` or `linear` strategies)
- **OutboundConfig**: Per-customer outbound dialing config (codecs, source number override)

## Common Commands

| Command | Purpose |
|---------|---------|
| `cd bin-call-manager && go build ./...` | Compile |
| `go test ./...` | Run all tests |
| `go test -v ./pkg/callhandler/...` | Test a specific package |
| `golangci-lint run -v --timeout 5m` | Lint |
| `go generate ./...` | Regenerate mocks |
| `./bin/call-control call get --id <uuid>` | Inspect a call (bypasses RabbitMQ) |
| `./bin/call-control call update-status --id <uuid> --status hangup` | Force-update call status |

## Architecture
→ [docs/architecture.md](docs/architecture.md)

## Domain / Business Logic
→ [docs/domain.md](docs/domain.md)

## Dependencies
→ [docs/dependencies.md](docs/dependencies.md)

## Operations
→ [docs/operations.md](docs/operations.md)

## CRITICAL Rules

### Handler Dependency Order

Handler initialization order is fixed in `cmd/call-manager/main.go`. Respect this order to avoid circular dependencies:

```
dbhandler
  ├── channelhandler (reqHandler, notifyHandler, db)
  ├── bridgehandler (reqHandler, notifyHandler, db)
  ├── externalMediaHandler (reqHandler, notifyHandler, db, channelHandler, bridgeHandler)
  ├── recordingHandler (reqHandler, notifyHandler, db, channelHandler, bridgeHandler)
  ├── confbridgeHandler (reqHandler, notifyHandler, db, cache, channelHandler, bridgeHandler, recordingHandler, externalMediaHandler)
  ├── groupcallHandler (reqHandler, notifyHandler, db)
  ├── recoveryHandler (reqHandler, homerAPI config)
  ├── callHandler (reqHandler, notifyHandler, db, confbridgeHandler, channelHandler, bridgeHandler, recordingHandler, externalMediaHandler, groupcallHandler, recoveryHandler)
  └── ariEventHandler (sockHandler, db, cache, reqHandler, notifyHandler, callHandler, confbridgeHandler, channelHandler, bridgeHandler, recordingHandler, externalMediaHandler)
```

### Protected Directory

`bin-call-manager/doc/` contains native RST daemon docs. Do NOT modify any file under `doc/`.

### Database Pattern

This service uses **direct SQL** — no Squirrel query builder. Soft deletes use `tm_delete` timestamp (`"9999-01-01 00:00:00.000000"` for active records). Follow the existing pattern when adding queries.

### Cache Strategy

All call/channel/bridge/confbridge writes are mirrored to Redis immediately. ARI event processing reads from cache first. Never skip the cache update when writing to MySQL.

### RST Docs

When adding or changing any call/confbridge/recording/groupcall fields visible via the API, update the RST source in `bin-api-manager/docsdev/source/` and rebuild. See root CLAUDE.md for the rebuild procedure.
