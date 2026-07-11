# bin-dbscheme-manager: replace seed-image build with real production alembic upgrade (VOIP-1246)

## Problem

`bin-dbscheme-manager`'s CircleCI workflow currently has an `approval` gate
that looks like it authorizes applying database migrations to production,
but it does not:

```yaml
bin-dbscheme-manager:
  when: << pipeline.parameters.run-bin-dbscheme-manager >>
  jobs:
    - build-approval:
        type: approval
    - bin-dbscheme-manager-build:
        <<: *context_production
        requires:
          - build-approval
```

`bin-dbscheme-manager-build` runs `docker-build`, which builds
`bin-dbscheme-manager/Dockerfile` and pushes `voipbin/bin-database:<sha>`
(and `:latest` on `main`). The Dockerfile:

1. Stage 1 (`python:3.11-slim`): spins up a throwaway local MariaDB, creates
   empty `bin_manager` and `asterisk` databases, runs
   `alembic upgrade head` against them, then `mysqldump`s the result to two
   `.sql` files.
2. Stage 2 (`mysql:8.0`): copies those dump files into
   `/docker-entrypoint-initdb.d/`.

`/docker-entrypoint-initdb.d/` only executes on a MySQL container's **first
boot with an empty data directory**. It never touches an already-running
database. So this pipeline produces a "seed" image usable for spinning up a
brand-new empty environment. It has never applied a migration to the live
production database. Per `docs/operations.md`, that has been done by a
human, manually, over VPN.

Decision (confirmed by pchero, 2026-07-11): stop building the seed image in
this workflow. Replace it with a job that runs `alembic upgrade head`
directly against the production `bin_manager` and `asterisk` databases,
using the same `CC_DATABASE_DSN` / `CC_DATABASE_DSN_ASTERISK` secrets the
k8s release jobs already have in the `production` CircleCI context.

## Non-goals

- Not changing how any other `bin-*-manager` service's CI/CD works.
- Not changing the `voipbin/sandbox` migration path.
  **Verified (not assumed):** `grep -rln "bin-database" ~/gitvoipbin/sandbox`
  returns zero matches. `sandbox/scripts/migrate.sh` clones
  `bin-dbscheme-manager` source directly at a pinned commit
  (`git clone --filter=blob:none --sparse` from
  `github.com/voipbin/monorepo.git`, sparse-checkout
  `bin-dbscheme-manager`) and runs alembic inside its own throwaway
  container — it never references or pulls the `voipbin/bin-database`
  Docker image this change removes. This claim was independently confirmed
  by grepping the actual `sandbox` repo checkout at `~/gitvoipbin/sandbox`,
  not inferred from the monorepo alone.
- Not building a rollback-automation job. `alembic downgrade` stays a manual,
  human-only, VPN-required action per `docs/operations.md` (unchanged).
- Not addressing large-table zero-downtime tooling (`pt-online-schema-change`
  / `gh-ost`). Out of scope for this ticket; flagged as a follow-up below.

## Design review history

### Round 1 findings and resolution

An independent adversarial review (delegate_task, round 1) found two
load-bearing defects in the initial draft:

1. **DSN format mismatch (confirmed via code, blocking).** `CC_DATABASE_DSN`
   is consumed today, in exactly one place in this codebase, by Go's
   `sql.Open("mysql", dsn)` (`bin-common-handler/pkg/databasehandler/main.go`),
   which uses the `go-sql-driver/mysql` DSN grammar:
   `user:password@tcp(host:port)/dbname?param=value`. This is **not** the
   same grammar as a SQLAlchemy `sqlalchemy.url`. Feeding the raw Go-format
   DSN into `alembic.ini` via `sed` as originally drafted would fail to
   parse. **Fix direction:** parse the Go-format DSN and construct a proper
   SQLAlchemy URL before writing `alembic.ini`.
2. **No concurrency guard (confirmed gap, blocking).** Nothing prevented two
   overlapping executions of this job from running `alembic upgrade head`
   against the same database concurrently. **Fix direction:** wrap each
   alembic invocation in a MySQL named lock (`GET_LOCK`/`RELEASE_LOCK`).

Also adopted as improvements: `alembic upgrade head --sql` dry-run preview
before the real apply, explicit `no_output_timeout`, and avoiding DSN
leakage into CircleCI logs.

