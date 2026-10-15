-- Baseline of the tables owned by bin-ai-manager.
-- Every statement is CREATE TABLE IF NOT EXISTS so that databases that already have the tables
-- (created by the former Alembic stream) are left untouched.
-- Table options: four tables (ai_ai_audits, ai_ai_prompt_histories, ai_ai_prompt_proposals,
-- ai_aicall_participants) declare DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci; the other
-- tables use the database default.
-- ai_messages.content keeps the column level charset of the former Alembic migration.
-- JSON columns are declared json (the Alembic migrations declared JSON).
CREATE TABLE IF NOT EXISTS ai_ai_audits (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  aicall_id binary(16) NOT NULL,
  ai_id binary(16) NOT NULL,
  prompt_history_id binary(16) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'progressing',
  overall_score int(11) DEFAULT NULL,
  evaluation json DEFAULT NULL,
  message_ids json DEFAULT NULL,
  language varchar(32) NOT NULL DEFAULT 'en-US',
  error text DEFAULT NULL,
  tm_create datetime(6) NOT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_aiaudit_aicall_ai (aicall_id,ai_id),
  KEY idx_aiaudit_customer_id (customer_id),
  KEY idx_aiaudit_tm_create (tm_create)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS ai_ai_prompt_histories (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  ai_id binary(16) NOT NULL,
  prompt longtext NOT NULL,
  tm_create datetime(6) NOT NULL,
  proposal_id binary(16) NOT NULL DEFAULT 0x00000000000000000000000000000000,
  PRIMARY KEY (id),
  KEY idx_ai_ai_prompt_histories_ai_id_tm_create (ai_id,tm_create),
  KEY idx_ai_ai_prompt_histories_customer_id (customer_id)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS ai_ai_prompt_proposals (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  ai_id binary(16) NOT NULL,
  audit_ids json NOT NULL,
  basis_prompt_history_id binary(16) NOT NULL,
  original_prompt text DEFAULT NULL,
  proposed_prompt text DEFAULT NULL,
  rationale text DEFAULT NULL,
  status varchar(32) NOT NULL DEFAULT 'progressing',
  error varchar(128) NOT NULL DEFAULT '',
  applied_prompt_history_id binary(16) NOT NULL DEFAULT 0x00000000000000000000000000000000,
  tm_create datetime(6) NOT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_aipromptproposal_customer_ai_create (customer_id,ai_id,tm_create),
  KEY idx_aipromptproposal_ai_status (ai_id,status,tm_delete),
  KEY idx_aipromptproposal_customer_status (customer_id,status,tm_create)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS ai_aicall_participants (
  ai_id binary(16) NOT NULL,
  aicall_id binary(16) NOT NULL,
  tm_create datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (ai_id,aicall_id),
  KEY idx_ai_aicall_participants_aicall_id (aicall_id)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS ai_aicalls (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  assistance_type varchar(255) DEFAULT '',
  assistance_id binary(16) DEFAULT NULL,
  ai_engine_model varchar(255) DEFAULT NULL,
  ai_tts_type varchar(255) DEFAULT '',
  ai_tts_voice_id varchar(255) DEFAULT '',
  ai_stt_type varchar(255) DEFAULT '',
  ai_vad_config json DEFAULT NULL,
  ai_smart_turn_enabled tinyint(1) NOT NULL DEFAULT 0,
  parameter json DEFAULT NULL,
  activeflow_id binary(16) DEFAULT NULL,
  reference_type varchar(255) DEFAULT NULL,
  reference_id binary(16) DEFAULT NULL,
  confbridge_id binary(16) DEFAULT NULL,
  pipecatcall_id binary(16) DEFAULT NULL,
  current_member_id binary(16) DEFAULT NULL,
  status varchar(255) DEFAULT NULL,
  stt_language varchar(255) NOT NULL DEFAULT '',
  tm_end datetime(6) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  metadata json NOT NULL DEFAULT (JSON_OBJECT()),
  active_reference_key binary(32) GENERATED ALWAYS AS (IF(status NOT IN ('terminated', 'terminating') AND reference_id != UNHEX('00000000000000000000000000000000'), UNHEX(SHA2(CONCAT_WS('|', customer_id, reference_type, reference_id), 256)), NULL)) STORED,
  listen_call_id binary(16) NOT NULL DEFAULT 0x00000000000000000000000000000000,
  PRIMARY KEY (id),
  UNIQUE KEY uq_aicall_active_reference_key (active_reference_key),
  KEY idx_ai_aicalls_customer_id (customer_id),
  KEY idx_ai_aicalls_reference_type (reference_type),
  KEY idx_ai_aicalls_reference_id (reference_id),
  KEY idx_ai_aicalls_create (tm_create),
  KEY idx_ai_aicalls_activeflow_id (activeflow_id),
  KEY idx_ai_aicalls_assistance_type (assistance_type),
  KEY idx_ai_aicalls_assistance_id (assistance_id),
  KEY idx_ai_aicalls_listen_call_id (listen_call_id)
);

CREATE TABLE IF NOT EXISTS ai_ais (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail text DEFAULT NULL,
  type varchar(255) NOT NULL DEFAULT 'normal',
  engine_model varchar(255) DEFAULT NULL,
  parameter json DEFAULT NULL,
  engine_key varchar(255) DEFAULT '',
  rag_id binary(16) DEFAULT NULL,
  init_prompt text DEFAULT NULL,
  tts_type varchar(255) DEFAULT '',
  tts_voice_id varchar(255) DEFAULT '',
  stt_type varchar(255) DEFAULT '',
  stt_language varchar(255) NOT NULL DEFAULT '',
  vad_config json DEFAULT NULL,
  smart_turn_enabled tinyint(1) NOT NULL DEFAULT 0,
  auto_aicall_audit_enabled tinyint(1) NOT NULL DEFAULT 0,
  tool_names json DEFAULT NULL,
  mcp_server_ids json DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  direct_id binary(16) DEFAULT NULL,
  direct_hash varchar(255) DEFAULT NULL,
  current_prompt_history_id binary(16) NOT NULL DEFAULT 0x00000000000000000000000000000000,
  is_insight_active tinyint(1) NOT NULL DEFAULT 0,
  active_insight_key binary(16) GENERATED ALWAYS AS (IF(type = 'insight' AND tm_delete IS NULL AND is_insight_active = TRUE, customer_id, NULL)) STORED,
  PRIMARY KEY (id),
  UNIQUE KEY uq_ai_active_insight_key (active_insight_key),
  KEY idx_ai_ais_create (tm_create),
  KEY idx_ai_ais_customer_id (customer_id)
);

CREATE TABLE IF NOT EXISTS ai_mcp_oauth_states (
  state varchar(64) NOT NULL,
  customer_id binary(16) NOT NULL,
  mcp_server_id binary(16) DEFAULT NULL,
  vendor varchar(64) NOT NULL,
  pkce_verifier varchar(128) NOT NULL,
  tm_create datetime(6) NOT NULL,
  tm_expire datetime(6) NOT NULL,
  PRIMARY KEY (state),
  KEY idx_ai_mcp_oauth_states_customer_id (customer_id),
  KEY idx_ai_mcp_oauth_states_tm_expire (tm_expire)
);

CREATE TABLE IF NOT EXISTS ai_mcp_servers (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  detail text DEFAULT NULL,
  url varchar(2048) DEFAULT NULL,
  status varchar(16) DEFAULT NULL,
  auth_type varchar(16) DEFAULT NULL,
  api_key_header varchar(255) DEFAULT NULL,
  secret_ciphertext blob DEFAULT NULL,
  secret_nonce binary(12) DEFAULT NULL,
  key_version smallint(6) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  oauth_vendor varchar(64) DEFAULT NULL,
  access_token_ciphertext blob DEFAULT NULL,
  access_token_nonce binary(12) DEFAULT NULL,
  access_token_expires_at datetime(6) DEFAULT NULL,
  refresh_token_ciphertext blob DEFAULT NULL,
  refresh_token_nonce binary(12) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_ai_mcp_servers_create (tm_create),
  KEY idx_ai_mcp_servers_customer_id (customer_id)
);

CREATE TABLE IF NOT EXISTS ai_messages (
  id binary(16) NOT NULL,
  customer_id binary(16) DEFAULT NULL,
  aicall_id binary(16) DEFAULT NULL,
  activeflow_id binary(16) DEFAULT NULL,
  direction varchar(255) DEFAULT NULL,
  role varchar(255) DEFAULT NULL,
  content text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  tool_calls json DEFAULT NULL,
  tool_call_id varchar(255) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  pipecatcall_id binary(16) DEFAULT NULL,
  delivery_status varchar(16) NOT NULL DEFAULT 'delivered',
  active_ai_id binary(16) NOT NULL DEFAULT 0x00000000000000000000000000000000,
  in_reply_to_message_id binary(16) NOT NULL DEFAULT 0x00000000000000000000000000000000,
  origin varchar(16) NOT NULL DEFAULT '',
  PRIMARY KEY (id),
  KEY idx_ai_messages_create (tm_create),
  KEY idx_ai_messages_customer_id (customer_id),
  KEY idx_ai_messages_aicall_id (aicall_id),
  KEY idx_ai_messages_activeflow_id (activeflow_id),
  KEY idx_ai_messages_pcc_delivery (pipecatcall_id,delivery_status)
);

CREATE TABLE IF NOT EXISTS ai_summaries (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  activeflow_id binary(16) NOT NULL,
  on_end_flow_id binary(16) NOT NULL,
  reference_type varchar(255) NOT NULL,
  reference_id binary(16) NOT NULL,
  status varchar(16) NOT NULL,
  language varchar(16) NOT NULL,
  content text NOT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_ai_summaries_customer_id (customer_id),
  KEY idx_ai_summaries_activeflow (activeflow_id),
  KEY idx_ai_summaries_on_end_flow_id (on_end_flow_id),
  KEY idx_ai_summaries_reference_id_language (reference_id,language),
  KEY idx_ai_summaries_reference_type (reference_type),
  KEY idx_ai_summaries_reference_id (reference_id)
);

CREATE TABLE IF NOT EXISTS ai_teams (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  name varchar(255) NOT NULL DEFAULT '',
  detail text DEFAULT NULL,
  start_member_id binary(16) DEFAULT NULL,
  members json DEFAULT NULL,
  parameter json DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  tm_delete datetime(6) DEFAULT NULL,
  direct_id binary(16) DEFAULT NULL,
  direct_hash varchar(255) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_ai_teams_tm_create (tm_create),
  KEY idx_ai_teams_customer_id (customer_id)
);
