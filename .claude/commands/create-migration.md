# create-migration

Create an Alembic database migration file following VoIPbin naming conventions.

⚠️ **Only the `asterisk` schema is accepted.** The `bin-manager` stream is frozen: every application table of the `bin_manager` database is owned by a service and changes through that service's `migrations/` directory (see `docs/workflows/service-owned-migrations.md`). Refuse `bin-manager` as the schema argument.

⚠️ **CRITICAL: This command ONLY creates the migration file. It does NOT apply migrations to the database.**

🚫 **AI MUST NEVER run `alembic upgrade` or `alembic downgrade` commands. Database migrations require explicit human authorization.**

## Usage

```bash
/create-migration <schema> <table> <action> <description>
```

## Arguments

- `schema`: Which database schema
  - `asterisk` - Asterisk configuration database (the only accepted value; `bin-manager` is frozen)
- `table`: Table name (e.g., "ps_endpoints", "ps_aors")
- `action`: Type of change
  - `add` - Adding columns, indexes, constraints
  - `remove` - Removing columns, indexes, constraints
  - `update` - Modifying existing columns
  - `create` - Creating new table
  - `drop` - Dropping table
- `description`: What's being changed (e.g., "column email phone_number", "table", "index idx_customer_id")

## What This Does

**This command ONLY creates a migration file. It does NOT modify the database.**

1. Validates the schema argument (only `asterisk`; `bin-manager` is refused because the stream is frozen)
2. Navigates to `bin-dbscheme-manager/asterisk_config/`
3. Checks if `alembic.ini` exists (errors if missing)
4. Generates migration with proper naming: `<table>_<action>_<description>`
5. Runs: `alembic -c alembic.ini revision -m "<migration_name>"` (creates file only)
6. Shows path to created migration file
7. Displays manual next steps for human to execute

## Examples

```bash
# Add column to existing table
/create-migration asterisk ps_endpoints add "column dtls_auto_gen"

# Create new table
/create-migration asterisk ps_example create table

# Add index
/create-migration asterisk ps_contacts add "index idx_endpoint"
```

## Output Example

```
✓ Schema: asterisk (asterisk database)
✓ Navigated to: bin-dbscheme-manager/asterisk_config
✓ Found alembic.ini (connected to: mysql://user@localhost/asterisk)
✓ Creating migration: ps_endpoints_add_column_dtls_auto_gen

Generated migration file:
  config/versions/b1201e50b736_ps_endpoints_add_column_dtls_auto_gen.py

⚠️  MANUAL NEXT STEPS (Human must execute these - AI will NOT do this):

  1. Edit migration file to add SQL:
     - Add SQL in upgrade() function
     - Add rollback SQL in downgrade() function

  2. Commit migration file:
     git add config/versions/b1201e50b736_*.py
     git commit -m "feat(dbscheme): add dtls_auto_gen to ps_endpoints"

  3. (HUMAN ONLY) Test locally on development database:
     alembic -c alembic.ini upgrade head
     mysql -u root -p asterisk -e "DESCRIBE ps_endpoints;"

  4. (HUMAN ONLY) Test rollback:
     alembic -c alembic.ini downgrade -1
     alembic -c alembic.ini upgrade head

  5. (HUMAN ONLY) Apply to production (VPN REQUIRED):
     alembic -c alembic.ini upgrade head

🚫 AI MUST NEVER execute steps 3-5. Database changes require human authorization.
```

## Error Handling

**Missing alembic.ini:**
```
✗ ERROR: alembic.ini not found in asterisk_config/

To fix:
  cd bin-dbscheme-manager/asterisk_config
  cp alembic.ini.sample alembic.ini

Edit alembic.ini and set your database connection:
  sqlalchemy.url = mysql://user:pass@host/asterisk
```

**Invalid schema:**
```
✗ ERROR: Invalid schema 'bin-manager'

The bin-manager stream is frozen. Change bin_manager tables through the owning
service's migrations/ directory. Valid schema:
  - asterisk (for asterisk database)
```

## Security Boundaries

**What AI CAN do:**
- ✅ Create migration files (`alembic revision`)
- ✅ Edit migration files to add SQL
- ✅ Commit migration files to git
- ✅ Show migration file contents
- ✅ Explain migration SQL

**What AI MUST NEVER do:**
- 🚫 Run `alembic upgrade` (applies migration to database)
- 🚫 Run `alembic downgrade` (rolls back database changes)
- 🚫 Execute migration SQL directly against database
- 🚫 Modify database schema outside of creating migration files

**Rationale:** Database schema changes are irreversible operations that can cause data loss or service outages. They require:
- Human review and approval
- Testing on development environment first
- Coordination with deployment schedule
- VPN access to production database

## Why This Command Exists

- 315 existing migrations show this is a frequent operation
- Strict naming convention must be followed exactly
- Easy to create migration in wrong directory
- `-c alembic.ini` flag commonly forgotten
- Automates safe file creation while preventing unsafe database operations
- Saves 2-3 minutes per migration
- Prevents migration naming inconsistencies

## Related Documentation

See `docs/workflows/common-workflows.md` section "Database Migrations" and `bin-dbscheme-manager/docs/migrations.md` for the workflow and patterns.
