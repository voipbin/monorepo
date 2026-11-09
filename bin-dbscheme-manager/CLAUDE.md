# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`bin-dbscheme-manager` manages the Asterisk PJSIP config database of VoIPbin with Alembic migrations (`asterisk_config/`). It does not manage any other database: the application tables are owned by the services (see "Monorepo Context").

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, RST sync, the general "Database schema changes: Alembic or service-owned migrations" rule) live in the root [CLAUDE.md](../CLAUDE.md). This file is the authoritative reference for Alembic-specific procedures.

## Documentation Index

- [docs/migrations.md](docs/migrations.md) — How to create migration files, naming conventions, and a worked example
- [docs/operations.md](docs/operations.md) — Environment access rules, emergency rollback, and common failure recovery

## CRITICAL: AI Agent Prohibition on alembic upgrade / alembic downgrade

**AI agents MUST NEVER run `alembic upgrade` or `alembic downgrade` against any database other than a local throwaway instance.**

- `alembic upgrade` applies schema changes to a live database. Running it against staging or production without human authorization risks data loss, locks, and downtime.
- `alembic downgrade` rolls back schema changes. Running it in production without explicit authorization can destroy data that services have already written under the new schema.

**What AI CAN do:**
- ✅ Create migration files (`alembic -c alembic.ini revision -m "..."`)
- ✅ Edit migration files to add SQL in `upgrade()` / `downgrade()` functions
- ✅ Run `alembic heads` / `alembic current` / `alembic history` (read-only status checks)
- ✅ Run `alembic upgrade head` against a local throwaway database only
- ✅ Commit migration files to git

**What AI MUST NEVER do:**
- NEVER run `alembic upgrade` against staging or production databases
- NEVER run `alembic downgrade` against staging or production databases
- NEVER manually create migration files with hand-picked revision IDs (see below)

## CRITICAL: Never Manually Create Migration Files

**NEVER manually create Alembic migration files or use placeholder revision IDs.**

Always use `alembic revision` to generate migration files — it creates unique revision IDs automatically and correctly chains `down_revision` to the current head.

```bash
# CORRECT: always use alembic revision to generate the file (asterisk_config)
cd asterisk_config
alembic -c alembic.ini revision -m "ps_endpoints_add_column_dtls_auto_gen"
# Then edit the generated file to add SQL in upgrade()/downgrade()

# WRONG — never create migration files manually with made-up revision IDs
# Manually creating a file with revision = 'a1b2c3d4e5f6' WILL collide with existing migrations
```

**Why:** Revision IDs must be globally unique across all migration files. Manually chosen IDs often collide with existing migrations, causing Alembic to fail with "Multiple head revisions" or "Revision is present more than once" errors.

**After generating the file:**
1. Open the generated file in `config/versions/`
2. Add your SQL to the `upgrade()` and `downgrade()` functions
3. Verify with `alembic -c alembic.ini heads` — should show exactly one head

## Common Commands

```bash
# --- Setup (one-time per environment) ---
cd asterisk_config
cp alembic.ini.sample alembic.ini
# Edit alembic.ini: set sqlalchemy.url = mysql://user:pass@host/asterisk

# --- Creating a migration ---
# run from bin-dbscheme-manager/asterisk_config
alembic -c alembic.ini revision -m "table_name_verb_what"

# --- Read-only status checks (safe for AI) ---
alembic -c alembic.ini current --verbose
alembic -c alembic.ini history --verbose
alembic -c alembic.ini heads

# --- Apply locally (local throwaway DB only) ---
alembic -c alembic.ini upgrade head

# --- Rollback locally (local throwaway DB only) ---
alembic -c alembic.ini downgrade -1
```

## Architecture

**One migration stream:** `asterisk_config/` manages the `asterisk` MySQL database; migrations in `config/versions/`. Always `cd` into `asterisk_config` and pass `-c alembic.ini`.

**No packaged/deployed image.** This project previously shipped a `Dockerfile` that baked a temporary-MariaDB-built schema dump into a `mysql:8.0`-based image via `/docker-entrypoint-initdb.d/`, plus a `k8s/` Job manifest to run it — both removed in VOIP-1398 as dead: no CircleCI job ever built or pushed the image, and `voipbin/voipbin`'s `install/scripts/migrate.sh` never referenced it (it sparse-checks-out this directory's alembic *source* at a pinned commit and runs `alembic upgrade head` directly, in its own container). This directory's actual consumers are: `migrate.sh` (source checkout), CircleCI's `migration-lint` job (structural validation only), and humans running `alembic` manually per `docs/operations.md`.

## Monorepo Context

The application tables are not managed here. Each service owns its tables and keeps the schema in its own `migrations/` directory (see [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md), which also lists which service owns which table). When adding a `db:` tag to a Go model, add a migration to that service's `migrations/`, not here. Updating schema in the service's test SQL files without a migration breaks production.

`requirements.txt` is hash pinned (`alembic`, `sqlalchemy<2.0`, `pymysql`); regenerate it with `uv pip compile requirements.in --generate-hashes --universal --python-version 3.11 -o requirements.txt`.
