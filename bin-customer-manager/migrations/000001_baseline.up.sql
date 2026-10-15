-- Baseline of the tables owned by bin-customer-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS customer_customers (
  id binary(16) NOT NULL,
  name varchar(255) DEFAULT NULL,
  detail text DEFAULT NULL,
  email varchar(255) DEFAULT NULL,
  phone_number varchar(255) DEFAULT NULL,
  address varchar(1024) DEFAULT NULL,
  webhook_method varchar(255) DEFAULT NULL,
  webhook_uri varchar(1023) DEFAULT NULL,
  webhook_secret varchar(255) NOT NULL DEFAULT '',
  billing_account_id binary(16) DEFAULT NULL,
  email_verified tinyint(1) NOT NULL DEFAULT 0,
  status varchar(16) NOT NULL DEFAULT 'active',
  terms_agreed_version varchar(64) NOT NULL DEFAULT '',
  terms_agreed_ip varchar(45) NOT NULL DEFAULT '',
  tm_deletion_scheduled datetime(6) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  metadata json DEFAULT NULL,
  identity_verification_status varchar(32) NOT NULL DEFAULT 'none',
  PRIMARY KEY (id),
  KEY idx_customer_customers_status (status)
);
CREATE TABLE IF NOT EXISTS customer_accesskeys (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail text DEFAULT NULL,
  tm_expire datetime(6) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  token_hash char(64) DEFAULT NULL,
  token_prefix varchar(16) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_customer_accesskeys_customer_id (customer_id),
  KEY idx_customer_accesskeys_token_hash (token_hash)
);