### Round 2 findings and resolution

Round 2 re-reviewed the round-1 fixes adversarially and found the fixes
themselves were broken or incomplete in three ways — **all now fixed in
this revision**:

1. **Query parameters silently dropped (confirmed, blocking).** The DSN
   conversion regex captured `user`/`pass`/`host`/`port`/`db` but discarded
   anything after `?` in the Go DSN (e.g. `?parseTime=true&tls=custom`,
   `?tls=skip-verify`). If production's DSN relies on a `tls=` parameter,
   dropping it silently could make the migration connection either fail
   outright or (worse) fall back to an unencrypted connection. **Fixed:**
   the converter now explicitly extracts and re-maps known
   security-relevant params (`tls`) onto the equivalent PyMySQL/SQLAlchemy
   query string instead of dropping the query string wholesale. See revised
   "DSN conversion" below.
2. **Lock-acquisition script had a fatal, always-failing regex bug
   (confirmed, blocking, NEW defect introduced by the round-1 fix itself).**
   The round-1 revision's second regex —
   `re.match(r'mysql\+pymysql://([^:***@]*)@([^:]+):(\d+)/(.+)', url)` —
   has only 4 capture groups unpacked into 5 variables
   (`user, pw, host, port, db = m.groups()`), and its first group's
   character class `[^:***@]*` cannot span the mandatory `user:pass@`
   colon that always exists in the converted URL (SQLAlchemy always emits
   `user:pass@host...`, with an empty-but-present password segment even
   when the password is blank). The regex therefore **never matches**,
   `m` is `None`, and `m.groups()` raises `AttributeError` on every run —
   meaning the concurrency guard introduced to fix round-1's gap #2 would
   have crashed the job on every single invocation, before ever reaching
   `GET_LOCK`. **Fixed:** the job no longer re-parses the already-converted
   URL string with a second regex. It reuses the same structured dict
   (`user`, `pass`, `host`, `port`, `db`) produced once during DSN
   conversion, entirely in-memory within a single `main()` invocation —
   `parse_dsn()` returns the dict once, and both the `alembic.ini` URL
   string (via `sqlalchemy_url(fields)`) and the `pymysql.connect(...)`
   kwargs for the locking step are derived directly from that same
   in-memory dict within the same process. (An earlier draft of this fix
   considered writing the dict to an intermediate file between separate
   steps; the committed design does not do this — everything happens in
   one Python process invocation per stream, which is simpler and removes
   any read/write-back surface entirely.) Single source of truth, no second
   regex, no
   possibility of the two parses disagreeing. See revised "DSN conversion"
   and "Locking" below.
