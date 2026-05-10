# bin-conference-manager

## Overview

`bin-conference-manager` manages multi-party audio conference sessions in VoIPbin. It owns the Conference and Conferencecall (participant) entities, coordinates with call-manager's confbridge for the actual audio mixing, and handles conference lifecycle operations including recording and transcription. It is a Class A standard Go RPC manager.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-conference-manager`.

## Key Concepts

- **Conference**: Top-level session entity; types are `conference` (manual stop), `connect` (auto-stops when 1 participant left), `queue` (managed by queue-manager)
- **Conferencecall**: A participant in a conference; tracks the `reference_id` (call UUID) and join/leave status
- **Confbridge**: The actual audio-mixing bridge in call-manager; linked via `confbridge_id` on the conference — this service does NOT own confbridge
- **Recording**: Recording sessions are managed by call-manager; this service initiates/stops them and stores the `recording_id`
- **Transcription**: Live transcription started/stopped via bin-transcribe-manager; `transcribe_id` stored on the conference
- **Pre/post flows**: Optional flow IDs executed before participants speak and after the conference ends

## Common Commands

| Command | Purpose |
|---------|---------|
| `cd bin-conference-manager && go build ./...` | Compile |
| `go test ./...` | Run all tests |
| `go test -v ./pkg/conferencehandler/...` | Test a specific package |
| `golangci-lint run -v --timeout 5m` | Lint |
| `go generate ./...` | Regenerate mocks |

## Architecture
→ [docs/architecture.md](docs/architecture.md)

## Domain / Business Logic
→ [docs/domain.md](docs/domain.md)

## Dependencies
→ [docs/dependencies.md](docs/dependencies.md)

## Operations
→ [docs/operations.md](docs/operations.md)

## CRITICAL Rules

### Confbridge Ownership

This service does NOT own confbridges — call-manager does. When creating a conference, the confbridge must be created in call-manager first; the returned `confbridge_id` is stored on the conference. Do not attempt to manage confbridges directly.

### Event-Driven Participant State

Conferencecall status is updated by consuming call-manager events (confbridge join/leave) via `subscribehandler`. Do not poll call-manager for participant state. If participant events are lost, conferencecalls may remain in `initiating` state indefinitely.

### Type-Specific Stop Logic

Only send `stop` to a conference when explicitly requested. The `connect` type auto-terminates via call-manager confbridge logic — do not add duplicate stop logic for this type in this service.
