# VOIP-1279: asterisk-proxy fail-fast — Implementation Plan

Date: 2026-08-02
Ticket: VOIP-1279
Design: [2026-08-02-asterisk-proxy-fail-fast-design.md](2026-08-02-asterisk-proxy-fail-fast-design.md)

## Steps

### 1. `pkg/listenhandler/main.go`

- Change `ListenHandler` interface: `Run() error` → `Run() (<-chan error, error)`.
- Rewrite `Run()`:
  - Build `listenQueues` by splitting+concatenating permanent and volatile
    queue-name lists exactly as `listenRun` does today.
  - Validate `listenQueues`: reject if any element is `""`, reject if any
    two elements are equal (return a plain `fmt.Errorf`, no broker call).
  - Declare each queue synchronously (permanent as `"normal"`, volatile as
    `"volatile"`) via `h.sockHandler.QueueCreate`. On first failure, return
    `(nil, err)` immediately — no further declarations, no consumers
    started.
  - On success, create `chErr := make(chan error, len(listenQueues))`.
  - For each queue, start a goroutine calling `ConsumeRPC`; only send on
    `chErr` when the returned error is non-nil (log it either way). This
    guard matters because `ConsumeRPC` also returns nil on `ctx.Done()` —
    unreachable today since `context.Background()` is passed, but the guard
    prevents a future context change from turning a clean shutdown into
    `Fatalf: err: <nil>` (design lines 176-181).
  - Return `(chErr, nil)`.
- Remove the old `listenRun` helper (folded into `Run()`), or keep as a
  private helper invoked synchronously from `Run()` — implementer's choice,
  keep it small and testable.
- Audit: confirm `Run()` has exactly one caller (`cmd/asterisk-proxy/main.go`)
  and that `pkg/listenhandler` has no `go:generate` directive / mock to
  regenerate for this interface change (grep for `ListenHandler` and for
  `.Run()` call sites across the monorepo before editing, to catch any other
  consumer). As of this plan being written there are none — re-verify at
  implementation time in case that has changed.

### 2. `cmd/asterisk-proxy/main.go`

- Review all five startup error paths; change exactly two, per the design's
  per-path classification:
  - `getAsteriskIDAddress` failure → change to `log.Fatalf`.
  - `listenHandler.Run()` returning a non-nil `error` → change to
    `log.Fatalf` at the call site (replaces today's `log.Errorf` + `return`).
  - `setProxyInfoRedis`, `setProxyInfoAnnotation`, `evtHandler.Run()` →
    leave unchanged (`log.Errorf` + `return`, exit code 0). Note in the diff
    or PR description that this is deliberate scoping (design lines 26-35):
    both docker-compose `restart: unless-stopped`/`always` and Kubernetes
    `RestartPolicy: Always` restart on any container exit regardless of exit
    code, so these paths don't need a non-zero exit to be healed by the
    restart policy; fixing their exit code is a separate, smaller cleanup
    not required by this ticket's acceptance criteria.
- Capture `chErr` from `listenHandler.Run()`.
- Replace the trailing `sig := <-chSigs` with:
  ```go
  select {
  case sig := <-chSigs:
      log.Infof("Terminating asterisk-proxy. sig: %v", sig)
  case err := <-chErr:
      log.Fatalf("Listen handler failed permanently. err: %v", err)
  }
  ```

### 3. Tests — `pkg/listenhandler/main_test.go` (new or extended)

Construct the handler under test directly as `&listenHandler{...}` (matching
the existing pattern in `proxy_handler_test.go`), setting at least
`sockHandler`, `rabbitQueueListenRequestPermanent`, and
`rabbitQueueListenRequestVolatile`. Use `MockSockHandler`
(`bin-common-handler/pkg/sockhandler`) with `gomock.NewController(t)` /
`defer mc.Finish()`, matching existing package convention.

**Synchronization note (required, not optional):** `ConsumeRPC` runs in a
goroutine per queue. A test asserting "`ConsumeRPC` was called" or "error was
delivered on `chErr`" must not call `mc.Finish()` (or let `defer` run it)
before that goroutine has actually invoked the mock — otherwise the test is
flaky (unmet-expectation failures) or panics ("call after Finish"). Use
`EXPECT().ConsumeRPC(...).DoAndReturn(func(...) error { defer close(done); return tt.consumeErr })`
with a per-queue `done chan struct{}`, and have the test `select` on all
`done` channels with a timeout (e.g. `time.After(2*time.Second)`) before
proceeding to assertions or letting the deferred `Finish()` run.

Cases:

- `Run()` returns a validation error and makes no `QueueCreate` call when the
  combined permanent+volatile queue list contains an empty element (e.g.
  permanent queue list `"asterisk.call.request,"`).
- `Run()` returns a validation error and makes no `QueueCreate` call when the
  combined list contains a duplicate element — cover both a duplicate within
  the permanent list and the volatile-name-collision case (permanent list
  value equal to the derived `asterisk.<id>.request`).
- `Run()` returns a non-nil error immediately when the permanent queue's
  `QueueCreate` fails; assert (via `gomock` call expectations / `Times(0)`)
  that `QueueCreate` for the volatile queue and `ConsumeRPC` are never
  called; returned channel is nil.
- `Run()` returns a non-nil error immediately when the volatile queue's
  `QueueCreate` fails after the permanent queue's `QueueCreate` succeeded;
  assert `ConsumeRPC` is never called for either queue.
- `Run()` succeeds: both queues' `QueueCreate` called, one `ConsumeRPC`
  goroutine started per queue, a non-nil `chErr` and nil `error` returned.
- A single `ConsumeRPC` mock returning an error delivers that error on
  `chErr`, read with a timeout (not a blocking read) to prove the channel is
  buffered and the send didn't need a receiver ready.
- Both `ConsumeRPC` mocks (permanent and volatile) returning errors
  concurrently both land on `chErr` without either goroutine blocking on
  send — this is the case that actually exercises the buffer being sized to
  `len(listenQueues)` rather than 1; read both off `chErr` with a timeout.
- Run the package's tests with `-race` (folded into step 4, not a separate
  invocation).

