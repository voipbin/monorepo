# Design: cut bin-call-manager over to Komodo-managed deploy (VOIP-1342)

Revision 7 — incorporates round-1 (3 CRITICAL, 4 HIGH, 5 MEDIUM), round-2
(3 HIGH, 6 MEDIUM), round-3 (1 CRITICAL, 1 HIGH, 3 MEDIUM), and round-4
(1 CRITICAL + reference cleanups) architect review findings, plus two
post-approval revisions from 대표님 (naming, and idempotent Stack
auto-create — both below). See "Review corrections log" at the end for
the full architect-review mapping.

**Naming decision (2026-08-16, post-merge-review, 대표님):** the new
container's name is `voipbin-call-manager` — matching GKE's `call-manager`
app name/label (verified in `bin-call-manager/k8s/deployment.yml`) rather
than `install/`'s host-wide `voipbin-<service>-mgr` abbreviation. This is
**deliberately different from the old container's name**,
`voipbin-call-mgr` (unchanged, still the live pre-cutover container's real
name throughout this document). The other 31 `bin-*-manager` services'
eventual Komodo cutover will follow the same `voipbin-<service>-manager`
convention, not `-mgr`, as a follow-up. **Consequence, not a footnote:**
because the two names differ, Docker no longer refuses to run both
containers at once the way it would if they shared a name — the
mutual-exclusion safety net earlier revisions of this design leaned on is
gone. §7's ordering (old removed before new deployed) still holds, but is
now enforced only by *procedure*, not by Docker itself — §7 adds an
explicit verification step for this.

## Goal

Replace `ssh-deploy.sh` with a Komodo-API-based deploy path for
bin-call-manager on bm-nyc-01 — the first production service to move off
`voipbin/voipbin`'s `install/` (`versions.lock`/dist-live compose)
mechanism entirely. `install/` stays as-is for external self-hosting users;
this change does not touch it except to comment out one service block (see
"Cutover sequencing"). Per 대표님's explicit decision, `versions.lock` and
digest resolution are not carried forward — image references use a plain
tag (`$CIRCLE_SHA1`), and CircleCI talks to Komodo directly over its public
HTTPS endpoint for routine deploys, not an SSH tunnel.

Scope: **bin-call-manager only**. The other 31 `bin-*-manager` services
stay on `ssh-deploy.sh` until this pilot is verified in production and a
follow-up ticket rolls the pattern out.

## Current state (verified)

- bin-call-manager's CircleCI job (`monorepo/.circleci/config_work.yml`)
  builds+pushes `voipbin/bin-call-manager:$CIRCLE_SHA1`, then runs
  `.circleci/scripts/ssh-deploy.sh voipbin/bin-call-manager call-manager`,
  which SSHes to bm-nyc-01, bumps a digest pin in the live
  `versions.lock`/`docker-compose.yml` under `/opt/voipbin/install`, and
  runs `docker compose pull/up`.
- The container runs as `voipbin-call-mgr`, currently attached to a
  network commonly referred to as `install_default`. **Not asserted as
  fact here** — Compose derives the network name from the active project
  name, which can come from the directory name, an explicit
  `COMPOSE_PROJECT_NAME`, or `install/.env` (`infra-komodo/README.md`
  notes the main stack's own README recommends exporting
  `COMPOSE_PROJECT_NAME=sandbox` in some contexts). §"Pre-cutover
  verification" below makes confirming the real name on the live host a
  mandatory first step, not an assumption.
- No volumes are mounted on `voipbin-call-mgr` — safe to remove/recreate
  with no data-loss concern.
- **`bin-call-manager`'s image is distroless** (`gcr.io/distroless/static-debian12`,
  confirmed via `bin-call-manager/Dockerfile`) — no shell, no `wget`. The
  live `docker-compose.yml.dist` healthcheck (`wget`-based) has never been
  able to pass on this image; it reports `unhealthy` permanently today.
  **This design does not copy that healthcheck into the new file** — see
  §1.
