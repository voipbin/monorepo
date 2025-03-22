create table call_recordings(
  -- identity
  id                binary(16),     -- recording's id(name)
  customer_id       binary(16),     -- customer id
  owner_type        varchar(255),   -- owner type
  owner_id          binary(16),     -- owner id

  activeflow_id     binary(16),     -- active flow id
  reference_type    varchar(16),    -- reference type. call, conference, ...
  reference_id      binary(16),     -- referenced id. call-id, conference-id
  status            varchar(255),   -- current status of record.
  format            varchar(16),    -- recording's format. wav, ...

  on_end_flow_id    binary(16),     -- on end flow id

  recording_name    varchar(255),   -- recording's basename
  filenames         json,           -- recording's filenames

  -- asterisk info
  asterisk_id       varchar(255), -- Asterisk id
  channel_ids       json,         -- channel ids

  -- timestamps
  tm_start  datetime(6),    -- record started timestamp.
  tm_end    datetime(6),    -- record ended timestamp.

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_call_recordings_tm_start on call_recordings(tm_start);
create index idx_call_recordings_customer_id on call_recordings(customer_id);
create index idx_call_recordings_owner_id on call_recordings(owner_id);
create index idx_call_recordings_reference_id on call_recordings(reference_id);
create index idx_call_recordings_recording_name on call_recordings(recording_name);
create index idx_call_recordings_on_end_flow_id on call_recordings(on_end_flow_id);
create index idx_call_recordings_activeflow_id on call_recordings(activeflow_id);