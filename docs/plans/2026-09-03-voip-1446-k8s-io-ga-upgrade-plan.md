# k8s.io/{api,apimachinery,client-go} alpha → GA Upgrade — Implementation Plan

VOIP-1446. Execution plan for
`docs/plans/2026-09-03-voip-1446-k8s-io-ga-upgrade-design.md` (5 review rounds, 2
consecutive approvals — read that document first; this plan does not repeat its
evidence or reasoning, only the sequence of actions and verification).

## Step 0: Prerequisites — re-verify before touching code

The design doc's "v0.37.0 is latest GA" claim is time-sensitive (written
2026-09-03). Before starting:

1. Re-run the `proxy.golang.org` check the design doc did:
   ```bash
   curl -s "https://proxy.golang.org/k8s.io/client-go/@v/list" | grep -E "^v0\.37\.[0-9]+$"
   curl -s "https://proxy.golang.org/k8s.io/api/@v/list" | grep -E "^v0\.37\.[0-9]+$"
   curl -s "https://proxy.golang.org/k8s.io/apimachinery/@v/list" | grep -E "^v0\.37\.[0-9]+$"
   ```
   If a `v0.37.x` patch or a `v0.38.0` GA (not `-alpha`/`-beta`/`-rc`) now exists,
   target that instead of `v0.37.0` throughout this plan — re-verify the three
   packages' go.mod files stay version-consistent with each other at whatever
   target is chosen, and re-check `client-go`'s own `go` directive requirement is
   still satisfied by `go 1.27.1`.
2. `golangci-lint run` in the verification steps below assumes a binary new
   enough to type-check a `go 1.27.1` module (any binary built with an older Go
   than the module it's checking will refuse to run, as first hit during
   VOIP-1447). Confirm with `golangci-lint --version` — if the `PATH` binary is
   too old, either fix `PATH` to point at a working one for this session, or use
   its full path explicitly in the commands below. This is a session-local
   environment fact each implementer needs to check for themselves, not a
   repo-documented convention — the repo's own commands (root `CLAUDE.md`, and
   VOIP-1447's own plan/design docs) all invoke bare `golangci-lint` on `PATH`.
