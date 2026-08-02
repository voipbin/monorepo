# VOIP-1279: asterisk-proxy fail-fast on listen handler failure — Design

Date: 2026-08-02
Ticket: VOIP-1279
Service: voip-asterisk-proxy

## Problem

When the proxy's RabbitMQ listen handler fails at startup (observed: queue declaration
failed because the delayed-message exchange plugin was missing), the error is only
logged and the process keeps running. The container stays Up, periodic Redis
address-key updates continue, but the request queues (`asterisk.<id>.request`,
`asterisk.call.request`, ...) have zero consumers. `restart: always` never fires
because the process never exits. Call setup is silently broken until a manual
restart.

Root causes (two layers):

1. `pkg/listenhandler/main.go` `Run()` does *all* work (queue declaration +
   consumer startup) inside a goroutine and unconditionally returns `nil`.
   Errors are logged and swallowed.
2. `cmd/asterisk-proxy/main.go` startup error paths use `log.Errorf` + `return`,
   which exits `main()` with **exit code 0**. Even if the error were propagated,
   the process would not signal failure.

   This is only partially addressed by this design: per the fatal-vs-soft
   principle in Change 2, `getAsteriskIDAddress` and the listen-handler paths
   move to `Fatalf`, but `setProxyInfoRedis` and `setProxyInfoAnnotation`
   keep their `log.Errorf` + `return` (exit 0) paths. This is a deliberate
   scoping choice, not an oversight, and it is harmless for the restart-heal
   requirement in this ticket: both `restart: always` (docker-compose) and
   Kubernetes' default `RestartPolicy: Always` restart on container exit
   regardless of exit code, they don't require non-zero specifically. Exit
   code correctness for those two paths is left as a smaller, separate
   cleanup, not required for this ticket's acceptance criteria.

Transient-loss analysis (revised after review): the shared
`bin-common-handler/pkg/rabbitmqhandler` layer self-heals **connection**-level
loss only (`reconnector` + `redeclareAll`, infinite 1s retry on `connect()`).
Two narrower gaps exist above that layer and are explicitly scoped below:

- Consumer **re-registration** after reconnect (`reconsumerAll`) is a
  best-effort retry bounded at 3 attempts; on exhaustion it logs and gives up
  permanently, leaving a live process with zero consumers on that queue. This
  is the same symptom as the ticket, but triggered post-startup rather than at
  startup, and fixing it means changing `bin-common-handler` (38-service blast
  radius per its admission rule). **Out of scope for this ticket** — filed as
  a follow-up (see Out of scope).
- `redeclareAll` itself logs-and-continues on a failed `QueueDeclare`, which is
  the upstream cause of the above. Same follow-up.

This ticket's fix targets the **startup** failure path only, which is both
what VOIP-1275 observed and what the acceptance criteria describe (the
sandbox replay is a start/break/fix/recover sequence, not a live-reconnect
sequence).

## Deployment model check

`docs/subsystems.md` documents two supported models: Kubernetes sidecar (proxy
and Asterisk as separate containers in one pod) and single-container
(Asterisk + proxy binary in one image). **This cannot be resolved from the
monorepo alone** — there are no Kubernetes manifests here (infra lives in a
separate repo), so which model production actually runs is not verifiable
from source under review. Rather than assume sidecar, this design proceeds on
a worse-case-safe basis:

- The ticket's own "Fix direction" explicitly asks for fail-fast-and-exit —
  this was written by the reporter after live-diagnosing the exact incident,
  so the fail-fast behavior itself is a requirement, not a design choice up
  for renegotiation here.
- Under single-container, a proxy crash-loop would restart the co-located
  Asterisk process and drop in-progress calls. This is a real, unverified
  residual risk. It is called out explicitly in the PR description so
  whoever deploys this can confirm the running topology before rollout, and
  captured in `docs/subsystems.md` (see Documentation updates) so it isn't
  lost. It is not something this design can mitigate in-process — no amount
  of in-process retry changes the outcome once the process has decided to
  exit, so gating on "confirm sidecar in production" (a deploy-time check,
  not a code change) is the correct place for this risk, not a code
  workaround.
