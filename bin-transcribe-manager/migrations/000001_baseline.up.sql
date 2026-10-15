-- Baseline of the tables owned by bin-transcribe-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the tables
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS transcribe_transcribes (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  activeflow_id binary(16) DEFAULT NULL,
  on_end_flow_id binary(16) DEFAULT NULL,
  reference_type varchar(16) DEFAULT NULL,
  reference_id binary(16) DEFAULT NULL,
  status varchar(16) DEFAULT NULL,
  host_id binary(16) DEFAULT NULL,
  language varchar(16) DEFAULT NULL,
  direction varchar(255) DEFAULT NULL,
  provider varchar(16) DEFAULT '',
  streaming_ids json DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_transcribe_transcribes_reference_id (reference_id),
  KEY idx_transcribe_transcribes_customer_id (customer_id),
  KEY transcribe_transcribes_activeflow_id (activeflow_id),
  KEY transcribe_transcribes_on_end_flow_id (on_end_flow_id)
);

CREATE TABLE IF NOT EXISTS transcribe_transcripts (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  transcribe_id binary(16) DEFAULT NULL,
  direction varchar(16) DEFAULT NULL,
  message text DEFAULT NULL,
  offset_ms bigint DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_transcribe_transcripts_customer_id (customer_id),
  KEY idx_transcribe_transcripts_transcribe_id (transcribe_id)
);
