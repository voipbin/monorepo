# VOIP-1418: Implementation plan

Status: **PLAN APPROVED** (round 3 + round 4, 2 consecutive Approve — CLAUDE.md
implementation-plan review loop satisfied). Round 4 implementer notes (non-blocking, apply
during coding): (1) the Grafana file needs a full pass — beyond the enumerated lines, a
`legendFormat: "Pod Changes/min"` literal and 5 panel titles also say "Pod"; (2) confirm
`models/pod` has no reverse dependency from `monitoringhandler`'s replacement before W2's
boundary, since deletion is deferred to W3; (3) call out the go.mod k8s.io/* delta
explicitly in the PR body; (4) highest test-writing effort belongs in
`dockerwatchhandler`'s state table, not the mechanical renames. Ready for implementation.
Normative source: the Approved design
(`docs/plans/2026-09-01-voip-1418-sentinel-docker-backend-design.md`, 2 consecutive Approve,
rounds 4+5). This plan adds only execution mechanics — do not re-derive design decisions here.

## Scope confirmation

**Revised during implementation (superseding the original "two repos, two PRs" scope
below)**: what was W0 (a standalone `monorepo-etc` Komodo Stack for docker-socket-proxy) is
now folded into W4, entirely within `monorepo`. Discovered while implementing: the design's
§3.2 premise ("first instance of this pattern in the fleet") was wrong — two
docker-socket-proxy instances already exist (`infra-prometheus` VOIP-1402,
`infra-loki` VOIP-1423), and both run the proxy as a **sidecar inside the consumer's own
compose file** on a dedicated `internal: true` network, not as a shared Stack on
`production`. `infra-loki` explicitly declined to reuse `infra-prometheus`'s existing proxy
for exactly this reason (blast-radius: on `production`, `docker inspect` via the proxy would
be reachable from any of ~50+ containers, not just the intended consumer). Design §3.2 is
corrected to match this precedent; see that section for the full reasoning. Net effect:
**one repository, one PR** — no monorepo-etc changes, no cross-repo sequencing.

- **`monorepo`** (this worktree, only): `bin-sentinel-manager` Docker backend rewrite
  (including its own embedded docker-socket-proxy sidecar, W4), `bin-call-manager`
  consumer-side change, `.circleci/config_work.yml` deploy job.

## Waves

### W0 — retired, folded into W4

Originally: a standalone `monorepo-etc` Komodo Stack. Superseded — see "Scope confirmation"
above. The research already done before the correction (digest pinning, the full explicit
ACL variable enumeration including the `PING=1`/`VERSION=1` correction, the read-only socket
mount) carries forward unchanged into W4's inline sidecar; only the deployment *shape*
changed (embedded sidecar + internal network, not a separate Stack on `production`).

### W1 — bin-sentinel-manager: new `models/container` package (additive, `models/pod` stays)

**Round-1 plan review correctly caught two errors here**: (a) this wave was labeled
"additive, no wiring yet" but originally also deleted `models/pod` — which
`pkg/monitoringhandler/run.go` (deleted only in W2) and six `bin-call-manager` files
(`subscribehandler/{main.go,sentinel_manager.go,sentinel_manager_test.go}`,
`callhandler/{event.go,main.go,mock_main.go}`, all migrated in W3) still import at this
point in the sequence; the tree would not build between W1 and W3. (b) the go.mod cleanup
claim below was backwards. Both fixed:

- `models/container/main.go`: `Event` type (design §3.5: `ContainerName`, `Service`,
  `AsteriskID`), `EventTypeContainerStarted`/`EventTypeContainerDied` constants.
- `EventSubscriptionID() string` on `*Event`: returns `AsteriskID` directly — **now decided
  at the design level, not plan level**, see design §6's second resolved item (added at this
  plan-review round: call-manager's existing subscription binds via the wildcard pattern
  `sentinel-manager.pod.*.deleted`, verified in `binding_golden_test.go`, so populating a
  real address here is additive and breaks nothing on the consumer side).
- Compile-time assertion: `var _ eventtopic.SubscriptionIdentifier = (*Event)(nil)` in a
  sibling `_test.go`.
- Behavioral test: mutation-checked (distinct `AsteriskID` values resolve correctly; empty
  `AsteriskID` resolves to `""`, not a panic).
- `models/pod/` is **NOT deleted in this wave** — it stays, unmodified, until every consumer
  (W2's `pkg/monitoringhandler`/`run_test.go`, W3's six `bin-call-manager` files) has
  migrated off it. The delete moves to the end of W3, once nothing in either service
  references it (verify with
  `grep -rl "bin-sentinel-manager/models/pod" --include="*.go" --exclude-dir=vendor .`
  returning empty before deleting).

### W2 — bin-sentinel-manager: Docker Events backend (replaces `pkg/monitoringhandler`)

- New dependency: `github.com/docker/docker/client` (+ its transitive deps) added via
  `go get`, not hand-edited into `go.mod`.
- New dependency: `github.com/go-redis/redis/v8` — **confirmed via round-1 plan review** as
  the fleet's already-standard choice (`bin-call-manager/pkg/cachehandler/main.go`; note
  upstream has since archived this in favor of `github.com/redis/go-redis/v9`, but matching
  the existing in-repo convention takes precedence over adopting a newer major version in a
  single service — a fleet-wide bump, if ever wanted, is its own separate change).
- `internal/config/config.go`: add `DockerSocketProxyAddress`, `RedisAddress` (or reuse
  whatever env var name convention `bin-call-manager`'s cachehandler already uses for
  consistency — check before inventing a new one), watched container name-prefix patterns
  (keep these as compile-time constants like today's K8s selectors, per design §3.1 — the
  design explicitly chose not to make this runtime config).
- `pkg/dockerwatchhandler/` (new package name — `monitoringhandler` is K8s-flavored
  terminology, rename for clarity since this is a full rewrite, not a patch):
  - `main.go`: interface + constructor (`requestHandler`/`notifyHandler` deps unchanged from
    today's `monitoringHandler`, plus a new Redis client dependency).
  - `state.go`: the per-container-name state table (design §3.3) — `map[string]*containerState`
    guarded by a `sync.Mutex` (design §3.3's "single goroutine or mutex-per-name" — a single
    mutex over the whole map is simpler and sufficiently fast at this cardinality;
    per-name mutexes would be premature optimization for ~6-10 entries).
  - `boot.go`: step 0 — list+inspect running watched containers at startup, seed the table,
    run one immediate refresh pass (design §3.3 step 0).
  - `refresh.go`: the 10s background loop — Redis SCAN, freshness filter
    (`remaining >= 24h - 12min`), sticky-last-known update semantics (design §3.3 step 2,
    the round-3/round-4-reviewed core mechanism — implement exactly as specified, this is
    the part with the most review scrutiny already behind it).
  - `events.go`: Docker Events API stream consumption (`type=container`,
    `event=start,die`, name-pattern filter), `since`-cursor reconnect (design §3.4), flap
    damping (design §3.4 — >3 restarts/60s per container name → WARN + skip), publish path
    (`container.Event` via `notifyHandler.PublishEvent`, `WithGlobalTopicPublish()`
    preserved from today's `cmd/sentinel-manager/main.go`).
  - Prometheus metrics: rename `sentinel_manager_pod_state_change_total` →
    `sentinel_manager_container_state_change_total` (labels: `container_name`, `service`,
    `state`) — a metric rename, not additive, since "pod" is equally misleading here; add
    the two new counters from design §3.3 (unresolved-id-published,
    resolved-entry-lost-fresh-candidate), both labeled by `container_name` per round-4
    review's polish item.
  - **`monitoring/grafana/dashboards/sentinel-manager.json` — caught by round-2 plan
    review, was missing entirely.** Four panels query the old metric name (lines 158, 192,
    219, 246 on `main`), two of them (`by (namespace)`, `by (pod)`) grouped by labels this
    rename deletes outright. Without updating this file in the same wave, the dashboard
    silently goes blank the moment this PR deploys — no error, just empty panels, easy to
    miss until someone needs it during an actual incident. Update all 4 panel queries to
    `sentinel_manager_container_state_change_total`, and the two label-grouped panels to
    `by (service)`/`by (container_name)` respectively (matching the new label set above).
    While in this file, also update the two panels' `legendFormat` templates
    (`"{{namespace}}"` → `"{{service}}"`, `"{{pod}}"` → `"{{container_name}}"` — otherwise
    the legend renders empty even though the data itself is correct) and any panel title
    still saying "Pod" (round-3 review flagged both as sitting immediately next to the lines
    already being edited — cheap to catch now, easy to miss as a follow-up).
- `cmd/sentinel-manager/main.go`: replace `runMonitoring`'s K8s selector map with the
  Docker name-prefix list; wire the new Redis client and docker-socket-proxy HTTP client
  into `dockerwatchhandler.NewDockerWatchHandler(...)`.
- `cmd/sentinel-control/`: the debugging CLI's `pod list`/`pod get` subcommands become
  `container list`/`container get` against the Docker backend (or the proxy's
  `/containers/json` directly) — keep the CLI's existing JSON-stdout/logs-stderr contract.
- Delete `k8s/` deployment manifests? **No — design §2 explicitly defers this to a
  follow-up ticket.** Leave `k8s/` in place, untouched, in this PR (it's already fully dead
  now that CI's release job is gone; removing it is cheap cleanup but out of THIS ticket's
  scope per the design's own boundary — resist scope creep here even though it would be a
  one-line temptation while already touching this service).
- `go.mod`: `k8s.io/client-go`, `k8s.io/api`, `k8s.io/apimachinery` become unused once W2's
  `pkg/monitoringhandler` and W3's final `models/pod` deletion land. **Round-1 plan review
  correctly caught that the previous version of this bullet was factually wrong**: `go mod
  tidy` (step 1 of root CLAUDE.md's *mandatory* verification workflow, run in W6) removes
  unused imports from `go.mod` automatically — it is not optional and cannot be skipped to
  "leave them in." **Accept the `go.mod`/`go.sum` delta in this PR** (these three modules
  drop out); design §7 only defers deleting the `k8s/` *manifest directory* (`k8s/*.yml`,
  which `go mod tidy` has no opinion on) to a follow-up, not the go.mod entries, which the
  mandatory workflow removes as an automatic side effect of this change regardless of intent.
  Call this out explicitly in the PR description as an expected, reviewed part of the diff.
- **Discovered during implementation (not anticipated by this plan): adding
  `github.com/docker/docker` bumps MVS-selected shared indirect dependencies fleet-wide**
  (`golang.org/x/net`, `golang.org/x/crypto`, `golang.org/x/sys`, `golang.org/x/text`,
  `google.golang.org/protobuf`, `go.yaml.in/yaml/v3`). Because every service reaches
  `bin-sentinel-manager` through the monorepo's local `replace` directives, `go mod tidy`
  (mandatory, per above) propagates these version bumps to all 38 services' `go.mod`/
  `go.sum` — 36 of them otherwise fail `go mod vendor` (`"updates to go.mod needed"`), which
  every Dockerfile runs during image build, making this a landmine for their next unrelated
  change if left untouched. Verified empirically as newly-introduced (not pre-existing drift)
  via a clean-worktree control comparison against `origin/main`. **Decision: keep in this
  PR** (not split into a separate PR — a standalone tidy is meaningless without the docker
  dependency that necessitates it, and would itself be re-reverted by the next unrelated
  tidy; also not avoided by dropping `github.com/docker/docker` — that would reopen design
  §3.1's already-approved library choice purely to avoid a mechanical diff). The per-service
  diff is purely six indirect version lines plus matching `go.sum` entries — no direct
  dependency or code changes in any of the 36 services. Call this out explicitly in the PR
  description alongside the `k8s.io/*` removal, as its own clearly-labeled item so reviewers
  don't mistake ~72 touched files for accidental scope.

### W3 — bin-call-manager: consumer-side change

- `pkg/callhandler/event.go`: `EventSMPodDeleted(ctx, p *smpod.Pod)` →
  `EventSMContainerDied(ctx, c *smcontainer.Event)` (design §3.6 — renamed, not kept, since
  we're touching the whole signature anyway and "pod" would be actively wrong here).
  Filter changes from `p.Namespace == asteriskPodNamespace && p.Labels["app"] ==
  asteriskPodLabelApp` to `c.Service == "asterisk-call"` (the `Service` field is exactly the
  filter target now, no indirection needed). **Add the empty-id guard the design's §3.3/§3.6
  review rounds required**: `if c.AsteriskID == "" { log...; return nil }` before calling
  `RecoveryStart` — this is new behavior, not a preserved no-op (round-2/round-3 review
  confirmed today's code has no such guard).
- `pkg/subscribehandler/main.go`: update the `sentinel-manager.pod.deleted` binding pattern
  to the new event type/routing key (design §3.5's rename); update the type-switch case
  (`m.Publisher == ... && m.Type == smpod.EventTypePodDeleted` → the container equivalent).
- `pkg/subscribehandler/sentinel_manager.go`: rename `processEventSMPodDeleted` →
  `processEventSMContainerDied`, update its import/unmarshal target to `smcontainer.Event`.
- `pkg/callhandler/main.go`, `mock_main.go`: interface method rename, mock regen
  (`go generate`).
- **Test files — round-1 plan review corrected this list; it was incomplete and one item
  was mischaracterized:**
  - `bin-call-manager/pkg/callhandler/event_test.go` currently covers only
    `Test_EventCUCustomerFrozen`/`Test_EventCUCustomerDeleted` — `EventSMPodDeleted` has
    **zero existing test coverage**. The new `EventSMContainerDied` test is net-new
    coverage, not a rewrite; per CLAUDE.md's aggressive-testing convention, cover the
    happy path, the `Service != "asterisk-call"` filter-skip, AND the new empty-id guard
    (three cases minimum, not one).
  - `bin-call-manager/pkg/subscribehandler/binding_golden_test.go` pins the literal
    `"sentinel-manager.pod.*.deleted"` string — **must be updated deliberately** to
    `"sentinel-manager.container.*.died"` in this wave, or CI fails on an intentional
    change that looks like a regression.
  - `bin-call-manager/pkg/subscribehandler/sentinel_manager_test.go`
    (`Test_processEvent_processEventSMPodDeleted`) — rewrite for the renamed handler.
  - `bin-sentinel-manager/models/pod/{event_test.go,main_test.go}` and
    `bin-sentinel-manager/pkg/monitoringhandler/run_test.go` — deleted alongside their
    packages in this wave (W2 already deletes `pkg/monitoringhandler`; this wave is where
    `models/pod` — the whole directory, both test files included — and its test finally go,
    per W1's corrected sequencing note).
  - **`models/pod/` deletion happens HERE, at the end of W3** (not in W1 — see W1's
    corrected note): by this point `pkg/monitoringhandler` (W2) and all six
    `bin-call-manager` consumers (this wave, above) have migrated off it. Verify with
    `grep -rl "bin-sentinel-manager/models/pod" --include="*.go" --exclude-dir=vendor .`
    returning empty immediately before deleting. Move
    `models/pod/routingkey_golden_test.go` → `models/container/routingkey_golden_test.go`
    at this point too (W1 created the new package but intentionally left its golden test
    for this wave, since the golden test's whole point is pinning the keys real publish
    sites produce, and the publish sites don't exist until W2).

### W4 — CI/CD

- `.circleci/config_work.yml`: add `bin-sentinel-manager-deploy` job to the
  `bin-sentinel-manager` workflow (currently ends at `build`), mirroring the
  `komodo-api-deploy.sh` pattern used by the other 32 services (render image tag → deploy).
  Remove the stale "release job removed (VOIP-1405)" comment block now that a real deploy
  job is back.
- New `bin-sentinel-manager/komodo/docker-compose.yml`: single replica (design §4) for the
  `sentinel-manager` service itself, **plus an embedded `docker-socket-proxy` sidecar** (see
  design §3.2's corrected reasoning and the "Scope confirmation" section above — this
  replaces the retired W0). Concretely:
  - New `internal: true` network in this compose file (e.g. `sentinel-docker-internal`),
    joined only by `sentinel-manager` and the sidecar — not `production`.
  - Sidecar service (suggested key: `docker-socket-proxy`, or
    `sentinel-docker-socket-proxy` if a more distinctive name reads better next to
    `infra-prometheus`'s `docker-socket-proxy`/`infra-loki`'s
    `loki-docker-socket-proxy` in fleet-wide dashboards — implementer's call): image
    `tecnativa/docker-socket-proxy:v0.5.0@sha256:1f5038b54f06c3e18422902cf00ba21803d1c97805aae032e5e6673d532d3459`
    (digest independently resolved and cross-checked against `infra-loki`'s own pin during
    the retired W0's research — reuse it rather than re-resolving). Explicit 29-variable ACL
    (`EVENTS=1`, `CONTAINERS=1`, `PING=1`, `VERSION=1`, everything else `0` — the retired
    W0's `komodo/docker-compose.yml` on the abandoned `monorepo-etc` branch
    `VOIP-1418-Docker-socket-proxy` has the full enumerated list with rationale comments per
    variable; copy the pattern). Read-only socket mount
    (`/var/run/docker.sock:/var/run/docker.sock:ro`). On the new internal network only, no
    published port.
  - `sentinel-manager`'s `DockerSocketProxyAddress` config resolves to the sidecar's
    in-compose service name over the internal network (e.g.
    `http://docker-socket-proxy:2375`) — this needs NO Komodo `[[BIN_MANAGER__...]]`
    variable at all, since it never leaves this one compose file.
  - Redis address DOES need the standard `[[BIN_MANAGER__...]]` interpolation convention —
    confirm exact variable naming against `infra-secret`'s existing
    `bin-manager/secrets.enc.yml` schema before inventing new names.
- `bin-sentinel-manager/Dockerfile`: confirm it doesn't assume `k8s/` RBAC-mounted service
  account tokens at runtime (it shouldn't — that was `k8s/rbac/*.yml` cluster-side config,
  not baked into the image — but verify).

### W5 — Docs

- `bin-sentinel-manager/CLAUDE.md`: replace the "CRITICAL: in-cluster only" /
  "CRITICAL: RBAC required" sections with the Docker-backend equivalents (docker-socket-proxy
  dependency, no RBAC concept). Update "Key implementation facts".
- `bin-sentinel-manager/docs/architecture.md`, `docs/domain.md`, `docs/operations.md`:
  re-extract via `docs/reference/extractor.sh bin-sentinel-manager` per root CLAUDE.md's
  service-docs-sync rule (this PR's `pkg/listenhandler`-equivalent routing table doesn't
  exist for this A2 service, but the events/config/domain sections do change materially).
- `bin-call-manager/docs/architecture.md` (or wherever its subscribeTargets are documented):
  same sync rule for the renamed event.
- `docs/reference/rabbitmq-queues-reference.md` (root, if this queue/event is catalogued
  there): update the sentinel-manager event-type entries.

### W6 — Global verification + evidence

Per root CLAUDE.md's mandatory verification workflow, run for **both** touched services
(`bin-sentinel-manager`, `bin-call-manager`) before any commit:

```
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Additional targeted checks:
- New golden test (`models/container/routingkey_golden_test.go`) passes with the new
  expected keys (this is an intentional key-string change per design §3.5 — do not treat a
  diff here as a regression to "fix" back).
- `dockerwatchhandler`'s state-table logic gets dedicated unit tests for: sticky-last-known
  (a refresh pass with no fresh candidate does not clear a resolved id), the freshness-filter
  boundary (`remaining` just above/below `24h-12min`), boot-time reconciliation seeding, and
  the flap-damping threshold — this is the highest-scrutiny part of the design across 5
  review rounds and deserves the highest test density in the implementation.
- `bin-call-manager`'s new empty-id guard gets an explicit test (design review flagged this
  as previously entirely absent — new coverage, not preserved coverage).
- Full cross-service build sweep (both services + anything importing
  `bin-sentinel-manager/models/pod` — check via `grep -rl "bin-sentinel-manager/models/pod"`
  before deleting it at the end of **W3** (round-2 review caught this line still said "W1",
  an orphaned reference left over from the W1/W3 resequencing fix — the deletion point is
  W3, not W1), since a stale import elsewhere would only surface as a build failure in that
  OTHER service, not in sentinel's own `go test`).

## Explicitly out of scope for this PR (per design §7, restated for the implementer)

- Deleting `bin-sentinel-manager/k8s/` manifests and `k8s.io/*` go.mod entries.
- Removing the dead `ASTERISK_ID` env var (VOIP-1365).
- Any change to `RecoveryStart`/Homer/PJSIP redial logic itself.
- Restart-vs-recreate MAC-stability empirical test (design §3.4 — informs flap-damping
  tuning, not a blocker; do the cheap kill-signal check early in W2 if time allows, but
  don't let it gate the PR).

## PR sequencing

1. W0 (monorepo-etc) — separate PR, separate repo, land or at least open first.
2. W1 → W2 → W3 → W4 → W5 → W6 (monorepo) — one PR, per root CLAUDE.md's default
   single-PR-per-task rule (multi-file, multi-wave, but one logical change: "give
   sentinel-manager a working Docker backend and restore its deploy path").

## Results (implementation, 2026-09-01)

W1-W6 implemented. W0 was re-scoped mid-implementation (see "Deviations" below).

| Wave | Outcome |
|---|---|
| W1 | `models/container` (Event, event-type + service constants, `EventSubscriptionID` returning the asterisk-id). Added `models/asteriskaddress` — not in the plan, see deviations. |
| W2 | `pkg/dockerwatchhandler` (`main.go`/`state.go`/`refresh.go`/`boot.go`/`events.go`), `pkg/cachehandler` (read-only Redis reverse scan), rewritten `cmd/sentinel-manager` + `cmd/sentinel-control`, extended `internal/config`, deleted `pkg/monitoringhandler`, renamed the Prometheus counter and updated `monitoring/grafana/dashboards/sentinel-manager.json` (full grep pass: 4 queries, 3 legendFormats, 5 panel titles). |
| W3 | `bin-call-manager`: `EventSMPodDeleted` → `EventSMContainerDied` with the new empty-id guard, subscribe dispatch + binding pattern, both golden tests, mock regen. `models/pod` deleted at the end of the wave; its golden test moved to `models/container/routingkey_golden_test.go`. |
| W4 | `bin-sentinel-manager-deploy` job + workflow edge in `.circleci/config_work.yml`; new `bin-sentinel-manager/komodo/docker-compose.yml` (sentinel-manager + socket-proxy sidecar). Dockerfile verified to need no change. |
| W5 | `bin-sentinel-manager/{CLAUDE.md,README.md,docs/*.md}`, `bin-call-manager/docs/architecture.md`, `docs/reference/rabbitmq-queues-reference.md`, `docs/architecture/service-dependency-graph.md`. |
| W6 | Full 5-step workflow green for both services, 0 lint issues. Plus a fleet-wide build sweep (38 modules). |

### Deviations from the plan

1. **W0 re-scoped by the coordinator mid-implementation.** The standalone `infra-docker-socket-proxy` Komodo Stack in monorepo-etc was cancelled. The proxy is now a sidecar inside `bin-sentinel-manager/komodo/docker-compose.yml`, on a Stack-local `internal: true` network joined only by sentinel-manager and the proxy — matching the shape monorepo-etc's `infra-prometheus` (VOIP-1402) and `infra-loki` (VOIP-1423) already use. Design §3.2's claim that this pattern had "no existing reference to copy" was factually wrong; both of those Stacks predate it and both deliberately kept their proxy OFF `production` for the blast-radius reason. No cross-repo dependency remains.

2. **`models/asteriskaddress` added (not in the plan).** The Redis key shape and the freshness rule (`TTL`, `RefreshInterval`, `FreshnessMargin`, `IsFresh`) needed a home shared by `pkg/cachehandler` and `pkg/dockerwatchhandler`. Putting the boundary rule in a models package makes it unit-testable in isolation, which is where the freshness-boundary tests live.

3. **`miniredis` added as a test dependency** for `pkg/cachehandler`, matching the precedent in `bin-webhook-manager`, `bin-contact-manager`, and `bin-api-manager`. The SCAN cursor loop and the TTL read are exercised against real Redis semantics rather than a mock's idea of them.

4. **Fleet-wide `go.mod`/`go.sum` tidy (36 services) — the largest unplanned consequence.** `github.com/docker/docker` raises the MVS-selected versions of `golang.org/x/{net,crypto,sys,text}`, `google.golang.org/protobuf`, and `go.yaml.in/yaml/v3`, and the monorepo's local `replace` directives propagate that to every module. Left untouched, 36 services fail both `go build ./...` and `go mod vendor` — which every Dockerfile runs — so their image builds would break on their next unrelated change. Verified NOT pre-existing via a clean detached worktree at `origin/main`. The per-service diff is only those six indirect version lines plus matching `go.sum` entries; no direct dependency or code changes anywhere.

5. **`k8s.io/*` removal is wider than sentinel.** `bin-call-manager` also drops `k8s.io/api`, `k8s.io/apimachinery`, `k8s.io/klog/v2`, `k8s.io/kube-openapi`, `k8s.io/utils`, and `sigs.k8s.io/*` — it was pulling them transitively through `bin-sentinel-manager/models/pod`. Expected and desirable.

### Still out of scope, as planned

- `bin-sentinel-manager/k8s/` manifests are untouched (design §7), even though now fully dead.
- The dead `ASTERISK_ID` env var (VOIP-1365).
- The restart-vs-recreate MAC-stability empirical check (design §3.4) — not run; it informs flap-damping tuning only, and the shipped threshold (3 deaths / 60s) is the design's stated starting point.

## Code review round 1 — fixes applied

code-reviewer + security-reviewer, run in parallel. Both independently flagged the same top
finding (#1). All 8 items addressed.

| # | Severity | Fix |
|---|---|---|
| 1 | HIGH (both) | `runService` now returns `error`; `run` propagates it; `main` turns it into `os.Exit(1)`. Docker-client-creation and Redis-connect failures return errors too, instead of log-and-return. The documented "exit non-zero / crash-loop" promise in design §3.2, `boot.go`, `events.go`, and the compose file is now actually implemented. |
| 2 | HIGH | `runEventLoop` returns `error` and gives up after `maxConsecutiveEmptyStreams` (20 ≈ 1 min) attempts that deliver no events, feeding the same fail-loud exit path. `consumeEvents` now reports whether an attempt delivered anything; a delivering attempt resets the budget. New counter `sentinel_manager_container_event_stream_reconnect_total{result="delivered"\|"empty"}` makes sub-threshold flapping alertable. A healthy stream blocks rather than returning, so an idle fleet never increments `empty`. |
| 3 | MEDIUM | New Grafana row "Recovery Health (leading indicators)" with three panels: unresolved-asterisk-id deaths, refresh misses, and event-stream reconnects. Each carries a `description` explaining what a non-zero value means and what to check. Existing rows shifted down; JSON re-validated. |
| 4 | MEDIUM | The same-entry id-change branch now **keeps the existing id** and logs WARN with old/new context, instead of silently overwriting. Conservative because the old id was resolved while this generation was demonstrably alive, whereas adopting an unexplained new one risks firing recovery against a different, still-live instance. Rationale recorded in a code comment; three table cases plus a repeated-pass stability test added. |
| 5 | MEDIUM | Residual-risk text corrected in `komodo/docker-compose.yml`, `CLAUDE.md`, and `docs/operations.md`. The proxy ACL is path-prefix based, so `CONTAINERS=1` also permits `/containers/{id}/archive`, `/export`, `/logs`, and `/attach/ws` — near-total read access to every container's data on the host, not "env vars via inspect". Network scope is stated as the entire mitigation. |
| 6 | LOW | `EventSMContainerDied` gets a `c == nil` guard. `json.Unmarshal` of a literal `null` into a `**Event` succeeds and leaves the pointer nil, which previously panicked the subscribe loop. Covered from both sides (handler-level nil case, and a `null` payload through `processEvent`) and mutation-checked. |
| 7 | LOW | `docs/conventions/naming.md` example updated to `smcontainer "monorepo/bin-sentinel-manager/models/container"`. Both `.docs-gen` extracts regenerated via `docs/reference/extractor.sh` — they picked up the new config flags and the renamed metric set. |
| 8 | LOW | `flapTracker.Forget` removed along with its tests; its one concurrency-test use replaced with a second `Record` call. |

Not fixed, by the reviewer's own scoping: unauthenticated Redis on the `production` network lets
any container there write `asterisk.<id>.address-internal` and steer `RecoveryStart`. Pre-existing
trust assumption (call-manager already trusted these keys); separate ticket material.

### Verification of the fixes

- Mutation-checked, each confirmed failing when the fix is reverted: the empty-id guard (#6, panics),
  the keep-existing-id branch (#4, 4 subtests fail), the give-up condition (#2, 2 tests fail).
- Fail-loud exit code (#1) confirmed at `exit 1` by running the built binary down the cobra
  `RunE`/`Execute` error path — the same path `runService`'s error now takes. **Caveat:** with an
  unreachable RabbitMQ, `sockHandler.Connect()` blocks before the docker/redis checks are reached,
  so a full end-to-end fail-loud run needs a reachable broker. That blocking is shared
  `bin-common-handler` behavior identical in all 38 services, not introduced here.
- `cmd/` has no test files in this repo (true of both cmds before this change), so the
  `runService` → `run` → `main` wiring is covered by inspection plus the binary check above rather
  than by a unit test; `dockerwatchhandler.Run`'s own error return is unit-tested.
