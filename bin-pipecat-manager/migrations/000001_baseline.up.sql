-- Baseline of the table owned by bin-pipecat-manager.
-- The statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- No table options are given on purpose: the database defaults are inherited, as Alembic did.
CREATE TABLE IF NOT EXISTS pipecat_pipecatcalls (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  activeflow_id binary(16) DEFAULT NULL,
  reference_type varchar(255) DEFAULT NULL,
  reference_id binary(16) DEFAULT NULL,
  host_id varchar(255) DEFAULT NULL,
  llm_type varchar(255) DEFAULT NULL,
  llm_messages json DEFAULT NULL,
  stt_type varchar(255) DEFAULT NULL,
  stt_language varchar(255) NOT NULL DEFAULT '',
  tts_type varchar(255) DEFAULT NULL,
  tts_language varchar(255) NOT NULL DEFAULT '',
  tts_voice_id varchar(255) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_pipecat_pipecatcalls_customer_id (customer_id),
  KEY idx_pipecat_pipecatcalls_activeflow_id (activeflow_id),
  KEY idx_pipecat_pipecatcalls_reference_type (reference_type),
  KEY idx_pipecat_pipecatcalls_reference_id (reference_id),
  KEY idx_pipecat_pipecatcalls_host_id (host_id),
  KEY idx_pipecat_pipecatcalls_tm_create (tm_create)
);
