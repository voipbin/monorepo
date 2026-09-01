# VOIP-1418: bin-sentinel-manager Docker-backend design

Status: §1-§7 **DESIGN APPROVED** (round 4 + round 5, 2 consecutive Approve — CLAUDE.md
design review loop satisfied) and **shipped** in PR #1240 (Docker-only backend, merged
into the branch, not yet merged to `main`). §8 is a **new addendum, DESIGN APPROVED** (round 7 + round 8, 2 consecutive Approve —
CLAUDE.md design review loop satisfied, 8 rounds total) — added 2026-09-01 per the CEO's
explicit direction after PR #1240 was opened. Eight rounds, several catching real
client-go/informer correctness gaps: round 1 (fail-loud contract not inherited), round 2
(mechanism under-specified), round 3 (`DeletedFinalStateUnknown` tombstone handling
missing), round 4 (`UpdateFunc` UID-mismatch gap), round 5 (death-path-closure audit found
no sixth gap, but left a duplicated paragraph and an overstated claim), round 6 (that
overstatement's self-contradiction, corrected), rounds 7-8 (verification passes, 3 cosmetic
nits plus a pre-existing §3.2/production-network cross-reference, all folded in). Ready for
implementation planning. VoIPBin is a self-hostable opensource CPaaS, and a self-hosted deployment on
Kubernetes needs sentinel-manager's stranded-call detection just as much as bm-nyc-01's
Docker-Compose deployment does. §1-§7's "replace K8s with Docker" framing should have been
"add Docker alongside K8s" — this was a real gap in the original design's trade-off
analysis, not a deliberate scope decision; VOIP-1335's original ask was dual-backend
support from the start, and this addendum is that ask, now genuinely unblocked (the
recovery-topology blocker that put VOIP-1335 on hold was resolved by VOIP-1402, and the
Docker backend §1-§7 already built is one of the two backends this needs — the other one,
covered here, is restoring K8s as a peer implementation rather than something the Docker
work replaced).

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
to anything that can reach the proxy; scope the proxy to its own dedicated `internal: true`
network only (per this section's own correction below — not `production`; corrected during
implementation after this paragraph was first written, see the "Correction" block that
follows) and treat this as an accepted, documented risk of the mitigation rather than
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

- ~~Deleting the now-dead `bin-sentinel-manager/k8s/` manifests and `k8s.io/*`
  dependencies.~~ **Superseded by §8**: these are being restored to active use, not deleted.
  This bullet is now moot — kept struck through rather than removed so the history of why
  it was originally deferred (and then reversed) stays legible.
- Removing `ASTERISK_ID` dead env var repo-wide (already tracked: VOIP-1365). Still deferred
  — unaffected by §8; that env var was already dead before AND after this addendum (the K8s
  backend never used it either; asterisk-id has always come from the pod annotation, not an
  env var, on the K8s side — see §8.2).
- Any redesign of the recovery RPC itself.

## 8. Addendum: restore Kubernetes as a second, peer backend

### 8.1 Why this wasn't caught in §1-§7

§1's "Confirmed current state" correctly established that bm-nyc-01 has no Kubernetes and
needed a working deploy path *now*. That framing then silently generalized into "sentinel
only needs to run where we currently deploy it" — never stated as an explicit assumption,
never surfaced as a trade-off for review. VoIPBin is built and marketed as a self-hostable
opensource CPaaS (see product positioning materials); a self-hoster running Kubernetes has
exactly the same need for stranded-call detection as bm-nyc-01 does, and the Docker-only PR
would have silently regressed that capability to zero for that deployment shape while
fixing it for this one. Five design-review rounds and four plan-review rounds on §1-§7 did
not catch this because none of the reviewers were asked to check "does this design serve
every deployment target VoIPBin supports," only "is this design internally sound for the
target it names." That's a real process gap worth remembering for future infra tickets:
**state the deployment-target assumption explicitly and ask a reviewer to challenge it**,
don't let "the environment I'm looking at right now" silently become "the only environment
that matters."

### 8.2 Why the K8s backend is *simpler* to restore than the Docker backend was to build

