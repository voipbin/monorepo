# create-migration

Create an Alembic database migration file following VoIPbin naming conventions.

⚠️ **CRITICAL: This command ONLY creates the migration file. It does NOT apply migrations to the database.**

🚫 **AI MUST NEVER run `alembic upgrade` or `alembic downgrade` commands. Database migrations require explicit human authorization.**

## Usage

```bash
/create-migration <schema> <table> <action> <description>
```

## Arguments

- `schema`: Which database schema
  - `bin-manager` - VoIPbin core database (voipbin)
  - `asterisk` - Asterisk configuration database
- `table`: Table name (e.g., "customers", "storage_files", "ps_endpoints")
- `action`: Type of change
  - `add` - Adding columns, indexes, constraints
  - `remove` - Removing columns, indexes, constraints
  - `update` - Modifying existing columns
  - `create` - Creating new table
  - `drop` - Dropping table
- `description`: What's being changed (e.g., "column email phone_number", "table", "index idx_customer_id")

## What This Does

**This command ONLY creates a migration file. It does NOT modify the database.**

1. Validates schema argument (bin-manager or asterisk)
2. Navigates to correct directory:
   - `bin-manager` → `bin-dbscheme-manager/bin-manager/`
   - `asterisk` → `bin-dbscheme-manager/asterisk_config/`
3. Checks if `alembic.ini` exists (errors if missing)
4. Generates migration with proper naming: `<table>_<action>_<description>`
5. Runs: `alembic -c alembic.ini revision -m "<migration_name>"` (creates file only)
6. Shows path to created migration file
7. Displays manual next steps for human to execute

## Examples

```bash
# Add column to existing table
/create-migration bin-manager storage_files add "column owner_type"

# Create new table
/create-migration bin-manager registrar_trunks create table

# Add index
/create-migration bin-manager calls add "index idx_customer_id_tm_create"

# Update asterisk schema
/create-migration asterisk ps_endpoints add "column dtls_auto_gen"

# Add multiple columns
/create-migration bin-manager customers add "column email phone_number address"

# Remove column
/create-migration bin-manager conversations remove "column legacy_field"
```

## Output Example

```
✓ Schema: bin-manager (voipbin database)
✓ Navigated to: /home/pchero/gitvoipbin/monorepo/bin-dbscheme-manager/bin-manager
✓ Found alembic.ini (connected to: mysql://user@localhost/voipbin)
✓ Creating migration: storage_files_add_column_owner_type

Generated migration file:
  main/versions/b1201e50b736_storage_files_add_column_owner_type.py

⚠️  MANUAL NEXT STEPS (Human must execute these - AI will NOT do this):

  1. Edit migration file to add SQL:
     - Add SQL in upgrade() function
     - Add rollback SQL in downgrade() function

  2. Commit migration file:
     git add main/versions/b1201e50b736_*.py
     git commit -m "feat(dbscheme): add owner_type to storage_files"

  3. (HUMAN ONLY) Test locally on development database:
     alembic -c alembic.ini upgrade head
     mysql -u root -p voipbin -e "DESCRIBE storage_files;"

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
✗ ERROR: alembic.ini not found in bin-manager/

To fix:
  cd bin-dbscheme-manager/bin-manager
  cp alembic.ini.sample alembic.ini

Edit alembic.ini and set your database connection:
  sqlalchemy.url = mysql://user:pass@host/voipbin
```

**Invalid schema:**
```
✗ ERROR: Invalid schema 'production'

Valid schemas:
  - bin-manager (for voipbin database)
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

See `CLAUDE.md` section "Database Migrations with Alembic" for detailed workflow and patterns.
