create table ai_aicalls(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id

  ai_id            binary(16),   -- ai id
  ai_engine_type   varchar(255), -- ai engine type
  ai_engine_model  varchar(255),
  ai_engine_data   json,

  activeflow_id     binary(16),
  reference_type    varchar(255),
  reference_id      binary(16),

  confbridge_id     binary(16),
  transcribe_id     binary(16),
  pipecatcall_id    binary(16),

  status  varchar(255),   -- status

  gender    varchar(255), -- gender
  language  varchar(255), -- language

  tts_streaming_id      binary(16),     -- tts streaming id
  tts_streaming_pod_id  varchar(255),   -- tts streaming pod id

  -- timestamps
  tm_end    datetime(6),  --
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_ai_aicalls_customer_id on ai_aicalls(customer_id);
create index idx_ai_aicalls_ai_id on ai_aicalls(ai_id);
create index idx_ai_aicalls_reference_type on ai_aicalls(reference_type);
create index idx_ai_aicalls_reference_id on ai_aicalls(reference_id);
create index idx_ai_aicalls_transcribe_id on ai_aicalls(transcribe_id);
create index idx_ai_aicalls_create on ai_aicalls(tm_create);
create index idx_ai_aicalls_activeflow_id on ai_aicalls(activeflow_id);