3. **`mysqlclient` vs `PyMySQL` — internally inconsistent, misleading
   rationale (confirmed, non-blocking but must be corrected in the doc).**
   Round 1's revision converted to `mysql+pymysql://` URLs, which means
   alembic's actual DBAPI driver is PyMySQL — `mysqlclient` (and its apt
   build deps `default-libmysqlclient-dev build-essential pkg-config`) is
   installed but never used by this job. The stated rationale ("keeps a
   mysqlclient build issue from breaking the locking step") was **incorrect
   reasoning** — the locking step never used mysqlclient either; it uses
   `pymysql.connect()` directly. **Fixed:** this job installs and uses
   PyMySQL exclusively. The `mysqlclient`/apt build-dependency installation
   is dropped for this new job (it is still required, and unchanged, for
   the existing Dockerfile-based local/sandbox alembic paths — not touched
   by this change).

Additionally, round 2 flagged that `filters: branches: only:` (as opposed
to the `or:` combinator, which IS proven elsewhere in this file) has no
prior precedent in this specific `config_work.yml`, even though it is
standard, long-established CircleCI syntax generally. This is accepted as
a residual, non-blocking verification item — it is exactly why Verification
plan items 3-4 (`circleci config validate` + a live branch-guard test)
are mandatory gates before merge, not optional nice-to-haves.

### Round 3 findings and resolution

Round 3 re-verified the round-2 `migrate.py` rewrite line by line
(regex trace on both with-query and without-query DSN forms, dict-key
survival through `pop`/assignment, control flow on the `sys.exit(1)` error
path, GET_LOCK parameter binding). Six of eight checked items were
confirmed correct as written. One blocking defect was found:

1. **`re.sub()` replacement-string backslash reinterpretation (confirmed,
   blocking, NEW defect surfaced only at round 3).**
   `re.sub(r"^sqlalchemy\.url.*$", f"sqlalchemy.url = {url}", ini_content,
   flags=re.MULTILINE)` passes the replacement as a plain string built from
   an f-string. `re.sub`'s **replacement-string** argument specially
   interprets backslash sequences (`\1`, `\g<name>`, and raises
   `re.error: bad escape` on a bare trailing `\`). Since `url` embeds the
   production database password — untrusted, externally-controlled secret
   data as far as this script's own reasoning is concerned — a password
   containing so much as a single literal backslash would crash the script
   mid-`alembic.ini`-write, before the migration lock is ever acquired.
   **Fixed:** the replacement is now passed as a callable
   (`lambda _m: f"sqlalchemy.url = {url}"`) instead of a string, which
   `re.sub` inserts literally with no backslash reinterpretation. See the
   updated code block above.

Round 3 also noted one **non-blocking, already-safe** edge case worth
recording rather than silently accepting: `DSN_RE`'s password group
(`[^@]*`) stops at the first `@`, so if the production password itself
contains a literal `@`, the overall regex fails to match at all (because
the required literal `@tcp(` can then never be reached) — `parse_dsn`
correctly falls into its existing `if not m: ... sys.exit(1)` branch with a
clear error message, rather than silently mis-parsing. This is a
fail-loud, not fail-silent, edge case, so it is not treated as a blocker,
but the actual production DSN's password should be confirmed not to
contain `@` before first production run (see Verification plan item 2,
extended).

With four independent adversarial review rounds now completed — rounds 1
through 3 each surfacing a genuine blocking defect in the prior round's
own fix, round 4 finding zero new code defects but one documentation
inconsistency (a stale "JSON file" description in the round-2 write-up
that didn't match the final in-memory implementation, corrected above) —
this design has met the `design-first-with-review-loops` minimum bar with
significant margin. One residual non-blocking note from round 4:
`sqlalchemy_url()`'s query-string builder does not URL-encode parameter
values (`urllib.parse.quote`); low risk given the current `tls`-only
allowlist, but worth hardening if the allowlist grows. A final PR-review
loop (separate, min 3 rounds, on the actual committed diff rather than
this design doc) still applies once implementation begins.

## Critical safety finding: branch scoping

**The existing `build-approval` job has no branch filter.** Under
`path-filtering`, `run-bin-dbscheme-manager` flips to `true` for **any**
branch (including open PR branches) that touches `bin-dbscheme-manager/*`
relative to `main`. Today that's low-risk: worst case, someone approves a
build of an untested Docker image. Once `build-approval` gates a live
`alembic upgrade head` against production, the same lack of branch scoping
becomes a severe hazard: **approving the button on an open, unmerged PR
branch would apply that branch's unreviewed migration files straight to
production**, bypassing code review entirely.

**Resolution (round 1 finding: no other workflow in this file uses `when:
{and: [...], equal: [...]}` — that combinator is asserted by CircleCI 2.1
docs but untested in this codebase). To avoid relying on unverified syntax
for a production-safety-critical gate, use the pattern this codebase
already demonstrably uses elsewhere** (`or: [...]` is proven working at
`config_work.yml` lines 762-799 for the `voip-*-proxy` workflows) combined
with job-level branch `filters`, which is the CircleCI-documented and
widely-used mechanism for restricting a job to a branch:

```yaml
bin-dbscheme-manager:
  when: << pipeline.parameters.run-bin-dbscheme-manager >>
  jobs:
    - build-approval:
        type: approval
        filters:
          branches:
            only: main
    - bin-dbscheme-manager-migrate:
        <<: *context_production
        filters:
          branches:
            only: main
        requires:
          - build-approval
```

`filters.branches.only` is CircleCI's standard, long-established job
filtering mechanism (distinct from the newer pipeline-level `when:` logic
statements) and needs no new combinator syntax. **This must still be
validated with `circleci config validate` locally before merge**, and
additionally verified by pushing this PR branch itself and confirming the
`bin-dbscheme-manager` workflow's `build-approval`/`migrate` jobs do NOT
appear as runnable on the PR branch (only the pass-through `when:` /
path-filter check should run).

Defense in depth is kept inside the job as a second, independent layer:

```yaml
- run:
    name: Guard - main branch only
    command: |
      if [ "$CIRCLE_BRANCH" != "main" ]; then
        echo "REFUSING: this job must only run on main (got: $CIRCLE_BRANCH)"
        exit 1
      fi
```

This is not purely redundant: per round-1 review, its actual value is as a
backstop against a `filters` misconfiguration (e.g. a future edit that
removes or typos the filter), which is exactly the class of syntax risk
flagged above. Keep it.

## Design

### Workflow change (`.circleci/config_work.yml`)

Remove `bin-dbscheme-manager-build`. Add `bin-dbscheme-manager-migrate`
(see filters above for the workflow-level change).

New job definition. A single shared Python helper module
(`bin-dbscheme-manager/ci/migrate.py`, committed as part of this PR — not
inlined as YAML heredocs, to keep the logic testable and reviewable as real
code rather than embedded shell/YAML) is invoked once per stream:

```yaml
  # bin-dbscheme-manager
  bin-dbscheme-manager-migrate:
    docker: *gcp_image
    resource_class: small
    steps:
      - checkout
      - run:
          name: Guard - main branch only
          command: |
            if [ "$CIRCLE_BRANCH" != "main" ]; then
              echo "REFUSING: this job must only run on main (got: $CIRCLE_BRANCH)"
              exit 1
            fi
      - run:
          name: Install PyMySQL + alembic
          command: |
            pip install --no-cache-dir "sqlalchemy<2.0" alembic PyMySQL
      - run:
          name: Upgrade bin_manager (voipbin core DB)
          no_output_timeout: 10m
          command: |
            python3 bin-dbscheme-manager/ci/migrate.py \
              --stream bin-manager \
              --dsn-env CC_DATABASE_DSN \
              --lock-name voipbin_alembic_bin_manager
      - run:
          name: Upgrade asterisk config DB
          no_output_timeout: 10m
          command: |
            python3 bin-dbscheme-manager/ci/migrate.py \
              --stream asterisk_config \
              --dsn-env CC_DATABASE_DSN_ASTERISK \
              --lock-name voipbin_alembic_asterisk_config
```

`bin-dbscheme-manager/ci/migrate.py` (new file, committed to the repo so it
is unit-testable and reviewable, instead of living only as an inline YAML
heredoc):

```python
#!/usr/bin/env python3
"""Convert a go-sql-driver/mysql DSN to a SQLAlchemy URL, write alembic.ini,
acquire a MySQL named lock, run alembic upgrade head (dry-run then real),
and release the lock. Used only by CI against production; never by AI
agents (see bin-dbscheme-manager/CLAUDE.md's alembic-upgrade prohibition —
this script is the one sanctioned exception, gated behind CircleCI human
approval + main-branch-only filters, not something an agent invokes)."""
import argparse
import os
import re
import subprocess
import sys

import pymysql

# go-sql-driver/mysql DSN grammar: user:pass@tcp(host:port)/dbname?k=v&...
DSN_RE = re.compile(
    r'^(?P<user>[^:]+):(?P<pass>[^@]*)@tcp\((?P<host>[^:]+):(?P<port>\d+)\)'
    r'/(?P<db>[^?]+)(?:\?(?P<query>.*))?$'
)

# Only forward query params that materially affect the connection's
# security/behavior. Silently dropping the whole query string (round-2
# finding) risks losing e.g. tls=... and connecting unencrypted.
FORWARDED_PARAMS = {"tls"}


def parse_dsn(dsn: str) -> dict:
    m = DSN_RE.match(dsn)
    if not m:
        print("ERROR: could not parse DSN format (expected "
              "user:pass@tcp(host:port)/db[?params])", file=sys.stderr)
        sys.exit(1)
    fields = m.groupdict()
    query = fields.pop("query") or ""
    params = dict(p.split("=", 1) for p in query.split("&") if "=" in p)
    fields["params"] = {k: v for k, v in params.items() if k in FORWARDED_PARAMS}
    dropped = set(params) - FORWARDED_PARAMS
    if dropped:
        print(f"NOTE: DSN query params not forwarded (not in allowlist): "
              f"{sorted(dropped)}", file=sys.stderr)
    return fields


def sqlalchemy_url(fields: dict) -> str:
    qs = "&".join(f"{k}={v}" for k, v in fields["params"].items())
    base = (f"mysql+pymysql://{fields['user']}:{fields['pass']}@"
            f"{fields['host']}:{fields['port']}/{fields['db']}")
    return f"{base}?{qs}" if qs else base


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--stream", required=True, choices=["bin-manager", "asterisk_config"])
    ap.add_argument("--dsn-env", required=True)
    ap.add_argument("--lock-name", required=True)
    args = ap.parse_args()

    raw_dsn = os.environ.get(args.dsn_env)
    if not raw_dsn:
        print(f"ERROR: {args.dsn_env} not set", file=sys.stderr)
        sys.exit(1)

    fields = parse_dsn(raw_dsn)  # single parse, single source of truth
    url = sqlalchemy_url(fields)  # for alembic.ini

    stream_dir = os.path.join(os.path.dirname(__file__), "..", args.stream)
    ini_path = os.path.join(stream_dir, "alembic.ini")
    with open(os.path.join(stream_dir, "alembic.ini.sample")) as f:
        ini_content = f.read()
    with open(ini_path, "w") as f:
        # NOTE: `url` (built from the production password) is passed via a
        # replacement FUNCTION, not a replacement STRING. re.sub() treats a
        # string replacement's backslashes specially (\1, \g<name>, a bare
        # trailing \ raises re.error). Since `url` embeds untrusted secret
        # data, a password containing a literal backslash would crash this
        # script mid-write if the replacement were a plain f-string (found
        # in round 3 review — see Design review history). A lambda forces
        # the replacement to be inserted literally, with no backslash
        # reinterpretation.
        f.write(re.sub(r"^sqlalchemy\.url.*$",
                        lambda _m: f"sqlalchemy.url = {url}",
                        ini_content, flags=re.MULTILINE))

    # Discrete fields (not a second URL parse) drive the DB connection used
    # only for the named lock — avoids the round-2 regex-mismatch bug where
    # re-parsing the already-converted URL string could disagree with the
    # first parse.
    conn = pymysql.connect(
        host=fields["host"], port=int(fields["port"]),
        user=fields["user"], password=fields["pass"], database=fields["db"],
    )
    cur = conn.cursor()
    cur.execute("SELECT GET_LOCK(%s, 30)", (args.lock_name,))
    if cur.fetchone()[0] != 1:
        print(f"ERROR: could not acquire lock '{args.lock_name}' "
              f"(another run in progress?)", file=sys.stderr)
        sys.exit(1)
    try:
        env = {**os.environ}  # subprocess inherits; alembic.ini has no secret in argv
        for cmd in (
            ["alembic", "-c", ini_path, "current", "--verbose"],
            ["alembic", "-c", ini_path, "upgrade", "head", "--sql"],
            ["alembic", "-c", ini_path, "upgrade", "head"],
            ["alembic", "-c", ini_path, "current", "--verbose"],
        ):
            result = subprocess.run(cmd, cwd=stream_dir, env=env,
                                     capture_output=True, text=True)
            # Redact defense-in-depth: even though alembic.ini's DSN is not
            # expected to appear in --verbose output (revision IDs only),
            # scrub the password value from anything captured before
            # printing, in case of an unexpected error path that echoes
            # config content.
            out = result.stdout.replace(fields["pass"], "***") if fields["pass"] else result.stdout
            err = result.stderr.replace(fields["pass"], "***") if fields["pass"] else result.stderr
            print(out)
            print(err, file=sys.stderr)
            if result.returncode != 0:
                sys.exit(result.returncode)
    finally:
        cur.execute("SELECT RELEASE_LOCK(%s)", (args.lock_name,))
        conn.close()


if __name__ == "__main__":
    main()
```

Notes:
- **Single parse, single source of truth.** Round 2's fatal bug was a
  second, independent regex re-parsing the already-converted URL for the
  locking step, which didn't match. This revision parses the Go DSN exactly
  once into a `dict`, and both the `alembic.ini` URL string and the
  `pymysql.connect()` kwargs are derived from that same dict — there is no
  second parse to disagree with the first.
- **Query parameters are not silently dropped.** Anything in the Go DSN's
  query string is inspected; only an explicit allowlist (`tls`, extendable
  later if another param proves relevant) is forwarded into the SQLAlchemy
  URL. Everything else is logged as dropped (visible in CI output) rather
  than silently discarded. This must be revisited once the actual
  production DSN's real query parameters are known (see Verification plan
  item 2) — if `tls` turns out not to be present but some other Go-specific
  param is load-bearing (e.g. `parseTime`, which is Go-driver-specific and
  has no PyMySQL equivalent — it's fine to drop since alembic doesn't parse
  time values the way the Go driver's struct scanning does), the allowlist
  is a single set literal to update.
- **PyMySQL only, no `mysqlclient`.** The prior draft's claim that
  installing `mysqlclient` isolated the locking step was backwards —
  `pymysql.connect()` never depended on `mysqlclient` in the first place.
  This revision drops the `mysqlclient`/apt build toolchain entirely from
  this job; alembic runs against a `mysql+pymysql://` URL end to end. The
  Dockerfile-based local/sandbox path (which does use `mysqlclient`) is
  unrelated and untouched.
- Sequential execution (`bin-manager` then `asterisk_config`) preserved as
  two separate CircleCI steps, each running the same script with a
  different `--stream`/`--dsn-env`/`--lock-name`. If `bin-manager` upgrade
  fails, the step fails and CircleCI does not proceed to the next step
  (avoids partial-and-inconsistent apply across streams). Separate lock
  names mean the two streams cannot false-contend with each other, only
  with a second concurrent run of the *same* stream.
- The `--sql` dry-run pass happens immediately before the real
  `upgrade head`, both inside the same lock acquisition, so the preview
  reflects exactly the state the real apply will see (no TOCTOU gap).
- The password is redacted from captured subprocess output as defense in
  depth, on top of the primary claim (still to be verified — see
  Verification plan item 6) that `alembic ... --verbose` does not print
  `sqlalchemy.url` in normal or error output.
- The GET_LOCK/RELEASE_LOCK pair uses a MySQL **named lock**, session-scoped:
  if the CircleCI job dies mid-migration (OOM, timeout, cancel), MySQL
  releases the lock automatically when that connection drops — no
  permanently stuck lock from an aborted CI job.
- No CircleCI `docker_layer_caching` concern anymore — this job builds
  nothing, so this design also incidentally fixes the original CI/CD speed
  complaint by removing the slowest part of the old workflow entirely.

### Failure handling

- If `alembic upgrade head` fails partway through a single migration file's
  DDL (MySQL DDL is largely non-transactional), `alembic_version` is left
  at the last successfully committed revision. A rerun of this job resumes
  correctly from that point — this is pre-existing alembic/MySQL behavior,
  unchanged by this design. Already covered operationally by the "Wrong
  revision head" and "Locked table during migration" sections of
  `docs/operations.md`.
- No automatic retry/`backoffLimit` is configured — a failed migration must
  surface immediately and loudly, not silently retry against a
  partially-migrated production schema. A human follows the existing
  runbook in `docs/operations.md`.
- The `GET_LOCK` guard means a retried/rerun job that overlaps with a still
  running prior attempt fails fast with a clear "another run in progress"
  error rather than racing.

### What's explicitly deferred (follow-up tickets, not blocking this one)

1. **Network reachability**: confirm CircleCI's `cimg/gcp` executor can
   reach the production DB host at all (VPC peering / IP allowlist). Must
   be verified before merge — see Verification plan.
2. **Large-table ALTER safety** (`pt-online-schema-change` / `gh-ost`): not
   introduced in this change. Migration authors remain responsible for safe
   DDL patterns on large hot tables, same as before.
3. **Staging tier**: `docs/operations.md` describes a staging step before
   production for manual runs. VoIPBin does not currently have a CI-wired
   staging DB target. Out of scope for this ticket; flagged as a known gap.
4. **Pre-flight revision assertion** (e.g. "current head must be exactly
   X before this upgrade may proceed"): the `--sql` dry-run + `current
   --verbose` before/after gives a human-auditable log, but does not
   automatically abort on an unexpected starting revision. Considered for
   this round but deferred — the two-role review loop should confirm this
   is an acceptable interim risk vs. a hard blocker.

## Verification plan (must pass before merge)

1. **Local dry run**: run `bin-dbscheme-manager/ci/migrate.py` against a
   local throwaway MySQL instance to confirm DSN parsing, alembic.ini
   generation, and the GET_LOCK/RELEASE_LOCK cycle all work end-to-end,
   including a deliberate concurrent-invocation test (run the script twice
   in parallel against the same lock name, confirm the second run fails
   fast with the "another run in progress" message instead of racing).
2. **DSN format confirmation**: pchero confirms the exact stored format of
   `CC_DATABASE_DSN` / `CC_DATABASE_DSN_ASTERISK` in the CircleCI
   production context, specifically whether the DSN carries any query
   parameters (e.g. `tls=`) that must be added to `FORWARDED_PARAMS` in
   `migrate.py` before this is trusted against production. Do not merge on
   an unverified guess about query parameters.
3. **Unit tests for `migrate.py`**: add `bin-dbscheme-manager/ci/test_migrate.py`
   covering `parse_dsn`/`sqlalchemy_url` against representative DSN strings
   (with and without query params, with an empty password) as part of this
   PR — this is real committed code now, not YAML-embedded shell, so it
   must carry normal test coverage per this repo's verification workflow.
4. **`circleci config validate`**: run locally against the modified
   `config_work.yml` to catch YAML/logic-statement errors before push.
5. **Branch-guard live test**: push this PR branch and confirm the
   `bin-dbscheme-manager` workflow's `build-approval` job does not become
   runnable (filters correctly exclude the branch), before merging to main.
6. **Network reachability check**: from a scratch CircleCI job on the
   `production` context, confirm TCP reachability to the DB host before
   trusting the full job to run un-rehearsed against prod.
7. **Alembic verbose-output audit**: confirm `alembic current --verbose`
   and `alembic upgrade head --sql` do not print the DSN/password on
   success or failure paths, in a local test against a dummy DB. The
   password-redaction in `migrate.py` is defense in depth, not a
   substitute for confirming this.
8. Design review loop (this skill, min 2 rounds — satisfied: round 1 and
   round 2 both completed with findings resolved) + PR review loop (min 3
   rounds) per `design-first-with-review-loops`.

## Rollback plan for THIS change

If the new job is merged and then found to be broken (e.g. DSN conversion
regex wrong for actual production format, unreachable network): the fix is
a follow-up PR reverting to the approval-gate-with-no-op state (restoring
`bin-dbscheme-manager-build`) or fixing forward. Reverting also restores the
seed-image path; per the verified Non-goals finding above, no other
consumer depends on it besides the (independent, source-clone-based)
sandbox, so a temporary revert carries no other blast radius.

## PR review history (implementation-level, after design approval)

Once design review reached round-5 APPROVED, implementation was committed
as PR #1086 and put through a separate PR review loop (per
`design-first-with-review-loops`, minimum 3 rounds).

**Round 1** (code-correctness focus): APPROVED with zero blocking defects.
Confirmed the committed `bin-dbscheme-manager/ci/migrate.py` matches the
design doc's intent with one positive drift (a `row is None` guard on
`cur.fetchone()` that the design doc's literal snippet lacked), confirmed
`test_migrate.py` imports and exercises the real module (not a duplicated
copy), confirmed the CircleCI YAML diff is syntactically/semantically
sound with `bin-dbscheme-manager-build` fully removed and no dangling
references, and confirmed the generated `alembic.ini` files are already
covered by an existing `bin-dbscheme-manager/.gitignore` entry (no new
secret-leak vector introduced).

**Round 2** (operational/failure-mode focus, deliberately different angle
from round 1): CHANGES_REQUESTED. Found a genuine operational risk not
covered by rounds 1-5 of design review or round 1 of PR review: the
original `migrate.py` used `subprocess.run(..., capture_output=True)` for
each alembic invocation, which buffers ALL output until the subprocess
exits before printing anything. For the real `alembic upgrade head` step
-- the one call that can legitimately run long on a production migration
-- this meant (a) CircleCI's `no_output_timeout: 10m` could false-positive
on a slow-but-healthy migration, since no bytes reach stdout to reset the
timeout clock until the whole command finishes, and (b) if the step
genuinely hung or was killed, the operator would have **zero** log output
to diagnose what was happening -- exactly the worst failure mode for a
job's first-ever run against production. **Fixed:** replaced the buffered
`subprocess.run(capture_output=True)` calls with a new `run_streamed()`
helper using `subprocess.Popen` + line-by-line forwarding (with the same
password redaction applied per-line), so `no_output_timeout` reflects real
progress and a hang leaves a partial, diagnosable log trail. Added
`TestRunStreamed` unit tests (streaming + exit-code propagation +
per-line redaction, verified passing). Also updated
`bin-dbscheme-manager/docs/operations.md`: added a "CI-driven production
migration" section documenting what the approval button now does, the
rerun-safety story (relies on alembic/MySQL's natural idempotency, not
explicit partial-failure detection -- stated plainly, not oversold), and
when to fall back to the manual VPN procedure; and a note on the
`no_output_timeout` risk under the existing "Locked table during
migration" entry, since this job has not yet run against a real production
table and 10 minutes is an untuned guess.

A minimum of one further PR review round is required before this can be
considered ready to merge (design-first-with-review-loops: min 3 PR-review
rounds, continuing until 2 consecutive APPROVED).

## Outcome: CI-driven migration abandoned after network-reachability test (2026-07-11)

After the PR review loop closed (7 rounds, 2 consecutive APPROVED), the
Verification plan's last outstanding item -- confirming CircleCI's
production-context executor can reach the production DB host -- was
tested directly, per pchero's request, before merge:

1. A throwaway branch (`VOIP-1246-dsn-test`) added a read-only probe job
   (`bin-dbscheme-manager/ci/dsn_probe.py`, reusing the already-reviewed
   `parse_dsn()` from `migrate.py`) that only ran `SHOW TABLES`, gated
   behind its own approval scoped strictly to that branch (never touching
   the `main`-only filters on the real migrate job).
2. Result: `parse_dsn()` correctly parsed the real production DSN (host,
   port, db, user; zero query params present, confirming the
   `FORWARDED_PARAMS` allowlist design was moot for the actual DSN format
   in use). But the `pymysql.connect()` call **timed out** --
   `pymysql.err.OperationalError: (2003, "Can't connect to MySQL server
   ... (timed out)")`. Confirmed by pchero: the production DB is
   intentionally not exposed to the public internet, and CircleCI's
   standard executors are outside VoIPBin's VPC.

This is a hard blocker that no further code review could have caught --
it required an actual network-level test from inside CircleCI against the
real production host. **Decision (pchero, 2026-07-11): abandon CI-driven
migration application. Revert to the original manual, VPN-only, human-only
procedure** (this was "option B" from the original scoping discussion,
now adopted after the more ambitious "option A" proved infeasible at the
infrastructure level rather than the code level).

**What was kept from this PR:**
- Removal of `bin-dbscheme-manager-build` (the seed-image build job) --
  still valid and unaffected by the network finding; confirmed unused by
  any consumer including `sandbox`.
- The `build-approval` gate is kept as a documented no-op checkpoint
  (renamed `migration-applied-checkpoint`) for recording "migration was
  manually applied and verified over VPN" in CircleCI history, per
  pchero's direction, rather than removed outright.

**What was removed:**
- `bin-dbscheme-manager-migrate` job and `bin-dbscheme-manager/ci/migrate.py`
  / `test_migrate.py` -- the actual CI-driven alembic invocation. Not
  reachable from CircleCI, so cannot run.
- `bin-dbscheme-manager/ci/dsn_probe.py` and its temporary CircleCI job --
  deleted along with the `VOIP-1246-dsn-test` branch once its one purpose
  (confirming the network-reachability blocker) was served.

`bin-dbscheme-manager/docs/operations.md` was updated to document this
outcome plainly (a new "Explored and rejected: CI-driven migration"
section) so a future attempt starts by testing network reachability
first, not last -- avoiding repeating 12 rounds of code-level review on a
design that infrastructure alone rules out.

All of this design/implementation work remains in git history
(this design doc, the deleted `migrate.py`/`test_migrate.py`, and the PR
review history above) in case CircleCI is ever given VPC-internal network
access to production (e.g. via a self-hosted runner), at which point the
already-reviewed `migrate.py` logic could be resurrected largely as-is.
