# bin-scheduler-manager — Implementation Plan (VOIP-1283, Phase 1)

Status: approved (plan review loop: 3 rounds — RC, Approve, Approve; final revision incorporates round-3 non-blocking guidance)
Design: [2026-08-01-bin-scheduler-manager-design.md](2026-08-01-bin-scheduler-manager-design.md) (approved, 4-round review)
Branch: `VOIP-1283-Add-bin-scheduler-manager` (worktree `.worktrees/VOIP-1283-Add-bin-scheduler-manager`)

All tasks execute in the worktree. One PR. Commit granularity: one commit per numbered
step (Steps 1+2 land as ONE commit — the module cycle between the new service and
`bin-common-handler` means neither passes verification alone; every commit must pass
the mandatory verification workflow). Squash title = branch name.

New third-party dependency introduced by this PR: `github.com/robfig/cron/v3`
(parser only). Owner package: `models/schedule` exposes the pure helpers
`ValidateCron(spec string) error` (parse + reject zero `Next()`) and
`NextRun(spec string, from time.Time) (time.Time, error)` — both `schedulehandler`
(CRUD validation) and `dispatchhandler` (slot computation) consume these; no other
package touches robfig directly.

## Step 1 — Service skeleton: go.mod, models, config

Files (all under `bin-scheduler-manager/`):
- `go.mod`: module `monorepo/bin-scheduler-manager`, go 1.25.3, replace block copied
  from `bin-tag-manager` **including the self-replace**
  (`replace monorepo/bin-scheduler-manager => ../bin-scheduler-manager` — the
  convention self-replaces, and the `bin-common-handler ↔ bin-scheduler-manager`
  cycle fails to resolve without it). `.gitignore`, `LICENSE` (copy bin-tag-manager's).
- `models/schedule/schedule.go`: `Schedule` struct — `commonidentity.Identity` embed,
  Name, Detail, Type (`Type` string type, `TypeRPC`), Cron, TargetQueue, TargetURI,
  TargetMethod, TargetDataType, TargetData (json.RawMessage), TimeoutMS, RetryMax,
  Enabled, TMNextRun/TMLastRun/TMCreate/TMUpdate/TMDelete (`*time.Time`), json+db tags
  (`db:"...,uuid"` on ids).
- `models/schedule/cron.go`: `ValidateCron`/`NextRun` (robfig/cron/v3 parser, UTC
  enforcement; see header note).
- `models/schedule/field.go`, `filters.go` (FieldStruct: customer_id, enabled, deleted),
  `event.go` (EventTypeScheduleCreated/Updated/Deleted, execution succeeded/failed),
  `webhook.go` (WebhookMessage + ConvertWebhookMessage + CreateWebhookEvent).
- `models/execution/execution.go`: `Execution` struct — Identity embed, ScheduleID,
  TriggerType (`cron`/`manual`), Status (`running`/`success`/`failed`/`abandoned`),
  StatusCode, Error, Result, AttemptCount, DurationMS, TMScheduled, TMDeadline,
  TMStart/TMEnd/TMCreate/TMUpdate/TMDelete; `field.go`, `filters.go` (schedule_id,
  status, deleted).
- `internal/config/main.go`: Bootstrap/LoadGlobalConfig/Get (bin-number-manager
  pattern) with standard vars + SCHEDULER_TICK_INTERVAL_SEC(10),
  SCHEDULER_DISPATCH_CONCURRENCY(10), SCHEDULER_EXECUTION_RETENTION_DAYS(90),
  SCHEDULER_BACKUP_DIR(no default), SCHEDULER_BACKUP_RETENTION_COUNT(7).
- Model unit tests (`*_test.go`) per convention (incl. cron validation table tests:
  valid specs, parse errors, never-matching `0 0 30 2 *` rejection).

## Step 2 — Shared library groundwork (`bin-common-handler`) [same commit as Step 1]

Files:
- `models/outline/queuename.go`: add
  `QueueNameSchedulerEvent/Request/Subscribe = "bin-manager.scheduler-manager.{event,request,subscribe}"`
  and exported `QueueNameRequestAll() []QueueName` enumerating every `*Request`
  constant **declared in `queuename.go`** (33 today — NOT the 30 referenced in
  `send_request.go`; the declaration set is the parity basis, so api/sentinel/user
  queues remain schedulable). Parity unit test (`queuename_test.go`): assert slice
  length + membership against the declared constants, so the hand-maintained list
  cannot silently drift.
- `models/outline/servicename.go`: add `ServiceNameSchedulerManager ServiceName = "scheduler-manager"`.
- `pkg/requesthandler/send_request.go`: add `sendRequestScheduler(...)` private sender
  (pattern: `sendRequestNumber`).
- `pkg/requesthandler/scheduler_schedule.go` (new): typed client methods
  `SchedulerV1ScheduleCreate/Get/Gets/Update/Delete/Execute`, `SchedulerV1ExecutionGets`
  using scheduler-manager **domain models** (per root CLAUDE.md: prefer domain types
  over transport DTOs for new client methods).
- `pkg/requesthandler/main.go`: interface additions + `go generate ./...` mock regen.
- `pkg/requesthandler/scheduler_schedule_test.go` (new): wire-shape tests per rpc.md
  §9.6 (queue, uri, method, marshaled body byte-for-byte, mocked sockhandler).
- `bin-common-handler/go.mod`: `require monorepo/bin-scheduler-manager` + `replace`.

Verify (Steps 1+2 combined commit): full 5-step workflow in `bin-scheduler-manager`
and `bin-common-handler`.

## Step 3 — cachehandler + dbhandler

- `pkg/cachehandler/main.go`: redis/v9 + redsync (bin-flow-manager pattern) + mockgen
  directive. Interface: `Connect`, `ScheduleGet/Set/Delete`,
  `LockSchedule(ctx, scheduleID uuid.UUID, ttl time.Duration) (unlock func(), err)` —
  try-once semantics (`redsync.WithTries(1)`); lock failure returns sentinel
  `ErrLockBusy`.
- `pkg/cachehandler/handler.go`: getSerialize/setSerialize, keys `schedule:<id>`,
  `scheduler:lock:<id>`.
- `pkg/dbhandler/main.go`: DBHandler interface + ErrNotFound + mockgen.
- `pkg/dbhandler/schedule.go`: squirrel CRUD (`scheduler_schedules` const), cache-aside,
  `ScheduleListDue(ctx, now, limit)`, `ScheduleListUninitialized(ctx, limit)` (enabled=1,
  tm_next_run IS NULL), `ScheduleInitNextRun(ctx, id, next) (bool, error)` (CAS expected
  NULL, does NOT touch tm_last_run), active-name lookup `ScheduleGetByCustomerIDName`
  (tm_delete IS NULL).
- `pkg/dbhandler/execution.go`:
  `ScheduleClaimAndCreateExecution(ctx, schedule, next, now, triggerType, tmScheduled)
  (*execution.Execution, error)` — CAS UPDATE (incl. `enabled=1 AND tm_delete IS NULL`)
  + execution INSERT in ONE transaction (design §5.3);
  `ExecutionHasRunning(ctx, scheduleID) (bool, error)` (overlap guard);
  `ExecutionComplete(ctx, id, status, statusCode, errStr, result, attemptCount)
  (bool, error)` — conditional `WHERE status='running'`;
  `ExecutionReapAbandoned(ctx, now) (int64, error)` (single-table UPDATE, design §5.6);
  `ExecutionList` (filters); `ExecutionPrune(ctx, cutoff, batch) (int64, error)`
  (`DELETE WHERE id IN (SELECT id ... LIMIT n)` — sqlite-portable form);
  `ExecutionCountAll`.
- `scripts/database_scripts_test/table_scheduler_schedules.sql`,
  `table_scheduler_executions.sql` (repo `table_` prefix convention; sqlite DDL
  mirroring design §4 incl. unique key `(schedule_id, trigger_type, tm_scheduled)`).
- `pkg/dbhandler/main_test.go` (sqlite + purse), `schedule_test.go`, `execution_test.go`.
- **Double-fire test** (design §10): two goroutine "replicas" against one sqlite DB, N
  parallel `ScheduleClaimAndCreateExecution` for the same slot → exactly one success.
- **kill -9 state-machine test**: claim+insert running, no completion; second instance's
  overlap check sees running → skip; advance clock past tm_deadline; reap → abandoned;
  next claim succeeds.

Verify: full 5-step in bin-scheduler-manager.

## Step 4 — schedulehandler + dispatchhandler + backuphandler

- `pkg/schedulehandler/main.go` (iface+ctor+mockgen), `schedule.go`: Create (validate:
  `schedule.ValidateCron`; method whitelist; `target_queue` ∈
  `outline.QueueNameRequestAll()`; type==rpc; active-name uniqueness), Get/Gets/Update
  (cron or enabled change ⇒ tm_next_run=NULL)/Delete (soft), `db.go` wrappers,
  `event.go` (notifyhandler webhook/event publishing).
- `pkg/dispatchhandler/main.go` (iface `Run(ctx)`, ctor, mockgen), `tick.go`: ticker
  loop (SCHEDULER_TICK_INTERVAL_SEC) → reap → init-uninitialized → list-due → semaphore
  goroutines; `claim.go`: lock (ErrLockBusy → skipped_lock) → overlap check
  (skipped_overlap once per slot — track last-skipped tm_scheduled in memory) → claim tx
  (skipped_cas) → unlock → dispatch; `dispatch.go`: `SendRequest` with per-schedule
  timeout, retry loop (retry_max, 5s linear backoff), completion write (conditional),
  result truncation 60,000 bytes, metrics, execution events; `manual.go`: manual
  execute (no tm_next_run mutation, trigger_type=manual, 409 on overlap, works on
  disabled schedules); `metrics.go`: prometheus registration (design §5.7 set incl.
  execution_rows refreshed per tick, backup_last_bytes).
- `pkg/backuphandler/main.go` (iface+mockgen, `Backup(ctx) (*Result, error)`,
  Result{Path, Bytes}), `backup.go`: DSN parse (go-sql-driver/mysql `ParseDSN`),
  defaults-extra-file (0600 tmp, cleanup), `mysqldump --single-transaction --routines
  --triggers --set-gtid-purged=OFF` via exec.CommandContext behind a `commandRunner`
  interface (mockable), gzip writer, retention prune (keep newest N),
  no-password-in-argv + redacted-logging guarantees; `backup_test.go` asserts argv and
  logged command contain no password material.
- Unit tests for every handler path listed in design §10.

Verify: full 5-step in bin-scheduler-manager.

## Step 5 — listenhandler + subscribehandler + cmd

- `pkg/listenhandler/main.go`: regex routes (design §6 table incl.
  `/v1/executions/prune`, `/v1/backups`), prom histogram (namespace
  `scheduler_manager` — NOT bin-tag-manager's `agent_manager` copy-paste bug),
  `Run(queue, exchangeDelay)`.
- `pkg/listenhandler/models/request/main.go` + `v1_schedules.go`
  (V1DataSchedulesPost/Put, flat per rpc.md §9.5; sibling-service naming convention);
  route handlers `v1_schedules.go`, `v1_executions.go`, `v1_backups.go` under
  `pkg/listenhandler/` (style A domain marshal; errors via cerrors mapping; 409 for
  overlap refusal).
- `pkg/subscribehandler/main.go` + `customermanager.go`: on customer_deleted,
  soft-delete that customer's schedules (nil-customer platform rows unaffected).
- `cmd/scheduler-manager/main.go`: bin-number-manager Bootstrap pattern; wires
  cache/db/requesthandler/notifyhandler/schedulehandler/backuphandler/dispatchhandler/
  listen/subscribe; starts dispatch loop goroutine with chDone shutdown.
- `cmd/scheduler-control/main.go`: direct-DB admin CLI — `schedule list|get|enable|
  disable`, `execution list` (read + enable/disable only in Phase 1).
- Tests: listenhandler routing tests, subscribehandler test, constructor tests.

Verify: full 5-step in bin-scheduler-manager.

## Step 6 — Migrations (bin-dbscheme-manager)

```bash
cd bin-dbscheme-manager/bin-manager
cp alembic.ini.sample alembic.ini   # alembic.ini is gitignored; sample is the tracked source
alembic -c alembic.ini revision -m "scheduler_schedules_create_table"
alembic -c alembic.ini revision -m "scheduler_executions_create_table"
alembic -c alembic.ini revision -m "scheduler_schedules_seed_platform_jobs"
alembic -c alembic.ini heads   # must print exactly ONE head
```
- Tables per design §4 (BINARY(16), DATETIME(6), MEDIUMTEXT result, indexes incl.
  `(status, tm_deadline)`, `(tm_create)`, unique `(schedule_id, trigger_type,
  tm_scheduled)`; InnoDB utf8mb4, raw SQL via `op.execute`).
- Seed migration: three rows per design §8 (`UNHEX(REPLACE(UUID(),'-',''))`, nil
  customer_id `UNHEX('00000000000000000000000000000000')`).
- NEVER run `alembic upgrade`/`downgrade` (repo rule).
- Deliberate deviation from design §8's "one migration creates both tables": three
  revisions (one per table + seed) — cleaner per-revision naming under the
  `<table>_<verb>_<type>` convention; net schema identical.

## Step 7 — Deployment + CI + docs

- `bin-scheduler-manager/Dockerfile`: build stage
  `public.ecr.aws/docker/library/golang:1.25-alpine` (repo standard); runtime stage
  `public.ecr.aws/docker/library/debian:bookworm-slim` + `mysql-community-client`
  8.0.x from MySQL's official APT repo (design §7 deviation, documented in service
  CLAUDE.md).
- `k8s/`: `namespace.yml`; `deployment.yml` (replicas: 2, image placeholder
  `scheduler-manager-image`, `command: ['./scheduler-manager']`, prometheus
  annotations :2112; env pattern: env var `DATABASE_DSN` ← secret `voipbin` key
  `DATABASE_DSN_BIN`, plus RABBITMQ_ADDRESS/REDIS_* per bin-number-manager; only
  SCHEDULER_* vars the config actually reads; backup volume/PVC intentionally NOT
  wired — production uses managed Cloud SQL backups; VOIP-1281 wires the RWX volume
  for self-hosted); `service.yml` (`---`); `kustomization.yml`.
- `.circleci/config.yml`: mapping line
  `bin-scheduler-manager/.*  run-bin-scheduler-manager true` (alphabetical).
- `.circleci/config_work.yml`: `run-bin-scheduler-manager` parameter; workflow block
  (`build-approval` → `-test` → `-build` → `-release`); jobs with
  `source-directory: bin-scheduler-manager` (test), `project-repository:
  bin-scheduler-manager` (build), `image-name: scheduler-manager` (release) — full
  parameter set copied from the bin-tag-manager jobs.
- `bin-number-manager/k8s/cronjob.yml`: **delete**; remove `# - cronjob.yml` from its
  `kustomization.yml`.
- Service docs: `CLAUDE.md` (Class A template + deviations: runtime image, generic
  dispatch as sanctioned exception to bin-common-handler queue-string rule AND
  rpc.md §9.1 typed-methods rule), `README.md`, `docs/architecture.md`,
  `docs/domain.md`, `docs/dependencies.md` (tmpl-generated), `docs/operations.md`
  (≥4 failure modes, config table, metrics incl. `max by (schedule_name)` gauge
  aggregation note, backup dump-and-restore smoke-test runbook).
- Monorepo doc-sync (ordered):
  1. `bash docs/reference/service-taxonomy-gen.sh > docs/reference/service-taxonomy.md`
     (script writes to stdout; extractor.sh greps the file for service class — MUST run
     before extractor).
  2. `bash docs/reference/extractor.sh bin-scheduler-manager` — writes extraction JSON
     to `docs/.docs-gen/`; it does NOT edit service docs, and its published-event
     package whitelist does not know `schedule`/`execution`, so the
     `docs/architecture.md` events section is written by hand.
  3. `docs/reference/rabbitmq-queues-reference.md`: three new scheduler queues.
  4. `docs/conventions/database.md` §7.0 prefix table: add `scheduler_` row.
  5. `docs/architecture/service-dependency-graph.md`: add scheduler-manager node +
     edges (number-manager via dispatch, self-RPC); update service count.
  6. Hardcoded service counts 34/37 → 35/38 where stated: root `CLAUDE.md`,
     `bin-common-handler/CLAUDE.md`, `docs/workflows/special-cases.md` (verify each
     actually states a count before editing).
- RST docs (`bin-api-manager/docsdev`): consciously N/A — Phase 1 has no user-visible
  API surface (restate in PR body so reviewers see it was a decision, not an omission).
- Deliberate deviation from design §3: `docs/reference/service-inventory.md` is NOT
  updated (it is a docs-refresh protected-directory list, not a service catalog);
  `service-taxonomy.md` + `service-dependency-graph.md` are the correct registries.
- Also sync the "Projects Affected" list (not just the count) in
  `docs/workflows/special-cases.md`.

Verify (Step 7): `circleci config validate` on the processed config where tooling
allows (else YAML lint), and `kubectl kustomize bin-number-manager/k8s` +
`kubectl kustomize bin-scheduler-manager/k8s` to prove both kustomizations still build.

## Step 8 — go.mod replace propagation + full verification

- Add `require monorepo/bin-scheduler-manager v0.0.0-...` + `replace ... =>
  ../bin-scheduler-manager` to **every** module that requires `bin-common-handler`:
  `grep -rl "monorepo/bin-common-handler" --include=go.mod .` (36 modules; script the
  edit, **excluding** `bin-common-handler` and `bin-scheduler-manager` themselves —
  both already carry the directives from Steps 1-2; net edit targets: 35 modules).
- Note for reviewers: between the Steps 1-2 commit and this step's commit, the 35
  sibling modules are transiently unbuildable (missing replace). This never reaches
  `main` (squash merge) and path-filtered CI is gated behind manual build-approval;
  the final branch state is what verification proves.
- Per `docs/workflows/special-cases.md` (bin-common-handler change → every dependent
  module), run in **each** module whose `go.mod` changed — no downgraded shortcut:
  ```bash
  go mod tidy && go mod vendor && go generate ./... && go clean -testcache && \
  go test ./... && golangci-lint run -v --timeout 5m
  ```
  (`go clean -testcache` is mandated there as "the #1 cause of missing test failures"
  when skipped. Each service's `go.mod` change also path-triggers its CI workflow —
  all gated behind manual build-approval — so local verification is the real gate.)
- `make lint-docs` at repo root (root docs/ files changed in Step 7).
- dbscheme: local `alembic heads` single-head check (CI migration-lint re-checks).

## Step 9 — Code review loop, PR

- Code review loop per session policy: minimum 3 rounds, 2 consecutive approvals,
  max 30; independent reviewer agents (go-reviewer + security/code reviewers).
- Pre-PR: `git fetch origin main`; `git merge-tree` conflict check; review
  `git log --oneline HEAD..origin/main`.
- Commit/PR body lists affected projects: `bin-scheduler-manager:` (new service),
  `bin-common-handler:` (queues/servicename/client methods),
  `bin-number-manager:` (CronJob removal), `bin-dbscheme-manager:` (migrations),
  `monorepo:`-level docs, `.circleci`. PR title `VOIP-1283-Add-bin-scheduler-manager`.
  No AI attribution. Single PR. NO merge without explicit user authorization.

## Acceptance mapping (design §2 ↔ ticket)

| Criterion | Where proven |
|---|---|
| number-renew daily, no CronJob manifest | seed migration (Step 6) + cronjob.yml deletion (Step 7) |
| kill -9 takeover, no double-fire, test-proven | Step 3 dbhandler tests (double-fire, kill -9 state machine) |
| state/last-run via API + Prometheus | Steps 4/5 (metrics, GET routes) |
| conventions: tests/mocks/CLAUDE.md/README | Steps 1-7 + Step 8 verification |

## Explicit out-of-scope (this PR)

- bin-trigger-sender deletion (follow-up after VOIP-1281 confirms nothing references it)
- billing/ai/customer ticker migration (Phase 2 ticket, filed at completion)
- sandbox/install onboarding (VOIP-1281), doctor checks (VOIP-1280)
- api-manager/openapi exposure (Phase 3)
