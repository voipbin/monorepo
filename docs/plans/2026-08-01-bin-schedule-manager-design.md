# bin-schedule-manager — Design (VOIP-1283, Phase 1)

Status: approved (design review loop: 4 rounds — RC, RC, Approve, Approve; final revision incorporates round-4 non-blocking implementation guidance)
Ticket: VOIP-1283 — absorb external cron jobs into the platform
Related: VOIP-1280 (doctor integration), VOIP-1281 (sandbox/install onboarding), VOIP-1282 (campaign/queue scheduler investigation, open)

## 1. Problem

Scheduled work in VoIPBin is spread across three disjoint mechanisms, none of which is
deployment-surface-portable or multi-replica-safe:

1. **K8s CronJob + `bin-trigger-sender`** — exactly one exists (`number-renew`,
   `bin-number-manager/k8s/cronjob.yml`), and it is currently commented out of
   `kustomization.yml`, i.e. not deployed by `kubectl apply -k`. The install repo and
   sandbox provide no CronJob equivalent at all.
2. **In-process `time.Ticker` goroutines** — billing-manager (monthly top-up, failed-event
   retry), ai-manager (proposal sweep), customer-manager (unverified cleanup, frozen
   expiry). All run on `replicas: 2` with **no leader election or lock**; the billing
   top-up is money-affecting.
3. **RabbitMQ `x-delayed-message` exchange** — one-shot deferred RPCs only
   (`bin-manager.delay`), published transient (not broker-restart-safe). Not a recurring
   scheduler and not treated as one here.

Decision (2026-08-01, CEO/CPO): eliminate external cron entirely. Schedules become
platform data, executed by a new microservice — wherever the services run, the schedules
run.

## 2. Goals / Non-goals

**Phase 1 goals (this design):**

- New standard-convention monorepo service `bin-schedule-manager` (MySQL via
  DATABASE_DSN, RabbitMQ RPC, Redis, Prometheus :2112).
- Schedule definitions in the DB, managed via internal RPC API. No redeploy to change a
  schedule.
- Single-active dispatch: N replicas never double-fire a job (test-proven), and a
  schedule never runs concurrently with itself (Forbid semantics, §5.4).
- Dispatch = RabbitMQ RPC using the existing `sock.Request` invocation shape (queue, uri,
  method, data) — absorbing what `bin-trigger-sender` does, as an internal library call.
- Execution audit trail (per-run rows) + per-job Prometheus metrics + last/next-run
  visibility via API (doctor integration point for VOIP-1280).
- Migrate `number-renew` as the first schedule (seeded by migration).
- In-stack scheduled DB backup for self-hosted deployments (mysqldump path), seeded
  **disabled** by default.
- Remove the `number-renew` CronJob manifest; retire `bin-trigger-sender` from the
  CronJob path (binary and CI stay until nothing references them).

**Non-goals (Phase 1):**

- Migrating the in-process tickers (billing/ai/customer) — Phase 2 ticket. The
  customer-manager tickers (unverified cleanup, frozen expiry) discovered during
  exploration are added to the Phase 2 inventory (§9 item 3).
- Campaign execution / queue timeouts — blocked on VOIP-1282 findings. This design only
  needs to not preclude them; since a schedule is a DB row targeting any RPC queue/URI,
  they are additive.
- Customer-facing scheduling API (Phase 3). Precluded-by-design check: schedule rows
  carry `customer_id` from day one; platform jobs use the nil UUID customer.
- Backfill semantics for missed runs (we fire at most once on catch-up, see §5.4).
- route-manager SIP OPTIONS health check: stays in-process permanently (per-instance
  semantics — each replica must probe its own connectivity; it is not a singleton job,
  so a central scheduler is the wrong tool. Recorded here as the Phase 2 evaluation
  outcome requested by the ticket).

## 3. Service shape

Standard Class A manager, cloned from the `bin-tag-manager`/`bin-number-manager`
conventions:

```
bin-schedule-manager/
  CLAUDE.md  README.md  LICENSE  Dockerfile  .gitignore  go.mod  go.sum
  cmd/schedule-manager/main.go        # daemon (cobra; Bootstrap/LoadGlobalConfig style)
  cmd/schedule-control/main.go        # admin CLI (direct DB/cache, no RabbitMQ)
  internal/config/main.go
  models/schedule/{schedule.go,cron.go,field.go,filters.go,event.go}
                                       # webhook.go deferred to Phase 3: Phase 1 schedules are all
                                       # nil-customer (PublishWebhook short-circuits), so a
                                       # WebhookMessage would be dead code — and the repo pre-commit
                                       # hook rightly demands RST struct docs for any webhook.go,
                                       # which Phase 1 deliberately has none of
  models/execution/{execution.go,field.go,filters.go}
  pkg/cachehandler/                    # redis/v9 + redsync (lock library precedent: bin-flow-manager)
  pkg/dbhandler/                       # squirrel; schedule.go, execution.go; sqlite tests
  pkg/schedulehandler/                 # CRUD + next-run computation + name uniqueness
  pkg/dispatchhandler/                 # tick loop, claim, dispatch, record, reap
  pkg/backuphandler/                   # mysqldump execution (see §7)
  pkg/listenhandler/                   # RPC API
  pkg/subscribehandler/                # customer-manager events (customer_deleted cleanup)
  scripts/database_scripts_test/{table_schedule_schedules.sql,table_schedule_executions.sql}
  docs/{architecture.md,domain.md,dependencies.md,operations.md}
  k8s/{namespace.yml,deployment.yml,service.yml,kustomization.yml}
```

