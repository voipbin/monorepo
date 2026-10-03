# bin-sentinel-manager — Architecture

Service class: **A2** (event-driven worker, no inbound RPC).

## Component Overview

`bin-sentinel-manager` is an Asterisk container lifecycle monitor with **two peer backends**. It watches either the Docker Engine's event stream (bm-nyc-01) or the Kubernetes API (self-hosted clusters) for the Asterisk workloads (call, conference, registrar) and publishes state-change events to RabbitMQ so downstream services can react to restarts, crashes, or removals.

`SENTINEL_BACKEND` picks the backend at startup — `docker` or `kubernetes`, no default, no auto-detection. Exactly one runs per process. VoIPBin is a self-hostable opensource CPaaS, so both deployment shapes are first-class; neither backend is a fallback for the other.

VOIP-1418 replaced the original Kubernetes-only implementation with the Docker backend after GKE was dismantled on 2026-08-20, then restored Kubernetes alongside it (design §8) once it became clear that a Docker-only service would silently regress stranded-call detection to zero for every self-hosted Kubernetes deployment.

```
cmd/sentinel-manager  --SENTINEL_BACKEND--> one of the two backends below
                                            (both satisfy monitoringbackend.MonitoringBackend)

=== SENTINEL_BACKEND=docker ===

        /var/run/docker.sock
                 │ (read-only bind mount)
                 ▼
      docker-socket-proxy  (sidecar, private `internal: true` network)
                 │  GET /events, /containers/json, /containers/{id}/json
                 ▼
        dockerwatchhandler
  ┌────────────────────────────────────────────┐
  │ boot.go     list+inspect running watched   │
  │             containers, seed state table   │
  │                                            │
  │ refresh.go  every 10s: SCAN Redis          │      Redis
  │             asterisk.*.address-internal, ◄──────  (written by
  │             freshness-filter, resolve ids  │      voip-asterisk-proxy)
  │                                            │
  │ events.go   consume start/die events,      │
  │             publish, flap-damp, reconnect  │
  │                                            │
  │ state.go    map[containerName]state        │
  └───────────────────┬────────────────────────┘
                      │ PublishEvent
                      ▼
                  RabbitMQ
          (global topic exchange
             `bin-manager.event`)
                      │
                      ▼
            Downstream consumers
            (bin-call-manager)
```

```
=== SENTINEL_BACKEND=kubernetes ===

      Kubernetes API server
                 │  list/watch pods (RBAC: pod-reader on `voip`)
                 ▼
        k8swatchhandler
  ┌────────────────────────────────────────────┐
  │ run.go      one SharedIndexInformer per     │
  │             (namespace, label-selector),    │
  │             fanned in through an errgroup   │
  │                                             │
  │             AddFunc    -> no-op             │
  │             UpdateFunc -> started           │
  │                           (+ died on a      │
  │                            UID mismatch)    │
  │             DeleteFunc -> died              │
  │                           (tombstone-aware) │
  │                                             │
  │ budget.go   consecutive watch-failure       │
  │             budget -> fatal                 │
  └───────────────────┬─────────────────────────┘
                      │ PublishEvent (identical container.Event)
                      ▼
                  RabbitMQ
```

The asterisk-id needs no resolution machinery on this side: voip-asterisk-proxy self-patches its own pod's `asterisk-id` annotation through the Kubernetes API, and annotations are visible to any watcher with read RBAC. The Docker backend's entire state table exists only because a container has no equivalent place to carry that id.

There is no HTTP server, no listenhandler, and no inbound RPC queue. All inputs come from the Docker event stream and Redis; all outputs are RabbitMQ events.

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|----------------|
| Entry point | `cmd/sentinel-manager/` | Parse and validate config, build the selected backend's dependencies (`buildBackend`), run it |
| Backend contract | `pkg/monitoringbackend/` | The one-method `MonitoringBackend` interface, plus the counters both backends share |
| Kubernetes monitor | `pkg/k8swatchhandler/` | Pod informers, callback-to-event mapping, watch-failure budget, errgroup fan-in |
| Core monitor | `pkg/dockerwatchhandler/` | Boot reconciliation, asterisk-id state table, Docker event stream, publish, Prometheus counters |
| Cache access | `pkg/cachehandler/` | Read-only reverse scan of `asterisk.<id>.address-internal` (value + remaining TTL) |
| Domain types | `models/container/` | `Event` payload, `EventTypeContainerStarted` / `EventTypeContainerDied` |
| Domain types | `models/asteriskaddress/` | Redis key shape and the freshness rule |
| CLI tool | `cmd/sentinel-control/` | Query container state for debugging (JSON output, same proxy endpoint) |

