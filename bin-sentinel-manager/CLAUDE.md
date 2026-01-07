# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-sentinel-manager is a Kubernetes pod monitoring service for VoIP infrastructure. It watches pod lifecycle events in Kubernetes namespaces and publishes notifications to RabbitMQ for other services to consume.

## Build and Test Commands

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

### Prometheus Metrics

The service exposes metrics at `/metrics` (port 2112 by default):
- `sentinel_manager_pod_state_change_total` - Counter with labels: namespace, pod app label, state (updated/deleted)

### Configuration

Environment variables / flags:
- `RABBITMQ_ADDRESS` - RabbitMQ connection (default: `amqp://guest:guest@localhost:5672`)
- `PROMETHEUS_ENDPOINT` - Metrics endpoint path (default: `/metrics`)
- `PROMETHEUS_LISTEN_ADDRESS` - Metrics server address (default: `:2112`)

### Kubernetes Deployment

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
