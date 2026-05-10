# bin-sentinel-manager — Architecture

Service class: **A2** (event-driven worker, no inbound RPC).

## Component Overview

`bin-sentinel-manager` is an in-cluster Kubernetes pod monitoring daemon. It watches the lifecycle of Asterisk pods (call, conference, registrar) using the Kubernetes informer framework and publishes state-change events to RabbitMQ so downstream services can react to pod restarts, crashes, or deletions.

```
Kubernetes API Server
        │  (list/watch pods)
        ▼
 monitoringhandler
  ┌─────────────────────────────┐
  │  SharedIndexInformer        │
  │  (one goroutine per         │
  │   namespace × label pair)   │
  │                             │
  │  AddFunc    → no-op         │
  │  UpdateFunc → runPodUpdated │
  │  DeleteFunc → runPodDeleted │
  └──────────┬──────────────────┘
             │ PublishEvent
             ▼
        RabbitMQ
  (QueueNameSentinelEvent)
             │
             ▼
   Downstream consumers
   (bin-call-manager, etc.)
```

There is no HTTP server, no listenhandler, and no inbound RPC queue. All inputs come from the Kubernetes watch stream; all outputs are RabbitMQ events.

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|----------------|
| Entry point | `cmd/sentinel-manager/` | Parse config (Cobra/Viper), authenticate in-cluster, start monitoringhandler |
| Core monitor | `pkg/monitoringhandler/` | Create `SharedIndexInformer` per namespace/selector, dispatch pod events, increment Prometheus counter |
| Domain types | `models/pod/` | Event type constants (`EventTypePodUpdated`, `EventTypePodDeleted`) |
| CLI tool | `pkg/sentinel-control/` | Query pod state for debugging (JSON output, uses same env vars) |

## Execution Model

### What triggers sentinel

Sentinel does not subscribe to RabbitMQ events. Its sole input source is the Kubernetes API watch stream, obtained via `rest.InClusterConfig()`. At startup it creates one `SharedIndexInformer` goroutine for each `(namespace, label-selector)` pair:

| Namespace | Label selector | Pods covered |
|-----------|---------------|--------------|
| `voip` | `app=asterisk-call` | Call-leg Asterisk pods |

Additional selectors can be added by extending the `selectors` map passed to `monitoringHandler.Run()`.

### What it does when triggered

- **AddFunc**: intentionally a no-op. Kubernetes delivers existing pods during the initial list as Add events; these are ignored because the pod may not be fully initialized.
- **UpdateFunc**: calls `runPodUpdated(ctx, pod)` — logs the event, publishes `EventTypePodUpdated` to RabbitMQ, increments the `pod_state_change_total` counter with `state=updated`.
- **DeleteFunc**: calls `runPodDeleted(ctx, pod)` — logs the event, publishes `EventTypePodDeleted` to RabbitMQ, increments the counter with `state=deleted`.

Context cancellation propagates via a `stopCh` channel into `podInformer.Run(stopCh)`, ensuring all goroutines shut down cleanly.

### What it produces

All outputs are RabbitMQ events published to `QueueNameSentinelEvent`. The payload is the full `corev1.Pod` struct serialized by `notifyhandler.PublishEvent`. Downstream consumers (e.g., `bin-call-manager`) match pods to active calls and perform cleanup when an Asterisk pod disappears unexpectedly.
