# VOIP-1418: bin-sentinel-manager Docker-backend design

Status: **DESIGN APPROVED** (round 4 + round 5, 2 consecutive Approve — CLAUDE.md design
review loop satisfied). Round 5's 3 non-blocking nits (orphaned §3.4 cross-reference,
ambiguous "refresh interval" wording, §3.2's "start-only" inspect claim) and the §3.5/§6
open question (event wire-string rename) are folded in below. Ready for implementation
planning.

## 1. Confirmed current state (2026-09-01, live-verified)

- `bin-sentinel-manager` is K8s-only: `rest.InClusterConfig()` is the sole auth path
  (`pkg/monitoringhandler/run.go`), no interface seam for a second backend, no `docker`
  dependency anywhere in the monorepo. GKE was fully decommissioned 2026-08-20; sentinel has
  had no deploy target since (CI's `bin-sentinel-manager-release` job was removed in
  VOIP-1405; test/build stay as quality gates). VOIP-1377 (deployment-target decision) and
  VOIP-1335 (dual-backend implementation attempt, put on hold 2026-08-20) are both closed;
  this ticket is their re-scoped successor per the CEO's 2026-08-28 decision: "sentinel is
  a needed service, not for retirement."
- **Corrected topology finding** (this design's own investigation superseded an earlier,
  wrong "single instance" conclusion reached from a `ListStacks` API read that turned out to
  be unreliable in this Komodo deployment): `voip-asterisk-{call,conference,registrar}-docker`
  each run as **2 live healthy instances** on bm-nyc-01 (`-1`/`-2` container-name suffix,
  fixed IPs on the `production` Docker network), confirmed via direct
  `InspectDockerContainer` reads, not the Stack-summary API. `voip-rtpengine-docker-1`/`-2`
  likewise. This landed via VOIP-1402 (merged 2026-08-26, live since 2026-08-29).
- **The two blockers VOIP-1335 flagged before going on hold**:
  1. *Queue-consumer gap during restart* — **resolved by VOIP-1402's own design**: both
     instances of a service share one RabbitMQ competing-consumer queue
     (`asterisk.call.request` etc.), so the surviving instance keeps consuming while its
     sibling restarts. No action needed here.
  2. *Asterisk-id instability across container recreation* — **re-confirmed still live**,
     without a disruptive restart test: `InspectDockerContainer` on all 4 running asterisk
     containers shows MAC addresses with no relationship to their static IPs (e.g.
     `172.24.0.101` → `72:ce:24:e6:51:2f`), and no `mac_address:` pin exists in any compose
     file. The `production` Docker network assigns MACs randomly on container creation.
     `voip-asterisk-proxy`'s `getAsteriskIDAddress` derives the asterisk-id from this MAC at
     runtime (compose comment: "AsteriskID is computed at runtime from the container's MAC
     address... never from hostname or an env var"; the `ASTERISK_ID` env var on the proxy
     sidecar is dead code, tracked for removal under VOIP-1365). **Conclusion: a container
     recreation (crash-restart, redeploy) gets a new random MAC, hence a new asterisk-id.
     Sentinel cannot assume a stable id-to-container mapping.**
- **Pipeline is real, not a void** (an earlier read of this session had wrongly concluded
  nobody consumes sentinel's events — corrected): `bin-call-manager/pkg/subscribehandler`
  binds `sentinel-manager.pod.deleted`; `callhandler.EventSMPodDeleted` filters on
  `p.Namespace == "voip" && p.Labels["app"] == "asterisk-call"`, then calls
  `RecoveryStart(ctx, p.Annotations["asterisk-id"])`, which pulls the last 24h of that
  asterisk-id's channels, looks up each one's SIP dialog via Homer, and PJSIP-redials it
  onto a live instance. This is the mechanism sentinel-manager exists to trigger.

## 2. Goal

Give sentinel-manager a Docker-native event-detection backend so it can run on bm-nyc-01
(no Kubernetes anywhere in this stack), and restore its CI/CD deploy path, while preserving
the existing recovery contract on the call-manager side as closely as possible.

**Non-goals**: redesigning the recovery RPC itself (`RecoveryStart`/Homer/PJSIP redial) —
out of scope, untouched. Removing the dead K8s code path — out of scope for THIS ticket
(K8s manifests can be deleted in a follow-up once the Docker backend is proven; see §7).

## 3. Architecture

### 3.1 Event source: Docker Events API, not an informer

Replace `pkg/monitoringhandler`'s K8s `SharedIndexInformer` with a watcher on the Docker
Engine API's `/events` stream (`docker/docker/client`, new dependency — no
`k8s.io/client-go`/`k8s.io/api` needed once this lands; those become removable in the
follow-up cleanup, §7). Filter to `type=container`, `event=start,die`, matching container
names against the compile-time selector list (equivalent to today's hardcoded
`map[string][]string` in `cmd/sentinel-manager/main.go`, just re-expressed as name-prefix
patterns instead of K8s label selectors):

