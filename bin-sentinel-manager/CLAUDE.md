# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-sentinel-manager` is a Kubernetes pod monitoring service for VoIP infrastructure. It watches pod lifecycle events in Kubernetes namespaces and publishes notifications to RabbitMQ for other services to consume.

**Key Concepts:**
- **Pod informer**: Uses `k8s.io/client-go` with label selectors to watch pods scoped to specific apps (`asterisk-call`, `asterisk-conference`, `asterisk-registrar`).
- **In-cluster only**: Authenticates via `rest.InClusterConfig()`; cannot run outside a Kubernetes cluster.
- **Publish, don't react**: Emits `pod_updated` and `pod_deleted` events; downstream services (e.g., `bin-call-manager`) decide what to do with them.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-sentinel-manager`.

## Common Commands

```bash
# Build the sentinel-manager daemon
go build -o ./bin/ ./cmd/...

# Run the daemon (requires Kubernetes in-cluster config)
./bin/sentinel-manager

# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Run a single test
go test -v -run TestName ./pkg/packagename/...

# Generate mocks (uses go.uber.org/mock)
go generate ./pkg/monitoringhandler/...
```

## sentinel-control CLI Tool

A command-line tool for querying Kubernetes pod monitoring data. **All output is JSON format** (stdout), logs go to stderr.

```bash
# List monitored pods in a namespace - returns JSON array
./bin/sentinel-control pod list --namespace <namespace>

# Get a specific pod by name - returns pod JSON
./bin/sentinel-control pod get --namespace <namespace> --name <pod-name>
```

Uses same environment variables as sentinel-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Architecture

### Service Layer Structure

The service follows a simple monitoring pattern:

1. **cmd/sentinel-manager/** - Main daemon entry point with Cobra/Viper configuration
2. **pkg/monitoringhandler/** - Kubernetes pod informer and event handling
3. **models/pod/** - Pod event type constants

### Kubernetes Monitoring Pattern

bin-sentinel-manager runs as an in-cluster Kubernetes service that:
- Uses `k8s.io/client-go` to create pod informers with label selectors
- Watches pods in configured namespaces (currently `voip` namespace)
- Monitors specific pod labels (`app=asterisk-call`, `app=asterisk-conference`, `app=asterisk-registrar`)
- Publishes events to RabbitMQ queue `QueueNameSentinelEvent` when pods are updated or deleted
- Exports Prometheus metrics tracking pod state changes

**Key Implementation Details:**
- Uses `cache.SharedIndexInformer` from `k8s.io/client-go/tools/cache`
- Each namespace/label selector pair runs in its own goroutine
- AddFunc intentionally empty (not used for registration events)
- UpdateFunc handles all pod state changes
- DeleteFunc handles pod deletions
- No resync period (set to 0) for efficiency

### Inter-Service Communication

- Uses RabbitMQ for message passing between microservices
- Publishes events via `notifyHandler.PublishEvent()` with event types:
  - `pod.EventTypePodUpdated` - When pod state changes
  - `pod.EventTypePodDeleted` - When pod is removed
- **Monorepo structure**: All sibling services are referenced via `replace` directives in go.mod pointing to `../bin-*-manager` directories

## Request Routing

N/A — `bin-sentinel-manager` is an event publisher only. It does not process RPC requests. There is no listenhandler.

## Event Subscriptions

This service does not subscribe to RabbitMQ events. Its inputs come from the Kubernetes API via informers; outputs are published events.

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared utilities (RabbitMQ via `notifyhandler`, models)

Always run `go mod vendor` after changing dependencies.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces co-located with handlers (`mock_*.go`)
- Table-driven tests with struct slices

```go
tests := []struct {
    name      string
    input     InputType
    mockSetup func(*MockHandler)
    expectRes ResultType
    expectErr bool
}{
    {"success case", input1, setupMock1, expected1, false},
    {"error case", input2, setupMock2, nil, true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        // test implementation
    })
}
```

## Key Implementation Details

### SharedIndexInformer
Uses `cache.SharedIndexInformer` from `k8s.io/client-go/tools/cache`. Each (namespace, label-selector) pair runs in its own goroutine. `AddFunc` is intentionally empty (registration events aren't useful here); `UpdateFunc` handles state changes; `DeleteFunc` handles removals. No resync period (set to 0) for efficiency.

### RBAC Requirements
Requires a `pod-reader` Role with `get`/`list`/`watch` verbs on pods in the target namespace, bound via RoleBinding to the service account.

## Configuration

Environment variables / flags:

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server | `amqp://guest:guest@localhost:5672` |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

## Prometheus Metrics

Service exposes metrics at `/metrics` (port 2112 by default):
- `sentinel_manager_pod_state_change_total` — counter (labels: `namespace`, pod `app` label, `state` ∈ `updated`/`deleted`)

## Kubernetes Deployment

The service requires specific RBAC permissions to watch pods:
- **Role**: `pod-reader` in target namespace (`voip`) with verbs: `get`, `list`, `watch` on pods
- **RoleBinding**: Links the pod-reader role to the service account
- **In-cluster config**: Uses `rest.InClusterConfig()` to authenticate with Kubernetes API

Deployment is configured in `k8s/deployment.yml` with:
- Single replica
- Prometheus scrape annotations
- Resource limits: 30m CPU, 20Mi memory
- Runs in `bin-manager` namespace
- Watches pods in `voip` namespace
