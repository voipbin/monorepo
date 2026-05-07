# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-dbscheme-manager` is the canonical home for database schema management in VoIPbin using Alembic migrations. This repository manages two separate database schemas:

- **bin-manager** — VoIPbin core database (`voipbin` database)
- **asterisk_config** — Asterisk-related database (`asterisk` database)

Each directory maintains its own Alembic migration history with separate `alembic.ini` configurations and `versions/` directories.

**Key Concepts:**
- **Two independent migration streams** — `bin-manager/main/versions/` and `asterisk_config/config/versions/`. Never mix them.
- **`alembic revision` is the only safe way to create migration files** — handpicked revision IDs collide.
- **VPN required for production migrations** — staging and production database access is gated by VPN.
- **Schema dumps baked into Docker image** — the production image has migrations applied during build, not at runtime.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, RST sync, the general "Database via Alembic only" rule) live in the root [CLAUDE.md](../CLAUDE.md). This file is the authoritative reference for Alembic-specific procedures.

## Architecture

### Database Architecture

**Two Independent Migration Streams:**
- `bin-manager/` → manages `voipbin` MySQL database
  - Alembic script location: `main/`
  - Migrations in: `main/versions/`
- `asterisk_config/` → manages `asterisk` MySQL database
  - Alembic script location: `config/`
  - Migrations in: `config/versions/`
  - Uses fixed Alembic table name: `alembic_version_config → alembic_version`

**Critical Setup Requirement:**
Before running any Alembic commands, you must copy `alembic.ini.sample` to `alembic.ini` and configure the database connection string in each directory. The sample files are version-controlled; the actual `alembic.ini` files are gitignored and must be configured locally.

## Common Commands

### Initial Configuration

**bin-manager setup:**
```bash
cd bin-manager
cp alembic.ini.sample alembic.ini
# Edit alembic.ini and set: sqlalchemy.url = mysql://user:pass@host/voipbin
```

**asterisk_config setup:**
```bash
cd asterisk_config
cp alembic.ini.sample alembic.ini
# Edit alembic.ini and set: sqlalchemy.url = mysql://user:pass@host/asterisk
```

### Creating Migrations

**Migration Naming Convention:**
```
<table_name>_<action>_<type>_<items>
```
Actions: `create`, `remove`, `add`, `update`
Types: `table`, `column`, `index`, etc.

**Examples:**
```bash
cd bin-manager
alembic -c alembic.ini revision -m "customers add column email phone_number address"
alembic -c alembic.ini revision -m "registrar_trunks create table"

cd asterisk_config
alembic -c alembic.ini revision -m "ps_endpoints add column dtls_auto_gen"
```

### Applying Migrations

**IMPORTANT:** You must be connected to VPN before running migrations against production/staging databases.

```bash
# bin-manager
cd bin-manager
alembic -c alembic.ini upgrade head

# asterisk_config
cd asterisk_config
alembic -c alembic.ini upgrade head
```

### Rolling Back Migrations

```bash
# Revert last migration
alembic -c alembic.ini downgrade -1

# Revert to specific revision
alembic -c alembic.ini downgrade <revision_id>
```

### Checking Migration Status

```bash
# Current applied migration
alembic current --verbose

# View all migrations
alembic history --verbose
```

## Docker Build Process

The Dockerfile uses a multi-stage build:

1. **Stage 1 (builder):**
   - Installs MariaDB and Python dependencies
   - Runs migrations against temporary local databases
   - Exports schema dumps with collation compatibility patches (converts `utf8mb4_uca1400_ai_ci` → `utf8mb4_general_ci`)

2. **Stage 2 (final):**
   - MySQL 8.0 base image
   - Imports schema dumps via `/docker-entrypoint-initdb.d/`

**Database creation (required before migrations):**
```sql
CREATE DATABASE voipbin CHARACTER SET utf8 COLLATE utf8_general_ci;
CREATE DATABASE asterisk CHARACTER SET utf8 COLLATE utf8_general_ci;
```

## CI/CD Integration

This project runs as a Kubernetes Job (see `k8s/job.yml`). The job pod persists until the next pipeline execution, then is automatically removed and recreated.

## Migration File Structure

Each migration file follows this pattern:
```python
"""migration_title

Revision ID: <hash>
Revises: <parent_hash>
Create Date: <timestamp>
"""
from alembic import op
import sqlalchemy as sa

revision = '<hash>'
down_revision = '<parent_hash>'
branch_labels = None
depends_on = None

def upgrade():
    op.execute("""ALTER TABLE ...""")

def downgrade():
    op.execute("""ALTER TABLE ...""")
```

Migrations use raw SQL via `op.execute()` rather than SQLAlchemy DDL operations.

## CRITICAL: UUID columns must be `BINARY(16)`

Every UUID column (`id`, `customer_id`, `agent_id`, any `_id` foreign key) **must** be declared as `BINARY(16)`. Never use `VARCHAR(36)`.

