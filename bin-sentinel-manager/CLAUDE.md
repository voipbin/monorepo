# bin-sentinel-manager

Service class: **A2** — container lifecycle monitor, with **two peer backends**. No inbound RPC. No HTTP server. Publishes container lifecycle events to RabbitMQ.

`SENTINEL_BACKEND` selects which backend runs: `docker` (Docker Events API, used by bm-nyc-01) or `kubernetes` (Kubernetes informers, used by self-hosted Kubernetes deployments). **There is no default and no auto-detection** — an unset or unknown value fails startup validation. Both backends publish the identical `container.Event` schema, so `bin-call-manager` has no idea which one produced a given event and never needs to.

VoIPBin is a self-hostable opensource CPaaS. Neither backend is a fallback for the other: a self-hoster running Kubernetes needs stranded-call detection exactly as much as bm-nyc-01 does. Do not treat the Kubernetes path as legacy or the Docker path as the "real" one.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Detailed documentation

- [docs/architecture.md](docs/architecture.md) — component overview, layer responsibilities, execution model
- [docs/domain.md](docs/domain.md) — domain entities, key business rules
- [docs/operations.md](docs/operations.md) — failure modes, debugging guide, configuration, Prometheus metrics

## CRITICAL: never mount the raw Docker socket (docker backend)

`/var/run/docker.sock` grants root-equivalent host access. This service talks **only** to a read-only `docker-socket-proxy` sidecar (`DOCKER_SOCKET_PROXY_ADDRESS`), declared in [komodo/docker-compose.yml](komodo/docker-compose.yml) on a private `internal: true` network shared with nothing else. The proxy grants exactly `EVENTS`, `CONTAINERS`, `PING`, `VERSION`; every other API family is explicitly denied.

Do not add a Docker API call to `dockerwatchhandler`'s `dockerClient` interface without checking it against that ACL — a call the proxy denies fails at runtime, not at compile time.

**The proxy ACL is path-prefix based, not per-endpoint, so `CONTAINERS=1` grants much more than the one inspect call this service makes.** Anything able to reach the proxy can, for *any* container on the host, read `/containers/{id}/json` (full config including every env-var secret), `/containers/{id}/archive` (arbitrary files out of the container filesystem), `/containers/{id}/export` (the whole filesystem), `/containers/{id}/logs`, and `/containers/{id}/attach/ws`. That is near-total read access to every container's data on bm-nyc-01 — not "env vars via inspect". It is read-only, so not host code execution, but on compromise it is close to full host data disclosure.

The image has no field-level or per-endpoint ACL, so **network scope is the entire mitigation**: the proxy lives on a Stack-local `internal: true` network whose only other member is this service. Never move it onto `production`, and never "consolidate" it into a shared proxy Stack, believing the exposure is env-vars-only.

## CRITICAL: pod-reader RBAC is required (kubernetes backend)

A `pod-reader` Role granting `get`/`list`/`watch` on pods in the `voip` namespace must be bound to the service account **before** deployment (`k8s/rbac/`). Without it the informer's initial `List` is denied and the process exits at startup once the sync deadline expires.

That last clause is load-bearing, and it is why `waitForCacheSync` is wrapped in a `context.WithTimeout`. A bare `cache.WaitForCacheSync` returns `false` only when its stop channel closes — under exactly this failure, client-go retries the denied `List` with backoff **forever**, so the bare call blocks and never returns. The pre-VOIP-1418 code claimed "RBAC required or the service exits" while actually hanging; the deadline is what finally makes that claim true. Never replace it with an unbounded wait.

## CRITICAL: three silent-failure traps in the kubernetes backend

Each of these fails with no panic, no error and no log — just a death event that never publishes, and calls that are never recovered. Each was caught by a separate design-review round as missing from a naive "restore the old code" approach.

1. **`UpdateFunc` must compare `oldPod.UID != newPod.UID`.** client-go only synthesizes a `Deleted` callback for keys *absent* from a relist's object set. A pod deleted and replaced under the same name while the watch was interrupted is still present in the relist, so it arrives as a `Replaced` delta through `UpdateFunc` and **no delete callback ever fires** for the dead generation. On a mismatch, publish `died` for the old pod *before* the new one's `started`.
2. **`DeleteFunc` must unwrap `cache.DeletedFinalStateUnknown` — never a bare type assertion.** A bare assertion panics on that shape. The reflexive fix (assert with `ok`, return on mismatch) is *worse than the panic*: it silently drops the death. The failure budget makes this path reachable **by design**, since its whole purpose is surviving the interruptions after which a tombstone appears.
3. **`AddFunc` stays a no-op, unconditionally.** Informers replay every existing pod as a synthetic Add on the initial list; publishing `started` there would misrepresent long-lived pods as freshly started on every sentinel restart. A genuinely new pod is still covered by its first `UpdateFunc`.