- **This risk is not confined to single-container.** `Fatalf` exits the whole
  process, which also kills the ARI/AMI event pump managed by
  `pkg/eventhandler` (`evtHandler.Run()`, launched at
  `cmd/asterisk-proxy/main.go:125`, before the listen handler). Today, a
  proxy stuck in the ticket's bug still relays ARI/AMI events (hangups, state
  changes) for in-progress calls even though new RPC requests time out —
  upstream services can at least see calls end and clean up/bill correctly.
  After this fix, on `Fatalf`, event relay stops too, **in sidecar mode as
  well as single-container**: in-progress calls lose their terminal-event
  path until the restarted proxy reconnects to ARI/AMI. This is accepted as
  an intentional trade-off — the ticket's explicit ask is "exit non-zero,
  don't run in a half-working state" — but it is a real behavior change
  beyond "new call setup fails faster," not a risk-free improvement, and
  should be stated as such in the PR description rather than only "restart
  policy heals it."
- The observed trigger (a missing broker plugin) is a **broker-side,
  fleet-correlated** condition: every asterisk-proxy pod hits the same
  failure simultaneously, since they all declare against the same broker.
  Combined with the point above, this fix turns "one proxy quietly stuck"
  into "the whole Asterisk fleet's event plane going down at once" for the
  duration of the outage — worse in aggregate blast radius than today's bug,
  though also more visible (crash-looping pods vs. a silently-stuck fleet)
  and self-healing once the broker-side issue is fixed, which today's
  behavior is not. Net, this still matches the ticket's explicit ask
  (visibility and self-healing over silent, unbounded outage), but it should
  be understood as a trade of "quieter, unbounded breakage" for "louder,
  bounded, fleet-wide breakage," not a strict improvement with no downside.
- No in-process retry/backoff is added (see next paragraph for why) — a
  crash-loop under sustained failure is accepted as intentional given the
  ticket's explicit ask, with backoff timing left entirely to the container
  runtime's restart policy (docker-compose / Kubernetes `RestartPolicy`),
  which already implements backoff and is the correct layer for it.

**Why no in-process retry:** an earlier revision of this design added a
5s/60s in-process retry loop to avoid a "storm" on a brief broker blip. On
review this didn't hold up: (a) connection-level loss is already retried
unboundedly one layer down in `rabbitmqhandler.connect()`, so the new retry
added nothing there; (b) the ticket's own trigger — a missing broker plugin —
is deterministic, not transient, so retrying it for 60s only delays the
correct fatal exit by 60s on every single restart attempt; (c) applying the
retry to queue declaration but not consumer startup (`ConsumeRPC`) left the
consumer path exposed to the exact same "storm" scenario the retry was meant
to prevent, since `startConsumers` can fail on the same broker-side
conditions (`channel.Qos`, `channel.Consume`). Rather than extend the retry
to both paths with a still-unanchored budget, this revision removes it:
`Run()` fails fast on the first declaration or consumer-startup error, and
backoff between restart attempts is delegated entirely to the container
runtime's restart policy — which is already the standard backoff mechanism
(`CrashLoopBackOff` on Kubernetes) and does not duplicate logic in-process.

## Fix

### Change 1 — `pkg/listenhandler/main.go`

- `Run()` declares all listen queues (permanent + volatile) **synchronously**
  (no retry — see above) and returns an error immediately if any declaration
  fails.
- Before declaring, `Run()` validates the **combined** permanent + volatile
  queue-name list (each split on `,`, concatenated the same way
  `listenRun` already does at `pkg/listenhandler/main.go:100-127`) for two
  conditions, failing fast with a plain validation error (no broker call) if
  either holds:
  - any element is an empty string (config typo, e.g. a trailing comma in
    `--rabbitmq_queue_listen`);
  - any two elements are equal (config typo, e.g. a copy-pasted duplicate
    entry, or `--rabbitmq_queue_listen` accidentally containing the same
    value as the derived volatile queue name `asterisk.<id>.request`).

  This second check is the one that actually matters operationally.
  `registerConsumer` (`bin-common-handler/pkg/rabbitmqhandler/consume.go:108-120`)
  rejects a second registration for a `queueName` that already has one, and
  under this design that becomes a permanent, self-inflicted `Fatalf` crash
  loop with no possible self-recovery (no broker fix can resolve a
  process-local duplicate list) — worse than today's behavior, where it is
  just a logged error and one queue silently loses its consumer. The
  empty-string check is weaker justification on its own: a single trailing
  comma produces exactly one empty element (not a duplicate, since
  `registerConsumer` only rejects *repeated* names), and an empty queue name
  reaching `QueueDeclare` triggers AMQP's broker-generated-name behavior,
  which this codebase does not capture — the local `r.queues[""]` entry and
  the broker's actual (different, generated) queue name diverge, so it fails
  differently (at bind/consume time) rather than cleanly. It's still worth
  rejecting outright rather than letting either failure mode occur, but the
  duplicate check is the one closing a real crash-loop hole.
