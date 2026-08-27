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

### Global topic exchange (VOIP-1404 / VOIP-1405)

`cmd/sentinel-manager/main.go` — the service's only NotifyHandler construction site — constructs
its NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`. Every event is therefore published
twice: once to the per-service fanout exchange `bin-manager.sentinel-manager.event` (unchanged,
still the system of record) and once to the global topic exchange `bin-manager.event`. A topic
publish failure never propagates to the caller and never affects the fanout publish.

**sentinel-manager is the one documented placeholder-by-design publisher.** The published payload
is the raw `*corev1.Pod` handed over by the informer, and a `corev1.Pod` carries no top-level `id`
in its JSON form (its identity lives under `metadata`). The subscription-address segment therefore
collapses to the `-` placeholder for every event:

| Event | Routing key |
|-------|-------------|
| `pod_updated` | `sentinel-manager.pod.-.updated` |
| `pod_deleted` | `sentinel-manager.pod.-.deleted` |

Consequences, all intentional (design §2.4):

- **Instance subscription of pod events is not supported.** Consumers bind at the type level
  (`sentinel-manager.pod.#`) and filter on the payload.
- **`sentinel_manager_topic_placeholder_total` grows in step with every publish.** For this service
  the healthy invariant is `placeholder_total ≈ topic_publish_total{result="ok"}` — that is not an
  alert condition, but the ratio still detects publish regressions and must not be ignored outright.
- **Do not attach a subscription-id override to `*corev1.Pod`** (external type) and do not wrap the
  payload in `models/pod.Pod` at the publish site — that would change the payload shape for every
  existing fanout consumer.

`pod_added` is declared in `models/pod` but never published: the informer's `AddFunc` is an
intentional no-op. The golden table pinning both keys is `models/pod/routingkey_golden_test.go`;
the schema is defined in monorepo
`docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md`.
