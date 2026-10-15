-- Baseline of the table owned by bin-direct-manager.
-- The statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS direct_directs (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  resource_type varchar(32) NOT NULL,
  resource_id binary(16) NOT NULL,
  hash varchar(255) NOT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_direct_directs_hash (hash),
  UNIQUE KEY idx_direct_directs_resource (resource_type, resource_id),
  KEY idx_direct_directs_customer_id (customer_id)
);