- Consumer startup (`ConsumeRPC`) stays in per-queue goroutines (it blocks
  until ctx cancellation on success, per `bin-common-handler` semantics — see
  Transient-loss analysis for why a `ConsumeRPC` error return is a
  startup-only signal here). The `ListenHandler` interface's `Run()`
  signature changes from `Run() error` to `Run() (<-chan error, error)`: the
  returned `error` is the synchronous declare/validation result (as today,
  just now non-nil on failure); the returned channel is owned and sized by
  `listenHandler` itself (buffered to the actual `len(listenQueues)`, known
  internally once the split completes — not passed in by the caller, which
  would otherwise have to duplicate the split logic to size it, reintroducing
  the exact drift risk the duplicate-name validation above closes). Each
  consumer goroutine sends its non-nil `ConsumeRPC` error on that channel
  (never on a nil result — `ConsumeRPC` also returns nil on `ctx.Done()`,
  which cannot occur today since `context.Background()` is passed, but the
  send is guarded explicitly so a future context change can't turn a clean
  shutdown into `Fatalf: err: <nil>`). This avoids a package-level mutable
  var (no `fatalf` indirection) and keeps a single, testable exit point in
  `main()`.
- `pkg/listenhandler` has no `go:generate` directive or mock today (the
  service's only mock is in `pkg/servicehandler`); this interface-signature
  change does not require mock regeneration, and none is added as part of
  this change.
- Note on the `bin-call-manager` reference: its `Run()` matches the
  synchronous-declare half of this change, but its `ConsumeRPC` goroutine
  still only logs-and-swallows (same as asterisk-proxy today). The
  error-channel escalation is a new, deliberately local pattern for this
  service, not an existing monorepo convention — flagged in case a
  service-wide follow-up is later warranted.
- Fatal-vs-soft principle applied consistently in this change (see Change 2
  for the concrete split): **fail fast only for failures that block
  establishing the RabbitMQ consumer topology** (queue declaration, consumer
  registration, and the inputs those two require — the Asterisk ID used in
  the volatile queue name). Auxiliary features unrelated to the consumer
  topology (Kubernetes annotation, GCS recording config, Redis address
  registration) are explicitly out of scope and keep today's soft-failure
  behavior — see Change 2 for why each one is classified the way it is.

### Change 2 — `cmd/asterisk-proxy/main.go`

```go
chErr, err := listenHandler.Run()
if err != nil {
    log.Fatalf("Could not run the listen handler correctly. err: %v", err)
}
...
select {
case sig := <-chSigs:
    log.Infof("Terminating asterisk-proxy. sig: %v", sig)
case err := <-chErr:
    log.Fatalf("Listen handler failed permanently. err: %v", err)
}
```

`Fatalf` is applied narrowly — only to the paths that actually strand the
process with a live-but-nonfunctional consumer topology, per-path:

- `listenHandler.Run()` returning an error at call time (now possible after
  Change 1) → `Fatalf` at the call site, same as before but now reachable.
- Listen-handler consumer failure delivered via `chErr` → `Fatalf` as above.
- `getAsteriskIDAddress` failure → `Fatalf`. Under the stated principle this
  is in scope because the resolved ID feeds directly into the volatile queue
  name (`asterisk.<id>.request`) — without it, `Run()` cannot even attempt
  correct queue declaration. (It also happens to gate Redis registration and
  annotation patching, but that is not why it is classified fatal here — see
  `setProxyInfoAnnotation` below for the contrasting case, to keep the
  principle, not the consequence, doing the classifying.)

Left as `log.Errorf` + `return` (unchanged), based on review of each path:

- `setProxyInfoRedis` — this one does *not* cleanly fall outside the stated
  principle: `docs/subsystems.md` states upstream services read this Redis
  key to construct the volatile queue name for targeted RPC routing, so a
  correctly-declared volatile queue that nobody can address is functionally
  close to this ticket's symptom (unreachable consumer topology). It is left
  non-fatal anyway, but for a narrower, code-grounded reason: the function
  itself cannot fail (dial happens lazily inside the update goroutine at
  `cmd/asterisk-proxy/main.go:196-198`, whose errors are already logged
  per-iteration, not returned to the caller) — so converting its `nil`-only
  return path to `Fatalf` would be a no-op today regardless of the
  principle. The *actual* gap — a persistent Redis write failure silently
  leaving the routing key stale/missing forever, logged but never escalated
  — is real and is called out explicitly under Out of scope rather than
  silently waved through as "unrelated."
- `setProxyInfoAnnotation` — under the stated principle, this is out of scope
  by construction: Kubernetes pod-annotation patching is bookkeeping metadata
  unrelated to establishing the RabbitMQ consumer topology (it doesn't feed
  queue names, and nothing in `pkg/listenhandler` reads it). It also happens
  to already have a documented soft-failure resolution in
  `docs/operations.md` (missing `patch` verb → `--kubernetes_disabled=true` or
  grant RBAC), which corroborates the classification but is not itself the
  rule being applied.
- `evtHandler.Run()` — confirmed dead path: `eventHandler.Run()` unconditionally
  returns `nil` (`pkg/eventhandler/main.go:63-72`), so this call site cannot
  currently return an error either way. Left unchanged; no behavior change to
  claim here.

## Out of scope

- `pkg/eventhandler` / ARI event loop: auto-reconnects every 1s internally
  (documented in service docs); does not strand the process. Unchanged.
- Post-startup consumer loss (`reconsumerAll` exhaustion, channel-level AMQP
  errors that end `consumeRPCWorker` without triggering `reconnector`): real
  gap, same symptom as this ticket, but requires a `bin-common-handler` change
  with a 38-service blast radius. Follow-up ticket to be filed after this
  merges, referencing this design doc.
- Health-check endpoint exposing consumer liveness (ticket lists it as an
  alternative; fail-fast is the chosen primary mechanism for the startup
  case; a liveness endpoint would also help catch the post-startup gap above
  and is a candidate approach for the follow-up ticket).
- Persistent Redis write failure for the address-key routing entry
  (`setProxyInfoRedis`'s per-iteration goroutine at
  `cmd/asterisk-proxy/main.go:196-198`) is logged but never escalated, even
  though it can leave the volatile queue effectively unaddressed by upstream
  services. Not fixed here (left non-fatal for the code-level reason given in
  Change 2), but noted so it isn't mistaken for "already covered."

## Acceptance criteria mapping

| AC | How satisfied | Residual gap |
|----|---------------|---------------|
| Exits non-zero when listen handler cannot be established | Change 1 (sync declare, no retry) + Change 2 (`Fatalf` via error channel), applied uniformly to declaration failure and consumer-startup failure | Post-startup consumer loss after a successful start is not covered — see Out of scope |
| `restart: always` self-heals after transient RabbitMQ failure | Two distinct mechanisms: (a) broker-unreachable at boot — `sockHandler.Connect()` blocks and retries internally every 1s, process never exits, no restart needed because it never stopped trying; (b) topology failure (this ticket's actual trigger, e.g. missing plugin) — immediate `Fatalf` → container restart → retries the whole sequence including (a), with backoff timing owned by the restart policy, not this code | Case (a) window is not itself a `Fatalf` path; documented as intentional (connection retry is already unbounded and self-healing without needing a process restart) |
| Sandbox scenario replay (break plugin → start → fix → recover) | Gates merge — see Testing | Recovery latency is bounded by the restart policy's backoff (see deployment note below), not by anything in this change |

Deployment note: `restart: always` is docker-compose syntax; production runs
on GKE where the equivalent is the pod's default `RestartPolicy: Always` with
exponential backoff (up to 5 min between attempts via CrashLoopBackOff). MTTR
after a broker fix therefore differs materially between sandbox
(docker-compose, near-immediate restart) and production (K8s, up to 5 min
backoff) — call this out during the sandbox replay so the observed recovery
time isn't assumed to hold in production. This is also the reason no
in-process retry/backoff was added (see Deployment model check): duplicating
backoff logic in-process would only create two backoff timers disagreeing
with each other.

## Testing

- Unit (mock `SockHandler`), run with `-race`:
  - `Run()` returns a validation error immediately (no queue declaration
    attempted) when the permanent or volatile queue-name list contains an
    empty element (trailing-comma config case).
  - `Run()` returns error immediately when permanent-queue declaration fails;
    no attempt is made to declare the volatile queue or start any consumer
    (fail-fast, no partial topology left half-declared beyond what the broker
    itself already accepted before the failing call). Note the precise shape
    of the ticket's own trigger for this case: `queueCreateNormal`
    (`bin-common-handler/pkg/rabbitmqhandler/queue.go:50-62`) first calls
    `QueueDeclare` — which **succeeds**, the queue now exists on the broker —
    and only then calls `queueConfig`, which fails at
    `ExchangeDeclareForDelay` because the delayed-message plugin is missing.
    So "declaration fails" here means the queue is declared but never
    finishes its exchange/bind config; the volatile queue is never attempted
    at all (not "declared but its own declare failed").
  - `Run()` returns error immediately when volatile-queue declaration fails,
    after the permanent queue already succeeded; assert no consumer goroutine
    is started for either queue.
  - `Run()` declares both queues and starts one consumer goroutine per queue
    on success.
  - A consumer goroutine's `ConsumeRPC` error is delivered on the (buffered)
    error channel without blocking, including the case where two queues both
    fail.
- `main()`-level: exercised indirectly (exit-code behavior is not something
  Go unit tests observe directly); the `select` wiring is simple enough that
  code review substitutes for a dedicated test — call out explicitly that
  AC1's exit-code requirement is verified end-to-end only by the sandbox
  replay below, not by unit tests.
- Full monorepo verification workflow (`go mod tidy && go mod vendor &&
  go generate ./... && go test ./... && golangci-lint run`).
- Sandbox replay per AC3 — **gates merge**, not deferred: break the
  delayed-message plugin, start the stack, confirm proxies exit/restart-loop,
  fix the plugin, confirm proxies recover consumers without manual
  intervention. Record observed timing. While in the crash-loop window,
  additionally check RabbitMQ queue depth on the permanent queue: its
  `QueueDeclare` succeeds even when the subsequent exchange config fails (see
  the precise mechanism noted in Testing above), so it exists on the broker
  with zero consumers and publishers keep enqueuing into it — see
  Documentation updates for the operator guidance this produces.

## Documentation updates (same commit, per root CLAUDE.md service-docs-sync rule)

- `docs/architecture.md` — listen-handler routing/behavior section: note the
  synchronous, no-retry declare-and-consume with fatal-on-first-failure
  behavior, and the empty-queue-name validation.
- `docs/operations.md`:
  - "RabbitMQ consumer not receiving requests" row — update resolution: the
    process now exits non-zero and restarts on sustained topology failure;
    add guidance that a *crash-looping* proxy (vs. a silently-idle one) is now
    the expected failure signature during a topology break, and that queue
    depth on the (already-declared) permanent queue should be checked after
    recovery for a backlog of stale, already-timed-out call requests
    accumulated during the crash-loop window.
  - Add a row (or extend the `--interface_name` config row) noting that a
    misconfigured/missing interface now causes a fatal exit rather than a
    silent no-op.
- `docs/subsystems.md` — Deployment Notes / Container model section: add a
  note that the single-container model, if used, means a listen-handler
  crash-loop restarts the co-located Asterisk process and drops in-progress
  calls; confirm the running topology (sidecar vs. single-container) before
  relying on this fail-fast behavior in that configuration.
