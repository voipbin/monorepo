-- Baseline of the table owned by bin-number-manager.
-- The statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS number_numbers (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  call_flow_id binary(16) DEFAULT NULL,
  message_flow_id binary(16) DEFAULT NULL,
  name varchar(255) DEFAULT '',
  detail text DEFAULT NULL,
  number varchar(255) DEFAULT NULL,
  type varchar(255) NOT NULL DEFAULT 'normal',
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  provider_name varchar(255) DEFAULT NULL,
  provider_reference_id varchar(255) DEFAULT NULL,
  status varchar(32) DEFAULT NULL,
  t38_enabled tinyint(1) DEFAULT NULL,
  emergency_enabled tinyint(1) DEFAULT NULL,
  metadata json DEFAULT NULL,
  tm_purchase datetime(6) DEFAULT NULL,
  tm_renew datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_number_numbers_number (number),
  KEY idx_number_numbers_customer_id (customer_id),
  KEY idx_number_numbers_call_flow_id (call_flow_id),
  KEY idx_number_numbers_message_flow_id (message_flow_id),
  KEY idx_number_numbers_provider_name (provider_name),
  KEY idx_number_numbers_tm_renew (tm_renew),
  KEY idx_number_numbers_type (type)
);