## Execution Model

### Backend selection

`internal/config` validates `SENTINEL_BACKEND` at startup and rejects an unset or unknown value outright. Validation is backend-conditional: `DOCKER_SOCKET_PROXY_ADDRESS` and `REDIS_ADDRESS` are required only for `docker`, and nothing extra is required for `kubernetes` beyond the in-cluster service-account mount that `rest.InClusterConfig()` reads. Validating all fields unconditionally would reject a perfectly good Kubernetes deployment for not supplying Redis config it has no reason to have.

Both `cmd/sentinel-manager` and `cmd/sentinel-control` enforce this. `InitConfig` (used by `cmd/sentinel-manager`) binds its own flags and then calls `LoadGlobalConfig` directly — the two binaries share one validation implementation structurally, not just by parallel test coverage. `LoadGlobalConfig` reloads and re-validates from viper on every call, with no caching, so a failed attempt is retryable rather than poisoning all future calls; it returns an error specifically so the CLI can fail the same way the service does.

### What triggers sentinel (docker backend)

Sentinel does not subscribe to RabbitMQ events. On the Docker backend its inputs are the Docker Events API stream and periodic Redis reads.

Watched containers are matched by compile-time name prefix, plus a bare replica index, which is what excludes the co-located `-proxy` sidecars:

| Container name pattern | `Service` value |
|---|---|
| `voip-asterisk-call-docker-<N>` | `asterisk-call` |
| `voip-asterisk-conference-docker-<N>` | `asterisk-conference` |
| `voip-asterisk-registrar-docker-<N>` | `asterisk-registrar` |

`voip-asterisk-call-docker-1-asterisk-call-proxy-1` shares the prefix but has no bare replica index after it, so it is not watched.

### What it does when triggered

**At startup (boot reconciliation).** `GET /containers/json` lists the running containers; each watched one is inspected once to resolve its IP and seeded into the state table with an unresolved asterisk-id. One immediate refresh pass then runs before the event loop starts. Without this, a sentinel restart would leave every already-running container untracked until its *next* recreation — an indefinite blind spot, not a bounded one, since sentinel runs single-replica.

**Every 10 seconds (refresh loop).** `SCAN asterisk.*.address-internal`, read each key's value (an IP) and its remaining TTL, and resolve each table entry's asterisk-id from the key matching its IP. Only a key whose remaining TTL is within `24h - 12min` counts, and an id already bound to a different live container is excluded. A pass that learns nothing leaves resolved ids untouched.

**On `start`.** One inspect resolves the container's IP (safe: the container is freshly running). A new table entry is created with an unresolved id, and `container_started` is published.

**On `die`.** The table entry is read and deleted in one critical section, and `container_died` is published carrying whatever id had been resolved. Sentinel never inspects or scans at die time — a dead container's inspect response has an empty IP address, and a die-time reverse scan cannot distinguish the id that just died from the one that just took over the same static IP.

**Flap damping.** Past 3 deaths of the same container inside 60 seconds, further deaths in that window are logged at WARN and not published: repeatedly firing recovery against a crash-looping container would spam Homer/PJSIP for channels that likely never established.

**Reconnect.** On any event-stream disconnect the loop pauses briefly and re-opens with `since=<last processed event + 1ns>`, so a proxy restart or network blip leaves a bounded gap rather than silently dropping a `die`.

### What triggers sentinel (kubernetes backend)

One `SharedIndexInformer` per `(namespace, label-selector)` pair — namespace `voip`, selectors `app=asterisk-call`, `app=asterisk-conference`, `app=asterisk-registrar` — all fanned in through an `errgroup` so any one informer's failure propagates out of `Run` instead of leaving the others running with silently reduced coverage.

The pod's `app` label is mapped to the typed `Service` constant through an explicit lookup; an unrecognized value is rejected at the publish boundary rather than passed through, because a raw passthrough would produce an event `bin-call-manager`'s filter silently never matches.

