-- Baseline of the tables owned by bin-conversation-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS conversation_accounts (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  type varchar(32) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail text DEFAULT NULL,
  secret text DEFAULT NULL,
  token text DEFAULT NULL,
  message_flow_id binary(16) NOT NULL DEFAULT (UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''))),
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  provider_data json DEFAULT NULL COMMENT 'Provider-specific credentials (JSON). WhatsApp: phone_number_id, app_secret.',
  PRIMARY KEY (id),
  KEY idx_conversation_accounts_customer_id (customer_id)
);

CREATE TABLE IF NOT EXISTS conversation_conversations (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  owner_type varchar(255) DEFAULT NULL,
  owner_id binary(16) DEFAULT NULL,
  account_id binary(16) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail text DEFAULT NULL,
  type varchar(255) DEFAULT NULL,
  dialog_id varchar(255) DEFAULT NULL,
  self json DEFAULT NULL,
  peer json DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  metadata json DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_conversation_conversations_customer_id (customer_id),
  KEY idx_conversation_conversations_reference_type_reference_id (type, dialog_id),
  KEY idx_conversation_conversations_owner_id (owner_id)
);

CREATE TABLE IF NOT EXISTS conversation_medias (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  type varchar(255) DEFAULT NULL,
  Filename varchar(2047) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_conversation_medias_customer_id (customer_id)
);

CREATE TABLE IF NOT EXISTS conversation_messages (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  conversation_id binary(16) DEFAULT NULL,
  direction varchar(255) DEFAULT '',
  status varchar(16) DEFAULT NULL,
  reference_type varchar(255) DEFAULT NULL,
  transaction_id varchar(255) DEFAULT '',
  text text DEFAULT NULL,
  subject varchar(255) NOT NULL DEFAULT '',
  medias json DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  reference_id binary(16) DEFAULT NULL,
  source json DEFAULT NULL,
  destination json DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_conversation_messages_customer_id (customer_id),
  KEY idx_conversation_messages_reference_type_reference_id (reference_type),
  KEY idx_conversation_messages_transaction_id (transaction_id)
);