3. Confirm the worktree is on current `main` (post-VOIP-1447) — `go 1.27.1` and no
   `godebug default=` line should already be present in `bin-sentinel-manager` and
   `voip-asterisk-proxy` (the two services this plan actually changes; also check
   `bin-call-manager`, which Step 3 verifies stays untouched, since it's on the
   same branch base). If any of the three still show `go 1.25.3` or an alpha
   k8s.io pin unexpectedly reverted, STOP — something is wrong with the branch
   base, do not proceed until resolved (this would mean VOIP-1447 isn't actually
   in this branch's history).

## Step 1: bin-sentinel-manager

Work in `bin-sentinel-manager/`.

### 1a. Bump k8s.io/* dependencies

```bash
cd bin-sentinel-manager
go get k8s.io/api@v0.37.0 k8s.io/apimachinery@v0.37.0 k8s.io/client-go@v0.37.0
```

(Use `go get`, not a manual go.mod edit, so the indirect `k8s.io/klog/v2`,
`k8s.io/kube-openapi`, `k8s.io/utils`, `sigs.k8s.io/*` versions get resolved
correctly alongside the direct bump — `go mod tidy` in the verification step below
will clean up formatting either way, but `go get` is the more standard tool for
"bump these specific deps and let the resolver do its job.")

### 1b. Migrate `pkg/k8swatchhandler/run.go`

Apply exactly the before/after shown in the design doc's "What actually needs to
change, per service → bin-sentinel-manager" section:

1. In `runInformer`, change the `&cache.ListWatch{ListFunc: ..., WatchFunc: ...}`
   struct literal to `&cache.ListWatch{ListWithContextFunc: ..., WatchFuncWithContext: ...}`,
   with parameters named `listCtx`/`watchCtx` (not `ctx`, to avoid shadowing the
   function's own `ctx`). Remove both `//nolint:staticcheck` comments.
2. Remove or rewrite the explanatory comment block above that struct literal (it
   currently narrates a deferral to this ticket — that deferral is what this
   change resolves).
3. Replace the `stopCh` + bridge-goroutine block with a direct
   `informer.RunWithContext(ctx)` call, per the design doc's exact before/after.
   This is not optional — see the design doc's justification (the field migration
   alone would make `listCtx`/`watchCtx` receive a *different* context object than
   `runInformer`'s own `ctx`, not "the same value" as an earlier draft of the
   design incorrectly assumed; `RunWithContext` makes it genuinely the same
   object).
4. Double check nothing else in the file references `stopCh` after removal (design
   doc already confirmed `waitForCacheSync` doesn't; re-confirm with a grep as a
   final sanity check, not a trust exercise).

### 1c. Migrate `pkg/k8swatchhandler/run_test.go`

In `newTestInformer()`, apply the same `ListWatch` field migration (no `Run`
call exists in this helper to touch — it's deliberately never run, see design
doc): rename the `ListFunc`/`WatchFunc` fields to `ListWithContextFunc`/
`WatchFuncWithContext`, and change each closure's signature to accept
`listCtx`/`watchCtx context.Context` as its first parameter (matching `run.go`'s
naming), passing that parameter through in place of the `context.Background()`
call the closure currently hardcodes internally. **Also update this file's copy
of the deferral comment** — `run_test.go:534-535` currently reads:

```go
// See the matching comment in run.go -- deprecated ListFunc/WatchFunc, migration deferred to
// VOIP-1446 alongside the k8s.io/* GA bump.
```

Replace with the design doc's supplied text (its `run_test.go` "after" snippet):

```go
// See the matching comment in run.go -- WithContext ListWatch fields, matching what
// client-go's Reflector actually calls.
```

Do not skip this: leaving the old comment would point at a `run.go` comment that
no longer exists (Step 1b.2 rewrites or removes it) and claim a deferral this very
commit resolves.

### 1d. Verify

```bash
cd bin-sentinel-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

(See Step 0 item 2 on which `golangci-lint` binary this needs to resolve to.)

Expect: `go.sum` picks up the new k8s.io/* dependency tree (this is a real,
expected diff, unlike VOIP-1447's godebug-only changes — inspect it for anything
surprising per the design doc's "go mod tidy pulling in an indirect version with
its own issues" risk, but a normal k8s.io GA dependency tree is expected here).
Zero SA1019 findings related to `ListFunc`/`WatchFunc` (they're gone). Zero new
test failures — the existing `run_test.go` suite (including
`Test_waitForCacheSync_*`) should pass unchanged in behavior, just against the
migrated helper.

If `go test` reveals anything unexpected: STOP, do not push through. This is
where the design doc's "API surface break" risk would actually manifest, low
probability as assessed, but this is the check that would catch it.

## Step 2: voip-asterisk-proxy

Work in `voip-asterisk-proxy/`.

### 2a. Bump k8s.io/* dependencies, then tidy immediately

```bash
cd voip-asterisk-proxy
go get k8s.io/apimachinery@v0.37.0 k8s.io/client-go@v0.37.0
go mod tidy
```

Run `go mod tidy` right away, before writing any test — the new test file (Step
2c) imports `k8s.io/client-go/kubernetes/fake` (never previously imported
anywhere in this module) and `k8s.io/api/core/v1` (currently indirect). Without
tidying first, the first `go test` invocation could fail with `missing go.sum
entry` rather than the actual assertion failure Step 2c is looking for — tidying
now separates "dependency resolution problem" from "the bug we're testing for."
Confirm `k8s.io/api` is promoted from `// indirect` to a direct require in
`go.mod` as part of this tidy (expected here, not deferred to later — the design
doc's earlier draft wrongly predicted it would stay indirect until the test file
existed; tidying now, even before the test file is written, makes no difference
to that promotion since `go mod tidy` only reacts to imports that already exist in
the module's `.go` files — so if it does NOT promote yet, that's expected too,
and the promotion will instead happen after Step 2c's test file is added and
tidied again as part of Step 2f's verification. Either way is fine; don't treat a
Step-2a no-op on `k8s.io/api` as an error).

### 2b. Widen `patchPodAnnotation`'s signature (behavior-neutral, do this BEFORE writing any test)

In `cmd/asterisk-proxy/annotation.go`, change:

```go
func patchPodAnnotation(clientset *kubernetes.Clientset, namespace, podName, annotationKey, annotationValue string) error {
```

to:

```go
func patchPodAnnotation(clientset kubernetes.Interface, namespace, podName, annotationKey, annotationValue string) error {
```

Do this step **before** Step 2c, not after. `k8s.io/client-go/kubernetes/fake.NewClientset()`
returns `*fake.Clientset`, which satisfies `kubernetes.Interface` but does **not**
satisfy `*kubernetes.Clientset` — a test file written against the pre-widening
signature would fail to compile, not fail with the assertion this TDD loop is
trying to observe, and a compile error is easy to misread as "the guard rail says
stop, so I'm done" when actually nothing was verified. Widening the signature
first (with no other change) is itself behavior-neutral and safe on its own —
confirm with a quick build (`go build ./...`) that nothing else broke before
moving on, since `patchPodAnnotation`'s only caller (`setProxyInfoAnnotation`)
already holds a `*kubernetes.Clientset` value, which satisfies the wider interface
automatically.

### 2c. TDD the annotation bug fix — write the failing test SECOND (now that it can compile)

Create `cmd/asterisk-proxy/annotation_test.go` (table-driven, matching this repo's
convention) with at minimum the "pod with no pre-existing annotations" case from
the design doc: seed `fake.NewClientset()` with a `*corev1.Pod` whose
`ObjectMeta.Annotations` is nil, call `patchPodAnnotation`, and assert an error is
returned. Name the test function `Test_patchPodAnnotation_*` per this repo's
existing convention (`Test_waitForCacheSync_*`, `Test_Run_gracefulShutdownReturnsNil`
in `bin-sentinel-manager/pkg/k8swatchhandler/run_test.go`) — this matters for the
next step, not just style.

Run it:

```bash
go test ./cmd/asterisk-proxy/... -run Test_patchPodAnnotation -v
```

**Do not trust a bare exit code here.** `go test -run <pattern>` with a pattern
that matches zero tests still exits 0 and prints "ok ... [no tests to run]" —
that would look identical to a pass. Read the `-v` output and confirm the test you
just wrote actually ran and actually failed, with the specific error you expect
("doc is missing path" or equivalent from the fake clientset's JSON-patch
library, confirming the RFC 6902 `add` operation rejected an absent parent path)
— not a compile error (would mean Step 2b was skipped or done wrong), not a
"no tests to run" message (would mean the `-run` pattern doesn't match your test's
actual name), and not an unrelated failure (would mean something else is broken
and needs its own diagnosis before proceeding). If the test unexpectedly passes
with the *current* `JSONPatchType` implementation still in place: STOP — the
design doc's premise about this bug would be wrong and needs re-investigation
before proceeding, not silent acceptance. Also expect this run to take a few
seconds of real wall-clock time before it fails — `patchPodAnnotation` retries
`defaultMaxRetries` (10) times at ~500ms apart before giving up, so this failing
test drives the full retry loop just like the "exhausted retries" case in Step
2e/2f does. Don't mistake that delay for a hang.

### 2d. Fix the patch type

Now that the failing test above confirms the bug, apply the fix in
`cmd/asterisk-proxy/annotation.go`: replace the `types.JSONPatchType` +
`[]map[string]string` payload + `escapedAnnotationKey`/
`strings.ReplaceAll(annotationKey, "/", "~1")` block with the
`types.MergePatchType` + nested-map payload from the design doc's exact
before/after code. Remove the now-unused escaping line.

Re-run the same test:

```bash
go test ./cmd/asterisk-proxy/... -run Test_patchPodAnnotation -v
```

Confirm it now PASSES (again, read the `-v` output, not just the exit code) —
this closes the TDD loop for the bug fix specifically.

### 2e. Fill out the rest of `annotation_test.go`

Add the remaining table-driven coverage from the design doc's test plan:

- Pod with pre-existing (non-empty) annotations — first-attempt success, assert
  the merge-patch payload shape.
- Retry-then-succeed (`PrependReactor` erroring for the first few calls).
- Exhausted retries (reactor always errors) — assert the error is returned after
  exactly `defaultMaxRetries` attempts, and accept the ~5s wall-clock cost as the
  design doc decided (do not add clock injection to make this faster — that was
  explicitly rejected as scope growth).
- Nonexistent pod (fake tracker returns `NotFound` for a pod never seeded) — a
  distinct case from "retries exhausted," since it exercises the un-reactored,
  naturally-occurring error path rather than an injected one.
- API error (from either of the above) surfaced verbatim on the final attempt, not
  swallowed or re-wrapped into something that loses the underlying cause.

No special handling needed for `cmd/asterisk-proxy`'s `init()` calling
`pflag.Parse()` — the design doc confirmed this is safe (proven pattern already in
`voip-kamailio-proxy/cmd/kamailio-proxy`, and `pflag` explicitly skips `go test`'s
own flags).

### 2f. Verify

```bash
cd voip-asterisk-proxy
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

(See the note at the end of Step 0 item 2 on which `golangci-lint` binary this
resolves to.) Confirm `k8s.io/api` is a direct require in `go.mod` at this point
if it wasn't already after Step 2a (see that step's note — either timing is fine,
this is just where it must be true by). Confirm the full `annotation_test.go`
suite passes, including the exhausted-retries case's ~5s runtime (don't mistake
that for a hang).

### 2g. Update `docs/operations.md`'s troubleshooting table

`voip-asterisk-proxy/docs/operations.md`'s "Common Failure Modes" table has a
"Pod annotation patch failing" row whose current phrasing ("Not running in
Kubernetes or service account lacks patch permission") is now incomplete and
undersells the severity: the JSONPatch-vs-annotations-absent bug this ticket fixes
was a third cause, and per the design doc's analysis, any failure in this call
site (including RBAC or non-k8s-environment cases already listed) makes
`main()` `return` before the RabbitMQ/ARI/AMI handlers are constructed — i.e. the
whole proxy silently fails to start (exit 0, not a crash), not merely "the
annotation doesn't get set." Update the row's "Likely cause" and "Resolution"
columns to reflect that this failure mode aborts startup entirely, and mention
`--kubernetes_disabled=true` as the fix for genuinely non-k8s environments (as
today) while noting that a k8s-environment failure here is a hard blocker, not a
degradation. This repo's `check-service-docs.sh` hook doesn't mechanically catch
this doc/code drift (it only watches `go.mod` replace-directive changes), so
there's no automated nudge if this step is skipped — do it deliberately.

## Failure handling (applies to Step 1d and Step 2f alike)

If any verification step in Step 1d or Step 2f fails in a way not anticipated by
the design doc (an actual API break, an unexpected go.sum conflict, a lint
finding beyond what's already documented as expected): stop, do not attempt to
push through with a workaround that wasn't part of the reviewed design. Diagnose
first — read the actual error, check whether it's covered by the design doc's
Risks section, and if it's genuinely new, that may mean the design doc itself
needs a fix-and-re-review cycle (small, targeted — not a full restart) before
continuing implementation. This mirrors VOIP-1447's Step 2a precedent: mixed
transient errors (a flaky test, a lock collision from unrelated concurrent work in
the same shared worktree) are expected and just need a clean re-run; anything
that reproduces consistently is not.

## Step 3: bin-call-manager — verify it's genuinely untouched

Per the design doc's Scope section, `bin-call-manager` needs no code changes, but
verify this claim empirically rather than trusting it silently:

```bash
cd bin-call-manager
go mod tidy
git status --short go.mod go.sum
```

If this produces a legitimate non-`k8s.io` diff (see below), run the rest of the
mandatory verification workflow on it too before committing —
`go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
— same as any other `go.sum`-touching change in this repo, no exception just
because the diff arrived indirectly.

Expect zero diff, but a diff here does not automatically mean the design doc's
"zero k8s.io footprint" claim was wrong — `bin-call-manager` locally `replace`s
and imports from `bin-sentinel-manager` (`models/container`), so
`bin-sentinel-manager`'s go.mod requirements participate in `bin-call-manager`'s
own dependency resolution. Bumping `client-go` in `bin-sentinel-manager` could
legitimately raise a **shared non-k8s** transitive dependency (e.g.
`golang.org/x/net`, `google.golang.org/protobuf`) that both modules happen to
pull in, producing a real, benign diff here with the "zero k8s.io footprint"
claim still fully intact. So: inspect what actually changed, don't just react to
"any diff."

- If the diff contains a new `k8s.io/*` line (direct or indirect) — **that** is
  the stop-the-line case. It would mean the design's Scope claim was wrong, and
  needs investigation before proceeding; do not commit it silently.
- If the diff is purely non-`k8s.io` version bumps — that's expected and fine,
  commit it as part of this ticket with its own `bin-call-manager:` bullet in
  Step 5's commit body (a one-line "picked up an updated transitive dependency
  via bin-sentinel-manager's k8s.io/* bump, no k8s.io/* footprint of its own"
  is sufficient — no separate design/plan review needed for a dependency-only,
  no-code-change diff like this).
