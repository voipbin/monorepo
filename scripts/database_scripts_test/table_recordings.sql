create table recordings(
  -- identity
  id                binary(16),     -- recording's id(name)
  customer_id       binary(16),     -- customer id

  reference_type    varchar(16),    -- reference type. call, conference, ...
  reference_id      binary(16),     -- referenced id. call-id, conference-id
  status            varchar(255),   -- current status of record.
  format            varchar(16),    -- recording's format. wav, ...

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

create index idx_recordings_tm_start on recordings(tm_start);
create index idx_recordings_customer_id on recordings(customer_id);
create index idx_recordings_reference_id on recordings(reference_id);
create index idx_recordings_recording_name on recordings(recording_name);