**Callback mapping.** `UpdateFunc` publishes `started`; `DeleteFunc` publishes `died`; `AddFunc` is an intentional no-op (informers replay existing pods as synthetic Adds on the initial list). Two additions make this safe rather than merely restored:

- `UpdateFunc` compares `oldPod.UID != newPod.UID` and publishes a `died` for the old generation first. A same-name pod replaced during a watch interruption is still present in the relist, so client-go delivers it as a `Replaced` delta and **never fires a delete callback** for the dead generation.
- `DeleteFunc` unwraps `cache.DeletedFinalStateUnknown`, which is how a deletion missed during an interruption arrives on the next relist.

**Accepted asymmetry with the Docker backend.** Because `AddFunc` is a no-op and a relist re-delivers unchanged objects through `UpdateFunc`, `started` is published *late* (on a new pod's first post-creation status transition, not at the creation instant) and *possibly more than once*. The identical-field-shape guarantee holds; `started` timing and cardinality are not part of it. Harmless today since `bin-call-manager` only consumes `died`, but a future consumer relying on `started` semantics needs to know this up front.

**Fail-loud.** `SetWatchErrorHandler` feeds a consecutive-failure budget rather than being treated as fatal on the first invocation (it fires on benign conditions like an apiserver rolling restart). Recovery resets it, signalled either by a delivered event or by a changed `LastSyncResourceVersion` — the latter matters because a selector matching zero pods delivers nothing even when perfectly healthy. `WaitForCacheSync` runs under an explicit deadline, without which a missing `pod-reader` RBAC role makes client-go retry the denied list forever and the call simply blocks.

### What it produces

RabbitMQ events published through `notifyhandler.PublishEvent`. The payload is `models/container.Event`:

```json
{"container_name":"voip-asterisk-call-docker-2","service":"asterisk-call","asterisk_id":"3e:50:6b:43:bb:32"}
```

Identical on both backends. On Kubernetes, `container_name` is the pod name and `asterisk_id` comes from the pod's `asterisk-id` annotation; on Docker they come from the container name and the Redis-resolved state table. `bin-call-manager` consumes `container_died`, filters on `service == "asterisk-call"`, and calls `RecoveryStart(asterisk_id)` — which pulls that instance's last 24h of channels, looks up each one's SIP dialog via Homer, and PJSIP-redials it onto a live instance. An event with an empty `asterisk_id` is dropped by call-manager's empty-id guard.

### Global topic exchange (VOIP-1404 / VOIP-1405 / VOIP-1407 / VOIP-1418)

`cmd/sentinel-manager/main.go` — the service's only NotifyHandler construction site — constructs its NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`. Every event is published to the global topic exchange `bin-manager.event`; since VOIP-1407 this is the sole publish path, and a topic publish failure propagates to the caller as an error.

**Since VOIP-1418, sentinel-manager is no longer a placeholder-by-design publisher.** The old payload was a raw `*corev1.Pod`, which carries no top-level `id`, so every key used the `-` placeholder. `container.Event.EventSubscriptionID()` now returns the resolved asterisk-id:

| Event | Routing key |
|-------|-------------|
| `container_started` | `sentinel-manager.container.-.started` (a fresh container has no resolved id yet) |
| `container_died` (resolved) | `sentinel-manager.container.<asterisk-id>.died` |
| `container_died` (unresolved) | `sentinel-manager.container.-.died` |

Consequences:

- **Instance subscription of container events is now possible.** Nothing binds that way today; `bin-call-manager` binds the wildcard `sentinel-manager.container.*.died`.
- **`sentinel_manager_topic_placeholder_total ≈ topic_publish_total{ok}` is no longer the healthy invariant.** It was, when every pod event was a placeholder. Now the placeholder is expected for every `container_started` and for a `container_died` whose id never resolved. A *rising* placeholder rate on `died` events specifically is a degraded state — cross-check `sentinel_manager_container_unresolved_asterisk_id_total`, which counts exactly that.

The golden table pinning these keys is `models/container/routingkey_golden_test.go`; the schema is defined in monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md`. The VOIP-1418 rename (`pod`/`updated`/`deleted` → `container`/`started`/`died`) was a deliberate, reviewed key-string change, safe because sentinel's only real consumer was updated in the same change; treat any future diff in that golden table as a regression unless equally deliberate.
