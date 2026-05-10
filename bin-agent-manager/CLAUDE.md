# bin-agent-manager

## Overview

`bin-agent-manager` manages call-center agents in VoIPbin: their identity, authentication, SIP contact addresses, permissions, real-time status (available/away/busy/offline/ringing), and tag-based routing membership. It is a Class A standard Go RPC manager.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-agent-manager`.

## Key Concepts

- **Agent**: A call-center operator with status, SIP addresses, permission flags, ring method, and tag IDs
- **Status**: Real-time availability — `available`, `away`, `busy`, `offline`, `ringing`; driven by call-manager events
- **Ring method**: `ringall` (all addresses simultaneously) or `linear` (addresses tried in sequence)
- **Permission**: Bitfield with project-level and customer-level flags (agent/admin/manager)
- **Tag IDs**: Used by queue-manager to filter eligible agents for routing; changing tags immediately affects queue membership
- **Password reset**: Requires `password_reset_base_url` config; uses email-manager for delivery

## Common Commands

| Command | Purpose |
|---------|---------|
| `cd bin-agent-manager && go build ./...` | Compile |
| `go test ./...` | Run all tests |
| `go test -v ./pkg/agenthandler/...` | Test a specific package |
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

### Status is Event-Driven

Agent status changes from `ringing` → `available` are driven by call-manager events received in `subscribehandler`. Do not rely on polling or timeouts for status recovery. If subscribehandler is down, agents can remain stuck in incorrect states.

### Password Reset Base URL

The `password_reset_base_url` config flag MUST be set in production for password reset emails to contain a valid link. Deploying without this flag causes silent failures in the forgot-password flow.

### Address Lookup Exactness

The `get_by_customer_id_address` endpoint does exact-match lookup on the SIP address string. Ensure SIP URI normalization is consistent when storing and querying addresses (same port, same parameters).