The Docker backend's hardest problem — resolving a dying container's asterisk-id, given
that the id is derived from a container's MAC address which is not stable across
recreation, and there is no equivalent of a Kubernetes annotation to carry it — **does not
exist on the K8s side**. `voip-asterisk-proxy/cmd/asterisk-proxy/annotation.go` (unchanged
by this or any recent ticket) has always self-patched its own pod's `asterisk-id`
annotation via the K8s API at startup, using its own pod's in-cluster service account
(`setProxyInfoAnnotation` → `patchPodAnnotation`, JSON-patch `add` on
`/metadata/annotations/asterisk-id`). A Kubernetes Pod object's annotations are visible to
any watcher with read RBAC on pods in that namespace — no reverse lookup, no state table, no
freshness filter, no sticky-last-known semantics, none of §3.3's five-review-round
machinery. This is exactly what the pre-VOIP-1418 `pkg/monitoringhandler` already did
(`p.Annotations["asterisk-id"]`, read directly in the pod-delete handler) — restoring it is
close to reverting §2's deletion, not building something new, modulo one adaptation (§8.3).

**Caveats on this mechanism (round-1 design review found these worth stating explicitly,
none of them block the approach, but the K8s backend's empty-`AsteriskID` path — §8.3 — is
not a corner case, it's an expected one)**:
- `setProxyInfoAnnotation` is gated by a `--kubernetes_disabled`-style flag on the proxy
  side — a proxy started with K8s support disabled never patches the annotation at all.
  Deployment config on the asterisk-proxy sidecar side must have this enabled for the
  annotation to exist; this addendum doesn't change that flag, just depends on its correct
  setting being a deployment precondition, same as it always was pre-VOIP-1418.
  - JSON-Patch `add` on a nested path requires the parent map (`/metadata/annotations`) to
  already exist — pod templates need at least one annotation present at creation time (K8s
  manifests already set `prometheus.io/scrape` etc., so this is satisfied in practice, but
  it's a real precondition, not automatic).
- The proxy's own K8s service account needs `patch` RBAC on pods (its own pod only, scoped
  by namespace/name in practice) — a separate RBAC concern from sentinel-manager's own
  read-only `pod-reader` role (§8.4); this addendum touches neither, both already exist and
  are unaffected by anything in this ticket.
- **Consequence**: there is a real window between a pod starting and the proxy's patch
  landing (network round-trip, retries up to `maxRetries`) during which the annotation is
  genuinely absent. A pod that dies inside that window produces a `container.Event` with
  `AsteriskID: ""` — not a bug, the correct and expected degrade path (§8.3's last bullet),
  but implementation should not treat "empty annotation" as a scenario needing special
  handling beyond what §3.6's existing empty-id guard already does.

### 8.3 Architecture: a `MonitoringBackend` interface, two implementations, one event schema

```go
// bin-sentinel-manager/pkg/monitoringbackend (new, small — just the interface)
type MonitoringBackend interface {
    Run(ctx context.Context) error
}
```

Both `dockerwatchhandler.Handler` (§2-§7, already implements a `Run(ctx) error` method) and
a restored `pkg/k8swatchhandler.Handler` satisfy this. `cmd/sentinel-manager/main.go`
constructs whichever one `SENTINEL_BACKEND` selects and calls `.Run(ctx)` — the rest of
`main.go` (RabbitMQ connect, `notifyHandler` construction, Prometheus HTTP server) is
identical regardless of backend.

**Interface contract, stated explicitly (round-1 design review required this)**: `Run`
returns `nil` **only** when `ctx` was cancelled (normal shutdown); **any other cause of the
watch loop stopping — informer sync failure, a watch that dies and cannot be
re-established, the client losing its connection to the API server — must return a non-nil
error.** This is the same fail-loud contract §3.2 already established for the Docker
backend, and it is **not automatically satisfied by restoring the old K8s code as-is** (see
§8.4's rewrite requirement below) — the pre-VOIP-1418 implementation predates that contract
entirely and violates it.

**Backend selection**: `SENTINEL_BACKEND` env var, values `kubernetes` | `docker`, **no
default — fail fast at startup if unset or invalid** (matches VOIP-1335's original explicit
"config selects the backend" design over any form of auto-detection; auto-detecting "am I
in a K8s pod or a Docker container" is exactly the kind of implicit, hard-to-audit behavior
this whole ticket has spent many review rounds trying to eliminate elsewhere — an operator
should have to say which one they mean).

**Config validation becomes backend-conditional (round-1 design review caught this was
missing)**: `DockerSocketProxyAddress` and the Redis address are required when
`SENTINEL_BACKEND=docker`, irrelevant when `=kubernetes` — validate only the fields the
selected backend actually needs, or a K8s deployment fails startup on Docker-only config it
was never given and has no reason to provide. Symmetrically, nothing K8s-specific needs
validating today (no equivalent required config beyond in-cluster auth, which
`rest.InClusterConfig()` either finds or doesn't).

**Unified event schema — the key simplification**: the restored K8s backend does **not**
resurrect `models/pod` or publish a raw `*corev1.Pod`. It publishes the same
`container.Event{ContainerName, Service, AsteriskID}` (§3.5) the Docker backend already
publishes, mapped from the pod object:
- `ContainerName` ← `pod.Name`
- `Service` ← `pod.Labels["app"]`, **mapped through an explicit lookup, not assigned
  directly** (round-1 design review caught this: `container.Event.Service` is a *typed*
  constant on the Docker side — `ServiceAsteriskCall` etc. — not a free string; a bare label
  passthrough means a typo'd or unexpected `app` label value silently produces an event
  `bin-call-manager`'s filter never matches, with no signal that anything went wrong). The
  three expected label values (`"asterisk-call"`, `"asterisk-conference"`,
  `"asterisk-registrar"`) map 1:1 to the three typed constants; **any other label value is
  rejected at the publish boundary** (log + skip, matching the Docker backend's
  `matchWatchedContainer` behavior of ignoring non-watched names rather than silently
  forwarding them unmapped).
- `AsteriskID` ← `pod.Annotations["asterisk-id"]` directly, no resolution step. If the key is
  absent (see §8.2's caveats on when `annotation.go` hasn't run yet), publish with `""`,
  same degrade path §3.3/§3.6 already established for the Docker backend's unresolved case —
  no new behavior needed on the consumer side.

**This is the whole point of having unified the schema back in §3.5**: `bin-call-manager`'s
consumer side (`EventSMContainerDied`, the empty-id guard, the subscribehandler binding)
needs **zero changes** for this addendum. It already only knows about `container.Event`; it
has no idea whether a given event came from Docker or Kubernetes, and should never need to.
If §3.5 had kept the Docker backend on a Docker-specific schema, this addendum would have
needed either a second consumer-side type switch or a K8s-specific adapter published under a
fake "container" shape — the unification already done makes this a clean two-producers,
one-consumer picture instead.

### 8.4 What gets restored vs. rewritten

- **Restored close to as-is**: `rest.InClusterConfig()` auth, the per-`(namespace,
  label-selector)` watch structure, the `AddFunc`-is-a-no-op / no-resync pattern.
  `bin-sentinel-manager/k8s/` manifests (deployment, service, namespace, RBAC
  role/rolebinding) — never deleted (§7 deferred that), now become live again rather than
  dead weight.
- **Rewritten, not merely restored (round-1 design review — this is the substantive
  change, not a footnote)**:
  1. **Publish call sites**: old code published `pod.Event` (raw `*corev1.Pod` wrapper,
     placeholder-by-design subscription id). New code builds `container.Event` from the pod
     via the mapping in §8.3 and calls the same `notifyHandler.PublishEvent` +
     `WithGlobalTopicPublish()` pattern `dockerwatchhandler` already uses.
     **Callback-to-event-type mapping, pinned explicitly (round-4 design review found §8.3's
     field mapping never stated which informer callback produces which event type)**:
     `DeleteFunc` (including the tombstone path in item 3 below) → `EventTypeContainerDied`.
     `UpdateFunc` → `EventTypeContainerStarted` (the direct analogue of the old code's
     `pod_updated`, now using §3.5's renamed constant). `AddFunc` stays the established
     no-op (informers replay existing pods as synthetic `Add`s on initial list; publishing
     "started" for every pod already running at sentinel's own boot would misrepresent
     long-lived pods as freshly started — same reasoning the pre-VOIP-1418 code already
     had, unchanged by this addendum). **A real asymmetry between the two backends, worth
     stating precisely rather than leaving implicit (round-5 design review flagged this;
     round-6 review caught that round 5's own fix overstated it into a self-contradiction
     with the relist bullet further down, which correctly says newly-created pods DO get a
     `started` — fixed here)**: `AddFunc`'s no-op status means a new pod's very *first*
     observation is dropped, but every subsequent status transition (e.g. `Pending` →
     scheduled → `Running`/`Ready`) fires `UpdateFunc`, which publishes `started` per this
     mapping — so the K8s backend does not *omit* `started` for a genuinely new pod, it
     publishes it **late** (on the pod's first post-creation status update, not at creation
     instant) **and potentially more than once** (every subsequent no-op status update also
     re-fires it, same as the relist case below), unlike the Docker backend's single
     at-actual-start `started` (§3.1). §8.3's "the consumer can't tell which backend
     produced an event" claim holds for field *shapes*; it does not extend to `started`'s
     *timing or cardinality* — harmless today since `bin-call-manager` only consumes `died`
     (§3.6), but a future consumer relying on `started` semantics needs to know this
     up front, not discover it by comparing behavior across deployments.
  2. **`UpdateFunc` must detect a same-key identity change, or a death is silently dropped —
     the direct K8s analogue of §3.3 rule 2, which round-4 design review found §8 had no
     counterpart for (blocking: this is the same failure class as the tombstone gap round 3
     caught, not a smaller variant of it)**: client-go's reflector only synthesizes a
     `Deleted` callback for keys *absent* from a relist's new object set. If a pod is deleted
     and a same-name replacement created while the watch was interrupted (stable-identity
     workloads; plausible on a rolling update or node eviction that lands before the watch
     recovers), the key is still present in the relist — client-go delivers this as a
     `Replaced` delta through `UpdateFunc`, and **no delete callback ever fires for the dead
     generation**. This drops the exact death event this service exists to detect, silently,
     with no panic and no counter — precisely the outcome this section's own "never
     silently skip" principle forbids, just via a different door than the tombstone one.
     Required: `UpdateFunc` compares `oldPod.UID != newPod.UID`; on a mismatch, publish a
     `died` for the old pod (built from the stale object client-go still hands the callback,
     annotation and all) **before** processing the update as a normal "started" for the new
     one. Count this path on the same observable-counter pattern as the tombstone case
     (item 3 below, the `DeletedFinalStateUnknown` bullet) — a same-key UID change detected this way is exactly as much a
     signal of watch instability as a tombstone is, and should be visible the same way.
  3. **Fail-loud propagation, entirely new — the old code never had this**: the pre-existing
     implementation (`git show origin/main:bin-sentinel-manager/pkg/monitoringhandler/run.go`)
     spawns informers in bare goroutines with no error channel back to the caller —
     `AddEventHandler` failure only logs from inside the goroutine, `podInformer.Run(stopCh)`
     returning is unobserved, and the outer `Run` blocks on `<-ctx.Done()` and returns `nil`
     unconditionally. client-go's list/watch retries internally and only logs on failure —
     exactly the "looks up, watches nothing, exits 0" failure mode `bin-sentinel-manager/
     CLAUDE.md` forbids, predating that rule entirely. The rewrite must close this, matching
     §8.3's interface contract:
     - **A consecutive-failure budget, the K8s analogue of `dockerwatchhandler`'s
       `maxConsecutiveEmptyStreams`/`healthyStreamLifetimeFactor` pair (round-2 design
       review required this — it is a design decision of the same weight §3.3 took five
       rounds on, not an implementation detail left to the coder).** `SetWatchErrorHandler`
       fires on routine, benign conditions too (apiserver rolling restart, `too old resource
       version` forcing a relist, a transient connection reset) — wiring it straight to "this
       is fatal" would self-restart a perfectly healthy system, and wiring it to "just log"
       reproduces the old silent-failure behavior under a new API. The handler increments a
       counter on each invocation; a **successful relist/resync resets it to zero** (the
       direct analogue of "delivery resets the budget" on the Docker side); exceeding a
       threshold (mirror the Docker side's magnitude — on the order of tens of consecutive
       failures, tuned during implementation the same way `maxConsecutiveEmptyStreams` was)
       is what actually converts to the fatal error `Run` returns. A three-valued outcome
       label (`resynced` / `transient-error` / `fatal`, or equivalent) on a Prometheus
       counter gives this the same observability `events.go`'s `result` label gives the
       Docker side.
     - **`WaitForCacheSync` at startup must run under an explicit bounded deadline, not a
       bare call (round-2 design review caught this the naive spec doesn't fail loud)**:
       `WaitForCacheSync` returns `false` only when its stop channel closes — under the
       canonical restore-this-ticket-fixes failure (missing `pod-reader` RBAC), the reflector's
       initial `List` is denied and the client retries with backoff *forever*, so the call
       simply blocks and never returns `false` on its own. Wrap it in a `context.WithTimeout`
       (a fixed startup-sync deadline, order of tens of seconds — tune during implementation)
       and treat a deadline-exceeded as the fatal-startup case. This is also the fix for a
       standing factual error in `bin-sentinel-manager/CLAUDE.md`'s pre-VOIP-1418 claim
       ("RBAC required... informer fails at startup and the service exits") — that was never
       actually true of the old code path; this rewrite is what makes it true. **While
       touching that file (round-3 review flagged this too): its current "Service class:
       Docker container lifecycle monitor" header and its "`k8s/` is dead and intentionally
       left in place... do not delete it" bullet are both artifacts of §1-§7's Docker-only
       PR and are directly reversed by this addendum — update both, not just the RBAC
       sentence, in the same implementation pass.**
     - Run each `(namespace, selector)` informer goroutine through an `errgroup.Group` (or
       equivalent explicit fan-in) so any one of them failing propagates out of `Run` instead
       of silently leaving the others running with reduced coverage and no signal. **Graceful
       shutdown must stay graceful through this fan-in (round-4 design review flagged this
       as worth carrying into implementation planning, non-blocking)**: each informer
       goroutine must return `nil` on its own stop-channel closing due to *parent* `ctx`
       cancellation — only a goroutine's own genuine failure should produce the non-nil error
       that `errgroup`'s `Wait()` surfaces; a normal shutdown must not get misreported as one
       sibling's failure just because `errgroup.WithContext`'s derived context also cancelled
       the others.
     - **A relist re-publishes `started` for every already-running watched pod, not just
       newly-created ones (round-4 design review: state this rather than let it be an
       unexamined side effect of round-2's own budget)**: `Replace()` delivers unchanged
       objects through `UpdateFunc` same as a real update, and round-2's consecutive-failure
       budget makes relists a routine, expected event rather than a rare one — so this is a
       recurring burst pattern, not a one-off startup artifact. Non-blocking today because
       `bin-call-manager`'s consumer only acts on `died` (§3.6), but implementation should
       not "fix" this by trying to suppress no-op updates — that reintroduces exactly the
       kind of unstable dependency on identity comparison this section's UID-mismatch check
       (above) needs to get right regardless; simpler to publish and let an uninterested
       consumer ignore it, matching how the Docker backend already treats its own `started`
       events.
     - **`DeleteFunc` must handle `cache.DeletedFinalStateUnknown` (round-3 design review
       caught this — the restored-as-is code does not, and round-2's own fix makes the gap
       worse, not incidental)**: the code this section restores "close to as-is"
       (`DeleteFunc: func(obj any) { pod := obj.(*corev1.Pod) ... }`) bare-asserts every
       delete callback's argument to `*corev1.Pod`. client-go delivers
       `cache.DeletedFinalStateUnknown` instead whenever a deletion was missed while the
       watch was interrupted and is only discovered on the next relist — a bare assertion
       panics on that shape. The reflexive fix (type-assert with `ok`, `return` on mismatch)
       is *wrong here*, not just incomplete: it silently drops exactly the death event this
       whole service exists to detect, the same failure class §3.3 spent five review rounds
       eliminating on the Docker side. Round-2's consecutive-failure budget makes this
       reachable **by design**, not a rare edge case: the budget's whole point is keeping the
       process alive through transient watch interruptions, and a relist after exactly such
       an interruption is precisely when a tombstone shows up. Required handling: unwrap
       `DeletedFinalStateUnknown.Obj` (which carries the last-known `*corev1.Pod`, annotation
       included) and publish from it exactly as a normal delete would — never bare-assert,
       never silently skip on the unknown-shape case — and count tombstone-recovered
       deletions on an observable counter (a label on the same delete-path metrics
       `dockerwatchhandler` already patterns after) so a spike in "detected via relist rather
       than live watch" is itself a visible signal of watch instability, not just silently
       absorbed.
  4. **Unresolved-annotation observability — pinned, not left open (round-2 design review
     flagged this as more than a mechanical detail)**: reuse the **identical** metric,
     `sentinel_manager_container_unresolved_asterisk_id_total`, **no `backend` label** (a
     given process runs exactly one backend for its lifetime, selected once at startup by
     `SENTINEL_BACKEND` — a label that can only ever hold one value for the process's whole
     life adds cardinality for zero discriminating power). Its Prometheus registration moves
     out of `dockerwatchhandler`'s `init()` into a neutral home both backends can reach
     without importing each other (`pkg/monitoringbackend`, or a small shared metrics
     package alongside it — implementation's call which, but it must not live inside either
     backend package once both need to increment it). `pkg/k8swatchhandler` increments the
     same counter whenever a K8s-sourced **`died`** event publishes with an empty `AsteriskID`, so
     this failure signature is visible identically regardless of which backend produced it.
     **Corrected during implementation (code review round 1 confirmed this reading is better than
     the literal text it replaces, which said "whenever a K8s-sourced event publishes with an empty
     `AsteriskID`")**: the scope is DEATHS ONLY, not any event. A `container_started` legitimately
     carries an empty id on both backends — always on the Docker side (a freshly started container
     has not been resolved yet) and throughout the annotation-patch window on the K8s side (§8.2) —
     so counting those would swamp the signal, contradict the counter's own Help string, and make
     the shipped Grafana panel fire constantly on a healthy cluster. That panel — already in PR
     #1240's `monitoring/grafana/dashboards/sentinel-manager.json`, the "Recovery Health" row, and
     described there as "one container death that will NOT trigger call recovery" — therefore
     covers both backends unchanged.
  - Package name `pkg/k8swatchhandler` (not the old `pkg/monitoringhandler` — that name was
    already identified as too generic when `dockerwatchhandler` was named; same reasoning
    applies here for symmetry).
- **New**: `pkg/monitoringbackend`'s one-method interface; `cmd/sentinel-manager/main.go`'s
  backend-selection branch; `internal/config`'s `SENTINEL_BACKEND` field, its
  fail-fast-on-invalid validation, and the backend-conditional validation split above.
- **go.mod**: `k8s.io/client-go`, `k8s.io/api`, `k8s.io/apimachinery` return (removed by
  §1-§7's `go mod tidy`, re-added by this addendum's). This will very likely re-ripple the
  same fleet-wide indirect-dependency MVS resolution §1-§7 already did once — verify
  direction (versions moving back down, or further up depending on what k8s.io/* now
  resolves against) rather than assuming it round-trips to identical numbers. **Invariant
  that must hold, stated explicitly per round-1 review**: `bin-sentinel-manager/models/
  container` (the shared event schema, §3.5/§8.3) must stay free of any `k8s.io/*` import —
  it is consumed by `bin-call-manager` (whose own go.mod must NOT reacquire a k8s.io/*
  dependency through this addendum) and referenced via a `replace` directive by at least one
  other module (`voip-kamailio-proxy/go.mod` also replaces `bin-sentinel-manager` directly,
  not just `bin-call-manager` — confirm no other such reference picks up a transitive k8s.io/*
  import either). The k8s.io/* dependency belongs to `pkg/k8swatchhandler` and `cmd/
  sentinel-manager` alone.

### 8.5 Testing

`pkg/k8swatchhandler` needs its own suite using `client-go`'s fake clientset — the deleted
`pkg/monitoringhandler/run_test.go` (recoverable from git history, same commit references as
§8.4) is close to a direct template, adjusted for the new `container.Event` publish
assertions instead of the old raw-pod ones. This is structurally simpler than
`dockerwatchhandler`'s test suite (no state table, no timing-dependent freshness logic) —
should not need anywhere near the same review scrutiny §3.3 required, but every new/changed
function still gets tests per this repo's standing testing convention, not just the
happy path (empty-annotation case, wrong-namespace/wrong-label filter-skip case, etc.,
mirroring the equivalent Docker-side test shapes already in the codebase for consistency).

### 8.6 Deployment packaging (no code implication, noted for completeness)

A K8s deployment of sentinel-manager needs `bin-sentinel-manager/k8s/*.yml` applied with
`SENTINEL_BACKEND=kubernetes` and does **not** need the Docker-socket-proxy sidecar or Redis
dependency §4's `komodo/docker-compose.yml` wires up — those are Docker-backend-only. A
Docker-Compose deployment (bm-nyc-01) needs the reverse. This is packaging, not code — the
binary is identical either way, config picks the path. No action needed in this repo beyond
what §4 (Docker path) and the restored `k8s/` manifests (K8s path) already provide; a
self-hoster choosing K8s uses the `k8s/` manifests, a self-hoster choosing Docker-Compose
uses `komodo/docker-compose.yml` as a reference (it's Komodo-specific, but the compose
service definition itself is generic).

### 8.7 Non-goals (unchanged from §2, restated for this addendum)

Still out of scope: any redesign of `RecoveryStart`/Homer/PJSIP itself, removing
`ASTERISK_ID` (VOIP-1365, unaffected either way), and auto-detection of which backend to
run (§8.3 — explicit config only).
