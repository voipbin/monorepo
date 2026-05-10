# bin-sentinel-manager

Service class: **A2** — in-cluster Kubernetes pod monitor. No inbound RPC. No HTTP server. Publishes pod lifecycle events to RabbitMQ.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Detailed documentation

- [docs/architecture.md](docs/architecture.md) — component overview, layer responsibilities, execution model
- [docs/domain.md](docs/domain.md) — domain entities, key business rules
- [docs/operations.md](docs/operations.md) — failure modes, debugging guide, configuration, Prometheus metrics

## CRITICAL: in-cluster only

`rest.InClusterConfig()` is the only supported auth mechanism. The service **cannot** run outside a Kubernetes pod without a kubeconfig shim. CI tests that need Kubernetes integration require a real or emulated cluster.

## CRITICAL: RBAC required

A `pod-reader` Role with `get`/`list`/`watch` on pods in the `voip` namespace must be bound to the service account **before** deployment, or the informer will fail at startup and the service will exit.

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

Debugging tool. All output is JSON on stdout; logs go to stderr.

```bash
./bin/sentinel-control pod list --namespace <namespace>
./bin/sentinel-control pod get  --namespace <namespace> --name <pod-name>
```

## Key implementation facts

- Entry point: `cmd/sentinel-manager/`
- Core logic: `pkg/monitoringhandler/` — one `SharedIndexInformer` goroutine per `(namespace, label-selector)` pair
- Event types: `models/pod/` — `EventTypePodUpdated`, `EventTypePodDeleted`
- `AddFunc` is intentionally a no-op; `resyncPeriod = 0`
- Published via `notifyHandler.PublishEvent()` to `QueueNameSentinelEvent`
- Prometheus counter: `sentinel_manager_pod_state_change_total` (labels: `namespace`, `pod`, `state`)
- Kubernetes deployment: `k8s/deployment.yml` — single replica, `bin-manager` namespace, watches `voip` namespace
- Testing: `go.uber.org/mock`, table-driven, mock files co-located (`mock_*.go`)
