# Storage Manager Database Migrations

This directory contains database migration scripts for the storage-manager service.

## Migration 001: Add Owner Fields

**Date:** 2026-01-15
**Issue:** Error 1054 (42S22): Unknown column 'owner_type' in 'field list'

### Problem
The `file.File` model was updated to embed `commonidentity.Owner`, which includes `owner_type` and `owner_id` fields. However, the production database table `storage_files` was missing these columns, causing runtime errors.

### Solution
Add `owner_type` and `owner_id` columns to the `storage_files` table with appropriate defaults and indexes.

### Apply Migration

```sql
-- Connect to production database
mysql -h <host> -u <user> -p <database>

-- Apply migration
SOURCE /path/to/monorepo/bin-storage-manager/scripts/migrations/001_add_owner_fields.sql;
```

### Rollback (if needed)

```sql
-- Connect to production database
mysql -h <host> -u <user> -p <database>

-- Rollback migration
SOURCE /path/to/monorepo/bin-storage-manager/scripts/migrations/001_add_owner_fields_rollback.sql;
```

### Verification

After applying the migration, verify the columns exist:

```sql
SHOW COLUMNS FROM storage_files LIKE 'owner%';
```

Expected output:
```
+------------+--------------+------+-----+---------+-------+
| Field      | Type         | Null | Key | Default | Extra |
+------------+--------------+------+-----+---------+-------+
| owner_type | varchar(255) | YES  |     |         |       |
| owner_id   | binary(16)   | YES  | MUL | ...     |       |
+------------+--------------+------+-----+---------+-------+
```

### Notes
- The migration uses `DEFAULT ''` for `owner_type` and a nil UUID for `owner_id` to handle existing rows
- An index is created on `owner_id` for efficient lookups (matching the test schema)
- Existing rows will have empty/nil owner values until explicitly updated