```
voip-asterisk-call-docker-*
voip-asterisk-conference-docker-*
voip-asterisk-registrar-docker-*
```

(The `-proxy` sidecar containers, e.g. `voip-asterisk-call-docker-1-asterisk-call-proxy-1`,
are explicitly excluded — recovery only cares about the main asterisk container's lifecycle,
matching today's K8s pod-level granularity.)

### 3.2 Docker socket access: read-only proxy, not a raw mount

`/var/run/docker.sock` grants root-equivalent host access. Per VOIP-1335's flagged risk,
front it with `docker-socket-proxy` (Tecnativa's, or equivalent) configured for
**read-only, `EVENTS=1` + `CONTAINERS=1` only** — every other API family
(`POST`, `EXEC`, `IMAGES`, `NETWORKS`, `VOLUMES`, ...) denied, all written out explicitly
rather than relying on image defaults (also `PING=1`/`VERSION=1`, needed for the
healthcheck and the Docker Go client's API-version negotiation — these expose no container
data). sentinel-manager talks to the proxy's HTTP endpoint, never touches the real socket.

**Correction (found during implementation, not anticipated at design time — this section
originally called this "first precedent of this pattern in the fleet; no existing reference
to copy," which was wrong)**: two docker-socket-proxy instances already exist —
`infra-prometheus` (VOIP-1402) and `infra-loki` (VOIP-1423) — and both use a shape this
section did not: the proxy runs as a **sidecar inside the consumer's own compose file**, on
a dedicated **`internal: true` network shared only with that one consumer**, deliberately
**not** on the shared `production` network. `infra-prometheus`'s compose records the reason
verbatim: on `production`, the proxy (and, through it, every container's env vars via
`docker inspect`) becomes reachable from *any* of the ~50+ containers on that network — a
materially larger blast radius than a proxy meant to serve one consumer needs. `infra-loki`
independently re-applied the same shape and explicitly declined to share `infra-prometheus`'s
existing proxy for this reason. A standalone shared Stack (what this section originally
specified) would have been the *first* deviation from an already-twice-repeated convention,
going against, not filling, an established pattern.

**Follow the established precedent, not the original text of this section**:
`bin-sentinel-manager/komodo/docker-compose.yml` (§4) gets its own `docker-socket-proxy`
sidecar service, plus a dedicated `internal: true` network joined only by sentinel-manager
and that sidecar. No cross-repo dependency, no `monorepo-etc` change, no new Komodo Stack —
this is entirely self-contained within sentinel-manager's own compose file, same as
`infra-prometheus`/`infra-loki`. sentinel-manager's Docker-socket-proxy endpoint config
value is simply the sidecar's in-compose service name (e.g. `http://docker-socket-proxy:2375`)
resolved over that internal network — not an externally-managed hostname.

`CONTAINERS=1` is required (not optional) — see §3.3, sentinel needs one
`InspectDockerContainer`-equivalent call per container at `start` time, plus the same call
per already-running container once at sentinel's own boot (§3.3 step 0). Never at `die` time
in either case — a dying/dead container's inspect response has an empty `IPAddress`, so a
die-time fallback inspect would silently fail and must not be relied on. This does expose
full container config (env vars, labels — i.e. secrets-in-env) via `/containers/{id}/json`
to anything that can reach the proxy; scope the proxy to the `production` network only (no
public exposure) and treat this as an accepted, documented risk of the mitigation rather than
a fully closed hole — Komodo's proxy has no finer-grained field-level ACL to shrink it
further. sentinel-manager must fail loud if the proxy is unreachable (exit non-zero /
crash-loop, which Komodo's own health monitoring already surfaces as an alert) rather than
silently running with a dead event stream — a sentinel that *looks* up but watches nothing is
worse than one that's visibly down. The proxy itself runs single-instance (matching
sentinel's own single-replica choice, §4) — if it's down, sentinel is blind and says so
loudly; no HA attempted for this first iteration.

### 3.3 State table: the mechanism that resolves both the IP and the asterisk-id, race-free

Round-1 review correctly rejected the original design here: resolving IP from the `die`
event's `Actor.Attributes` doesn't work (Docker only puts `image`/`name`/`labels`/`exitCode`
there, not `NetworkSettings`), and reverse-scanning Redis *at die time* is ambiguous — the
dying container's static IP can already be reused by a fresh replacement that registered its
own (fresher) key under a different id before sentinel processes the `die` event, so a
scan-at-die-time can't reliably tell "the id that just died" from "the id that just took over
the same IP".

Fix: sentinel keeps an in-memory state table, `map[containerName]lastKnownState{ip, asteriskID,
observedAt}`, populated **before** the death, never resolved reactively at death:

0. **On process start (boot)** — **added in response to round-2 review**: the table is
   otherwise only ever populated by `start` *events*, which sentinel only observes for
   containers that start *after* sentinel itself is already watching. Every sentinel restart
   (its own crash, or a routine redeploy) would otherwise leave already-running asterisk
   containers with no table entry — an indefinite blind spot for those containers until their
   *next* recreation, not a bounded one, since sentinel has no replica to cover for it while
   down (§4). Fix: on startup, list currently-running containers matching the watched name
   patterns (`GET /containers/json`, one call), `InspectDockerContainer` each to seed `{ip,
   asteriskID: "", observedAt: now}`, then run one immediate (not wait-10s) pass of step 2
   below before entering the normal event loop. This is the same `CONTAINERS=1` proxy
   permission already required for step 1 — no new scope.
1. **On `start`**: one `InspectDockerContainer` call resolves the container's IP (this call is
   safe — the container is freshly running). Store `{ip, asteriskID: "", observedAt: now}`.
2. **Background refresh loop** (every 10s, independent of the event stream): for every
   watched container currently in the table, `SCAN asterisk.*.address-internal` once per
   tick, `GET` each match's value (the IP). **Round-2 review correctly rejected building a
   bare `map[ip]id` here**: the key has a 24h TTL refreshed every 5 min (`Set` + `Sleep(5*
   time.Minute)` in `voip-asterisk-proxy/cmd/asterisk-proxy/main.go` — note a stale doc
   comment nearby says "every 3 min"; the code, not the comment, is the 5-min source of
   truth), so a dead generation's key for a given IP can coexist with a live generation's key
   for the *same* IP for up to 24h — an unfiltered `map[ip]id` silently picks whichever the
   scan happens to return last, which can bind a live container to a dead generation's id (or
   vice versa). Fix: **only accept a candidate as "current occupant of this IP" if its
   remaining TTL is within one of the proxy's 5-min key-refresh intervals of full**
   (`remaining >= 24h - 12min` — this threshold is keyed to the Redis key's own 5-min refresh
   cadence, not sentinel's unrelated 10s background-loop cadence; round-3 review widened this
   from an initial 6-minute margin,
   which was tight enough that one missed `Set` from a Redis blip, GC pause, or the loop's
   own non-ticker `Sleep`-based drift could misclassify a genuinely healthy container as
   stale).

   **Sticky-last-known, not overwrite-with-unknown — required, per round-3 review**: a
   refresh pass that finds no fresh candidate for a table entry's IP **must leave that
   entry's `asteriskID` unchanged**, not reset it to `""`. This applies to the **refresh loop
   only** — entry *creation* (step 0 and step 1) unconditionally starts from `asteriskID: ""`;
   stickiness governs updates to an existing entry, not initialization. The freshness filter's
   job is only to decide whether *this pass* learned anything new; it is never itself evidence
   that a previously-resolved id is wrong — and per round-4 review, this is not a heuristic but
   correct **by invariant**: the asterisk-id derives from the container's MAC, which is fixed
   for that container object's entire lifetime, and one table entry spans exactly one
   container generation (created on `start`/boot §3.3-step-0, destroyed on `die` §3.3-step-4).
   So the true id is *constant* over an entry's life — "learn once, never unlearn" cannot go
   wrong, and symmetrically cannot go stale-forever either: a resolved id is consumed at most
   once (by the `die` that deletes its entry), so there is no path to firing recovery
   repeatedly against a permanently-dead generation's id. Freshness gates *learning* only,
   never *forgetting*.

   Without stickiness, a single missed refresh cycle racing a `die` event would regress a
   correctly-resolved id back to empty and, combined with §3.6's new empty-id guard, **silently
   skip the exact recovery sentinel exists to trigger** — the failure mode round-3 review
   flagged as the one path that would defeat the whole design. Log at WARN (with a Prometheus
   counter, mirroring the existing `sentinel_manager_pod_state_change_total` pattern, labeled
   by `container_name` so alerting can distinguish one persistently-unresolvable instance from
   a fleet-wide rate) whenever a `die` publishes with an unresolved id, and separately whenever
   a refresh pass finds zero fresh candidates for an entry that already has a resolved id (a
   leading indicator the *next* death for that container may go unrecovered, worth alerting on
   before it happens, not just after).

   In the pathological case where two generations' keys for the same IP are *both* within the
   freshness window (old instance's last refresh landed just before it died, new instance
   already started, resolved its own id, and refreshed within the same few minutes) sentinel
   additionally excludes any id already bound to a *different* container name currently alive
   in its own table — this narrows the theoretical remaining ambiguity to a same-second
   overlap the freshness filter alone doesn't resolve; documented as a residual, not
   eliminated to zero, but no longer "whichever the scan yields last."
3. **On `die`**: read the table entry for that container name **as it stood before this
   event** (the background loop and the event handler both only ever *read-then-write* their
   own table entries; a `die` handler takes whatever `asteriskID` was last observed, does not
   re-scan). Pass it to the publish path (§3.5). If `asteriskID` is still `""` (container died
   before ever completing one refresh cycle — possible for a crash within the first 10s of
   life, before it even registered with Redis, in which case there is genuinely no prior
   channel history to recover anyway), publish with an empty id. **Round-2 review correctly
   caught that this does NOT already degrade safely on the consumer side** — unlike VOIP-1419's
   `-` placeholder (a *routing-key* concern, unrelated to this payload field),
   `callhandler.EventSMPodDeleted`/its renamed successor calls `RecoveryStart` unconditionally
   today with no empty-id guard, and `RecoveryStart` → `GetChannelsForRecovery` would be
   called with an empty string. §3.6 must add an explicit `if AsteriskID == "" { return nil }`
   early return as part of this change — a new guard, not existing behavior.
4. **On `die`**, also delete the table entry (the next `start` for that name rebuilds it from
   scratch — don't let a stale entry silently answer a *future* death for a same-name
   container with the wrong generation's last-known id).

This still has one residual, deliberately-accepted race: if a replacement container starts,
gets inspected, and completes a Redis-scan refresh cycle (worst case ~10s) *before* sentinel
processes the original `die` event, the table would already have been overwritten by the
new generation's data under a *reused container name* if step 4's delete-on-die loses a race
with step 1/2's own writes for what looks like the same name. Implementation must sequence
per-container-name state mutation through a single goroutine (or a mutex-per-name) so `die`'s
read+delete for generation N cannot interleave with generation N+1's `start` write — this
makes the residual window "practically zero" (bounded by Docker's own event delivery order
guarantee per container, which is FIFO) rather than eliminating it in every theoretical
interleaving. Document this as the known limitation; it is strictly better than the original
design's window (which was the *entire* time between death and sentinel's reactive scan, not
a same-tick race).

### 3.4 Event-stream resilience

- **Reconnect**: the Docker Events API call takes a `since` timestamp; on any stream
  disconnect (proxy restart, network blip), reconnect with `since=<last processed event
  time>` so a gap doesn't silently drop a `die` while sentinel was reconnecting. Bounded gap
  only (not a full replay) — acceptable because `RecoveryStart` (§1) is already fire-and-forget
  and best-effort on the call-manager side; sentinel's own event delivery doesn't need a
  stronger guarantee than the mechanism it's feeding.
- **Flap damping**: if the same container name dies and restarts more than, say, 3 times in
  60 seconds, log at WARN and skip triggering `RecoveryStart` on the later occurrences in that
  window (a flapping container is a symptom to alert on, not something to keep redialing calls
  against — repeated recovery attempts against a container stuck in a crash-loop would just
  spam Homer/PJSIP for channels that likely never had a chance to establish). Threshold is a
  starting point for implementation to tune, not a hard requirement.
- **start-immediately-followed-by-die dedup**: naturally handled by §3.3's table design — a
  `start` with no completed refresh cycle before the next `die` simply publishes with an
  unresolved id (§3.3 point 3), which is already the correct degraded behavior, not a special
  case to add extra code for.
- **Restart-in-place vs. recreate — flagged by round-2 review, needs an implementation-time
  empirical check, not resolved by this design alone**: Docker's `restart: unless-stopped`
  policy reacting to a crashed process restarts the *same* container object (same container
  ID); only an explicit recreate (`compose up --force-recreate`, or a deploy that changes the
  compose content) produces a *new* container ID. Whether the same-container-ID case also
  preserves the MAC (and therefore the asterisk-id) — as opposed to VOIP-1335's empirical
  finding, which may have specifically exercised the recreate path — determines how often
  `AsteriskID` actually changes in the common crash-and-auto-restart case versus only on
  deploys. This does not change the design (the Redis-based resolution in §3.3 is correct
  either way — a stable id simply means the resolved value matches what it was before), but
  it does affect how aggressively flap-damping should trigger and is worth a cheap
  verification early in implementation (send a watched container's process a kill signal so
  the restart policy reacts, without a full `docker stop`/recreate, and re-inspect the MAC)
  before tuning the flap-damping threshold in §3.4.

### 3.5 Event schema (sentinel → call-manager)

Today's payload is a raw `*corev1.Pod` (wrapped in `pod.Event` for VOIP-1419's explicit
`EventSubscriptionID`). Docker has no equivalent struct. New minimal model,
`bin-sentinel-manager/models/container` (replaces `models/pod`):

```go
type Event struct {
    ContainerName string // e.g. "voip-asterisk-call-docker-2"
    Service       string // "asterisk-call" | "asterisk-conference" | "asterisk-registrar"
    AsteriskID    string // resolved per §3.3; "" if unresolved
}
```

`EventTypeContainerDied` replaces `EventTypePodDeleted`; `EventTypeContainerStarted`
replaces `EventTypePodUpdated` (today's `AddFunc` no-op / `UpdateFunc`-drives-"updated"
semantics don't map cleanly to Docker's `start`/`die` pair — `start` is the natural
"updated" analogue since Docker doesn't have K8s's watch-resync distinction). **Decided
(resolves the §6 open question rather than leaving it open)**: the wire string is renamed,
not reused verbatim — "pod" is actively misleading once nothing is a pod, and sentinel
currently has zero real subscribers besides call-manager, which is being updated in this
same change, so there is no external consumer this rename could silently break. Update
`models/pod/routingkey_golden_test.go`'s expectations accordingly (its `sentinel-manager.
pod.-.updated`/`.deleted` golden keys move to the new `container` type name — this is an
intentional, reviewed key-string change, unlike VOIP-1419's constraint elsewhere that
routing-key values must not move).

### 3.6 call-manager consumer-side change

`callhandler.EventSMPodDeleted(ctx, p *smpod.Pod)` filters on
`p.Namespace == "voip" && p.Labels["app"] == "asterisk-call"` and reads
`p.Annotations["asterisk-id"]`. With the new `container.Event` schema this becomes a filter
on `Service == "asterisk-call"` and a direct `AsteriskID` field read — simpler, not more
complex, since Docker has no namespace/label/annotation indirection to replicate. Rename to
`EventSMContainerDied` (or keep `EventSMPodDeleted`'s name if minimizing diff is preferred —
implementation decision, functionally identical either way).

## 4. CI/CD (VOIP-1418 task 2)

Once the code targets Docker, add `bin-sentinel-manager-deploy` to
`.circleci/config_work.yml`'s `bin-sentinel-manager` workflow (currently ends at `build`),
mirroring the other 32 services' `komodo-api-deploy.sh` pattern (render image tag →
`komodo-api-deploy.sh bin-sentinel-manager bin-sentinel-manager/komodo/docker-compose.yml`).
New `komodo/docker-compose.yml` under `bin-sentinel-manager/` (currently absent — the one
gap among all `bin-*-manager` services besides the DB-scheme/openapi/trigger-sender
non-deployables). Single replica (sentinel watches, doesn't serve traffic — no HA need of
its own). **Correction from an earlier draft of this section** (round-2 review caught the
error): missing a `die` event during sentinel's own downtime is NOT a self-healing gap —
without §3.3 step 0's boot-time reconciliation, a missed death would leave that container
permanently unresolvable until its *next* recreation. With step 0 in place, the actual
exposure is narrower and correctly bounded: only a container that both starts *and* dies
entirely within sentinel's own downtime window is missed (no `start` event to seed from, and
boot-time reconciliation only sees what's running *at* boot) — bounded by how long sentinel
itself is down for, which a single-replica deploy keeps short (seconds, a compose restart),
not by anything about the watched containers' own behavior.

## 5. Rollout risk / verification plan

- Docker-socket-proxy Stack is genuinely new infrastructure — stand it up and verify its
  API-family allowlist (deny-by-default, allow only `EVENTS`+`CONTAINERS` GET) *before*
  wiring sentinel to it; a misconfigured proxy exposing `POST`/`EXEC` would be a worse
  security posture than not building this at all.
- Redis reverse-scan is a new read pattern against a production key namespace
  call-manager already owns — read-only, no risk to existing writers, but confirm the `SCAN`
  cursor pattern doesn't block Redis under the existing key volume (low: bounded by live
  instance count, single digits).
- Recommend a staged verification once deployed: manually `docker stop` one non-primary
  instance (e.g. `voip-asterisk-registrar-docker-2`, least call-traffic-bearing of the three
  during a low-traffic window, with 대표님's sign-off on timing) and confirm sentinel
  publishes the event, call-manager logs `EventSMPodDeleted`/renamed-equivalent firing, and
  the resolved asterisk-id matches Redis's last-known value for that instance — before
  trusting this in an actual crash scenario.

## 6. Open questions for review

- §3.5: **resolved** — wire string renamed, decision recorded in §3.5 directly.
- §3.5: **resolved (added at implementation-plan review — planner correctly flagged this as
  a design-level question the plan couldn't decide on its own)**: `container.Event`'s
  `EventSubscriptionID()` returns `AsteriskID` directly (empty string degrades to the `-`
  placeholder via the standard `eventtopic.normalizeSubscriptionID` path — no special-casing
  needed). This is a deliberate departure from the old `pod.Event`, which returned `""`
  unconditionally because a K8s Pod carried no addressable identity at all. A resolved
  asterisk-id *is* a real, meaningful address once §3.3's state table exists — treating it as
  a placeholder forever, now that one is available, would waste the identity the whole
  redesign goes to such lengths to resolve. This does not require any change on the consumer
  side: `bin-call-manager`'s existing subscription binds via
  `eventtopic.PatternForEventType(ServiceNameSentinelManager, EventTypePodDeleted)`, verified
  (`binding_golden_test.go`) to produce the wildcard pattern `sentinel-manager.pod.*.deleted`
  — i.e. call-manager already matches *any* subscription-id segment, not one specific
  instance, so populating a real (non-`-`) address here is additive (opens the door to a
  future consumer subscribing to one specific instance's events) and changes nothing about
  today's only real consumer. The rename to `sentinel-manager.container.*.died` (§3.5) covers
  updating this pattern string alongside the type rename.
- §4: does sentinel need its own Komodo Stack env vars beyond the Docker-socket-proxy
  endpoint address? (Redis address is likely already a standard `[[BIN_MANAGER__...]]`
  convention value — confirm during implementation, not a design blocker.)

## 7. Explicitly deferred (follow-up tickets, not this one)

- Deleting the now-dead `bin-sentinel-manager/k8s/` manifests and `k8s.io/*` dependencies.
- Removing `ASTERISK_ID` dead env var repo-wide (already tracked: VOIP-1365).
- Any redesign of the recovery RPC itself.
