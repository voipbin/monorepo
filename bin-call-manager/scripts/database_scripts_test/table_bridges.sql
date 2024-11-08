create table call_bridges(
  -- identity
  asterisk_id varchar(255), -- Asterisk id
  id          varchar(255), -- bridge id
  name        varchar(255), -- bridge name

  -- info
  type    varchar(255),   -- bridge's type
  tech    varchar(255),   -- bridge's technology
  class   varchar(255),   -- bridge's class
  creator varchar(255),   -- bridge creator

  -- video info
  video_mode      varchar(255), -- video's mode.
  video_source_id varchar(255),

  -- joined channel info
  channel_ids json, -- joined channel ids

  -- reference info
  reference_type  varchar(255),   -- reference type
  reference_id    binary(16),     -- reference id

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_call_bridges_create on call_bridges(tm_create);
create index idx_call_bridges_reference_id on call_bridges(reference_id);
