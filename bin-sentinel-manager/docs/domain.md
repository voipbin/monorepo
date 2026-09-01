# bin-sentinel-manager — Domain

## Domain Entities

### Container event (`models/container`)

The primary output unit. When a watched Asterisk container starts or dies, sentinel publishes a RabbitMQ message with one of two event types:

| Event type constant | Value | Meaning |
|--------------------|-------|---------|
| `container.EventTypeContainerStarted` | `container_started` | A watched container entered the running state |
| `container.EventTypeContainerDied` | `container_died` | A watched container's main process exited |

The payload is deliberately minimal — three fields, no Docker object embedded:

| Field | JSON | Meaning |
|---|---|---|
| `ContainerName` | `container_name` | Compose-generated name, e.g. `voip-asterisk-call-docker-2` |
| `Service` | `service` | Logical workload: `asterisk-call`, `asterisk-conference`, or `asterisk-registrar` |
| `AsteriskID` | `asterisk_id` | The instance's asterisk-id, resolved before the death; `""` when it was never resolvable |

This replaces the former `models/pod` package, whose payload was a verbatim Kubernetes `corev1.Pod` and whose consumers had to dig through `metadata.namespace`, `metadata.labels["app"]`, and `metadata.annotations["asterisk-id"]`. Docker has no equivalent object, and the recovery contract only ever needed these three values.

`Event.EventSubscriptionID()` returns `AsteriskID` directly, making a resolved instance addressable on the global topic exchange. The old `pod.Event` returned `""` unconditionally because a Pod carried no top-level identity at all.

### Asterisk address (`models/asteriskaddress`)

The Redis key family `asterisk.<asterisk-id>.address-internal`, written by `voip-asterisk-proxy` and read (never written) by sentinel:

| Constant | Value | Meaning |
|---|---|---|
| `TTL` | 24h | The full time-to-live the proxy sets on every write |
| `RefreshInterval` | 5m | How often the proxy re-writes the key, restoring the full TTL |
| `FreshnessMargin` | 12m | How far below a full TTL a key may sit and still count as current |

`AsteriskAddress.IsFresh()` is `remaining TTL >= TTL - FreshnessMargin` (23h48m).

### Watched container classes

Determined by compile-time name prefix, not runtime configuration:

| Container name | `Service` |
|---|---|
| `voip-asterisk-call-docker-<N>` | `asterisk-call` |
| `voip-asterisk-conference-docker-<N>` | `asterisk-conference` |
| `voip-asterisk-registrar-docker-<N>` | `asterisk-registrar` |

The `<N>` must be a bare run of digits. This is what excludes the co-located `-proxy` sidecars, which share the prefix.

## Key Business Rules

1. **The asterisk-id is resolved before the death, never at it.** A dying container's inspect response has an empty `IPAddress`, and a reverse Redis scan at die time cannot distinguish "the id that just died" from "the id that just took over the same static IP". Resolution therefore runs continuously in the background; the `die` handler only reads what is already known.

2. **Freshness gates learning, never forgetting (sticky last-known).** The Redis key's 24h TTL is refreshed every 5 minutes, so a dead generation's key for an IP can coexist with the live generation's key for the same IP for up to 24h. Only a key within `24h - 12min` of full counts as evidence about the current occupant. But a pass that finds no fresh candidate leaves an already-resolved id **unchanged** — a stale scan is not evidence that a resolved id is wrong.

   This is correct by invariant, not by heuristic: the asterisk-id derives from the container's MAC, which is fixed for that container object's whole lifetime, and one table entry spans exactly one container generation. The true id is therefore *constant* over an entry's life. "Learn once, never unlearn" cannot go wrong, and cannot go stale-forever either, because a resolved id is consumed at most once — by the `die` that deletes its entry.

3. **Entry creation always starts unresolved.** Stickiness governs updates within a generation. A same-name replacement container starts from `AsteriskID: ""` so it can never inherit the dead generation's id.

4. **An id already bound to a different live container is excluded.** In the pathological case where two generations' keys for one IP are both inside the freshness window, this narrows the ambiguity. It does not eliminate it: a same-second overlap remains a documented residual, and an ambiguous pass falls back to sticky (learn nothing).

5. **A container recreation gets a new asterisk-id.** The `production` Docker network assigns MACs randomly on container creation and no compose file pins one, so sentinel cannot assume a stable id-to-container mapping. This is the entire reason the state table exists.

6. **Flap damping.** Past 3 deaths of one container inside 60 seconds, later deaths in that window are not published. A crash-looping container is a symptom to alert on, not something to keep redialing calls against.

7. **An unresolved id is published, not suppressed.** A container that died before its first successful resolution publishes with `asterisk_id: ""`. `bin-call-manager` guards on that and skips recovery — there is genuinely no prior channel history to recover for a container that never registered with Redis. The `sentinel_manager_container_unresolved_asterisk_id_total` counter is the only signal that this happened.

8. **Fail loud, never watch nothing.** An unreachable socket proxy at startup exits the process rather than degrading to an idle watcher. Komodo's health monitoring surfaces the resulting crash-loop.

9. **Read-only, everywhere.** Sentinel never writes to Redis and never issues a mutating Docker API call. The socket proxy enforces the latter independently of the code.
