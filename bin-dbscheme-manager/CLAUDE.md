# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Database schema management for VoIPBin using Alembic migrations. This repository manages two separate database schemas:

- **bin-manager** - VoIPBin core database (`voipbin` database)
- **asterisk_config** - Asterisk-related database (`asterisk` database)

Each directory maintains its own Alembic migration history with separate `alembic.ini` configurations and `versions/` directories.

## Database Architecture

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

## Asterisk Schema Updates

To sync latest Asterisk schema (when updating Asterisk version):
```bash
cd /path/to/asterisk/repo
git pull
cp contrib/ast-db-manage/config/versions/* /path/to/bin-dbscheme-manager/asterisk_config/config/versions/
```

## Important Notes

- Always work within the appropriate subdirectory (`bin-manager/` or `asterisk_config/`) when running Alembic commands
- Each directory has its own independent migration chain - never mix them
- The `-c alembic.ini` flag is required for all Alembic commands since the config file is in the subdirectory
- Migration files should include both `upgrade()` and `downgrade()` functions for rollback capability
- VPN connection is mandatory for running migrations against remote databases
