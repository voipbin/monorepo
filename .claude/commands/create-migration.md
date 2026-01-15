# create-migration

Create an Alembic database migration following VoIPbin naming conventions.

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

1. Validates schema argument (bin-manager or asterisk)
2. Navigates to correct directory:
   - `bin-manager` → `bin-dbscheme-manager/bin-manager/`
   - `asterisk` → `bin-dbscheme-manager/asterisk_config/`
3. Checks if `alembic.ini` exists (errors if missing)
4. Generates migration with proper naming: `<table>_<action>_<description>`
5. Runs: `alembic -c alembic.ini revision -m "<migration_name>"`
6. Shows path to created migration file
7. Displays next steps reminder

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

Next steps:
  1. Edit migration file to add SQL:
     - Add SQL in upgrade() function
     - Add rollback SQL in downgrade() function

  2. Test locally:
     alembic -c alembic.ini upgrade head
     mysql -u root -p voipbin -e "DESCRIBE storage_files;"

  3. Test rollback:
     alembic -c alembic.ini downgrade -1
     alembic -c alembic.ini upgrade head

  4. Commit migration:
     git add main/versions/b1201e50b736_*.py
     git commit -m "feat(dbscheme): add owner_type to storage_files"

  5. Apply to production (VPN REQUIRED):
     alembic -c alembic.ini upgrade head

IMPORTANT: VPN connection required for production migrations!
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

## Why This Command Exists

- 315 existing migrations show this is a frequent operation
- Strict naming convention must be followed exactly
- Easy to create migration in wrong directory
- `-c alembic.ini` flag commonly forgotten
- VPN requirement for production often overlooked
- Saves 2-3 minutes per migration
- Prevents migration naming inconsistencies

## Related Documentation

See `CLAUDE.md` section "Database Migrations with Alembic" for detailed workflow and patterns.