### 4. Verification workflow (mandatory, run in `voip-asterisk-proxy/`)

```bash
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test -race ./... && \
golangci-lint run -v --timeout 5m
```

### 5. Documentation updates (same commit)

- `voip-asterisk-proxy/docs/architecture.md` — listen-handler section: describe
  synchronous no-retry declare, empty/duplicate validation, fatal-on-first-
  failure, and the error-channel consumer-failure signal.
- `voip-asterisk-proxy/docs/operations.md`:
  - Update "RabbitMQ consumer not receiving requests" row resolution.
  - Update/add the `--interface_name` config row to note fatal-on-missing.
  - Add guidance to check permanent-queue depth after recovering from a
    crash-loop window.
- `voip-asterisk-proxy/docs/subsystems.md` — Deployment Notes: add the
  single-container residual-risk note (Asterisk restart on proxy crash-loop).

### 5b. Rollout-safety check (before requesting merge)

The new empty/duplicate queue-name validation converts a config quirk that
today is a logged oddity into a permanent, self-inflicted crash loop no
broker-side fix can resolve (design lines ~130-153). Since this monorepo does
not contain the Kubernetes manifests for production deployment, add an
explicit check as part of the PR, not an assumption: confirm (with whoever
owns the deployment config, or by inspecting it directly if accessible) that
every deployed asterisk-proxy instance's `--rabbitmq_queue_listen` /
`RABBITMQ_QUEUE_LISTEN` value is non-empty, has no stray/trailing commas, and
does not collide with the derived volatile name `asterisk.<id>.request`.
State this check's result in the PR description.

### 6. Sandbox replay (gates merge, per design's Testing section)

This exercises the **sidecar** case only. In `~/gitvoipbin/sandbox`, all
three asterisk-proxy instances run as separate containers using
`network_mode: "service:asterisk-<call|registrar|conference>"` to share
Asterisk's network namespace — a proxy crash-loop there does not restart
Asterisk. That matches the design's scoping (the single-container risk is a
deploy-time topology check, not something this replay can exercise) — state
this explicitly in the PR description rather than letting the replay result
read as covering both deployment models.

Getting the branch's code into the sandbox: `docker-compose.yml` pins the
three proxy services by image digest, not by local build context, so
starting the stack as-is exercises the *published* image, not this branch —
that would produce a false "replay passed" against pre-fix behavior. Before
replaying:

1. Build from the **monorepo/worktree root**, not the service subdirectory —
   the Dockerfile does `COPY ./ .` then `cd voip-asterisk-proxy && go mod vendor`,
   so it needs the whole tree (including sibling modules referenced by
   `replace` directives, e.g. `bin-common-handler`) as build context:
   ```
   docker build -t voipbin/voip-asterisk-proxy:VOIP-1279-local -f voip-asterisk-proxy/Dockerfile .
   ```
   (run from the worktree root, e.g.
   `~/gitvoipbin/monorepo/.worktrees/VOIP-1279-Asterisk-proxy-fail-fast-listen-handler/`).
   Check for a `Makefile`/build script in `~/gitvoipbin/sandbox` first and
   prefer it if one already encodes this correctly.
2. Add a `docker-compose.override.yml` in the sandbox directory pinning
   `image: voipbin/voip-asterisk-proxy:VOIP-1279-local` for all three proxy
   services (`asterisk-call-proxy`, `asterisk-registrar-proxy`,
   `asterisk-conference-proxy`).
3. Confirm each container actually started from the local image
   (`docker inspect` image ID, or a log line unique to the new build) before
   proceeding — don't rely on "the compose file says so."

