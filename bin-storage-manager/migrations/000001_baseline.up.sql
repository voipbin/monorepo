-- Baseline of the tables owned by bin-storage-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the tables
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS storage_accounts (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  total_file_count bigint DEFAULT NULL,
  total_file_size bigint DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_storage_accounts_customer_id (customer_id)
);

CREATE TABLE IF NOT EXISTS storage_files (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  account_id binary(16) DEFAULT NULL,
  owner_id binary(16) DEFAULT NULL,
  owner_type varchar(255) DEFAULT '',
  reference_type varchar(255) DEFAULT NULL,
  reference_id binary(16) DEFAULT NULL,
  type varchar(255) NOT NULL DEFAULT '',
  name varchar(255) DEFAULT NULL,
  detail text DEFAULT NULL,
  bucket_name text DEFAULT NULL,
  filename varchar(1023) DEFAULT NULL,
  filepath text DEFAULT NULL,
  filesize bigint DEFAULT NULL,
  uri_bucket text DEFAULT NULL,
  uri_download text DEFAULT NULL,
  tm_download_expire datetime(6) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_storage_files_customer_id (customer_id),
  KEY idx_storage_files_account_id (account_id),
  KEY idx_storage_files_owner_id (owner_id),
  KEY idx_storage_files_reference_id (reference_id)
);
