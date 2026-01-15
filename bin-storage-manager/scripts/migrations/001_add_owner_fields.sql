-- Migration: Add owner_type and owner_id columns to storage_files table
-- Date: 2026-01-15
-- Description: Adds Owner fields from commonidentity.Owner to support ownership tracking

ALTER TABLE storage_files
  ADD COLUMN owner_type VARCHAR(255) DEFAULT '' AFTER account_id,
  ADD COLUMN owner_id BINARY(16) DEFAULT (UNHEX(REPLACE('00000000-0000-0000-0000-000000000000','-',''))) AFTER owner_type;

-- Add index for owner_id lookups
CREATE INDEX idx_storage_files_owner_id ON storage_files(owner_id);

-- Verify the columns were added
SHOW COLUMNS FROM storage_files LIKE 'owner%';