Two related prohibitions: **do not suppress no-op `UpdateFunc` invocations** (a relist legitimately re-publishes `started` for already-running pods; the consumer ignores `started`, and equality-based suppression would reintroduce exactly the identity-comparison fragility rule 1 exists to handle), and **do not "fix" `started` timing** — the K8s backend publishes it late and possibly repeatedly, which is an accepted, documented asymmetry with the Docker backend, not a gap.

## CRITICAL: fail loud, never watch nothing

This is the `MonitoringBackend` contract, and it binds **both** backends: `Run` returns `nil` **only** when `ctx` was cancelled. Any other cause of the watch stopping returns a non-nil error, `runService` propagates it, and `main` exits **non-zero** so Komodo reports a crash-loop rather than a container that exited successfully. A sentinel that *looks* up but watches nothing is worse than one that is visibly down — never downgrade any of this to a warning-and-continue, and never swallow the error on the way out of `runService`.

On the Kubernetes side the equivalent mechanism is `watchFailureBudget`: `SetWatchErrorHandler` fires on entirely benign conditions (apiserver rolling restart, `too old resource version`, a transient reset), so a single invocation is never fatal — only `maxConsecutiveWatchFailures` in a row with no intervening recovery is. Recovery is signalled two ways, and both are needed: any delivered event, and a changed `LastSyncResourceVersion` (the latter is the only signal available when the selector matches zero pods, where a healthy watch legitimately delivers nothing). Counted on `sentinel_manager_pod_watch_health_total{outcome="resynced"|"transient-error"|"fatal"}`.

On the Docker side the same rule covers proxy loss *after* boot, which is the easier one to get wrong: `runEventLoop` reconnects with a `since` cursor, but only up to `maxConsecutiveEmptyStreams` attempts that end *immediately* with no events (≈1 minute). Past that it returns an error into the same exit path.

Two things reset that budget, and both matter: an attempt that **delivered** events, and an attempt that delivered nothing but **survived** past `healthyStreamLifetime` (10× the reconnect delay). The second one is not optional — on a genuinely idle fleet nothing starts or dies for hours, so a long-lived eventless stream ending on a proxy restart is normal, and counting those would accumulate across days into a self-inflicted restart of a healthy system. Every attempt is counted on `sentinel_manager_container_event_stream_reconnect_total{result="delivered"|"idle"|"empty"}`. `empty` is the primary alertable signal, but a sustained high `idle` rate is worth an eye too: a *hung* proxy or dockerd that accepts the connection, holds it past the longevity threshold, then drops without ever streaming classifies as `idle` forever and never trips the give-up exit. The gap is bounded (the `since` cursor replays it on reconnect) and visible in the panel by result, but the counter alone will not fire on it.

## CRITICAL: the asterisk-id is resolved before the death, never at it (docker backend only)

None of this applies to the Kubernetes backend, and that asymmetry is the whole reason the K8s side is so much smaller: voip-asterisk-proxy self-patches its own pod's `asterisk-id` annotation through the Kubernetes API, and annotations are visible to any watcher with read RBAC. So there is no reverse lookup, no state table, no freshness filter and no stickiness — the id is read straight off the pod. An absent annotation (the pod died inside the proxy's patch window) publishes an empty id, which is the same degrade path below and is *expected* rather than exceptional.

On the Docker side, a dying container's `inspect` response has an empty `IPAddress`, and a reverse Redis scan at die time cannot tell "the id that just died" from "the id that just took over the same static IP". So `pkg/dockerwatchhandler` keeps an in-memory state table that resolves each watched container's asterisk-id continuously, and the `die` handler only *reads* it.

Three rules in that table are load-bearing and were the focus of five design-review rounds plus the code review that followed:

1. **Sticky last-known.** A refresh pass that finds no fresh candidate for an entry's IP leaves that entry's `AsteriskID` unchanged. Freshness gates *learning*, never *forgetting*. Regressing a resolved id to `""` would, combined with call-manager's empty-id guard, silently skip the exact recovery this service exists to trigger. `stateTable.Resolve` refuses an empty id structurally, so no code path can regress one.
2. **Entry creation always starts unresolved.** Stickiness governs updates *within one container generation*. A same-name replacement container must not inherit the dead generation's id.
3. **An id *change* on an already-resolved entry is rejected, not applied.** The id derives from a MAC that is fixed per container object, so this branch structurally cannot fire; if it does, the new value is no more trustworthy than the old one, and adopting it risks firing recovery against a different, still-live instance. The existing id is kept, logged at WARN, and counted on `sentinel_manager_container_asterisk_id_conflict_total` — the plausible real trigger is a missed die+start pair, in which case the kept id is the stale one, so this must be visible rather than silent.

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

- Entry point: `cmd/sentinel-manager/` — `buildBackend` is the one place the two backends diverge; everything else (RabbitMQ, notify handler, Prometheus server, fail-loud exit chain) is backend-agnostic
- Backend contract: `pkg/monitoringbackend/` — the one-method `MonitoringBackend` interface, plus the counters both backends share so one Grafana panel covers either deployment
- Docker backend: `pkg/dockerwatchhandler/` — `boot.go` (startup reconciliation), `refresh.go` (10s Redis resolution loop), `events.go` (Docker Events stream), `state.go` (state table + flap tracker)
- Kubernetes backend: `pkg/k8swatchhandler/` — `run.go` (informers, callbacks, errgroup fan-in), `budget.go` (consecutive watch-failure budget)
- Redis reads: `pkg/cachehandler/` — read-only reverse scan of `asterisk.*.address-internal`
- Event model: `models/container/` — `EventTypeContainerStarted`, `EventTypeContainerDied`
- Address model: `models/asteriskaddress/` — key parsing and the freshness rule
- Watched containers (docker): compile-time name prefixes in `pkg/dockerwatchhandler/main.go` (`voip-asterisk-{call,conference,registrar}-docker-<N>`); the `-proxy` sidecars are excluded by requiring a bare replica index after the prefix
- Watched pods (kubernetes): compile-time `watchTargets` in `pkg/k8swatchhandler/main.go` — namespace `voip`, selectors `app=asterisk-{call,conference,registrar}`. The `app` label is mapped to the typed `Service` constant through an explicit lookup; an unrecognized value is rejected at the publish boundary, never passed through
- Published via `notifyHandler.PublishEvent()` to `QueueNameSentinelEvent`, with `WithGlobalTopicPublish()`
- Prometheus counters, shared by both backends (`pkg/monitoringbackend`): `sentinel_manager_container_state_change_total` (labels: `container_name`, `service`, `state`), `sentinel_manager_container_unresolved_asterisk_id_total`
- Prometheus counters, docker backend only: `sentinel_manager_container_asterisk_id_refresh_miss_total`, `sentinel_manager_container_asterisk_id_conflict_total`, `sentinel_manager_container_event_stream_reconnect_total` (label: `result`)
- Prometheus counters, kubernetes backend only: `sentinel_manager_pod_watch_health_total` (labels: `namespace`, `selector`, `outcome`), `sentinel_manager_pod_died_detection_total` (label: `source` — `live`|`tombstone`|`uid-mismatch`|`unrecoverable`)
- Deployment (docker): `komodo/docker-compose.yml`, single replica, deployed by CircleCI's `bin-sentinel-manager-deploy` job. Sets `SENTINEL_BACKEND=docker` as a hardcoded literal
- Deployment (kubernetes): `k8s/*.yml` — **live again**, not dead. It sets `SENTINEL_BACKEND=kubernetes` and requires the `k8s/rbac/` role/rolebinding. An earlier revision of this file called `k8s/` dead and deferred its deletion; that was an artifact of the Docker-only stage of this work and is reversed
- Both deployment descriptors MUST carry a `SENTINEL_BACKEND` value. Startup validation has no default, so removing either line crash-loops that deployment
- Testing: `go.uber.org/mock`, table-driven, mock files co-located (`mock_*.go`); `pkg/cachehandler` tests run against in-process `miniredis`; `pkg/k8swatchhandler` tests run against client-go's `fake.NewClientset()`
- `k8s.io/*` is confined to `pkg/k8swatchhandler` and `cmd/sentinel-manager`. `models/container` in particular must stay free of it — `bin-call-manager` and `voip-kamailio-proxy` both reach this module through `replace` directives, and a k8s.io import in a shared model would drag that dependency into their build lists
