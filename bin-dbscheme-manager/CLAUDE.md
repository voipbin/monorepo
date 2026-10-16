# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`bin-dbscheme-manager` is the canonical home for database schema management in VoIPbin using Alembic migrations. It holds two independent Alembic streams: `asterisk_config/` for the Asterisk PJSIP config database, and `bin-manager/` for the `bin_manager` database, which is frozen because every application table there is now owned by a service (see below).

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, RST sync, the general "Database schema changes: Alembic or service-owned migrations" rule) live in the root [CLAUDE.md](../CLAUDE.md). This file is the authoritative reference for Alembic-specific procedures.

## Documentation Index

- [docs/migrations.md](docs/migrations.md) — How to create migration files, naming conventions, and a worked example
- [docs/schema-ownership.md](docs/schema-ownership.md) — Which service owns each database table
- [docs/operations.md](docs/operations.md) — Environment access rules, emergency rollback, and common failure recovery

## CRITICAL: bin-manager stream is frozen

Do not add new Alembic migrations to `bin-manager/` (the `bin_manager` database). Every application table in `bin_manager` (except the unused legacy `users` table) is owned by a service that keeps its own `migrations/` directory and applies it at startup; the services are listed in [docs/schema-ownership.md](docs/schema-ownership.md). Never write an Alembic migration for those tables. The `asterisk_config/` stream is not frozen and keeps using Alembic.

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
# CORRECT: always use alembic revision to generate the file (asterisk_config only; bin-manager/ is frozen)
cd asterisk_config
alembic -c alembic.ini revision -m "ps_endpoints_add_column_dtls_auto_gen"
# Then edit the generated file to add SQL in upgrade()/downgrade()

# WRONG — never create migration files manually with made-up revision IDs
# Manually creating a file with revision = 'a1b2c3d4e5f6' WILL collide with existing migrations
```

**Why:** Revision IDs must be globally unique across all migration files. Manually chosen IDs often collide with existing migrations, causing Alembic to fail with "Multiple head revisions" or "Revision is present more than once" errors.

**After generating the file:**
1. Open the generated file in `config/versions/` (the `bin-manager` stream and its `main/versions/` are frozen)
2. Add your SQL to the `upgrade()` and `downgrade()` functions
3. Verify with `alembic -c alembic.ini heads` — should show exactly one head

## Common Commands

```bash
# --- Setup (one-time per environment) ---
cd bin-manager
cp alembic.ini.sample alembic.ini
# Edit alembic.ini: set sqlalchemy.url = mysql://user:pass@host/bin_manager

cd asterisk_config
cp alembic.ini.sample alembic.ini
# Edit alembic.ini: set sqlalchemy.url = mysql://user:pass@host/asterisk

# --- Creating a migration (asterisk_config only; bin-manager/ is frozen) ---
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

**Two independent migration streams:**
- `bin-manager/` → the `bin_manager` MySQL database; migrations in `main/versions/`. Frozen: no new migrations, the tables are owned by services
- `asterisk_config/` → manages `asterisk` MySQL database; migrations in `config/versions/`

Never mix migrations between the two streams. Always `cd` into the correct subdirectory and pass `-c alembic.ini`.

**No packaged/deployed image.** This project previously shipped a `Dockerfile` that baked a temporary-MariaDB-built schema dump into a `mysql:8.0`-based image via `/docker-entrypoint-initdb.d/`, plus a `k8s/` Job manifest to run it — both removed in VOIP-1398 as dead: no CircleCI job ever built or pushed the image, and `voipbin/voipbin`'s `install/scripts/migrate.sh` never referenced it (it sparse-checks-out this directory's alembic *source* at a pinned commit and runs `alembic upgrade head` directly, in its own container). This directory's actual consumers are: `migrate.sh` (source checkout), CircleCI's `migration-lint` job (structural validation only), the `bin-dbscheme-manager-test` job (see "Classifier and tests" below), and humans running `alembic` manually per `docs/operations.md`.

## CRITICAL: UUID columns must be `BINARY(16)`

Every UUID column (`id`, `customer_id`, `agent_id`, any `_id` foreign key) **must** be declared as `BINARY(16)`. Never use `VARCHAR(36)`.

Go's MySQL driver sends `uuid.UUID` values as 16 raw bytes. `VARCHAR(36)` silently stores them as garbage characters and JOINs never match. Full rationale and ALTER-fix template are in [docs/conventions/database.md §7.0a](../docs/conventions/database.md).

## Monorepo Context

The schema of each `bin_manager` table lives in the owning service's `migrations/` directory (see [docs/schema-ownership.md](docs/schema-ownership.md) and [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md)). When adding a `db:` tag to a Go model, add a migration to that service's `migrations/`, not here. Updating schema in the service's test SQL files without a migration breaks production.

## Classifier and tests

`classifier/` decides whether a pending Alembic range is provably additive. It is a library plus tests; nothing here connects to or changes a real database.

- `source_gate.py`: default-deny gate over the revision files (directory listing, encoding, import-time code, the body of `upgrade()`).
- `render.py`, `sqltext.py`, `statements.py`: offline render of one step, a quote-aware tokenizer with `normalize()`, and the statement allow-list.
- `replay.py`: a guard that lets only the classified statements, in order, reach a server.
- `classify.py`: ties them together per stream. `module_level_exceptions.txt` lists, by sha256, files that predate the import-time rules; a changed file is not covered.

Anything outside the allow-list is reported as not provably additive (a person applies it); it is never guessed at. The authoring rules are in `docs/migrations.md`.

Tests: `pip install --require-hashes -r requirements-dev.txt`, start a MariaDB 12.3 server, export `DBSCHEME_TEST_DB_URL=mariadb+pymysql://root@127.0.0.1:3306`, then `pytest bin-dbscheme-manager/tests`. Without the variable the server tests are skipped. `requirements.txt` is hash pinned (`alembic`, `sqlalchemy<2.0`, `pymysql`); regenerate it with `uv pip compile requirements.in --generate-hashes --universal --python-version 3.11 -o requirements.txt`.
