# bin-dbscheme-manager: PR-branch migration validation job

Status: DRAFT (design review round 3 response, v4)
Author: Hermes (CPO)
Date: 2026-07-18

## 1. Problem

`.circleci/config_work.yml`'s `bin-dbscheme-manager` workflow has exactly one
job, `migration-applied-checkpoint`, and it is hard-restricted to `main`:

```yaml
bin-dbscheme-manager:
  when: << pipeline.parameters.run-bin-dbscheme-manager >>
  jobs:
    - migration-applied-checkpoint:
        type: approval
        filters:
          branches:
            only: main
```

Consequence: on any PR branch that adds/edits an Alembic migration under
`bin-dbscheme-manager/`, `run-bin-dbscheme-manager` still flips to `true`
(path-filtering has no branch restriction), but the workflow's only job is
filtered out by `only: main`, so **zero CircleCI checks appear for
dbscheme-manager changes on PR branches.** Confirmed empirically on PR #1112
(`gh pr checks 1112` shows no dbscheme-manager job at all, even though the
diff adds `bin-dbscheme-manager/bin-manager/main/versions/1a1f28d6842c_....py`).

This is not a bug in the `main`-only restriction itself -- that restriction
is intentional and correct (see §2) -- it is a **gap**: PR branches get no
automated signal at all about whether the migration file they're adding is
even structurally valid (parses, chains onto exactly one existing head, no
duplicate revision IDs). Today that check is 100% manual (a human reads the
diff and eyeballs `down_revision`).

## 2. Non-goals / what does NOT change

- `migration-applied-checkpoint` stays exactly as-is: `type: approval`,
  `filters: branches: only: main`, no automated action. Per VOIP-1246
  (documented in `config_work.yml` lines 370-380 and
  `bin-dbscheme-manager/docs/operations.md`), VoIPBin's production
  `bin_manager`/`asterisk` MySQL databases are not reachable from CircleCI's
  executors -- confirmed by a network probe that timed out at the socket
  level. Actually applying a migration to staging/production remains a
  human-only, VPN-only, manual procedure. This design does **not** attempt
  to make CI apply migrations -- that is out of scope and was already
  explored/rejected.
- No new job talks to the real `bin_manager`/`asterisk` databases, staging or
  production. The new job (§3) never opens a network connection to a MySQL
  host.
- No change to any other service's workflow.

## 3. Design

Add a second, **unapproved, always-runs-when-relevant** job to the
`bin-dbscheme-manager` workflow, scoped to ALL branches (no `filters:`
block), gated by the same `run-bin-dbscheme-manager` pipeline parameter that
already exists:

**Open risk, spike-test before merge:** there is **no existing precedent in
this repo** for mixing a filtered and an unfiltered job under the same
`when:`-gated workflow name -- `filters:` appears exactly once in the entire
`config_work.yml` (on `migration-applied-checkpoint` itself, lines 386-388).
The YAML shape below is the intended structure, not a pattern already proven
to work elsewhere in this codebase. Before merging, push this branch itself
(which touches `bin-dbscheme-manager/**`) and confirm via `gh pr checks` that
BOTH `migration-lint` (unfiltered) and `migration-applied-checkpoint`
(main-only) schedule with the behavior each is supposed to have -- i.e.
`migration-lint` runs immediately on this PR branch, `migration-applied-checkpoint`
does not appear at all (branch is not `main`).

```yaml
bin-dbscheme-manager:
  when: << pipeline.parameters.run-bin-dbscheme-manager >>
  jobs:
    # Bare-string form (no colon/mapping), matching the precedent used
    # elsewhere in this file for zero-parameter job invocations (e.g. the
    # `- checkout` step idiom). A `- migration-lint:` colon form with only
    # a comment underneath parses as `{'migration-lint': None}` in plain
    # YAML, which is syntactically valid but untested against CircleCI's
    # own job-invocation schema in this repo -- the bare-string form avoids
    # that risk entirely since migration-lint needs zero invocation
    # parameters (no requires/filters/context).
    - migration-lint
    - migration-applied-checkpoint:
        type: approval
        filters:
          branches:
            only: main
```

### 3.1 What `migration-lint` actually checks

The user's ask was explicitly for a **lightweight** job -- not a full
Docker-build migration dry-run against a throwaway MySQL (that already
exists as the Dockerfile's build-stage behavior, is comparatively heavy
[installs mariadb-server, boots a real mysqld], and is not wired into CI at
all today; wiring it in is a bigger, separate decision the user did not ask
for here).

Alembic's `heads` and `history` commands are **metadata-only** operations:
they parse every revision file in `versions/` and walk the `down_revision`
graph to build/print the revision DAG. They do **not** open a database
connection. This gives a genuinely lightweight check that is meaningfully
stronger than "the file exists" (which is trivially true any time a diff
touches the path) while still requiring no DB, no Docker, no mysql client:

