create table recordings(
  -- identity
  id                binary(16),     -- recording's id(name)
  user_id           integer,        -- user id
  type              varchar(16),    -- type of record. call, conference
  reference_id      binary(16),     -- referenced id. call-id, conference-id
  status            varchar(255),   -- current status of record.
  format            varchar(16),    -- recording's format. wav, ...
  filename          varchar(255),   -- recording's filename.

  -- asterisk info
  asterisk_id       varchar(255), -- Asterisk id
  channel_id        varchar(255), -- channel id

  -- timestamps
  tm_start  datetime(6),    -- record started timestamp.
  tm_end    datetime(6),    -- record ended timestamp.

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_recordings_user_id on recordings(user_id);
create index idx_recordings_tm_start on recordings(tm_start);
create index idx_recordings_filename on recordings(filename);
