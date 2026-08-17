# Design: roll out Komodo-managed deploy to 16 Tier 1 bin-*-manager services (VOIP-1347)

## Goal

Extend the Komodo-managed deploy pattern proven by VOIP-1342 (bin-call-manager,
merged in PR #1188) to 16 additional services:

agent, billing, campaign, conference, conversation, direct, flow, outdial,
queue, route, tag, talk, transfer, webhook, contact, webchat
(`bin-<short>-manager` for each).

This is an **application of an already-verified pattern**, not new discovery
work — see [2026-08-16-komodo-call-manager-cutover-design.md](2026-08-16-komodo-call-manager-cutover-design.md)
for the original spike, empirical verification, and rationale. This doc only
covers what differs for Tier 1: the things that are identical are stated
once and not re-derived.

## What's identical to bin-call-manager

- **Compose file shape:** `<service>/komodo/docker-compose.yml`, one service
  block, `image: voipbin/bin-<short>-manager:__IMAGE_TAG__`, `json-file`
  logging (10m/3 files), `container_name: voipbin-<short>-manager`,
  `restart: always`, `command: ./<short>-manager`, attached to the shared
  `production` network (`external: true`).
- **No healthcheck:** all 16 services build on
  `gcr.io/distroless/static-debian12` (confirmed per-service Dockerfile) —
  same as bin-call-manager, no shell/wget available for a Docker healthcheck.
- **Binary name convention:** `./<short>-manager` confirmed per service via
  each `Dockerfile`'s `go build -o /app/bin/ ./cmd/...` and each `cmd/`
  directory listing — all 16 follow the same `<short>-manager` binary name
  as `call-manager` does. No exceptions found.
- **Env block:** `DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`,
  `REDIS_DATABASE=1`, `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS` —
  byte-identical across all 16 in `voipbin/install/docker-compose.yml.dist`.
  `REDIS_PASSWORD` is absent for all 16, same as bin-call-manager — omitted
  entirely rather than left commented out.
  - **Tier 1's env list is simpler than bin-call-manager's.** bin-call-manager
    additionally sets `PROJECT_BASE_DOMAIN`, `PROJECT_BUCKET_NAME`, and the
    three `HOMER_*` vars (media storage + SIP capture integration specific to
    call-manager). None of the 16 Tier 1 services have those envs in
    `install/docker-compose.yml.dist`, so they are correctly omitted here —
    not an oversight.
- **CI mechanics reused as-is, unmodified:** `.circleci/scripts/render-image-tag.sh`
  and `.circleci/scripts/komodo-api-deploy.sh` are already generic (operate
  on a `komodo/docker-compose.yml` path and a service name passed as
  arguments) and already exercised in production by bin-call-manager. The
  16 new `<service>-deploy` CI jobs call them exactly as
  `bin-call-manager-deploy` does, with the same
  `CC_KOMODO_POLL_ATTEMPTS=60 CC_KOMODO_POLL_INTERVAL_S=3` budget.
- **These 16 services had no deploy job at all before this PR** (only
  `-test`/`-build`) — the old SSH `ssh-deploy.sh` wiring was removed for all
  non-call-manager services in a prior cleanup
  (`817c24db6 NOJIRA-Remove-ssh-deploy-bm-nyc-01 (#1189)`, immediately after
  the bin-call-manager cutover). This PR adds new `-deploy` jobs; it does
  not replace or conflict with anything currently running.
- **Komodo Variables (`[[BIN_MANAGER__*]]`):** resolved by Komodo itself from
  its global Variables store, already synced from infra-secret. Nothing in
  this PR or CI supplies these values — same as bin-call-manager.

## Decision: start on `production`, skip the `install_default` detour

bin-call-manager's Stack originally targeted `install_default` and was
corrected to the shared `production` network later (VOIP-1343, filed
alongside VOIP-1342) once that Komodo-managed backbone network existed. All
16 Tier 1 compose files here are written directly against `production` from
the start — there is no interim `install_default` step to migrate away from.
Precondition (same as bin-call-manager, already satisfied on bm-nyc-01):
db/redis/rabbitmq must be reachable from `production`, which
`monorepo-etc/infra-komodo/komodo/backfill-install-default.sh --apply` has
already bridged.

## Cutover procedure (standard, 15 of 16 services)

Per service, after this PR merges and its `-deploy` job runs for the first
time:

1. **Deploy via CI** — the new `<service>-deploy` job pushes the compose
   file to Komodo and triggers a deploy. Because the new container's name
   (`voipbin-<short>-manager`) differs from the old install-managed
   container's name, Docker does not refuse to run both at once. The new
   container joins as an **additional** RabbitMQ competing consumer
   alongside the old one — confirmed zero-gap-safe by the bin-call-manager
   pilot (VOIP-1342): RabbitMQ delivers each message to exactly one
   competing consumer, so having two consumers briefly overlap causes no
   duplicate processing or dropped messages, only doubled idle
   polling/heartbeat overhead for the overlap window.

   **Old container name, per service — check before assuming
   `voipbin-<short>-mgr`:** 14 of the 16 services have a pinned
   `container_name: voipbin-<short>-mgr` in
   `voipbin/install/docker-compose.yml.dist` (e.g.
   `voipbin-route-mgr`, `voipbin-webhook-mgr`) — for those, step 3 below is
   `docker rm voipbin-<short>-mgr` as expected. **`bin-contact-manager` and
   `bin-webchat-manager` have no `container_name:` pinned** in that file, so
   Compose auto-names their old containers (`install-<service>-1`-style,
   scoped to the install project's working directory). For those two, do
   not guess the name — run `docker compose -f
   /opt/voipbin/install/docker-compose.yml ps <short>-manager` (or `docker
   ps | grep <short>`) on bm-nyc-01 first to find the actual old container
   before removing it.
2. **Verify at the application log level, not just container running-state.**
   VOIP-1342's false-positive lesson: the CI job's 3-consecutive-running
   gate only checks Docker process state, not application health — a
   container can be "running" while its app is crash-looping on a bad DB
   connection (e.g. before the `production` network backfill was applied).
   Before removing the old container, `docker logs` the **new** container
   and confirm:
   - a connection-success line for RabbitMQ/DB/Redis, and
   - the absence of `CRITICAL`/`Error`-level lines from the boot path.
3. **Remove the old container** once the new one is confirmed healthy at the
   log level.
4. **Comment out the removed service's block** in bm-nyc-01's live
   `/opt/voipbin/install/docker-compose.yml` (SSH), matching VOIP-1342's
   step 6 — prevents the install-managed compose project from recreating the
   old container on its next `docker compose up`.
5. **Confirm a single RabbitMQ consumer** remains for the service's queues
   (via the RabbitMQ management UI/API) — the overlap window is closed.

For 15 of the 16 services, there is no urgency forcing step 3 to happen
quickly — "leave both running, verify at leisure, then cut over" is safe
because none of them have a background loop that would compound the
overlap's cost beyond doubled idle consumer overhead.

## Deviation: bin-route-manager needs a short overlap window

`bin-route-manager` runs a background health-check ticker
(`pkg/healthcheckhandler/health.go`, `Run()`, started from
`cmd/route-manager/main.go`) that sends outbound SIP OPTIONS probes to every
configured carrier on an interval and writes `ProviderUpdateHealthStatus` to
the DB. During old+new container overlap, **both** containers run this
ticker independently, which:

- **Doubles outbound SIP OPTIONS probe traffic** to every configured
  carrier for the duration of the overlap — carriers may rate-limit or flag
  this, unlike the other 15 services' pure idle-consumer overhead.
- **Produces concurrent, non-corrupting but wasteful DB writes**
  (`ProviderUpdateHealthStatus` from both containers, last-write-wins on the
  same rows — no data corruption, just redundant writes).

This was verified to be unique to route-manager: a grep for
`time.NewTicker`/`time.Tick`/`cron\.` across all 16 Tier 1 services' source
trees found this loop only in bin-route-manager.

**Deviation from the standard flow:** for bin-route-manager specifically,
after confirming the new container is healthy at the application log level
(cutover step 2 above), remove the old container **promptly** — do not leave
both running for an extended soak the way the other 15 services can. The
verification step itself (log-level health check) is unchanged; only the
old-container removal timing is tightened for this one service.

## Doc gap: bin-webchat-manager

`bin-webchat-manager` has no `docs/operations.md` (only `docs/plans/`), so
its `## Deployment` section was added to `bin-webchat-manager/README.md`
instead. It is missing the standard architecture/operations/domain/dependencies
doc suite that `docs/reference/extractor.sh` produces for other services —
[VOIP-1352](https://voipbin.atlassian.net/browse/VOIP-1352) tracks
generating that suite; it is out of scope for this PR.

## Testing

No Go source changes — this PR only touches `komodo/docker-compose.yml`
files (new), `.circleci/config_work.yml` (CI wiring), and doc files. The
5-step Go verification workflow (`go mod tidy && go mod vendor && go
generate && go test && golangci-lint`) is a no-op here and was not run
against all 16 services' source trees. What was verified instead:

- `.circleci/config_work.yml` parses as valid YAML
  (`python3 -c "import yaml; yaml.safe_load(open(...))"`).
- Each of the 16 new `-deploy` jobs is present under `jobs:` and wired into
  its service's workflow block with `requires: [<service>-build]`, matching
  `bin-call-manager-deploy`'s pattern.
- `git status` confirms only `komodo/` (16 new compose files), 15
  services' `docs/operations.md` (webchat has none — see the Doc gap
  section above), all 16 `README.md` files, two small stale-count fixes in
  `bin-call-manager`'s own `docs/operations.md` and
  `komodo/docker-compose.yml` (its "other 31 services" framing was
  falsified by this PR the moment it merges — see below), this design doc,
  and `.circleci/config_work.yml` changed — no `.go` files touched.

## Explicitly out of scope

- Running any of the 16 new `-deploy` jobs, or performing any cutover step,
  is **not** part of this PR. This PR ends at an open, reviewable PR — the
  actual deploy/cutover for each service is a separate follow-up step after
  human review, same as VOIP-1342.
- Generating bin-webchat-manager's missing doc suite (tracked in
  [VOIP-1352](https://voipbin.atlassian.net/browse/VOIP-1352)).
