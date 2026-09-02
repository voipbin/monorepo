# Fleet-wide Go Toolchain Upgrade 1.25 → 1.27 — Design

VOIP-1447. Triggered by [VOIP-1446](https://voipbin.atlassian.net/browse/VOIP-1446)
(upgrade `bin-sentinel-manager`'s `k8s.io/{api,apimachinery,client-go}` off the
`v0.36.0-alpha.0` pin it has carried since VOIP-1418). This document covers the
fleet-wide consequence that upgrade turned out to have, not the k8s.io bump
itself — that resumes as VOIP-1446 once this lands.

## Context

Every GA release of `k8s.io/client-go` from `v0.36.0-beta.0` onward (including
the smallest possible GA step, `v0.36.0`, and the latest, `v0.37.0`) declares
`go 1.26.0` in its own `go.mod` — confirmed by fetching each candidate
version's `go.mod` directly from `proxy.golang.org`. Only the alpha
pre-releases (`v0.36.0-alpha.0`, `v0.36.0-alpha.1`) stay on `go 1.25.0`. There
is no GA version reachable without raising `bin-sentinel-manager`'s own `go`
directive past the monorepo's current fleet-wide `go 1.25.3`.

That alone would be a two-service change (`bin-sentinel-manager`,
`voip-asterisk-proxy` — the only two modules with a direct `k8s.io/*`
dependency). It is not, because of how this monorepo's local `replace`
directives propagate `go` directive requirements:

- `bin-call-manager` has a real (non-orphaned) `replace monorepo/bin-sentinel-manager
  => ../bin-sentinel-manager` backed by production imports of
  `monorepo/bin-sentinel-manager/models/container`
  (`pkg/callhandler/main.go`, `pkg/callhandler/event.go`,
  `pkg/subscribehandler/main.go`, `pkg/subscribehandler/sentinel_manager.go`).
  Empirically confirmed in the VOIP-1446 worktree: bumping only
  `bin-sentinel-manager/go.mod`'s `go` line to `1.27.1` and running
  `go build ./pkg/callhandler/...` inside `bin-call-manager` fails immediately
  with `module ../bin-sentinel-manager requires go >= 1.27.1 (running go
  1.25.3)`. `GOTOOLCHAIN=auto` does **not** rescue this — Go's toolchain
  auto-switch only activates on the *main* module's own `go`/`toolchain`
  directive, never transitively through a `replace`d dependency's stricter
  requirement.
- `bin-call-manager` is itself `replace`d (with a matching `require`, i.e. not
  orphaned) by **34 of the other 38 Go modules** in the monorepo, including
  `bin-common-handler`. (A 35th, `voip-kamailio-proxy`, also carries a
  `replace monorepo/bin-call-manager` line but has no matching `require` —
  Go ignores a replace for a module absent from the graph, so that one edge
  propagates nothing. `voip-kamailio-proxy` is still pulled into scope
  regardless, through its real `replace`+`require` on `bin-common-handler`
  below.) `bin-common-handler` — the shared library — in turn is `replace`d by
  **36 of the other 38 modules** (37 files match `replace
  monorepo/bin-common-handler`, but one of those is `bin-common-handler/go.mod`
  itself — a harmless self-replace — so the real dependent count is 36).
  Confirmed via `grep` and, for the highest-risk cases, by directly
  reproducing the same build failure in `bin-common-handler`,
  `bin-api-manager`, `bin-flow-manager`, and `bin-storage-manager` (the last
  one via an `// indirect` require only — module-graph pruning does not save
  it; the failure is identical). `bin-common-handler` and `bin-call-manager`
  also `replace`/`require` **each other** — a genuine local dependency cycle —
  which on its own rules out any "bump in dependency order" sequencing for
  those two, independent of the other 34.
- A workaround that keeps the *toolchain* current fleet-wide but leaves each
  service's own `go` directive untouched does not work either: rebuilding
  `bin-common-handler` (still declaring `go 1.25.3`) with an actual `go1.27.1`
  binary, against a `bin-call-manager` bumped to `go 1.27.1`, still fails under
  the default `-mod=readonly` mode with `go: updates to go.mod needed; to
  update it: go mod tidy` — i.e. the `go` directive lines across the replace
  graph must be mutually consistent for a normal (non-`-mod=mod`) build to
  succeed, independent of which toolchain binary is running it.

So the two-service k8s.io bump forces a `go.mod` `go`-directive bump on every
module that (transitively, through local `replace`s) reaches
`bin-sentinel-manager` — which is effectively the whole fleet.
**대표님 결정: since the fleet is forced to touch nearly every `go.mod` anyway,
bump all active Go modules in the monorepo to the same target in one
coordinated change**, rather than trying to draw an artificial boundary that
the module graph doesn't actually respect.

## Scope

**38 of the 39** `bin-*`/`voip-*` Go modules (every directory with a
`go.mod`, except `bin-trigger-sender` — see below; `bin-dbscheme-manager` is
Python and is out of scope regardless, it has no `go.mod`). Full list
captured at `git ls-files bin-*/go.mod voip-*/go.mod` — of the 36
Dockerfile-bearing services in scope (37 fleet-wide minus the excluded
`bin-trigger-sender`), 35 are on `golang:1.25-alpine`, one
(`bin-pipecat-manager`) on `golang:1.25-bookworm`, and two Go modules
(`bin-common-handler`, `bin-openapi-manager`) have no Dockerfile at all (not
deployed as their own container image; `bin-dbscheme-manager` also has no
Dockerfile but is excluded above as it has no `go.mod` either).
`bin-openapi-manager` is the only in-scope module that does *not*
`replace`/`require` `bin-common-handler` — it has no local-module dependency
chain into this at all, but is included for fleet consistency (same `go`
directive floor as everything else, same CircleCI executor image).

**`bin-trigger-sender` is explicitly OUT of scope** — resolved as dead code
with zero production deployment (see "Sequencing" below for the investigation)
rather than bumped for consistency. Its `-test` CI job keeps working
unmodified against the shared, upgraded `go_image` anchor without any
`go.mod` change of its own, since it was never part of the replace-chain that
forces this upgrade in the first place.

**Target Go version: 1.27.1** — confirmed the latest stable release
(`stable: true` on `https://go.dev/dl/?mode=json`) as of this check;
`go1.26.8` is also stable but older. 대표님 explicitly chose "latest" over the
dependency's bare minimum (`go 1.26.0`). Re-verify at implementation time in
case a newer patch/minor has shipped since this document was written — do not
silently use a stale number if `go1.27.2` or `go1.28.0` has since gone stable.

## What actually needs to change, per service

1. **`go.mod`**: `go` directive `1.25.3` → `1.27.1` (or the re-verified latest
   stable) **plus a new `godebug default=go1.25` line** (see Risks below for
   why — this pins runtime GODEBUG defaults to 1.25 semantics while still
   raising the language/stdlib floor to 1.27; both lines land together, not
   as a separate follow-up). Whatever `go mod tidy` does beyond that (dependency
   graph churn from the raised floor) is expected and should be committed as-is
   — this mirrors the "purely mechanical indirect-version churn" ripple already
   accepted in VOIP-1418's PR #1240 across 36+ services.
2. **`Dockerfile`** (35 in-scope services on `-alpine`, 1 on `-bookworm`, 2
   with none — `bin-trigger-sender`'s Dockerfile is untouched, per Scope):
   `FROM public.ecr.aws/docker/library/golang:1.25-*` → `golang:1.27.1-*`
   (matching each service's existing base — alpine stays alpine,
   `bin-pipecat-manager`'s bookworm stays bookworm). **Pin the exact patch
   version (`1.27.1-alpine`), not the floating minor tag (`1.27-alpine`).**
   The floating tag can move independently of `go.mod`'s exact `go 1.27.1`
   directive — if `1.27-alpine` ever resolves to an earlier patch than what
   `go.mod` requires, every image build fails with `go.mod requires go >=
   1.27.1`, and the official `golang` image sets `GOTOOLCHAIN=local`, so
   there is no auto-download rescue inside the build. Pinning the exact patch
   keeps the Dockerfile and `go.mod` mechanically in sync. Confirmed
   `public.ecr.aws/docker/library/golang:1.27-alpine` resolves (HTTP 200 via
   the ECR Public registry API, the actual registry these Dockerfiles pull
   from — not just Docker Hub, which was checked first and is a different
   mirror); re-verify the exact `1.27.1-alpine` and `1.27.1-bookworm` tags
   similarly at implementation time.

   **Tradeoff, recorded deliberately — and stated accurately about the
   repo's CURRENT state, not an idealized one.** Every one of the 37
   Dockerfiles today uses a floating MINOR tag (`golang:1.25-alpine` /
   `golang:1.25-bookworm`), which moves across `1.25.x` patch releases
   automatically on rebuild — this design is what introduces exact-patch
   pinning to the fleet's Dockerfiles, not a continuation of an existing
   pattern. (`go.mod`, separately, has always pinned an exact patch — `go
   1.25.3`, not `go 1.25` — and continues to; that half of the convention
   argument holds.) So the mismatch hazard this design guards against (a
   floating `1.27-alpine` resolving to an earlier patch than `go.mod`
   requires) has technically existed since day one under `1.25-alpine` + `go
   1.25.3` and has never fired in practice — this isn't a new risk being
   introduced, just one being closed off deliberately rather than left latent
   for a second time.

   The real, current cost of the choice made here: moving Dockerfiles to
   exact pins gives up the free ride floating tags currently provide — every
   future Go 1.27.x patch release, including security patches, now needs an
   explicit fleet-wide PR to adopt, rather than arriving automatically on the
   next image rebuild. Accepted anyway, for two reasons: it keeps the
   Dockerfile and `go.mod` mechanically self-consistent going forward (no
   silent drift between "what `go.mod` requires" and "what the build image
   actually provides"), and a future patch-version PR is exactly the same
   mechanical, low-risk, same-shape change this document already is — not a
   new kind of burden, just a recurring instance of this one.
3. **CircleCI (`.circleci/config_work.yml`)**: the shared
   `go_image: &go_image - image: cimg/go:1.25.3` anchor (38 alias usages
   across every `-test` job plus `bin-openapi-manager-validate`) becomes
   `cimg/go:1.27.1` in place — one edit, not a second parallel anchor, since
   after this design every service moves together. Confirmed `cimg/go:1.27.1`
   exists as a real Docker Hub tag. `gotestsum` (invoked directly in the
   `go-test` command with no explicit install step) is expected to ship
   pre-bundled in `cimg/go` convenience images per CircleCI's own convention;
   this is not independently verifiable via registry API alone and is an
   implementation-time empirical check — if `bin-*-test` fails on
   `gotestsum: command not found` after the anchor swap, that is the signal,
   not a design failure.
4. **Full verification workflow** per service (root CLAUDE.md, unchanged
   sequence): `go mod tidy && go mod vendor && go generate ./... && go test
   ./... && golangci-lint run -v --timeout 5m`. No exceptions for "trivial"
   services — a `go` directive bump has caused `go.sum` drift and generator
   output changes before and will again.

Explicitly **not** in scope for this document: the `k8s.io/*` dependency bump
itself for `bin-sentinel-manager`/`voip-asterisk-proxy`, or the
`patchPodAnnotation` signature widening / new `annotation_test.go` for
`voip-asterisk-proxy` — those resume under VOIP-1446 once this lands, as a
much smaller, now-unblocked 2-service change.

## Sequencing

**Atomic in the module graph, staged in deployment — these are two different
questions, and the first round of this design conflated them.**

### Module graph: one PR, no exceptions

A `go` directive bump has no meaningful per-service canary at the *build*
layer: the failure mode this document exists to describe (a `replace`d
dependency's `go` directive exceeding a consumer's) is triggered by the
*shape of the module graph on `main`*, not by which service is deployed when.
Landing the `go.mod`/Dockerfile changes in small batches over multiple PRs
would put `main` through a series of intermediate states where some modules
are bumped and their (not-yet-bumped) dependents are broken — exactly the
failure this document empirically reproduced, just self-inflicted on `main`
instead of caught in one PR's CI. `bin-common-handler` and `bin-call-manager`
`replace`/`require` each other, which independently rules out any
dependency-ordered rollout for those two alone.

**Therefore: one PR, all 38 in-scope modules' `go.mod` + Dockerfile + the
shared CircleCI anchor, landed atomically.** (`bin-trigger-sender` excluded —
see Scope above.) This does not mean one giant manual
edit — the mechanical per-service work (bump the two version strings, run the
5-step verification) is independent per module and safely parallelizable
(each is a separate Go module with no cross-module test execution), so
implementation fans out across services and only the final commit/PR is
single and atomic.

### Deployment: staged, per-service, exactly like the nearest precedent

This repo already ran a fleet-wide, build-time-only change across almost the
same 37 services — [docs/plans/2026-07-31-fleet-static-build-distroless-design.md](2026-07-31-fleet-static-build-distroless-design.md)
(VOIP-1277, static build + distroless runtime for every Go Dockerfile). That
design also concluded "one PR" for the exact same module-graph-consistency
reason, but explicitly did **not** treat one-PR-for-the-code as one-shot for
*production*:

> 서비스별 CircleCI build-approval 수동 승인 구조 유지. 한 번에 전 서비스를
> 배포하지 않고 1개씩 승인·확인·진행 (readinessProbe 부재로 인한 위험 완화
> 절차). 이미지 태그 단위 롤백 가능.

This design adopts the same discipline, with two corrections to what "the
same discipline" actually means here.

**First correction — the workflow shape is not uniform across all 39, and the
first draft of this section implicitly assumed it was.** Checked directly
against `.circleci/config_work.yml`:

- **33 services** genuinely follow `build-approval (manual) → <svc>-test →
  <svc>-build → <svc>-deploy`, each with its own `komodo/docker-compose.yml`
  and Komodo-based `-deploy` job. Only one job in the entire file
  (`migration-applied-checkpoint`) carries a `branches: only: main` filter —
  every one of these 33 `-deploy` jobs runs against production on approval
  regardless of merge state. A single PR approved 33 times in a row would be
  33 production cutovers onto a brand-new Go runtime, with no independent
  confirmation between them — see the staged plan below.
- **`voip-asterisk-proxy`, `voip-rtpengine-proxy`, `voip-kamailio-proxy`**
  (the data-plane proxies co-located with Asterisk/RTPEngine/Kamailio) have
  **no `-deploy` job in this repo's CI at all** — `voip-asterisk-proxy` is
  `build-approval → build` only; the other two are `build-approval → test →
  build`. Their images reach production through whatever mechanism already
  manages that fleet outside this repo's CircleCI automation (the
  `voip-kamailio-ansible`/infra tooling referenced elsewhere in this
  monorepo's project history) — this design does not control or change that
  path. It still rebuilds their images with the new Go toolchain, so whoever
  owns that out-of-band rollout needs to know a new image exists; naming that
  handoff explicitly here rather than assuming a `-deploy` job that doesn't
  exist.
- **`bin-common-handler`** (test-only, no build/deploy — it's the shared
  library, never its own container) and **`bin-openapi-manager`**
  (`build-approval → validate` only) have no deployable artifact at all;
  nothing to stage for them beyond the standard verification workflow.
- Not a workflow-shape exception but worth recording alongside these:
  **`bin-pipecat-manager`'s CI `-test` job is commented out**
  (`config_work.yml`'s `bin-pipecat-manager` workflow block has
  `# - bin-pipecat-manager-test:` disabled) — its actual CI shape is
  `build-approval → build → deploy`, not the full standard shape. Pre-existing,
  not introduced by this change, and doesn't affect this design's own plan
  since the mandatory LOCAL 5-step verification workflow already runs `go
  test ./...` for every service regardless of what CI does — but noted here
  for the same reason `bin-pipecat-manager` is already called out as the
  fleet's most toolchain-sensitive build elsewhere in this document.
- **`bin-trigger-sender`** — see the dedicated note below; its actual
  deployment status turned out to be unclear enough that it needs resolving
  before implementation, not just "re-verify later."

**Second correction — strict one-at-a-time for all 33 deployed services is
the wrong transplant of the precedent's discipline, not a wrong discipline
per se.** The distroless design's one-at-a-time rule was tied to a specific,
named hazard in that change: 32 of 34 services had no `readinessProbe`, so a
start-then-exit failure could let a Kubernetes rollout silently replace
healthy pods with dead ones — serialization gave a human the verification the
missing probe should have provided. Neither half of that premise holds
identically here: the fleet no longer runs Kubernetes rollouts (Komodo status
checks replace `kubectl rollout status` below), and the risk this change
introduces — a Go 1.27 runtime behavior change in TLS/`net/http`/crypto
defaults — is **homogeneous across services**, not service-specific the way
a Dockerfile's COPY/ENTRYPOINT differences were. The 30th serialized deploy
carries almost no more information than the 3rd; a strict 33-step serial
rollout would mostly be cost with no matching signal, and a rollout that
predictably gets abandoned partway through (because it takes days) is worse
than a right-sized batched one specified up front.

**Deployment plan**:
1. **Canary tier (serial, 2-3 services):** `bin-call-manager` (highest
   call-volume traffic; the module whose replace-chain position forced this
   whole change) plus one HTTP/TLS-facing service (`bin-api-manager` or
   `bin-hook-manager` — the actual risk surface a Go runtime bump touches).
   Approve, confirm healthy, wait a deliberate observation window (not just
   an instant health check — the class of regression this guards against,
   e.g. a TLS handshake default change, may not fail on the first request),
   before proceeding.
2. **Promotion criterion:** canary tier shows no new error-rate/restart
   signal for its observation window. If it does, stop, do not batch, treat
   it as this design's central risk having actually fired and re-plan.
3. **Batches of 3-5 thereafter** for the remaining ~30 deployed services,
   confirming each batch healthy before starting the next — not
   one-at-a-time, not all-at-once.
4. After each `<svc>-deploy` job (any tier), confirm the container is healthy
   via Komodo container status + `GetContainerLog` (this fleet is bare-metal
   Docker on bm-nyc-01 via Komodo, not Kubernetes — there is no `kubectl
   rollout status` equivalent to reach for). On any failure, roll back via
   image-tag redeploy (the previous `$CIRCLE_SHA1` tag) rather than reverting
   the PR — the PR revert is for the *module graph*, not for a single bad
   deploy.
5. **Expected duration:** almost certainly spans multiple sessions/days given
   33 services in batches with observation windows between them — this is
   not a same-day rollout, and the plan should not be read as if it were.

**`bin-trigger-sender` — RESOLVED (2026-09-02): zero production deployment,
excluded from this ticket's scope entirely.** The open question above was
closed by reading `docs/plans/2026-08-01-bin-schedule-manager-design.md`
(VOIP-1281, Done), which had already investigated this exact service in
detail while designing its replacement: the `number-renew` CronJob
(`bin-number-manager/k8s/cronjob.yml`) was **already commented out of
`kustomization.yml` before that design even started** — i.e. never actually
applied by `kubectl apply -k` — and that design's own risk assessment for
deleting the manifest was "risk ≈ 0" for exactly that reason. `bin-trigger-sender`'s
one function (`number-renew`) was absorbed into `bin-schedule-manager`'s
proper scheduling system as an internal library call; the CronJob manifest
was deleted; `bin-trigger-sender`'s binary and CI entries were deliberately
left in place at the time ("binary + CI remain") pending a "trivial follow-up"
retirement once install/sandbox confirmed nothing invokes it, tracked under
VOIP-1281's cutover checklist. VOIP-1281 itself is now **Done** (closed
2026-08-02) — its retirement checklist passed — but the actual follow-up
(delete `bin-trigger-sender`'s directory + CircleCI entries) was never
executed; the service still sits in the tree, unused, a month later.

**Decision**: exclude `bin-trigger-sender` from this ticket's `go.mod`/
Dockerfile bump. Upgrading a service confirmed dead and pending deletion is
wasted work, and it never appears in any of this document's deployment-risk
discussion since it has nothing to deploy. Its `-test` CI job continues to
work unmodified against the shared `go_image` anchor once that anchor moves
to `cimg/go:1.27.1` — a newer toolchain building an older-declared (`go
1.25.3`) module needs no `go.mod` change, and `bin-trigger-sender` has no
`replace`/`require` on `bin-common-handler` or `bin-sentinel-manager` (per
the Context section), so it was never part of the forcing cascade to begin
with. Its actual retirement (directory + CI entry deletion) is unrelated
dead-code cleanup, not a Go-toolchain concern — flagged as a small, ready,
independent follow-up rather than folded into this PR.

### Pre-merge verification: build every image locally first

Also per the distroless precedent (`전 서비스(34개) docker build를 머지 전
로컬 완주` — every service's docker build run to completion locally before
merge): run `docker build -f <service>/Dockerfile .` for all 36 in-scope
Dockerfile-bearing services locally (or in a scratch CI run) **before** opening the PR,
not just `go test`/`golangci-lint`. The 5-step verification workflow (below)
exercises the Go toolchain but not the Docker base-image change; a bad
`golang:1.27.1-*` tag or a base-image-specific build failure (e.g.
`bin-pipecat-manager`'s `libsoxr-dev`/`ffmpeg` apt packages against a newer
Debian base) would otherwise only surface after merge, during the staged
production rollout above.

### Verification order

Reordered to front-load both the module-graph risk (unchanged from the first
draft) and the build-environment outliers a purely graph-based order would
leave until "~32 services, any order":

1. `bin-sentinel-manager` and `bin-call-manager` (the two modules whose
   replace-chain position actually forced this document to exist) — if these
   two don't come out clean, nothing else will either.
2. `bin-common-handler` (36-module blast radius if this breaks).
3. The three build-environment outliers, run early rather than folded into
   "the rest": `bin-pipecat-manager` (`CGO_ENABLED=1`, `libsoxr-dev`,
   `-bookworm` base — the fleet's most toolchain-sensitive build by far),
   `bin-api-manager` (its own `go-test-api-manager` CircleCI command, not the
   shared `go-test`), `bin-openapi-manager` (the only `-validate` job,
   oapi-codegen-driven).
4. Remaining ~32 services, in any order, in parallel (38 in-scope − 6 named
   above; `bin-trigger-sender` is out of scope entirely, not merely deferred
   to this bucket).

## Risks and mitigations

- **`bin-pipecat-manager`'s `-bookworm` base tag** is the one Dockerfile that
  doesn't follow the fleet's `-alpine` pattern; a scripted fleet-wide
  find/replace on `1.25-alpine` alone would silently skip it. Handle
  explicitly.
- **`gotestsum` availability on `cimg/go:1.27.1`** (noted above) — cheap to
  find out empirically on the first `-test` job re-run after the anchor swap,
  not worth pre-verifying further.
- **GODEBUG default rollover — mitigated by pinning, not just noted.**
  Raising a module's own `go` directive from `go1.25` to `go1.27` also raises
  its `GODEBUG` defaults for that span (the same mechanism `k8s.io/client-go`
  uses via its own `godebug default=go1.26` line — a dependency's `godebug`
  line only affects that dependency's own package-level defaults, not the
  importing module's, so this doesn't leak into our services automatically,
  but our own `go` directive bump does). This is the one genuinely
  runtime-behavioral surface in an otherwise build-time-only change — TLS,
  `net/http`, and `crypto` default changes are the usual suspects across Go
  minor releases, and it's exactly what the canary/batched deployment staging
  above exists to catch if it fires.

  Go supports decoupling "raise the toolchain floor" from "adopt the new
  GODEBUG defaults that come with it": adding `godebug default=go1.25` to
  each service's own `go.mod` keeps GODEBUG behavior pinned to Go 1.25
  semantics even while the `go` directive (and therefore the compiler/stdlib
  API surface available to the code, including what `k8s.io/client-go`
  v0.37.0 needs) moves to 1.27. **Adopted as part of this design**: every
  service's `go.mod` gets both `go 1.27.1` and `godebug default=go1.25`. This
  directly narrows the deployment risk this document spends the most words
  on — with GODEBUG frozen, the remaining behavior delta between the old and
  new binaries is compiler/stdlib bug-fixes and performance changes, not
  intentional default-flipping, which is a meaningfully smaller thing to
  canary for. Skimming the Go 1.26/1.27 release notes' GODEBUG section once,
  fleet-wide, to confirm this covers what's actually relevant (RabbitMQ/AMQP,
  Redis, HTTP servers/clients, gRPC, WebSocket, GCS are this fleet's
  network-facing surface) is still worth doing before implementation, to
  make sure `godebug default=go1.25` isn't itself masking a fix we actually
  want.
- **`golangci-lint` version.** No `.golangci.yml` exists anywhere in the repo
  (so no version-pinned config to conflict), but whatever `golangci-lint`
  binary a developer or this session runs locally must be new enough to
  typecheck a `go 1.27.1` module. CI's own lint step is already
  commented-out/disabled repo-wide (pre-existing OOM issue on `small`
  resource class, unrelated to this ticket) — root CLAUDE.md's mandatory
  local lint step is the only place this can actually surface.
- **Fleet Go-version consistency, with one accepted exception.** Unlike the
  originally-considered 2-or-3-service partial bump (which would have left
  the fleet permanently split across two Go versions with one CircleCI
  anchor awkwardly forked), this design keeps 38 of 39 modules on the same
  version and the CircleCI config on a single shared anchor. The one
  exception, `bin-trigger-sender`, is confirmed dead code pending deletion
  (see Scope/Sequencing), not a live fragmentation — no follow-up "fleet
  alignment" ticket is needed as a consequence of this change.
- **Codegen tooling compatibility.** `go generate ./...` (step 3 of the
  mandatory workflow) drives `go.uber.org/mock` (mockgen) fleet-wide, plus
  `oapi-codegen` (`bin-openapi-manager`/`bin-api-manager`) and
  `protoc-gen-go` where used. A Go 1.25→1.27 jump can change a generator's
  output or break it outright; this is exactly the kind of thing step 3
  exists to catch, but expect at least one generator-output diff somewhere in
  the 38 in-scope services and don't treat a non-empty diff there as a bug in
  this change — verify it's cosmetic (formatting/comment changes) before
  committing.
- **Go 1.27 recency vs. Go 1.26 maturity — recorded as an informed choice,
  not an oversight.** Go 1.27 was roughly a month old at the time this
  document was written; `go1.26.8` is a mature patch release and satisfies
  `k8s.io/client-go`'s actual floor (`go 1.26.0`) exactly, with no version to
  spare. 대표님 explicitly chose "latest stable" over "the dependency's bare
  minimum" when asked. Noting the tradeoff here so the choice is on the
  record as deliberate for a change touching 38 production services, not
  because either version is expected to cause problems.
- **Rollback.** Module-graph level: revert the PR (all 38 in-scope modules
  move back together, same atomicity argument as the forward move). Deployment level:
  per-service image-tag rollback to the pre-upgrade `$CIRCLE_SHA1`, exactly
  as the distroless precedent used — this is why deployment is staged
  per-service rather than bulk-approved, so a bad rollout on one service
  doesn't force rolling back services that already deployed cleanly.
- **Sandbox digest-lock consequence.** The distroless design flagged that
  `sandbox/` pins image digests, and a fleet-wide image rebuild changes every
  digest, requiring a coordinated separate PR in that repo once the new
  images are live. A Go-version bump rebuilds 36 of the 37 images the same
  way (`bin-trigger-sender`'s image is unchanged, per Scope) —
  this needs the same follow-up. Not solved here; flagged so it isn't
  rediscovered mid-rollout the way the annotation-testability and
  bin-call-manager cascade findings were rediscovered mid-analysis in this
  document's own history.
- **CI capacity.** 38 services' worth of `go mod vendor` + `go test` +
  (locally) `golangci-lint`, on a raised `go` floor that generally means
  larger vendor trees and longer builds, run on CircleCI's `small`
  `resource_class` — the same class that already OOMs on `golangci-lint`
  today (hence CI lint being commented out repo-wide). Not expected to be
  blocking since CI lint is already off, but worth watching `-test` job
  duration/memory during the staged rollout in case the raised floor pushes
  `go test`/`go mod vendor` itself over `small`'s ceiling on the heavier
  services (`bin-api-manager`, `bin-flow-manager`).
- **`oapi-codegen` is installed unpinned in CI** (`go install
  github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest` in
  `bin-openapi-manager-validate`), separately from whatever version a
  developer has installed locally. This is a pre-existing gap, not introduced
  by this change, but it means a CI-only `go generate`/validate diff after
  this bump could come from an independently-drifted `oapi-codegen@latest`
  rather than from the Go version bump itself — worth ruling out explicitly
  if that job fails, rather than assuming this design caused it.
- **`GOTOOLCHAIN=auto` works fine for the common case — the earlier warning
  in this document is narrower than it might read.** The Context section's
  finding that `GOTOOLCHAIN=auto` does NOT rescue a `replace`d dependency's
  stricter `go` requirement is specifically about that transitive case. For
  the ordinary case — a developer on an older local Go building a module
  whose own `go.mod` now says `1.27.1` — `GOTOOLCHAIN=auto` DOES work
  (empirically confirmed during this analysis: `bin-call-manager` built
  successfully once *its own* `go.mod` declared `1.27.1`, auto-downloading
  the toolchain). So this change does not strand developers on an older
  local Go; they get a transparent toolchain download the first time they
  build a bumped service. No `toolchain` directive is being added to any
  `go.mod` — none exists today, and `GOTOOLCHAIN=auto`'s default behavior is
  sufficient without one.

## Non-goals

- Does not touch `k8s.io/*` dependency versions themselves (VOIP-1446, after
  this lands).
- Does not touch `bin-dbscheme-manager` (Python, no `go.mod`).
- Does not introduce a `go.work` file or otherwise change the monorepo's
  independent-module structure — each service keeps its own `go.mod`, this
  just moves the shared floor all of them declare.
- Does not address the `models/container`-via-`replace` coupling between
  `bin-sentinel-manager` and `bin-call-manager` architecturally (an
  alternative considered and set aside in favor of the fleet-wide bump per
  대표님's explicit decision) — if a future dependency bump on
  `bin-sentinel-manager` forces this same situation again, extracting
  `models/container` into its own minimal module becomes worth reconsidering
  at that time, not now.

## References

- [VOIP-1446](https://voipbin.atlassian.net/browse/VOIP-1446) — the triggering
  ticket, resumes after this lands
- [VOIP-1447](https://voipbin.atlassian.net/browse/VOIP-1447) — this ticket
- [docs/plans/2026-07-31-fleet-static-build-distroless-design.md](2026-07-31-fleet-static-build-distroless-design.md)
  (VOIP-1277) — the actual nearest precedent: a fleet-wide, build-time-only
  change across nearly the same 37 services, one PR + staged per-service
  deployment via `build-approval` gates + image-tag rollback. This design's
  "Sequencing" section directly adopts that doc's deployment discipline.
- [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md](2026-08-18-bin-manager-komodo-rollout-tier1-design.md),
  [-tier2-](2026-08-18-bin-manager-komodo-rollout-tier2-design.md),
  [2026-08-28-bin-manager-two-replica-rollout-design.md](2026-08-28-bin-manager-two-replica-rollout-design.md)
  — earlier fleet-wide rollout docs with *runtime*-behavior-change tiers,
  cited for contrast: this change's risk is build-time (module graph), so
  its atomicity boundary (one PR) differs from theirs, even though its
  deployment-staging discipline matches the distroless precedent above
