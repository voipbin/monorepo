create table cm_bridges(
  -- identity
  asterisk_id varchar(255), -- Asterisk id
  id          varchar(255),       -- bridge id
  name        varchar(255),       -- bridge name

  -- info
  type    varchar(255),   -- bridge's type
  tech    varchar(255),   -- bridge's technology
  class   varchar(255),   -- bridge's class
  creator varchar(255),   -- bridge creator


  video_mode      varchar(255),
  video_source_id varchar(255),

  channel_ids json,

  -- conference info
  conference_id   binary(16),   -- conference's id
  conference_type varchar(255), -- conference's type

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  primary key(id)
);

create index idx_cm_bridges_create on cm_bridges(tm_create);
