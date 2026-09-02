# k8s.io/{api,apimachinery,client-go} alpha → GA Upgrade — Design

VOIP-1446. Originally opened as a small dependency-version bump on top of VOIP-1418
(bin-sentinel-manager's dual backend). Investigation found every GA release of
`k8s.io/client-go` requires `go 1.26.0+`, which cascaded (via local `replace`
directives) into a fleet-wide Go toolchain upgrade — split out and completed as
[VOIP-1447](https://voipbin.atlassian.net/browse/VOIP-1447) (merged 2026-09-02, 38
of the monorepo's 39 modules now on `go 1.27.1` — the 39th, `bin-trigger-sender`,
was deliberately excluded as confirmed dead code, unrelated to this ticket). This
document covers what actually remains for
VOIP-1446 now that its blocker is resolved: the `k8s.io/*` dependency bump itself,
plus one deferred code migration and one deferred test addition, both of which were
explicitly scoped out of VOIP-1447 to keep that PR mechanical.

See `docs/plans/2026-09-02-fleet-go-toolchain-upgrade-design.md` for the toolchain
work's own design/evidence — not repeated here. See the VOIP-1446 Jira comments for
the full round-by-round issue-analysis history.

## Context

`bin-sentinel-manager` and `voip-asterisk-proxy` are the only two modules in the
monorepo that import `k8s.io/*` (confirmed by grep across all 38 in-scope modules,
carried over from the VOIP-1446/1447 investigation and re-confirmed for this
document — see Scope below). Both currently pin:

```
k8s.io/api v0.36.0-alpha.0
k8s.io/apimachinery v0.36.0-alpha.0
k8s.io/client-go v0.36.0-alpha.0
```

Running a pre-release (alpha) version of a core dependency in self-hosted production
code is the underlying hygiene problem this ticket exists to fix. `v0.37.0` is the
current latest GA for all three (released 2026-08-26, verified directly against
`proxy.golang.org`; no `v0.37.x` patch exists yet, next cycle is `v0.38.0-alpha.0`
already underway). `client-go@v0.37.0`'s own go.mod requires `k8s.io/api v0.37.0` and
`k8s.io/apimachinery v0.37.0` (version-consistent) and declares `go 1.26.0` — already
satisfied by both services' current `go 1.27.1` (from VOIP-1447), so this bump
introduces no further toolchain requirement. **This "v0.37.0 is latest" fact is
time-sensitive** — re-run the same `proxy.golang.org` check at implementation time
rather than trusting this document if any real time has passed since it was
written (2026-09-03); target whatever is the actual latest GA at that moment
instead of assuming it is still exactly `v0.37.0`.

Both services already carry a deliberate SA1019 lint suppression pointing at this
ticket:

```go
// ListFunc/WatchFunc are deprecated in favor of ListWithContext/WatchWithContext as
// of a later client-go than the v0.36.0-alpha.0 this service currently pins
// (VOIP-1418). Migrating is part of VOIP-1446's k8s.io/* GA bump, not this
// purely-mechanical Go toolchain upgrade (VOIP-1447) -- deferred rather than done
// here to keep this PR's blast radius to go.mod/Dockerfile/CI only.
&cache.ListWatch{
    ListFunc: func(options metav1.ListOptions) (runtime.Object, error) { //nolint:staticcheck // see above, VOIP-1446
        ...
    },
    WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) { //nolint:staticcheck // see above, VOIP-1446
        ...
    },
},
```

This exists in both `bin-sentinel-manager/pkg/k8swatchhandler/run.go` (production)
and `run_test.go`'s `newTestInformer()` helper (test fixture) — identical
construction, identical suppression, both need the same migration.

## Scope

**In scope:**

1. `bin-sentinel-manager`: `k8s.io/{api,apimachinery,client-go}` `v0.36.0-alpha.0` →
   `v0.37.0`, plus whatever transitive `k8s.io/klog/v2`, `k8s.io/kube-openapi`,
   `k8s.io/utils`, `sigs.k8s.io/*` versions `go mod tidy` resolves to for that GA
   line. `ListFunc`/`WatchFunc` → `ListWithContextFunc`/`WatchFuncWithContext`
   migration in both `run.go` and `run_test.go`, including switching `run.go`'s
   `informer.Run(stopCh)` to `informer.RunWithContext(ctx)` (removing the `stopCh`
   bridge goroutine) — required alongside the field rename, not optional; see
   "What actually needs to change" below for why.
2. `voip-asterisk-proxy`: `k8s.io/{apimachinery,client-go}` `v0.36.0-alpha.0` →
   `v0.37.0` (`k8s.io/api` moves from indirect to direct as a side effect of the new
   test file's imports — see below). New `annotation_test.go` covering
   `patchPodAnnotation`, made possible by widening its parameter from
   `*kubernetes.Clientset` to `kubernetes.Interface`. Also, as a fix surfaced by
   writing that test's coverage (not a separate scope addition — see "What actually
   needs to change" below): switch `patchPodAnnotation`'s patch from
   `types.JSONPatchType` to `types.MergePatchType`, fixing a real bug where a pod
   with no pre-existing annotations can never receive one.

**Explicitly not in scope (confirm before implementation, do not re-derive from
scratch — this was already resolved in VOIP-1447):**

- `go` directive, Dockerfile base image, CircleCI `go_image` anchor for either
  service — already `go 1.27.1` / `golang:1.27.1-alpine` / `cimg/go:1.27.1` via
  VOIP-1447. Nothing to change here.
- `bin-call-manager` — was pulled into VOIP-1447's scope only because of the Go
  toolchain cascade (it locally `replace`s `bin-sentinel-manager`'s
  `models/container` package). That cascade is fully resolved, confirmed the
  strongest possible way: `bin-call-manager/go.mod` and `go.sum` contain zero
  `k8s.io` lines at all — Go's module graph pruning already excludes that whole
  dependency tree from it (a stronger guarantee than "no `k8s.io/*` import in its
  own source," which is also true but is the weaker of the two checks). Verification
  step for the implementation plan: run `go mod tidy` in `bin-call-manager` after
  this ticket's changes land elsewhere and confirm zero diff — cheap, and closes off
  any doubt.
- `voip-kamailio-proxy` — has an inert `replace monorepo/bin-sentinel-manager` with
  no matching `require` and no actual import; confirmed unaffected, no change.
- Every other service in the monorepo — confirmed zero `k8s.io/*` references outside
  the two in scope.

## What actually needs to change, per service

### bin-sentinel-manager

**go.mod**: bump the three direct `k8s.io/*` requires to `v0.37.0`; let `go mod
tidy` resolve the indirect `k8s.io/klog/v2`, `k8s.io/kube-openapi`, `k8s.io/utils`,
`sigs.k8s.io/{json,randfill,structured-merge-diff/v6,yaml}` versions rather than
hand-picking them — those are internal to the k8s.io release train and should track
whatever `client-go v0.37.0` itself was built and tested against.

**`pkg/k8swatchhandler/run.go`**: in `runInformer`, change

```go
&cache.ListWatch{
    ListFunc: func(options metav1.ListOptions) (runtime.Object, error) { //nolint:staticcheck // see above, VOIP-1446
        options.LabelSelector = target.LabelSelector
        return h.clientset.CoreV1().Pods(target.Namespace).List(ctx, options)
    },
    WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) { //nolint:staticcheck // see above, VOIP-1446
        options.LabelSelector = target.LabelSelector
        return h.clientset.CoreV1().Pods(target.Namespace).Watch(ctx, options)
    },
},
```

to

```go
&cache.ListWatch{
    ListWithContextFunc: func(listCtx context.Context, options metav1.ListOptions) (runtime.Object, error) {
        options.LabelSelector = target.LabelSelector
        return h.clientset.CoreV1().Pods(target.Namespace).List(listCtx, options)
    },
    WatchFuncWithContext: func(watchCtx context.Context, options metav1.ListOptions) (watch.Interface, error) {
        options.LabelSelector = target.LabelSelector
        return h.clientset.CoreV1().Pods(target.Namespace).Watch(watchCtx, options)
    },
},
```

Named `listCtx`/`watchCtx` rather than `ctx` deliberately: `runInformer`'s own `ctx`
stays live and in scope for `handleUpdate`/`handleDelete`/`waitForCacheSync` later in
the same function, and this file's comment density is otherwise entirely about
getting cancellation semantics right — shadowing `ctx` here, even though harmless,
would read as careless in a file this deliberate about that exact topic.

**This also requires changing how the informer is run**, a few lines below this
struct literal (`stopCh`/bridge-goroutine block, currently):

```go
stopCh := make(chan struct{})
go func() {
    <-ctx.Done()
    close(stopCh)
}()

done := make(chan struct{})
go func() {
    defer close(done)
    informer.Run(stopCh)
}()
```

to:

```go
done := make(chan struct{})
go func() {
    defer close(done)
    informer.RunWithContext(ctx)
}()
```

**Why this second change is necessary, not optional:** an earlier draft of this
migration kept `informer.Run(stopCh)` unchanged and claimed the `listCtx`/`watchCtx`
passed into the new fields would be "the same `ctx` value, just threaded explicitly
instead of captured." That is false, and matters enough to fix rather than
footnote. `sharedIndexInformer.Run(stopCh)` (v0.37.0) is defined as
`s.RunWithContext(wait.ContextForChannel(stopCh))` — it does not receive or forward
`runInformer`'s `ctx` at all; it synthesizes a *new*, unrelated `context.Context`
from the channel. Left as `Run(stopCh)`, `listCtx`/`watchCtx` would still work
correctly (cancellation still propagates: `ctx.Done()` closes `stopCh`, which
`ContextForChannel`'s internal watcher turns back into a cancelled context, one
goroutine hop later, carrying no values either way — so nothing observably breaks),
but the field rename's own justification would rest on a claim that isn't true, and
a later reader who takes "same value" at face value could reasonably assume `ctx`
values set upstream (e.g. request-scoped logging fields, should any get added later)
propagate into the list/watch calls when they would not.

Switching to `informer.RunWithContext(ctx)` — a method already on the
`cache.SharedInformer` interface `SharedIndexInformer` embeds, so no type change is
needed at the call site — removes the `stopCh` channel and its bridge goroutine
entirely and passes `runInformer`'s actual `ctx` straight through. This makes the
"same context" property genuinely true instead of merely equivalent-in-effect, and
is a net reduction in code (one fewer channel, one fewer goroutine) rather than an
addition — so it does not meaningfully widen this migration's footprint. Nothing
else in the function depends on `stopCh` (`waitForCacheSync` uses its own
`context.WithTimeout(ctx, ...)`, independent of this channel), so removing it is
self-contained.

Remove both `//nolint:staticcheck` comments (no longer suppressing anything) and
remove/rewrite the explanatory comment block above the struct literal (it currently
says "deferred... as part of VOIP-1446" — that deferral is what this change
resolves, so the comment either goes or gets replaced with a short note on why the
`WithContext` variants and `RunWithContext` are used together, matching the file's
existing comment density).

This is a safe, behavior-preserving migration otherwise: `client-go v0.37.0`'s
`Reflector` (`tools/cache/reflector.go`) exclusively calls the `WithContext`
*methods* (`ListWithContext`/`WatchWithContext`), never the legacy `List`/`Watch`
methods — that much is already true today, before this migration, and doesn't
change. What the migration actually changes is one layer deeper: right now,
`ListWithContextFunc`/`WatchFuncWithContext` (the *fields*) are nil, because this
code populates only the deprecated `ListFunc`/`WatchFunc` fields — so today,
`ListWithContext()`/`WatchWithContext()` are the methods Reflector calls, but
internally they fall back to invoking the old fields since the new ones are empty.
After this migration, the new fields are populated directly, so the same methods
Reflector already calls invoke the new fields directly instead of falling back —
removing an indirection, not changing which method Reflector calls or introducing
a new one. Separately confirmed during design review: v0.37.0 adds no
new `Deprecated:` markers to any other API this file uses (`SetWatchErrorHandler`,
`informer.Run`/`RunWithContext`, `cache.WaitForCacheSync`) — only advisory
"Contextual logging:" comments — so this migration does not open a second front of
staticcheck findings elsewhere in the file.

**`pkg/k8swatchhandler/run_test.go`**: identical migration in `newTestInformer()`
(lines ~537/540 as of this writing) — same field rename, same nolint removal.
`newTestInformer`'s own doc comment says it "builds an informer against the fake
clientset without running it," and that's accurate: its only two callers
(`Test_waitForCacheSync_*`) never call `Run`/`RunWithContext` on the result — the
whole point of this helper is to hand `waitForCacheSync` something that never
syncs. So there is no `Run` call in this file to migrate at all (unlike `run.go`);
only the `ListWatch` struct literal changes here:

```go
func (h *k8sWatchHandler) newTestInformer() cache.SharedIndexInformer {
    return cache.NewSharedIndexInformer(
        // See the matching comment in run.go -- WithContext ListWatch fields, matching what
        // client-go's Reflector actually calls.
        &cache.ListWatch{
            ListWithContextFunc: func(listCtx context.Context, options metav1.ListOptions) (runtime.Object, error) {
                return h.clientset.CoreV1().Pods(watchedNamespace).List(listCtx, options)
            },
            WatchFuncWithContext: func(watchCtx context.Context, options metav1.ListOptions) (watch.Interface, error) {
                return h.clientset.CoreV1().Pods(watchedNamespace).Watch(watchCtx, options)
            },
        },
        &corev1.Pod{},
        0,
        cache.Indexers{},
    )
}
```

`context.Background()`, previously hardcoded inside the two closures, is simply
dropped — since the informer built here is never run, the closures themselves are
never invoked in the current tests either. The field rename here is not required
for the code to compile or the existing tests to keep passing; it is done so this
file doesn't keep the deprecated `ListFunc`/`WatchFunc` fields (which would
otherwise reintroduce a fresh, unexplained SA1019 finding once the matching
`//nolint` comment and its "deferred to VOIP-1446" framing are removed from
`run.go`) — this file's copy of that same deferral needs to be resolved too, not
left dangling with no ticket to point at.

### voip-asterisk-proxy

**go.mod**: bump `k8s.io/apimachinery` and `k8s.io/client-go` direct requires to
`v0.37.0`. The new test file below constructs a `*corev1.Pod` value directly (to
seed the fake clientset), which is a new direct import of `k8s.io/api/core/v1` —
`go mod tidy` will therefore promote `k8s.io/api` from indirect to a direct require
as a result of adding the test, not as a separate deliberate step. Called out here
so the go.sum/go.mod diff review during implementation doesn't flag it as
unexpected. `k8s.io/klog/v2`, `k8s.io/kube-openapi`, `k8s.io/utils`,
`sigs.k8s.io/*` indirects follow `go mod tidy`, same as bin-sentinel-manager.

**`cmd/asterisk-proxy/annotation.go`**: widen `patchPodAnnotation`'s first parameter

```go
func patchPodAnnotation(clientset *kubernetes.Clientset, namespace, podName, annotationKey, annotationValue string) error {
```

to

```go
func patchPodAnnotation(clientset kubernetes.Interface, namespace, podName, annotationKey, annotationValue string) error {
```

Verified safe: the function's only client call is
`clientset.CoreV1().Pods(namespace).Patch(...)`, which is defined on
`kubernetes.Interface` — no `*Clientset`-only method is used anywhere in the
function. The one caller, `setProxyInfoAnnotation`, already holds the value as
`*kubernetes.Clientset` (from `kubernetes.NewForConfig`), which satisfies
`kubernetes.Interface` automatically — no caller-side change needed. This mirrors
the pattern `bin-sentinel-manager/pkg/k8swatchhandler` already uses (its `clientset`
field is typed `kubernetes.Interface`, injected, fake-clientset-testable).

**Same function, second change** (see the "Finding surfaced by writing this test
plan" note below for the full justification): switch the `Patch` call's patch type
and payload from RFC 6902 JSON Patch to RFC 7386 JSON Merge Patch —

```go
patchPayload := []map[string]string{
    {
        "op":    "add",
        "path":  fmt.Sprintf("/metadata/annotations/%s", escapedAnnotationKey),
        "value": annotationValue,
    },
}
patchBytes, err := json.Marshal(patchPayload)
...
clientset.CoreV1().Pods(namespace).Patch(context.TODO(), podName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
```

to

```go
patchPayload := map[string]any{
    "metadata": map[string]any{
        "annotations": map[string]string{
            annotationKey: annotationValue,
        },
    },
}
patchBytes, err := json.Marshal(patchPayload)
...
clientset.CoreV1().Pods(namespace).Patch(context.TODO(), podName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
```

which also drops the now-unneeded `escapedAnnotationKey := strings.ReplaceAll(annotationKey, "/", "~1")`
line entirely (merge patch keys are plain JSON object keys, not RFC 6901 JSON
Pointer path segments, so no escaping is needed regardless of whether
`annotationKey` contains `/`).

**`cmd/asterisk-proxy/annotation_test.go`** (new file): table-driven tests against
`k8s.io/client-go/kubernetes/fake.NewClientset()` (already part of the `client-go`
module the service already depends on — no new dependency). Minimum coverage:

- **Pod with pre-existing (non-empty) annotations** — seed the fake clientset with a
  `*corev1.Pod` whose `ObjectMeta.Annotations` already has at least one key, assert
  the new annotation lands via the fake tracker after a first-attempt success, and
  assert the patch payload shape.
- **Pod with NO pre-existing annotations** (`ObjectMeta.Annotations == nil`) — see
  the finding below; this case is not optional, add it and expect it to currently
  fail before the fix described below is applied.
- Retry-then-succeed — a fake clientset `PrependReactor` returning an error for the
  first few calls, verifying the function retries and eventually succeeds.
- Exhausted retries — reactor always errors, verify the function returns a
  wrapped error after exactly `defaultMaxRetries` attempts (not fewer, not more).
- Nonexistent pod / API error surfaced verbatim (not swallowed) on the final
  attempt.

**Finding surfaced by writing this test plan, and the fix it requires (in scope for
this ticket):** `patchPodAnnotation` currently builds an RFC 6902 JSON Patch with a
single `add` operation targeting `/metadata/annotations/<key>`. JSON Patch's `add`
operation requires the parent object (`/metadata/annotations`) to already exist —
if a pod has never had any annotation applied to it (`ObjectMeta.Annotations` is
`omitempty` and genuinely absent from the object, which is the normal state for a
freshly created pod that nothing else has annotated yet), the patch fails with "doc
is missing path" on every single attempt, all `defaultMaxRetries` of them, since
retrying an inherently-malformed patch never helps. This is not a client-go-version
issue — the same failure would occur against the real apiserver today, on
`v0.36.0-alpha.0`, unrelated to this ticket's dependency bump. Its likelihood is
environment-dependent — the K8s annotation path only runs when the
`kubernetes_disabled` flag is left at its default `false` (an opt-out flag, not an
opt-in one — this path is enabled unless explicitly turned off), and in many real
clusters something else (a CNI plugin, an admission webhook) has already put at
least one annotation on a pod before this code runs, which would mask the bug in
practice — so "a pod with zero pre-existing annotations" is a real, reachable
failure mode, not a guaranteed-to-occur one. **Its impact where it does occur is
worse than a missing label, though**: `main.go:93-99` calls
`setProxyInfoAnnotation` and, on any error, logs and `return`s from `main()`
*before* the RabbitMQ request handler, the ARI/AMI event handler, or the service
handler are ever constructed (those are all created later in the same function).
So the actual consequence is not "the pod runs with an empty asterisk-id
annotation" — it's that `voip-asterisk-proxy` never finishes starting at all, after
burning ~5s on the doomed retries, and the co-located Asterisk instance is
completely unreachable over the message bus. The process also exits cleanly (a bare
`return`, not `log.Fatalf`), so the container reports success rather than a
detectable failure, which is a worse diagnostic shape than an outright crash would
be. `bin-sentinel-manager`'s K8s watch backend never seeing this pod's annotation
is a secondary, downstream symptom of the same startup abort, not the primary
consequence. Worth fixing regardless of how often it triggers today, since the fix
is small and this function is already being touched.

This is being fixed as part of this ticket rather than filed as a separate
follow-up, because: (a) it is only discoverable as a direct consequence of the test
coverage this ticket already adds (the function had zero test coverage before this
ticket, precisely because `patchPodAnnotation` couldn't be tested without the
`kubernetes.Interface` widening this ticket is already making), and (b) the fix is
small and contained to the same function already being touched, not a separate
subsystem. **Fix**: switch the patch from `types.JSONPatchType` (RFC 6902) to
`types.MergePatchType` (RFC 7386 JSON Merge Patch) — a merge patch of
`{"metadata":{"annotations":{"<key>":"<value>"}}}` creates the `annotations` object
if it is absent, rather than requiring it to pre-exist, and correctly merges into it
if present. This also removes the need for the current manual `~1`-escaping of `/`
characters in the annotation key (`strings.ReplaceAll(annotationKey, "/", "~1")`),
since merge patch addresses fields by normal JSON object nesting rather than
RFC 6901 JSON Pointer paths — a net simplification, not just a bug fix. The
"pod with no pre-existing annotations" test case above is what would have caught
this had it existed before; add it first (expect it to fail against the current
JSONPatch implementation), then apply the `MergePatchType` fix, then confirm it
passes — a small TDD loop within this otherwise dependency-bump-shaped ticket.

**Retry-case timing, explicitly accepted rather than left ambiguous**: four
identifiers are involved here and are easy to conflate, so spelled out precisely —
the retry loop's actual sleep comes from a *local variable*
`retryDelay := time.Millisecond * 500` declared inside the loop
(`annotation.go:117`), which shadows the *unused package-level constant*
`retryDelay = 3 * time.Second` (see Non-goals) and has nothing to do with the *log
message* on the preceding line (`annotation.go:116`), which separately references
the *constant* `defaultRetryDelay` (`main.go:63`) purely for its printed value. The
local variable's `500ms` and `defaultRetryDelay`'s `500ms` happen to be numerically
identical, which is a coincidence, not a dependency — the sleep does not read
`defaultRetryDelay`, it hardcodes the same number independently. That coincidence
is why the 5-second-cost estimate below holds regardless of which of these *two*
500ms-valued identifiers a reader assumes governs the sleep (it would not hold if a
reader instead assumed the unused 3-second package constant governs it — it does
not govern anything, live or dead code alike). It is not evidence any of them are
the same value on purpose. The fourth identifier, `defaultMaxRetries = 10`
(`main.go:62`), is the one that genuinely governs the loop bound — the only one of
the four that isn't shadowed, dead, or merely cosmetic. With no injectable clock or
configurable retry count reaching `patchPodAnnotation` today, the "exhausted
retries" test case costs on the order of 5 seconds of real wall-clock sleep
(10 × 500ms), and "retry-then-succeed" costs proportionally less depending on how
many attempts it takes. This ticket's Non-goals section deliberately excludes
touching the retry constants (see below) — that exclusion stands, and the ~5s cost
is accepted as-is rather than resolved by adding clock/delay injection, because (a)
`voip-asterisk-proxy` has no CircleCI `-test` job (see below), so this cost is paid
only by whoever runs the local verification workflow, not by every CI run, and (b)
making the delay injectable would mean widening `patchPodAnnotation`'s signature
further (a `time.Duration`/count parameter, or a package-level indirection) purely
to serve test speed, which is exactly the kind of scope growth this ticket's
Non-goals are trying to avoid. If this function later gains real callers with
tighter latency requirements, revisit then — not preemptively here.

**`annotation_test.go` will be the first test file in `cmd/asterisk-proxy`, and that
package's `init()` calls `pflag.Parse()` and starts a Prometheus HTTP listener
goroutine** — worth pre-empting the obvious "does `go test` even work here" doubt
rather than letting an implementer re-discover it. It does: this is the exact
`init()`-calls-`pflag.Parse()` shape already proven safe elsewhere in this repo —
`voip-kamailio-proxy/cmd/kamailio-proxy` has the identical pattern and an existing,
CI-gated (`voip-kamailio-proxy-test`) passing test suite. The pinned `pflag`
version explicitly recognizes and skips `go test`'s own `-test.*` flags rather than
erroring on them. No special handling needed in the new test file.

This service has no CircleCI `-test` job (pre-existing, unrelated to this ticket —
`voip-asterisk-proxy`'s pipeline only has a `-build` job, unlike
`voip-rtpengine-proxy` which does have `-test`). The new test file therefore only
runs as part of the mandatory local verification workflow before commit, not gated
in CI. This is a known, accepted gap being carried forward, not something this
ticket is scoped to fix (adding a CI test job for this service is a separate,
larger change — CircleCI job wiring, not a dependency bump — and is left as a
possible future follow-up, not filed as a blocking task here since it doesn't
block this ticket's own completion criteria).

## Risks and mitigations

**API surface break between v0.36.0-alpha.0 and v0.37.0 GA.** The APIs actually used
by both services — `rest.InClusterConfig`, `kubernetes.NewForConfig`,
`kubernetes.Interface`/`*Clientset`, `CoreV1().Pods().{List,Watch,Patch}`,
`cache.NewSharedIndexInformer`, `cache.SharedIndexInformer`,
`cache.DeletedFinalStateUnknown`, `SetWatchErrorHandler`, `corev1.Pod`,
`metav1.{ListOptions,PatchOptions}`, `types.MergePatchType` (the target of this
ticket's own patch-type fix above — `types.JSONPatchType` is being removed from
use entirely, not carried forward) — are among the oldest, most stable surfaces in
client-go (predate the alpha/GA distinction this ticket is even about).
Mitigation: the full verification workflow (`go build`, `go test`,
`golangci-lint`) will catch any actual breakage immediately; this is not a
runtime-only risk category like VOIP-1447's GODEBUG concern.

**Production deployment blast radius and rollback (this is a materially different
question from "no K8s traffic to test against" below, and needs its own answer).**
`bin-sentinel-manager`'s CircleCI pipeline is test → build → **deploy to bm-nyc-01
via Komodo**, same as every other in-fleet service — merging this ticket rebuilds
and redeploys the actual production `bin-sentinel-manager` binary, with a new
`k8s.io/*` dependency tree, even though the code path that tree serves (the K8s
watch backend) carries no production traffic today (the Docker backend does, see
below). The redeploy itself is still real production exposure for the container:
a build failure, a panic on startup, or a broken `go.mod`/`go.sum` would take down
the service that VoIPBin's stranded-call detection actually depends on (the Docker
backend), regardless of whether the K8s code path is exercised. Post-deploy check
(implementation plan should make this an explicit step, matching the VOIP-1250 and
VOIP-1447 precedent of live-verifying via Komodo rather than trusting CI green):
confirm the container is `running`/healthy on bm-nyc-01 and its boot log shows the
normal Docker-backend reconciliation sequence (asterisk-id resolution, docker event
stream opened) with no new errors. Rollback: revert this PR's merge commit and
redeploy the prior image tag via Komodo, same mechanism as any other service on this
fleet — nothing new needed. `voip-asterisk-proxy`'s CircleCI pipeline has a
`-build` job only, no `-deploy` job (pre-existing, confirmed in Scope/Non-goals) —
so this ticket's `voip-asterisk-proxy` changes produce a new image that CI publishes
but nothing in this repo automatically deploys; whatever out-of-band mechanism
already manages that image (same situation VOIP-1447 documented for the three
`voip-*-proxy` services generally) picks it up on its own schedule, not this
ticket's concern to trigger or verify.

**No production K8s backend traffic to validate against.** `bin-sentinel-manager`'s
K8s watch backend is not the one actually running in production on bm-nyc-01 (a
bare-metal Docker host) — the Docker backend is. This was already true before this
ticket and is unrelated to it; the existing unit/fake-clientset test suite is the
only verification surface available, same as it was for the original VOIP-1418
work. Not a new risk this ticket introduces, but worth being explicit that "deployed
and verified live" (the bar VOIP-1447 met) is not achievable here — unit/integration
test coverage is the ceiling.

**`go mod tidy` pulling in an indirect version with its own issues.** Low
probability given `client-go v0.37.0`'s own go.sum pins its indirects to versions
the k8s.io project itself tested against. Mitigation: the verification workflow's
`go test`/`golangci-lint` steps would surface anything, and the diff review (design
doc + PR review) will inspect the actual `go.sum` diff for anything surprising
(a new module appearing, a major-version jump in something unrelated) before commit.

**Removing the nolint-suppressed deprecation from two files touches an
explanatory comment block that itself narrates the deferral.** Low risk, purely
editorial — the comment either gets deleted (since the code now reflects the
"after" state) or rewritten to explain the `WithContext` choice going forward. Not
a functional risk, flagged here only so the implementation plan doesn't skip it and
leave a stale comment claiming a migration is deferred when it no longer is.

## Non-goals

- Does not touch `bin-call-manager`, `voip-kamailio-proxy`, or any other service —
  confirmed zero `k8s.io/*` footprint outside the two in scope.
- Does not add a CircleCI `-test` job for `voip-asterisk-proxy` — pre-existing gap,
  out of scope (see above).
- Does not change `go` directive, Dockerfile base image, or CircleCI `go_image`
  anchor for either service — already done by VOIP-1447.
- Does not address the pre-existing dead `maxRetries`/`retryDelay` constants in
  `annotation.go` (the loop bound actually comes from `defaultMaxRetries`
  in `main.go`; the sleep is a local `retryDelay := time.Millisecond * 500`
  declared inside the retry loop, which shadows the unused package-level
  `retryDelay` constant and ignores `defaultRetryDelay` entirely, even though
  that constant happens to hold the identical value — see the Risks section's
  "Retry-case timing" entry for the full detail) — flagged during issue analysis
  as a pre-existing, unrelated finding. Left as a possible opportunistic
  follow-up, not bundled into this dependency-bump
  ticket to keep its diff focused; may be worth a small separate NOJIRA cleanup
  later.
- Does not attempt to deploy/verify the K8s watch backend live in production — no
  production traffic exists on that code path (see Risks above); unit/fake-clientset
  test coverage is the verification ceiling for this ticket.

## References

- [VOIP-1446](https://voipbin.atlassian.net/browse/VOIP-1446) Jira ticket and its
  issue-analysis comment history.
- [VOIP-1447](https://voipbin.atlassian.net/browse/VOIP-1447) and
  `docs/plans/2026-09-02-fleet-go-toolchain-upgrade-design.md` — the Go toolchain
  work this ticket depended on.
- `k8s.io/client-go` v0.37.0 source, specifically `tools/cache/listwatch.go` and
  `tools/cache/reflector.go` (github.com/kubernetes/client-go, tag `v0.37.0`).
