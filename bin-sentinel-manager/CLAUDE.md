# bin-sentinel-manager

Service class: **A2** — Docker container lifecycle monitor. No inbound RPC. No HTTP server. Publishes container lifecycle events to RabbitMQ.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Detailed documentation

- [docs/architecture.md](docs/architecture.md) — component overview, layer responsibilities, execution model
- [docs/domain.md](docs/domain.md) — domain entities, key business rules
- [docs/operations.md](docs/operations.md) — failure modes, debugging guide, configuration, Prometheus metrics

## CRITICAL: never mount the raw Docker socket

`/var/run/docker.sock` grants root-equivalent host access. This service talks **only** to a read-only `docker-socket-proxy` sidecar (`DOCKER_SOCKET_PROXY_ADDRESS`), declared in [komodo/docker-compose.yml](komodo/docker-compose.yml) on a private `internal: true` network shared with nothing else. The proxy grants exactly `EVENTS`, `CONTAINERS`, `PING`, `VERSION`; every other API family is explicitly denied.

Do not add a Docker API call to `dockerwatchhandler`'s `dockerClient` interface without checking it against that ACL — a call the proxy denies fails at runtime, not at compile time.

## CRITICAL: fail loud, never watch nothing

If the socket proxy is unreachable at startup, `dockerwatchhandler.Run` returns an error and the process exits (Komodo surfaces the crash-loop). A sentinel that *looks* up but watches nothing is worse than one that is visibly down — never downgrade that to a warning-and-continue.

## CRITICAL: the asterisk-id is resolved before the death, never at it

A dying container's `inspect` response has an empty `IPAddress`, and a reverse Redis scan at die time cannot tell "the id that just died" from "the id that just took over the same static IP". So `pkg/dockerwatchhandler` keeps an in-memory state table that resolves each watched container's asterisk-id continuously, and the `die` handler only *reads* it.

Two rules in that table are load-bearing and were the focus of five design-review rounds:

1. **Sticky last-known.** A refresh pass that finds no fresh candidate for an entry's IP leaves that entry's `AsteriskID` unchanged. Freshness gates *learning*, never *forgetting*. Regressing a resolved id to `""` would, combined with call-manager's empty-id guard, silently skip the exact recovery this service exists to trigger. `stateTable.Resolve` refuses an empty id structurally, so no code path can regress one.
2. **Entry creation always starts unresolved.** Stickiness governs updates *within one container generation*. A same-name replacement container must not inherit the dead generation's id.

The freshness threshold is `remaining TTL >= 24h - 12min`, keyed to voip-asterisk-proxy's own 5-minute key-refresh cadence (not to sentinel's 10s loop).

## Common commands

```bash
# Build
go build -o ./bin/ ./cmd/...

# Test
go test ./...

# Full verification (required before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## sentinel-control CLI

Debugging tool. All output is JSON on stdout; logs go to stderr. It dials the same read-only socket proxy the service uses.

```bash
./bin/sentinel-control container list --name voip-asterisk-call-docker
./bin/sentinel-control container get  --name voip-asterisk-call-docker-1
```

## Key implementation facts

- Entry point: `cmd/sentinel-manager/`
- Core logic: `pkg/dockerwatchhandler/` — `boot.go` (startup reconciliation), `refresh.go` (10s Redis resolution loop), `events.go` (Docker Events stream), `state.go` (state table + flap tracker)
- Redis reads: `pkg/cachehandler/` — read-only reverse scan of `asterisk.*.address-internal`
- Event model: `models/container/` — `EventTypeContainerStarted`, `EventTypeContainerDied`
- Address model: `models/asteriskaddress/` — key parsing and the freshness rule
- Watched containers: compile-time name prefixes in `pkg/dockerwatchhandler/main.go` (`voip-asterisk-{call,conference,registrar}-docker-<N>`); the `-proxy` sidecars are excluded by requiring a bare replica index after the prefix
- Published via `notifyHandler.PublishEvent()` to `QueueNameSentinelEvent`, with `WithGlobalTopicPublish()`
- Prometheus counters: `sentinel_manager_container_state_change_total` (labels: `container_name`, `service`, `state`), `sentinel_manager_container_unresolved_asterisk_id_total`, `sentinel_manager_container_asterisk_id_refresh_miss_total`
- Deployment: `komodo/docker-compose.yml`, single replica, deployed by CircleCI's `bin-sentinel-manager-deploy` job
- `k8s/` is **dead** and intentionally left in place — its removal is deferred to a follow-up ticket (design §7). Do not delete it as drive-by cleanup.
- Testing: `go.uber.org/mock`, table-driven, mock files co-located (`mock_*.go`); `pkg/cachehandler` tests run against in-process `miniredis`
