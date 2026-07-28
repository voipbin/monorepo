create table ai_ais(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id

  -- info
  name        varchar(255),   -- name
  detail      text,           -- detail description

  engine_type   varchar(255),
  engine_model  varchar(255),
  parameter     json,
  engine_key    varchar(255),
  rag_id        binary(16),

  init_prompt   text,           -- initial prompt

  tts_type      varchar(255),
  tts_voice_id  varchar(255),

  stt_type      varchar(255),
  stt_language  varchar(255),

  vad_config  json,           -- VAD configuration

  smart_turn_enabled  boolean not null default 0,      -- smart turn detection enabled

  auto_aicall_audit_enabled  boolean not null default 0,   -- auto aicall audit enabled

  type  varchar(255) not null default 'normal',   -- ai type: normal, insight

  tool_names  json,           -- enabled tools for this AI

  current_prompt_history_id  binary(16),   -- current prompt history id

  direct_id     binary(16),     -- direct id
  direct_hash   varchar(255),   -- direct hash

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  -- SQLite-compatible stand-in for the MySQL STORED generated column
  -- active_insight_key (BINARY(16), see bin-dbscheme-manager migration
  -- for SQUARE-23, mirroring the active_reference_key pattern in
  -- table_ai_aicalls.sql). Computes customer_id only when type='insight'
  -- and the row is not soft-deleted; NULL otherwise (any number of NULLs
  -- may coexist under UNIQUE), enforcing "at most one active Insight AI
  -- per customer" without affecting type='normal' rows.
  active_insight_key varchar(255) GENERATED ALWAYS AS (
    CASE WHEN type = 'insight' AND tm_delete IS NULL
      THEN customer_id
      ELSE NULL
    END
  ) STORED,

  primary key(id)
);

create index idx_ai_ais_create on ai_ais(tm_create);
create index idx_ai_ais_customer_id on ai_ais(customer_id);
create unique index uq_ai_active_insight_key on ai_ais(active_insight_key);
