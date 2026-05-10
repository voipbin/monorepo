# Operations

Operational reference for applying, verifying, and recovering Alembic migrations.

## Environment Access

Only humans may run `alembic upgrade` or `alembic downgrade` against staging or production. AI agents may assist with local development only.

| Environment | alembic upgrade allowed? | alembic downgrade allowed? | Who can run? |
|---|---|---|---|
| local (developer laptop) | Yes, with caution | Yes | Developer or AI agent |
| staging | Yes, with human review | With explicit sign-off | Human only |
| production | Yes, with change-control approval | Emergency use only (see below) | Human only, VPN required |

**Before running against any remote database:**
1. Connect to VPN.
2. Confirm target database URL in `alembic.ini` points to the correct environment.
3. Run `alembic current` to verify the current head before applying.
4. Run against staging first; never skip straight to production.

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

---

### Collation mismatch after dump import

**Symptom:** Importing the Docker schema dump fails with `Unknown collation: 'utf8mb4_uca1400_ai_ci'`.

**Cause:** The dump was exported from MariaDB (which uses `utf8mb4_uca1400_ai_ci`) but imported into MySQL 8.0 (which does not support that collation).

**Fix:** The Dockerfile already applies a `sed` patch to rewrite `utf8mb4_uca1400_ai_ci` → `utf8mb4_general_ci` during export. If the issue appears outside the Docker build, apply the same substitution manually to the dump file before importing.
