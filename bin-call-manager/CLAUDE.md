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

This service uses the **Squirrel query builder**, like the rest of the monorepo (migrated in VOIP-1078). Raw SQL strings are forbidden — see [docs/conventions/database.md](../docs/conventions/database.md) §7.1.

- Reads go through `commondatabasehandler.ScanRow` + `GetDBFields`; writes go through `PrepareFields` + `SetMap`. Do not hand-roll column-by-column `row.Scan` or inline datetime parsing.
- Use struct-based `PrepareFields` only for full-row INSERTs, and assign the timestamp fields explicitly first — it emits every `db`-tagged field unconditionally, so an omitted assignment writes SQL `NULL`.
- Use map-based `PrepareFields` for partial updates. Put **dereferenced** `uuid.UUID` values in the map, never `*uuid.UUID`: the pointer is not converted and reaches the driver as a 36-char string, which does not fit a `BINARY(16)` column.
- The one sanctioned raw-SQL escape hatch is `squirrel.Expr`, for MySQL JSON functions and atomic arithmetic. The recurring JSON shapes live in `pkg/dbhandler/json_expr.go`; every call site needs a `WHY` comment. SQLite cannot execute these, so they are pinned by golden `ToSql()` assertions in `pkg/dbhandler/json_expr_golden_test.go` rather than by round-trip tests.

**Empty-collection normalization:** whether a nil slice/map is stored as `[]`/`{}`, the JSON literal `null`, or SQL `NULL` differs per field *and per call site*. The authoritative table is the doc comment at the top of `pkg/dbhandler/normalization.go` — consult it before touching any JSON column, and do not "tidy" it into a uniform rule.

**Soft deletes:** this service has two co-existing conventions, and they are deliberately **not** unified:
- `tm_delete IS NULL` for active records — used by `call_bridges`, `call_channels`, `call_outbound_configs`.
- the `tm_delete = "9999-01-01 00:00:00.000000"` sentinel for active records — used by the other tables.

Match whichever convention the table you are touching already uses. `commondatabasehandler.ApplyFields` translates a `deleted` filter to `tm_delete IS NULL` / `IS NOT NULL`, so it suits the first convention only.

### Cache Strategy

All call/channel/bridge/confbridge writes are mirrored to Redis immediately. ARI event processing reads from cache first. Never skip the cache update when writing to MySQL.

### RST Docs

When adding or changing any call/confbridge/recording/groupcall fields visible via the API, update the RST source in `bin-api-manager/docsdev/source/` and rebuild. See root CLAUDE.md for the rebuild procedure.