`bin-common-handler` additions (sanctioned exception):

- `models/outline/queuename.go`: `QueueNameScheduleEvent/Request/Subscribe`, plus an
  exported `QueueNameRequestAll() []QueueName` (or `QueueName.ValidRequestQueue()`)
  helper enumerating the `*.request` constants — the scheduler's `target_queue`
  CRUD validation uses it, so the allowlist cannot silently drift from the constants
- `models/outline/servicename.go`: `ServiceNameScheduleManager = "schedule-manager"`
- `pkg/requesthandler/schedule_schedule.go` (+ `send_request.go` sender): typed client
  methods `ScheduleV1ScheduleCreate/Get/Gets/Update/Delete/Execute`,
  `ScheduleV1ExecutionGets` — consumers: VOIP-1280 doctor and api-manager (Phase 3).
  Zero in-repo callers at merge time is accepted (the doctor integration is the
  immediate next ticket); `schedule-control` is NOT a consumer — it uses direct
  DB/cache access like every other `*-control` CLI.

CI registration (part of this PR's deliverables, per the service-onboarding checklist):

- `.circleci/config.yml`: path-filter mapping line
  `bin-schedule-manager/.*  run-bin-schedule-manager true` (alphabetical position)
- `.circleci/config_work.yml`: `run-bin-schedule-manager` pipeline parameter, workflow
  block (`build-approval` → `-test` → `-build` → `-release`), and the three jobs
  following the `bin-tag-manager` pattern.

Doc-sync deliverables: `docs/reference/rabbitmq-queues-reference.md` (three new queues),
`docs/reference/service-inventory.md`, `docs/conventions/database.md` §7.0 prefix table
(new `schedule_` row), architecture dependency graph regeneration. No
`bin-api-manager/docsdev` RST changes: Phase 1 has no user-visible API surface, so the
root CLAUDE.md RST rule is consciously N/A.

No other `bin-common-handler` API changes: dispatch uses the existing exported
`SendRequest(ctx, queue, uri, method, timeout, delay, dataType, data)` as-is (the
`queue` parameter is `commonoutline.QueueName`, so the DB string is converted at the
call site). `sock.Request.RequestID` is not settable through this path (`sendRequest`
builds the wire struct without it; RequestID exists for api-manager's inbound HTTP
propagation); correlation between an execution row and downstream logs is achieved by
logging the execution UUID on the scheduler side at dispatch start/end. Making
RequestID caller-settable would be a shared-library change with 37-service blast
radius, deliberately avoided in Phase 1.

Deployment: `replicas: 2` from day one — the whole point is that this is safe.

## 4. Data model

Two tables, `bin-dbscheme-manager` migrations (raw SQL via `op.execute`, BINARY(16)
UUIDs, DATETIME(6) timestamps, nil-active soft delete — current canonical style).

### 4.1 `schedule_schedules`

| column | type | notes |
|---|---|---|
| id | BINARY(16) PK | |
| customer_id | BINARY(16) NOT NULL | nil UUID = platform job. Multi-tenant from day one (Phase 3) |
| name | VARCHAR(255) NOT NULL | stable handle for seeds/doctor. Uniqueness among **active** rows per customer, enforced in `schedulehandler` (create/update check over `tm_delete IS NULL`); no DB unique key — a unique index would either block name reuse after soft delete or (with `tm_delete` in the key) not constrain active rows at all, since MySQL permits multiple NULLs in unique indexes. Known race: two simultaneous creates of the same name can both pass the app-level check (harmless in Phase 1 — seeds only; a Phase 3 to-do before customer exposure) |
| detail | VARCHAR(1024) | human description |
| type | VARCHAR(32) NOT NULL | `rpc` (Phase 1: only value; enum lives in Go) |
| cron | VARCHAR(128) NOT NULL | 5-field cron expression, evaluated in UTC |
| target_queue | VARCHAR(255) NOT NULL | e.g. `bin-manager.number-manager.request` |
| target_uri | VARCHAR(1024) NOT NULL | e.g. `/v1/numbers/renew` |
| target_method | VARCHAR(16) NOT NULL | POST/GET/PUT/DELETE |
| target_data_type | VARCHAR(255) | `application/json` |
| target_data | JSON | request payload |
| timeout_ms | INT NOT NULL | RPC timeout for the dispatch |
| retry_max | INT NOT NULL DEFAULT 0 | in-run immediate retries on failure (see §5.5) |
| enabled | TINYINT(1) NOT NULL DEFAULT 1 | |
| tm_next_run | DATETIME(6) | NULL = compute on next scan |
| tm_last_run | DATETIME(6) | last successful claim time (fire time, not completion) |
| tm_create / tm_update / tm_delete | DATETIME(6) | tm_delete NULL = active |

Indexes: PK, `idx_schedule_schedules_customer_name (customer_id, name)` (non-unique,
lookup only), `idx_schedule_schedules_next_run (enabled, tm_next_run)`.

Notes:

- The target columns are exactly the `bin-trigger-sender` CLI surface
  (`-queue/-uri/-method/-data_type/-data/-timeout`) — the absorption is 1:1.
- `type` exists so Phase 3 can add e.g. `flow` execution without a schema change; Phase 1
  code rejects anything but `rpc`.
- Cron is UTC-only in Phase 1 (matches the K8s CronJob semantics being replaced). A
  `timezone` column is a compatible later addition (Phase 3 concern).

### 4.2 `schedule_executions` (audit trail / dead-letter record)

| column | type | notes |
|---|---|---|
| id | BINARY(16) PK | |
| customer_id | BINARY(16) NOT NULL | copied from schedule |
| schedule_id | BINARY(16) NOT NULL | |
| trigger_type | VARCHAR(16) NOT NULL | `cron` / `manual` (see §6 execute). Named `trigger_type`, not `trigger`: `TRIGGER` is a MySQL reserved word / SQLite keyword, and the shared data path (`GetDBFields`/`PrepareFields`/squirrel) emits identifiers unquoted |
| status | VARCHAR(32) NOT NULL | `running` / `success` / `failed` / `abandoned` |
| status_code | INT | RPC response status code (0 if transport error) |
| error | TEXT | error string on failure/abandonment |
| result | MEDIUMTEXT | RPC response body on success, truncated to 60,000 bytes (uniform mechanism: the dispatcher stores whatever the target returned — for `/v1/backups` that is `{"path","bytes"}`; number-renew returns the full renewed-number list, which can exceed `TEXT`'s 65,535-byte cap, hence MEDIUMTEXT + explicit bound). A truncated payload may be invalid JSON — the column is an audit aid, not a parseable contract |
| attempt_count | INT NOT NULL | 1 + retries actually used |
| duration_ms | INT | total wall time of the run |
| tm_scheduled | DATETIME(6) NOT NULL | the tm_next_run this row consumed (idempotency key part); claim wall time for manual runs |
| tm_deadline | DATETIME(6) NOT NULL | reap deadline, computed in Go at INSERT: `tm_start + timeout_ms×(retry_max+1) + 5000×retry_max + 60000` ms. Denormalized so the budget is pinned to the run, not to the mutable schedule row |
| tm_start / tm_end | DATETIME(6) | |
| tm_create / tm_update / tm_delete | DATETIME(6) | soft-delete column present per convention; unused by retention (see below) |

Indexes: PK, `idx_schedule_executions_schedule (schedule_id, tm_create)`,
`idx_schedule_executions_reap (status, tm_deadline)` (serves the per-tick reap),
`idx_schedule_executions_tm_create (tm_create)` (serves the age-based retention DELETE),
`uq_schedule_executions_schedule_scheduled (schedule_id, trigger_type, tm_scheduled)` —
the unique key is a hard DB-level backstop against double-fire (second claimant's INSERT
fails).

Retention: rows older than `SCHEDULE_EXECUTION_RETENTION_DAYS` (default 90) are
**hard-DELETEd** by the `execution-retention` housekeeping schedule (§6.1) — audit rows
have no afterlife requirement, and soft-deleting an audit trail would leave the
unbounded-growth problem in place. `tm_delete` exists solely for convention/tooling
uniformity (PrepareFields, filters) and is never set by the service.

## 5. Dispatch engine

### 5.1 Cron parsing

New dependency: `github.com/robfig/cron/v3` — **parser only**, not its runner. First
cron library in the monorepo (verified: no robfig/gronx/gocron anywhere). Justification:
cron-expression parsing is exactly the class of code that should not be hand-rolled
(DST/day-of-week/step edge cases); robfig/v3 is the de-facto standard, small, and
dependency-free. Usage:

```go
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
sched, err := cronParser.Parse(spec)        // validation at CRUD time
next := sched.Next(now.UTC())               // next-run computation; argument MUST be .UTC()
```

`Schedule.Next` honors the argument's location, so every call site passes `.UTC()`.
Clock assumption: `now` is the claiming replica's wall clock; deployment surfaces must
run NTP-synced nodes (standard for K8s/GCE; noted in operations.md). The CAS (§5.3)
makes clock skew safe against double-fire; skew only shifts a slot's fire time by the
skew amount.

### 5.2 Tick loop

Every replica runs the same loop (no standing leader election — locks are per-claim):

```
every SCHEDULE_TICK_INTERVAL_SEC (default 10s):
  reap()                                    # abandoned-execution sweep, §5.6
  schedules := db.ScheduleListDue(now)      # enabled=1, tm_delete IS NULL, tm_next_run <= now
  for s in schedules:
      go claimAndDispatch(s)                # bounded by semaphore (max concurrent dispatches)
```

Schedules with `tm_next_run IS NULL` (fresh create, or cron just changed) are picked up
by a companion query — **filtered to `enabled=1`** — and initialized race-safely with a
**distinct** CAS statement that sets only `tm_next_run` (expected value NULL;
`squirrel.Eq{col: nil}` renders `IS NULL`) — it deliberately does not touch
`tm_last_run`, so a schedule that has never fired never claims it has. Enabling a
schedule (PUT enabled=1) resets `tm_next_run` to NULL: the next tick computes it
forward from now, so a long-disabled schedule **never catch-up-fires at enable time** —
it waits for its next matching slot (the operator-friendly behavior for enabling
`database-backup` at 15:00 with a `0 2 * * *` cron).

### 5.3 Claim: Redis lock + DB CAS (belt and braces)

Double-fire safety does not rest on Redis alone:

1. **Redis lock (redsync, contention filter):**
   `mutex := locker.NewMutex("schedule:lock:"+scheduleID, redsync.WithExpiry(30*time.Second), redsync.WithTries(1))`.
   `TryLockContext` failure → another replica is on it → skip silently. The lock guards
   only the claim critical section (steps 2-4) and is **released immediately after the
   execution row is written — it is not held across the RPC** (a 5-30 min dispatch would
   outlive any sane expiry; correctness never depends on it). redsync is already an
   established dependency (bin-flow-manager, bin-email-manager, bin-storage-manager use
   blocking `Lock()`; the try-once variant is new but same library).
2. **Overlap guard (Forbid semantics, §5.4):** `SELECT` for an existing execution with
   `schedule_id = :id AND status = 'running'` — if found, skip this slot without
   advancing `tm_next_run` (the schedule fires as soon as a later tick finds the run
   finished; the slot is not lost, it is late — matching CronJob Forbid behavior). The
   abandoned-execution reaper (§5.6) guarantees a crashed run cannot wedge this check
   forever.
3. **DB CAS (correctness):** the winner performs the atomic claim:
   ```sql
   UPDATE schedule_schedules
      SET tm_next_run = :next, tm_last_run = :now, tm_update = :now
    WHERE id = :id AND tm_next_run = :expected_next_run
      AND enabled = 1 AND tm_delete IS NULL
   ```
   `RowsAffected == 0` → someone else claimed between our SELECT and UPDATE (possible
   under Redis failover), or the schedule was disabled/deleted since the scan → skip.
   `:next = cron.Next(now)`. The `enabled`/`tm_delete` re-assertion prevents a
   just-disabled schedule from firing one last time.
4. **Unique execution row (backstop):** INSERT `schedule_executions(schedule_id,
   trigger_type='cron', tm_scheduled=:expected_next_run, status='running',
   tm_deadline=...)` — the unique key makes a third-path double-fire physically
   unrecordable; a duplicate-key error aborts the dispatch.

Steps 3 and 4 run in **one DB transaction** (conventions "Transaction Pattern"): a crash
between them can then neither consume a slot without an audit row nor leave
`tm_last_run` asserting a run that never started.

Only after a successful claim does the RPC fire. This gives at-most-once per scheduled
slot. Crash-after-claim-before-dispatch loses that slot's work (accepted: at-most-once
is the right default for jobs like number-renew, which releases numbers on insufficient
balance — firing twice is worse than firing zero times and catching up next period); the
orphaned `running` row is cleaned by the reaper (§5.6). kill -9 recovery: the dead
replica held no standing role; the surviving replica's next tick claims the next slot
naturally, within one tick interval + schedule period — satisfying the acceptance
criterion.

### 5.4 Overlap policy and missed runs

- **Overlap = Forbid** (via §5.3 step 2): a schedule never runs concurrently with
  itself, from either the cron path or the manual path. There is no Allow/Replace
  option in Phase 1.
- **Catch-up:** if `now - tm_next_run` is large (replica outage spanning slots): fire
  **once**, then `tm_next_run = cron.Next(now)` — never backfill each missed slot.

### 5.5 Dispatch + retry

Dispatch is `requesthandler.SendRequest(ctx, queue, uri, method, timeout_ms, 0,
dataType, data)` — the existing exported generic sender that all typed RPC methods wrap
(gains circuit breaker + request metrics for free). Publisher identity is
`schedule-manager` (set by `NewRequestHandler(sockHandler, serviceName)`). The
execution UUID is logged at dispatch start and completion for correlation (see §3 for
why it is not injected as RequestID).

Failure (transport error or non-2xx) → immediate re-attempt up to `retry_max` times with
linear 5s backoff, all within the same execution row (`attempt_count`). Exhausted →
`status=failed`, error recorded, `dispatch_total{result="failed"}` incremented, event
published. The failed execution row **is** the dead-letter record; no separate queue.
Next scheduled slot proceeds normally regardless. Note: all Phase 1 seeds use
`retry_max=0`, so the retry path's only Phase 1 coverage is its unit tests — flagged
here so that is a conscious state, not an oversight.

`number-renew` seed uses `timeout_ms = 300000` (5 min): the handler paginates the full
number set with a per-number billing RPC; the old CronJob's 3s timeout was a latent
failure (the send-side would time out and "fail" the job while the server-side
completed).

### 5.6 Abandoned-execution reaper

A `running` execution whose replica died mid-dispatch would otherwise stay `running`
forever — misreporting the audit trail and, worse, wedging the §5.3 overlap guard
permanently. Every tick begins with a cheap reap:

```sql
UPDATE schedule_executions
   SET status='abandoned', error='replica died mid-dispatch', tm_end=:now, tm_update=:now
 WHERE status='running' AND tm_deadline < :now
```

A **single-table** conditional update — buildable with squirrel (mandatory per
conventions §7.1; `UpdateBuilder` has no JOIN) and portable to the sqlite test harness
(no JOIN, no INTERVAL arithmetic), which is what makes the §10 kill -9 acceptance test
executable. The worst-case budget (`timeout_ms × (retry_max+1)` + backoff + 60s margin)
was denormalized into `tm_deadline` at INSERT time (§4.2), which also pins the budget to
the run — editing a schedule's `timeout_ms` mid-flight does not retroactively move a
live run's deadline. Served by `idx_schedule_executions_reap (status, tm_deadline)`.
The reap is idempotent and safe to run on every replica concurrently. `abandoned` counts
as a terminal failure for metrics/doctor purposes.

Reap-vs-completion race: if the 60s margin is ever exceeded by a still-in-flight run,
the reaper marks it `abandoned` while the dispatcher later tries to write
`success`/`failed`. The completion UPDATE is therefore **conditional on
`status='running'`** — an abandoned row stays abandoned (one terminal state, no
double-count in `dispatch_total`, no premature Forbid release followed by a rewrite).

### 5.7 Metrics (namespace `schedule_manager`)

- `dispatch_total{schedule_name, result="success|failed|abandoned|skipped_lock|skipped_cas|skipped_overlap"}` (counter; `skipped_overlap` is incremented once per skipped *slot*, not per tick — the tick loop suppresses repeats while the same run is in flight)
- `dispatch_duration_ms{schedule_name}` (histogram)
- `schedule_last_success_timestamp{schedule_name}` (gauge — the doctor/alerting hook)
- `schedule_lag_seconds{schedule_name}` (gauge: now − tm_next_run when overdue)
- `execution_rows` (gauge, total row count, refreshed once per tick loop — never
  computed per Prometheus scrape, so scrape frequency cannot drive `COUNT(*)` load —
  alarms if `execution-retention` stops working, closing the silent-failure loop on the
  housekeeping schedule itself)
- `backup_last_bytes` (gauge — size of the most recent successful dump; doctor flags 0)
- standard listenhandler `receive_request_process_time`

Label-cardinality decision: `schedule_name` is acceptable while all schedules are
platform-owned (Phase 1/2, single-digit count, nil-customer namespace). Phase 3
customer schedules will NOT be per-schedule labeled (unbounded cardinality, names only
unique per customer) — they get aggregate metrics keyed by `customer-facing` result
only. Recorded now because VOIP-1280 alerting builds on these labels.

Per-replica gauge semantics: each replica reports its own gauge values, and the replica
that did not dispatch reports stale/absent values. Alert rules MUST aggregate with
`max by (schedule_name)` — recorded in operations.md so the first VOIP-1280 alert rule
is not written against `min`/single-series.

## 6. RPC API (listenhandler)

| Route | Method | Purpose |
|---|---|---|
| `/v1/schedules` | POST | create (validates: cron parses AND `Next()` is non-zero — a syntactically valid but never-matching spec like `0 0 30 2 *` is rejected 400; method whitelist; `target_queue` must match a `models/outline` queue constant — turning a typo into an immediate 400 instead of a blocked dispatch slot; active-name uniqueness; type=rpc only) |
| `/v1/schedules?page_size&page_token&filters` | GET | list (filters: customer_id, enabled, deleted) |
| `/v1/schedules/<id>` | GET / PUT / DELETE | get / update (cron change ⇒ tm_next_run=NULL) / soft-delete |
| `/v1/schedules/<id>/execute` | POST | manual fire-now (see below) |
| `/v1/executions?page_size&page_token&filters` | GET | audit trail (filters: schedule_id, status) |
| `/v1/executions/prune` | POST | retention pruning (internal; invoked by the `execution-retention` schedule). Batched `DELETE WHERE id IN (SELECT id FROM ... WHERE tm_create < :cutoff LIMIT 1000)` in a loop — the subquery form (not `DELETE ... LIMIT`) is used because mattn/go-sqlite3 lacks `SQLITE_ENABLE_UPDATE_DELETE_LIMIT`, keeping the statement testable in the sqlite harness. Served by `idx_schedule_executions_tm_create` |
| `/v1/backups` | POST | run DB backup now (internal; invoked by the `database-backup` schedule, see §7) |

**Manual execute semantics (`/v1/schedules/<id>/execute`):** a manual run **never
touches `tm_next_run` or `tm_last_run`** — test-firing `number-renew` at 14:00 must not
consume tonight's 01:00 slot. Concurrency protection: the same Redis lock
(`schedule:lock:<id>`) and the same §5.3 step 2 overlap guard apply (a manual run is
refused with 409 while any run of that schedule is in flight, and vice versa the cron
path skips while a manual run is in flight); the execution row is inserted with
`trigger_type='manual'`, `tm_scheduled=now` — the
`(schedule_id, trigger_type, tm_scheduled)` unique key dedupes concurrent manual fires
at microsecond granularity as a backstop behind the lock+guard. Works on disabled
schedules (that is the point of a manual test-fire).

DTOs flat per rpc.md §9.5 (`V1DataSchedulesPost` etc.). Errors: `cerrors.VoipbinError`
envelope per listenhandler-error-mapping.md. CRUD and execution outcomes publish
**internal** events via notifyhandler `PublishEvent`
(`schedule_created/updated/deleted`, `execution_succeeded/failed` — fleet event-name
convention, publisher-scoped without a service prefix).
Customer-visible webhook events (WebhookMessage + RST struct docs) arrive with Phase 3,
when schedules gain real customer owners — in Phase 1 every schedule is nil-customer
and `PublishWebhook` short-circuits on the nil customer anyway.

### 6.1 Internal housekeeping schedules

Execution-retention pruning runs as a real seeded schedule (`execution-retention`,
daily, target = own queue `/v1/executions/prune`) so housekeeping goes through the same
claim machinery instead of a second bespoke ticker. This also dogfoods the engine.
Failure signal if it is ever disabled/deleted: the `execution_rows` gauge (§5.7) grows
without bound and doctor (VOIP-1280) checks the schedule's existence + last success.

**Self-RPC worker budget:** `/v1/backups` and `/v1/executions/prune` arrive on the
service's own request queue and occupy one of the listenhandler's 10 consumer workers
for their duration (up to 30 min for a large backup). With at most two internal
schedules and Forbid overlap, worst case is 2 of 10 workers busy; acceptable, and
recorded in operations.md. If Phase 3 multiplies long-running self-RPCs, move them to an
async accept-then-work pattern then. Load-bearing fact making this safe: the shared RPC
consume path **acks before processing** (`rabbitmqhandler/consume.go`), so a 30-minute
handler cannot trip the broker's `consumer_timeout` or trigger redelivery; the sender's
context timeout is the only clock.

Also note: the DB-driven `target_queue` string is a sanctioned, validated exception to
**two** conventions — `bin-common-handler/CLAUDE.md`'s "no free-form queue strings" rule
and `docs/conventions/rpc.md` §9.1's "use typed request methods, never construct raw
RPC requests" (a generic dispatcher is definitionally untyped). Both exceptions are
recorded in the service CLAUDE.md; CRUD-time validation against the `models/outline`
enumeration helper (see §3, route table) keeps the invariant equivalent.

## 7. Scheduled DB backup (self-hosted)

Backup is modeled as a normal `rpc` schedule targeting **schedule-manager's own queue**,
`POST /v1/backups` — keeping dispatch uniform (one engine, one audit trail) instead of a
special `builtin` schedule type.

`pkg/backuphandler` executes `mysqldump --single-transaction --routines --triggers
--set-gtid-purged=OFF` as a subprocess against `DATABASE_DSN`'s host, gzips to
`SCHEDULE_BACKUP_DIR/voipbin-<UTC timestamp>.sql.gz`, then prunes to
`SCHEDULE_BACKUP_RETENTION_COUNT` (default 7) newest files. Host/offsite copy stays an
operator concern (ticket scope). Failure surfaces exactly like any job failure
(execution row + metrics + event).

Consequences, called out explicitly:

- **Client/server pairing:** the platform DB is MySQL 8.0 (see
  `bin-dbscheme-manager/Dockerfile` — `FROM mysql:8.0`; MariaDB appears only in its
  builder stage, and that Dockerfile already has to `sed` collation names between the
  two engines — evidence they are not interchangeable). The backup client is therefore
  **genuine `mysqldump` 8.0 from the official MySQL APT repository**
  (`mysql-community-client`), NOT `mariadb-client` (alpine's `mysql-client` is a
  MariaDB alias; MariaDB tooling against MySQL 8.0 is an unsupported pairing —
  `caching_sha2_password` auth, incompatible dump output).
- **Image deviation:** the fleet standard is distroless static; a subprocess needs a
  binary-carrying base. Runtime stage: `debian:bookworm-slim` + `mysql-community-client`
  (pinned 8.0.x) from MySQL's official APT repo. Documented in the service CLAUDE.md as
  a justified deviation (same status as bin-call-manager's raw-SQL exception). A
  Go-native dumper was rejected: correctness risk for zero operational gain. Client pin
  note: MySQL 8.0 passed EOL in 2026-04; the client pin follows whatever the platform
  DB server runs — when the platform upgrades to 8.4/9.x, bumping this package is part
  of that migration's checklist.
- **Credential hygiene:** the DB password is passed via a generated
  `--defaults-extra-file` (0600 temp file, removed after the run) — never argv
  (world-readable via `/proc/<pid>/cmdline`) and not `MYSQL_PWD` (documented by Oracle
  as insecure: inherited by children, readable via `/proc/<pid>/environ`).
  Host/port/user/dbname go as flags. Error paths log a **redacted** command line and
  never echo the DSN. The backuphandler unit tests assert argv and the logged command
  contain no password material.
- **Verification:** acceptance includes a dump-and-restore smoke test — restore the
  produced dump into a scratch MySQL 8.0 and diff table counts (scripted in the
  service's docs/operations.md; automated as part of VOIP-1281's sandbox onboarding
  where a real MySQL is available; CI covers command construction + retention logic
  with the exec boundary mocked).
- **Storage under `replicas: 2` (requirement, not deferred):** the backup job arrives on
  the shared request queue, so **any replica may execute it**. Therefore
  `SCHEDULE_BACKUP_DIR` MUST be one volume **shared by all scheduler replicas**
  (ReadWriteMany in K8s; a shared named volume/bind mount in compose), and retention
  pruning operates on that shared set. An RWO PVC is explicitly a misconfiguration: the
  second replica cannot mount it and sits Pending, silently breaking the `replicas: 2`
  failover premise. Alternative for surfaces without RWX storage: a dedicated
  single-replica writer arrangement, at the operator's choice. The *wiring* (which PVC
  class, compose volume name) is VOIP-1281's job; the requirement is this design's.
  **`SCHEDULE_BACKUP_DIR` has no default and must be explicitly configured** — a
  default like `/var/backups/voipbin` would be writable container-local storage, so a
  surface that forgot to mount the volume would "succeed" into ephemeral disk (silent
  backup loss). Unset dir → the backup job fails loudly with an actionable error;
  nothing else breaks. Each successful run records `{"path": ..., "bytes": ...}` in the
  execution row's `result` column (§4.2) and exports `backup_last_bytes` (gauge), so a
  doctor check can flag a zero-byte or missing dump.
- **Seeded disabled** (`enabled=0`, cron `0 2 * * *`): production GCP uses managed Cloud
  SQL backups; enabling in self-hosted stacks is a VOIP-1281 onboarding step (one API
  call or schedule-control command, no redeploy).

## 8. Seeds (migration)

One Alembic migration creates both tables; a second seeds **three** rows (INSERT with
`UNHEX(REPLACE(UUID(),'-',''))` ids, nil-UUID customer_id):

| name | cron (UTC) | target | data | timeout | enabled |
|---|---|---|---|---|---|
| `number-renew` | `0 1 * * *` | number-manager `/v1/numbers/renew` POST | `{"days":28}` | 300000 | 1 |
| `database-backup` | `0 2 * * *` | schedule-manager `/v1/backups` POST | `{}` | 1800000 | 0 |
| `execution-retention` | `30 2 * * *` | schedule-manager `/v1/executions/prune` POST | `{}` | 600000 | 1 |

Plus matching sqlite DDL in `scripts/database_scripts_test/`.

Note: `bin-dbscheme-manager`'s image bakes migration output into a schema dump at build
time, so the seeded `UUID()` values are fixed per image build and identical across
self-hosted installs — harmless (they are handles, not secrets), but not
environment-unique. The `name` column, not the id, is the stable cross-environment
handle.

## 9. Cutover / removal

1. `bin-number-manager/k8s/cronjob.yml` — **deleted**; the commented `# - cronjob.yml`
   line removed from its kustomization.yml. (It was already not applied; risk ≈ 0.)
2. `bin-trigger-sender` — untouched in this PR (binary + CI remain). Its retirement
   (delete directory + CircleCI entries) is a trivial follow-up once install/sandbox
   confirm nothing invokes it (tracked in VOIP-1281's cutover checklist).
3. In-process tickers — untouched (Phase 2 ticket, to be filed at Phase 1 completion,
   inventory: billing×2, ai×1 + startup sweeps, customer×2).

## 10. Testing strategy

- **Unit (gomock, table-driven, conventions per testing.md):** schedulehandler CRUD +
  cron validation + active-name uniqueness; dispatchhandler claim logic (CAS zero-rows
  path, lock-contention path, overlap-guard skip, retry/backoff, catch-up single-fire,
  manual-execute non-consumption of tm_next_run); backuphandler (command construction
  incl. no-password-in-argv assertion, retention pruning — subprocess exec behind an
  interface, mocked); reaper (budget arithmetic, idempotence).
- **Double-fire proof (acceptance criterion):** dbhandler-level test against sqlite:
  two dispatchhandler instances sharing one DB, mocked cachehandler where both replicas'
  `TryLock` succeeds (simulated Redis split-brain), N concurrent ticks → assert exactly
  one CAS winner and exactly one execution row (unique key holds). Caveat recorded in
  the test file: sqlite serializes writers, so this proves the SQL-level invariant, not
  MySQL REPEATABLE READ interleavings — the CAS is a single conditional UPDATE, whose
  row-lock semantics under MySQL make the same guarantee, but that step is argued, not
  executed, in CI.
- **kill -9 mid-dispatch (acceptance criterion):** test that (a) instance A claims and
  inserts a `running` execution, then vanishes (no completion write); (b) instance B's
  overlap guard skips while the row is within budget; (c) after the budget elapses,
  B's reap marks it `abandoned` and the next due slot claims and fires normally. This
  covers failover end-to-end at the state-machine level.
- **dbhandler tests:** standard in-memory sqlite with SQL fixtures loaded via the `smotes/purse` library (repo convention).
- **listenhandler tests:** standard request-routing tests with mocked handlers.
- **requesthandler client tests (bin-common-handler):** per rpc.md §9.6, the new
  `ScheduleV1*` methods get client-side tests asserting the marshaled wire shape
  (queue, uri, method, body) byte-for-byte against a mocked sockhandler.

## 11. Config

| Env | Default | |
|---|---|---|
| DATABASE_DSN / RABBITMQ_ADDRESS / REDIS_ADDRESS / REDIS_PASSWORD / REDIS_DATABASE / PROMETHEUS_* | standard | |
| SCHEDULE_TICK_INTERVAL_SEC | 10 | scan cadence |
| SCHEDULE_DISPATCH_CONCURRENCY | 10 | max in-flight jobs per replica |
| SCHEDULE_EXECUTION_RETENTION_DAYS | 90 | audit pruning |
| SCHEDULE_BACKUP_DIR | (no default — must be explicitly set; see §7) | |
| SCHEDULE_BACKUP_RETENTION_COUNT | 7 | |

## 12. Rollout / risk

- The service is additive; nothing depends on it at merge time. The only behavioral
  change to an existing surface is deleting an already-inactive CronJob manifest.
- If the dispatch loop misbehaves, `schedule-control schedule disable <name>` (direct
  DB) or scaling to 0 stops all firing; no other service degrades.
- number-renew semantics change slightly vs the CronJob: send-side timeout 3s → 5min
  (fixes a latent failure), fire-and-record instead of Job backoffLimit retries
  (retry_max=0 for this destructive job: a failed run waits for tomorrow's slot, same as
  Forbid + failed-Job behavior).
- New deps: robfig/cron/v3 (parser), redsync (already in-repo), debian-slim +
  mysql-community-client runtime image (scheduler only).

## 13. Open items deliberately deferred

- VOIP-1282 outcome → campaign/queue schedules become rows later; no engine change.
- Phase 2 ticker migration ticket (inventory in §9 item 3).
- Phase 3 multi-tenant API exposure via api-manager/openapi (schema already carries
  customer_id; name uniqueness already scoped per customer).
- Timezone-aware cron, per-schedule jitter, overlap policies beyond Forbid — YAGNI until
  a schedule needs them.
