create table confbridges(
  -- identity
  id            binary(16),   -- id
  conference_id binary(16),   -- conference id
  type          varchar(255), -- type
  bridge_id     varchar(255), -- conference's bridge id

  channel_call_ids  json, -- channel_id-call_id in the confbridge

  -- record info
  recording_id   binary(16),  -- current record id
  recording_ids  json,        -- record ids

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_confbridges_create on confbridges(tm_create);
create index idx_confbridges_conference_id on confbridges(conference_id);
create index idx_confbridges_bridge_id on confbridges(bridge_id);
