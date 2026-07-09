create table ai_aicalls(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id

  assistance_type  varchar(255),  -- assistance type (ai or team)
  assistance_id    binary(16),    -- assistance id (ai or team uuid)
  ai_engine_model  varchar(255),
  ai_tts_type      varchar(255),
  ai_tts_voice_id  varchar(255),
  ai_stt_type      varchar(255),
  ai_vad_config    json,
  ai_smart_turn_enabled boolean not null default 0,
  parameter        json,

  activeflow_id     binary(16),
  reference_type    varchar(255),
  reference_id      binary(16),

  confbridge_id       binary(16),
  pipecatcall_id      binary(16),
  current_member_id   binary(16),

  status  varchar(255),   -- status

  gender    varchar(255), -- gender
  stt_language  varchar(255), -- stt language

  metadata  json,   -- metadata

  -- timestamps
  tm_end    datetime(6),  --
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  -- SQLite-compatible stand-in for the MySQL STORED generated column
  -- active_reference_key (BINARY(32) SHA2(...) hash, see
  -- bin-dbscheme-manager a5a40c93d3e6). This concat-based generated column
  -- preserves the same invariant (unique per active (customer_id,
  -- reference_type, reference_id), terminated/terminating rows AND legacy
  -- rows with a zero-value reference_id excluded) at test-data scale
  -- without needing a SHA2 equivalent in SQLite.
  active_reference_key varchar(255) GENERATED ALWAYS AS (
    CASE WHEN status NOT IN ('terminated', 'terminating')
      AND reference_id != X'00000000000000000000000000000000'
      THEN customer_id || '|' || reference_type || '|' || reference_id
      ELSE NULL
    END
  ) STORED,

  primary key(id)
);

create unique index uq_aicall_active_reference_key on ai_aicalls(active_reference_key);

create index idx_ai_aicalls_customer_id on ai_aicalls(customer_id);
create index idx_ai_aicalls_assistance_type on ai_aicalls(assistance_type);
create index idx_ai_aicalls_assistance_id on ai_aicalls(assistance_id);
create index idx_ai_aicalls_reference_type on ai_aicalls(reference_type);
create index idx_ai_aicalls_reference_id on ai_aicalls(reference_id);
create index idx_ai_aicalls_create on ai_aicalls(tm_create);
create index idx_ai_aicalls_activeflow_id on ai_aicalls(activeflow_id);
