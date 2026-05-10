# bin-sentinel-manager — Domain

## Domain Entities

### Pod event

The primary output unit. When a Kubernetes pod in a watched namespace changes state or is removed, sentinel wraps the `corev1.Pod` object into a RabbitMQ message with one of two event types:

| Event type constant | Value | Meaning |
|--------------------|-------|---------|
| `pod.EventTypePodUpdated` | `pod_updated` | Pod spec or status changed (phase transition, container restart, readiness flip) |
| `pod.EventTypePodDeleted` | `pod_deleted` | Pod was removed from the namespace |

The event payload is the verbatim Kubernetes `Pod` object, giving consumers full access to pod labels, annotations, phase, and container statuses.

### Watched pod classes

Sentinel's label-selector configuration determines which Asterisk pod types are monitored. The current selector set covers:

| Pod label (`app=`) | Meaning |
|--------------------|---------|
| `asterisk-call` | Asterisk instance handling SIP/RTP for individual calls |

Other Asterisk workloads (conference, registrar) can be added to the `selectors` map at startup without code changes.

### Namespace scope

Sentinel watches pods in the `voip` namespace where Asterisk pods run, but itself is deployed in the `bin-manager` namespace. The distinction matters for RBAC: the `pod-reader` Role must be granted in the `voip` namespace, not in `bin-manager`.

## Key Business Rules

1. **Add events are ignored.** When the informer starts, Kubernetes delivers all existing pods as Add events. Sentinel discards these because the pod may not yet be fully initialized and downstream services have their own state for live pods.

2. **Every update is forwarded.** Sentinel does not filter pod updates by phase or container state. Consumers decide which transitions are relevant. This keeps sentinel's logic minimal and avoids hard-coding phase semantics.

3. **Delete is authoritative.** A `EventTypePodDeleted` event is the signal that an Asterisk pod no longer exists. Downstream services (call-manager, conference-manager) use this to detect unexpectedly terminated call legs and initiate cleanup — for example, hanging up active calls whose media server has disappeared.

4. **No resync.** The informer is created with `resyncPeriod = 0`. Sentinel relies on Kubernetes's change-detection rather than periodic full reconciliation. This avoids flooding downstream services with spurious re-notifications.

5. **In-cluster only.** Authentication is via `rest.InClusterConfig()`. The service cannot run outside a Kubernetes cluster. Local development requires a kubeconfig workaround or a test cluster.

6. **RBAC gating.** If the pod-reader RoleBinding is missing or misconfigured, the informer's List call fails at startup. The service exits rather than silently watching nothing — fail-fast behaviour is intentional.