- If there's genuinely zero diff, nothing to commit for this service, as
  originally expected.

## Step 4: Docker build verification (both services)

Unlike VOIP-1447, this ticket doesn't touch Dockerfiles or the Go toolchain, so a
full docker-build sweep isn't required by the same logic — but since the k8s.io
dependency tree changed substantially (many transitive deps), a docker build
verifies the Dockerfile's own `go mod vendor` step (run fresh inside the build,
independent of this machine's local module cache) still resolves cleanly:

```bash
cd ~/gitvoipbin/monorepo/.worktrees/VOIP-1446-Upgrade-k8s-io-deps-to-GA
docker build -t voip1446-verify-sentinel:test -f bin-sentinel-manager/Dockerfile .
docker build -t voip1446-verify-asterisk:test -f voip-asterisk-proxy/Dockerfile .
docker rmi voip1446-verify-sentinel:test voip1446-verify-asterisk:test
```

## Step 5: Commit and PR

**First, commit the plan documents themselves** — `docs/plans/2026-09-03-voip-1446-k8s-io-ga-upgrade-design.md`
and this plan file are new files in this worktree and belong in the PR, matching
VOIP-1447's precedent (`docs/plans/2026-09-02-fleet-go-toolchain-upgrade-{design,plan}.md`
are tracked on `main`). Fold them into the `bin-sentinel-manager` commit below
(first commit on the branch, `docs:`-prefixed bullet) rather than a separate
commit — no need for a third commit just for documentation on a two-service
ticket. The `docs:` prefix itself is not a hypothetical extrapolation of this
repo's `bin-<service-name>:` convention — it's the literal prefix VOIP-1447 used
for this same kind of bullet (`- docs: Add 2026-09-02-fleet-go-toolchain-upgrade-design.md
and -plan.md`, on `main`), so this plan is following an established precedent, not
inventing one.

