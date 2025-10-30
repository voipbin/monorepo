create table pipecat_pipecatcalls(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id

  activeflow_id     binary(16),
  reference_type    varchar(255),
  reference_id      binary(16),

  host_id varchar(255),  -- host id

  llm_type      varchar(255),
  llm_messages  json,
  stt_type      varchar(255),
  tts_type      varchar(255),
  tts_voice_id  varchar(255),

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_pipecat_pipecatcalls_customer_id on pipecat_pipecatcalls(customer_id);
create index idx_pipecat_pipecatcalls_activeflow_id on pipecat_pipecatcalls(activeflow_id);
create index idx_pipecat_pipecatcalls_reference_type on pipecat_pipecatcalls(reference_type);
create index idx_pipecat_pipecatcalls_reference_id on pipecat_pipecatcalls(reference_id);
create index idx_pipecat_pipecatcalls_host_id on pipecat_pipecatcalls(host_id);
create index idx_pipecat_pipecatcalls_tm_create on pipecat_pipecatcalls(tm_create);

