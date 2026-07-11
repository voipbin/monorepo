# Operations

Operational reference for applying, verifying, and recovering Alembic migrations.

## Environment Access

Only humans may run `alembic upgrade` or `alembic downgrade` against staging or production manually (VPN-based, ad hoc). AI agents may assist with local development only. As of VOIP-1246, production `bin_manager` and `asterisk` schema upgrades also run automatically via CircleCI's `bin-dbscheme-manager-migrate` job (see "CI-driven production migration" below) — that job is the one sanctioned exception to "human only", gated behind a CircleCI approval click restricted to the `main` branch, not an AI agent invoking alembic directly.

| Environment | alembic upgrade allowed? | alembic downgrade allowed? | Who can run? |
|---|---|---|---|
| local (developer laptop) | Yes, with caution | Yes | Developer or AI agent |
| staging | Yes, with human review | With explicit sign-off | Human only |
| production | Yes, via CI (see below) or manual change-control approval | Emergency use only (see below), manual only | CI job (upgrade only) or human, VPN required for manual/downgrade |

**Before running against any remote database manually:**
1. Connect to VPN.
2. Confirm target database URL in `alembic.ini` points to the correct environment.
3. Run `alembic current` to verify the current head before applying.
4. Run against staging first; never skip straight to production.

## CI-driven production migration (VOIP-1246)

`bin-dbscheme-manager`'s CircleCI workflow includes a `build-approval` gate (restricted to the `main` branch) followed by `bin-dbscheme-manager-migrate`, which runs `bin-dbscheme-manager/ci/migrate.py` against the live production `bin_manager` and `asterisk` databases using the `CC_DATABASE_DSN` / `CC_DATABASE_DSN_ASTERISK` secrets already present in the `production` CircleCI context.

**What clicking approval on `main` actually does now:** runs `alembic upgrade head` for both streams (`bin-manager` then `asterisk_config`, sequentially), each wrapped in a MySQL named lock (`GET_LOCK`/`RELEASE_LOCK`) to prevent two overlapping runs, preceded by an `alembic upgrade head --sql` dry-run logged for audit. Subprocess output streams to the CircleCI job log line-by-line as it's produced (not buffered until the command exits), so a slow-but-healthy migration doesn't go dark in the logs and a genuine hang still leaves a diagnosable trail up to the point it stopped.

**If the job fails (either stream):**
1. Check the CircleCI job log — the two streams run as separate steps ("Upgrade bin_manager (voipbin core DB)" / "Upgrade asterisk config DB"), so which stream failed is visually unambiguous (green vs. red step).
2. A **rerun of the job is normally safe**: `alembic upgrade head` is naturally idempotent (a stream already at head is a no-op; a partially-applied stream resumes from its last committed revision), and the named lock is session-scoped so a dead/cancelled prior run's lock is released automatically. This safety currently relies on alembic/MySQL's own idempotent behavior around DDL, not on explicit partial-failure detection in the job itself — see "Locked table during migration" below for the one case (a long-running ALTER hitting `no_output_timeout`) where a rerun may not be enough and manual intervention via the procedure below is needed.
3. If a rerun doesn't resolve it, or the failure looks like data corruption / an unsafe partial state, fall back to the manual VPN-based procedure in this document — connect to VPN, run `alembic current --verbose` against the affected stream to see exactly where it stands, and proceed from there (including Emergency Rollback below, if needed).

**When to use CI vs. manual VPN procedure:** normal migrations (new columns/tables/indexes from routine feature work) go through CI on merge to `main`. The manual VPN procedure remains the only path for `alembic downgrade` (CI only ever runs `upgrade head`, never downgrade) and for any migration the operator wants to review to right before applying, no matter what.

---

## Emergency Rollback

Use `alembic downgrade` only when a migration has caused a production incident and reverting is the fastest path to restore service.

**Procedure:**

