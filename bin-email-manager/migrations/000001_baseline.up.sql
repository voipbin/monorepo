-- Baseline of the table owned by bin-email-manager.
-- The statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS email_emails (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  activeflow_id binary(16) DEFAULT NULL,
  provider_type varchar(255) DEFAULT NULL,
  provider_reference_id varchar(255) DEFAULT NULL,
  source json DEFAULT NULL,
  destinations json DEFAULT NULL,
  status varchar(255) DEFAULT NULL,
  subject varchar(255) DEFAULT NULL,
  content text DEFAULT NULL,
  attachments json DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_email_emails_customer_id (customer_id),
  KEY idx_email_emails_activeflow_id (activeflow_id),
  KEY idx_email_emails_provider_type (provider_type),
  KEY idx_email_emails_provider_reference_id (provider_reference_id)
);