Per this repo's branch/commit convention, one commit per service (matches the
VOIP-1447 precedent's granularity, cheap given squash-merge collapses it anyway),
plus a third only if Step 3 found a legitimate `bin-call-manager` diff to commit:

```
VOIP-1446-Upgrade-k8s-io-deps-to-GA

- docs: Add 2026-09-03-voip-1446-k8s-io-ga-upgrade-design.md and -plan.md
- bin-sentinel-manager: Upgrade k8s.io/{api,apimachinery,client-go} v0.36.0-alpha.0
  -> v0.37.0 GA; migrate cache.ListWatch from deprecated ListFunc/WatchFunc to
  ListWithContextFunc/WatchFuncWithContext (run.go and run_test.go), switching
  informer.Run(stopCh) to informer.RunWithContext(ctx) so the migrated fields
  receive the actual caller context instead of one synthesized from a bridged
  channel
```

```
VOIP-1446-Upgrade-k8s-io-deps-to-GA

- voip-asterisk-proxy: Upgrade k8s.io/{apimachinery,client-go} v0.36.0-alpha.0 ->
  v0.37.0 GA (k8s.io/api promoted from indirect to direct by the new test file's
  import); widen patchPodAnnotation's parameter from *kubernetes.Clientset to
  kubernetes.Interface, enabling fake-clientset testing; add annotation_test.go;
  fix a real bug found by writing that coverage -- patchPodAnnotation used
  types.JSONPatchType with an "add" op targeting /metadata/annotations, which
  fails permanently on any pod with no pre-existing annotations (the parent path
  doesn't exist yet); switched to types.MergePatchType, which creates the
  annotations object if absent, also removing the now-unneeded ~1 path-escaping;
  update docs/operations.md's troubleshooting table to reflect that a failure
  here aborts the proxy's startup entirely, not just the annotation
```

