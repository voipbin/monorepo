-- Rollback Migration: Remove owner_type and owner_id columns from storage_files table
-- Date: 2026-01-15

-- Drop index first
DROP INDEX IF EXISTS idx_storage_files_owner_id ON storage_files;

-- Remove columns
ALTER TABLE storage_files
  DROP COLUMN IF EXISTS owner_id,
  DROP COLUMN IF EXISTS owner_type;

-- Verify the columns were removed
SHOW COLUMNS FROM storage_files LIKE 'owner%';