- `komodo-api-deploy.sh` (VOIP-1341, merged, PR #1187) exists today and
  talks to Komodo over an SSH tunnel (`ssh` to bm-nyc-01, then
  `curl 127.0.0.1:9120` inside that session). **This design keeps it,
  unmodified, as the break-glass fallback** (§2) rather than deleting it —
  see "No break-glass path" finding below for why.
- **Secrets are already in Komodo.** `NOJIRA-Retire-infra-vault-for-komodo-secrets`
  (PR #90, merged) synced `infra-secret/secrets-source/bin-manager/**` into
  Komodo Variables named `BIN_MANAGER__<KEY>`. Confirmed present:
  `BIN_MANAGER__DATABASE_DSN_BIN`, `BIN_MANAGER__RABBITMQ_ADDRESS`,
  `BIN_MANAGER__REDIS_ADDRESS`, `BIN_MANAGER__REDIS_PASSWORD`,
  `BIN_MANAGER__HOMER_API_ADDRESS`, `BIN_MANAGER__HOMER_AUTH_TOKEN`,
  `BIN_MANAGER__HOMER_WHITELIST`, `BIN_MANAGER__PROJECT_BASE_DOMAIN`,
  `BIN_MANAGER__GCP_BUCKET_NAME_MEDIA`.
- **Correction (round 1):** whether Komodo actually interpolates
  `[[NAMESPACE__KEY]]` references into a Stack's `file_contents` at deploy
  time is **NOT yet empirically verified**. PR #90's own spike
  (`monorepo-etc/docs/.../2026-08-15-retire-infra-vault-komodo-secrets-design.md`
  §4.3) verified only the Variables CRUD API and explicitly deferred
  deploy-time interpolation behavior ("Implementers should not assume
  either answer" for what happens with an unresolved `[[VAR]]`). This
  design's secrets story depends on interpolation actually working — §"Step
  0" below makes verifying it, empirically, against a disposable Stack, a
  hard precondition before the real cutover, not an assumption carried
  into production.
- **Correction (round 1) — citation fix, not a retraction:** "`UpdateStack`
  is PATCH-style, only replaces `file_contents`" *was* empirically verified
  — via a disposable test Stack (`claude-env-verify-test`) — during the
  now-discarded `VOIP-1341-Wire-komodo-deploy-bin-call-manager` draft's
  design work on 2026-08-14, not in PR #90's spike (round 1 review
  correctly flagged the original wrong citation). That empirical result
  itself is still valid; only the surrounding document was discarded per
  대표님's "design from scratch" instruction. Re-confirm it holds as part of
  Step 0, since it's cheap to re-check while already standing up a
  disposable Stack for the interpolation test.
- **CI credentials already exist and are broad.** `CC_KOMODO_API_KEY`/`CC_KOMODO_API_SECRET`
  are already registered in the `production` CircleCI context, and the
  `circleci` Komodo Service User is already promoted to **Admin** (per PR
  #90's spike). Round 1 initially had CI *not* auto-create Stacks because
  of this — **superseded post-approval** (대표님, 2026-08-16): CI now does
  create the Stack, idempotently, when missing — see §2 item 1.
- `RABBITMQ_ADDRESS`/`DATABASE_DSN_BIN` in the `bin-manager` namespace are
  already full connection strings (`amqp://user:pass@host:port`,
  `user:pass@tcp(host:port)/db`), not decomposed into separate
  user/password/host fields — the compose fragment references the whole
  string as one variable, not a reassembly (unlike the discarded draft's
  `DATABASE_DSN=root:${MYSQL_ROOT_PASSWORD}@tcp(...)` construction, which
  assumed a different, incorrect key shape for this namespace).
- **`https://komodo.voipbin.net` has two independent, routine failure
  modes**, both already documented from operating the same endpoint for
  secrets sync (`infra-secret-sync-komodo.sh`):
  1. The Caddy route is deleted by any `setup-host.sh` rerun on bm-nyc-01
     (normal host-fix operation, not exceptional) — **symptom: HTTP 404**.
  2. `docker network connect komodo_default voipbin-web-proxy` is a
     non-persistent, manual attachment, undone whenever `voipbin-web-proxy`
     is recreated — including **routinely by the main stack's own CircleCI
     pipeline** — **symptom: HTTP 502**.
  Both are plain HTTP responses (not connection-refused/TLS errors), and
  both need to be special-cased in the rewritten script (§2), matching
  `infra-secret-sync-komodo.sh`'s own handling of the identical endpoint.

## Design

### Step 0 (mandatory, before any CI wiring or cutover): empirical spike

Using a disposable Stack (e.g. `claude-verify-call-manager-interp`,
destroyed after), confirm — do not assume:

1. A compose `environment` entry referencing `[[BIN_MANAGER__SOME_KEY]]`
   actually resolves to that Variable's real value in the deployed
   container (`docker inspect --format '{{json .Config.Env}}'` on
   bm-nyc-01, read not printed if secret-shaped).
2. What happens if a `[[VAR]]` reference does **not** resolve (typo a
   nonexistent variable name) — empty string, the literal `[[VAR]]` text,
   or a hard deploy failure. Whichever it is, the rewritten script (§2)
   must be able to detect and fail loud on it, not just log it.
3. Re-confirm `UpdateStack` with only `file_contents` in the request body
   does not disturb other Stack config fields (`server_id`,
   `poll_for_updates`, `auto_update`, `webhook_enabled`).
4. **(Round 2 M3: generalized from `DATABASE_DSN_BIN` alone.)** For
   *every* `BIN_MANAGER__*` variable referenced in §1's compose fragment
   (`DATABASE_DSN_BIN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`,
   `HOMER_API_ADDRESS`, `HOMER_AUTH_TOKEN`, `HOMER_WHITELIST`,
   `PROJECT_BASE_DOMAIN`, `GCP_BUCKET_NAME_MEDIA`, and `REDIS_PASSWORD` if
   §6 determines it's needed — see §1 note): confirm the stored value's
   shape matches what bin-call-manager's code expects verbatim, not a
   `.dist`-style unresolved placeholder (e.g. `${RABBITMQ_HOST:-rabbitmq}`)
   that would interpolate "successfully" into a literal, broken string. If
   any doesn't match, fix at the secret source (`infra-secret`), not in the
   compose fragment.

Document the results inline in this file (updating this section) before
proceeding to implementation. If 1 or 2 comes back unfavorable (silent
empty-string interpolation with no detectable signal), stop and redesign
the secrets-injection mechanism — do not proceed with a design whose
secrets path can fail silently into a running-but-broken container.

**Status (code review round 1, post-implementation): NOT YET PERFORMED.**
The scripts/compose fragment/CI wiring in this PR were implemented and
reviewed against Komodo's published API docs/source rather than against a
disposable Stack on the live bm-nyc-01 instance (no access to that
instance from this session). This is safe to merge as code because: (a)
`bin-call-manager-deploy`'s CI wiring only affects *routine* deploys that
happen after the real cutover, which is itself a manual, watched operation
per §7; and (b) §7 step 3 explicitly requires this Step 0 spike to be
completed, with results recorded here, before that manual cutover
proceeds. Do not run the actual cutover (§7) until this section is updated
with real results.

### 1. `bin-call-manager/komodo/docker-compose.yml` (new, git-tracked)

Directory named `komodo/`, sibling to `k8s/` (대표님's explicit convention
— per-service deploy definitions live alongside the service, one directory
per deploy target).

**The env list below is a starting draft, authored from `.dist`'s
call-manager block — it is NOT authoritative.** Round 2 (H3) found that
`.dist` and the live, operator-owned host file are known to diverge (the
whole reason §6 exists for the network name), and §1's env list inherited
the same problem: e.g. `.dist` has no `REDIS_PASSWORD` for call-manager,
but `BIN_MANAGER__REDIS_PASSWORD` exists as a Komodo Variable and
`bin-call-manager` reads `REDIS_PASSWORD` from its config
(`internal/config/main.go`) — whether it's actually required in production
is unknown from `.dist` alone. **§6 (pre-cutover verification) now captures
the live container's real env set and this file's final env list must be
reconciled against that capture before cutover, not assumed from `.dist`.**

```yaml
services:
  call-manager:
    image: voipbin/bin-call-manager:__IMAGE_TAG__
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    container_name: voipbin-call-mgr
    restart: always
    environment:
      - DATABASE_DSN=[[BIN_MANAGER__DATABASE_DSN_BIN]]
      - RABBITMQ_ADDRESS=[[BIN_MANAGER__RABBITMQ_ADDRESS]]
      - REDIS_ADDRESS=[[BIN_MANAGER__REDIS_ADDRESS]]
      - REDIS_DATABASE=1
      # - REDIS_PASSWORD=[[BIN_MANAGER__REDIS_PASSWORD]]  # add only if §6's live capture shows it set today
      - PROMETHEUS_ENDPOINT=/metrics
      - PROMETHEUS_LISTEN_ADDRESS=:2112
      - PROJECT_BASE_DOMAIN=[[BIN_MANAGER__PROJECT_BASE_DOMAIN]]
      - PROJECT_BUCKET_NAME=[[BIN_MANAGER__GCP_BUCKET_NAME_MEDIA]]
      - HOMER_API_ADDRESS=[[BIN_MANAGER__HOMER_API_ADDRESS]]
      - HOMER_AUTH_TOKEN=[[BIN_MANAGER__HOMER_AUTH_TOKEN]]
      - HOMER_WHITELIST=[[BIN_MANAGER__HOMER_WHITELIST]]
    command: ./call-manager
    networks:
      default: {}
networks:
  default:
    name: __NETWORK_NAME__
    external: true
```

Notes:
- **`healthcheck` intentionally omitted** (round 1 finding) — the live
  `wget`-based check can never pass on this distroless image; copying it
  into a *new* file would be an active regression, not preserved status
  quo. No functional replacement is defined here either, since distroless
  has no shell/exec target to probe from inside the container — see §5 for
  how deploy success is actually gated instead.
- `__IMAGE_TAG__` is a placeholder substituted by CI before each deploy
  (see §3) — never committed as a real tag.
- `__NETWORK_NAME__` is a placeholder filled in once, by hand, after §6
  confirms the real live network name (round 1 finding — not hardcoded as
  `install_default` without checking).
- `[[BIN_MANAGER__*]]` references are resolved by Komodo at deploy time —
  contingent on Step 0 confirming this actually works as intended.
- `depends_on` intentionally omitted (would be a hard Compose parse error
  referencing services this file doesn't define; `restart: always` +
  the app's own reconnect handling is the actual safety net today and
  doesn't change here).

### 2. `komodo-api-deploy.sh` rewrite: direct HTTPS, SSH kept as fallback

**The existing SSH-tunnel `komodo-api-deploy.sh` (VOIP-1341) is not
deleted.** It's renamed to `komodo-api-deploy-ssh-fallback.sh` (code
review round 2 also fixed one latent log-leakage gap in it — see §"Log
leakage guard" below, otherwise functionally unchanged), and documented as
the break-glass path for a production incident where
`https://komodo.voipbin.net` itself is down (round 1 "no break-glass path"
finding — this service no longer has `ssh-deploy.sh` available at all
after cutover, so *something* manual must still work).

New `komodo-api-deploy.sh` — direct HTTPS to `https://komodo.voipbin.net`,
authenticated via `X-Api-Key`/`X-Api-Secret` headers from
`CC_KOMODO_API_KEY`/`CC_KOMODO_API_SECRET`. Same argv-avoidance discipline
as today's script and `infra-secret-sync-komodo.sh` (`curl -K -`, secrets
never in argv/env dump). All calls go through one `API_BASE` + `call_api`
seam (matches `infra-secret-sync-komodo.sh`'s own stated rationale — a
future credential/domain change becomes a one-line edit).

Interface: `komodo-api-deploy.sh <stack-name> <compose-file>`

1. **Idempotent ensure-exists: `GetStack` first; if missing, `CreateStack`.**
   Revised post-approval (대표님's explicit instruction, 2026-08-16 —
   supersedes both this section's original text and round-1's finding
   below, which had gone the other way for good reasons that no longer
   win out against operational reality): a per-service *manual* bootstrap
   step doesn't scale once this pattern rolls out to the other 31
   `bin-*-manager` services — that would be 31 more watched, by-hand
   Stack creations. CI now creates the Stack itself, idempotently, the
   first time it's missing.
   - **Detecting "missing" is by error-message match, not HTTP status.**
     Confirmed empirically (2026-08-16, from a real CI failure, not
     assumed): Komodo's `GetStack` returns **HTTP 500** — not 404 — with
     body `{"error":"Did not find any Stack matching <name>","trace":[]}`
     for a nonexistent Stack. `komodo-api-deploy.sh` matches on that exact
     substring before deciding to create; any other failure (auth, a real
     500, the 404/502 network-fragility modes above) still aborts without
     attempting a create. **Unverified edge case (code-review round 5):**
     if Komodo ever returns this same "not found" message for "exists but
     caller lacks read permission" (a common API pattern, to avoid leaking
     resource existence), this script would attempt `CreateStack` against
     an already-existing name. The CI credential is Admin, not
     read-restricted, so this is unlikely in practice — but it's untested.
     Confirm before rolling this pattern out past this one pilot service.
   - `CreateStack` uses `server_id` = `$KOMODO_SERVER_NAME` (default
     `bm-nyc-01`, overridable via `CC_KOMODO_SERVER_NAME` per job) — safe
     to pass the plain server **name**, not an internal id: Komodo's own
     `validate_config` resolves `server_id` by name if it doesn't look
     like an id ("in case it comes in as name", per its source), so no
     separate `ListServers` lookup is needed to resolve bm-nyc-01's actual
     Server resource id first. **Not yet confirmed against the live
     instance** (code-review round 5 finding) — this specific behavior is
     read from Komodo's source, not observed; §7 step 7 (the first real
     bootstrap run) is effectively this assumption's first live test, so
     watch it interactively rather than trusting it unattended the first
     time.
   - Same config values the (now-superseded) manual-bootstrap text always
     specified: empty `file_contents` (the very next `UpdateStack` call
     fills it), `poll_for_updates`/`auto_update`/`webhook_enabled` all
     `false` — Komodo still does nothing on its own; CI is still the only
     trigger. This did not change, only *who* performs the create did.
   - **Round-1's original objection (recorded for the record, not
     retracted as *wrong*, just overridden):** the CI credential is full
     Komodo Admin, so an unattended auto-create path is real, if narrow,
     blast radius — a typo'd `<stack-name>` argument creates a stray
     Stack rather than failing loud. Accepted as a known, bounded
     trade-off: `<stack-name>` is a literal string in version-controlled
     CI config (`config_work.yml`), not attacker- or user-supplied input,
     so the realistic failure mode is a caught-in-review typo, not a
     runtime injection.
2. **Before `UpdateStack`, guard against any unfilled placeholder token**
   (round 3 M3): `grep -Eq '__[A-Z_]+__' <rendered compose> && fail` — this
   catches not just a missed `__IMAGE_TAG__` substitution (already handled
   by `render-image-tag.sh`'s own check, §3) but also `__NETWORK_NAME__`,
   which is filled in by hand once into the git-committed file (§1) and
   has no per-deploy script to guard it otherwise. Fails closed on the CI
   side with a clear message, instead of failing mid-cutover as a Komodo-side
   `external: true` network-not-found error.
3. `UpdateStack(stack, config: {file_contents: <rendered compose>})` —
   PATCH-style, verified not to disturb other config (Step 0, item 3).
4. `DeployStack(stack)`.
5. Poll `GetUpdate` until `status == Complete`; exit 0 only if
   `success == true`.
6. **`call_api`'s error handling special-cases 404 and 502** exactly as
   `infra-secret-sync-komodo.sh` does against the same endpoint: a 404
   response prints a pointer to `infra-komodo/README.md`'s Caddy-route
   recovery steps; a 502 prints the `docker network connect
   komodo_default voipbin-web-proxy` recovery command. A generic
   "non-2xx" message is not acceptable here, since these two codes are the
   expected/routine failure shapes for this endpoint, not edge cases.
7. **Log leakage guard (round 1 MEDIUM finding):** the request bodies sent
   are safe to log as-is (un-interpolated `[[VAR]]` placeholders, never
   real secret values — a deliberate property of this design, worth
   keeping). Komodo's own `GetUpdate` response logs are *post*-interpolation
   `docker compose` output, and a compose parse error could echo real
   DSN/AMQP values into CircleCI's job log. On failure, do not print
   `GetUpdate`'s raw log content verbatim; print the update ID and a
   pointer to check it via the Komodo UI/API directly (an authenticated
   channel), not CI's own (broadly readable) log.
8. **Secrets-at-rest surface, stated explicitly (round 2 M6):** after
   interpolation, real production DSN/AMQP credentials exist in Komodo's
   Mongo and on bm-nyc-01's Periphery-managed stack directory, readable by
   anyone authenticated to `komodo.voipbin.net` — which per
   `infra-komodo/README.md` is internet-exposed behind a single password,
   no MFA. Not a new regression (PR #90 already put these values in
   Komodo Variables; the old path had them in `install/.env` on the same
   host either way), but this design newly routes production call-path
   credentials through that surface and that trade should be explicit, not
   implicit.

### 3. Image-tag injection (`render-image-tag.sh`, new)

`render-image-tag.sh <compose-file> <tag>`:
- `grep -q '__IMAGE_TAG__' "$compose_file" || { echo "placeholder not found"; exit 1; }`
- In-place substitute `__IMAGE_TAG__` → `<tag>` (`$CIRCLE_SHA1`).
- Touches nothing else in the file.

### 4. CI wiring

`bin-call-manager-deploy` job body changes from:
```
command: .circleci/scripts/ssh-deploy.sh voipbin/bin-call-manager call-manager
```
to:
```
command: |
  .circleci/scripts/render-image-tag.sh bin-call-manager/komodo/docker-compose.yml "$CIRCLE_SHA1"
  .circleci/scripts/komodo-api-deploy.sh bin-call-manager bin-call-manager/komodo/docker-compose.yml
```
Requires `CC_KOMODO_API_KEY`/`CC_KOMODO_API_SECRET` in the `production`
context — already present (verify not rotated since PR #90, don't assume).

**Poll timeout, set explicitly** (round 1 MEDIUM finding): default
30×2s=60s may not cover a cold image pull for this service's actual image
size. Measure during implementation/testing and set
`CC_KOMODO_POLL_ATTEMPTS`/`CC_KOMODO_POLL_INTERVAL_S` explicitly for this
job rather than relying on the library default. On `DEPLOY_TIMEOUT`,
document that the deploy may still land after CI gives up — an operator
must check Komodo directly (§2's log-leakage guard is why CI's own output
isn't the source of truth here) before assuming failure and retrying.

### 5. Deploy-success gate

Two layers, neither fully sufficient alone (round 1 CRITICAL finding: a
"container is running" gate cannot by itself distinguish a healthy
call-manager from one that booted with an empty/broken DSN):

**a. Automated, every deploy (routine CI runs):** Komodo's own
`GetUpdate` `Complete`+`success` (started, not necessarily healthy) plus a
poll of `ListStackServices` (Komodo's read endpoint for per-service
container state, per its published API docs/source) for 3 consecutive
`.container.state == "running"` samples, same numeric gate `ssh-deploy.sh`
already uses, read via Komodo's API instead of SSH `docker inspect`.
**Implemented in `komodo-api-deploy.sh` (code review round 1), but NOT YET
empirically verified against the live Komodo instance** — confirm
`ListStackServices`'s actual response shape on bm-nyc-01 matches what the
script assumes; if it doesn't, this gate will consistently report
not-running and every deploy will fail it (fail-closed, not fail-open, but
still worth confirming before relying on it unattended). If it turns out
Komodo's read API doesn't expose this in enough detail after all, SSH may
need to stay in the loop for this specific check (decide
from the real response shape, not assumed here).

**b. Manual, hard-blocking, cutover only (§7). Revised again per round 3
(C1) — round 2's `call-control call get --id 00000000-...` fix does not
work: `resolveUUID` rejects the nil UUID before any DB call
(`cmd/call-control/main.go`), `Get` on a real-but-missing ID returns an
*error* (`CALL_NOT_FOUND`, exit 1) rather than a "clean success" shape, and
critically the CLI's `initHandler` connects to RabbitMQ too — with an
**unbounded retry loop** (`rabbitmqhandler.connect()`), so if RabbitMQ is
unreachable this command hangs forever instead of erroring. "Bypasses
RabbitMQ entirely" was false. Two checks, both required, revised:

```bash
# (i) Negative gate — narrowed to boot-path failure strings (round 3 M2:
# the earlier broader pattern would false-positive on unrelated transient
# dial failures, e.g. Homer/asterisk-proxy, at DebugLevel log volume).
# Must be empty.
docker logs --since 2m voipbin-call-manager | \
  grep -Ei 'could not connect to the database|could not initialize the cache|could not connect to rabbitmq|panic:'

# (ii) Positive gate — exercises DB connectivity via the control CLI,
# bound to a timeout since the RabbitMQ leg of initHandler retries
# unboundedly on failure rather than erroring (round 3 C1):
timeout 30 docker exec voipbin-call-manager /app/bin/call-control call list \
  --customer-id 11111111-1111-1111-1111-111111111111 --limit 1
```
`initHandler` connects DB → Redis (both via a real `Ping`, per
`databasehandler.Connect`/`cachehandler.Connect`) → RabbitMQ, in that
order, so (ii) *does* exercise all three — just not by "bypassing"
RabbitMQ, by racing against its retry loop under a timeout. Pass criterion
for (ii): **exit 0** (an empty `[]` result for a customer ID with no calls
is the expected, successful shape — `callHandler.List` returns an empty
slice with nil error on no rows, not an error, unlike `Get`). A **non-zero
exit** (DB/Redis connect failure surfaces as a returned error from
`initHandler`) or a **timeout** (RabbitMQ unreachable) are both fail.
**Open item to verify empirically before relying on this in production:**
confirm `call list`'s exact exit code and output shape on an empty result
against a real bm-nyc-01 Komodo-deployed container — this reasoning is
derived from reading `pkg/callhandler/db.go`/`cmd/call-control/main.go`,
not yet observed live.
Either check failing is a hard abort trigger, not a judgment call. Both
are manual (SSH-based observability/exec, not a deploy mechanism) and
apply only to the cutover itself; routine subsequent deploys rely on gate
(a) plus whatever monitoring/alerting already watches this service in
production (unchanged by this design).

### 6. Pre-cutover verification (round 1 HIGH; expanded round 2 — H2, H3)

**Status: DONE (2026-08-16).** Ran against the live host after the first
CI run correctly failed closed on the unfilled `__NETWORK_NAME__`
placeholder (the guard doing exactly its job). Results: `com.docker.compose.project`
= `install`, network = `install_default` (matches the pre-verification
assumption, but was not trusted without checking — see below). Live env
key set matches §1's draft list exactly; `REDIS_PASSWORD` confirmed **not**
set on the live container, so it stays commented out in
`bin-call-manager/komodo/docker-compose.yml`. `__NETWORK_NAME__` has been
filled in with `install_default` in that file. `$LIVE_PROJECT` in §7/
rollback below is `install` — matches what was already written there, now
confirmed rather than assumed.

Before touching anything, on bm-nyc-01, capture both the network/project
identity and the full live env in one pass:
```bash
docker inspect voipbin-call-mgr --format \
  '{{index .Config.Labels "com.docker.compose.project"}} {{json .NetworkSettings.Networks}}'
docker inspect voipbin-call-mgr --format '{{json .Config.Env}}' > pre-cutover-env.json
```
- The `com.docker.compose.project` label is the **single source of truth**
  for the `-p` value used everywhere in §7 (step 3, rollback) — round 2
  (H2) found the design asserting `-p install` in the same document that
  refuses to assert the network name, which is inconsistent: a wrong `-p`
  on step 3 makes the `rm` a silent no-op (conflict surfaces mid-cutover
  at step 6 instead), and a wrong `-p` on rollback creates a *second*,
  differently-projected container during an incident. Use the captured
  value, not `install`, in every command below.
- The network name fills `__NETWORK_NAME__` in §1. Do not assume
  `install_default`; a stale, empty, differently-named network would
  attach successfully (defeating the `external: true` fail-closed
  property) while leaving `db`/`redis`/`rabbitmq` unreachable.
- `pre-cutover-env.json` is the **authoritative source for §1's final env
  list** (round 2 H3) — reconcile every `KEY=value` in it against §1's
  draft list before cutover: every key present live must appear in §1
  (add `REDIS_PASSWORD` if it's actually set), and every key in §1 must
  correspond to something the live container actually has. Treat
  "zero unexpected diff between this capture and §1's final list" as a
  **pre-swap** gate — this replaces relying solely on the **post**-swap
  diff (§7 step 7) to discover a missing/extra variable, which only finds
  the problem after the outage window has already opened.

### 7. Cutover sequencing

(Below, `$LIVE_PROJECT` is the `com.docker.compose.project` value captured
in §6, and `$LIVE_NETWORK` is the network name captured there — not
hardcoded as `install`/`install_default`.)

**Round 4 correction (CRITICAL, supersedes round 3's H1 fix):**
`build-approval` in `.circleci/config_work.yml` gates the **entire**
`test → build → deploy` chain, upstream of all three — not a pause point
between build and deploy (the config's own comment: "Single approval gate
... covers the whole pipeline including this deploy... A second
deploy-specific approval was tried and doubles as staging, and requiring
two clicks per deploy added [friction]"). Round 3's "merge, let build run,
then approve before deploy" sequencing is not executable against this
topology: not approving means no image ever gets pushed (step 4's pre-pull
would have nothing to pull); approving runs straight through to deploy
with no pause, hitting the exact `container_name` conflict steps 5-6
existed to prevent, unattended.

**Fix: decouple the risky swap from CI's pipeline entirely for this one,
first cutover deploy.** The swap is inherently a manual, watched operation
regardless (this design has said so throughout) — the correction is to
stop trying to route it through CI's single-gate topology at all, and
instead do the actual cutover by hand, with the CI wiring merged
separately, once Komodo already owns the container and no name conflict
exists to protect against.

1. **Pre-swap checkpoint:** §6's captures (project/network identity,
   `pre-cutover-env.json`), plus the current image ref
   (`docker inspect voipbin-call-mgr --format '{{.Image}}'`) — all done
   first, well before anything below.
2. **Stack bootstrap — no longer a separate manual step.** §2 item 1's
   revision means step 7 (running `komodo-api-deploy.sh` by hand) creates
   the Stack itself, idempotently, the first time it doesn't find one.
   Nothing to do here ahead of time.
3. **Build and push the image by hand**, from a workstation (not via CI,
   and not tied to a `main` merge SHA — any clear, traceable tag works,
   e.g. the feature branch's current commit). Matches what CI's
   `docker-build` command does — repo root context, not the service
   subdirectory (the Dockerfile's `COPY ./ .` then `cd bin-call-manager`
   depends on it):
   ```bash
   docker build --tag voipbin/bin-call-manager:<manual-tag> \
     -f bin-call-manager/Dockerfile .
   docker push voipbin/bin-call-manager:<manual-tag>
   ```
4. **Pre-pull that exact tag** on bm-nyc-01:
   `docker pull voipbin/bin-call-manager:<manual-tag>` — moves the cold
   pull outside the outage window that opens at step 5.
5. **Remove the old instance:**
   `cd /opt/voipbin/install && docker compose -p "$LIVE_PROJECT" rm -sf call-manager`
   (not `down` — preserves the shared network, now cross-project-owned;
   `-p "$LIVE_PROJECT"` per round 2 H2 — never hardcode `install`). The
   outage window starts here.
6. **Comment out** (never delete) call-manager's block in **both**
   `voipbin/voipbin install/docker-compose.yml` (the live host file) **and
   note in `install/README.md`** that `docker-compose.yml.dist` still
   contains this block too (round 1 finding: a host rebuild/fresh install
   regenerates live from `.dist` and reintroduces the exact
   `container_name` conflict this step exists to prevent — `.dist` itself
   is out of scope to edit here since it's the external self-hosting
   template, but the risk must be recorded where an operator will see it).
   This step is manual SSH — outside Komodo's API surface entirely, since
   this file belongs to a different, non-Komodo-managed system. One-time.
   **Between this step and step 10 (CI wiring merge), `bin-call-manager-deploy`
   still runs the old `ssh-deploy.sh` if some unrelated merge gets
   approved** — it would hit a commented-out service and fail loudly
   (`bump-image-digest.sh`/`docker compose pull` against a service Compose
   no longer knows about) without touching the Komodo-managed container.
   Fail-closed, not a production hazard, but keep this window short and
   expect a red CI build as the (harmless) symptom if it happens.
   **Mandatory check before step 7 (naming decision, 대표님, 2026-08-16):**
   confirm the old container is actually gone —
   `docker ps -a --filter "name=^/voipbin-call-mgr$" --format '{{.Names}}'`
   must return nothing. Because the new container's name
   (`voipbin-call-manager`) deliberately differs from the old one
   (`voipbin-call-mgr`), Docker will **not** refuse to start the new
   container even if the old one is somehow still running — the
   mutual-exclusion this design originally relied on (same name = OS-level
   refusal to double-run) no longer applies. This check is what replaces
   it; do not skip it because step 5 "already removed" the old container —
   confirm, don't assume.
7. **Run `render-image-tag.sh` + `komodo-api-deploy.sh` by hand** from a
   workstation with `CC_KOMODO_API_KEY`/`CC_KOMODO_API_SECRET` available
   locally, targeting the manually-pushed tag from step 3 — watched, not
   queued through CI. This is the actual cutover trigger; the image is
   already local from step 4, so it should not need to pull.
8. **Confirm (all of, not any of):** container up, gate 5a's
   3-consecutive-running check passed (checked by hand here, same
   criteria), env diff against `pre-cutover-env.json` (zero unexpected
   differences — confirming §6's pre-swap reconciliation), **and** gate
   5b's two checks (narrowed negative-log-grep empty, `call-control call
   list` exits 0 within its timeout) both pass.
9. **Abort criterion:** any of step 8's checks fails → stop immediately,
   roll back (below) — do not retry blindly or debug live in production.
10. **Only after step 8 confirms success:** merge the CI wiring change
    (§4) to `main` through the normal, unmodified `build-approval` gate.
    This is no longer a risky moment — Komodo already owns
    `voipbin-call-manager`, so the next CI-triggered deploy is an ordinary
    image update on an already-Komodo-managed Stack, not a name-conflict
    situation. `bin-call-manager-build` will rebuild and re-push the same
    source at a new (`main`-merge) SHA — redundant with step 3's manual
    build but harmless — and `bin-call-manager-deploy` runs normally from
    then on.

**Traffic-gap impact (round 1 MEDIUM finding — expanded, not just async
queuing; step numbers corrected round 3 M1):** `bin-call-manager` is a
**synchronous RabbitMQ RPC target** for the rest of the monorepo, not only
an async event consumer. During the gap between steps 5 and 7, RPC callers
(api-manager creating a call,
flow-manager, hangup paths) block until their own RPC timeouts expire —
user-visible failures during the gap, not just delayed background work.
ARI events for calls already in progress (hangup, bridge changes) are also
delayed, risking channel-state desync and billing-timing drift for calls
active at swap time. **Maximum acceptable gap: state a concrete number
before executing this in production** (not defined in this design — a
production-operations judgment call for 대표님, informed by current call
volume at the chosen swap window) and treat exceeding it as an independent
abort trigger, not just "reconnect and move on." Schedule the swap in the
lowest-traffic window available.

**Monitoring continuity (naming decision, 대표님, 2026-08-16 — new
concern, didn't exist while the name was unchanged):** whatever scrapes
`:2112/metrics` for Prometheus and whatever dashboards/alerts reference
this service by container name both need to be checked for whether they
key on `voipbin-call-mgr` specifically (breaks at cutover, needs updating
to `voipbin-call-manager`) or on something name-independent (compose
service label, a static target list keyed by IP/port, etc. — unaffected).
**Not yet checked in this design** — confirm before or immediately after
the cutover, not as an afterthought if alerts go quiet.

**Rollback — ordered, CI-wiring-first (round 1 HIGH finding: the original
draft's rollback left the merged CI change live, so the next unrelated
merge touching bin-call-manager would silently re-run the new path over a
rolled-back container):**
**If aborting at §7 step 9 (before step 10's CI merge):** the CI wiring
was never merged — there is nothing to revert, simply don't do step 10.
Steps 2-3 below still apply.

**If aborting after step 10 has already merged:**
```bash
# 1. FIRST: revert/disable the CI wiring change (§4) so no subsequent
#    merge re-triggers komodo-api-deploy.sh against this service.
# 2. Destroy the Komodo Stack (round 2 M1 — otherwise ANY later Deploy
#    action against it, UI click or otherwise, recreates the
#    voipbin-call-manager container step 3 below removes):
#    DestroyStack(bin-call-manager) via the Komodo API/UI.
# 3. On bm-nyc-01, remove the NEW (Komodo-managed) container - note the
#    name difference from the old one (naming decision, 대표님, 2026-08-16):
docker rm -f voipbin-call-manager 2>/dev/null || true
# Un-comment call-manager's block in install/docker-compose.yml, which
# recreates the OLD container under its original name (voipbin-call-mgr):
cd /opt/voipbin/install && docker compose -p "$LIVE_PROJECT" up -d call-manager
```

**`install_default` (or whatever §6 confirms the real name is) becomes a
shared, cross-project network as of this cutover** — `docker compose -p
install down` (full teardown) would now conflict with the Komodo-managed
container still attached to it. Document this in `install/README.md` as a
standing constraint, not a one-time migration note (same place as the
`.dist` landmine note above).

**Auditability (round 1 MEDIUM finding):** dropping `versions.lock` means
git no longer records what's running on bm-nyc-01 for this service. State
plainly: that answer now lives in Komodo's stored `file_contents` for the
`bin-call-manager` Stack (queryable via `GetStack`), and the stale
`bin-call-manager` entry left behind in the live `versions.lock` after
cutover is intentional rollback ballast, not drift to be cleaned up.

## Testing

- **Existing `.circleci/tests/komodo-api-deploy.bats` is renamed to
  `komodo-api-deploy-ssh-fallback.bats`** (paths/references updated to
  match the §2 script rename), preserving its coverage of the retained
  SSH-tunnel fallback script rather than silently losing it or having it
  retarget the new HTTPS script by accident (round 2 M4). Note: as of this
  writing, bats isn't wired into any `.circleci/*.yml` pipeline stage
  (developer-run only) — this rename doesn't change CI behavior, only
  keeps the test suite honest about what it covers.
- New bats suite for the rewritten `komodo-api-deploy.sh` (stub `curl`,
  assert on request bodies/headers, the 404/502 special-casing, and the
  log-redaction-on-failure behavior) and for `render-image-tag.sh`.
- Step 0's empirical spike (above) — required, not optional, before any of
  the above is trusted in production.
- Manual/watched verification of the actual cutover on bm-nyc-01 per §7 —
  a production step, not something CI verifies unattended the first time.

## Explicitly out of scope

- Migrating any service other than bin-call-manager.
- Decomposing the rest of the monolithic `docker-compose.yml`.
- Fixing the pre-existing broken `wget`-based healthcheck on the *old*
  path (moot for the new path, which drops the healthcheck — §1).
- Migrating the shared network to a project-agnostic name/ownership.
- Changing anything in `voipbin/voipbin`'s tracked `install/` files beyond
  documentation (`install/README.md`) — no code/compose changes there.

## Review corrections log

**Round 1** (3 CRITICAL, 4 HIGH, 5 MEDIUM):
- CRITICAL 1 (unverified `[[VAR]]` interpolation cited as confirmed) →
  Step 0 added; claim downgraded throughout.
- CRITICAL 2 (dead healthcheck copied forward; gate can't detect broken-but-running) →
  healthcheck dropped from new file (§1); two-layer gate added (§5) —
  further revised in round 2, see H1 below.
- CRITICAL 3 (routine 404/502 fragility unaddressed; no break-glass path) →
  §2 (special-casing) and the SSH-tunnel script kept as fallback.
- HIGH 4 (rollback doesn't stop CI re-deploying over it; missing `cd`) →
  rollback reordered CI-wiring-first, `cd` added (§7) — `cd` also added to
  the forward path in round 2 (M5).
- HIGH 5 (network name asserted, not verified) → §6 added; compose
  fragment uses a placeholder instead of hardcoding `install_default`.
- HIGH 6 (CI auto-creates Stacks, reversing VOIP-1341's own design intent) →
  dropped; Stack creation is manual-only (§2 item 1, §7 step 2).
- HIGH 7 (PATCH-style `UpdateStack` claim mis-cited) → citation corrected;
  re-verification folded into Step 0.
- MEDIUM 8 (traffic-gap analysis understated sync-RPC impact) → expanded
  in §7.
- MEDIUM 9 (Komodo update logs could leak secrets into CI logs) → §2 added
  (later renumbered to item 7 by round-3's M3 insertion).
- MEDIUM 10 (`.dist` file still a landmine) → noted in §7.
- MEDIUM 11 (loss of git-based deployed-version auditability unstated) →
  noted in §7.
- MEDIUM 12 (poll timeout budget unstated) → §4 addresses explicitly.

**Round 2** (3 HIGH, 6 MEDIUM — round 1's fixes verified correct except
where noted):
- H1 (gate 5b's `ContactStatusChange` grep unreliable and doesn't exercise
  DB/Redis) → replaced with a two-part negative-log-grep +
  `call-control call get` DB round-trip check (§5).
- H2 (`-p install` hardcoded in the same doc that refuses to assert the
  network name) → §6 now captures the real compose-project label
  alongside the network name; §7 and rollback use `$LIVE_PROJECT`
  throughout instead of a literal `install`.
- H3 (new file's env list authored from `.dist`, divergence discovered
  only post-swap) → §1 marks the env list as a non-authoritative draft;
  §6 now captures the live container's actual env pre-cutover and makes
  reconciliation a pre-swap gate.
- M1 (rollback leaves the Komodo Stack live, re-deployable into a
  conflict) → `DestroyStack` added as an explicit rollback step.
- M2 (cold image pull sits inside the outage window) → pre-pull step
  added between Stack bootstrap and container removal (§7).
- M3 (Step 0 item 4 shape-checked only one variable) → generalized to all
  referenced `BIN_MANAGER__*` variables.
- M4 (rename breaks/silently retargets the existing bats suite) →
  existing suite explicitly renamed alongside the script; new suite
  created for the new script (Testing).
- M5 (missing `cd` in the forward path, only fixed in rollback) → added.
- M6 (secrets-at-rest exposure via Komodo's admin surface unstated) → §2
  item 8 added.

**Round 3** (1 CRITICAL, 1 HIGH, 3 MEDIUM):
- C1 (gate 5b(ii) cannot pass as written — nil UUID rejected pre-DB-call,
  `Get`'s not-found is an error not a success shape, and the CLI's
  RabbitMQ dial retries unboundedly rather than erroring, so "bypasses
  RabbitMQ" was false and a RabbitMQ-down state would hang the check
  forever) → replaced with `call list --customer-id <random-non-nil-uuid>`
  under `timeout 30`, pass-on-exit-0 (empty-list is a real List-vs-Get
  behavior difference), explicitly flagged for empirical confirmation
  against a live container (§5).
- H1 (pre-pull in the old step 3 pulled a tag that didn't correspond to
  what step 6 actually deployed, since the merge commit's image didn't
  exist yet — no real outage-window reduction) → cutover reordered: merge
  CI wiring first (build pushes the real SHA, deploy stays behind
  `build-approval`), pre-pull *that* SHA, then remove the old container,
  then approve the deploy (§7).
- M1 (traffic-gap paragraph referenced stale step numbers after M2's
  insertion) → corrected to match the reordered steps.
- M2 (negative-log-grep patterns too broad for a mandatory-abort gate,
  would false-positive on unrelated transient dial failures at DebugLevel
  log volume) → narrowed to the specific boot-path failure strings (§5).
- M3 (nothing guards against `__NETWORK_NAME__` shipping unfilled) →
  generic `__[A-Z_]+__` placeholder guard added to `komodo-api-deploy.sh`
  before every `UpdateStack` call (§2).

**Round 4** (1 CRITICAL, plus 3 stale-reference cleanups):
- CRITICAL (round 3's H1 fix doesn't execute against the real CI
  topology — `build-approval` gates the entire `test → build → deploy`
  chain, not a pause point between build and deploy; approving to get a
  build also runs the deploy straight through with no room for the
  manual old-container-removal/comment-out steps in between) → the risky
  swap is decoupled from CI's single-gate pipeline entirely: build/push
  the image by hand, do the removal/comment-out/deploy by hand
  (watched), confirm success, and only then merge the CI wiring change
  for *future* ordinary deploys, which no longer have a name conflict to
  worry about (§7, fully restructured).
- Stale cross-references cleaned up: the round-1 corrections log's "§2
  item 5/6" citations (renumbered by round-3's M3 insertion), the
  rollback's `DestroyStack` comment (referenced "step 4", the list only
  has 3 steps), and `bin-call-manager/CLAUDE.md`'s command table, which
  still claimed `call-control call get` "(bypasses RabbitMQ)" — corrected
  in the same PR since round-3's C1 established `initHandler` does dial
  RabbitMQ (with an unbounded retry loop), and gate 5b now depends on
  that corrected understanding.

**Post-approval naming revision** (대표님, 2026-08-16, after PR #1188 was
opened — not an architect-review round, but logged here for the same
reason):
- New container renamed `voipbin-call-mgr` → `voipbin-call-manager`,
  matching GKE's `call-manager` app name rather than `install/`'s
  `-mgr`-abbreviated convention (verified against
  `bin-call-manager/k8s/deployment.yml` before accepting the change, not
  taken on faith — GKE's actual name has no `voipbin-` prefix at all,
  which was surfaced and 대표님 chose `voipbin-call-manager` explicitly
  over matching GKE exactly).
- Consequence handled: the old/new name match this design leaned on for
  automatic Docker-level mutual exclusion no longer holds. §7 gained an
  explicit "confirm the old container is actually gone" check before
  deploying the new one, and a monitoring-continuity flag (container-name-keyed
  scrape configs/dashboards/alerts, not yet checked).
- `bin-call-manager/komodo/docker-compose.yml`, and every design-doc
  reference to the *new* container (gate 5b's commands, the post-cutover
  "Komodo already owns..." note, and the rollback's `docker rm`) updated
  accordingly. References to the *old* container (pre-cutover checkpoint,
  §6's live-host captures) correctly stay `voipbin-call-mgr` — that
  container's real name is unchanged by this decision.

**Post-approval Stack-bootstrap revision** (대표님, 2026-08-16, after the
naming revision above): CI now creates the Komodo Stack itself,
idempotently, reversing round-1's "does not create Stacks" finding — see
§2 item 1 for the full reasoning and the empirically-confirmed detection
mechanism (Komodo returns HTTP 500, not 404, for "Stack not found").
§7 step 2's separate manual bootstrap is removed accordingly — folded into
step 7's own first run.

**Follow-up dependency, not in this PR's scope (대표님, 2026-08-16), tracked
as [VOIP-1343](https://voipbin.atlassian.net/browse/VOIP-1343):** a
neutral shared network `production`, owned by neither the `install`
compose project nor any Komodo Stack, was built out in a separate session
(`NOJIRA-Add-production-network-komodo-stack`, monorepo-etc PR #92) — this
is the same "network genuinely owned by neither compose project"
follow-up this document already flagged (see the `install_default`
shared-network note above). **Verified live on bm-nyc-01 (2026-08-16): the
`production` network exists, but `db`/`redis`/`rabbitmq` are not attached
to it yet** — `backfill-install-default.sh` (in that PR) needs to run
first. Once that's done, `bin-call-manager/komodo/docker-compose.yml`'s
`networks.default.name` should move from `install_default` to
`production` — but not before, and not as part of this pilot's cutover.
Sequencing matters: that network must exist and carry the shared infra
containers *before* any service tries to attach to it, so this is a
prerequisite for the *next* service's Komodo migration at the earliest,
not a same-day follow-up to this one.
