create table call_confbridges(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id

  type          varchar(255), -- type
  status        varchar(16),  -- status
  bridge_id     varchar(255), -- conference's bridge id
  flags         json,

  channel_call_ids  json, -- channel_id-call_id in the confbridge

  -- record info
  recording_id   binary(16),  -- current record id
  recording_ids  json,        -- record ids

  -- external media info
  external_media_id binary(16),

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_call_confbridges_create on call_confbridges(tm_create);
create index idx_call_confbridges_customer_id on call_confbridges(customer_id);
create index idx_call_confbridges_bridge_id on call_confbridges(bridge_id);