```bash
# 1. Connect to VPN and confirm you are targeting the right database.
cd bin-dbscheme-manager/bin-manager
alembic -c alembic.ini current --verbose
# Verify the currently applied revision matches what you expect.

# 2. Check what the previous revision is.
alembic -c alembic.ini history --verbose | head -20

# 3. Revert the last migration.
alembic -c alembic.ini downgrade -1

# 4. Verify state after rollback.
alembic -c alembic.ini current --verbose

# 5. Confirm the affected service is healthy (check its logs and metrics).
```

**Revert to a specific revision (if more than one migration must be undone):**

```bash
alembic -c alembic.ini downgrade <target_revision_id>
```

**What to check before rolling back:**

- Does the `downgrade()` function actually exist and reverse the change cleanly? Verify in the migration file before running.
- Are any Go services currently writing to the affected table? If yes, drain or stop them first to avoid a mixed-state window.
- Does rolling back drop a column that a newer code version still reads? Coordinate with the deployment plan.

## Common Failures

### Missing column at runtime

**Symptom:** Go service logs `Error 1054: Unknown column 'X' in 'field list'` or panics on startup.

**Cause:** A migration that added column `X` was not yet applied to this environment, but a new code version that references it was deployed.

**Fix:**
1. Run `alembic current` on the target database to find the applied head.
2. Compare against `alembic heads` — identify the unapplied migrations.
3. Apply them: `alembic upgrade head`.
4. Restart the affected service.

**Prevention:** Always apply migrations before deploying the code version that depends on them.

---

### Multiple heads (branching)

**Symptom:** `alembic heads` shows more than one revision hash; `alembic upgrade head` fails with "Multiple head revisions".

**Cause:** Two developers created migrations independently from the same parent, producing a branch in the revision chain.

**Fix:**
1. Identify the two branch tips: `alembic heads`.
2. Create a merge migration: `alembic merge -m "merge_heads" <rev1> <rev2>`.
3. The generated file will have both revisions as `down_revision = (<rev1>, <rev2>)`.
4. Commit and apply: `alembic upgrade head`.

**Prevention:** Always `git pull` and check `alembic heads` before creating a new migration.

---

### Wrong revision head (stale local state)

**Symptom:** `alembic upgrade head` says "Already up to date" but the column or table is missing from the database.

**Cause:** The local `alembic_version` table in the database has a stale or manually edited revision ID.

**Fix:**
1. Check what the database thinks is current: `alembic current`.
2. Check what the latest migration file is: `alembic history | head -5`.
3. If the database is ahead of or mismatched with the migration chain, use `alembic stamp <correct_revision>` to reset the pointer, then re-run `alembic upgrade head`.

---

### Locked table during migration

**Symptom:** Migration hangs or times out on a large table ALTER.

**Cause:** The table is actively used by a running service holding a write lock, or the ALTER itself locks the table for its duration (common with `ADD COLUMN` on large tables in older MySQL).

**Fix:**
1. Use `pt-online-schema-change` or `gh-ost` for large-table ALTERs in production.
2. Schedule the migration during low-traffic windows.
3. Break the migration into smaller steps (add column as nullable first, backfill, then add constraints).

**If this happens via the CI job (`bin-dbscheme-manager-migrate`):** each `- run:` step has a 10-minute `no_output_timeout`. Output now streams line-by-line as alembic produces it, so a hang genuinely shows a stalled log (no new lines for 10 minutes) rather than a false alarm from output buffering — but 10 minutes is still an untuned guess for how long a real production migration takes, since this job has not run against a large production table yet. If a legitimate large-table migration is expected to take longer, raise `no_output_timeout` on that step ahead of time rather than discovering the false-timeout live.

---

### Collation mismatch after dump import

**Symptom:** Importing the Docker schema dump fails with `Unknown collation: 'utf8mb4_uca1400_ai_ci'`.

**Cause:** The dump was exported from MariaDB (which uses `utf8mb4_uca1400_ai_ci`) but imported into MySQL 8.0 (which does not support that collation).

**Fix:** The Dockerfile already applies a `sed` patch to rewrite `utf8mb4_uca1400_ai_ci` → `utf8mb4_general_ci` during export. If the issue appears outside the Docker build, apply the same substitution manually to the dump file before importing.
