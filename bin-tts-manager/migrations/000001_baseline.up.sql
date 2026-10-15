-- Baseline of the tables owned by bin-tts-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the table
-- (created by the former Alembic stream) are left untouched.
-- The only table option is the one the Alembic stream declared (DEFAULT CHARSET=utf8, i.e. utf8mb3).
CREATE TABLE IF NOT EXISTS tts_manager_speaking (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  reference_type varchar(32) NOT NULL DEFAULT '',
  reference_id binary(16) NOT NULL,
  language varchar(16) NOT NULL DEFAULT '',
  provider varchar(32) NOT NULL DEFAULT '',
  voice_id varchar(255) NOT NULL DEFAULT '',
  direction varchar(8) NOT NULL DEFAULT '',
  status varchar(16) NOT NULL DEFAULT '',
  pod_id varchar(64) NOT NULL DEFAULT '',
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_tts_manager_speaking_customer_id (customer_id),
  KEY idx_tts_manager_speaking_reference (reference_type, reference_id),
  KEY idx_tts_manager_speaking_status (status),
  KEY idx_tts_manager_speaking_ref_status (customer_id, reference_type, reference_id, status)
) DEFAULT CHARSET=utf8mb3;
