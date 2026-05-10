# CLAUDE.md — bin-api-manager

`bin-api-manager` is the sole HTTP/REST API gateway for the VoIPbin platform. It receives all external REST requests, enforces authentication and authorization, and fans them out to ~30 backend services via RabbitMQ RPC. It owns no business logic — that lives entirely in the backend managers.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-api-manager`.

## Docs

- [docs/architecture.md](docs/architecture.md) — component diagram, package structure, middleware stack, backend service map
- [docs/routing.md](docs/routing.md) — full REST route table with backend service column (~30 domain groups)
- [docs/auth.md](docs/auth.md) — JWT/accesskey authentication flow, permission model, two-level handler pattern
- [docs/operations.md](docs/operations.md) — configuration flags, Prometheus metrics, common failure modes, debugging guide

## CRITICAL: RST Docs Sync

**When adding or changing any user-visible feature, you MUST update `docsdev/source/` and rebuild HTML.**

The RST docs at `docsdev/source/` are the live user-facing documentation served at https://docs.voipbin.net/. Stale docs actively mislead customers.

**Applies when you:**
- Add or modify API endpoints (update the resource's `*_overview.rst`, `*_tutorial.rst`, `*_struct.rst`)
- Add new event types, statuses, or fields visible in responses
- Change billing, pricing, or webhook behavior

**Required steps:**
```bash
# 1. Edit RST source files in docsdev/source/
# 2. Clean rebuild
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
# 3. Force-add built HTML (root .gitignore excludes build/)
git add -f bin-api-manager/docsdev/build/
# 4. Commit RST sources and build/ together
```

**RST struct docs MUST match `WebhookMessage`, not internal model structs.** Always compare against `models/<entity>/webhook.go` in the backend service, not the internal model. Fields like `PodID`, `Username`, `PermissionIDs` are stripped by `ConvertWebhookMessage()` and must NOT appear in RST docs.

See also: [docs/workflows/special-cases.md](../docs/workflows/special-cases.md)

## Common Commands

### Prerequisites (ZMQ native libs)
```bash
apt update && apt install -y pkg-config libzmq5 libzmq3-dev libczmq4 libczmq-dev
```

### Build
```bash
go mod download
go mod vendor
go build ./cmd/...
```

### Test
```bash
go test -v $(go list ./...)
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -func=cp.out
```

### Lint / Verify
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

### Code generation (after OpenAPI spec changes)
```bash
# 1. Update spec in bin-openapi-manager first
cd ../bin-openapi-manager && go generate ./...
# 2. Regenerate server code in this service
cd ../bin-api-manager && go generate ./...
```

### Health check
```bash
./bin/api-control health
```

## Key Patterns

### OpenAPI-First Workflow
Never modify handler code without updating `bin-openapi-manager/openapi/openapi.yaml` first. The generated types in `gens/openapi_server/gen.go` are the contract. Never hand-edit the generated file.

### Two-Level Handler Pattern
Every resource: private `resourceGet()` (no permission check) + public `ResourceGet()` (calls helper, runs `hasPermission`, returns `.ConvertWebhookMessage()`). See [docs/auth.md](docs/auth.md).

### WebhookMessage Conversion
External responses MUST use `.ConvertWebhookMessage()`. Returning the internal struct leaks internal fields.

### Service Name Constants
Use `commonoutline.ServiceName*` (from `bin-common-handler/models/outline/servicename.go`) instead of string literals for service names in RPC calls.

### Auth/Authorization Boundary
Auth and ownership checks belong ONLY in bin-api-manager. Backend services never check JWT or customer ownership. See [docs/auth.md](docs/auth.md).

## Architecture Quick Reference

```
Client (HTTPS) → bin-api-manager → RabbitMQ RPC → ~30 backend managers
                      ↓
               WebSocket clients  (pkg/websockhandler)
               bin-webhook-manager (event forwarding)
               Audiosocket port 9000 (AI audio streaming)
```

Packages: `server/` (HTTP handlers) → `pkg/servicehandler/` (auth+RPC delegation) → `bin-common-handler/pkg/requesthandler` (RabbitMQ RPC).

## External Links

- API Reference (ReDoc): https://api.voipbin.net/redoc/index.html
- API Reference (Swagger): https://api.voipbin.net/swagger/index.html
- Developer Docs: https://docs.voipbin.net/