If Step 3 found a legitimate `bin-call-manager` diff, a third commit:

```
VOIP-1446-Upgrade-k8s-io-deps-to-GA

- bin-call-manager: Pick up an updated transitive dependency via
  bin-sentinel-manager's k8s.io/* bump (local replace); no k8s.io/* footprint of
  its own, confirmed by go mod tidy producing zero k8s.io lines
```

Before opening the PR: `git fetch origin main`, conflict check
(`git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"`),
review `git log --oneline HEAD..origin/main` for anything landed since this branch
was created. PR body: narrative summary, then bulleted `bin-sentinel-manager:` /
`voip-asterisk-proxy:` project-prefixed changes per this repo's convention — no
markdown headers, no test-plan section, no AI attribution.

**Do not merge without explicit authorization** (repo-standard rule, applies here
same as every other ticket). After merge, per the design doc's Risks section:
`bin-sentinel-manager` redeploys automatically via CircleCI → Komodo — live-verify
on bm-nyc-01 the same way VOIP-1250/VOIP-1447 did (container `running`, boot log
shows the normal Docker-backend reconciliation sequence with no new errors), not
just trust CI green. `voip-asterisk-proxy` has no `-deploy` job — its new image
reaches production through whatever out-of-band mechanism already handles that,
not this ticket's concern to trigger.
