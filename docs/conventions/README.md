# Conventions

> **This is the authoritative source for all coding conventions in this monorepo.**
> For workflow rules (verification, git, deployment), see [CLAUDE.md](../../CLAUDE.md).
> For architecture details, see [architecture-deep-dive.md](../architecture/architecture-deep-dive.md).

This directory is the **single source of truth** for VoIPbin coding conventions. It supersedes the previous monolithic `docs/coding-conventions.md`.

Recommended reading order for new engineers:

1. [package-structure.md](package-structure.md) — file layout and the bin-common-handler 3-service admission rule
2. [naming.md](naming.md) — Go naming conventions
3. [imports.md](imports.md) — import grouping and ordering
4. [error-handling.md](error-handling.md) — wrap, log, propagate
5. [logging.md](logging.md) — Debug for retrieved data, Info for external events
6. [models.md](models.md) — struct definitions, field tags, JSON marshaling
7. [database.md](database.md) — squirrel + commondatabasehandler + dbhandler-only access
8. [handlers.md](handlers.md) — handler interface + struct + constructor pattern
9. [rpc.md](rpc.md) — inter-service RabbitMQ RPC
10. [api-design.md](api-design.md) — atomic responses, WebhookMessage pattern reference
11. [events.md](events.md) — pub/sub
12. [configuration.md](configuration.md) — env vars, viper
13. [testing.md](testing.md) — mockgen, table-driven tests, coverage targets
14. [metrics.md](metrics.md) — Prometheus naming, no name collisions with shared library
15. [security.md](security.md) — secrets, input validation, authorization
16. [direct-resource-types.md](direct-resource-types.md) — no magic strings for direct resource types
