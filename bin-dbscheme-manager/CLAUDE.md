# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`bin-dbscheme-manager` is the canonical home for database schema management in VoIPbin using Alembic migrations. It owns two independent MySQL schemas: the `voipbin` core database (managed under `bin-manager/`) and the Asterisk PJSIP config database (managed under `asterisk_config/`).

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, RST sync, the general "Database via Alembic only" rule) live in the root [CLAUDE.md](../CLAUDE.md). This file is the authoritative reference for Alembic-specific procedures.

## Documentation Index

- [docs/migrations.md](docs/migrations.md) — How to create migration files, naming conventions, and a worked example
- [docs/schema-ownership.md](docs/schema-ownership.md) — Which service owns each database table
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
# CORRECT — always use alembic revision to generate the file
cd bin-manager
alembic -c alembic.ini revision -m "ai_aicalls_add_column_current_member_id"
# Then edit the generated file to add SQL in upgrade()/downgrade()

# WRONG — never create migration files manually with made-up revision IDs
# Manually creating a file with revision = 'a1b2c3d4e5f6' WILL collide with existing migrations
```

**Why:** Revision IDs must be globally unique across all migration files. Manually chosen IDs often collide with existing migrations, causing Alembic to fail with "Multiple head revisions" or "Revision is present more than once" errors.

**After generating the file:**
1. Open the generated file in `main/versions/` (or `config/versions/`)
2. Add your SQL to the `upgrade()` and `downgrade()` functions
3. Verify with `alembic -c alembic.ini heads` — should show exactly one head

## Common Commands

```bash
# --- Setup (one-time per environment) ---
cd bin-manager
cp alembic.ini.sample alembic.ini
# Edit alembic.ini: set sqlalchemy.url = mysql://user:pass@host/voipbin

cd asterisk_config
cp alembic.ini.sample alembic.ini
# Edit alembic.ini: set sqlalchemy.url = mysql://user:pass@host/asterisk

# --- Creating a migration ---
cd bin-manager
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

**Two independent migration streams:**
- `bin-manager/` → manages `voipbin` MySQL database; migrations in `main/versions/`
- `asterisk_config/` → manages `asterisk` MySQL database; migrations in `config/versions/`

Never mix migrations between the two streams. Always `cd` into the correct subdirectory and pass `-c alembic.ini`.

**Docker build:** The production image applies migrations against a temporary local MariaDB during build and exports schema dumps. Schema dumps are imported at container startup via `/docker-entrypoint-initdb.d/`. See the Dockerfile for details.

## CRITICAL: UUID columns must be `BINARY(16)`

Every UUID column (`id`, `customer_id`, `agent_id`, any `_id` foreign key) **must** be declared as `BINARY(16)`. Never use `VARCHAR(36)`.

Go's MySQL driver sends `uuid.UUID` values as 16 raw bytes. `VARCHAR(36)` silently stores them as garbage characters and JOINs never match. Full rationale and ALTER-fix template are in [docs/conventions/database.md §7.0a](../docs/conventions/database.md).

## Monorepo Context

Every Go manager service that uses MySQL relies on the schema produced here. When adding a `db:` tag to a Go model, you MUST also create a corresponding Alembic migration here. Updating schema in `scripts/database_scripts/table_*.sql` (used in tests) without a migration breaks production.