```python
# CORRECT
CREATE TABLE call_outbound_configs (
    id           BINARY(16)   NOT NULL,
    customer_id  BINARY(16)   NOT NULL,
    ...
)

# WRONG — Go's MySQL driver sends uuid.UUID as 16 raw bytes; VARCHAR(36)
# silently stores them as garbage characters and JOINs never match.
CREATE TABLE call_outbound_configs (
    id           VARCHAR(36)  NOT NULL,
    customer_id  VARCHAR(36)  NOT NULL,
    ...
)
```

**Bootstrap migrations** that copy UUIDs across tables must use compatible byte representations. To generate a new UUID for `BINARY(16)`, use `UNHEX(REPLACE(UUID(), '-', ''))` (not bare `UUID()`).

This rule is enforced by the PostToolUse hook `.claude/scripts/check-migration-uuid-columns.sh`. Full rationale and ALTER-fix template are in [docs/conventions/database.md §7.0a](../docs/conventions/database.md).

## Asterisk Schema Updates

To sync latest Asterisk schema (when updating Asterisk version):
```bash
cd /path/to/asterisk/repo
git pull
cp contrib/ast-db-manage/config/versions/* /path/to/bin-dbscheme-manager/asterisk_config/config/versions/
```

## CRITICAL: Never Manually Create Migration Files

**NEVER manually create Alembic migration files or use placeholder revision IDs.**

Always use `alembic revision` to generate migration files — it creates unique revision IDs automatically and correctly chains the `down_revision` to the current head.

```bash
# ✅ CORRECT — always use alembic revision to generate the file
cd bin-manager
alembic -c alembic.ini revision -m "ai_aicalls add column current_member_id"
# Then edit the generated file to add SQL in upgrade()/downgrade()

# ❌ WRONG — never create migration files manually with made-up revision IDs
# Manually creating a file with revision = 'a1b2c3d4e5f6' WILL collide with existing migrations
```

**Why:** Revision IDs must be globally unique across all migration files. Manually chosen IDs (like `a1b2c3d4e5f6`) often collide with existing migrations, causing Alembic to fail with "Multiple head revisions" or "Revision is present more than once" errors. The `alembic revision` command is the only safe way to generate these IDs.

**After generating the file:**
1. Open the generated file in `main/versions/` (or `config/versions/`)
2. Add your SQL to the `upgrade()` and `downgrade()` functions
3. Verify with `alembic -c alembic.ini heads` — should show exactly one head

## Request Routing

N/A — schema-management tool, not a runtime service. No `cmd/` entrypoint, no listen queue.

## Event Subscriptions

N/A — runs as a Kubernetes Job, not a long-lived service. No SubscribeHandler.

## Monorepo Context

This repository is referenced by other services indirectly: every Go manager service that uses MySQL relies on the schema produced here. Changes to the schema do not require `replace` directives because there is no Go module dependency — the contract is the database itself.

When adding a `db:` tag to a Go model, you MUST also create a corresponding Alembic migration here. Updating the schema in `scripts/database_scripts/table_*.sql` (used in tests) without a migration breaks production.

## Testing Patterns

- Migrations are validated against a temporary local MariaDB during the Docker build (see "Docker Build Process" above). If a migration is malformed or conflicts, the Docker build fails.
- Verify locally with `alembic -c alembic.ini heads` — expect exactly one head per migration stream.
- Run `alembic -c alembic.ini upgrade head` against a throwaway local database before pushing to confirm the migration applies cleanly.

## Key Implementation Details

### Migration File Generation
ALWAYS use `alembic -c alembic.ini revision -m "<message>"` to generate migration files. Hand-picked revision IDs collide with existing migrations. See "CRITICAL: Never Manually Create Migration Files" above for the full rationale.

### Collation Compatibility Patches
The Docker build exports schema dumps with collation patches (`utf8mb4_uca1400_ai_ci` → `utf8mb4_general_ci`) to keep dumps importable by MySQL 8.0.

### Two-Database Discipline
Always work within the appropriate subdirectory (`bin-manager/` or `asterisk_config/`) when running Alembic commands. Each directory has its own independent migration chain — never mix them. The `-c alembic.ini` flag is required for all Alembic commands since the config file is in the subdirectory.

## Configuration

Configuration lives in the per-subdirectory `alembic.ini` files. Both files are gitignored — copy from `alembic.ini.sample` and set `sqlalchemy.url` to your database connection string. There are no environment-variable runtime configs because this is a schema tool, not a service.

## Prometheus Metrics

N/A — runs as a Kubernetes Job and exits after completing migrations. No metrics surface.

## Important Notes

- Always work within the appropriate subdirectory (`bin-manager/` or `asterisk_config/`) when running Alembic commands
- Each directory has its own independent migration chain — never mix them
- The `-c alembic.ini` flag is required for all Alembic commands since the config file is in the subdirectory
- Migration files should include both `upgrade()` and `downgrade()` functions for rollback capability
- VPN connection is mandatory for running migrations against remote databases
