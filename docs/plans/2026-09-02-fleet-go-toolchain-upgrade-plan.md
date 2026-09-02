# Fleet-wide Go Toolchain Upgrade — Implementation Plan

Execution plan for [2026-09-02-fleet-go-toolchain-upgrade-design.md](2026-09-02-fleet-go-toolchain-upgrade-design.md)
(VOIP-1447), approved after 12 design-review rounds. This document is the
concrete, step-by-step translation of that design into an executable task
list — it does not re-derive or re-justify anything the design already
settled; see that document for the "why" behind every step here.

## Step 0 — Prerequisites (once, before touching any service)

1. **Re-verify the target Go version.** `curl -s https://go.dev/dl/?mode=json`
   → confirm the latest `stable: true` release (design assumed `1.27.1`).
   Then confirm all THREE images this plan depends on exist for that exact
   version, not just the Go release itself:
   - `cimg/go:<version>` (Docker Hub — CircleCI executor)
   - `public.ecr.aws/docker/library/golang:<version>-alpine` (ECR Public — 35 Dockerfiles)
   - `public.ecr.aws/docker/library/golang:<version>-bookworm` (ECR Public — `bin-pipecat-manager`)

   **If the newest stable Go patch doesn't have a matching `cimg/go` tag yet**
   (this happens — CircleCI's convenience images lag upstream releases): step
   down to the newest version for which all three exist. `go.mod`, the
   Dockerfiles, and the CircleCI executor must all agree on one version; do
   not let them drift.
2. **GODEBUG sanity check.** Skim the Go 1.26 and 1.27 release notes'
   GODEBUG/compatibility sections once, fleet-wide (per design's Risks
   section) against this fleet's actual network-facing surface
   (RabbitMQ/AMQP, Redis, HTTP servers/clients, gRPC, WebSocket, GCS).
   Confirm `godebug default=go1.25` (Step 2 below) isn't itself suppressing a
   bug fix this fleet actually wants, not just an intentional default flip.
   If it is, note the exception explicitly before proceeding — do not
   silently apply `go1.25` defaults everywhere without this check.
3. **Local tooling check**, once, not per-service:
   - `go version` / `go env GOTOOLCHAIN` — confirm not pinned to
     `GOTOOLCHAIN=local` on an old Go (would block every `go mod tidy` below;
     `auto`, the default, transparently downloads what's needed).
   - `golangci-lint --version` — confirm new enough to typecheck a module
     declaring the target Go version. This blocks step 2's verification for
     ALL 38 services if stale; find out once here, not on service 1 of 38.
   - `docker --version` and enough free disk for 36 sequential/parallel image
     builds (Step 4).

## Step 1 — The 38-service list (enumerated, not indirected)

Every module below gets the go.mod + (where noted) Dockerfile change in Step
2. `bin-trigger-sender` is the only `bin-*`/`voip-*` Go module NOT on this
list (confirmed dead code, no production deployment — see design's
Sequencing section; its `-test` job needs no change of its own once the
shared CircleCI anchor moves in Step 3).

**Tier 1 — module-graph risk, sequential, must be clean before anything else proceeds:**
- [ ] `bin-sentinel-manager` (alpine)
- [ ] `bin-call-manager` (alpine)

**Tier 2 — blast-radius gate, sequential, must be clean before Tier 4 starts:**
- [ ] `bin-common-handler` (no Dockerfile)

**Tier 3 — build-environment outliers, can run parallel to each other once Tier 2 is clean:**
- [ ] `bin-pipecat-manager` (**bookworm** — the fleet's only non-alpine Go build; `CGO_ENABLED=1` + `libsoxr-dev`/`ffmpeg` apt packages, the design's top build-risk citation)
- [ ] `bin-api-manager` (alpine — own `go-test-api-manager` CircleCI command, not the shared `go-test`)
- [ ] `bin-openapi-manager` (no Dockerfile — the only `-validate` CI job, oapi-codegen-driven; note CI's `oapi-codegen@latest` is unpinned, a pre-existing drift source unrelated to this change)

**Tier 4 — remaining 32, parallel, any order, only after Tiers 1-3 are all clean:**
- [ ] `bin-agent-manager` (alpine)
- [ ] `bin-ai-manager` (alpine)
- [ ] `bin-billing-manager` (alpine)
- [ ] `bin-campaign-manager` (alpine)
- [ ] `bin-conference-manager` (alpine)
- [ ] `bin-contact-manager` (alpine)
- [ ] `bin-conversation-manager` (alpine)
- [ ] `bin-customer-manager` (alpine)
- [ ] `bin-direct-manager` (alpine)
- [ ] `bin-email-manager` (alpine)
- [ ] `bin-flow-manager` (alpine)
- [ ] `bin-hook-manager` (alpine)
- [ ] `bin-message-manager` (alpine)
- [ ] `bin-number-manager` (alpine)
- [ ] `bin-outdial-manager` (alpine)
- [ ] `bin-queue-manager` (alpine)
- [ ] `bin-rag-manager` (alpine)
- [ ] `bin-registrar-manager` (alpine)
- [ ] `bin-route-manager` (alpine)
- [ ] `bin-schedule-manager` (alpine — runtime stage is `debian:bookworm-slim` for `mariadb-client`, per its own CLAUDE.md, but the Go **build** stage is alpine like the rest; do not "fix" the runtime stage, it's intentional and out of scope)
- [ ] `bin-storage-manager` (alpine)
- [ ] `bin-tag-manager` (alpine)
- [ ] `bin-talk-manager` (alpine)
- [ ] `bin-timeline-manager` (alpine)
- [ ] `bin-transcribe-manager` (alpine)
- [ ] `bin-transfer-manager` (alpine)
- [ ] `bin-tts-manager` (alpine)
- [ ] `bin-webchat-manager` (alpine)
- [ ] `bin-webhook-manager` (alpine)
- [ ] `voip-asterisk-proxy` (alpine — **no `-deploy` job in this repo's CI**, see Step 5)
- [ ] `voip-kamailio-proxy` (alpine — **no `-deploy` job in this repo's CI**, see Step 5)
- [ ] `voip-rtpengine-proxy` (alpine — **no `-deploy` job in this repo's CI**, see Step 5)

(2 + 1 + 3 + 32 = 38, matching the design's scope count.)

## Step 2 — Per-service mechanical change

For each service above, in tier order (Tier 1 and 2 sequential and gating;
Tier 3 and 4 parallelizable within their tier once their gate is clean):

1. **`<service>/go.mod`**: change `go 1.25.3` → `go <version>` (the version
   confirmed in Step 0 — this is the ONLY line that uses the target
   version). On the line immediately after, add:
   ```
   godebug default=go1.25
   ```
   **This second line is always literally `go1.25`, regardless of what
   `<version>` is** — it is not a placeholder to substitute, it's the
   deliberate GODEBUG freeze from the design's Risks section. Do not
   "consistently" change it to match the `go` directive.
2. **`<service>/Dockerfile`** (skip for `bin-common-handler` and
   `bin-openapi-manager` — no Dockerfile): change
   ```
   FROM public.ecr.aws/docker/library/golang:1.25-alpine AS build
   ```
   to
   ```
   FROM public.ecr.aws/docker/library/golang:<version>-alpine AS build
   ```
   (substitute `-bookworm` for `bin-pipecat-manager`, the sole exception).
   Full registry path included deliberately — do not drop the
   `public.ecr.aws/docker/library/` prefix.

   **`<version>` here means the exact patch (`1.27.1`), never the floating
   minor tag (`1.27`).** The line being replaced today uses a floating minor
   tag (`1.25-alpine`), so writing `golang:1.27-alpine` by pattern-analogy is
   the natural-looking mistake — and it defeats the one deliberate
   convention change this plan makes (design's per-service Dockerfile
   section: exact-patch pinning replaces floating tags fleet-wide,
   specifically because the official `golang` image sets `GOTOOLCHAIN=local`,
   so a floating tag resolving below `go.mod`'s floor fails the build with
   no auto-download rescue).
3. Run the full 5-step verification workflow from `<service>/`:
   ```
   go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
   ```
   Commit the resulting `go.mod`/`go.sum` (not `vendor/` — gitignored).
   Watch `-test` job duration/memory for `bin-api-manager` and
   `bin-flow-manager` in particular (design's CI-capacity risk item — larger
   vendor trees on a raised `go` floor, on CircleCI's already-tight `small`
   `resource_class`).
   **Commit granularity: one commit per service** (the design's "only the
   final commit/PR is single and atomic" means atomic-on-`main`, which this
   repo's squash-merge convention delivers regardless of intermediate commit
   count — not literally one commit during execution), title
   `VOIP-1447-Fleet-go-toolchain-upgrade` matching the branch name (root
   CLAUDE.md's commit-title rule applies per-commit, not just to the PR)
   — this repo squash-merges, so 38 intermediate commits collapse into one
   on `main` and cost nothing; per-service commits also mean a
   `git bisect`/revert during execution can isolate one bad service without
   touching the other 37.
4. **A non-empty `go generate` diff is expected in some services** (mockgen/
   oapi-codegen/protoc-gen-go output can shift on a toolchain bump, per the
   design's Risks section) — inspect it, confirm it's cosmetic
   (formatting/comment-only), and commit it alongside. If a diff looks
   behavioral rather than cosmetic, stop and escalate (see Step 2a).

### Step 2a — Failure handling (read before starting, not after hitting one)

`go.mod`/Dockerfile bumps are expected to be 100% mechanical. If verification
fails on any service, the failure mode determines the response — **do not
improvise a fix mid-run**:

- **`go test` fails** on a genuinely new-in-target-version stdlib/runtime
  behavior (not a flaky/pre-existing failure — re-run once to rule that out
  first): stop, do not patch the test to paper over it, escalate for a
  decision on whether this is a real regression to fix in application code
  (out of scope for a "mechanical" PR — see below) or a test assumption that
  needs updating.
- **`golangci-lint` reports NEW findings** that only exist for the raised Go
  version (plausible — no `.golangci.yml` exists anywhere in this repo
  today, so there's no version-pinned config to conflict, but a newer
  linter/compiler surface can still find new things): fix trivially-safe
  findings (e.g. a newly-flagged but genuinely dead branch) inline; for
  anything requiring non-trivial logic change, escalate rather than expand
  scope.
- **`docker build` fails** (Step 4): almost certainly a base-image
  compatibility issue (design flags `bin-pipecat-manager`'s
  `libsoxr-dev`/`ffmpeg` apt packages against a newer Debian base as the
  likeliest case). Fix the Dockerfile's package/step compatibility, re-test
  that one service's build in isolation before re-running the full fleet.
- **A CircleCI `-test` or `-validate` job fails after Step 3's anchor swap, but the same
  service passed locally in Step 2**: check `gotestsum: command not found`
  first (design's Risks section flags this as expected-but-unverified —
  `gotestsum` is invoked with no explicit install step, expected to ship
  pre-bundled in `cimg/go` convenience images per CircleCI's own convention,
  but this was never confirmed against the actual image). For
  `bin-openapi-manager-validate` specifically, also rule out the pre-existing
  unpinned `oapi-codegen@latest` install (`config_work.yml`) drifting
  independently of this change before assuming the Go bump caused it.
- **General rule**: this PR's blast radius is `go.mod` + `godebug` line +
  Dockerfile base image + generated-code churn. Anything requiring a
  non-mechanical application-code change to pass is a signal to STOP, not to
  push through — because of the design's atomicity constraint, you cannot
  simply drop a failing service from the PR — for the 36 modules in the
  `bin-common-handler`/`bin-call-manager` replace graph, a service left on
  `go 1.25.3` while others move to `<version>` reintroduces the exact
  `replace`-chain-mismatch failure this whole document exists to prevent;
  `bin-openapi-manager` has no local-module replace chain at all (design's
  Scope section), so dropping it specifically would only break fleet Go-
  version consistency, not the module graph — still not a decision to make
  unilaterally, just a different (smaller) kind of inconsistency if it ever
  came up. If a service turns out to need a real code fix, that's a scope
  decision for 대표님, not something to resolve unilaterally mid-execution.
  One option to raise at that point, not to apply automatically: stepping
  the whole target down to `go1.26.8` (mature, still satisfies
  `k8s.io/client-go`'s actual `go 1.26.0` floor per the design) rather than
  `1.27.1`, if the blocking issue turns out to be specific to 1.27.
- **Mid-flight transient errors are expected and are not evidence of a
  design flaw**: between the first service's bump and the last, the repo is
  briefly in a mixed state. Any module still at `go 1.25.3` that `replace`s
  an already-bumped module will fail immediately with `module ../X requires
  go >= <version> (running go 1.25.3)` — this is the exact mechanism the
  design's Context section empirically reproduced, now happening on purpose,
  tier by tier. It means: **finish editing every module's own `go.mod` in a
  tier before running verification on any module in that tier**, and never
  run a repo-wide build/test sweep until all 38 are edited. It does NOT mean
  something is broken.

### Step 2b — Final cross-module re-verification sweep (once, after all 38 are edited)

The tier gating in Step 2a prevents failures WHILE editing is in progress; it
does not guarantee the final state is still clean once every module has
moved. Concretely: 34 of the other 38 modules `replace`+`require`
`monorepo/bin-pipecat-manager` (a genuinely widely-depended-on module, not
just the 6 named in Tiers 1-3 — a 35th, `voip-kamailio-proxy`, carries only
an orphaned `replace` with no matching `require`, the same shape the design's
Context section already documents for its `bin-call-manager` edge, so it
propagates nothing and doesn't count here), so Tier 1/2's `go mod tidy`
output — committed early, before Tier 3 touches `bin-pipecat-manager` —
depends on local replace-targets whose own `go.mod` can still change later
in the run. Directionally this can't cause a hard failure (a consumer ahead
of its replace-targets is always safe), but a Tier 1/2 module's committed
`go.mod`/`go.sum` can go stale relative to what a fresh `tidy` would produce
once every replace-target has also moved.

After all 38 modules in Step 1's list are edited and individually verified,
re-run the **full 5-step workflow** (`go mod tidy && go mod vendor && go
generate ./... && go test ./... && golangci-lint run -v --timeout 5m` — the
same command as Step 2.3, not a reduced set: `go mod vendor` is what Step 4's
`docker build`s below actually consume, and a `tidy`-driven dependency-version
change here can surface new `golangci-lint` findings that CI cannot catch on
its own, since CI lint is disabled repo-wide) once more across all 38,
committing any residual churn as one additional commit. No tier ordering is
needed for this pass — everything is already at the target version, so the
`requires go >= X` failure mode is structurally impossible regardless of
order — but **the pass must converge, not just run once**: if it produces
any diff, re-run it again and keep repeating until a pass produces none (or
equivalently, until `git status --porcelain` is clean after a pass), since a
module tidied earlier in the same sweep may have read a target's pre-sweep
`go.mod` and gone stale again one step further out. In practice this is
expected to converge immediately or be a no-op (both `1.25.3` and the target
version are ≥1.17, so module-graph pruning behavior doesn't change) — but the
plan can't assert zero churn in advance, so verify convergence rather than
assuming a single pass was enough. This is the Go equivalent of Step 4's
Docker sweep below (and must complete before Step 4, since Step 4 consumes
the `go.sum` this step finalizes): a closing gate that catches staleness the
per-module, tier-ordered pass structurally cannot.

## Step 3 — Shared CircleCI change (once, not per-service)

- `.circleci/config_work.yml`: change the `go_image` anchor definition
  (`&go_image - image: cimg/go:1.25.3`) to `cimg/go:<version>`. Single edit;
  all 38 alias usages (37 `-test` jobs + `bin-openapi-manager-validate`)
  inherit it automatically, including `bin-trigger-sender-test` — which
  needs no `go.mod` change of its own, since a newer toolchain building an
  older-declared (`go 1.25.3`) module works without modification and
  `bin-trigger-sender` was never part of the replace-chain forcing this
  upgrade to begin with.

## Step 4 — Pre-merge Docker build verification (once, before opening the PR)

Per design's "Pre-merge verification" section: `docker build -f
<service>/Dockerfile .` (repo-root context) for all 36 in-scope
Dockerfile-bearing services (38 in Step 1's list minus `bin-common-handler`
and `bin-openapi-manager`), run to completion locally or in a scratch CI run.
This is what Step 2's 5-step workflow does NOT exercise (it never invokes
Docker) and is where a bad base-image tag or a base-image-specific build
failure would otherwise only surface post-merge, during staged production
rollout. See Step 2a for failure handling.

## Step 5 — Commit and PR

- Branch/title: `VOIP-1447-Fleet-go-toolchain-upgrade`, matching Step 2's
  per-commit titles.
- PR body lists every touched service with `bin-<service>:`/
  `voip-<service>:` prefixes, plus `.circleci:` for the shared anchor
  change. **Explicitly call out in the PR body** that `voip-asterisk-proxy`,
  `voip-kamailio-proxy`, and `voip-rtpengine-proxy` have no `-deploy` job in
  this repo's CI — their rebuilt images reach production through whatever
  mechanism already manages that fleet out-of-band, and that owner needs to
  know new images exist once this merges (per design's Sequencing section;
  this plan does not control or execute that handoff, only surfaces it).
- Standard pre-PR checks: `git fetch origin main`, conflict check via
  `git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main`,
  review `git log --oneline HEAD..origin/main`.
- Open PR. Do NOT merge without explicit authorization (standing repo rule)
  — and per design, merging is not deploying: production rollout (canary →
  batches, per design's Sequencing) is a separate, subsequent, manual/
  observational action after merge, not something this PR's CI triggers
  automatically for the 33 services with a `-deploy` job.
- **File two follow-up Jira tickets at PR time** (not GitHub issues — this
  repo's issue tracker is Jira), each standalone with enough context to act
  without this PR's discussion thread:
  1. `bin-trigger-sender` retirement (delete directory + CircleCI entries) —
     confirmed safe (VOIP-1281, Done 2026-08-02) but never executed a month
     later; this is the second time this exact "confirmed-safe, never
     executed" gap has been documented, worth a real ticket this time
     instead of a comment.
  2. Sandbox `sandbox/` image-digest-lock refresh, once these images are
     live (per design, mirrors the same follow-up the distroless precedent
     required).

## Explicitly not in this plan

- `bin-trigger-sender`'s dead-code retirement itself (ticketed per Step 5,
  not executed here).
- The k8s.io/* dependency bump, or `voip-asterisk-proxy`'s
  `patchPodAnnotation` widening + new test (VOIP-1446, resumes after this
  merges).
- Actual staged production deployment execution (canary/batch approval,
  Komodo health confirmation) — happens after merge, manual/observational
  per the design, not a code change this plan produces.
- Sandbox digest-lock refresh itself (ticketed per Step 5, separate repo).

## References

- [2026-09-02-fleet-go-toolchain-upgrade-design.md](2026-09-02-fleet-go-toolchain-upgrade-design.md) — full rationale, empirical evidence, and risk analysis this plan executes
- [VOIP-1447](https://voipbin.atlassian.net/browse/VOIP-1447)