Breaking the delayed-message plugin: `~/gitvoipbin/sandbox/docker-compose.yml`'s
`rabbitmq` service entrypoint runs, on every container start: if
`/opt/rabbitmq-extra-plugins/rabbitmq_delayed_message_exchange-3.13.0.ez`
(volume-mounted from `./config/rabbitmq/plugins/`) is **missing**, download
it; then unconditionally `rabbitmq-plugins enable --offline
rabbitmq_delayed_message_exchange`. This means **removing or renaming the
file does not work** — both make the existence check false and trigger a
fresh download, silently undoing the break on the next container start. The
break must happen without changing whether the file is present at that exact
path:

1. Back up
   `~/gitvoipbin/sandbox/config/rabbitmq/plugins/rabbitmq_delayed_message_exchange-3.13.0.ez`
   to a location outside the mounted directory (e.g. `/tmp`), so it can be
   restored byte-for-byte afterward.
2. Truncate or corrupt the file **in place, keeping the same filename** (e.g.
   `truncate -s 0 <path>` or overwrite with garbage bytes) so the
   entrypoint's existence check stays true (no re-download) but
   `rabbitmq-plugins enable` fails to load a valid plugin.
3. Also check whether `enabled_plugins` state persists in the `rabbitmq_data`
   named volume from a prior run (independent of the `.ez` file) — if the
   plugin was already enabled in a previous sandbox session, a corrupted
   `.ez` alone may not un-enable it. If so, additionally remove/reset the
   relevant named volume (`docker compose down -v` scoped to that volume, or
   the sandbox's documented data-reset procedure) before starting the replay,
   so the replay starts from a genuinely plugin-absent state.

Replay procedure:

- With the plugin broken and the branch's build in place, start the RabbitMQ
  service first and verify the **precondition** before starting the proxies:
  RabbitMQ reports healthy (`docker compose ps` shows `Up (healthy)`, or
  equivalent), and `rabbitmq-plugins list -e` does **not** list
  `rabbitmq_delayed_message_exchange`. This matters because the design draws
  a hard line between "broker unreachable" (proxies retry forever inside
  `sockHandler.Connect()`, never `Fatalf`) and "broker reachable but topology
  declaration fails" (proxies `Fatalf`) — if RabbitMQ itself is unhealthy
  rather than merely missing the plugin, the replay would land in the wrong
  case and still show "proxies crash-looping," giving a false pass.
- Start (or restart) the three proxy services. Confirm all three exit
  non-zero and enter a restart loop (not silently "Up") — check both exit
  code, using the actual container IDs since these services have no
  `container_name` set (e.g.
  `docker inspect --format='{{.State.ExitCode}}' $(docker compose ps -aq asterisk-call-proxy)`,
  repeated per service), and logs.
- Fix the plugin: restore the backed-up `.ez` file exactly (reverse step 2
  above), and if step 3 required a volume reset, allow the entrypoint to
  re-provision from a clean state. Confirm all three proxies recover
  consumers without manual `docker compose restart`.
- Check RabbitMQ permanent-queue depth (`asterisk.call.request` etc.) for
  backlog accumulated during the outage window; note it in the PR
  description per the ops-doc guidance from step 5.
- Record observed restart/recovery timing, and the sandbox's actual restart
  policy value as configured (`unless-stopped` for all three proxy services,
  not `always`) alongside it — the design's AC2 table distinguishes
  docker-compose from Kubernetes MTTR on the premise of "restart on any
  exit," which `unless-stopped` also satisfies, but record the literal
  policy string observed rather than assuming `always`.
- Revert the `docker-compose.override.yml` / local image tag, confirm the
  `.ez` file backup was restored and matches the original (checksum), and
  confirm the plugin state is otherwise as it was before the replay.

### 7. File the follow-up ticket

Per the design's Out-of-scope section (post-startup consumer loss —
`reconsumerAll` exhaustion after 3 attempts, `redeclareAll` log-and-continue
on failed `QueueDeclare`): file a follow-up Jira ticket against
`bin-common-handler` referencing this design doc, describing the gap and
that any fix there has a 38-service verification blast radius per the
package's admission rule. Link it from the PR description.

## Order of work

1 → 2 → 3 → 4 (iterate 1-4 until verification is clean) → 5 → 5b → 6 → 7.

## PR description must include

- Summary of the fix direction (from the design doc).
- The accepted trade-off: event-pump loss on `Fatalf`, fleet-correlated
  restart risk, single-container residual risk (ask deployer to confirm
  topology before rollout).
- Result of the rollout-safety check (step 5b): confirmed queue-name config
  is empty/duplicate-clean for all deployed instances, or an explanation of
  what could not be confirmed and why.
- Sandbox replay results and timing, explicitly noting it covers the
  sidecar case only, plus the actual restart policy value observed.
- Link to the follow-up ticket filed in step 7.
