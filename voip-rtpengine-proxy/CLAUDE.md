# voip-rtpengine-proxy

Service class: **A+sub** — Go RPC proxy co-located with an RTPEngine media proxy daemon. Bridges RTPEngine NG protocol commands with the VoIPbin RabbitMQ message bus, and manages tcpdump-based call recording.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Detailed documentation

- [docs/architecture.md](docs/architecture.md) — component overview, layer responsibilities, request routing
- [docs/domain.md](docs/domain.md) — domain entities, key business rules
- [docs/dependencies.md](docs/dependencies.md) — monorepo and external dependencies
- [docs/operations.md](docs/operations.md) — failure modes, debugging guide, configuration, Prometheus metrics
- [docs/subsystems.md](docs/subsystems.md) — RTPEngine daemon overview, NG protocol, recording, deployment notes

## Common commands

```bash
# Build
go build -o bin/rtpengine-proxy ./cmd/rtpengine-proxy

# Test
go test ./...

# Generate mocks
go generate ./...

# Full verification (required before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Key implementation facts

- Entry point: `cmd/rtpengine-proxy/main.go`; configuration in `cmd/rtpengine-proxy/init.go`
- Instance identity: IPv4 address of `--interface_name` (default `eth0`); stored in Redis as `rtpengine.<ip>.address-internal`, refreshed every 5 min
- Single RPC endpoint: `POST /v1/commands` — dispatches by `type` field: `ng`, `exec`, or `kill`
- NG client: `pkg/ngclient/` — UDP/bencode, cookie-based request correlation, configurable timeout
- Process manager: `pkg/processmanager/` — manages tcpdump processes, enforces UUID IDs and command allowlist, 20-process cap, 20-min safety timeout
- GCS uploader: `pkg/gcsuploader/` — uploads pcap files; used by process manager on kill and by pcap watcher
- Pcap watcher: `pkg/pcapwatcher/` — watches `RTPENGINE_RECORDING_DIR` and uploads closed files to GCS; disabled if env vars unset
- Mocks: `go:generate` in `pkg/ngclient/main.go`, `pkg/processmanager/main.go`, `pkg/gcsuploader/main.go`
- Logging: `sirupsen/logrus` + `joonix` formatter, debug level by default

## Adding a new command type

1. Add a new `Type` constant in `models/command/command.go`
2. Add a new case in `processCommandPost()` in `pkg/listenhandler/command.go`
3. Implement the handler method on `listenHandler`
4. Write unit tests for the new case in `pkg/listenhandler/command_test.go`
