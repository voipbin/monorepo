# voip-kamailio-proxy

Service class: **A+sub** — Go RPC proxy co-located with a Kamailio SIP proxy daemon. Exposes SIP OPTIONS health check endpoints over RabbitMQ.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Detailed documentation

- [docs/architecture.md](docs/architecture.md) — component overview, layer responsibilities, request routing
- [docs/domain.md](docs/domain.md) — domain entities, key business rules
- [docs/dependencies.md](docs/dependencies.md) — monorepo and external dependencies
- [docs/operations.md](docs/operations.md) — failure modes, debugging guide, configuration, Prometheus metrics
- [docs/subsystems.md](docs/subsystems.md) — Kamailio daemon overview, configuration, deployment notes

## Common commands

```bash
# Build
go build -o bin/kamailio-proxy ./cmd/kamailio-proxy

# Test
go test ./...

# Full verification (required before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Key implementation facts

- Entry point: `cmd/kamailio-proxy/main.go`; configuration in `cmd/kamailio-proxy/init.go`
- Instance identity: MAC address of `--interface_name` (default `eth0`); used to name the volatile queue `voip.kamailio.<mac>.request`
- Single RPC endpoint: `POST /v1/providers/health` — sends raw UDP SIP OPTIONS to `hostname:5060`
- SIP OPTIONS logic: `pkg/siphandler/health.go` — always returns a `HealthCheckResult`, never a Go error
- Any SIP response (any code) = healthy; timeout/network error = unhealthy
- No Redis, no database, no Kubernetes annotations — purely RabbitMQ + raw UDP
- SIP timeout configurable via `SIP_TIMEOUT` env var (Go duration string, default `5s`)
- Logging: `sirupsen/logrus` + `joonix` formatter, debug level by default

## Adding a new RPC endpoint

1. Define request/response structs in `pkg/listenhandler/request/`
2. Add regex pattern in `pkg/listenhandler/main.go`
3. Implement handler method on `listenHandler`
4. Add case to `processRequest()` switch
5. Write tests alongside the handler file