1. **Python syntax/import validity of every migration file** -- `alembic
   heads` fails loudly if any revision file in `versions/` fails to import
   (syntax error, missing `revision`/`down_revision` module-level vars).
2. **Exactly one head per stream** -- `alembic heads` prints every "head"
   revision (a revision with no children). More than one head means either
   a duplicate/misassigned `down_revision` or two migrations were authored
   in parallel against the same base without one rebasing onto the other.
   Both streams (`bin-manager/` for `voipbin` DB, `asterisk_config/` for
   `asterisk` DB) are independent and must each resolve to exactly one head.
3. **No unresolvable revision references** -- `alembic history` walks the
   full chain from base to head(s); a broken `down_revision` (referencing a
   revision ID that doesn't exist) raises `alembic.util.exc.CommandError`
   and fails the job.

Concretely, the job:

```yaml
migration-lint:
  docker:
    - image: python:3.11-slim
  steps:
    - checkout
    - run:
        name: Install alembic (metadata-only, no DB driver needed)
        no_output_timeout: 2m
        command: pip install --no-cache-dir "sqlalchemy<2.0" alembic
    - run:
        name: Validate bin-manager migration chain (voipbin DB)
        command: |
          cd bin-dbscheme-manager/bin-manager
          cp alembic.ini.sample alembic.ini
          # alembic.ini requires a sqlalchemy.url key to parse, but `heads`/
          # `history` never open it -- a syntactically valid but unreachable
          # placeholder is sufficient and deliberately used instead of a
          # real credential.
          HEAD_COUNT=$(alembic -c alembic.ini heads | grep -c '(head)')
          echo "Detected $HEAD_COUNT head(s):"
          alembic -c alembic.ini heads --verbose
          if [ "$HEAD_COUNT" -ne 1 ]; then
            echo "ERROR: expected exactly 1 head, found $HEAD_COUNT. Multiple heads means two migrations were authored against the same down_revision without rebasing -- see bin-dbscheme-manager/docs/migrations.md."
            exit 1
          fi
          alembic -c alembic.ini history --verbose > /dev/null
    - run:
        name: Validate asterisk_config migration chain (asterisk DB)
        command: |
          cd bin-dbscheme-manager/asterisk_config
          cp alembic.ini.sample alembic.ini
          HEAD_COUNT=$(alembic -c alembic.ini heads | grep -c '(head)')
          echo "Detected $HEAD_COUNT head(s):"
          alembic -c alembic.ini heads --verbose
          if [ "$HEAD_COUNT" -ne 1 ]; then
            echo "ERROR: expected exactly 1 head, found $HEAD_COUNT."
            exit 1
          fi
          alembic -c alembic.ini history --verbose > /dev/null
```

(Exact `alembic.ini.sample` placeholder DSN and CircleCI YAML anchors to be
finalized against the real file during implementation; the above is the
structural shape, not final byte-for-byte YAML.)

### 3.2 Why `alembic.ini.sample` (not a real production DSN)

`alembic.ini.sample` already exists in both `bin-manager/` and
`asterisk_config/` as the checked-in template (`cp alembic.ini.sample
alembic.ini` is the exact same bootstrap step the Dockerfile's build stage
already performs). It contains a placeholder `sqlalchemy.url`. Since
`heads`/`history` never connect, the placeholder is never dereferenced --
using it (rather than any `CC_*` production secret) means this job needs
**zero secrets, zero `context: production` attachment**, which keeps it
safely runnable on any branch including forks/untrusted PRs (not currently a
concern for this org's access model, but good hygiene regardless).

### 3.3 Failure mode / what a red `migration-lint` looks like to the author

- Two migrations both set `down_revision = 'X'` (didn't rebase after
  pulling `main`) -> `HEAD_COUNT` = 2 -> job fails with the multiple-heads
  message above, pointing at `docs/migrations.md`.
- A migration file has a Python syntax error -> `alembic heads` itself
  raises an import error -> job fails with alembic's native traceback.
- A migration's `down_revision` points at a revision ID that doesn't exist
  in the tree (typo, or the base revision was deleted) -> `alembic history`
  raises `CommandError: Can't locate revision identified by '...'` -> job
  fails.

None of these require a human to have VPN access or a real DB credential to
diagnose -- the CircleCI job output alone tells the author exactly what to
fix, without waiting for the (necessarily main-only, human-gated) approval
checkpoint to even become visible.

## 4. Design review checklist (for the review loop)

- [x] Confirm `alembic heads`/`alembic history` genuinely require no DB
      connection. **Verified two ways, round 1 of the design-review loop:**
      (a) empirically -- both commands run instantly (exit 0) against
      `alembic.ini.sample`'s placeholder DSN (`mysql://root@localhost/bin_manager`,
      no mysqld running) in this worktree; (b) at the source level -- traced
      the installed alembic package's `command.py`: `heads()` (lines 551-575)
      only calls `ScriptDirectory.from_config(config)` + `get_heads()`, pure
      filesystem/AST parsing, no `EnvironmentContext` ever instantiated.
      `history()` (lines 482-548) only instantiates `EnvironmentContext` (which
      is what would trigger a DB connect) in the `base == "current" or head ==
      "current" or environment` branch -- `environment` requires either the
      `revision_environment` alembic.ini option (not set here) or
      `--indicate-current`/`-r current:...` on the command line (not used by
      this design's plain `alembic history --verbose` invocation). The design's
      invocation takes the `else` branch -> `_display_history` -> pure
      `ScriptDirectory.walk_revisions()`, no DB connect. Confirmed no gap
      between the plain and `--verbose` invocations.
- [x] Confirm the exact CircleCI 2.1 YAML syntax for a job with **no**
      `filters:` block sitting alongside a sibling job that DOES have one is
      genuinely unverified in-repo (not a false alarm) -- **confirmed round 1**:
      `filters:` appears exactly once in all of `config_work.yml`, at
      `migration-applied-checkpoint` itself (lines 386-388). No existing
      workflow in this repo mixes filtered/unfiltered siblings under one
      workflow name. This is a genuine open risk requiring the spike-test
      called out in §3 above -- not resolvable by static review alone.
  - [x] Verify whether `path-filtering`'s branch scope itself already
        constrains something. **Confirmed round 1**: `path-filtering/filter`
        (`config.yml:11-15`) has no branch restriction of its own -- it runs
        unconditionally in the `check-changes` setup workflow on every
        pipeline trigger, diffs against `base-revision: main`, and only sets
        `run-bin-dbscheme-manager` (and sibling parameters) as booleans based
        on the diffed paths. `migration-lint`'s only gate is
        `run-bin-dbscheme-manager` (`config_work.yml:66-68,381-388`), itself
        driven purely by whether the diff touches `bin-dbscheme-manager/**` --
        adding an unfiltered sibling job is purely additive within
        `config_work.yml` and cannot cause `migration-lint` to run on branches
        that don't touch this path.
- [x] Confirm the job does not require `context: production`. Confirmed by
      design (§3.2: zero secrets needed since `heads`/`history` never
      dereference the placeholder DSN) -- no other step in the job needs
      network/secret access either (pip install from PyPI, local filesystem
      parse only).
- [ ] Confirm this job is not accidentally gated behind
      `migration-applied-checkpoint`'s approval (it must NOT `require:` the
      approval job -- it should run independently and immediately).
- [ ] Confirm no existing job elsewhere in `config_work.yml` is named
      `migration-lint` (name collision check) -- **confirmed clean as of
      round 1** (`grep -n "migration-lint" .circleci/config_work.yml` returns
      no matches before this design's changes are applied), but re-verify at
      implementation time in case another change lands first.
- [ ] Confirm `bin-dbscheme-manager/docs/migrations.md` gets a short addition
      documenting this new automated check. **Target file decided (round 3):
      `docs/migrations.md`**, not `docs/operations.md` -- chosen because it
      is the file the job's own failure message already cross-references for
      the multiple-heads case (§3.1 job step, "see
      bin-dbscheme-manager/docs/migrations.md"). **Concrete required
      content** (per design-review round 2 -- this is not optional wording,
      the implementer must include all three points): (1) what
      `migration-lint` checks (Python syntax/import validity of every
      revision file, exactly-one-head-per-stream, no unresolvable
      `down_revision` references -- metadata-only, no DB connection); (2)
      that it runs on EVERY branch/PR touching `bin-dbscheme-manager/**`,
      independently of and without requiring
      `migration-applied-checkpoint`'s main-only manual approval; (3) where
      to look when it fails -- the CircleCI job's own log output names the
      specific problem (multiple heads, import error, or broken
      `down_revision` reference) and cites `docs/migrations.md` for the
      multiple-heads case specifically.
- [x] Verify multi-head detection actually fires as designed. **Verified
      round 1**: created two throwaway revision files both pointing at the
      same `down_revision` (the real current head) in `bin-manager/main/versions/`,
      confirmed `alembic heads` printed both as separate `(head)` lines and
      `grep -c '(head)'` correctly reported 2; removed the throwaway files and
      confirmed the count reverted to 1. Also verified `--verbose` mode does
      NOT double-count `(head)` per revision (each revision's `--verbose`
      block contains exactly one `Rev: <id> (head)` line, not two) -- the
      `grep -c '(head)'` logic in §3.1's job step is safe under `--verbose`
      too.

## 5. Implementation checklist

1. Add `migration-lint` job definition to `.circleci/config_work.yml`
   (verified YAML, see §3.1).
2. Add it to the `bin-dbscheme-manager` workflow's `jobs:` list, alongside
   (not replacing) `migration-applied-checkpoint`.
3. Local dry-run: `python3 -c "import yaml; yaml.safe_load(open('.circleci/config_work.yml')); print('OK')"`.
4. Local functional verification: manually run the exact commands from
   §3.1 against the current repo state (both streams) and confirm each
   reports exactly 1 head today (baseline, before this PR's own migration
   changes are considered -- this branch itself won't add a migration).
5. Doc addition: add a short subsection to `bin-dbscheme-manager/docs/migrations.md`
   (target file decided per §4 round-3 finding, not operations.md)
   containing the three concrete points specified in the §4 checklist item
   above (what it checks, that it's branch-independent of the manual
   approval, where to look on failure).
6. PR, push, and confirm via `gh pr checks` that BOTH jobs schedule with
   the intended behavior on this non-main branch (the §3 spike-test).
7. PR review loop (min 3 rounds).

## 6. Scope assessment (round 1 review response)

Confirmed right-sized as designed, with one wording correction (round 3):
the lightweight-only approach (no DB, no Docker) matches the CEO's explicit
constraint on *mechanism* precisely -- but the CEO's verbatim ask was to
check "마이그레이션 파일 존재 여부만" (only whether the migration file exists).
§3.1 deliberately goes beyond literal file-existence to structural/metadata
validity (syntax parseability, exactly-one-head-per-stream, resolvable
`down_revision` chain) -- reasoned there as a meaningful improvement over a
check that "is trivially true any time a diff touches the path." This is a
good-faith, reasoned scope expansion, not an exact literal-ask match; it is
flagged here explicitly for CEO awareness rather than asserted as precise
conformance. It is still firmly within the "lightweight, no DB, no Docker,
no approval gate" boundary the CEO set, and does not add anything beyond
metadata parsing.

A cross-stream revision-ID collision check (bin-manager vs asterisk_config
referencing each other) would be further, unwarranted overengineering
beyond even this already-expanded scope: Alembic revision IDs are already
stream-scoped by construction (separate `versions/` directories, separate
`script_location` in each stream's own `alembic.ini`) -- there is no code
path by which a `bin-manager` migration could reference an `asterisk_config`
revision ID or vice versa, so this is not a real failure mode the job needs
to guard against.

## 7. Changelog

- v1 (2026-07-18, round 0): initial draft.
- v2 (2026-07-18, round 1 response): incorporated design-review round 1
  findings -- moved the "no in-repo precedent for mixed filtered/unfiltered
  siblings" caveat into §3 itself (previously only in the checklist),
  checked off 4 of 6 §4 items with round-1 verification evidence (DB-connection
  claim independently re-derived from alembic source, CircleCI precedent
  search, path-filtering branch-scope confirmation, multi-head detection
  re-verified under `--verbose`), added §6 scope assessment, added a job-name
  collision checklist item.
- v3 (2026-07-18, round 2 response): incorporated design-review round 2
  findings (both non-blocking) -- added `no_output_timeout: 2m` to the
  shared `pip install` step in §3.1's job YAML (covers both streams'
  validation steps since they run sequentially in the same job/container
  after the single install step) so a slow/unreachable PyPI mirror fails
  fast rather than relying on CircleCI's default inactivity timeout;
  specified the exact three-point required content for the
  `docs/operations.md`/`docs/migrations.md` addition (previously deferred
  entirely to implementation time) in both §4's checklist item and §5's
  implementation-checklist item 5. Round 2 independently re-verified all of
  round 1's citations from scratch (line numbers, grep results, alembic
  source logic, `--verbose` head-count behavior for both streams) and found
  them all accurate.
- v4 (2026-07-18, round 3 response): incorporated design-review round 3
  findings (all 3 non-blocking) -- (a) switched the §3 job-list YAML from
  the untested `- migration-lint:` colon/null-mapping form to the
  bare-string `- migration-lint` idiom, matching this repo's established
  zero-parameter job-invocation precedent; independently re-verified via
  `python3 -c "import yaml; ..."` that the bare-string form parses correctly
  alongside a mapping-form sibling in the same job list; (b) decided the doc
  addition's target file explicitly as `docs/migrations.md` (not
  `docs/operations.md`), since that is the file the job's own failure
  message already cross-references, removing the last implementer judgment
  call; (c) reworded §6 to acknowledge the design deliberately goes beyond
  the CEO's literal "file exists" ask to structural/metadata validity,
  flagged explicitly for CEO awareness rather than asserted as precise
  conformance -- the underlying design decision itself is unchanged, only
  the characterization of it. Round 3 independently re-verified all
  round-1/round-2 citations from scratch and found them still accurate; no
  drift found in the §7 changelog against real diffs.
