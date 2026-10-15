-- Baseline of the tables owned by bin-message-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS message_messages (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  `type` varchar(255) DEFAULT NULL,
  source json DEFAULT NULL,
  targets json DEFAULT NULL,
  provider_name varchar(255) DEFAULT NULL,
  provider_reference_id varchar(255) DEFAULT NULL,
  `text` text DEFAULT NULL,
  medias json DEFAULT NULL,
  direction varchar(255) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_message_messages_customer_id (customer_id),
  KEY idx_message_messages_provider_name (provider_name),
  KEY idx_message_messages_provider_reference_id (provider_reference_id)
);
