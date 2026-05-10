# voip-asterisk-proxy

Service class: **A+sub** — Go RPC proxy co-located with an Asterisk PBX daemon. Bridges Asterisk ARI/AMI with the VoIPbin RabbitMQ message bus.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Detailed documentation

- [docs/architecture.md](docs/architecture.md) — component overview, layer responsibilities, request routing
- [docs/domain.md](docs/domain.md) — domain entities, key business rules
- [docs/dependencies.md](docs/dependencies.md) — monorepo and external dependencies
- [docs/operations.md](docs/operations.md) — failure modes, debugging guide, configuration, Prometheus metrics
- [docs/subsystems.md](docs/subsystems.md) — Asterisk daemon overview, ARI/AMI config, deployment notes

## Common commands

```bash
# Build
go build -o bin/asterisk-proxy ./cmd/asterisk-proxy

# Test
go test ./...

# Generate mocks
go generate ./...

# Full verification (required before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Key implementation facts

- Entry point: `cmd/asterisk-proxy/main.go`; configuration in `cmd/asterisk-proxy/init.go`
- Three concurrent handlers: `EventHandler` (ARI/AMI → RabbitMQ events), `ListenHandler` (RabbitMQ → ARI/AMI RPC), `ServiceHandler` (GCS recording uploads)
- Instance identity: MAC address of `--interface_name` (default `eth0`); stored in Redis and Kubernetes pod annotation `asterisk-id`
- Routing patterns in `pkg/listenhandler/main.go`: `^/ari/`, `^/ami/`, `^/proxy/recording_file_move$`
- ARI and AMI connections auto-reconnect every 1 second on failure
- Kubernetes annotation patching can be disabled with `--kubernetes_disabled=true`
- Mocks: `go.uber.org/mock`, `go:generate` directive in `pkg/servicehandler/main.go`
- Logging: `sirupsen/logrus` + `joonix` formatter, debug level by default

## Adding a new proxy endpoint

1. Define request/response structs in `pkg/listenhandler/request/`
2. Add regex pattern in `pkg/listenhandler/main.go`
3. Implement handler method (e.g., in `pkg/listenhandler/proxy_handler.go`)
4. Add case to `processRequest()` switch
5. Write tests in `pkg/listenhandler/proxy_handler_test.go`
